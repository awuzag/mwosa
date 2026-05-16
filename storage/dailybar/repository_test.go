package dailybar

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	"github.com/ev3rlit/mwosa/storage"
)

func TestDailyBarStoreUpsertIsIdempotent(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	reader, writer, err := NewRepositories(database)
	if err != nil {
		t.Fatalf("new repositories: %v", err)
	}
	bar := dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		Name:         "KODEX 200",
		TradingDate:  "2024-04-15",
		Close:        "35120",
		Extensions: map[string]string{
			"nav":        "35155.1",
			"bssIdxClpr": "3500.12",
		},
	}

	if _, err := writer.UpsertDailyBars(context.Background(), []dailybar.Bar{bar}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	bar.Close = "35130"
	if _, err := writer.UpsertDailyBars(context.Background(), []dailybar.Bar{bar}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	assertDailyBarRowCounts(t, database, 0, 1)

	bars, err := reader.QueryDailyBars(context.Background(), daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "2024-04-15",
		To:           "2024-04-15",
	})
	if err != nil {
		t.Fatalf("query: %v", err)
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
	if bars[0].Extensions["bssIdxClpr"] != "3500.12" {
		t.Fatalf("bssIdxClpr extension = %q, want 3500.12", bars[0].Extensions["bssIdxClpr"])
	}
	if got := countDailyBarExtensionV2Rows(t, database); got != 1 {
		t.Fatalf("daily_bar_extension_v2 rows = %d, want only non-promoted extension row", got)
	}
	stored := getStoredDailyBarV2Row(t, database)
	if stored.SchemaVersion != storage.DailyBarV2SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", stored.SchemaVersion, storage.DailyBarV2SchemaVersion)
	}
}

