//go:build integration

package aggregate_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	storageaggregate "github.com/awuzag/mwosa/storage/aggregate"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestServiceRunsLocalCollectionAggregateJQ(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_aggregate_executor_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	_, err = runtime.Database().Collection("candidate_source").InsertMany(ctx, []any{
		bson.M{"symbol": "005930", "market": "krx", "traded_amount": 1000, "change_pct": 3.421},
		bson.M{"symbol": "000660", "market": "krx", "traded_amount": 2000, "change_pct": 2.1},
		bson.M{"symbol": "AAPL", "market": "us", "traded_amount": 5000, "change_pct": 1.1},
	})
	require.NoError(t, err)

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	executor, err := aggregateservice.NewMongoExecutor(runtime.Database())
	require.NoError(t, err)
	service, err := aggregateservice.NewService(repository, aggregateservice.WithExecutor(executor))
	require.NoError(t, err)

	spec, err := aggregateservice.LoadSpecBytes(ctx, []byte(`
kind: Aggregate
schema_version: 1
name: krx-candidates
params:
  market:
    type: string
    default: krx
  limit:
    type: int
    default: 2
workspace:
  ttl: 2h
pipeline:
  - name: source
    type: local_collection
    collection: candidate_source
    filter:
      market: "${params.market}"
  - name: ranked
    type: aggregate
    from: source
    pipeline:
      - $sort:
          traded_amount: -1
      - $limit: "${params.limit}"
  - name: shaped
    type: jq
    from: ranked
    query: |
      map({symbol, traded_amount, change_pct})
output:
  from: shaped
  default_format: table
  columns:
    - key: ordinal
      title: "#"
      format: integer
    - key: symbol
      title: 코드
    - key: traded_amount
      title: 거래대금
      format: number
      precision: 0
`))
	require.NoError(t, err)
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: "krx-candidates", Spec: spec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)

	detail, output, err := service.Run(ctx, aggregateservice.RunRequest{Name: "krx-candidates", Alias: "latest", Params: []string{"limit=1"}})
	require.NoError(t, err)
	require.Equal(t, aggregateservice.RunSucceeded, detail.Run.Status)
	require.Len(t, detail.Items, 1)
	assert.JSONEq(t, `{"symbol":"000660","traded_amount":2000,"change_pct":2.1}`, string(detail.Items[0].PayloadJSON))
	assertSharedStageItems(t, ctx, runtime.Database(), detail.Run.ID, "source", 2, 2*time.Hour)

	header, rows := output.TableRows()
	assert.Equal(t, []string{"#", "코드", "거래대금"}, header)
	assert.Equal(t, [][]string{{"1", "000660", "2000"}}, rows)

	limited := spec
	limited.Name = "limited-candidates"
	limited.Workspace.MaxRows = 1
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: limited.Name, Spec: limited, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)
	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: limited.Name, Alias: "limited"})
	require.ErrorContains(t, err, "stage row limit exceeded")

	history, err := service.History(ctx, aggregateservice.RunHistoryFilter{Name: "krx-candidates"})
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "latest", history[0].Alias)
	_, err = service.History(ctx, aggregateservice.RunHistoryFilter{Status: "unknown"})
	require.ErrorContains(t, err, "unsupported aggregate run status")

	loaded, err := service.InspectRun(ctx, "latest", 10)
	require.NoError(t, err)
	require.Len(t, loaded.Items, 1)

	reuseSpec, err := aggregateservice.LoadSpecBytes(ctx, []byte(`
kind: Aggregate
schema_version: 1
name: reuse-candidates
pipeline:
  - name: previous
    type: aggregate_run
    run: latest
  - name: shaped
    type: jq
    from: previous
    query: |
      map({symbol, reused: true})
output:
  from: shaped
  default_format: table
  columns:
    - key: symbol
      title: 코드
    - key: reused
      title: reused
`))
	require.NoError(t, err)
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: "reuse-candidates", Spec: reuseSpec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)
	reused, _, err := service.Run(ctx, aggregateservice.RunRequest{Name: "reuse-candidates", Alias: "reused"})
	require.NoError(t, err)
	require.Len(t, reused.Items, 1)
	assert.JSONEq(t, `{"symbol":"000660","reused":true}`, string(reused.Items[0].PayloadJSON))
}

