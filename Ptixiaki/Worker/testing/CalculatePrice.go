package testing

import (
	db "Ptixiaki/Worker/database"
	explain "Ptixiaki/Worker/explainability"
	tr "Ptixiaki/Worker/training"
	"Ptixiaki/Worker/types"
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Discovery struct {
	ItemType    string
	SubType     string
	Transaction string
}

type Calculator struct {
	Ctx context.Context
	Db  *pgxpool.Pool

	BaseQuery string
}

func (c *Calculator) GetPropData(uid string) (Discovery, error) {
	var d Discovery

	query := "SELECT item_type, subtype, transaction_type FROM properties WHERE uid = $1"

	err := c.Db.QueryRow(c.Ctx, query, uid).Scan(&d.ItemType, &d.SubType, &d.Transaction)
	if err != nil {
		return Discovery{}, fmt.Errorf("failed to get prop data for uid %s: %v", uid, err)
	}

	return d, nil
}

func (c *Calculator) GetAllPropertyData(uid string, Ptype Discovery) (any, error) {
	rules := db.Rules{}
	rules.Init()

	// 1. Build the base query from your rules
	query := rules.BuildInferenceQuery(Ptype.ItemType, Ptype.SubType)
	c.BaseQuery = query

	// 2. Append the specific UID filter
	// Your rules use $1 for the transaction type, so uid becomes $2
	query += "\nAND p.uid = $2"

	// 3. Execute the query
	rows, err := c.Db.Query(c.Ctx, query, Ptype.Transaction, uid)
	log.Printf("Executing query for uid %s, transaction %s: %s", uid, Ptype.Transaction, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed for uid %s: %v", uid, err)
	}
	defer rows.Close()

	// 4. Route to the correct struct based on discovery metadata
	switch Ptype.ItemType {
	case "residence":
		if Ptype.SubType == "building" {
			data, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[types.ResidentialBuilding])
			return data, err
		} else { // "properties"
			data, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[types.ResidentialProperty])
			return data, err
		}

	case "landplot":
		if Ptype.SubType == "agricultural" {
			data, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[types.LandplotAgric])
			return data, err
		} else { // "landplot"
			data, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[types.LandPlotResidential])
			return data, err
		}

	case "commercial":
		data, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[types.Commercial])
		return data, err

	default:
		return nil, fmt.Errorf("unsupported item type discovered: %s", Ptype.ItemType)
	}
}
func (c *Calculator) PrepareInferencePayload(uid string) ([]float32, error) {
	// 1. Discover Property Metadata
	discovery, err := c.GetPropData(uid)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %v", err)
	}

	// 🛠️ THE FIX: Translate raw DB subtypes into your Training Pipeline Umbrellas
	if discovery.ItemType == "residence" {
		if discovery.SubType == "apartment" || discovery.SubType == "maisonette" || discovery.SubType == "detached_house" {
			discovery.SubType = "properties"
		}
	} else if discovery.ItemType == "landplot" {
		if discovery.SubType != "agricultural" { // Assuming anything else is standard landplot
			discovery.SubType = "landplot"
		}
	} else if discovery.ItemType == "commercial" {
		discovery.SubType = "default"
	}

	// 2. Fetch Raw Database Model
	rawDbData, err := c.GetAllPropertyData(uid, discovery)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch db data: %v", err)
	}

	// 3. Encode into ML Features Struct
	encodedStruct, err := EncodeSingleProperty(rawDbData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode property: %v", err)
	}

	// 4. Initialize your central Tree Configs to get exact paths and dropped columns
	config := tr.AttrTreeConfigs{}
	config.Initialize(discovery.ItemType, discovery.SubType)

	// Format exactly how Python saved them: e.g., "residence_properties_sell"
	itemSubTrans := fmt.Sprintf("%s_%s_%s", discovery.ItemType, discovery.SubType, discovery.Transaction)

	// 5. Build Paths

	GeoPremium, _, err := c.GetGeoPremium(uid)

	if err != nil {
		return nil, fmt.Errorf("failed to get geo premium1: %v", err)
	}
	// CRITICAL: We use Phase2 category mappings because this is the final inference tree
	mappingsFileName := fmt.Sprintf("CategoryMappings_Phase2_%s.txt", itemSubTrans)
	mappingsPath := filepath.Join(config.BasePathOutputs, "CategoryMappings", mappingsFileName)

	if err != nil {
		return nil, fmt.Errorf("failed to get geo premium1: %v", err)
	}

	// 7. Extract the exact dropped columns based on the type
	var pythonDroppedCols []string
	switch discovery.ItemType {
	case "residence":
		if discovery.SubType == "building" {
			pythonDroppedCols = config.ResidentialBuildingColumnsDropped
		} else {
			pythonDroppedCols = config.ResidentialPropertyColumnsDropped
		}
	case "landplot":
		if discovery.SubType == "agricultural" {
			pythonDroppedCols = config.LandplotAgriDropped
		} else {
			pythonDroppedCols = config.LandplotDropped
		}
	case "commercial":
		pythonDroppedCols = config.CommercialDropped
	default:
		return nil, fmt.Errorf("unsupported item type for dropping columns: %s", discovery.ItemType)
	}

	// 8. Build the Final Float32 Tensor Array
	featureVector, err := BuildFeatureVector(
		encodedStruct,
		GeoPremium,
		mappingsPath,
		pythonDroppedCols,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build final feature array: %v", err)
	}
	if explain.IsExplainableTraining(discovery.ItemType, discovery.SubType, discovery.Transaction) {
		if err := explain.ValidateFeatureNames(featureVector.Names); err != nil {
			return nil, fmt.Errorf("invalid residence/property Phase2 feature contract: %w", err)
		}
	}

	return featureVector.Values, nil
}

