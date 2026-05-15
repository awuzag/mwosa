package backtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileEvaluationBuildsYearlyParameterGridCases(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-grid",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodYearly, From: "2023-01-01", To: "2024-12-31"},
		Parameters: map[string][]any{
			"indicators.trend.params.window": {2, 3},
			"risk.max_positions":             {1, 2},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
		Ranking: EvaluationRankingSpec{Objective: MetricCalmar, Order: RankingOrderDesc},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 8)
	assert.Equal(t, "sma-grid-2023-p1", plan.Cases[0].Name)
	assert.Equal(t, "2023-01-01", plan.Cases[0].Run.Data.From)
	assert.Equal(t, "2023-12-31", plan.Cases[0].Run.Data.To)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricCalmar)
	assert.Equal(t, 2.0, plan.Cases[0].Strategy.Indicators["trend"].Params["window"])
	assert.Equal(t, 1, plan.Cases[0].Strategy.Risk.MaxPositions)
}

func TestEvaluationConstraintsAndRanking(t *testing.T) {
	passing := EvaluationCaseResult{
		CaseID:            "case-1",
		Metrics:           Metrics{MetricCalmar: 1.5, MetricMaxDrawdown: -0.2, MetricCAGR: 0.1, MetricTurnover: 2, MetricTradeCount: 10, MetricExposure: 0.8, MetricUnfilledCount: 1, MetricDataIssueCount: 0},
		PassedConstraints: true,
	}
	failingConstraints := EvaluateConstraints(passing.Metrics, EvaluationConstraintSet{
		MaxDrawdownLTE:       floatPtr(0.1),
		MinCAGRGTE:           floatPtr(0.08),
		MaxExposureLTE:       floatPtr(0.5),
		MaxUnfilledCountLTE:  floatPtr(0),
		MaxDataIssueCountLTE: floatPtr(-1),
	})
	assert.False(t, ConstraintsPassed(failingConstraints))
	assertConstraintResult(t, failingConstraints, "max_exposure_lte", false, 0.8, 0.5)
	assertConstraintResult(t, failingConstraints, "max_unfilled_count_lte", false, 1, 0)
	assertConstraintResult(t, failingConstraints, "max_data_issue_count_lte", false, 0, -1)

	passing.ConstraintResults = EvaluateConstraints(passing.Metrics, EvaluationConstraintSet{
		MaxDrawdownLTE:       floatPtr(0.25),
		MaxExposureLTE:       floatPtr(0.9),
		MaxUnfilledCountLTE:  floatPtr(1),
		MaxDataIssueCountLTE: floatPtr(0),
	})
	passing.PassedConstraints = ConstraintsPassed(passing.ConstraintResults)
	other := EvaluationCaseResult{
		CaseID:            "case-2",
		Metrics:           Metrics{MetricCalmar: 0.5},
		PassedConstraints: true,
	}
	ranked, err := RankEvaluationResults([]EvaluationCaseResult{other, passing}, EvaluationRankingSpec{Objective: MetricCalmar, Order: RankingOrderDesc})
	require.NoError(t, err)

	require.Len(t, ranked, 2)
	assert.Equal(t, "case-1", ranked[0].CaseID)
	assert.Equal(t, 1, ranked[0].Rank)
}

func TestCompileEvaluationIncludesConstraintMetricDependencies(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "constraint-metrics",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2024-01-01", To: "2024-01-31"},
		Metrics:       MetricSelectionSpec{Preset: "core"},
		Constraints: EvaluationConstraintSet{
			MaxExposureLTE:       floatPtr(0.75),
			MaxUnfilledCountLTE:  floatPtr(0),
			MaxDataIssueCountLTE: floatPtr(0),
		},
		Ranking: EvaluationRankingSpec{Objective: MetricCAGR},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 1)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricExposure)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricUnfilledCount)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricDataIssueCount)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricCAGR)
}

