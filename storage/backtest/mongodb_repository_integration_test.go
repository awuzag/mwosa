//go:build integration

package backtest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	core "github.com/awuzag/mwosa/packages/backtest"
	backtestservice "github.com/awuzag/mwosa/service/backtest"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoBacktestRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_backtest_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	sqliteDatabase := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, sqliteDatabase.Close())
	})
	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertBacktestStrategyRepositoryContract(t, sqliteRepository)
	assertBacktestStrategyRepositoryContract(t, mongoRepository)
	assertBacktestRunRepositoryContract(t, requireBacktestRunRepository(t, sqliteRepository))
	assertBacktestRunRepositoryContract(t, requireBacktestRunRepository(t, mongoRepository))
	assertBacktestEvaluationRepositoryContract(t, requireBacktestEvaluationRepository(t, sqliteRepository))
	assertBacktestEvaluationRepositoryContract(t, requireBacktestEvaluationRepository(t, mongoRepository))
	assertMongoBacktestDocumentShape(t, runtime)
}

func requireBacktestRunRepository(t *testing.T, repository backtestservice.StrategyRepository) backtestservice.BacktestRunRepository {
	t.Helper()

	runs, ok := repository.(backtestservice.BacktestRunRepository)
	require.True(t, ok)
	return runs
}

func requireBacktestEvaluationRepository(t *testing.T, repository backtestservice.StrategyRepository) backtestservice.EvaluationRepository {
	t.Helper()

	evaluations, ok := repository.(backtestservice.EvaluationRepository)
	require.True(t, ok)
	return evaluations
}

func assertBacktestStrategyRepositoryContract(t *testing.T, repository backtestservice.StrategyRepository) {
	t.Helper()

	ctx := context.Background()
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	spec := repositoryTestSpec(50)
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)
	created, err := repository.UpsertStrategyWithVersion(ctx, backtestservice.SavedStrategy{
		ID:              "strategy-1",
		Name:            "sma-cross",
		ActiveVersionID: "version-1",
		CreatedAt:       now,
		UpdatedAt:       now,
	}, backtestservice.SavedStrategyVersion{
		ID:            "version-1",
		StrategyID:    "strategy-1",
		Version:       1,
		SchemaVersion: core.SchemaVersion,
		SpecJSON:      specJSON,
		SpecHash:      "sha256:strategy-v1",
		CreatedAt:     now,
	}, now)
	require.NoError(t, err)
	require.Equal(t, "sma-cross", created.Strategy.Name)
	require.Equal(t, 1, created.ActiveVersion.Version)
	require.Equal(t, spec.Name, created.Spec.Name)

	updatedSpec := repositoryTestSpec(25)
	updatedJSON, err := json.Marshal(updatedSpec)
	require.NoError(t, err)
	updated, err := repository.UpsertStrategyWithVersion(ctx, backtestservice.SavedStrategy{
		ID:        "strategy-unused",
		Name:      "sma-cross",
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}, backtestservice.SavedStrategyVersion{
		ID:            "version-2",
		SchemaVersion: core.SchemaVersion,
		SpecJSON:      updatedJSON,
		SpecHash:      "sha256:strategy-v2",
		CreatedAt:     now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "strategy-1", updated.ActiveVersion.StrategyID)
	require.Equal(t, 2, updated.ActiveVersion.Version)
	require.Equal(t, 25.0, updated.Spec.Sizing.Value)

	listed, err := repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "version-2", listed[0].ActiveVersion.ID)

	inspected, err := repository.GetStrategy(ctx, "sma-cross")
	require.NoError(t, err)
	require.JSONEq(t, string(updatedJSON), string(inspected.ActiveVersion.SpecJSON))

	require.NoError(t, repository.DeleteStrategy(ctx, "sma-cross", now.Add(2*time.Minute)))
	listed, err = repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Empty(t, listed)
	_, err = repository.GetStrategy(ctx, "sma-cross")
	require.Error(t, err)
}