func PredictPrice(features []float32, mp *types.ModelPredictor) (float32, error) {
	if mp == nil || mp.Session == nil {
		return 0, fmt.Errorf("model predictor is not loaded")
	}
	if len(features) != len(mp.InputData) {
		return 0, fmt.Errorf("feature count mismatch: got %d, model expects %d", len(features), len(mp.InputData))
	}
	for i, value := range features {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return 0, fmt.Errorf("feature %d is not finite", i)
		}
	}
	mp.Mu.Lock()
	defer mp.Mu.Unlock()

	copy(mp.InputData, features)

	err := mp.Session.Run()
	if err != nil {
		return 0, err
	}

	prediction := mp.OutputData[0]
	if math.IsNaN(float64(prediction)) || math.IsInf(float64(prediction), 0) {
		return 0, fmt.Errorf("model returned a non-finite prediction")
	}
	return prediction, nil

}

type FeatureVector struct {
	Names  []string
	Values []float32
}

func BuildFeatureArray(encodedData any, geoPremium float64, mappingsFilePath string, droppedColumns []string) ([]float32, error) {
	vector, err := BuildFeatureVector(encodedData, geoPremium, mappingsFilePath, droppedColumns)
	if err != nil {
		return nil, err
	}
	return vector.Values, nil
}

func BuildFeatureVector(encodedData any, geoPremium float64, mappingsFilePath string, droppedColumns []string) (FeatureVector, error) {
	catMap, err := LoadCategoryMappings(mappingsFilePath)
	if err != nil {
		return FeatureVector{}, fmt.Errorf("failed to load mappings: %v", err)
	}

	fmt.Printf("\n=== LOADED CATEGORY MAP ===\n")
	for col, m := range catMap {
		fmt.Printf("Column [%s]:\n", col)
		for k, v := range m {
			fmt.Printf("  '%s' -> %d\n", k, v)
		}
	}
	fmt.Printf("===========================\n")

	fmt.Printf("\n=== RAW ENCODED STRUCT ===\n%+v\n==========================\n", encodedData)

	dropMap := make(map[string]bool)
	for _, col := range droppedColumns {
		dropMap[col] = true
	}

	dropMap["UID"] = true
	dropMap["AskingPricePerSqm"] = true
	dropMap["AskingPrice"] = true

	val := reflect.ValueOf(encodedData)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return FeatureVector{}, fmt.Errorf("expected a struct for encodedData, got %v", val.Kind())
	}

	var features []float32
	var featureNames []string

	fmt.Println("\n--- ONNX Feature Payload Dump ---")

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := val.Type().Field(i).Name

		if dropMap[fieldName] {
			continue
		}

		var featureVal float32

		switch field.Kind() {
		case reflect.String:
			strVal := strings.TrimSpace(field.String())

			if colMap, hasCol := catMap[fieldName]; hasCol {
				if mappedInt, exists := colMap[strVal]; exists {
					featureVal = float32(mappedInt)
				} else {
					featureVal = -1.0
					fmt.Printf("WARNING: Category '%s' not found in column '%s'\n", strVal, fieldName)
				}
			} else {
				featureVal = -1.0
				fmt.Printf("WARNING: Column '%s' not found in mappings file\n", fieldName)
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			featureVal = float32(field.Int())

		case reflect.Float32, reflect.Float64:
			featureVal = float32(field.Float())

		case reflect.Bool:
			if field.Bool() {
				featureVal = 1.0
			} else {
				featureVal = 0.0
			}

		default:
			return FeatureVector{}, fmt.Errorf("unsupported field type %v for field %s", field.Kind(), fieldName)
		}

		fmt.Printf("%02d | %-25s : %v (Raw: %v)\n", len(features), fieldName, featureVal, field.Interface())
		featureNames = append(featureNames, fieldName)
		features = append(features, featureVal)
	}

	fmt.Printf("%02d | %-25s : %v\n", len(features), "GeoPremium", float32(geoPremium))
	featureNames = append(featureNames, "GeoPremium")
	features = append(features, float32(geoPremium))

	fmt.Println("------------------------------------")

	return FeatureVector{Names: featureNames, Values: features}, nil
}
