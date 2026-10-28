package explainability

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

type MetaRecord struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	Transaction   string `json:"transaction"`
	Target        struct {
		Name string `json:"name"`
		Unit string `json:"unit"`
	} `json:"target"`
	FeatureNames           []string `json:"feature_names"`
	ModelFeatureNames      []string `json:"model_feature_names"`
	FeatureCount           int      `json:"feature_count"`
	PropertyCount          int      `json:"property_count"`
	InteractionCount       int      `json:"interaction_count"`
	InteractionSampleCount int      `json:"interaction_sample_count"`
	InteractionSampleSeed  int      `json:"interaction_sample_seed"`
	BaselinePricePerSqm    float64  `json:"baseline_price_per_sqm"`
	TrainingR2             float64  `json:"training_r2"`
	ValidationR2           float64  `json:"validation_r2"`
	ModelSHA256            string   `json:"model_sha256"`
	TrainedAt              string   `json:"trained_at"`
}

type PropertyRecord struct {
	Type                  string    `json:"type"`
	UID                   string    `json:"uid"`
	Contributions         []float64 `json:"contributions"`
	PredictionPricePerSqm float64   `json:"prediction_price_per_sqm"`
	AdditivityError       float64   `json:"additivity_error"`
}

type ImportanceRecord struct {
	Type        string  `json:"type"`
	FeatureName string  `json:"feature_name"`
	MeanAbs     float64 `json:"mean_abs"`
	MeanSigned  float64 `json:"mean_signed"`
	Percentage  float64 `json:"percentage"`
	Rank        int     `json:"rank"`
}

type InteractionRecord struct {
	Type       string  `json:"type"`
	Feature1   string  `json:"feature_1"`
	Feature2   string  `json:"feature_2"`
	MeanAbs    float64 `json:"mean_abs"`
	MeanSigned float64 `json:"mean_signed"`
	Rank       int     `json:"rank"`
}

type TrainingRecords struct {
	Meta          MetaRecord
	TrainedAt     time.Time
	Importance    []ImportanceRecord
	Interactions  []InteractionRecord
	PropertyCount int
}

type StreamParser struct {
	expectedTransaction string
	onProperty          func(PropertyRecord) error
	onProgress          func(ProgressRecord)
	meta                *MetaRecord
	trainedAt           time.Time
	propertyUIDs        map[string]struct{}
	importanceFeatures  map[string]struct{}
	importanceRanks     map[int]struct{}
	interactionPairs    map[string]struct{}
	interactionRanks    map[int]struct{}
	importance          []ImportanceRecord
	interactions        []InteractionRecord
}

func NewStreamParser(expectedTransaction string, onProperty func(PropertyRecord) error) (*StreamParser, error) {
	if err := ValidateTransaction(expectedTransaction); err != nil {
		return nil, err
	}
	if onProperty == nil {
		onProperty = func(PropertyRecord) error { return nil }
	}
	return &StreamParser{
		expectedTransaction: expectedTransaction,
		onProperty:          onProperty,
		propertyUIDs:        make(map[string]struct{}),
		importanceFeatures:  make(map[string]struct{}),
		importanceRanks:     make(map[int]struct{}),
		interactionPairs:    make(map[string]struct{}),
		interactionRanks:    make(map[int]struct{}),
	}, nil
}

// SetProgressHandler installs an optional observer. Progress may precede meta
// (for example during model fitting) but never bypasses data validation.
func (p *StreamParser) SetProgressHandler(handler func(ProgressRecord)) {
	p.onProgress = handler
}

