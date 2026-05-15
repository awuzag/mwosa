package backtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmarkAlphaBetaMetricsUseAlignedReturns(t *testing.T) {
	registry, err := DefaultMetricRegistry()
	require.NoError(t, err)

	result := Result{
		Benchmark: BenchmarkSpec{Symbol: "BMK"},
		EquityCurve: []EquityPoint{
			{Time: date("2024-01-02"), Equity: 100},
			{Time: date("2024-01-03"), Equity: 110},
			{Time: date("2024-01-04"), Equity: 99},
			{Time: date("2024-01-05"), Equity: 118.8},
		},
	}
	series := map[string][]Bar{
		"BMK": {
			{Time: date("2024-01-02"), Symbol: "BMK", Close: 100},
			{Time: date("2024-01-03"), Symbol: "BMK", Close: 105},
			{Time: date("2024-01-04"), Symbol: "BMK", Close: 99.75},
			{Time: date("2024-01-05"), Symbol: "BMK", Close: 109.725},
		},
	}
	ctx := MetricContext{Result: result, Series: series}

	beta, err := registry.defs[MetricBenchmarkBeta].Calculate(ctx)
	require.NoError(t, err)
	alpha, err := registry.defs[MetricBenchmarkAlpha].Calculate(ctx)
	require.NoError(t, err)

	assert.InDelta(t, 2.0, beta, 0.0001)
	assert.InDelta(t, 0.0, alpha, 0.0001)
}

func TestResearchPresetIncludesBenchmarkCompareMetricsWhenBenchmarkConfigured(t *testing.T) {
	registry, err := DefaultMetricRegistry()
	require.NoError(t, err)

	selected, err := resolveMetricSelection(ReportSpec{Metrics: MetricSelectionSpec{Preset: "research"}}, true, registry)
	require.NoError(t, err)

	assert.Contains(t, selected, MetricBenchmarkTotalReturn)
	assert.Contains(t, selected, MetricExcessReturn)
	assert.Contains(t, selected, MetricBenchmarkMaxDrawdown)
	assert.Contains(t, selected, MetricRelativeDrawdown)
	assert.Contains(t, selected, MetricMonthlyWinRateVsBenchmark)
	assert.Contains(t, selected, MetricBenchmarkAlpha)
	assert.Contains(t, selected, MetricBenchmarkBeta)
}

func TestDataIssueCountUsesDataEventLedger(t *testing.T) {
	registry, err := DefaultMetricRegistry()
	require.NoError(t, err)

	value, err := registry.defs[MetricDataIssueCount].Calculate(MetricContext{
		Result: Result{
			DataEvents: []Event{
				{Layer: "data", Type: "data_issue", Reason: "missing_bar"},
				{Layer: "data", Type: "data_issue", Reason: "no_trade_bar"},
			},
			ExecutionEvents: []Event{
				{Layer: "execution", Type: "deferred_no_trade_bar"},
			},
		},
	})
	require.NoError(t, err)

	assert.InDelta(t, 2.0, value, 0.0001)
}
