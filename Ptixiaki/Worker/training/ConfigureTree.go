package training

import (
	"fmt"
	"strings"
)

type AttrTreeConfigs struct {
	ResidentialPropertyColumnsDropped []string
	ResidentialBuildingColumnsDropped []string
	LandplotDropped                   []string
	LandplotAgriDropped               []string
	CommercialDropped                 []string
	Categoricals                      []string
	ResidentialTreeParams             []string
	LandplotTreeParams                []string
	CommercialTreeParams              []string
	BasePathCsvs                      string
	BasePathOutputs                   string
}

func (t *AttrTreeConfigs) Initialize(item_type string, subtype string) {
	t.ResidentialPropertyColumnsDropped = []string{
		"UID", "TransactionType", "AskingPrice", "Distance", "IsRenovated",
		"AgeSq", "InteractionAgeDistance", "LogInteraction", "AdCreationAge",
		"ZoneValue", "GeoLat", "GeoLon",
	}

	t.ResidentialBuildingColumnsDropped = []string{
		"UID", "TransactionType", "AskingPrice", "Distance", "IsRenovated",
		"AgeSq", "InteractionAgeDistance", "LogInteraction", "AdCreationAge",
		"ZoneValue", "GeoLat", "GeoLon", "SubType", "NoView", "ViewForest",
		"ViewMountain", "ViewSea", "ViewUnlimited", "HeatFrom", "WC", "AC",
		"Bedrooms", "Pool", "Garden", "FloorLevel", "FloorCount",
	}

	t.LandplotDropped = []string{
		"UID", "TransactionType", "AskingPrice", "AdCreationAge",
		"ZoneValue", "GeoLat", "GeoLon", "SubType",
	}

	t.LandplotAgriDropped = []string{
		"UID", "TransactionType", "AskingPrice", "AdCreationAge",
		"ZoneValue", "GeoLat", "GeoLon", "SubType", "BuildingRatio",
		"CoverageFactor",
	}

	t.CommercialDropped = []string{
		"UID", "TransactionType", "AskingPrice", "AdCreationAge",
		"ZoneValue", "GeoLat", "GeoLon",
	}

	t.Categoricals = []string{
		"SubType", "AdminName", "HeatType", "HeatFrom",
		"EnergyPerformance", "PowerSupply",
	}
	t.BasePathCsvs = "Worker/DataCsv"
	t.BasePathOutputs = "Worker/Models"
	t.LandplotTreeParams = []string{"8", "30", "0.05", "500", "0.8", "0.8"}
	t.ResidentialTreeParams = []string{"8", "30", "0.05", "500", "0.8", "0.8"}
	t.CommercialTreeParams = []string{"8", "30", "0.05", "500", "0.8", "0.8"}
}

func (t *AttrTreeConfigs) BuildPythonArgs(transactionType, itemType, subType, treeType string) []string {
	itemTypeSubtype := fmt.Sprintf("%s_%s_%s", itemType, subType, transactionType)
	inputCsv := fmt.Sprintf("%s/%s_training_data.csv", t.BasePathCsvs, itemTypeSubtype)

	outJson := fmt.Sprintf("%s/JsonOutputs", t.BasePathOutputs)
	outOnnx := fmt.Sprintf("%s/TreeOutputs", t.BasePathOutputs)
	outMap := fmt.Sprintf("%s/CategoryMappings", t.BasePathOutputs)

	var dropped []string
	var params []string

	switch itemType {
	case "residence":
		params = t.ResidentialTreeParams
		if subType == "building" {
			dropped = t.ResidentialBuildingColumnsDropped
		} else {
			dropped = t.ResidentialPropertyColumnsDropped
		}
	case "landplot":
		params = t.LandplotTreeParams
		if subType == "agricultural" {
			dropped = t.LandplotAgriDropped
		} else {
			dropped = t.LandplotDropped
		}
	case "commercial":
		params = t.CommercialTreeParams
		dropped = t.CommercialDropped
	}

	return []string{
		"--input_csv", inputCsv,
		"--output_json_dir", outJson,
		"--output_onnx_dir", outOnnx,
		"--output_mappings_dir", outMap,
		"--tree_type", treeType,
		"--item_type_subtype", itemTypeSubtype,
		"--columns_dropped", strings.Join(dropped, ","),
		"--categorical_cols", strings.Join(t.Categoricals, ","),
		"--max_depth", params[0],
		"--min_child_weight", params[1],
		"--learning_rate", params[2],
		"--n_estimators", params[3],
		"--subsample", params[4],
		"--colsample_bytree", params[5],
	}
}