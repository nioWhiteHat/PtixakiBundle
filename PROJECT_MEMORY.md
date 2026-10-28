# Ptixiaki project memory

Last architecture update: 2026-10-09

This is durable repository context for future work. It records what the bundle currently does, not what it is intended to become. Verify the source touched by a task and update this file when a contract changes.

## Scope reviewed

The review covered all first-party source and configuration:

- 40 Go files (about 4,700 lines) under `Ptixiaki/`.
- `Ptixiaki-ml-microservice/`, `ml_shared/`, and `Ptixiaki/Worker/Models/TreeCode/Tree.py`.
- React/Vite source, CSS, and build configuration under `Ptixiaki-frontend/Ptixiaki-Frontend/`.
- The complete `schema_dump.sql` PostgreSQL schema.
- Dependency manifests, example request files, and the names/formats of generated data/model artifacts.

Third-party internals in the committed `node_modules/` and `venv/` were not treated as application code. Generated frontend `dist/`, CSV contents, prediction/GeoPremium JSON bodies, ONNX binaries, the executable, and the ONNX Runtime DLL were inventoried and sampled for their contracts rather than line-reviewed as source.

The workspace root currently has no Git metadata. The apparent roughly 18,000-file bundle is mostly vendored dependencies; first-party code is roughly 9,200 lines including SQL and styles.

## System at a glance

```text
React/TypeScript/Vite auth, portfolio/details, area analysis
        /register/*, /User/*, /analysis/* via same-origin Vite proxy
                    |
Client -> Go HTTP service :8080 -> PostgreSQL/PostGIS
                 |     |  -> cached administrative-area search + Nominatim address lookup
                 |     `-> FastAPI :5000 market analysis + cached UBJ explanations
                 `-> ONNX Runtime + Phase2 model/mapping artifacts

Offline/on-demand training:
PostgreSQL -> Go feature extraction -> CSV -> Tree.py/XGBoost Attr model
  -> in-sample residual JSON -> spatial GeoPremium -> Phase2 CSV/model
  -> ONNX + UBJ + manifest + SHAP stream -> estimates/stats/Elo/explanation tables
```

The Go process is both HTTP API and ML orchestration host. It directly imports Worker packages, owns a PostgreSQL pool, loads native ONNX sessions, and shells out to Python for retraining. FastAPI still supports area-analysis training, and now also keeps the latest native XGBoost sell/rent models in a thread-safe in-memory registry for per-property TreeSHAP explanations. The Property Co Pilot frontend connects auth, property browsing, portfolio, property detail, area search and area analysis to Go. It never calls Python directly.

## Development caches and temporary outputs

Setup policy updated 2026-10-28: the user requested removal of the bundle-root `.gocache/` and `.cache/` directories and does not want them recreated. These contained Go build/test cache entries, temporary development executables and development logs; application source and trained model artifacts do not depend on them.

Use Go's default cache outside the bundle (`%LOCALAPPDATA%\go-build`). If the default is not writable, choose a writable operating-system temporary directory outside the bundle. Keep temporary development executables and redirected logs outside the bundle as well, normally under `%TEMP%\ptixiaki-dev`. Do not use any bundle-local `.gocache/` or `.cache/` paths in future commands. Existing references in historical audit notes describe past runs only.

## Component map

### Go composition root and API

`Ptixiaki/main.go` opens the database, initializes ONNX Runtime from `onnxengine.dll`, loads two fixed Phase2 residence/property models, registers handlers on Go's default `ServeMux`, and listens without TLS or server timeouts on port 8080. Configuration, URLs, paths, model shapes, and ports are hard-coded.

Current routes:

| Route | Auth | Current behavior |
| --- | --- | --- |
| `/predict/sell?uid=...` | No | Predicts one existing property using the sell Phase2 ONNX model; plain-text response. The rent route is commented out. |
| `/train` | No | Serially retrains residence/properties sell and rent, validates/persists complete SHAP datasets, reloads both Python UBJ models, then atomically swaps both Go ONNX predictors. Expensive and mutating; failures return HTTP 500 while the previous active files/tables/sessions remain available where publication has not committed. |
| `/register/SignUp` | No | Accepts email, password, and caller-controlled `admin`; inserts legacy credentials and sets a JWT cookie. |
| `/register/SignIn` | No | Checks a bcrypt password in `user_credentials` and sets a JWT cookie. |
| `/User/AddProperty?transaction=...` | JWT cookie | POST-only. Validates inputs and requires AddressOSMType/AddressOSMID from Nominatim address search; verifies the selected canonical address/coordinates through the server cache or Nominatim lookup. Asking per m² may be omitted/null. Persists, predicts and computes Elo plus best-effort SHAP. Returns HTTP 201 with `{"message":"Property added successfully","uid":"property_..."}` and a Location header for its owner-only view. |
| `/User/ViewProperty?uid=...` | JWT cookie | GET full stored property inputs, nullable estimates, optional asking/image, Elo/GeoPremium, local SHAP contributions/percentages and global importance. Owner-only (unknown/other-user UID => 404), read-only repeatable-read DB snapshot. Legacy POST `{"PropertyUid":"..."}` remains accepted; response uses the new snake_case outer contract. |
| `/User/BrowseProperties` | JWT cookie | GET residence listings; optional `min_lat,max_lat,min_lon,max_lon` (omit all for all areas), `transaction` (default sell), one-based `index` (default 1), inclusive min/max construction year, renovation year, estimated/asking price per m², square metres and Elo. At most 20 rows plus `has_next`, highest Elo first with UID tie-breaker; portfolio-like overviews with explicit nullable image. |
| `/User/BrowseProperty?uid=...` | JWT cookie | GET the selected residence listing's full existing detail/SHAP contract without portfolio ownership; missing image is explicitly null. Same read-only snapshot reader, no Python call or writes. |
| `/User/Portfolio` | JWT cookie | GET-only portfolio summary: property count, total estimated value, and per-property overview fields including nullable image source. |
| `/User/DeleteProperty?property_uid=...` | JWT cookie | DELETE-only; removes only the authenticated user's matching `user_properties` ownership row and leaves the underlying property and derived records intact. |
| `/test` | No | Imports fixed model artifacts into database tables; returns an empty response and can terminate the process. |
| `/analysis/autocomplete?q=...` | JWT cookie | Searches stored administrative boundaries, with accent-insensitive Greek/Latin matching and common city aliases. `kind=address` submits the entered address to Nominatim and returns canonical display_name, lat/lon, osm_type/osm_id and address details. Address search is explicitly submitted with Enter/Find address; the public provider prohibits search-as-you-type. Administrative search still uses the one-hour database catalogue. |
| `/analysis/CreateAnalysisForArea` | JWT cookie | POST bbox; clusters listings and asks Python for sell/rent explanations. Optional `warnings` preserves partial results when a group is too small or Python fails. |

