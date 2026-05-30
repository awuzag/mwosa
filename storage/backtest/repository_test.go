package backtest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	backtestservice "github.com/awuzag/mwosa/service/backtest"
	"github.com/awuzag/mwosa/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositorySavedStrategyLifecycle(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	repository, err := NewRepository(database)
	require.NoError(t, err)

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
		SpecHash:      "hash-1",
		CreatedAt:     now,
	}, now)
	require.NoError(t, err)
	assert.Equal(t, "sma-cross", created.Strategy.Name)
	assert.Equal(t, 1, created.ActiveVersion.Version)
	assert.Equal(t, spec.Name, created.Spec.Name)

	listed, err := repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	inspected, err := repository.GetStrategy(ctx, "sma-cross")
	require.NoError(t, err)
	assert.JSONEq(t, string(specJSON), string(inspected.ActiveVersion.SpecJSON))

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
		SpecHash:      "hash-2",
		CreatedAt:     now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 2, updated.ActiveVersion.Version)
	assert.Equal(t, 25.0, updated.Spec.Sizing.Value)

	require.NoError(t, repository.DeleteStrategy(ctx, "sma-cross", now.Add(2*time.Minute)))
	listed, err = repository.ListStrategies(ctx)
	require.NoError(t, err)
	assert.Empty(t, listed)
	_, err = repository.GetStrategy(ctx, "sma-cross")
	require.Error(t, err)
}

