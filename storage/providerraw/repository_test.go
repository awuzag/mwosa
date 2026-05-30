package providerraw

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
)

func TestUpsertSnapshot(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	result, err := repository.UpsertSnapshot(ctx, Snapshot{
		Provider:         provider.ProviderKRX,
		Group:            provider.GroupKRXETPDailyTrade,
		Operation:        provider.OperationETFByddTrd,
		BaseDate:         "20240415",
		CanonicalSupport: "daily_bar,instrument",
		Rows: []map[string]string{
			{"ISU_CD": "069500", "ISU_NM": "KODEX 200"},
		},
		RowCount: 1,
	})
	if err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
	if result.RowsAffected == 0 || result.BaseDate != "2024-04-15" {
		t.Fatalf("unexpected write result: %+v", result)
	}

	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var row storage.ProviderRawSnapshotRow
	if err := client.NewSelect().Model(&row).Where("provider = ?", string(provider.ProviderKRX)).Scan(ctx); err != nil {
		t.Fatalf("select snapshot: %v", err)
	}
	if row.Operation != string(provider.OperationETFByddTrd) || row.RowCount != 1 || row.PayloadJSON == "" {
		t.Fatalf("unexpected row: %+v", row)
	}

	snapshots, err := repository.ListSnapshots(ctx, Query{
		Provider:       provider.ProviderKRX,
		Operation:      provider.OperationETFByddTrd,
		From:           "2024-04-01",
		To:             "2024-04-30",
		IncludePayload: true,
	})
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].BaseDate != "2024-04-15" || snapshots[0].Payload == nil {
		t.Fatalf("unexpected snapshots: %+v", snapshots)
	}
}
