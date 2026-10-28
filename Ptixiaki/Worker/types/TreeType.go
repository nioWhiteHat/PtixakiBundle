package types

import (
	"sync"

	"github.com/yalue/onnxruntime_go"
)

type ModelPredictor struct {
	Mu           sync.Mutex
	Session      *onnxruntime_go.AdvancedSession
	InputTensor  *onnxruntime_go.Tensor[float32]
	OutputTensor *onnxruntime_go.Tensor[float32]
	InputData    []float32
	OutputData   []float32
}