// ParseLine consumes the line-oriented stdout protocol. Non-protocol lines are
// reported as unrecognized so the command runner can reject stdout pollution.
func (p *StreamParser) ParseLine(line string) (bool, error) {
	if !strings.HasPrefix(line, RecordPrefix) {
		return false, nil
	}
	payload := strings.TrimPrefix(line, RecordPrefix)
	if strings.TrimSpace(payload) == "" {
		return true, fmt.Errorf("empty SHAP record")
	}

	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return true, fmt.Errorf("decode SHAP record envelope: %w", err)
	}

	switch envelope.Type {
	case "progress":
		var record ProgressRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return true, fmt.Errorf("decode progress record: %w", err)
		}
		if err := validateProgressRecord(record); err != nil {
			return true, err
		}
		if p.onProgress != nil {
			p.onProgress(record)
		}

	case "meta":
		if p.meta != nil {
			return true, fmt.Errorf("duplicate meta record")
		}
		if len(p.propertyUIDs) != 0 || len(p.importance) != 0 || len(p.interactions) != 0 {
			return true, fmt.Errorf("meta record must be emitted first")
		}
		var record MetaRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return true, fmt.Errorf("decode meta record: %w", err)
		}
		trainedAt, err := validateMetaRecord(record, p.expectedTransaction)
		if err != nil {
			return true, err
		}
		p.meta = &record
		p.trainedAt = trainedAt

	case "property":
		if p.meta == nil {
			return true, fmt.Errorf("property record arrived before meta")
		}
		var record PropertyRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return true, fmt.Errorf("decode property record: %w", err)
		}
		if err := p.validateProperty(record); err != nil {
			return true, err
		}
		if _, duplicate := p.propertyUIDs[record.UID]; duplicate {
			return true, fmt.Errorf("duplicate property uid %q", record.UID)
		}
		if err := p.onProperty(record); err != nil {
			return true, fmt.Errorf("stage property %q: %w", record.UID, err)
		}
		p.propertyUIDs[record.UID] = struct{}{}

	case "importance":
		if p.meta == nil {
			return true, fmt.Errorf("importance record arrived before meta")
		}
		var record ImportanceRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return true, fmt.Errorf("decode importance record: %w", err)
		}
		if err := p.validateImportance(record); err != nil {
			return true, err
		}
		p.importanceFeatures[record.FeatureName] = struct{}{}
		p.importanceRanks[record.Rank] = struct{}{}
		p.importance = append(p.importance, record)

	case "interaction":
		if p.meta == nil {
			return true, fmt.Errorf("interaction record arrived before meta")
		}
		var record InteractionRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return true, fmt.Errorf("decode interaction record: %w", err)
		}
		if err := p.validateInteraction(record); err != nil {
			return true, err
		}
		pairKey := canonicalPair(record.Feature1, record.Feature2)
		p.interactionPairs[pairKey] = struct{}{}
		p.interactionRanks[record.Rank] = struct{}{}
		p.interactions = append(p.interactions, record)

	default:
		return true, fmt.Errorf("unknown SHAP record type %q", envelope.Type)
	}

	return true, nil
}

func (p *StreamParser) Finalize() (TrainingRecords, error) {
	if p.meta == nil {
		return TrainingRecords{}, fmt.Errorf("missing meta record")
	}
	meta := *p.meta
	if len(p.propertyUIDs) != meta.PropertyCount {
		return TrainingRecords{}, fmt.Errorf("property count mismatch: got %d, expected %d", len(p.propertyUIDs), meta.PropertyCount)
	}
	if len(p.importance) != meta.FeatureCount {
		return TrainingRecords{}, fmt.Errorf("importance count mismatch: got %d, expected %d", len(p.importance), meta.FeatureCount)
	}
	if len(p.interactions) != meta.InteractionCount {
		return TrainingRecords{}, fmt.Errorf("interaction count mismatch: got %d, expected %d", len(p.interactions), meta.InteractionCount)
	}
	for rank := 1; rank <= meta.FeatureCount; rank++ {
		if _, ok := p.importanceRanks[rank]; !ok {
			return TrainingRecords{}, fmt.Errorf("missing importance rank %d", rank)
		}
	}
	for rank := 1; rank <= meta.InteractionCount; rank++ {
		if _, ok := p.interactionRanks[rank]; !ok {
			return TrainingRecords{}, fmt.Errorf("missing interaction rank %d", rank)
		}
	}
	for _, feature := range expectedFeatureNames {
		if _, ok := p.importanceFeatures[feature]; !ok {
			return TrainingRecords{}, fmt.Errorf("missing importance feature %q", feature)
		}
	}
	var totalMeanAbs, totalPercentage float64
	for _, record := range p.importance {
		totalMeanAbs += record.MeanAbs
		totalPercentage += record.Percentage
	}
	if totalMeanAbs == 0 {
		if math.Abs(totalPercentage) > 1e-9 {
			return TrainingRecords{}, fmt.Errorf("all-zero model has non-zero importance percentage sum %.9g", totalPercentage)
		}
	} else if math.Abs(totalPercentage-100) > 0.01 {
		return TrainingRecords{}, fmt.Errorf("importance percentages sum to %.9g, expected 100", totalPercentage)
	}

	return TrainingRecords{
		Meta:          meta,
		TrainedAt:     p.trainedAt,
		Importance:    slices.Clone(p.importance),
		Interactions:  slices.Clone(p.interactions),
		PropertyCount: len(p.propertyUIDs),
	}, nil
}

