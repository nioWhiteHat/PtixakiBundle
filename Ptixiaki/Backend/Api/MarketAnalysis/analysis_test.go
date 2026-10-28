package marketanalysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestAreaSearchGreekEnglishAndTypo(t *testing.T) {
	areas := []searchArea{
		{name: areaSearchKey("Περιφερειακή Ενότητα Πειραιώς"), level: 6, NominatimResponse: NominatimResponse{DisplayName: "Region"}},
		{name: areaSearchKey("Δήμος Πειραιώς"), level: 7, NominatimResponse: NominatimResponse{DisplayName: "Municipality"}},
	}
	for _, query := range []string{"Pir", "Pira", "Pirae", "Piraeus", "Pireaus", "Πειραι", "ΠΕΙΡΑΙΏΣ"} {
		matches := matchAreas(areas, areaSearchKey(query))
		if len(matches) != 2 || matches[0].DisplayName != "Municipality" {
			t.Errorf("query %q: %+v", query, matches)
		}
	}
	if len(matchAreas(areas, areaSearchKey("Thessaloniki"))) != 0 {
		t.Fatal("unrelated city matched")
	}
}

func TestKMeansDegenerateDataTerminates(t *testing.T) {
	for _, n := range []int{0, 1, 2, 15} {
		points := make([]DataPoint, n)
		for i := range points {
			points[i] = DataPoint{SquareMeters: 80, Age: 20, RenovAge: 10, FloorLevel: 2}
		}
		got := CreateClusters(points, 10)
		count := 0
		for _, group := range got {
			count += len(group)
		}
		if count != n || (n > 0 && len(got) != 1) {
			t.Fatalf("%d identical rows: %v", n, got)
		}
	}
	points := []DataPoint{{SquareMeters: 20}, {SquareMeters: 100}, {SquareMeters: 25}, {SquareMeters: 105}}
	if !reflect.DeepEqual(CreateClusters(points, 10), CreateClusters(points, 10)) {
		t.Fatal("non-deterministic clustering")
	}
}

func TestAreaModelsSkipSmallGroupAndPreserveOtherResult(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req MLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if len(req.Data) != 10 || req.Data[0].TransactionType != "sell" {
			t.Error("small group was sent to Python")
		}
		_ = json.NewEncoder(w).Encode(MLResponse{Message: "Success", R2Score: .6})
	}))
	defer server.Close()
	properties := make([]ModeledResidentialProperty, 11)
	for i := range properties {
		properties[i].TransactionType = "sell"
	}
	properties[10].TransactionType = "rent"
	ap := AnalysisPipeline{Ctx: context.Background(), MLURL: server.URL}
	sell, rent, err := ap.TrainSeparateModels(properties)
	if err != nil || sell == nil || rent != nil || calls != 1 || len(ap.Warnings) != 1 || !strings.Contains(ap.Warnings[0], "1 eligible") {
		t.Fatalf("sell=%v rent=%v warnings=%v err=%v", sell, rent, ap.Warnings, err)
	}
}

func TestAreaModelFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "internal Python diagnostic", 500) }))
	defer server.Close()
	properties := make([]ModeledResidentialProperty, 10)
	for i := range properties {
		properties[i].TransactionType = "sell"
	}
	ap := AnalysisPipeline{Ctx: context.Background(), MLURL: server.URL}
	sell, _, err := ap.TrainSeparateModels(properties)
	if err != nil || sell != nil || len(ap.Warnings) == 0 || strings.Contains(ap.Warnings[0], "internal Python") {
		t.Fatal("model error must become a safe partial-result warning")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ap.Ctx = ctx
	if _, _, err = ap.TrainSeparateModels(properties); err != context.Canceled {
		t.Fatalf("cancellation not propagated: %v", err)
	}
}
