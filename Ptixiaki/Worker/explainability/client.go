package explainability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type ExplainPropertyRequest struct {
	Transaction string    `json:"transaction"`
	Features    []float32 `json:"features"`
}

type ExplainPropertyResponse struct {
	Transaction           string    `json:"transaction"`
	FeatureNames          []string  `json:"feature_names"`
	Contributions         []float64 `json:"contributions"`
	BaselinePricePerSqm   float64   `json:"baseline_price_per_sqm"`
	PredictionPricePerSqm float64   `json:"prediction_price_per_sqm"`
	AdditivityError       float64   `json:"additivity_error"`
	ModelSHA256           string    `json:"model_sha256"`
	TrainedAt             string    `json:"trained_at"`
}

type ReloadedModel struct {
	Transaction       string `json:"transaction"`
	ModelSHA256       string `json:"model_sha256"`
	TrainedAt         string `json:"trained_at"`
	FeatureCount      int    `json:"feature_count"`
	ArtifactDirectory string `json:"artifact_directory"`
}

type ReloadModelsResponse struct {
	Message string          `json:"message"`
	Models  []ReloadedModel `json:"models"`
}

func NewClient() *Client {
	return &Client{
		BaseURL: DefaultServiceBaseURL,
		HTTPClient: &http.Client{
			Timeout: HTTPTimeout,
		},
	}
}

func (c *Client) ExplainProperty(ctx context.Context, transaction string, features []float32, onnxPrediction float32) (ExplainPropertyResponse, error) {
	if err := ValidateTransaction(transaction); err != nil {
		return ExplainPropertyResponse{}, err
	}
	if len(features) != FeatureCount {
		return ExplainPropertyResponse{}, fmt.Errorf("feature count mismatch: got %d, expected %d", len(features), FeatureCount)
	}
	for i, value := range features {
		if !isFinite(float64(value)) {
			return ExplainPropertyResponse{}, fmt.Errorf("feature %d is not finite", i)
		}
	}
	if !isFinite(float64(onnxPrediction)) {
		return ExplainPropertyResponse{}, fmt.Errorf("ONNX prediction is not finite")
	}

	body, err := json.Marshal(ExplainPropertyRequest{Transaction: transaction, Features: features})
	if err != nil {
		return ExplainPropertyResponse{}, fmt.Errorf("encode ExplainProperty request: %w", err)
	}
	response, err := c.post(ctx, ExplainPropertyPath, body)
	if err != nil {
		return ExplainPropertyResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ExplainPropertyResponse{}, serviceStatusError(response)
	}

	parsed, err := ParseExplainPropertyResponse(response.Body, transaction, onnxPrediction)
	if err != nil {
		return ExplainPropertyResponse{}, err
	}
	return parsed, nil
}

func ParseExplainPropertyResponse(body io.Reader, expectedTransaction string, onnxPrediction float32) (ExplainPropertyResponse, error) {
	var response ExplainPropertyResponse
	if err := decodeLimitedJSON(body, &response); err != nil {
		return ExplainPropertyResponse{}, fmt.Errorf("decode ExplainProperty response: %w", err)
	}
	if err := ValidateExplainPropertyResponse(response, expectedTransaction, onnxPrediction); err != nil {
		return ExplainPropertyResponse{}, err
	}
	return response, nil
}

