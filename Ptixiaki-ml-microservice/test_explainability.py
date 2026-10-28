from __future__ import annotations

from contextlib import redirect_stdout
import csv
import importlib.util
import io
import json
from pathlib import Path
import re
from types import SimpleNamespace
import sys
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import numpy as np
import pandas as pd
import xgboost as xgb
from fastapi import HTTPException


BUNDLE_ROOT = Path(__file__).resolve().parent.parent
if str(BUNDLE_ROOT) not in sys.path:
    sys.path.insert(0, str(BUNDLE_ROOT))

import main as service
from ml_shared.explain_artifacts import (
    ArtifactValidationError,
    atomic_save_xgboost_model,
    atomic_write_json,
    build_explain_manifest,
    explain_artifact_filenames,
    load_explain_artifact,
    sha256_file,
    utc_now_iso,
)
from ml_shared.shap_validation import ADDITIVITY_ATOL, ADDITIVITY_RTOL, additivity_tolerance


TREE_PATH = (
    BUNDLE_ROOT
    / "Ptixiaki"
    / "Worker"
    / "Models"
    / "TreeCode"
    / "Tree.py"
)
WORKER_DEPENDENCIES_AVAILABLE = importlib.util.find_spec("onnxmltools") is not None


def load_tree_module():
    tree_spec = importlib.util.spec_from_file_location("ptixiaki_tree", TREE_PATH)
    assert tree_spec is not None and tree_spec.loader is not None
    tree_module = importlib.util.module_from_spec(tree_spec)
    tree_spec.loader.exec_module(tree_module)
    return tree_module


def trained_booster(feature_count: int = 3) -> tuple[xgb.Booster, pd.DataFrame]:
    rng = np.random.default_rng(42)
    values = rng.normal(size=(24, feature_count)).astype(np.float32)
    columns = [f"f{index}" for index in range(feature_count)]
    features = pd.DataFrame(values, columns=columns)
    target = 125.0 + values[:, 0] * 8.0 - values[:, 1] * 3.0
    model = xgb.XGBRegressor(
        n_estimators=12,
        max_depth=2,
        learning_rate=0.2,
        tree_method="hist",
        random_state=42,
    )
    model.fit(features, target)
    return model.get_booster(), features


def write_artifact_pair(
    directory: Path,
    transaction: str,
    booster: xgb.Booster,
    feature_count: int,
) -> dict:
    model_file, manifest_file = explain_artifact_filenames(transaction)
    model_path = directory / model_file
    atomic_save_xgboost_model(booster, model_path)
    manifest = build_explain_manifest(
        transaction=transaction,
        feature_names=[f"Feature{index}" for index in range(feature_count)],
        encoded_feature_names=[f"f{index}" for index in range(feature_count)],
        categorical_mappings={},
        target_name="AskingPricePerSqm",
        target_unit="EUR_per_square_meter",
        trained_at=utc_now_iso(),
        model_sha256=sha256_file(model_path),
        row_count=24,
        training_r2=0.95,
        validation_r2=0.8,
        validation_fraction=0.2,
        hyperparameters={"random_state": 42},
    )
    atomic_write_json(directory / manifest_file, manifest)
    return manifest