func validateMetaRecord(record MetaRecord, expectedTransaction string) (time.Time, error) {
	if record.SchemaVersion != 1 {
		return time.Time{}, fmt.Errorf("unsupported SHAP schema_version %d", record.SchemaVersion)
	}
	if record.Transaction != expectedTransaction {
		return time.Time{}, fmt.Errorf("meta transaction mismatch: got %q, expected %q", record.Transaction, expectedTransaction)
	}
	if err := ValidateFeatureNames(record.FeatureNames); err != nil {
		return time.Time{}, err
	}
	if record.Target.Name != TargetName || record.Target.Unit != TargetUnit {
		return time.Time{}, fmt.Errorf("unexpected target contract: name=%q unit=%q", record.Target.Name, record.Target.Unit)
	}
	expectedModelFeatures := make([]string, FeatureCount)
	for index := range expectedModelFeatures {
		expectedModelFeatures[index] = fmt.Sprintf("f%d", index)
	}
	if !slices.Equal(record.ModelFeatureNames, expectedModelFeatures) {
		return time.Time{}, fmt.Errorf("model feature order mismatch: got %v, expected %v", record.ModelFeatureNames, expectedModelFeatures)
	}
	if record.FeatureCount != FeatureCount || record.FeatureCount != len(record.FeatureNames) {
		return time.Time{}, fmt.Errorf("feature_count mismatch: got %d, expected %d", record.FeatureCount, FeatureCount)
	}
	if record.PropertyCount <= 0 {
		return time.Time{}, fmt.Errorf("property_count must be positive, got %d", record.PropertyCount)
	}
	if record.InteractionSampleCount <= 0 || record.InteractionSampleCount > record.PropertyCount {
		return time.Time{}, fmt.Errorf("interaction_sample_count must be in 1..%d, got %d", record.PropertyCount, record.InteractionSampleCount)
	}
	if record.InteractionSampleSeed < 0 {
		return time.Time{}, fmt.Errorf("interaction_sample_seed must be non-negative")
	}
	expectedInteractions := record.FeatureCount * (record.FeatureCount - 1) / 2
	if record.InteractionCount != expectedInteractions {
		return time.Time{}, fmt.Errorf("interaction_count mismatch: got %d, expected %d", record.InteractionCount, expectedInteractions)
	}
	for name, value := range map[string]float64{
		"baseline_price_per_sqm": record.BaselinePricePerSqm,
		"training_r2":            record.TrainingR2,
		"validation_r2":          record.ValidationR2,
	} {
		if !isFinite(value) {
			return time.Time{}, fmt.Errorf("meta %s must be finite", name)
		}
	}
	if err := validateSHA256(record.ModelSHA256); err != nil {
		return time.Time{}, err
	}
	return parseTimestamp(record.TrainedAt)
}