func ValidateExplainPropertyResponse(response ExplainPropertyResponse, expectedTransaction string, onnxPrediction float32) error {
	if err := ValidateTransaction(expectedTransaction); err != nil {
		return err
	}
	if response.Transaction != expectedTransaction {
		return fmt.Errorf("explanation transaction mismatch: got %q, expected %q", response.Transaction, expectedTransaction)
	}
	if err := ValidateFeatureNames(response.FeatureNames); err != nil {
		return err
	}
	if len(response.Contributions) != FeatureCount {
		return fmt.Errorf("explanation has %d contributions, expected %d", len(response.Contributions), FeatureCount)
	}
	for i, value := range response.Contributions {
		if !isFinite(value) {
			return fmt.Errorf("explanation contribution %d is not finite", i)
		}
	}
	for name, value := range map[string]float64{
		"baseline_price_per_sqm":   response.BaselinePricePerSqm,
		"prediction_price_per_sqm": response.PredictionPricePerSqm,
		"additivity_error":         response.AdditivityError,
	} {
		if !isFinite(value) {
			return fmt.Errorf("explanation %s must be finite", name)
		}
	}
	if !isFinite(float64(onnxPrediction)) {
		return fmt.Errorf("ONNX prediction is not finite")
	}

	reconstructed := response.BaselinePricePerSqm
	for _, contribution := range response.Contributions {
		reconstructed += contribution
	}
	computedError := reconstructed - response.PredictionPricePerSqm
	additivityTolerance := shapAdditivityTolerance(response.PredictionPricePerSqm, response.BaselinePricePerSqm, response.Contributions)
	if !isFinite(computedError) || !isFinite(additivityTolerance) || math.Abs(computedError) > additivityTolerance {
		return fmt.Errorf("explanation reconstruction error %.9g exceeds tolerance %.9g", computedError, additivityTolerance)
	}
	if math.Abs(response.AdditivityError) > additivityTolerance || !withinTolerance(response.AdditivityError, computedError, AdditivityAbsTolerance, AdditivityRelTolerance) {
		return fmt.Errorf("reported additivity error %.9g does not match computed %.9g", response.AdditivityError, computedError)
	}
	if !withinTolerance(response.PredictionPricePerSqm, float64(onnxPrediction), ONNXNativeAbsTolerance, ONNXNativeRelTolerance) {
		return fmt.Errorf(
			"native prediction %.9g differs from ONNX prediction %.9g beyond abs=%g rel=%g",
			response.PredictionPricePerSqm,
			onnxPrediction,
			ONNXNativeAbsTolerance,
			ONNXNativeRelTolerance,
		)
	}
	if err := validateSHA256(response.ModelSHA256); err != nil {
		return err
	}
	if _, err := parseTimestamp(response.TrainedAt); err != nil {
		return err
	}
	return nil
}

func (c *Client) ReloadModels(ctx context.Context, transactions []string, expectedModelSHA256 map[string]string) (ReloadModelsResponse, error) {
	if len(transactions) == 0 {
		transactions = []string{"sell", "rent"}
	}
	seen := make(map[string]struct{}, len(transactions))
	for _, transaction := range transactions {
		if err := ValidateTransaction(transaction); err != nil {
			return ReloadModelsResponse{}, err
		}
		if _, duplicate := seen[transaction]; duplicate {
			return ReloadModelsResponse{}, fmt.Errorf("duplicate reload transaction %q", transaction)
		}
		seen[transaction] = struct{}{}
		expectedHash, ok := expectedModelSHA256[transaction]
		if !ok {
			return ReloadModelsResponse{}, fmt.Errorf("missing expected model hash for %q", transaction)
		}
		if err := validateSHA256(expectedHash); err != nil {
			return ReloadModelsResponse{}, fmt.Errorf("expected hash for %q: %w", transaction, err)
		}
	}
	if len(expectedModelSHA256) != len(transactions) {
		return ReloadModelsResponse{}, fmt.Errorf("expected model hashes must exactly match reload transactions")
	}

	body, err := json.Marshal(struct {
		Transactions        []string          `json:"transactions"`
		ExpectedModelSHA256 map[string]string `json:"expected_model_sha256"`
	}{Transactions: transactions, ExpectedModelSHA256: expectedModelSHA256})
	if err != nil {
		return ReloadModelsResponse{}, fmt.Errorf("encode ReloadExplainModels request: %w", err)
	}
	response, err := c.post(ctx, ReloadModelsPath, body)
	if err != nil {
		return ReloadModelsResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ReloadModelsResponse{}, serviceStatusError(response)
	}

	var parsed ReloadModelsResponse
	if err := decodeLimitedJSON(response.Body, &parsed); err != nil {
		return ReloadModelsResponse{}, fmt.Errorf("decode ReloadExplainModels response: %w", err)
	}
	if parsed.Message != "Success" {
		return ReloadModelsResponse{}, fmt.Errorf("unexpected reload message %q", parsed.Message)
	}
	if len(parsed.Models) != len(transactions) {
		return ReloadModelsResponse{}, fmt.Errorf("reload returned %d models, expected %d", len(parsed.Models), len(transactions))
	}
	for i, model := range parsed.Models {
		if model.Transaction != transactions[i] {
			return ReloadModelsResponse{}, fmt.Errorf("reload model %d transaction mismatch: got %q, expected %q", i, model.Transaction, transactions[i])
		}
		if model.FeatureCount != FeatureCount {
			return ReloadModelsResponse{}, fmt.Errorf("reload model %q has %d features, expected %d", model.Transaction, model.FeatureCount, FeatureCount)
		}
		if err := validateSHA256(model.ModelSHA256); err != nil {
			return ReloadModelsResponse{}, err
		}
		if model.ModelSHA256 != expectedModelSHA256[model.Transaction] {
			return ReloadModelsResponse{}, fmt.Errorf("reload model %q hash does not match the Worker publication", model.Transaction)
		}
		if _, err := parseTimestamp(model.TrainedAt); err != nil {
			return ReloadModelsResponse{}, err
		}
		if strings.TrimSpace(model.ArtifactDirectory) == "" {
			return ReloadModelsResponse{}, fmt.Errorf("reload model %q omitted artifact_directory", model.Transaction)
		}
	}
	return parsed, nil
}