func assertSharedStageItems(t *testing.T, ctx context.Context, database *mongo.Database, runID string, stage string, expected int64, ttl time.Duration) {
	t.Helper()
	collection := database.Collection("aggregate_stage_items")
	count, err := collection.CountDocuments(ctx, bson.D{{Key: "_aggregate_run_id", Value: runID}, {Key: "_aggregate_stage", Value: stage}})
	require.NoError(t, err)
	assert.Equal(t, expected, count)
	if expected > 0 {
		var item struct {
			ExpiresAt time.Time `bson:"expires_at"`
		}
		require.NoError(t, collection.FindOne(ctx, bson.D{{Key: "_aggregate_run_id", Value: runID}, {Key: "_aggregate_stage", Value: stage}}).Decode(&item))
		assert.WithinDuration(t, time.Now().Add(ttl), item.ExpiresAt, time.Minute)
	}

	names, err := database.ListCollectionNames(ctx, bson.D{})
	require.NoError(t, err)
	for _, name := range names {
		assert.NotContains(t, name, "aggregate_tmp_", "aggregate execution must not create per-run collections")
	}

	cursor, err := collection.Indexes().List(ctx)
	require.NoError(t, err)
	defer cursor.Close(ctx)

	var indexes []bson.M
	require.NoError(t, cursor.All(ctx, &indexes))
	for _, index := range indexes {
		if index["name"] == "aggregate_stage_items_expires_at_ttl" {
			assert.Equal(t, int32(0), index["expireAfterSeconds"])
			return
		}
	}
	t.Fatalf("aggregate stage item ttl index not found: %#v", indexes)
}

func TestServicePersistsFailedAggregateRun(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_aggregate_failed_executor_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	executor, err := aggregateservice.NewMongoExecutor(runtime.Database())
	require.NoError(t, err)
	service, err := aggregateservice.NewService(repository, aggregateservice.WithExecutor(executor))
	require.NoError(t, err)

	spec := minimalAggregateSpec()
	spec.Name = "broken"
	spec.Pipeline[0].Collection = "broken_source"
	spec.Pipeline = append(spec.Pipeline, aggregateservice.StageSpec{Name: "broken_jq", Type: aggregateservice.StageJQ, From: "universe", Query: "map("})
	spec.Output.From = "broken_jq"
	_, err = runtime.Database().Collection("broken_source").InsertOne(ctx, bson.M{"symbol": "005930"})
	require.NoError(t, err)
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: "broken", Spec: spec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)

	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: "broken", Alias: "failed"})
	require.ErrorContains(t, err, "execute aggregate jq")

	loaded, err := service.InspectRun(ctx, "failed", 10)
	require.NoError(t, err)
	assert.Equal(t, aggregateservice.RunFailed, loaded.Run.Status)
	assert.Contains(t, loaded.Run.ErrorMessage, "execute aggregate jq")
	assertSharedStageItems(t, ctx, runtime.Database(), loaded.Run.ID, "universe", 1, 24*time.Hour)
}

func TestServiceRejectsMissingLocalCollection(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI, Database: "mwosa_aggregate_missing_collection_test"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close(context.Background())) })
	require.NoError(t, runtime.Init(ctx))

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	executor, err := aggregateservice.NewMongoExecutor(runtime.Database())
	require.NoError(t, err)
	service, err := aggregateservice.NewService(repository, aggregateservice.WithExecutor(executor))
	require.NoError(t, err)

	spec := minimalAggregateSpec()
	spec.Name = "missing-source"
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: spec.Name, Spec: spec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)
	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: spec.Name, Alias: "missing-source-run"})
	require.ErrorContains(t, err, "local collection does not exist")

	loaded, err := service.InspectRun(ctx, "missing-source-run", 10)
	require.NoError(t, err)
	assert.Equal(t, aggregateservice.RunFailed, loaded.Run.Status)
	var stages []aggregateservice.StageSummary
	require.NoError(t, json.Unmarshal(loaded.Run.StagesJSON, &stages))
	require.Len(t, stages, 1)
	assert.Equal(t, "universe", stages[0].Name)
	assert.Contains(t, stages[0].Error, "local collection does not exist")
}

