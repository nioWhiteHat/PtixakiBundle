package testing

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type SummaryStatistics struct {
	MeanResidual     float64 `json:"MeanResidual"`
	StdDevResidual   float64 `json:"StdDevResidual"`
	MeanGeoPremium   float64 `json:"MeanGeoPremium"`
	StdDevGeoPremium float64 `json:"StdDevGeoPremium"`
}

// CalculateFinalStatistics reads both final JSONs, computes global Mean and StdDev, and saves a summary
func CalculateFinalStatistics(phase2JsonPath string, geoPremJsonPath string, outputDir string, itemType string, subType string, transaction string) error {
	fmt.Printf("   -> [STATS] Calculating Global Summary Statistics...\n")

	// 1. Process Phase 2 Residuals
	type minimalPhase2 struct {
		ResidualPercentage float64 `json:"ResidualPercentage"`
	}

	p2Data, err := os.ReadFile(phase2JsonPath)
	if err != nil {
		return fmt.Errorf("failed to read phase 2 json: %v", err)
	}

	var p2Records []minimalPhase2
	if err := json.Unmarshal(p2Data, &p2Records); err != nil {
		return fmt.Errorf("failed to unmarshal phase 2 json: %v", err)
	}

	var sumResiduals, meanResidual, stdResidual float64
	nP2 := float64(len(p2Records))
	if nP2 > 0 {
		for _, r := range p2Records {
			sumResiduals += r.ResidualPercentage
		}
		meanResidual = sumResiduals / nP2

		var sumSqDiff float64
		for _, r := range p2Records {
			diff := r.ResidualPercentage - meanResidual
			sumSqDiff += diff * diff
		}
		stdResidual = math.Sqrt(sumSqDiff / nP2)
	}

	// 2. Process Geo Premiums
	type minimalGeoPrem struct {
		GeoPremium float64 `json:"GeoPremium"`
	}

	gpData, err := os.ReadFile(geoPremJsonPath)
	if err != nil {
		return fmt.Errorf("failed to read geopremium json: %v", err)
	}

	var gpRecords []minimalGeoPrem
	if err := json.Unmarshal(gpData, &gpRecords); err != nil {
		return fmt.Errorf("failed to unmarshal geopremium json: %v", err)
	}

	var sumGeo, meanGeo, stdGeo float64
	nGP := float64(len(gpRecords))
	if nGP > 0 {
		for _, r := range gpRecords {
			sumGeo += r.GeoPremium
		}
		meanGeo = sumGeo / nGP

		var sumSqDiff float64
		for _, r := range gpRecords {
			diff := r.GeoPremium - meanGeo
			sumSqDiff += diff * diff
		}
		stdGeo = math.Sqrt(sumSqDiff / nGP)
	}

	// 3. Save to JSON
	stats := SummaryStatistics{
		MeanResidual:     meanResidual,
		StdDevResidual:   stdResidual,
		MeanGeoPremium:   meanGeo,
		StdDevGeoPremium: stdGeo,
	}

	os.MkdirAll(outputDir, os.ModePerm)
	fileName := fmt.Sprintf("SummaryStats_%s_%s_%s.json", itemType, subType, transaction)
	fullOutputPath := filepath.Join(outputDir, fileName)

	outFile, err := os.Create(fullOutputPath)
	if err != nil {
		return fmt.Errorf("failed to create summary stats file: %v", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode summary stats json: %v", err)
	}

	fmt.Printf("✅ [SUCCESS] Global stats saved to: %s\n", fullOutputPath)
	return nil
}