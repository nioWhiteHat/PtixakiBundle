package explainability

import (
	"math"
	"strings"
	"testing"
)

func TestCancellationAwareAdditivityMatchesOnlineAndTraining(t *testing.T) {
	t.Parallel()
	for _, prediction := range []float64{0, 1, -1} {
		contributions := make([]float64, FeatureCount)
		contributions[0] = prediction - 1000 + 0.005
		reconstructed := 1000 + contributions[0]
		record := PropertyRecord{
			UID: "cancellation", Contributions: contributions,
			PredictionPricePerSqm: prediction, AdditivityError: reconstructed - prediction,
		}
		parser := &StreamParser{meta: &MetaRecord{BaselinePricePerSqm: 1000}}
		response := ExplainPropertyResponse{
			Transaction: "sell", FeatureNames: ExpectedFeatureNames(),
			Contributions: contributions, BaselinePricePerSqm: 1000,
			PredictionPricePerSqm: prediction, AdditivityError: record.AdditivityError,
			ModelSHA256: strings.Repeat("c", 64), TrainedAt: "2026-09-15T00:00:00Z",
		}
		if err := parser.validateProperty(record); err != nil {
			t.Fatalf("training rejected float32 cancellation at %g: %v", prediction, err)
		}
		if err := ValidateExplainPropertyResponse(response, "sell", float32(prediction)); err != nil {
			t.Fatalf("online rejected float32 cancellation at %g: %v", prediction, err)
		}
		// The reported error must still match the independently computed error.
		record.AdditivityError = 0
		response.AdditivityError = 0
		if parser.validateProperty(record) == nil || ValidateExplainPropertyResponse(response, "sell", float32(prediction)) == nil {
			t.Fatal("accepted an incorrect reported error")
		}
		// A materially wrong contribution must not slip through the new scale.
		contributions[0] += 1
		record.AdditivityError = 1000 + contributions[0] - prediction
		response.AdditivityError = record.AdditivityError
		if parser.validateProperty(record) == nil || ValidateExplainPropertyResponse(response, "sell", float32(prediction)) == nil {
			t.Fatal("accepted a materially non-additive explanation")
		}
		contributions[0], contributions[1] = math.MaxFloat64, math.MaxFloat64
		if parser.validateProperty(record) == nil || ValidateExplainPropertyResponse(response, "sell", float32(prediction)) == nil {
			t.Fatal("accepted overflow in reconstruction/tolerance")
		}
	}
}

func TestAdditivityToleranceScale(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		prediction, baseline float64
		contributions        []float64
		want                 float64
	}{
		{0, 1000, []float64{-999.995}, 0.02099995},
		{10000, 1000, []float64{9000}, 0.101},
		{0, 0, []float64{0}, 0.001},
		{10000, 0, []float64{1}, 0.101},
	} {
		got := shapAdditivityTolerance(test.prediction, test.baseline, test.contributions)
		if math.Abs(got-test.want) > 1e-12 {
			t.Fatalf("tolerance = %.12g, expected %.12g", got, test.want)
		}
	}
}
