# Ptixiaki repository guidance

## Persistent project context

- Read `PROJECT_MEMORY.md` before changing this bundle. It contains the current architecture, endpoint, database, ML-contract, validation, and risk map.
- Treat `PROJECT_MEMORY.md` as a maintained snapshot, not a substitute for checking the affected source. Update it when architecture, routes, schema contracts, model features, artifact names, setup, or verification commands change.
- The bundle root is not a Git checkout, but `Ptixiaki/` is. Preserve user files and unrelated dirty/generated changes; do not assume the nested checkout is clean.

## Source and generated artifacts

- First-party source is under `Ptixiaki/`, `Ptixiaki-ml-microservice/`, `ml_shared/`, `Ptixiaki-frontend/Ptixiaki-Frontend/src/`, and `schema_dump.sql`.
- `node_modules/`, `venv/`, frontend `dist/`, Worker CSV/JSON/mapping/ONNX outputs, `Ptixiaki.exe`, and `onnxengine.dll` are dependencies or generated/runtime artifacts. Do not review or edit their internals as application source unless the task explicitly targets them.
- Residence/property Phase2 retraining publishes ONNX, UBJ, and manifest files as one rollback-safe latest-only artifact set. The Go `/train` mutex prevents concurrent jobs; do not run production retraining casually because it also replaces explanation tables and derived model data.

## Development caches and temporary outputs

- Do not recreate `.gocache/` or `.cache/` anywhere in this bundle. The user requested their removal on 2026-10-28.
- Keep Go's build cache outside the bundle. Use its default Windows cache (`%LOCALAPPDATA%\go-build`); if that location is unavailable, use a writable directory under the operating system's temporary directory, outside this bundle.
- Keep temporary development executables and redirected stdout/stderr logs outside the bundle, normally under `%TEMP%\ptixiaki-dev`. Do not redirect builds or logs into the removed folders.

## Security baseline

- Never print, copy into documentation, or newly commit the credential/session values present in `Ptixiaki/Database/DbConnection.go`, `Ptixiaki/signin.json`, or `Ptixiaki/cookie.txt`.
- Treat the committed database credential and current JWT signing key as compromised. Any deployment-oriented work must move secrets to environment/secret management, enable database TLS, use a least-privileged database role, and rotate exposed values.
- Do not expose `/train`, `/test`, signup-controlled admin privileges, or cross-user property access as-is.

## Current verification baseline

- Go: from `Ptixiaki`, `go test ./...` passes. Tests cover Elo math, exact explainability vectors/contracts, CSV category mappings, streaming records, persistence validation, API client behavior, model reload helpers, and property browse filters/pagination/auth/detail contracts. Browse has an opt-in `BROWSE_LIVE_DB_TEST=1` read-only database test; see `Backend/Api/UserActions/BROWSING.md`. Never enable it casually as part of unrelated work.
- Frontend: from `Ptixiaki-frontend/Ptixiaki-Frontend`, run `npm run lint`, `npm run build` (including TypeScript checks), and `npm test` (85 tests). Property Co Pilot includes auth, Browse, portfolio/property creation, shared property details and area analysis through Vite proxies `/register`, `/User`, `/analysis` to localhost:8080. Browse defaults to all-area sale listings, filters area/sale-rent/asking EUR per m²/size, and uses 20-row pages with `has_next`; filters/page survive detail/back navigation. UI tests mock HTTP; real browser and read-only PostgreSQL Browse checks passed on 2026-09-27. Add-property address entry reuses the area autocomplete component/API with `kind=address`, submitting address searches to Nominatim with Enter/Find address and verifying selected provider coordinates by OSM type/ID before saving; new streets do not need saved properties. Public Nominatim forbids search-as-you-type, so only administrative-area search is debounced. Area autocomplete is always two-second debounced and searches Go's cached `admin_boundaries`, not public Nominatim; no environment flag is needed. See `PROJECT_MEMORY.md` for the separately verified real Piraeus analysis and remaining limitations.

- Python: from `Ptixiaki-ml-microservice`, system `python -m unittest -v test_explainability.py` passes all ten tests, including a synthetic end-to-end Phase2 train with sampled interactions/progress, full-population contributions/importance, deterministic sampling, ONNX/UBJ parity, endpoint validation, schema/feature parity, cancellation-aware numerical checks and accurate failing-row diagnostics. The service venv passes five service/shared/repository tests and skips five Worker tests because it lacks `onnxmltools`; do not treat that venv as a complete Worker environment.
- PostgreSQL: `schema_dump.sql` is schema-only and requires PostgreSQL extensions/roles. No local `psql` was available when this baseline was recorded, so restore validation is outstanding.
- Existing databases must apply `Ptixiaki/Database/migrations/002_create_explainability_persistence.sql` before `/train` can publish SHAP results.

## Readability

- Prefer descriptive names and multiline control flow/JSX. Do not compress conditionals, handlers or multiple statements onto a single line. The user prefers simple code without explanatory source comments; keep architecture and setup notes in Markdown documentation.
