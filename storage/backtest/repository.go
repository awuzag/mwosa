package backtest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	"github.com/awuzag/mwosa/packages/idgen"
	backtestservice "github.com/awuzag/mwosa/service/backtest"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type repository struct {
	database *storage.Database
}

var _ backtestservice.StrategyRepository = (*repository)(nil)
var _ backtestservice.EvaluationRepository = (*repository)(nil)
var _ backtestservice.BacktestRunRepository = (*repository)(nil)

func NewRepository(database *storage.Database) (backtestservice.StrategyRepository, error) {
	if database == nil {
		return nil, oops.In("backtest_strategy_repository").New("backtest strategy repository database is nil")
	}
	return &repository{database: database}, nil
}

func (r *repository) CreateStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	strategyRow := strategyToRow(in)
	if _, err := tx.NewInsert().Model(&strategyRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create backtest strategy sqlite row")
	}
	versionRow := versionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create backtest strategy version sqlite row")
	}
	if err := tx.Commit(); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit backtest strategy transaction")
	}
	committed = true

	return detailFromRows(&strategyRow, &versionRow)
}

func (r *repository) ListStrategies(ctx context.Context) ([]backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository")
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	var rows []storage.BacktestStrategyRow
	if err := client.NewSelect().
		Model(&rows).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list backtest strategy sqlite rows")
	}

	details := make([]backtestservice.SavedStrategyDetail, 0, len(rows))
	for i := range rows {
		version, err := r.getVersionByID(ctx, rows[i].ActiveVersionID)
		if err != nil {
			return nil, errb.With("strategy_id", rows[i].ID, "version_id", rows[i].ActiveVersionID).Wrapf(err, "load active backtest strategy version")
		}
		detail, err := detailFromRows(&rows[i], &version)
		if err != nil {
			return nil, errb.With("strategy_id", rows[i].ID).Wrap(err)
		}
		details = append(details, detail)
	}
	return details, nil
}

func (r *repository) GetStrategy(ctx context.Context, name string) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}

	var row storage.BacktestStrategyRow
	if err := client.NewSelect().
		Model(&row).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedStrategyDetail{}, errb.Errorf("backtest strategy not found: %s", name)
		}
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "get backtest strategy sqlite row")
	}
	version, err := r.getVersionByID(ctx, row.ActiveVersionID)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.With("strategy_id", row.ID, "version_id", row.ActiveVersionID).Wrapf(err, "load active backtest strategy version")
	}
	return detailFromRows(&row, &version)
}

func (r *repository) AddStrategyVersion(ctx context.Context, name string, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", name, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin update backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var strategyRow storage.BacktestStrategyRow
	if err := tx.NewSelect().
		Model(&strategyRow).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedStrategyDetail{}, errb.Errorf("backtest strategy not found: %s", name)
		}
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "load backtest strategy for version update")
	}
	version.StrategyID = strategyRow.ID
	versionRow := versionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create updated backtest strategy version")
	}
	if _, err := tx.NewUpdate().
		Model((*storage.BacktestStrategyRow)(nil)).
		Set("active_version_id = ?", version.ID).
		Set("updated_at = ?", now).
		Where("id = ?", strategyRow.ID).
		Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "update active backtest strategy version")
	}
	strategyRow.ActiveVersionID = version.ID
	strategyRow.UpdatedAt = now
	if err := tx.Commit(); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit update backtest strategy transaction")
	}
	committed = true

	return detailFromRows(&strategyRow, &versionRow)
}

func (r *repository) UpsertStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin upsert backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var strategyRow storage.BacktestStrategyRow
	err = tx.NewSelect().
		Model(&strategyRow).
		Where("name = ?", in.Name).
		Where("deleted_at IS NULL").
		Scan(ctx)
	switch {
	case err == nil:
		activeVersion, err := r.getVersionByIDTx(ctx, tx, strategyRow.ActiveVersionID)
		if err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.With("active_version_id", strategyRow.ActiveVersionID).Wrapf(err, "load active backtest strategy version")
		}
		version.StrategyID = strategyRow.ID
		version.Version = activeVersion.Version + 1
		versionRow := versionToRow(version)
		if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy version")
		}
		if _, err := tx.NewUpdate().
			Model((*storage.BacktestStrategyRow)(nil)).
			Set("active_version_id = ?", version.ID).
			Set("updated_at = ?", now).
			Where("id = ?", strategyRow.ID).
			Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "update active backtest strategy version")
		}
		strategyRow.ActiveVersionID = version.ID
		strategyRow.UpdatedAt = now
		if err := tx.Commit(); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit upsert backtest strategy transaction")
		}
		committed = true
		return detailFromRows(&strategyRow, &versionRow)
	case errors.Is(err, sql.ErrNoRows):
		version.Version = 1
		version.StrategyID = in.ID
		in.ActiveVersionID = version.ID
		strategyRow = strategyToRow(in)
		versionRow := versionToRow(version)
		if _, err := tx.NewInsert().Model(&strategyRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy sqlite row")
		}
		if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy version sqlite row")
		}
		if err := tx.Commit(); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit upsert backtest strategy transaction")
		}
		committed = true
		return detailFromRows(&strategyRow, &versionRow)
	default:
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "load backtest strategy for upsert")
	}
}

