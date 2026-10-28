package pipelines

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GeoData struct {
	UID              string  `json:"UID"`
	GeoPremium       float64 `json:"GeoPremium"`
	GeoPremiumStdDev float64 `json:"GeoPremiumStdDev"`
	GeoPremiumZScore float64 `json:"GeoPremiumZScore"`
}

type PhaseData struct {
	UID                  string  `json:"UID"`
	AskingPricePerSqm    float64 `json:"AskingPricePerSqm"`
	PredictedPricePerSqm float64 `json:"PredictedPricePerSqm"`
	ResidualPercentage   float64 `json:"ResidualPercentage"`
}

type EstimationRow struct {
	UID              string
	GeoPremium       float64
	LocalPredStd     float64
	GeoPremiumZScore float64

	AskingPricePerSqm    float64
	ResidualPercentage   float64
	PredictedPricePerSqm float64
	AttrPrice            float64
}

func InsertToDb(ctx context.Context, db *pgxpool.Pool, GeoJsonPath string, Phase1predictionsPath string, Phase2predictionsPath string, Transaction string) error {
	geoBytes, err := os.ReadFile(GeoJsonPath)
	if err != nil {
		return err
	}
	var geoArr []GeoData
	if err := json.Unmarshal(geoBytes, &geoArr); err != nil {
		return err
	}

	p1Bytes, err := os.ReadFile(Phase1predictionsPath)
	if err != nil {
		return err
	}
	var p1Arr []PhaseData
	if err := json.Unmarshal(p1Bytes, &p1Arr); err != nil {
		return err
	}

	p2Bytes, err := os.ReadFile(Phase2predictionsPath)
	if err != nil {
		return err
	}
	var p2Arr []PhaseData
	if err := json.Unmarshal(p2Bytes, &p2Arr); err != nil {
		return err
	}

	rowsMap := make(map[string]*EstimationRow)

	for _, g := range geoArr {
		rowsMap[g.UID] = &EstimationRow{
			UID:              g.UID,
			GeoPremium:       g.GeoPremium,
			LocalPredStd:     g.GeoPremiumStdDev,
			GeoPremiumZScore: g.GeoPremiumZScore,
		}
	}

	for _, p1 := range p1Arr {
		if row, exists := rowsMap[p1.UID]; exists {
			row.AttrPrice = p1.PredictedPricePerSqm
		}
	}

	for _, p2 := range p2Arr {
		if row, exists := rowsMap[p2.UID]; exists {
			row.AskingPricePerSqm = p2.AskingPricePerSqm
			row.PredictedPricePerSqm = p2.PredictedPricePerSqm
			row.ResidualPercentage = p2.ResidualPercentage
		}
	}

	batch := &pgx.Batch{}
	query := `INSERT INTO estimation_data 
		(uid, geo_premium, local_pred_std, geo_premium_zscore, asking_price_per_sqm, residual_percentage, predicted_price_per_sqm, attr_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (uid) DO UPDATE SET
		geo_premium = EXCLUDED.geo_premium,
		local_pred_std = EXCLUDED.local_pred_std,
		geo_premium_zscore = EXCLUDED.geo_premium_zscore,
		asking_price_per_sqm = EXCLUDED.asking_price_per_sqm,
		residual_percentage = EXCLUDED.residual_percentage,
		predicted_price_per_sqm = EXCLUDED.predicted_price_per_sqm,
		attr_price = EXCLUDED.attr_price;`

	PropertiesArray := make([]EstimationRow, 0, len(geoArr))
	ResSum := 0.0
	for _, r := range rowsMap {
		ResSum += r.ResidualPercentage
		PropertiesArray = append(PropertiesArray, *r)
		batch.Queue(query, r.UID, r.GeoPremium, r.LocalPredStd, r.GeoPremiumZScore, r.AskingPricePerSqm, r.ResidualPercentage, r.PredictedPricePerSqm, r.AttrPrice)
	}
	ResMean := ResSum / float64(len(PropertiesArray))
	ResStdSum := 0.0
	for _, r := range PropertiesArray {
		ResStdSum += (r.ResidualPercentage - ResMean) * (r.ResidualPercentage - ResMean)
	}
	ResStd := math.Sqrt(ResStdSum / float64(len(PropertiesArray)))
	br := db.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err = br.Exec(); err != nil {
			_ = br.Close()
			return err
		}
	}
	if err = br.Close(); err != nil {
		return err
	}
	GlobalGeoMean, GlobalGeoStd, err := CalculateMathForGeoPremium(PropertiesArray)
	if err != nil {
		return err
	}
	GeoStatsname := fmt.Sprintf("Geo_stats_%s", Transaction)
	queryGlobalStats := `
		INSERT INTO global_stats (stat_name, mean, std)
		VALUES ($1, $2, $3)
		ON CONFLICT (stat_name) DO UPDATE SET
		mean = EXCLUDED.mean,
		std = EXCLUDED.std
	`
	_, err = db.Exec(ctx, queryGlobalStats, GeoStatsname, GlobalGeoMean, GlobalGeoStd)
	if err != nil {
		return err
	}
	ResStatsname := fmt.Sprintf("Price_stats_%s", Transaction)
	queryResStats := `
		INSERT INTO global_stats (stat_name, mean, std)
		VALUES ($1, $2, $3)
		ON CONFLICT (stat_name) DO UPDATE SET
		mean = EXCLUDED.mean,
		std = EXCLUDED.std
	`
	_, err = db.Exec(ctx, queryResStats, ResStatsname, ResMean, ResStd)
	if err != nil {
		return err
	}
	err = CalculateEloForProperties(ctx, db, PropertiesArray, Transaction)
	if err != nil {
		return err
	}
	return nil
}
func CalculateMathForGeoPremium(properties []EstimationRow) (float64, float64, error) {
	if len(properties) == 0 {
		return 0, 0, nil
	}
	var sum float64
	for _, prop := range properties {
		sum += prop.GeoPremium
	}
	globalGeoMean := sum / float64(len(properties))

	var squaredDiffSum float64
	for _, prop := range properties {
		squaredDiff := (prop.GeoPremium - globalGeoMean) * (prop.GeoPremium - globalGeoMean)
		squaredDiffSum += squaredDiff
	}
	globalGeoStd := math.Sqrt(squaredDiffSum / float64(len(properties)))

	return globalGeoMean, globalGeoStd, nil
}
