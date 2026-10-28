package database

import (
	"fmt"
	"strings"
)

type Rules struct {
	// Selects
	GlobalSelect                       string
	ResidentialSelect                  string
	ResidentialSpecificsSelect         string
	LandplotSelect                     string
	LandplotResidentialSpecificsSelect string
	CommercialSelect                   string

	// Tables (JOINs)
	GlobalTables      string
	ResidentialTables string
	LandplotTables    string
	CommercialTables  string

	// Rules (WHEREs)
	GlobalRules              string
	ResidentialPropertyRules string
	ResidentialBuildingRules string
	LandplotAgricRules       string
	LandplotResRules         string
	CommercialRules          string
}

func (R *Rules) Init() {
	// ==========================================
	// SELECT STATEMENTS
	// ==========================================
	R.GlobalSelect = `
		SELECT 
			p.uid,
			p.item_type,
			p.subtype,
			ba.square_meters,
			ba.zone_value,
			pgd.geo_lat,
			pgd.geo_lon,
			ab.name AS admin_name,
			p.transaction_type,
			p.asking_price,
			p.ad_creation_date
	`

	R.ResidentialSelect = `
		, ba.facade_type,
		bd.construction_year,
		bd.renovation_year,
		bd.heat,
		bd.storage_room,
		bd.aluminum_frames,
		bd.energy_performance_certificate,
		bd.elevator,
		bd.parking
	`

	R.ResidentialSpecificsSelect = `
		, ba.view AS view_type,
		bd.heat_from,
		CASE 
			WHEN bd.bathrooms IS NOT NULL AND bd.wc IS NULL THEN bd.bathrooms
			WHEN bd.bathrooms IS NOT NULL AND bd.wc IS NOT NULL AND bd.bathrooms >= bd.wc THEN bd.bathrooms
			WHEN bd.wc IS NULL AND bd.bathrooms IS NULL THEN NULL
			ELSE 0 
		END AS bathrooms,
		CASE 
			WHEN bd.wc IS NOT NULL AND bd.bathrooms IS NULL THEN bd.wc
			WHEN bd.wc IS NOT NULL AND bd.bathrooms IS NOT NULL AND bd.wc > bd.bathrooms THEN bd.wc
			WHEN bd.wc IS NULL AND bd.bathrooms IS NULL THEN NULL
			ELSE 0 
		END AS wc,
		bd.ac,
		bd.floor_level,
		rs.bedrooms,
		rs.pool,
		rs.garden
	`

	R.LandplotSelect = `
		, ba.view AS view_type,
		ba.facade_type,
		ls.facade_meters,
		ls.buildable,
		ls.in_city_plan
	`

	R.LandplotResidentialSpecificsSelect = `
		, ls.building_ratio,
		ls.coverage_factor AS coverage_ratio
	`

	R.CommercialSelect = `
		, ba.facade_type,
		ba.marketabillity_factor,
		bd.construction_year,
		bd.renovation_year,
		bd.heat,
		bd.heat_from,
		bd.storage_room,
		CASE 
			WHEN bd.bathrooms IS NOT NULL AND bd.wc IS NULL THEN bd.bathrooms
			WHEN bd.bathrooms IS NOT NULL AND bd.wc IS NOT NULL AND bd.bathrooms >= bd.wc THEN bd.bathrooms
			WHEN bd.wc IS NULL AND bd.bathrooms IS NULL THEN NULL
			ELSE 0 
		END AS bathrooms,
		CASE 
			WHEN bd.wc IS NOT NULL AND bd.bathrooms IS NULL THEN bd.wc
			WHEN bd.wc IS NOT NULL AND bd.bathrooms IS NOT NULL AND bd.wc > bd.bathrooms THEN bd.wc
			WHEN bd.wc IS NULL AND bd.bathrooms IS NULL THEN NULL
			ELSE 0 
		END AS wc,
		bd.ac,
		bd.floor_level,
		bd.aluminum_frames,
		bd.energy_performance_certificate,
		bd.parking,
		cs.power_supply,
		cs.cargo_elevator,
		cs.loading_ramp
	`

	// ==========================================
	// TABLES (JOINS)
	// ==========================================
	R.GlobalTables = `
		FROM properties p
		LEFT JOIN base_attributes ba ON p.uid = ba.property_uid
		LEFT JOIN property_geo_data pgd ON p.uid = pgd.property_uid
		LEFT JOIN admin_boundaries ab ON ab.admin_level = 6 AND ST_Contains(ab.geom, pgd.geom)
	`

	R.ResidentialTables = `
		JOIN building_details bd ON p.uid = bd.property_uid
		JOIN residential_specifics rs ON p.uid = rs.property_uid
	`

	R.LandplotTables = `
		JOIN landplot_specifics ls ON p.uid = ls.property_uid
		JOIN valuations v ON p.uid = v.property_uid
	`

	R.CommercialTables = `
		JOIN building_details bd ON p.uid = bd.property_uid
		JOIN commercial_specifics cs ON p.uid = cs.property_uid
		JOIN valuations v ON p.uid = v.property_uid
	`

	// ==========================================
	// RULES (WHERE CLAUSES)
	// ==========================================
	R.GlobalRules = `
		WHERE 1=1
		  AND p.uid IS NOT NULL
		  AND p.item_type IS NOT NULL
		  AND p.subtype IS NOT NULL
		  AND ba.square_meters IS NOT NULL
		  AND ba.zone_value IS NOT NULL
		  AND pgd.geo_lat IS NOT NULL
		  AND pgd.geo_lon IS NOT NULL
		  AND ab.name IS NOT NULL
		  AND p.transaction_type IS NOT NULL
		  AND p.asking_price IS NOT NULL
		  AND p.asking_price > 0
		  AND p.ad_creation_date IS NOT NULL
	`

	R.ResidentialPropertyRules = `
		AND bd.construction_year IS NOT NULL 
		AND ba.square_meters > 10 
		AND p.transaction_type = $1
		AND p.item_type = 'residence'
		AND p.subtype IN ('apartment','detached_house','maisonette')
		AND bd.under_construction = false
		AND bd.construction_year <= 2026
		AND bd.construction_year >= 1900
		AND (p.transaction_type != 'sell' OR p.asking_price > 1000)
	`

	R.ResidentialBuildingRules = `
		AND bd.construction_year IS NOT NULL 
		AND ba.square_meters > 10 
		AND p.transaction_type = $1
		AND p.item_type = 'residence'
		AND p.subtype = 'building'
		AND bd.under_construction = false
		AND bd.construction_year <= 2026
		AND bd.construction_year >= 1900
		AND (p.transaction_type != 'sell' OR p.asking_price > 1000)
	`

	R.LandplotAgricRules = `
		AND ba.square_meters > 60
		AND p.transaction_type = $1
		AND p.item_type = 'landplot'
		AND p.subtype = 'agricultural_land'
		AND (p.transaction_type != 'sell' OR p.asking_price > 1000)
	`

	R.LandplotResRules = `
		AND ba.square_meters > 60
		AND (ls.building_ratio IS NOT NULL OR ls.coverage_factor IS NOT NULL)
		AND p.transaction_type = $1
		AND p.item_type = 'landplot'
		AND p.subtype = 'landplot'
		AND (p.transaction_type != 'sell' OR p.asking_price > 1000)
	`

	R.CommercialRules = `
		AND ba.square_meters > 10 
		AND p.transaction_type = $1
		AND p.item_type = 'commercial'
		AND bd.under_construction = false
		AND (bd.construction_year IS NULL OR (bd.construction_year <= 2026 AND bd.construction_year >= 1900))
		AND (p.transaction_type != 'sell' OR p.asking_price > 1000)
	`
}