func (r *repository) DeleteStrategy(ctx context.Context, name string, deletedAt time.Time) error {
	errb := oops.In("backtest_strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return errb.Wrap(err)
	}
	result, err := client.NewUpdate().
		Model((*storage.BacktestStrategyRow)(nil)).
		Set("deleted_at = ?", deletedAt).
		Set("updated_at = ?", deletedAt).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return errb.Wrapf(err, "delete backtest strategy sqlite row")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return errb.Wrapf(err, "read delete backtest strategy affected rows")
	}
	if affected == 0 {
		return errb.Errorf("backtest strategy not found: %s", name)
	}
	return nil
}

func (r *repository) SaveRun(ctx context.Context, run backtestservice.SavedBacktestRun, now time.Time) (backtestservice.SavedBacktestRunDetail, error) {
	errb := oops.In("backtest_run_repository").With("run_id", run.ID, "run", run.RunName)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrap(err)
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	row := backtestRunToRow(run)
	if _, err := client.NewInsert().Model(&row).Exec(ctx); err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrapf(err, "create backtest run row")
	}
	return backtestRunDetailFromRow(&row)
}

func (r *repository) ListRuns(ctx context.Context) ([]backtestservice.SavedBacktestRun, error) {
	errb := oops.In("backtest_run_repository")
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	var rows []storage.BacktestRunRow
	if err := client.NewSelect().Model(&rows).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list backtest run rows")
	}
	out := make([]backtestservice.SavedBacktestRun, 0, len(rows))
	for _, row := range rows {
		run := backtestRunFromRow(&row)
		run.ResultJSON = nil
		run.MetricsJSON = nil
		out = append(out, run)
	}
	return out, nil
}

func (r *repository) GetRun(ctx context.Context, ref string) (backtestservice.SavedBacktestRunDetail, error) {
	errb := oops.In("backtest_run_repository").With("ref", ref)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrap(err)
	}
	var row storage.BacktestRunRow
	query := client.NewSelect().Model(&row).Order("created_at DESC").Limit(1)
	switch {
	case looksLikeID(ref):
		query = query.Where("id = ?", ref)
	case strings.HasPrefix(ref, "sha256:"):
		query = query.Where("result_hash = ?", ref)
	default:
		query = query.Where("run_name = ?", ref)
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedBacktestRunDetail{}, errb.Errorf("backtest run not found: %s", ref)
		}
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrapf(err, "get backtest run row")
	}
	return backtestRunDetailFromRow(&row)
}

