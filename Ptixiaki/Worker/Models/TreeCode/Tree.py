from __future__ import annotations

import argparse
import csv
import io
import json
import os
from pathlib import Path
import sys
import time
from typing import Any
import uuid

import numpy as np
import onnxmltools
import pandas as pd
from sklearn.metrics import r2_score
from sklearn.model_selection import train_test_split
from skl2onnx.common.data_types import FloatTensorType
import xgboost as xgb


# Resolve the shared contract from this file, not from the Go process cwd.
BUNDLE_ROOT = Path(__file__).resolve().parents[4]
if str(BUNDLE_ROOT) not in sys.path:
    sys.path.insert(0, str(BUNDLE_ROOT))

from ml_shared.explain_artifacts import (  # noqa: E402
    PRICE_TARGET_NAME,
    PRICE_TARGET_UNIT,
    atomic_save_xgboost_model,
    atomic_write_bytes,
    atomic_write_json,
    atomic_write_text,
    build_explain_manifest,
    explain_artifact_filenames,
    RESIDENCE_PROPERTY_PHASE2_FEATURES,
    sha256_file,
    utc_now_iso,
)
from ml_shared.shap_validation import (  # noqa: E402
    ADDITIVITY_ATOL,
    ADDITIVITY_RTOL,
    additivity_tolerance,
)


VALIDATION_SEED = 42
INTERACTION_SAMPLE_SIZE = 2000
INTERACTION_SAMPLE_SEED = 42
EXPLAINABLE_PHASE2_MODELS = {
    "residence_properties_sell": "sell",
    "residence_properties_rent": "rent",
}
EXPECTED_RESIDENCE_PROPERTY_PHASE2_FEATURES = list(
    RESIDENCE_PROPERTY_PHASE2_FEATURES
)


def log(message: str) -> None:
    print(message, file=sys.stderr, flush=True)


def emit_shap_record(record: dict[str, Any]) -> None:
    encoded = json.dumps(
        record,
        ensure_ascii=False,
        separators=(",", ":"),
        allow_nan=False,
    )
    print(f"SHAP_RECORD\t{encoded}", flush=True)


def emit_progress(
    args: argparse.Namespace,
    stage: str,
    completed: int,
    total: int,
    started_at: float,
) -> None:
    # Go owns the human-readable progress logs. Python sends only structured
    # events on the same pipe as the explanation records.
    if args.emit_shap_records:
        emit_shap_record(
            {
                "type": "progress",
                "stage": stage,
                "completed": completed,
                "total": total,
                "elapsed_seconds": time.perf_counter() - started_at,
            }
        )


def interaction_sample_indices(uids: list[str], sample_size: int, seed: int) -> np.ndarray:
    if not uids or sample_size <= 0 or seed < 0:
        raise ValueError(
            "interaction sampling requires rows, a positive sample size and a nonnegative seed"
        )
    if len(uids) <= sample_size:
        return np.arange(len(uids))
    # Sort by stable UID before random selection so database row ordering does
    # not change which properties enter the sample.
    ordered = np.asarray(sorted(range(len(uids)), key=uids.__getitem__))
    selected = np.random.default_rng(seed).choice(
        len(uids), size=sample_size, replace=False
    )
    return ordered[np.sort(selected)]


