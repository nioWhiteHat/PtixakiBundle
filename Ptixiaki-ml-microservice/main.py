from __future__ import annotations

from contextlib import asynccontextmanager
from dataclasses import dataclass, field
import logging
import math
import os
from pathlib import Path
import sys
import threading
from typing import Any, Dict, List, Literal, Sequence

from fastapi import FastAPI, HTTPException
import numpy as np
import pandas as pd
from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator
import shap
from sklearn.metrics import r2_score
from sklearn.preprocessing import StandardScaler
import xgboost as xgb


BUNDLE_ROOT = Path(__file__).resolve().parent.parent
if str(BUNDLE_ROOT) not in sys.path:
    sys.path.insert(0, str(BUNDLE_ROOT))

from ml_shared.explain_artifacts import (  # noqa: E402
    ArtifactValidationError,
    LoadedExplainArtifact,
    PRICE_TARGET_NAME,
    PRICE_TARGET_UNIT,
    RESIDENCE_PROPERTY_PHASE2_FEATURES,
    explain_artifact_filenames,
    load_explain_artifact,
)
from ml_shared.shap_validation import (  # noqa: E402
    ADDITIVITY_ATOL as EXPLANATION_ADDITIVITY_ATOL,
    ADDITIVITY_RTOL as EXPLANATION_ADDITIVITY_RTOL,
    additivity_tolerance,
)


logger = logging.getLogger("ptixiaki-ml")
Transaction = Literal["sell", "rent"]


def _expanded_model_root_candidates(root: Path) -> list[Path]:
    return [
        root,
        root / "TreeOutputs",
        root / "Models" / "TreeOutputs",
        root / "Worker" / "Models" / "TreeOutputs",
    ]


def discover_model_roots() -> tuple[Path, ...]:
    """Resolve explicit configuration first, then stable source/cwd fallbacks."""

    service_directory = Path(__file__).resolve().parent
    roots: list[Path] = []
    configured_root = os.getenv("PTIXIAKI_MODEL_ROOT", "").strip()
    if configured_root:
        roots.extend(_expanded_model_root_candidates(Path(configured_root).expanduser()))

    roots.extend(
        [
            service_directory / "models",
            service_directory / "Models" / "TreeOutputs",
            service_directory.parent / "Ptixiaki" / "Worker" / "Models" / "TreeOutputs",
            Path.cwd() / "Worker" / "Models" / "TreeOutputs",
            Path.cwd() / "Ptixiaki" / "Worker" / "Models" / "TreeOutputs",
        ]
    )

    unique_roots: list[Path] = []
    seen: set[str] = set()
    for root in roots:
        normalized = root.resolve(strict=False)
        key = os.path.normcase(str(normalized))
        if key not in seen:
            seen.add(key)
            unique_roots.append(normalized)
    return tuple(unique_roots)


@dataclass
class RegisteredExplainModel:
    artifact: LoadedExplainArtifact
    prediction_lock: threading.Lock = field(default_factory=threading.Lock)