Handlers do not restrict HTTP methods, request content types, or body sizes. There is no health route, graceful shutdown, rate limiting, recovery middleware, request timeout policy, or stable shared API schema. Several response types serialize with mixed PascalCase and snake_case/camel-like naming.

### Authentication and identity

`Backend/Auth` uses bcrypt plus an HMAC JWT stored in an HttpOnly, Secure, SameSite=Strict cookie. The signing key is a trivial literal duplicated in issue/verify code. Verification accepts any HMAC signing method rather than exactly HS256 and has no issuer, audience, revocation, or server-side session state. The token's admin claim is never used for authorization.

Signup validates DNS MX records synchronously, requires a password longer than eight characters, and then limits it to ASCII alphanumerics. Most importantly, the public request chooses its own admin flag. The Secure cookie will not be sent over the service's own plaintext HTTP endpoint during local browser development.

The code writes the legacy `user_credentials` table. The schema also has the newer `users` plus `auth_providers` design, but this bundle does not bridge them. `user_credentials.email` is not unique in the dump. This split identity model conflicts with foreign keys used by newer business, billing, feedback, and subscription tables.

### Add/view property flow

`Backend/Api/UserActions` performs this sequence:

1. Ignore caller-provided UID/geography/zone fields. Require AddressOSMType/AddressOSMID from address search, resolve the selected canonical address through the server-side Nominatim cache or stable OSM-object lookup, and use those provider coordinates. A new street does not need an existing property in the database.
2. Copy the zone value from the geographically nearest existing property, without a country, radius, or property-type constraint.
3. In one transaction, insert `properties`, `base_attributes`, `building_details`, `property_geo_data`, `residential_specifics`, and `user_properties`, then commit.
4. After the commit, calculate a training-compatible GeoPremium from up to 50 eligible phase-one neighbor residuals within 300 metres, prepare the exact 30-feature inference vector, predict, insert `estimation_data`, calculate/update Elo, insert `properties_elo`, and best-effort request/persist a native TreeSHAP explanation for that same vector.

The derived work is not atomic with the base inserts. Any inference/comparable/statistics/Elo failure returns an error while leaving a half-created listing. SHAP failure is deliberately nonfatal and a later full train backfills it. Retrying creates another ULID, and the API now returns the created UID alongside the existing success message so the frontend can open the owner-only detail page. The transaction query is now restricted to exact `sell` or `rent`. The detailed view path enforces ownership through `user_properties`.

The portfolio endpoint reads per-square-metre asking/estimated values from `estimation_data`, sums `predicted_price_per_sqm * square_meters` for `total_estimated_value`, and returns nullable values when derived data is absent. It reads the first ordered image from `property_images`; a URL is returned unchanged and binary bytes become a browser-ready base64 `data:` URL. Missing images (including when migration 001 is absent) are now omitted from both portfolio and detail JSON, rather than `image: null`.

Property-detail/optional-asking follow-up (2026-09-15): `PropertyFeatures.AskingPricePerSqm` is `*float64`; omission/null remains SQL NULL in `properties.asking_price`, `estimation_data.asking_price_per_sqm` and `residual_percentage`. Explicit zero/negative/nonfinite prices are rejected, along with invalid required fields/flags, inconsistent renovation input and integer-overflowing totals. AddProperty accepts one JSON object (1 MiB limit), rejects unknown fields and returns the created UID alongside its success message (updated 2026-10-09). Training/comparable queries require positive asking labels. `BuildInferenceQuery` removes only asking-price eligibility filters; its COALESCE to zero exists solely to scan the legacy internal struct and is dropped from the model vector, never saved as an invented asking price. A nil residual skips global/local deal terms and their queries: Elo = `10000 + 3000*GeoPremiumZ`; priced properties keep the existing three-term formula. Residual comparables exclude unknown/zero-priced records. Existing schema columns already allow NULL: no migration required.

`Backend/Api/UserActions/PropertyDetails.go` replaces the old incomplete detail response. `features` contains persisted inputs under their existing PascalCase names plus available geography; outer fields include `uid`, `transaction`, per-m² and total estimate/asking, integer Elo, GeoPremium/z-score and nullable residual. `explanation` is null if absent, otherwise contains signed contributions with local absolute-magnitude percentages, baseline, explained prediction, additivity error, model hash/time and a flag comparing the importance model. `feature_importance` contains the transaction's global mean-absolute/signed SHAP and percentage/rank/hash/time rows. These global percentages are distinct from per-property percentages. Missing asking and image keys are omitted. Read endpoint does not generate missing explanations or call the Python API. Known limitation: unpriced properties are excluded from supervised retraining; a later full explanation-table replacement can remove their online explanation, leaving it unavailable until refreshed. Existing normalized storage cannot reconstruct unknown historical inputs; missing scalar attribute values retain legacy zero/empty defaults. Images are read, not uploaded, by these routes.

Portfolio property objects explicitly include `uid`. Removing one uses `DELETE /User/DeleteProperty?property_uid=...`; the SQL predicate includes both the JWT user UID and property UID, returns 404 when that association is absent, and deliberately does not delete from `properties`, feature, estimation, image, or Elo tables.

