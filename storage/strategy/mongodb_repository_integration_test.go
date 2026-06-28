//go:build integration

package strategy

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoStrategyRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_strategy_contract_test",
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

	assertStrategyRepositoryContract(t, sqliteRepository)
	assertStrategyRepositoryContract(t, mongoRepository)
	assertMongoStrategyDocumentShape(t, runtime)
}

func assertStrategyRepositoryContract(t *testing.T, repository strategyservice.Repository) {
	t.Helper()

	ctx := context.Background()
	createdAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	strategy := strategyservice.Strategy{
		ID:              "strategy-1",
		Name:            "etf-leaders",
		Engine:          strategyservice.EngineJQ,
		ActiveVersionID: "strategy-version-1",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	version := strategyservice.StrategyVersion{
		ID:                 "strategy-version-1",
		StrategyID:         "strategy-1",
		Version:            1,
		QueryText:          `.[:10]`,
		QueryHash:          "query-hash-1",
		InputDataset:       "etf_daily_metrics",
		InputSchemaVersion: 1,
		ParamsJSON:         json.RawMessage(`{"limit":10}`),
		SpecJSON:           json.RawMessage(`{"kind":"ScreenStrategy","schema_version":1}`),
		SpecHash:           "spec-hash-1",
		CreatedAt:          createdAt,
		Note:               "first",
	}
	detail, err := repository.CreateStrategyWithVersion(ctx, strategy, version)
	require.NoError(t, err)
	require.Equal(t, "etf-leaders", detail.Strategy.Name)
	require.Equal(t, 1, detail.ActiveVersion.Version)

	secondVersion := version
	secondVersion.ID = "strategy-version-2"
	secondVersion.Version = 2
	secondVersion.QueryText = `.[:5]`
	secondVersion.QueryHash = "query-hash-2"
	secondVersion.SpecHash = "spec-hash-2"
	secondVersion.Note = "second"
	secondVersion.CreatedAt = createdAt.Add(time.Hour)
	detail, err = repository.AddStrategyVersion(ctx, "etf-leaders", strategyservice.EngineJQ, secondVersion, createdAt.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, "strategy-version-2", detail.Strategy.ActiveVersionID)
	require.Equal(t, 2, detail.ActiveVersion.Version)

	latest, err := repository.GetStrategyVersion(ctx, "etf-leaders", strategyservice.StrategyVersionRef{Version: "latest"})
	require.NoError(t, err)
	require.Equal(t, "strategy-version-2", latest.ActiveVersion.ID)

	pinned, err := repository.GetStrategyVersion(ctx, "etf-leaders", strategyservice.StrategyVersionRef{SpecHash: "spec-hash-1"})
	require.NoError(t, err)
	require.Equal(t, "strategy-version-1", pinned.ActiveVersion.ID)
	require.JSONEq(t, `{"limit":10}`, string(pinned.ActiveVersion.ParamsJSON))

	list, err := repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "strategy-version-2", list[0].ActiveVersion.ID)

	finishedAt := createdAt.Add(2 * time.Hour)
	run := strategyservice.ScreenRun{
		ID:                 "run-1",
		Alias:              "latest-run",
		StrategyID:         "strategy-1",
		StrategyVersionID:  "strategy-version-2",
		QueryHash:          "query-hash-2",
		InputDataset:       "etf_daily_metrics",
		InputSchemaVersion: 1,
		ParamsJSON:         json.RawMessage(`{"limit":5}`),
		DataAsOf:           "2026-05-01",
		StartedAt:          createdAt.Add(time.Hour),
		FinishedAt:         &finishedAt,
		Status:             strategyservice.ScreenRunSucceeded,
		ResultCount:        1,
		ResultHash:         "result-hash",
		ResultSizeBytes:    42,
		SummaryJSON:        json.RawMessage(`{"count":1}`),
	}
	items := []strategyservice.ScreenRunItem{
		{
			ID:          "run-item-1",
			ScreenRunID: "run-1",
			Ordinal:     0,
			Symbol:      "069500",
			PayloadJSON: json.RawMessage(`{"symbol":"069500","score":100}`),
		},
	}
	runDetail, err := repository.CreateScreenRun(ctx, run, items)
	require.NoError(t, err)
	require.Equal(t, "etf-leaders", runDetail.Strategy.Name)
	require.Len(t, runDetail.Items, 1)

	history, err := repository.ListScreenRuns(ctx, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "latest-run", history[0].Alias)

	loadedRun, err := repository.GetScreenRun(ctx, "latest-run")
	require.NoError(t, err)
	require.Equal(t, "run-1", loadedRun.Run.ID)
	require.Equal(t, "069500", loadedRun.Items[0].Symbol)
	require.JSONEq(t, `{"symbol":"069500","score":100}`, string(loadedRun.Items[0].PayloadJSON))

	require.NoError(t, repository.ArchiveStrategy(ctx, "etf-leaders", createdAt.Add(3*time.Hour)))
	list, err = repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Empty(t, list)
}

func assertMongoStrategyDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var strategy struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Versions      []bson.M `bson:"versions"`
	}
	require.NoError(t, runtime.Database().
		Collection("screen_strategies").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "screen_strategies:strategy-1"}}).
		Decode(&strategy))
	require.Equal(t, "1.0.0", strategy.SchemaVersion)
	require.GreaterOrEqual(t, strategy.Revision, int64(3))
	require.Len(t, strategy.Versions, 2)

	var run struct {
		ID               string `bson:"_id"`
		SchemaVersion    string `bson:"schema_version"`
		StrategySnapshot bson.M `bson:"strategy_snapshot"`
	}
	require.NoError(t, runtime.Database().
		Collection("screen_runs").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "screen_runs:run-1"}}).
		Decode(&run))
	require.Equal(t, "1.0.0", run.SchemaVersion)
	require.Equal(t, "etf-leaders", run.StrategySnapshot["name"])

	var item struct {
		ID      string `bson:"_id"`
		Payload bson.M `bson:"payload"`
	}
	require.NoError(t, runtime.Database().
		Collection("screen_run_items").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "screen_run_items:run-1:0"}}).
		Decode(&item))
	require.Equal(t, "069500", item.Payload["symbol"])
}
