package explainability

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	RecordPrefix = "SHAP_RECORD\t"

	FeatureCount = 30

	DefaultServiceBaseURL = "http://localhost:5000"
	ExplainPropertyPath   = "/ExplainProperty"
	ReloadModelsPath      = "/ReloadExplainModels"

	HTTPTimeout                   = 20 * time.Second
	TrainingProcessTimeout        = 2 * time.Hour
	MaxResponseBytes              = 2 << 20
	ScannerMaxTokenBytes          = 16 << 20
	PropertyStageBatchSize        = 500
	TrainingSHAPBatchSize         = 256
	TrainingInteractionSampleSize = 2000
	TrainingInteractionSampleSeed = 42
	ValidationFraction            = 0.2
	TargetName                    = "AskingPricePerSqm"
	TargetUnit                    = "EUR_per_square_meter"

	// SHAP is calculated by the native XGBoost model. These tolerances allow
	// float32 summation noise, scaled by the magnitude of the terms being added
	// (not only the prediction, which can be close to zero after cancellation).
	AdditivityAbsTolerance = 1e-3
	AdditivityRelTolerance = 1e-5

	// ONNX conversion and native XGBoost can differ slightly because both paths
	// use float32. Allow EUR 0.10/m2 absolute or 0.01% relative error.
	ONNXNativeAbsTolerance = 0.10
	ONNXNativeRelTolerance = 1e-4
)

var expectedFeatureNames = []string{
	"SubType",
	"AdminName",
	"SquareMeters",
	"Age",
	"RenovAge",
	"NoView",
	"ViewForest",
	"ViewMountain",
	"ViewSea",
	"ViewUnlimited",
	"NoFacade",
	"FacadeCorner",
	"FacadeFrontage",
	"FacadeInterior",
	"FacadeThrough",
	"HeatType",
	"HeatFrom",
	"EnergyPerformance",
	"WC",
	"AC",
	"Elevator",
	"Parking",
	"StorageRoom",
	"AluminumFrames",
	"Bedrooms",
	"Pool",
	"Garden",
	"FloorLevel",
	"FloorCount",
	"GeoPremium",
}

var propertyImpactColumns = []string{
	"subtype_impact_per_sqm",
	"admin_name_impact_per_sqm",
	"square_meters_impact_per_sqm",
	"age_impact_per_sqm",
	"renov_age_impact_per_sqm",
	"no_view_impact_per_sqm",
	"view_forest_impact_per_sqm",
	"view_mountain_impact_per_sqm",
	"view_sea_impact_per_sqm",
	"view_unlimited_impact_per_sqm",
	"no_facade_impact_per_sqm",
	"facade_corner_impact_per_sqm",
	"facade_frontage_impact_per_sqm",
	"facade_interior_impact_per_sqm",
	"facade_through_impact_per_sqm",
	"heat_type_impact_per_sqm",
	"heat_from_impact_per_sqm",
	"energy_performance_impact_per_sqm",
	"wc_impact_per_sqm",
	"ac_impact_per_sqm",
	"elevator_impact_per_sqm",
	"parking_impact_per_sqm",
	"storage_room_impact_per_sqm",
	"aluminum_frames_impact_per_sqm",
	"bedrooms_impact_per_sqm",
	"pool_impact_per_sqm",
	"garden_impact_per_sqm",
	"floor_level_impact_per_sqm",
	"floor_count_impact_per_sqm",
	"geo_premium_impact_per_sqm",
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

type Tables struct {
	PropertyAnalysis    string
	FeatureImportance   string
	FeatureInteractions string
	StatsName           string
}

var tablesByTransaction = map[string]Tables{
	"sell": {
		PropertyAnalysis:    "properties_analysis_sell",
		FeatureImportance:   "feature_importance_sell",
		FeatureInteractions: "features_interactions_sell",
		StatsName:           "Price_stats_sell",
	},
	"rent": {
		PropertyAnalysis:    "properties_analysis_rent",
		FeatureImportance:   "feature_importance_rent",
		FeatureInteractions: "features_interactions_rent",
		StatsName:           "Price_stats_rent",
	},
}

func ExpectedFeatureNames() []string {
	return slices.Clone(expectedFeatureNames)
}

func PropertyImpactColumns() []string {
	return slices.Clone(propertyImpactColumns)
}

func TablesForTransaction(transaction string) (Tables, error) {
	tables, ok := tablesByTransaction[transaction]
	if !ok {
		return Tables{}, fmt.Errorf("unsupported transaction %q: expected sell or rent", transaction)
	}
	return tables, nil
}

func ValidateTransaction(transaction string) error {
	_, err := TablesForTransaction(transaction)
	return err
}

func IsExplainableTraining(itemType, subType, transaction string) bool {
	if itemType != "residence" || subType != "properties" {
		return false
	}
	return transaction == "sell" || transaction == "rent"
}

func ValidateFeatureNames(actual []string) error {
	if !slices.Equal(actual, expectedFeatureNames) {
		return fmt.Errorf("feature order mismatch: got %v, expected %v", actual, expectedFeatureNames)
	}
	return nil
}

func validateSHA256(value string) error {
	if !sha256Pattern.MatchString(value) {
		return fmt.Errorf("invalid model_sha256 %q", value)
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid RFC3339 timestamp %q: %w", value, err)
	}
	return parsed, nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func withinTolerance(actual, expected, absolute, relative float64) bool {
	scale := math.Max(math.Abs(actual), math.Abs(expected))
	return math.Abs(actual-expected) <= absolute+relative*scale
}

// shapAdditivityTolerance matches ml_shared/shap_validation.py. Callers check
// finite inputs and reject non-finite reconstruction/tolerance after summation.
func shapAdditivityTolerance(prediction, baseline float64, contributions []float64) float64 {
	termMagnitude := math.Abs(baseline)
	for _, contribution := range contributions {
		termMagnitude += math.Abs(contribution)
	}
	return AdditivityAbsTolerance + AdditivityRelTolerance*math.Max(math.Abs(prediction), termMagnitude)
}
