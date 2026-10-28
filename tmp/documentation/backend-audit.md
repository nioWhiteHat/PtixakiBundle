# Backend and database source audit for project documentation

Audit date: 2026-10-04. This is evidence for the main documentation author, not an application change. Current source was re-read; existing memory was used only as a map. Paths below are relative to `S:/ptixiaki-bundle/` unless stated otherwise.

## Scope and fresh verification

Read the composition root; all user-action handlers/readers; property validation; auth implementation with signing-key values filtered out; geocoding; browse/portfolio/detail/delete; market-analysis pipeline, search, clustering and HTTP adapter; training orchestration; explainability persistence/client/artifact contract; GeoPremium, residual and Elo math; SQL schema relevant tables, keys, spatial indexes and extensions; migrations 001 and 002. No credentials, cookie contents, private records or generated training datasets were read. No production training, live DB requests, account creation or application edits occurred.

Fresh command run from `Ptixiaki/`, with `GOCACHE=S:\ptixiaki-bundle\.cache\go-build`: `go test ./...` passed (2026-10-04). Some package results were cached. Auth, external geocoding, Database and Worker/training report no test files. Integration tests remain opt-in and were NOT enabled. Historical live Browse (2026-09-27) and Piraeus Analysis (2026-09-17) claims should be labelled historical, not freshly repeated.

The nested Go checkout already contains many modified/generated/untracked files. They were preserved. `go.mod:3` specifies Go 1.25.1; direct core dependencies include pgx/v5 and onnxruntime_go. The root is not a Git repository.

## Implemented system, not aspirational scope

Go is a combined HTTP API, database coordinator, ONNX inference host and process-level training orchestrator. It is not a thin proxy. Its HTTP frontend is the standard `net/http` ServeMux, listening on port 8080; runtime dependencies include PostgreSQL/PostGIS, an ONNX Runtime shared library and final model files. Python is called in TWO ways: a child process for global training, and an HTTP service for request-local area analysis and cached native explanations. The React client does not need to call Python directly.

The complete SQL dump contains a much larger business/billing/reporting/scheduler design than the working Go API implements. Do not claim subscriptions, payments, organizational roles, report delivery, scraping or scheduling as delivered capabilities merely because tables exist. Online valuation and explainability are specifically wired to residence apartment/detached-house/maisonette sell and rent models. Other Worker segment configurations exist but are not active web inference models.

## Route inventory

Route registration is authoritative at `Ptixiaki/main.go:54` through 67. JWT-protected means wrapped by `auth.Authenticate`, not a demonstrated comprehensive production access-control model.

| Route | Enforced method / auth | Input and result |
| --- | --- | --- |
| `/register/SignUp` | No explicit method restriction; public | JSON `email,password,admin`. Password length >8 and ASCII letters/digits; email domain MX lookup. Inserts legacy `user_credentials`, creates 24-hour JWT cookie, returns 201 plain text `User created successfully!`. Client-controlled admin is a known defect, not intended safe role assignment. |
| `/register/SignIn` | No explicit method restriction; public | JSON `email,password`; bcrypt verify; returns 200 plain text and cookie or 401. |
| `/User/AddProperty?transaction=sell\|rent` | POST; JWT | One JSON object, max 1 MiB, unknown fields rejected. `PropertyFeatures` PascalCase fields. Asking per m² may be omitted/null. Creates base listing, estimates/ranks and best-effort explains it. 201 `{"message":"Property added successfully"}` only; no UID returned. |
| `/User/Portfolio` | GET; JWT | Current user's linked listings; `property_count`, `total_estimated_value`, `properties`. Count includes records without predictions. No pagination here. |
| `/User/ViewProperty?uid=...` | GET, legacy POST accepted; JWT + ownership | Legacy body key `PropertyUid` if UID query absent. Consistent stored full detail; other-user/missing UID returns 404. |
| `/User/DeleteProperty?property_uid=...` | DELETE; JWT + ownership | Deletes ONLY matching `user_properties` link, not listing/valuation/image records. 200 success message; 404 if no owned association. |
| `/User/BrowseProperties` | GET; JWT | Optional bbox, transaction, index and ranges. Max20 listings plus `has_next`; no total count. All eligible residential dataset listings, not just portfolio. |
| `/User/BrowseProperty?uid=...` | GET; JWT, no ownership requirement | Same stored detail reader scoped to supported residential listing types; missing image explicit null. 404 for absent/unsupported listing. |
| `/analysis/autocomplete?q=...` | No explicit method restriction; JWT | Cached administrative-boundary catalogue, max8 suggestions, same Nominatim-shaped JSON as historical implementation. No public Nominatim request for autocomplete. |
| `/analysis/CreateAnalysisForArea` | POST; JWT | JSON numeric `MinLat,MaxLat,MinLon,MaxLon`; returns `points,clusters,sales_report,rent_report` plus optional warnings. Each report can be null while clusters remain useful. |
| `/predict/sell?uid=...` | No explicit method restriction; public | Live ONNX prediction for existing property UID, plain-text result. Rent route is commented out. |
| `/train` | No explicit method restriction; public | Synchronous, serial sell then rent global retraining/persistence/reload. 200 plain-text success or sanitized 500. Expensive and database-mutating; must not be publicly exposed. |
| `/test` | No explicit method restriction; public | Imports fixed sell artifacts into derived database records; not a health check. Calls `log.Fatal` on failure. Unsafe operational route. |

