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
	assert.Equal(t, result.TotalReturn, result.Metrics.TotalReturn)
	assert.NotEmpty(t, result.ResultHash)

	repeated, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)
	assert.Equal(t, result.ResultHash, repeated.ResultHash)
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

func date(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
