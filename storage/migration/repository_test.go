package migration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	migrationcore "github.com/awuzag/mwosa/migration"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
)

func TestDailyBarV1ToV2Executor(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	seedDailyBarV1Row(t, ctx, database)

	reader, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		t.Fatalf("new daily bar repositories: %v", err)
	}
	executor, err := NewDailyBarV1ToV2Executor(database, writer)
	if err != nil {
		t.Fatalf("new daily bar migration executor: %v", err)
	}
	rows, err := executor.Apply(ctx)
	if err != nil {
		t.Fatalf("migrate daily bar v1 to v2: %v", err)
	}
	if rows != 1 {
		t.Fatalf("rows migrated = %d, want 1", rows)
	}

	bars, err := reader.QueryDailyBars(ctx, dailyQuery())
	if err != nil {
		t.Fatalf("query migrated bars: %v", err)
	}
	if len(bars) != 1 {
		t.Fatalf("migrated bars len = %d, want 1", len(bars))
	}
	if bars[0].Close != "35120" {
		t.Fatalf("close = %q, want 35120", bars[0].Close)
	}
	if bars[0].Extensions["nav"] != "35155.1" {
		t.Fatalf("nav extension = %q, want 35155.1", bars[0].Extensions["nav"])
	}

	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var v2 storage.DailyBarV2Row
	if err := client.NewSelect().Model(&v2).Limit(1).Scan(ctx); err != nil {
		t.Fatalf("select v2 row: %v", err)
	}
	if v2.SchemaVersion != storage.DailyBarV2SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", v2.SchemaVersion, storage.DailyBarV2SchemaVersion)
	}
}

func TestMigrationRepositoryRecordsAppliedRuns(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new migration repository: %v", err)
	}
	definition := migrationcore.Definition{
		ID:          "test_migration",
		Name:        "Test migration",
		Resource:    "test",
		FromVersion: "1",
		ToVersion:   "2.0.0",
	}
	appliedAt := time.UnixMilli(1_770_000_000_000).UTC()
	if _, err := repository.RecordApplied(ctx, definition, 3, appliedAt); err != nil {
		t.Fatalf("record applied: %v", err)
	}
	run, ok, err := repository.GetRun(ctx, definition.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if !ok {
		t.Fatal("migration run was not found")
	}
	if run.Status != migrationcore.StatusApplied || run.RowsMigrated != 3 {
		t.Fatalf("run = %+v, want applied rows=3", run)
	}
}

func TestDailyBarV2ExtensionCleanupExecutor(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	seedDailyBarV1Row(t, ctx, database)
	_, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		t.Fatalf("new daily bar repositories: %v", err)
	}
	v1ToV2, err := NewDailyBarV1ToV2Executor(database, writer)
	if err != nil {
		t.Fatalf("new daily bar v1 to v2 executor: %v", err)
	}
	if _, err := v1ToV2.Apply(ctx); err != nil {
		t.Fatalf("apply daily bar v1 to v2: %v", err)
	}
	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var v2 storage.DailyBarV2Row
	if err := client.NewSelect().Model(&v2).Limit(1).Scan(ctx); err != nil {
		t.Fatalf("select v2 row: %v", err)
	}
	rows := []storage.DailyBarExtensionV2Row{
		{InstrumentID: v2.InstrumentID, SourceID: v2.SourceID, TradingDate: v2.TradingDate, Key: "nav", Value: "35155.1"},
		{InstrumentID: v2.InstrumentID, SourceID: v2.SourceID, TradingDate: v2.TradingDate, Key: "bssIdxClpr", Value: "3500.12"},
	}
	if _, err := client.NewInsert().Model(&rows).Exec(ctx); err != nil {
		t.Fatalf("insert extension rows: %v", err)
	}
	if _, err := client.ExecContext(ctx, "CREATE INDEX idx_daily_bar_extension_v2_bar ON daily_bar_extension_v2 (instrument_id, source_id, trading_date)"); err != nil {
		t.Fatalf("create redundant index: %v", err)
	}

	executor, err := NewDailyBarV2ExtensionCleanupExecutor(database)
	if err != nil {
		t.Fatalf("new cleanup executor: %v", err)
	}
	deleted, err := executor.Apply(ctx)
	if err != nil {
		t.Fatalf("apply cleanup: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted rows = %d, want 1", deleted)
	}
	var remaining int
	if err := client.QueryRowContext(ctx, "SELECT count(*) FROM daily_bar_extension_v2 WHERE key = 'bssIdxClpr'").Scan(&remaining); err != nil {
		t.Fatalf("count remaining extension rows: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining non-promoted extension rows = %d, want 1", remaining)
	}
	var indexCount int
	if err := client.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type = 'index' AND name = 'idx_daily_bar_extension_v2_bar'").Scan(&indexCount); err != nil {
		t.Fatalf("count redundant index: %v", err)
	}
	if indexCount != 0 {
		t.Fatalf("redundant extension index count = %d, want 0", indexCount)
	}
}

func seedDailyBarV1Row(t *testing.T, ctx context.Context, database *storage.SQLDatabase) {
	t.Helper()
	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	row := storage.DailyBarV1Row{
		Provider:                     string(provider.ProviderDataGo),
		ProviderGroup:                string(provider.GroupSecuritiesProductPrice),
		Operation:                    string(provider.OperationGetETFPriceInfo),
		Market:                       string(provider.MarketKRX),
		SecurityType:                 string(provider.SecurityTypeETF),
		Symbol:                       "069500",
		ISIN:                         "KR7069500007",
		Name:                         "KODEX 200",
		TradingDate:                  "2024-04-15",
		Currency:                     "KRW",
		OpeningPrice:                 "35000",
		HighestPrice:                 "35200",
		LowestPrice:                  "34900",
		ClosingPrice:                 "35120",
		PriceChangeFromPreviousClose: "120",
		TradedVolume:                 "1000",
		TradedAmount:                 "35120000",
		MarketCapitalization:         "1000000000",
		ExtensionsJSON:               `{"nav":"35155.1"}`,
		CreatedAt:                    time.Now().UTC(),
		UpdatedAt:                    time.Now().UTC(),
	}
	if _, err := client.NewInsert().Model(&row).Exec(ctx); err != nil {
		t.Fatalf("insert v1 row: %v", err)
	}
}

func dailyQuery() daily.Query {
	return daily.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "2024-04-15",
		To:           "2024-04-15",
	}
}