func TestEvaluationWeightedObjectiveRanking(t *testing.T) {
	conservative := EvaluationCaseResult{
		CaseID:            "conservative",
		Metrics:           Metrics{MetricCAGR: 0.08, MetricMaxDrawdown: -0.05, MetricTurnover: 0.4},
		PassedConstraints: true,
	}
	aggressive := EvaluationCaseResult{
		CaseID:            "aggressive",
		Metrics:           Metrics{MetricCAGR: 0.12, MetricMaxDrawdown: -0.20, MetricTurnover: 2.0},
		PassedConstraints: true,
	}

	ranked, err := RankEvaluationResults([]EvaluationCaseResult{aggressive, conservative}, EvaluationRankingSpec{
		Objective: EvaluationObjectiveWeightedScore,
		Order:     RankingOrderDesc,
		Weights: map[string]float64{
			MetricCAGR:        1,
			MetricMaxDrawdown: 1,
			MetricTurnover:    -0.01,
		},
	})
	require.NoError(t, err)

	require.Len(t, ranked, 2)
	assert.Equal(t, "conservative", ranked[0].CaseID)
	assert.Equal(t, EvaluationObjectiveWeightedScore, ranked[0].Objective)
	assert.InDelta(t, 0.026, ranked[0].ObjectiveValue, 0.0001)
}

func TestBuildRegimeSplitAggregatesCasesByTag(t *testing.T) {
	split := BuildRegimeSplit([]EvaluationCaseResult{
		{
			CaseID:            "case-1",
			Metrics:           Metrics{MetricCAGR: 0.10, MetricCalmar: 1.2},
			RegimeTags:        []string{"bull", "low_vol"},
			PassedConstraints: true,
			Rank:              2,
			Objective:         MetricCalmar,
			ObjectiveValue:    1.2,
		},
		{
			CaseID:            "case-2",
			Metrics:           Metrics{MetricCAGR: 0.20, MetricCalmar: 1.8},
			RegimeTags:        []string{"bull", "high_vol"},
			PassedConstraints: true,
			Rank:              1,
			Objective:         MetricCalmar,
			ObjectiveValue:    1.8,
		},
		{
			CaseID:            "case-3",
			Metrics:           Metrics{MetricCAGR: -0.05, MetricCalmar: -0.5},
			RegimeTags:        []string{"bear", "high_vol"},
			PassedConstraints: false,
			Objective:         MetricCalmar,
			ObjectiveValue:    -0.5,
		},
	})

	require.Len(t, split, 4)
	assert.Equal(t, "bear", split[0].Tag)
	assert.Equal(t, 1, split[0].CaseCount)
	assert.Equal(t, 0, split[0].PassedCount)
	assert.Equal(t, "case-3", split[0].BestCaseID)
	assert.InDelta(t, -0.05, split[0].AverageMetrics[MetricCAGR], 0.0001)

	assert.Equal(t, "bull", split[1].Tag)
	assert.Equal(t, 2, split[1].CaseCount)
	assert.Equal(t, 2, split[1].PassedCount)
	assert.Equal(t, "case-2", split[1].BestCaseID)
	assert.InDelta(t, 1.5, split[1].AverageObjective, 0.0001)
	assert.InDelta(t, 0.15, split[1].AverageMetrics[MetricCAGR], 0.0001)

	assert.Equal(t, "high_vol", split[2].Tag)
	assert.Equal(t, 2, split[2].CaseCount)
	assert.Equal(t, "case-2", split[2].BestCaseID)
	assert.Equal(t, "low_vol", split[3].Tag)
}

func TestRegimeTagsUseEvaluationRegimeThresholds(t *testing.T) {
	result := Result{
		TotalReturn: 0.03,
		EquityCurve: []EquityPoint{
			{Time: date("2024-01-02"), Equity: 100},
			{Time: date("2024-01-03"), Equity: 104},
			{Time: date("2024-01-04"), Equity: 101},
			{Time: date("2024-01-05"), Equity: 105},
		},
	}
	benchmark := []Bar{
		{Time: date("2024-01-02"), Close: 100},
		{Time: date("2024-01-05"), Close: 103},
	}

	defaultTags := RegimeTags(result, benchmark)
	assert.Contains(t, defaultTags, "sideways")

	customTags := RegimeTagsWithSpec(result, benchmark, EvaluationRegimeSpec{
		ReturnThreshold:     0.02,
		VolatilityThreshold: 0.01,
	})
	assert.Contains(t, customTags, "bull")
	assert.Contains(t, customTags, "high_vol")
}

