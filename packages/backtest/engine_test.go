package backtest

import (
	"context"
	"math"
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
	assert.Equal(t, Timeframe1Day, result.Timeframes.Requested)
	assert.Equal(t, Timeframe1Day, result.Timeframes.Source)
	assert.Equal(t, Timeframe1Day, result.Timeframes.Execution)
	assert.False(t, result.Timeframes.Resample.Enabled)
	assert.Equal(t, 2, result.Timeframes.Warmup.Bars)
	assert.Equal(t, "indicator_lookback_bars", result.Timeframes.Warmup.Policy)
	assert.Equal(t, EngineVersion, result.Runtime.EngineVersion)
	assert.Equal(t, DefaultIndicatorRegistryVersion, result.Runtime.IndicatorRegistryVersion)
	assert.Equal(t, DefaultMetricRegistryVersion, result.Runtime.MetricRegistryVersion)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "fill", "portfolio_mutation", "order_intent", "fill", "portfolio_mutation")
	assert.Equal(t, []string{"069500"}, result.Symbols)
	assert.InDelta(t, -1152.0, result.RealizedPnL, 0.0001)
	assert.InDelta(t, 0.0, result.WinRate, 0.0001)
	assert.InDelta(t, -0.230769, result.AverageTradeRet, 0.0001)
	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "trade_count", "win_rate"}, result.SelectedMetrics)
	assert.InDelta(t, result.TotalReturn, result.Metrics["total_return"], 0.0001)
	assert.InDelta(t, result.FinalEquity, result.Metrics["final_equity"], 0.0001)
	assert.NotEmpty(t, result.DataFingerprint)
	assert.NotEmpty(t, result.ResultHash)

	repeated, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)
	assert.Equal(t, result.DataFingerprint, repeated.DataFingerprint)
	assert.Equal(t, result.ResultHash, repeated.ResultHash)
}

func TestDefaultRegistriesExposeRuntimeVersions(t *testing.T) {
	indicators, err := DefaultIndicatorRegistry()
	require.NoError(t, err)
	assert.Equal(t, DefaultIndicatorRegistryVersion, indicators.Version())

	customIndicators, err := NewIndicatorRegistry(SMA())
	require.NoError(t, err)
	assert.Equal(t, CustomIndicatorRegistryVersion, customIndicators.Version())

	metrics, err := DefaultMetricRegistry()
	require.NoError(t, err)
	assert.Equal(t, DefaultMetricRegistryVersion, metrics.Version())

	customMetrics, err := NewMetricRegistry(fieldMetric(MetricTotalReturn, func(result Result) float64 {
		return result.TotalReturn
	}))
	require.NoError(t, err)
	assert.Equal(t, CustomMetricRegistryVersion, customMetrics.Version())
}

func TestEngineRejectsNonMonotonicBarFrames(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := Compile(alwaysLongStrategySpec(), testRunSpec(), registry)
	require.NoError(t, err)

	engine, err := NewEngine(scriptedFeed{frames: []BarFrame{
		{
			Time: date("2024-01-03"),
			Bars: map[string]Bar{
				"069500": {Time: date("2024-01-03"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
			},
		},
		{
			Time: date("2024-01-02"),
			Bars: map[string]Bar{
				"069500": {Time: date("2024-01-02"), Symbol: "069500", Open: 9, High: 9, Low: 9, Close: 9},
			},
		},
	}})
	require.NoError(t, err)

	_, err = engine.Run(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "historical bar frame time must be strictly increasing")
}

func TestTimeframeQualifiedValueUsesLastClosedFrameWithoutFutureLeakage(t *testing.T) {
	bars := []Bar{
		{Time: date("2024-01-03"), Symbol: "069500", Timeframe: Timeframe1Day, Close: 10},
		{Time: date("2024-01-04"), Symbol: "069500", Timeframe: Timeframe1Day, Close: 11},
		{Time: date("2024-01-05"), Symbol: "069500", Timeframe: Timeframe1Week, Close: 99},
		{Time: date("2024-01-05"), Symbol: "069500", Timeframe: Timeframe1Day, Close: 12},
	}
	ctx := ruleContext{
		symbol: "069500",
		index:  1,
		bars:   bars,
		series: map[string][]Bar{"069500": bars},
	}
	expr := ValueExpr{Kind: "price", Price: "close", Timeframe: Timeframe1Week}

	value, ok, err := currentValue(expr, ctx)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, value)

	ctx.index = 3
	value, ok, err = currentValue(expr, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 99.0, value, 0.0001)
}

func TestCompileValidatesTimeframeQualifiedValueExpressions(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry.Args[0].Timeframe = Timeframe1Week
	_, err = Compile(strategy, testRunSpec(), registry)
	require.NoError(t, err)

	strategy = alwaysLongStrategySpec()
	strategy.Entry.Args[0].Timeframe = "2w"
	_, err = Compile(strategy, testRunSpec(), registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported weekly timeframe")

	strategy = alwaysLongStrategySpec()
	strategy.Entry.Args[0] = ValueExpr{Kind: "position", Position: "holding_bars", Timeframe: Timeframe1Week}
	_, err = Compile(strategy, testRunSpec(), registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "position and portfolio value expressions cannot be timeframe scoped")
}

func TestEngineSupportsSameCloseAndNextCloseFillTiming(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()

	sameCloseRun := testRunSpec()
	sameCloseRun.Execution.Fill = FillSameClose
	sameClosePlan, err := Compile(strategy, sameCloseRun, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 11, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 13, Volume: 1000},
	}))
	require.NoError(t, err)

	sameCloseResult, err := engine.Run(context.Background(), sameClosePlan)
	require.NoError(t, err)
	require.Len(t, sameCloseResult.Trades, 1)
	assert.Equal(t, date("2024-01-02"), sameCloseResult.Trades[0].Time)
	assert.InDelta(t, 11.0, sameCloseResult.Trades[0].Price, 0.0001)

	nextCloseRun := testRunSpec()
	nextCloseRun.Execution.Fill = FillNextClose
	nextClosePlan, err := Compile(strategy, nextCloseRun, registry)
	require.NoError(t, err)

	nextCloseResult, err := engine.Run(context.Background(), nextClosePlan)
	require.NoError(t, err)
	require.Len(t, nextCloseResult.Trades, 1)
	assert.Equal(t, date("2024-01-03"), nextCloseResult.Trades[0].Time)
	assert.InDelta(t, 13.0, nextCloseResult.Trades[0].Price, 0.0001)
}

func TestEngineSupportsIntrabarOHLCFillTimingWithoutSameBarLookahead(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillIntrabarOHLC
	run.Execution.IntrabarAmbiguityPolicy = IntrabarAmbiguityOpenHighLowClose
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 11, Low: 9, Close: 11, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 14, Low: 10, Close: 13, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, date("2024-01-03"), result.Trades[0].Time)
	assert.InDelta(t, 12.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, FillIntrabarOHLC, result.Execution.Fill)
	assert.Equal(t, IntrabarAmbiguityOpenHighLowClose, result.Execution.IntrabarAmbiguityPolicy)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "intrabar_path", "fill", "portfolio_mutation")
	assert.Equal(t, IntrabarAmbiguityOpenHighLowClose, result.ExecutionEvents[1].Reason)
	assert.InDelta(t, 12.0, result.ExecutionEvents[1].Price, 0.0001)
}

func TestEngineSupportsLimitOrderType(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeLimit
	run.Execution.LimitPrice = 10
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 11, High: 12, Low: 11, Close: 11, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 12, Low: 9, Close: 12, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-03"), result.Trades[0].Time)
	assert.InDelta(t, 500.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 10.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 11000.0, result.FinalEquity, 0.0001)
	assert.Equal(t, 1, result.UnfilledCount)
	assert.Equal(t, OrderTypeLimit, result.Execution.OrderType)
	assert.InDelta(t, 10.0, result.Execution.LimitPrice, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "unfilled", "order_intent", "fill", "portfolio_mutation")
	assert.Equal(t, "limit_not_reached", result.ExecutionEvents[1].Reason)
}

func TestEngineCarriesUnfilledPendingOrderWithGTCTimeInForce(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillNextOpen
	run.Execution.OrderType = OrderTypeLimit
	run.Execution.LimitPrice = 10
	run.Execution.TimeInForce = TimeInForceGTC
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 11, High: 12, Low: 11, Close: 11, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 13, Low: 11, Close: 12, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 11, High: 11, Low: 9, Close: 11, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, date("2024-01-04"), result.Trades[0].Time)
	assert.InDelta(t, 10.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, TimeInForceGTC, result.Execution.TimeInForce)
	assert.Equal(t, 1, result.UnfilledCount)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "unfilled", "fill", "portfolio_mutation")
	assert.Equal(t, "limit_not_reached", result.ExecutionEvents[1].Reason)
}

