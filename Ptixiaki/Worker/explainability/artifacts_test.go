package explainability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocalModelHashesValidatesPublishedPair(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	modelFile, manifestFile, err := nativeArtifactFilenames("sell")
	if err != nil {
		t.Fatal(err)
	}
	modelBytes := []byte("synthetic native model")
	digest := sha256.Sum256(modelBytes)
	modelHash := hex.EncodeToString(digest[:])
	if err := os.WriteFile(filepath.Join(directory, modelFile), modelBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	modelFeatures := make([]string, FeatureCount)
	for index := range modelFeatures {
		modelFeatures[index] = fmt.Sprintf("f%d", index)
	}
	manifest := localManifest{
		SchemaVersion:        manifestSchemaVersion,
		ModelFile:            modelFile,
		ModelSHA256:          modelHash,
		Transaction:          "sell",
		FeatureCount:         FeatureCount,
		OriginalFeatureNames: ExpectedFeatureNames(),
		ModelFeatureNames:    modelFeatures,
		Target: map[string]string{
			"name": TargetName,
			"unit": TargetUnit,
		},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(directory, manifestFile)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	hashes, err := LoadLocalModelHashes(directory, []string{"sell"})
	if err != nil {
		t.Fatalf("load local model hashes: %v", err)
	}
	if hashes["sell"] != modelHash {
		t.Fatalf("hash = %q, want %q", hashes["sell"], modelHash)
	}

	manifest.ModelSHA256 = fmt.Sprintf("%064d", 0)
	manifestBytes, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLocalModelHashes(directory, []string{"sell"}); err == nil {
		t.Fatal("expected a mismatched UBJ hash to fail")
	}
}
