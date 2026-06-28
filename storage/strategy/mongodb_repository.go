package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	strategyservice "github.com/awuzag/mwosa/service/strategy"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	strategies *mongo.Collection
	runs       *mongo.Collection
	items      *mongo.Collection
}

type strategyMongoDocument struct {
	ID              string                         `bson:"_id"`
	SchemaVersion   string                         `bson:"schema_version"`
	Revision        int64                          `bson:"revision"`
	CreatedAt       time.Time                      `bson:"created_at"`
	UpdatedAt       time.Time                      `bson:"updated_at"`
	ArchivedAt      *time.Time                     `bson:"archived_at,omitempty"`
	StrategyID      string                         `bson:"strategy_id"`
	Name            string                         `bson:"name"`
	Engine          string                         `bson:"engine"`
	ActiveVersionID string                         `bson:"active_version_id"`
	Versions        []strategyVersionMongoDocument `bson:"versions"`
}

type strategyVersionMongoDocument struct {
	ID                 string    `bson:"id"`
	StrategyID         string    `bson:"strategy_id"`
	Version            int       `bson:"version"`
	QueryText          string    `bson:"query_text"`
	QueryHash          string    `bson:"query_hash"`
	InputDataset       string    `bson:"input_dataset"`
	InputSchemaVersion int       `bson:"input_schema_version"`
	Params             any       `bson:"params"`
	Spec               any       `bson:"spec"`
	SpecHash           string    `bson:"spec_hash"`
	CreatedAt          time.Time `bson:"created_at"`
	Note               string    `bson:"note,omitempty"`
}

type screenRunMongoDocument struct {
	ID                 string                       `bson:"_id"`
	SchemaVersion      string                       `bson:"schema_version"`
	Revision           int64                        `bson:"revision"`
	CreatedAt          time.Time                    `bson:"created_at"`
	UpdatedAt          time.Time                    `bson:"updated_at"`
	RunID              string                       `bson:"run_id"`
	Alias              string                       `bson:"alias,omitempty"`
	StrategyID         string                       `bson:"strategy_id"`
	StrategyVersionID  string                       `bson:"strategy_version_id"`
	QueryHash          string                       `bson:"query_hash"`
	InputDataset       string                       `bson:"input_dataset"`
	InputSchemaVersion int                          `bson:"input_schema_version"`
	Params             any                          `bson:"params"`
	DataFrom           string                       `bson:"data_from,omitempty"`
	DataTo             string                       `bson:"data_to,omitempty"`
	DataAsOf           string                       `bson:"data_as_of,omitempty"`
	StartedAt          time.Time                    `bson:"started_at"`
	FinishedAt         *time.Time                   `bson:"finished_at,omitempty"`
	Status             string                       `bson:"status"`
	ResultCount        int                          `bson:"result_count"`
	ResultHash         string                       `bson:"result_hash,omitempty"`
	ResultSizeBytes    int64                        `bson:"result_size_bytes"`
	Summary            any                          `bson:"summary"`
	ErrorMessage       string                       `bson:"error_message,omitempty"`
	StrategySnapshot   strategySnapshotDocument     `bson:"strategy_snapshot"`
	VersionSnapshot    strategyVersionMongoDocument `bson:"version_snapshot"`
}

type strategySnapshotDocument struct {
	StrategyID      string     `bson:"strategy_id"`
	Name            string     `bson:"name"`
	Engine          string     `bson:"engine"`
	ActiveVersionID string     `bson:"active_version_id"`
	CreatedAt       time.Time  `bson:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at"`
	ArchivedAt      *time.Time `bson:"archived_at,omitempty"`
}

type screenRunItemMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
	ItemID        string    `bson:"item_id"`
	RunID         string    `bson:"run_id"`
	Ordinal       int       `bson:"ordinal"`
	Symbol        string    `bson:"symbol,omitempty"`
	Payload       any       `bson:"payload"`
}

var _ strategyservice.Repository = (*mongoRepository)(nil)

