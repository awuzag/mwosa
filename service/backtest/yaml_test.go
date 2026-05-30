package backtest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeYAMLStreamLoadsStrategyAndBacktestRun(t *testing.T) {
	bundle, err := Decode(context.Background(), strings.NewReader(sampleYAML()))
	require.NoError(t, err)

	assert.Equal(t, core.KindStrategy, bundle.Strategy.Kind)
	assert.Equal(t, "sma-cross", bundle.Strategy.Name)
	assert.Equal(t, "sma", bundle.Strategy.Indicators["trend"].ID)
	assert.Equal(t, "crosses_above", bundle.Strategy.Entry.Operator)
	assert.Equal(t, core.KindBacktestRun, bundle.Run.Kind)
	assert.Equal(t, "sma-cross-run", bundle.Run.Name)
	assert.Equal(t, []string{"069500"}, bundle.Run.Universe.Symbols)
	assert.Equal(t, "core", bundle.Run.Report.Metrics.Preset)
	assert.Equal(t, []string{"trade_count"}, bundle.Run.Report.Metrics.Exclude)
}

func TestDecodeYAMLStreamLoadsTimeframeQualifiedValueExpression(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `    - price: close
    - ref: trend`, `    - timeframe:
        id: 1w
        value:
          price: close
    - ref: trend`, 1)

	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	require.Len(t, bundle.Strategy.Entry.Args, 2)
	assert.Equal(t, "price", bundle.Strategy.Entry.Args[0].Kind)
	assert.Equal(t, "close", bundle.Strategy.Entry.Args[0].Price)
	assert.Equal(t, core.Timeframe1Week, bundle.Strategy.Entry.Args[0].Timeframe)
}

func TestDecodeYAMLStreamLoadsEvaluationSearch(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-random-search
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: expanding
  from: 2024-01-02
  to: 2024-01-08
  window:
    days: 2
  step:
    days: 1
search:
  mode: random
  seed: 7
  samples: 4
  parameters:
    indicators.trend.params.window:
      min: 2
      max: 5
      step: 1
    sizing.value:
      values: [25, 50]
metrics:
  preset: research
constraints:
  max_exposure_lte: 0.75
  max_unfilled_count_lte: 2
  max_data_issue_count_lte: 0
ranking:
  objective: weighted_score
  order: desc
  weights:
    cagr: 1
    turnover: -0.1`, 1)

	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	assert.Equal(t, core.KindEvaluation, bundle.Evaluation.Kind)
	assert.Equal(t, core.EvaluationPeriodExpanding, bundle.Evaluation.Periods.Mode)
	assert.Equal(t, core.SearchModeRandom, bundle.Evaluation.Search.Mode)
	assert.Equal(t, int64(7), bundle.Evaluation.Search.Seed)
	assert.Equal(t, 4, bundle.Evaluation.Search.Samples)
	require.Contains(t, bundle.Evaluation.Search.Parameters, "sizing.value")
	assert.Equal(t, []any{25, 50}, bundle.Evaluation.Search.Parameters["sizing.value"].Values)
	assert.Equal(t, core.EvaluationObjectiveWeightedScore, bundle.Evaluation.Ranking.Objective)
	assert.Equal(t, map[string]float64{core.MetricCAGR: 1, core.MetricTurnover: -0.1}, bundle.Evaluation.Ranking.Weights)
	require.NotNil(t, bundle.Evaluation.Constraints.MaxExposureLTE)
	require.NotNil(t, bundle.Evaluation.Constraints.MaxUnfilledCountLTE)
	require.NotNil(t, bundle.Evaluation.Constraints.MaxDataIssueCountLTE)
	assert.InDelta(t, 0.75, *bundle.Evaluation.Constraints.MaxExposureLTE, 0.0001)
	assert.InDelta(t, 2.0, *bundle.Evaluation.Constraints.MaxUnfilledCountLTE, 0.0001)
	assert.InDelta(t, 0.0, *bundle.Evaluation.Constraints.MaxDataIssueCountLTE, 0.0001)
}

func TestDecodeYAMLStreamLoadsBayesianSearchExtensionFields(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-bayesian-search
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-08
search:
  mode: bayesian
  seed: 11
  samples: 6
  initial_samples: 3
  acquisition: expected_improvement
  parameters:
    indicators.trend.params.window:
      min: 2
      max: 8
      step: 1
metrics:
  preset: research`, 1)

	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	assert.Equal(t, core.SearchModeBayesian, bundle.Evaluation.Search.Mode)
	assert.Equal(t, int64(11), bundle.Evaluation.Search.Seed)
	assert.Equal(t, 6, bundle.Evaluation.Search.Samples)
	assert.Equal(t, 3, bundle.Evaluation.Search.InitialSamples)
	assert.Equal(t, "expected_improvement", bundle.Evaluation.Search.Acquisition)
	require.Contains(t, bundle.Evaluation.Search.Parameters, "indicators.trend.params.window")
}