Primary evidence: `main.go:87` (analysis), 133 (autocomplete), 171 (portfolio), 191 (delete), 228 (add), 284/328 (auth), 371 (train), 438 (predict); `Backend/Api/UserActions/PropertyDetailsHandler.go:11`; `BrowseProperties.go:37`.

### API values and semantics

`PortfolioPropertyOverview` (`Portfolio.go:15`) has UID, estimated/asking €/m², integer m², address, optional image, integer Elo and GeoPremium. Values are nullable pointers where unavailable. Portfolio total is the sum of available `estimated_price_per_sqm * square_meters`; it presently mixes sale and rental portfolio records in one number and should not be described as a financially homogeneous asset value.

Image storage is URL OR bytes (not arbitrary HTML source). Primary image is first by `position,created_at`; bytes become a `data:<mime>;base64,...` source. Image loading is batched, with graceful missing-image-table handling. Portfolio and owner detail omit absent image; Browse and Browse detail return explicit `image:null`. No image upload endpoint is implemented. `Portfolio.go:112,160`.

`PropertyDetails` (`PropertyDetails.go:26`) returns outer snake_case fields: UID, transaction, features (existing PascalCase input contract), per-m²/total estimates, optional per-m²/total asking, Elo, GeoPremium and global GeoPremium z-score, nullable residual, optional image, nullable `explanation`, array `feature_importance`. Original normalized storage does not preserve every conceivable historical input; many missing scalar features default to zero/empty on read, so say "stored/reconstructed features", not exact original submitted JSON. Feature aliases/admin boundary are derived as appropriate.

Explanation includes baseline, explained prediction, additivity error, hash, explained_at, model-consistency flag and 30 signed feature contributions. Each local percentage is `100*abs(contribution)/sum(abs(all contributions))` (zero if denominator zero). Global feature importance separately exposes full-population mean absolute/signed SHAP, percentage, rank and hash/time. `PropertyDetails.go:200,237`. Read endpoints do not call Python or generate missing SHAP. Full detail reads use a read-only REPEATABLE READ transaction (`PropertyDetails.go:78`).

### Browse query contract

`BrowseProperties.go:60` builds parameterized values; no user-provided SQL identifiers. Optional bbox (`min_lon,min_lat,max_lon,max_lat`) defaults to world bounds when all four are absent; any supplied bbox requires all four and ordered finite valid coordinates. `transaction` defaults sell, must sell/rent. `index` defaults1, positive int64 within safe offset range.

Inclusive min/max pairs combine with AND: `construction_year`, `renovation_year`, `estimated_price_per_sqm`, `elo`, `asking_price_per_sqm`, `square_meters`. Integer-valued filters are years, Elo and size; price bounds allow decimal. Inverted ranges yield no matches. SQL geometry test is `geom && ST_MakeEnvelope(...,4326)`: it searches the rectangle, not the municipality's exact polygon. All-area search still requires a geolocated listing.

