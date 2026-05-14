package backtest

import (
	"testing"

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
		Metrics:           Metrics{MetricCalmar: 1.5, MetricMaxDrawdown: -0.2, MetricCAGR: 0.1, MetricTurnover: 2, MetricTradeCount: 10},
		PassedConstraints: true,
	}
	failingConstraints := EvaluateConstraints(passing.Metrics, EvaluationConstraintSet{
		MaxDrawdownLTE: floatPtr(0.1),
		MinCAGRGTE:     floatPtr(0.08),
	})
	assert.False(t, ConstraintsPassed(failingConstraints))

	passing.ConstraintResults = EvaluateConstraints(passing.Metrics, EvaluationConstraintSet{MaxDrawdownLTE: floatPtr(0.25)})
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
