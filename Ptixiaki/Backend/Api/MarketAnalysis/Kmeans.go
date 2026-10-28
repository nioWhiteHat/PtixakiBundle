package marketanalysis

import (
	"fmt"
	"log"
	"math"
)

type DataPoint struct {
	SquareMeters         float64
	Age                  int
	RenovAge             int
	FloorLevel           int
	GeoPremium           float64
	AskingPricePerSqm    float64
	EstimatedPricePerSqm float64
	Uid                  string
	Elo                  float64
	TransactionType      string
	ClusterID            int
	AgeZ                 float64
	AskingZ              float64
	EstimatedZ           float64
	FloorZ               float64
	GeoPremiumZ          float64
	EloZ                 float64
}
type Cluster struct {
	ClusterID        int
	TransactionType  string
	AgeMean          float64
	AgeStdDev        float64
	AskingMean       float64
	AskingStdDev     float64
	EstimatedMean    float64
	EstimatedStdDev  float64
	FloorMean        float64
	FloorStdDev      float64
	GeoPremiumMean   float64
	GeoPremiumStdDev float64
	EloMean          float64
	EloStdDev        float64
	Top5             []DataPoint
	Bottom5          []DataPoint
}

type ClusterPoint struct {
	Features      [4]float64
	OriginalIndex int
}

func Standardize(data []DataPoint) []ClusterPoint {
	if len(data) == 0 {
		return nil
	}

	var sums, sqSums [4]float64
	n := float64(len(data))

	for _, p := range data {
		feats := [4]float64{p.SquareMeters, float64(p.Age), float64(p.RenovAge), float64(p.FloorLevel)}
		for i := 0; i < 4; i++ {
			sums[i] += feats[i]
			sqSums[i] += feats[i] * feats[i]
		}
	}

	var means, stdDevs [4]float64
	for i := 0; i < 4; i++ {
		means[i] = sums[i] / n
		variance := (sqSums[i] / n) - (means[i] * means[i])
		if variance > 0 {
			stdDevs[i] = math.Sqrt(variance)
		} else {
			stdDevs[i] = 1.0
		}
	}

	stdData := make([]ClusterPoint, len(data))
	for i, p := range data {
		feats := [4]float64{p.SquareMeters, float64(p.Age), float64(p.RenovAge), float64(p.FloorLevel)}
		var stdFeats [4]float64
		for j := 0; j < 4; j++ {
			stdFeats[j] = (feats[j] - means[j]) / stdDevs[j]
		}
		stdData[i] = ClusterPoint{
			Features:      stdFeats,
			OriginalIndex: i,
		}
	}

	return stdData
}

func distanceSq(a, b [4]float64) float64 {
	var sum float64
	for i := 0; i < 4; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

func runKMeans(data []ClusterPoint, k int) ([][4]float64, map[int][]ClusterPoint, float64) {
	if len(data) == 0 || k < 1 {
		return nil, map[int][]ClusterPoint{}, 0
	}
	// Farthest-point seeds are distinct and deterministic. Identical rows need one group.
	centroids := [][4]float64{data[0].Features}
	for len(centroids) < k {
		var candidate [4]float64
		farthest := 0.0
		for _, point := range data {
			nearest := math.MaxFloat64
			for _, center := range centroids {
				nearest = math.Min(nearest, distanceSq(point.Features, center))
			}
			if nearest > farthest {
				candidate, farthest = point.Features, nearest
			}
		}
		if farthest == 0 {
			break
		}
		centroids = append(centroids, candidate)
	}
	k = len(centroids)

	var clusters map[int][]ClusterPoint
	var wcss float64

	for iteration := 0; iteration < 100; iteration++ {
		clusters = make(map[int][]ClusterPoint)
		wcss = 0

		for _, p := range data {
			minDist := math.MaxFloat64
			closest := 0
			for i, c := range centroids {
				dist := distanceSq(p.Features, c)
				if dist < minDist {
					minDist = dist
					closest = i
				}
			}
			clusters[closest] = append(clusters[closest], p)
			wcss += minDist
		}

		converged := true
		for i := 0; i < k; i++ {
			pts := clusters[i]
			if len(pts) == 0 {
				// Retain an unused center; endlessly reseeding caused small-area hangs.
				continue
			}
			var newCenter [4]float64
			for _, p := range pts {
				for j := 0; j < 4; j++ {
					newCenter[j] += p.Features[j]
				}
			}
			for j := 0; j < 4; j++ {
				newCenter[j] /= float64(len(pts))
			}

			if distanceSq(centroids[i], newCenter) > 1e-9 {
				converged = false
			}
			centroids[i] = newCenter
		}

		if converged {
			break
		}
	}

	return centroids, clusters, wcss
}

func FindOptimalK(data []ClusterPoint, maxK int) int {
	if len(data) < maxK {
		maxK = len(data)
	}
	if maxK <= 2 {
		return maxK
	}

	wcssValues := make([]float64, maxK+1)
	for k := 1; k <= maxK; k++ {
		_, _, wcss := runKMeans(data, k)
		wcssValues[k] = wcss
	}

	x1, y1 := 1.0, wcssValues[1]
	x2, y2 := float64(maxK), wcssValues[maxK]

	a := y2 - y1
	b := x1 - x2
	c := x2*y1 - y2*x1

	maxDist := -1.0
	optimalK := 1

	for k := 1; k <= maxK; k++ {
		x0, y0 := float64(k), wcssValues[k]
		dist := math.Abs(a*x0+b*y0+c) / math.Sqrt(a*a+b*b)
		if dist > maxDist {
			maxDist = dist
			optimalK = k
		}
	}

	return optimalK
}

func CreateClusters(data []DataPoint, maxSearchK int) map[int][]DataPoint {
	if len(data) == 0 {
		return nil
	}

	stdData := Standardize(data)
	optimalK := FindOptimalK(stdData, maxSearchK)
	_, clustersStd, _ := runKMeans(stdData, optimalK)

	finalClusters := make(map[int][]DataPoint)
	for clusterID, pts := range clustersStd {
		for _, p := range pts {
			finalClusters[clusterID] = append(finalClusters[clusterID], data[p.OriginalIndex])
		}
	}

	return finalClusters
}

func PrintClusters(clusters map[int][]DataPoint) {
	for clusterID, points := range clusters {
		fmt.Printf("Cluster ID: %d | Total Properties: %d\n", clusterID, len(points))
		fmt.Println("---------------------------------------------------------------------------------------------------")

		for i, p := range points {
			fmt.Printf("%3d. UID: %s | SqM: %6.2f | Age: %3d | RenovAge: %3d | Floor: %2d\n",
				i+1, p.Uid, p.SquareMeters, p.Age, p.RenovAge, p.FloorLevel)
		}

		fmt.Println()
	}
	log.Default().Println("total clusters:", len(clusters))
}