func TestDailyBarStoreStreamsRowsInTradingDateOrder(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	reader, writer, err := NewRepositories(database)
	if err != nil {
		t.Fatalf("new repositories: %v", err)
	}
	_, err = writer.UpsertDailyBars(context.Background(), []dailybar.Bar{
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			TradingDate:  "2024-04-16",
			Close:        "35200",
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			TradingDate:  "2024-04-15",
			Close:        "35120",
		},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	stream, err := reader.(daily.StreamRepository).StreamDailyBars(context.Background(), daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "2024-04-15",
		To:           "2024-04-16",
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()
	first, ok, err := stream.Next(context.Background())
	if err != nil || !ok {
		t.Fatalf("first next: ok=%v err=%v", ok, err)
	}
	second, ok, err := stream.Next(context.Background())
	if err != nil || !ok {
		t.Fatalf("second next: ok=%v err=%v", ok, err)
	}
	if first.TradingDate != "2024-04-15" || second.TradingDate != "2024-04-16" {
		t.Fatalf("stream order = %s, %s", first.TradingDate, second.TradingDate)
	}
	if _, ok, err := stream.Next(context.Background()); err != nil || ok {
		t.Fatalf("stream end: ok=%v err=%v", ok, err)
	}
}

func TestDailyBarStoreUpsertPreservesCreatedAtAndRefreshesUpdatedAt(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	_, writer, err := NewRepositories(database)
	if err != nil {
		t.Fatalf("new repositories: %v", err)
	}
	bar := dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		TradingDate:  "2024-04-15",
		Close:        "35120",
	}

	if _, err := writer.UpsertDailyBars(context.Background(), []dailybar.Bar{bar}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	first := getStoredDailyBarV2Row(t, database)

	time.Sleep(10 * time.Millisecond)
	bar.Close = "35130"
	if _, err := writer.UpsertDailyBars(context.Background(), []dailybar.Bar{bar}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	second := getStoredDailyBarV2Row(t, database)

	if second.CreatedAtMS != first.CreatedAtMS {
		t.Fatalf("created_at_ms = %d, want preserved %d", second.CreatedAtMS, first.CreatedAtMS)
	}
	if second.UpdatedAtMS <= first.UpdatedAtMS {
		t.Fatalf("updated_at_ms = %d, want after %d", second.UpdatedAtMS, first.UpdatedAtMS)
	}
}

func TestDailyBarCoverageSummariesUseStoredV2Rows(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	reader, writer, err := NewRepositories(database)
	if err != nil {
		t.Fatalf("new repositories: %v", err)
	}
	bars := []dailybar.Bar{
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			Name:         "KODEX 200",
			TradingDate:  "2024-04-15",
			Close:        "35120",
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			Name:         "KODEX 200",
			TradingDate:  "2024-04-16",
			Close:        "35200",
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "123456",
			Name:         "OTHER ETF",
			TradingDate:  "2024-04-16",
			Close:        "1000",
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationID("getStockPriceInfo"),
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			Symbol:       "005930",
			Name:         "Samsung Electronics",
			TradingDate:  "2024-04-16",
			Close:        "80000",
		},
	}
	if _, err := writer.UpsertDailyBars(context.Background(), bars); err != nil {
		t.Fatalf("seed daily bars: %v", err)
	}

	summary, err := reader.SummarizeDailyBarStorage(context.Background(), daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
	})
	if err != nil {
		t.Fatalf("summarize storage: %v", err)
	}
	if summary.RecordType != "daily_bar" || summary.Symbols != 2 || summary.Bars != 3 || summary.Dates != 2 || summary.From != "2024-04-15" || summary.To != "2024-04-16" {
		t.Fatalf("summary = %+v, want etf aggregate counts and range", summary)
	}

	coverage, err := reader.QueryDailyBarCoverage(context.Background(), daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err != nil {
		t.Fatalf("query coverage: %v", err)
	}
	if coverage.Symbol != "069500" || coverage.Name != "KODEX 200" || coverage.Bars != 2 || coverage.Dates != 2 || coverage.From != "2024-04-15" || coverage.To != "2024-04-16" {
		t.Fatalf("coverage = %+v, want 069500 range", coverage)
	}
}

func TestNewRepositoriesRequiresDatabase(t *testing.T) {
	if _, _, err := NewRepositories(nil); err == nil {
		t.Fatal("NewRepositories nil database error is nil")
	}
	if _, err := NewReadRepository(nil); err == nil {
		t.Fatal("NewReadRepository nil database error is nil")
	}
	if _, err := NewWriteRepository(nil); err == nil {
		t.Fatal("NewWriteRepository nil database error is nil")
	}
}

func countDailyBarExtensionV2Rows(t *testing.T, database *storage.Database) int {
	t.Helper()

	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var got int
	if err := client.QueryRowContext(context.Background(), "SELECT count(*) FROM daily_bar_extension_v2").Scan(&got); err != nil {
		t.Fatalf("count daily_bar_extension_v2 rows: %v", err)
	}
	return got
}

func getStoredDailyBarV2Row(t *testing.T, database *storage.Database) storage.DailyBarV2Row {
	t.Helper()

	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var row storage.DailyBarV2Row
	if err := client.NewSelect().Model(&row).Limit(1).Scan(context.Background()); err != nil {
		t.Fatalf("select row: %v", err)
	}
	return row
}

func assertDailyBarRowCounts(t *testing.T, database *storage.Database, wantV1Rows, wantV2Rows int) {
	t.Helper()

	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var gotV1Rows int
	if err := client.QueryRowContext(context.Background(), "SELECT count(*) FROM daily_bar").Scan(&gotV1Rows); err != nil {
		t.Fatalf("count daily_bar v1 rows: %v", err)
	}
	var gotV2Rows int
	if err := client.QueryRowContext(context.Background(), "SELECT count(*) FROM daily_bar_v2").Scan(&gotV2Rows); err != nil {
		t.Fatalf("count daily_bar v2 rows: %v", err)
	}
	if gotV1Rows != wantV1Rows {
		t.Fatalf("daily_bar v1 rows = %d, want %d", gotV1Rows, wantV1Rows)
	}
	if gotV2Rows != wantV2Rows {
		t.Fatalf("daily_bar_v2 rows = %d, want %d", gotV2Rows, wantV2Rows)
	}
}
