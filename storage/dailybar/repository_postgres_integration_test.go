//go:build integration

package dailybar

import (
	"context"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	coredailybar "github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
)

func TestPostgresDailyBarRepositoryUpsertsAndReadsBars(t *testing.T) {
	ctx := context.Background()
	postgres := integrationtest.StartPostgres(t)
	database := storage.NewDatabaseWithConfig(storage.DatabaseConfig{
		Backend: storage.BackendPostgres,
		URL:     postgres.DSN,
	})
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	reader, writer, err := NewRepositories(database)
	if err != nil {
		t.Fatalf("new repositories: %v", err)
	}

	bar := coredailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		ISIN:         "KR7069500007",
		Name:         "KODEX 200",
		TradingDate:  "2024-04-15",
		Currency:     "KRW",
		Open:         "35000",
		High:         "35250",
		Low:          "34900",
		Close:        "35120",
		Change:       "120",
		ChangeRate:   "0.34",
		Volume:       "123456",
		TradedValue:  "4321000000",
		MarketCap:    "10000000000000",
		Extensions: map[string]string{
			"nav":        "35155.1",
			"bssIdxClpr": "3500.12",
		},
	}

	if _, err := writer.UpsertDailyBars(ctx, []coredailybar.Bar{bar}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	bar.Close = "35130"
	bar.Extensions["bssIdxClpr"] = "3501.23"
	if _, err := writer.UpsertDailyBars(ctx, []coredailybar.Bar{bar}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	bars, err := reader.QueryDailyBars(ctx, daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "2024-04-15",
		To:           "2024-04-15",
	})
	if err != nil {
		t.Fatalf("query daily bars: %v", err)
	}
	if len(bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(bars))
	}
	if bars[0].Close != "35130" {
		t.Fatalf("close = %q, want updated close 35130", bars[0].Close)
	}
	if bars[0].Extensions["nav"] != "35155.1" {
		t.Fatalf("nav extension = %q, want 35155.1", bars[0].Extensions["nav"])
	}
	if bars[0].Extensions["bssIdxClpr"] != "3501.23" {
		t.Fatalf("bssIdxClpr extension = %q, want 3501.23", bars[0].Extensions["bssIdxClpr"])
	}
}
