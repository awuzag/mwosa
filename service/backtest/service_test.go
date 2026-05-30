package backtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	universeservice "github.com/awuzag/mwosa/service/universe"
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
	assert.Equal(t, core.Timeframe1Day, validation.Timeframes.Requested)
	assert.Equal(t, core.Timeframe1Day, validation.Timeframes.Source)
	assert.Equal(t, 2, validation.Timeframes.Warmup.Bars)
	assert.Equal(t, []string{"total_return", "final_equity", "max_drawdown", "win_rate", "average_trade_return"}, validation.Metrics)

	result, err := service.Run(context.Background(), path)
	require.NoError(t, err)

	require.Len(t, repo.queries, 1)
	assert.Equal(t, provider.MarketKRX, repo.queries[0].Market)
	assert.Empty(t, repo.queries[0].SecurityType)
	assert.Empty(t, repo.queries[0].Symbol)
	assert.Equal(t, "2024-01-02", repo.queries[0].From)
	assert.Equal(t, "2024-01-08", repo.queries[0].To)
	assert.Equal(t, "sma-cross", result.StrategyName)
	assert.Equal(t, []string{"069500"}, result.Symbols)
	assert.Equal(t, []core.InstrumentIdentity{{Symbol: "069500", Market: "krx", SecurityType: "etf"}}, result.Instruments)
	assert.Equal(t, "krx", result.Market)
	assert.Equal(t, "1d", result.Timeframe)
	assert.Equal(t, core.Timeframe1Day, result.Timeframes.Requested)
	assert.Equal(t, core.Timeframe1Day, result.Timeframes.Source)
	assert.False(t, result.Timeframes.Resample.Enabled)
	assert.Equal(t, "next_open", result.Execution.Fill)
	require.Len(t, result.Trades, 2)
	assert.NotEmpty(t, result.EquityCurve)
	assert.NotContains(t, result.Metrics, "trade_count")
	assert.Contains(t, result.Metrics, "average_trade_return")
	assert.NotEmpty(t, result.ResultHash)
	assert.Equal(t, "once", result.Universe.Schedule)
	assert.Equal(t, []string{"069500"}, result.Universe.SelectedSymbols)
}

func TestServiceRunAndSavePersistsSingleBacktestRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	runRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, runRepo)
	require.NoError(t, err)

	detail, err := service.RunAndSave(context.Background(), path)
	require.NoError(t, err)

	assert.NotEmpty(t, detail.Run.ID)
	assert.Equal(t, "sma-cross-run", detail.Run.RunName)
	assert.Equal(t, detail.Result.ResultHash, detail.Run.ResultHash)
	assert.Equal(t, detail.Result.DataFingerprint, detail.Run.DataFingerprint)
	assert.NotEmpty(t, detail.Run.StrategyHash)
	assert.NotEmpty(t, detail.Run.RunHash)
	assert.NotEmpty(t, detail.Run.ResultJSON)
	assert.NotEmpty(t, detail.Run.MetricsJSON)

	inspected, err := service.InspectRun(context.Background(), detail.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, detail.Result.ResultHash, inspected.Result.ResultHash)

	listed, err := service.ListRuns(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].ResultJSON)
}

