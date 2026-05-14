package backtest

import (
	"math"
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
	MetricCAGR                      = "cagr"
	MetricVolatility                = "volatility"
	MetricSharpe                    = "sharpe"
	MetricCalmar                    = "calmar"
	MetricTurnover                  = "turnover"
	MetricProfitFactor              = "profit_factor"
	MetricExposure                  = "exposure"
	MetricDataIssueCount            = "data_issue_count"
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

var researchMetricIDs = []string{
	MetricTotalReturn,
	MetricCAGR,
	MetricMaxDrawdown,
	MetricVolatility,
	MetricSharpe,
	MetricCalmar,
	MetricTurnover,
	MetricTradeCount,
	MetricWinRate,
	MetricProfitFactor,
	MetricExposure,
	MetricUnfilledCount,
	MetricDataIssueCount,
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
		fieldMetric(MetricCAGR, cagr),
		fieldMetric(MetricVolatility, volatility),
		fieldMetric(MetricSharpe, sharpe),
		fieldMetric(MetricCalmar, calmar),
		fieldMetric(MetricTurnover, turnover),
		fieldMetric(MetricProfitFactor, profitFactor),
		fieldMetric(MetricExposure, exposure),
		fieldMetric(MetricDataIssueCount, dataIssueCount),
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
	if selection.Preset != "" && selection.Preset != "core" && selection.Preset != "research" {
		return nil, errb.With("preset", selection.Preset).New("unsupported metric preset")
	}

	preset := coreMetricIDs
	if selection.Preset == "research" {
		preset = researchMetricIDs
	}
	selected := make([]string, 0, len(preset)+len(selection.Include))
	seen := make(map[string]struct{})
	for _, id := range preset {
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

func ResolveMetricSelection(selection MetricSelectionSpec, hasBenchmark bool) ([]string, error) {
	registry, err := DefaultMetricRegistry()
	if err != nil {
		return nil, err
	}
	return resolveMetricSelection(ReportSpec{Metrics: selection}, hasBenchmark, registry)
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

func cagr(result Result) float64 {
	if result.InitialCash <= 0 || result.FinalEquity <= 0 || len(result.EquityCurve) < 2 {
		return 0
	}
	years := result.EquityCurve[len(result.EquityCurve)-1].Time.Sub(result.EquityCurve[0].Time).Hours() / 24 / 365.25
	if years <= 0 {
		return 0
	}
	return math.Pow(result.FinalEquity/result.InitialCash, 1/years) - 1
}

func volatility(result Result) float64 {
	return annualizedReturnStdDev(equityReturns(result.EquityCurve))
}

func sharpe(result Result) float64 {
	return annualizedSharpe(equityReturns(result.EquityCurve))
}

func calmar(result Result) float64 {
	dd := math.Abs(result.MaxDrawdown)
	if dd == 0 {
		return 0
	}
	return cagr(result) / dd
}

func turnover(result Result) float64 {
	if len(result.Trades) == 0 || len(result.EquityCurve) == 0 {
		return 0
	}
	var notional float64
	for _, trade := range result.Trades {
		notional += trade.Notional
	}
	avgEquity := averageEquity(result.EquityCurve)
	if avgEquity <= 0 {
		return 0
	}
	return notional / avgEquity
}

func profitFactor(result Result) float64 {
	var grossProfit float64
	var grossLoss float64
	for _, trade := range result.Trades {
		if trade.Side != SideSell {
			continue
		}
		if trade.RealizedPnL > 0 {
			grossProfit += trade.RealizedPnL
		}
		if trade.RealizedPnL < 0 {
			grossLoss += math.Abs(trade.RealizedPnL)
		}
	}
	if grossLoss == 0 {
		return 0
	}
	return grossProfit / grossLoss
}

func exposure(result Result) float64 {
	if len(result.EquityCurve) == 0 {
		return 0
	}
	var total float64
	for _, point := range result.EquityCurve {
		if point.Equity <= 0 {
			continue
		}
		total += point.PositionsValue / point.Equity
	}
	return total / float64(len(result.EquityCurve))
}

func dataIssueCount(result Result) float64 {
	var count int
	for _, event := range result.ExecutionEvents {
		switch event.Type {
		case "deferred_no_bar", "deferred_no_trade_bar":
			count++
		}
	}
	return float64(count)
}

func equityReturns(curve []EquityPoint) []float64 {
	if len(curve) < 2 {
		return nil
	}
	out := make([]float64, 0, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		prev := curve[i-1].Equity
		if prev <= 0 {
			continue
		}
		out = append(out, curve[i].Equity/prev-1)
	}
	return out
}

func annualizedReturnStdDev(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := average(returns)
	var variance float64
	for _, value := range returns {
		delta := value - mean
		variance += delta * delta
	}
	variance /= float64(len(returns) - 1)
	return math.Sqrt(variance) * math.Sqrt(252)
}

func annualizedSharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	stddev := annualizedReturnStdDev(returns)
	if stddev == 0 {
		return 0
	}
	return average(returns) * 252 / stddev
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func averageEquity(curve []EquityPoint) float64 {
	if len(curve) == 0 {
		return 0
	}
	var total float64
	for _, point := range curve {
		total += point.Equity
	}
	return total / float64(len(curve))
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