Property browsing (updated 2026-09-27): `Backend/Api/UserActions/BrowseProperties.go` provides a parameterized spatial query and overview loader. Both browse routes cover residence apartments/detached houses/maisonettes with sell/rent transactions, including user-added properties, across the dataset rather than only owned properties. This broader authenticated read access is intentional; no owner identity or ownership links are exposed. The old owner-only `/User/ViewProperty` behavior remains unchanged. List filtering uses `geo.geom && ST_MakeEnvelope(min_lon,min_lat,max_lon,max_lat,4326)`; omitting all four coordinates uses world bounds (properties still need geolocation), while a partial bbox is rejected. Optional `_min`/`_max` parameters for `construction_year`, `renovation_year`, `estimated_price_per_sqm`, `elo`, `asking_price_per_sqm`, and `square_meters` combine with AND. Prices are per m²; asking filters reuse the overview expression, including total-asking/area fallback. Ranges are inclusive and missing values do not match. Basic parsing/bbox/index checks only; inverted ranges return an empty array. Sorting is `e.elo DESC NULLS LAST, p.uid ASC`; offset is `(index-1)*20`. LIMIT 21 provides a lookahead; at most 20 rows are returned with `has_next`, avoiding a total-count query. Response is `{index,page_size,has_next,properties}` with overview fields plus construction/renovation years. Primary images use the existing batched loader for the returned 20 rows only; absent images are `image:null` without changing older portfolio/detail omission rules. The shared detail reader distinguishes owner-scoped and browse-scoped queries. Stored SHAP is reused, never recomputed. No migration, dependency, model, or Python change is needed. `Backend/Api/UserActions/BROWSING.md` documents requests and tests.

### Market analysis

`Backend/Api/MarketAnalysis` modifies Worker SQL using exact string replacements to add bounding-box filters, then performs standardized k-means and per-cluster calculations. It re-fetches/encodes selected residence comparables and calls `http://localhost:5000/TrainAndExplain` for sell/rent groups with at least ten modeled rows. HTTP calls now carry the request context and a four-minute timeout. Fewer than ten rows or a Python failure yields a per-transaction warning rather than discarding clusters and the other model. Empty areas return empty arrays without calling Python. Go logs pipeline stages and upstream failures; HTTP errors use safe codes. The area encoder now fills Elevator from the stored field.

The query/JSON manipulation is brittle, geographical coordinates are fetched but not populated into returned points, and numeric type assertions can silently become zero. K-means now uses distinct farthest-point seeds, retains empty centers and caps iterations at 100; identical rows terminate as a single group. Seeds are deterministic for a given input order, though database/map order remains unspecified. Year 2026 is still hard-coded in feature encoding. Cancelling HTTP does not guarantee stopping Python's already-running CPU computation.

`AreaSearch.go` loads real `admin_boundaries` names and bounding boxes (levels 6–10) into an application-scoped catalogue, refreshed after one hour. It ranks municipalities first, normalizes Greek accents/transliteration and supports common English aliases including Piraeus/Pireaus. The existing `display_name`, `lat`, `lon`, `boundingbox` contract is retained. Coverage is limited to stored boundaries; arbitrary English translations/misspellings are not promised. Administrative-area autocomplete does not call Nominatim. Address mode uses the existing Geo helper, extended with Search and Resolve: a 12-second HTTP timeout, application User-Agent, one external request per second and a bounded one-hour cache (256 entries). Search results populate stable OSM-object lookup entries so saving normally reuses exactly the selected coordinates without another provider request. NOMINATIM_URL can override the default https://nominatim.openstreetmap.org base URL. The UI credits Nominatim/OpenStreetMap and explicitly submits address requests in accordance with the public provider’s prohibition on autocomplete.

`Backend/Api/MarketAnalysis/MLR.go` is empty.

### Worker training pipeline

`Worker/Pipelines/TrainingPipeline.go` implements:

1. `training.GenerateTrainingData`: build SQL using `Worker/database/QueryBuilder.go`, collect typed rows, engineer features, and write the Attr/phase-one CSV.
2. Run `Worker/Models/TreeCode/Tree.py` using `python` from `PATH`.
3. `training.CalcGeoPremium`: for each phase-one residual, average up to 50 neighbor residuals within 300 metres and calculate a population standard deviation/z-score.
4. Merge `GeoPremium` into a Phase2 CSV and run the same Python trainer again.
5. For residence/properties Phase2 jobs, stream strict SHAP metadata, 1 property record per row, 30 full-population global importance records, and 435 unique global interaction-pair records estimated from at most 2,000 sampled properties per transaction into transaction-local staging tables. Structured progress events in the same stdout pipe are validated/formatted/logged by Go, not stored in the database.
6. Insert estimation rows/global statistics and calculate Elo, then atomically replace the transaction-specific explanation/importance/interaction tables.

Supported artifact segments are the Cartesian product of:

- `residence_properties`, `residence_building`, `landplot_landplot`, `landplot_agricultural`, and `commercial_default`.
- `sell` and `rent`.

Each of the ten combinations has Attr and Phase2 CSV/prediction/mapping/ONNX artifacts; Phase2 adds GeoPremium as its last feature. Feature counts are:

| Segment | Attr | Phase2 |
| --- | ---: | ---: |
| residence properties | 29 | 30 |
| residence building | 15 | 16 |
| landplot landplot | 23 | 24 |
| landplot agricultural | 21 | 22 |
| commercial default | 29 | 30 |

`main.go` only loads the two 30-feature residence-properties Phase2 models. Attr predictor fields are never loaded, and `PredictPricePipeline` ignores its Attr predictor argument.

Artifact naming is a hard contract:

- CSV: `{item}_{subtype}_{transaction}_training_data.csv`, with `_Phase2_` before `training_data` for phase two.
- Prediction: `{Attr|Phase2}_{item}_{subtype}_{transaction}_PricePredictions.json`.
- ONNX: `{Attr|Phase2}_{item}_{subtype}_{transaction}_Tree.onnx`.
- Mapping: `CategoryMappings_{Attr|Phase2}_{item}_{subtype}_{transaction}.txt`.
- Geo: `GeoPrem_{item}_{subtype}_{transaction}.json`.
- Native XGBoost: `Phase2_residence_properties_{transaction}_Tree.ubj`.
- Explainability manifest: `Phase2_residence_properties_{transaction}_Manifest.json`.

