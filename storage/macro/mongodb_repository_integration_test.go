//go:build integration

package macro

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoMacroRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_macro_contract_test",
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

	assertMacroRepositoryContract(t, sqliteReader, sqliteWriter)
	assertMacroRepositoryContract(t, mongoReader, mongoWriter)
	assertMongoMacroDocumentShape(t, runtime)
}

func assertMacroRepositoryContract(t *testing.T, reader macroservice.ReadRepository, writer macroservice.WriteRepository) {
	t.Helper()

	ctx := context.Background()
	indicator := macrorole.Indicator{
		ID:           "ecos.base-rate",
		Preset:       macrorole.PresetKeyStatistics,
		Provider:     provider.ProviderECOS,
		SourceCode:   "722Y001",
		SourceName:   "ECOS key statistics",
		SourceURL:    "https://ecos.bok.or.kr",
		Name:         "Bank of Korea base rate",
		FriendlyName: "base rate",
		Category:     "rates",
		Frequency:    macrorole.FrequencyMonthly,
		Unit:         "%",
		Scale:        "percent",
		Active:       true,
		ProviderDoc: &macrorole.ProviderDocument{
			Provider:      provider.ProviderECOS,
			SchemaVersion: "1.0.0",
			Document: map[string]any{
				"stat_code": "722Y001",
				"item_code": "0101000",
			},
			UpdatedAt: "2024-04-12T00:00:00Z",
		},
	}
	result, err := writer.UpsertIndicators(ctx, []macrorole.Indicator{indicator})
	require.NoError(t, err)
	require.Equal(t, 1, result.IndicatorsWritten)
	require.Equal(t, 1, result.SourcesWritten)
	require.Equal(t, 1, result.DocumentsWritten)

	indicator.FriendlyName = "policy rate"
	indicator.ProviderDoc.Document["item_code"] = "0101001"
	result, err = writer.UpsertIndicators(ctx, []macrorole.Indicator{indicator})
	require.NoError(t, err)
	require.Equal(t, 1, result.IndicatorsWritten)

	indicators, err := reader.QueryIndicators(ctx, macroservice.IndicatorQuery{
		ProviderID: provider.ProviderECOS,
		Preset:     macrorole.PresetKeyStatistics,
	})
	require.NoError(t, err)
	require.Len(t, indicators, 1)
	require.Equal(t, "policy rate", indicators[0].FriendlyName)
	require.Equal(t, "ECOS key statistics", indicators[0].SourceName)
	require.True(t, indicators[0].Active)

	observations := []macrorole.Observation{
		{
			IndicatorID: "ecos.base-rate",
			Provider:    provider.ProviderECOS,
			SourceCode:  "722Y001",
			Period:      "2024-04",
			Value:       "3.50",
			PublishedAt: "2024-04-11",
			CollectedAt: "2024-04-12T00:00:00Z",
			Revision:    0,
		},
		{
			IndicatorID: "ecos.base-rate",
			Provider:    provider.ProviderECOS,
			SourceCode:  "722Y001",
			Period:      "2024-05",
			Value:       "3.50",
			PublishedAt: "2024-05-11",
			CollectedAt: "2024-05-12T00:00:00Z",
			Revision:    0,
		},
	}
	observationResult, err := writer.UpsertObservations(ctx, observations)
	require.NoError(t, err)
	require.Equal(t, 2, observationResult.ObservationsWritten)

	gotObservations, err := reader.QueryObservations(ctx, macroservice.ObservationQuery{
		IndicatorID: "ecos.base-rate",
		From:        "2024-04",
		To:          "2024-04",
	})
	require.NoError(t, err)
	require.Len(t, gotObservations, 1)
	require.Equal(t, "2024-04", gotObservations[0].Period)
	require.Equal(t, provider.ProviderECOS, gotObservations[0].Provider)
	require.Equal(t, "722Y001", gotObservations[0].SourceCode)
}

func assertMongoMacroDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var indicator struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Sources       []bson.M `bson:"sources"`
		ProviderDocs  []struct {
			Document map[string]string `bson:"document"`
		} `bson:"provider_docs"`
	}
	require.NoError(t, runtime.Database().
		Collection("macro_indicators").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "macro_indicators:ecos.base-rate"}}).
		Decode(&indicator))
	require.Equal(t, "1.0.0", indicator.SchemaVersion)
	require.GreaterOrEqual(t, indicator.Revision, int64(2))
	require.Len(t, indicator.Sources, 1)
	require.Len(t, indicator.ProviderDocs, 1)
	require.Equal(t, "0101001", indicator.ProviderDocs[0].Document["item_code"])

	var observation struct {
		ID            string    `bson:"_id"`
		SchemaVersion string    `bson:"schema_version"`
		Revision      int64     `bson:"revision"`
		CollectedAt   time.Time `bson:"collected_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("macro_observations").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "macro_observations:ecos.base-rate:2024-04:0"}}).
		Decode(&observation))
	require.Equal(t, "1.0.0", observation.SchemaVersion)
	require.Equal(t, int64(1), observation.Revision)
	require.Equal(t, "2024-04-12T00:00:00.000Z", storagemongodb.ISOTime(observation.CollectedAt).String())
}