func TestDecodeYAMLStreamLoadsWalkForwardPeriodMode(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-walk-forward-only
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: walk_forward
  from: 2024-01-02
  to: 2024-01-08
parameters:
  indicators.trend.params.window: [2, 3]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc
walk_forward:
  train:
    days: 3
  test:
    days: 1
  step:
    days: 1
  select:
    objective: calmar`, 1)

	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	assert.Equal(t, core.EvaluationPeriodWalkForward, bundle.Evaluation.Periods.Mode)
	assert.Equal(t, 3, bundle.Evaluation.WalkForward.Train.Days)
	assert.Equal(t, 1, bundle.Evaluation.WalkForward.Test.Days)
	assert.Equal(t, core.MetricCalmar, bundle.Evaluation.WalkForward.Select.Objective)
}

func TestDecodeYAMLStreamLoadsEvaluationRegimeThresholds(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: regime-threshold-evaluation
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-08
metrics:
  preset: research
regime:
  return_threshold: 0.02
  volatility_threshold: 0.15`, 1)

	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	assert.InDelta(t, 0.02, bundle.Evaluation.Regime.ReturnThreshold, 0.0001)
	assert.InDelta(t, 0.15, bundle.Evaluation.Regime.VolatilityThreshold, 0.0001)
}

func TestDecodeYAMLStreamSupportsRebalanceOrderType(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: rebalance", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.OrderTypeRebalance, plan.OrderType)
}

func TestDecodeYAMLStreamSupportsLimitOrderType(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: limit\n  limit_price: 10", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.InDelta(t, 10.0, bundle.Run.Execution.LimitPrice, 0.0001)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.OrderTypeLimit, plan.OrderType)
	assert.InDelta(t, 10.0, plan.LimitPrice, 0.0001)
}

func TestDecodeYAMLStreamSupportsGTCTimeInForce(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: limit\n  limit_price: 10\n  time_in_force: gtc", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, core.TimeInForceGTC, bundle.Run.Execution.TimeInForce)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.TimeInForceGTC, plan.TimeInForce)
}

func TestDecodeYAMLStreamSupportsCancelOnRebalanceTimeInForce(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: rebalance\n  time_in_force: cancel_on_rebalance", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, core.TimeInForceCancelOnRebalance, bundle.Run.Execution.TimeInForce)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.TimeInForceCancelOnRebalance, plan.TimeInForce)
	assert.Equal(t, core.OrderTypeRebalance, plan.OrderType)
}

func TestDecodeYAMLStreamSupportsStopLimitOrderType(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: stop_limit\n  stop_price: 12\n  limit_price: 11\n  intrabar_ambiguity_policy: optimistic", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.InDelta(t, 12.0, bundle.Run.Execution.StopPrice, 0.0001)
	assert.Equal(t, core.IntrabarAmbiguityOptimistic, bundle.Run.Execution.IntrabarAmbiguityPolicy)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.OrderTypeStopLimit, plan.OrderType)
	assert.InDelta(t, 12.0, plan.StopPrice, 0.0001)
	assert.InDelta(t, 11.0, plan.LimitPrice, 0.0001)
	assert.Equal(t, core.IntrabarAmbiguityOptimistic, plan.IntrabarAmbiguityPolicy)
}

func TestDecodeYAMLStreamSupportsIntrabarOHLCFillTiming(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: intrabar_ohlc\n  intrabar_ambiguity_policy: open_high_low_close", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, core.FillIntrabarOHLC, bundle.Run.Execution.Fill)
	assert.Equal(t, core.IntrabarAmbiguityOpenHighLowClose, bundle.Run.Execution.IntrabarAmbiguityPolicy)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.FillIntrabarOHLC, plan.Fill)
	assert.Equal(t, core.IntrabarAmbiguityOpenHighLowClose, plan.IntrabarAmbiguityPolicy)
}

func TestDecodeYAMLStreamSupportsTrailingStopOrderType(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  order_type: trailing_stop\n  trailing_stop_pct: 10", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.InDelta(t, 10.0, bundle.Run.Execution.TrailingStopPct, 0.0001)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.OrderTypeTrailingStop, plan.OrderType)
	assert.InDelta(t, 10.0, plan.TrailingStopPct, 0.0001)
}

