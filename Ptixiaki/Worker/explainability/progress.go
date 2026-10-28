package explainability

import (
	"fmt"
	"time"
)

// ProgressRecord contains measurements only. Go formats and prints the logs;
// progress records never contribute to the persisted SHAP dataset.
type ProgressRecord struct {
	Type           string  `json:"type"`
	Stage          string  `json:"stage"`
	Completed      int     `json:"completed"`
	Total          int     `json:"total"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
}

var progressStageLabels = map[string]string{
	"prepare":           "Preparing training data",
	"validation":        "Fitting validation model",
	"training":          "Fitting final all-row model",
	"onnx":              "Converting ONNX model",
	"artifacts":         "Staging model artifacts",
	"shap_properties":   "Property SHAP (all properties)",
	"shap_interactions": "SHAP interactions (sample)",
	"activate":          "Activating model artifacts",
}

func validateProgressRecord(record ProgressRecord) error {
	if _, ok := progressStageLabels[record.Stage]; !ok {
		return fmt.Errorf("unknown progress stage %q", record.Stage)
	}
	if record.Total <= 0 || record.Completed < 0 || record.Completed > record.Total {
		return fmt.Errorf("invalid progress counts for %q: %d/%d", record.Stage, record.Completed, record.Total)
	}
	if !isFinite(record.ElapsedSeconds) || record.ElapsedSeconds < 0 {
		return fmt.Errorf("invalid progress elapsed_seconds for %q", record.Stage)
	}
	if record.Stage != "shap_properties" && record.Stage != "shap_interactions" && record.Total != 1 {
		return fmt.Errorf("stage %q must use total=1", record.Stage)
	}
	return nil
}

type progressLogger struct {
	transaction string
	printf      func(string, ...any)
	lastStage   string
	lastPrinted time.Time
}

func (p *progressLogger) report(record ProgressRecord) {
	now := time.Now()
	// Always show stage boundaries; avoid flooding the terminal with fast batches.
	if record.Stage == p.lastStage && record.Completed != 0 && record.Completed != record.Total && now.Sub(p.lastPrinted) < 5*time.Second {
		return
	}
	p.lastStage, p.lastPrinted = record.Stage, now
	label := progressStageLabels[record.Stage]
	if record.Stage != "shap_properties" && record.Stage != "shap_interactions" {
		state := "started"
		if record.Completed == record.Total {
			state = "completed"
		}
		p.printf("[TRAIN %s] %s: %s (elapsed %.1fs)", p.transaction, label, state, record.ElapsedSeconds)
		return
	}
	eta := ""
	if record.Completed > 0 && record.Completed < record.Total {
		remaining := record.ElapsedSeconds / float64(record.Completed) * float64(record.Total-record.Completed)
		eta = fmt.Sprintf(" | approx ETA %.0fs", remaining)
	}
	p.printf("[TRAIN %s] %s: %d/%d rows (%.1f%%) | elapsed %.1fs%s",
		p.transaction, label, record.Completed, record.Total,
		100*float64(record.Completed)/float64(record.Total), record.ElapsedSeconds, eta)
}
