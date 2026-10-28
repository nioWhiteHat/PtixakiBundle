# Phase2 price explainability

This subsystem explains the same 30-value residence/property Phase2 vector that
the Go service sends to ONNX. Values are expressed in the model target unit,
EUR per square metre.

## Why there are two model files

- `Phase2_residence_properties_{sell|rent}_Tree.onnx` is the serving model used
  by Go for fast price prediction.
- `Phase2_residence_properties_{sell|rent}_Tree.ubj` is the same fitted XGBoost
  booster in XGBoost's native stable model format. The Python service loads it
  to calculate TreeSHAP values.
- `Phase2_residence_properties_{sell|rent}_Manifest.json` binds the UBJ file to
  its SHA-256, exact feature order, categorical encodings, target unit, training
  time, row count, metrics, and hyperparameters.

Only the current sell and rent artifact sets are kept. Temporary files are used
while training and are promoted to the fixed latest paths only after the full
explanation pass succeeds.

## Exact feature contract

The ordered vector is:

```text
SubType, AdminName, SquareMeters, Age, RenovAge,
NoView, ViewForest, ViewMountain, ViewSea, ViewUnlimited,
NoFacade, FacadeCorner, FacadeFrontage, FacadeInterior, FacadeThrough,
HeatType, HeatFrom, EnergyPerformance, WC, AC, Elevator, Parking,
StorageRoom, AluminumFrames, Bedrooms, Pool, Garden,
FloorLevel, FloorCount, GeoPremium
```

Categorical values are converted with the mappings produced by the same
training run. An unknown category is encoded as `-1`. Both Go and Python reject
a different order or feature count.

## Training workflow

```text
PostgreSQL/PostGIS
  -> Phase 1 CSV and Attr XGBoost model
  -> spatial GeoPremium calculation
  -> Phase 2 CSV
  -> deterministic validation model (R2 only)
  -> final Phase2 XGBoost model trained on all rows
       |-> ONNX prediction artifact
       |-> native UBJ + manifest
       `-> batched TreeSHAP stream to Go
             |-> one 30-contribution row per property
             |-> 30 global feature-importance rows
             `-> 435 global interaction-pair rows estimated from a sample
  -> one PostgreSQL transaction replaces the previous explanation snapshot
  -> Go reloads ONNX
  -> Python is asked to validate and atomically reload both UBJ/manifest pairs
```

The worker does not create a giant intermediate SHAP JSON file. It writes
compact `SHAP_RECORD` JSON lines to stdout; Go validates and stages them in
bounded batches. A malformed, incomplete, duplicate, non-finite, or
non-additive stream is rolled back, leaving the previous database snapshot.

## SHAP math

For a property `x`, the native XGBoost TreeSHAP result obeys:

```text
predicted_price_per_sqm(x) = baseline_price_per_sqm
                              + sum(feature_contribution_i(x))
```

A positive contribution raises the estimate above the baseline. A negative
contribution lowers it. There is one baseline per model, not one baseline per
feature. Every batch and every on-demand response is checked against this
identity within a small float32 tolerance.

The numerical allowance is cancellation-aware:

```text
allowed_error = 0.001 + 0.00001 * max(
    abs(prediction), abs(baseline) + sum(abs(feature contributions))
)
```

Native XGBoost returns float32 outputs. Large positive/negative terms can nearly
cancel to a small prediction, so scaling solely by that prediction wrongly
rejects harmless rounding. `ml_shared/shap_validation.py` shares this rule
between the Worker and Python API (including model-load smoke tests); Go uses
the same formula for streamed and on-demand explanations. Nothing is adjusted
to force equality: the raw baseline, prediction, feature impacts, and signed
error are preserved. Material inconsistencies and incorrect reported errors
still fail validation. A failing batch now names an actual violating property,
chosen by error minus its own allowance, rather than the largest absolute error
which may belong to a passing property. Restart both Go and the Python API after
this numerical-validation change so they agree on the contract.

## Interpreting the two R2 values

`validation_r2` is calculated by fitting a separate model on 80% of the input
rows and scoring the remaining 20%, using seed 42. `training_r2` is calculated
after fitting the final serving model on all rows and scoring those same rows.
The second is in-sample fit, not unseen-property accuracy, so it commonly looks
better. R2 is not the percentage of correctly priced properties.

The current Phase2 split is **not a leak-free evaluation of the whole pipeline**:
Phase1 fitting and target-derived GeoPremium features are calculated on the full
dataset before the Phase2 CSV is split. Held-out targets can therefore influence
those features indirectly. A reliable evaluation needs an outer split before
Phase1/GeoPremium construction, with validation features built using only
training-side information (and out-of-fold target-derived features for fitting).
Spatial or time-based holdouts should reflect the intended deployment. This is
a separate model-evaluation improvement; sampling and the numerical fix do not
change the current fitting or scoring design.

Global feature importance is based on mean absolute SHAP magnitude:

```text
importance_i = mean(abs(contribution_i))
importance_percentage_i = importance_i / sum(all importance) * 100
```

The signed mean is stored too, so the application can distinguish features
that generally raise prices from features that generally lower them.

Interactions use XGBoost's TreeSHAP interaction matrix. Only off-diagonal,
unique feature pairs are stored globally; with 30 features this is
`30 * 29 / 2 = 435` rows per model. Per-property interaction matrices are not
stored because they would create tens of millions of rows without helping the
main property explanation UI.

Ordinary per-property explanations and feature importance still use **every
training property**. Only global interactions are sampled: a uniform sample
without replacement of at most 2,000 properties per transaction, with seed 42.
Selection sorts by UID first, so reordering the same input properties does not
change sample membership. If there are fewer than 2,000 properties, all are used.
Each pair's signed and absolute means are divided by the actual sample count,
not the training population. All 435 pairs are still stored, but their averages
are sample estimates and can miss rare effects. Model fitting, R2, predictions,
Elo, and AddProperty's on-demand contributions are unchanged.

The Go pipeline sets `TrainingInteractionSampleSize` and
`TrainingInteractionSampleSeed` in `Worker/explainability/contract.go`, passed as
`--interaction_sample_size` and `--interaction_sample_seed`. Standalone Python
uses the same defaults. The manifest's `interaction_sampling` object records
method, population count, actual sample count, and seed; stream metadata carries
the sample count and seed too. The existing database tables need no new migration
for this change (migration 002 remains required).

## Training progress in the Go terminal

Python sends structured `progress` events over its existing stdout pipe. Go's
`Worker/explainability/progress.go` validates/formats these events and prints
timestamped `[TRAIN sell]` or `[TRAIN rent]` logs. This separates data preparation,
validation fitting, final fitting, ONNX conversion, artifact staging, all-row
property SHAP, sampled interactions, and artifact activation. For SHAP batches,
logs include processed rows, total rows, percentage, stage elapsed time, and an
approximate remaining time. Fast intermediate events are throttled to one log
per five seconds; starts and completions always print. Progress arrives after a
completed batch, not from inside an XGBoost calculation; a slow batch can still
leave a gap between updates. Non-batched stages report start and completion,
without invented percentages. Go also logs database staging, dependent
prediction/statistics/Elo persistence, publication, and successful commit.

Legacy Python diagnostics and errors still go to stderr, inherited by Go. The
new progress messages themselves are printed by Go, and progress records are
never persisted as SHAP data. A 100% SHAP stage is not a database commit; wait for
the publication success log and ultimately the `/train` response.

Restart/rebuild Go before the next `/train` (`go run .` from `Ptixiaki/`): older
running binaries do not understand the new progress record type. This update
does not change the Python API, so a service restart is not required just for
sampling/logging. It reduces interaction work, not the number of CPU threads;
high CPU usage during a calculation remains normal.

## On-demand property workflow

After AddProperty builds the exact Phase2 vector and gets the ONNX estimate, Go
posts the vector to `POST /ExplainProperty`:

```json
{
  "transaction": "sell",
  "features": [0, 12, 82, 22, 7]
}
```

The shortened array above is illustrative; a real request must contain all 30
ordered finite values. The service chooses the in-memory sell or rent booster,
calculates native prediction plus contributions, checks additivity, and returns
the model hash and training time. Go verifies feature order, additivity, and
agreement between the native and ONNX predictions before upserting the proper
sell/rent property-analysis table.

Explanation is deliberately best-effort during AddProperty: a temporarily
unavailable Python service does not undo a successfully persisted property,
estimate, and Elo. The next full training run backfills its explanation.

## Persistence

- `properties_analysis_sell` / `properties_analysis_rent`: one latest row per
  property with 30 signed contribution columns, baseline, explained prediction,
  additivity error, model hash, and explanation time.
- `feature_importance_sell` / `feature_importance_rent`: 30 latest global rows.
- `features_interactions_sell` / `features_interactions_rent`: 435 latest global
  pair rows.
- `global_stats`: the existing `Price_stats_sell` and `Price_stats_rent` rows
  also hold the latest baseline, training R2, validation R2, model hash, and
  model training time.

Apply `Database/migrations/002_create_explainability_persistence.sql` before
running `/train` against an existing database. The worker performs an early
schema preflight and fails with that migration hint if the contract is absent.

## Python model service

- `POST /ReloadExplainModels` validates hashes, manifests, feature contracts,
  model dimensions, and a TreeSHAP smoke test before swapping the in-memory
  registry. Go sends the exact freshly published sell/rent UBJ hashes in
  `expected_model_sha256`; a different or misconfigured model root is rejected
  before the registry swap.
- `POST /ExplainProperty` explains one exact model-ready vector.
- `POST /TrainAndExplain` remains available for the separate market-analysis
  workflow.

Set `PTIXIAKI_MODEL_ROOT` when the service cannot discover
`Ptixiaki/Worker/Models/TreeOutputs` from its source location or working
directory.