func (r *repository) SaveEvaluation(ctx context.Context, experiment backtestservice.SavedExperiment, cases []backtestservice.SavedExperimentCase, steps []backtestservice.SavedWalkForwardStep, now time.Time) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("name", experiment.Name, "experiment_id", experiment.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "begin backtest evaluation transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if experiment.CreatedAt.IsZero() {
		experiment.CreatedAt = now
	}
	experimentRow := experimentToRow(experiment)
	if _, err := tx.NewInsert().Model(&experimentRow).Exec(ctx); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "create backtest experiment row")
	}
	for _, item := range cases {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		caseRow := experimentCaseToRow(item)
		if _, err := tx.NewInsert().Model(&caseRow).Exec(ctx); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID).Wrapf(err, "create backtest experiment case row")
		}
		resultID, err := idgen.NewUUIDV7()
		if err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID).Wrapf(err, "generate backtest result id")
		}
		resultRow := storage.BacktestResultRow{
			ID:                       resultID,
			ExperimentCaseID:         item.ID,
			ResultJSON:               string(item.ResultJSON),
			EngineVersion:            item.EngineVersion,
			IndicatorRegistryVersion: item.IndicatorRegistry,
			MetricRegistryVersion:    item.MetricRegistry,
			ResultHash:               item.ResultHash,
			CreatedAt:                item.CreatedAt,
		}
		if _, err := tx.NewInsert().Model(&resultRow).Exec(ctx); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID).Wrapf(err, "create backtest result row")
		}
		var metrics core.Metrics
		if err := json.Unmarshal(item.MetricsJSON, &metrics); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID).Wrapf(err, "decode metric summary json")
		}
		metricIDs := make([]string, 0, len(metrics))
		for metric := range metrics {
			metricIDs = append(metricIDs, metric)
		}
		slices.Sort(metricIDs)
		for _, metric := range metricIDs {
			metricID, err := idgen.NewUUIDV7()
			if err != nil {
				return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID, "metric", metric).Wrapf(err, "generate metric summary id")
			}
			metricRow := storage.BacktestMetricSummaryRow{
				ID:               metricID,
				ExperimentCaseID: item.ID,
				Metric:           metric,
				Value:            metrics[metric],
				CreatedAt:        item.CreatedAt,
			}
			if _, err := tx.NewInsert().Model(&metricRow).Exec(ctx); err != nil {
				return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID, "metric", metric).Wrapf(err, "create metric summary row")
			}
		}
	}
	for _, item := range steps {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		row := walkForwardStepToRow(item)
		if _, err := tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("step", item.StepIndex).Wrapf(err, "create walk-forward step row")
		}
	}
	if err := tx.Commit(); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "commit backtest evaluation transaction")
	}
	committed = true
	return backtestservice.SavedEvaluationDetail{Experiment: experiment, Cases: cases, WalkForward: steps}, nil
}

func (r *repository) ListEvaluations(ctx context.Context) ([]backtestservice.SavedEvaluationSummary, error) {
	errb := oops.In("backtest_evaluation_repository")
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	var rows []storage.BacktestExperimentRow
	if err := client.NewSelect().Model(&rows).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list backtest experiment rows")
	}
	out := make([]backtestservice.SavedEvaluationSummary, 0, len(rows))
	for _, row := range rows {
		detail, err := r.evaluationDetailByID(ctx, row.ID)
		if err != nil {
			return nil, errb.With("experiment_id", row.ID).Wrap(err)
		}
		var best *backtestservice.SavedExperimentCase
		for i := range detail.Cases {
			if detail.Cases[i].Rank == 1 {
				item := detail.Cases[i]
				best = &item
				break
			}
		}
		out = append(out, backtestservice.SavedEvaluationSummary{
			Experiment: detail.Experiment,
			CaseCount:  len(detail.Cases),
			BestCase:   best,
		})
	}
	return out, nil
}

func (r *repository) GetEvaluation(ctx context.Context, ref string) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("ref", ref)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	var row storage.BacktestExperimentRow
	query := client.NewSelect().Model(&row).Order("created_at DESC").Limit(1)
	if looksLikeID(ref) {
		query = query.Where("id = ?", ref)
	} else {
		query = query.Where("name = ?", ref)
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedEvaluationDetail{}, errb.Errorf("backtest evaluation not found: %s", ref)
		}
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "get backtest evaluation row")
	}
	return r.evaluationDetailByID(ctx, row.ID)
}