func NewMongoRepository(database *mongo.Database) (strategyservice.Repository, error) {
	if database == nil {
		return nil, oops.In("strategy_repository").New("mongodb database is nil")
	}
	return &mongoRepository{
		strategies: database.Collection("screen_strategies"),
		runs:       database.Collection("screen_runs"),
		items:      database.Collection("screen_run_items"),
	}, nil
}

func (r *mongoRepository) CreateStrategyWithVersion(ctx context.Context, in strategyservice.Strategy, version strategyservice.StrategyVersion) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	document, err := strategyToMongoDocument(in, version)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}
	if _, err := r.strategies.InsertOne(ctx, document); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "create strategy mongodb document")
	}
	return strategyservice.StrategyDetail{
		Strategy:      mongoDocumentToStrategy(document),
		ActiveVersion: mongoDocumentToStrategyVersion(document.Versions[0]),
	}, nil
}

func (r *mongoRepository) ListStrategies(ctx context.Context) ([]strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb")
	cursor, err := r.strategies.Find(ctx, bson.D{{Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "list strategy mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []strategyMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode strategy mongodb documents")
	}
	details := make([]strategyservice.StrategyDetail, 0, len(documents))
	for _, document := range documents {
		version, err := activeMongoVersion(document)
		if err != nil {
			return nil, errb.With("strategy_id", document.StrategyID).Wrap(err)
		}
		details = append(details, strategyservice.StrategyDetail{
			Strategy:      mongoDocumentToStrategy(document),
			ActiveVersion: mongoDocumentToStrategyVersion(version),
		})
	}
	return details, nil
}

func (r *mongoRepository) GetStrategy(ctx context.Context, name string) (strategyservice.StrategyDetail, error) {
	document, err := r.getStrategyDocument(ctx, name)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	version, err := activeMongoVersion(document)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	return strategyservice.StrategyDetail{
		Strategy:      mongoDocumentToStrategy(document),
		ActiveVersion: mongoDocumentToStrategyVersion(version),
	}, nil
}

func (r *mongoRepository) GetStrategyVersion(ctx context.Context, name string, ref strategyservice.StrategyVersionRef) (strategyservice.StrategyDetail, error) {
	document, err := r.getStrategyDocument(ctx, name)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	version, err := activeMongoVersion(document)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	if strings.TrimSpace(ref.Version) != "" && strings.TrimSpace(ref.Version) != "latest" {
		versionNumber, parseErr := strconv.Atoi(ref.Version)
		if parseErr != nil {
			return strategyservice.StrategyDetail{}, oops.In("strategy_repository").With("name", name, "version", ref.Version).Wrapf(parseErr, "parse strategy version")
		}
		version, err = mongoVersionByNumber(document, versionNumber)
		if err != nil {
			return strategyservice.StrategyDetail{}, err
		}
	}
	if strings.TrimSpace(ref.SpecHash) != "" {
		version, err = mongoVersionBySpecHash(document, ref.SpecHash)
		if err != nil {
			return strategyservice.StrategyDetail{}, err
		}
	}
	return strategyservice.StrategyDetail{
		Strategy:      mongoDocumentToStrategy(document),
		ActiveVersion: mongoDocumentToStrategyVersion(version),
	}, nil
}

func (r *mongoRepository) AddStrategyVersion(ctx context.Context, name string, engine strategyservice.Engine, version strategyservice.StrategyVersion, now time.Time) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "name", name, "engine", engine, "version_id", version.ID)
	current, err := r.getStrategyDocument(ctx, name)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	version.StrategyID = current.StrategyID
	nextVersion, err := strategyVersionToMongoDocument(version)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}
	current.Engine = string(engine)
	current.ActiveVersionID = version.ID
	current.UpdatedAt = now
	current.Revision++
	current.Versions = append(current.Versions, nextVersion)
	result, err := r.strategies.ReplaceOne(ctx, bson.D{{Key: "_id", Value: current.ID}, {Key: "revision", Value: current.Revision - 1}}, current)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "add strategy version mongodb document")
	}
	if result.MatchedCount == 0 {
		return strategyservice.StrategyDetail{}, storagemongodb.NewRevisionConflictError("screen_strategies", current.ID, current.Revision-1)
	}
	return strategyservice.StrategyDetail{
		Strategy:      mongoDocumentToStrategy(current),
		ActiveVersion: mongoDocumentToStrategyVersion(nextVersion),
	}, nil
}

