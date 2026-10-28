package UserActions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
)

const BrowsePageSize = 20

const browseResidencePredicate = `p.item_type = 'residence'
 AND p.subtype IN ('apartment', 'detached_house', 'maisonette')
 AND p.transaction_type IN ('sell', 'rent')`

const browseAskingPrice = `COALESCE(NULLIF(ed.asking_price_per_sqm, 0),
 NULLIF(p.asking_price, 0)::double precision / NULLIF(b.square_meters, 0))`

type BrowsePropertyOverview struct {
	PortfolioPropertyOverview
	Image            *string `json:"image"`
	ConstructionYear *int    `json:"construction_year"`
	RenovationYear   *int    `json:"renovation_year"`
}

type BrowsePropertiesResponse struct {
	Index      int64                    `json:"index"`
	PageSize   int                      `json:"page_size"`
	HasNext    bool                     `json:"has_next"`
	Properties []BrowsePropertyOverview `json:"properties"`
}

func ServeBrowseProperties(w http.ResponseWriter, r *http.Request, db propertyQueryer) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query, args, index, err := buildBrowseQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	response, err := loadBrowseProperties(r.Context(), db, query, args, index)
	if err != nil {
		log.Printf("browse properties: %v", err)
		http.Error(w, "failed to browse properties", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encode browse properties: %v", err)
	}
}

func buildBrowseQuery(values url.Values) (string, []any, int64, error) {
	index := int64(1)
	if raw := values.Get("index"); raw != "" {
		var err error
		index, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || index < 1 || index > math.MaxInt64/BrowsePageSize {
			return "", nil, 0, fmt.Errorf("index must be a positive page number")
		}
	}
	transaction := values.Get("transaction")
	if transaction == "" {
		transaction = "sell"
	}
	if transaction != "sell" && transaction != "rent" {
		return "", nil, 0, fmt.Errorf("transaction must be sell or rent")
	}
	bounds := [4]float64{-180, -90, 180, 90}
	hasArea := values.Has("min_lon") || values.Has("min_lat") || values.Has("max_lon") || values.Has("max_lat")
	for i, name := range []string{"min_lon", "min_lat", "max_lon", "max_lat"} {
		if !hasArea {
			break
		}
		value, err := strconv.ParseFloat(values.Get(name), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return "", nil, 0, fmt.Errorf("%s is required and must be a number", name)
		}
		bounds[i] = value
	}
	if bounds[0] < -180 || bounds[2] > 180 || bounds[1] < -90 || bounds[3] > 90 || bounds[0] >= bounds[2] || bounds[1] >= bounds[3] {
		return "", nil, 0, fmt.Errorf("invalid bounding box")
	}
	args := []any{bounds[0], bounds[1], bounds[2], bounds[3], transaction}
	query := `SELECT p.uid, ed.predicted_price_per_sqm, ` + browseAskingPrice + `,
 b.square_meters, geo.address_text, e.elo, ed.geo_premium,
 bd.construction_year, bd.renovation_year
 FROM properties p
 JOIN property_geo_data geo ON geo.property_uid = p.uid
 LEFT JOIN base_attributes b ON b.property_uid = p.uid
 LEFT JOIN building_details bd ON bd.property_uid = p.uid
 LEFT JOIN estimation_data ed ON ed.uid = p.uid
 LEFT JOIN properties_elo e ON e.property_uid = p.uid
 WHERE ` + browseResidencePredicate + `
 AND geo.geom && ST_MakeEnvelope($1, $2, $3, $4, 4326)
 AND p.transaction_type = $5`

	for _, filter := range []struct {
		name, column string
		integer      bool
	}{
		{"construction_year", "bd.construction_year", true},
		{"renovation_year", "bd.renovation_year", true},
		{"estimated_price_per_sqm", "ed.predicted_price_per_sqm", false},
		{"elo", "e.elo", true},
		{"asking_price_per_sqm", browseAskingPrice, false},
		{"square_meters", "b.square_meters", true},
	} {
		for _, bound := range []struct{ suffix, operator string }{{"min", ">="}, {"max", "<="}} {
			name := filter.name + "_" + bound.suffix
			if raw := values.Get(name); raw != "" {
				value, err := strconv.ParseFloat(raw, 64)
				if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || (filter.integer && value != math.Trunc(value)) {
					return "", nil, 0, fmt.Errorf("invalid %s", name)
				}
				args = append(args, value)
				query += fmt.Sprintf("\n AND %s %s $%d::double precision", filter.column, bound.operator, len(args))
			}
		}
	}
	args = append(args, (index-1)*BrowsePageSize)
	query += fmt.Sprintf("\n ORDER BY e.elo DESC NULLS LAST, p.uid ASC LIMIT 21 OFFSET $%d", len(args))
	return query, args, index, nil
}

func loadBrowseProperties(ctx context.Context, db propertyQueryer, query string, args []any, index int64) (BrowsePropertiesResponse, error) {
	response := BrowsePropertiesResponse{Index: index, PageSize: BrowsePageSize, Properties: make([]BrowsePropertyOverview, 0)}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return response, err
	}
	defer rows.Close()
	uids := make([]string, 0, BrowsePageSize)
	for rows.Next() {
		if len(response.Properties) == BrowsePageSize {
			response.HasNext = true
			break
		}
		var property BrowsePropertyOverview
		if err := rows.Scan(&property.UID, &property.EstimatedPricePerSqm, &property.AskingPricePerSqm,
			&property.SquareMeters, &property.Address, &property.Elo, &property.GeoPremium,
			&property.ConstructionYear, &property.RenovationYear); err != nil {
			return response, err
		}
		uids = append(uids, property.UID)
		response.Properties = append(response.Properties, property)
	}
	if err := rows.Err(); err != nil {
		return response, err
	}
	rows.Close()
	images, err := loadPrimaryPortfolioImages(ctx, db, uids)
	if err != nil {
		return response, err
	}
	for i := range response.Properties {
		response.Properties[i].Image = images[response.Properties[i].UID]
	}
	return response, nil
}