func (r *repository) evaluationDetailByID(ctx context.Context, id string) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("experiment_id", id)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	var experimentRow storage.BacktestExperimentRow
	if err := client.NewSelect().Model(&experimentRow).Where("id = ?", id).Scan(ctx); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "get backtest experiment row")
	}
	var caseRows []storage.BacktestExperimentCaseRow
	if err := client.NewSelect().Model(&caseRows).Where("experiment_id = ?", id).Order("rank ASC").Order("case_id ASC").Scan(ctx); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "list backtest experiment case rows")
	}
	cases := make([]backtestservice.SavedExperimentCase, 0, len(caseRows))
	for _, row := range caseRows {
		item := experimentCaseFromRow(&row)
		var resultRow storage.BacktestResultRow
		if err := client.NewSelect().Model(&resultRow).Where("experiment_case_id = ?", row.ID).Scan(ctx); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", row.CaseID).Wrapf(err, "get backtest result row")
		}
		item.ResultJSON = json.RawMessage(resultRow.ResultJSON)
		metrics := core.Metrics{}
		var metricRows []storage.BacktestMetricSummaryRow
		if err := client.NewSelect().Model(&metricRows).Where("experiment_case_id = ?", row.ID).Order("metric ASC").Scan(ctx); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", row.CaseID).Wrapf(err, "list metric summary rows")
		}
		for _, metricRow := range metricRows {
			metrics[metricRow.Metric] = metricRow.Value
		}
		metricsJSON, err := json.Marshal(metrics)
		if err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", row.CaseID).Wrapf(err, "encode metric summary json")
		}
		item.MetricsJSON = metricsJSON
		cases = append(cases, item)
	}
	var stepRows []storage.BacktestWalkForwardStepRow
	if err := client.NewSelect().Model(&stepRows).Where("experiment_id = ?", id).Order("step_index ASC").Scan(ctx); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "list walk-forward step rows")
	}
	steps := make([]backtestservice.SavedWalkForwardStep, 0, len(stepRows))
	for _, row := range stepRows {
		steps = append(steps, walkForwardStepFromRow(&row))
	}
	return backtestservice.SavedEvaluationDetail{
		Experiment:  experimentFromRow(&experimentRow),
		Cases:       cases,
		WalkForward: steps,
	}, nil
}

func (r *repository) getVersionByID(ctx context.Context, id string) (storage.BacktestStrategyVersionRow, error) {
	client, err := r.database.Client(ctx)
	if err != nil {
		return storage.BacktestStrategyVersionRow{}, err
	}
	return r.getVersionByIDTx(ctx, client, id)
}

type versionSelector interface {
	NewSelect() *bun.SelectQuery
}

func (r *repository) getVersionByIDTx(ctx context.Context, client versionSelector, id string) (storage.BacktestStrategyVersionRow, error) {
	var row storage.BacktestStrategyVersionRow
	if err := client.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		return storage.BacktestStrategyVersionRow{}, err
	}
	return row, nil
}

func strategyToRow(in backtestservice.SavedStrategy) storage.BacktestStrategyRow {
	return storage.BacktestStrategyRow{
		ID:              in.ID,
		Name:            in.Name,
		ActiveVersionID: in.ActiveVersionID,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		DeletedAt:       in.DeletedAt,
	}
}

func versionToRow(in backtestservice.SavedStrategyVersion) storage.BacktestStrategyVersionRow {
	return storage.BacktestStrategyVersionRow{
		ID:            in.ID,
		StrategyID:    in.StrategyID,
		Version:       in.Version,
		SchemaVersion: in.SchemaVersion,
		SpecJSON:      string(in.SpecJSON),
		SpecHash:      in.SpecHash,
		CreatedAt:     in.CreatedAt,
	}
}

func strategyFromRow(row *storage.BacktestStrategyRow) backtestservice.SavedStrategy {
	return backtestservice.SavedStrategy{
		ID:              row.ID,
		Name:            row.Name,
		ActiveVersionID: row.ActiveVersionID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       row.DeletedAt,
	}
}

func versionFromRow(row *storage.BacktestStrategyVersionRow) backtestservice.SavedStrategyVersion {
	return backtestservice.SavedStrategyVersion{
		ID:            row.ID,
		StrategyID:    row.StrategyID,
		Version:       row.Version,
		SchemaVersion: row.SchemaVersion,
		SpecJSON:      json.RawMessage(row.SpecJSON),
		SpecHash:      row.SpecHash,
		CreatedAt:     row.CreatedAt,
	}
}

func backtestRunToRow(in backtestservice.SavedBacktestRun) storage.BacktestRunRow {
	return storage.BacktestRunRow{
		ID:                       in.ID,
		RunName:                  in.RunName,
		StrategyName:             in.StrategyName,
		Market:                   in.Market,
		Timeframe:                in.Timeframe,
		PeriodFrom:               in.PeriodFrom,
		PeriodTo:                 in.PeriodTo,
		StrategyHash:             in.StrategyHash,
		RunHash:                  in.RunHash,
		EngineVersion:            in.EngineVersion,
		IndicatorRegistryVersion: in.IndicatorRegistry,
		MetricRegistryVersion:    in.MetricRegistry,
		DataFingerprint:          in.DataFingerprint,
		ResultHash:               in.ResultHash,
		ResultJSON:               string(in.ResultJSON),
		MetricsJSON:              string(in.MetricsJSON),
		CreatedAt:                in.CreatedAt,
	}
}

