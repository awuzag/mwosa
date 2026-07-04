//go:build integration

package aggregate

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoAggregateRepositoryContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_aggregate_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	repository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	assertAggregateRepositoryContract(t, repository)
	assertMongoAggregateDocumentShape(t, runtime)
}

func assertAggregateRepositoryContract(t *testing.T, repository aggregateservice.Repository) {
	t.Helper()

	ctx := context.Background()
	createdAt := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	aggregate := aggregateservice.Aggregate{
		ID:              "aggregate-1",
		Name:            "krx-candidates",
		ActiveVersionID: "version-1",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	version := aggregateservice.Version{
		ID:          "version-1",
		AggregateID: "aggregate-1",
		Version:     1,
		YAMLText:    "kind: Aggregate\n",
		SpecJSON:    json.RawMessage(`{"kind":"Aggregate","schema_version":1}`),
		SpecHash:    "spec-hash-1",
		CreatedAt:   createdAt,
		Note:        "first",
	}
	detail, err := repository.CreateAggregateWithVersion(ctx, aggregate, version)
	require.NoError(t, err)
	require.Equal(t, "krx-candidates", detail.Aggregate.Name)
	require.Equal(t, 1, detail.ActiveVersion.Version)

	secondVersion := version
	secondVersion.ID = "version-2"
	secondVersion.Version = 2
	secondVersion.SpecHash = "spec-hash-2"
	secondVersion.CreatedAt = createdAt.Add(time.Hour)
	detail, err = repository.AddAggregateVersion(ctx, "krx-candidates", secondVersion, createdAt.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, "version-2", detail.Aggregate.ActiveVersionID)

	pinned, err := repository.GetAggregateVersion(ctx, "krx-candidates", aggregateservice.VersionRef{SpecHash: "spec-hash-1"})
	require.NoError(t, err)
	require.Equal(t, "version-1", pinned.ActiveVersion.ID)
	require.JSONEq(t, `{"kind":"Aggregate","schema_version":1}`, string(pinned.ActiveVersion.SpecJSON))

	list, err := repository.ListAggregates(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "version-2", list[0].ActiveVersion.ID)

	finishedAt := createdAt.Add(2 * time.Hour)
	run := aggregateservice.Run{
		ID:                 "run-1",
		Alias:              "latest-run",
		AggregateID:        "aggregate-1",
		AggregateVersionID: "version-2",
		AggregateName:      "krx-candidates",
		Version:            2,
		SpecHash:           "spec-hash-2",
		ParamsJSON:         json.RawMessage(`{"limit":5}`),
		StagesJSON:         json.RawMessage(`[{"name":"universe","status":"succeeded"}]`),
		PipelineJSON:       json.RawMessage(`[{"name":"universe","type":"local_collection"}]`),
		StartedAt:          createdAt.Add(time.Hour),
		FinishedAt:         &finishedAt,
		Status:             aggregateservice.RunSucceeded,
		ResultCount:        1,
		ResultHash:         "result-hash",
		ResultSizeBytes:    42,
		SummaryJSON:        json.RawMessage(`{"count":1}`),
	}
	items := []aggregateservice.RunItem{
		{
			ID:          "item-1",
			RunID:       "run-1",
			Ordinal:     0,
			PayloadJSON: json.RawMessage(`{"symbol":"005930","score":100}`),
		},
	}
	runDetail, err := repository.CreateRun(ctx, run, items)
	require.NoError(t, err)
	require.Equal(t, "krx-candidates", runDetail.Aggregate.Name)
	require.Len(t, runDetail.Items, 1)

	history, err := repository.ListRuns(ctx, aggregateservice.RunHistoryFilter{Name: "krx-candidates", Limit: 10})
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "latest-run", history[0].Alias)

	loadedRun, err := repository.GetRun(ctx, "latest-run", 10)
	require.NoError(t, err)
	require.Equal(t, "run-1", loadedRun.Run.ID)
	require.JSONEq(t, `{"symbol":"005930","score":100}`, string(loadedRun.Items[0].PayloadJSON))

	require.NoError(t, repository.ArchiveAggregate(ctx, "krx-candidates", createdAt.Add(3*time.Hour)))
	list, err = repository.ListAggregates(ctx)
	require.NoError(t, err)
	require.Empty(t, list)
}

func assertMongoAggregateDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var aggregate struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Name          string `bson:"name"`
	}
	require.NoError(t, runtime.Database().
		Collection("aggregates").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "aggregates:aggregate-1"}}).
		Decode(&aggregate))
	require.Equal(t, "1.0.0", aggregate.SchemaVersion)
	require.Equal(t, "krx-candidates", aggregate.Name)
	require.GreaterOrEqual(t, aggregate.Revision, int64(3))

	var version struct {
		ID       string `bson:"_id"`
		SpecHash string `bson:"spec_hash"`
	}
	require.NoError(t, runtime.Database().
		Collection("aggregate_versions").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "aggregate_versions:version-1"}}).
		Decode(&version))
	require.Equal(t, "spec-hash-1", version.SpecHash)

	var run struct {
		ID                string `bson:"_id"`
		AggregateSnapshot bson.M `bson:"aggregate_snapshot"`
	}
	require.NoError(t, runtime.Database().
		Collection("aggregate_runs").
		FindOne(context.Background(), bson.D{{Key: "_id", Value: "aggregate_runs:run-1"}}).
		Decode(&run))
	require.Equal(t, "krx-candidates", run.AggregateSnapshot["name"])
}
