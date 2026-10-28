# Property Co Pilot frontend

React 19 + TypeScript + Vite. Includes **authentication, property browsing, portfolio, property creation/details, and area analysis**.
“Create account” switches to the signup form in the same auth card; “Sign in”
switches back. It is not a separate browser window or an overlay.

## Run locally

Use Node 22.20 or newer. From this directory:

```powershell
npm install
npm run dev
```

Open **http://localhost:5173**. The frontend can be viewed without starting any
backend. To actually sign in or create an account, run the existing Go service
on **http://localhost:8080** in another terminal using the bundle's Go/ONNX setup.
The Python microservice is not called by these authentication forms.

Vite proxies `/register/*`, `/User/*`, and `/analysis/*` to Go, so the browser makes same-origin requests
without adding CORS to the backend. The development server is localhost-only
and uses port 5173 with strict port checking. Use `localhost` consistently.

## Authentication contract

- Sign in: `POST /register/SignIn`, JSON `{email, password}`.
- Sign up: `POST /register/SignUp`, JSON `{email, password, admin: false}`.
- Both successful responses are **plain text**, not JSON. Go sets its existing
  HttpOnly session cookie, and requests use `credentials: 'include'`.
- Confirmation is checked in the browser and never sent to Go.
- Signup passwords must have 9–72 English letters/digits only. This matches Go's
  current minimum/alphanumeric rules and bcrypt's 72-byte maximum. It is the
  existing backend policy, not a newly recommended password-security policy.
- Validation errors, hidden/revealed passwords, pending submissions, sanitized
  server errors, connection failures, cancellation, and a 15-second timeout
  are handled. Duplicate submissions are blocked.
- Credentials/tokens are not saved in localStorage or sessionStorage. Passwords
  are cleared when switching forms or after successful authentication.

Successful sign-in or signup opens the workspace with **Browse selected by
default**. The hamburger opens a left sidebar with Browse, Portfolio, and
Analysis. Selecting an item updates the active selection and document title,
then closes the menu. The close button, Escape, or clicking outside also closes
it. Browse, Portfolio and Analysis are functional. The layout follows the reference's left-menu pattern with
our own slate/copper colors and dimensions, using React state and CSS only.

Navigation uses simple hash URLs: `#/browse`, `#/portfolio`, `#/analysis`, and
`#/portfolio/add`, `#/portfolio/property?uid=...`, and `#/browse/property?uid=...`. Property cards open the UID's full detail page;
the back link and browser Back work. No routing library is needed. Authentication
state is still in memory: refresh requires signing in again, then resumes the
hash route. A 401 response offers a return to sign in without exposing private
data. Session restoration, logout, admin login, edit/delete
forms, password recovery, and Google login are deferred.

## Browse

Browse loads sale listings across all areas initially. Filter by an area selected
from the two-second autocomplete, sale/rent, asking price in **€/m²**, and size in
m². Rental prices are monthly. Leave either end of a range blank for no limit.
Search applies the filters and returns to page 1; Clear filters restores defaults.

`GET /User/BrowseProperties` returns 20 overviews per page, ordered by highest
Elo with UID as a stable tie-breaker. A `has_next` flag controls Next; Previous
returns to the preceding page. Results use the same `PropertyCard` as Portfolio
inside a scrollable list. Images use the same URL/data-URL/SVG fallback.

Applied filters and the page are in the hash URL. Clicking a card uses
`GET /User/BrowseProperty?uid=...` and the shared detail page, including the saved
features, estimates/asking, Elo, GeoPremium and local/global SHAP. Back to results
restores the same filters and page. These reads require the existing sign-in
cookie but not portfolio ownership. No Python call or retraining is involved.

## Portfolio and property details

Click **Add property** on Portfolio to open `#/portfolio/add`. Enter the full
address from the suggestions, property type, area, construction year and
floors; fill in optional renovation/asking price and building features as needed.
Sale asking prices are €/m²; rental asking prices are monthly €/m². Blank asking
is omitted from the request, so it stays unknown rather than becoming zero.

Address entry reuses the same `AreaSearch` component and authenticated
`/analysis/autocomplete` endpoint as Market Analysis, with `kind=address`.
Type a street and city, then press **Enter** or **Find address** to search
Nominatim. Select the correct result to preview its latitude/longitude. New
addresses do not need an existing property in the database. Editing the text
clears the selection and disables saving. Include a house number when available;
a street-only result uses the street location rather than a specific building.