func TestEngineCancelsPendingOrderOnRebalanceTimeInForce(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{Operator: "monthly"},
			{
				Operator: "target_weight_changed",
				Args:     []ValueExpr{{Kind: "value", Value: 5}},
			},
		},
	}

	run := testRunSpec()
	run.Data.From = "2024-01-31"
	run.Data.To = "2024-03-02"
	run.Execution.Fill = FillNextOpen
	run.Execution.OrderType = OrderTypeRebalance
	run.Execution.TimeInForce = TimeInForceCancelOnRebalance
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-31"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-02-01"), Symbol: "069500", Open: 100, High: 200, Low: 100, Close: 200, Volume: 1000},
		{Time: date("2024-02-02"), Symbol: "069500", Open: 0, High: 0, Low: 0, Close: 200, Volume: 0, TradedAmount: 0},
		{Time: date("2024-03-01"), Symbol: "069500", Open: 0, High: 0, Low: 0, Close: 200, Volume: 0, TradedAmount: 0},
		{Time: date("2024-03-02"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, TimeInForceCancelOnRebalance, result.Execution.TimeInForce)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "fill", "portfolio_mutation", "order_intent", "deferred_no_trade_bar", "deferred_no_trade_bar", "order_cancelled", "order_intent", "fill", "portfolio_mutation")
	assert.Equal(t, "cancel_on_rebalance", result.ExecutionEvents[6].Reason)
	assert.Equal(t, SideSell, result.ExecutionEvents[6].Side)
	assert.InDelta(t, 12.0, result.ExecutionEvents[6].Quantity, 0.0001)
	assert.Equal(t, "rebalance", result.Trades[1].Reason)
	assert.Equal(t, date("2024-03-02"), result.Trades[1].Time)
}

func TestCompileRejectsLimitOrderWithoutLimitPrice(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.OrderType = OrderTypeLimit

	_, err = Compile(alwaysLongStrategySpec(), run, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit_price")
}

func TestEngineSupportsStopOrderType(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeStop
	run.Execution.StopPrice = 12
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 11, Low: 10, Close: 11, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 11, High: 13, Low: 11, Close: 13, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-03"), result.Trades[0].Time)
	assert.InDelta(t, 416.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 12.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, 1, result.UnfilledCount)
	assert.Equal(t, OrderTypeStop, result.Execution.OrderType)
	assert.InDelta(t, 12.0, result.Execution.StopPrice, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "unfilled", "order_intent", "stop_triggered", "fill", "portfolio_mutation")
	assert.Equal(t, "stop_not_triggered", result.ExecutionEvents[1].Reason)
	assert.Equal(t, "stop_price", result.ExecutionEvents[3].Reason)
}

func TestEngineSupportsTrailingStopOrderType(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeTrailingStop
	run.Execution.TrailingStopPct = 10
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 118, High: 120, Low: 115, Close: 118, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 118, High: 119, Low: 107, Close: 108, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-01-04"), result.Trades[1].Time)
	assert.InDelta(t, 108.0, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 10400.0, result.FinalEquity, 0.0001)
	assert.Equal(t, 1, result.UnfilledCount)
	assert.Equal(t, OrderTypeTrailingStop, result.Execution.OrderType)
	assert.InDelta(t, 10.0, result.Execution.TrailingStopPct, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents,
		"order_intent", "fill", "portfolio_mutation",
		"order_intent", "trailing_stop_updated", "unfilled",
		"order_intent", "trailing_stop_updated", "stop_triggered", "fill", "portfolio_mutation",
	)
	assert.Equal(t, "trailing_stop_not_triggered", result.ExecutionEvents[5].Reason)
	assert.InDelta(t, 90.0, result.ExecutionEvents[4].Price, 0.0001)
	assert.InDelta(t, 120.0, result.ExecutionEvents[7].Amount, 0.0001)
	assert.InDelta(t, 108.0, result.ExecutionEvents[7].Price, 0.0001)
	assert.Equal(t, "trailing_stop", result.ExecutionEvents[8].Reason)
}

func TestCompileRejectsTrailingStopOrderWithoutTrailingStopPct(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.OrderType = OrderTypeTrailingStop

	_, err = Compile(alwaysLongStrategySpec(), run, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing_stop_pct")
}

func TestEngineSupportsStopLimitOrderTypeWithIntrabarAmbiguityPolicy(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeStopLimit
	run.Execution.StopPrice = 12
	run.Execution.LimitPrice = 11
	conservativePlan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	ambiguousBar := Bar{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 13, Low: 10, Close: 12, Volume: 1000}
	engine, err := NewEngine(NewMemoryFeed([]Bar{ambiguousBar}))
	require.NoError(t, err)

	conservativeResult, err := engine.Run(context.Background(), conservativePlan)
	require.NoError(t, err)
	assert.Empty(t, conservativeResult.Trades)
	assert.Equal(t, 1, conservativeResult.UnfilledCount)
	assert.Equal(t, IntrabarAmbiguityConservative, conservativeResult.Execution.IntrabarAmbiguityPolicy)
	assertExecutionEventTypes(t, conservativeResult.ExecutionEvents, "order_intent", "stop_triggered", "unfilled")
	assert.Equal(t, "intrabar_ambiguous", conservativeResult.ExecutionEvents[2].Reason)

	run.Execution.IntrabarAmbiguityPolicy = IntrabarAmbiguityOptimistic
	optimisticPlan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)
	engine, err = NewEngine(NewMemoryFeed([]Bar{ambiguousBar}))
	require.NoError(t, err)

	optimisticResult, err := engine.Run(context.Background(), optimisticPlan)
	require.NoError(t, err)
	require.Len(t, optimisticResult.Trades, 1)
	assert.Equal(t, OrderTypeStopLimit, optimisticResult.Execution.OrderType)
	assert.Equal(t, IntrabarAmbiguityOptimistic, optimisticResult.Execution.IntrabarAmbiguityPolicy)
	assert.InDelta(t, 11.0, optimisticResult.Trades[0].Price, 0.0001)
	assertExecutionEventTypes(t, optimisticResult.ExecutionEvents, "order_intent", "stop_triggered", "fill", "portfolio_mutation")
}

func TestCompileRejectsStopOrdersWithoutRequiredPrices(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	stopRun := testRunSpec()
	stopRun.Execution.OrderType = OrderTypeStop
	_, err = Compile(alwaysLongStrategySpec(), stopRun, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stop_price")

	stopLimitRun := testRunSpec()
	stopLimitRun.Execution.OrderType = OrderTypeStopLimit
	stopLimitRun.Execution.StopPrice = 12
	_, err = Compile(alwaysLongStrategySpec(), stopLimitRun, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit_price")
}

func TestEngineCarriesPartialFillWhenLiquidityCapsOrder(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillNextOpen
	run.Execution.Liquidity = LiquiditySpec{MaxParticipationRate: 0.10}
	run.Execution.PartialFill = PartialFillSpec{Policy: PartialFillCarry}
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000, TradedAmount: 10000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10, TradedAmount: 100},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 20, TradedAmount: 200},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 5000, TradedAmount: 50000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 3)
	assert.InDelta(t, 1.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 2.0, result.Trades[1].Quantity, 0.0001)
	assert.InDelta(t, 497.0, result.Trades[2].Quantity, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "partial_fill", "fill", "portfolio_mutation", "partial_fill", "fill", "portfolio_mutation", "fill", "portfolio_mutation")
	assert.Equal(t, 0, result.UnfilledCount)
}

func TestEngineCancelsPartialFillRemainderWithIOCTimeInForce(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillNextOpen
	run.Execution.TimeInForce = TimeInForceIOC
	run.Execution.Liquidity = LiquiditySpec{MaxParticipationRate: 0.10}
	run.Execution.PartialFill = PartialFillSpec{Policy: PartialFillCarry}
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000, TradedAmount: 10000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10, TradedAmount: 100},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 5000, TradedAmount: 50000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.InDelta(t, 1.0, result.Trades[0].Quantity, 0.0001)
	assert.Equal(t, TimeInForceIOC, result.Execution.TimeInForce)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "partial_fill", "fill", "order_cancelled", "portfolio_mutation")
	assert.Equal(t, "time_in_force_ioc", result.ExecutionEvents[3].Reason)
	assert.InDelta(t, 499.0, result.ExecutionEvents[3].Quantity, 0.0001)
	assert.InDelta(t, 4990.0, result.ExecutionEvents[3].Amount, 0.0001)
}

