package universe

import (
	"context"
	"slices"
	"time"

	"github.com/samber/oops"
)

const (
	KindMarketRegime = "MarketRegime"
	RegimeUnknown    = "unknown"
)

type MarketRegimeSpec struct {
	Kind          string               `json:"kind" yaml:"kind"`
	SchemaVersion int                  `json:"schema_version" yaml:"schema_version"`
	Name          string               `json:"name" yaml:"name"`
	Spec          MarketRegimeBodySpec `json:"spec" yaml:"spec"`
}

type MarketRegimeBodySpec struct {
	Benchmark MarketRegimeBenchmarkSpec `json:"benchmark" yaml:"benchmark"`
	Rules     []MarketRegimeRuleSpec    `json:"rules" yaml:"rules"`
}

type MarketRegimeBenchmarkSpec struct {
	Symbol       string `json:"symbol" yaml:"symbol"`
	Market       string `json:"market,omitempty" yaml:"market,omitempty"`
	SecurityType string `json:"security_type,omitempty" yaml:"security_type,omitempty"`
}

type MarketRegimeRuleSpec struct {
	Regime string                    `json:"regime" yaml:"regime"`
	When   MarketRegimeConditionSpec `json:"when" yaml:"when"`
}

type MarketRegimeConditionSpec struct {
	Return20DGTE     *float64  `json:"return_20d_gte,omitempty" yaml:"return_20d_gte,omitempty"`
	Return20DLTE     *float64  `json:"return_20d_lte,omitempty" yaml:"return_20d_lte,omitempty"`
	Return20DBetween []float64 `json:"return_20d_between,omitempty" yaml:"return_20d_between,omitempty"`
	CloseAboveMA20   *bool     `json:"close_above_ma20,omitempty" yaml:"close_above_ma20,omitempty"`
	CloseBelowMA20   *bool     `json:"close_below_ma20,omitempty" yaml:"close_below_ma20,omitempty"`
	MA20AboveMA60    *bool     `json:"ma20_above_ma60,omitempty" yaml:"ma20_above_ma60,omitempty"`
	MA20BelowMA60    *bool     `json:"ma20_below_ma60,omitempty" yaml:"ma20_below_ma60,omitempty"`
}

type MarketRegimeResult struct {
	Kind             string                    `json:"kind"`
	Name             string                    `json:"name"`
	AsOf             string                    `json:"as_of"`
	Benchmark        MarketRegimeBenchmarkSpec `json:"benchmark"`
	Regime           string                    `json:"regime"`
	MatchedRuleIndex int                       `json:"matched_rule_index"`
	Metrics          MarketRegimeMetrics       `json:"metrics"`
}

type MarketRegimeMetrics struct {
	Close     float64 `json:"close"`
	Return20D float64 `json:"return_20d"`
	MA20      float64 `json:"ma20"`
	MA60      float64 `json:"ma60"`
	BarCount  int     `json:"bar_count"`
}

func EvaluateMarketRegime(ctx context.Context, spec MarketRegimeSpec, bars []Bar, asOf time.Time) (MarketRegimeResult, error) {
	if err := ctx.Err(); err != nil {
		return MarketRegimeResult{}, oops.In("market_regime").Wrap(err)
	}
	if err := ValidateMarketRegimeSpec(spec); err != nil {
		return MarketRegimeResult{}, err
	}
	filtered := benchmarkBars(spec.Spec.Benchmark, bars, asOf)
	metrics, err := marketRegimeMetrics(filtered)
	if err != nil {
		return MarketRegimeResult{}, err
	}
	result := MarketRegimeResult{
		Kind:             KindMarketRegime,
		Name:             spec.Name,
		AsOf:             asOf.Format(time.DateOnly),
		Benchmark:        spec.Spec.Benchmark,
		Regime:           RegimeUnknown,
		MatchedRuleIndex: -1,
		Metrics:          metrics,
	}
	for index, rule := range spec.Spec.Rules {
		if marketRegimeConditionMatches(rule.When, metrics) {
			result.Regime = rule.Regime
			result.MatchedRuleIndex = index
			return result, nil
		}
	}
	return result, nil
}

