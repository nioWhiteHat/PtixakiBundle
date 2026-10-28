package api

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestOptionalAskingInput(t *testing.T) {
	base := `{"SubType":"apartment","SquareMeters":82,"Address":"Synthetic address","ConstructionYear":2004`
	for _, suffix := range []string{`}`, `,"AskingPricePerSqm":null}`, `,"AskingPricePerSqm":2650}`} {
		prop, err := DecodePropertyInput(strings.NewReader(base + suffix))
		if err != nil {
			t.Fatal(err)
		}
		if suffix != `,"AskingPricePerSqm":2650}` && prop.AskingPricePerSqm != nil {
			t.Fatal("omitted/null asking must stay nil")
		}
	}
	for _, invalid := range []string{`,"AskingPricePerSqm":0}`, `,"AskingPricePerSqm":-1}`, `,"AskingPricePerSqm":"2650"}`, `,"AskingPricePerSqm":1e100}`, `,"Unknown":1}`, `} {}`} {
		if _, err := DecodePropertyInput(strings.NewReader(base + invalid)); err == nil {
			t.Fatalf("accepted invalid input %s", invalid)
		}
	}
	good, _ := DecodePropertyInput(strings.NewReader(base + `}`))
	for _, value := range []float64{math.NaN(), math.Inf(1), -1, 0} {
		prop := good
		prop.AskingPricePerSqm = &value
		if ValidatePropertyInput(prop) == nil {
			t.Fatal("accepted invalid price")
		}
	}
	for _, mutate := range []func(*PropertyFeatures){
		func(p *PropertyFeatures) { p.Address = "" }, func(p *PropertyFeatures) { p.SquareMeters = 0 },
		func(p *PropertyFeatures) { p.IsRenovated = 1; p.RenovYear = 0 }, func(p *PropertyFeatures) { p.RenovYear = 2019 },
		func(p *PropertyFeatures) { p.AC = 2 }, func(p *PropertyFeatures) { p.SubType = "building" },
	} {
		prop := good
		mutate(&prop)
		if ValidatePropertyInput(prop) == nil {
			t.Fatal("accepted invalid property")
		}
	}
	encoded, _ := json.Marshal(good)
	if strings.Contains(string(encoded), "AskingPricePerSqm") {
		t.Fatal("unknown asking should be omitted in details input fields")
	}
}
