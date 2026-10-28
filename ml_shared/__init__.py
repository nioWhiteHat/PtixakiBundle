"""Shared contracts for durable XGBoost explainability artifacts."""

from .explain_artifacts import (
    ArtifactValidationError,
    LoadedExplainArtifact,
    PRICE_TARGET_NAME,
    PRICE_TARGET_UNIT,
    RESIDENCE_PROPERTY_PHASE2_FEATURES,
    atomic_save_xgboost_model,
    atomic_write_bytes,
    atomic_write_json,
    atomic_write_text,
    build_explain_manifest,
    explain_artifact_filenames,
    load_explain_artifact,
    sha256_file,
    utc_now_iso,
)

__all__ = [
    "ArtifactValidationError",
    "LoadedExplainArtifact",
    "PRICE_TARGET_NAME",
    "PRICE_TARGET_UNIT",
    "RESIDENCE_PROPERTY_PHASE2_FEATURES",
    "atomic_save_xgboost_model",
    "atomic_write_bytes",
    "atomic_write_json",
    "atomic_write_text",
    "build_explain_manifest",
    "explain_artifact_filenames",
    "load_explain_artifact",
    "sha256_file",
    "utc_now_iso",
]
