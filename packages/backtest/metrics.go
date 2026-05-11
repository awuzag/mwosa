package backtest

import (
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	MetricTotalReturn               = "total_return"
	MetricFinalEquity               = "final_equity"
	MetricMaxDrawdown               = "max_drawdown"
	MetricTradeCount                = "trade_count"
	MetricWinRate                   = "win_rate"
	MetricAverageTradeReturn        = "average_trade_return"
	MetricRealizedPnL               = "realized_pnl"
	MetricUnfilledCount             = "unfilled_count"
	MetricBenchmarkTotalReturn      = "benchmark_total_return"
	MetricExcessReturn              = "excess_return"
	MetricBenchmarkMaxDrawdown      = "benchmark_max_drawdown"
	MetricRelativeDrawdown          = "relative_drawdown"
	MetricMonthlyWinRateVsBenchmark = "monthly_win_rate_vs_benchmark"
)

var coreMetricIDs = []string{
	MetricTotalReturn,
	MetricFinalEquity,
	MetricMaxDrawdown,
	MetricTradeCount,
	MetricWinRate,
}

type Metrics map[string]float64

type MetricRegistry struct {
	defs map[string]MetricDefinition
}

type MetricDefinition struct {
	ID                string
	RequiresBenchmark bool
	Calculate         func(MetricContext) (float64, error)
}

type MetricContext struct {
	Result Result
	Series map[string][]Bar
}

func NewMetricRegistry(definitions ...MetricDefinition) (MetricRegistry, error) {
	registry := MetricRegistry{defs: make(map[string]MetricDefinition, len(definitions))}
	for _, definition := range definitions {
		if strings.TrimSpace(definition.ID) == "" {
			return MetricRegistry{}, oops.In("backtest_metric_registry").New("metric id is empty")
		}
		if definition.Calculate == nil {
			return MetricRegistry{}, oops.In("backtest_metric_registry").With("metric", definition.ID).New("metric calculator is nil")
		}
		if _, exists := registry.defs[definition.ID]; exists {
			return MetricRegistry{}, oops.In("backtest_metric_registry").With("metric", definition.ID).New("metric id is duplicated")
		}
		registry.defs[definition.ID] = definition
	}
	return registry, nil
}

func DefaultMetricRegistry() (MetricRegistry, error) {
	return NewMetricRegistry(
		fieldMetric(MetricTotalReturn, func(result Result) float64 { return result.TotalReturn }),
		fieldMetric(MetricFinalEquity, func(result Result) float64 { return result.FinalEquity }),
		fieldMetric(MetricMaxDrawdown, func(result Result) float64 { return result.MaxDrawdown }),
		fieldMetric(MetricTradeCount, func(result Result) float64 { return float64(result.TradeCount) }),
		fieldMetric(MetricWinRate, func(result Result) float64 { return result.WinRate }),
		fieldMetric(MetricAverageTradeReturn, func(result Result) float64 { return result.AverageTradeRet }),
		fieldMetric(MetricRealizedPnL, func(result Result) float64 { return result.RealizedPnL }),
		fieldMetric(MetricUnfilledCount, func(result Result) float64 { return float64(result.UnfilledCount) }),
		benchmarkMetric(MetricBenchmarkTotalReturn, benchmarkTotalReturn),
		benchmarkMetric(MetricExcessReturn, func(ctx MetricContext) (float64, error) {
			benchmarkReturn, err := benchmarkTotalReturn(ctx)
			if err != nil {
				return 0, err
			}
			return ctx.Result.TotalReturn - benchmarkReturn, nil
		}),
		benchmarkMetric(MetricBenchmarkMaxDrawdown, benchmarkMaxDrawdown),
		benchmarkMetric(MetricRelativeDrawdown, func(ctx MetricContext) (float64, error) {
			benchmarkDD, err := benchmarkMaxDrawdown(ctx)
			if err != nil {
				return 0, err
			}
			return ctx.Result.MaxDrawdown - benchmarkDD, nil
		}),
		benchmarkMetric(MetricMonthlyWinRateVsBenchmark, monthlyWinRateVsBenchmark),
	)
}

func (r MetricRegistry) Definition(id string) (MetricDefinition, bool) {
	definition, ok := r.defs[id]
	return definition, ok
}

func fieldMetric(id string, value func(Result) float64) MetricDefinition {
	return MetricDefinition{
		ID: id,
		Calculate: func(ctx MetricContext) (float64, error) {
			return value(ctx.Result), nil
		},
	}
}

func benchmarkMetric(id string, calculate func(MetricContext) (float64, error)) MetricDefinition {
	return MetricDefinition{
		ID:                id,
		RequiresBenchmark: true,
		Calculate:         calculate,
	}
}

