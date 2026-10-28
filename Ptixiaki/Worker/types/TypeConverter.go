package types

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func ParseMonthsSince(t *time.Time) int {
	if t == nil {
		return 0
	}

	now := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	months := (now.Year()-t.Year())*12 + int(now.Month()-t.Month())

	if months < 0 {
		return 0
	}
	return months
}

func ParseView(viewSlice []string) (int, int, int, int, int) {
	var forest, mountain, sea, unlimited, NoView int
	for _, rawString := range viewSlice {
		cleanStr := strings.Trim(rawString, "{}")
		traits := strings.Split(cleanStr, ",")
		for _, trait := range traits {
			switch strings.ToLower(strings.TrimSpace(trait)) {
			case "forest":
				forest = 1
			case "mountain":
				mountain = 1
			case "sea":
				sea = 1
			case "unlimited":
				unlimited = 1
			default:
				NoView = 1
			}
		}
	}
	return forest, mountain, sea, unlimited, NoView
}

func ParseFacade(facadeSlice []string) (int, int, int, int, int) {
	var corner, frontage, interior, through, None int
	for _, rawString := range facadeSlice {
		cleanStr := strings.Trim(rawString, "{}")
		traits := strings.Split(cleanStr, ",")
		for _, trait := range traits {
			switch strings.ToLower(strings.TrimSpace(trait)) {
			case "corner":
				corner = 1
			case "frontage":
				frontage = 1
			case "interior":
				interior = 1
			case "through":
				through = 1
			default:
				None = 1
			}
		}
	}
	return corner, frontage, interior, through, None
}

func ParseFloors(floors []string) (int, int) {
	if len(floors) == 0 {
		return 0, 1
	}
	minF := 2147483647
	uniqueFloors := make(map[int]struct{})
	for _, f := range floors {
		cleanF := strings.Trim(f, "{}")
		if val, err := strconv.Atoi(cleanF); err == nil {
			uniqueFloors[val] = struct{}{}
			if val < minF {
				minF = val
			}
		}
	}
	if minF == 2147483647 {
		return 0, 1
	}
	return minF, len(uniqueFloors)
}

func BoolToInt(b *bool) int {
	if b != nil && *b {
		return 1
	}
	return 0
}

func ParseStr(s *string) string {
	if s != nil && *s != "" {
		return *s
	}
	return "Unknown"
}

func ParseWC(bathrooms, wc *int) int {
	count := 0
	if bathrooms != nil {
		count += *bathrooms
	}
	if wc != nil {
		count += *wc
	}
	return count
}
func ParseEnergyClass(s *string) string {
	if s == nil || *s == "" {
		return "Unknown"
	}

	// Clean up spaces and make uppercase
	val := strings.ToUpper(strings.TrimSpace(*s))

	// Replace Greek characters with English equivalents
	replacer := strings.NewReplacer(
		"Α", "A", // Greek Alpha
		"Β", "B", // Greek Beta
		"Γ", "C", // Greek Gamma
		"Δ", "D", // Greek Delta
		"Ε", "E", // Greek Epsilon
		"Ζ", "F", // Greek Zeta
		"Η", "G", // Greek Eta
	)
	val = replacer.Replace(val)

	// Valid energy classes typically found in EPCs
	validClasses := map[string]bool{
		"A+": true, "A": true, "A-": true,
		"B+": true, "B": true, "B-": true,
		"C+": true, "C": true, "C-": true,
		"D": true, "E": true, "F": true, "G": true,
	}

	if validClasses[val] {
		return val
	}

	// If it was some random garbage string, flag it so XGBoost ignores it
	return "Unknown"
}

