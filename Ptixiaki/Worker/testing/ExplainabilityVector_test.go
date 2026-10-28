package testing

import (
	explain "Ptixiaki/Worker/explainability"
	training "Ptixiaki/Worker/training"
	types "Ptixiaki/Worker/types"
	"os"
	"path/filepath"
	"slices"
	stdtesting "testing"
)

func TestLoadCategoryMappingsSupportsQuotedComma(t *stdtesting.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "mappings.csv")
	contents := "HeatFrom,central,1\nHeatFrom,\"fan_coil,heat_pump\",4\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	mappings, err := LoadCategoryMappings(path)
	if err != nil {
		t.Fatalf("load mappings: %v", err)
	}
	if got := mappings["HeatFrom"]["fan_coil,heat_pump"]; got != 4 {
		t.Fatalf("quoted comma mapping = %d, want 4", got)
	}
}

func TestResidentialPhase2VectorOrderIncludesElevator(t *stdtesting.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "mappings.csv")
	contents := "SubType,apartment,1\nAdminName,Athens,2\nHeatType,autonomous,3\nHeatFrom,central,4\nEnergyPerformance,A,5\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	encoded := types.ResidentialFeatures{
		SubType:           "apartment",
		AdminName:         "Athens",
		SquareMeters:      90,
		HeatType:          "autonomous",
		HeatFrom:          "central",
		EnergyPerformance: "A",
		Elevator:          1,
	}
	config := training.AttrTreeConfigs{}
	config.Initialize("residence", "properties")
	vector, err := BuildFeatureVector(encoded, 2.5, path, config.ResidentialPropertyColumnsDropped)
	if err != nil {
		t.Fatalf("build vector: %v", err)
	}
	if !slices.Equal(vector.Names, explain.ExpectedFeatureNames()) {
		t.Fatalf("feature order mismatch:\n got %v\nwant %v", vector.Names, explain.ExpectedFeatureNames())
	}
	if len(vector.Values) != explain.FeatureCount {
		t.Fatalf("feature count = %d, want %d", len(vector.Values), explain.FeatureCount)
	}
	if vector.Values[20] != 1 {
		t.Fatalf("Elevator feature = %v, want 1", vector.Values[20])
	}
	if vector.Values[29] != 2.5 {
		t.Fatalf("GeoPremium feature = %v, want 2.5", vector.Values[29])
	}
}

func TestEncodeSingleResidentialPropertyPopulatesElevator(t *stdtesting.T) {
	t.Parallel()

	elevator := true
	raw := types.ResidentialProperty{
		Residential: types.Residential{ConstructionYear: 2000, Elevator: &elevator},
	}
	encoded, err := EncodeSingleProperty(raw)
	if err != nil {
		t.Fatalf("encode property: %v", err)
	}
	features, ok := encoded.(types.ResidentialFeatures)
	if !ok {
		t.Fatalf("unexpected encoded type %T", encoded)
	}
	if features.Elevator != 1 {
		t.Fatalf("Elevator = %d, want 1", features.Elevator)
	}
}
