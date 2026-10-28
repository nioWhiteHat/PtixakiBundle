package pipelines

import (
	db "Ptixiaki/Worker/database"
	"context"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EloData struct {
	Uid            string
	Elo            float64
	DealValue      float64
	GeoZscoreValue float64
	LocalDealValue float64
}

func CalculateEloForProperties(ctx context.Context, pool *pgxpool.Pool, Properties []EstimationRow, transaction string) error {
	StdOfResiduals, MeanOfResiduals := CalculateMathForDealValue(Properties)

	PropertiesMap := make(map[string]EstimationRow, len(Properties))
	var allUIDs []string
	for _, prop := range Properties {
		PropertiesMap[prop.UID] = prop
		allUIDs = append(allUIDs, prop.UID)
	}

	r := db.Rules{}
	r.Init()
	baseQuery := r.BuildQuery("residence", "properties")

	// DEBUG: This will reveal if BuildQuery is injecting the %! error
	fmt.Println("--- DEBUG BASEQUERY ---")
	fmt.Println(baseQuery)
	fmt.Println("-----------------------")

	neighborMap := make(map[string][]string)
	batchSize := 5000
	totalProps := len(allUIDs)

	for i := 0; i < totalProps; i += batchSize {
		end := i + batchSize
		if end > totalProps {
			end = totalProps
		}
		batchUIDs := allUIDs[i:end]

		query := fmt.Sprintf(`
			WITH candidate_pool AS (
				%s
			)
			SELECT target_ids.uid, cp.uid
			FROM unnest($2::text[]) AS target_ids(uid)
			JOIN property_geo_data target_geo ON target_geo.property_uid = target_ids.uid
			CROSS JOIN LATERAL (
				SELECT cp.uid
				FROM candidate_pool cp
				JOIN property_geo_data cp_geo ON cp.uid = cp_geo.property_uid
				WHERE ST_DWithin(target_geo.geom::geography, cp_geo.geom::geography, 150)
				  AND cp.uid != target_ids.uid
				ORDER BY target_geo.geom::geography <-> cp_geo.geom::geography
				LIMIT 20
			) cp;
		`, baseQuery)

		// Notice we are passing `transaction` first ($1), and `batchUIDs` second ($2)
		rows, err := pool.Query(ctx, query, transaction, batchUIDs)
		if err != nil {
			return err
		}

		for rows.Next() {
			var targetUID, neighborUID string
			if err := rows.Scan(&targetUID, &neighborUID); err != nil {
				rows.Close()
				return err
			}
			neighborMap[targetUID] = append(neighborMap[targetUID], neighborUID)
		}
		rows.Close()

		log.Printf("Fetched neighbors batch %d to %d", i+1, end)
	}

	EloList := make([]EloData, 0, len(Properties))

	for _, prop := range Properties {
		neighbors := neighborMap[prop.UID]
		neighboursDataList := make([]EstimationRow, 0, len(neighbors))

		for _, nUID := range neighbors {
			if nData, exists := PropertiesMap[nUID]; exists {
				neighboursDataList = append(neighboursDataList, nData)
			}
		}

		LocalStd, LocalMean := CalculateMathForDealValue(neighboursDataList)
		DealValue := CalculateZScore(prop.ResidualPercentage, MeanOfResiduals, StdOfResiduals)

		LocalDealValue := CalculateZScore(prop.ResidualPercentage, LocalMean, LocalStd)

		Elo := CalculateEloScore(DealValue, LocalDealValue, prop.GeoPremiumZScore)

		EloList = append(EloList, EloData{
			Uid:            prop.UID,
			DealValue:      DealValue,
			GeoZscoreValue: prop.GeoPremiumZScore,
			LocalDealValue: LocalDealValue,
			Elo:            Elo,
		})
	}

	log.Println("Starting final Elo batch insert...")

	insertQuery := `
		INSERT INTO properties_elo (property_uid, elo)
		VALUES ($1, $2)
		ON CONFLICT (property_uid) DO UPDATE SET
		elo = EXCLUDED.elo;
	`

	batch := &pgx.Batch{}
	for _, data := range EloList {
		if data.Uid != "" {
			batch.Queue(insertQuery, data.Uid, data.Elo)
		}
	}

	results := pool.SendBatch(ctx, batch)
	for range EloList {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return err
		}
	}
	results.Close()

	name := fmt.Sprintf("Price_stats_%s", transaction)
	statQuery := `
		INSERT INTO global_stats (stat_name, mean, std)
		VALUES ($1, $2, $3)
		ON CONFLICT (stat_name) DO UPDATE SET
		mean = EXCLUDED.mean,
		std = EXCLUDED.std;
	`
	_, err := pool.Exec(ctx, statQuery, name, MeanOfResiduals, StdOfResiduals)
	if err != nil {
		return err
	}

	log.Println("Done.")
	return nil
}

func CalculateMathForDealValue(Properties []EstimationRow) (float64, float64) {
	if len(Properties) == 0 {
		return 0, 0
	}

	var SumOfResiduals float64
	var SumOfSquaredResiduals float64

	for _, prop := range Properties {
		SumOfResiduals += prop.ResidualPercentage
	}
	MeanOfResiduals := SumOfResiduals / float64(len(Properties))

	for _, prop := range Properties {
		diff := prop.ResidualPercentage - MeanOfResiduals
		SumOfSquaredResiduals += diff * diff
	}

	StdOfResiduals := math.Sqrt(SumOfSquaredResiduals / float64(len(Properties)))

	return StdOfResiduals, MeanOfResiduals
}

func CalculateResidualPercentage(askingPricePerSqm float64, estimatedPricePerSqm float64) float64 {
	if askingPricePerSqm == 0 {
		return 0
	}
	return (askingPricePerSqm - estimatedPricePerSqm) / askingPricePerSqm * 100
}

func CalculateOptionalResidualPercentage(askingPricePerSqm *float64, estimatedPricePerSqm float64) *float64 {
	if askingPricePerSqm == nil {
		return nil
	}
	value := CalculateResidualPercentage(*askingPricePerSqm, estimatedPricePerSqm)
	return &value
}

func CalculateZScore(value float64, mean float64, std float64) float64 {
	if std == 0 {
		return 0
	}
	return (value - mean) / std
}

func CalculateEloScore(globalResidualZ float64, localResidualZ float64, geoPremiumZ float64) float64 {
	return 10000 - 3000*globalResidualZ - 2000*localResidualZ + 3000*geoPremiumZ
}

func GetData(ctx context.Context, pool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, transaction string, uid string) ([]string, error) {
	r := db.Rules{}
	r.Init()
	baseQuery := r.BuildQuery("residence", "properties")

	query := fmt.Sprintf(`
		WITH candidate_pool AS (
			%s
		),
		target_prop AS (
			SELECT geom::geography AS geog
			FROM property_geo_data
			WHERE property_uid = $2
		)
		SELECT cp.uid
		FROM property_geo_data cp_geo
		JOIN candidate_pool cp ON cp.uid = cp_geo.property_uid
		CROSS JOIN target_prop t
		WHERE cp_geo.property_uid != $2 
		  AND ST_DWithin(cp_geo.geom::geography, t.geog, 150)
		ORDER BY cp_geo.geom::geography <-> t.geog
		LIMIT 20;
	`, baseQuery)

	rows, err := pool.Query(ctx, query, transaction, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []string
	for rows.Next() {
		var neighborUid string
		if err := rows.Scan(&neighborUid); err != nil {
			return nil, err
		}
		candidates = append(candidates, neighborUid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}