func TestServiceComparesSavedBacktestRuns(t *testing.T) {
	dir := t.TempDir()
	leftPath := filepath.Join(dir, "left.yaml")
	rightPath := filepath.Join(dir, "right.yaml")
	require.NoError(t, os.WriteFile(leftPath, []byte(sampleYAML()), 0o644))
	require.NoError(t, os.WriteFile(rightPath, []byte(strings.Replace(sampleYAML(), "value: 50", "value: 25", 1)), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	runRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, runRepo)
	require.NoError(t, err)

	left, err := service.RunAndSave(context.Background(), leftPath)
	require.NoError(t, err)
	right, err := service.RunAndSave(context.Background(), rightPath)
	require.NoError(t, err)

	comparison, err := service.CompareRuns(context.Background(), left.Run.ID, right.Run.ID)
	require.NoError(t, err)

	assert.False(t, comparison.SameStrategyHash)
	assert.True(t, comparison.SameRunHash)
	assert.True(t, comparison.SameDataFingerprint)
	assert.False(t, comparison.SameResultHash)
	assert.NotEmpty(t, comparison.Metrics)
	var sawTotalReturn bool
	for _, metric := range comparison.Metrics {
		if metric.Metric == core.MetricTotalReturn {
			sawTotalReturn = true
			assert.True(t, metric.LeftPresent)
			assert.True(t, metric.RightPresent)
		}
	}
	assert.True(t, sawTotalReturn)
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

func TestServiceRunsMixedUniverseUsingCandidateSecurityTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mixed.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleMixedUniverseYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"005930": mixedCanonicalDailyBars("005930", provider.SecurityTypeStock),
		"069500": mixedCanonicalDailyBars("069500", provider.SecurityTypeETF),
		"580001": mixedCanonicalDailyBars("580001", provider.SecurityTypeETN),
	}}
	service, err := NewService(repo)
	require.NoError(t, err)

	result, err := service.Run(context.Background(), path)
	require.NoError(t, err)

	assert.Equal(t, []string{"005930", "069500", "580001"}, result.Symbols)
	assert.ElementsMatch(t, []core.InstrumentIdentity{
		{Symbol: "005930", Market: "krx", SecurityType: "stock"},
		{Symbol: "069500", Market: "krx", SecurityType: "etf"},
		{Symbol: "580001", Market: "krx", SecurityType: "etn"},
	}, result.Instruments)
	require.Len(t, repo.queries, 2)
	assert.Empty(t, repo.queries[0].SecurityType)
	assert.Equal(t, "", repo.queries[0].Symbol)
	assert.Empty(t, repo.queries[1].SecurityType)
	assert.Empty(t, repo.queries[1].Symbol)
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

	assert.Equal(t, []string{"252670", "069500", "102110"}, explain.SelectedSymbols)
	require.Len(t, explain.Steps, 2)
	assert.Equal(t, "combine.union", explain.Steps[0].ID)
}