func TestBuildRobustnessReportSummarizesSensitivityStabilityAndOutOfSample(t *testing.T) {
	report := BuildRobustnessReport([]EvaluationCaseResult{
		{
			CaseID:            "case-2024-w20",
			Period:            Period{From: date("2024-01-01"), To: date("2024-12-31")},
			Parameters:        map[string]any{"indicators.trend.params.window": 20},
			Metrics:           Metrics{MetricCalmar: 1.0},
			PassedConstraints: true,
			Objective:         MetricCalmar,
			ObjectiveValue:    1.0,
			Rank:              1,
		},
		{
			CaseID:            "case-2024-w60",
			Period:            Period{From: date("2024-01-01"), To: date("2024-12-31")},
			Parameters:        map[string]any{"indicators.trend.params.window": 60},
			Metrics:           Metrics{MetricCalmar: 0.8},
			PassedConstraints: true,
			Objective:         MetricCalmar,
			ObjectiveValue:    0.8,
			Rank:              2,
		},
		{
			CaseID:            "case-2025-w20",
			Period:            Period{From: date("2025-01-01"), To: date("2025-12-31")},
			Parameters:        map[string]any{"indicators.trend.params.window": 20},
			Metrics:           Metrics{MetricCalmar: 0.7},
			PassedConstraints: true,
			Objective:         MetricCalmar,
			ObjectiveValue:    0.7,
			Rank:              2,
		},
		{
			CaseID:            "case-2025-w60",
			Period:            Period{From: date("2025-01-01"), To: date("2025-12-31")},
			Parameters:        map[string]any{"indicators.trend.params.window": 60},
			Metrics:           Metrics{MetricCalmar: 0.85},
			PassedConstraints: true,
			Objective:         MetricCalmar,
			ObjectiveValue:    0.85,
			Rank:              1,
		},
	}, []WalkForwardStepResult{
		{Index: 1, TrainObjective: 1.0, TestMetrics: Metrics{MetricCalmar: 0.5}},
		{Index: 2, TrainObjective: 0.8, TestMetrics: Metrics{MetricCalmar: 0.4}},
	}, EvaluationRankingSpec{Objective: MetricCalmar, Order: RankingOrderDesc}, 1)

	require.Len(t, report.ParameterSensitivity, 1)
	sensitivity := report.ParameterSensitivity[0]
	assert.Equal(t, "indicators.trend.params.window", sensitivity.Parameter)
	assert.Equal(t, 4, sensitivity.CaseCount)
	assert.Equal(t, "20", sensitivity.BestValue)
	assert.Equal(t, "case-2024-w20", sensitivity.BestCaseID)
	assert.InDelta(t, 0.025, sensitivity.ObjectiveRange, 0.0001)
	require.Len(t, sensitivity.Values, 2)
	assert.Equal(t, "20", sensitivity.Values[0].Value)
	assert.InDelta(t, 0.85, sensitivity.Values[0].AverageObjective, 0.0001)

	assert.Equal(t, 1, report.TopNStability.TopN)
	assert.Equal(t, 2, report.TopNStability.PeriodCount)
	assert.Equal(t, 1, report.TopNStability.ComparedPairs)
	assert.InDelta(t, 0.0, report.TopNStability.AverageOverlap, 0.0001)

	require.NotNil(t, report.OutOfSampleDegradation)
	assert.Equal(t, MetricCalmar, report.OutOfSampleDegradation.Objective)
	assert.Equal(t, 2, report.OutOfSampleDegradation.StepCount)
	assert.InDelta(t, 0.9, report.OutOfSampleDegradation.AverageTrainObjective, 0.0001)
	assert.InDelta(t, 0.45, report.OutOfSampleDegradation.AverageTestObjective, 0.0001)
	assert.InDelta(t, 0.45, report.OutOfSampleDegradation.AverageDegradation, 0.0001)
	assert.InDelta(t, 0.5, report.OutOfSampleDegradation.DegradationPct, 0.0001)
}

