package marketanalysis

import (
	d "Ptixiaki/Worker/database"
	"context"
	"fmt"

	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

func (ap AnalysisPipeline) GetAreaDataPoints(con *pgxpool.Pool, ctx context.Context, bbox BoundingBox) ([]DataPoint, error) {
	db := d.Rules{}
	db.Init()
	query := db.BuildQuery("residence", "properties")
	query = strings.Replace(query, "SELECT", "SELECT e.predicted_price_per_sqm, e.geo_premium, p.uid as uid, p_elo.elo as elo, p.transaction_type as transaction_type,", 1)
	query = strings.Replace(query, "LEFT JOIN admin_boundaries ab ON ab.admin_level = 6 AND ST_Contains(ab.geom, pgd.geom)", "", 1)
	query = strings.Replace(query, "ab.name AS admin_name,", "", 1)
	query = strings.Replace(query, "AND ab.name IS NOT NULL", "", 1)
	query = strings.Replace(query, "AND p.transaction_type = $1", "", 1)
	query = strings.Replace(query,
		"JOIN residential_specifics rs ON p.uid = rs.property_uid",
		"JOIN residential_specifics rs ON p.uid = rs.property_uid\n JOIN estimation_data e ON p.uid = e.uid\n JOIN properties_elo p_elo ON p.uid = p_elo.property_uid",
		1)
	query += "\n AND pgd.geom && ST_MakeEnvelope($1, $2, $3, $4, 4326)"
	rows, err := con.Query(ctx, query, bbox.MinLon, bbox.MinLat, bbox.MaxLon, bbox.MaxLat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	var dataPoints []DataPoint
	currentYear := time.Now().Year()

	for _, row := range results {
		var sqm float64
		switch v := row["square_meters"].(type) {
		case float64:
			sqm = v
		case float32:
			sqm = float64(v)
		case int:
			sqm = float64(v)
		case int32:
			sqm = float64(v)
		case int64:
			sqm = float64(v)
		}

		var askingPrice float64
		switch v := row["asking_price"].(type) {
		case float64:
			askingPrice = v
		case int:
			askingPrice = float64(v)
		case int32:
			askingPrice = float64(v)
		case int64:
			askingPrice = float64(v)
		}

		var predictedPrice float64
		switch v := row["predicted_price_per_sqm"].(type) {
		case float64:
			predictedPrice = v
		case float32:
			predictedPrice = float64(v)
		}

		var geoPremium float64
		switch v := row["geo_premium"].(type) {
		case float64:
			geoPremium = v
		case float32:
			geoPremium = float64(v)
		}

		var constYear int
		switch v := row["construction_year"].(type) {
		case int:
			constYear = v
		case int32:
			constYear = int(v)
		case int64:
			constYear = int(v)
		}

		renovYear := constYear
		if row["renovation_year"] != nil {
			switch v := row["renovation_year"].(type) {
			case int:
				renovYear = v
			case int32:
				renovYear = int(v)
			case int64:
				renovYear = int(v)
			}
		}

		var floorLevel int
		if rawFloor, ok := row["floor_level"]; ok && rawFloor != nil {
			strVal := fmt.Sprintf("%v", rawFloor)

			strVal = strings.Trim(strVal, "[]{} ")

			parts := strings.Fields(strings.ReplaceAll(strVal, ",", " "))

			if len(parts) > 0 {
				last := parts[len(parts)-1]
				if f, err := strconv.ParseFloat(last, 64); err == nil {
					floorLevel = int(f)
				}
			}

		}
		var uid string
		if rawUid, ok := row["uid"]; ok && rawUid != nil {
			uid = fmt.Sprintf("%v", rawUid)
		}
		var elo float64
		if rawElo, ok := row["elo"]; ok && rawElo != nil {
			switch v := rawElo.(type) {
			case int32:
				elo = float64(v)
			case int:
				elo = float64(v)
			case int64:
				elo = float64(v)
			case float64:
				elo = v
			case float32:
				elo = float64(v)
			default:
				elo = 0 // Or handle the unexpected type appropriately
			}
		}
		var transactionType string
		if rawType, ok := row["transaction_type"]; ok && rawType != nil {
			transactionType = fmt.Sprintf("%v", rawType)
		}
		dp := DataPoint{
			SquareMeters:         sqm,
			Age:                  currentYear - constYear,
			RenovAge:             currentYear - renovYear,
			FloorLevel:           floorLevel,
			Uid:                  uid,
			GeoPremium:           geoPremium,
			AskingPricePerSqm:    askingPrice / sqm,
			EstimatedPricePerSqm: predictedPrice,
			Elo:                  elo,
			TransactionType:      transactionType,
		}
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}

func (ap *AnalysisPipeline) GetDataForMicroService(uids []string) ([]ModeledResidentialProperty, error) {
	db := d.Rules{}
	db.Init()
	query := db.BuildQuery("residence", "properties")
	query = strings.Replace(query, "SELECT", "SELECT e.geo_premium as geo_premium,", 1)
	query = strings.Replace(query, "WHERE", "WHERE p.uid = ANY($1) AND", 1)
	query = strings.Replace(query, "LEFT JOIN admin_boundaries ab ON ab.admin_level = 6 AND ST_Contains(ab.geom, pgd.geom)", "LEFT JOIN admin_boundaries ab ON ab.admin_level = 6 AND ST_Contains(ab.geom, pgd.geom) LEFT JOIN estimation_data e on e.uid = p.uid", 1)
	query = strings.Replace(query, "AND p.transaction_type = $1", "", 1)

	rows, err := ap.Db.Query(ap.Ctx, query, uids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	properties, err := pgx.CollectRows(rows, pgx.RowToStructByName[ResidentialProperty])
	if err != nil {
		return nil, err
	}
	var Properties []ModeledResidentialProperty
	for _, property := range properties {
		var Features ModeledResidentialProperty
		Features, err := EncodeProperty(property)
		if err != nil {
			return nil, err
		}
		Properties = append(Properties, Features)
	}
	return Properties, nil
}