The latest artifact set is authoritative; old revisions are not retained. Phase2 publication stages ONNX/UBJ/manifest outputs and restores the prior complete set if activation fails. `main.go` serializes `/train`, preloads and smoke-tests both new ONNX files, asks FastAPI to atomically reload sell and rent native models, and only then swaps the Go predictors under an RW mutex.

The current bundle contains 20 CSVs, 20 ONNX models, 20 mapping files, 20 prediction JSONs, 10 GeoPremium JSONs, and two unused standardization JSONs. Paths are relative to the process working directory and outputs are overwritten directly with no lock or atomic swap.

### Feature engineering and inference

Database structs are in `Worker/types/DatabaseTypes.go`, model feature structs in `ModeledTypes.go`, and bulk preprocessing in `TypeConverter.go`. `testing/Convert.go` duplicates much of the conversion logic and can drift.

Important semantics:

- Residence/commercial age and renovation interactions use a literal year 2026. `ParseMonthsSince` uses a frozen date of 2026-05-04, so time-based features are already stale.
- Views/facades become indicator flags; categories such as heat, source, energy class, subtype, admin area, and power supply use saved integer mappings.
- Missing booleans become false/zero. Several missing numerics also become zero rather than an explicit missing marker.
- Residence `Elevator` is now selected and populated consistently in bulk and single-property vectors. All six land `Angle*` fields remain defined but unpopulated.
- Category mappings are now written/read as quoted CSV, including categories containing commas.
- XGBoost is trained on integer category codes as continuous numeric features, imposing arbitrary ordinal meaning.
- Residence/property Phase2 inference and explanation share an explicit 30-feature ordered contract. The manifest stores original/model feature names, target name/unit, transaction, model hash, training/validation R², and training time; Python refuses mismatched manifests. The SHAP baseline is derived from native `pred_contribs`, streamed into `global_stats`, and recomputed by the same native model for on-demand explanations.
- ONNX uses names `input` -> `variable`. A predictor reuses one input/output tensor. Its mutex serializes calls, but `copy` does not reject long/short vectors or clear unused tail values.

Online residence/property inference calculates GeoPremium using the same core definition as training: the mean and population standard deviation of up to 50 eligible phase-one residuals within 300 metres. It derives those neighbor residuals from `asking_price_per_sqm` and `attr_price` in `estimation_data`; newly added properties have no `attr_price`, so they do not become GeoPremium comparables until an offline training run populates it. The training subtype mapping remains wrong for commercial/default and landplot/agricultural versus their raw database subtype values. Existing artifacts show zero neighbors/constant-zero premiums for all four affected sell/rent combinations.

### Python tree trainer

`Worker/Models/TreeCode/Tree.py` validates the input, creates a deterministic seed-42 validation split, reports held-out validation R², retrains the final XGBoost model on all valid rows, and reports full-data training R². It writes prediction JSON/mappings/ONNX as before. For residence/property Phase2 it additionally writes native UBJ plus a hash-bound manifest and emits newline-delimited `SHAP_RECORD` JSON to stdout in bounded batches.

TreeSHAP property contributions are native `pred_contribs`: `prediction = baseline + sum(feature impacts)` in price-per-square-metre units. The trainer checks this additivity for every property and computes global feature importance over the full dataset. It runs the much more expensive native `pred_interactions` in a separate pass over a uniform, without-replacement sample of at most 2,000 properties per sell/rent model (seed 42, sorted by UID before selection for input-order stability). Datasets at/below the cap use all rows. All 435 unordered pair means are still stored, but now divide by the sample count and are sample estimates. Manifest `interaction_sampling` records method/population/sample/seed, and stream metadata records sample count/seed. Go constants `TrainingInteractionSampleSize` / `TrainingInteractionSampleSeed` pass the matching CLI arguments; Python standalone defaults match. No database migration is added for this sampling change. Residuals remain `(actual - predicted) / actual * 100`; the online path uses the same sign. Cross-validation/early stopping are still not implemented, and final training R² remains in-sample; validation R² is the generalization indicator.

`Worker/explainability/progress.go` prints `[TRAIN sell|rent]` progress from structured Python stdout events for preparation, validation, fitting, ONNX, staging, all-row SHAP, sampled interactions, and activation. Batched SHAP shows actual processed/total rows, percent, elapsed time, and approximate ETA, throttled to five seconds between intermediate updates while always showing start/completion. A batch must finish before its update; there is no inside-XGBoost heartbeat or CPU cap. Go also logs database staging/prediction-statistics-Elo persistence/publication/commit. Legacy stderr diagnostics remain inherited. Rebuild/restart Go before running the new script: the old parser rejects `progress` records. Python API and on-demand explanations are unchanged.

Numerical-validation follow-up (2026-09-15): native float32 SHAP terms can nearly cancel, so `ml_shared/shap_validation.py` and Go's `shapAdditivityTolerance` now use `0.001 + 1e-5 * max(abs(prediction), abs(baseline) + sum(abs(contributions)))`. This applies to offline streaming, Go's online response checks, the Python API and reload smoke tests. Predictions/contributions/baselines are never adjusted or clamped; mismatches, wrong reported errors, and overflow are still rejected. The Worker failure message selects the greatest positive excess over row-specific tolerance (the old max-absolute-error selection could name a passing row). Restart both Go and the Python API for this change. A read-only audit of the existing staged sell model across all 123,449 current CSV rows found two failures under the old rule, none under the corrected rule, max absolute reconstruction difference 0.02434754 EUR/m², and 24 negative native predictions. Negative prices are a separate, unresolved modeling issue; no production artifacts or DB rows were published by this audit.

R² evaluation caveat: validation R² fits a separate model on 80% and scores the remaining 20%; final training R² fits/scores all rows and is not an unseen-property accuracy estimate. Phase2 GeoPremium is computed from target-derived Phase1 residuals on the full population before that split, so even validation R² is potentially optimistic due to leakage. Clean end-to-end evaluation needs an outer split before Phase1/GeoPremium construction, training-only validation comparables, and out-of-fold target-derived training features; consider spatial/time holdouts to match intended deployment. This evaluation design was diagnosed but not changed by the numerical fix.