Asking uses estimation-data asking, falling back from nonzero total asking divided by nonzero size. Ordering is Elo DESC NULLS LAST then UID ASC. `OFFSET (index-1)*20 LIMIT21` returns20 max plus a lookahead boolean. Concurrent changes can shift offset-based pages; no frozen result-set token exists. `BrowseProperties.go:128,133`.

Privacy caveat: Browse does not check `user_properties.is_published`; even user-added rows inserted with false can be browsed by other authenticated users. This broad browse access was intentionally requested, but publication/privacy semantics must be resolved before deployment. Ownership IDs are not returned. Owner detail still independently checks ownership.

## AddProperty lifecycle and partial-failure boundaries

Evidence: `UserActions.go:20`, `DataBaseActions.go:19`, `main.go:228`; validation `Backend/Api/PropertyValidation.go:13,29`.

1. Verify transaction and simple property input. Supported subtype, nonblank <=255-character address, m²>10, construction year1900..current, flags0/1, coherent renovation year, nonnegative WC/bedrooms, and positive finite optional asking with int32-compatible total. Handler clears caller UID, coordinates, zone, distance and admin name.
2. Geocode address using public Nominatim. It takes the first result, with a user-agent header; current `http.Client{}` has no timeout/request context or status-code check. Get zone value from nearest stored property without radius/country/type restriction. (`ExternalServices/Geo/GetGeoData.go:23,48`.)
3. Generate `property_` + ULID. Transaction inserts `properties`, `base_attributes`, `building_details`, `property_geo_data`, `residential_specifics`, `user_properties(is_published=false)`. Total asking is integer-truncated from €/m²*size; unpriced remains SQL NULL. Commit at `DataBaseActions.go:206`.
4. After commit, fetch/encode inference features, calculate GeoPremium from database comparables, obtain 30-feature Phase2 prediction. Inference permits absent asking via separate query rules, not by training on a zero label (`Worker/database/QueryBuilder.go:269`). Attr predictor argument is unused (`testing/PredictPrice1.go:10`).
5. Calculate GeoPremium again for persistence (not reused from inference), insert `estimation_data`, calculate score, update GeoPremium z-score, insert `properties_elo`. These operations are not in the base transaction.
6. Best-effort HTTP `/ExplainProperty` with exact inference vector; Go validates feature names, finite numbers, SHAP additivity and native/ONNX agreement then upserts one property explanation. Timeout20s; error is logged but property success remains201 (`DataBaseActions.go:271`; `explainability/client.go:111,235`).

Thus base insert success does not imply complete derived data. Prediction/stats/Elo failures return500 after leaving an owned base listing; retries create another UID (no idempotency key). Missing SHAP alone is nonfatal. A later train replaces explanation tables with current training population: an unpriced property's online explanation can disappear since unpriced rows are excluded from supervised training. Returning only a success message was explicitly requested, so do not claim the endpoint returns the new UID.

## Exact mathematical definitions

This project's "Elo" is a custom static weighted z-score ranking, NOT chess-style iterative Elo with pairwise matches or calibrated win probabilities. Higher means more favorable under the chosen residual/location heuristic, not guaranteed investment quality.

Let `A_i` be asking €/m², `P1_i` attribute-model prediction, `P2_i` final Phase2 prediction. Phase1 residual is `r1_i=100*(A_i-P1_i)/A_i`. GeoPremium `g_i` is the arithmetic mean of neighboring Phase1 residuals within300m, up to50 neighbors, same transaction and residence grouping. Local residual dispersion is population standard deviation (divide by n, not n-1). Positive GeoPremium indicates nearby asking prices exceed attribute-only predictions on average; it is not a causal, official or guaranteed price surcharge. (`Worker/training/GeoPremium.go:36,136`; `testing/GetGeoPremium.go:21`.)

Final deal residual is `r_i=100*(A_i-P2_i)/A_i`. A lower asking than estimate gives a NEGATIVE residual. Define global residual z-score `(r_i-mean(r))/std(r)`, global GeoPremium z-score `(g_i-mean(g))/std(g)`, and local residual z-score `(r_i-mean(neighbor final residuals))/std(neighbor final residuals)`. Local deal neighbors: up to20 within150m excluding self, eligibility from residence-property training query, same transaction. All stds are population stds; zero std makes z-score0. There is NO division of final residual by the Phase1/GeoPremium neighborhood std as a substitute for local deal z-score.

