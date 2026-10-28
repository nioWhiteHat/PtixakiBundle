package explainability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseExplainPropertyResponse(t *testing.T) {
	t.Parallel()

	response := ExplainPropertyResponse{
		Transaction:           "sell",
		FeatureNames:          ExpectedFeatureNames(),
		Contributions:         make([]float64, FeatureCount),
		BaselinePricePerSqm:   100,
		PredictionPricePerSqm: 100,
		AdditivityError:       0,
		ModelSHA256:           strings.Repeat("c", 64),
		TrainedAt:             "2026-09-11T10:00:00.123456Z",
	}
	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseExplainPropertyResponse(strings.NewReader(string(body)), "sell", 100.05)
	if err != nil {
		t.Fatalf("parse valid response: %v", err)
	}
	if parsed.ModelSHA256 != response.ModelSHA256 {
		t.Fatalf("hash mismatch: got %q", parsed.ModelSHA256)
	}

	if _, err := ParseExplainPropertyResponse(strings.NewReader(string(body)), "sell", 101); err == nil {
		t.Fatal("expected stale/native-vs-ONNX response mismatch to fail")
	}
}

func TestReloadModelsSendsAndVerifiesExpectedHashes(t *testing.T) {
	t.Parallel()
	expectedHash := strings.Repeat("d", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != ReloadModelsPath {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var payload struct {
			Transactions        []string          `json:"transactions"`
			ExpectedModelSHA256 map[string]string `json:"expected_model_sha256"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(payload.Transactions) != 1 || payload.Transactions[0] != "sell" || payload.ExpectedModelSHA256["sell"] != expectedHash {
			http.Error(w, "missing hash handshake", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(ReloadModelsResponse{
			Message: "Success",
			Models: []ReloadedModel{{
				Transaction:       "sell",
				ModelSHA256:       expectedHash,
				TrainedAt:         "2026-09-12T10:00:00Z",
				FeatureCount:      FeatureCount,
				ArtifactDirectory: "test-artifacts",
			}},
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	response, err := client.ReloadModels(
		context.Background(),
		[]string{"sell"},
		map[string]string{"sell": expectedHash},
	)
	if err != nil {
		t.Fatalf("reload models: %v", err)
	}
	if response.Models[0].ModelSHA256 != expectedHash {
		t.Fatalf("reloaded hash = %q, want %q", response.Models[0].ModelSHA256, expectedHash)
	}
	if _, err := client.ReloadModels(context.Background(), []string{"sell"}, nil); err == nil {
		t.Fatal("expected a missing hash handshake to fail")
	}
}
