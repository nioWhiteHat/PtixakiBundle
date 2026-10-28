package training

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PredictionData struct {
	UID                  string  `json:"UID"`
	AskingPricePerSqm    float64 `json:"AskingPricePerSqm"`
	PredictedPricePerSqm float64 `json:"PredictedPricePerSqm"`
	ResidualPercentage   float64 `json:"ResidualPercentage"`
}

// GeoPremiumResult represents the final structured data ready for Phase 2 training
type GeoPremiumResult struct {
	UID                  string  `json:"UID"`
	AskingPricePerSqm    float64 `json:"AskingPricePerSqm"`
	PredictedPricePerSqm float64 `json:"PredictedPricePerSqm"`
	ResidualPercentage   float64 `json:"ResidualPercentage"`
	NeighborCount        int     `json:"NeighborCount"` // <-- Added to track how many candidates passed
	GeoPremium           float64 `json:"GeoPremium"`       
	GeoPremiumStdDev     float64 `json:"GeoPremiumStdDev"` 
	GeoPremiumZScore     float64 `json:"GeoPremiumZScore"` 
}

// CalcGeoPremium computes local spatial residuals restricted ONLY to the current property group
func CalcGeoPremium(ctx context.Context, pool *pgxpool.Pool, ItemType string, SubType string, Transaction string, FilePath string, JsonOutputPath string) error {
	fmt.Printf("\n🌍 [START] Spatial GeoPremium Engine for %s %s (%s)...\n", ItemType, SubType, Transaction)

	// 1. Read and Unmarshal the Python JSON Output
	fileData, err := os.ReadFile(FilePath)
	if err != nil {
		return fmt.Errorf("failed to read json: %v", err)
	}

	var predictions []PredictionData
	if err := json.Unmarshal(fileData, &predictions); err != nil {
		return fmt.Errorf("failed to unmarshal json: %v", err)
	}

	// 2. Build the Fast-Lookup Map and the master UID list
	predMap := make(map[string]PredictionData, len(predictions))
	var allUIDs []string
	for _, p := range predictions {
		predMap[p.UID] = p
		allUIDs = append(allUIDs, p.UID)
	}
	
	totalProps := len(allUIDs)
	fmt.Printf("   -> [DATA] Loaded %d valid predictions from JSON.\n", totalProps)

	// 3. Define the Allowed Subtypes for the SQL Filter
	var allowedSubtypes []string
	if SubType == "properties" {
		allowedSubtypes = []string{"apartment", "maisonette", "detached_house"}
	} else {
		allowedSubtypes = []string{SubType}
	}

	neighborMap := make(map[string][]string)
	batchSize := 5000
	totalBatches := (totalProps + batchSize - 1) / batchSize

	fmt.Printf("   -> [POSTGIS] Starting spatial queries in %d batches...\n", totalBatches)

	// 4. Batch Process UIDs through the Database
	for i := 0; i < totalProps; i += batchSize {
		end := i + batchSize
		if end > totalProps {
			end = totalProps
		}
		batchUIDs := allUIDs[i:end]
		batchNum := (i / batchSize) + 1

		fmt.Printf("      - Fetching Batch %d/%d (Properties %d to %d)...\n", batchNum, totalBatches, i+1, end)

		// Restructured Query: Filter the candidate pool first, then do the spatial search.
		query := `
		WITH filtered_pool AS (
			SELECT geo.property_uid, geo.geom
			FROM property_geo_data geo
			JOIN properties prop ON geo.property_uid = prop.uid
			WHERE prop.transaction_type = $2
			  AND prop.item_type = $3
			  AND prop.subtype = ANY($4::text[])
		)
		SELECT t.property_uid AS target_uid, n.property_uid AS neighbor_uid
		FROM unnest($1::text[]) AS target_ids(uid)
		JOIN property_geo_data t ON t.property_uid = target_ids.uid
		CROSS JOIN LATERAL (
			SELECT fp.property_uid
			FROM filtered_pool fp
			WHERE ST_DWithin(t.geom::geography, fp.geom::geography, 300)
			  AND fp.property_uid != t.property_uid
			ORDER BY t.geom::geography <-> fp.geom::geography
			LIMIT 50
		) n;
		`

		rows, err := pool.Query(ctx, query, batchUIDs, Transaction, ItemType, allowedSubtypes)
		if err != nil {
			return fmt.Errorf("database query failed for batch %d: %v", batchNum, err)
		}

		for rows.Next() {
			var targetUID, neighborUID string
			if err := rows.Scan(&targetUID, &neighborUID); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan row: %v", err)
			}
			
			// Final check to ensure neighbor survived the Phase 1 Tree filters
			if _, exists := predMap[neighborUID]; exists {
				neighborMap[targetUID] = append(neighborMap[targetUID], neighborUID)
			}
		}
		rows.Close()
	}

	fmt.Println("   -> [MATH] Calculating local spatial premiums and standard deviations...")

	// 5. Calculate Area GeoPremium & StdDev for each property
	type intermediate struct {
		UID           string
		Mean          float64
		StdDev        float64
		NeighborCount int
	}
	var intermediates []intermediate
	var sumGlobalGeo float64

	for _, targetUID := range allUIDs {
		neighbors := neighborMap[targetUID]
		var validResiduals []float64
		var sumResiduals float64

		for _, nUID := range neighbors {
			neighborData := predMap[nUID]
			validResiduals = append(validResiduals, neighborData.ResidualPercentage)
			sumResiduals += neighborData.ResidualPercentage
		}

		var mean, stddev float64
		n := len(validResiduals)
		
		if n > 0 {
			mean = sumResiduals / float64(n)

			// StdDev requires calculating the squared differences
			var sumSqDiff float64
			for _, res := range validResiduals {
				diff := res - mean
				sumSqDiff += diff * diff
			}
			stddev = math.Sqrt(sumSqDiff / float64(n)) 
		}

		sumGlobalGeo += mean
		intermediates = append(intermediates, intermediate{
			UID:           targetUID,
			Mean:          mean,
			StdDev:        stddev,
			NeighborCount: n, // Save the count here
		})
	}

	fmt.Println("   -> [MATH] Computing global Z-Scores across the entire dataset...")

	// 6. Calculate Global Z-Score Metrics
	N := float64(len(intermediates))
	var globalMean, globalStdDev float64
	
	if N > 0 {
		globalMean = sumGlobalGeo / N
		var sumGlobalSqDiff float64

		for _, inter := range intermediates {
			diff := inter.Mean - globalMean
			sumGlobalSqDiff += diff * diff
		}
		globalStdDev = math.Sqrt(sumGlobalSqDiff / N)
	}

	// 7. Build Final Slice with Computed Z-Scores
	var finalResults []GeoPremiumResult
	for _, inter := range intermediates {
		pred := predMap[inter.UID]

		var zScore float64
		if globalStdDev > 0 {
			zScore = (inter.Mean - globalMean) / globalStdDev
		}

		finalResults = append(finalResults, GeoPremiumResult{
			UID:                  pred.UID,
			AskingPricePerSqm:    pred.AskingPricePerSqm,
			PredictedPricePerSqm: pred.PredictedPricePerSqm,
			ResidualPercentage:   pred.ResidualPercentage,
			NeighborCount:        inter.NeighborCount, // Pass the count into the final JSON struct
			GeoPremium:           inter.Mean,
			GeoPremiumStdDev:     inter.StdDev,
			GeoPremiumZScore:     zScore,
		})
	}

	// 8. Write Data to New JSON
	fmt.Println("   -> [FILE] Saving final results to disk...")
	os.MkdirAll(JsonOutputPath, os.ModePerm)
	fileName := fmt.Sprintf("GeoPrem_%s_%s_%s.json", ItemType, SubType, Transaction)
	fullOutputPath := filepath.Join(JsonOutputPath, fileName)

	outFile, err := os.Create(fullOutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(finalResults); err != nil {
		return fmt.Errorf("failed to encode output json: %v", err)
	}

	fmt.Printf("✅ [SUCCESS] Calculated GeoPremium for %d properties. Saved to: %s\n\n", len(finalResults), fullOutputPath)
	return nil
}