func ParseBedrooms(b *int) int {
	if b != nil {
		return *b
	}
	return -1 // Standard numerical flag for "missing" in XGBoost
}
func ExportPropertiesToCSV[T any](raw []T, filepath string) error {
	var encodedData any

	switch v := any(raw).(type) {
	case []ResidentialProperty:
		var res []ResidentialFeatures
		for _, r := range v {
			f := ResidentialFeatures{}

			f.UID = r.Global.Uid
			f.SubType = r.Global.SubType
			f.AdminName = ParseStr(r.Global.AdminName)
			f.TransactionType = r.Global.TransactionType
			f.AskingPrice = r.Global.AskingPrice
			f.SquareMeters = r.Global.SquareMeters
			f.ZoneValue = r.Global.ZoneValue
			f.GeoLat = r.Global.GeoLat
			f.GeoLon = r.Global.GeoLon

			if f.SquareMeters > 0 {
				f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
			}
			f.AdCreationAge = ParseMonthsSince(r.Global.AdCreationDate)

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
				f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = ParseView(*r.ResidentialPropertySpecifics.ViewType)
			} else {
				f.NoView = 1
			}

			if r.Residential.FacadeType != nil {
				f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = ParseFacade(*r.Residential.FacadeType)
			} else {
				f.NoFacade = 1
			}

			f.Bedrooms = ParseBedrooms(r.ResidentialPropertySpecifics.BedRooms)
			if r.ResidentialPropertySpecifics.FloorLevel != nil {
				f.FloorLevel, f.FloorCount = ParseFloors(*r.ResidentialPropertySpecifics.FloorLevel)
			} else {
				f.FloorCount = 1
			}

			f.HeatType = ParseStr(r.Residential.Heat)
			f.HeatFrom = ParseStr(r.ResidentialPropertySpecifics.HeatFrom)
			f.EnergyPerformance = ParseEnergyClass(r.Residential.EnergyPerformance)
			f.WC = ParseWC(r.ResidentialPropertySpecifics.Bathrooms, r.ResidentialPropertySpecifics.Wc)

			f.AC = BoolToInt(r.ResidentialPropertySpecifics.Ac)
			f.Elevator = BoolToInt(r.Residential.Elevator)
			f.Pool = BoolToInt(r.ResidentialPropertySpecifics.Pool)
			f.Garden = BoolToInt(r.ResidentialPropertySpecifics.Garden)
			f.StorageRoom = BoolToInt(r.Residential.Storage_room)
			f.AluminumFrames = BoolToInt(r.Residential.AluminumFrames)
			f.Parking = BoolToInt(r.Residential.Parking)

			res = append(res, f)
		}
		encodedData = res

	case []LandPlotResidential:
		var lp []LandplotFeatures
		for _, r := range v {
			f := LandplotFeatures{}

			f.UID = r.Global.Uid
			f.SubType = r.Global.SubType
			f.AdminName = ParseStr(r.Global.AdminName)
			f.TransactionType = r.Global.TransactionType
			f.AskingPrice = r.Global.AskingPrice
			f.SquareMeters = r.Global.SquareMeters
			f.ZoneValue = r.Global.ZoneValue
			f.GeoLat = r.Global.GeoLat
			f.GeoLon = r.Global.GeoLon

			if f.SquareMeters > 0 {
				f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
			}
			f.AdCreationAge = ParseMonthsSince(r.Global.AdCreationDate)

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

			f.Buildable = BoolToInt(r.Landplot.Buildable)
			f.InCityPlan = BoolToInt(r.Landplot.InCityPlan)

			if r.Landplot.ViewType != nil {
				f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = ParseView(*r.Landplot.ViewType)
			} else {
				f.NoView = 1
			}

			if r.Landplot.FacadeType != nil {
				f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = ParseFacade(*r.Landplot.FacadeType)
			} else {
				f.NoFacade = 1
			}

			lp = append(lp, f)
		}
		encodedData = lp

	case []ResidentialBuilding:
		var res []ResidentialFeatures
		for _, r := range v {
			f := ResidentialFeatures{}

			f.UID = r.Global.Uid
			f.SubType = r.Global.SubType
			f.AdminName = ParseStr(r.Global.AdminName)
			f.TransactionType = r.Global.TransactionType
			f.AskingPrice = r.Global.AskingPrice
			f.SquareMeters = r.Global.SquareMeters
			f.ZoneValue = r.Global.ZoneValue
			f.GeoLat = r.Global.GeoLat
			f.GeoLon = r.Global.GeoLon

			if f.SquareMeters > 0 {
				f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
			}
			f.AdCreationAge = ParseMonthsSince(r.Global.AdCreationDate)

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
				f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = ParseFacade(*r.Residential.FacadeType)
			} else {
				f.NoFacade = 1
			}

			f.HeatType = ParseStr(r.Residential.Heat)
			f.EnergyPerformance = ParseEnergyClass(r.Residential.EnergyPerformance)

			f.StorageRoom = BoolToInt(r.Residential.Storage_room)
			f.AluminumFrames = BoolToInt(r.Residential.AluminumFrames)
			f.Elevator = BoolToInt(r.Residential.Elevator)
			f.Parking = BoolToInt(r.Residential.Parking)

			f.Bedrooms = -1
			f.FloorLevel = -1
			f.WC = -1
			f.NoView = -1

			res = append(res, f)
		}
		encodedData = res

	case []LandplotAgric:
		var lp []LandplotFeatures
		for _, r := range v {
			f := LandplotFeatures{}

			f.UID = r.Global.Uid
			f.SubType = r.Global.SubType
			f.AdminName = ParseStr(r.Global.AdminName)
			f.TransactionType = r.Global.TransactionType
			f.AskingPrice = r.Global.AskingPrice
			f.SquareMeters = r.Global.SquareMeters
			f.ZoneValue = r.Global.ZoneValue
			f.GeoLat = r.Global.GeoLat
			f.GeoLon = r.Global.GeoLon

			if f.SquareMeters > 0 {
				f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
			}
			f.AdCreationAge = ParseMonthsSince(r.Global.AdCreationDate)

			f.FacadeMeters = -1
			if r.Landplot.FacadeMeters != nil {
				f.FacadeMeters = *r.Landplot.FacadeMeters
			}

			f.Buildable = BoolToInt(r.Landplot.Buildable)
			f.InCityPlan = BoolToInt(r.Landplot.InCityPlan)

			if r.Landplot.ViewType != nil {
				f.ViewForest, f.ViewMountain, f.ViewSea, f.ViewUnlimited, f.NoView = ParseView(*r.Landplot.ViewType)
			} else {
				f.NoView = 1
			}

			if r.Landplot.FacadeType != nil {
				f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = ParseFacade(*r.Landplot.FacadeType)
			} else {
				f.NoFacade = 1
			}

			f.BuildingRatio = -1.0
			f.CoverageFactor = -1.0

			lp = append(lp, f)
		}
		encodedData = lp

	case []Commercial:
		var com []CommercialFeatures
		for _, r := range v {
			f := CommercialFeatures{}

			f.UID = r.Global.Uid
			f.SubType = r.Global.SubType
			f.AdminName = ParseStr(r.Global.AdminName)
			f.TransactionType = r.Global.TransactionType
			f.AskingPrice = r.Global.AskingPrice
			f.SquareMeters = r.Global.SquareMeters
			f.ZoneValue = r.Global.ZoneValue
			f.GeoLat = r.Global.GeoLat
			f.GeoLon = r.Global.GeoLon

			if f.SquareMeters > 0 {
				f.AskingPricePerSqm = float64(f.AskingPrice) / float64(f.SquareMeters)
			}
			f.AdCreationAge = ParseMonthsSince(r.Global.AdCreationDate)

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
				f.FacadeCorner, f.FacadeFrontage, f.FacadeInterior, f.FacadeThrough, f.NoFacade = ParseFacade(*r.CommercialSpecifics.FacadeType)
			} else {
				f.NoFacade = 1
			}

			if r.CommercialSpecifics.FloorLevel != nil {
				f.FloorLevel, f.FloorCount = ParseFloors(*r.CommercialSpecifics.FloorLevel)
			} else {
				f.FloorCount = 1
			}

			f.HeatType = ParseStr(r.CommercialSpecifics.Heat)
			f.HeatFrom = ParseStr(r.CommercialSpecifics.HeatFrom)

			f.EnergyPerformance = ParseEnergyClass(r.CommercialSpecifics.EnergyPerformance)
			f.WC = ParseWC(r.CommercialSpecifics.Bathrooms, r.CommercialSpecifics.Wc)

			f.AC = BoolToInt(r.CommercialSpecifics.Ac)
			f.StorageRoom = BoolToInt(r.CommercialSpecifics.Storage_room)
			f.AluminumFrames = BoolToInt(r.CommercialSpecifics.AluminumFrames)
			f.Parking = BoolToInt(r.CommercialSpecifics.Parking)
			f.CargoElevator = BoolToInt(r.CommercialSpecifics.CargoElevator)
			f.LoadingRamp = BoolToInt(r.CommercialSpecifics.LoadingRamp)

			if r.CommercialSpecifics.PowerSupply != nil && len(*r.CommercialSpecifics.PowerSupply) > 0 {
				f.PowerSupply = strings.Join(*r.CommercialSpecifics.PowerSupply, ",")
			} else {
				f.PowerSupply = "Unknown"
			}

			com = append(com, f)
		}
		encodedData = com

	default:
		return fmt.Errorf("unsupported slice type")
	}

	return writeCSV(filepath, encodedData)
}

func writeCSV(filepath string, data any) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	sliceVal := reflect.ValueOf(data)
	if sliceVal.Kind() != reflect.Slice || sliceVal.Len() == 0 {
		return fmt.Errorf("data must be a non-empty slice")
	}

	elemType := sliceVal.Index(0).Type()

	var headers []string
	for i := 0; i < elemType.NumField(); i++ {
		headers = append(headers, elemType.Field(i).Name)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for i := 0; i < sliceVal.Len(); i++ {
		elemVal := sliceVal.Index(i)
		var row []string

		for j := 0; j < elemVal.NumField(); j++ {
			fieldVal := elemVal.Field(j)

			switch fieldVal.Kind() {
			case reflect.String:
				row = append(row, fieldVal.String())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				row = append(row, strconv.FormatInt(fieldVal.Int(), 10))
			case reflect.Float32, reflect.Float64:
				row = append(row, strconv.FormatFloat(fieldVal.Float(), 'f', -1, 64))
			case reflect.Bool:
				row = append(row, strconv.FormatBool(fieldVal.Bool()))
			default:
				row = append(row, fmt.Sprintf("%v", fieldVal.Interface()))
			}
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
