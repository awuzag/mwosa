//go:build integration

package kis

import (
	"context"
	"os"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/stretchr/testify/require"
)

func TestLiveKISFetchRawDailyItemChartPrice(t *testing.T) {
	appKey := firstEnv("MWOSA_KIS_APP_KEY", "KIS_APP_KEY", "APP_KEY")
	appSecret := firstEnv("MWOSA_KIS_APP_SECRET", "KIS_APP_SECRET", "APP_SECRET")
	if appKey == "" || appSecret == "" {
		t.Skip("set MWOSA_KIS_APP_KEY and MWOSA_KIS_APP_SECRET to run live KIS raw API smoke")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	p, err := New(Config{
		AppKey:      appKey,
		AppSecret:   appSecret,
		AccessToken: firstEnv("MWOSA_KIS_ACCESS_TOKEN", "KIS_ACCESS_TOKEN"),
		Virtual:     firstEnv("MWOSA_KIS_VIRTUAL", "KIS_VIRTUAL") == "true",
	})
	require.NoError(t, err)

	result, err := p.FetchRaw(ctx, RawRequest{
		OperationID: provider.OperationID("inquire-daily-itemchartprice"),
		Input: map[string]string{
			"FID_INPUT_ISCD":      "005930",
			"FID_INPUT_DATE_1":    "20250102",
			"FID_INPUT_DATE_2":    "20250131",
			"FID_PERIOD_DIV_CODE": "D",
		},
	})
	require.NoError(t, err)
	require.Equal(t, provider.ProviderKIS, result.Provider)
	require.Equal(t, provider.OperationID("inquire-daily-itemchartprice"), result.Operation)
	require.NotEmpty(t, result.Endpoint)
	require.GreaterOrEqual(t, result.RowCount, 0)
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