class ExplainArtifactTests(unittest.TestCase):
    def test_additivity_tolerance_scales_with_cancelled_terms(self) -> None:
        self.assertAlmostEqual(float(additivity_tolerance(0, 1000, [-999.995])), 0.02099995)
        self.assertAlmostEqual(float(additivity_tolerance(10000, 1000, [9000])), 0.101)
        self.assertAlmostEqual(float(additivity_tolerance(0, 0, [0])), 0.001)
        self.assertAlmostEqual(float(additivity_tolerance(10000, 0, [1])), 0.101)
        np.testing.assert_allclose(
            additivity_tolerance([0, 10000], [1000, 1000], [[-999.995], [9000]]),
            [0.02099995, 0.101],
        )
        for invalid in (-1, np.nan, np.inf):
            with self.assertRaises(ValueError):
                additivity_tolerance(0, 0, [0], atol=invalid)
            with self.assertRaises(ValueError):
                additivity_tolerance(0, 0, [0], rtol=invalid)
        contract = (BUNDLE_ROOT / "Ptixiaki" / "Worker" / "explainability" / "contract.go").read_text(encoding="utf-8")
        for name, expected in (("AdditivityAbsTolerance", ADDITIVITY_ATOL), ("AdditivityRelTolerance", ADDITIVITY_RTOL)):
            value = re.search(rf"{name}\s*=\s*([0-9.e-]+)", contract)
            self.assertIsNotNone(value)
            self.assertEqual(float(value[1]), expected)

    def test_service_accepts_cancellation_noise_but_rejects_bad_explanations(self) -> None:
        feature_count = len(service.RESIDENCE_PROPERTY_PHASE2_FEATURES)
        booster, _ = trained_booster(feature_count)
        manifest = {
            "original_feature_names": list(service.RESIDENCE_PROPERTY_PHASE2_FEATURES),
            "model_feature_names": [f"f{i}" for i in range(feature_count)],
            "feature_count": feature_count, "model_sha256": "a" * 64,
            "trained_at": "2026-09-15T00:00:00Z",
        }
        artifact = SimpleNamespace(booster=booster, manifest=manifest)
        registry = service.ExplainModelRegistry()
        registry._models["sell"] = service.RegisteredExplainModel(artifact=artifact)
        with_bias = np.zeros((1, feature_count + 1), dtype=np.float32)
        with_bias[0, -1] = 1000
        with_bias[0, 0] = -999.995

        def predict(matrix, **kwargs):
            return with_bias if kwargs.get("pred_contribs") else np.array([0], dtype=np.float32)

        with patch.object(booster, "predict", side_effect=predict):
            registry._smoke_test(artifact)
            result = registry.explain("sell", [0] * feature_count)
            self.assertGreater(result["additivity_error"], ADDITIVITY_ATOL)
            self.assertEqual(result["contributions"][0], float(with_bias[0, 0]))
            with_bias[0, 0] = -999.5
            with self.assertRaisesRegex(RuntimeError, "additivity"):
                registry.explain("sell", [0] * feature_count)
            with self.assertRaisesRegex(ArtifactValidationError, "additivity"):
                registry._smoke_test(artifact)

    @unittest.skipUnless(WORKER_DEPENDENCIES_AVAILABLE, "Worker ONNX dependencies are unavailable")
    def test_worker_additivity_reports_the_actual_failing_property(self) -> None:
        tree = load_tree_module()
        booster, _ = trained_booster(2)
        features = pd.DataFrame(np.zeros((2, 2), dtype=np.float32), columns=["f0", "f1"])
        manifest = {
            "schema_version": 1, "transaction": "sell",
            "target": {"name": "AskingPricePerSqm", "unit": "EUR_per_square_meter"},
            "feature_count": 2, "original_feature_names": ["First", "Second"],
            "model_feature_names": ["f0", "f1"], "model_sha256": "a" * 64,
            "trained_at": "2026-09-15T00:00:00Z", "training_r2": 0.9, "validation_r2": 0.8,
        }
        args = SimpleNamespace(
            emit_shap_records=True, interaction_sample_size=2, interaction_sample_seed=42,
            shap_batch_size=2, shap_additivity_atol=ADDITIVITY_ATOL, shap_additivity_rtol=ADDITIVITY_RTOL,
        )
        # The first row's larger absolute error is within ITS allowance. Only
        # the second row fails, reproducing the misleading production message.
        with_bias = np.array([[9000.06, 0, 1000], [-999.96, 0, 1000]], dtype=np.float32)

        def predict(matrix, **kwargs):
            if kwargs.get("pred_interactions"):
                return np.zeros((2, 3, 3), dtype=np.float32)
            return with_bias if kwargs.get("pred_contribs") else np.array([10000, 0], dtype=np.float32)

        with patch.object(booster, "predict", side_effect=predict), redirect_stdout(io.StringIO()):
            with self.assertRaisesRegex(ValueError, "UID actually-failing-property") as caught:
                tree.emit_native_shap_records(
                    booster=booster, features=features, uids=["passing-property", "actually-failing-property"],
                    manifest=manifest, args=args,
                )
        error = float(re.search(r"error=([^,]+)", str(caught.exception))[1])
        allowed = float(re.search(r"allowed=([^,]+)", str(caught.exception))[1])
        self.assertGreater(abs(error), allowed)
        with_bias[1, 0] = -999.995
        output = io.StringIO()
        with patch.object(booster, "predict", side_effect=predict), redirect_stdout(output):
            tree.emit_native_shap_records(
                booster=booster, features=features, uids=["passing-property", "cancellation-property"],
                manifest=manifest, args=args,
            )
        records = [json.loads(line.split("\t", 1)[1]) for line in output.getvalue().splitlines()]
        self.assertEqual(sum(record["type"] == "property" for record in records), 2)

    def test_repository_feature_and_schema_contracts_are_aligned(self) -> None:
        dropped = {
            "UID",
            "TransactionType",
            "AskingPrice",
            "Distance",
            "IsRenovated",
            "AgeSq",
            "InteractionAgeDistance",
            "LogInteraction",
            "AdCreationAge",
            "ZoneValue",
            "GeoLat",
            "GeoLon",
        }
        for transaction in ("sell", "rent"):
            csv_path = (
                BUNDLE_ROOT
                / "Ptixiaki"
                / "Worker"
                / "DataCsv"
                / f"residence_properties_{transaction}_Phase2_training_data.csv"
            )
            columns = list(pd.read_csv(csv_path, nrows=0).columns)
            features = tuple(
                column
                for column in columns
                if column not in dropped and column != service.PRICE_TARGET_NAME
            )
            self.assertEqual(
                features,
                service.RESIDENCE_PROPERTY_PHASE2_FEATURES,
                f"{transaction} Phase2 CSV feature order drifted",
            )

        migration = (
            BUNDLE_ROOT
            / "Ptixiaki"
            / "Database"
            / "migrations"
            / "002_create_explainability_persistence.sql"
        ).read_text(encoding="utf-8")
        schema_dump = (BUNDLE_ROOT / "schema_dump.sql").read_text(encoding="utf-8")
        expected_tables = (
            "properties_analysis_sell",
            "properties_analysis_rent",
            "feature_importance_sell",
            "feature_importance_rent",
            "features_interactions_sell",
            "features_interactions_rent",
        )
        for table in expected_tables:
            self.assertIn(f"public.{table}", migration)
            self.assertIn(f"public.{table}", schema_dump)
        for transaction in ("sell", "rent"):
            table_start = migration.index(
                f"CREATE TABLE IF NOT EXISTS public.properties_analysis_{transaction}"
            )
            table_end = migration.index("\n);", table_start)
            table_definition = migration[table_start:table_end]
            impact_columns = re.findall(
                r"^\s+[a-z0-9_]+_impact_per_sqm double precision NOT NULL,?$",
                table_definition,
                flags=re.MULTILINE,
            )
            self.assertEqual(len(impact_columns), 30)
        for column in (
            "baseline_price_per_sqm",
            "training_r2",
            "validation_r2",
            "model_sha256",
            "model_trained_at",
        ):
            self.assertIn(f"ADD COLUMN IF NOT EXISTS {column}", migration)

    @unittest.skipUnless(
        WORKER_DEPENDENCIES_AVAILABLE,
        "the current interpreter does not have the Worker ONNX dependencies",
    )
    def test_phase2_cli_publishes_complete_activation_and_clean_stream(self) -> None:
        phase2_source = (
            BUNDLE_ROOT
            / "Ptixiaki"
            / "Worker"
            / "DataCsv"
            / "residence_properties_sell_Phase2_training_data.csv"
        )
        source_columns = list(pd.read_csv(phase2_source, nrows=0).columns)
        rows = []
        for row_index in range(12):
            row = {}
            for column_index, column in enumerate(source_columns):
                row[column] = float(row_index + column_index + 1)
            row.update(
                {
                    "UID": f"property-test-{row_index}",
                    "TransactionType": "sell",
                    "SubType": "apartment" if row_index % 2 else "maisonette",
                    "AdminName": "Synthetic Area",
                    "HeatType": "autonomous",
                    "HeatFrom": "fan_coil,heat_pump",
                    "EnergyPerformance": "A",
                    "AskingPricePerSqm": float(100 + row_index * 5),
                }
            )
            rows.append(row)

        dropped = (
            "UID,TransactionType,AskingPrice,Distance,IsRenovated,AgeSq,"
            "InteractionAgeDistance,LogInteraction,AdCreationAge,ZoneValue,GeoLat,GeoLon"
        )
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary = Path(temporary_directory)
            input_csv = temporary / "input.csv"
            json_directory = temporary / "json"
            tree_directory = temporary / "trees"
            mappings_directory = temporary / "mappings"
            pd.DataFrame(rows, columns=source_columns).to_csv(input_csv, index=False)

            command = [
                sys.executable,
                str(TREE_PATH),
                "--input_csv",
                str(input_csv),
                "--output_json_dir",
                str(json_directory),
                "--output_onnx_dir",
                str(tree_directory),
                "--output_mappings_dir",
                str(mappings_directory),
                "--tree_type",
                "Phase2",
                "--item_type_subtype",
                "residence_properties_sell",
                "--columns_dropped",
                dropped,
                "--categorical_cols",
                "SubType,AdminName,HeatType,HeatFrom,EnergyPerformance,PowerSupply",
                "--max_depth",
                "2",
                "--min_child_weight",
                "1",
                "--learning_rate",
                "0.2",
                "--n_estimators",
                "4",
                "--subsample",
                "1",
                "--colsample_bytree",
                "1",
                "--emit_shap_records",
                "--shap_batch_size",
                "4",
                "--interaction_sample_size",
                "5",
                "--interaction_sample_seed",
                "42",
            ]
            completed = subprocess.run(
                command,
                cwd=BUNDLE_ROOT / "Ptixiaki",
                check=True,
                capture_output=True,
                text=True,
            )

            stream_lines = completed.stdout.splitlines()
            self.assertTrue(stream_lines)
            self.assertTrue(
                all(line.startswith("SHAP_RECORD\t") for line in stream_lines),
                completed.stdout,
            )
            records = [json.loads(line.split("\t", 1)[1]) for line in stream_lines]
            meta = next(record for record in records if record["type"] == "meta")
            self.assertEqual(meta["property_count"], 12)
            self.assertEqual(meta["feature_count"], 30)
            self.assertEqual(meta["interaction_count"], 435)
            self.assertEqual(meta["interaction_sample_count"], 5)
            self.assertEqual(meta["interaction_sample_seed"], 42)
            progress = [record for record in records if record["type"] == "progress"]
            self.assertEqual(list(dict.fromkeys(record["stage"] for record in progress)), [
                "prepare", "validation", "training", "onnx", "artifacts",
                "shap_properties", "shap_interactions", "activate",
            ])
            for stage in dict.fromkeys(record["stage"] for record in progress):
                events = [record for record in progress if record["stage"] == stage]
                self.assertEqual(events[0]["completed"], 0)
                self.assertEqual(events[-1]["completed"], events[-1]["total"])
                self.assertTrue(all(event["elapsed_seconds"] >= 0 for event in events))
                self.assertEqual([event["completed"] for event in events], sorted(event["completed"] for event in events))
            self.assertEqual(sum(record["type"] == "property" for record in records), 12)
            self.assertEqual(sum(record["type"] == "importance" for record in records), 30)
            self.assertEqual(sum(record["type"] == "interaction" for record in records), 435)

            base_name = "Phase2_residence_properties_sell"
            expected_paths = [
                json_directory / f"{base_name}_PricePredictions.json",
                tree_directory / f"{base_name}_Tree.onnx",
                tree_directory / f"{base_name}_Tree.ubj",
                tree_directory / f"{base_name}_Manifest.json",
                mappings_directory / f"CategoryMappings_{base_name}.txt",
            ]
            self.assertTrue(all(path.is_file() for path in expected_paths))
            self.assertFalse(list(temporary.rglob("*.staged.*")))
            self.assertFalse(list(temporary.rglob("*.backup.*")))

            loaded = load_explain_artifact(tree_directory, "sell")
            self.assertEqual(loaded.manifest["feature_count"], 30)
            self.assertTrue(np.isfinite(loaded.manifest["training_r2"]))
            self.assertTrue(np.isfinite(loaded.manifest["validation_r2"]))
            self.assertEqual(loaded.manifest["interaction_sampling"], {
                "method": "uniform_without_replacement", "population_count": 12,
                "sample_count": 5, "seed": 42,
            })

            # Prove that ONNX prediction and the native UBJ used by TreeSHAP
            # represent the same fitted tree for real categorical/numeric rows.
            import onnxruntime as ort

            encoded = pd.DataFrame(rows)[
                loaded.manifest["original_feature_names"]
            ].copy()
            for column, mapping in loaded.manifest["categorical_mappings"].items():
                encoded[column] = encoded[column].astype(str).map(mapping)
            encoded_values = encoded.to_numpy(dtype=np.float32)
            native_predictions = loaded.booster.predict(
                xgb.DMatrix(
                    encoded_values,
                    feature_names=loaded.manifest["model_feature_names"],
                )
            )
            onnx_session = ort.InferenceSession(
                str(expected_paths[1]),
                providers=["CPUExecutionProvider"],
            )
            onnx_predictions = np.asarray(
                onnx_session.run(None, {"input": encoded_values})[0]
            ).reshape(-1)
            np.testing.assert_allclose(
                onnx_predictions,
                native_predictions,
                rtol=1e-4,
                atol=0.1,
            )
            with open(expected_paths[0], encoding="utf-8") as prediction_file:
                saved_predictions = np.asarray(
                    [
                        row["PredictedPricePerSqm"]
                        for row in json.load(prediction_file)
                    ]
                )
            np.testing.assert_allclose(
                onnx_predictions,
                saved_predictions,
                rtol=1e-4,
                atol=0.1,
            )

            with open(expected_paths[-1], newline="", encoding="utf-8") as mapping_file:
                mapping_rows = list(csv.reader(mapping_file))
            self.assertIn(["HeatFrom", "fan_coil,heat_pump", "0"], mapping_rows)

    def test_manifest_model_round_trip_and_hash_validation(self) -> None:
        booster, _ = trained_booster()
        with tempfile.TemporaryDirectory() as temporary_directory:
            directory = Path(temporary_directory)
            manifest = write_artifact_pair(directory, "sell", booster, 3)
            loaded = load_explain_artifact(directory, "sell")
            self.assertEqual(loaded.manifest["model_sha256"], manifest["model_sha256"])
            self.assertEqual(loaded.booster.feature_names, ["f0", "f1", "f2"])

            _, manifest_file = explain_artifact_filenames("sell")
            manifest["model_sha256"] = "0" * 64
            atomic_write_json(directory / manifest_file, manifest)
            with self.assertRaisesRegex(ArtifactValidationError, "SHA-256"):
                load_explain_artifact(directory, "sell")

    def test_registry_reload_is_atomic_and_explanation_is_additive(self) -> None:
        feature_count = len(service.RESIDENCE_PROPERTY_PHASE2_FEATURES)
        booster, features = trained_booster(feature_count)
        with tempfile.TemporaryDirectory() as temporary_directory:
            directory = Path(temporary_directory)
            model_file, _ = explain_artifact_filenames("sell")
            atomic_save_xgboost_model(booster, directory / model_file)
            sell_manifest = build_explain_manifest(
                transaction="sell",
                feature_names=service.RESIDENCE_PROPERTY_PHASE2_FEATURES,
                encoded_feature_names=[f"f{index}" for index in range(feature_count)],
                categorical_mappings={},
                target_name=service.PRICE_TARGET_NAME,
                target_unit=service.PRICE_TARGET_UNIT,
                trained_at=utc_now_iso(),
                model_sha256=sha256_file(directory / model_file),
                row_count=24,
                training_r2=0.95,
                validation_r2=0.8,
                validation_fraction=0.2,
                hyperparameters={"random_state": 42},
            )
            _, manifest_file = explain_artifact_filenames("sell")
            atomic_write_json(directory / manifest_file, sell_manifest)

            rent_model_file, rent_manifest_file = explain_artifact_filenames("rent")
            atomic_save_xgboost_model(booster, directory / rent_model_file)
            rent_manifest = dict(sell_manifest)
            rent_manifest.update(
                transaction="rent",
                model_file=rent_model_file,
                model_sha256=sha256_file(directory / rent_model_file),
            )
            atomic_write_json(directory / rent_manifest_file, rent_manifest)
            registry = service.ExplainModelRegistry([directory])
            expected_hashes = {
                "sell": sell_manifest["model_sha256"],
                "rent": rent_manifest["model_sha256"],
            }
            summaries = registry.reload(["sell", "rent"], expected_hashes)
            self.assertEqual([item["transaction"] for item in summaries], ["sell", "rent"])

            values = features.iloc[0].tolist()
            before = registry.explain("sell", values)
            reconstructed = before["baseline_price_per_sqm"] + sum(before["contributions"])
            self.assertAlmostEqual(
                reconstructed - before["prediction_price_per_sqm"],
                before["additivity_error"],
                places=7,
            )

            # Exercise the actual endpoint/Pydantic contracts, including the hash
            # handshake and invalid feature-vector rejection.
            previous_registry = service.explain_registry
            service.explain_registry = registry
            try:
                reload_response = service.reload_explain_models(
                    service.ReloadExplainModelsRequest(
                        transactions=["sell", "rent"],
                        expected_model_sha256=expected_hashes,
                    )
                )
                self.assertEqual(reload_response.message, "Success")
                explanation_response = service.explain_property(
                    service.ExplainPropertyRequest(
                        transaction="sell",
                        features=values,
                    )
                )
                self.assertEqual(
                    explanation_response.model_sha256,
                    expected_hashes["sell"],
                )
                with self.assertRaises(HTTPException) as invalid_context:
                    service.explain_property(
                        service.ExplainPropertyRequest(
                            transaction="sell",
                            features=values[:-1],
                        )
                    )
                self.assertEqual(invalid_context.exception.status_code, 400)
                with self.assertRaises(HTTPException) as hash_context:
                    service.reload_explain_models(
                        service.ReloadExplainModelsRequest(
                            transactions=["sell", "rent"],
                            expected_model_sha256={
                                "sell": "0" * 64,
                                "rent": expected_hashes["rent"],
                            },
                        )
                    )
                self.assertEqual(hash_context.exception.status_code, 503)
            finally:
                service.explain_registry = previous_registry

            # Invalidate one requested replacement. The already-live sell model
            # must remain available because the group swap never happens.
            rent_manifest["model_sha256"] = "f" * 64
            atomic_write_json(directory / rent_manifest_file, rent_manifest)
            with self.assertRaises(ArtifactValidationError):
                registry.reload(["sell", "rent"])
            after = registry.explain("sell", values)
            self.assertEqual(after["model_sha256"], before["model_sha256"])
            self.assertAlmostEqual(
                after["prediction_price_per_sqm"],
                before["prediction_price_per_sqm"],
            )

    @unittest.skipUnless(
        WORKER_DEPENDENCIES_AVAILABLE,
        "the current interpreter does not have the Worker ONNX dependencies",
    )
    def test_stream_contract_and_cardinalities(self) -> None:
        tree = load_tree_module()
        booster, features = trained_booster()
        feature_count = features.shape[1]
        manifest = {
            "schema_version": 1,
            "transaction": "sell",
            "target": {
                "name": "AskingPricePerSqm",
                "unit": "EUR_per_square_meter",
            },
            "feature_count": feature_count,
            "original_feature_names": ["First", "Second", "Third"],
            "model_feature_names": ["f0", "f1", "f2"],
            "model_sha256": "a" * 64,
            "trained_at": "2026-09-11T00:00:00Z",
            "training_r2": 0.95,
            "validation_r2": 0.8,
        }
        args = SimpleNamespace(
            emit_shap_records=True,
            interaction_sample_size=7,
            interaction_sample_seed=42,
            shap_batch_size=5,
            shap_additivity_atol=0.001,
            shap_additivity_rtol=0.00001,
        )
        uids = [f"property-{index}" for index in range(len(features))]
        output = io.StringIO()
        with patch.object(booster, "predict", wraps=booster.predict) as predict, redirect_stdout(output):
            tree.emit_native_shap_records(
                booster=booster,
                features=features,
                uids=uids,
                manifest=manifest,
                args=args,
            )
        interaction_rows = sum(
            call.args[0].num_row() for call in predict.call_args_list
            if call.kwargs.get("pred_interactions")
        )
        self.assertEqual(interaction_rows, args.interaction_sample_size)

        lines = output.getvalue().splitlines()
        self.assertTrue(lines)
        self.assertTrue(all(line.startswith("SHAP_RECORD\t") for line in lines))
        records = [json.loads(line.split("\t", 1)[1]) for line in lines]
        meta = next(record for record in records if record["type"] == "meta")
        self.assertEqual(meta["type"], "meta")
        self.assertEqual(meta["property_count"], len(features))
        self.assertEqual(meta["feature_count"], 3)
        self.assertEqual(meta["interaction_count"], 3)
        self.assertEqual(meta["interaction_sample_count"], 7)
        self.assertIn("training_r2", meta)
        self.assertIn("validation_r2", meta)

        properties = [record for record in records if record["type"] == "property"]
        importances = [record for record in records if record["type"] == "importance"]
        interactions = [record for record in records if record["type"] == "interaction"]
        self.assertEqual(len(properties), len(features))
        self.assertEqual(len(importances), 3)
        self.assertEqual(len(interactions), 3)
        self.assertTrue(all("feature_name" in record for record in importances))
        first_property = properties[0]
        reconstructed = meta["baseline_price_per_sqm"] + sum(
            first_property["contributions"]
        )
        self.assertAlmostEqual(
            reconstructed - first_property["prediction_price_per_sqm"],
            first_property["additivity_error"],
            places=7,
        )

        # Full-population contributions and importance are NOT sampled.
        expected_contributions = booster.predict(xgb.DMatrix(features), pred_contribs=True)[:, :-1]
        np.testing.assert_allclose([record["contributions"] for record in properties], expected_contributions)
        expected_abs = np.abs(expected_contributions.astype(np.float64)).mean(axis=0)
        for record in importances:
            index = record["feature_index"]
            self.assertAlmostEqual(record["mean_abs"], expected_abs[index])
            self.assertAlmostEqual(record["percentage"], expected_abs[index] / expected_abs.sum() * 100)

        # Compare with the independent native result on exactly the selected
        # rows, catching accidental division by the entire population count.
        selected = tree.interaction_sample_indices(uids, 7, 42)
        expected_interactions = booster.predict(xgb.DMatrix(features.iloc[selected]), pred_interactions=True).astype(np.float64)
        for record in interactions:
            pair = expected_interactions[:, record["feature_1_index"], record["feature_2_index"]]
            self.assertAlmostEqual(record["mean_abs"], np.abs(pair).mean())
            self.assertAlmostEqual(record["mean_signed"], pair.mean())
        property_progress = [record["completed"] for record in records if record.get("stage") == "shap_properties"]
        interaction_progress = [record["completed"] for record in records if record.get("stage") == "shap_interactions"]
        self.assertEqual(property_progress, [0, 5, 10, 15, 20, 24])
        self.assertEqual(interaction_progress, [0, 5, 7])

        # A small population uses every row, with the same correct denominator.
        args.interaction_sample_size = 2000
        full_output = io.StringIO()
        with redirect_stdout(full_output):
            tree.emit_native_shap_records(
                booster=booster, features=features, uids=uids, manifest=manifest, args=args,
            )
        full_records = [json.loads(line.split("\t", 1)[1]) for line in full_output.getvalue().splitlines()]
        full_meta = next(record for record in full_records if record["type"] == "meta")
        self.assertEqual(full_meta["interaction_sample_count"], len(features))
        full_interactions = booster.predict(xgb.DMatrix(features), pred_interactions=True).astype(np.float64)
        for record in full_records:
            if record["type"] == "interaction":
                pair = full_interactions[:, record["feature_1_index"], record["feature_2_index"]]
                self.assertAlmostEqual(record["mean_abs"], np.abs(pair).mean())
                self.assertAlmostEqual(record["mean_signed"], pair.mean())

    @unittest.skipUnless(WORKER_DEPENDENCIES_AVAILABLE, "Worker ONNX dependencies are unavailable")
    def test_interaction_sampling_is_bounded_and_reproducible(self) -> None:
        tree = load_tree_module()
        uids = [f"property-{index}" for index in range(100)]
        selected = tree.interaction_sample_indices(uids, 7, 42)
        self.assertEqual(len(set(selected)), 7)
        np.testing.assert_array_equal(selected, tree.interaction_sample_indices(uids, 7, 42))
        # PostgreSQL may return the same training properties in a different order.
        reversed_uids = list(reversed(uids))
        reverse_selected = tree.interaction_sample_indices(reversed_uids, 7, 42)
        self.assertEqual({uids[index] for index in selected}, {reversed_uids[index] for index in reverse_selected})
        self.assertNotEqual(set(selected), set(tree.interaction_sample_indices(uids, 7, 43)))
        np.testing.assert_array_equal(tree.interaction_sample_indices(uids, 100, 42), np.arange(100))
        np.testing.assert_array_equal(tree.interaction_sample_indices(uids, 2000, 42), np.arange(100))
        for invalid in (([], 7, 42), (uids, 0, 42), (uids, -1, 42), (uids, 7, -1)):
            with self.assertRaises(ValueError):
                tree.interaction_sample_indices(*invalid)

        # Keep the Go pipeline and standalone Python defaults aligned.
        contract = (BUNDLE_ROOT / "Ptixiaki" / "Worker" / "explainability" / "contract.go").read_text(encoding="utf-8")
        self.assertRegex(contract, rf"TrainingInteractionSampleSize\s*=\s*{tree.INTERACTION_SAMPLE_SIZE}\b")
        self.assertRegex(contract, rf"TrainingInteractionSampleSeed\s*=\s*{tree.INTERACTION_SAMPLE_SEED}\b")

    @unittest.skipUnless(
        WORKER_DEPENDENCIES_AVAILABLE,
        "the current interpreter does not have the Worker ONNX dependencies",
    )
    def test_artifact_activation_restores_previous_set_on_failure(self) -> None:
        tree = load_tree_module()
        with tempfile.TemporaryDirectory() as temporary_directory:
            directory = Path(temporary_directory)
            first_final = directory / "first.json"
            second_final = directory / "second.json"
            first_staged = directory / ".first.test.staged.json"
            missing_second_staged = directory / ".second.test.staged.json"
            first_final.write_text("old-first", encoding="utf-8")
            second_final.write_text("old-second", encoding="utf-8")
            first_staged.write_text("new-first", encoding="utf-8")

            with self.assertRaises(FileNotFoundError):
                tree.activate_staged_artifacts(
                    [
                        (first_staged, first_final),
                        (missing_second_staged, second_final),
                    ],
                    "test",
                )

            self.assertEqual(first_final.read_text(encoding="utf-8"), "old-first")
            self.assertEqual(second_final.read_text(encoding="utf-8"), "old-second")
            self.assertFalse(list(directory.glob("*.backup.*")))


if __name__ == "__main__":
    unittest.main()
