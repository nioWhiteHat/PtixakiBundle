package pipelines

import (
	explain "Ptixiaki/Worker/explainability"
	tr "Ptixiaki/Worker/training"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrainingPipeline struct {
	Db  *pgxpool.Pool
	Ctx context.Context
}

func InitTrainingPipeline(db *pgxpool.Pool, ctx context.Context) TrainingPipeline {
	return TrainingPipeline{
		Db:  db,
		Ctx: ctx,
	}
}
func (T *TrainingPipeline) TrainJob(ItemType string, SubType string, Transaction string) error {

	// 1. Generate Training Data (Phase 1 CSV)
	err := tr.GenerateTrainingData(T.Ctx, T.Db, ItemType, SubType, Transaction)
	if err != nil {
		return err
	}

	// 2. Phase 1 Tree Execution
	Creator := tr.AttrTreeConfigs{}
	Creator.Initialize(ItemType, SubType)
	Phase1Params := Creator.BuildPythonArgs(Transaction, ItemType, SubType, "Attr")
	scriptPath := "Worker/Models/TreeCode/Tree.py"

	err = RunPythonTree(T.Ctx, scriptPath, Phase1Params)
	if err != nil {
		return err
	}

	// 3. Geo Premium Engine Execution
	InputJsonPath := BuildOutputPath(Creator.BasePathOutputs, ItemType, SubType, Transaction, "JsonOutputs", "Attr")
	GeoPremOutputDir := filepath.Join(Creator.BasePathOutputs, "GeoPremiums")

	err = tr.CalcGeoPremium(T.Ctx, T.Db, ItemType, SubType, Transaction, InputJsonPath, GeoPremOutputDir)
	if err != nil {
		return err
	}

	// 4. Merge Geo Premium into Phase 2 CSV
	OriginalCsvPath := BuildCsvPath(Creator.BasePathCsvs, ItemType, SubType, Transaction)
	GeoJsonFileName := fmt.Sprintf("GeoPrem_%s_%s_%s.json", ItemType, SubType, Transaction)
	GeoJsonPath := filepath.Join(GeoPremOutputDir, GeoJsonFileName)
	NewCsvPath := BuildCsvPath(Creator.BasePathCsvs, ItemType, SubType, Transaction+"_Phase2")

	err = tr.MergeGeoPremiumToCSV(OriginalCsvPath, GeoJsonPath, NewCsvPath)
	if err != nil {
		return err
	}

	// 5. Phase 2 Tree Execution
	// Generate arguments but tag it as "Phase2" so Python names the outputs correctly
	Phase2Params := Creator.BuildPythonArgs(Transaction, ItemType, SubType, "Phase2")

	// Dynamically override the --input_csv argument to use our new merged CSV
	for i, param := range Phase2Params {
		if param == "--input_csv" {
			Phase2Params[i+1] = NewCsvPath
			break
		}
	}

	// Residence/property sell and rent are the two online models. Their final
	// phase streams a strict SHAP protocol and is published transactionally.
	Phase1predictionsPath := BuildOutputPath(Creator.BasePathOutputs, ItemType, SubType, Transaction, "JsonOutputs", "Attr")
	Phase2predictionsPath := BuildOutputPath(Creator.BasePathOutputs, ItemType, SubType, Transaction, "JsonOutputs", "Phase2")
	explainableTraining := explain.IsExplainableTraining(ItemType, SubType, Transaction)
	if explainableTraining {
		Phase2Params = append(Phase2Params,
			"--emit_shap_records",
			"--shap_batch_size", strconv.Itoa(explain.TrainingSHAPBatchSize),
			"--interaction_sample_size", strconv.Itoa(explain.TrainingInteractionSampleSize),
			"--interaction_sample_seed", strconv.Itoa(explain.TrainingInteractionSampleSeed),
			"--shap_additivity_atol", strconv.FormatFloat(explain.AdditivityAbsTolerance, 'g', -1, 64),
			"--shap_additivity_rtol", strconv.FormatFloat(explain.AdditivityRelTolerance, 'g', -1, 64),
			"--validation_fraction", strconv.FormatFloat(explain.ValidationFraction, 'g', -1, 64),
			"--target_unit", explain.TargetUnit,
			"--native_model_dir", filepath.Join(Creator.BasePathOutputs, "TreeOutputs"),
		)
		err = explain.RunTrainingAndPersist(
			T.Ctx,
			T.Db,
			scriptPath,
			Phase2Params,
			Transaction,
			func(ctx context.Context) error {
				return InsertToDb(ctx, T.Db, GeoJsonPath, Phase1predictionsPath, Phase2predictionsPath, Transaction)
			},
		)
	} else {
		err = RunPythonTree(T.Ctx, scriptPath, Phase2Params)
	}
	if err != nil {
		return err
	}
	if !explainableTraining {
		if err := InsertToDb(T.Ctx, T.Db, GeoJsonPath, Phase1predictionsPath, Phase2predictionsPath, Transaction); err != nil {
			return err
		}
	}
	return nil
}

func BuildCsvPath(basepath string, ItemType string, SubType string, Transaction string) string {
	fileName := fmt.Sprintf("%s_%s_%s_training_data.csv", ItemType, SubType, Transaction)
	return filepath.Join(basepath, fileName)
}

func BuildOutputPath(basepath string, ItemType string, SubType string, Transaction string, OutputType string, prefix string) string {
	var fileName string

	switch OutputType {
	case "CategoryMappings":
		fileName = fmt.Sprintf("CategoryMappings_%s_%s_%s_%s.txt", prefix, ItemType, SubType, Transaction)
	case "JsonOutputs":
		fileName = fmt.Sprintf("%s_%s_%s_%s_PricePredictions.json", prefix, ItemType, SubType, Transaction)
	case "TreeOutputs":
		fileName = fmt.Sprintf("%s_%s_%s_%s_Tree.onnx", prefix, ItemType, SubType, Transaction)
	}

	return filepath.Join(basepath, OutputType, fileName)
}

func RunPythonTree(ctx context.Context, scriptPath string, args []string) error {
	// Combine the script path with the generated arguments
	cmdArgs := append([]string{scriptPath}, args...)

	// Initialize the command to run 'python' (Use 'python3' if on Linux/Mac)
	runCtx, cancel := context.WithTimeout(ctx, explain.TrainingProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "python", cmdArgs...)

	// Bind Python's output directly to Go's console so you can see prints/errors
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("\n🚀 Launching Python Training: %s\n", scriptPath)

	// Execute and wait for completion
	err := cmd.Run()
	if err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("python script timed out or was cancelled: %w", runCtx.Err())
		}
		return fmt.Errorf("python script crashed: %v", err)
	}

	fmt.Println("✅ Python execution finished successfully.")
	return nil
}
