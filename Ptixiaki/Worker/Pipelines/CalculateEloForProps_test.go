package pipelines

import (
	"math"
	"testing"
)

func TestCalculateResidualPercentageBelowEstimateIsNegative(t *testing.T) {
	got := CalculateResidualPercentage(2650, 4268.066406)
	want := -61.05910966037735
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("CalculateResidualPercentage() = %v, want %v", got, want)
	}
}

func TestLowerAskingRelativeToEstimateProducesHigherElo(t *testing.T) {
	goodDealResidualZ := -1.0
	overpricedResidualZ := 1.0

	goodDealElo := CalculateEloScore(goodDealResidualZ, goodDealResidualZ, 0)
	overpricedElo := CalculateEloScore(overpricedResidualZ, overpricedResidualZ, 0)
	if goodDealElo <= overpricedElo {
		t.Fatalf("good deal Elo %v must be greater than overpriced Elo %v", goodDealElo, overpricedElo)
	}
}

func TestCalculateMathForDealValuePreservesDecimals(t *testing.T) {
	properties := []EstimationRow{
		{ResidualPercentage: -1.5},
		{ResidualPercentage: 0.5},
	}

	std, mean := CalculateMathForDealValue(properties)
	if mean != -0.5 {
		t.Fatalf("mean = %v, want -0.5", mean)
	}
	if std != 1 {
		t.Fatalf("std = %v, want 1", std)
	}
}

func TestCalculateZScoreWithZeroStandardDeviation(t *testing.T) {
	if got := CalculateZScore(5, 5, 0); got != 0 {
		t.Fatalf("CalculateZScore() = %v, want 0", got)
	}
}
