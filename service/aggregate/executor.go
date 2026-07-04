package aggregate

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type StageSummary struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Rows       int    `json:"rows"`
	Collection string `json:"collection,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ExecutionResult struct {
	Rows   []json.RawMessage `json:"rows"`
	Stages []StageSummary    `json:"stages"`
}

type Executor interface {
	Execute(ctx context.Context, spec Spec, params map[string]any, runID string) (ExecutionResult, error)
}

type RawFetchRequest struct {
	Provider  string
	Operation string
	Input     map[string]string
	Context   map[string]any
}

type RawFetchResult struct {
	Provider  string `json:"provider"`
	Group     string `json:"provider_group,omitempty"`
	Operation string `json:"operation"`
	Endpoint  string `json:"endpoint,omitempty"`
	Response  any    `json:"response"`
	RowCount  int    `json:"row_count"`
	BaseDate  string `json:"base_date,omitempty"`
}

type RawFetcher interface {
	FetchRaw(ctx context.Context, req RawFetchRequest) (RawFetchResult, error)
}

type ProviderFetchRequest struct {
	Provider string
	Role     string
	Request  map[string]string
	Context  map[string]any
}

type ProviderFetchResult struct {
	Provider string         `json:"provider"`
	Role     string         `json:"role"`
	Payload  map[string]any `json:"payload"`
}

type ProviderFetcher interface {
	FetchProvider(ctx context.Context, req ProviderFetchRequest) (ProviderFetchResult, error)
}

type MongoExecutor struct {
	database        *mongo.Database
	providerFetcher ProviderFetcher
	rawFetcher      RawFetcher
	now             func() time.Time
}

type MongoExecutorOption func(*MongoExecutor) error

func WithRawFetcher(fetcher RawFetcher) MongoExecutorOption {
	return func(executor *MongoExecutor) error {
		executor.rawFetcher = fetcher
		return nil
	}
}

func WithProviderFetcher(fetcher ProviderFetcher) MongoExecutorOption {
	return func(executor *MongoExecutor) error {
		executor.providerFetcher = fetcher
		return nil
	}
}

func NewMongoExecutor(database *mongo.Database, opts ...MongoExecutorOption) (MongoExecutor, error) {
	if database == nil {
		return MongoExecutor{}, oops.In("aggregate_executor").New("mongodb database is nil")
	}
	executor := MongoExecutor{database: database, now: time.Now}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&executor); err != nil {
			return MongoExecutor{}, err
		}
	}
	return executor, nil
}

func (e MongoExecutor) Execute(ctx context.Context, spec Spec, params map[string]any, runID string) (ExecutionResult, error) {
	errb := oops.In("aggregate_executor").With("name", spec.Name, "run_id", runID)
	if e.database == nil {
		return ExecutionResult{}, errb.New("mongodb database is nil")
	}
	aliases := map[string]string{}
	stageRows := map[string][]json.RawMessage{}
	summaries := make([]StageSummary, 0, len(spec.Pipeline))
	for _, stage := range spec.Pipeline {
		collectionName := tempCollectionName(runID, stage.Name)
		summary := StageSummary{Name: stage.Name, Type: string(stage.Type), Status: "succeeded", Collection: collectionName}
		rows, err := e.executeStage(ctx, stage, TemplateContext{Params: params}, aliases, stageRows, collectionName)
		if err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, errb.With("stage", stage.Name, "type", stage.Type).Wrap(err)
		}
		if err := e.materialize(ctx, collectionName, rows, runID, stage.Name); err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, errb.With("stage", stage.Name).Wrap(err)
		}
		aliases[stage.Name] = collectionName
		stageRows[stage.Name] = rows
		summary.Rows = len(rows)
		summaries = append(summaries, summary)
	}
	rows, ok := stageRows[spec.Output.From]
	if !ok {
		return ExecutionResult{Stages: summaries}, errb.With("output_from", spec.Output.From).New("aggregate output stage was not produced")
	}
	return ExecutionResult{Rows: rows, Stages: summaries}, nil
}

func (e MongoExecutor) executeStage(ctx context.Context, stage StageSpec, template TemplateContext, aliases map[string]string, stageRows map[string][]json.RawMessage, collectionName string) ([]json.RawMessage, error) {
	switch stage.Type {
	case StageProvider:
		return e.executeProvider(ctx, stage, template, stageRows)
	case StageLocalCollection:
		return e.executeLocalCollection(ctx, stage, template)
	case StageLocalDataset:
		return e.executeLocalDataset(ctx, stage, template)
	case StageSnapshot:
		return e.executeSnapshot(ctx, stage, template)
	case StageAggregateRun:
		return e.executeAggregateRun(ctx, stage)
	case StageProviderRaw:
		return e.executeProviderRaw(ctx, stage, template, stageRows)
	case StageAggregate:
		return e.executeMongoAggregate(ctx, stage, template, aliases)
	case StageJQ:
		rows, ok := stageRows[stage.From]
		if !ok {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "from", stage.From).New("jq input stage was not produced")
		}
		return ExecuteJQRows(ctx, rows, stage.Query)
	default:
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "type", stage.Type).Errorf("unsupported aggregate stage type: %s", stage.Type)
	}
}

func (e MongoExecutor) executeProvider(ctx context.Context, stage StageSpec, template TemplateContext, stageRows map[string][]json.RawMessage) ([]json.RawMessage, error) {
	if e.providerFetcher == nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "provider", stage.Provider, "role", stage.Role).New("aggregate provider fetcher is not configured")
	}
	contexts, err := foreachContexts(stage, stageRows)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(contexts))
	for _, each := range contexts {
		resolved, err := ResolveTemplateValue(stage.Request, TemplateContext{Params: template.Params, Each: each})
		if err != nil {
			return nil, err
		}
		result, err := e.providerFetcher.FetchProvider(ctx, ProviderFetchRequest{
			Provider: stage.Provider,
			Role:     stage.Role,
			Request:  mapStringValues(resolved),
			Context:  each,
		})
		if err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "provider", stage.Provider, "role", stage.Role).Wrapf(err, "fetch provider")
		}
		row := map[string]any{}
		for key, value := range result.Payload {
			row[key] = value
		}
		row["context"] = each
		row["provider"] = result.Provider
		row["role"] = result.Role
		data, err := json.Marshal(row)
		if err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "encode provider row")
		}
		out = append(out, data)
	}
	return out, nil
}

func (e MongoExecutor) executeProviderRaw(ctx context.Context, stage StageSpec, template TemplateContext, stageRows map[string][]json.RawMessage) ([]json.RawMessage, error) {
	if e.rawFetcher == nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "provider", stage.Provider).New("aggregate provider_raw fetcher is not configured")
	}
	contexts, err := foreachContexts(stage, stageRows)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(contexts))
	for _, each := range contexts {
		resolved, err := ResolveTemplateValue(stage.Params, TemplateContext{Params: template.Params, Each: each})
		if err != nil {
			return nil, err
		}
		input := mapStringValues(resolved)
		result, err := e.rawFetcher.FetchRaw(ctx, RawFetchRequest{
			Provider:  stage.Provider,
			Operation: stage.Operation,
			Input:     input,
			Context:   each,
		})
		if err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "provider", stage.Provider, "operation", stage.Operation).Wrapf(err, "fetch provider raw")
		}
		row := map[string]any{
			"context":        each,
			"provider":       result.Provider,
			"provider_group": result.Group,
			"operation":      result.Operation,
			"endpoint":       result.Endpoint,
			"row_count":      result.RowCount,
			"base_date":      result.BaseDate,
			"payload":        result.Response,
		}
		data, err := json.Marshal(row)
		if err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "encode provider_raw row")
		}
		out = append(out, data)
	}
	return out, nil
}

func foreachContexts(stage StageSpec, stageRows map[string][]json.RawMessage) ([]map[string]any, error) {
	contexts := []map[string]any{{}}
	if stage.Foreach == nil {
		return contexts, nil
	}
	rows, ok := stageRows[stage.Foreach.Stage]
	if !ok {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "foreach_stage", stage.Foreach.Stage).New("foreach stage was not produced")
	}
	contexts = make([]map[string]any, 0, len(rows))
	for index, row := range rows {
		var object map[string]any
		if err := json.Unmarshal(row, &object); err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "row", index).Wrapf(err, "decode foreach row")
		}
		value, ok := object[stage.Foreach.Field]
		if !ok {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "field", stage.Foreach.Field).New("foreach field is missing")
		}
		contexts = append(contexts, map[string]any{stage.Foreach.As: value})
	}
	return contexts, nil
}

func (e MongoExecutor) executeLocalCollection(ctx context.Context, stage StageSpec, template TemplateContext) ([]json.RawMessage, error) {
	filterValue, err := ResolveTemplateValue(stage.Filter, template)
	if err != nil {
		return nil, err
	}
	filter := bson.D{}
	if filterValue != nil {
		filter = bsonDocument(filterValue)
	}
	cursor, err := e.database.Collection(stage.Collection).Find(ctx, filter)
	if err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "collection", stage.Collection).Wrapf(err, "query local collection")
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "collection", stage.Collection).Wrapf(err, "decode local collection")
	}
	return bsonDocumentsToRows(docs)
}

func (e MongoExecutor) executeLocalDataset(ctx context.Context, stage StageSpec, template TemplateContext) ([]json.RawMessage, error) {
	collection := strings.TrimSpace(stage.Dataset)
	if collection == "" {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).New("local_dataset requires dataset")
	}
	localStage := stage
	localStage.Collection = collection
	return e.executeLocalCollection(ctx, localStage, template)
}

func (e MongoExecutor) executeSnapshot(ctx context.Context, stage StageSpec, template TemplateContext) ([]json.RawMessage, error) {
	paramsValue, err := ResolveTemplateValue(stage.Params, template)
	if err != nil {
		return nil, err
	}
	params := mapStringValues(paramsValue)
	filter := bson.D{}
	if strings.TrimSpace(stage.Provider) != "" {
		filter = append(filter, bson.E{Key: "source.provider", Value: strings.TrimSpace(stage.Provider)})
	}
	if group := strings.TrimSpace(params["provider_group"]); group != "" {
		filter = append(filter, bson.E{Key: "source.provider_group", Value: group})
	}
	if strings.TrimSpace(stage.Operation) != "" {
		filter = append(filter, bson.E{Key: "source.operation", Value: strings.TrimSpace(stage.Operation)})
	}
	if baseDate := firstNonEmpty(params["base_date"], params["as_of"]); baseDate != "" {
		filter = append(filter, bson.E{Key: "source.base_date", Value: baseDate})
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "source.base_date", Value: -1}})
	if limitText := strings.TrimSpace(params["limit"]); limitText != "" {
		if limit, parseErr := strconv.ParseInt(limitText, 10, 64); parseErr == nil && limit > 0 {
			findOptions.SetLimit(limit)
		}
	}
	cursor, err := e.database.Collection("provider_raw_snapshots").Find(ctx, filter, findOptions)
	if err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "query provider raw snapshots")
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "decode provider raw snapshots")
	}
	return bsonDocumentsToRows(docs)
}

func (e MongoExecutor) executeAggregateRun(ctx context.Context, stage StageSpec) ([]json.RawMessage, error) {
	runFilter := bson.D{}
	if ref := strings.TrimSpace(stage.Run); ref != "" {
		runFilter = bson.D{{Key: "$or", Value: bson.A{bson.D{{Key: "run_id", Value: ref}}, bson.D{{Key: "alias", Value: ref}}}}}
	} else if name := strings.TrimSpace(stage.Aggregate); name != "" {
		runFilter = bson.D{{Key: "aggregate_name", Value: name}, {Key: "status", Value: string(RunSucceeded)}}
	} else {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).New("aggregate_run requires run or aggregate")
	}
	var runDoc bson.M
	if err := e.database.Collection("aggregate_runs").FindOne(ctx, runFilter, options.FindOne().SetSort(bson.D{{Key: "started_at", Value: -1}})).Decode(&runDoc); err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "load aggregate run")
	}
	runID, _ := runDoc["run_id"].(string)
	if runID == "" {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).New("aggregate run document missing run_id")
	}
	cursor, err := e.database.Collection("aggregate_run_items").Find(ctx, bson.D{{Key: "run_id", Value: runID}}, options.Find().SetSort(bson.D{{Key: "ordinal", Value: 1}}))
	if err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "run_id", runID).Wrapf(err, "load aggregate run items")
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "run_id", runID).Wrapf(err, "decode aggregate run items")
	}
	rows := make([]json.RawMessage, 0, len(docs))
	for _, doc := range docs {
		payload, _ := doc["payload"].(any)
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, oops.In("aggregate_executor").With("stage", stage.Name, "run_id", runID).Wrapf(err, "encode aggregate run item")
		}
		rows = append(rows, data)
	}
	return rows, nil
}

func (e MongoExecutor) executeMongoAggregate(ctx context.Context, stage StageSpec, template TemplateContext, aliases map[string]string) ([]json.RawMessage, error) {
	source, ok := aliases[stage.From]
	if !ok {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "from", stage.From).New("aggregate input stage was not produced")
	}
	pipeline, err := resolveMongoPipeline(stage.Pipeline, template, aliases)
	if err != nil {
		return nil, err
	}
	cursor, err := e.database.Collection(source).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "from", stage.From).Wrapf(err, "execute mongodb aggregation")
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name).Wrapf(err, "decode mongodb aggregation")
	}
	return bsonDocumentsToRows(docs)
}

func (e MongoExecutor) materialize(ctx context.Context, collectionName string, rows []json.RawMessage, runID string, stage string) error {
	collection := e.database.Collection(collectionName)
	if err := ensureTempCollectionIndex(ctx, collection); err != nil {
		return oops.In("aggregate_executor").With("stage", stage, "collection", collectionName).Wrapf(err, "create aggregate temp ttl index")
	}
	_, _ = collection.DeleteMany(ctx, bson.D{})
	if len(rows) == 0 {
		return nil
	}
	docs := make([]any, 0, len(rows))
	expiresAt := e.now().Add(24 * time.Hour)
	for index, row := range rows {
		var doc bson.M
		if err := bson.UnmarshalExtJSON(row, true, &doc); err != nil {
			var generic map[string]any
			if jsonErr := json.Unmarshal(row, &generic); jsonErr != nil {
				return oops.In("aggregate_executor").With("stage", stage, "row", index).Wrapf(jsonErr, "decode materialized row")
			}
			doc = bson.M(generic)
		}
		delete(doc, "_id")
		doc["_aggregate_run_id"] = runID
		doc["_aggregate_stage"] = stage
		doc["expires_at"] = expiresAt
		docs = append(docs, doc)
	}
	if _, err := collection.InsertMany(ctx, docs, options.InsertMany().SetOrdered(true)); err != nil {
		return oops.In("aggregate_executor").With("stage", stage, "collection", collectionName).Wrapf(err, "materialize aggregate stage")
	}
	return nil
}

func ensureTempCollectionIndex(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().
			SetName("aggregate_tmp_expires_at_ttl").
			SetExpireAfterSeconds(0),
	})
	return err
}

func resolveMongoPipeline(in []map[string]any, template TemplateContext, aliases map[string]string) (mongo.Pipeline, error) {
	resolved := make(mongo.Pipeline, 0, len(in))
	for _, stage := range in {
		value, err := ResolveTemplateValue(stage, template)
		if err != nil {
			return nil, err
		}
		stageMap, ok := value.(map[string]any)
		if !ok {
			return nil, oops.In("aggregate_executor").New("resolved mongodb stage is not an object")
		}
		replaceLookupAliases(stageMap, aliases)
		resolved = append(resolved, bsonDocument(stageMap))
	}
	return resolved, nil
}

func replaceLookupAliases(stage map[string]any, aliases map[string]string) {
	lookup, ok := stage["$lookup"].(map[string]any)
	if !ok {
		return
	}
	from, ok := lookup["from"].(string)
	if !ok {
		return
	}
	if physical, exists := aliases[from]; exists {
		lookup["from"] = physical
	}
}

func bsonDocumentsToRows(docs []bson.M) ([]json.RawMessage, error) {
	rows := make([]json.RawMessage, 0, len(docs))
	for _, doc := range docs {
		cleanInternalFields(doc)
		data, err := json.Marshal(doc)
		if err != nil {
			return nil, oops.In("aggregate_executor").Wrapf(err, "encode aggregate row")
		}
		rows = append(rows, data)
	}
	return rows, nil
}

func cleanInternalFields(doc bson.M) {
	delete(doc, "_id")
	delete(doc, "_aggregate_run_id")
	delete(doc, "_aggregate_stage")
	delete(doc, "expires_at")
}

func bsonDocument(value any) bson.D {
	switch typed := value.(type) {
	case nil:
		return bson.D{}
	case bson.D:
		return typed
	case bson.M:
		out := make(bson.D, 0, len(typed))
		for key, value := range typed {
			out = append(out, bson.E{Key: key, Value: bsonValue(value)})
		}
		return out
	case map[string]any:
		out := make(bson.D, 0, len(typed))
		for key, value := range typed {
			out = append(out, bson.E{Key: key, Value: bsonValue(value)})
		}
		return out
	default:
		return bson.D{}
	}
}

func bsonValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return bsonDocument(typed)
	case []any:
		out := make(bson.A, 0, len(typed))
		for _, item := range typed {
			out = append(out, bsonValue(item))
		}
		return out
	default:
		return typed
	}
}

func mapStringValues(value any) map[string]string {
	out := map[string]string{}
	object, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, child := range object {
		out[key] = toString(child)
	}
	return out
}

var tempCollectionUnsafe = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func tempCollectionName(runID string, stage string) string {
	runID = tempCollectionUnsafe.ReplaceAllString(runID, "_")
	stage = tempCollectionUnsafe.ReplaceAllString(stage, "_")
	return strings.Trim(strings.Join([]string{"aggregate_tmp", runID, stage}, "_"), "_")
}