func TestEngineAppliesSideSpecificCommissionMinimumFeeAndFixedSlippage(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.Commission = CostSpec{Type: CostTypeBPS, BuyValue: 10, SellValue: 20, MinFee: 5}
	run.Execution.Tax = CostSpec{Type: CostTypeBPS, SellValue: 10}
	run.Execution.ExchangeFee = CostSpec{Type: CostTypeFixedAmount, Value: 1}
	run.Execution.Slippage = CostSpec{Type: CostTypeFixedAmount, BuyValue: 1, SellValue: 2}
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 120, High: 120, Low: 120, Close: 120, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 49.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 101.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 5.0, result.Trades[0].Commission, 0.0001)
	assert.InDelta(t, 0.0, result.Trades[0].Tax, 0.0001)
	assert.InDelta(t, 1.0, result.Trades[0].ExchangeFee, 0.0001)
	assert.InDelta(t, 6.0, result.Trades[0].TotalCost, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[0].SlippageBps, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 118.0, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 11.564, result.Trades[1].Commission, 0.0001)
	assert.InDelta(t, 5.782, result.Trades[1].Tax, 0.0001)
	assert.InDelta(t, 1.0, result.Trades[1].ExchangeFee, 0.0001)
	assert.InDelta(t, 18.346, result.Trades[1].TotalCost, 0.0001)
	assert.InDelta(t, 166.6667, result.Trades[1].SlippageBps, 0.0001)
	assert.InDelta(t, 10808.654, result.FinalEquity, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "fill", "cost", "cost", "portfolio_mutation", "order_intent", "fill", "cost", "cost", "cost", "portfolio_mutation")
	assert.Equal(t, "tax", result.ExecutionEvents[8].Reason)
	assert.Equal(t, "exchange_fee", result.ExecutionEvents[9].Reason)
}

func TestEngineAppliesParticipationSlippageFromFillQuantity(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.Slippage = CostSpec{Type: CostTypeParticipation, SellValue: 1000}
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 50.0, result.Trades[1].Quantity, 0.0001)
	assert.InDelta(t, 99.5, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 50.0, result.Trades[1].SlippageBps, 0.0001)
	assert.InDelta(t, -25.0, result.Trades[1].RealizedPnL, 0.0001)
}

func TestEngineAppliesSpreadProxySlippageFromBarRange(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.Slippage = CostSpec{Type: CostTypeSpreadProxy, Value: 0.5}
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 49.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 101.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[0].SlippageBps, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 99.0, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[1].SlippageBps, 0.0001)
	assert.InDelta(t, -98.0, result.Trades[1].RealizedPnL, 0.0001)
}

func TestEngineAppliesVolatilitySlippageFromBarRangePct(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.Slippage = CostSpec{Type: CostTypeVolatility, Value: 0.25}
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 102, Low: 98, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 100, High: 102, Low: 98, Close: 100, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 49.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 101.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[0].SlippageBps, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 99.0, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[1].SlippageBps, 0.0001)
	assert.InDelta(t, -98.0, result.Trades[1].RealizedPnL, 0.0001)
}

func TestEngineAppliesATRSlippageFromIntentTimeSnapshot(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillNextOpen
	run.Execution.Slippage = CostSpec{Type: CostTypeATR, Value: 0.5, Window: 2}
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 100, High: 150, Low: 50, Close: 100, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-04"), result.Trades[0].Time)
	assert.InDelta(t, 49.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 101.0, result.Trades[0].Price, 0.0001)
	assert.InDelta(t, 100.0, result.Trades[0].SlippageBps, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "unfilled", "order_intent", "fill", "portfolio_mutation")
	assert.Equal(t, "slippage_atr_not_ready", result.ExecutionEvents[1].Reason)
}

func TestCompileRejectsATRSlippageWithoutWindow(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Slippage = CostSpec{Type: CostTypeATR, Value: 0.5}
	_, err = Compile(alwaysLongStrategySpec(), run, registry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atr slippage requires positive window")
}

func TestEngineAppliesLotSizeAndTickSizeConstraints(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 0},
		},
	}
	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.LotSize = 10
	run.Execution.TickSize = 0.1
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100.03, High: 100.03, Low: 100.03, Close: 100.03, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 99.97, High: 99.97, Low: 99.97, Close: 99.97, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 40.0, result.Trades[0].Quantity, 0.0001)
	assert.InDelta(t, 100.1, result.Trades[0].Price, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 40.0, result.Trades[1].Quantity, 0.0001)
	assert.InDelta(t, 99.9, result.Trades[1].Price, 0.0001)
	assert.InDelta(t, -8.0, result.Trades[1].RealizedPnL, 0.0001)
}

func TestEngineCancelsPartialFillWhenConfigured(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Liquidity = LiquiditySpec{MaxParticipationRate: 0.10}
	run.Execution.PartialFill = PartialFillSpec{Policy: PartialFillCancel}
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000, TradedAmount: 10000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10, TradedAmount: 100},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000, TradedAmount: 10000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.InDelta(t, 1.0, result.Trades[0].Quantity, 0.0001)
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "partial_fill", "fill", "order_cancelled", "portfolio_mutation")
	assert.Equal(t, "partial_fill_cancel", result.ExecutionEvents[3].Reason)
	assert.Equal(t, 0, result.UnfilledCount)
}

func TestEngineSupportsTemporalAndPortfolioStateRules(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{
				Operator: "between",
				Args: []ValueExpr{
					{Kind: "price", Price: "close"},
					{Kind: "value", Value: 10},
					{Kind: "value", Value: 20},
				},
			},
			{
				Operator: "rising",
				Args: []ValueExpr{
					{Kind: "price", Price: "close"},
					{Kind: "value", Value: 2},
				},
			},
			{
				Operator: "lt",
				Args: []ValueExpr{
					{Kind: "portfolio", Portfolio: "position_count"},
					{Kind: "value", Value: 1},
				},
			},
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "portfolio", Portfolio: "cash_pct"},
					{Kind: "value", Value: 90},
				},
			},
		},
	}
	strategy.Exit = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{Operator: "position_exists"},
			{
				Operator: "gte",
				Args: []ValueExpr{
					{Kind: "position", Position: "holding_bars"},
					{Kind: "value", Value: 2},
				},
			},
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "position", Position: "unrealized_return"},
					{Kind: "value", Value: 0.15},
				},
			},
		},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 11, High: 11, Low: 11, Close: 11, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 14, High: 14, Low: 14, Close: 14, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-03"), result.Trades[0].Time)
	assert.InDelta(t, 11.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-01-05"), result.Trades[1].Time)
	assert.InDelta(t, 14.0, result.Trades[1].Price, 0.0001)
}

func TestEngineSupportsPortfolioDrawdownRule(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Exit = RuleExpr{
		Operator: "lte",
		Args: []ValueExpr{
			{Kind: "portfolio", Portfolio: "portfolio_drawdown"},
			{Kind: "value", Value: -0.10},
		},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 120, High: 120, Low: 120, Close: 120, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 80, High: 80, Low: 80, Close: 80, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-02"), result.Trades[0].Time)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-01-04"), result.Trades[1].Time)
	assert.InDelta(t, 80.0, result.Trades[1].Price, 0.0001)
}

func TestEngineSupportsRebalanceOrderType(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeRebalance
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 50.0, result.Trades[0].Quantity, 0.0001)
	assert.Equal(t, "entry", result.Trades[0].Reason)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 12.0, result.Trades[1].Quantity, 0.0001)
	assert.Equal(t, "rebalance", result.Trades[1].Reason)
	assert.InDelta(t, 15000.0, result.FinalEquity, 0.0001)
}

func TestEngineSupportsTargetWeightChangedRule(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "target_weight_changed",
		Args:     []ValueExpr{{Kind: "value", Value: 5}},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeRebalance
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-02"), result.Trades[0].Time)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-01-03"), result.Trades[1].Time)
	assert.Equal(t, "rebalance", result.Trades[1].Reason)
}

