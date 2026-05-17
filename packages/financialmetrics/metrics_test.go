package financialmetrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateFinancialMetrics(t *testing.T) {
	metrics := Calculate([]AccountValue{
		account("2024", "annual", "revenue", 1000),
		account("2024", "annual", "operating_income", 100),
		account("2024", "annual", "net_income", 50),
		account("2024", "annual", "total_assets", 500),
		account("2024", "annual", "total_liabilities", 200),
		account("2024", "annual", "equity", 300),
		account("2025", "annual", "revenue", 1200),
		account("2025", "annual", "operating_income", 180),
		account("2025", "annual", "net_income", 90),
		account("2025", "annual", "total_assets", 600),
		account("2025", "annual", "total_liabilities", 240),
		account("2025", "annual", "equity", 360),
	}, 2)

	byKey := make(map[string]Metric)
	for _, metric := range metrics {
		byKey[metric.FiscalYear+"\x00"+metric.Metric] = metric
	}
	require.Equal(t, "0.20000000", byKey["2025\x00"+MetricRevenueGrowthYoY].ValueDecimal)
	require.Equal(t, int64(2000), *byKey["2025\x00"+MetricRevenueGrowthYoY].ValueBP)
	require.Equal(t, "0.15000000", byKey["2025\x00"+MetricOperatingMargin].ValueDecimal)
	require.Equal(t, "0.25000000", byKey["2025\x00"+MetricROE].ValueDecimal)
	require.Equal(t, "0.66666667", byKey["2025\x00"+MetricDebtToEquity].ValueDecimal)
	require.Equal(t, "interest_expense is not mapped", byKey["2025\x00"+MetricInterestCoverage].UncomputableReason)
}

func account(year string, period string, canonical string, amount int64) AccountValue {
	return AccountValue{
		StatementID:      1,
		InstrumentID:     2,
		Provider:         "opendart",
		ProviderGroup:    "financials",
		Operation:        "fnlttSinglAcntAll",
		ReportCode:       "11011",
		FsDiv:            "CFS",
		RceptNo:          "rcept",
		FiscalYear:       year,
		FiscalPeriod:     period,
		AsOfDate:         year + "-12-31",
		CanonicalAccount: canonical,
		AmountMinor:      amount,
	}
}