func TestDecodeYAMLStreamSupportsSideSpecificExecutionCosts(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `commission:
    type: bps
    value: 0`, `commission:
    type: bps
    buy_value: 1
    sell_value: 2
    min_fee: 5`, 1)
	payload = strings.Replace(payload, `slippage:
    type: bps
    value: 0`, `slippage:
    type: fixed_amount
    buy_value: 1
    sell_value: 2
  tax:
    type: bps
    sell_value: 10
  exchange_fee:
    type: fixed_amount
    value: 1`, 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.CostTypeBPS, plan.Commission.Type)
	assert.InDelta(t, 1.0, plan.Commission.BuyValue, 0.0001)
	assert.InDelta(t, 2.0, plan.Commission.SellValue, 0.0001)
	assert.InDelta(t, 5.0, plan.Commission.MinFee, 0.0001)
	assert.Equal(t, core.CostTypeFixedAmount, plan.Slippage.Type)
	assert.InDelta(t, 1.0, plan.Slippage.BuyValue, 0.0001)
	assert.InDelta(t, 2.0, plan.Slippage.SellValue, 0.0001)
	assert.Equal(t, core.CostTypeBPS, plan.Tax.Type)
	assert.InDelta(t, 10.0, plan.Tax.SellValue, 0.0001)
	assert.Equal(t, core.CostTypeFixedAmount, plan.ExchangeFee.Type)
	assert.InDelta(t, 1.0, plan.ExchangeFee.Value, 0.0001)
}

func TestDecodeYAMLStreamSupportsParticipationSlippage(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `slippage:
    type: bps
    value: 0`, `slippage:
    type: participation
    value: 1000`, 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.CostTypeParticipation, plan.Slippage.Type)
	assert.InDelta(t, 1000.0, plan.Slippage.Value, 0.0001)
}

func TestDecodeYAMLStreamSupportsSpreadProxySlippage(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `slippage:
    type: bps
    value: 0`, `slippage:
    type: spread_proxy
    value: 0.5`, 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.CostTypeSpreadProxy, plan.Slippage.Type)
	assert.InDelta(t, 0.5, plan.Slippage.Value, 0.0001)
}

func TestDecodeYAMLStreamSupportsVolatilitySlippage(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `slippage:
    type: bps
    value: 0`, `slippage:
    type: volatility
    value: 0.25`, 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.CostTypeVolatility, plan.Slippage.Type)
	assert.InDelta(t, 0.25, plan.Slippage.Value, 0.0001)
}

func TestDecodeYAMLStreamSupportsATRSlippage(t *testing.T) {
	payload := strings.Replace(sampleYAML(), `slippage:
    type: bps
    value: 0`, `slippage:
    type: atr
    value: 0.5
    window: 14`, 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.Equal(t, core.CostTypeATR, plan.Slippage.Type)
	assert.InDelta(t, 0.5, plan.Slippage.Value, 0.0001)
	assert.Equal(t, 14, plan.Slippage.Window)
}

func TestDecodeYAMLStreamSupportsLotAndTickSize(t *testing.T) {
	payload := strings.Replace(sampleYAML(), "execution:\n  fill: next_open", "execution:\n  fill: next_open\n  lot_size: 10\n  tick_size: 0.05", 1)
	bundle, err := Decode(context.Background(), strings.NewReader(payload))
	require.NoError(t, err)
	assert.InDelta(t, 10.0, bundle.Run.Execution.LotSize, 0.0001)
	assert.InDelta(t, 0.05, bundle.Run.Execution.TickSize, 0.0001)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	assert.InDelta(t, 10.0, plan.LotSize, 0.0001)
	assert.InDelta(t, 0.05, plan.TickSize, 0.0001)
}

func TestDecodeAndCompileStrategySupportsTargetWeightChangedRule(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: target-weight-changed
entry:
  target_weight_changed:
    - value: 5
exit:
  lt:
    - price: close
    - value: 0
sizing:
  type: percent_of_equity
  value: 50
`))
	require.NoError(t, err)

	run := core.BacktestRunSpec{
		Kind:          core.KindBacktestRun,
		SchemaVersion: core.SchemaVersion,
		Name:          "target-weight-changed-run",
		Strategy:      core.StrategyRef{Name: "target-weight-changed"},
		Data:          core.DataSpec{Market: "krx", Timeframe: "1d", From: "2024-01-02", To: "2024-01-08"},
		Universe:      core.UniverseSpec{Symbols: []string{"069500"}},
		Portfolio:     core.PortfolioSpec{InitialCash: 10000, Currency: "KRW"},
		Execution:     core.ExecutionSpec{Fill: core.FillSameClose, OrderType: core.OrderTypeRebalance},
	}
	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(strategy, run, registry)
	require.NoError(t, err)
	assert.Equal(t, "target_weight_changed", plan.Entry.Operator)
	assert.Equal(t, 5.0, plan.Entry.Args[0].Value)
}

func TestDecodeAndCompileStrategySupportsCalendarRebalanceRules(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: calendar-rebalance
entry:
  all:
    - monthly: {}
    - target_weight_changed:
        - value: 5
exit:
  lt:
    - price: close
    - value: 0
sizing:
  type: percent_of_equity
  value: 50
`))
	require.NoError(t, err)

	run := core.BacktestRunSpec{
		Kind:          core.KindBacktestRun,
		SchemaVersion: core.SchemaVersion,
		Name:          "calendar-rebalance-run",
		Strategy:      core.StrategyRef{Name: "calendar-rebalance"},
		Data:          core.DataSpec{Market: "krx", Timeframe: "1d", From: "2024-01-02", To: "2024-02-08"},
		Universe:      core.UniverseSpec{Symbols: []string{"069500"}},
		Portfolio:     core.PortfolioSpec{InitialCash: 10000, Currency: "KRW"},
		Execution:     core.ExecutionSpec{Fill: core.FillSameClose, OrderType: core.OrderTypeRebalance},
	}
	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(strategy, run, registry)
	require.NoError(t, err)
	assert.Equal(t, "all", plan.Entry.Operator)
	require.Len(t, plan.Entry.Rules, 2)
	assert.Equal(t, "monthly", plan.Entry.Rules[0].Operator)
	assert.Equal(t, "target_weight_changed", plan.Entry.Rules[1].Operator)
}