func TestEngineSupportsCalendarRebalanceRules(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{Operator: "monthly"},
			{
				Operator: "target_weight_changed",
				Args:     []ValueExpr{{Kind: "value", Value: 5}},
			},
		},
	}

	run := testRunSpec()
	run.Data.From = "2024-01-31"
	run.Data.To = "2024-02-02"
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeRebalance
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-31"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-02-01"), Symbol: "069500", Open: 200, High: 200, Low: 200, Close: 200, Volume: 1000},
		{Time: date("2024-02-02"), Symbol: "069500", Open: 300, High: 300, Low: 300, Close: 300, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-31"), result.Trades[0].Time)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-02-01"), result.Trades[1].Time)
	assert.Equal(t, "rebalance", result.Trades[1].Reason)
}

func TestRuleSupportsCalendarScheduleRules(t *testing.T) {
	bars := []Bar{
		{Time: date("2024-01-05"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-09"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-02-01"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
	}
	ctx := ruleContext{symbol: "069500", bars: bars, prices: map[string]float64{"069500": 10}, portfolio: newPortfolio(10000)}

	ctx.index = 0
	weekly, err := evaluateRule(RuleExpr{Operator: "weekly"}, ctx)
	require.NoError(t, err)
	assert.True(t, weekly)
	monthly, err := evaluateRule(RuleExpr{Operator: "monthly"}, ctx)
	require.NoError(t, err)
	assert.True(t, monthly)
	firstTradingDay, err := evaluateRule(RuleExpr{Operator: "first_trading_day"}, ctx)
	require.NoError(t, err)
	assert.True(t, firstTradingDay)

	ctx.index = 1
	weekly, err = evaluateRule(RuleExpr{Operator: "weekly"}, ctx)
	require.NoError(t, err)
	assert.True(t, weekly)
	monthly, err = evaluateRule(RuleExpr{Operator: "monthly"}, ctx)
	require.NoError(t, err)
	assert.False(t, monthly)

	ctx.index = 2
	weekly, err = evaluateRule(RuleExpr{Operator: "weekly"}, ctx)
	require.NoError(t, err)
	assert.False(t, weekly)

	ctx.index = 3
	monthly, err = evaluateRule(RuleExpr{Operator: "monthly"}, ctx)
	require.NoError(t, err)
	assert.True(t, monthly)
	firstTradingDay, err = evaluateRule(RuleExpr{Operator: "first_trading_day"}, ctx)
	require.NoError(t, err)
	assert.True(t, firstTradingDay)
}

func TestRuleAndIndicatorsSupportExtendedPriceSources(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	adjustedClose, ok := priceValue(Bar{Close: 11}, "adjusted_close")
	require.True(t, ok)
	assert.InDelta(t, 11.0, adjustedClose, 0.0001)

	strategy := alwaysLongStrategySpec()
	strategy.Indicators = map[string]IndicatorSpec{
		"amount_sma": {
			ID:     "sma",
			Source: ValueExpr{Kind: "price", Price: "amount"},
			Params: map[string]float64{"window": 2},
		},
	}
	strategy.Entry = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "price", Price: "traded_amount"},
					{Kind: "ref", Ref: "amount_sma"},
				},
			},
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "price", Price: "market_cap"},
					{Kind: "value", Value: 900000},
				},
			},
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "price", Price: "nav"},
					{Kind: "value", Value: 100},
				},
			},
			{
				Operator: "gt",
				Args: []ValueExpr{
					{Kind: "price", Price: "adjusted_close"},
					{Kind: "price", Price: "close"},
				},
			},
		},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, AdjustedClose: 10, Volume: 100, TradedAmount: 1000, MarketCap: 1000000, NAV: 110},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 11, High: 11, Low: 11, Close: 11, AdjustedClose: 12, Volume: 200, TradedAmount: 3000, MarketCap: 1200000, NAV: 120},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, AdjustedClose: 13, Volume: 300, TradedAmount: 3000, MarketCap: 1200000, NAV: 120},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 1)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-03"), result.Trades[0].Time)
}

func TestEngineSupportsForNBarsChangedAndNewHighRules(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := alwaysLongStrategySpec()
	strategy.Entry = RuleExpr{
		Operator: "all",
		Rules: []RuleExpr{
			{
				Operator: "for_n_bars",
				Rule: &RuleExpr{
					Operator: "gt",
					Args: []ValueExpr{
						{Kind: "price", Price: "close"},
						{Kind: "value", Value: 10},
					},
				},
				Args: []ValueExpr{{Kind: "value", Value: 2}},
			},
			{
				Operator: "new_high",
				Args: []ValueExpr{
					{Kind: "price", Price: "close"},
					{Kind: "value", Value: 3},
				},
			},
			{
				Operator: "changed",
				Args:     []ValueExpr{{Kind: "price", Price: "close"}},
			},
		},
	}
	strategy.Exit = RuleExpr{
		Operator: "new_low",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 2},
		},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 11, High: 11, Low: 11, Close: 11, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 9, High: 9, Low: 9, Close: 9, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, date("2024-01-04"), result.Trades[0].Time)
	assert.InDelta(t, 12.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, date("2024-01-08"), result.Trades[1].Time)
	assert.InDelta(t, 9.0, result.Trades[1].Price, 0.0001)
}

func TestRuleSupportsBarsSinceAndCooldown(t *testing.T) {
	bars := []Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 9, High: 9, Low: 9, Close: 9, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 11, High: 11, Low: 11, Close: 11, Volume: 1000},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 8, High: 8, Low: 8, Close: 8, Volume: 1000},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 13, High: 13, Low: 13, Close: 13, Volume: 1000},
	}
	ctx := ruleContext{symbol: "069500", bars: bars, prices: map[string]float64{"069500": 13}, portfolio: newPortfolio(10000)}
	child := RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{Kind: "price", Price: "close"},
			{Kind: "value", Value: 10},
		},
	}

	barsSince := RuleExpr{Operator: "bars_since", Rule: &child, Args: []ValueExpr{{Kind: "value", Value: 1}}}
	ctx.index = 3
	matched, err := evaluateRule(barsSince, ctx)
	require.NoError(t, err)
	assert.True(t, matched)

	ctx.index = 4
	matched, err = evaluateRule(barsSince, ctx)
	require.NoError(t, err)
	assert.False(t, matched)

	cooldown := RuleExpr{Operator: "cooldown", Rule: &child, Args: []ValueExpr{{Kind: "value", Value: 1}}}
	ctx.index = 2
	matched, err = evaluateRule(cooldown, ctx)
	require.NoError(t, err)
	assert.False(t, matched)

	ctx.index = 4
	matched, err = evaluateRule(cooldown, ctx)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestEngineSupportsStopExitRules(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	tests := []struct {
		name           string
		exit           RuleExpr
		bars           []Bar
		expectedTime   time.Time
		expectedPrice  float64
		expectedReason string
	}{
		{
			name: "stop_loss",
			exit: RuleExpr{
				Operator: "stop_loss",
				Args:     []ValueExpr{{Kind: "value", Value: 0.10}},
			},
			bars: []Bar{
				{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: date("2024-01-03"), Symbol: "069500", Open: 89, High: 89, Low: 89, Close: 89, Volume: 1000},
			},
			expectedTime:  date("2024-01-03"),
			expectedPrice: 89,
		},
		{
			name: "take_profit",
			exit: RuleExpr{
				Operator: "take_profit",
				Args:     []ValueExpr{{Kind: "value", Value: 0.20}},
			},
			bars: []Bar{
				{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: date("2024-01-03"), Symbol: "069500", Open: 121, High: 121, Low: 121, Close: 121, Volume: 1000},
			},
			expectedTime:  date("2024-01-03"),
			expectedPrice: 121,
		},
		{
			name: "time_stop",
			exit: RuleExpr{
				Operator: "time_stop",
				Args:     []ValueExpr{{Kind: "value", Value: 2}},
			},
			bars: []Bar{
				{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: date("2024-01-03"), Symbol: "069500", Open: 101, High: 101, Low: 101, Close: 101, Volume: 1000},
				{Time: date("2024-01-04"), Symbol: "069500", Open: 102, High: 102, Low: 102, Close: 102, Volume: 1000},
			},
			expectedTime:  date("2024-01-04"),
			expectedPrice: 102,
		},
		{
			name: "trailing_stop",
			exit: RuleExpr{
				Operator: "trailing_stop",
				Args:     []ValueExpr{{Kind: "value", Value: 0.10}},
			},
			bars: []Bar{
				{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: date("2024-01-03"), Symbol: "069500", Open: 120, High: 120, Low: 120, Close: 120, Volume: 1000},
				{Time: date("2024-01-04"), Symbol: "069500", Open: 105, High: 105, Low: 105, Close: 105, Volume: 1000},
			},
			expectedTime:  date("2024-01-04"),
			expectedPrice: 105,
		},
		{
			name: "volatility_stop",
			exit: RuleExpr{
				Operator: "volatility_stop",
				Args: []ValueExpr{
					{
						Kind: "indicator",
						Indicator: &IndicatorSpec{
							ID:     "atr",
							Source: ValueExpr{Kind: "price", Price: "close"},
							Params: map[string]float64{"window": 2},
						},
					},
					{Kind: "value", Value: 1},
				},
			},
			bars: []Bar{
				{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: date("2024-01-03"), Symbol: "069500", Open: 102, High: 104, Low: 100, Close: 102, Volume: 1000},
				{Time: date("2024-01-04"), Symbol: "069500", Open: 92, High: 94, Low: 91, Close: 92, Volume: 1000},
			},
			expectedTime:  date("2024-01-04"),
			expectedPrice: 92,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := alwaysLongStrategySpec()
			strategy.Exit = tt.exit

			run := testRunSpec()
			run.Execution.Fill = FillSameClose
			plan, err := Compile(strategy, run, registry)
			require.NoError(t, err)

			engine, err := NewEngine(NewMemoryFeed(tt.bars))
			require.NoError(t, err)

			result, err := engine.Run(context.Background(), plan)
			require.NoError(t, err)

			require.Len(t, result.Trades, 2)
			assert.Equal(t, SideSell, result.Trades[1].Side)
			assert.Equal(t, tt.expectedTime, result.Trades[1].Time)
			assert.InDelta(t, tt.expectedPrice, result.Trades[1].Price, 0.0001)
			assert.Equal(t, tt.name, result.Trades[1].Reason)
		})
	}
}

