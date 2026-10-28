package UserActions

import (
	explain "Ptixiaki/Worker/explainability"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type detailTestRow struct {
	values []any
	err    error
}

func (r detailTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan width %d != %d", len(dest), len(r.values))
	}
	for i, value := range r.values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(encoded, dest[i]); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}

type detailTestRows struct {
	pgx.Rows
	data  [][]any
	index int
	err   error
}

func (r *detailTestRows) Next() bool {
	if r.index >= len(r.data) {
		return false
	}
	r.index++
	return true
}
func (r *detailTestRows) Scan(dest ...any) error {
	return (detailTestRow{values: r.data[r.index-1]}).Scan(dest...)
}
func (r *detailTestRows) Close()     {}
func (r *detailTestRows) Err() error { return r.err }

type detailTestDB struct {
	tx      *detailTestTx
	options pgx.TxOptions
}

func (db *detailTestDB) BeginTx(_ context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	db.options = options
	return db.tx, nil
}

type detailTestTx struct {
	pgx.Tx
	t                                     *testing.T
	transaction                           string
	noAsking, noExplanation, noImageTable bool
	imageURL                              *string
	imageData                             []byte
	mime                                  *string
	importanceHash                        string
	dataErr                               error
	committed, rolledBack                 bool
	queries                               int
}

func (tx *detailTestTx) Commit(context.Context) error   { tx.committed = true; return nil }
func (tx *detailTestTx) Rollback(context.Context) error { tx.rolledBack = true; return nil }
func (tx *detailTestTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	tx.queries++
	switch {
	case query == propertyDetailsQuery || query == browsePropertyDetailsQuery:
		if len(args) == 0 || args[0] != "property_test" {
			return detailTestRow{err: pgx.ErrNoRows}
		}
		if query == propertyDetailsQuery && (len(args) != 2 || args[1] != "owner") {
			return detailTestRow{err: pgx.ErrNoRows}
		}
		if query == browsePropertyDetailsQuery && len(args) != 1 {
			tx.t.Fatal("browse detail should need only the property UID")
		}
		if tx.dataErr != nil {
			return detailTestRow{err: tx.dataErr}
		}
		var asking, residual any = 2650.0, -20.0
		if tx.noAsking {
			asking, residual = nil, nil
		}
		return detailTestRow{values: []any{
			"property_test", tx.transaction, "apartment", 82, 1600, []string{"Sea", "Mountain"}, []string{"Corner", "Frontage"},
			2004, 2019, "autonomous", "gas", 2, 1, 1, 1, 1, 1, "B", []string{"3", "4"},
			40.63, 22.94, "Synthetic address", 2, 0, 1, asking, 3200.0, 11500, 18.0, 0.5, residual, "Synthetic region",
		}}
	case strings.Contains(query, "FROM \"properties_analysis_"+tx.transaction+"\""):
		if len(args) != 1 || args[0] != "property_test" {
			tx.t.Fatal("explanation must be scoped to authorized UID")
		}
		if tx.noExplanation {
			return detailTestRow{err: pgx.ErrNoRows}
		}
		impacts := make([]float64, explain.FeatureCount)
		impacts[0], impacts[1] = 300, -100
		return detailTestRow{values: []any{impacts, 3000.0, 3200.0, 0.0, strings.Repeat("a", 64), time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)}}
	case strings.Contains(query, "to_regclass"):
		return detailTestRow{values: []any{!tx.noImageTable}}
	default:
		tx.t.Fatalf("unexpected QueryRow: %s", query)
		return detailTestRow{err: errors.New("unexpected query")}
	}
}
func (tx *detailTestTx) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	tx.queries++
	switch {
	case strings.Contains(query, "FROM \"feature_importance_"+tx.transaction+"\""):
		rows := make([][]any, 0, explain.FeatureCount)
		hash := tx.importanceHash
		if hash == "" {
			hash = strings.Repeat("a", 64)
		}
		for i, name := range explain.ExpectedFeatureNames() {
			rows = append(rows, []any{name, 10.0, -2.0, 100.0 / 30, i + 1, hash, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)})
		}
		return &detailTestRows{data: rows}, nil
	case strings.Contains(query, "FROM property_images"):
		if tx.noImageTable {
			tx.t.Fatal("must not query absent image table")
		}
		if len(args) != 1 || len(args[0].([]string)) != 1 || args[0].([]string)[0] != "property_test" {
			tx.t.Fatal("images must be scoped to authorized UID")
		}
		if tx.imageURL == nil && tx.imageData == nil {
			return &detailTestRows{}, nil
		}
		return &detailTestRows{data: [][]any{{"property_test", tx.imageURL, tx.imageData, tx.mime}}}, nil
	default:
		tx.t.Fatalf("unexpected Query: %s", query)
		return nil, errors.New("unexpected query")
	}
}

