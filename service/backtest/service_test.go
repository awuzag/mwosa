package backtest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceValidatesAndRunsYAMLAgainstDailyBarRepository(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	repo := &recordingDailyBarRepository{bars: map[string][]dailybar.Bar{
		"069500": sampleCanonicalDailyBars(),
	}}
	service, err := NewService(repo)
	require.NoError(t, err)

	validation, err := service.Validate(context.Background(), path)
	require.NoError(t, err)
	assert.True(t, validation.Valid)
	assert.Equal(t, "sma-cross", validation.StrategyName)
	assert.Equal(t, "sma-cross-run", validation.RunName)
	assert.Equal(t, "next_open", validation.Execution.Fill)
	assert.Equal(t, "sma", validation.Indicators["trend"])

	result, err := service.Run(context.Background(), path)
	require.NoError(t, err)

	require.Len(t, repo.queries, 1)
	assert.Equal(t, provider.MarketKRX, repo.queries[0].Market)
	assert.Equal(t, provider.SecurityTypeETF, repo.queries[0].SecurityType)
	assert.Equal(t, "069500", repo.queries[0].Symbol)
	assert.Equal(t, "2024-01-02", repo.queries[0].From)
	assert.Equal(t, "2024-01-08", repo.queries[0].To)
	assert.Equal(t, "sma-cross", result.StrategyName)
	assert.Equal(t, []string{"069500"}, result.Symbols)
	assert.Equal(t, "krx", result.Market)
	assert.Equal(t, "etf", result.SecurityType)
	assert.Equal(t, "1d", result.Timeframe)
	assert.Equal(t, "next_open", result.Execution.Fill)
	require.Len(t, result.Trades, 2)
	assert.NotEmpty(t, result.EquityCurve)
	assert.NotEmpty(t, result.ResultHash)
}

func TestServiceErrorsWhenDailyBarsAreMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	service, err := NewService(&recordingDailyBarRepository{bars: map[string][]dailybar.Bar{}})
	require.NoError(t, err)

	_, err = service.Run(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canonical daily bars not found")
	assert.Contains(t, err.Error(), "symbol=069500")
}

type recordingDailyBarRepository struct {
	bars    map[string][]dailybar.Bar
	queries []daily.Query
}

func (r *recordingDailyBarRepository) QueryDailyBars(_ context.Context, query daily.Query) ([]dailybar.Bar, error) {
	r.queries = append(r.queries, query)
	return append([]dailybar.Bar(nil), r.bars[query.Symbol]...), nil
}

func sampleCanonicalDailyBars() []dailybar.Bar {
	return []dailybar.Bar{
		canonicalDailyBar("2024-01-02", "10", "10", "10", "10"),
		canonicalDailyBar("2024-01-03", "9", "9", "9", "9"),
		canonicalDailyBar("2024-01-04", "12", "12", "12", "12"),
		canonicalDailyBar("2024-01-05", "13", "13", "11", "11"),
		canonicalDailyBar("2024-01-08", "10", "10", "10", "10"),
	}
}

func canonicalDailyBar(date string, open string, high string, low string, closePrice string) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		TradingDate:  date,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        closePrice,
		Volume:       "1000",
	}
}