func TestCompileEvaluationBuildsWalkForwardSteps(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-wf",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{From: "2020-01-01", To: "2024-12-31"},
		Parameters: map[string][]any{
			"indicators.trend.params.window": {2, 3},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
		WalkForward: WalkForwardSpec{
			Train:  DurationSpec{Years: 2},
			Test:   DurationSpec{Years: 1},
			Step:   DurationSpec{Years: 1},
			Select: WalkForwardSelectionSpec{Objective: MetricCalmar},
		},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.WalkForward, 3)
	assert.Equal(t, "2020-01-01", plan.WalkForward[0].Train.From.Format("2006-01-02"))
	assert.Equal(t, "2022-01-01", plan.WalkForward[0].Test.From.Format("2006-01-02"))
	assert.Len(t, plan.WalkForward[0].Cases, 2)
	assert.Len(t, plan.WalkForward[0].TestCases, 2)
}

func TestCompileEvaluationAppliesRegimeBenchmarkToCases(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-benchmark-regime",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Metrics:       MetricSelectionSpec{Preset: "research"},
		Regime: EvaluationRegimeSpec{
			Benchmark: BenchmarkSpec{Symbol: "102110", Name: "KODEX 200"},
		},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 1)
	assert.Equal(t, "102110", plan.Cases[0].Run.Benchmark.Symbol)
	assert.Equal(t, "KODEX 200", plan.Cases[0].Run.Benchmark.Name)
	assert.Equal(t, "krx", plan.Cases[0].Plan.Benchmark.Market)
}

func TestCompileEvaluationSupportsWalkForwardPeriodMode(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-wf-only",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodWalkForward, From: "2020-01-01", To: "2024-12-31"},
		Parameters: map[string][]any{
			"indicators.trend.params.window": {2, 3},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
		WalkForward: WalkForwardSpec{
			Train:  DurationSpec{Years: 2},
			Test:   DurationSpec{Years: 1},
			Step:   DurationSpec{Years: 1},
			Select: WalkForwardSelectionSpec{Objective: MetricCalmar},
		},
	}, registry)
	require.NoError(t, err)

	assert.Empty(t, plan.Cases)
	require.Len(t, plan.WalkForward, 3)
	assert.Equal(t, "2020-01-01", plan.WalkForward[0].Train.From.Format(time.DateOnly))
	assert.Equal(t, "2022-01-01", plan.WalkForward[0].Test.From.Format(time.DateOnly))
	assert.Len(t, plan.WalkForward[0].Cases, 2)
	assert.Len(t, plan.WalkForward[0].TestCases, 2)
}

func TestCompileEvaluationBuildsExpandingPeriodCases(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-expanding",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods: EvaluationPeriodsSpec{
			Mode:   EvaluationPeriodExpanding,
			From:   "2020-01-01",
			To:     "2020-01-05",
			Window: DurationSpec{Days: 2},
			Step:   DurationSpec{Days: 1},
		},
		Parameters: map[string][]any{
			"sizing.value": {25, 50},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 8)
	assert.Equal(t, "sma-expanding-expanding-1-p1", plan.Cases[0].Name)
	assert.Equal(t, "2020-01-01", plan.Cases[0].Run.Data.From)
	assert.Equal(t, "2020-01-02", plan.Cases[0].Run.Data.To)
	assert.Equal(t, "2020-01-01", plan.Cases[2].Run.Data.From)
	assert.Equal(t, "2020-01-03", plan.Cases[2].Run.Data.To)
	assert.Equal(t, "2020-01-01", plan.Cases[6].Run.Data.From)
	assert.Equal(t, "2020-01-05", plan.Cases[6].Run.Data.To)
}

