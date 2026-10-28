package marketanalysis

import "math"

func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func CalculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sumSquaredDifferences float64
	for _, value := range values {
		difference := value - mean
		sumSquaredDifferences += difference * difference
	}
	variance := sumSquaredDifferences / float64(len(values))
	return math.Sqrt(variance)
}

func CalculateZscore(value float64, mean float64, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (value - mean) / stdDev
}

