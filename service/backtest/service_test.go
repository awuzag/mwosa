package backtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	universeservice "github.com/ev3rlit/mwosa/service/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceValidatesAndRunsYAMLAgainstDailyBarRepository(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	service, err := NewService(repo)
	require.NoError(t, err)

	validation, err := service.Validate(context.Background(), path)
	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, "sma-cross", validation.StrategyName)
	assert.Equal(t, "sma-cross-run", validation.RunName)
	assert.Equal(t, "next_open", validation.Execution.Fill)
	assert.Equal(t, "sma", validation.Indicators["trend"])
	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "win_rate", "average_trade_return"}, validation.Metrics)

	result, err := service.Run(context.Background(), path)
	require.NoError(t, err)

	require.Len(t, repo.queries, 1)
	assert.Equal(t, provider.MarketKRX, repo.queries[0].Market)
	assert.Equal(t, provider.SecurityTypeETF, repo.queries[0].SecurityType)
	assert.Equal(t, "069500", repo.queries[0].Symbol)
	assert.Equal(t, "2024-01-02", repo.queries[0].From)
	assert.Equal(t, "2024-01-08", repo.queries[0].To)
	assert.Equal(t, "sma-cross", result.StrategyName)
	assert.Equal(t, []string{"069500"}, result.Symbols)
	assert.Equal(t, "krx", result.Market)
	assert.Equal(t, "etf", result.SecurityType)
	assert.Equal(t, "1d", result.Timeframe)
	assert.Equal(t, "next_open", result.Execution.Fill)
	require.Len(t, result.Trades, 2)
	assert.NotEmpty(t, result.EquityCurve)
	assert.NotContains(t, result.Metrics, "trade_count")
	assert.Contains(t, result.Metrics, "average_trade_return")
	assert.NotEmpty(t, result.ResultHash)
	assert.Equal(t, "once", result.Universe.Schedule)
	assert.Equal(t, []string{"069500"}, result.Universe.SelectedSymbols)
}

func TestServiceValidatesDailyBarsUniversePipelineWithExplain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipeline.yaml")
	require.NoError(t, os.WriteFile(path, []byte(samplePipelineYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
		"102110": sampleBenchmarkCanonicalDailyBars(),
	}}
	service, err := NewService(repo)
	require.NoError(t, err)

	validation, err := service.Validate(context.Background(), path)
	require.NoError(t, err)

	assert.True(t, validation.Valid)
	assert.Equal(t, []string{"102110"}, validation.Symbols)
	assert.Equal(t, "once", validation.Universe.Schedule)
	assert.Equal(t, []string{"102110"}, validation.Universe.SelectedSymbols)
	require.Len(t, validation.Universe.Steps, 4)
	assert.Equal(t, "source.daily_bars", validation.Universe.Steps[0].ID)
	assert.Equal(t, "rank.by_field", validation.Universe.Steps[2].ID)
	assert.NotEmpty(t, validation.Universe.Decisions)
}

func TestServiceInspectUniverseLoadsFileAndScreenSources(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "universe.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte("symbol,score\n252670,3\n"), 0o644))
	yamlPath := filepath.Join(dir, "pipeline.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(sampleExternalUniverseYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{}}
	screenRepo := fakeScreenRepository{items: []strategyservice.ScreenRunItem{
		{Ordinal: 0, Symbol: "069500", PayloadJSON: json.RawMessage(`{"symbol":"069500","score":2}`)},
	}}
	screenRunner := fakeScreenRunner{items: []strategyservice.ScreenRunItem{
		{Ordinal: 0, Symbol: "102110", PayloadJSON: json.RawMessage(`{"symbol":"102110","score":1}`)},
	}}
	service, err := NewServiceWithUniverseSources(repo, nil, screenRepo, screenRunner)
	require.NoError(t, err)

	explain, err := service.InspectUniverse(context.Background(), yamlPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"069500", "102110", "252670"}, explain.SelectedSymbols)
	require.Len(t, explain.Steps, 2)
	assert.Equal(t, "combine.union", explain.Steps[0].ID)
}

