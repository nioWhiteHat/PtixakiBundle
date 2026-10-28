package training

import (
	"context"
	"fmt"
	"path/filepath"

	db "Ptixiaki/Worker/database"
	t "Ptixiaki/Worker/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Trainer struct {
}

func GenerateTrainingData(ctx context.Context, pool *pgxpool.Pool, itemType string, subType string, transactionType string) error {
	rules := db.Rules{}
	rules.Init()

	query := rules.BuildQuery(itemType, subType)
	
	filename := fmt.Sprintf("%s_%s_%s_training_data.csv", itemType, subType, transactionType)
	savePath := filepath.Join("Worker","DataCsv", filename)

	fmt.Printf("Fetching %s %s and saving to %s...\n", transactionType, subType, savePath)

	switch itemType + "_" + subType {
	case "residence_properties": // Matches the new umbrella
		data, err := db.ExecuteQuery[t.ResidentialProperty](ctx, pool, query, transactionType)
		if err != nil {
			return err
		}
		return t.ExportPropertiesToCSV(data, savePath)

	case "residence_building":
		data, err := db.ExecuteQuery[t.ResidentialBuilding](ctx, pool, query, transactionType)
		if err != nil {
			return err
		}
		return t.ExportPropertiesToCSV(data, savePath)

	case "landplot_landplot": // Updated from land_plot
		data, err := db.ExecuteQuery[t.LandPlotResidential](ctx, pool, query, transactionType)
		if err != nil {
			return err
		}
		return t.ExportPropertiesToCSV(data, savePath)

	case "landplot_agricultural": // Updated from land_agricultural
		data, err := db.ExecuteQuery[t.LandplotAgric](ctx, pool, query, transactionType)
		if err != nil {
			return err
		}
		return t.ExportPropertiesToCSV(data, savePath)

	case "commercial_default": // Matches the new umbrella
		data, err := db.ExecuteQuery[t.Commercial](ctx, pool, query, transactionType)
		if err != nil {
			return err
		}
		return t.ExportPropertiesToCSV(data, savePath)

	default:
		return fmt.Errorf("unhandled category combination: %s_%s", itemType, subType)
	}
}