func (r *mongoRepository) ArchiveStrategy(ctx context.Context, name string, archivedAt time.Time) error {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "name", name)
	result, err := r.strategies.UpdateOne(ctx,
		bson.D{{Key: "name", Value: name}, {Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "archived_at", Value: archivedAt}, {Key: "updated_at", Value: archivedAt}}}, {Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}}},
	)
	if err != nil {
		return errb.Wrapf(err, "archive strategy mongodb document")
	}
	if result.MatchedCount == 0 {
		return errb.Errorf("strategy not found: %s", name)
	}
	return nil
}

func (r *mongoRepository) CreateScreenRun(ctx context.Context, run strategyservice.ScreenRun, items []strategyservice.ScreenRunItem) (strategyservice.ScreenRunDetail, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "screen_run_id", run.ID, "alias", run.Alias, "strategy_id", run.StrategyID)
	strategyDocument, err := r.getStrategyDocumentByID(ctx, run.StrategyID)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy")
	}
	versionDocument, err := mongoVersionByID(strategyDocument, run.StrategyVersionID)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy version")
	}
	runDocument, err := screenRunToMongoDocument(run, strategyDocument, versionDocument)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrap(err)
	}
	if _, err := r.runs.InsertOne(ctx, runDocument); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "create screen run mongodb document")
	}
	itemDocuments := make([]screenRunItemMongoDocument, 0, len(items))
	for _, item := range items {
		itemDocument, err := screenRunItemToMongoDocument(item)
		if err != nil {
			return strategyservice.ScreenRunDetail{}, errb.With("ordinal", item.Ordinal).Wrap(err)
		}
		if _, err := r.items.InsertOne(ctx, itemDocument); err != nil {
			return strategyservice.ScreenRunDetail{}, errb.With("ordinal", item.Ordinal).Wrapf(err, "create screen run item mongodb document")
		}
		itemDocuments = append(itemDocuments, itemDocument)
	}
	return screenRunDetailFromMongoDocuments(runDocument, strategyDocument, versionDocument, itemDocuments), nil
}

func (r *mongoRepository) ListScreenRuns(ctx context.Context, limit int) ([]strategyservice.ScreenRun, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "limit", limit)
	cursor, err := r.runs.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "started_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, errb.Wrapf(err, "list screen run mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []screenRunMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode screen run mongodb documents")
	}
	runs := make([]strategyservice.ScreenRun, 0, len(documents))
	for _, document := range documents {
		runs = append(runs, mongoDocumentToScreenRun(document))
	}
	return runs, nil
}

func (r *mongoRepository) GetScreenRun(ctx context.Context, ref string) (strategyservice.ScreenRunDetail, error) {
	errb := oops.In("strategy_repository").With("backend", "mongodb", "ref", ref)
	var runDocument screenRunMongoDocument
	err := r.runs.FindOne(ctx, bson.D{{Key: "$or", Value: bson.A{bson.D{{Key: "run_id", Value: ref}}, bson.D{{Key: "alias", Value: ref}}}}}).Decode(&runDocument)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return strategyservice.ScreenRunDetail{}, errb.Errorf("screen run not found: %s", ref)
	}
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "get screen run mongodb document")
	}
	strategyDocument, err := r.getStrategyDocumentByID(ctx, runDocument.StrategyID)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy")
	}
	versionDocument, err := mongoVersionByID(strategyDocument, runDocument.StrategyVersionID)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy version")
	}
	itemDocuments, err := r.listScreenRunItemDocuments(ctx, runDocument.RunID)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrap(err)
	}
	return screenRunDetailFromMongoDocuments(runDocument, strategyDocument, versionDocument, itemDocuments), nil
}