`Score_i=10000 - 3000*z_global_residual - 2000*z_local_residual + 3000*z_global_geopremium`.

Formula helper shared by training and add endpoint: `Worker/Pipelines/CalculateEloForProps.go:183,198,205`; online calls at `Backend/Api/Elo/CalculateElo.go:16,48`. Missing asking gives missing residual and skips both deal terms, so `Score=10000+3000*z_global_geopremium`; it does not invent asking0 or residual0. Scores computed as float64 but `properties_elo.elo` is INTEGER in SQL, losing fractional precision on storage. No clamp to conventional Elo ranges.

Important parity qualification: formula is shared, but live neighborhoods/data can differ from a training snapshot. In particular offline GeoPremium selects50 raw subtype/transaction neighbors THEN filters to UIDs surviving Phase1; online filters eligible asking/attr_price BEFORE LIMIT50. Offline no-neighbor result is premium0/std0; online returns error. New listings lack attr_price until training and cannot act as GeoPremium comparables. Therefore do not claim exact offline/online value identity in all edge cases.

Global stats rows are semantic keys `Geo_stats_sell`, `Geo_stats_rent` (mean/std of GeoPremium) and `Price_stats_sell`, `Price_stats_rent` (mean/std of final residual). Model baseline, training/validation R², native hash and trained_at are additional columns on Price_stats rows, not replacement of residual statistics. Stat names omit segment, so manually training other sell/rent segments can overwrite the same statistics. `InsertTrainingData.go:128`; migration002:176.

## Global training publication and reload boundaries

`main.go:387`: `TrainingMu` serializes /train requests. Sell completes before rent begins. `Worker/Pipelines/TrainingPipeline.go:27` creates Attr CSV, launches child Python, derives GeoPremium from Attr residual JSON, merges Phase2 CSV, runs Phase2 with SHAP streaming, persists predictions/statistics/Elo and explanations. Generic segment support exists but only residence/properties sell+rent uses this native explainability publication.

`RunTrainingAndPersist` (`Worker/explainability/training.go:63`) preflights schema with LIMIT0, creates transaction-local temporary tables, starts Python with2-hour timeout, parses strict `SHAP_RECORD\t` JSON lines. Stderr is inherited. Scanner max16MiB/token; property rows are COPY-staged in batches500. Python SHAP batches256, interaction sample max2000/seed42. Protocol parser validates metadata/counts/features/additivity, rejecting malformed/unexpected stdout. Go formats progress, not the browser.

After successful process exit and stream finalization, Go persists legacy predictions/stats/Elo through a callback using pool operations outside the SHAP transaction, then replaces the transaction's explanation+importance+interaction tables and model stats in one SQL transaction (`training.go:293`). Old per-transaction explanations survive precommit failure, but file outputs and legacy derived tables may already have changed. No global transaction spans files, both sell/rent jobs, DB and both process memories.

Go then preloads/smoke-tests new ONNX sessions (`main.go:496,550`), validates native UBJ+manifest locally and computes each UBJ hash (`explainability/artifacts.go:40`), POSTs transaction names and expected hashes to Python (no files or paths uploaded), verifies response, and publishes Go session pointers (`main.go:406,416,429,563`). Python switches first. Timeout after Python switches can leave services on different generations; old Go sessions remain if reload verification fails. Estimate requests should not overlap global training in the agreed demo workflow. Hash is integrity/identity, not signature/trust or encryption.

Concurrency: application RWMutex permits concurrent readers, blocking model swap until in-flight predictor users finish. Add holds read lock across whole flow, so slow geocoding/DB/SHAP delays swap. Each `ModelPredictor` separately has exclusive mutex for shared ONNX tensor buffers, so same-model predictions serialize. Other routes that do not use those locks continue concurrently. `PredictPrice` NOW validates pointer/session, exact feature length and finite inputs/output (`Worker/testing/CalculatePrice.go:189`). The older memory statement that copy permits wrong sizes is stale.