func TestLoadFileLoadsYAMLAndRunsBacktest(t *testing.T) {
	path := writeTempYAML(t, sampleYAML())

	bundle, err := LoadFile(context.Background(), path)
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)
	engine, err := core.NewEngine(core.NewMemoryFeed(sampleBars()))
	require.NoError(t, err)

	result, err := engine.Run(context.Background(), plan)
	require.NoError(t, err)

	require.Len(t, result.Trades, 2)
	assert.Equal(t, core.SideBuy, result.Trades[0].Side)
	assert.Equal(t, core.SideSell, result.Trades[1].Side)
	assert.InDelta(t, 8848.0, result.FinalEquity, 0.0001)
	assert.NotEmpty(t, result.ResultHash)
}

func TestUniversePipelineExampleFixtureCompiles(t *testing.T) {
	bundle, err := LoadFile(context.Background(), "../../examples/backtest/universe-pipeline/universe-pipeline.yaml")
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)

	assert.Equal(t, "liquidity-leader-sma", plan.StrategyName)
	assert.Equal(t, "liquidity-leaders-krx-etf", plan.RunName)
	assert.Equal(t, "monthly", plan.Universe.Schedule.Frequency)
	assert.Equal(t, "liquidate", plan.Universe.PositionPolicy)
	require.Len(t, plan.Universe.Pipeline, 6)
	assert.Equal(t, "source.daily_bars", plan.Universe.Pipeline[0].ID)
	assert.Equal(t, "filter.security_type", plan.Universe.Pipeline[1].ID)
	assert.Equal(t, "rank.weighted", plan.Universe.Pipeline[5].ID)
}

func TestTurtleBreakoutExampleFixtureCompiles(t *testing.T) {
	bundle, err := LoadFile(context.Background(), "../../examples/backtest/turtle-breakout/turtle-breakout.yaml")
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)

	assert.Equal(t, "turtle-breakout", plan.StrategyName)
	assert.Equal(t, "turtle-breakout-krx-etf", plan.RunName)
	assert.Equal(t, "sma", plan.Indicators["long_trend"].ID)
	assert.Equal(t, "roc", plan.Indicators["momentum_60"].ID)
	assert.Equal(t, "donchian_high", plan.Indicators["entry_channel"].ID)
	assert.Equal(t, "donchian_low", plan.Indicators["exit_channel"].ID)
	require.Len(t, plan.Entries, 1)
	assert.Equal(t, "all", plan.Entries[0].Operator)
	require.Len(t, plan.Stops, 1)
	assert.Equal(t, "volatility_stop", plan.Stops[0].Operator)
}