func TestEngineSupportsRoleBasedStrategyRules(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := StrategySpec{
		Kind:          KindStrategy,
		SchemaVersion: SchemaVersion,
		Name:          "role-based",
		Entries: []RuleExpr{{
			Operator: "gt",
			Args: []ValueExpr{
				{Kind: "price", Price: "close"},
				{Kind: "value", Value: 0},
			},
		}},
		Exits: []RuleExpr{{
			Operator: "lt",
			Args: []ValueExpr{
				{Kind: "price", Price: "close"},
				{Kind: "value", Value: 0},
			},
		}},
		Stops: []RuleExpr{{
			Operator: "take_profit",
			Args:     []ValueExpr{{Kind: "value", Value: 0.05}},
		}},
		Rebalance: []RuleExpr{{
			Operator: "target_weight_changed",
			Args:     []ValueExpr{{Kind: "value", Value: 1}},
		}},
		Sizing: SizingSpec{Type: SizingPercentOfEquity, Value: 50},
		Risk:   RiskSpec{MaxPositions: 1},
	}
	run := testRunSpec()
	run.Strategy = StrategyRef{Name: "role-based"}
	run.Execution.Fill = FillSameClose
	run.Execution.OrderType = OrderTypeRebalance
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	assert.Equal(t, "gt", plan.Entry.Operator)
	assert.Equal(t, "lt", plan.Exit.Operator)
	require.Len(t, plan.Entries, 1)
	require.Len(t, plan.Exits, 1)
	require.Len(t, plan.Stops, 1)
	require.Len(t, plan.Rebalance, 1)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 106, High: 106, Low: 106, Close: 106, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.Equal(t, "entry", result.Trades[0].Reason)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.Equal(t, "take_profit", result.Trades[1].Reason)
}

func TestValueExpressionsSupportCrossSectionalUniverseValues(t *testing.T) {
	current := date("2024-01-02")
	series := map[string][]Bar{
		"AAA": {{Time: current, Symbol: "AAA", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000}},
		"BBB": {{Time: current, Symbol: "BBB", Open: 120, High: 120, Low: 120, Close: 120, Volume: 1000}},
		"CCC": {{Time: current, Symbol: "CCC", Open: 80, High: 80, Low: 80, Close: 80, Volume: 1000}},
	}
	currentBars := map[string]Bar{}
	currentIndexes := map[string]int{}
	for symbol, bars := range series {
		currentBars[symbol] = bars[0]
		currentIndexes[symbol] = 0
	}
	ctx := ruleContext{
		symbol:         "AAA",
		index:          0,
		bars:           series["AAA"],
		series:         series,
		activeSymbols:  []string{"AAA", "BBB", "CCC"},
		currentBars:    currentBars,
		currentIndexes: currentIndexes,
		prices:         closePrices(currentBars),
	}

	rank, ok, err := currentValue(ValueExpr{Kind: "rank", Args: []ValueExpr{{Kind: "price", Price: "close"}}}, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 2.0, rank, 0.0001)

	percentile, ok, err := currentValue(ValueExpr{Kind: "percentile", Args: []ValueExpr{{Kind: "price", Price: "close"}}}, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 50.0, percentile, 0.0001)

	strength, ok, err := currentValue(ValueExpr{Kind: "relative_strength", Args: []ValueExpr{{Kind: "price", Price: "close"}}}, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 0.0, strength, 0.0001)

	spread, ok, err := currentValue(ValueExpr{Kind: "spread", Args: []ValueExpr{{Kind: "price", Price: "close"}, {Kind: "value", Value: 90}}}, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 10.0, spread, 0.0001)

	ratio, ok, err := currentValue(ValueExpr{Kind: "ratio", Args: []ValueExpr{{Kind: "price", Price: "close"}, {Kind: "value", Value: 50}}}, ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 2.0, ratio, 0.0001)
}

func TestEngineSupportsCrossSectionalValueExpressions(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := StrategySpec{
		Kind:          KindStrategy,
		SchemaVersion: SchemaVersion,
		Name:          "cross-sectional",
		Entries: []RuleExpr{{
			Operator: "lte",
			Args: []ValueExpr{
				{Kind: "rank", Args: []ValueExpr{{Kind: "price", Price: "close"}}},
				{Kind: "value", Value: 2},
			},
		}},
		Exits: []RuleExpr{{
			Operator: "lt",
			Args: []ValueExpr{
				{Kind: "price", Price: "close"},
				{Kind: "value", Value: 0},
			},
		}},
		Sizing: SizingSpec{Type: SizingPercentOfEquity, Value: 10},
		Risk:   RiskSpec{MaxPositions: 10},
	}
	run := testRunSpec()
	run.Strategy = StrategyRef{Name: "cross-sectional"}
	run.Universe = UniverseSpec{Symbols: []string{"AAA", "BBB", "CCC"}}
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "AAA", Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
		{Time: date("2024-01-02"), Symbol: "BBB", Open: 120, High: 120, Low: 120, Close: 120, Volume: 1000},
		{Time: date("2024-01-02"), Symbol: "CCC", Open: 80, High: 80, Low: 80, Close: 80, Volume: 1000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, "AAA", result.Trades[0].Symbol)
	assert.Equal(t, "BBB", result.Trades[1].Symbol)
	assert.Equal(t, []string{"AAA", "BBB", "CCC"}, result.Symbols)
}

func TestEngineEquityCurveMatchesIndependentTradeReplay(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := Compile(testStrategySpec(), testRunSpec(), registry)
	require.NoError(t, err)

	bars := testBars()
	engine, err := NewEngine(NewMemoryFeed(bars))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	assertEquityCurveMatchesReplay(t, result, bars)
}

func TestCompileDefaultsToCoreMetrics(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	plan, err := Compile(testStrategySpec(), testRunSpec(), registry)
	require.NoError(t, err)

	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "trade_count", "win_rate"}, plan.SelectedMetrics)
}

func TestCompileAcceptsIntradayReadyAndResampledTimeframes(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	for _, timeframe := range []string{"1m", "5m", "15m", "30m", "1h", "1d", "1w", "1mo", "custom"} {
		run := testRunSpec()
		run.Data.Timeframe = timeframe
		plan, err := Compile(testStrategySpec(), run, registry)
		require.NoError(t, err)
		assert.Equal(t, timeframe, plan.Timeframe)
		assert.Equal(t, timeframe, plan.DataRequest().Timeframe)
	}
}