class ExplainModelRegistry:
    """Immutable-entry registry with validated, all-or-nothing requested reloads."""

    def __init__(self, model_roots: Sequence[Path] | None = None) -> None:
        self._model_roots = tuple(model_roots or discover_model_roots())
        self._models: dict[str, RegisteredExplainModel] = {}
        self._lock = threading.RLock()

    @property
    def model_roots(self) -> tuple[Path, ...]:
        return self._model_roots

    def _find_and_load(self, transaction: str) -> RegisteredExplainModel:
        model_file, manifest_file = explain_artifact_filenames(transaction)
        for root in self._model_roots:
            model_path = root / model_file
            manifest_path = root / manifest_file
            if model_path.is_file() and manifest_path.is_file():
                artifact = load_explain_artifact(root, transaction)
                manifest = artifact.manifest
                if tuple(manifest["original_feature_names"]) != RESIDENCE_PROPERTY_PHASE2_FEATURES:
                    raise ArtifactValidationError(
                        "manifest does not match the residence/property Phase2 feature contract"
                    )
                if manifest["target"] != {
                    "name": PRICE_TARGET_NAME,
                    "unit": PRICE_TARGET_UNIT,
                }:
                    raise ArtifactValidationError(
                        "manifest does not match the price-per-square-metre target contract"
                    )
                self._smoke_test(artifact)
                return RegisteredExplainModel(artifact=artifact)
        searched = ", ".join(str(root) for root in self._model_roots)
        raise ArtifactValidationError(
            f"no complete {transaction!r} UBJ/manifest pair found; searched: {searched}"
        )

    @staticmethod
    def _smoke_test(artifact: LoadedExplainArtifact) -> None:
        manifest = artifact.manifest
        model_feature_names = list(manifest["model_feature_names"])
        feature_count = int(manifest["feature_count"])
        values = np.zeros((1, feature_count), dtype=np.float32)
        matrix = xgb.DMatrix(values, feature_names=model_feature_names)
        prediction = np.asarray(
            artifact.booster.predict(matrix, validate_features=True), dtype=np.float64
        ).reshape(-1)
        contributions = np.asarray(
            artifact.booster.predict(
                matrix,
                pred_contribs=True,
                validate_features=True,
            ),
            dtype=np.float64,
        )
        if prediction.shape != (1,) or contributions.shape != (1, feature_count + 1):
            raise ArtifactValidationError("native model smoke-test output shape is invalid")
        if not np.isfinite(prediction).all() or not np.isfinite(contributions).all():
            raise ArtifactValidationError("native model smoke-test output is non-finite")
        reconstructed = float(contributions[0, -1] + contributions[0, :-1].sum())
        allowed_error = float(additivity_tolerance(
            prediction[0], contributions[0, -1], contributions[0, :-1],
        ))
        if abs(reconstructed - float(prediction[0])) > allowed_error:
            raise ArtifactValidationError("native model fails the TreeSHAP additivity smoke test")

    def reload(
        self,
        transactions: Sequence[str],
        expected_model_sha256: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        requested = list(transactions)
        if not requested or len(set(requested)) != len(requested):
            raise ValueError("transactions must be non-empty and unique")

        # Do all filesystem, hash, manifest, model, and prediction validation
        # before acquiring the swap lock. A failure leaves the live map intact.
        replacements = {
            transaction: self._find_and_load(transaction) for transaction in requested
        }
        if expected_model_sha256 is not None:
            if set(expected_model_sha256) != set(requested):
                raise ValueError(
                    "expected_model_sha256 keys must exactly match transactions"
                )
            for transaction, registered in replacements.items():
                actual_hash = registered.artifact.manifest["model_sha256"]
                if expected_model_sha256[transaction] != actual_hash:
                    raise ArtifactValidationError(
                        f"{transaction} model hash does not match the Worker publication"
                    )
        with self._lock:
            updated = dict(self._models)
            updated.update(replacements)
            self._models = updated

        return [self._summary(transaction, replacements[transaction]) for transaction in requested]

    @staticmethod
    def _summary(
        transaction: str,
        registered: RegisteredExplainModel,
    ) -> dict[str, Any]:
        manifest = registered.artifact.manifest
        return {
            "transaction": transaction,
            "model_sha256": manifest["model_sha256"],
            "trained_at": manifest["trained_at"],
            "feature_count": manifest["feature_count"],
            "artifact_directory": str(registered.artifact.artifact_directory),
        }

    def explain(self, transaction: str, features: Sequence[float]) -> dict[str, Any]:
        with self._lock:
            registered = self._models.get(transaction)
        if registered is None:
            raise LookupError(f"no explainability model is loaded for {transaction!r}")

        manifest = registered.artifact.manifest
        feature_names = list(manifest["original_feature_names"])
        model_feature_names = list(manifest["model_feature_names"])
        feature_count = int(manifest["feature_count"])
        if len(features) != feature_count:
            raise ValueError(
                f"expected {feature_count} features in exact manifest order, got {len(features)}"
            )

        values = np.asarray(features, dtype=np.float64)
        if values.shape != (feature_count,) or not np.isfinite(values).all():
            raise ValueError("features must be a finite one-dimensional numeric array")
        float32_values = values.astype(np.float32)
        if not np.isfinite(float32_values).all():
            raise ValueError("features exceed the finite float32 range used by XGBoost")

        matrix = xgb.DMatrix(
            float32_values.reshape(1, feature_count),
            feature_names=model_feature_names,
        )
        with registered.prediction_lock:
            native_prediction = np.asarray(
                registered.artifact.booster.predict(
                    matrix,
                    validate_features=True,
                ),
                dtype=np.float64,
            ).reshape(-1)
            with_bias = np.asarray(
                registered.artifact.booster.predict(
                    matrix,
                    pred_contribs=True,
                    validate_features=True,
                ),
                dtype=np.float64,
            )

        if native_prediction.shape != (1,) or with_bias.shape != (1, feature_count + 1):
            raise RuntimeError("native XGBoost explanation returned an unexpected shape")
        if not np.isfinite(native_prediction).all() or not np.isfinite(with_bias).all():
            raise RuntimeError("native XGBoost explanation returned non-finite values")

        ordered_contributions = with_bias[0, :-1]
        baseline = float(with_bias[0, -1])
        prediction = float(native_prediction[0])
        additivity_error = float(baseline + ordered_contributions.sum() - prediction)
        allowed_error = float(additivity_tolerance(prediction, baseline, ordered_contributions))
        if abs(additivity_error) > allowed_error:
            raise RuntimeError(
                "native XGBoost explanation failed additivity validation: "
                f"error={additivity_error}, allowed={allowed_error}"
            )
        return {
            "transaction": transaction,
            "feature_names": feature_names,
            "contributions": [float(value) for value in ordered_contributions],
            "baseline_price_per_sqm": baseline,
            "prediction_price_per_sqm": prediction,
            "additivity_error": additivity_error,
            "model_sha256": manifest["model_sha256"],
            "trained_at": manifest["trained_at"],
        }


explain_registry = ExplainModelRegistry()


@asynccontextmanager
async def lifespan(_: FastAPI):
    try:
        loaded = explain_registry.reload(("sell", "rent"))
        logger.info("Loaded explainability models at startup: %s", loaded)
    except (ArtifactValidationError, OSError, ValueError, xgb.core.XGBoostError) as exc:
        # Prediction/explanation endpoints remain explicit (503) until durable
        # pairs exist. Never publish a partially validated startup registry.
        logger.warning("Explainability models were not loaded at startup: %s", exc)
    yield


app = FastAPI(lifespan=lifespan)


class TrainRequest(BaseModel):
    data: List[Dict[str, Any]]
    drop_columns: List[str]
    categorical_columns: List[str]
    max_depth: int
    min_child_weight: float
    learning_rate: float
    n_estimators: int
    target_column: str = "AskingPricePerSqm"


class ExplainPropertyRequest(BaseModel):
    model_config = ConfigDict(extra="forbid", allow_inf_nan=False)

    transaction: Transaction
    features: List[float] = Field(min_length=1)

    @field_validator("features", mode="before")
    @classmethod
    def reject_boolean_features(cls, value: Any) -> Any:
        if isinstance(value, list) and any(isinstance(item, bool) for item in value):
            raise ValueError("features must contain numbers, not booleans")
        return value

    @field_validator("features")
    @classmethod
    def require_finite_features(cls, values: List[float]) -> List[float]:
        if not all(math.isfinite(value) for value in values):
            raise ValueError("features must contain only finite numbers")
        return values


class ExplainPropertyResponse(BaseModel):
    transaction: Transaction
    feature_names: List[str]
    contributions: List[float]
    baseline_price_per_sqm: float
    prediction_price_per_sqm: float
    additivity_error: float
    model_sha256: str
    trained_at: str


class ReloadExplainModelsRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    transactions: List[Transaction] = Field(
        default_factory=lambda: ["sell", "rent"],
        min_length=1,
        max_length=2,
    )
    expected_model_sha256: Dict[Transaction, str]

    @field_validator("transactions")
    @classmethod
    def require_unique_transactions(
        cls,
        transactions: List[Transaction],
    ) -> List[Transaction]:
        if len(set(transactions)) != len(transactions):
            raise ValueError("transactions must be unique")
        return transactions

    @field_validator("expected_model_sha256")
    @classmethod
    def require_sha256_values(
        cls,
        hashes: Dict[Transaction, str],
    ) -> Dict[Transaction, str]:
        if not all(
            len(value) == 64
            and all(character in "0123456789abcdef" for character in value)
            for value in hashes.values()
        ):
            raise ValueError("expected model hashes must be lowercase SHA-256 values")
        return hashes

    @model_validator(mode="after")
    def require_matching_hash_keys(self) -> "ReloadExplainModelsRequest":
        if set(self.expected_model_sha256) != set(self.transactions):
            raise ValueError(
                "expected_model_sha256 keys must exactly match transactions"
            )
        return self


class ReloadedModel(BaseModel):
    transaction: Transaction
    model_sha256: str
    trained_at: str
    feature_count: int
    artifact_directory: str


class ReloadExplainModelsResponse(BaseModel):
    message: str
    models: List[ReloadedModel]


@app.post("/ReloadExplainModels", response_model=ReloadExplainModelsResponse)
def reload_explain_models(
    payload: ReloadExplainModelsRequest,
) -> ReloadExplainModelsResponse:
    try:
        summaries = explain_registry.reload(
            payload.transactions,
            payload.expected_model_sha256,
        )
    except (ArtifactValidationError, OSError, ValueError, xgb.core.XGBoostError) as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    return ReloadExplainModelsResponse(message="Success", models=summaries)


@app.post("/ExplainProperty", response_model=ExplainPropertyResponse)
def explain_property(payload: ExplainPropertyRequest) -> ExplainPropertyResponse:
    try:
        result = explain_registry.explain(payload.transaction, payload.features)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except LookupError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except (RuntimeError, xgb.core.XGBoostError) as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    return ExplainPropertyResponse(**result)


@app.post("/TrainAndExplain")
def train_and_explain(payload: TrainRequest):
    if len(payload.data) < 10:
        raise HTTPException(status_code=400, detail="Dataset must have at least 10 points.")

    df = pd.DataFrame(payload.data)
    uids = df["UID"].tolist() if "UID" in df.columns else [str(i) for i in range(len(df))]
    df = df.drop(columns=payload.drop_columns, errors="ignore")

    y = df[payload.target_column]
    X = df.drop(columns=[payload.target_column], errors="ignore")

    for column in payload.categorical_columns:
        if column in X.columns:
            X[column] = X[column].astype("category")

    numerical_columns = X.select_dtypes(include=["int64", "float64"]).columns
    scaler = StandardScaler()
    if len(numerical_columns) > 0:
        X[numerical_columns] = scaler.fit_transform(X[numerical_columns])

    model = xgb.XGBRegressor(
        max_depth=payload.max_depth,
        min_child_weight=payload.min_child_weight,
        learning_rate=payload.learning_rate,
        n_estimators=payload.n_estimators,
        enable_categorical=True,
        tree_method="hist",
    )
    model.fit(X, y)

    predictions = model.predict(X)
    r2 = float(r2_score(y, predictions))

    feature_importances = pd.DataFrame(
        {
            "feature": X.columns,
            "importance": float(0),
        }
    )
    feature_importances["importance"] = model.feature_importances_
    feature_importances = feature_importances.sort_values(
        "importance", ascending=False
    ).to_dict(orient="records")

    explainer = shap.TreeExplainer(model)
    shap_values = explainer.shap_values(X)
    base_value = (
        float(explainer.expected_value)
        if not isinstance(explainer.expected_value, np.ndarray)
        else float(explainer.expected_value[0])
    )

    examples = []
    for row_index in range(10):
        impacts = []
        for feature_index, column in enumerate(X.columns):
            value = float(shap_values[row_index, feature_index])
            if abs(value) > 0:
                impacts.append({"feature": column, "money_added": value})
        impacts = sorted(impacts, key=lambda item: abs(item["money_added"]), reverse=True)
        examples.append(
            {
                "uid": uids[row_index],
                "actual_price": float(y.iloc[row_index]),
                "predicted_price": float(predictions[row_index]),
                "feature_impacts": impacts,
            }
        )

    interaction_values = explainer.shap_interaction_values(X)
    mean_abs_interactions = np.abs(interaction_values).mean(axis=0)
    interactions_list = []
    feature_count = len(X.columns)
    for left in range(feature_count):
        for right in range(left + 1, feature_count):
            value = float(mean_abs_interactions[left, right])
            if value > 0:
                interactions_list.append(
                    {
                        "feature_1": X.columns[left],
                        "feature_2": X.columns[right],
                        "interaction_strength": value,
                    }
                )
    top_interactions = sorted(
        interactions_list,
        key=lambda item: item["interaction_strength"],
        reverse=True,
    )[:5]

    return {
        "r2_score": r2,
        "base_price": base_value,
        "feature_importances": feature_importances,
        "top_interactions": top_interactions,
        "examples": examples,
        "message": "Success",
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=5000)