### Elo/statistics persistence

`InsertTrainingData.go` joins artifact rows by UID but silently substitutes zero when a phase is missing. It queues a very large pgx batch; the persistence path now consumes every result and closes the batch before statistics/Elo begin, but the operation is still not wrapped in one transaction. Global statistic names contain only `sell` or `rent`, not the model segment. Their writes upsert, which makes repeated residence training idempotent but means training a different sell/rent model group overwrites the same statistics. Training and online Elo now share one scoring function: `10000 - 3000*globalResidualZ - 2000*localResidualZ + 3000*geoPremiumZ`. Both use floating-point population statistics, safe zero-standard-deviation handling, the training residual sign, and up to 20 residence/property neighbors within 150 metres. The neighbor query is still hard-coded to residence/property data for land/commercial jobs. Go computes float Elo while `properties_elo.elo` is an integer column; pgx converts the value through its integer wrapper, truncating fractional precision.

The online Elo path reads the new property's own stored GeoPremium instead of copying another property's value. Row scan/iteration errors are checked, and its local sample no longer contains a zero-filled prefix. `Worker/Pipelines/CalculateEloForProps_test.go` verifies scoring direction, residual sign, decimal-preserving statistics, and zero-standard-deviation behavior. `go test ./...` passes as of 2026-09-11.

### FastAPI explainability service

`Ptixiaki-ml-microservice/main.py` preserves POST `/TrainAndExplain` for market analysis and adds POST `/ReloadExplainModels` plus POST `/ExplainProperty`. On startup/reload it validates the sell/rent manifest hashes, exact 30-feature order, target contract, and native UBJ models, then atomically replaces its in-memory registry. Go includes the exact freshly published UBJ hashes in reload requests, so a misconfigured Python model root is rejected before the swap. `/ExplainProperty` accepts transaction plus the exact ordered numeric vector and returns baseline, 30 named per-m² contributions, native prediction, additivity error, model hash, and timestamp. Go also compares the native prediction with ONNX within a tight absolute/relative tolerance before persisting the row.

The new cached explanation endpoints have strict validation and registry locking, but the service still has no authentication/request-size/concurrency limits and binds `0.0.0.0:5000`. The older `/TrainAndExplain` remains resource-sensitive and presents in-sample R².

`requirements.txt` pins 74 packages although the service imports only FastAPI/Pydantic, pandas, NumPy, XGBoost, SHAP, scikit-learn, and Uvicorn. The committed Python 3.14 venv was created at another absolute path; installed versions differ from the manifest (including NumPy, XGBoost, SHAP, and pandas), and some listed ONNX packages are absent. The Worker invokes a separate system `python`, so the two Python paths are not one reproducible environment.

### Frontend

As of 2026-09-27, the frontend is branded **Property Co Pilot**, using React 19 + TypeScript 6 + Vite 8. `src/main.tsx` and `src/App.tsx` replace the JSX counter starter. The workspace includes Browse, Portfolio, a shared owner/browse PropertyDetail, and Analysis. Create account switches to signup within the same card; there is no modal or routing dependency. The user-supplied `S:/eiv-navbar/nineframes-eiv-frontend/` was inspected read-only as design inspiration, not copied. The slate/copper design has different panel proportions, original local SVG property artwork/placeholders, responsive layouts, accessible labels/focus/errors, password toggles, and reduced-motion support. No remote fonts/assets or reference-app auth logic/dependencies were imported.

`src/auth.ts` owns typed validation and POST requests to `/register/SignIn` (`email,password`) and `/register/SignUp` (`email,password,admin:false`). Requests include cookies, understand plain-text success responses, sanitize errors rather than expose SQL internals, and use a 15-second timeout. `src/components/AuthForm.tsx` handles pending/success/error states, prevents duplicate submits, aborts on unmount, and clears passwords on mode changes/success; no credentials are persisted in browser storage. Signup confirmation stays client-side. Signup validates 9–72 English letters/digits only to match the existing Go policy and bcrypt maximum; sign-in does not impose signup restrictions on existing passwords. The backend still validates email-domain MX records.

`npm run dev` binds localhost:5173 (strict port); `vite.config.js` proxies `/register/*`, `/User/*`, `/analysis/*` to localhost:8080. Go must run for real authentication/portfolio; both Go and Python must run for area analysis. Previewing the auth UI needs only Vite. The unchanged backend cookie is Secure/HttpOnly/SameSite=Strict: use localhost consistently for development and HTTPS plus a same-origin API reverse proxy for deployment. Real credentials/cookie round trips were not tested; a successful response alone does not prove cookie retention. Do not weaken cookie settings. Existing backend security risks remain.

Successful sign-in/signup calls `onAuthenticated` to replace the auth card with `src/components/Workspace.tsx`. Browse is the default, initially loading sale listings across all areas. The hamburger sidebar links to `#/browse`, `#/portfolio`, `#/analysis`; cards link to `#/portfolio/property?uid=...` or `#/browse/property?...&uid=...`. `useSyncExternalStore` observes hash changes, so browser Back/Forward work without a router library. Page changes update document title, focus the main region and scroll to the top; menu dismissal returns focus to its toggle. Auth state remains in memory: refresh returns to sign in and subsequent login resumes the hash route. A 401 offers Sign in again. No session tokens or property data are cached in browser storage. The header is sticky. Admin, edit/delete forms, session restoration, logout, password reset and Google auth remain deferred. Portfolio now includes an Add property action at `#/portfolio/add`; the saved UID replaces the form route with `#/portfolio/property?uid=...`.

`pages/Browse.tsx` / `Browse.css` provide area selection (shared two-second autocomplete), sale/rent, asking EUR/m² and integer m² ranges. Optional ranges and all-area searches are supported. Search resets to page 1; Clear filters restores defaults. The UI uses `/User/BrowseProperties` and Previous/Next controls with `has_next`; 20 portfolio-style cards sit in a scrollable results region. Applied filters and index live in the hash URL and survive detail/back navigation. `components/PropertyCard.tsx` is shared with Portfolio; it uses URL/data-URL images or the existing local SVG fallback. `PropertyDetail` selects `/User/BrowseProperty` for market listings and retains `/User/ViewProperty` for portfolio ownership. Both render the same saved features, prices, Elo, GeoPremium and local/global SHAP. AreaSearch styles now live in `components/AreaSearch.css`. Recent area-search Go code/tests, shared request handling and touched JSX were expanded into readable multiline code; the user prefers simple code without explanatory source comments. No new frontend dependency was added.

