"""Durable, validated artifacts shared by model training and explanation serving."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import re
import tempfile
from typing import Any, Mapping, Sequence

import xgboost as xgb


MANIFEST_SCHEMA_VERSION = 1
MODEL_FORMAT = "xgboost-ubj"
SUPPORTED_TRANSACTIONS = ("sell", "rent")
PRICE_TARGET_NAME = "AskingPricePerSqm"
PRICE_TARGET_UNIT = "EUR_per_square_meter"
RESIDENCE_PROPERTY_PHASE2_FEATURES = (
    "SubType",
    "AdminName",
    "SquareMeters",
    "Age",
    "RenovAge",
    "NoView",
    "ViewForest",
    "ViewMountain",
    "ViewSea",
    "ViewUnlimited",
    "NoFacade",
    "FacadeCorner",
    "FacadeFrontage",
    "FacadeInterior",
    "FacadeThrough",
    "HeatType",
    "HeatFrom",
    "EnergyPerformance",
    "WC",
    "AC",
    "Elevator",
    "Parking",
    "StorageRoom",
    "AluminumFrames",
    "Bedrooms",
    "Pool",
    "Garden",
    "FloorLevel",
    "FloorCount",
    "GeoPremium",
)
_SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


class ArtifactValidationError(ValueError):
    """Raised when a native model and its manifest do not form a trusted pair."""


@dataclass(frozen=True)
class LoadedExplainArtifact:
    booster: xgb.Booster
    manifest: dict[str, Any]
    artifact_directory: Path


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def explain_artifact_filenames(transaction: str) -> tuple[str, str]:
    if transaction not in SUPPORTED_TRANSACTIONS:
        raise ValueError(f"unsupported transaction: {transaction!r}")
    stem = f"Phase2_residence_properties_{transaction}"
    return f"{stem}_Tree.ubj", f"{stem}_Manifest.json"


def sha256_file(path: os.PathLike[str] | str) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _prepare_destination(path: os.PathLike[str] | str) -> Path:
    destination = Path(path)
    destination.parent.mkdir(parents=True, exist_ok=True)
    return destination


def atomic_write_bytes(path: os.PathLike[str] | str, data: bytes) -> None:
    destination = _prepare_destination(path)
    fd, temporary_name = tempfile.mkstemp(
        dir=destination.parent,
        prefix=f".{destination.name}.",
        suffix=".tmp",
    )
    try:
        with os.fdopen(fd, "wb") as temporary:
            temporary.write(data)
            temporary.flush()
            os.fsync(temporary.fileno())
        os.replace(temporary_name, destination)
    except BaseException:
        try:
            os.unlink(temporary_name)
        except FileNotFoundError:
            pass
        raise


def atomic_write_text(
    path: os.PathLike[str] | str,
    text: str,
    *,
    encoding: str = "utf-8",
) -> None:
    atomic_write_bytes(path, text.encode(encoding))


def atomic_write_json(path: os.PathLike[str] | str, value: Mapping[str, Any]) -> None:
    payload = json.dumps(
        value,
        ensure_ascii=False,
        indent=2,
        allow_nan=False,
    )
    atomic_write_text(path, payload + "\n")


def atomic_save_xgboost_model(booster: xgb.Booster, path: os.PathLike[str] | str) -> None:
    """Save through a same-directory UBJ temporary file, then atomically replace."""

    destination = _prepare_destination(path)
    if destination.suffix.lower() != ".ubj":
        raise ValueError("native explainability models must use the .ubj extension")

    fd, temporary_name = tempfile.mkstemp(
        dir=destination.parent,
        prefix=f".{destination.stem}.",
        suffix=".ubj",
    )
    os.close(fd)
    try:
        booster.save_model(temporary_name)
        with open(temporary_name, "r+b") as temporary:
            os.fsync(temporary.fileno())
        os.replace(temporary_name, destination)
    except BaseException:
        try:
            os.unlink(temporary_name)
        except FileNotFoundError:
            pass
        raise


def build_explain_manifest(
    *,
    transaction: str,
    feature_names: Sequence[str],
    encoded_feature_names: Sequence[str],
    categorical_mappings: Mapping[str, Mapping[str, int]],
    target_name: str,
    target_unit: str,
    trained_at: str,
    model_sha256: str,
    row_count: int,
    training_r2: float,
    validation_r2: float,
    validation_fraction: float,
    hyperparameters: Mapping[str, Any],
) -> dict[str, Any]:
    model_file, _ = explain_artifact_filenames(transaction)
    return {
        "schema_version": MANIFEST_SCHEMA_VERSION,
        "model_format": MODEL_FORMAT,
        "model_file": model_file,
        "model_sha256": model_sha256,
        "feature_count": len(feature_names),
        "original_feature_names": list(feature_names),
        "model_feature_names": list(encoded_feature_names),
        "categorical_mappings": {
            str(column): {str(category): int(code) for category, code in mapping.items()}
            for column, mapping in categorical_mappings.items()
        },
        "unknown_category_code": -1,
        "target": {"name": target_name, "unit": target_unit},
        "transaction": transaction,
        "trained_at": trained_at,
        "row_count": int(row_count),
        "training_r2": float(training_r2),
        "validation_r2": float(validation_r2),
        "validation": {
            "metric": "r2",
            "score": float(validation_r2),
            "seed": 42,
            "fraction": float(validation_fraction),
        },
        "hyperparameters": dict(hyperparameters),
    }


def _require_string_list(manifest: Mapping[str, Any], key: str) -> list[str]:
    value = manifest.get(key)
    if not isinstance(value, list) or not value or not all(isinstance(item, str) and item for item in value):
        raise ArtifactValidationError(f"manifest {key!r} must be a non-empty string array")
    if len(set(value)) != len(value):
        raise ArtifactValidationError(f"manifest {key!r} contains duplicate names")
    return value


def _validate_trained_at(value: Any) -> str:
    if not isinstance(value, str) or not value:
        raise ArtifactValidationError("manifest 'trained_at' must be a timestamp string")
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as exc:
        raise ArtifactValidationError("manifest 'trained_at' is not ISO-8601") from exc
    if parsed.tzinfo is None:
        raise ArtifactValidationError("manifest 'trained_at' must include a timezone")
    return value


def load_explain_artifact(
    artifact_directory: os.PathLike[str] | str,
    transaction: str,
) -> LoadedExplainArtifact:
    """Load and fully validate one immutable UBJ + manifest pair."""

    directory = Path(artifact_directory)
    model_file, manifest_file = explain_artifact_filenames(transaction)
    model_path = directory / model_file
    manifest_path = directory / manifest_file

    if not model_path.is_file() or not manifest_path.is_file():
        raise ArtifactValidationError(
            f"missing artifact pair for {transaction!r} in {directory}"
        )

    try:
        with open(manifest_path, "r", encoding="utf-8") as source:
            manifest = json.load(source)
    except (OSError, json.JSONDecodeError) as exc:
        raise ArtifactValidationError(f"cannot read manifest {manifest_path}: {exc}") from exc

    if not isinstance(manifest, dict):
        raise ArtifactValidationError("manifest root must be an object")
    if manifest.get("schema_version") != MANIFEST_SCHEMA_VERSION:
        raise ArtifactValidationError("unsupported manifest schema_version")
    if manifest.get("model_format") != MODEL_FORMAT:
        raise ArtifactValidationError("manifest model_format is not xgboost-ubj")
    if manifest.get("model_file") != model_file:
        raise ArtifactValidationError("manifest model_file does not match the requested model")
    if manifest.get("transaction") != transaction:
        raise ArtifactValidationError("manifest transaction does not match the requested model")

    feature_names = _require_string_list(manifest, "original_feature_names")
    encoded_feature_names = _require_string_list(manifest, "model_feature_names")
    if type(manifest.get("feature_count")) is not int or manifest["feature_count"] != len(feature_names):
        raise ArtifactValidationError("manifest feature_count does not match original_feature_names")
    if type(manifest.get("row_count")) is not int or manifest["row_count"] <= 0:
        raise ArtifactValidationError("manifest row_count must be a positive integer")
    for metric_name in ("training_r2", "validation_r2"):
        metric = manifest.get(metric_name)
        if type(metric) not in (int, float) or not math.isfinite(float(metric)):
            raise ArtifactValidationError(f"manifest {metric_name} must be finite")
    expected_encoded_names = [f"f{index}" for index in range(len(feature_names))]
    if encoded_feature_names != expected_encoded_names:
        raise ArtifactValidationError("model_feature_names must be the ordered f0..fN contract")
    if manifest.get("unknown_category_code") != -1:
        raise ArtifactValidationError("manifest unknown_category_code must be -1")

    validation = manifest.get("validation")
    if not isinstance(validation, dict) or validation.get("metric") != "r2":
        raise ArtifactValidationError("manifest validation metric must be r2")
    if validation.get("seed") != 42:
        raise ArtifactValidationError("manifest validation seed must be 42")
    fraction = validation.get("fraction")
    score = validation.get("score")
    if not isinstance(fraction, (int, float)) or not math.isfinite(float(fraction)):
        raise ArtifactValidationError("manifest validation fraction must be finite")
    if not 0.0 < float(fraction) < 1.0:
        raise ArtifactValidationError("manifest validation fraction must be strictly between 0 and 1")
    if type(score) not in (int, float) or not math.isfinite(float(score)):
        raise ArtifactValidationError("manifest validation score must be finite")
    if float(score) != float(manifest["validation_r2"]):
        raise ArtifactValidationError("manifest validation scores disagree")

    target = manifest.get("target")
    if not isinstance(target, dict) or not isinstance(target.get("name"), str) or not target.get("name"):
        raise ArtifactValidationError("manifest target.name is required")
    if not isinstance(target.get("unit"), str) or not target.get("unit"):
        raise ArtifactValidationError("manifest target.unit is required")
    _validate_trained_at(manifest.get("trained_at"))

    mappings = manifest.get("categorical_mappings")
    if not isinstance(mappings, dict):
        raise ArtifactValidationError("manifest categorical_mappings must be an object")
    for column, mapping in mappings.items():
        if column not in feature_names or not isinstance(mapping, dict):
            raise ArtifactValidationError("manifest contains an invalid categorical mapping")
        if not all(isinstance(category, str) and type(code) is int for category, code in mapping.items()):
            raise ArtifactValidationError("categorical mapping entries must be string-to-integer")
        codes = list(mapping.values())
        if len(set(codes)) != len(codes) or sorted(codes) != list(range(len(codes))):
            raise ArtifactValidationError(
                "categorical mapping codes must be unique, nonnegative, and contiguous"
            )

    expected_hash = manifest.get("model_sha256")
    if not isinstance(expected_hash, str) or not _SHA256_RE.fullmatch(expected_hash):
        raise ArtifactValidationError("manifest model_sha256 is invalid")
    actual_hash = sha256_file(model_path)
    if actual_hash != expected_hash:
        raise ArtifactValidationError("model SHA-256 does not match the manifest")

    booster = xgb.Booster()
    try:
        booster.load_model(model_path)
    except xgb.core.XGBoostError as exc:
        raise ArtifactValidationError(f"cannot load native XGBoost model: {exc}") from exc

    if booster.num_features() != len(feature_names):
        raise ArtifactValidationError("model feature count does not match the manifest")
    if booster.feature_names != encoded_feature_names:
        raise ArtifactValidationError("model feature names do not match model_feature_names")

    return LoadedExplainArtifact(
        booster=booster,
        manifest=manifest,
        artifact_directory=directory.resolve(),
    )
