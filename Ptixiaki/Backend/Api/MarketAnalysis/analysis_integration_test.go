package marketanalysis

import (
	database "Ptixiaki/Database"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAnalysisLiveDatabase(t *testing.T) {
	if os.Getenv("ANALYSIS_LIVE_TEST") != "1" {
		t.Skip("set ANALYSIS_LIVE_TEST=1")
	}
	db, err := database.InitDB1()
	if err != nil {
		t.Fatal("database connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	catalog := AreaCatalog{}
	areas, err := catalog.Search(ctx, db, "Pireaus")
	if err != nil {
		t.Fatal("could not read area catalogue")
	}
	if len(areas) == 0 || areas[0].DisplayName != "Δήμος Πειραιώς, Greece" {
		t.Fatalf("municipality not ranked first: %+v", areas)
	}
	coordinates := make([]float64, 4)
	for index, coordinate := range areas[0].BoundingBox {
		coordinates[index], err = strconv.ParseFloat(coordinate, 64)
		if err != nil {
			t.Fatal(err)
		}
	}
	boundingBox := BoundingBox{
		MinLat: coordinates[0],
		MaxLat: coordinates[1],
		MinLon: coordinates[2],
		MaxLon: coordinates[3],
	}
	t.Logf("Pireaus autocomplete -> %s; bounds %+v", areas[0].DisplayName, boundingBox)
	pipeline := AnalysisPipeline{Db: db, Ctx: ctx}
	smallArea := BoundingBox{
		MinLat: 40.6510784,
		MaxLat: 40.6511784,
		MinLon: 22.9443282,
		MaxLon: 22.9444282,
	}
	points, clusters, sell, rent, err := pipeline.RunPipeline(smallArea)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || len(clusters) != 1 || sell != nil || rent != nil || len(pipeline.Warnings) == 0 {
		t.Fatal("screenshot's one-property area must return clusters and an insufficient-data warning")
	}
	t.Log("Screenshot bounds now return one cluster and an explicit model-size warning, not HTTP 500")
	if os.Getenv("ANALYSIS_LIVE_ML_TEST") != "1" {
		return
	}
	points, clusters, sell, rent, err = pipeline.RunPipeline(boundingBox)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) < 10 || len(clusters) == 0 || sell == nil || rent == nil {
		t.Fatalf("incomplete real Piraeus analysis: points=%d clusters=%d warnings=%v", len(points), len(clusters), pipeline.Warnings)
	}
	if len(sell.Examples) != 10 || len(rent.Examples) != 10 {
		t.Fatal("expected ten real explanations per transaction")
	}
	if _, err = json.Marshal([]any{points, clusters, sell, rent}); err != nil {
		t.Fatal(err)
	}
	t.Logf("Full Piraeus pipeline: %d points, %d cluster/transaction groups; sell R2=%.4f rent R2=%.4f; 10 explanations each", len(points), len(clusters), sell.R2Score, rent.R2Score)
}
