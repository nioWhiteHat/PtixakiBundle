package explainability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

const manifestSchemaVersion = 1

type localManifest struct {
	SchemaVersion        int               `json:"schema_version"`
	ModelFile            string            `json:"model_file"`
	ModelSHA256          string            `json:"model_sha256"`
	Transaction          string            `json:"transaction"`
	FeatureCount         int               `json:"feature_count"`
	OriginalFeatureNames []string          `json:"original_feature_names"`
	ModelFeatureNames    []string          `json:"model_feature_names"`
	Target               map[string]string `json:"target"`
}

func nativeArtifactFilenames(transaction string) (string, string, error) {
	if err := ValidateTransaction(transaction); err != nil {
		return "", "", err
	}
	stem := fmt.Sprintf("Phase2_residence_properties_%s", transaction)
	return stem + "_Tree.ubj", stem + "_Manifest.json", nil
}


func LoadLocalModelHashes(directory string, transactions []string) (map[string]string, error) {
	if len(transactions) == 0 {
		return nil, fmt.Errorf("at least one transaction is required")
	}
	hashes := make(map[string]string, len(transactions))
	for _, transaction := range transactions {
		if _, duplicate := hashes[transaction]; duplicate {
			return nil, fmt.Errorf("duplicate transaction %q", transaction)
		}
		modelFile, manifestFile, err := nativeArtifactFilenames(transaction)
		if err != nil {
			return nil, err
		}

		manifestPath := filepath.Join(directory, manifestFile)
		manifestBytes, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read %s manifest: %w", transaction, err)
		}
		var manifest localManifest
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, fmt.Errorf("decode %s manifest: %w", transaction, err)
		}
		if manifest.SchemaVersion != manifestSchemaVersion || manifest.Transaction != transaction {
			return nil, fmt.Errorf("%s manifest identity is invalid", transaction)
		}
		if manifest.ModelFile != modelFile {
			return nil, fmt.Errorf("%s manifest model_file is %q, expected %q", transaction, manifest.ModelFile, modelFile)
		}
		if manifest.FeatureCount != FeatureCount || !slices.Equal(manifest.OriginalFeatureNames, expectedFeatureNames) {
			return nil, fmt.Errorf("%s manifest feature contract is invalid", transaction)
		}
		expectedModelFeatures := make([]string, FeatureCount)
		for index := range expectedModelFeatures {
			expectedModelFeatures[index] = fmt.Sprintf("f%d", index)
		}
		if !slices.Equal(manifest.ModelFeatureNames, expectedModelFeatures) {
			return nil, fmt.Errorf("%s manifest model feature order is invalid", transaction)
		}
		if manifest.Target["name"] != TargetName || manifest.Target["unit"] != TargetUnit {
			return nil, fmt.Errorf("%s manifest target contract is invalid", transaction)
		}
		if err := validateSHA256(manifest.ModelSHA256); err != nil {
			return nil, fmt.Errorf("%s manifest: %w", transaction, err)
		}

		modelPath := filepath.Join(directory, modelFile)
		model, err := os.Open(modelPath)
		if err != nil {
			return nil, fmt.Errorf("open %s native model: %w", transaction, err)
		}
		digest := sha256.New()
		_, copyErr := io.Copy(digest, model)
		closeErr := model.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("hash %s native model: %w", transaction, copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s native model: %w", transaction, closeErr)
		}
		actualHash := hex.EncodeToString(digest.Sum(nil))
		if actualHash != manifest.ModelSHA256 {
			return nil, fmt.Errorf("%s native model hash does not match its manifest", transaction)
		}
		hashes[transaction] = actualHash
	}
	return hashes, nil
}