type geoPremMinimal struct {
	UID        string  `json:"UID"`
	GeoPremium float64 `json:"GeoPremium"`
}

// MergeGeoPremiumToCSV reads the original CSV and the new JSON, and creates a combined CSV.
func MergeGeoPremiumToCSV(csvInputPath string, jsonInputPath string, csvOutputPath string) error {
	fmt.Println("   -> [MERGE] Injecting GeoPremium into Phase 2 CSV...")

	// 1. Read the JSON and build a quick lookup map
	fileData, err := os.ReadFile(jsonInputPath)
	if err != nil {
		return fmt.Errorf("failed to read geopremium json: %v", err)
	}

	var jsonRecords []geoPremMinimal
	if err := json.Unmarshal(fileData, &jsonRecords); err != nil {
		return fmt.Errorf("failed to unmarshal geopremium json: %v", err)
	}

	geoMap := make(map[string]float64, len(jsonRecords))
	for _, rec := range jsonRecords {
		geoMap[rec.UID] = rec.GeoPremium
	}

	// 2. Open the Original CSV for reading
	fIn, err := os.Open(csvInputPath)
	if err != nil {
		return fmt.Errorf("failed to open original csv: %v", err)
	}
	defer fIn.Close()
	csvReader := csv.NewReader(fIn)

	// 3. Create the New CSV for writing
	fOut, err := os.Create(csvOutputPath)
	if err != nil {
		return fmt.Errorf("failed to create phase 2 csv: %v", err)
	}
	defer fOut.Close()
	csvWriter := csv.NewWriter(fOut)
	defer csvWriter.Flush()

	// 4. Read the Header and find the UID column
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("failed to read csv header: %v", err)
	}

	uidIdx := -1
	for i, col := range header {
		if col == "UID" {
			uidIdx = i
			break
		}
	}
	if uidIdx == -1 {
		return fmt.Errorf("could not find 'UID' column in original csv")
	}

	// Add the new column to the header and write it
	header = append(header, "GeoPremium")
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write header to new csv: %v", err)
	}

	// 5. Stream the rest of the rows, append the GeoPremium, and write
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading csv row: %v", err)
		}

		uid := record[uidIdx]
		
		// Lookup the GeoPremium (defaults to 0.0 if not found, e.g., if it was filtered out)
		geoPrem := 0.0
		if val, exists := geoMap[uid]; exists {
			geoPrem = val
		}

		// Append as a string formatted to 6 decimal places
		record = append(record, fmt.Sprintf("%.6f", geoPrem))
		
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("error writing row to new csv: %v", err)
		}
	}

	fmt.Printf("✅ [SUCCESS] Merged GeoPremium! Phase 2 data ready at: %s\n", csvOutputPath)
	return nil
}