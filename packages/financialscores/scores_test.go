package financialscores

import (
	"testing"

	"github.com/ev3rlit/mwosa/packages/financialmetrics"
	"github.com/stretchr/testify/require"
)

func TestCalculateScoresFromMetricsAndValuation(t *testing.T) {
	roe := int64(1800)
	per := int64(120000)

	scores := Calculate(Input{
		Metrics: map[string]Metric{
			financialmetrics.MetricROE: {ValueBP: &roe},
		},
		Valuation: &Valuation{PerBP: &per},
	})

	require.True(t, scores.HasSignal())
	require.Equal(t, Version, scores.ScoreVersion)
	require.NotNil(t, scores.QualityScore)
	require.Equal(t, 72, *scores.QualityScore)
	require.NotNil(t, scores.ValuationScore)
	require.Equal(t, 82, *scores.ValuationScore)
	require.Nil(t, scores.GrowthScore)
	require.Equal(t, "no usable growth metrics", scores.Uncomputable["growth_score"])
	require.Equal(t, "financial_metrics.roe.value_bp", scores.Components["quality.roe"].Source)
}

func TestCalculatePreservesUncomputableMetricReason(t *testing.T) {
	scores := Calculate(Input{
		Metrics: map[string]Metric{
			financialmetrics.MetricCurrentRatio: {UncomputableReason: "current assets unavailable"},
		},
	})

	require.True(t, scores.HasSignal())
	require.Nil(t, scores.QualityScore)
	require.Equal(t, "current assets unavailable", scores.Uncomputable["quality.current_ratio"])
	require.Equal(t, "valuation snapshot is missing", scores.Uncomputable["valuation_score"])
}
