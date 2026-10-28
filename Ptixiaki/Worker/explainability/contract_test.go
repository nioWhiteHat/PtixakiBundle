package explainability

import "testing"

func TestTablesForTransaction(t *testing.T) {
	t.Parallel()

	sell, err := TablesForTransaction("sell")
	if err != nil {
		t.Fatalf("sell tables: %v", err)
	}
	if sell.PropertyAnalysis != "properties_analysis_sell" || sell.FeatureImportance != "feature_importance_sell" || sell.FeatureInteractions != "features_interactions_sell" || sell.StatsName != "Price_stats_sell" {
		t.Fatalf("unexpected sell tables: %+v", sell)
	}

	rent, err := TablesForTransaction("rent")
	if err != nil {
		t.Fatalf("rent tables: %v", err)
	}
	if rent.PropertyAnalysis != "properties_analysis_rent" || rent.FeatureImportance != "feature_importance_rent" || rent.FeatureInteractions != "features_interactions_rent" || rent.StatsName != "Price_stats_rent" {
		t.Fatalf("unexpected rent tables: %+v", rent)
	}

	for _, invalid := range []string{"", "Sell", "lease", "sell; DROP TABLE properties"} {
		if _, err := TablesForTransaction(invalid); err == nil {
			t.Fatalf("expected transaction %q to be rejected", invalid)
		}
	}
}
