package marketanalysis

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalysisPipeline struct {
	Db       *pgxpool.Pool
	Ctx      context.Context
	Warnings []string
	MLURL    string
}
type NominatimResponse struct {
	DisplayName string   `json:"display_name"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	BoundingBox []string `json:"boundingbox"`
	Address     struct {
		Suburb       string `json:"suburb"`
		CityDistrict string `json:"city_district"`
		City         string `json:"city"`
		Town         string `json:"town"`
		Village      string `json:"village"`
	} `json:"address"`
}

func (ap *AnalysisPipeline) Init(db *pgxpool.Pool, ctx context.Context) {
	ap.Db = db
	ap.Ctx = ctx
}
func (ap *AnalysisPipeline) RunPipeline(bbox BoundingBox) ([]DataPoint, []Cluster, *MLResponse, *MLResponse, error) {
	ap.Warnings = nil
	log.Printf("[ANALYSIS] Loading properties inside selected area")
	Properties, err := ap.GetAreaDataPoints(ap.Db, ap.Ctx, bbox)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("load area properties: %w", err)
	}
	log.Printf("[ANALYSIS] Found %d properties; calculating clusters", len(Properties))
	if len(Properties) == 0 {
		return []DataPoint{}, []Cluster{}, nil, nil, nil
	}

	clustersData := CreateClusters(Properties, 10)
	clusters := CalculateClusterStats(clustersData)
	Points, finalClusters := EnrichDataAndFindExtremes(clusters, clustersData)

	var uids []string
	for _, p := range Points {
		uids = append(uids, p.Uid)
	}

	ModeledProperties, err := ap.GetDataForMicroService(uids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("prepare model properties: %w", err)
	}

	log.Printf("[ANALYSIS] Prepared %d model rows", len(ModeledProperties))

	r1, r2, err := ap.TrainSeparateModels(ModeledProperties)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	PrintMLReport("SALES", r1)
	PrintMLReport("RENTALS", r2)

	return Points, finalClusters, r1, r2, nil
}

func PrintAnalysisResults(points []DataPoint, clusters []Cluster) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("=== ANALYSIS PIPELINE RESULTS | Total Points: %d | Total Clusters: %d ===\n", len(points), len(clusters))
	fmt.Printf("%s\n\n", strings.Repeat("=", 80))

	for _, c := range clusters {
		fmt.Printf(">>> CLUSTER %d (Type: %s) <<<\n", c.ClusterID, c.TransactionType)

		// Print Means and StdDevs nicely aligned
		fmt.Printf("  [Stats]   Age: %6.1f (±%5.1f) | Floor: %5.1f (±%4.1f) | GeoPremium: %6.2f (±%5.2f)\n",
			c.AgeMean, c.AgeStdDev, c.FloorMean, c.FloorStdDev, c.GeoPremiumMean, c.GeoPremiumStdDev)
		fmt.Printf("  [Stats]   Ask: %6.0f (±%5.0f) | Est:   %6.0f (±%5.0f) | Elo:        %6.0f (±%5.0f)\n",
			c.AskingMean, c.AskingStdDev, c.EstimatedMean, c.EstimatedStdDev, c.EloMean, c.EloStdDev)

		fmt.Printf("\n  --- TOP 5 EXTREMES ---\n")
		if len(c.Top5) == 0 {
			fmt.Printf("      None\n")
		}
		for i, p := range c.Top5 {
			printCompactDataPoint(i+1, p)
		}

		fmt.Printf("\n  --- BOTTOM 5 EXTREMES ---\n")
		if len(c.Bottom5) == 0 {
			fmt.Printf("      None\n")
		}
		for i, p := range c.Bottom5 {
			printCompactDataPoint(i+1, p)
		}
		fmt.Printf("%s\n", strings.Repeat("-", 80))
	}
}

// printCompactDataPoint prints a highly condensed, single-line summary of a DataPoint
func printCompactDataPoint(rank int, p DataPoint) {
	// Safely truncate UID so it doesn't wreck the terminal formatting
	shortUid := p.Uid
	if len(shortUid) > 12 {
		shortUid = shortUid[:12] + ".."
	}

	fmt.Printf("  %d. %-14s | Sqm: %5.1f | Age: %3d (Z:%+.1f) | Floor: %2d (Z:%+.1f) | Ask/m²: %5.0f (Z:%+.1f) | Est/m²: %5.0f (Z:%+.1f) | Elo: %4.0f (Z:%+.1f)\n",
		rank,
		shortUid,
		p.SquareMeters,
		p.Age, p.AgeZ,
		p.FloorLevel, p.FloorZ,
		p.AskingPricePerSqm, p.AskingZ,
		p.EstimatedPricePerSqm, p.EstimatedZ,
		p.Elo, p.EloZ,
	)
}