func TestRelativeStrengthRotationExampleFixtureCompiles(t *testing.T) {
	bundle, err := LoadFile(context.Background(), "../../examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml")
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)

	assert.Equal(t, "relative-strength-rotation", plan.StrategyName)
	assert.Equal(t, "relative-strength-rotation-krx-etf", plan.RunName)
	assert.Equal(t, "weekly", plan.Universe.Schedule.Frequency)
	assert.Equal(t, "liquidate", plan.Universe.PositionPolicy)
	assert.Equal(t, core.OrderTypeRebalance, plan.OrderType)
	assert.Equal(t, core.TimeInForceCancelOnRebalance, plan.TimeInForce)
	require.Len(t, plan.Universe.Pipeline, 7)
	assert.Equal(t, "source.daily_bars", plan.Universe.Pipeline[0].ID)
	assert.Equal(t, "rank.by_field", plan.Universe.Pipeline[5].ID)
	assert.Equal(t, "rank.weighted", plan.Universe.Pipeline[6].ID)
}

func TestDualMomentumRotationExampleFixtureCompiles(t *testing.T) {
	bundle, err := LoadFile(context.Background(), "../../examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml")
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)

	assert.Equal(t, "dual-momentum-rotation", plan.StrategyName)
	assert.Equal(t, "dual-momentum-rotation-krx-etf", plan.RunName)
	assert.Equal(t, "monthly", plan.Universe.Schedule.Frequency)
	assert.Equal(t, "liquidate", plan.Universe.PositionPolicy)
	assert.Equal(t, core.OrderTypeRebalance, plan.OrderType)
	assert.Equal(t, 2, plan.Risk.MaxPositions)
	require.Len(t, plan.Universe.Pipeline, 6)
	assert.Equal(t, "source.daily_bars", plan.Universe.Pipeline[0].ID)
	assert.Equal(t, "filter.field", plan.Universe.Pipeline[3].ID)
	assert.Equal(t, "rank.by_field", plan.Universe.Pipeline[5].ID)
}

func TestTurtleBreakoutBuyAndHoldBaselineExampleFixtureCompiles(t *testing.T) {
	bundle, err := LoadFile(context.Background(), "../../examples/backtest/turtle-breakout/buy-and-hold-2024-02-02-to-2024-10-21.yaml")
	require.NoError(t, err)

	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(bundle.Strategy, bundle.Run, registry)
	require.NoError(t, err)

	assert.Equal(t, "buy-and-hold-equal-weight", plan.StrategyName)
	assert.Equal(t, "buy-and-hold-krx-etf-2024-02-02-to-2024-10-21", plan.RunName)
	assert.Equal(t, core.FillSameClose, plan.Fill)
	assert.InDelta(t, 33.3333333333, plan.Sizing.Value, 0.0001)
	require.Len(t, plan.Exits, 1)
	assert.Equal(t, "gte", plan.Exits[0].Operator)
}

func TestDecodeStrategyOnlySupportsNestedRulesAndInlineIndicators(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: nested-rules
indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 3
entry:
  all:
    - any:
        - gt:
            - price: close
            - ref: trend
        - gt:
            - indicator:
                id: rsi
                source:
                  price: close
                params:
                  window: 2
            - value: 50
    - not:
        lt:
          - price: close
          - value: 0
exit:
  lt:
    - price: close
    - value: 1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)
	assert.Equal(t, "nested-rules", strategy.Name)
	assert.Equal(t, "all", strategy.Entry.Operator)
	require.Len(t, strategy.Entry.Rules, 2)
	assert.Equal(t, "not", strategy.Entry.Rules[1].Operator)
}

func TestDecodeStrategySupportsRoleBasedRules(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: role-based
entries:
  - gt:
      - price: close
      - value: 0
exits:
  - lt:
      - price: close
      - value: 0
stops:
  - take_profit:
      - value: 0.05
rebalance:
  - monthly: {}
  - target_weight_changed:
      - value: 1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	assert.Equal(t, "role-based", strategy.Name)
	require.Len(t, strategy.Entries, 1)
	assert.Equal(t, "gt", strategy.Entries[0].Operator)
	require.Len(t, strategy.Exits, 1)
	assert.Equal(t, "lt", strategy.Exits[0].Operator)
	require.Len(t, strategy.Stops, 1)
	assert.Equal(t, "take_profit", strategy.Stops[0].Operator)
	require.Len(t, strategy.Rebalance, 2)
	assert.Equal(t, "monthly", strategy.Rebalance[0].Operator)
	assert.Equal(t, "target_weight_changed", strategy.Rebalance[1].Operator)
	assert.Equal(t, "gt", strategy.Entry.Operator)
	assert.Equal(t, "lt", strategy.Exit.Operator)
}