func backtestRunFromRow(row *storage.BacktestRunRow) backtestservice.SavedBacktestRun {
	return backtestservice.SavedBacktestRun{
		ID:                row.ID,
		RunName:           row.RunName,
		StrategyName:      row.StrategyName,
		Market:            row.Market,
		Timeframe:         row.Timeframe,
		PeriodFrom:        row.PeriodFrom,
		PeriodTo:          row.PeriodTo,
		StrategyHash:      row.StrategyHash,
		RunHash:           row.RunHash,
		EngineVersion:     row.EngineVersion,
		IndicatorRegistry: row.IndicatorRegistryVersion,
		MetricRegistry:    row.MetricRegistryVersion,
		DataFingerprint:   row.DataFingerprint,
		ResultHash:        row.ResultHash,
		ResultJSON:        json.RawMessage(row.ResultJSON),
		MetricsJSON:       json.RawMessage(row.MetricsJSON),
		CreatedAt:         row.CreatedAt,
	}
}

func backtestRunDetailFromRow(row *storage.BacktestRunRow) (backtestservice.SavedBacktestRunDetail, error) {
	run := backtestRunFromRow(row)
	var result core.Result
	if err := json.Unmarshal(run.ResultJSON, &result); err != nil {
		return backtestservice.SavedBacktestRunDetail{}, oops.In("backtest_run_repository").With("run_id", row.ID).Wrapf(err, "decode backtest result json")
	}
	return backtestservice.SavedBacktestRunDetail{Run: run, Result: result}, nil
}

func experimentToRow(in backtestservice.SavedExperiment) storage.BacktestExperimentRow {
	return storage.BacktestExperimentRow{
		ID:               in.ID,
		Name:             in.Name,
		StrategyName:     in.StrategyName,
		BaseRunName:      in.BaseRunName,
		SchemaVersion:    in.SchemaVersion,
		SpecJSON:         string(in.SpecJSON),
		SpecHash:         in.SpecHash,
		StrategySpecHash: in.StrategySpecHash,
		DataFrom:         in.DataFrom,
		DataTo:           in.DataTo,
		CreatedAt:        in.CreatedAt,
	}
}

func experimentFromRow(row *storage.BacktestExperimentRow) backtestservice.SavedExperiment {
	return backtestservice.SavedExperiment{
		ID:               row.ID,
		Name:             row.Name,
		StrategyName:     row.StrategyName,
		BaseRunName:      row.BaseRunName,
		SchemaVersion:    row.SchemaVersion,
		SpecJSON:         json.RawMessage(row.SpecJSON),
		SpecHash:         row.SpecHash,
		StrategySpecHash: row.StrategySpecHash,
		DataFrom:         row.DataFrom,
		DataTo:           row.DataTo,
		CreatedAt:        row.CreatedAt,
	}
}

func experimentCaseToRow(in backtestservice.SavedExperimentCase) storage.BacktestExperimentCaseRow {
	return storage.BacktestExperimentCaseRow{
		ID:                       in.ID,
		ExperimentID:             in.ExperimentID,
		CaseID:                   in.CaseID,
		CaseName:                 in.CaseName,
		RunName:                  in.RunName,
		PeriodFrom:               in.PeriodFrom,
		PeriodTo:                 in.PeriodTo,
		ParameterJSON:            string(in.ParameterJSON),
		RegimeTagsJSON:           string(in.RegimeTagsJSON),
		Status:                   in.Status,
		PassedConstraints:        in.PassedConstraints,
		Rank:                     in.Rank,
		Objective:                in.Objective,
		ObjectiveValue:           in.ObjectiveValue,
		StrategyHash:             in.StrategyHash,
		RunHash:                  in.RunHash,
		EngineVersion:            in.EngineVersion,
		IndicatorRegistryVersion: in.IndicatorRegistry,
		MetricRegistryVersion:    in.MetricRegistry,
		DataFingerprint:          in.DataFingerprint,
		ResultHash:               in.ResultHash,
		CreatedAt:                in.CreatedAt,
	}
}