func assertBacktestRunRepositoryContract(t *testing.T, repository backtestservice.BacktestRunRepository) {
	t.Helper()

	ctx := context.Background()
	now := time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC)
	result := core.Result{
		RunName:      "sma-cross-run",
		StrategyName: "sma-cross",
		Symbols:      []string{"069500"},
		Period:       core.Period{From: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), To: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)},
		Market:       "krx",
		Timeframe:    "1d",
		Runtime: core.RuntimeMetadata{
			EngineVersion:            core.EngineVersion,
			IndicatorRegistryVersion: core.DefaultIndicatorRegistryVersion,
			MetricRegistryVersion:    core.DefaultMetricRegistryVersion,
		},
		Currency:        "KRW",
		Metrics:         core.Metrics{core.MetricTotalReturn: 0.12},
		DataFingerprint: "sha256:data",
		ResultHash:      "sha256:result",
	}
	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	metricsJSON, err := json.Marshal(result.Metrics)
	require.NoError(t, err)

	saved, err := repository.SaveRun(ctx, backtestservice.SavedBacktestRun{
		ID:                "018f0000-0000-7000-8000-000000000001",
		RunName:           result.RunName,
		StrategyName:      result.StrategyName,
		Market:            result.Market,
		Timeframe:         result.Timeframe,
		PeriodFrom:        "2024-01-02",
		PeriodTo:          "2024-01-08",
		StrategyHash:      "sha256:strategy",
		RunHash:           "sha256:run",
		EngineVersion:     result.Runtime.EngineVersion,
		IndicatorRegistry: result.Runtime.IndicatorRegistryVersion,
		MetricRegistry:    result.Runtime.MetricRegistryVersion,
		DataFingerprint:   result.DataFingerprint,
		ResultHash:        result.ResultHash,
		ResultJSON:        resultJSON,
		MetricsJSON:       metricsJSON,
		CreatedAt:         now,
	}, now)
	require.NoError(t, err)
	require.Equal(t, "018f0000-0000-7000-8000-000000000001", saved.Run.ID)
	require.Equal(t, result.ResultHash, saved.Result.ResultHash)

	listed, err := repository.ListRuns(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Empty(t, listed[0].ResultJSON)
	require.Empty(t, listed[0].MetricsJSON)
	require.Equal(t, "sha256:strategy", listed[0].StrategyHash)

	byID, err := repository.GetRun(ctx, "018f0000-0000-7000-8000-000000000001")
	require.NoError(t, err)
	require.Equal(t, "sma-cross-run", byID.Result.RunName)
	require.Equal(t, core.EngineVersion, byID.Run.EngineVersion)

	byName, err := repository.GetRun(ctx, "sma-cross-run")
	require.NoError(t, err)
	require.Equal(t, "018f0000-0000-7000-8000-000000000001", byName.Run.ID)

	byHash, err := repository.GetRun(ctx, "sha256:result")
	require.NoError(t, err)
	require.Equal(t, "018f0000-0000-7000-8000-000000000001", byHash.Run.ID)
}

func assertBacktestEvaluationRepositoryContract(t *testing.T, repository backtestservice.EvaluationRepository) {
	t.Helper()

	ctx := context.Background()
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	experimentID := "018f0000-0000-7000-8000-000000000002"
	saved, err := repository.SaveEvaluation(ctx, backtestservice.SavedExperiment{
		ID:               experimentID,
		Name:             "sma-grid",
		StrategyName:     "sma-cross",
		BaseRunName:      "sma-cross-run",
		SchemaVersion:    core.SchemaVersion,
		SpecJSON:         json.RawMessage(`{"kind":"Evaluation","grid":{"sizing":[25,50]}}`),
		SpecHash:         "sha256:evaluation-spec",
		StrategySpecHash: "sha256:strategy",
		DataFrom:         "2024-01-02",
		DataTo:           "2024-01-08",
		CreatedAt:        now,
	}, []backtestservice.SavedExperimentCase{{
		ID:                "case-row-1",
		ExperimentID:      experimentID,
		CaseID:            "case-001",
		CaseName:          "sma-grid-case",
		RunName:           "sma-cross-run",
		PeriodFrom:        "2024-01-02",
		PeriodTo:          "2024-01-08",
		ParameterJSON:     json.RawMessage(`{"sizing.value":50}`),
		RegimeTagsJSON:    json.RawMessage(`["sideways"]`),
		Status:            "passed",
		PassedConstraints: true,
		Rank:              1,
		Objective:         core.MetricCalmar,
		ObjectiveValue:    1.5,
		ResultJSON:        json.RawMessage(`{"run_name":"sma-cross-run","result_hash":"sha256:evaluation-result","data_fingerprint":"sha256:data","metrics":{"total_return":0.1}}`),
		MetricsJSON:       json.RawMessage(`{"calmar":1.5,"total_return":0.1}`),
		StrategyHash:      "sha256:case-strategy",
		RunHash:           "sha256:case-run",
		EngineVersion:     core.EngineVersion,
		IndicatorRegistry: core.DefaultIndicatorRegistryVersion,
		MetricRegistry:    core.DefaultMetricRegistryVersion,
		DataFingerprint:   "sha256:data",
		ResultHash:        "sha256:evaluation-result",
		CreatedAt:         now,
	}}, []backtestservice.SavedWalkForwardStep{{
		ID:                    "step-row-1",
		ExperimentID:          experimentID,
		StepIndex:             1,
		TrainFrom:             "2024-01-02",
		TrainTo:               "2024-01-03",
		TestFrom:              "2024-01-04",
		TestTo:                "2024-01-04",
		SelectedParameterJSON: json.RawMessage(`{"sizing.value":50}`),
		TrainCaseID:           "wf-1-train-1",
		TestCaseID:            "wf-1-test-1",
		TrainObjective:        1.25,
		TestMetricsJSON:       json.RawMessage(`{"calmar":0.75}`),
		StrategyHash:          "sha256:step-strategy",
		RunHash:               "sha256:step-run",
		EngineVersion:         core.EngineVersion,
		IndicatorRegistry:     core.DefaultIndicatorRegistryVersion,
		MetricRegistry:        core.DefaultMetricRegistryVersion,
		DataFingerprint:       "sha256:step-data",
		ResultHash:            "sha256:step-result",
		CreatedAt:             now,
	}}, now)
	require.NoError(t, err)
	require.Equal(t, "sma-grid", saved.Experiment.Name)
	require.Len(t, saved.Cases, 1)
	require.Len(t, saved.WalkForward, 1)

	summaries, err := repository.ListEvaluations(ctx)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, 1, summaries[0].CaseCount)
	require.NotNil(t, summaries[0].BestCase)
	require.Equal(t, "case-001", summaries[0].BestCase.CaseID)

	byName, err := repository.GetEvaluation(ctx, "sma-grid")
	require.NoError(t, err)
	require.Equal(t, experimentID, byName.Experiment.ID)
	require.JSONEq(t, `{"kind":"Evaluation","grid":{"sizing":[25,50]}}`, string(byName.Experiment.SpecJSON))
	require.Len(t, byName.Cases, 1)
	require.Equal(t, "sha256:case-strategy", byName.Cases[0].StrategyHash)
	require.JSONEq(t, `{"run_name":"sma-cross-run","result_hash":"sha256:evaluation-result","data_fingerprint":"sha256:data","metrics":{"total_return":0.1}}`, string(byName.Cases[0].ResultJSON))
	require.JSONEq(t, `{"calmar":1.5,"total_return":0.1}`, string(byName.Cases[0].MetricsJSON))
	require.Len(t, byName.WalkForward, 1)
	require.Equal(t, "sha256:step-strategy", byName.WalkForward[0].StrategyHash)
	require.JSONEq(t, `{"calmar":0.75}`, string(byName.WalkForward[0].TestMetricsJSON))

	byID, err := repository.GetEvaluation(ctx, experimentID)
	require.NoError(t, err)
	require.Equal(t, "sma-grid", byID.Experiment.Name)
}

func assertMongoBacktestDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var strategy struct {
		ID            string    `bson:"_id"`
		SchemaVersion string    `bson:"schema_version"`
		Revision      int64     `bson:"revision"`
		CreatedAt     time.Time `bson:"created_at"`
		UpdatedAt     time.Time `bson:"updated_at"`
		DeletedAt     time.Time `bson:"deleted_at"`
		Versions      []bson.M  `bson:"versions"`
	}
	require.NoError(t, runtime.Database().
		Collection("backtest_strategies").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "backtest_strategies:strategy-1"}}).
		Decode(&strategy))
	require.Equal(t, "1.0.0", strategy.SchemaVersion)
	require.GreaterOrEqual(t, strategy.Revision, int64(2))
	require.Len(t, strategy.Versions, 2)
	require.Equal(t, "version-2", strategy.Versions[1]["id"])
	require.False(t, strategy.CreatedAt.IsZero())
	require.False(t, strategy.UpdatedAt.IsZero())
	require.False(t, strategy.DeletedAt.IsZero())

	var run struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Result        bson.M `bson:"result"`
		Metrics       bson.M `bson:"metrics"`
	}
	require.NoError(t, runtime.Database().
		Collection("backtest_runs").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "backtest_runs:018f0000-0000-7000-8000-000000000001"}}).
		Decode(&run))
	require.Equal(t, "1.0.0", run.SchemaVersion)
	require.Equal(t, "sma-cross-run", run.Result["run_name"])
	require.Equal(t, 0.12, run.Metrics[string(core.MetricTotalReturn)])

	var experiment struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Spec          bson.M   `bson:"spec"`
		WalkForward   []bson.M `bson:"walk_forward_steps"`
	}
	require.NoError(t, runtime.Database().
		Collection("backtest_experiments").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "backtest_experiments:018f0000-0000-7000-8000-000000000002"}}).
		Decode(&experiment))
	require.Equal(t, "1.0.0", experiment.SchemaVersion)
	require.Equal(t, "Evaluation", experiment.Spec["kind"])
	require.Len(t, experiment.WalkForward, 1)

	var evaluationCase struct {
		ID      string `bson:"_id"`
		Result  bson.M `bson:"result"`
		Metrics bson.M `bson:"metrics"`
	}
	require.NoError(t, runtime.Database().
		Collection("backtest_experiment_cases").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "backtest_experiment_cases:018f0000-0000-7000-8000-000000000002:case-001"}}).
		Decode(&evaluationCase))
	require.Equal(t, "sma-cross-run", evaluationCase.Result["run_name"])
	require.Equal(t, 1.5, evaluationCase.Metrics[string(core.MetricCalmar)])
}