func resolveMetricSelection(report ReportSpec, hasBenchmark bool, registry MetricRegistry) ([]string, error) {
	errb := oops.In("backtest_metric_selection")
	selection := report.Metrics
	if selection.Preset != "" && selection.Preset != "core" {
		return nil, errb.With("preset", selection.Preset).New("unsupported metric preset")
	}

	selected := make([]string, 0, len(coreMetricIDs)+len(selection.Include))
	seen := make(map[string]struct{})
	for _, id := range coreMetricIDs {
		selected = appendMetric(selected, seen, id)
	}
	for _, id := range selection.Include {
		definition, err := requireMetric(registry, id)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		if definition.RequiresBenchmark && !hasBenchmark {
			return nil, errb.With("metric", id).Errorf("benchmark metric requires benchmark: metric=%s", id)
		}
		selected = appendMetric(selected, seen, id)
	}
	for _, id := range selection.Exclude {
		definition, err := requireMetric(registry, id)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		if definition.RequiresBenchmark && !hasBenchmark {
			return nil, errb.With("metric", id).Errorf("benchmark metric requires benchmark: metric=%s", id)
		}
		if _, ok := seen[id]; !ok {
			return nil, errb.With("metric", id).New("metric exclude target is not selected")
		}
		delete(seen, id)
		next := selected[:0]
		for _, selectedID := range selected {
			if selectedID != id {
				next = append(next, selectedID)
			}
		}
		selected = next
	}
	return selected, nil
}

func requireMetric(registry MetricRegistry, id string) (MetricDefinition, error) {
	definition, ok := registry.Definition(id)
	if !ok {
		return MetricDefinition{}, oops.In("backtest_metric_registry").With("metric", id).Errorf("metric is not registered: metric=%s", id)
	}
	return definition, nil
}

func appendMetric(selected []string, seen map[string]struct{}, id string) []string {
	if _, ok := seen[id]; ok {
		return selected
	}
	seen[id] = struct{}{}
	return append(selected, id)
}

func calculateSelectedMetrics(plan StrategyPlan, result Result, series map[string][]Bar) (Metrics, error) {
	out := make(Metrics, len(plan.SelectedMetrics))
	ctx := MetricContext{Result: result, Series: series}
	for _, id := range plan.SelectedMetrics {
		definition, ok := plan.metricRegistry.Definition(id)
		if !ok {
			return nil, oops.In("backtest_metrics").With("metric", id).New("metric is not registered")
		}
		value, err := definition.Calculate(ctx)
		if err != nil {
			return nil, oops.In("backtest_metrics").With("metric", id).Wrap(err)
		}
		out[id] = value
	}
	return out, nil
}

func benchmarkTotalReturn(ctx MetricContext) (float64, error) {
	bars, err := benchmarkBarsForMetrics(ctx)
	if err != nil {
		return 0, err
	}
	first := bars[0].Close
	last := bars[len(bars)-1].Close
	if first <= 0 {
		return 0, oops.In("backtest_metrics").New("benchmark first close must be positive")
	}
	return last/first - 1, nil
}

func benchmarkMaxDrawdown(ctx MetricContext) (float64, error) {
	bars, err := benchmarkBarsForMetrics(ctx)
	if err != nil {
		return 0, err
	}
	curve := make([]EquityPoint, 0, len(bars))
	for _, bar := range bars {
		curve = append(curve, EquityPoint{Time: bar.Time, Equity: bar.Close})
	}
	return maxDrawdown(curve), nil
}

func monthlyWinRateVsBenchmark(ctx MetricContext) (float64, error) {
	bars, err := benchmarkBarsForMetrics(ctx)
	if err != nil {
		return 0, err
	}
	strategyReturns := monthlyReturnsFromEquity(ctx.Result.EquityCurve)
	benchmarkReturns := monthlyReturnsFromBars(bars)
	var wins int
	var compared int
	for month, strategyReturn := range strategyReturns {
		benchmarkReturn, ok := benchmarkReturns[month]
		if !ok {
			continue
		}
		compared++
		if strategyReturn > benchmarkReturn {
			wins++
		}
	}
	if compared == 0 {
		return 0, nil
	}
	return float64(wins) / float64(compared), nil
}

func benchmarkBarsForMetrics(ctx MetricContext) ([]Bar, error) {
	symbol := ctx.Result.Benchmark.Symbol
	if strings.TrimSpace(symbol) == "" {
		return nil, oops.In("backtest_metrics").New("benchmark is required")
	}
	bars := ctx.Series[symbol]
	if len(bars) < 2 {
		return nil, oops.In("backtest_metrics").With("symbol", symbol).New("benchmark requires at least two bars")
	}
	return bars, nil
}

func monthlyReturnsFromEquity(curve []EquityPoint) map[string]float64 {
	type monthSpan struct {
		first float64
		last  float64
	}
	spans := make(map[string]monthSpan)
	for _, point := range curve {
		key := monthKey(point.Time)
		span := spans[key]
		if span.first == 0 {
			span.first = point.Equity
		}
		span.last = point.Equity
		spans[key] = span
	}
	out := make(map[string]float64, len(spans))
	for key, span := range spans {
		if span.first > 0 {
			out[key] = span.last/span.first - 1
		}
	}
	return out
}

func monthlyReturnsFromBars(bars []Bar) map[string]float64 {
	type monthSpan struct {
		first float64
		last  float64
	}
	spans := make(map[string]monthSpan)
	for _, bar := range bars {
		key := monthKey(bar.Time)
		span := spans[key]
		if span.first == 0 {
			span.first = bar.Close
		}
		span.last = bar.Close
		spans[key] = span
	}
	out := make(map[string]float64, len(spans))
	for key, span := range spans {
		if span.first > 0 {
			out[key] = span.last/span.first - 1
		}
	}
	return out
}

func monthKey(value time.Time) string {
	return value.Format("2006-01")
}
