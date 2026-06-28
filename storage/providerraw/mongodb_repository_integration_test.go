//go:build integration

package providerraw

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type rawSnapshotRepository interface {
	UpsertSnapshot(ctx context.Context, snapshot Snapshot) (WriteResult, error)
	ListSnapshots(ctx context.Context, query Query) ([]SnapshotRecord, error)
}

func TestMongoProviderRawRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_providerraw_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	sqliteDatabase := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, sqliteDatabase.Close())
	})
	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertProviderRawRepositoryContract(t, sqliteRepository)
	assertProviderRawRepositoryContract(t, mongoRepository)
	assertMongoProviderRawDocumentShape(t, runtime)
}

func assertProviderRawRepositoryContract(t *testing.T, repository rawSnapshotRepository) {
	t.Helper()

	ctx := context.Background()
	snapshot := Snapshot{
		Provider:         provider.ProviderKRX,
		Group:            provider.GroupKRXETPDailyTrade,
		Operation:        provider.OperationETFByddTrd,
		BaseDate:         "20240415",
		CanonicalSupport: "daily_bar,instrument",
		Rows: []map[string]string{
			{"ISU_CD": "069500", "ISU_NM": "KODEX 200"},
		},
		RowCount: 1,
	}
	result, err := repository.UpsertSnapshot(ctx, snapshot)
	require.NoError(t, err)
	require.Equal(t, "2024-04-15", result.BaseDate)
	require.Equal(t, int64(1), result.RowsAffected)

	snapshot.CanonicalSupport = "daily_bar"
	snapshot.Rows = []map[string]string{
		{"ISU_CD": "069500", "ISU_NM": "KODEX 200", "CLSPRC": "35000"},
	}
	result, err = repository.UpsertSnapshot(ctx, snapshot)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.RowsAffected)

	snapshots, err := repository.ListSnapshots(ctx, Query{
		Provider:       provider.ProviderKRX,
		Operation:      provider.OperationETFByddTrd,
		From:           "2024-04-01",
		To:             "2024-04-30",
		IncludePayload: true,
	})
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, "2024-04-15", snapshots[0].BaseDate)
	require.Equal(t, "daily_bar", snapshots[0].CanonicalSupport)
	require.Equal(t, 1, snapshots[0].RowCount)
	require.NotNil(t, snapshots[0].Payload)
	require.Greater(t, snapshots[0].CreatedAtMS, int64(0))
	require.Greater(t, snapshots[0].UpdatedAtMS, int64(0))
}

func assertMongoProviderRawDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var stored struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Source        struct {
			Provider      string `bson:"provider"`
			ProviderGroup string `bson:"provider_group"`
			Operation     string `bson:"operation"`
			BaseDate      string `bson:"base_date"`
		} `bson:"source"`
		Payload bson.A `bson:"payload"`
	}
	require.NoError(t, runtime.Database().
		Collection("provider_raw_snapshots").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "provider_raw_snapshots:krx:etpDailyTrade:etf_bydd_trd:2024-04-15"}}).
		Decode(&stored))
	require.Equal(t, "1.0.0", stored.SchemaVersion)
	require.Equal(t, int64(2), stored.Revision)
	require.Equal(t, "2024-04-15", stored.Source.BaseDate)
	require.Len(t, stored.Payload, 1)
}