func TestServiceUniverseFileReadersSupportJSONAndNDJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "universe.json")
	ndjsonPath := filepath.Join(dir, "universe.ndjson")
	require.NoError(t, os.WriteFile(jsonPath, []byte(`[{"symbol":"069500","score":1},{"symbol":"102110","score":2}]`), 0o644))
	require.NoError(t, os.WriteFile(ndjsonPath, []byte("{\"symbol\":\"069500\",\"score\":1}\n{\"symbol\":\"102110\",\"score\":2}\n"), 0o644))

	jsonRows, err := universeservice.ReadCandidateFile(jsonPath)
	require.NoError(t, err)
	assert.Equal(t, []coreSymbol{{"069500"}, {"102110"}}, candidateSymbols(jsonRows))

	ndjsonRows, err := universeservice.ReadCandidateFile(ndjsonPath)
	require.NoError(t, err)
	assert.Equal(t, []coreSymbol{{"069500"}, {"102110"}}, candidateSymbols(ndjsonRows))

	_, err = universeservice.ReadCandidateFile(filepath.Join(dir, "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open universe source file")
}

func TestServiceConstructorsAndRepositoryErrorPaths(t *testing.T) {
	_, err := NewServiceWithRegistry(&recordingDailyBarRepository{}, mustIndicatorRegistry(t))
	require.NoError(t, err)
	_, err = NewService(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily bar reader is nil")

	service, err := NewService(&recordingDailyBarRepository{})
	require.NoError(t, err)
	_, err = service.ListStrategies(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backtest strategy repository is nil")
	_, err = service.InspectStrategy(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires name")
	err = service.DeleteStrategy(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires name")
}

func TestServiceMapperAndLookbackErrorPaths(t *testing.T) {
	_, err := canonicalDailyBarToBacktestBar(canonicalDailyBarForSymbol("069500", "bad-date", "1", "1", "1", "1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse trading date")

	_, err = canonicalDailyBarToBacktestBar(canonicalDailyBarForSymbol("069500", "2024-01-02", "", "1", "1", "1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily bar numeric field is required")

	_, err = canonicalDailyBarToBacktestBar(canonicalDailyBarForSymbol("069500", "2024-01-02", "x", "1", "1", "1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse daily bar numeric field")

	lookback := universeservice.MaxLookbackDays([]core.UniverseSelectorStepSpec{
		{ID: "combine.union", Params: map[string]any{"pipelines": []any{
			map[string]any{"pipeline": []any{
				map[string]any{"id": "source.daily_bars", "params": map[string]any{"lookback_days": "30"}},
				map[string]any{"id": "transform.window_metrics", "params": map[string]any{"metrics": map[string]any{
					"ret": map[string]any{"id": "return", "params": map[string]any{"window": 63.0}},
				}}},
			}},
		}}},
	})
	assert.Equal(t, 63, lookback)
}

func TestServiceUpsertStrategyInputErrors(t *testing.T) {
	service, err := NewServiceWithRepository(&recordingDailyBarRepository{}, newMemoryBacktestStrategyRepository())
	require.NoError(t, err)
	_, err = service.UpsertStrategy(context.Background(), SaveStrategyRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backtest strategy name is required")

	service, err = NewService(&recordingDailyBarRepository{})
	require.NoError(t, err)
	_, err = service.UpsertStrategy(context.Background(), SaveStrategyRequest{Name: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backtest strategy repository is nil")
}

func TestServiceLoadsBenchmarkBarsForBenchmarkMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAMLWithBenchmarkMetrics()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
		"102110": sampleBenchmarkCanonicalDailyBars(),
	}}
	service, err := NewService(repo)
	require.NoError(t, err)

	result, err := service.Run(context.Background(), path)
	require.NoError(t, err)

	require.Len(t, repo.queries, 2)
	assert.Equal(t, "069500", repo.queries[0].Symbol)
	assert.Equal(t, "102110", repo.queries[1].Symbol)
	assert.Contains(t, result.Metrics, "benchmark_total_return")
	assert.Contains(t, result.Metrics, "excess_return")
}

func TestServiceErrorsWhenDailyBarsAreMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	service, err := NewService(&recordingDailyBarRepository{bars: map[string][]dailybar.Bar{}})
	require.NoError(t, err)

	_, err = service.Run(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canonical daily bars not found")
	assert.Contains(t, err.Error(), "symbol=069500")
}

type recordingDailyBarRepository struct {
	bars    map[string][]dailybar.Bar
	queries []daily.Query
}

type fakeScreenRepository struct {
	items []strategyservice.ScreenRunItem
}

func (r fakeScreenRepository) GetScreenRun(context.Context, string) (strategyservice.ScreenRunDetail, error) {
	return strategyservice.ScreenRunDetail{Items: r.items}, nil
}

type fakeScreenRunner struct {
	items []strategyservice.ScreenRunItem
}

type coreSymbol struct {
	Symbol string
}

func candidateSymbols(rows []core.UniverseCandidate) []coreSymbol {
	out := make([]coreSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, coreSymbol{Symbol: row.Symbol})
	}
	return out
}

func mustIndicatorRegistry(t *testing.T) core.IndicatorRegistry {
	t.Helper()
	registry, err := core.DefaultIndicatorRegistry()
	require.NoError(t, err)
	return registry
}

func (r fakeScreenRunner) Screen(context.Context, strategyservice.ScreenStrategyRequest) (strategyservice.ScreenRunDetail, error) {
	return strategyservice.ScreenRunDetail{Items: r.items}, nil
}

func (r *recordingDailyBarRepository) QueryDailyBars(_ context.Context, query daily.Query) ([]dailybar.Bar, error) {
	r.queries = append(r.queries, query)
	if query.Symbol == "" {
		out := make([]dailybar.Bar, 0)
		for _, rows := range r.bars {
			out = append(out, rows...)
		}
		return out, nil
	}
	return append([]dailybar.Bar(nil), r.bars[query.Symbol]...), nil
}

func sampleCanonicalDailyBars() []dailybar.Bar {
	return []dailybar.Bar{
		canonicalDailyBar("2024-01-02", "10", "10", "10", "10"),
		canonicalDailyBar("2024-01-03", "9", "9", "9", "9"),
		canonicalDailyBar("2024-01-04", "12", "12", "12", "12"),
		canonicalDailyBar("2024-01-05", "13", "13", "11", "11"),
		canonicalDailyBar("2024-01-08", "10", "10", "10", "10"),
	}
}

func sampleBenchmarkCanonicalDailyBars() []dailybar.Bar {
	return []dailybar.Bar{
		canonicalDailyBarForSymbol("102110", "2024-01-02", "100", "100", "100", "100"),
		canonicalDailyBarForSymbol("102110", "2024-01-03", "105", "105", "105", "105"),
		canonicalDailyBarForSymbol("102110", "2024-01-04", "95", "95", "95", "95"),
		canonicalDailyBarForSymbol("102110", "2024-01-05", "110", "110", "110", "110"),
		canonicalDailyBarForSymbol("102110", "2024-01-08", "110", "110", "110", "110"),
	}
}

func canonicalDailyBar(date string, open string, high string, low string, closePrice string) dailybar.Bar {
	return canonicalDailyBarForSymbol("069500", date, open, high, low, closePrice)
}

func canonicalDailyBarForSymbol(symbol string, date string, open string, high string, low string, closePrice string) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       symbol,
		TradingDate:  date,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        closePrice,
		Volume:       "1000",
	}
}

func sampleYAMLWithBenchmarkMetrics() string {
	return strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `benchmark:
  symbol: "102110"
  name: benchmark
report:
  metrics:
    preset: core
    include:
      - benchmark_total_return
      - excess_return`, 1)
}

func samplePipelineYAML() string {
	return strings.Replace(sampleYAML(), `data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-02
  to: 2024-01-08
universe:
  symbols: ["069500"]`, `data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-03
  to: 2024-01-08
universe:
  pipeline:
    - id: source.daily_bars
      params:
        market: krx
        security_type: etf
        lookback_days: 5
    - id: transform.latest_per_symbol
    - id: rank.by_field
      params:
        field: close
        order: desc
        limit: 1
    - id: debug.assert_count
      params:
        min: 1
        max: 1`, 1)
}

func sampleExternalUniverseYAML() string {
	return `
kind: Strategy
schema_version: 1
name: external-universe
entry:
  gt:
    - price: close
    - value: 0
exit:
  lt:
    - price: close
    - value: 0
sizing:
  type: percent_of_equity
  value: 10
---
kind: BacktestRun
schema_version: 1
name: external-universe-run
strategy:
  name: external-universe
data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-02
  to: 2024-01-08
universe:
  pipeline:
    - id: combine.union
      params:
        pipelines:
          - name: saved
            pipeline:
              - id: source.saved_screen
                params:
                  name: saved-leaders
          - name: strategy
            pipeline:
              - id: source.screen_strategy
                params:
                  name: momentum
          - name: file
            pipeline:
              - id: source.file
                params:
                  path: universe.csv
    - id: rank.by_field
      params:
        field: score
        order: desc
portfolio:
  initial_cash: 10000
execution:
  fill: next_open
`
}
