package marketanalysis

import (
	"math"
	"sort"
)

func CalculateClusterStats(clusters map[int][]DataPoint) []Cluster {
	var results []Cluster

	for clusterID, points := range clusters {
		groupedPoints := map[string][]DataPoint{
			"sell": {},
			"rent": {},
		}

		for _, p := range points {
			groupedPoints[p.TransactionType] = append(groupedPoints[p.TransactionType], p)
		}

		for tType, tPoints := range groupedPoints {
			if len(tPoints) == 0 {
				continue
			}

			var age, asking, est, floor, geo, elo []float64
			for _, p := range tPoints {
				age = append(age, float64(p.Age))
				asking = append(asking, p.AskingPricePerSqm)
				est = append(est, p.EstimatedPricePerSqm)
				floor = append(floor, float64(p.FloorLevel))
				geo = append(geo, p.GeoPremium)
				elo = append(elo, p.Elo)
			}

			c := Cluster{
				ClusterID:       clusterID,
				TransactionType: tType,
				AgeMean:         CalculateMean(age),
				AskingMean:      CalculateMean(asking),
				EstimatedMean:   CalculateMean(est),
				FloorMean:       CalculateMean(floor),
				GeoPremiumMean:  CalculateMean(geo),
				EloMean:         CalculateMean(elo),
			}

			c.AgeStdDev = CalculateStandardDeviation(age, c.AgeMean)
			c.AskingStdDev = CalculateStandardDeviation(asking, c.AskingMean)
			c.EstimatedStdDev = CalculateStandardDeviation(est, c.EstimatedMean)
			c.FloorStdDev = CalculateStandardDeviation(floor, c.FloorMean)
			c.GeoPremiumStdDev = CalculateStandardDeviation(geo, c.GeoPremiumMean)
			c.EloStdDev = CalculateStandardDeviation(elo, c.EloMean)

			results = append(results, c)
		}
	}

	return results
}




func EnrichDataAndFindExtremes(stats []Cluster, clusters map[int][]DataPoint) ([]DataPoint, []Cluster) {
	statsMap := make(map[int]map[string]*Cluster)
	for i := range stats {
		cid := stats[i].ClusterID
		ttype := stats[i].TransactionType
		if statsMap[cid] == nil {
			statsMap[cid] = make(map[string]*Cluster)
		}
		statsMap[cid][ttype] = &stats[i]
	}

	var allEnrichedPoints []DataPoint

	for clusterID, points := range clusters {
		groupedPoints := map[string][]DataPoint{
			"sell": {},
			"rent": {},
		}
		for _, p := range points {
			groupedPoints[p.TransactionType] = append(groupedPoints[p.TransactionType], p)
		}

		for tType, tPoints := range groupedPoints {
			n := len(tPoints)
			if n == 0 {
				continue
			}

			clusterStat := statsMap[clusterID][tType]
			if clusterStat == nil {
				continue
			}

			var enrichedPoints []DataPoint

			for _, p := range tPoints {
				p.ClusterID = clusterID
				p.AgeZ = CalculateZscore(float64(p.Age), clusterStat.AgeMean, clusterStat.AgeStdDev)
				p.AskingZ = CalculateZscore(p.AskingPricePerSqm, clusterStat.AskingMean, clusterStat.AskingStdDev)
				p.EstimatedZ = CalculateZscore(p.EstimatedPricePerSqm, clusterStat.EstimatedMean, clusterStat.EstimatedStdDev)
				p.FloorZ = CalculateZscore(float64(p.FloorLevel), clusterStat.FloorMean, clusterStat.FloorStdDev)
				p.GeoPremiumZ = CalculateZscore(p.GeoPremium, clusterStat.GeoPremiumMean, clusterStat.GeoPremiumStdDev)
				p.EloZ = CalculateZscore(p.Elo, clusterStat.EloMean, clusterStat.EloStdDev)

				enrichedPoints = append(enrichedPoints, p)
				allEnrichedPoints = append(allEnrichedPoints, p)
			}

			sort.Slice(enrichedPoints, func(i, j int) bool {
				return enrichedPoints[i].Elo < enrichedPoints[j].Elo
			})

			cutoff := int(math.Ceil(float64(n) * 0.05))
			if cutoff == 0 && n > 0 {
				cutoff = 1
			}
			if cutoff*2 > n {
				cutoff = n / 2
			}

			var bottom5, top5 []DataPoint
			for i := 0; i < cutoff; i++ {
				bottom5 = append(bottom5, enrichedPoints[i])
			}
			for i := n - cutoff; i < n; i++ {
				top5 = append(top5, enrichedPoints[i])
			}

			clusterStat.Bottom5 = bottom5
			clusterStat.Top5 = top5
		}
	}

	return allEnrichedPoints, stats
}