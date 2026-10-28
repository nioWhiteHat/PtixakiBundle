package UserActions

import (
	auth "Ptixiaki/Backend/Auth"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const testBrowseURL = "/User/BrowseProperties?min_lat=37.90&max_lat=38.02&min_lon=23.55&max_lon=23.70"

func TestBrowseQueryFiltersAndPagination(t *testing.T) {
	values, _ := url.ParseQuery(strings.SplitN(testBrowseURL, "?", 2)[1])
	values.Set("index", "3")
	values.Set("transaction", "rent")
	for name, value := range map[string]string{
		"construction_year_min": "1990", "construction_year_max": "2025",
		"renovation_year_min": "2010", "renovation_year_max": "2026",
		"estimated_price_per_sqm_min": "5.25", "estimated_price_per_sqm_max": "20.5",
		"elo_min": "9000", "elo_max": "18000",
	} {
		values.Set(name, value)
	}
	query, args, index, err := buildBrowseQuery(values)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{23.55, 37.90, 23.70, 38.02, "rent", 1990.0, 2025.0, 2010.0, 2026.0, 5.25, 20.5, 9000.0, 18000.0, int64(40)}
	if index != 3 || !reflect.DeepEqual(args, want) {
		t.Fatalf("index=%d args=%v, want %v", index, args, want)
	}
	for _, clause := range []string{
		"ST_MakeEnvelope($1, $2, $3, $4, 4326)", "p.transaction_type = $5", browseResidencePredicate,
		"bd.construction_year >= $6", "bd.construction_year <= $7",
		"bd.renovation_year >= $8", "bd.renovation_year <= $9",
		"ed.predicted_price_per_sqm >= $10", "ed.predicted_price_per_sqm <= $11",
		"e.elo >= $12", "e.elo <= $13", "ORDER BY e.elo DESC NULLS LAST, p.uid ASC LIMIT 21 OFFSET $14",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("missing query clause %q", clause)
		}
	}
	if strings.Contains(query, "user_properties") {
		t.Fatal("browse results must not require portfolio ownership")
	}
}

func TestBrowseQueryDefaultsAndBasicErrors(t *testing.T) {
	base, _ := url.ParseQuery(strings.SplitN(testBrowseURL, "?", 2)[1])
	query, args, index, err := buildBrowseQuery(base)
	if err != nil || index != 1 || args[4] != "sell" || args[5] != int64(0) || !strings.Contains(query, "LIMIT 21 OFFSET $6") {
		t.Fatalf("bad defaults: index=%d args=%v err=%v", index, args, err)
	}
	for _, test := range []struct{ key, value string }{
		{"min_lat", ""}, {"min_lat", "NaN"}, {"max_lon", "Inf"}, {"min_lat", "40"},
		{"min_lon", "-181"}, {"max_lat", "91"}, {"index", "0"}, {"index", "-1"}, {"index", "1.5"},
		{"index", "9223372036854775807"}, {"transaction", "sell'; DROP TABLE properties;--"},
		{"construction_year_min", "old"}, {"renovation_year_max", "2020.5"},
		{"estimated_price_per_sqm_min", "NaN"}, {"elo_min", "0 OR 1=1"},
		{"asking_price_per_sqm_min", "NaN"}, {"asking_price_per_sqm_max", "Inf"},
		{"square_meters_min", "82.5"}, {"square_meters_max", "large"},
	} {
		values, _ := url.ParseQuery(base.Encode())
		values.Set(test.key, test.value)
		if _, _, _, err := buildBrowseQuery(values); err == nil {
			t.Errorf("accepted %s=%s", test.key, test.value)
		}
	}
}

func TestBrowseAllAreasAndPriceSizeFilters(t *testing.T) {
	values := url.Values{
		"transaction":              {"rent"},
		"index":                    {"2"},
		"asking_price_per_sqm_min": {"5.25"},
		"asking_price_per_sqm_max": {"15.75"},
		"square_meters_min":        {"50"},
		"square_meters_max":        {"100"},
	}
	query, args, index, err := buildBrowseQuery(values)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{-180.0, -90.0, 180.0, 90.0, "rent", 5.25, 15.75, 50.0, 100.0, int64(20)}
	if index != 2 || !reflect.DeepEqual(args, want) {
		t.Fatalf("wrong filters: index=%d args=%v", index, args)
	}
	for _, clause := range []string{
		browseAskingPrice + " >= $6::double precision",
		browseAskingPrice + " <= $7::double precision",
		"b.square_meters >= $8::double precision",
		"b.square_meters <= $9::double precision",
		"LIMIT 21 OFFSET $10",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("missing filter %q", clause)
		}
	}
}

