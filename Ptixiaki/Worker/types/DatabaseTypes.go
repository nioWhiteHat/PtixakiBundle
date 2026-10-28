package types

import "time"

type Global struct {
	Uid             string     `db:"uid"`
	ItemType        string     `db:"item_type"`
	SubType         string     `db:"subtype"`
	SquareMeters    int        `db:"square_meters"`
	ZoneValue       int        `db:"zone_value"`
	GeoLat          float64    `db:"geo_lat"`
	GeoLon          float64    `db:"geo_lon"`
	AdminName       *string    `db:"admin_name"`
	TransactionType string     `db:"transaction_type"`
	AskingPrice     int        `db:"asking_price"`
	AdCreationDate  *time.Time `db:"ad_creation_date"`
}

type Residential struct {
	FacadeType        *[]string `db:"facade_type"`
	ConstructionYear  int       `db:"construction_year"`
	RenovationYear    *int      `db:"renovation_year"`
	Heat              *string   `db:"heat"`
	Storage_room      *bool     `db:"storage_room"`
	AluminumFrames    *bool     `db:"aluminum_frames"`
	EnergyPerformance *string   `db:"energy_performance_certificate"`
	Elevator          *bool     `db:"elevator"`
	Parking           *bool     `db:"parking"`
}

type ResidentialPropertySpecifics struct {
	ViewType   *[]string `db:"view_type"`
	HeatFrom   *string   `db:"heat_from"`
	Bathrooms  *int      `db:"bathrooms"`
	Wc         *int      `db:"wc"`
	Ac         *bool     `db:"ac"`
	FloorLevel *[]string `db:"floor_level"`
	BedRooms   *int      `db:"bedrooms"`
	Pool       *bool     `db:"pool"`
	Garden     *bool     `db:"garden"`
}

type ResidentialProperty struct {
	Global
	Residential
	ResidentialPropertySpecifics
}

type ResidentialBuilding struct {
	Global
	Residential
}

type Landplot struct {
	ViewType     *[]string `db:"view_type"`
	FacadeType   *[]string `db:"facade_type"`
	FacadeMeters *int      `db:"facade_meters"`
	Buildable    *bool     `db:"buildable"`
	InCityPlan   *bool     `db:"in_city_plan"`
}

type LandplotResidentialSpecifics struct {
	BuildingRatio *float64 `db:"building_ratio"`
	CoverageRatio *float64 `db:"coverage_ratio"`
}

type LandplotAgric struct {
	Global
	Landplot
}

type LandPlotResidential struct {
	Global
	Landplot
	LandplotResidentialSpecifics
}

type CommercialSpecifics struct {
	FacadeType          *[]string `db:"facade_type"`
	MarketabilityFactor *float64  `db:"marketabillity_factor"`
	ConstructionYear    *int      `db:"construction_year"`
	RenovationYear      *int      `db:"renovation_year"`
	Heat                *string   `db:"heat"`
	HeatFrom            *string   `db:"heat_from"`
	Storage_room        *bool     `db:"storage_room"`
	Bathrooms           *int      `db:"bathrooms"`
	Wc                  *int      `db:"wc"`
	Ac                  *bool     `db:"ac"`
	FloorLevel          *[]string `db:"floor_level"`
	AluminumFrames      *bool     `db:"aluminum_frames"`
	EnergyPerformance   *string   `db:"energy_performance_certificate"`
	Parking             *bool     `db:"parking"`
	PowerSupply         *[]string `db:"power_supply"`
	CargoElevator       *bool     `db:"cargo_elevator"`
	LoadingRamp         *bool     `db:"loading_ramp"`
}

type Commercial struct {
	Global
	CommercialSpecifics
}