func TestMemoryFeedResamplesDailyBarsToWeeklyFrames(t *testing.T) {
	stream, err := NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 11, Low: 9, Close: 10, Volume: 100, TradedAmount: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 13, Low: 11, Close: 12, Volume: 200, TradedAmount: 2000},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 14, High: 15, Low: 13, Close: 14, Volume: 300, TradedAmount: 3000},
	}).Open(context.Background(), DataRequest{
		Symbols:   []string{"069500"},
		From:      date("2024-01-02"),
		To:        date("2024-01-08"),
		Timeframe: Timeframe1Week,
	})
	require.NoError(t, err)
	defer stream.Close()

	first, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, date("2024-01-03"), first.Time)
	assert.Equal(t, date("2024-01-03"), first.Bars["069500"].Time)
	assert.Equal(t, Timeframe1Week, first.Bars["069500"].Timeframe)
	assert.Equal(t, BarSessionRegular, first.Bars["069500"].Session)
	assert.Equal(t, BarStatusOK, first.Bars["069500"].Status)
	assert.InDelta(t, 10.0, first.Bars["069500"].Open, 0.0001)
	assert.InDelta(t, 13.0, first.Bars["069500"].High, 0.0001)
	assert.InDelta(t, 9.0, first.Bars["069500"].Low, 0.0001)
	assert.InDelta(t, 12.0, first.Bars["069500"].Close, 0.0001)
	assert.InDelta(t, 300.0, first.Bars["069500"].Volume, 0.0001)
	assert.InDelta(t, 3000.0, first.Bars["069500"].TradedAmount, 0.0001)

	second, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, date("2024-01-08"), second.Time)
	assert.Equal(t, Timeframe1Week, second.Bars["069500"].Timeframe)
	assert.InDelta(t, 14.0, second.Bars["069500"].Open, 0.0001)
	assert.InDelta(t, 14.0, second.Bars["069500"].Close, 0.0001)

	_, ok, err = stream.Next(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMemoryFeedAnnotatesIntradayReadyBarMetadata(t *testing.T) {
	stream, err := NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 0, High: 0, Low: 0, Close: 10},
	}).Open(context.Background(), DataRequest{
		Symbols:   []string{"069500"},
		From:      date("2024-01-02"),
		To:        date("2024-01-02"),
		Timeframe: Timeframe5Min,
	})
	require.NoError(t, err)
	defer stream.Close()

	frame, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)

	bar := frame.Bars["069500"]
	assert.Equal(t, Timeframe5Min, bar.Timeframe)
	assert.Equal(t, BarSessionRegular, bar.Session)
	assert.Equal(t, BarStatusNoTrade, bar.Status)
}

func TestEngineResultExplainsDailyResampleTimeframeMetadata(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	run := testRunSpec()
	run.Data.Timeframe = Timeframe1Week
	plan, err := Compile(alwaysLongStrategySpec(), run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 11, Low: 9, Close: 10, Volume: 100, TradedAmount: 1000},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 13, Low: 11, Close: 12, Volume: 200, TradedAmount: 2000},
		{Time: date("2024-01-08"), Symbol: "069500", Open: 14, High: 15, Low: 13, Close: 14, Volume: 300, TradedAmount: 3000},
		{Time: date("2024-01-09"), Symbol: "069500", Open: 16, High: 17, Low: 15, Close: 16, Volume: 400, TradedAmount: 4000},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	assert.Equal(t, Timeframe1Week, result.Timeframe)
	assert.Equal(t, Timeframe1Week, result.Timeframes.Requested)
	assert.Equal(t, Timeframe1Day, result.Timeframes.Source)
	assert.Equal(t, Timeframe1Week, result.Timeframes.Execution)
	assert.True(t, result.Timeframes.Resample.Enabled)
	assert.Equal(t, Timeframe1Day, result.Timeframes.Resample.Source)
	assert.Equal(t, Timeframe1Week, result.Timeframes.Resample.Target)
	assert.Equal(t, "ohlcv_last_close_sum_volume", result.Timeframes.Resample.Policy)
	assert.Equal(t, "iso_week", result.Timeframes.Resample.Boundary)
	assert.Equal(t, 0, result.Timeframes.Warmup.Bars)
	assert.Equal(t, "indicator_lookback_bars", result.Timeframes.Warmup.Policy)
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
		Include: []string{"benchmark_total_return", "excess_return", "benchmark_max_drawdown", "relative_drawdown", "monthly_win_rate_vs_benchmark", "benchmark_alpha", "benchmark_beta"},
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
	assert.Contains(t, result.Metrics, "benchmark_alpha")
	assert.Contains(t, result.Metrics, "benchmark_beta")
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
	assertExecutionEventTypes(t, result.ExecutionEvents, "order_intent", "deferred_no_trade_bar", "fill", "portfolio_mutation")
	assert.Equal(t, "execution", result.ExecutionEvents[1].Layer)
	assert.Equal(t, "069500", result.ExecutionEvents[1].Symbol)
	assert.Equal(t, SideBuy, result.ExecutionEvents[1].Side)
	assert.Equal(t, "no_trade_bar", result.ExecutionEvents[1].Reason)
	require.Len(t, result.DataEvents, 1)
	assert.Equal(t, "data", result.DataEvents[0].Layer)
	assert.Equal(t, "data_issue", result.DataEvents[0].Type)
	assert.Equal(t, "no_trade_bar", result.DataEvents[0].Reason)
	assert.Equal(t, Timeframe1Day, result.DataEvents[0].Timeframe)
	assert.Equal(t, BarSessionRegular, result.DataEvents[0].Session)
	assert.Equal(t, BarStatusNoTrade, result.DataEvents[0].Status)
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

func TestEngineEvaluatesArithmeticValueExpressions(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	strategy := testStrategySpec()
	strategy.Indicators = nil
	strategy.Entry = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{
				Kind: "div",
				Args: []ValueExpr{
					{
						Kind: "sub",
						Args: []ValueExpr{
							{Kind: "price", Price: "close"},
							{Kind: "value", Value: 10},
						},
					},
					{Kind: "value", Value: 10},
				},
			},
			{Kind: "value", Value: 0.1},
		},
	}
	strategy.Exit = RuleExpr{
		Operator: "gt",
		Args: []ValueExpr{
			{
				Kind: "abs",
				Args: []ValueExpr{
					{
						Kind: "sub",
						Args: []ValueExpr{
							{Kind: "price", Price: "close"},
							{Kind: "value", Value: 13},
						},
					},
				},
			},
			{
				Kind: "max",
				Args: []ValueExpr{
					{Kind: "value", Value: 1},
					{
						Kind: "min",
						Args: []ValueExpr{
							{Kind: "value", Value: 3},
							{Kind: "value", Value: 2},
						},
					},
				},
			},
		},
	}

	run := testRunSpec()
	run.Execution.Fill = FillSameClose
	plan, err := Compile(strategy, run, registry)
	require.NoError(t, err)

	engine, err := NewEngine(NewMemoryFeed([]Bar{
		{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
		{Time: date("2024-01-03"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12},
		{Time: date("2024-01-04"), Symbol: "069500", Open: 13, High: 13, Low: 13, Close: 13},
		{Time: date("2024-01-05"), Symbol: "069500", Open: 16, High: 16, Low: 16, Close: 16},
	}))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, SideBuy, result.Trades[0].Side)
	assert.InDelta(t, 12.0, result.Trades[0].Price, 0.0001)
	assert.Equal(t, SideSell, result.Trades[1].Side)
	assert.InDelta(t, 16.0, result.Trades[1].Price, 0.0001)
}

