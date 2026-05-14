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
	assert.Equal(t, 1, result.Evaluation.LookbackDays)
	assert.Equal(t, 1.0, result.Confidence)
	assert.Equal(t, 1, result.StableDays)
	assert.Equal(t, 0, result.Transitions)
	assert.Equal(t, []string{"uptrend"}, result.RecentRegimes)
	require.NotEmpty(t, result.Evidence)
	assert.Equal(t, "return_20d_gte", result.Evidence[0].Code)
	assert.True(t, result.Evidence[0].Passed)
}

func TestEvaluateMarketRegimeUsesRecentTradingRowsForStability(t *testing.T) {
	up := true
	minReturn := 0.03
	sidewaysMin := -0.03
	sidewaysMax := 0.03
	spec := MarketRegimeSpec{
		Kind:          KindMarketRegime,
		SchemaVersion: 1,
		Name:          "us-growth-regime",
		Spec: MarketRegimeBodySpec{
			Benchmark: MarketRegimeBenchmarkSpec{Symbol: "379810", Market: "krx", SecurityType: "etf"},
			Evaluation: MarketRegimeEvaluationSpec{
				LookbackDays: 10,
				ConfirmDays:  7,
			},
			Rules: []MarketRegimeRuleSpec{
				{
					Regime: "uptrend",
					When: MarketRegimeConditionSpec{
						Return20DGTE:   &minReturn,
						CloseAboveMA20: &up,
						MA20AboveMA60:  &up,
					},
				},
				{
					Regime: "sideways",
					When: MarketRegimeConditionSpec{
						Return20DBetween: []float64{sidewaysMin, sidewaysMax},
					},
				},
			},
		},
	}

	bars := make([]Bar, 0, 70)
	for i := 0; i < 70; i++ {
		closePrice := 100.0
		if i >= 61 {
			closePrice = 100 + float64(i-60)
		}
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
	assert.InDelta(t, 0.7, result.Confidence, 0.0000001)
	assert.Equal(t, 7, result.StableDays)
	assert.Equal(t, 1, result.Transitions)
	assert.True(t, result.Confirmed)
	assert.Equal(t, []string{"sideways", "sideways", "sideways", "uptrend", "uptrend", "uptrend", "uptrend", "uptrend", "uptrend", "uptrend"}, result.RecentRegimes)
	require.Len(t, result.RecentEvaluations, 10)
	assert.Equal(t, "2024-03-10", result.RecentEvaluations[9].AsOf)
	require.Len(t, result.RuleEvaluations, 2)
	assert.True(t, result.RuleEvaluations[0].Matched)
	assert.False(t, result.RuleEvaluations[1].Matched)
}

func TestEvaluateMarketRegimeFailsWhenLookbackHistoryIsInsufficient(t *testing.T) {
	up := true
	minReturn := 0.03
	spec := MarketRegimeSpec{
		Kind:          KindMarketRegime,
		SchemaVersion: 1,
		Name:          "us-growth-regime",
		Spec: MarketRegimeBodySpec{
			Benchmark: MarketRegimeBenchmarkSpec{Symbol: "379810", Market: "krx", SecurityType: "etf"},
			Evaluation: MarketRegimeEvaluationSpec{
				LookbackDays: 10,
				ConfirmDays:  7,
			},
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

	bars := make([]Bar, 0, 69)
	for i := 0; i < 69; i++ {
		bars = append(bars, Bar{
			Time:         date("2024-01-01").AddDate(0, 0, i),
			Symbol:       "379810",
			Market:       "krx",
			SecurityType: "etf",
			Close:        100 + float64(i),
		})
	}

	_, err := EvaluateMarketRegime(context.Background(), spec, bars, date("2024-03-09"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not have enough closed bars")
}

func TestEvaluateMarketRegimeFailsWhenAsOfBarIsMissing(t *testing.T) {
	spec := MarketRegimeSpec{
		Kind:          KindMarketRegime,
		SchemaVersion: 1,
		Name:          "us-growth-regime",
		Spec: MarketRegimeBodySpec{
			Benchmark: MarketRegimeBenchmarkSpec{Symbol: "379810", Market: "krx", SecurityType: "etf"},
			Rules: []MarketRegimeRuleSpec{{
				Regime: "sideways",
				When: MarketRegimeConditionSpec{
					Return20DBetween: []float64{-0.03, 0.03},
				},
			}},
		},
	}
	bars := make([]Bar, 0, 70)
	for i := 0; i < 70; i++ {
		bars = append(bars, Bar{
			Time:         date("2024-01-01").AddDate(0, 0, i),
			Symbol:       "379810",
			Market:       "krx",
			SecurityType: "etf",
			Close:        100,
		})
	}

	_, err := EvaluateMarketRegime(context.Background(), spec, bars, date("2024-03-11"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bar missing for as_of")
}
