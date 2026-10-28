package elo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
)

type scoringRow struct {
	values []any
	err    error
}

func (r scoringRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("wrong scan width")
	}
	for i, v := range r.values {
		b, _ := json.Marshal(v)
		if err := json.Unmarshal(b, dest[i]); err != nil {
			return err
		}
	}
	return nil
}

type scoringRows struct {
	pgx.Rows
	data  [][]any
	index int
}

func (r *scoringRows) Next() bool {
	if r.index >= len(r.data) {
		return false
	}
	r.index++
	return true
}
func (r *scoringRows) Scan(dest ...any) error {
	return (scoringRow{values: r.data[r.index-1]}).Scan(dest...)
}
func (r *scoringRows) Close()     {}
func (r *scoringRows) Err() error { return nil }

type scoringDB struct {
	t                             *testing.T
	residual                      *float64
	geo, geoMean, geoStd          float64
	priceQueries, neighborQueries int
	missingGeo                    bool
}

func (db *scoringDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	if strings.Contains(query, "FROM estimation_data") {
		return scoringRow{values: []any{db.geo, db.residual}}
	}
	if args[0] == "Geo_stats_sell" || args[0] == "Geo_stats_rent" {
		if db.missingGeo {
			return scoringRow{err: pgx.ErrNoRows}
		}
		return scoringRow{values: []any{db.geoMean, db.geoStd}}
	}
	db.priceQueries++
	return scoringRow{values: []any{10.0, 10.0}}
}
func (db *scoringDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	db.neighborQueries++
	if db.residual == nil {
		db.t.Fatal("unpriced property must not need deal comparables")
	}
	if strings.Contains(query, "WITH candidate_pool") {
		return &scoringRows{data: [][]any{{"n1"}, {"n2"}}}, nil
	}
	if !strings.Contains(query, "residual_percentage IS NOT NULL") || !strings.Contains(query, "asking_price_per_sqm > 0") {
		db.t.Fatal("unpriced neighbors must not enter residual statistics")
	}
	return &scoringRows{data: [][]any{{"n1", -10.0}, {"n2", 10.0}}}, nil
}
func TestNoAskingEloUsesOnlyGeoPremium(t *testing.T) {
	for _, transaction := range []string{"sell", "rent"} {
		for _, test := range []struct{ geo, std, want float64 }{{18, 10, 13000}, {8, 0, 10000}, {-2, 10, 7000}} {
			db := &scoringDB{t: t, geo: test.geo, geoMean: 8, geoStd: test.std}
			elo, _, err := CalculateElo(context.Background(), db, "property", transaction)
			if err != nil || elo != test.want {
				t.Fatalf("got %g/%v, want %g", elo, err, test.want)
			}
			if db.priceQueries != 0 || db.neighborQueries != 0 {
				t.Fatal("unknown asking must bypass both deal terms")
			}
		}
	}
}
func TestPricedEloKeepsExistingThreeTermFormula(t *testing.T) {
	residual := -20.0
	db := &scoringDB{t: t, residual: &residual, geo: 18, geoMean: 8, geoStd: 10}
	elo, z, err := CalculateElo(context.Background(), db, "property", "sell")
	if err != nil || elo != 26000 || z != 1 {
		t.Fatalf("got elo=%g geoZ=%g err=%v", elo, z, err)
	}
	if db.priceQueries != 1 || db.neighborQueries != 2 {
		t.Fatal("priced property must use global and local residuals")
	}
	db.missingGeo = true
	if _, _, err := CalculateElo(context.Background(), db, "property", "sell"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("missing model stats must not silently become zero")
	}
}