func TestServiceInspectUniversePassesScreenStrategyVersionRef(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "screen-strategy-ref.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(`
kind: Strategy
schema_version: 1
name: screen-ref
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
name: screen-ref-run
strategy:
  name: screen-ref
data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-02
  to: 2024-01-08
universe:
  pipeline:
    - id: source.screen_strategy
      params:
        name: etf-uptrend
        spec_hash: sha256:fixed
portfolio:
  initial_cash: 10000
execution:
  fill: next_open
`), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{}}
	screenRunner := &recordingScreenRunner{items: []strategyservice.ScreenRunItem{
		{Ordinal: 0, Symbol: "069500", PayloadJSON: json.RawMessage(`{"symbol":"069500"}`)},
	}}
	service, err := NewServiceWithUniverseSources(repo, nil, nil, screenRunner)
	require.NoError(t, err)

	explain, err := service.InspectUniverse(context.Background(), yamlPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"069500"}, explain.SelectedSymbols)
	require.Len(t, screenRunner.requests, 1)
	assert.Equal(t, "etf-uptrend", screenRunner.requests[0].Name)
	assert.Equal(t, "sha256:fixed", screenRunner.requests[0].SpecHash)
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

func TestCanonicalDailyBarMapperCarriesAmountMarketCapAndNAV(t *testing.T) {
	bar, err := canonicalDailyBarToBacktestBar(dailybar.Bar{
		TradingDate: "2024-01-02",
		Symbol:      "069500",
		Open:        "10",
		High:        "12",
		Low:         "9",
		Close:       "11",
		Volume:      "1000",
		TradedValue: "11000",
		MarketCap:   "1000000000",
		Extensions:  map[string]string{"nav": "10.5", "adjusted_close": "10.8"},
	})
	require.NoError(t, err)

	assert.InDelta(t, 10.8, bar.AdjustedClose, 0.0001)
	assert.InDelta(t, 11000.0, bar.TradedAmount, 0.0001)
	assert.InDelta(t, 1000000000.0, bar.MarketCap, 0.0001)
	assert.InDelta(t, 10.5, bar.NAV, 0.0001)
	assert.Equal(t, core.Timeframe1Day, bar.Timeframe)
	assert.Equal(t, core.BarSessionRegular, bar.Session)
	assert.Equal(t, core.BarStatusOK, bar.Status)

	bar, err = canonicalDailyBarToBacktestBar(canonicalDailyBarForSymbol("069500", "2024-01-03", "10", "12", "9", "11"))
	require.NoError(t, err)
	assert.Zero(t, bar.AdjustedClose)
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

	require.Len(t, repo.queries, 1)
	assert.Equal(t, provider.MarketKRX, repo.queries[0].Market)
	assert.Empty(t, repo.queries[0].SecurityType)
	assert.Empty(t, repo.queries[0].Symbol)
	assert.Contains(t, result.Metrics, "benchmark_total_return")
	assert.Contains(t, result.Metrics, "excess_return")
	assert.Contains(t, result.Metrics, "benchmark_alpha")
	assert.Contains(t, result.Metrics, "benchmark_beta")
}

func TestDailyBarFeedUsesSingleMarketCursorForMultiSymbolFrames(t *testing.T) {
	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"AAA": {
			canonicalDailyBarForSymbol("AAA", "2024-01-02", "10", "10", "10", "10"),
			canonicalDailyBarForSymbol("AAA", "2024-01-03", "11", "11", "11", "11"),
		},
		"BBB": {
			canonicalDailyBarForSymbol("BBB", "2024-01-02", "20", "20", "20", "20"),
			canonicalDailyBarForSymbol("BBB", "2024-01-03", "21", "21", "21", "21"),
		},
	}}
	from, err := time.Parse(time.DateOnly, "2024-01-02")
	require.NoError(t, err)
	to, err := time.Parse(time.DateOnly, "2024-01-03")
	require.NoError(t, err)

	stream, err := newDailyBarFeed(repo).Open(context.Background(), core.DataRequest{
		From: from,
		To:   to,
		Instruments: []core.InstrumentIdentity{
			{Symbol: "AAA", Market: "krx", SecurityType: "etf"},
			{Symbol: "BBB", Market: "krx", SecurityType: "etf"},
		},
	})
	require.NoError(t, err)
	defer stream.Close()

	require.Len(t, repo.queries, 1)
	assert.Equal(t, provider.Market("krx"), repo.queries[0].Market)
	assert.Empty(t, repo.queries[0].SecurityType)
	assert.Empty(t, repo.queries[0].Symbol)

	first, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2024-01-02", first.Time.Format(time.DateOnly))
	assert.ElementsMatch(t, []string{"AAA", "BBB"}, mapKeys(first.Bars))

	second, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2024-01-03", second.Time.Format(time.DateOnly))
	assert.ElementsMatch(t, []string{"AAA", "BBB"}, mapKeys(second.Bars))

	_, ok, err = stream.Next(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDailyBarFeedResamplesCanonicalDailyBarsToMonthlyFrames(t *testing.T) {
	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"AAA": {
			canonicalDailyBarForSymbol("AAA", "2024-01-30", "10", "12", "9", "11"),
			canonicalDailyBarForSymbol("AAA", "2024-01-31", "11", "13", "10", "12"),
			canonicalDailyBarForSymbol("AAA", "2024-02-01", "20", "21", "19", "20"),
		},
	}}
	from, err := time.Parse(time.DateOnly, "2024-01-30")
	require.NoError(t, err)
	to, err := time.Parse(time.DateOnly, "2024-02-01")
	require.NoError(t, err)

	stream, err := newDailyBarFeed(repo).Open(context.Background(), core.DataRequest{
		From:      from,
		To:        to,
		Timeframe: core.Timeframe1Month,
		Instruments: []core.InstrumentIdentity{
			{Symbol: "AAA", Market: "krx", SecurityType: "etf"},
		},
	})
	require.NoError(t, err)
	defer stream.Close()

	first, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2024-01-31", first.Time.Format(time.DateOnly))
	assert.Equal(t, core.Timeframe1Month, first.Bars["AAA"].Timeframe)
	assert.Equal(t, core.BarSessionRegular, first.Bars["AAA"].Session)
	assert.Equal(t, core.BarStatusOK, first.Bars["AAA"].Status)
	assert.InDelta(t, 10.0, first.Bars["AAA"].Open, 0.0001)
	assert.InDelta(t, 13.0, first.Bars["AAA"].High, 0.0001)
	assert.InDelta(t, 9.0, first.Bars["AAA"].Low, 0.0001)
	assert.InDelta(t, 12.0, first.Bars["AAA"].Close, 0.0001)

	second, ok, err := stream.Next(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2024-02-01", second.Time.Format(time.DateOnly))
	assert.Equal(t, core.Timeframe1Month, second.Bars["AAA"].Timeframe)
	assert.InDelta(t, 20.0, second.Bars["AAA"].Open, 0.0001)
	assert.InDelta(t, 20.0, second.Bars["AAA"].Close, 0.0001)
}

func TestServiceEvaluationParallelismIsDeterministic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluation.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleDeterministicEvaluationYAML()), 0o644))

	sequential := runEvaluationForParallelism(t, path, 1)
	parallel := runEvaluationForParallelism(t, path, 4)

	assertEvaluationRunDeterministic(t, sequential, parallel)
}

func TestServiceEvaluationStoresCaseHashesAndDataFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluation.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleDeterministicEvaluationYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	backtestRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, backtestRepo)
	require.NoError(t, err)

	_, err = service.RunEvaluation(context.Background(), path, RunEvaluationOptions{Parallelism: 2})
	require.NoError(t, err)

	detail, err := service.InspectEvaluation(context.Background(), "deterministic-evaluation")
	require.NoError(t, err)
	require.NotEmpty(t, detail.Cases)
	for _, item := range detail.Cases {
		assert.NotEmpty(t, item.StrategyHash)
		assert.NotEmpty(t, item.RunHash)
		assert.NotEmpty(t, item.EngineVersion)
		assert.NotEmpty(t, item.IndicatorRegistry)
		assert.NotEmpty(t, item.MetricRegistry)
		assert.NotEmpty(t, item.DataFingerprint)
		assert.NotEmpty(t, item.ResultHash)
	}
}

func TestServiceEvaluationRegimeBenchmarkDrivesTagsAndBenchmarkMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluation-regime.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleRegimeBenchmarkEvaluationYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
		"102110": sampleBenchmarkCanonicalDailyBars(),
	}}
	backtestRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, backtestRepo)
	require.NoError(t, err)

	result, err := service.RunEvaluation(context.Background(), path, RunEvaluationOptions{Parallelism: 1})
	require.NoError(t, err)

	require.NotEmpty(t, result.Cases)
	assert.Equal(t, "102110", result.Cases[0].Result.Benchmark.Symbol)
	assert.Contains(t, result.Cases[0].Metrics, core.MetricBenchmarkTotalReturn)
	assert.Contains(t, result.Cases[0].RegimeTags, "bull")

	detail, err := service.InspectEvaluation(context.Background(), "regime-benchmark-evaluation")
	require.NoError(t, err)
	require.NotEmpty(t, detail.Cases)
	assert.Contains(t, string(detail.Cases[0].RegimeTagsJSON), "bull")
}

func TestServiceEvaluationRegimeThresholdsDriveTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluation-regime-threshold.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleRegimeThresholdEvaluationYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	backtestRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, backtestRepo)
	require.NoError(t, err)

	result, err := service.RunEvaluation(context.Background(), path, RunEvaluationOptions{Parallelism: 1})
	require.NoError(t, err)

	require.NotEmpty(t, result.Cases)
	assert.Contains(t, result.Cases[0].RegimeTags, "bear")
	assert.Contains(t, result.Cases[0].RegimeTags, "high_vol")
}

func TestServiceWalkForwardStoresStepHashesAndDataFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluation.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleWalkForwardEvaluationYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	backtestRepo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(repo, backtestRepo)
	require.NoError(t, err)

	result, err := service.RunEvaluation(context.Background(), path, RunEvaluationOptions{Parallelism: 2})
	require.NoError(t, err)
	require.NotEmpty(t, result.WalkForward)
	assert.NotEmpty(t, result.WalkForward[0].DataFingerprint)

	detail, err := service.InspectEvaluation(context.Background(), "deterministic-walk-forward")
	require.NoError(t, err)
	require.NotEmpty(t, detail.WalkForward)
	for _, step := range detail.WalkForward {
		assert.NotEmpty(t, step.StrategyHash)
		assert.NotEmpty(t, step.RunHash)
		assert.NotEmpty(t, step.EngineVersion)
		assert.NotEmpty(t, step.IndicatorRegistry)
		assert.NotEmpty(t, step.MetricRegistry)
		assert.NotEmpty(t, step.DataFingerprint)
		assert.NotEmpty(t, step.ResultHash)
	}
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
	mu      sync.Mutex
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

type recordingScreenRunner struct {
	items    []strategyservice.ScreenRunItem
	requests []strategyservice.ScreenStrategyRequest
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

func (r *recordingScreenRunner) Screen(_ context.Context, req strategyservice.ScreenStrategyRequest) (strategyservice.ScreenRunDetail, error) {
	r.requests = append(r.requests, req)
	return strategyservice.ScreenRunDetail{Items: r.items}, nil
}

func (r *recordingDailyBarRepository) QueryDailyBars(_ context.Context, query daily.Query) ([]dailybar.Bar, error) {
	r.mu.Lock()
	r.queries = append(r.queries, query)
	r.mu.Unlock()
	if query.Symbol == "" {
		out := make([]dailybar.Bar, 0)
		for _, rows := range r.bars {
			for _, row := range rows {
				if query.Market != "" && row.Market != query.Market {
					continue
				}
				if query.SecurityType != "" && row.SecurityType != query.SecurityType {
					continue
				}
				out = append(out, row)
			}
		}
		sortDailyBars(out)
		return out, nil
	}
	out := make([]dailybar.Bar, 0, len(r.bars[query.Symbol]))
	for _, row := range r.bars[query.Symbol] {
		if query.Market != "" && row.Market != query.Market {
			continue
		}
		if query.SecurityType != "" && row.SecurityType != query.SecurityType {
			continue
		}
		out = append(out, row)
	}
	sortDailyBars(out)
	return out, nil
}

func (r *recordingDailyBarRepository) StreamDailyBars(ctx context.Context, query daily.Query) (daily.BarStream, error) {
	rows, err := r.QueryDailyBars(ctx, query)
	if err != nil {
		return nil, err
	}
	return &recordingDailyBarStream{rows: rows}, nil
}

type recordingDailyBarStream struct {
	rows   []dailybar.Bar
	offset int
}

func (s *recordingDailyBarStream) Next(ctx context.Context) (dailybar.Bar, bool, error) {
	if err := ctx.Err(); err != nil {
		return dailybar.Bar{}, false, err
	}
	if s.offset >= len(s.rows) {
		return dailybar.Bar{}, false, nil
	}
	row := s.rows[s.offset]
	s.offset++
	return row, true, nil
}

func (s *recordingDailyBarStream) Close() error {
	return nil
}

func sortDailyBars(rows []dailybar.Bar) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TradingDate != rows[j].TradingDate {
			return rows[i].TradingDate < rows[j].TradingDate
		}
		return rows[i].Symbol < rows[j].Symbol
	})
}

func mapKeys[K comparable, V any](values map[K]V) []K {
	keys := make([]K, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
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

func mixedCanonicalDailyBars(symbol string, securityType provider.SecurityType) []dailybar.Bar {
	return []dailybar.Bar{
		canonicalDailyBarForIdentity(symbol, securityType, "2024-01-02", "10", "10", "10", "10"),
		canonicalDailyBarForIdentity(symbol, securityType, "2024-01-03", "11", "11", "11", "11"),
		canonicalDailyBarForIdentity(symbol, securityType, "2024-01-04", "12", "12", "12", "12"),
	}
}

func canonicalDailyBarForIdentity(symbol string, securityType provider.SecurityType, date string, open string, high string, low string, closePrice string) dailybar.Bar {
	row := canonicalDailyBarForSymbol(symbol, date, open, high, low, closePrice)
	row.SecurityType = securityType
	return row
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
      - excess_return
      - benchmark_alpha
      - benchmark_beta`, 1)
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

