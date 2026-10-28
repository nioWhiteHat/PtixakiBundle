package explainability

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type propertyStager struct {
	ctx   context.Context
	tx    pgx.Tx
	rows  [][]any
	count int
}

func (s *propertyStager) add(record PropertyRecord) error {
	s.rows = append(s.rows, []any{
		record.UID,
		record.Contributions,
		record.PredictionPricePerSqm,
		record.AdditivityError,
	})
	if len(s.rows) >= PropertyStageBatchSize {
		return s.flush()
	}
	return nil
}

func (s *propertyStager) flush() error {
	if len(s.rows) == 0 {
		return nil
	}
	count, err := s.tx.CopyFrom(
		s.ctx,
		pgx.Identifier{"shap_property_stage"},
		[]string{"property_uid", "contributions", "prediction_price_per_sqm", "additivity_error"},
		pgx.CopyFromRows(s.rows),
	)
	if err != nil {
		return err
	}
	if count != int64(len(s.rows)) {
		return fmt.Errorf("staged %d property rows, expected %d", count, len(s.rows))
	}
	s.count += int(count)
	s.rows = s.rows[:0]
	return nil
}

// RunTrainingAndPersist runs the strict line-oriented SHAP protocol. All rows
// are first written to transaction-local staging tables; production tables are
// only replaced after the Python process exits successfully and every record
// and declared count validates.
func RunTrainingAndPersist(
	ctx context.Context,
	db *pgxpool.Pool,
	scriptPath string,
	args []string,
	transaction string,
	beforePublish func(context.Context) error,
) error {
	tables, err := TablesForTransaction(transaction)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithTimeout(ctx, TrainingProcessTimeout)
	defer cancel()
	started := time.Now()
	log.Printf("[TRAIN %s] Checking explanation schema and preparing database staging", transaction)

	tx, err := db.Begin(runCtx)
	if err != nil {
		return fmt.Errorf("begin explanation publication transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if err := preflightPersistenceSchema(runCtx, tx, tables); err != nil {
		return err
	}
	if err := createStagingTables(runCtx, tx); err != nil {
		return err
	}

	stager := &propertyStager{ctx: runCtx, tx: tx, rows: make([][]any, 0, PropertyStageBatchSize)}
	parser, err := NewStreamParser(transaction, stager.add)
	if err != nil {
		return err
	}
	progress := &progressLogger{transaction: transaction, printf: log.Printf}
	parser.SetProgressHandler(progress.report)

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.CommandContext(runCtx, "python", cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	
	if err != nil {
		return fmt.Errorf("open Python stdout: %w", err)
	}
	cmd.Stderr = os.Stderr

	log.Printf("[TRAIN %s] Launching Python training with explanations: %s", transaction, scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Python training: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), ScannerMaxTokenBytes)
	var protocolErr error
	metaLogged := false
	for scanner.Scan() {
		line := scanner.Text()
		recognized, parseErr := parser.ParseLine(line)
		if parseErr != nil {
			protocolErr = parseErr
			break
		}
		if !recognized && strings.TrimSpace(line) != "" {
			protocolErr = fmt.Errorf("unexpected non-protocol Python stdout: %q", line)
			break
		}
		if parser.meta != nil && !metaLogged {
			log.Printf("[TRAIN %s] Explaining all %d properties; global interactions use %d properties (seed %d)",
				transaction, parser.meta.PropertyCount, parser.meta.InteractionSampleCount, parser.meta.InteractionSampleSeed)
			metaLogged = true
		}
	}
	if protocolErr == nil {
		protocolErr = scanner.Err()
	}
	if protocolErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if protocolErr != nil {
		return fmt.Errorf("invalid Python explanation stream: %w", protocolErr)
	}
	if waitErr != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("Python explanation training timed out after %s", TrainingProcessTimeout)
		}
		return fmt.Errorf("Python training process failed: %w", waitErr)
	}

	records, err := parser.Finalize()
	if err != nil {
		return fmt.Errorf("validate Python explanation stream: %w", err)
	}
	log.Printf("[TRAIN %s] Python finished; validated %d property, %d importance and %d interaction records. Finishing database staging", transaction, records.PropertyCount, len(records.Importance), len(records.Interactions))
	if err := stager.flush(); err != nil {
		return fmt.Errorf("flush property explanation staging: %w", err)
	}
	if stager.count != records.PropertyCount {
		return fmt.Errorf("staged property count mismatch: got %d, expected %d", stager.count, records.PropertyCount)
	}
	if err := stageAggregateRecords(runCtx, tx, records); err != nil {
		return err
	}
	if beforePublish != nil {
		log.Printf("[TRAIN %s] Persisting predictions, global statistics and Elo", transaction)
		if err := beforePublish(runCtx); err != nil {
			return fmt.Errorf("complete dependent training persistence before explanation publication: %w", err)
		}
	}
	log.Printf("[TRAIN %s] Publishing explanation tables and model statistics", transaction)
	if err := publishStagedRecords(runCtx, tx, transaction, records); err != nil {
		return err
	}
	if err := tx.Commit(runCtx); err != nil {
		return fmt.Errorf("commit explanation publication: %w", err)
	}

	log.Printf("[TRAIN %s] Explanation publication committed successfully (elapsed %s)", transaction, time.Since(started).Round(time.Second))
	return nil
}

