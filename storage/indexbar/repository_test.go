package indexbar

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	indexservice "github.com/awuzag/mwosa/service/index"
	"github.com/awuzag/mwosa/storage"
)

func TestIndexBarStoreUpsertIsIdempotent(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	reader, writer, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	bar := indexbar.Bar{
		Provider:    provider.ProviderKRX,
		Group:       provider.GroupKRXIndexDailyTrade,
		Operation:   provider.OperationKOSPIDDTrd,
		Market:      provider.MarketKRX,
		IndexCode:   "KOSPI",
		Name:        "KOSPI",
		Family:      "KOSPI",
		TradingDate: "2024-04-15",
		Currency:    "KRW",
		Open:        "2660",
		High:        "2680.1",
		Low:         "2655.2",
		Close:       "2670.43",
		Change:      "11.39",
		ChangeRate:  "0.43",
		Volume:      "450000000",
		TradedValue: "9000000000000",
		MarketCap:   "2100000000000000",
		Extensions:  map[string]string{"provider_note": "first"},
	}

	if _, err := writer.UpsertIndexBars(context.Background(), []indexbar.Bar{bar}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	bar.Close = "2671.01"
	bar.Extensions["provider_note"] = "updated"
	if _, err := writer.UpsertIndexBars(context.Background(), []indexbar.Bar{bar}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	assertIndexRowCounts(t, database, 1, 1, 1, 1)

	bars, err := reader.QueryIndexBars(context.Background(), indexservice.Query{
		Market:    provider.MarketKRX,
		IndexCode: "KOSPI",
		From:      "2024-04-15",
		To:        "2024-04-15",
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(bars))
	}
	if bars[0].Close != "2671.01" || bars[0].ChangeRate != "0.43" || bars[0].Extensions["provider_note"] != "updated" {
		t.Fatalf("unexpected bar: %+v", bars[0])
	}
}

func assertIndexRowCounts(t *testing.T, database *storage.Database, wantIndexes, wantSources, wantBars, wantExtensions int) {
	t.Helper()
	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	counts := map[string]int{
		"index_v1":               0,
		"index_source_v1":        0,
		"index_bar_v1":           0,
		"index_bar_extension_v1": 0,
	}
	for table := range counts {
		var got int
		if err := client.QueryRowContext(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = got
	}
	if counts["index_v1"] != wantIndexes || counts["index_source_v1"] != wantSources || counts["index_bar_v1"] != wantBars || counts["index_bar_extension_v1"] != wantExtensions {
		t.Fatalf("counts = %+v, want index/source/bar/extension = %d/%d/%d/%d", counts, wantIndexes, wantSources, wantBars, wantExtensions)
	}
}
