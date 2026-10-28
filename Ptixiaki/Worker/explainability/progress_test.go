package explainability

import (
	"bytes"
	"log"
	"math"
	"strings"
	"testing"
	"time"
)

func TestProgressValidation(t *testing.T) {
	t.Parallel()
	for stage := range progressStageLabels {
		parser, _ := NewStreamParser("sell", nil)
		consumeRecord(t, parser, ProgressRecord{Type: "progress", Stage: stage, Total: 1})
		if _, err := parser.Finalize(); err == nil {
			t.Fatal("progress must not replace the required SHAP records")
		}
	}
	invalid := []ProgressRecord{
		{Stage: "unknown", Total: 1},
		{Stage: "shap_properties", Total: 0},
		{Stage: "shap_properties", Total: 10, Completed: -1},
		{Stage: "shap_properties", Total: 10, Completed: 11},
		{Stage: "shap_interactions", Total: 10, ElapsedSeconds: -1},
		{Stage: "shap_interactions", Total: 10, ElapsedSeconds: math.NaN()},
		{Stage: "shap_interactions", Total: 10, ElapsedSeconds: math.Inf(1)},
		{Stage: "onnx", Total: 10},
	}
	for _, record := range invalid {
		if err := validateProgressRecord(record); err == nil {
			t.Fatalf("accepted invalid progress: %+v", record)
		}
		if isFinite(record.ElapsedSeconds) {
			record.Type = "progress"
			parser, _ := NewStreamParser("sell", nil)
			if _, err := parser.ParseLine(marshalRecord(t, record)); err == nil {
				t.Fatalf("parser accepted invalid progress: %+v", record)
			}
		}
	}
}

func TestGoProgressLogsAndThrottling(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	reporter := progressLogger{transaction: "sell", printf: log.New(&output, "", 0).Printf}
	reporter.report(ProgressRecord{Stage: "onnx", Total: 1})
	reporter.report(ProgressRecord{Stage: "onnx", Total: 1, Completed: 1, ElapsedSeconds: 2})
	reporter.report(ProgressRecord{Stage: "shap_interactions", Total: 2000})
	reporter.lastPrinted = time.Now().Add(-6 * time.Second)
	reporter.report(ProgressRecord{Stage: "shap_interactions", Total: 2000, Completed: 500, ElapsedSeconds: 10})
	before := output.Len()
	reporter.report(ProgressRecord{Stage: "shap_interactions", Total: 2000, Completed: 501, ElapsedSeconds: 10.1})
	if output.Len() != before {
		t.Fatal("fast intermediate progress was not throttled")
	}
	reporter.report(ProgressRecord{Stage: "shap_interactions", Total: 2000, Completed: 2000, ElapsedSeconds: 40})
	for _, expected := range []string{
		"[TRAIN sell] Converting ONNX model: started",
		"Converting ONNX model: completed (elapsed 2.0s)",
		"SHAP interactions (sample): 0/2000 rows (0.0%)",
		"500/2000 rows (25.0%) | elapsed 10.0s | approx ETA 30s",
		"2000/2000 rows (100.0%) | elapsed 40.0s",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("missing %q in Go logs:\n%s", expected, output.String())
		}
	}
}
