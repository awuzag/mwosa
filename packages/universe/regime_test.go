package universe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateMarketRegimeMatchesFirstRuleByAsOfMetrics(t *testing.T) {
	up := true
	minReturn := 0.03
	spec := MarketRegimeSpec{
		Kind:          KindMarketRegime,
		SchemaVersion: 1,
		Name:          "us-growth-regime",
		Spec: MarketRegimeBodySpec{
			Benchmark: MarketRegimeBenchmarkSpec{Symbol: "379810", Market: "krx", SecurityType: "etf"},
			Rules: []MarketRegimeRuleSpec{{
				Regime: "uptrend",
				When: MarketRegimeConditionSpec{
					Return20DGTE:   &minReturn,
					CloseAboveMA20: &up,
					MA20AboveMA60:  &up,
				},
			}},
		},
	}

	bars := make([]Bar, 0, 70)
	for i := 0; i < 70; i++ {
		closePrice := 100 + float64(i)
		bars = append(bars, Bar{
			Time:         date("2024-01-01").AddDate(0, 0, i),
			Symbol:       "379810",
			Market:       "krx",
			SecurityType: "etf",
			Close:        closePrice,
		})
	}

	result, err := EvaluateMarketRegime(context.Background(), spec, bars, date("2024-03-10"))
	require.NoError(t, err)

	assert.Equal(t, "uptrend", result.Regime)
	assert.Equal(t, 0, result.MatchedRuleIndex)
	assert.Greater(t, result.Metrics.Return20D, 0.03)
}
