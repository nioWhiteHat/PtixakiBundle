package UserActions

import (
	p "Ptixiaki/Backend/Api"
	explain "Ptixiaki/Worker/explainability"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"math"
	"strconv"
	"strings"
	"time"
)

type propertyQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}
type PropertyReader interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type PropertyDetails struct {
	UID                  string                    `json:"uid"`
	Transaction          string                    `json:"transaction"`
	Features             p.PropertyFeatures        `json:"features"`
	EstimatedPricePerSqm *float64                  `json:"estimated_price_per_sqm"`
	EstimatedTotalPrice  *float64                  `json:"estimated_total_price"`
	AskingPricePerSqm    *float64                  `json:"asking_price_per_sqm,omitempty"`
	AskingTotalPrice     *float64                  `json:"asking_total_price,omitempty"`
	Elo                  *int                      `json:"elo"`
	GeoPremium           *float64                  `json:"geo_premium"`
	GeoPremiumZScore     *float64                  `json:"geo_premium_zscore"`
	ResidualPercentage   *float64                  `json:"residual_percentage"`
	Image                *string                   `json:"image,omitempty"`
	Explanation          *PropertyExplanation      `json:"explanation"`
	FeatureImportance    []GlobalFeatureImportance `json:"feature_importance"`
}
type PropertyExplanation struct {
	BaselinePricePerSqm       float64                `json:"baseline_price_per_sqm"`
	ExplainedPredictionPerSqm float64                `json:"explained_prediction_per_sqm"`
	AdditivityError           float64                `json:"additivity_error"`
	ModelSHA256               string                 `json:"model_sha256"`
	ExplainedAt               time.Time              `json:"explained_at"`
	MatchesImportanceModel    *bool                  `json:"matches_importance_model"`
	Contributions             []PropertyContribution `json:"contributions"`
}
type PropertyContribution struct {
	FeatureName               string  `json:"feature_name"`
	ImpactPerSqm              float64 `json:"impact_per_sqm"`
	LocalImportancePercentage float64 `json:"local_importance_percentage"`
}
type GlobalFeatureImportance struct {
	FeatureName          string    `json:"feature_name"`
	MeanAbsSHAPPerSqm    float64   `json:"mean_abs_shap_per_sqm"`
	MeanSignedSHAPPerSqm float64   `json:"mean_signed_shap_per_sqm"`
	ImportancePercentage float64   `json:"importance_percentage"`
	Rank                 int       `json:"rank"`
	ModelSHA256          string    `json:"model_sha256"`
	CalculatedAt         time.Time `json:"calculated_at"`
}

// A read-only snapshot keeps all returned database records consistent during
// concurrent publication. Authorization is checked before any analysis/image read.
func (u *User) ViewProperty(db PropertyReader, ctx context.Context, uid string) (PropertyDetails, error) {
	if strings.TrimSpace(uid) == "" || strings.TrimSpace(u.Uid) == "" {
		return PropertyDetails{}, pgx.ErrNoRows
	}
	return readPropertyDetails(db, ctx, propertyDetailsQuery, uid, u.Uid)
}

// Browsing is available to authenticated users, without portfolio ownership.
func ViewBrowseProperty(db PropertyReader, ctx context.Context, uid string) (PropertyDetails, error) {
	return readPropertyDetails(db, ctx, browsePropertyDetailsQuery, uid)
}