func detailRequest(method, url, body, user string, tx *detailTestTx) (*httptest.ResponseRecorder, *detailTestDB) {
	r := httptest.NewRequest(method, url, strings.NewReader(body))
	if user != "" {
		r = r.WithContext(context.WithValue(r.Context(), "user_uid", user))
	}
	w := httptest.NewRecorder()
	db := &detailTestDB{tx: tx}
	ServePropertyDetails(w, r, db)
	return w, db
}

func TestPropertyDetailsFullResponse(t *testing.T) {
	for _, transaction := range []string{"sell", "rent"} {
		t.Run(transaction, func(t *testing.T) {
			url := "https://example.com/property.jpg"
			tx := &detailTestTx{t: t, transaction: transaction, imageURL: &url}
			w, db := detailRequest("GET", "/User/ViewProperty?uid=property_test", "", "owner", tx)
			if w.Code != 200 {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			var detail PropertyDetails
			if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
				t.Fatal(err)
			}
			if !tx.committed || db.options.AccessMode != pgx.ReadOnly || db.options.IsoLevel != pgx.RepeatableRead {
				t.Fatal("expected consistent read-only transaction")
			}
			if detail.Transaction != transaction || detail.UID != "property_test" || *detail.Elo != 11500 || *detail.GeoPremium != 18 {
				t.Fatalf("missing scoring fields: %+v", detail)
			}
			if *detail.AskingPricePerSqm != 2650 || *detail.AskingTotalPrice != 217300 || *detail.EstimatedPricePerSqm != 3200 || *detail.EstimatedTotalPrice != 262400 {
				t.Fatal("wrong per-sqm/total prices")
			}
			f := detail.Features
			if f.UID != detail.UID || f.SubType != "apartment" || f.SquareMeters != 82 || f.ConstructionYear != 2004 || f.RenovYear != 2019 || f.IsRenovated != 1 || f.Address != "Synthetic address" {
				t.Fatalf("lost entered fields: %+v", f)
			}
			if f.ViewSea != 1 || f.ViewMountain != 1 || f.NoView != 0 || f.FacadeCorner != 1 || f.FacadeFrontage != 1 || f.NoFacade != 0 {
				t.Fatal("lost view/facade flags")
			}
			if f.HeatType != "autonomous" || f.HeatFrom != "gas" || f.EnergyPerformance != "B" || f.WC != 2 || f.AC != 1 || f.Elevator != 1 || f.Parking != 1 || f.StorageRoom != 1 || f.AluminumFrames != 1 || f.Bedrooms != 2 || f.Pool != 0 || f.Garden != 1 {
				t.Fatal("lost amenities")
			}
			if len(f.FloorLevel) != 2 || f.FloorLevel[0] != 3 || f.FloorLevel[1] != 4 {
				t.Fatal("lost floor levels")
			}
			if detail.Image == nil || *detail.Image != url {
				t.Fatal("missing image")
			}
			if detail.Explanation == nil || len(detail.Explanation.Contributions) != 30 || len(detail.FeatureImportance) != 30 {
				t.Fatal("missing SHAP data")
			}
			if !*detail.Explanation.MatchesImportanceModel || detail.Explanation.BaselinePricePerSqm != 3000 {
				t.Fatal("wrong explanation metadata")
			}
			if detail.Explanation.Contributions[0].LocalImportancePercentage != 75 || detail.Explanation.Contributions[1].LocalImportancePercentage != 25 || detail.Explanation.Contributions[1].ImpactPerSqm != -100 {
				t.Fatal("wrong local SHAP magnitude/sign")
			}
		})
	}
}

