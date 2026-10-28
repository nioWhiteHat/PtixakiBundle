package testing

import (
	t "Ptixiaki/Worker/types"
	"fmt"
	"math"
	"strings"
)

func EncodeSingleProperty(raw any) (any, error) {
	switch r := raw.(type) {

	case t.ResidentialProperty:
		f := t.ResidentialFeatures{}
		f.UID = r.Global.Uid
		f.SubType = r.Global.SubType
		f.AdminName = t.ParseStr(r.Global.AdminName)
		f.TransactionType = r.Global.TransactionType
		f.AskingPrice = r.Global.AskingPrice
		f.SquareMeters = r.Global.SquareMeters
		f.ZoneValue = r.Global.ZoneValue
		f.GeoLat = r.Global.GeoLat
		f.GeoLon = r.Global.GeoLon

		if f.SquareMeters > 0 {
			f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
		}
		f.AdCreationAge = t.ParseMonthsSince(r.Global.AdCreationDate)

		f.Age = 2026 - r.Residential.ConstructionYear
		f.RenovAge = f.Age
		if r.Residential.RenovationYear != nil {
			f.RenovAge = 2026 - *r.Residential.RenovationYear
		}
		f.Distance = f.Age - f.RenovAge
		f.InteractionAgeDistance = f.Age * f.Distance
		f.AgeSq = f.Age * f.Age
		f.LogInteraction = math.Log(float64(f.Age)+1.0) * float64(f.Distance)
		if f.Age > f.RenovAge {
			f.IsRenovated = 1
		}

		if r.ResidentialPropertySpecifics.ViewType != nil {
			f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = t.ParseView(*r.ResidentialPropertySpecifics.ViewType)
		} else {
			f.NoView = 1
		}

		if r.Residential.FacadeType != nil {
			f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = t.ParseFacade(*r.Residential.FacadeType)
		} else {
			f.NoFacade = 1
		}

		f.Bedrooms = t.ParseBedrooms(r.ResidentialPropertySpecifics.BedRooms)
		if r.ResidentialPropertySpecifics.FloorLevel != nil {
			f.FloorLevel, f.FloorCount = t.ParseFloors(*r.ResidentialPropertySpecifics.FloorLevel)
		} else {
			f.FloorCount = 1
		}

		f.HeatType = t.ParseStr(r.Residential.Heat)
		f.HeatFrom = t.ParseStr(r.ResidentialPropertySpecifics.HeatFrom)
		f.EnergyPerformance = t.ParseEnergyClass(r.Residential.EnergyPerformance)
		f.WC = t.ParseWC(r.ResidentialPropertySpecifics.Bathrooms, r.ResidentialPropertySpecifics.Wc)

		f.AC = t.BoolToInt(r.ResidentialPropertySpecifics.Ac)
		f.Elevator = t.BoolToInt(r.Residential.Elevator)
		f.Pool = t.BoolToInt(r.ResidentialPropertySpecifics.Pool)
		f.Garden = t.BoolToInt(r.ResidentialPropertySpecifics.Garden)
		f.StorageRoom = t.BoolToInt(r.Residential.Storage_room)
		f.AluminumFrames = t.BoolToInt(r.Residential.AluminumFrames)
		f.Parking = t.BoolToInt(r.Residential.Parking)

		return f, nil

	case t.LandPlotResidential:
		f := t.LandplotFeatures{}
		f.UID = r.Global.Uid
		f.SubType = r.Global.SubType
		f.AdminName = t.ParseStr(r.Global.AdminName)
		f.TransactionType = r.Global.TransactionType
		f.AskingPrice = r.Global.AskingPrice
		f.SquareMeters = r.Global.SquareMeters
		f.ZoneValue = r.Global.ZoneValue
		f.GeoLat = r.Global.GeoLat
		f.GeoLon = r.Global.GeoLon

		if f.SquareMeters > 0 {
			f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
		}
		f.AdCreationAge = t.ParseMonthsSince(r.Global.AdCreationDate)

		f.BuildingRatio = -1.0
		if r.LandplotResidentialSpecifics.BuildingRatio != nil {
			f.BuildingRatio = *r.LandplotResidentialSpecifics.BuildingRatio
		}

		f.CoverageFactor = -1.0
		if r.LandplotResidentialSpecifics.CoverageRatio != nil {
			f.CoverageFactor = *r.LandplotResidentialSpecifics.CoverageRatio
		}

		f.FacadeMeters = -1
		if r.Landplot.FacadeMeters != nil {
			f.FacadeMeters = *r.Landplot.FacadeMeters
		}

		f.Buildable = t.BoolToInt(r.Landplot.Buildable)
		f.InCityPlan = t.BoolToInt(r.Landplot.InCityPlan)

		if r.Landplot.ViewType != nil {
			f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = t.ParseView(*r.Landplot.ViewType)
		} else {
			f.NoView = 1
		}

		if r.Landplot.FacadeType != nil {
			f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = t.ParseFacade(*r.Landplot.FacadeType)
		} else {
			f.NoFacade = 1
		}

		return f, nil

	case t.ResidentialBuilding:
		f := t.ResidentialFeatures{}
		f.UID = r.Global.Uid
		f.SubType = r.Global.SubType
		f.AdminName = t.ParseStr(r.Global.AdminName)
		f.TransactionType = r.Global.TransactionType
		f.AskingPrice = r.Global.AskingPrice
		f.SquareMeters = r.Global.SquareMeters
		f.ZoneValue = r.Global.ZoneValue
		f.GeoLat = r.Global.GeoLat
		f.GeoLon = r.Global.GeoLon

		if f.SquareMeters > 0 {
			f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
		}
		f.AdCreationAge = t.ParseMonthsSince(r.Global.AdCreationDate)

		f.Age = 2026 - r.Residential.ConstructionYear
		f.RenovAge = f.Age
		if r.Residential.RenovationYear != nil {
			f.RenovAge = 2026 - *r.Residential.RenovationYear
		}
		f.Distance = f.Age - f.RenovAge
		f.InteractionAgeDistance = f.Age * f.Distance
		f.AgeSq = f.Age * f.Age
		f.LogInteraction = math.Log(float64(f.Age)+1.0) * float64(f.Distance)
		if f.Age > f.RenovAge {
			f.IsRenovated = 1
		}

		if r.Residential.FacadeType != nil {
			f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = t.ParseFacade(*r.Residential.FacadeType)
		} else {
			f.NoFacade = 1
		}

		f.HeatType = t.ParseStr(r.Residential.Heat)
		f.EnergyPerformance = t.ParseEnergyClass(r.Residential.EnergyPerformance)

		f.StorageRoom = t.BoolToInt(r.Residential.Storage_room)
		f.AluminumFrames = t.BoolToInt(r.Residential.AluminumFrames)
		f.Elevator = t.BoolToInt(r.Residential.Elevator)
		f.Parking = t.BoolToInt(r.Residential.Parking)

		f.Bedrooms = -1
		f.FloorLevel = -1
		f.WC = -1
		f.NoView = -1

		return f, nil

	case t.LandplotAgric:
		f := t.LandplotFeatures{}
		f.UID = r.Global.Uid
		f.SubType = r.Global.SubType
		f.AdminName = t.ParseStr(r.Global.AdminName)
		f.TransactionType = r.Global.TransactionType
		f.AskingPrice = r.Global.AskingPrice
		f.SquareMeters = r.Global.SquareMeters
		f.ZoneValue = r.Global.ZoneValue
		f.GeoLat = r.Global.GeoLat
		f.GeoLon = r.Global.GeoLon

		if f.SquareMeters > 0 {
			f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
		}
		f.AdCreationAge = t.ParseMonthsSince(r.Global.AdCreationDate)

		f.FacadeMeters = -1
		if r.Landplot.FacadeMeters != nil {
			f.FacadeMeters = *r.Landplot.FacadeMeters
		}

		f.Buildable = t.BoolToInt(r.Landplot.Buildable)
		f.InCityPlan = t.BoolToInt(r.Landplot.InCityPlan)

		if r.Landplot.ViewType != nil {
			f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = t.ParseView(*r.Landplot.ViewType)
		} else {
			f.NoView = 1
		}

		if r.Landplot.FacadeType != nil {
			f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = t.ParseFacade(*r.Landplot.FacadeType)
		} else {
			f.NoFacade = 1
		}

		f.BuildingRatio = -1.0
		f.CoverageFactor = -1.0

		return f, nil

	case t.Commercial:
		f := t.CommercialFeatures{}
		f.UID = r.Global.Uid
		f.SubType = r.Global.SubType
		f.AdminName = t.ParseStr(r.Global.AdminName)
		f.TransactionType = r.Global.TransactionType
		f.AskingPrice = r.Global.AskingPrice
		f.SquareMeters = r.Global.SquareMeters
		f.ZoneValue = r.Global.ZoneValue
		f.GeoLat = r.Global.GeoLat
		f.GeoLon = r.Global.GeoLon

		if f.SquareMeters > 0 {
			f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
		}
		f.AdCreationAge = t.ParseMonthsSince(r.Global.AdCreationDate)

		if r.CommercialSpecifics.ConstructionYear != nil {
			f.Age = 2026 - *r.CommercialSpecifics.ConstructionYear
			f.RenovAge = f.Age
			if r.CommercialSpecifics.RenovationYear != nil {
				f.RenovAge = 2026 - *r.CommercialSpecifics.RenovationYear
			}
			f.Distance = f.Age - f.RenovAge
			f.InteractionAgeDistance = f.Age * f.Distance
			f.AgeSq = f.Age * f.Age
			f.LogInteraction = math.Log(float64(f.Age)+1.0) * float64(f.Distance)
			if f.Age > f.RenovAge {
				f.IsRenovated = 1
			}
		}

		if r.CommercialSpecifics.MarketabilityFactor != nil {
			f.MarketabillityFactor = *r.CommercialSpecifics.MarketabilityFactor
		}

		if r.CommercialSpecifics.FacadeType != nil {
			f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = t.ParseFacade(*r.CommercialSpecifics.FacadeType)
		} else {
			f.NoFacade = 1
		}

		if r.CommercialSpecifics.FloorLevel != nil {
			f.FloorLevel, f.FloorCount = t.ParseFloors(*r.CommercialSpecifics.FloorLevel)
		} else {
			f.FloorCount = 1
		}

		f.HeatType = t.ParseStr(r.CommercialSpecifics.Heat)
		f.HeatFrom = t.ParseStr(r.CommercialSpecifics.HeatFrom)

		f.EnergyPerformance = t.ParseEnergyClass(r.CommercialSpecifics.EnergyPerformance)
		f.WC = t.ParseWC(r.CommercialSpecifics.Bathrooms, r.CommercialSpecifics.Wc)

		f.AC = t.BoolToInt(r.CommercialSpecifics.Ac)
		f.StorageRoom = t.BoolToInt(r.CommercialSpecifics.Storage_room)
		f.AluminumFrames = t.BoolToInt(r.CommercialSpecifics.AluminumFrames)
		f.Parking = t.BoolToInt(r.CommercialSpecifics.Parking)
		f.CargoElevator = t.BoolToInt(r.CommercialSpecifics.CargoElevator)
		f.LoadingRamp = t.BoolToInt(r.CommercialSpecifics.LoadingRamp)

		if r.CommercialSpecifics.PowerSupply != nil && len(*r.CommercialSpecifics.PowerSupply) > 0 {
			f.PowerSupply = strings.Join(*r.CommercialSpecifics.PowerSupply, ",")
		} else {
			f.PowerSupply = "Unknown"
		}

		return f, nil

	default:
		return nil, fmt.Errorf("unsupported property type for encoding: %T", raw)
	}
}