func TestRepositorySavedBacktestRunLifecycle(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	repository, err := NewRepository(database)
	require.NoError(t, err)
	runs, ok := repository.(backtestservice.BacktestRunRepository)
	require.True(t, ok)

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

	saved, err := runs.SaveRun(ctx, backtestservice.SavedBacktestRun{
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
	assert.Equal(t, "018f0000-0000-7000-8000-000000000001", saved.Run.ID)
	assert.Equal(t, result.ResultHash, saved.Result.ResultHash)

	listed, err := runs.ListRuns(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].ResultJSON)
	assert.Equal(t, "sha256:strategy", listed[0].StrategyHash)
	assert.Equal(t, core.EngineVersion, listed[0].EngineVersion)
	assert.Equal(t, core.DefaultIndicatorRegistryVersion, listed[0].IndicatorRegistry)
	assert.Equal(t, core.DefaultMetricRegistryVersion, listed[0].MetricRegistry)

	byID, err := runs.GetRun(ctx, "018f0000-0000-7000-8000-000000000001")
	require.NoError(t, err)
	assert.Equal(t, "sma-cross-run", byID.Result.RunName)
	assert.Equal(t, core.EngineVersion, byID.Run.EngineVersion)
	assert.Equal(t, core.EngineVersion, byID.Result.Runtime.EngineVersion)

	byName, err := runs.GetRun(ctx, "sma-cross-run")
	require.NoError(t, err)
	assert.Equal(t, "018f0000-0000-7000-8000-000000000001", byName.Run.ID)

	byHash, err := runs.GetRun(ctx, "sha256:result")
	require.NoError(t, err)
	assert.Equal(t, "018f0000-0000-7000-8000-000000000001", byHash.Run.ID)
}

func TestRepositorySavedEvaluationCaseHashes(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	repository, err := NewRepository(database)
	require.NoError(t, err)
	evaluations, ok := repository.(backtestservice.EvaluationRepository)
	require.True(t, ok)

	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	_, err = evaluations.SaveEvaluation(ctx, backtestservice.SavedExperiment{
		ID:               "experiment-1",
		Name:             "sma-grid",
		StrategyName:     "sma-cross",
		BaseRunName:      "sma-cross-run",
		SchemaVersion:    core.SchemaVersion,
		SpecJSON:         json.RawMessage(`{"kind":"Evaluation"}`),
		SpecHash:         "sha256:spec",
		StrategySpecHash: "sha256:strategy",
		DataFrom:         "2024-01-02",
		DataTo:           "2024-01-08",
		CreatedAt:        now,
	}, []backtestservice.SavedExperimentCase{{
		ID:                "case-row-1",
		ExperimentID:      "experiment-1",
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
		ResultJSON:        json.RawMessage(`{"run_name":"sma-cross-run","result_hash":"sha256:result","data_fingerprint":"sha256:data","metrics":{"total_return":0.1}}`),
		MetricsJSON:       json.RawMessage(`{"total_return":0.1}`),
		StrategyHash:      "sha256:case-strategy",
		RunHash:           "sha256:case-run",
		EngineVersion:     core.EngineVersion,
		IndicatorRegistry: core.DefaultIndicatorRegistryVersion,
		MetricRegistry:    core.DefaultMetricRegistryVersion,
		DataFingerprint:   "sha256:data",
		ResultHash:        "sha256:result",
		CreatedAt:         now,
	}}, nil, now)
	require.NoError(t, err)

	detail, err := evaluations.GetEvaluation(ctx, "sma-grid")
	require.NoError(t, err)
	require.Len(t, detail.Cases, 1)
	assert.Equal(t, "sha256:case-strategy", detail.Cases[0].StrategyHash)
	assert.Equal(t, "sha256:case-run", detail.Cases[0].RunHash)
	assert.Equal(t, core.EngineVersion, detail.Cases[0].EngineVersion)
	assert.Equal(t, core.DefaultIndicatorRegistryVersion, detail.Cases[0].IndicatorRegistry)
	assert.Equal(t, core.DefaultMetricRegistryVersion, detail.Cases[0].MetricRegistry)
	assert.Equal(t, "sha256:data", detail.Cases[0].DataFingerprint)
	assert.Equal(t, "sha256:result", detail.Cases[0].ResultHash)
}

func TestRepositorySavedWalkForwardStepHashes(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	repository, err := NewRepository(database)
	require.NoError(t, err)
	evaluations, ok := repository.(backtestservice.EvaluationRepository)
	require.True(t, ok)

	now := time.Date(2026, 5, 15, 10, 30, 0, 0, time.UTC)
	_, err = evaluations.SaveEvaluation(ctx, backtestservice.SavedExperiment{
		ID:               "experiment-1",
		Name:             "sma-walk-forward",
		StrategyName:     "sma-cross",
		BaseRunName:      "sma-cross-run",
		SchemaVersion:    core.SchemaVersion,
		SpecJSON:         json.RawMessage(`{"kind":"Evaluation"}`),
		SpecHash:         "sha256:spec",
		StrategySpecHash: "sha256:strategy",
		DataFrom:         "2024-01-02",
		DataTo:           "2024-01-08",
		CreatedAt:        now,
	}, nil, []backtestservice.SavedWalkForwardStep{{
		ID:                    "step-row-1",
		ExperimentID:          "experiment-1",
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

	detail, err := evaluations.GetEvaluation(ctx, "sma-walk-forward")
	require.NoError(t, err)
	require.Len(t, detail.WalkForward, 1)
	assert.Equal(t, "sha256:step-strategy", detail.WalkForward[0].StrategyHash)
	assert.Equal(t, "sha256:step-run", detail.WalkForward[0].RunHash)
	assert.Equal(t, core.EngineVersion, detail.WalkForward[0].EngineVersion)
	assert.Equal(t, core.DefaultIndicatorRegistryVersion, detail.WalkForward[0].IndicatorRegistry)
	assert.Equal(t, core.DefaultMetricRegistryVersion, detail.WalkForward[0].MetricRegistry)
	assert.Equal(t, "sha256:step-data", detail.WalkForward[0].DataFingerprint)
	assert.Equal(t, "sha256:step-result", detail.WalkForward[0].ResultHash)
	assert.JSONEq(t, `{"calmar":0.75}`, string(detail.WalkForward[0].TestMetricsJSON))
}

func repositoryTestSpec(sizing float64) core.StrategySpec {
	return core.StrategySpec{
		Kind:          core.KindStrategy,
		SchemaVersion: core.SchemaVersion,
		Name:          "sma-cross",
		Indicators: map[string]core.IndicatorSpec{
			"trend": {
				ID:     "sma",
				Source: core.ValueExpr{Kind: "price", Price: "close"},
				Params: map[string]float64{"window": 2},
			},
		},
		Entry:  core.RuleExpr{Operator: "gt", Args: []core.ValueExpr{{Kind: "price", Price: "close"}, {Kind: "ref", Ref: "trend"}}},
		Exit:   core.RuleExpr{Operator: "lt", Args: []core.ValueExpr{{Kind: "price", Price: "close"}, {Kind: "ref", Ref: "trend"}}},
		Sizing: core.SizingSpec{Type: core.SizingPercentOfEquity, Value: sizing},
	}
}