`pages/Portfolio.tsx` GETs `/User/Portfolio` for the actual property count/estimated-value sum and overview list: UID link, address, area, estimate/asking per m², Elo and GeoPremium. It discloses that the existing server total combines sale/rental values without separation. `pages/PropertyDetail.tsx` GETs owner-only `/User/ViewProperty?uid=...`; it presents all persisted input groups, geography, estimate/asking totals/per-m², Elo, GeoPremium/z-score and residual. `PropertyExplanation.tsx` keeps local signed €/m² SHAP impacts and local absolute shares separate from global mean-absolute/signed SHAP importance. Top eight rows expand to all, metadata and model mismatch warnings are exposed, and null SHAP does not hide the property. Nullable asking remains Not provided; omitted/null/broken images use an original SVG placeholder. URL/base64 images are ordinary img sources, never injected HTML. Shared `property.ts` contracts and `usePropertyData` implement cookie-bearing requests, cancellation/stale-result guards, retry and safe errors. Reads have a 20-second timeout.

Property creation (2026-10-09): `pages/AddProperty.tsx` / `AddProperty.css` and `addProperty.ts` expose sale/rent, subtype, full address, whole m², construction/optional renovation year, floor list, optional asking €/m², rooms, heating/energy and amenities/views/facade. Blank asking is omitted, renovation determines IsRenovated, and flags use the backend's 0/1 contract. Address entry reuses `AreaSearch` in address mode and `/analysis/autocomplete?...&kind=address`. Press Enter or Find address to query Nominatim, then select a result to preview its lat/lon. A selected canonical address and OSM type/ID are required; edits clear the selection and disable saving. Go resolves the same provider location and returns safe HTTP 422 codes for missing/changed selections. Street-only matches show a note to include a house number for building precision; OSM coverage determines precision. The former saved-property-address restriction has been removed. No new dependency, route or schema was added. Browser/backend validation precedes writes; submissions include cookies, block duplicates, abort on unmount and ignore late results. The browser waits up to two minutes and gives no automatic retries. Failed/uncertain saves direct users to check Portfolio because the existing post-commit derived-data flow can leave a saved property on error. `ServeAddProperty` preserves the POST/auth/body/validation contract, discards caller-derived coordinates and sanitizes internal creation errors. `User.AddProperty` propagates InsertProperty's generated UID; HTTP 201 adds `uid` and `Location` while retaining `message`, and the frontend opens that property's owner-only detail page. Both Go and Vite must run; restart Go after changing the contract.

`pages/Analysis.tsx` sends the chosen administrative bbox as numeric `{MinLat,MaxLat,MinLon,MaxLon}` to POST `/analysis/CreateAnalysisForArea`; it uses a five-minute browser timeout and explicit Stop waiting (does not promise Python CPU cancellation). `/analysis/autocomplete?q=...` is always two-second debounced, searching cached DB boundaries. The prior Find areas button and environment flag were removed. Dropdown selection, keyboard navigation, query edits and stale-request cancellation are handled. Editing the area clears stale errors. Attribution is displayed. Safe error codes distinguish invalid bounds and data-loading failure; returned `warnings` explain missing models without hiding results. No automatic model retries or duplicate in-flight submissions.

`AnalysisResults`, `ClusterPlot`, `AnalysisPoints` and `AreaModelReport` expose every returned result group: points; per-cluster/sell/rent means and standard deviations; top/bottom 5% Elo tails; all 17 point fields including six cluster-relative z-scores; both area models' training R², SHAP baseline, XGBoost importance (not SHAP importance), mean-absolute SHAP interactions, and all examples/contributions. The scatter plot uses area/age, explicitly not geography; K-means actually uses standardised area/age/renovation age/floor. The plot alone is thinned to <=2,000 points on large responses; full property tables use 20-row client-side pages. Analysis results stay in component memory and clear when leaving the page. Backend small-group and nontermination fixes were subsequently implemented and live-tested on 2026-09-17 (below). README documents setup/contracts/scope. `npm run build` includes strict TS checks; UI tests use mocked HTTP. Production subpath hosting is not configured.

## PostgreSQL schema memory

`schema_dump.sql` is a schema-only dump produced by pg_dump 18 from PostgreSQL 17.5 and then extended with portfolio images and explainability persistence. It requires `pg_cron`, `pg_partman`, `pg_trgm`, PostGIS, and `unaccent`, assumes owner `postgres`, and references a pre-existing `mcp_readonly` role. The explainability addition contributes six logical tables. There are no views, materialized views, row-level-security policies, or data rows.

Logical table catalog (every application table in the dump):

- Property/listing: `properties`, `base_attributes`, `building_details`, `residential_specifics`, `commercial_specifics`, `landplot_specifics`, `property_geo_data`, `property_images`, `realtors`, `property_realtors`, `user_properties`, `user_saved_properties`, `business_saved_properties`, `deleted_properties`.
- Valuation, ranking, and model data: `valuations`, `valuations_history`, `estimated_value_snapshot`, `estimation_data`, `global_stats`, `properties_elo`, `properties_analysis_sell`, `properties_analysis_rent`, `feature_importance_sell`, `feature_importance_rent`, `features_interactions_sell`, `features_interactions_rent`, `property_rankings`, `ranking_models`, `area_market_premiums`, `market_premium_policies`, `marketabillity_factor`, `marketabillity_factor_batches`, `valuation_adjustment_policies`, `valuation_backtest_jobs`, `template_public_valuations_history`, `worker_artifact_revisions`, `segment_metadata`, `gov_area_geocoding`, `gov_verified_sales`.
- Identity and organizations: `users`, `auth_providers`, `user_credentials`, `businesses`, `user_businesses`, `signup_whitelist`.
- Plans, contracts, credits, and billing: `individual_plans`, `individual_plan_features`, `individual_subscriptions`, `user_credits`, `credit_adjustments`, `business_contracts`, `business_features`, `business_commission_settings`, `business_portfolio_monthly_history`, `contract_offers`, `bulk_subscription_change_jobs`, `bulk_subscription_change_targets`, `billing_preferences`, `payment_methods`, `payments`, `payment_refunds`, `invoice_tasks`, `admin_charge_waivers`.
- Reports and engagement: `detailed_reports`, `detailed_report_views`, `detailed_report_forwards`, `call_requests`, `push_subscriptions`, `notification_log`, `user_feedback`, `white_label_settings`.
- Administration, boundaries, and scheduling: `admin_action_records`, `admin_active_user_snapshots`, `admin_audit_events`, `admin_boundaries`, `admin_boundary_negatives`, `scheduler_jobs`, `scheduler_job_runs`, `scheduler_job_scrape_sources`, `scheduler_maintenance_chain_runs`.