func (p *StreamParser) validateProperty(record PropertyRecord) error {
	if strings.TrimSpace(record.UID) == "" {
		return fmt.Errorf("property uid is empty")
	}
	if len(record.Contributions) != FeatureCount {
		return fmt.Errorf("property %q has %d contributions, expected %d", record.UID, len(record.Contributions), FeatureCount)
	}
	for i, value := range record.Contributions {
		if !isFinite(value) {
			return fmt.Errorf("property %q contribution %d is not finite", record.UID, i)
		}
	}
	if !isFinite(record.PredictionPricePerSqm) || !isFinite(record.AdditivityError) {
		return fmt.Errorf("property %q prediction/additivity values must be finite", record.UID)
	}
	reconstructed := p.meta.BaselinePricePerSqm
	for _, contribution := range record.Contributions {
		reconstructed += contribution
	}
	computedError := reconstructed - record.PredictionPricePerSqm
	tolerance := shapAdditivityTolerance(record.PredictionPricePerSqm, p.meta.BaselinePricePerSqm, record.Contributions)
	if !isFinite(computedError) || !isFinite(tolerance) || math.Abs(computedError) > tolerance {
		return fmt.Errorf("property %q SHAP reconstruction error %.9g exceeds tolerance %.9g", record.UID, computedError, tolerance)
	}
	if math.Abs(record.AdditivityError) > tolerance || !withinTolerance(record.AdditivityError, computedError, AdditivityAbsTolerance, AdditivityRelTolerance) {
		return fmt.Errorf("property %q reported additivity error %.9g does not match computed %.9g", record.UID, record.AdditivityError, computedError)
	}
	return nil
}

func (p *StreamParser) validateImportance(record ImportanceRecord) error {
	if !slices.Contains(expectedFeatureNames, record.FeatureName) {
		return fmt.Errorf("unknown importance feature %q", record.FeatureName)
	}
	if _, duplicate := p.importanceFeatures[record.FeatureName]; duplicate {
		return fmt.Errorf("duplicate importance feature %q", record.FeatureName)
	}
	if record.Rank < 1 || record.Rank > FeatureCount {
		return fmt.Errorf("importance rank %d is outside 1..%d", record.Rank, FeatureCount)
	}
	if _, duplicate := p.importanceRanks[record.Rank]; duplicate {
		return fmt.Errorf("duplicate importance rank %d", record.Rank)
	}
	if !isFinite(record.MeanAbs) || record.MeanAbs < 0 || !isFinite(record.MeanSigned) || !isFinite(record.Percentage) {
		return fmt.Errorf("importance values for %q are invalid", record.FeatureName)
	}
	if record.Percentage < 0 || record.Percentage > 100 {
		return fmt.Errorf("importance percentage for %q is outside 0..100", record.FeatureName)
	}
	return nil
}

func (p *StreamParser) validateInteraction(record InteractionRecord) error {
	if !slices.Contains(expectedFeatureNames, record.Feature1) || !slices.Contains(expectedFeatureNames, record.Feature2) {
		return fmt.Errorf("unknown interaction pair %q/%q", record.Feature1, record.Feature2)
	}
	if record.Feature1 == record.Feature2 {
		return fmt.Errorf("self interaction %q is not part of the pair contract", record.Feature1)
	}
	pairKey := canonicalPair(record.Feature1, record.Feature2)
	if _, duplicate := p.interactionPairs[pairKey]; duplicate {
		return fmt.Errorf("duplicate interaction pair %q/%q", record.Feature1, record.Feature2)
	}
	if record.Rank < 1 || record.Rank > p.meta.InteractionCount {
		return fmt.Errorf("interaction rank %d is outside 1..%d", record.Rank, p.meta.InteractionCount)
	}
	if _, duplicate := p.interactionRanks[record.Rank]; duplicate {
		return fmt.Errorf("duplicate interaction rank %d", record.Rank)
	}
	if !isFinite(record.MeanAbs) || record.MeanAbs < 0 || !isFinite(record.MeanSigned) {
		return fmt.Errorf("interaction values for %q/%q are invalid", record.Feature1, record.Feature2)
	}
	return nil
}

func canonicalPair(left, right string) string {
	if left > right {
		left, right = right, left
	}
	return left + "\x00" + right
}