func TestDefaultIndicatorRegistrySupportsExpandedTrendAndBandIndicators(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	bars := []Bar{{Close: 10}, {Close: 12}, {Close: 14}, {Close: 16}}

	ema, ok := registry.Definition("ema")
	require.True(t, ok)
	emaValues, err := ema.Calculate(IndicatorSpec{
		ID:     "ema",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3},
	}, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(emaValues[0]))
	assert.True(t, math.IsNaN(emaValues[1]))
	assert.InDelta(t, 12.0, emaValues[2], 0.0001)
	assert.InDelta(t, 14.0, emaValues[3], 0.0001)

	wma, ok := registry.Definition("wma")
	require.True(t, ok)
	wmaValues, err := wma.Calculate(IndicatorSpec{
		ID:     "wma",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3},
	}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 12.6666, wmaValues[2], 0.001)
	assert.InDelta(t, 14.6666, wmaValues[3], 0.001)

	hma, ok := registry.Definition("hma")
	require.True(t, ok)
	hmaValues, err := hma.Calculate(IndicatorSpec{
		ID:     "hma",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 4},
	}, []Bar{{Close: 1}, {Close: 2}, {Close: 3}, {Close: 4}, {Close: 5}, {Close: 6}})
	require.NoError(t, err)
	assert.True(t, math.IsNaN(hmaValues[3]))
	assert.InDelta(t, 5.0, hmaValues[4], 0.0001)
	assert.InDelta(t, 6.0, hmaValues[5], 0.0001)

	kama, ok := registry.Definition("kama")
	require.True(t, ok)
	kamaValues, err := kama.Calculate(IndicatorSpec{
		ID:     "kama",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3, "fast_window": 2, "slow_window": 5},
	}, []Bar{{Close: 1}, {Close: 2}, {Close: 3}, {Close: 4}, {Close: 5}, {Close: 6}})
	require.NoError(t, err)
	assert.True(t, math.IsNaN(kamaValues[0]))
	assert.True(t, math.IsNaN(kamaValues[1]))
	assert.InDelta(t, 2.0, kamaValues[2], 0.0001)
	assert.InDelta(t, 2.8889, kamaValues[3], 0.001)
	assert.InDelta(t, 3.8272, kamaValues[4], 0.001)
	assert.InDelta(t, 4.7929, kamaValues[5], 0.001)

	roc, ok := registry.Definition("roc")
	require.True(t, ok)
	rocValues, err := roc.Calculate(IndicatorSpec{
		ID:     "roc",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 2},
	}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 40.0, rocValues[2], 0.0001)
	assert.InDelta(t, 33.3333, rocValues[3], 0.001)

	upper, ok := registry.Definition("bollinger_upper")
	require.True(t, ok)
	upperValues, err := upper.Calculate(IndicatorSpec{
		ID:     "bollinger_upper",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3, "multiplier": 2},
	}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 15.266, upperValues[2], 0.001)

	middle, ok := registry.Definition("bollinger_middle")
	require.True(t, ok)
	middleValues, err := middle.Calculate(IndicatorSpec{
		ID:     "bollinger_middle",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3, "multiplier": 2},
	}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 12.0, middleValues[2], 0.0001)

	lower, ok := registry.Definition("bollinger_lower")
	require.True(t, ok)
	lowerValues, err := lower.Calculate(IndicatorSpec{
		ID:     "bollinger_lower",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3, "multiplier": 2},
	}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 8.734, lowerValues[2], 0.001)
}

func TestDefaultIndicatorRegistrySupportsKeltnerChannels(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	bars := []Bar{
		{High: 11, Low: 9, Close: 10},
		{High: 13, Low: 11, Close: 12},
		{High: 15, Low: 13, Close: 14},
		{High: 17, Low: 15, Close: 16},
	}
	spec := IndicatorSpec{
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3, "atr_window": 3, "multiplier": 1},
	}

	middle, ok := registry.Definition("keltner_middle")
	require.True(t, ok)
	middleSpec := spec
	middleSpec.ID = "keltner_middle"
	middleValues, err := middle.Calculate(middleSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(middleValues[1]))
	assert.InDelta(t, 12.0, middleValues[2], 0.0001)
	assert.InDelta(t, 14.0, middleValues[3], 0.0001)

	upper, ok := registry.Definition("keltner_upper")
	require.True(t, ok)
	upperSpec := spec
	upperSpec.ID = "keltner_upper"
	upperValues, err := upper.Calculate(upperSpec, bars)
	require.NoError(t, err)
	assert.InDelta(t, 14.6666, upperValues[2], 0.001)
	assert.InDelta(t, 17.0, upperValues[3], 0.0001)

	lower, ok := registry.Definition("keltner_lower")
	require.True(t, ok)
	lowerSpec := spec
	lowerSpec.ID = "keltner_lower"
	lowerValues, err := lower.Calculate(lowerSpec, bars)
	require.NoError(t, err)
	assert.InDelta(t, 9.3333, lowerValues[2], 0.001)
	assert.InDelta(t, 11.0, lowerValues[3], 0.0001)
}

func TestDefaultIndicatorRegistrySupportsMACDOutputs(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)
	macd, ok := registry.Definition("macd")
	require.True(t, ok)

	bars := []Bar{
		{Close: 10},
		{Close: 12},
		{Close: 15},
		{Close: 19},
		{Close: 24},
		{Close: 30},
	}
	base := IndicatorSpec{
		ID:     "macd",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"fast_window": 2, "slow_window": 3, "signal_window": 2},
	}
	lineValues, err := macd.Calculate(base, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(lineValues[1]))
	assert.InDelta(t, 1.3333, lineValues[2], 0.001)
	assert.InDelta(t, 2.3302, lineValues[5], 0.001)

	signalSpec := base
	signalSpec.Output = "signal"
	signalValues, err := macd.Calculate(signalSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(signalValues[2]))
	assert.InDelta(t, 1.4444, signalValues[3], 0.001)
	assert.InDelta(t, 2.1379, signalValues[5], 0.001)

	histogramSpec := base
	histogramSpec.Output = "histogram"
	histogramValues, err := macd.Calculate(histogramSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(histogramValues[2]))
	assert.InDelta(t, 0.1111, histogramValues[3], 0.001)
	assert.InDelta(t, 0.1924, histogramValues[5], 0.001)
}

func TestDefaultIndicatorRegistrySupportsStochasticOutputs(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)
	stochastic, ok := registry.Definition("stochastic")
	require.True(t, ok)

	bars := []Bar{
		{High: 11, Low: 9, Close: 10},
		{High: 13, Low: 10, Close: 12},
		{High: 15, Low: 11, Close: 14},
		{High: 16, Low: 12, Close: 13},
		{High: 18, Low: 13, Close: 17},
	}
	base := IndicatorSpec{
		ID:     "stochastic",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"k_window": 3, "d_window": 2},
	}

	kValues, err := stochastic.Calculate(base, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(kValues[1]))
	assert.InDelta(t, 83.3333, kValues[2], 0.001)
	assert.InDelta(t, 50.0, kValues[3], 0.0001)
	assert.InDelta(t, 85.7142, kValues[4], 0.001)

	dSpec := base
	dSpec.Output = "d"
	dValues, err := stochastic.Calculate(dSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(dValues[2]))
	assert.InDelta(t, 66.6666, dValues[3], 0.001)
	assert.InDelta(t, 67.8571, dValues[4], 0.001)
}

func TestDefaultIndicatorRegistrySupportsADXAndDirectionalIndicators(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)
	adx, ok := registry.Definition("adx")
	require.True(t, ok)

	bars := []Bar{
		{High: 10, Low: 9, Close: 9.5},
		{High: 12, Low: 10, Close: 11},
		{High: 14, Low: 11, Close: 13},
		{High: 16, Low: 12, Close: 15},
		{High: 18, Low: 13, Close: 17},
		{High: 20, Low: 14, Close: 19},
	}
	base := IndicatorSpec{ID: "adx", Params: map[string]float64{"window": 3}}

	adxValues, err := adx.Calculate(base, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(adxValues[4]))
	assert.InDelta(t, 100.0, adxValues[5], 0.0001)

	diPlusSpec := base
	diPlusSpec.Output = "di_plus"
	diPlusValues, err := adx.Calculate(diPlusSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(diPlusValues[2]))
	assert.InDelta(t, 63.1579, diPlusValues[3], 0.001)
	assert.InDelta(t, 52.9412, diPlusValues[4], 0.001)
	assert.InDelta(t, 44.2623, diPlusValues[5], 0.001)

	diMinus, ok := registry.Definition("di_minus")
	require.True(t, ok)
	diMinusValues, err := diMinus.Calculate(IndicatorSpec{ID: "di_minus", Params: map[string]float64{"window": 3}}, bars)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, diMinusValues[3], 0.0001)
	assert.InDelta(t, 0.0, diMinusValues[5], 0.0001)
}

func TestDefaultIndicatorRegistrySupportsZScore(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)
	zscore, ok := registry.Definition("zscore")
	require.True(t, ok)

	values, err := zscore.Calculate(IndicatorSpec{
		ID:     "zscore",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3},
	}, []Bar{{Close: 10}, {Close: 12}, {Close: 14}, {Close: 16}})
	require.NoError(t, err)
	assert.True(t, math.IsNaN(values[0]))
	assert.True(t, math.IsNaN(values[1]))
	assert.InDelta(t, 1.2247, values[2], 0.001)
	assert.InDelta(t, 1.2247, values[3], 0.001)

	flatValues, err := zscore.Calculate(IndicatorSpec{
		ID:     "zscore",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": 3},
	}, []Bar{{Close: 10}, {Close: 10}, {Close: 10}})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, flatValues[2], 0.0001)
}