func TestDecodeStrategySupportsCrossSectionalValueExpressions(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: cross-sectional-values
entry:
  all:
    - lte:
        - rank:
            - price: close
        - value: 3
    - gte:
        - percentile:
            - price: volume
        - value: 50
    - gt:
        - relative_strength:
            - price: close
        - value: 0
    - gt:
        - spread:
            - price: close
            - value: 100
        - value: 0
    - gt:
        - ratio:
            - price: close
            - value: 100
        - value: 1
    - lte:
        - universe_rank:
            - price: traded_amount
        - value: 5
exit:
  lt:
    - price: close
    - value: 1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	require.Len(t, strategy.Entry.Rules, 6)
	assert.Equal(t, "rank", strategy.Entry.Rules[0].Args[0].Kind)
	assert.Equal(t, "percentile", strategy.Entry.Rules[1].Args[0].Kind)
	assert.Equal(t, "relative_strength", strategy.Entry.Rules[2].Args[0].Kind)
	assert.Equal(t, "spread", strategy.Entry.Rules[3].Args[0].Kind)
	assert.Equal(t, "ratio", strategy.Entry.Rules[4].Args[0].Kind)
	assert.Equal(t, "universe_rank", strategy.Entry.Rules[5].Args[0].Kind)
}

func TestDecodeStrategySupportsPositionAndPortfolioValueExpressions(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: state-rules
entry:
  all:
    - lt:
        - portfolio: position_count
        - value: 3
    - gt:
        - portfolio: cash_pct
        - value: 20
exit:
  all:
    - position_exists: []
    - gte:
        - position: holding_bars
        - value: 5
    - lt:
        - position: drawdown_from_entry
        - value: -0.1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	require.Len(t, strategy.Entry.Rules, 2)
	assert.Equal(t, "portfolio", strategy.Entry.Rules[0].Args[0].Kind)
	assert.Equal(t, "position_count", strategy.Entry.Rules[0].Args[0].Portfolio)
	require.Len(t, strategy.Exit.Rules, 3)
	assert.Equal(t, "position_exists", strategy.Exit.Rules[0].Operator)
	assert.Equal(t, "position", strategy.Exit.Rules[1].Args[0].Kind)
	assert.Equal(t, "holding_bars", strategy.Exit.Rules[1].Args[0].Position)
}

func TestDecodeStrategySupportsArithmeticValueExpressions(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: arithmetic-values
indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 2
entry:
  gt:
    - div:
        - sub:
            - price: close
            - ref: trend
        - ref: trend
    - value: 0.05
exit:
  lt:
    - abs:
        - sub:
            - price: close
            - ref: trend
    - max:
        - value: 1
        - min:
            - value: 2
            - value: 3
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	require.Len(t, strategy.Entry.Args, 2)
	assert.Equal(t, "div", strategy.Entry.Args[0].Kind)
	require.Len(t, strategy.Entry.Args[0].Args, 2)
	assert.Equal(t, "sub", strategy.Entry.Args[0].Args[0].Kind)
	assert.Equal(t, "ref", strategy.Entry.Args[0].Args[1].Kind)
	assert.Equal(t, "abs", strategy.Exit.Args[0].Kind)
	assert.Equal(t, "max", strategy.Exit.Args[1].Kind)

	run := core.BacktestRunSpec{
		Kind:          core.KindBacktestRun,
		SchemaVersion: core.SchemaVersion,
		Name:          "arithmetic-values-run",
		Strategy:      core.StrategyRef{Name: "arithmetic-values"},
		Data: core.DataSpec{
			Market:    "krx",
			Timeframe: "1d",
			From:      "2024-01-02",
			To:        "2024-01-08",
		},
		Universe:  core.UniverseSpec{Symbols: []string{"069500"}},
		Portfolio: core.PortfolioSpec{InitialCash: 10000, Currency: "KRW"},
		Execution: core.ExecutionSpec{Fill: core.FillNextOpen},
	}
	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	_, err = core.Compile(strategy, run, registry)
	require.NoError(t, err)
}

func TestDecodeStrategySupportsStopRules(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: stop-rules
entry:
  gt:
    - price: close
    - value: 0
exit:
  any:
    - stop_loss:
        - value: 0.08
    - take_profit:
        - value: 0.20
    - trailing_stop:
        - value: 0.10
    - time_stop:
        - value: 20
    - volatility_stop:
        - indicator:
            id: atr
            source:
              price: close
            params:
              window: 14
        - value: 2
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	require.Len(t, strategy.Exit.Rules, 5)
	assert.Equal(t, "stop_loss", strategy.Exit.Rules[0].Operator)
	assert.Equal(t, 0.08, strategy.Exit.Rules[0].Args[0].Value)
	assert.Equal(t, "take_profit", strategy.Exit.Rules[1].Operator)
	assert.Equal(t, "trailing_stop", strategy.Exit.Rules[2].Operator)
	assert.Equal(t, "time_stop", strategy.Exit.Rules[3].Operator)
	assert.Equal(t, "volatility_stop", strategy.Exit.Rules[4].Operator)
	require.Len(t, strategy.Exit.Rules[4].Args, 2)
	require.NotNil(t, strategy.Exit.Rules[4].Args[0].Indicator)
	assert.Equal(t, "atr", strategy.Exit.Rules[4].Args[0].Indicator.ID)
	assert.Equal(t, 2.0, strategy.Exit.Rules[4].Args[1].Value)
}