def is_explainable_phase2(args: argparse.Namespace) -> bool:
    return (
        args.tree_type == "Phase2"
        and args.item_type_subtype in EXPLAINABLE_PHASE2_MODELS
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input_csv", type=str, required=True)
    parser.add_argument("--output_json_dir", type=str, required=True)
    parser.add_argument("--output_onnx_dir", type=str, required=True)
    parser.add_argument("--output_mappings_dir", type=str, required=True)
    parser.add_argument("--tree_type", type=str, required=True)
    parser.add_argument("--item_type_subtype", type=str, required=True)

    parser.add_argument("--target_col", type=str, default=PRICE_TARGET_NAME)
    parser.add_argument("--target_unit", type=str, default=PRICE_TARGET_UNIT)
    parser.add_argument("--columns_dropped", type=str, required=True)
    parser.add_argument("--categorical_cols", type=str, required=True)

    parser.add_argument("--max_depth", type=int, default=8)
    parser.add_argument("--min_child_weight", type=int, default=30)
    parser.add_argument("--learning_rate", type=float, default=0.05)
    parser.add_argument("--n_estimators", type=int, default=500)
    parser.add_argument("--subsample", type=float, default=0.8)
    parser.add_argument("--colsample_bytree", type=float, default=0.8)

    # UBJ + manifest export is automatic for final residence-properties Phase2
    # models. The expensive SHAP stream is opt-in and batch-bounded.
    parser.add_argument("--native_model_dir", type=str, default=None)
    parser.add_argument("--validation_fraction", type=float, default=0.2)
    parser.add_argument("--emit_shap_records", action="store_true")
    parser.add_argument("--shap_batch_size", type=int, default=256)
    parser.add_argument("--interaction_sample_size", type=int, default=INTERACTION_SAMPLE_SIZE)
    parser.add_argument("--interaction_sample_seed", type=int, default=INTERACTION_SAMPLE_SEED)
    parser.add_argument("--shap_additivity_atol", type=float, default=ADDITIVITY_ATOL)
    parser.add_argument("--shap_additivity_rtol", type=float, default=ADDITIVITY_RTOL)

    args = parser.parse_args()
    if not 0.0 < args.validation_fraction < 1.0:
        parser.error("--validation_fraction must be strictly between 0 and 1")
    if args.shap_batch_size <= 0:
        parser.error("--shap_batch_size must be greater than zero")
    if args.interaction_sample_size <= 0 or args.interaction_sample_seed < 0:
        parser.error("interaction sample size must be positive and seed must be nonnegative")
    if any(not np.isfinite(value) or value < 0 for value in (args.shap_additivity_atol, args.shap_additivity_rtol)):
        parser.error("SHAP additivity tolerances must be finite and nonnegative")
    if not args.target_unit.strip():
        parser.error("--target_unit cannot be empty")
    if args.max_depth < 0 or args.min_child_weight < 0:
        parser.error("tree depth and child weight cannot be negative")
    if args.learning_rate <= 0 or args.n_estimators <= 0:
        parser.error("learning rate and estimator count must be greater than zero")
    if not 0.0 < args.subsample <= 1.0 or not 0.0 < args.colsample_bytree <= 1.0:
        parser.error("subsample and colsample_bytree must be in (0, 1]")
    if args.emit_shap_records and not is_explainable_phase2(args):
        parser.error(
            "--emit_shap_records is only valid for Phase2 residence_properties_sell/rent"
        )
    if is_explainable_phase2(args) and (
        args.target_col != PRICE_TARGET_NAME
        or args.target_unit.strip() != PRICE_TARGET_UNIT
    ):
        parser.error(
            "Phase2 residence-properties requires the shared price-per-square-metre target contract"
        )
    return args


def model_hyperparameters(args: argparse.Namespace) -> dict[str, Any]:
    return {
        "tree_method": "hist",
        "max_depth": args.max_depth,
        "min_child_weight": args.min_child_weight,
        "learning_rate": args.learning_rate,
        "n_estimators": args.n_estimators,
        "subsample": args.subsample,
        "colsample_bytree": args.colsample_bytree,
        "random_state": VALIDATION_SEED,
    }


def new_model(args: argparse.Namespace) -> xgb.XGBRegressor:
    return xgb.XGBRegressor(**model_hyperparameters(args))


def validate_target(df: pd.DataFrame, target_column: str) -> pd.Series:
    if target_column not in df.columns:
        raise ValueError(f"target column {target_column!r} is missing")
    try:
        target = pd.to_numeric(df[target_column], errors="raise").astype("float64")
    except (TypeError, ValueError) as exc:
        raise ValueError(f"target column {target_column!r} must be numeric") from exc

    values = target.to_numpy(dtype=np.float64, copy=False)
    if not np.isfinite(values).all():
        raise ValueError(f"target column {target_column!r} contains non-finite values")
    if np.any(values == 0.0):
        raise ValueError(
            f"target column {target_column!r} contains zero; residual percentage would divide by zero"
        )
    return target


def validate_explainability_uids(df: pd.DataFrame) -> list[str]:
    if "UID" not in df.columns:
        raise ValueError("explainable Phase2 training requires a UID column")
    if df["UID"].isna().any():
        raise ValueError("explainable Phase2 training requires non-null UIDs")

    uids = [str(value).strip() for value in df["UID"].tolist()]
    if any(not uid for uid in uids):
        raise ValueError("explainable Phase2 training requires non-empty UIDs")
    if len(set(uids)) != len(uids):
        raise ValueError("explainable Phase2 training requires unique UIDs")
    return uids


def prepare_features(
    df: pd.DataFrame,
    args: argparse.Namespace,
) -> tuple[pd.DataFrame, list[str], dict[str, dict[str, int]]]:
    dropped = [column.strip() for column in args.columns_dropped.split(",") if column.strip()]
    present_dropped = [column for column in dropped if column in df.columns]
    features = df.drop(columns=present_dropped + [args.target_col]).copy()

    if features.shape[1] == 0:
        raise ValueError("training matrix has no feature columns")
    if not features.columns.is_unique:
        raise ValueError("training feature columns must be unique")

    original_feature_names = [str(column) for column in features.columns]
    if is_explainable_phase2(args):
        if original_feature_names != EXPECTED_RESIDENCE_PROPERTY_PHASE2_FEATURES:
            raise ValueError(
                "Phase2 residence-properties feature order mismatch; "
                f"expected {EXPECTED_RESIDENCE_PROPERTY_PHASE2_FEATURES}, got {original_feature_names}"
            )

    configured_categoricals = [
        column.strip() for column in args.categorical_cols.split(",") if column.strip()
    ]
    actual_categoricals = [
        column for column in configured_categoricals if column in features.columns
    ]
    categorical_mappings: dict[str, dict[str, int]] = {}

    for column in actual_categoricals:
        features[column] = features[column].astype("category")
        categories = [str(category) for category in features[column].cat.categories]
        categorical_mappings[column] = {
            category: index for index, category in enumerate(categories)
        }
        features[column] = features[column].cat.codes.astype("float32")

    for column in features.columns:
        if column not in actual_categoricals:
            try:
                features[column] = pd.to_numeric(
                    features[column], errors="raise"
                ).astype("float32")
            except (TypeError, ValueError) as exc:
                raise ValueError(f"feature column {column!r} must be numeric") from exc

    values = features.to_numpy(dtype=np.float32, copy=False)
    if not np.isfinite(values).all():
        bad_columns = [
            str(column)
            for column in features.columns
            if not np.isfinite(
                features[column].to_numpy(dtype=np.float32, copy=False)
            ).all()
        ]
        raise ValueError(f"training matrix contains non-finite values in columns: {bad_columns}")

    model_feature_names = [f"f{index}" for index in range(features.shape[1])]
    features.columns = model_feature_names
    if list(features.columns) != model_feature_names:
        raise ValueError("failed to establish the exact f0..fN model feature order")

    return features, original_feature_names, categorical_mappings


def calculate_validation_r2(
    features: pd.DataFrame,
    target: pd.Series,
    args: argparse.Namespace,
) -> float:
    if len(features) < 4:
        raise ValueError("at least four rows are required for deterministic validation R²")

    train_x, validation_x, train_y, validation_y = train_test_split(
        features,
        target,
        test_size=args.validation_fraction,
        random_state=VALIDATION_SEED,
    )
    if len(train_x) < 2 or len(validation_x) < 2:
        raise ValueError("validation split must contain at least two training and validation rows")

    validation_model = new_model(args)
    validation_model.fit(train_x, train_y)
    validation_predictions = validation_model.predict(validation_x)
    score = float(r2_score(validation_y, validation_predictions))
    if not np.isfinite(score):
        raise ValueError("deterministic validation R² is not finite")
    return score


def mapping_text(categorical_mappings: dict[str, dict[str, int]]) -> str:
    output = io.StringIO(newline="")
    writer = csv.writer(output, lineterminator="\n")
    for column, mapping in categorical_mappings.items():
        for category, code in mapping.items():
            writer.writerow((column, category, code))
    return output.getvalue()


def validate_native_feature_contract(
    booster: xgb.Booster,
    model_feature_names: list[str],
) -> None:
    if booster.num_features() != len(model_feature_names):
        raise ValueError(
            f"native model has {booster.num_features()} features, expected {len(model_feature_names)}"
        )
    if booster.feature_names != model_feature_names:
        raise ValueError(
            f"native model feature order mismatch: {booster.feature_names} != {model_feature_names}"
        )


def staging_path(destination: Path, activation_id: str) -> Path:
    return destination.with_name(
        f".{destination.stem}.{activation_id}.staged{destination.suffix}"
    )


def activate_staged_artifacts(
    artifacts: list[tuple[Path, Path]],
    activation_id: str,
) -> None:
    """Promote one latest-only artifact set and restore the old set on error."""

    backups: list[tuple[Path, Path]] = []
    promoted: list[Path] = []
    try:
        for staged_path, final_path in artifacts:
            backup_path = final_path.with_name(
                f".{final_path.stem}.{activation_id}.backup{final_path.suffix}"
            )
            if final_path.exists():
                os.replace(final_path, backup_path)
                backups.append((backup_path, final_path))
            os.replace(staged_path, final_path)
            promoted.append(final_path)
    except BaseException:
        for final_path in reversed(promoted):
            try:
                final_path.unlink()
            except FileNotFoundError:
                pass
        for backup_path, final_path in reversed(backups):
            if backup_path.exists():
                os.replace(backup_path, final_path)
        raise
    else:
        for backup_path, _ in backups:
            try:
                backup_path.unlink()
            except FileNotFoundError:
                pass


def stage_native_explainability_artifacts(
    *,
    model: xgb.XGBRegressor,
    args: argparse.Namespace,
    original_feature_names: list[str],
    categorical_mappings: dict[str, dict[str, int]],
    training_r2: float,
    validation_r2: float,
    row_count: int,
    activation_id: str,
) -> tuple[dict[str, Any], list[tuple[Path, Path]]]:
    transaction = EXPLAINABLE_PHASE2_MODELS[args.item_type_subtype]
    native_directory = Path(args.native_model_dir or args.output_onnx_dir)
    native_directory.mkdir(parents=True, exist_ok=True)
    model_filename, manifest_filename = explain_artifact_filenames(transaction)
    model_path = native_directory / model_filename
    manifest_path = native_directory / manifest_filename
    staged_model_path = staging_path(model_path, activation_id)
    staged_manifest_path = staging_path(manifest_path, activation_id)

    booster = model.get_booster()
    model_feature_names = [f"f{index}" for index in range(len(original_feature_names))]
    validate_native_feature_contract(booster, model_feature_names)

    try:
        atomic_save_xgboost_model(booster, staged_model_path)
        model_sha256 = sha256_file(staged_model_path)
        trained_at = utc_now_iso()
        manifest = build_explain_manifest(
            transaction=transaction,
            feature_names=original_feature_names,
            encoded_feature_names=model_feature_names,
            categorical_mappings=categorical_mappings,
            target_name=args.target_col,
            target_unit=args.target_unit.strip(),
            trained_at=trained_at,
            model_sha256=model_sha256,
            row_count=row_count,
            training_r2=training_r2,
            validation_r2=validation_r2,
            validation_fraction=args.validation_fraction,
            hyperparameters=model_hyperparameters(args),
        )
        if args.emit_shap_records:
            manifest["interaction_sampling"] = {
                "method": (
                    "uniform_without_replacement"
                    if row_count > args.interaction_sample_size
                    else "full_population"
                ),
                "population_count": row_count,
                "sample_count": min(row_count, args.interaction_sample_size),
                "seed": args.interaction_sample_seed,
            }
        atomic_write_json(staged_manifest_path, manifest)
    except BaseException:
        for staged_path in (staged_model_path, staged_manifest_path):
            try:
                staged_path.unlink()
            except FileNotFoundError:
                pass
        raise
    return manifest, [
        (staged_model_path, model_path),
        (staged_manifest_path, manifest_path),
    ]


def emit_native_shap_records(
    *,
    booster: xgb.Booster,
    features: pd.DataFrame,
    uids: list[str],
    manifest: dict[str, Any],
    args: argparse.Namespace,
) -> None:
    original_feature_names = list(manifest["original_feature_names"])
    model_feature_names = list(manifest["model_feature_names"])
    feature_count = int(manifest["feature_count"])
    interaction_count = feature_count * (feature_count - 1) // 2
    validate_native_feature_contract(booster, model_feature_names)

    if list(features.columns) != model_feature_names:
        raise ValueError("SHAP matrix columns do not exactly match model_feature_names")
    if features.shape != (len(uids), feature_count):
        raise ValueError("SHAP matrix shape does not match UID and manifest contracts")

    contribution_abs_sum = np.zeros(feature_count, dtype=np.float64)
    contribution_signed_sum = np.zeros(feature_count, dtype=np.float64)
    pair_i, pair_j = np.triu_indices(feature_count, k=1)
    interaction_abs_sum = np.zeros(interaction_count, dtype=np.float64)
    interaction_signed_sum = np.zeros(interaction_count, dtype=np.float64)
    baseline: float | None = None
    sampled_indices = interaction_sample_indices(
        uids, args.interaction_sample_size, args.interaction_sample_seed
    )
    sample_count = len(sampled_indices)
    contributions_started = time.perf_counter()
    emit_progress(args, "shap_properties", 0, len(features), contributions_started)

    for start in range(0, len(features), args.shap_batch_size):
        end = min(start + args.shap_batch_size, len(features))
        batch_values = features.iloc[start:end].to_numpy(dtype=np.float32, copy=False)
        matrix = xgb.DMatrix(batch_values, feature_names=model_feature_names)

        native_predictions = np.asarray(
            booster.predict(matrix, validate_features=True), dtype=np.float64
        ).reshape(-1)
        contributions_with_bias = np.asarray(
            booster.predict(matrix, pred_contribs=True, validate_features=True),
            dtype=np.float64,
        )
        expected_contribution_shape = (end - start, feature_count + 1)
        if contributions_with_bias.shape != expected_contribution_shape:
            raise ValueError(
                "unexpected pred_contribs shape: "
                f"{contributions_with_bias.shape}, expected {expected_contribution_shape}"
            )
        if not np.isfinite(native_predictions).all() or not np.isfinite(
            contributions_with_bias
        ).all():
            raise ValueError("native predictions or SHAP contributions contain non-finite values")

        batch_baselines = contributions_with_bias[:, -1]
        if baseline is None:
            baseline = float(batch_baselines[0])
            emit_shap_record(
                {
                    "type": "meta",
                    "schema_version": manifest["schema_version"],
                    "transaction": manifest["transaction"],
                    "target": manifest["target"],
                    "property_count": len(features),
                    "feature_count": feature_count,
                    "interaction_count": interaction_count,
                    "interaction_sample_count": sample_count,
                    "interaction_sample_seed": args.interaction_sample_seed,
                    "feature_names": original_feature_names,
                    "model_feature_names": model_feature_names,
                    "baseline_price_per_sqm": baseline,
                    "model_sha256": manifest["model_sha256"],
                    "trained_at": manifest["trained_at"],
                    "training_r2": manifest["training_r2"],
                    "validation_r2": manifest["validation_r2"],
                    "shap_batch_size": args.shap_batch_size,
                }
            )
        if not np.allclose(
            batch_baselines,
            baseline,
            rtol=args.shap_additivity_rtol,
            atol=args.shap_additivity_atol,
        ):
            raise ValueError("SHAP baseline is not stable across properties")

        contributions = contributions_with_bias[:, :-1]
        reconstructed = batch_baselines + contributions.sum(axis=1, dtype=np.float64)
        signed_errors = reconstructed - native_predictions
        absolute_errors = np.abs(signed_errors)
        allowed_errors = additivity_tolerance(
            native_predictions, batch_baselines, contributions,
            atol=args.shap_additivity_atol, rtol=args.shap_additivity_rtol,
        )
        if np.any(absolute_errors > allowed_errors):
            # Pick an actual violation, not the largest absolute error: a
            # higher-priced row can have a larger error yet pass its tolerance.
            worst_index = int(np.argmax(absolute_errors - allowed_errors))
            raise ValueError(
                "TreeSHAP additivity check failed for UID "
                f"{uids[start + worst_index]}: error={signed_errors[worst_index]}, "
                f"allowed={allowed_errors[worst_index]}, "
                f"prediction={native_predictions[worst_index]}, "
                f"reconstructed={reconstructed[worst_index]}"
            )

        contribution_abs_sum += np.abs(contributions).sum(axis=0, dtype=np.float64)
        contribution_signed_sum += contributions.sum(axis=0, dtype=np.float64)
        for offset, uid in enumerate(uids[start:end]):
            emit_shap_record(
                {
                    "type": "property",
                    "uid": uid,
                    "contributions": [float(value) for value in contributions[offset]],
                    "prediction_price_per_sqm": float(native_predictions[offset]),
                    "additivity_error": float(signed_errors[offset]),
                }
            )

        emit_progress(args, "shap_properties", end, len(features), contributions_started)

    row_count = float(len(features))
    mean_abs = contribution_abs_sum / row_count
    mean_signed = contribution_signed_sum / row_count
    total_mean_abs = float(mean_abs.sum())
    importance_order = sorted(
        range(feature_count), key=lambda index: (-mean_abs[index], index)
    )
    for rank, index in enumerate(importance_order, start=1):
        percentage = (
            0.0
            if total_mean_abs == 0.0
            else float(mean_abs[index] / total_mean_abs * 100.0)
        )
        emit_shap_record(
            {
                "type": "importance",
                "feature_name": original_feature_names[index],
                "feature_index": index,
                "mean_abs": float(mean_abs[index]),
                "mean_signed": float(mean_signed[index]),
                "percentage": percentage,
                "rank": rank,
            }
        )

    # Global interactions are estimated from a bounded, reproducible sample.
    # Ordinary explanations and feature importance above still use EVERY row.
    interactions_started = time.perf_counter()
    emit_progress(args, "shap_interactions", 0, sample_count, interactions_started)
    for start in range(0, sample_count, args.shap_batch_size):
        end = min(start + args.shap_batch_size, sample_count)
        batch_values = features.iloc[sampled_indices[start:end]].to_numpy(
            dtype=np.float32, copy=False
        )
        matrix = xgb.DMatrix(batch_values, feature_names=model_feature_names)
        interactions = np.asarray(
            booster.predict(matrix, pred_interactions=True, validate_features=True),
            dtype=np.float64,
        )
        expected_shape = (end - start, feature_count + 1, feature_count + 1)
        if interactions.shape != expected_shape:
            raise ValueError(
                f"unexpected pred_interactions shape: {interactions.shape}, expected {expected_shape}"
            )
        if not np.isfinite(interactions).all():
            raise ValueError("SHAP interaction values contain non-finite values")
        pair_values = interactions[:, pair_i, pair_j]
        interaction_abs_sum += np.abs(pair_values).sum(axis=0, dtype=np.float64)
        interaction_signed_sum += pair_values.sum(axis=0, dtype=np.float64)
        emit_progress(args, "shap_interactions", end, sample_count, interactions_started)

    interaction_mean_abs = interaction_abs_sum / sample_count
    interaction_mean_signed = interaction_signed_sum / sample_count
    interaction_order = sorted(
        range(interaction_count),
        key=lambda index: (
            -interaction_mean_abs[index],
            int(pair_i[index]),
            int(pair_j[index]),
        ),
    )
    for rank, pair_index in enumerate(interaction_order, start=1):
        left = int(pair_i[pair_index])
        right = int(pair_j[pair_index])
        emit_shap_record(
            {
                "type": "interaction",
                "feature_1": original_feature_names[left],
                "feature_2": original_feature_names[right],
                "feature_1_index": left,
                "feature_2_index": right,
                "mean_abs": float(interaction_mean_abs[pair_index]),
                "mean_signed": float(interaction_mean_signed[pair_index]),
                "rank": rank,
            }
        )


def main() -> None:
    args = parse_args()
    stage_started = time.perf_counter()
    emit_progress(args, "prepare", 0, 1, stage_started)

    json_directory = Path(args.output_json_dir)
    onnx_directory = Path(args.output_onnx_dir)
    mappings_directory = Path(args.output_mappings_dir)
    json_directory.mkdir(parents=True, exist_ok=True)
    onnx_directory.mkdir(parents=True, exist_ok=True)
    mappings_directory.mkdir(parents=True, exist_ok=True)

    base_name = f"{args.tree_type}_{args.item_type_subtype}"
    json_output_path = json_directory / f"{base_name}_PricePredictions.json"
    onnx_output_path = onnx_directory / f"{base_name}_Tree.onnx"
    mappings_output_path = mappings_directory / f"CategoryMappings_{base_name}.txt"

    df = pd.read_csv(args.input_csv)
    if df.empty:
        raise ValueError("training CSV must contain at least one data row")
    if not df.columns.is_unique:
        raise ValueError("training CSV column names must be unique")

    target = validate_target(df, args.target_col)
    uids = validate_explainability_uids(df) if is_explainable_phase2(args) else []
    features, original_feature_names, categorical_mappings = prepare_features(df, args)
    model_feature_names = list(features.columns)
    emit_progress(args, "prepare", 1, 1, stage_started)

    stage_started = time.perf_counter()
    emit_progress(args, "validation", 0, 1, stage_started)
    validation_r2 = calculate_validation_r2(features, target, args)
    emit_progress(args, "validation", 1, 1, stage_started)
    log(
        f"Validation {base_name}: R2={validation_r2:.6f} "
        f"(seed={VALIDATION_SEED}, fraction={args.validation_fraction})"
    )

    log(f"Training final {base_name} model on all {len(features)} rows...")
    stage_started = time.perf_counter()
    emit_progress(args, "training", 0, 1, stage_started)
    model = new_model(args)
    model.fit(features, target)
    predictions = np.asarray(model.predict(features), dtype=np.float64)
    if predictions.shape != (len(df),) or not np.isfinite(predictions).all():
        raise ValueError("final model predictions are missing or non-finite")
    training_r2 = float(r2_score(target, predictions))
    if not np.isfinite(training_r2):
        raise ValueError("all-row training R² is not finite")
    log(f"Final all-row {base_name}: R2={training_r2:.6f}")
    emit_progress(args, "training", 1, 1, stage_started)

    output_df = pd.DataFrame(index=df.index)
    if "UID" in df.columns:
        output_df["UID"] = df["UID"]
    output_df["AskingPricePerSqm"] = target
    output_df["PredictedPricePerSqm"] = predictions
    # Preserve the existing sign and percentage semantics exactly.
    output_df["ResidualPercentage"] = ((target - predictions) / target) * 100.0
    if not np.isfinite(
        output_df["ResidualPercentage"].to_numpy(dtype=np.float64)
    ).all():
        raise ValueError("residual percentages are non-finite")
    prediction_json = output_df.to_json(orient="records", indent=2)

    log("Converting final model to ONNX format...")
    stage_started = time.perf_counter()
    emit_progress(args, "onnx", 0, 1, stage_started)
    initial_types = [("input", FloatTensorType([None, features.shape[1]]))]
    onnx_model = onnxmltools.convert_xgboost(
        model,
        initial_types=initial_types,
        target_opset=15,
    )
    onnx_bytes = onnx_model.SerializeToString()
    emit_progress(args, "onnx", 1, 1, stage_started)

    if is_explainable_phase2(args):
        stage_started = time.perf_counter()
        emit_progress(args, "artifacts", 0, 1, stage_started)
        activation_id = uuid.uuid4().hex
        staged_artifacts: list[tuple[Path, Path]] = []
        try:
            staged_mapping_path = staging_path(mappings_output_path, activation_id)
            staged_json_path = staging_path(json_output_path, activation_id)
            staged_onnx_path = staging_path(onnx_output_path, activation_id)
            atomic_write_text(staged_mapping_path, mapping_text(categorical_mappings))
            staged_artifacts.append((staged_mapping_path, mappings_output_path))
            atomic_write_text(staged_json_path, prediction_json)
            staged_artifacts.append((staged_json_path, json_output_path))
            atomic_write_bytes(staged_onnx_path, onnx_bytes)
            staged_artifacts.append((staged_onnx_path, onnx_output_path))

            manifest, native_artifacts = stage_native_explainability_artifacts(
                model=model,
                args=args,
                original_feature_names=original_feature_names,
                categorical_mappings=categorical_mappings,
                training_r2=training_r2,
                validation_r2=validation_r2,
                row_count=len(features),
                activation_id=activation_id,
            )
            staged_artifacts.extend(native_artifacts)
            if manifest["model_feature_names"] != model_feature_names:
                raise ValueError("manifest and training model feature orders differ")
            emit_progress(args, "artifacts", 1, 1, stage_started)
            if args.emit_shap_records:
                emit_native_shap_records(
                    booster=model.get_booster(),
                    features=features,
                    uids=uids,
                    manifest=manifest,
                    args=args,
                )

            # Publish only after the complete validation/SHAP pass. Manifest is
            # last so a concurrent/restarted loader never accepts a new UBJ with
            # stale metadata; hash validation rejects the brief transition.
            stage_started = time.perf_counter()
            emit_progress(args, "activate", 0, 1, stage_started)
            activate_staged_artifacts(staged_artifacts, activation_id)
            staged_artifacts.clear()
            emit_progress(args, "activate", 1, 1, stage_started)
            log(f"SUCCESS: activated mapping {mappings_output_path}")
            log(f"SUCCESS: activated predictions {json_output_path}")
            log(f"SUCCESS: activated ONNX model {onnx_output_path}")
            log(
                "SUCCESS: activated native UBJ and manifest in "
                f"{Path(args.native_model_dir or args.output_onnx_dir)}"
            )
        finally:
            for staged_path, _ in staged_artifacts:
                try:
                    staged_path.unlink()
                except FileNotFoundError:
                    pass
    else:
        atomic_write_text(mappings_output_path, mapping_text(categorical_mappings))
        atomic_write_text(json_output_path, prediction_json)
        atomic_write_bytes(onnx_output_path, onnx_bytes)
        log(f"SUCCESS: atomically saved mapping to {mappings_output_path}")
        log(f"SUCCESS: atomically saved JSON predictions to {json_output_path}")
        log(f"SUCCESS: atomically saved ONNX model to {onnx_output_path}")


if __name__ == "__main__":
    main()