func TestBrowseHasNextWithoutReturningExtraListing(t *testing.T) {
	for _, rowCount := range []int{0, 19, 20, 21} {
		t.Run(fmt.Sprintf("%d rows", rowCount), func(t *testing.T) {
			db := &browseTestDB{t: t}
			for index := 0; index < rowCount; index++ {
				db.data = append(db.data, []any{
					fmt.Sprintf("property_%02d", index),
					3200.0, 2650.0, 82, "Test address", 11500, 18.0, 2004, 2019,
				})
			}
			page, err := loadBrowseProperties(context.Background(), db, "browse query", nil, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Properties) != min(rowCount, BrowsePageSize) {
				t.Fatalf("wrong page size: %d", len(page.Properties))
			}
			if page.HasNext != (rowCount > BrowsePageSize) {
				t.Fatalf("wrong has_next for %d rows: %t", rowCount, page.HasNext)
			}
		})
	}
}

type browseTestDB struct {
	t         *testing.T
	query     string
	args      []any
	data      [][]any
	images    [][]any
	noImages  bool
	err       error
	rowsError error
}

func (db *browseTestDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	if strings.Contains(query, "FROM property_images") {
		uids := args[0].([]string)
		if len(uids) != min(len(db.data), BrowsePageSize) {
			db.t.Fatal("images must be fetched only for this page")
		}
		return &detailTestRows{data: db.images}, nil
	}
	db.query, db.args = query, args
	return &detailTestRows{data: db.data, err: db.rowsError}, db.err
}

func (db *browseTestDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if !strings.Contains(query, "to_regclass") {
		db.t.Fatalf("unexpected query: %s", query)
	}
	return detailTestRow{values: []any{!db.noImages}}
}

func TestBrowseOverviewPageAndImages(t *testing.T) {
	db := &browseTestDB{t: t}
	for i := 0; i < BrowsePageSize; i++ {
		db.data = append(db.data, []any{fmt.Sprintf("property_%02d", i), 3200.5, 2650.0, 82, "Synthetic address", 11500, 18.0, 2004, 2019})
	}
	db.data[2] = []any{"property_02", nil, nil, nil, nil, nil, nil, nil, nil}
	db.images = [][]any{
		{"property_00", "https://example.com/home.jpg", nil, nil},
		{"property_01", nil, []byte("image"), "image/png"},
	}
	w := httptest.NewRecorder()
	ServeBrowseProperties(w, httptest.NewRequest("GET", testBrowseURL+"&index=2", nil), db)
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body BrowsePropertiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Index != 2 || body.PageSize != 20 || len(body.Properties) != 20 || db.args[len(db.args)-1] != int64(20) {
		t.Fatalf("wrong pagination: %+v", body)
	}
	p := body.Properties[0]
	if p.UID != "property_00" || *p.EstimatedPricePerSqm != 3200.5 || *p.AskingPricePerSqm != 2650 || *p.SquareMeters != 82 || *p.Elo != 11500 || *p.GeoPremium != 18 || *p.ConstructionYear != 2004 || *p.RenovationYear != 2019 {
		t.Fatalf("lost overview fields: %+v", p)
	}
	if *p.Image != "https://example.com/home.jpg" || !strings.HasPrefix(*body.Properties[1].Image, "data:image/png;base64,") {
		t.Fatal("wrong image representation")
	}
	if body.Properties[2].EstimatedPricePerSqm != nil || body.Properties[2].AskingPricePerSqm != nil || body.Properties[2].Elo != nil {
		t.Fatal("unknown values became zero")
	}
	if strings.Count(w.Body.String(), `"image":null`) != 18 {
		t.Fatal("missing images must be explicit null, not omitted")
	}
}

