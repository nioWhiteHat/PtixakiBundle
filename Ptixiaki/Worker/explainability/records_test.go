package explainability

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestStreamParserValidatesCompleteProtocol(t *testing.T) {
	t.Parallel()

	var staged []PropertyRecord
	parser, err := NewStreamParser("sell", func(record PropertyRecord) error {
		staged = append(staged, record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var progress []ProgressRecord
	parser.SetProgressHandler(func(record ProgressRecord) { progress = append(progress, record) })
	consumeRecord(t, parser, ProgressRecord{Type: "progress", Stage: "prepare", Total: 1})

	meta := MetaRecord{
		Type:                   "meta",
		SchemaVersion:          1,
		Transaction:            "sell",
		FeatureNames:           ExpectedFeatureNames(),
		ModelFeatureNames:      testModelFeatureNames(),
		FeatureCount:           FeatureCount,
		PropertyCount:          1,
		InteractionCount:       FeatureCount * (FeatureCount - 1) / 2,
		InteractionSampleCount: 1,
		InteractionSampleSeed:  42,
		BaselinePricePerSqm:    100,
		TrainingR2:             0.95,
		ValidationR2:           0.81,
		ModelSHA256:            strings.Repeat("a", 64),
		TrainedAt:              "2026-09-11T10:00:00Z",
	}
	meta.Target.Name = "AskingPricePerSqm"
	meta.Target.Unit = "EUR_per_square_meter"
	consumeRecord(t, parser, meta)
	consumeRecord(t, parser, ProgressRecord{Type: "progress", Stage: "shap_properties", Total: 1})
	consumeRecord(t, parser, PropertyRecord{
		Type:                  "property",
		UID:                   "property_1",
		Contributions:         make([]float64, FeatureCount),
		PredictionPricePerSqm: 100,
		AdditivityError:       0,
	})
	consumeRecord(t, parser, ProgressRecord{Type: "progress", Stage: "shap_properties", Completed: 1, Total: 1, ElapsedSeconds: 0.2})

	for index, feature := range ExpectedFeatureNames() {
		consumeRecord(t, parser, ImportanceRecord{
			Type:        "importance",
			FeatureName: feature,
			MeanAbs:     1,
			MeanSigned:  0,
			Percentage:  100.0 / FeatureCount,
			Rank:        index + 1,
		})
	}

	rank := 1
	features := ExpectedFeatureNames()
	for left := 0; left < len(features); left++ {
		for right := left + 1; right < len(features); right++ {
			consumeRecord(t, parser, InteractionRecord{
				Type:       "interaction",
				Feature1:   features[left],
				Feature2:   features[right],
				MeanAbs:    0,
				MeanSigned: 0,
				Rank:       rank,
			})
			rank++
		}
	}

	records, err := parser.Finalize()
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if records.PropertyCount != 1 || len(staged) != 1 || len(records.Importance) != FeatureCount || len(records.Interactions) != meta.InteractionCount {
		t.Fatalf("unexpected parsed counts: properties=%d staged=%d importance=%d interactions=%d", records.PropertyCount, len(staged), len(records.Importance), len(records.Interactions))
	}
	if len(progress) != 3 || progress[2].Completed != 1 {
		t.Fatalf("unexpected progress callbacks: %+v", progress)
	}
	for _, sampleCount := range []int{0, -1, meta.PropertyCount + 1} {
		invalid := meta
		invalid.InteractionSampleCount = sampleCount
		if _, err := validateMetaRecord(invalid, "sell"); err == nil {
			t.Fatalf("accepted invalid sample count %d", sampleCount)
		}
	}
	meta.InteractionSampleSeed = -1
	if _, err := validateMetaRecord(meta, "sell"); err == nil {
		t.Fatal("accepted negative interaction sample seed")
	}
}

func TestStreamParserRejectsFeatureOrderAndDuplicateProperty(t *testing.T) {
	t.Parallel()

	features := ExpectedFeatureNames()
	features[0], features[1] = features[1], features[0]
	parser, _ := NewStreamParser("rent", nil)
	meta := MetaRecord{
		Type:                   "meta",
		SchemaVersion:          1,
		Transaction:            "rent",
		FeatureNames:           features,
		ModelFeatureNames:      testModelFeatureNames(),
		FeatureCount:           FeatureCount,
		PropertyCount:          1,
		InteractionCount:       FeatureCount * (FeatureCount - 1) / 2,
		InteractionSampleCount: 1,
		InteractionSampleSeed:  42,
		BaselinePricePerSqm:    100,
		TrainingR2:             1,
		ValidationR2:           1,
		ModelSHA256:            strings.Repeat("b", 64),
		TrainedAt:              "2026-09-11T10:00:00Z",
	}
	meta.Target.Name = "AskingPricePerSqm"
	meta.Target.Unit = "EUR_per_square_meter"
	line := marshalRecord(t, meta)
	if _, err := parser.ParseLine(line); err == nil {
		t.Fatal("expected swapped feature order to fail")
	}

	meta.FeatureNames = ExpectedFeatureNames()
	parser, _ = NewStreamParser("rent", nil)
	consumeRecord(t, parser, meta)
	property := PropertyRecord{Type: "property", UID: "duplicate", Contributions: make([]float64, FeatureCount), PredictionPricePerSqm: 100}
	consumeRecord(t, parser, property)
	if _, err := parser.ParseLine(marshalRecord(t, property)); err == nil {
		t.Fatal("expected duplicate property to fail")
	}
}

func consumeRecord(t *testing.T, parser *StreamParser, record any) {
	t.Helper()
	recognized, err := parser.ParseLine(marshalRecord(t, record))
	if err != nil {
		t.Fatalf("parse record: %v", err)
	}
	if !recognized {
		t.Fatal("record was not recognized")
	}
}

func marshalRecord(t *testing.T, record any) string {
	t.Helper()
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return RecordPrefix + string(payload)
}

func testModelFeatureNames() []string {
	names := make([]string, FeatureCount)
	for index := range names {
		names[index] = fmt.Sprintf("f%d", index)
	}
	return names
}