func PersistPropertyExplanation(ctx context.Context, db *pgxpool.Pool, propertyUID string, response ExplainPropertyResponse) error {
	tables, err := TablesForTransaction(response.Transaction)
	if err != nil {
		return err
	}
	if strings.TrimSpace(propertyUID) == "" {
		return fmt.Errorf("property uid is empty")
	}
	if _, err := parseTimestamp(response.TrainedAt); err != nil {
		return err
	}
	explainedAt := time.Now().UTC()

	columns := append([]string{"property_uid"}, PropertyImpactColumns()...)
	columns = append(columns,
		"baseline_price_per_sqm",
		"explained_prediction_per_sqm",
		"additivity_error",
		"model_sha256",
		"explained_at",
	)
	quotedColumns := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	assignments := make([]string, 0, len(columns)-1)
	for i, column := range columns {
		quoted := pgx.Identifier{column}.Sanitize()
		quotedColumns[i] = quoted
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		if i > 0 {
			assignments = append(assignments, fmt.Sprintf("%s = EXCLUDED.%s", quoted, quoted))
		}
	}

	args := make([]any, 0, len(columns))
	args = append(args, propertyUID)
	for _, contribution := range response.Contributions {
		args = append(args, contribution)
	}
	args = append(args,
		response.BaselinePricePerSqm,
		response.PredictionPricePerSqm,
		response.AdditivityError,
		response.ModelSHA256,
		explainedAt,
	)

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (property_uid) DO UPDATE SET %s",
		pgx.Identifier{tables.PropertyAnalysis}.Sanitize(),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(assignments, ", "),
	)
	if _, err := db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert property explanation: %w", err)
	}
	return nil
}

func ExplainAndPersist(ctx context.Context, db *pgxpool.Pool, client *Client, propertyUID, transaction string, features []float32, onnxPrediction float32) error {
	response, err := client.ExplainProperty(ctx, transaction, features, onnxPrediction)
	if err != nil {
		return err
	}
	return PersistPropertyExplanation(ctx, db, propertyUID, response)
}

func (c *Client) post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultServiceBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: HTTPTimeout}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create %s request: %w", path, err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	return response, nil
}

func decodeLimitedJSON(body io.Reader, destination any) error {
	limited := io.LimitReader(body, MaxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > MaxResponseBytes {
		return fmt.Errorf("response exceeds %d bytes", MaxResponseBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values in response")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func serviceStatusError(response *http.Response) error {
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBytes))
	if err != nil {
		return fmt.Errorf("service returned HTTP %d", response.StatusCode)
	}
	var detail struct {
		Detail any `json:"detail"`
	}
	if json.Unmarshal(data, &detail) == nil && detail.Detail != nil {
		return fmt.Errorf("service returned HTTP %d: %v", response.StatusCode, detail.Detail)
	}
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return fmt.Errorf("service returned HTTP %d: %s", response.StatusCode, message)
}