func (R *Rules) BuildQuery(itemType string, subType string) string {
	var selectBuilder strings.Builder
	var tableBuilder strings.Builder
	var ruleBuilder strings.Builder

	selectBuilder.WriteString(R.GlobalSelect)
	tableBuilder.WriteString(R.GlobalTables)
	ruleBuilder.WriteString(R.GlobalRules)

	switch itemType {
	case "residence":
		tableBuilder.WriteString(R.ResidentialTables)
		selectBuilder.WriteString(R.ResidentialSelect)

		if subType == "building" {
			ruleBuilder.WriteString(R.ResidentialBuildingRules)
		} else if subType == "properties" { // Matches the new umbrella
			selectBuilder.WriteString(R.ResidentialSpecificsSelect)
			ruleBuilder.WriteString(R.ResidentialPropertyRules)
		}

	case "landplot": // Matches the new itemType umbrella
		tableBuilder.WriteString(R.LandplotTables)
		selectBuilder.WriteString(R.LandplotSelect)

		if subType == "agricultural" {
			ruleBuilder.WriteString(R.LandplotAgricRules)
		} else if subType == "landplot" { // Matches the new subType umbrella
			selectBuilder.WriteString(R.LandplotResidentialSpecificsSelect)
			ruleBuilder.WriteString(R.LandplotResRules)
		}

	case "commercial":
		tableBuilder.WriteString(R.CommercialTables)
		selectBuilder.WriteString(R.CommercialSelect)
		ruleBuilder.WriteString(R.CommercialRules)
	}

	return fmt.Sprintf("%s\n%s\n%s", selectBuilder.String(), tableBuilder.String(), ruleBuilder.String())
}

// BuildInferenceQuery permits an unpriced target while leaving the supervised
// training/comparable filters intact. AskingPrice is dropped from the model
// vector; COALESCE is only for the legacy non-nullable Go input struct, never
// persisted as a fabricated asking price or used as a training label.
func (R Rules) BuildInferenceQuery(itemType, subType string) string {
	R.GlobalSelect = strings.Replace(R.GlobalSelect, "p.asking_price,", "COALESCE(p.asking_price, 0) AS asking_price,", 1)
	R.GlobalRules = strings.ReplaceAll(R.GlobalRules, "AND p.asking_price IS NOT NULL", "")
	R.GlobalRules = strings.ReplaceAll(R.GlobalRules, "AND p.asking_price > 0", "")
	for _, rules := range []*string{&R.ResidentialPropertyRules, &R.ResidentialBuildingRules, &R.LandplotAgricRules, &R.LandplotResRules, &R.CommercialRules} {
		*rules = strings.ReplaceAll(*rules, "AND (p.transaction_type != 'sell' OR p.asking_price > 1000)", "")
	}
	return R.BuildQuery(itemType, subType)
}
