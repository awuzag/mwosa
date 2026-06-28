//go:build integration

package indexbar

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	coreindexbar "github.com/awuzag/mwosa/providers/core/indexbar"
	indexservice "github.com/awuzag/mwosa/service/index"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoIndexBarRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_indexbar_contract_test",
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
	sqliteReader, sqliteWriter, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoReader, mongoWriter, err := NewMongoRepositories(runtime.Database())
	require.NoError(t, err)

	assertIndexBarRepositoryContract(t, sqliteReader, sqliteWriter)
	assertIndexBarRepositoryContract(t, mongoReader, mongoWriter)
	assertMongoIndexBarDocumentShape(t, runtime)
}

func assertIndexBarRepositoryContract(t *testing.T, reader indexservice.ReadRepository, writer indexservice.WriteRepository) {
	t.Helper()

	ctx := context.Background()
	bars := []coreindexbar.Bar{
		{
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
		},
		{
			Provider:    provider.ProviderKRX,
			Group:       provider.GroupKRXIndexDailyTrade,
			Operation:   provider.OperationKOSPIDDTrd,
			Market:      provider.MarketKRX,
			IndexCode:   "KOSPI",
			Name:        "KOSPI",
			Family:      "KOSPI",
			TradingDate: "2024-04-16",
			Currency:    "KRW",
			Close:       "2681.11",
			ChangeRate:  "0.4",
			Volume:      "460000000",
		},
	}
	result, err := writer.UpsertIndexBars(ctx, bars)
	require.NoError(t, err)
	require.Equal(t, 2, result.BarsWritten)
	require.Equal(t, 2, result.RowsAffected)

	bars[0].Close = "2671.01"
	bars[0].Extensions["provider_note"] = "updated"
	result, err = writer.UpsertIndexBars(ctx, bars[:1])
	require.NoError(t, err)
	require.Equal(t, 1, result.BarsWritten)
	require.Equal(t, 1, result.RowsAffected)

	got, err := reader.QueryIndexBars(ctx, indexservice.Query{
		Market:    provider.MarketKRX,
		IndexCode: "KOSPI",
		From:      "2024-04-15",
		To:        "2024-04-16",
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "2024-04-15", got[0].TradingDate)
	require.Equal(t, "2024-04-16", got[1].TradingDate)
	require.Equal(t, "2671.01", got[0].Close)
	require.Equal(t, "0.43", got[0].ChangeRate)
	require.Equal(t, "updated", got[0].Extensions["provider_note"])
	require.Equal(t, "Asia/Seoul", got[0].Extensions["index.timezone"])
	require.Equal(t, "price", got[0].Extensions["index.type"])
}

func assertMongoIndexBarDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var indexDoc struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Sources       []bson.M `bson:"sources"`
	}
	require.NoError(t, runtime.Database().
		Collection("indexes").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "indexes:krx:KOSPI"}}).
		Decode(&indexDoc))
	require.Equal(t, "1.0.0", indexDoc.SchemaVersion)
	require.GreaterOrEqual(t, indexDoc.Revision, int64(2))
	require.Len(t, indexDoc.Sources, 1)
	require.Equal(t, "KOSPI", indexDoc.Sources[0]["provider_symbol"])

	var barDoc struct {
		ID            string            `bson:"_id"`
		SchemaVersion string            `bson:"schema_version"`
		Revision      int64             `bson:"revision"`
		Extensions    map[string]string `bson:"extensions"`
	}
	require.NoError(t, runtime.Database().
		Collection("index_bars").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "index_bars:krx:KOSPI:2024-04-15:krx:indexDailyTrade:kospi_dd_trd"}}).
		Decode(&barDoc))
	require.Equal(t, "1.0.0", barDoc.SchemaVersion)
	require.GreaterOrEqual(t, barDoc.Revision, int64(2))
	require.Equal(t, "updated", barDoc.Extensions["provider_note"])
}