func TestPropertyDetailsOptionalFieldsAndModelMismatch(t *testing.T) {
	for _, missingImageTable := range []bool{false, true} {
		tx := &detailTestTx{t: t, transaction: "sell", noAsking: true, noExplanation: true, noImageTable: missingImageTable}
		w, _ := detailRequest("GET", "/User/ViewProperty?uid=property_test", "", "owner", tx)
		if w.Code != 200 {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		for _, absent := range []string{"image", "asking_price_per_sqm", "asking_total_price"} {
			if _, ok := body[absent]; ok {
				t.Fatalf("expected %s to be omitted", absent)
			}
		}
		if _, ok := body["features"].(map[string]any)["AskingPricePerSqm"]; ok {
			t.Fatal("fabricated asking input")
		}
		if body["explanation"] != nil || body["residual_percentage"] != nil || body["elo"] == nil {
			t.Fatal("optional fields must remain unknown, without hiding Elo")
		}
	}
	mime := "image/png"
	tx := &detailTestTx{t: t, transaction: "sell", imageData: []byte("bytes"), mime: &mime, importanceHash: strings.Repeat("b", 64)}
	w, _ := detailRequest("GET", "/User/ViewProperty?uid=property_test", "", "owner", tx)
	var detail PropertyDetails
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if w.Code != 200 || detail.Image == nil || !strings.HasPrefix(*detail.Image, "data:image/png;base64,") {
		t.Fatal("missing binary image data URL")
	}
	if detail.Explanation == nil || *detail.Explanation.MatchesImportanceModel {
		t.Fatal("must expose a model mismatch")
	}
}

func TestPropertyDetailsHTTPAndOwnership(t *testing.T) {
	for _, test := range []struct {
		method, url, body, user string
		status                  int
	}{
		{"GET", "/User/ViewProperty?uid=property_test", "", "other-user", 404},
		{"GET", "/User/ViewProperty?uid=unknown", "", "owner", 404},
		{"GET", "/User/ViewProperty", "", "owner", 400},
		{"GET", "/User/ViewProperty?uid=property_test", "", "", 401},
		{"DELETE", "/User/ViewProperty?uid=property_test", "", "owner", 405},
		{"POST", "/User/ViewProperty", `{"PropertyUid":"property_test"}`, "owner", 200},
	} {
		tx := &detailTestTx{t: t, transaction: "sell"}
		w, _ := detailRequest(test.method, test.url, test.body, test.user, tx)
		if w.Code != test.status {
			t.Fatalf("%+v got %d: %s", test, w.Code, w.Body.String())
		}
		if test.status == 404 && (tx.queries != 1 || tx.committed) {
			t.Fatal("unauthorized property must not trigger analysis/image reads")
		}
	}
	tx := &detailTestTx{t: t, transaction: "sell", dataErr: errors.New("private database details")}
	w, _ := detailRequest("GET", "/User/ViewProperty?uid=property_test", "", "owner", tx)
	if w.Code != 500 || strings.Contains(w.Body.String(), "private database") || tx.committed {
		t.Fatal("database failure must be rolled back and sanitized")
	}
	if !strings.Contains(propertyDetailsQuery, "up.user_uid = $2") || !strings.Contains(propertyDetailsQuery, "WHERE p.uid = $1") {
		t.Fatal("ownership predicates must remain parameterized")
	}
}
