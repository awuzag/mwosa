//go:build integration

package composition

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	compositionrole "github.com/awuzag/mwosa/providers/core/composition"
	compositionservice "github.com/awuzag/mwosa/service/composition"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoCompositionRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_composition_contract_test",
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

	assertCompositionRepositoryContract(t, sqliteRepository)
	assertCompositionRepositoryContract(t, mongoRepository)
	assertMongoCompositionDocumentShape(t, runtime)
}

func assertCompositionRepositoryContract(t *testing.T, repository compositionservice.Repository) {
	t.Helper()

	ctx := context.Background()
	aggregate := sampleComposition()
	writeResult, err := repository.UpsertComposition(ctx, aggregate)
	require.NoError(t, err)
	require.Equal(t, 1, writeResult.CompositionsStored)
	require.Equal(t, 2, writeResult.MembersStored)

	got, err := repository.GetComposition(ctx, compositionservice.Query{
		ProviderID:   provider.ProviderKIS,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		AsOfDate:     "2026-05-17",
	})
	require.NoError(t, err)
	requireCompositionEqual(t, aggregate, got)

	aggregate.Members = []compositionrole.CompositionMember{
		{
			Instrument: compositionrole.InstrumentRef{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				Symbol:       "005930",
				Name:         "삼성전자",
			},
			Weight: compositionrole.DecimalValue{Value: "29.01"},
		},
	}
	writeResult, err = repository.UpsertComposition(ctx, aggregate)
	require.NoError(t, err)
	require.Equal(t, 1, writeResult.CompositionsStored)
	require.Equal(t, 1, writeResult.MembersStored)

	got, err = repository.GetComposition(ctx, compositionservice.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		AsOfDate:     "2026-05-17",
	})
	require.NoError(t, err)
	require.Len(t, got.Members, 1)
	require.Equal(t, "005930", got.Members[0].Instrument.Symbol)
	require.Equal(t, "29.01", got.Members[0].Weight.Value)
}

func requireCompositionEqual(t *testing.T, want compositionrole.Composition, got compositionrole.Composition) {
	t.Helper()

	require.Equal(t, want.Source, got.Source)
	require.Equal(t, want.Subject, got.Subject)
	require.Equal(t, want.AsOfDate, got.AsOfDate)
	require.Equal(t, want.ObservedAtMS, got.ObservedAtMS)
	require.Equal(t, want.Members, got.Members)
}

func assertMongoCompositionDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var stored struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Members       []bson.M `bson:"members"`
	}
	require.NoError(t, runtime.Database().
		Collection("compositions").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "compositions:krx:etf:069500:kis:quote:etfComponentStockPrice:2026-05-17:1779010800000"}}).
		Decode(&stored))
	require.Equal(t, "1.0.0", stored.SchemaVersion)
	require.GreaterOrEqual(t, stored.Revision, int64(2))
	require.Len(t, stored.Members, 1)
	require.Equal(t, "005930", stored.Members[0]["symbol"])
}