func TestDecodeAndCompileStrategySupportsExpandedIndicators(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: expanded-indicators
indicators:
  trend:
    id: ema
    source:
      price: close
    params:
      window: 3
  momentum:
    id: roc
    source:
      price: close
    params:
      window: 2
  smooth_trend:
    id: hma
    source:
      price: close
    params:
      window: 4
  adaptive_trend:
    id: kama
    source:
      price: close
    params:
      window: 3
      fast_window: 2
      slow_window: 5
  oscillator:
    id: stochastic
    source:
      price: close
    output: d
    params:
      k_window: 3
      d_window: 2
  trend_strength:
    id: adx
    source:
      price: close
    output: di_plus
    params:
      window: 3
  mean_reversion:
    id: zscore
    source:
      price: close
    params:
      window: 3
  return_correlation:
    id: correlation
    source:
      price: close
    compare:
      price: nav
    params:
      window: 3
  return_beta:
    id: beta
    source:
      price: close
    compare:
      price: nav
    params:
      window: 3
  upper_band:
    id: bollinger_upper
    source:
      price: close
    params:
      window: 3
      multiplier: 2
  volatility_band:
    id: keltner_upper
    source:
      price: close
    params:
      window: 3
      atr_window: 3
      multiplier: 1.5
  momentum_spread:
    id: macd
    source:
      price: close
    output: histogram
    params:
      fast_window: 2
      slow_window: 3
      signal_window: 2
entry:
  all:
    - gt:
        - price: close
        - ref: trend
    - gt:
        - ref: momentum
        - value: 0
exit:
  gt:
    - price: close
    - ref: upper_band
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	run := core.BacktestRunSpec{
		Kind:          core.KindBacktestRun,
		SchemaVersion: core.SchemaVersion,
		Name:          "expanded-indicators-run",
		Strategy:      core.StrategyRef{Name: "expanded-indicators"},
		Data: core.DataSpec{
			Market:    "krx",
			Timeframe: "1d",
			From:      "2024-01-02",
			To:        "2024-01-08",
		},
		Universe:  core.UniverseSpec{Symbols: []string{"069500"}},
		Portfolio: core.PortfolioSpec{InitialCash: 10000, Currency: "KRW"},
		Execution: core.ExecutionSpec{Fill: core.FillNextOpen},
	}
	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	plan, err := core.Compile(strategy, run, registry)
	require.NoError(t, err)
	assert.Equal(t, "ema", plan.Indicators["trend"].ID)
	assert.Equal(t, "roc", plan.Indicators["momentum"].ID)
	assert.Equal(t, "hma", plan.Indicators["smooth_trend"].ID)
	assert.Equal(t, "kama", plan.Indicators["adaptive_trend"].ID)
	assert.Equal(t, "stochastic", plan.Indicators["oscillator"].ID)
	assert.Equal(t, "d", plan.Indicators["oscillator"].Output)
	assert.Equal(t, "adx", plan.Indicators["trend_strength"].ID)
	assert.Equal(t, "di_plus", plan.Indicators["trend_strength"].Output)
	assert.Equal(t, "zscore", plan.Indicators["mean_reversion"].ID)
	assert.Equal(t, "correlation", plan.Indicators["return_correlation"].ID)
	assert.Equal(t, "nav", plan.Indicators["return_correlation"].Compare.Price)
	assert.Equal(t, "beta", plan.Indicators["return_beta"].ID)
	assert.Equal(t, "nav", plan.Indicators["return_beta"].Compare.Price)
	assert.Equal(t, "bollinger_upper", plan.Indicators["upper_band"].ID)
	assert.Equal(t, "keltner_upper", plan.Indicators["volatility_band"].ID)
	assert.Equal(t, "macd", plan.Indicators["momentum_spread"].ID)
	assert.Equal(t, "histogram", plan.Indicators["momentum_spread"].Output)
}