`valuations_history` is range-partitioned by `archived_at`, with a default partition and monthly partitions from February through November 2026. Each has indexes for archive time, property UID, and effective maximum value. No pg_partman/pg_cron scheduling call is included in the dump, so ongoing partition/archival maintenance is not established here.

Database functions provide overloaded global/segment/subtype valuation adjustments, point-in-boundary market premiums, clamped value application, confidence/MAE ranges, monthly valuation archiving, delete-time property snapshots, polymorphic billing/payment cleanup, and marketability geometry construction. Triggers only cover marketability geometry and six manual user/business cleanup cases.

Important schema concerns:

- `GRANT ALL ON SCHEMA public TO PUBLIC` is broad. `mcp_readonly` receives SELECT on every table and future table, including password hashes, payment/customer metadata, push credentials, PII, audit data, and report token hashes. Least-privilege views/column grants are needed.
- There is no RLS. Application authorization is therefore the only tenant/user boundary in this bundle, and it is currently incomplete.
- `archive_monthly_valuations()` does not copy the newer effective/adjustment/market fields present in the history table.
- `apply_market_premium_only(..., as_of, ...)` accepts but ignores `as_of`; fallback uses the current active global premium, so historical replay is not deterministic.
- `area_market_premiums` uses a nullable-column UNIQUE constraint with default NULL-distinct semantics, allowing duplicate fallback rows. Matching only on boundary name also risks collisions across areas/levels.
- `valuations_history` references live `properties` with ON DELETE CASCADE, so deleting a property removes its valuation history even though a separate deletion archive exists.
- `deleted_properties`, `marketabillity_factor`, and valuation-history rows have no ordinary primary key; some uniqueness is supplied only by indexes or not at all.
- Many status, subtype, owner, coordinate, and monetary semantics are unconstrained strings/numbers. Several polymorphic owner references cannot be enforced by foreign keys.
- The dump is not immediately portable: extensions must be installed/configured, roles must exist or ACL/owner statements must be skipped, and the pg_dump 18 psql meta-commands may require a compatible client.

Only a subset of this schema is used by the Go bundle. Newer `users`/business/billing/reporting/scheduler/ranking-model structures have no matching HTTP implementation here. `Database/migrations/001_create_property_images.sql` adds portfolio images. `Database/migrations/002_create_explainability_persistence.sql` adds the six latest-only SHAP tables and extends `global_stats` with baseline, training/validation R², native model SHA-256, and trained-at columns. Global sell/rent model metadata is stored in the existing `Price_stats_sell` and `Price_stats_rent` rows.

## Highest-priority known problems

1. Rotate and remove the committed PostgreSQL superuser credential; require TLS and a least-privileged runtime role. Remove/redact the checked-in sign-in and cookie fixtures.
2. Replace the forgeable literal JWT key; require exact signing method and proper claims; remove caller-controlled admin signup; add real role/tenant/object authorization.
3. Protect or remove `/train` and `/test`; authenticate prediction as appropriate; remove the remaining `log.Fatal` behavior from `/test`; add method/body/rate/time limits.
4. Make property creation and its derived inference/estimate/Elo work recoverable and idempot-safe. The created UID is now returned and detail reads enforce ownership.
5. Extend the new explicit residence/property Phase2 feature manifest approach to every other model segment. Fix remaining land constant features, subtype mapping, GeoPremium skew for affected non-residence segments, and stale dates.
6. Add graceful shutdown/resource cleanup and harden failure recovery around the remaining legacy derived-data writes; the new residence model artifact/reload path is validated and race-protected.
7. Correct global-stat scoping, batch result handling, float/integer Elo mismatch, zero-division paths, and k-means termination/determinism.
8. Reconcile the legacy Go data model with the supplied schema, especially the split identity tables.
9. Add tests around auth/authorization, add-property rollback/retry, SQL mappings, feature vectors/dimensions, model reload concurrency, GeoPremium/Elo math, k-means edge cases, API contracts, and schema restore.
10. Extend the frontend beyond authentication, Browse, portfolio/details and area analysis: agree on session restoration/logout, implement remaining edit/delete property forms, and resolve the remaining legacy backend computation risks.

## Verification snapshot

