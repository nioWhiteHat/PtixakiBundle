package marketanalysis

import (
	conf "Ptixiaki/Worker/training"
	t "Ptixiaki/Worker/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

func EncodeProperty(property ResidentialProperty) (ModeledResidentialProperty, error) {
	r := property
	f := ModeledResidentialProperty{}
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
	f.Pool = t.BoolToInt(r.ResidentialPropertySpecifics.Pool)
	f.Garden = t.BoolToInt(r.ResidentialPropertySpecifics.Garden)
	f.StorageRoom = t.BoolToInt(r.Residential.Storage_room)
	f.AluminumFrames = t.BoolToInt(r.Residential.AluminumFrames)
	f.Parking = t.BoolToInt(r.Residential.Parking)
	f.Elevator = t.BoolToInt(r.Residential.Elevator)
	f.GeoPremium = r.GeoPremium

	return f, nil
}

type MLRequest struct {
	Data               []ModeledResidentialProperty `json:"data"`
	DropColumns        []string                     `json:"drop_columns"`
	CategoricalColumns []string                     `json:"categorical_columns"`
	MaxDepth           int                          `json:"max_depth"`
	MinChildWeight     float64                      `json:"min_child_weight"`
	LearningRate       float64                      `json:"learning_rate"`
	NEstimators        int                          `json:"n_estimators"`
	TargetColumn       string                       `json:"target_column"`
}

type MLResponse struct {
	R2Score            float64               `json:"r2_score"`
	BasePrice          float64               `json:"base_price"`
	FeatureImportances []FeatureImportance   `json:"feature_importances"`
	TopInteractions    []Interaction         `json:"top_interactions"`
	Examples           []PropertyExplanation `json:"examples"`
	Message            string                `json:"message"`
}

type FeatureImportance struct {
	Feature    string  `json:"feature"`
	Importance float64 `json:"importance"`
}

type Interaction struct {
	Feature1            string  `json:"feature_1"`
	Feature2            string  `json:"feature_2"`
	InteractionStrength float64 `json:"interaction_strength"`
}

type PropertyExplanation struct {
	UID            string          `json:"uid"`
	ActualPrice    float64         `json:"actual_price"`
	PredictedPrice float64         `json:"predicted_price"`
	FeatureImpacts []FeatureImpact `json:"feature_impacts"`
}

type FeatureImpact struct {
	Feature    string  `json:"feature"`
	MoneyAdded float64 `json:"money_added"`
}

func callMLService(ctx context.Context, endpoint string, reqData MLRequest) (*MLResponse, error) {
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	if endpoint == "" {
		endpoint = "http://localhost:5000/TrainAndExplain"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 4 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("Python analysis HTTP %d: %s", resp.StatusCode, body)
	}

	var mlResp MLResponse
	if err := json.NewDecoder(resp.Body).Decode(&mlResp); err != nil {
		return nil, err
	}

	return &mlResp, nil
}

func PrintMLReport(title string, resp *MLResponse) {
	if resp == nil {
		fmt.Printf("\n--- No data for %s (Skipped) ---\n", title)
		return
	}

	fmt.Printf("\n=== %s MODEL RESULTS ===\n", title)
	fmt.Printf("Accuracy (R2 Score): %.2f%%\n", resp.R2Score*100)
	fmt.Printf("Average Base Price:  €%.2f/sqm\n", resp.BasePrice)

	fmt.Println("\n-- Top 5 Global Features --")
	for i, f := range resp.FeatureImportances {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %-20s (Weight: %.4f)\n", i+1, f.Feature, f.Importance)
	}

	fmt.Println("\n-- Top 5 Feature Interactions --")
	for i, inter := range resp.TopInteractions {
		fmt.Printf("%d. %s <--> %s (Strength: %.2f)\n", i+1, inter.Feature1, inter.Feature2, inter.InteractionStrength)
	}

	fmt.Println("\n-- Example Properties (Showing 2 of 10) --")
	for i, ex := range resp.Examples {
		if i >= 2 {
			break
		} // Only print 2 to keep terminal clean
		fmt.Printf("\nUID: %s | Actual: €%.2f | Predicted: €%.2f\n", ex.UID, ex.ActualPrice, ex.PredictedPrice)
		for j, imp := range ex.FeatureImpacts {
			if j >= 3 {
				break
			} // Show top 3 impacts per property
			sign := "+"
			if imp.MoneyAdded < 0 {
				sign = ""
			}
			fmt.Printf("   -> %-20s: %s€%.2f\n", imp.Feature, sign, imp.MoneyAdded)
		}
	}
	fmt.Println("\n=======================================")
}

func (ap *AnalysisPipeline) TrainSeparateModels(properties []ModeledResidentialProperty) (*MLResponse, *MLResponse, error) {
	settings := conf.AttrTreeConfigs{}
	settings.Initialize("residence", "property")
	ctx := ap.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	results := make(map[string]*MLResponse)
	for _, transaction := range []string{"sell", "rent"} {
		group := []ModeledResidentialProperty{}
		for _, property := range properties {
			if property.TransactionType == transaction {
				group = append(group, property)
			}
		}
		if len(group) < 10 {
			ap.Warnings = append(ap.Warnings, fmt.Sprintf("%s model skipped: %d eligible properties; at least 10 are required. Available cluster statistics are still shown.", transaction, len(group)))
			continue
		}
		log.Printf("[ANALYSIS %s] Training and explaining %d properties", transaction, len(group))
		result, err := callMLService(ctx, ap.MLURL, MLRequest{
			Data: group, DropColumns: settings.ResidentialPropertyColumnsDropped,
			CategoricalColumns: settings.Categoricals, MaxDepth: 6, MinChildWeight: 1,
			LearningRate: 0.1, NEstimators: 100, TargetColumn: "AskingPricePerSqm",
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			log.Printf("[ANALYSIS %s] %v", transaction, err)
			ap.Warnings = append(ap.Warnings, fmt.Sprintf("The %s model could not be generated. Check the Python service on port 5000 and its logs. Cluster statistics are still available.", transaction))
			continue
		}
		log.Printf("[ANALYSIS %s] Model and explanations complete", transaction)
		results[transaction] = result
	}
	return results["sell"], results["rent"], nil
}