func (r *mongoRepository) getStrategyDocument(ctx context.Context, name string) (strategyMongoDocument, error) {
	var document strategyMongoDocument
	err := r.strategies.FindOne(ctx, bson.D{{Key: "name", Value: name}, {Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return strategyMongoDocument{}, oops.In("strategy_repository").With("backend", "mongodb", "name", name).Errorf("strategy not found: %s", name)
	}
	if err != nil {
		return strategyMongoDocument{}, oops.In("strategy_repository").With("backend", "mongodb", "name", name).Wrapf(err, "get strategy mongodb document")
	}
	return document, nil
}

func (r *mongoRepository) getStrategyDocumentByID(ctx context.Context, id string) (strategyMongoDocument, error) {
	var document strategyMongoDocument
	err := r.strategies.FindOne(ctx, bson.D{{Key: "strategy_id", Value: id}}).Decode(&document)
	if err != nil {
		return strategyMongoDocument{}, oops.In("strategy_repository").With("backend", "mongodb", "strategy_id", id).Wrapf(err, "get strategy mongodb document by id")
	}
	return document, nil
}

func (r *mongoRepository) listScreenRunItemDocuments(ctx context.Context, runID string) ([]screenRunItemMongoDocument, error) {
	cursor, err := r.items.Find(ctx, bson.D{{Key: "run_id", Value: runID}}, options.Find().SetSort(bson.D{{Key: "ordinal", Value: 1}}))
	if err != nil {
		return nil, oops.In("strategy_repository").With("backend", "mongodb", "screen_run_id", runID).Wrapf(err, "load screen run item mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []screenRunItemMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("strategy_repository").With("backend", "mongodb", "screen_run_id", runID).Wrapf(err, "decode screen run item mongodb documents")
	}
	return documents, nil
}

func strategyToMongoDocument(in strategyservice.Strategy, version strategyservice.StrategyVersion) (strategyMongoDocument, error) {
	versionDocument, err := strategyVersionToMongoDocument(version)
	if err != nil {
		return strategyMongoDocument{}, err
	}
	return strategyMongoDocument{
		ID:              strategyMongoID(in.ID),
		SchemaVersion:   storagemongodb.SchemaVersion1,
		Revision:        1,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		ArchivedAt:      in.ArchivedAt,
		StrategyID:      in.ID,
		Name:            in.Name,
		Engine:          string(in.Engine),
		ActiveVersionID: in.ActiveVersionID,
		Versions:        []strategyVersionMongoDocument{versionDocument},
	}, nil
}

func strategyVersionToMongoDocument(version strategyservice.StrategyVersion) (strategyVersionMongoDocument, error) {
	params, err := rawJSONToBSONValue(version.ParamsJSON)
	if err != nil {
		return strategyVersionMongoDocument{}, err
	}
	spec, err := rawJSONToBSONValue(version.SpecJSON)
	if err != nil {
		return strategyVersionMongoDocument{}, err
	}
	specHash := version.SpecHash
	if specHash == "" {
		specHash = version.QueryHash
	}
	return strategyVersionMongoDocument{
		ID:                 version.ID,
		StrategyID:         version.StrategyID,
		Version:            version.Version,
		QueryText:          version.QueryText,
		QueryHash:          version.QueryHash,
		InputDataset:       version.InputDataset,
		InputSchemaVersion: version.InputSchemaVersion,
		Params:             params,
		Spec:               spec,
		SpecHash:           specHash,
		CreatedAt:          version.CreatedAt,
		Note:               version.Note,
	}, nil
}

func screenRunToMongoDocument(run strategyservice.ScreenRun, strategy strategyMongoDocument, version strategyVersionMongoDocument) (screenRunMongoDocument, error) {
	params, err := rawJSONToBSONValue(run.ParamsJSON)
	if err != nil {
		return screenRunMongoDocument{}, err
	}
	summary, err := rawJSONToBSONValue(run.SummaryJSON)
	if err != nil {
		return screenRunMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	return screenRunMongoDocument{
		ID:                 screenRunMongoID(run.ID),
		SchemaVersion:      storagemongodb.SchemaVersion1,
		Revision:           1,
		CreatedAt:          now,
		UpdatedAt:          now,
		RunID:              run.ID,
		Alias:              strings.TrimSpace(run.Alias),
		StrategyID:         run.StrategyID,
		StrategyVersionID:  run.StrategyVersionID,
		QueryHash:          run.QueryHash,
		InputDataset:       run.InputDataset,
		InputSchemaVersion: run.InputSchemaVersion,
		Params:             params,
		DataFrom:           run.DataFrom,
		DataTo:             run.DataTo,
		DataAsOf:           run.DataAsOf,
		StartedAt:          run.StartedAt,
		FinishedAt:         run.FinishedAt,
		Status:             string(run.Status),
		ResultCount:        run.ResultCount,
		ResultHash:         run.ResultHash,
		ResultSizeBytes:    run.ResultSizeBytes,
		Summary:            summary,
		ErrorMessage:       run.ErrorMessage,
		StrategySnapshot:   strategySnapshotFromMongoDocument(strategy),
		VersionSnapshot:    version,
	}, nil
}

func screenRunItemToMongoDocument(item strategyservice.ScreenRunItem) (screenRunItemMongoDocument, error) {
	payload, err := rawJSONToBSONValue(item.PayloadJSON)
	if err != nil {
		return screenRunItemMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	return screenRunItemMongoDocument{
		ID:            screenRunItemMongoID(item.ScreenRunID, item.Ordinal),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		ItemID:        item.ID,
		RunID:         item.ScreenRunID,
		Ordinal:       item.Ordinal,
		Symbol:        item.Symbol,
		Payload:       payload,
	}, nil
}

func mongoDocumentToStrategy(document strategyMongoDocument) strategyservice.Strategy {
	return strategyservice.Strategy{
		ID:              document.StrategyID,
		Name:            document.Name,
		Engine:          strategyservice.Engine(document.Engine),
		ActiveVersionID: document.ActiveVersionID,
		CreatedAt:       document.CreatedAt,
		UpdatedAt:       document.UpdatedAt,
		ArchivedAt:      document.ArchivedAt,
	}
}

func mongoDocumentToStrategyVersion(document strategyVersionMongoDocument) strategyservice.StrategyVersion {
	return strategyservice.StrategyVersion{
		ID:                 document.ID,
		StrategyID:         document.StrategyID,
		Version:            document.Version,
		QueryText:          document.QueryText,
		QueryHash:          document.QueryHash,
		InputDataset:       document.InputDataset,
		InputSchemaVersion: document.InputSchemaVersion,
		ParamsJSON:         bsonValueToRawJSON(document.Params),
		SpecJSON:           bsonValueToRawJSON(document.Spec),
		SpecHash:           document.SpecHash,
		CreatedAt:          document.CreatedAt,
		Note:               document.Note,
	}
}

func mongoDocumentToScreenRun(document screenRunMongoDocument) strategyservice.ScreenRun {
	return strategyservice.ScreenRun{
		ID:                 document.RunID,
		Alias:              document.Alias,
		StrategyID:         document.StrategyID,
		StrategyVersionID:  document.StrategyVersionID,
		QueryHash:          document.QueryHash,
		InputDataset:       document.InputDataset,
		InputSchemaVersion: document.InputSchemaVersion,
		ParamsJSON:         bsonValueToRawJSON(document.Params),
		DataFrom:           document.DataFrom,
		DataTo:             document.DataTo,
		DataAsOf:           document.DataAsOf,
		StartedAt:          document.StartedAt,
		FinishedAt:         document.FinishedAt,
		Status:             strategyservice.ScreenRunStatus(document.Status),
		ResultCount:        document.ResultCount,
		ResultHash:         document.ResultHash,
		ResultSizeBytes:    document.ResultSizeBytes,
		SummaryJSON:        bsonValueToRawJSON(document.Summary),
		ErrorMessage:       document.ErrorMessage,
	}
}

func mongoDocumentToScreenRunItem(document screenRunItemMongoDocument) strategyservice.ScreenRunItem {
	return strategyservice.ScreenRunItem{
		ID:          document.ItemID,
		ScreenRunID: document.RunID,
		Ordinal:     document.Ordinal,
		Symbol:      document.Symbol,
		PayloadJSON: bsonValueToRawJSON(document.Payload),
	}
}

func screenRunDetailFromMongoDocuments(run screenRunMongoDocument, strategy strategyMongoDocument, version strategyVersionMongoDocument, items []screenRunItemMongoDocument) strategyservice.ScreenRunDetail {
	detail := strategyservice.ScreenRunDetail{
		Run:             mongoDocumentToScreenRun(run),
		Strategy:        mongoDocumentToStrategy(strategy),
		StrategyVersion: mongoDocumentToStrategyVersion(version),
		Items:           make([]strategyservice.ScreenRunItem, 0, len(items)),
	}
	for _, item := range items {
		detail.Items = append(detail.Items, mongoDocumentToScreenRunItem(item))
	}
	return detail
}

func strategySnapshotFromMongoDocument(document strategyMongoDocument) strategySnapshotDocument {
	return strategySnapshotDocument{
		StrategyID:      document.StrategyID,
		Name:            document.Name,
		Engine:          document.Engine,
		ActiveVersionID: document.ActiveVersionID,
		CreatedAt:       document.CreatedAt,
		UpdatedAt:       document.UpdatedAt,
		ArchivedAt:      document.ArchivedAt,
	}
}

func activeMongoVersion(document strategyMongoDocument) (strategyVersionMongoDocument, error) {
	return mongoVersionByID(document, document.ActiveVersionID)
}

func mongoVersionByID(document strategyMongoDocument, id string) (strategyVersionMongoDocument, error) {
	for _, version := range document.Versions {
		if version.ID == id {
			return version, nil
		}
	}
	return strategyVersionMongoDocument{}, oops.In("strategy_repository").With("strategy_id", document.StrategyID, "version_id", id).New("strategy version not found")
}

func mongoVersionByNumber(document strategyMongoDocument, number int) (strategyVersionMongoDocument, error) {
	for _, version := range document.Versions {
		if version.Version == number {
			return version, nil
		}
	}
	return strategyVersionMongoDocument{}, oops.In("strategy_repository").With("strategy_id", document.StrategyID, "version", number).Errorf("strategy version not found: %s", document.Name)
}

func mongoVersionBySpecHash(document strategyMongoDocument, specHash string) (strategyVersionMongoDocument, error) {
	for _, version := range document.Versions {
		if version.SpecHash == specHash {
			return version, nil
		}
	}
	return strategyVersionMongoDocument{}, oops.In("strategy_repository").With("strategy_id", document.StrategyID, "spec_hash", specHash).Errorf("strategy version not found: %s", document.Name)
}

func rawJSONToBSONValue(raw json.RawMessage) (any, error) {
	normalized := normalizeRawMessage(raw)
	var value any
	if err := json.Unmarshal(normalized, &value); err != nil {
		return nil, oops.In("strategy_repository").Wrapf(err, "decode json payload")
	}
	return value, nil
}

func bsonValueToRawJSON(value any) json.RawMessage {
	if value == nil {
		return json.RawMessage(`{}`)
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(bytes)
}

func strategyMongoID(strategyID string) string {
	return strings.Join([]string{"screen_strategies", strategyID}, ":")
}

func screenRunMongoID(runID string) string {
	return strings.Join([]string{"screen_runs", runID}, ":")
}

func screenRunItemMongoID(runID string, ordinal int) string {
	return strings.Join([]string{"screen_run_items", runID, strconv.Itoa(ordinal)}, ":")
}