func TestCompileEvaluationBuildsBoundedSearchCases(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-bounded",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Search: EvaluationSearchSpec{
			Mode: SearchModeBounded,
			Parameters: map[string]EvaluationSearchParameterSpec{
				"indicators.trend.params.window": {Min: floatPtr(2), Max: floatPtr(4), Step: floatPtr(1)},
				"sizing.value":                   {Values: []any{25, 50}},
			},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 6)
	assert.Equal(t, 2.0, plan.Cases[0].Parameters["indicators.trend.params.window"])
	assert.Equal(t, 25.0, plan.Cases[0].Parameters["sizing.value"])
	assert.Equal(t, 4.0, plan.Cases[5].Parameters["indicators.trend.params.window"])
	assert.Equal(t, 50.0, plan.Cases[5].Parameters["sizing.value"])
}

func TestCompileEvaluationRejectsBayesianSearchWithoutOptimizer(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	_, err = CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-bayesian",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Search: EvaluationSearchSpec{
			Mode:           SearchModeBayesian,
			Seed:           11,
			Samples:        3,
			InitialSamples: 2,
			Acquisition:    "expected_improvement",
			Parameters: map[string]EvaluationSearchParameterSpec{
				"indicators.trend.params.window": {Min: floatPtr(2), Max: floatPtr(6), Step: floatPtr(1)},
			},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
	}, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter search optimizer is not registered")
	assert.Contains(t, err.Error(), SearchModeBayesian)
}

func TestCompileEvaluationUsesCustomBayesianSearchOptimizer(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	var received EvaluationSearchSpec
	optimizers := EvaluationSearchOptimizerRegistry{
		SearchModeBayesian: EvaluationSearchOptimizerFunc(func(spec EvaluationSearchSpec) ([]map[string]any, error) {
			received = spec
			return []map[string]any{
				{"indicators.trend.params.window": 3.0, "sizing.value": 25.0},
				{"indicators.trend.params.window": 5.0, "sizing.value": 50.0},
			}, nil
		}),
	}
	plan, err := CompileEvaluationWithSearchOptimizers(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-bayesian",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Search: EvaluationSearchSpec{
			Mode:           SearchModeBayesian,
			Seed:           11,
			Samples:        2,
			InitialSamples: 1,
			Acquisition:    "expected_improvement",
			Parameters: map[string]EvaluationSearchParameterSpec{
				"indicators.trend.params.window": {Min: floatPtr(2), Max: floatPtr(6), Step: floatPtr(1)},
				"sizing.value":                   {Values: []any{25, 50}},
			},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
	}, registry, optimizers)
	require.NoError(t, err)

	assert.Equal(t, SearchModeBayesian, received.Mode)
	assert.Equal(t, "expected_improvement", received.Acquisition)
	assert.Equal(t, 1, received.InitialSamples)
	require.Len(t, plan.Cases, 2)
	assert.Equal(t, 3.0, plan.Cases[0].Strategy.Indicators["trend"].Params["window"])
	assert.Equal(t, 25.0, plan.Cases[0].Strategy.Sizing.Value)
	assert.Equal(t, 5.0, plan.Cases[1].Strategy.Indicators["trend"].Params["window"])
	assert.Equal(t, 50.0, plan.Cases[1].Strategy.Sizing.Value)
}

func TestCompileEvaluationIncludesWeightedObjectiveMetrics(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-weighted",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Metrics:       MetricSelectionSpec{Preset: "core"},
		Ranking: EvaluationRankingSpec{
			Objective: EvaluationObjectiveWeightedScore,
			Weights: map[string]float64{
				MetricCAGR:     1,
				MetricTurnover: -0.1,
			},
		},
	}, registry)
	require.NoError(t, err)

	require.Len(t, plan.Cases, 1)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricCAGR)
	assert.Contains(t, plan.Cases[0].Plan.SelectedMetrics, MetricTurnover)
}

func TestCompileEvaluationRejectsInvalidWeightedObjective(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	_, err = CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-weighted-invalid",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Ranking:       EvaluationRankingSpec{Objective: EvaluationObjectiveWeightedScore},
	}, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weighted objective requires weights")

	_, err = CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-weighted-unknown",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Ranking: EvaluationRankingSpec{
			Objective: EvaluationObjectiveWeightedScore,
			Weights:   map[string]float64{"mystery_metric": 1},
		},
	}, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metric is not registered")
}