func TestDefaultIndicatorRegistrySupportsCorrelationAndBeta(t *testing.T) {
	registry, err := DefaultIndicatorRegistry()
	require.NoError(t, err)

	bars := []Bar{
		{Close: 100, NAV: 100},
		{Close: 110, NAV: 105},
		{Close: 132, NAV: 115.5},
		{Close: 171.6, NAV: 132.825},
	}
	spec := IndicatorSpec{
		Source:  ValueExpr{Kind: "price", Price: "close"},
		Compare: ValueExpr{Kind: "price", Price: "nav"},
		Params:  map[string]float64{"window": 2},
	}

	correlation, ok := registry.Definition("correlation")
	require.True(t, ok)
	correlationSpec := spec
	correlationSpec.ID = "correlation"
	correlationValues, err := correlation.Calculate(correlationSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(correlationValues[0]))
	assert.True(t, math.IsNaN(correlationValues[1]))
	assert.InDelta(t, 1.0, correlationValues[2], 0.0001)
	assert.InDelta(t, 1.0, correlationValues[3], 0.0001)

	beta, ok := registry.Definition("beta")
	require.True(t, ok)
	betaSpec := spec
	betaSpec.ID = "beta"
	betaValues, err := beta.Calculate(betaSpec, bars)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(betaValues[0]))
	assert.True(t, math.IsNaN(betaValues[1]))
	assert.InDelta(t, 2.0, betaValues[2], 0.0001)
	assert.InDelta(t, 2.0, betaValues[3], 0.0001)
}

func TestExpandedIndicatorsHaveStreamingRuntimeParity(t *testing.T) {
	for _, spec := range []IndicatorSpec{
		{ID: "ema", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3}},
		{ID: "wma", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3}},
		{ID: "hma", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 4}},
		{ID: "kama", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3, "fast_window": 2, "slow_window": 5}},
		{ID: "roc", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 2}},
		{ID: "bollinger_upper", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3, "multiplier": 2}},
		{ID: "macd", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"fast_window": 2, "slow_window": 3, "signal_window": 2}},
		{ID: "macd", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"fast_window": 2, "slow_window": 3, "signal_window": 2}, Output: "signal"},
		{ID: "macd", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"fast_window": 2, "slow_window": 3, "signal_window": 2}, Output: "histogram"},
		{ID: "keltner_middle", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3, "atr_window": 3, "multiplier": 1}},
		{ID: "keltner_upper", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3, "atr_window": 3, "multiplier": 1}},
		{ID: "keltner_lower", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3, "atr_window": 3, "multiplier": 1}},
		{ID: "stochastic", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"k_window": 3, "d_window": 2}},
		{ID: "stochastic", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"k_window": 3, "d_window": 2}, Output: "d"},
		{ID: "adx", Params: map[string]float64{"window": 3}},
		{ID: "adx", Params: map[string]float64{"window": 3}, Output: "di_plus"},
		{ID: "di_minus", Params: map[string]float64{"window": 3}},
		{ID: "zscore", Source: ValueExpr{Kind: "price", Price: "close"}, Params: map[string]float64{"window": 3}},
		{ID: "correlation", Source: ValueExpr{Kind: "price", Price: "close"}, Compare: ValueExpr{Kind: "price", Price: "nav"}, Params: map[string]float64{"window": 2}},
		{ID: "beta", Source: ValueExpr{Kind: "price", Price: "close"}, Compare: ValueExpr{Kind: "price", Price: "nav"}, Params: map[string]float64{"window": 2}},
	} {
		t.Run(spec.ID, func(t *testing.T) {
			registry, err := DefaultIndicatorRegistry()
			require.NoError(t, err)
			definition, ok := registry.Definition(spec.ID)
			require.True(t, ok)
			bars := []Bar{
				{High: 11, Low: 9, Close: 10, NAV: 10},
				{High: 13, Low: 10, Close: 12, NAV: 11},
				{High: 15, Low: 11, Close: 14, NAV: 13},
				{High: 16, Low: 12, Close: 13, NAV: 12},
				{High: 18, Low: 13, Close: 17, NAV: 16},
				{High: 20, Low: 14, Close: 19, NAV: 17},
			}
			batch, err := definition.Calculate(spec, bars)
			require.NoError(t, err)
			runtime, err := newIndicatorRuntime(spec)
			require.NoError(t, err)
			streaming := make([]float64, 0, len(bars))
			for _, bar := range bars {
				value, err := runtime.Add(bar)
				require.NoError(t, err)
				streaming = append(streaming, value)
			}
			assertFloatSeriesEqual(t, batch, streaming)
		})
	}
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

func alwaysLongStrategySpec() StrategySpec {
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
	return strategy
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

type scriptedFeed struct {
	frames []BarFrame
}

func (f scriptedFeed) Open(context.Context, DataRequest) (BarStream, error) {
	return &scriptedStream{frames: append([]BarFrame(nil), f.frames...)}, nil
}

type scriptedStream struct {
	frames []BarFrame
	offset int
}

func (s *scriptedStream) Next(context.Context) (BarFrame, bool, error) {
	if s.offset >= len(s.frames) {
		return BarFrame{}, false, nil
	}
	frame := s.frames[s.offset]
	s.offset++
	return frame, true, nil
}

func (s *scriptedStream) Close() error {
	return nil
}

func assertEquityCurveMatchesReplay(t *testing.T, result Result, bars []Bar) {
	t.Helper()

	require.NotEmpty(t, result.EquityCurve)
	assert.InDelta(t, result.FinalEquity, result.EquityCurve[len(result.EquityCurve)-1].Equity, 0.0001)

	barsByTime := map[time.Time]map[string]Bar{}
	for _, bar := range bars {
		if barsByTime[bar.Time] == nil {
			barsByTime[bar.Time] = map[string]Bar{}
		}
		barsByTime[bar.Time][bar.Symbol] = bar
	}

	tradesByTime := map[time.Time][]Trade{}
	for _, trade := range result.Trades {
		require.Greater(t, trade.Price, 0.0)
		require.Greater(t, trade.Quantity, 0.0)
		require.Greater(t, trade.Notional, 0.0)
		assert.InDelta(t, trade.Price*trade.Quantity, trade.Notional, 0.0001)
		assert.GreaterOrEqual(t, trade.Commission, 0.0)
		assert.GreaterOrEqual(t, trade.Tax, 0.0)
		assert.GreaterOrEqual(t, trade.ExchangeFee, 0.0)
		assert.GreaterOrEqual(t, trade.totalCost(), 0.0)
		tradesByTime[trade.Time] = append(tradesByTime[trade.Time], trade)
	}

	cash := result.InitialCash
	positions := map[string]float64{}
	var lastTime time.Time
	appliedTrades := 0
	for index, point := range result.EquityCurve {
		if index > 0 {
			require.True(t, point.Time.After(lastTime), "equity curve time must be ascending")
		}
		lastTime = point.Time

		for _, trade := range tradesByTime[point.Time] {
			switch trade.Side {
			case SideBuy:
				cash -= trade.Notional + trade.totalCost()
				positions[trade.Symbol] += trade.Quantity
			case SideSell:
				require.GreaterOrEqual(t, positions[trade.Symbol]+1e-9, trade.Quantity)
				cash += trade.Notional - trade.totalCost()
				positions[trade.Symbol] -= trade.Quantity
				if positions[trade.Symbol] <= 1e-9 {
					delete(positions, trade.Symbol)
				}
			default:
				t.Fatalf("unsupported trade side: %s", trade.Side)
			}
			appliedTrades++
		}

		var positionsValue float64
		for symbol, quantity := range positions {
			bySymbol, ok := barsByTime[point.Time]
			require.True(t, ok, "missing bars for equity point %s", point.Time.Format(time.DateOnly))
			bar, ok := bySymbol[symbol]
			require.True(t, ok, "missing close price for %s at %s", symbol, point.Time.Format(time.DateOnly))
			positionsValue += quantity * bar.Close
		}

		assert.InDelta(t, cash, point.Cash, 0.0001)
		assert.InDelta(t, positionsValue, point.PositionsValue, 0.0001)
		assert.InDelta(t, cash+positionsValue, point.Equity, 0.0001)
	}
	assert.Equal(t, len(result.Trades), appliedTrades)
}

func assertExecutionEventTypes(t *testing.T, events []Event, expected ...string) {
	t.Helper()

	actual := make([]string, 0, len(events))
	for _, event := range events {
		actual = append(actual, event.Type)
	}
	assert.Equal(t, expected, actual)
}

func assertFloatSeriesEqual(t *testing.T, expected []float64, actual []float64) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for i := range expected {
		if math.IsNaN(expected[i]) {
			assert.True(t, math.IsNaN(actual[i]), "index %d", i)
			continue
		}
		assert.InDelta(t, expected[i], actual[i], 0.0001, "index %d", i)
	}
}

func date(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
