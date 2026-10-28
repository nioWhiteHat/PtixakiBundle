package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

func DecodePropertyInput(reader io.Reader) (PropertyFeatures, error) {
	var prop PropertyFeatures
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&prop); err != nil {
		return prop, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return prop, fmt.Errorf("request must contain exactly one JSON object")
	}
	return prop, ValidatePropertyInput(prop)
}


func ValidatePropertyInput(prop PropertyFeatures) error {
	if prop.SubType != "apartment" && prop.SubType != "maisonette" && prop.SubType != "detached_house" {
		return fmt.Errorf("SubType must be apartment, maisonette or detached_house")
	}
	if strings.TrimSpace(prop.Address) == "" || utf8.RuneCountInString(prop.Address) > 255 {
		return fmt.Errorf("Address must contain between 1 and 255 characters")
	}
	if prop.SquareMeters <= 10 || prop.SquareMeters > math.MaxInt32 {
		return fmt.Errorf("SquareMeters must be greater than 10 and within the supported integer range")
	}
	if prop.ConstructionYear < 1900 || prop.ConstructionYear > time.Now().Year() {
		return fmt.Errorf("ConstructionYear must be between 1900 and the current year")
	}
	if prop.AskingPricePerSqm != nil {
		asking := *prop.AskingPricePerSqm
		if math.IsNaN(asking) || math.IsInf(asking, 0) || asking <= 0 {
			return fmt.Errorf("AskingPricePerSqm must be a positive finite number, or omitted/null")
		}
		total := asking * float64(prop.SquareMeters)
		if total < 1 || total > math.MaxInt32 {
			return fmt.Errorf("total asking price must be between 1 and %d", math.MaxInt32)
		}
	}
	flags := map[string]int{
		"IsRenovated": prop.IsRenovated, "NoView": prop.NoView,
		"ViewForest": prop.ViewForest, "ViewMountain": prop.ViewMountain,
		"ViewSea": prop.ViewSea, "ViewUnlimited": prop.ViewUnlimited,
		"NoFacade": prop.NoFacade, "FacadeCorner": prop.FacadeCorner,
		"FacadeFrontage": prop.FacadeFrontage, "FacadeInterior": prop.FacadeInterior,
		"FacadeThrough": prop.FacadeThrough, "AC": prop.AC, "Elevator": prop.Elevator,
		"Parking": prop.Parking, "StorageRoom": prop.StorageRoom,
		"AluminumFrames": prop.AluminumFrames, "Pool": prop.Pool, "Garden": prop.Garden,
	}
	for name, value := range flags {
		if value != 0 && value != 1 {
			return fmt.Errorf("%s must be 0 or 1", name)
		}
	}
	if prop.IsRenovated == 1 && (prop.RenovYear < prop.ConstructionYear || prop.RenovYear > time.Now().Year()) {
		return fmt.Errorf("RenovYear must be between ConstructionYear and the current year when IsRenovated is 1")
	}
	if prop.IsRenovated == 0 && prop.RenovYear != 0 {
		return fmt.Errorf("set IsRenovated to 1 when supplying RenovYear")
	}
	if prop.WC < 0 || prop.Bedrooms < 0 || prop.WC > math.MaxInt32 || prop.Bedrooms > math.MaxInt32 {
		return fmt.Errorf("WC and Bedrooms must be non-negative supported integers")
	}
	return nil
}
