package financialscores

import "github.com/ev3rlit/mwosa/packages/financialmetrics"

const Version = "fundamentals-score/v1"

type Input struct {
	Metrics   map[string]Metric
	Valuation *Valuation
}

type Metric struct {
	ValueBP            *int64
	UncomputableReason string
}

type Valuation struct {
	PerBP           *int64
	PbrBP           *int64
	PsrBP           *int64
	DividendYieldBP *int64
}

type Scores struct {
	ScoreVersion   string               `json:"score_version"`
	ValuationScore *int                 `json:"valuation_score,omitempty"`
	QualityScore   *int                 `json:"quality_score,omitempty"`
	GrowthScore    *int                 `json:"growth_score,omitempty"`
	Components     map[string]Component `json:"components,omitempty"`
	Uncomputable   map[string]string    `json:"uncomputable,omitempty"`
}

type Component struct {
	Source  string `json:"source"`
	ValueBP *int64 `json:"value_bp,omitempty"`
	Score   int    `json:"score"`
}

func Calculate(input Input) Scores {
	scores := Scores{
		ScoreVersion: Version,
		Components:   map[string]Component{},
		Uncomputable: map[string]string{},
	}

	scores.QualityScore = scoreQuality(input.Metrics, scores.Components, scores.Uncomputable)
	scores.GrowthScore = scoreGrowth(input.Metrics, scores.Components, scores.Uncomputable)
	scores.ValuationScore = scoreValuation(input.Valuation, scores.Components, scores.Uncomputable)

	if len(scores.Components) == 0 {
		scores.Components = nil
	}
	if len(scores.Uncomputable) == 0 {
		scores.Uncomputable = nil
	}
	return scores
}

func (s Scores) HasSignal() bool {
	return s.ValuationScore != nil || s.QualityScore != nil || s.GrowthScore != nil || len(s.Uncomputable) > 0
}

func scoreQuality(metrics map[string]Metric, components map[string]Component, uncomputable map[string]string) *int {
	values := []int{}
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.roe", financialmetrics.MetricROE, 0, 2500, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.roa", financialmetrics.MetricROA, 0, 1200, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.operating_margin", financialmetrics.MetricOperatingMargin, 0, 2500, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.net_margin", financialmetrics.MetricNetMargin, 0, 2000, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.debt_to_equity", financialmetrics.MetricDebtToEquity, 5000, 30000, true)
	values = appendMetricScore(values, components, uncomputable, metrics, "quality.current_ratio", financialmetrics.MetricCurrentRatio, 5000, 20000, false)
	if len(values) == 0 {
		uncomputable["quality_score"] = "no usable quality metrics"
		return nil
	}
	return intPtr(averageInt(values))
}

func scoreGrowth(metrics map[string]Metric, components map[string]Component, uncomputable map[string]string) *int {
	values := []int{}
	values = appendMetricScore(values, components, uncomputable, metrics, "growth.revenue_growth_yoy", financialmetrics.MetricRevenueGrowthYoY, -2000, 3000, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "growth.operating_income_growth_yoy", financialmetrics.MetricOperatingIncomeGrowthYoY, -3000, 5000, false)
	values = appendMetricScore(values, components, uncomputable, metrics, "growth.net_income_growth_yoy", financialmetrics.MetricNetIncomeGrowthYoY, -3000, 5000, false)
	if len(values) == 0 {
		uncomputable["growth_score"] = "no usable growth metrics"
		return nil
	}
	return intPtr(averageInt(values))
}

func scoreValuation(valuation *Valuation, components map[string]Component, uncomputable map[string]string) *int {
	if valuation == nil {
		uncomputable["valuation_score"] = "valuation snapshot is missing"
		return nil
	}
	values := []int{}
	values = appendValuationScore(values, components, "valuation.per", "valuation.per_bp", valuation.PerBP, 80000, 300000, true)
	values = appendValuationScore(values, components, "valuation.pbr", "valuation.pbr_bp", valuation.PbrBP, 7000, 30000, true)
	values = appendValuationScore(values, components, "valuation.psr", "valuation.psr_bp", valuation.PsrBP, 7000, 50000, true)
	values = appendValuationScore(values, components, "valuation.dividend_yield", "valuation.dividend_yield_bp", valuation.DividendYieldBP, 0, 500, false)
	if len(values) == 0 {
		uncomputable["valuation_score"] = "no usable valuation metrics"
		return nil
	}
	return intPtr(averageInt(values))
}

func appendMetricScore(values []int, components map[string]Component, uncomputable map[string]string, metrics map[string]Metric, key string, metricName string, low int64, high int64, lowerIsBetter bool) []int {
	metric, ok := metrics[metricName]
	if !ok {
		return values
	}
	if metric.ValueBP == nil {
		if metric.UncomputableReason != "" {
			uncomputable[key] = metric.UncomputableReason
		} else {
			uncomputable[key] = "metric value is missing"
		}
		return values
	}
	score := scaledScore(*metric.ValueBP, low, high, lowerIsBetter)
	components[key] = Component{
		Source:  "financial_metrics." + metricName + ".value_bp",
		ValueBP: metric.ValueBP,
		Score:   score,
	}
	return append(values, score)
}

func appendValuationScore(values []int, components map[string]Component, key string, source string, value *int64, low int64, high int64, lowerIsBetter bool) []int {
	if value == nil {
		return values
	}
	score := scaledScore(*value, low, high, lowerIsBetter)
	components[key] = Component{
		Source:  source,
		ValueBP: value,
		Score:   score,
	}
	return append(values, score)
}

func scaledScore(value int64, low int64, high int64, lowerIsBetter bool) int {
	if high <= low {
		return 0
	}
	if value <= low {
		if lowerIsBetter {
			return 100
		}
		return 0
	}
	if value >= high {
		if lowerIsBetter {
			return 0
		}
		return 100
	}
	numerator := (value - low) * 100
	score := int(numerator / (high - low))
	if lowerIsBetter {
		score = 100 - score
	}
	return clampScore(score)
}

func averageInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	total := 0
	for _, value := range values {
		total += value
	}
	return total / len(values)
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func intPtr(value int) *int {
	return &value
}