func TestServiceRecordsCancelledRunWithExecutionID(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI, Database: "mwosa_aggregate_cancelled_run_test"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close(context.Background())) })
	require.NoError(t, runtime.Init(ctx))

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	executor := &cancelExecutor{}
	service, err := aggregateservice.NewService(repository, aggregateservice.WithExecutor(executor))
	require.NoError(t, err)
	spec := minimalAggregateSpec()
	spec.Name = "cancelled"
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: spec.Name, Spec: spec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)

	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: spec.Name, Alias: "cancelled-run"})
	require.ErrorIs(t, err, context.Canceled)
	loaded, err := service.InspectRun(ctx, "cancelled-run", 10)
	require.NoError(t, err)
	assert.Equal(t, aggregateservice.RunCancelled, loaded.Run.Status)
	assert.Equal(t, executor.runID, loaded.Run.ID)
}

type cancelExecutor struct {
	runID string
}

func (e *cancelExecutor) Execute(_ context.Context, _ aggregateservice.Spec, _ map[string]any, runID string) (aggregateservice.ExecutionResult, error) {
	e.runID = runID
	return aggregateservice.ExecutionResult{Stages: []aggregateservice.StageSummary{{Name: "source", Type: "local_collection", Status: "failed"}}}, context.Canceled
}

func TestServiceRejectsDuplicateAliasBeforeExecution(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI, Database: "mwosa_aggregate_duplicate_alias_test"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close(context.Background())) })
	require.NoError(t, runtime.Init(ctx))

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	executor := &countingExecutor{}
	service, err := aggregateservice.NewService(repository, aggregateservice.WithExecutor(executor))
	require.NoError(t, err)
	spec := minimalAggregateSpec()
	spec.Name = "alias-check"
	_, err = service.Upsert(ctx, aggregateservice.UpsertRequest{Name: spec.Name, Spec: spec, YAMLText: "kind: Aggregate\n"})
	require.NoError(t, err)

	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: spec.Name, Alias: "same-alias"})
	require.NoError(t, err)
	_, _, err = service.Run(ctx, aggregateservice.RunRequest{Name: spec.Name, Alias: "same-alias"})
	require.ErrorContains(t, err, "alias already exists")
	assert.Equal(t, 1, executor.calls)
}

type countingExecutor struct {
	calls int
}

func (e *countingExecutor) Execute(_ context.Context, _ aggregateservice.Spec, _ map[string]any, _ string) (aggregateservice.ExecutionResult, error) {
	e.calls++
	return aggregateservice.ExecutionResult{Rows: []json.RawMessage{}, Stages: []aggregateservice.StageSummary{}}, nil
}

func TestServiceDoesNotCreateDuplicateActiveVersion(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI, Database: "mwosa_aggregate_duplicate_version_test"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close(context.Background())) })
	require.NoError(t, runtime.Init(ctx))

	repository, err := storageaggregate.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	service, err := aggregateservice.NewService(repository)
	require.NoError(t, err)
	spec := minimalAggregateSpec()
	spec.Name = "deduplicated"
	request := aggregateservice.UpsertRequest{Name: spec.Name, Spec: spec, YAMLText: "kind: Aggregate\n"}
	_, err = service.Upsert(ctx, request)
	require.NoError(t, err)
	_, err = service.Upsert(ctx, request)
	require.NoError(t, err)

	detail, err := service.Inspect(ctx, spec.Name, aggregateservice.VersionRef{})
	require.NoError(t, err)
	assert.Len(t, detail.Versions, 1)
}

func minimalAggregateSpec() aggregateservice.Spec {
	return aggregateservice.Spec{
		Kind:          aggregateservice.KindAggregate,
		SchemaVersion: 1,
		Name:          "krx-candidates",
		Params: map[string]aggregateservice.ParamSpec{
			"as_of": {Type: aggregateservice.ParamDate, Default: "2026-07-01"},
			"limit": {Type: aggregateservice.ParamInt, Default: 20},
		},
		Pipeline: []aggregateservice.StageSpec{
			{Name: "universe", Type: aggregateservice.StageLocalCollection, Collection: "missing_collection"},
		},
		Output: aggregateservice.OutputSpec{
			From:          "universe",
			DefaultFormat: "table",
			Columns:       []aggregateservice.OutputColumnSpec{{Key: "symbol", Title: "코드"}},
		},
	}
}