Startup: `main.go:451,473` initializes DB and ONNX Runtime from working-directory `onnxengine.dll`. Both final ONNX files exist -> load them into Go (Python startup is separate). If either is missing -> startup calls full training, which is expensive and mutating. No separate health/start-everything route, graceful shutdown, HTTP server timeout config or deployment orchestration is implemented. Paths assume Go working directory.

## Market analysis is a separate temporary workflow

Area autocomplete (`AreaSearch.go:28`) loads admin_boundaries levels6..10 into mutex-protected one-hour catalogue; min2 normalized characters, max8 matches. Accent-insensitive Greek/Latin conversion, common aliases including Piraeus/Pireaus. Municipality levels7/8 ranked before level6, then9/10. Bboxes are strings `[south,north,west,east]`, not `[lon,lat,...]`. Coverage limited to stored boundaries. No geocoder request here; Add still uses Nominatim.

`AnalysisPipeline.go:35`: query area, cluster, compute statistics/outliers, then prepare full feature rows and call `/TrainAndExplain` separately for sell/rent groups. Empty area returns empty arrays/no Python call. Group<10 is skipped with warning; upstream model failure can preserve clusters/other report. Client timeout4minutes and request context propagation. Cancellation does not necessarily stop Python CPU work. `PrepareDataForMicroService.go:204`.

K-means uses standardized `[square meters, age, renovation age, floor]`, not latitude/longitude. Tries k=1..min(10,n), chooses elbow via maximum perpendicular distance to endpoint chord of WCSS curve, reruns selected k. Deterministic farthest-point initial seeds for a fixed input order, <=100 iterations, retains unused centers; duplicates can collapse groups. Input SQL/map iteration not globally ordered, so reproducibility is not universally promised. `Kmeans.go:53,101,180`.

Cluster stats split each cluster by transaction (thus potentially two summaries per cluster). Population means/stds and six z-scores: age, floor, asking, estimate, GeoPremium, Elo. Extremes are top/bottom CEIL(5% of group) by Elo, constrained not to overlap; NOT exactly five properties (`AnalysisActions.go:114`). Models are request-local (see Python audit for algorithms/results) and do not replace global artifacts/tables. Response point data contain17 fields, no lat/lon, so front-end plot must not be called geographic map.

SQL composition uses fragile exact string replacements (`MarketAnalysis/DatabaseActions.go:23,174`). Current-year age is used for points but model encoders use hardcoded2026. Generic numeric type switches can silently zero unexpected types. These are limitations, not documented guarantees.

## Database design for an honest ER diagram

Schema-only dump: 91 CREATE TABLE statements including 11 valuation-history child partitions (default + Feb-Nov2026); roughly80 logical tables. PostgreSQL17.5 source/pg_dump18 recorded in header; no dataset rows. Required extensions: pg_cron, pg_partman, pg_trgm, PostGIS, unaccent (`schema_dump.sql:26,65,79,93,107`). Restore expects roles/owners including postgres and mcp_readonly; isolated restore not verified here.

Active core:

| Entity | Key / purpose | SQL evidence |
| --- | --- | --- |
| properties | UID PK; item/subtype/transaction; optional total asking, listing dates/source | schema_dump.sql:1793 |
| base_attributes | property_uid PK; size, zone, array views/facades | :820 |
| building_details | property_uid PK; years, amenities, energy, floor array | :863 |
| residential_specifics | property_uid PK; bedrooms,pool,garden | :2055 |
| property_geo_data | property_uid PK; lat/lon,address; generated Point4326 | :1917 |
| user_credentials | UID PK; legacy email/hash/admin; email NOT unique | :2214,3554 |
| user_properties | composite(property_uid,user_uid); ownership link, published flag,time | :2263,3578 |
| estimation_data | uid PK; attr/final prediction, residual, GeoPremium,std/z | :1258,3146 |
| properties_elo | property_uid PK; integer Elo | :1905,3346 |
| property_images | composite(property_uid,position); exactly one URL/bytea source | :1933; migration001 |
| admin_boundaries | stored named administrative geometries; lookup/search | :707 |
| global_stats | stat_name PK; mean/std plus optional latest model metadata | :1357 |
| properties_analysis_sell/rent | property_uid PK; 30 double impact columns, baseline,prediction,error,hash,time | :1813,1859; migration002 |
| feature_importance_sell/rent | feature_name PK; mean abs/signed,percent,rank,hash,time | :1277,1297 |
| features_interactions_sell/rent | pair PK; distinct-feature check, mean abs/signed,rank,hash,time | :1317,1337 |