The form sends `AddressOSMType` and `AddressOSMID` with the selected canonical
address. Go resolves that stable OSM object from its server cache or Nominatim
lookup and ignores caller-supplied lat/lon. A missing or changed selection returns
HTTP 422 and asks for the address to be selected again.

The existing Go geocoder uses a 12-second HTTP timeout, one external request per
second and a bounded one-hour cache. `NOMINATIM_URL` overrides the default provider
base URL if needed. The public [Nominatim usage policy](https://operations.osmfoundation.org/policies/nominatim/)
prohibits search-as-you-type, so address lookup requires explicit submission.
The UI credits Nominatim and OpenStreetMap contributors.

The form posts the existing PascalCase feature JSON to
`POST /User/AddProperty?transaction=sell|rent`. Go returns HTTP 201 with
`{message: "Property added successfully", uid: "property_..."}` and a Location
header. Success opens `#/portfolio/property?uid=...` for that exact property,
replacing the form's history entry. Duplicate submissions are blocked and the
request is cancelled when leaving. Failed or uncertain saves suggest checking
Portfolio before trying again because Go currently commits base inputs before
finishing valuation/Elo. There is no automatic retry. Run Go for saves; Python
provides best-effort SHAP. Restart the Go development service after changing its
response contract; older binaries return no UID and cannot support the redirect.

`GET /User/Portfolio` supplies the count, total estimated value and all overview
cards. Each card shows area, address, estimated/asking price per m², Elo and
GeoPremium. The total is the backend's sum of available estimate × area values;
the endpoint does not distinguish sale values from rental values in that sum.
Missing values stay missing; a missing asking price is not shown as zero.

`GET /User/ViewProperty?uid=...` supplies the owner-only full view: saved inputs,
location, estimates and optional asking totals/per-m² values, Elo, GeoPremium,
z-score and residual. SHAP contributions are signed €/m² changes, with local
absolute-importance percentages. Global SHAP importance is a separate table.
Both show the top eight initially and can expand to every returned feature.
Baseline, explained estimate, model metadata and mismatched-model warnings are
available. This is a read-only UI; it does not calculate or regenerate SHAP.

Image URLs and base64 image data URLs are used in ordinary `<img>` elements,
never injected as HTML. Omitted/null/broken images show a local SVG placeholder.
Requests are cancelled on navigation and late results are ignored.

## Area analysis

Choose Analysis, type an area, wait two seconds, select a suggestion,
then **Analyse area**. Search uses the existing authenticated
`GET /analysis/autocomplete?q=...`. The stored area's bounding box is converted from
`[south,north,west,east]` strings to a JSON body with numeric
`MinLat,MaxLat,MinLon,MaxLon`, sent to `POST /analysis/CreateAnalysisForArea`.

The result includes separate sales/rentals views, K-means groups, all six cluster
means/standard deviations, the endpoint's top/bottom 5% Elo tails, and all property
fields/z-scores in a horizontally scrollable table (20 rows per client-side page).
The scatter plot shows area versus age. Clustering itself also uses renovation age
and floor, and is not geographic. Only the visual plot is thinned to at most 2,000
points for large responses; no table records or cluster statistics are discarded.

The fresh area models' training R², SHAP baseline, XGBoost feature importances,
SHAP feature interactions, and every returned example/contribution are displayed.
Area-model importance is **not** global SHAP importance, and its R² is in-sample,
not validation accuracy. Existing global model files are not replaced by this UI.

Run **both Go and the Python microservice** for real analysis. There is a five-minute
browser timeout and a Stop waiting button; aborting the browser request does not
guarantee cancellation of work already running in Go/Python. No automatic retries
or duplicate analysis submissions are issued. Groups with fewer than ten eligible
properties skip model training with a visible warning; their cluster data still
loads. A Python failure also preserves clusters and any other successful model.
K-means uses distinct deterministic seeds and a 100-iteration cap. Go logs the
actual failing stage; the browser receives safe error codes rather than SQL details.

### Area autocomplete

Go now searches the existing `admin_boundaries` table (levels 6–10), cached in
memory for one hour. It only offers real administrative areas, not businesses or
individual addresses. Greek names are accent-insensitive; Latin transliteration
and common city aliases are supported, including Piraeus/Pireaus. Municipalities
rank before wider regions and districts. Search coverage is limited to boundaries
already in the database; try the Greek name when an English spelling is unknown.
No Nominatim requests or environment flag are needed for administrative-area autocomplete. The same
`display_name` and `[south,north,west,east]` response shape is retained for callers.
Restart Go after updating this code; Vite alone cannot update a running Go binary.

The backend cookie retains `Secure; HttpOnly; SameSite=Strict`. Use HTTPS for
deployment; do not strip these attributes to make a deployed HTTP site work.
For local development, use a modern browser's localhost exception; if your
browser does not accept that cookie, use a trusted local HTTPS setup. A
successful HTTP response alone does not prove that the browser retained the
cookie. Live browser checks use a session the user signs into themselves;
credentials and cookies are never copied into test fixtures.

## Verification

```powershell
npm run lint
npm run typecheck
npm test
npm run build
```

The Vitest/Testing Library tests cover validation, API payloads, cookie
credentials, plain-text success, safe error messages, form switching and
focus, password visibility, mismatch errors, duplicate submission prevention,
successful login/signup into Browse, retry, credential clearing, unmount
cancellation, default selection, menu toggling, section switching, keyboard
focus, Escape, outside-click dismissal, portfolio/detail rendering, URL navigation,
all local/global SHAP rows, missing images/asking, stale-request cancellation,
debounced area search, bbox mapping, cluster filters, result pagination,
report rendering, request cancellation, duplicate prevention, browse filters,
pagination, detail/back navigation, empty/error states, and API errors.
All HTTP calls in tests are mocked: no accounts or database records are created.
All 85 tests pass, including property creation, sale/rental payloads, optional asking,
validation, encoded-UID navigation, duplicate saves, expired sessions, late responses,
address selection, coordinate preview and invalidated selections. Previous synthetic previews checked desktop/mobile layouts.
On 2026-09-27, signed-in browser checks verified real Piraeus autocomplete,
combined sale/rent, asking-price and size filters, two non-overlapping 20-row pages,
matching overview/detail prices and scores, SHAP rendering, back navigation and
the scrollable SVG-fallback list. The read-only Go database test also passed all
six filters and pagination. Browse testing does not train models or change records.
Earlier real Piraeus analysis verification is recorded in `PROJECT_MEMORY.md`.

`npm run preview` serves the production build locally. This is a preview, not a
production backend or deployment configuration. A deployed static frontend
needs a same-origin reverse proxy for `/register/*`, `/User/*`, and `/analysis/*` to Go over an HTTPS origin.
The auth/backend security issues documented in `PROJECT_MEMORY.md` still apply;
this frontend does not make the existing backend deployment-ready.

## Source map

- `src/App.tsx`: auth shell, branding, form switching, and login-to-workspace handoff.
- `src/components/AuthForm.tsx`: typed forms, input controls, and request states.
- `src/components/Workspace.tsx` / `Workspace.css`: hamburger sidebar, hash navigation, and page shell.
- `src/pages/Portfolio.tsx` / `PropertyDetail.tsx`: live portfolio and owner-only detail views.
- `src/pages/AddProperty.tsx` / `AddProperty.css` / `src/addProperty.ts`: validated property creation and saved-UID navigation.
- `src/pages/Browse.tsx` / `Browse.css`: filters, URL-backed pagination and scrollable listings.
- `src/components/PropertyCard.tsx`: shared portfolio/browse overview; `PropertyDetail` also supports browse reads.
- `src/property.ts` / `src/hooks/usePropertyData.ts`: typed contracts, formatting and cancellable API reads.
- `src/components/PropertyExplanation.tsx`: local SHAP contributions and global importance.
- `src/pages/Analysis.tsx` / `src/analysis.ts`: area analysis flow and Go response types.
- `src/components/AreaSearch.tsx` / `AreaSearch.css`: shared area/address selection; two-second area debounce and explicitly submitted Nominatim address lookup.
- `src/components/AnalysisResults.tsx`, `ClusterPlot.tsx`, `AnalysisPoints.tsx`, `AreaModelReport.tsx`: all returned cluster/model data.
- `src/auth.ts`: validation and the Go HTTP contract.
- `src/components/PropertyIllustration.tsx`: original decorative SVG artwork.
- `src/App.css` / `src/index.css`: responsive slate/copper theme and reduced-motion support.

The provided Nineframes/EIV project was read only as a visual reference. The
new design uses different colors, panel proportions, original property artwork,
and its own brand. None of that project's auth code, services, fonts, or assets
were imported. No external font or image requests are needed.