func TestDecodeStrategySupportsTemporalRules(t *testing.T) {
	strategy, err := DecodeStrategy(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: temporal-rules
entry:
  all:
    - for_n_bars:
        bars: 3
        rule:
          gt:
            - price: close
            - value: 10
    - new_high:
        - price: close
        - value: 20
    - changed:
        - price: close
    - bars_since:
        bars: 2
        rule:
          gt:
            - price: volume
            - value: 1000
exit:
  cooldown:
    bars: 5
    rule:
      new_low:
        - price: close
        - value: 10
sizing:
  type: percent_of_equity
  value: 10
`))
	require.NoError(t, err)

	require.Len(t, strategy.Entry.Rules, 4)
	assert.Equal(t, "for_n_bars", strategy.Entry.Rules[0].Operator)
	require.NotNil(t, strategy.Entry.Rules[0].Rule)
	assert.Equal(t, "gt", strategy.Entry.Rules[0].Rule.Operator)
	require.Len(t, strategy.Entry.Rules[0].Args, 1)
	assert.Equal(t, 3.0, strategy.Entry.Rules[0].Args[0].Value)
	assert.Equal(t, "new_high", strategy.Entry.Rules[1].Operator)
	assert.Equal(t, "changed", strategy.Entry.Rules[2].Operator)
	assert.Equal(t, "bars_since", strategy.Entry.Rules[3].Operator)
	require.NotNil(t, strategy.Entry.Rules[3].Rule)
	assert.Equal(t, "gt", strategy.Entry.Rules[3].Rule.Operator)
	assert.Equal(t, 2.0, strategy.Entry.Rules[3].Args[0].Value)
	assert.Equal(t, "cooldown", strategy.Exit.Operator)
	require.NotNil(t, strategy.Exit.Rule)
	assert.Equal(t, "new_low", strategy.Exit.Rule.Operator)
	assert.Equal(t, 5.0, strategy.Exit.Args[0].Value)
}

func TestDecodeYAMLErrorsAreExplicit(t *testing.T) {
	_, err := Decode(context.Background(), strings.NewReader(`kind: Unknown`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported YAML document kind")

	_, err = Decode(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: bad
entry:
  gt:
    price: close
exit:
  gt:
    - price: close
    - value: 1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "comparison rule requires sequence args")

	_, err = Decode(context.Background(), strings.NewReader(`
kind: Strategy
schema_version: 1
name: bad
entry:
  gt:
    - price: close
    - value: abc
exit:
  gt:
    - price: close
    - value: 1
sizing:
  type: percent_of_equity
  value: 10
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse numeric value expression")
}

func TestDecodeReportMetricsSequenceAndInvalidShape(t *testing.T) {
	bundle, err := Decode(context.Background(), strings.NewReader(strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `report:
  metrics:
    - total_return
    - win_rate`, 1)))
	require.NoError(t, err)
	assert.Equal(t, []string{"total_return", "win_rate"}, bundle.Run.Report.Metrics.Include)

	_, err = Decode(context.Background(), strings.NewReader(strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `report:
  metrics: 10`, 1)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report metrics must be mapping or sequence")
}

func writeTempYAML(t *testing.T, payload string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "backtest-*.yaml")
	require.NoError(t, err)
	_, err = file.WriteString(payload)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	return file.Name()
}

func sampleYAML() string {
	return `
kind: Strategy
schema_version: 1
name: sma-cross
indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 2
entry:
  crosses_above:
    - price: close
    - ref: trend
exit:
  crosses_below:
    - price: close
    - ref: trend
sizing:
  type: percent_of_equity
  value: 50
risk:
  max_positions: 1
  max_symbol_weight_pct: 60
---
kind: BacktestRun
schema_version: 1
name: sma-cross-run
strategy:
  name: sma-cross
data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-02
  to: 2024-01-08
universe:
  symbols: ["069500"]
portfolio:
  initial_cash: 10000
  currency: KRW
execution:
  fill: next_open
  commission:
    type: bps
    value: 0
  slippage:
    type: bps
    value: 0
report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count
`
}

func sampleBars() []core.Bar {
	return []core.Bar{
		{Time: mustDate("2024-01-02"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
		{Time: mustDate("2024-01-03"), Symbol: "069500", Open: 9, High: 9, Low: 9, Close: 9},
		{Time: mustDate("2024-01-04"), Symbol: "069500", Open: 12, High: 12, Low: 12, Close: 12},
		{Time: mustDate("2024-01-05"), Symbol: "069500", Open: 13, High: 13, Low: 11, Close: 11},
		{Time: mustDate("2024-01-08"), Symbol: "069500", Open: 10, High: 10, Low: 10, Close: 10},
	}
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
