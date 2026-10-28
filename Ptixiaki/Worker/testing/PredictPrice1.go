package testing

import t "Ptixiaki/Worker/types"

type PredictionResult struct {
	Price    float32
	Features []float32
}

func PredictPricePipeline(uid string, TreeAttr *t.ModelPredictor, Tree *t.ModelPredictor, Calculator Calculator) (PredictionResult, error) {
	Features, err := Calculator.PrepareInferencePayload(uid)
	if err != nil {

		return PredictionResult{}, err
	}
	prediction, err := PredictPrice(Features, Tree)
	if err != nil {
		return PredictionResult{}, err
	}
	return PredictionResult{Price: prediction, Features: Features}, nil
}