Relationships: normalized attribute/geography tables use property_uid PK+FK -> properties, ON DELETE CASCADE, mostly DEFERRABLE INITIALLY DEFERRED (`schema_dump.sql:5020..5068`). Images and property SHAP have cascading property FKs. Elo has property FK but no cascade. Important: `estimation_data.uid` and `user_properties.property_uid` are logical joins but have no property FK in this supplied dump; do not draw all lines as enforced FKs. `user_properties.user_uid` DOES reference legacy user_credentials (`:5148`). Newer users has uniqueemail and links auth_providers/orgs/subscriptions; it is a separate identity design, not auto-synchronized with this API.

Spatial indexing includes GiST on Point4326 and geography cast (`:4078,4086`) and administrative geometries (`:3700`). Point geometry generated from `ST_MakePoint(geo_lon,geo_lat)`, so longitude first. Radius queries cast to geography to measure metres; bbox query stays4326 degree geometry. Do not confuse SQL market-premium policies with residual-based application GeoPremium.

The schema's billing/subscriptions/contracts/credits/reports/business/scheduler/valuation-policy/history tables and functions are broader inherited design. They are not proof those feature workflows exist in this Go app. No RLS policies detected. Broad public schema privilege at:5316 and readonly-role table/default SELECT grants include sensitive tables; least privilege needs review.

Migration001 adds images and portfolio index. Migration002 creates the six SHAP tables and extends global_stats, latest-only (no revision archive). The dump already includes both. Existing older DBs need migrations; no migration runner is present in startup. Preflight detects missing002 before expensive Phase2 explanation work, but not before earlier Attr phase.

## Security, reproducibility and deployment limits to disclose

- Repository guidance reports exposed database secret and JWT signing literal: never reproduce values; treat as compromised, rotate and move to secret configuration before deployment. JWT creation usesHS256; verifier accepts HMAC family, not exact method. Middleware checks UID but does not apply admin roles. Cookie Secure+HttpOnly+SameSiteStrict,24h; localhost special handling is browser-specific, do not claim Secure cookies can never work on localhost HTTP.
- Public /train,/test,/predict and caller-controlled signup admin are unsafe. SignUp/SignIn lack consistent request limits/method enforcement/rate limiting. No email verification delivery, password reset/session revocation/logout endpoint or admin implementation.
- SQL parameterization and ownership predicates are real safeguards, not proof of production security. No RLS; Browse intentionally widens property read access, ignores published flag. Some internal error details still exposed by Add/auth/predict handlers.
- Age feature construction hardcodes2026 (`types/TypeConverter.go:185`); listing month reference frozen2026-05-04 (`:19`); do not describe continuously updated temporal features.
- Predictions target advertised asking €/m², NOT verified sale transactions. Global train/test split does not fully isolate upstream target-derived GeoPremium; held-out R² can be optimistic (see ML audit). No independent appraisal guarantee or causal SHAP claim.
- The project has runnable proof-of-concept functionality, not completed production operations: no deployment packaging, integrated startup orchestration, health route or load/performance certification. Test success is specific to suites and historical read-only checks, not a security or valuation-accuracy certification.

## Memory discrepancies worth correcting in documentation

1. The old blanket statement "handlers do not restrict methods/body sizes" is false for newer endpoints; use route-by-route matrix above.
2. The old claim of unchecked ONNX feature length is false in current PredictPrice.
3. "All model outputs overwritten directly" is false for staged Phase2 publication; no GLOBAL atomic transaction remains true.
4. "Offline and online GeoPremium identical" needs the candidate-selection/no-neighbor caveat above.
5. User ownership is enforced on owner detail; broader Browse access is separately intentional, not the same old unrestricted owner endpoint bug.
6. Missing-image null/omitted differs by route, and empty SHAP is allowed.
7. Startup missing models can retrain automatically, rather than merely returning unavailable.