func TestBrowseEmptyPageAndHTTPFailures(t *testing.T) {
	for _, test := range []struct {
		method, url string
		db          browseTestDB
		status      int
	}{
		{"GET", testBrowseURL + "&index=50", browseTestDB{}, 200},
		{"POST", testBrowseURL, browseTestDB{}, 405},
		{"GET", "/User/BrowseProperties", browseTestDB{}, 200},
		{"GET", testBrowseURL, browseTestDB{err: errors.New("private database details")}, 500},
		{"GET", testBrowseURL, browseTestDB{rowsError: errors.New("private row error")}, 500},
	} {
		test.db.t = t
		w := httptest.NewRecorder()
		ServeBrowseProperties(w, httptest.NewRequest(test.method, test.url, nil), &test.db)
		if w.Code != test.status || strings.Contains(w.Body.String(), "private") {
			t.Fatalf("%s %s: status=%d body=%s", test.method, test.url, w.Code, w.Body.String())
		}
		if test.status == 200 && !strings.Contains(w.Body.String(), `"properties":[]`) {
			t.Fatal("empty page must be an empty array")
		}
	}
}

func TestBrowseDetailReusesFeaturesAndSHAPWithoutOwnership(t *testing.T) {
	for _, transaction := range []string{"sell", "rent"} {
		tx := &detailTestTx{t: t, transaction: transaction}
		db := &detailTestDB{tx: tx}
		r := httptest.NewRequest("GET", "/User/BrowseProperty?uid=property_test", nil)
		r = r.WithContext(context.WithValue(r.Context(), "user_uid", "not-the-owner"))
		w := httptest.NewRecorder()
		ServeBrowsePropertyDetails(w, r, db)
		var detail PropertyDetails
		if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
			t.Fatalf("status=%d body=%s: %v", w.Code, w.Body.String(), err)
		}
		if w.Code != 200 || !tx.committed || detail.Transaction != transaction || detail.Features.SquareMeters != 82 || *detail.Elo != 11500 || *detail.GeoPremium != 18 || *detail.AskingPricePerSqm != 2650 || *detail.EstimatedPricePerSqm != 3200 {
			t.Fatalf("incomplete browse detail: %+v", detail)
		}
		if detail.Explanation == nil || len(detail.Explanation.Contributions) != 30 || len(detail.FeatureImportance) != 30 || detail.Explanation.Contributions[1].ImpactPerSqm != -100 {
			t.Fatal("browse detail lost SHAP data")
		}
		if !strings.Contains(w.Body.String(), `"image":null`) {
			t.Fatal("missing detail image must be null")
		}
	}
	if strings.Contains(browsePropertyDetailsQuery, "user_properties") || !strings.Contains(browsePropertyDetailsQuery, browseResidencePredicate) {
		t.Fatal("browse detail must use the same residence scope as browse results")
	}
}

func TestBrowseDetailNotFoundAndMethods(t *testing.T) {
	for _, test := range []struct {
		method, url string
		status      int
	}{
		{"GET", "/User/BrowseProperty?uid=missing", 404},
		{"GET", "/User/BrowseProperty", 400},
		{"POST", "/User/BrowseProperty?uid=property_test", 405},
	} {
		tx := &detailTestTx{t: t, transaction: "sell"}
		r := httptest.NewRequest(test.method, test.url, nil)
		r = r.WithContext(context.WithValue(r.Context(), "user_uid", "viewer"))
		w := httptest.NewRecorder()
		ServeBrowsePropertyDetails(w, r, &detailTestDB{tx: tx})
		if w.Code != test.status || tx.committed || (test.status == 404 && tx.queries != 1) {
			t.Fatalf("%s: status=%d body=%s", test.url, w.Code, w.Body.String())
		}
	}
}

func TestBrowseAuthWrapper(t *testing.T) {
	for _, handler := range []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) {
			ServeBrowseProperties(w, r, nil)
		},
		func(w http.ResponseWriter, r *http.Request) {
			ServeBrowsePropertyDetails(w, r, nil)
		},
	} {
		for _, cookie := range []string{"", "not-a-token"} {
			r := httptest.NewRequest("GET", testBrowseURL, nil)
			if cookie != "" {
				r.AddCookie(&http.Cookie{Name: "token", Value: cookie})
			}
			w := httptest.NewRecorder()
			auth.Authenticate(handler)(w, r)
			if w.Code != 401 {
				t.Fatalf("unauthenticated request got %d", w.Code)
			}
		}
	}
}