func sampleMixedUniverseYAML() string {
	return `
kind: Strategy
schema_version: 1
name: mixed-universe
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
risk:
  max_positions: 3
---
kind: BacktestRun
schema_version: 1
name: mixed-universe-run
strategy:
  name: mixed-universe
data:
  market: krx
  timeframe: 1d
  from: 2024-01-03
  to: 2024-01-04
universe:
  pipeline:
    - id: combine.union
      params:
        pipelines:
          - pipeline:
              - id: source.daily_bars
                params:
                  lookback_days: 5
              - id: transform.latest_per_symbol
              - id: filter.security_type
                params:
                  value: stock
          - pipeline:
              - id: source.daily_bars
                params:
                  lookback_days: 5
              - id: transform.latest_per_symbol
              - id: filter.security_type
                params:
                  value: etf
          - pipeline:
              - id: source.daily_bars
                params:
                  lookback_days: 5
              - id: transform.latest_per_symbol
              - id: filter.security_type
                params:
                  value: etn
portfolio:
  initial_cash: 10000
execution:
  fill: next_open
`
}

func sampleDeterministicEvaluationYAML() string {
	return strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: deterministic-evaluation
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-08
parameters:
  indicators.trend.params.window: [2, 3]
  sizing.value: [25, 50]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc`, 1)
}

func sampleRegimeBenchmarkEvaluationYAML() string {
	payload := strings.Replace(sampleDeterministicEvaluationYAML(), "name: deterministic-evaluation", "name: regime-benchmark-evaluation", 1)
	return strings.Replace(payload, `ranking:
  objective: calmar
  order: desc`, `ranking:
  objective: calmar
  order: desc
regime:
  benchmark:
    symbol: "102110"
    name: benchmark`, 1)
}

func sampleRegimeThresholdEvaluationYAML() string {
	payload := strings.Replace(sampleDeterministicEvaluationYAML(), "name: deterministic-evaluation", "name: regime-threshold-evaluation", 1)
	return strings.Replace(payload, `ranking:
  objective: calmar
  order: desc`, `ranking:
  objective: calmar
  order: desc
regime:
  return_threshold: 0.01
  volatility_threshold: 0.01`, 1)
}

func sampleWalkForwardEvaluationYAML() string {
	return strings.Replace(sampleYAML(), `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, `---
kind: Evaluation
schema_version: 1
name: deterministic-walk-forward
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-05
parameters:
  indicators.trend.params.window: [2, 3]
  sizing.value: [25, 50]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc
walk_forward:
  train:
    days: 2
  test:
    days: 1
  step:
    days: 1
  select:
    objective: calmar
    order: desc`, 1)
}