func preflightPersistenceSchema(ctx context.Context, tx pgx.Tx, tables Tables) error {
	impactColumns := PropertyImpactColumns()
	quotedImpactColumns := make([]string, len(impactColumns))
	for i, column := range impactColumns {
		quotedImpactColumns[i] = pgx.Identifier{column}.Sanitize()
	}
	checks := []string{
		fmt.Sprintf(
			`SELECT property_uid, %s, baseline_price_per_sqm,
				explained_prediction_per_sqm, additivity_error, model_sha256, explained_at
			 FROM %s LIMIT 0`,
			strings.Join(quotedImpactColumns, ", "),
			pgx.Identifier{tables.PropertyAnalysis}.Sanitize(),
		),
		fmt.Sprintf(
			`SELECT feature_name, mean_abs_shap_per_sqm, mean_signed_shap_per_sqm,
				importance_percentage, importance_rank, calculated_at, model_sha256
			 FROM %s LIMIT 0`,
			pgx.Identifier{tables.FeatureImportance}.Sanitize(),
		),
		fmt.Sprintf(
			`SELECT feature_1, feature_2, mean_signed_interaction_per_sqm,
				mean_abs_interaction_per_sqm, importance_rank, calculated_at, model_sha256
			 FROM %s LIMIT 0`,
			pgx.Identifier{tables.FeatureInteractions}.Sanitize(),
		),
		`SELECT stat_name, mean, std, baseline_price_per_sqm, training_r2,
			validation_r2, model_sha256, model_trained_at
		 FROM global_stats LIMIT 0`,
	}
	for _, query := range checks {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("explainability persistence schema preflight failed (apply migration 002): %w", err)
		}
		rows.Close()
	}
	return nil
}

func createStagingTables(ctx context.Context, tx pgx.Tx) error {
	statements := []string{
		`CREATE TEMP TABLE shap_property_stage (
			property_uid text PRIMARY KEY,
			contributions double precision[] NOT NULL,
			prediction_price_per_sqm double precision NOT NULL,
			additivity_error double precision NOT NULL
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE shap_importance_stage (
			feature_name text PRIMARY KEY,
			mean_abs double precision NOT NULL,
			mean_signed double precision NOT NULL,
			percentage double precision NOT NULL,
			importance_rank integer NOT NULL UNIQUE
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE shap_interaction_stage (
			feature_1 text NOT NULL,
			feature_2 text NOT NULL,
			mean_abs double precision NOT NULL,
			mean_signed double precision NOT NULL,
			importance_rank integer NOT NULL UNIQUE,
			PRIMARY KEY (feature_1, feature_2)
		) ON COMMIT DROP`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("create explanation staging table: %w", err)
		}
	}
	return nil
}

func stageAggregateRecords(ctx context.Context, tx pgx.Tx, records TrainingRecords) error {
	importanceRows := make([][]any, 0, len(records.Importance))
	for _, record := range records.Importance {
		importanceRows = append(importanceRows, []any{record.FeatureName, record.MeanAbs, record.MeanSigned, record.Percentage, record.Rank})
	}
	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"shap_importance_stage"},
		[]string{"feature_name", "mean_abs", "mean_signed", "percentage", "importance_rank"},
		pgx.CopyFromRows(importanceRows),
	)
	if err != nil {
		return fmt.Errorf("stage feature importance: %w", err)
	}
	if count != int64(len(importanceRows)) {
		return fmt.Errorf("staged %d importance rows, expected %d", count, len(importanceRows))
	}

	interactionRows := make([][]any, 0, len(records.Interactions))
	for _, record := range records.Interactions {
		interactionRows = append(interactionRows, []any{record.Feature1, record.Feature2, record.MeanAbs, record.MeanSigned, record.Rank})
	}
	count, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"shap_interaction_stage"},
		[]string{"feature_1", "feature_2", "mean_abs", "mean_signed", "importance_rank"},
		pgx.CopyFromRows(interactionRows),
	)
	if err != nil {
		return fmt.Errorf("stage feature interactions: %w", err)
	}
	if count != int64(len(interactionRows)) {
		return fmt.Errorf("staged %d interaction rows, expected %d", count, len(interactionRows))
	}
	return nil
}

