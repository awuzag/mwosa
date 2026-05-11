package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineRunsCompiledStrategyPlanWithNextOpenFills(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := Compile(testStrategySpec(), testRunSpec(), registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed(testBars()))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, "069500", result.Trades[0].Symbol)
	assert.InDelta(t, 384.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 13.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 10.0, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 8848.0, result.FinalEquity, 0.0001)
	assert.InDelta(t, -0.1152, result.TotalReturn, 0.0001)
	assert.InDelta(t, -0.1152, result.MaxDrawdown, 0.0001)
	assert.Equal(t, 0, result.UnfilledCount)
	assert.Equal(t, "next_open", result.Execution.Fill)
	assert.Equal(t, []string{"069500"}, result.Symbols)
	assert.InDelta(t, -1152.0, result.RealizedPnL, 0.0001)
	assert.InDelta(t, 0.0, result.WinRate, 0.0001)
	assert.InDelta(t, -0.230769, result.AverageTradeRet, 0.0001)
	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "trade_count", "win_rate"}, result.SelectedMetrics)
	assert.InDelta(t, result.TotalReturn, result.Metrics["total_return"], 0.0001)
	assert.InDelta(t, result.FinalEquity, result.Metrics["final_equity"], 0.0001)
	assert.NotEmpty(t, result.ResultHash)

	repeated, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)
	assert.Equal(t, result.ResultHash, repeated.ResultHash)
}

func TestCompileDefaultsToCoreMetrics(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := Compile(testStrategySpec(), testRunSpec(), registry)
	require.NoError(t, err)

	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "trade_count", "win_rate"}, plan.SelectedMetrics)
}

func TestCompileIncludesAndExcludesMetrics(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Report.Metrics = MetricSelectionSpec{
		Include: []string{"average_trade_return", "average_trade_return"},
		Exclude: []string{"trade_count"},
	}
	plan, err := Compile(testStrategySpec(), run, registry)
	require.NoError(t, err)

	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "win_rate", "average_trade_return"}, plan.SelectedMetrics)
}

func TestCompileRejectsUnknownMetric(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Report.Metrics = MetricSelectionSpec{Include: []string{"mystery_return"}}
	_, err = Compile(testStrategySpec(), run, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metric is not registered")
	assert.Contains(t, err.Error(), "mystery_return")
}

func TestCompileRejectsBenchmarkMetricWithoutBenchmark(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Report.Metrics = MetricSelectionSpec{Include: []string{"benchmark_total_return"}}
	_, err = Compile(testStrategySpec(), run, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "benchmark metric requires benchmark")
	assert.Contains(t, err.Error(), "benchmark_total_return")
}

func TestEngineReportsBenchmarkMetricsWhenBenchmarkConfigured(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Benchmark = BenchmarkSpec{Symbol: "102110", Name: "benchmark"}
	run.Report.Metrics = MetricSelectionSpec{
		Include: []string{"benchmark_total_return", "excess_return", "benchmark_max_drawdown", "relative_drawdown", "monthly_win_rate_vs_benchmark"},
	}
	plan, err := Compile(testStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed(append(testBars(), benchmarkBars()...)))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	assert.Contains(t, result.Metrics, "benchmark_total_return")
	assert.Contains(t, result.Metrics, "excess_return")
	assert.Contains(t, result.Metrics, "benchmark_max_drawdown")
	assert.Contains(t, result.Metrics, "relative_drawdown")
	assert.Contains(t, result.Metrics, "monthly_win_rate_vs_benchmark")
	assert.InDelta(t, 0.10, result.Metrics["benchmark_total_return"], 0.0001)
}

func TestEngineReportsRiskRejections(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := testStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	strategy.Risk = RiskSpec{MaxSymbolWeightPct: 10}
	run := testRunSpec()
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed(testBars()))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.NotEmpty(t, result.RiskEvents)
	assert.Equal(t, "risk", result.RiskEvents[0].Layer)
	assert.Equal(t, "rejected", result.RiskEvents[0].Type)
	assert.Equal(t, "max_symbol_weight_pct", result.RiskEvents[0].Reason)
	assert.Empty(t, result.Trades)
}

func TestEngineDefersNextOpenFillOnNoTradeBarUntilNextFillableBar(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := testStrategySpec()
	strategy.Indicators = nil
	strategy.Entry = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	strategy.Exit = RuleExpr{
		Operator: "lt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	plan, err := Compile(strategy, testRunSpec(), registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 0, High: 0, Low: 0, Close: 10, Volume: 0},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-04"), result.Trades[0].Time)
	assert.InDelta(t, 12.0, result.Trades[0].Price, 0.0001)
	require.Len(t, result.ExecutionEvents, 1)
	assert.Equal(t, "execution", result.ExecutionEvents[0].Layer)
	assert.Equal(t, "deferred_no_trade_bar", result.ExecutionEvents[0].Type)
	assert.Equal(t, "069500", result.ExecutionEvents[0].Symbol)
	assert.Equal(t, SideBuy, result.ExecutionEvents[0].Side)
	assert.Equal(t, "no_trade_bar", result.ExecutionEvents[0].Reason)
	assert.Equal(t, 1, result.UnfilledCount)
}

