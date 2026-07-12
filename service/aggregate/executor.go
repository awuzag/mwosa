package aggregate

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	aggregateStageItemsCollection = "aggregate_stage_items"
	defaultWorkspaceTTL           = 24 * time.Hour
	defaultWorkspaceTimeout       = 5 * time.Minute
	defaultWorkspaceMaxRows       = 100000
	defaultWorkspaceMaxFanout     = 5000
)

type workspaceLimits struct {
	ttl       time.Duration
	timeout   time.Duration
	maxRows   int
	maxFanout int
}

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
	limits, err := resolveWorkspaceLimits(spec.Workspace)
	if err != nil {
		return ExecutionResult{}, errb.Wrap(err)
	}
	if summary, err := e.validateLocalSources(ctx, spec); err != nil {
		return ExecutionResult{Stages: []StageSummary{summary}}, errb.Wrap(err)
	}
	if err := ensureStageItemsIndexes(ctx, e.database.Collection(aggregateStageItemsCollection)); err != nil {
		return ExecutionResult{}, errb.Wrapf(err, "ensure aggregate stage item indexes")
	}
	executionCtx, cancel := context.WithTimeout(ctx, limits.timeout)
	defer cancel()
	aliases := map[string]string{}
	stageRows := map[string][]json.RawMessage{}
	summaries := make([]StageSummary, 0, len(spec.Pipeline))
	for _, stage := range spec.Pipeline {
		summary := StageSummary{Name: stage.Name, Type: string(stage.Type), Status: "succeeded", Collection: aggregateStageItemsCollection}
		if stage.Foreach != nil && len(stageRows[stage.Foreach.Stage]) > limits.maxFanout {
			err := errb.With("stage", stage.Name, "fanout", len(stageRows[stage.Foreach.Stage]), "max_fanout", limits.maxFanout).New("aggregate stage fan-out limit exceeded")
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, err
		}
		rows, err := e.executeStage(executionCtx, stage, TemplateContext{Params: params}, aliases, stageRows, runID)
		if err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, errb.With("stage", stage.Name, "type", stage.Type).Wrap(err)
		}
		if len(rows) > limits.maxRows {
			err := errb.With("stage", stage.Name, "rows", len(rows), "max_rows", limits.maxRows).New("aggregate stage row limit exceeded")
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, err
		}
		if err := e.materialize(executionCtx, rows, runID, stage.Name, limits.ttl); err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			summaries = append(summaries, summary)
			return ExecutionResult{Stages: summaries}, errb.With("stage", stage.Name).Wrap(err)
		}
		aliases[stage.Name] = stage.Name
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

func (e MongoExecutor) executeStage(ctx context.Context, stage StageSpec, template TemplateContext, aliases map[string]string, stageRows map[string][]json.RawMessage, runID string) ([]json.RawMessage, error) {
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
		return e.executeMongoAggregate(ctx, stage, template, aliases, runID)
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
	exists, err := e.collectionExists(ctx, stage.Collection)
	if err != nil {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "collection", stage.Collection).Wrapf(err, "check local collection")
	}
	if !exists {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "collection", stage.Collection).Errorf("local collection does not exist: %s", stage.Collection)
	}
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

func (e MongoExecutor) executeMongoAggregate(ctx context.Context, stage StageSpec, template TemplateContext, aliases map[string]string, runID string) ([]json.RawMessage, error) {
	sourceStage, ok := aliases[stage.From]
	if !ok {
		return nil, oops.In("aggregate_executor").With("stage", stage.Name, "from", stage.From).New("aggregate input stage was not produced")
	}
	pipeline, err := resolveMongoPipeline(stage.Pipeline, template, aliases, runID)
	if err != nil {
		return nil, err
	}
	pipeline = append(mongo.Pipeline{stageItemsMatch(runID, sourceStage)}, pipeline...)
	cursor, err := e.database.Collection(aggregateStageItemsCollection).Aggregate(ctx, pipeline)
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

func (e MongoExecutor) materialize(ctx context.Context, rows []json.RawMessage, runID string, stage string, ttl time.Duration) error {
	collection := e.database.Collection(aggregateStageItemsCollection)
	filter := bson.D{{Key: "_aggregate_run_id", Value: runID}, {Key: "_aggregate_stage", Value: stage}}
	if _, err := collection.DeleteMany(ctx, filter); err != nil {
		return oops.In("aggregate_executor").With("stage", stage, "collection", aggregateStageItemsCollection).Wrapf(err, "clear aggregate stage items")
	}
	if len(rows) == 0 {
		return nil
	}
	docs := make([]any, 0, len(rows))
	now := e.now()
	expiresAt := now.Add(ttl)
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
		doc["_id"] = strings.Join([]string{"aggregate_stage_items", runID, stage, strconv.Itoa(index)}, ":")
		doc["schema_version"] = "1.0.0"
		doc["revision"] = int64(1)
		doc["created_at"] = now
		doc["updated_at"] = now
		doc["_aggregate_run_id"] = runID
		doc["_aggregate_stage"] = stage
		doc["_aggregate_ordinal"] = index
		doc["expires_at"] = expiresAt
		docs = append(docs, doc)
	}
	if _, err := collection.InsertMany(ctx, docs, options.InsertMany().SetOrdered(true)); err != nil {
		return oops.In("aggregate_executor").With("stage", stage, "collection", aggregateStageItemsCollection).Wrapf(err, "materialize aggregate stage")
	}
	return nil
}

