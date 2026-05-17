package financialmetrics

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

const FormulaVersion = "financial-metrics/v1"

const (
	MetricRevenueGrowthYoY         = "revenue_growth_yoy"
	MetricOperatingIncomeGrowthYoY = "operating_income_growth_yoy"
	MetricNetIncomeGrowthYoY       = "net_income_growth_yoy"
	MetricOperatingMargin          = "operating_margin"
	MetricNetMargin                = "net_margin"
	MetricROE                      = "roe"
	MetricROA                      = "roa"
	MetricDebtToEquity             = "debt_to_equity"
	MetricCurrentRatio             = "current_ratio"
	MetricInterestCoverage         = "interest_coverage"
)

type AccountValue struct {
	StatementID      int64
	InstrumentID     int64
	Provider         string
	ProviderGroup    string
	Operation        string
	ReportCode       string
	FsDiv            string
	RceptNo          string
	FiscalYear       string
	FiscalPeriod     string
	AsOfDate         string
	CanonicalAccount string
	AmountMinor      int64
}

type Metric struct {
	StatementID        int64
	InstrumentID       int64
	Metric             string
	FiscalYear         string
	FiscalPeriod       string
	AsOfDate           string
	ValueDecimal       string
	ValueBP            *int64
	ValueMinor         *int64
	FormulaVersion     string
	Provenance         map[string]any
	UncomputableReason string
}

func Calculate(values []AccountValue, windowYears int) []Metric {
	periods := periodsFromValues(values, windowYears)
	byPeriod := make(map[string]map[string]AccountValue)
	for _, value := range values {
		if value.CanonicalAccount == "" || value.FiscalYear == "" {
			continue
		}
		key := periodKey(value.FiscalYear, value.FiscalPeriod)
		if byPeriod[key] == nil {
			byPeriod[key] = make(map[string]AccountValue)
		}
		byPeriod[key][value.CanonicalAccount] = value
	}

	out := make([]Metric, 0, len(periods)*10)
	for _, period := range periods {
		current := byPeriod[periodKey(period.FiscalYear, period.FiscalPeriod)]
		previous := byPeriod[periodKey(strconv.Itoa(period.Year-1), period.FiscalPeriod)]
		out = append(out,
			growthMetric(MetricRevenueGrowthYoY, period, current["revenue"], previous["revenue"]),
			growthMetric(MetricOperatingIncomeGrowthYoY, period, current["operating_income"], previous["operating_income"]),
			growthMetric(MetricNetIncomeGrowthYoY, period, current["net_income"], previous["net_income"]),
			ratioMetric(MetricOperatingMargin, period, current["operating_income"], current["revenue"], "operating_income", "revenue"),
			ratioMetric(MetricNetMargin, period, current["net_income"], current["revenue"], "net_income", "revenue"),
			ratioMetric(MetricROE, period, current["net_income"], current["equity"], "net_income", "equity"),
			ratioMetric(MetricROA, period, current["net_income"], current["total_assets"], "net_income", "total_assets"),
			ratioMetric(MetricDebtToEquity, period, current["total_liabilities"], current["equity"], "total_liabilities", "equity"),
			uncomputable(MetricCurrentRatio, period, "current_assets and current_liabilities are not mapped"),
			uncomputable(MetricInterestCoverage, period, "interest_expense is not mapped"),
		)
	}
	return out
}

type periodRef struct {
	Year         int
	FiscalYear   string
	FiscalPeriod string
	AsOfDate     string
}

func periodsFromValues(values []AccountValue, windowYears int) []periodRef {
	seen := make(map[string]periodRef)
	for _, value := range values {
		year, err := strconv.Atoi(value.FiscalYear)
		if err != nil {
			continue
		}
		key := periodKey(value.FiscalYear, value.FiscalPeriod)
		seen[key] = periodRef{
			Year:         year,
			FiscalYear:   value.FiscalYear,
			FiscalPeriod: value.FiscalPeriod,
			AsOfDate:     value.AsOfDate,
		}
	}
	periods := make([]periodRef, 0, len(seen))
	for _, period := range seen {
		periods = append(periods, period)
	}
	sort.Slice(periods, func(i int, j int) bool {
		if periods[i].Year == periods[j].Year {
			return periods[i].FiscalPeriod < periods[j].FiscalPeriod
		}
		return periods[i].Year > periods[j].Year
	})
	if windowYears > 0 && len(periods) > windowYears {
		periods = periods[:windowYears]
	}
	sort.Slice(periods, func(i int, j int) bool {
		if periods[i].Year == periods[j].Year {
			return periods[i].FiscalPeriod < periods[j].FiscalPeriod
		}
		return periods[i].Year < periods[j].Year
	})
	return periods
}

