//go:build integration

package dailybar

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
)

func TestMongoDailyBarRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_dailybar_contract_test",
	})
	if err != nil {
		t.Fatalf("new mongodb runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Fatalf("close mongodb runtime: %v", err)
		}
	})
	if err := runtime.Init(ctx); err != nil {
		t.Fatalf("init mongodb runtime: %v", err)
	}

	sqliteDatabase := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := sqliteDatabase.Close(); err != nil {
			t.Fatalf("close sqlite database: %v", err)
		}
	})
	sqliteReader, sqliteWriter, err := NewRepositories(sqliteDatabase)
	if err != nil {
		t.Fatalf("new sqlite repositories: %v", err)
	}
	mongoReader, mongoWriter, err := NewMongoRepositories(runtime.Database())
	if err != nil {
		t.Fatalf("new mongodb repositories: %v", err)
	}

	assertDailyBarRepositoryContract(t, sqliteReader, sqliteWriter)
	assertDailyBarRepositoryContract(t, mongoReader, mongoWriter)
}

func assertDailyBarRepositoryContract(t *testing.T, reader daily.ReadRepository, writer daily.WriteRepository) {
	t.Helper()

	ctx := context.Background()
	bars := []dailybar.Bar{
		{
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
			Close:        "35120",
			Volume:       "1000",
			Extensions: map[string]string{
				"nav":        "35155.1",
				"bssIdxClpr": "3500.12",
			},
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			ISIN:         "KR7069500007",
			Name:         "KODEX 200",
			TradingDate:  "2024-04-16",
			Currency:     "KRW",
			Close:        "35200",
			Volume:       "1200",
		},
	}
	result, err := writer.UpsertDailyBars(ctx, bars)
	if err != nil {
		t.Fatalf("upsert bars: %v", err)
	}
	if result.BarsWritten != 2 || result.RowsAffected != 2 {
		t.Fatalf("write result = %+v, want 2 bars and rows", result)
	}

	got, err := reader.QueryDailyBars(ctx, daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "2024-04-15",
		To:           "2024-04-16",
	})
	if err != nil {
		t.Fatalf("query bars: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("bars len = %d, want 2", len(got))
	}
	if got[0].TradingDate != "2024-04-15" || got[1].TradingDate != "2024-04-16" {
		t.Fatalf("bars order = %s, %s", got[0].TradingDate, got[1].TradingDate)
	}
	if got[0].Extensions["nav"] != "35155.1" || got[0].Extensions["bssIdxClpr"] != "3500.12" {
		t.Fatalf("extensions = %#v, want stored extensions", got[0].Extensions)
	}

	summary, err := reader.SummarizeDailyBarStorage(ctx, daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
	})
	if err != nil {
		t.Fatalf("summarize bars: %v", err)
	}
	if summary.Symbols != 1 || summary.Bars != 2 || summary.Dates != 2 || summary.From != "2024-04-15" || summary.To != "2024-04-16" {
		t.Fatalf("summary = %+v, want two bars for one symbol", summary)
	}

	coverage, err := reader.QueryDailyBarCoverage(ctx, daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if coverage.Name != "KODEX 200" || coverage.Bars != 2 || coverage.Dates != 2 {
		t.Fatalf("coverage = %+v, want KODEX 200 two bars", coverage)
	}
}
