package UserActions

import (
	p "Ptixiaki/Backend/Api"
	e "Ptixiaki/Backend/Api/Elo"
	Gd "Ptixiaki/Backend/ExternalServices/Geo"
	calc "Ptixiaki/Worker/Pipelines"
	explain "Ptixiaki/Worker/explainability"
	pred "Ptixiaki/Worker/testing"
	t "Ptixiaki/Worker/types"
	"context"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

func InsertProperty(db *pgxpool.Pool, ctx context.Context, prop p.PropertyFeatures, geoData Gd.Place, zoneValue int, uid string, TreeAttr *t.ModelPredictor, Tree *t.ModelPredictor, transaction string) (string, error) {

	latF, err := strconv.ParseFloat(geoData.Lat, 64)
	if err != nil {
		log.Println("Error parsing latitude:", err)
		return "", err
	}

	lonF, err := strconv.ParseFloat(geoData.Lon, 64)
	if err != nil {
		log.Println("Error parsing longitude:", err)
		return "", err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var views []string
	if prop.ViewForest == 1 {
		views = append(views, "Forest")
	}
	if prop.ViewMountain == 1 {
		views = append(views, "Mountain")
	}
	if prop.ViewSea == 1 {
		views = append(views, "Sea")
	}
	if prop.ViewUnlimited == 1 {
		views = append(views, "Unlimited")
	}
	if prop.NoView == 1 {
		views = append(views, "None")
	}

	var facades []string
	if prop.FacadeCorner == 1 {
		facades = append(facades, "Corner")
	}
	if prop.FacadeFrontage == 1 {
		facades = append(facades, "Frontage")
	}
	if prop.FacadeInterior == 1 {
		facades = append(facades, "Interior")
	}
	if prop.FacadeThrough == 1 {
		facades = append(facades, "Through")
	}
	if prop.NoFacade == 1 {
		facades = append(facades, "None")
	}

	var floors []string
	for _, fl := range prop.FloorLevel {
		floors = append(floors, strconv.Itoa(fl))
	}

	var askingPrice *int
	if prop.AskingPricePerSqm != nil {
		value := int(float64(prop.SquareMeters) * *prop.AskingPricePerSqm)
		askingPrice = &value
	}
	constructionYear := prop.ConstructionYear

	var renovationYear *int
	if prop.IsRenovated == 1 {
		ry := prop.RenovYear
		renovationYear = &ry
	}
	PropertyUid := GetNewPropertyUlid()

	log.Printf("Starting transaction for PropertyUid: %s", PropertyUid)

	propQuery := `
        INSERT INTO properties (
            uid, item_type, transaction_type, asking_price, subtype, ad_creation_date
        ) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
    `
	_, err = tx.Exec(ctx, propQuery,
		PropertyUid,
		"residence",
		transaction,
		askingPrice,
		prop.SubType,
	)
	if err != nil {
		log.Printf("Failed INSERT properties: %v", err)
		return "", err
	}
	log.Printf("Success INSERT properties: uid=%s, item_type=residence, transaction=%s, has_asking_price=%t, subtype=%s", PropertyUid, transaction, askingPrice != nil, prop.SubType)

	baseQuery := `
        INSERT INTO base_attributes (
            property_uid, square_meters, facade_type, view, zone_value
        ) VALUES ($1, $2, $3, $4, $5)
    `
	_, err = tx.Exec(ctx, baseQuery,
		PropertyUid,
		prop.SquareMeters,
		facades,
		views,
		zoneValue,
	)
	if err != nil {
		log.Printf("Failed INSERT base_attributes: %v", err)
		return "", err
	}
	log.Printf("Success INSERT base_attributes: square_meters=%d, facades=%v, views=%v, zone_value=%d", prop.SquareMeters, facades, views, zoneValue)

	buildingQuery := `
		INSERT INTO building_details (
			property_uid, floor_level, construction_year, renovation_year,
			heat, heat_from, WC, ac, elevator, parking, storage_room, 
			aluminum_frames, energy_performance_certificate, under_construction
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, false
		)
	`
	_, err = tx.Exec(ctx, buildingQuery,
		PropertyUid,
		floors,
		constructionYear,
		renovationYear,
		prop.HeatType,
		prop.HeatFrom,
		prop.WC,
		prop.AC == 1,
		prop.Elevator == 1,
		prop.Parking == 1,
		prop.StorageRoom == 1,
		prop.AluminumFrames == 1,
		prop.EnergyPerformance,
	)
	if err != nil {
		log.Printf("Failed INSERT building_details: %v", err)
		return "", err
	}
	log.Printf("Success INSERT building_details: floors=%v, construction_year=%d, heat=%s", floors, constructionYear, prop.HeatType)

	geoQuery := `
        INSERT INTO property_geo_data (
            property_uid, geo_lat, geo_lon, address_text
        ) VALUES ($1, $2, $3, $4)
    `
	_, err = tx.Exec(ctx, geoQuery,
		PropertyUid,
		latF,
		lonF,
		prop.Address,
	)
	if err != nil {
		log.Printf("Failed INSERT property_geo_data: %v", err)
		return "", err
	}
	log.Printf("Success INSERT property_geo_data: lat=%f, lon=%f, address=%s", latF, lonF, prop.Address)

	residentialQuery := `
        INSERT INTO residential_specifics (
            property_uid, bedrooms, pool, garden
        ) VALUES ($1, $2, $3, $4)
    `
	_, err = tx.Exec(ctx, residentialQuery,
		PropertyUid,
		prop.Bedrooms,
		prop.Pool == 1,
		prop.Garden == 1,
	)
	if err != nil {
		log.Printf("Failed INSERT residential_specifics: %v", err)
		return "", err
	}
	log.Printf("Success INSERT residential_specifics: bedrooms=%d, pool=%v, garden=%v", prop.Bedrooms, prop.Pool == 1, prop.Garden == 1)

	userPropertyQuery := `
        INSERT INTO user_properties (
            user_uid, property_uid, is_published
        ) VALUES ($1, $2, $3)
    `
	_, err = tx.Exec(ctx, userPropertyQuery, uid, PropertyUid, false)
	if err != nil {
		log.Printf("Failed INSERT user_properties: %v", err)
		return "", err
	}
	log.Printf("Success INSERT user_properties: user_uid=%s, property_uid=%s", uid, PropertyUid)

	result := tx.Commit(ctx)
	if result != nil {
		log.Println("Transaction commit failed:", result)
		return "", result
	}
	log.Printf("Transaction COMMITTED successfully for PropertyUid: %s", PropertyUid)

	calculator := pred.Calculator{Db: db, Ctx: ctx}
	prediction, err := pred.PredictPricePipeline(PropertyUid, TreeAttr, Tree, calculator)
	if err != nil {
		log.Println("Error predicting price:", err)
		return "", err
	}
	EstimatedPrice := prediction.Price
	log.Printf("Prediction successful: EstimatedPrice=%f", EstimatedPrice)

	GeoPremium, GeoStd, err := calculator.GetGeoPremium(PropertyUid)
	if err != nil {
		log.Default().Printf("Error getting GeoPremium for uid %s: %v", PropertyUid, err)
	} else {
		log.Printf("Retrieved GeoPremium=%f, GeoStd=%f", GeoPremium, GeoStd)
	}
	ResidualPercentage := calc.CalculateOptionalResidualPercentage(prop.AskingPricePerSqm, float64(EstimatedPrice))
	if ResidualPercentage != nil {
		log.Println("Calculated ResidualPercentage:", *ResidualPercentage)
	} else {
		log.Println("No asking price supplied: residual is NULL; Elo uses GeoPremium only")
	}
	query := `insert into estimation_data(uid,geo_premium,local_pred_std,predicted_price_per_sqm,residual_percentage, asking_price_per_sqm)
    values($1,$2,$3,$4,$5, $6)`
	_, err = db.Exec(ctx, query, PropertyUid, GeoPremium, GeoStd, EstimatedPrice, ResidualPercentage, prop.AskingPricePerSqm)
	if err != nil {
		log.Default().Printf("Error inserting estimation_data for uid %s: %v", PropertyUid, err)
		return "", err
	}
	log.Println("Success INSERT estimation_data")

	Elo, geoZscore, err := e.CalculateElo(ctx, db, PropertyUid, transaction)
	if err != nil {
		log.Default().Printf("Error calculating Elo for uid %s: %v", PropertyUid, err)
		return "", err
	}
	log.Printf("Calculated Elo=%f, geoZscore=%f", Elo, geoZscore)

	query = `
    UPDATE estimation_data SET geo_premium_zscore = $1 WHERE uid = $2
    `
	_, err = db.Exec(ctx, query, geoZscore, PropertyUid)
	if err != nil {
		log.Default().Printf("Error updating estimation_data for uid %s: %v", PropertyUid, err)
		return "", err
	}
	log.Println("Success UPDATE estimation_data with geo_premium_zscore")

	query = `insert into properties_elo(property_uid,elo)values($1,$2)`
	_, err = db.Exec(ctx, query, PropertyUid, Elo)
	if err != nil {
		log.Default().Printf("Error inserting properties_elo for uid %s: %v", PropertyUid, err)
		return "", err
	}
	log.Printf("Success INSERT properties_elo: elo=%f", Elo)

	
	explanationCtx, cancelExplanation := context.WithTimeout(ctx, explain.HTTPTimeout)
	defer cancelExplanation()
	if err := explain.ExplainAndPersist(
		explanationCtx,
		db,
		explain.NewClient(),
		PropertyUid,
		transaction,
		prediction.Features,
		EstimatedPrice,
	); err != nil {
		log.Printf("property %s was added successfully but its %s explanation was not persisted: %v", PropertyUid, transaction, err)
	} else {
		log.Printf("Success UPSERT property explanation for uid=%s transaction=%s", PropertyUid, transaction)
	}

	log.Println("Property inserted completely and successfully with UID:", PropertyUid)
	return PropertyUid, result
}
func GetNewPropertyUlid() string {
	ulid := "property_" + ulid.Make().String()
	return ulid
}