- Nominatim address verification (2026-10-09): lint, TypeScript/build, all 85 frontend tests and `go test ./...` passed. UI tests cover explicit Enter/Find address requests, selection/coordinate preview, editing invalidation, mandatory selection, safe 422 feedback, OSM type/ID payload and successful-save detail redirect. HTTP-fake Go tests cover Greek query encoding, stable-object lookup, exact search-to-save coordinate parity, changed-address rejection, caching, invalid coordinates, cancellation, upstream failure and the one-request-per-second limit. A live Nominatim query for `ερεχθειου` returned eight results, including Athens, Alimos, Korydallos and Peristeri; resolving each selection retained exactly the returned coordinates. No properties, database records or models were created/changed. The localhost Go service was rebuilt/restarted for the updated contract. Browser automation remained unavailable, so a real browser save was not exercised.
- Add-property verification (2026-10-09): `npm run lint`, `npm run build` (including TypeScript), all 82 frontend tests and `go test ./...` passed. New tests cover empty-portfolio entry, sale/rental payloads, omitted asking, retained address, renovation/floor validation, encoded-UID redirect/detail/back navigation, duplicate prevention, unmount/late responses, expired sessions, safe failures and legacy missing-UID responses. Go handler tests verify 201/message/UID/Location, authenticated owner/transaction, preserved trimmed address, ignored caller-derived values and rejection before creation. A read-only query found saved addresses on the 20 most recent user-property records; the user confirmed the address display was working. No live property was created and no records/models were changed during verification. Browser automation was unavailable because its trusted Node helper exited; real creation was not exercised. The localhost Go service was rebuilt/restarted to activate the new contract.
- `go test ./...`: passes, including explainability, Elo, vector, mapping, API-client, full-detail sell/rent field mapping, owner-only GET/legacy POST handlers, omitted asking/image keys, image URL/data URL, local/global importance separation, optional-asking validation, neutral-deal Elo and inference-vs-training label-filter tests. New detail/Elo tests use database fakes; no live AddProperty/DB mutation was performed for this endpoint change.
- Browse verification (2026-09-16): Go browse tests cover all filter placeholders, coordinate order, page offsets/defaults, minimal invalid-input handling, overview/null image mapping, empty pages, error sanitization, auth rejection, non-owner detail access and unchanged owner-only behavior. `BROWSE_LIVE_DB_TEST=1 go test ./Backend/Api/UserActions -run '^TestBrowsePropertiesLiveDatabase$' -count=1 -timeout=60s -v` passed against the configured PostgreSQL database: sell/rent pages 1 and 2 each returned 20 non-overlapping rows; both detail readers returned 30 global importance records; exact inclusive combined filters matched construction year, renovation year, estimation and Elo. Only reads/read-only transactions were used; no records or model artifacts were changed. Ordinary tests skip that integration test unless explicitly enabled. For future runs, keep GOCACHE outside the bundle as described in the development cache policy above.
- `npm run lint`: passes under Node 22.20.0/npm 11.11.0, including the new TypeScript frontend.
- `npm run build` passes strict `tsc --noEmit` and Vite bundling; all 85 frontend tests pass. Tests cover auth/nav, portfolio/detail contracts, all SHAP rows, optional images/asking, cancellation/stale results, two-second area search, bbox mapping, partial-model warnings, error clearing, clusters/model output, Browse filter application, pagination/detail/back, reset and empty/error states. UI HTTP is mocked. Prior desktop/mobile synthetic-response previews verified layout/navigation. Real unauthenticated Vite proxy checks returned 401. Browser authentication remains user-driven; no credentials or session cookies are copied into tests.
- Browse live verification (2026-09-27): the read-only PostgreSQL test passed all six combined exact-inclusive filters, all-area defaults, sell/rent pages, `has_next`, non-overlap and full detail readers (30 importance records). A user-signed-in browser verified Pireaus autocomplete, Piraeus sale asking 1000–2000 EUR/m² and size 50–100 m², disjoint 20-row pages 1/2, identical overview/detail values and stored SHAP, and back navigation restoring filters/page. Rental asking 5–15 EUR/m² with the same area/size also worked. Results scroll independently, SVG placeholders render for missing photos, and the narrow layout had no horizontal overflow or browser console errors. Missing stored addresses are shown honestly as unavailable. Go's localhost development process was rebuilt/restarted; no DB writes, Python calls, training, or model artifact changes were made for Browse.
- Analysis live verification (2026-09-17): opt-in `ANALYSIS_LIVE_TEST=1 ANALYSIS_LIVE_ML_TEST=1 go test ./Backend/Api/MarketAnalysis -run TestAnalysisLiveDatabase -count=1 -timeout=5m -v` passed against the configured DB and running Python service. `Pireaus` resolves to Δήμος Πειραιώς, bbox south/north/west/east `37.926325,37.965802,23.620993,23.673672`. Full pipeline returned 7,725 points (5,890 sell, 1,835 rent), seven cluster/transaction groups, both model reports/interactions and ten explanations each in about 8 seconds. The screenshot's erroneous tiny Thessaloniki box contains one property; it now returns its cluster plus an insufficient-data warning instead of HTTP 500. This test only reads DB and trains request-local models; it never calls `/train`, writes tables or publishes artifacts. Ordinary test runs skip it. Unit tests cover Greek/English/typo matching, degenerate K-means, minimum group sizes, partial failures/cancellation and auth wrappers. Go was rebuilt and its localhost:8080 development process restarted; Python was unchanged.
- System Python: `python -m unittest -v test_explainability.py` passes all ten tests, including a synthetic 30-feature Phase2 CLI run with sampled interactions and complete progress events, ONNX/UBJ prediction parity, all 435 interaction records, deterministic bounded sampling, sample-mean correctness, unchanged all-row contributions/importance, endpoint/hash checks, schema/CSV contract parity, atomic registry reload, artifact rollback, cancellation-aware additivity and correct failing-row diagnostics. Go tests also cover progress/log throttling, sampling metadata, offline/online additivity parity, incorrect reported errors and overflow rejection.
- Microservice venv: the same command passes five service/shared/repository tests and skips five Worker tests because `onnxmltools` is absent there. The FastAPI module and routes import successfully. Production `/train` and database writes were not run to verify the 2026-09-15 changes; the existing sell model was checked read-only across all 123,449 CSV rows.
- `go test -race ./...` and `go vet ./...` pass with their caches directed inside the workspace.
- No local `psql` executable was available, so the dump has not been restored in an isolated database.

## Open decisions before feature work

- Which schema generation is authoritative: the large supplied dump or the smaller legacy Go model?
- Is the public-DDNS database a disposable test database or shared/production-like infrastructure? Treat it as sensitive until confirmed.
- Should file-based ONNX artifacts remain authoritative, or should `ranking_models`/artifact revisions in PostgreSQL become the model registry?
- What is the intended user/business/role model and which endpoints should guests, individual users, agents, business admins, and platform admins access?
- What are the required frontend workflows and stable JSON contracts?
- What deployment topology should connect the frontend, Go service, Python service, database, ONNX Runtime, and external Nominatim dependency?