func experimentCaseFromRow(row *storage.BacktestExperimentCaseRow) backtestservice.SavedExperimentCase {
	return backtestservice.SavedExperimentCase{
		ID:                row.ID,
		ExperimentID:      row.ExperimentID,
		CaseID:            row.CaseID,
		CaseName:          row.CaseName,
		RunName:           row.RunName,
		PeriodFrom:        row.PeriodFrom,
		PeriodTo:          row.PeriodTo,
		ParameterJSON:     json.RawMessage(row.ParameterJSON),
		RegimeTagsJSON:    json.RawMessage(row.RegimeTagsJSON),
		Status:            row.Status,
		PassedConstraints: row.PassedConstraints,
		Rank:              row.Rank,
		Objective:         row.Objective,
		ObjectiveValue:    row.ObjectiveValue,
		StrategyHash:      row.StrategyHash,
		RunHash:           row.RunHash,
		EngineVersion:     row.EngineVersion,
		IndicatorRegistry: row.IndicatorRegistryVersion,
		MetricRegistry:    row.MetricRegistryVersion,
		DataFingerprint:   row.DataFingerprint,
		ResultHash:        row.ResultHash,
		CreatedAt:         row.CreatedAt,
	}
}

func walkForwardStepToRow(in backtestservice.SavedWalkForwardStep) storage.BacktestWalkForwardStepRow {
	return storage.BacktestWalkForwardStepRow{
		ID:                       in.ID,
		ExperimentID:             in.ExperimentID,
		StepIndex:                in.StepIndex,
		TrainFrom:                in.TrainFrom,
		TrainTo:                  in.TrainTo,
		TestFrom:                 in.TestFrom,
		TestTo:                   in.TestTo,
		SelectedParameterJSON:    string(in.SelectedParameterJSON),
		TrainCaseID:              in.TrainCaseID,
		TestCaseID:               in.TestCaseID,
		TrainObjective:           in.TrainObjective,
		TestMetricsJSON:          string(in.TestMetricsJSON),
		StrategyHash:             in.StrategyHash,
		RunHash:                  in.RunHash,
		EngineVersion:            in.EngineVersion,
		IndicatorRegistryVersion: in.IndicatorRegistry,
		MetricRegistryVersion:    in.MetricRegistry,
		DataFingerprint:          in.DataFingerprint,
		ResultHash:               in.ResultHash,
		CreatedAt:                in.CreatedAt,
	}
}

func walkForwardStepFromRow(row *storage.BacktestWalkForwardStepRow) backtestservice.SavedWalkForwardStep {
	return backtestservice.SavedWalkForwardStep{
		ID:                    row.ID,
		ExperimentID:          row.ExperimentID,
		StepIndex:             row.StepIndex,
		TrainFrom:             row.TrainFrom,
		TrainTo:               row.TrainTo,
		TestFrom:              row.TestFrom,
		TestTo:                row.TestTo,
		SelectedParameterJSON: json.RawMessage(row.SelectedParameterJSON),
		TrainCaseID:           row.TrainCaseID,
		TestCaseID:            row.TestCaseID,
		TrainObjective:        row.TrainObjective,
		TestMetricsJSON:       json.RawMessage(row.TestMetricsJSON),
		StrategyHash:          row.StrategyHash,
		RunHash:               row.RunHash,
		EngineVersion:         row.EngineVersion,
		IndicatorRegistry:     row.IndicatorRegistryVersion,
		MetricRegistry:        row.MetricRegistryVersion,
		DataFingerprint:       row.DataFingerprint,
		ResultHash:            row.ResultHash,
		CreatedAt:             row.CreatedAt,
	}
}

func looksLikeID(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) != 5 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
				return false
			}
		}
	}
	return true
}

func detailFromRows(strategy *storage.BacktestStrategyRow, version *storage.BacktestStrategyVersionRow) (backtestservice.SavedStrategyDetail, error) {
	detail := backtestservice.SavedStrategyDetail{
		Strategy:      strategyFromRow(strategy),
		ActiveVersion: versionFromRow(version),
	}
	if err := json.Unmarshal(detail.ActiveVersion.SpecJSON, &detail.Spec); err != nil {
		return backtestservice.SavedStrategyDetail{}, oops.In("backtest_strategy_repository").With("strategy_id", strategy.ID, "version_id", version.ID).Wrapf(err, "decode canonical backtest strategy spec")
	}
	if detail.Spec.Kind == "" {
		detail.Spec = core.StrategySpec{}
	}
	return detail, nil
}
