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

	defaultMarketRegimeLookbackDays = 1
	defaultMarketRegimeConfirmDays  = 1
	marketRegimeMetricsBarCount     = 61
)

type MarketRegimeSpec struct {
	Kind          string               `json:"kind" yaml:"kind"`
	SchemaVersion int                  `json:"schema_version" yaml:"schema_version"`
	Name          string               `json:"name" yaml:"name"`
	Spec          MarketRegimeBodySpec `json:"spec" yaml:"spec"`
}

type MarketRegimeBodySpec struct {
	Benchmark  MarketRegimeBenchmarkSpec  `json:"benchmark" yaml:"benchmark"`
	Evaluation MarketRegimeEvaluationSpec `json:"evaluation,omitempty" yaml:"evaluation,omitempty"`
	Rules      []MarketRegimeRuleSpec     `json:"rules" yaml:"rules"`
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

type MarketRegimeEvaluationSpec struct {
	LookbackDays  int      `json:"lookback_days,omitempty" yaml:"lookback_days,omitempty"`
	ConfirmDays   int      `json:"confirm_days,omitempty" yaml:"confirm_days,omitempty"`
	MinConfidence *float64 `json:"min_confidence,omitempty" yaml:"min_confidence,omitempty"`
}

type MarketRegimeResult struct {
	Kind              string                         `json:"kind"`
	Name              string                         `json:"name"`
	AsOf              string                         `json:"as_of"`
	Benchmark         MarketRegimeBenchmarkSpec      `json:"benchmark"`
	Evaluation        MarketRegimeEvaluationSpec     `json:"evaluation"`
	Regime            string                         `json:"regime"`
	MatchedRuleIndex  int                            `json:"matched_rule_index"`
	Metrics           MarketRegimeMetrics            `json:"metrics"`
	Confidence        float64                        `json:"confidence"`
	StableDays        int                            `json:"stable_days"`
	Transitions       int                            `json:"transitions"`
	Confirmed         bool                           `json:"confirmed"`
	RecentRegimes     []string                       `json:"recent_regimes"`
	RecentEvaluations []MarketRegimeRecentEvaluation `json:"recent_evaluations"`
	Evidence          []MarketRegimeEvidence         `json:"evidence"`
	RuleEvaluations   []MarketRegimeRuleEvaluation   `json:"rule_evaluations"`
}

type MarketRegimeMetrics struct {
	Close     float64 `json:"close"`
	Return20D float64 `json:"return_20d"`
	MA20      float64 `json:"ma20"`
	MA60      float64 `json:"ma60"`
	BarCount  int     `json:"bar_count"`
}

type MarketRegimeRecentEvaluation struct {
	AsOf             string              `json:"as_of"`
	Regime           string              `json:"regime"`
	MatchedRuleIndex int                 `json:"matched_rule_index"`
	Metrics          MarketRegimeMetrics `json:"metrics"`
}

type MarketRegimeEvidence struct {
	Code           string   `json:"code"`
	Field          string   `json:"field"`
	Op             string   `json:"op"`
	Actual         float64  `json:"actual"`
	Threshold      *float64 `json:"threshold,omitempty"`
	ThresholdMax   *float64 `json:"threshold_max,omitempty"`
	ThresholdField string   `json:"threshold_field,omitempty"`
	Passed         bool     `json:"passed"`
}

type MarketRegimeRuleEvaluation struct {
	RuleIndex int                    `json:"rule_index"`
	Regime    string                 `json:"regime"`
	Matched   bool                   `json:"matched"`
	Evidence  []MarketRegimeEvidence `json:"evidence"`
}

func EvaluateMarketRegime(ctx context.Context, spec MarketRegimeSpec, bars []Bar, asOf time.Time) (MarketRegimeResult, error) {
	if err := ctx.Err(); err != nil {
		return MarketRegimeResult{}, oops.In("market_regime").Wrap(err)
	}
	if err := ValidateMarketRegimeSpec(spec); err != nil {
		return MarketRegimeResult{}, err
	}
	evaluation, err := NormalizeMarketRegimeEvaluationSpec(spec.Spec.Evaluation)
	if err != nil {
		return MarketRegimeResult{}, err
	}
	filtered := benchmarkBars(spec.Spec.Benchmark, bars, asOf)
	if len(filtered) == 0 {
		return MarketRegimeResult{}, oops.In("market_regime").With("symbol", spec.Spec.Benchmark.Symbol, "as_of", asOf.Format(time.DateOnly)).New("market regime benchmark bar missing for as_of")
	}
	latest := filtered[len(filtered)-1]
	if latest.Time.Format(time.DateOnly) != asOf.Format(time.DateOnly) {
		return MarketRegimeResult{}, oops.In("market_regime").With("symbol", spec.Spec.Benchmark.Symbol, "as_of", asOf.Format(time.DateOnly), "latest_bar", latest.Time.Format(time.DateOnly)).New("market regime benchmark bar missing for as_of")
	}
	requiredBars := MarketRegimeRequiredBarCount(evaluation)
	if len(filtered) < requiredBars {
		return MarketRegimeResult{}, oops.In("market_regime").
			With("symbol", spec.Spec.Benchmark.Symbol, "as_of", asOf.Format(time.DateOnly), "bar_count", len(filtered), "required_bar_count", requiredBars, "lookback_days", evaluation.LookbackDays).
			New("market regime benchmark does not have enough closed bars for evaluation")
	}
	recent := make([]marketRegimePoint, 0, evaluation.LookbackDays)
	for index := len(filtered) - evaluation.LookbackDays; index < len(filtered); index++ {
		point, err := evaluateMarketRegimePoint(spec, filtered[:index+1])
		if err != nil {
			return MarketRegimeResult{}, err
		}
		recent = append(recent, point)
	}
	final := recent[len(recent)-1]
	confidence := marketRegimeConfidence(final.Regime, recent)
	stableDays := marketRegimeStableDays(final.Regime, recent)
	transitions := marketRegimeTransitions(recent)
	recentRegimes := make([]string, 0, len(recent))
	recentEvaluations := make([]MarketRegimeRecentEvaluation, 0, len(recent))
	for _, item := range recent {
		recentRegimes = append(recentRegimes, item.Regime)
		recentEvaluations = append(recentEvaluations, MarketRegimeRecentEvaluation{
			AsOf:             item.AsOf,
			Regime:           item.Regime,
			MatchedRuleIndex: item.MatchedRuleIndex,
			Metrics:          item.Metrics,
		})
	}
	confirmed := stableDays >= evaluation.ConfirmDays
	if evaluation.MinConfidence != nil {
		confirmed = confirmed && confidence >= *evaluation.MinConfidence
	}
	result := MarketRegimeResult{
		Kind:              KindMarketRegime,
		Name:              spec.Name,
		AsOf:              asOf.Format(time.DateOnly),
		Benchmark:         spec.Spec.Benchmark,
		Evaluation:        evaluation,
		Regime:            final.Regime,
		MatchedRuleIndex:  final.MatchedRuleIndex,
		Metrics:           final.Metrics,
		Confidence:        confidence,
		StableDays:        stableDays,
		Transitions:       transitions,
		Confirmed:         confirmed,
		RecentRegimes:     recentRegimes,
		RecentEvaluations: recentEvaluations,
		Evidence:          final.Evidence,
		RuleEvaluations:   final.RuleEvaluations,
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
	if _, err := NormalizeMarketRegimeEvaluationSpec(spec.Spec.Evaluation); err != nil {
		return err
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

func NormalizeMarketRegimeEvaluationSpec(spec MarketRegimeEvaluationSpec) (MarketRegimeEvaluationSpec, error) {
	if spec.LookbackDays == 0 {
		spec.LookbackDays = defaultMarketRegimeLookbackDays
	}
	if spec.ConfirmDays == 0 {
		spec.ConfirmDays = defaultMarketRegimeConfirmDays
	}
	errb := oops.In("market_regime").With("lookback_days", spec.LookbackDays, "confirm_days", spec.ConfirmDays)
	if spec.LookbackDays < 0 {
		return MarketRegimeEvaluationSpec{}, errb.New("market regime evaluation lookback_days must be positive")
	}
	if spec.ConfirmDays < 0 {
		return MarketRegimeEvaluationSpec{}, errb.New("market regime evaluation confirm_days must be positive")
	}
	if spec.LookbackDays == 0 || spec.ConfirmDays == 0 {
		return MarketRegimeEvaluationSpec{}, errb.New("market regime evaluation days must be positive")
	}
	if spec.ConfirmDays > spec.LookbackDays {
		return MarketRegimeEvaluationSpec{}, errb.New("market regime evaluation confirm_days must not exceed lookback_days")
	}
	if spec.MinConfidence != nil && (*spec.MinConfidence < 0 || *spec.MinConfidence > 1) {
		return MarketRegimeEvaluationSpec{}, errb.With("min_confidence", *spec.MinConfidence).New("market regime evaluation min_confidence must be between 0 and 1")
	}
	return spec, nil
}

func MarketRegimeRequiredBarCount(evaluation MarketRegimeEvaluationSpec) int {
	if evaluation.LookbackDays <= 0 {
		evaluation.LookbackDays = defaultMarketRegimeLookbackDays
	}
	return marketRegimeMetricsBarCount + evaluation.LookbackDays - 1
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
	if len(bars) < marketRegimeMetricsBarCount {
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
	return marketRegimeConditionEvidence(condition, metrics).matched
}

type marketRegimePoint struct {
	AsOf             string
	Regime           string
	MatchedRuleIndex int
	Metrics          MarketRegimeMetrics
	Evidence         []MarketRegimeEvidence
	RuleEvaluations  []MarketRegimeRuleEvaluation
}

type marketRegimeConditionEvaluation struct {
	matched  bool
	evidence []MarketRegimeEvidence
}

func evaluateMarketRegimePoint(spec MarketRegimeSpec, bars []Bar) (marketRegimePoint, error) {
	metrics, err := marketRegimeMetrics(bars)
	if err != nil {
		return marketRegimePoint{}, err
	}
	point := marketRegimePoint{
		AsOf:             bars[len(bars)-1].Time.Format(time.DateOnly),
		Regime:           RegimeUnknown,
		MatchedRuleIndex: -1,
		Metrics:          metrics,
		RuleEvaluations:  make([]MarketRegimeRuleEvaluation, 0, len(spec.Spec.Rules)),
	}
	for index, rule := range spec.Spec.Rules {
		evaluation := marketRegimeConditionEvidence(rule.When, metrics)
		point.RuleEvaluations = append(point.RuleEvaluations, MarketRegimeRuleEvaluation{
			RuleIndex: index,
			Regime:    rule.Regime,
			Matched:   evaluation.matched,
			Evidence:  evaluation.evidence,
		})
		if point.MatchedRuleIndex == -1 && evaluation.matched {
			point.Regime = rule.Regime
			point.MatchedRuleIndex = index
			point.Evidence = evaluation.evidence
		}
	}
	return point, nil
}

func marketRegimeConditionEvidence(condition MarketRegimeConditionSpec, metrics MarketRegimeMetrics) marketRegimeConditionEvaluation {
	evidence := make([]MarketRegimeEvidence, 0, 7)
	if condition.Return20DGTE != nil && metrics.Return20D < *condition.Return20DGTE {
		evidence = append(evidence, thresholdEvidence("return_20d_gte", "return_20d", "gte", metrics.Return20D, condition.Return20DGTE, nil, "", false))
	} else if condition.Return20DGTE != nil {
		evidence = append(evidence, thresholdEvidence("return_20d_gte", "return_20d", "gte", metrics.Return20D, condition.Return20DGTE, nil, "", true))
	}
	if condition.Return20DLTE != nil && metrics.Return20D > *condition.Return20DLTE {
		evidence = append(evidence, thresholdEvidence("return_20d_lte", "return_20d", "lte", metrics.Return20D, condition.Return20DLTE, nil, "", false))
	} else if condition.Return20DLTE != nil {
		evidence = append(evidence, thresholdEvidence("return_20d_lte", "return_20d", "lte", metrics.Return20D, condition.Return20DLTE, nil, "", true))
	}
	if len(condition.Return20DBetween) == 2 {
		minValue := condition.Return20DBetween[0]
		maxValue := condition.Return20DBetween[1]
		passed := metrics.Return20D >= minValue && metrics.Return20D <= maxValue
		evidence = append(evidence, thresholdEvidence("return_20d_between", "return_20d", "between", metrics.Return20D, &minValue, &maxValue, "", passed))
	}
	if condition.CloseAboveMA20 != nil {
		passed := (metrics.Close > metrics.MA20) == *condition.CloseAboveMA20
		evidence = append(evidence, thresholdEvidence("close_above_ma20", "close", "gt", metrics.Close, &metrics.MA20, nil, "ma20", passed))
	}
	if condition.CloseBelowMA20 != nil {
		passed := (metrics.Close < metrics.MA20) == *condition.CloseBelowMA20
		evidence = append(evidence, thresholdEvidence("close_below_ma20", "close", "lt", metrics.Close, &metrics.MA20, nil, "ma20", passed))
	}
	if condition.MA20AboveMA60 != nil {
		passed := (metrics.MA20 > metrics.MA60) == *condition.MA20AboveMA60
		evidence = append(evidence, thresholdEvidence("ma20_above_ma60", "ma20", "gt", metrics.MA20, &metrics.MA60, nil, "ma60", passed))
	}
	if condition.MA20BelowMA60 != nil {
		passed := (metrics.MA20 < metrics.MA60) == *condition.MA20BelowMA60
		evidence = append(evidence, thresholdEvidence("ma20_below_ma60", "ma20", "lt", metrics.MA20, &metrics.MA60, nil, "ma60", passed))
	}
	for _, item := range evidence {
		if !item.Passed {
			return marketRegimeConditionEvaluation{matched: false, evidence: evidence}
		}
	}
	return marketRegimeConditionEvaluation{matched: true, evidence: evidence}
}

func thresholdEvidence(code string, field string, op string, actual float64, threshold *float64, thresholdMax *float64, thresholdField string, passed bool) MarketRegimeEvidence {
	return MarketRegimeEvidence{
		Code:           code,
		Field:          field,
		Op:             op,
		Actual:         actual,
		Threshold:      threshold,
		ThresholdMax:   thresholdMax,
		ThresholdField: thresholdField,
		Passed:         passed,
	}
}

func marketRegimeConfidence(finalRegime string, recent []marketRegimePoint) float64 {
	if len(recent) == 0 {
		return 0
	}
	matches := 0
	for _, item := range recent {
		if item.Regime == finalRegime {
			matches++
		}
	}
	return float64(matches) / float64(len(recent))
}

func marketRegimeStableDays(finalRegime string, recent []marketRegimePoint) int {
	stable := 0
	for index := len(recent) - 1; index >= 0; index-- {
		if recent[index].Regime != finalRegime {
			break
		}
		stable++
	}
	return stable
}

func marketRegimeTransitions(recent []marketRegimePoint) int {
	transitions := 0
	for index := 1; index < len(recent); index++ {
		if recent[index-1].Regime != recent[index].Regime {
			transitions++
		}
	}
	return transitions
}