func ValidateMarketRegimeSpec(spec MarketRegimeSpec) error {
	errb := oops.In("market_regime").With("name", spec.Name)
	if spec.Kind != KindMarketRegime {
		return errb.With("kind", spec.Kind).New("market regime kind must be MarketRegime")
	}
	if spec.SchemaVersion != 0 && spec.SchemaVersion != 1 {
		return errb.With("schema_version", spec.SchemaVersion).New("unsupported market regime schema version")
	}
	if spec.Name == "" {
		return errb.New("market regime name is required")
	}
	if spec.Spec.Benchmark.Symbol == "" {
		return errb.New("market regime benchmark symbol is required")
	}
	if len(spec.Spec.Rules) == 0 {
		return errb.New("market regime requires at least one rule")
	}
	for index, rule := range spec.Spec.Rules {
		if rule.Regime == "" {
			return errb.With("index", index).New("market regime rule regime is required")
		}
		if len(rule.When.Return20DBetween) != 0 && len(rule.When.Return20DBetween) != 2 {
			return errb.With("index", index).New("return_20d_between requires two values")
		}
	}
	return nil
}

func benchmarkBars(benchmark MarketRegimeBenchmarkSpec, bars []Bar, asOf time.Time) []Bar {
	out := make([]Bar, 0, len(bars))
	for _, bar := range bars {
		if bar.Symbol != benchmark.Symbol {
			continue
		}
		if benchmark.Market != "" && bar.Market != benchmark.Market {
			continue
		}
		if benchmark.SecurityType != "" && bar.SecurityType != benchmark.SecurityType {
			continue
		}
		if bar.Time.After(asOf) {
			continue
		}
		out = append(out, bar)
	}
	slices.SortFunc(out, func(a, b Bar) int {
		return a.Time.Compare(b.Time)
	})
	return out
}

func marketRegimeMetrics(bars []Bar) (MarketRegimeMetrics, error) {
	if len(bars) < 61 {
		return MarketRegimeMetrics{}, oops.In("market_regime").With("bar_count", len(bars)).New("market regime benchmark requires at least 61 closed bars")
	}
	latest := bars[len(bars)-1]
	if latest.Close <= 0 {
		return MarketRegimeMetrics{}, oops.In("market_regime").With("symbol", latest.Symbol).New("market regime benchmark close must be positive")
	}
	start := bars[len(bars)-21].Close
	if start <= 0 {
		return MarketRegimeMetrics{}, oops.In("market_regime").With("symbol", latest.Symbol).New("market regime benchmark return base close must be positive")
	}
	return MarketRegimeMetrics{
		Close:     latest.Close,
		Return20D: latest.Close/start - 1,
		MA20:      averageClose(bars[len(bars)-20:]),
		MA60:      averageClose(bars[len(bars)-60:]),
		BarCount:  len(bars),
	}, nil
}

func averageClose(bars []Bar) float64 {
	var sum float64
	for _, bar := range bars {
		sum += bar.Close
	}
	return sum / float64(len(bars))
}

func marketRegimeConditionMatches(condition MarketRegimeConditionSpec, metrics MarketRegimeMetrics) bool {
	if condition.Return20DGTE != nil && metrics.Return20D < *condition.Return20DGTE {
		return false
	}
	if condition.Return20DLTE != nil && metrics.Return20D > *condition.Return20DLTE {
		return false
	}
	if len(condition.Return20DBetween) == 2 {
		if metrics.Return20D < condition.Return20DBetween[0] || metrics.Return20D > condition.Return20DBetween[1] {
			return false
		}
	}
	if condition.CloseAboveMA20 != nil && (metrics.Close > metrics.MA20) != *condition.CloseAboveMA20 {
		return false
	}
	if condition.CloseBelowMA20 != nil && (metrics.Close < metrics.MA20) != *condition.CloseBelowMA20 {
		return false
	}
	if condition.MA20AboveMA60 != nil && (metrics.MA20 > metrics.MA60) != *condition.MA20AboveMA60 {
		return false
	}
	if condition.MA20BelowMA60 != nil && (metrics.MA20 < metrics.MA60) != *condition.MA20BelowMA60 {
		return false
	}
	return true
}