func ensureStageItemsIndexes(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "_aggregate_run_id", Value: 1}, {Key: "_aggregate_stage", Value: 1}, {Key: "_aggregate_ordinal", Value: 1}},
			Options: options.Index().
				SetName("aggregate_stage_items_run_stage_ordinal_unique").
				SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().
				SetName("aggregate_stage_items_expires_at_ttl").
				SetExpireAfterSeconds(0),
		},
	})
	return err
}

func resolveMongoPipeline(in []map[string]any, template TemplateContext, aliases map[string]string, runID string) (mongo.Pipeline, error) {
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
		if err := replaceLookupAliases(stageMap, aliases, runID); err != nil {
			return nil, err
		}
		resolved = append(resolved, bsonDocument(stageMap))
	}
	return resolved, nil
}

func replaceLookupAliases(stage map[string]any, aliases map[string]string, runID string) error {
	lookup, ok := stage["$lookup"].(map[string]any)
	if !ok {
		return nil
	}
	from, ok := lookup["from"].(string)
	if !ok {
		return nil
	}
	if sourceStage, exists := aliases[from]; exists {
		lookup["from"] = aggregateStageItemsCollection
		match := map[string]any{"$match": map[string]any{
			"_aggregate_run_id": runID,
			"_aggregate_stage":  sourceStage,
		}}
		pipeline, err := lookupPipeline(lookup["pipeline"])
		if err != nil {
			return oops.In("aggregate_executor").With("lookup_from", from).Wrap(err)
		}
		lookup["pipeline"] = append([]any{match}, pipeline...)
	}
	return nil
}

func lookupPipeline(value any) ([]any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []any:
		return typed, nil
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, stage := range typed {
			out = append(out, stage)
		}
		return out, nil
	default:
		return nil, oops.In("aggregate_executor").New("mongodb lookup pipeline must be an array")
	}
}

func stageItemsMatch(runID string, stage string) bson.D {
	return bson.D{{Key: "$match", Value: bson.D{
		{Key: "_aggregate_run_id", Value: runID},
		{Key: "_aggregate_stage", Value: stage},
	}}}
}

func workspaceTTL(workspace WorkspaceSpec) (time.Duration, error) {
	limits, err := resolveWorkspaceLimits(workspace)
	return limits.ttl, err
}

func resolveWorkspaceLimits(workspace WorkspaceSpec) (workspaceLimits, error) {
	limits := workspaceLimits{
		ttl:       defaultWorkspaceTTL,
		timeout:   defaultWorkspaceTimeout,
		maxRows:   defaultWorkspaceMaxRows,
		maxFanout: defaultWorkspaceMaxFanout,
	}
	if value := strings.TrimSpace(workspace.TTL); value != "" {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return workspaceLimits{}, oops.In("aggregate_workspace").With("ttl", value).Wrapf(err, "parse aggregate workspace ttl")
		}
		if ttl <= 0 {
			return workspaceLimits{}, oops.In("aggregate_workspace").With("ttl", value).New("aggregate workspace ttl must be positive")
		}
		limits.ttl = ttl
	}
	if value := strings.TrimSpace(workspace.Timeout); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return workspaceLimits{}, oops.In("aggregate_workspace").With("timeout", value).Wrapf(err, "parse aggregate workspace timeout")
		}
		if timeout <= 0 {
			return workspaceLimits{}, oops.In("aggregate_workspace").With("timeout", value).New("aggregate workspace timeout must be positive")
		}
		limits.timeout = timeout
	}
	if workspace.MaxRows < 0 {
		return workspaceLimits{}, oops.In("aggregate_workspace").With("max_rows", workspace.MaxRows).New("aggregate workspace max_rows must be positive")
	}
	if workspace.MaxRows > 0 {
		limits.maxRows = workspace.MaxRows
	}
	if workspace.MaxFanout < 0 {
		return workspaceLimits{}, oops.In("aggregate_workspace").With("max_fanout", workspace.MaxFanout).New("aggregate workspace max_fanout must be positive")
	}
	if workspace.MaxFanout > 0 {
		limits.maxFanout = workspace.MaxFanout
	}
	return limits, nil
}

func (e MongoExecutor) collectionExists(ctx context.Context, name string) (bool, error) {
	names, err := e.database.ListCollectionNames(ctx, bson.D{{Key: "name", Value: strings.TrimSpace(name)}})
	if err != nil {
		return false, err
	}
	return len(names) > 0, nil
}

func (e MongoExecutor) validateLocalSources(ctx context.Context, spec Spec) (StageSummary, error) {
	for _, stage := range spec.Pipeline {
		var collection string
		switch stage.Type {
		case StageLocalCollection:
			collection = strings.TrimSpace(stage.Collection)
		case StageLocalDataset:
			collection = strings.TrimSpace(stage.Dataset)
		default:
			continue
		}
		summary := StageSummary{Name: stage.Name, Type: string(stage.Type), Status: "failed", Collection: aggregateStageItemsCollection}
		exists, err := e.collectionExists(ctx, collection)
		if err != nil {
			sourceErr := oops.In("aggregate_executor").With("stage", stage.Name, "collection", collection).Wrapf(err, "check aggregate local source")
			summary.Error = sourceErr.Error()
			return summary, sourceErr
		}
		if !exists {
			sourceErr := oops.In("aggregate_executor").With("stage", stage.Name, "collection", collection).Errorf("local collection does not exist: %s", collection)
			summary.Error = sourceErr.Error()
			return summary, sourceErr
		}
	}
	return StageSummary{}, nil
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
	delete(doc, "schema_version")
	delete(doc, "revision")
	delete(doc, "created_at")
	delete(doc, "updated_at")
	delete(doc, "_aggregate_run_id")
	delete(doc, "_aggregate_stage")
	delete(doc, "_aggregate_ordinal")
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