func publishStagedRecords(ctx context.Context, tx pgx.Tx, transaction string, records TrainingRecords) error {
	tables, err := TablesForTransaction(transaction)
	if err != nil {
		return err
	}
	propertyTable := pgx.Identifier{tables.PropertyAnalysis}.Sanitize()
	importanceTable := pgx.Identifier{tables.FeatureImportance}.Sanitize()
	interactionTable := pgx.Identifier{tables.FeatureInteractions}.Sanitize()

	if _, err := tx.Exec(ctx, "DELETE FROM "+propertyTable); err != nil {
		return fmt.Errorf("clear old property explanations: %w", err)
	}
	impactColumns := PropertyImpactColumns()
	quotedImpactColumns := make([]string, len(impactColumns))
	contributionExpressions := make([]string, len(impactColumns))
	for i, column := range impactColumns {
		quotedImpactColumns[i] = pgx.Identifier{column}.Sanitize()
		contributionExpressions[i] = fmt.Sprintf("contributions[%d]", i+1)
	}
	propertyInsert := fmt.Sprintf(`
		INSERT INTO %s (
			property_uid, %s, baseline_price_per_sqm,
			explained_prediction_per_sqm, additivity_error, model_sha256, explained_at
		)
		SELECT property_uid, %s, $1, prediction_price_per_sqm,
			additivity_error, $2, $3
		FROM shap_property_stage`,
		propertyTable,
		strings.Join(quotedImpactColumns, ", "),
		strings.Join(contributionExpressions, ", "),
	)
	if _, err := tx.Exec(ctx, propertyInsert, records.Meta.BaselinePricePerSqm, records.Meta.ModelSHA256, records.TrainedAt); err != nil {
		return fmt.Errorf("publish property explanations: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM "+importanceTable); err != nil {
		return fmt.Errorf("clear old feature importance: %w", err)
	}
	importanceInsert := fmt.Sprintf(`
		INSERT INTO %s (
			feature_name, mean_abs_shap_per_sqm, mean_signed_shap_per_sqm,
			importance_percentage, importance_rank, calculated_at, model_sha256
		)
		SELECT feature_name, mean_abs, mean_signed, percentage,
			importance_rank, $1, $2
		FROM shap_importance_stage`, importanceTable)
	if _, err := tx.Exec(ctx, importanceInsert, records.TrainedAt, records.Meta.ModelSHA256); err != nil {
		return fmt.Errorf("publish feature importance: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM "+interactionTable); err != nil {
		return fmt.Errorf("clear old feature interactions: %w", err)
	}
	interactionInsert := fmt.Sprintf(`
		INSERT INTO %s (
			feature_1, feature_2, mean_signed_interaction_per_sqm,
			mean_abs_interaction_per_sqm, importance_rank, calculated_at, model_sha256
		)
		SELECT feature_1, feature_2, mean_signed, mean_abs,
			importance_rank, $1, $2
		FROM shap_interaction_stage`, interactionTable)
	if _, err := tx.Exec(ctx, interactionInsert, records.TrainedAt, records.Meta.ModelSHA256); err != nil {
		return fmt.Errorf("publish feature interactions: %w", err)
	}

	// mean/std are legacy NOT NULL fields. A normal training run has already
	// created the row; the zero values are only a safe insert fallback. On
	// conflict, this deliberately updates only model metadata.
	const statsUpsert = `
		INSERT INTO global_stats (
			stat_name, mean, std, baseline_price_per_sqm, training_r2,
			validation_r2, model_sha256, model_trained_at
		) VALUES ($1, 0, 0, $2, $3, $4, $5, $6)
		ON CONFLICT (stat_name) DO UPDATE SET
			baseline_price_per_sqm = EXCLUDED.baseline_price_per_sqm,
			training_r2 = EXCLUDED.training_r2,
			validation_r2 = EXCLUDED.validation_r2,
			model_sha256 = EXCLUDED.model_sha256,
			model_trained_at = EXCLUDED.model_trained_at`
	if _, err := tx.Exec(
		ctx,
		statsUpsert,
		tables.StatsName,
		records.Meta.BaselinePricePerSqm,
		records.Meta.TrainingR2,
		records.Meta.ValidationR2,
		records.Meta.ModelSHA256,
		records.TrainedAt,
	); err != nil {
		return fmt.Errorf("upsert global model statistics: %w", err)
	}
	return nil
}
