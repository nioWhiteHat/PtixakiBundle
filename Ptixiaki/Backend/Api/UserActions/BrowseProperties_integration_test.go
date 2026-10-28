package UserActions

import (
	database "Ptixiaki/Database"
	"context"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestBrowsePropertiesLiveDatabase(t *testing.T) {
	if os.Getenv("BROWSE_LIVE_DB_TEST") != "1" {
		t.Skip("set BROWSE_LIVE_DB_TEST=1 for read-only database verification")
	}
	db, err := database.InitDB1()
	if err != nil {
		t.Fatal("could not connect to the configured database")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal("could not start read-only test transaction")
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()
	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '15s'"); err != nil {
		t.Fatal(err)
	}
	base := url.Values{"min_lat": {"34"}, "max_lat": {"42"}, "min_lon": {"19"}, "max_lon": {"30"}}
	search := func(values url.Values) BrowsePropertiesResponse {
		t.Helper()
		query, args, index, err := buildBrowseQuery(values)
		if err != nil {
			t.Fatal(err)
		}
		page, err := loadBrowseProperties(ctx, tx, query, args, index)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Properties) > BrowsePageSize {
			t.Fatal("database query returned more than 20 properties")
		}
		return page
	}
	allAreas := search(url.Values{})
	if len(allAreas.Properties) != BrowsePageSize || !allAreas.HasNext {
		t.Fatal("default all-area browse did not return a full page and next-page flag")
	}
	for _, transaction := range []string{"sell", "rent"} {
		base.Set("transaction", transaction)
		base.Set("index", "1")
		first := search(base)
		base.Set("index", "2")
		second := search(base)
		seen := make(map[string]bool)
		for _, page := range []BrowsePropertiesResponse{first, second} {
			if len(page.Properties) != BrowsePageSize || !page.HasNext {
				t.Fatalf("expected a full %s page with more results available", transaction)
			}
			for _, property := range page.Properties {
				if seen[property.UID] {
					t.Fatal("pages overlap within the same database snapshot")
				}
				seen[property.UID] = true
			}
		}
		if len(first.Properties) == 0 {
			t.Fatalf("no %s properties available to verify details", transaction)
		}
		detail, err := ViewBrowseProperty(db, ctx, first.Properties[0].UID)
		if err != nil {
			t.Fatal(err)
		}
		if detail.UID != first.Properties[0].UID || detail.Transaction != transaction {
			t.Fatal("browse detail does not match the selected property")
		}
		t.Logf("%s: pages contain %d/%d properties; detail loaded with %d global importance records", transaction, len(first.Properties), len(second.Properties), len(detail.FeatureImportance))
	}
	base.Set("transaction", "sell")
	base.Set("index", "1")
	for _, name := range []string{
		"construction_year", "renovation_year", "estimated_price_per_sqm",
		"elo", "asking_price_per_sqm", "square_meters",
	} {
		base.Set(name+"_min", "1")
	}
	candidates := search(base)
	if len(candidates.Properties) == 0 {
		t.Fatal("no fully scored/renovated property available to verify combined filters")
	}
	p := candidates.Properties[0]
	for name, value := range map[string]float64{
		"construction_year":       float64(*p.ConstructionYear),
		"renovation_year":         float64(*p.RenovationYear),
		"estimated_price_per_sqm": *p.EstimatedPricePerSqm,
		"elo":                     float64(*p.Elo),
		"asking_price_per_sqm":    *p.AskingPricePerSqm,
		"square_meters":           float64(*p.SquareMeters),
	} {
		raw := strconv.FormatFloat(value, 'g', -1, 64)
		base.Set(name+"_min", raw)
		base.Set(name+"_max", raw)
	}
	filtered := search(base)
	if len(filtered.Properties) == 0 {
		t.Fatal("exact inclusive filters excluded their source property")
	}
	for _, found := range filtered.Properties {
		if *found.ConstructionYear != *p.ConstructionYear || *found.RenovationYear != *p.RenovationYear || *found.EstimatedPricePerSqm != *p.EstimatedPricePerSqm || *found.Elo != *p.Elo {
			t.Fatal("database returned a property outside the supplied filters")
		}
		if *found.AskingPricePerSqm != *p.AskingPricePerSqm || *found.SquareMeters != *p.SquareMeters {
			t.Fatal("database returned a property outside the asking price or size range")
		}
	}
	t.Log("all six inclusive min/max filters, all-area search and pagination verified against PostgreSQL")
}