func TestEngineDoesNotTreatContradictoryOpenZeroBarAsNoTradeBar(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := testStrategySpec()
	strategy.Indicators = nil
	strategy.Entry = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	strategy.Exit = RuleExpr{
		Operator: "lt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	plan, err := Compile(strategy, testRunSpec(), registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 0, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid market data bar")
	assert.NotContains(t, err.Error(), "fill price must be positive")
	assert.Empty(t, result.ExecutionEvents)
}

func TestDefaultIndicatorRegistrySupportsMomentumAndChannelIndicators(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	rsi, ok := registry.Definition("rsi")
	require.True(t, ok)
	rsiValues, err := rsi.Calculate(IndicatorSpec{
		ID:     "rsi",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 2},
	}, []Bar{
		{Close: 10},
		{Close: 12},
		{Close: 11},
		{Close: 13},
	})
	require.NoError(t, err)
	assert.InDelta(t, 66.6666, rsiValues[2], 0.001)

	high, ok := registry.Definition("donchian_high")
	require.True(t, ok)
	highValues, err := high.Calculate(IndicatorSpec{
		ID:     "donchian_high",
		Source: ValueExpr{Kind: "price", Price: "high"},
		Params: map[string]float64{"window": 3},
	}, []Bar{
		{High: 10},
		{High: 12},
		{High: 11},
		{High: 13},
	})
	require.NoError(t, err)
	assert.InDelta(t, 12, highValues[2], 0.0001)
	assert.InDelta(t, 13, highValues[3], 0.0001)

	low, ok := registry.Definition("donchian_low")
	require.True(t, ok)
	lowValues, err := low.Calculate(IndicatorSpec{
		ID:     "donchian_low",
		Source: ValueExpr{Kind: "price", Price: "low"},
		Params: map[string]float64{"window": 3},
	}, []Bar{
		{Low: 10},
		{Low: 12},
		{Low: 11},
		{Low: 13},
	})
	require.NoError(t, err)
	assert.InDelta(t, 10, lowValues[2], 0.0001)
	assert.InDelta(t, 11, lowValues[3], 0.0001)
}

func testStrategySpec() StrategySpec {
	return StrategySpec{
		Kind:          KindStrategy,
		SchemaVersion: SchemaVersion,
		Name:          "sma-cross",
		Indicators: map[string]IndicatorSpec{
			"trend": {
				ID:     "sma",
				Source: ValueExpr{Kind: "price", Price: "close"},
				Params: map[string]float64{"window": 2},
			},
		},
		Entry: RuleExpr{
			Operator: "crosses_above",
			Args: []ValueExpr{
				{Kind: "price", Price: "close"},
				{Kind: "ref", Ref: "trend"},
			},
		},
		Exit: RuleExpr{
			Operator: "crosses_below",
			Args: []ValueExpr{
				{Kind: "price", Price: "close"},
				{Kind: "ref", Ref: "trend"},
			},
		},
		Sizing: SizingSpec{Type: SizingPercentOfEquity, Value: 50},
		Risk:   RiskSpec{MaxPositions: 1, MaxSymbolWeightPct: 60},
	}
}

func testRunSpec() BacktestRunSpec {
	return BacktestRunSpec{
		Kind:          KindBacktestRun,
		SchemaVersion: SchemaVersion,
		Name:          "sma-cross-run",
		Strategy:      StrategyRef{Name: "sma-cross"},
		Data: DataSpec{
			Market:       "krx",
			SecurityType: "etf",
			Timeframe:    "1d",
			From:         "2024-01-02",
			To:           "2024-01-08",
		},
		Universe:  UniverseSpec{Symbols: []string{"069500"}},
		Portfolio: PortfolioSpec{InitialCash: 10000, Currency: "KRW"},
		Execution: ExecutionSpec{
			Fill:       FillNextOpen,
			Commission: CostSpec{Type: "bps", Value: 0},
			Slippage:   CostSpec{Type: "bps", Value: 0},
		},
	}
}

func testBars() []Bar {
	return []Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 9, High: 9, Low: 9, Close: 9},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 13, High: 13, Low: 11, Close: 11},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
	}
}

func benchmarkBars() []Bar {
	return []Bar{
		{Time: date("2024-01-02"), Symbol: "102110", Open: 100, High: 100, Low: 100, Close: 100},
		{Time: date("2024-01-03"), Symbol: "102110", Open: 105, High: 105, Low: 105, Close: 105},
		{Time: date("2024-01-04"), Symbol: "102110", Open: 95, High: 95, Low: 95, Close: 95},
		{Time: date("2024-01-05"), Symbol: "102110", Open: 110, High: 110, Low: 110, Close: 110},
		{Time: date("2024-01-08"), Symbol: "102110", Open: 110, High: 110, Low: 110, Close: 110},
	}
}

func date(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