func TestCompileEvaluationBuildsDeterministicRandomSearchCases(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	spec := EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-random",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Search: EvaluationSearchSpec{
			Mode:    SearchModeRandom,
			Seed:    7,
			Samples: 4,
			Parameters: map[string]EvaluationSearchParameterSpec{
				"indicators.trend.params.window": {Min: floatPtr(2), Max: floatPtr(10), Step: floatPtr(1)},
				"sizing.value":                   {Values: []any{25, 50, 75}},
			},
		},
		Metrics: MetricSelectionSpec{Preset: "research"},
	}

	first, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), spec, registry)
	require.NoError(t, err)
	second, err := CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), spec, registry)
	require.NoError(t, err)

	require.Len(t, first.Cases, 4)
	require.Len(t, second.Cases, 4)
	for index := range first.Cases {
		assert.Equal(t, first.Cases[index].Parameters, second.Cases[index].Parameters)
	}
}

func TestCompileEvaluationRejectsAmbiguousParameterInputs(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	_, err = CompileEvaluation(sampleEvaluationStrategy(), sampleEvaluationRun(), EvaluationSpec{
		Kind:          KindEvaluation,
		SchemaVersion: SchemaVersion,
		Name:          "sma-invalid-search",
		Strategy:      StrategyRef{Name: "sma-grid"},
		BaseRun:       EvaluationBaseRunRef{Ref: "sma-grid-run"},
		Periods:       EvaluationPeriodsSpec{Mode: EvaluationPeriodExplicit, From: "2020-01-01", To: "2020-01-03"},
		Parameters: map[string][]any{
			"sizing.value": {25, 50},
		},
		Search: EvaluationSearchSpec{Mode: SearchModeRandom, Samples: 2},
	}, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use parameters and search together")
}

func assertConstraintResult(t *testing.T, results []ConstraintResult, id string, passed bool, actual float64, limit float64) {
	t.Helper()

	for _, result := range results {
		if result.ID != id {
			continue
		}
		assert.Equal(t, passed, result.Passed)
		assert.InDelta(t, actual, result.Actual, 0.0001)
		assert.InDelta(t, limit, result.Limit, 0.0001)
		return
	}
	t.Fatalf("constraint result %q not found in %#v", id, results)
}

func sampleEvaluationStrategy() StrategySpec {
	return StrategySpec{
		Kind:          KindStrategy,
		SchemaVersion: SchemaVersion,
		Name:          "sma-grid",
		Indicators: map[string]IndicatorSpec{
			"trend": {
				ID:     "sma",
				Source: ValueExpr{Kind: "price", Price: "close"},
				Params: map[string]float64{"window": 2},
			},
		},
		Entry: RuleExpr{Operator: "crosses_above", Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "ref", Ref: "trend"},
		}},
		Exit: RuleExpr{Operator: "crosses_below", Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "ref", Ref: "trend"},
		}},
		Sizing: SizingSpec{Type: SizingPercentOfEquity, Value: 50},
		Risk:   RiskSpec{MaxPositions: 1},
	}
}

func sampleEvaluationRun() BacktestRunSpec {
	return BacktestRunSpec{
		Kind:          KindBacktestRun,
		SchemaVersion: SchemaVersion,
		Name:          "sma-grid-run",
		Strategy:      StrategyRef{Name: "sma-grid"},
		Data:          DataSpec{Market: "krx", Timeframe: "1d", From: "2020-01-01", To: "2024-12-31"},
		Universe: UniverseSpec{Pipeline: []UniverseSelectorStepSpec{{
			ID: "source.symbols",
			Params: map[string]any{
				"symbols": []any{"069500"},
				"fields":  map[string]any{"market": "krx", "security_type": "etf"},
			},
		}}},
		Portfolio: PortfolioSpec{InitialCash: 10000},
		Execution: ExecutionSpec{Fill: FillNextOpen},
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