func growthMetric(metric string, period periodRef, current AccountValue, previous AccountValue) Metric {
	base := metricBase(metric, period, current, previous)
	switch {
	case current.CanonicalAccount == "":
		return withReason(base, "current account value is missing")
	case previous.CanonicalAccount == "":
		return withReason(base, "previous year account value is missing")
	case previous.AmountMinor <= 0:
		return withReason(base, "previous year account value must be positive")
	default:
		numerator := current.AmountMinor - previous.AmountMinor
		return withRatio(base, numerator, previous.AmountMinor)
	}
}

func ratioMetric(metric string, period periodRef, numerator AccountValue, denominator AccountValue, numeratorName string, denominatorName string) Metric {
	base := metricBase(metric, period, numerator, denominator)
	switch {
	case numerator.CanonicalAccount == "":
		return withReason(base, numeratorName+" account value is missing")
	case denominator.CanonicalAccount == "":
		return withReason(base, denominatorName+" account value is missing")
	case denominator.AmountMinor == 0:
		return withReason(base, denominatorName+" account value must not be zero")
	default:
		return withRatio(base, numerator.AmountMinor, denominator.AmountMinor)
	}
}

func metricBase(metric string, period periodRef, values ...AccountValue) Metric {
	out := Metric{
		Metric:         metric,
		FiscalYear:     period.FiscalYear,
		FiscalPeriod:   period.FiscalPeriod,
		AsOfDate:       period.AsOfDate,
		FormulaVersion: FormulaVersion,
		Provenance: map[string]any{
			"formula_version": FormulaVersion,
		},
	}
	statementIDs := make([]int64, 0, len(values))
	for _, value := range values {
		if value.StatementID == 0 {
			continue
		}
		if out.StatementID == 0 {
			out.StatementID = value.StatementID
		}
		if out.InstrumentID == 0 {
			out.InstrumentID = value.InstrumentID
		}
		statementIDs = append(statementIDs, value.StatementID)
		if out.Provenance["provider"] == nil && value.Provider != "" {
			out.Provenance["provider"] = value.Provider
			out.Provenance["provider_group"] = value.ProviderGroup
			out.Provenance["operation"] = value.Operation
			out.Provenance["report_code"] = value.ReportCode
			out.Provenance["fs_div"] = value.FsDiv
			out.Provenance["rcept_no"] = value.RceptNo
		}
	}
	if len(statementIDs) > 0 {
		out.Provenance["statement_ids"] = statementIDs
	}
	return out
}

func uncomputable(metric string, period periodRef, reason string) Metric {
	return withReason(metricBase(metric, period), reason)
}

func withReason(metric Metric, reason string) Metric {
	metric.UncomputableReason = reason
	if metric.Provenance == nil {
		metric.Provenance = make(map[string]any)
	}
	metric.Provenance["uncomputable_reason"] = reason
	return metric
}

func withRatio(metric Metric, numerator int64, denominator int64) Metric {
	ratio := float64(numerator) / float64(denominator)
	bp := int64(math.Round(ratio * 10000))
	metric.ValueBP = &bp
	metric.ValueDecimal = strconv.FormatFloat(ratio, 'f', 8, 64)
	if metric.Provenance == nil {
		metric.Provenance = make(map[string]any)
	}
	metric.Provenance["numerator"] = strconv.FormatInt(numerator, 10)
	metric.Provenance["denominator"] = strconv.FormatInt(denominator, 10)
	return metric
}

func periodKey(year string, period string) string {
	return strings.TrimSpace(year) + "\x00" + strings.TrimSpace(period)
}