func readPropertyDetails(db PropertyReader, ctx context.Context, query string, args ...any) (PropertyDetails, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PropertyDetails{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	detail, err := loadPropertyDetails(ctx, tx, query, args...)
	if err != nil {
		return PropertyDetails{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PropertyDetails{}, err
	}
	return detail, nil
}

const propertyDetailsSelect = `
SELECT p.uid, p.transaction_type, p.subtype,
    COALESCE(b.square_meters, 0), COALESCE(b.zone_value, 0), b.view, b.facade_type,
    COALESCE(bd.construction_year, 0), bd.renovation_year,
    COALESCE(bd.heat, ''), COALESCE(bd.heat_from, ''), COALESCE(bd.wc, 0),
    COALESCE(bd.ac, false)::int, COALESCE(bd.elevator, false)::int,
    COALESCE(bd.parking, false)::int, COALESCE(bd.storage_room, false)::int,
    COALESCE(bd.aluminum_frames, false)::int,
    COALESCE(bd.energy_performance_certificate, ''), bd.floor_level,
    COALESCE(geo.geo_lat, 0), COALESCE(geo.geo_lon, 0), COALESCE(geo.address_text, ''),
    COALESCE(rs.bedrooms, 0), COALESCE(rs.pool, false)::int, COALESCE(rs.garden, false)::int,
    COALESCE(NULLIF(ed.asking_price_per_sqm, 0), NULLIF(p.asking_price, 0)::double precision / NULLIF(b.square_meters, 0)),
    ed.predicted_price_per_sqm, e.elo, ed.geo_premium, ed.geo_premium_zscore, ed.residual_percentage,
    COALESCE((SELECT ab.name FROM admin_boundaries ab
              WHERE ab.admin_level = 6 AND ST_Contains(ab.geom, geo.geom)
              ORDER BY ab.name LIMIT 1), '')
FROM properties p
LEFT JOIN base_attributes b ON b.property_uid = p.uid
LEFT JOIN building_details bd ON bd.property_uid = p.uid
LEFT JOIN property_geo_data geo ON geo.property_uid = p.uid
LEFT JOIN residential_specifics rs ON rs.property_uid = p.uid
LEFT JOIN estimation_data ed ON ed.uid = p.uid
LEFT JOIN properties_elo e ON e.property_uid = p.uid`

const propertyDetailsQuery = propertyDetailsSelect + `
JOIN user_properties up ON up.property_uid = p.uid AND up.user_uid = $2
WHERE p.uid = $1`

const browsePropertyDetailsQuery = propertyDetailsSelect + `
WHERE p.uid = $1 AND ` + browseResidencePredicate

func loadPropertyDetails(ctx context.Context, db propertyQueryer, query string, args ...any) (PropertyDetails, error) {
	d := PropertyDetails{FeatureImportance: make([]GlobalFeatureImportance, 0)}
	f := &d.Features
	var views, facades, floors []string
	var renovationYear *int
	err := db.QueryRow(ctx, query, args...).Scan(
		&d.UID, &d.Transaction, &f.SubType, &f.SquareMeters, &f.ZoneValue, &views, &facades,
		&f.ConstructionYear, &renovationYear, &f.HeatType, &f.HeatFrom, &f.WC,
		&f.AC, &f.Elevator, &f.Parking, &f.StorageRoom, &f.AluminumFrames, &f.EnergyPerformance, &floors,
		&f.GeoLat, &f.GeoLon, &f.Address, &f.Bedrooms, &f.Pool, &f.Garden,
		&d.AskingPricePerSqm, &d.EstimatedPricePerSqm, &d.Elo, &d.GeoPremium, &d.GeoPremiumZScore, &d.ResidualPercentage,
		&f.AdminName,
	)
	if err != nil {
		return PropertyDetails{}, err
	}
	f.UID, f.AskingPricePerSqm = d.UID, d.AskingPricePerSqm
	if renovationYear != nil {
		f.RenovYear, f.IsRenovated = *renovationYear, 1
	}
	viewFlags := map[string]*int{"None": &f.NoView, "Forest": &f.ViewForest, "Mountain": &f.ViewMountain, "Sea": &f.ViewSea, "Unlimited": &f.ViewUnlimited}
	for _, value := range views {
		if flag := viewFlags[value]; flag != nil {
			*flag = 1
		}
	}
	facadeFlags := map[string]*int{"None": &f.NoFacade, "Corner": &f.FacadeCorner, "Frontage": &f.FacadeFrontage, "Interior": &f.FacadeInterior, "Through": &f.FacadeThrough}
	for _, value := range facades {
		if flag := facadeFlags[value]; flag != nil {
			*flag = 1
		}
	}
	f.FloorLevel = make([]int, 0, len(floors))
	for _, floor := range floors {
		if value, err := strconv.Atoi(floor); err == nil {
			f.FloorLevel = append(f.FloorLevel, value)
		}
	}
	if f.SquareMeters > 0 {
		if d.EstimatedPricePerSqm != nil {
			total := *d.EstimatedPricePerSqm * float64(f.SquareMeters)
			d.EstimatedTotalPrice = &total
		}
		if d.AskingPricePerSqm != nil {
			total := *d.AskingPricePerSqm * float64(f.SquareMeters)
			d.AskingTotalPrice = &total
		}
	}
	tables, err := explain.TablesForTransaction(d.Transaction)
	if err != nil {
		return PropertyDetails{}, err
	}
	d.Explanation, err = loadStoredExplanation(ctx, db, tables.PropertyAnalysis, d.UID)
	if err != nil {
		return PropertyDetails{}, err
	}
	d.FeatureImportance, err = loadGlobalImportance(ctx, db, tables.FeatureImportance)
	if err != nil {
		return PropertyDetails{}, err
	}
	if d.Explanation != nil && len(d.FeatureImportance) > 0 {
		matches := true
		for _, feature := range d.FeatureImportance {
			matches = matches && feature.ModelSHA256 == d.Explanation.ModelSHA256
		}
		d.Explanation.MatchesImportanceModel = &matches
	}
	images, err := loadPrimaryPortfolioImages(ctx, db, []string{d.UID})
	if err != nil {
		return PropertyDetails{}, err
	}
	d.Image = images[d.UID]
	return d, nil
}

func loadStoredExplanation(ctx context.Context, db propertyQueryer, table, uid string) (*PropertyExplanation, error) {
	columns := explain.PropertyImpactColumns()
	for i, column := range columns {
		columns[i] = pgx.Identifier{column}.Sanitize()
	}
	query := fmt.Sprintf(`SELECT ARRAY[%s], baseline_price_per_sqm, explained_prediction_per_sqm,
        additivity_error, model_sha256, explained_at FROM %s WHERE property_uid = $1`,
		strings.Join(columns, ","), pgx.Identifier{table}.Sanitize())
	var result PropertyExplanation
	var impacts []float64
	err := db.QueryRow(ctx, query, uid).Scan(&impacts, &result.BaselinePricePerSqm, &result.ExplainedPredictionPerSqm,
		&result.AdditivityError, &result.ModelSHA256, &result.ExplainedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := explain.ExpectedFeatureNames()
	if len(impacts) != len(names) {
		return nil, fmt.Errorf("invalid stored SHAP feature count")
	}
	var totalMagnitude float64
	for _, value := range impacts {
		totalMagnitude += math.Abs(value)
	}
	result.Contributions = make([]PropertyContribution, len(names))
	for i, name := range names {
		percentage := 0.0
		if totalMagnitude > 0 {
			percentage = math.Abs(impacts[i]) / totalMagnitude * 100
		}
		result.Contributions[i] = PropertyContribution{name, impacts[i], percentage}
	}
	return &result, nil
}

func loadGlobalImportance(ctx context.Context, db propertyQueryer, table string) ([]GlobalFeatureImportance, error) {
	query := fmt.Sprintf(`SELECT feature_name, mean_abs_shap_per_sqm, mean_signed_shap_per_sqm,
        importance_percentage, importance_rank, model_sha256, calculated_at FROM %s ORDER BY importance_rank`, pgx.Identifier{table}.Sanitize())
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]GlobalFeatureImportance, 0, explain.FeatureCount)
	for rows.Next() {
		var item GlobalFeatureImportance
		if err := rows.Scan(&item.FeatureName, &item.MeanAbsSHAPPerSqm, &item.MeanSignedSHAPPerSqm, &item.ImportancePercentage, &item.Rank, &item.ModelSHA256, &item.CalculatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