func runEvaluationForParallelism(t *testing.T, path string, parallelism int) EvaluationRunResult {
	t.Helper()

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	service, err := NewServiceWithRepository(repo, newMemoryBacktestStrategyRepository())
	require.NoError(t, err)

	result, err := service.RunEvaluation(context.Background(), path, RunEvaluationOptions{Parallelism: parallelism})
	require.NoError(t, err)
	return result
}

func assertEvaluationRunDeterministic(t *testing.T, want EvaluationRunResult, got EvaluationRunResult) {
	t.Helper()

	require.Len(t, got.Cases, len(want.Cases))
	require.Len(t, got.Ranking, len(want.Ranking))

	for index := range want.Cases {
		assertEvaluationCaseDeterministic(t, want.Cases[index], got.Cases[index])
	}
	for index := range want.Ranking {
		assertEvaluationCaseDeterministic(t, want.Ranking[index], got.Ranking[index])
	}
}

func assertEvaluationCaseDeterministic(t *testing.T, want core.EvaluationCaseResult, got core.EvaluationCaseResult) {
	t.Helper()

	assert.Equal(t, want.CaseID, got.CaseID)
	assert.Equal(t, want.CaseName, got.CaseName)
	assert.Equal(t, want.Period, got.Period)
	assert.Equal(t, want.Parameters, got.Parameters)
	assert.Equal(t, want.Result.ResultHash, got.Result.ResultHash)
	assert.Equal(t, want.Rank, got.Rank)
	assert.Equal(t, want.Objective, got.Objective)
	assert.InDelta(t, want.ObjectiveValue, got.ObjectiveValue, 0.0000001)

	for _, metric := range []string{
		core.MetricTotalReturn,
		core.MetricMaxDrawdown,
		core.MetricTradeCount,
		core.MetricCalmar,
		core.MetricTurnover,
	} {
		require.Contains(t, got.Metrics, metric)
		require.Contains(t, want.Metrics, metric)
		assert.InDelta(t, want.Metrics[metric], got.Metrics[metric], 0.0000001, metric)
	}
}
