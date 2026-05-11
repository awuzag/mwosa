package backtest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
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
