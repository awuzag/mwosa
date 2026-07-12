package aggregate

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	aggregates *mongo.Collection
	versions   *mongo.Collection
	runs       *mongo.Collection
	items      *mongo.Collection
}

type aggregateMongoDocument struct {
	ID              string     `bson:"_id"`
	SchemaVersion   string     `bson:"schema_version"`
	Revision        int64      `bson:"revision"`
	CreatedAt       time.Time  `bson:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at"`
	ArchivedAt      *time.Time `bson:"archived_at,omitempty"`
	AggregateID     string     `bson:"aggregate_id"`
	Name            string     `bson:"name"`
	ActiveVersionID string     `bson:"active_version_id"`
}

type versionMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
	VersionID     string    `bson:"version_id"`
	AggregateID   string    `bson:"aggregate_id"`
	Version       int       `bson:"version"`
	YAMLText      string    `bson:"yaml_text"`
	Spec          any       `bson:"spec"`
	SpecHash      string    `bson:"spec_hash"`
	Note          string    `bson:"note,omitempty"`
}

type runMongoDocument struct {
	ID                 string                    `bson:"_id"`
	SchemaVersion      string                    `bson:"schema_version"`
	Revision           int64                     `bson:"revision"`
	CreatedAt          time.Time                 `bson:"created_at"`
	UpdatedAt          time.Time                 `bson:"updated_at"`
	RunID              string                    `bson:"run_id"`
	Alias              string                    `bson:"alias,omitempty"`
	AggregateID        string                    `bson:"aggregate_id"`
	AggregateVersionID string                    `bson:"aggregate_version_id"`
	AggregateName      string                    `bson:"aggregate_name"`
	Version            int                       `bson:"version"`
	SpecHash           string                    `bson:"spec_hash"`
	Params             any                       `bson:"params"`
	Stages             any                       `bson:"stages"`
	Pipeline           any                       `bson:"pipeline"`
	StartedAt          time.Time                 `bson:"started_at"`
	FinishedAt         *time.Time                `bson:"finished_at,omitempty"`
	Status             string                    `bson:"status"`
	ResultCount        int                       `bson:"result_count"`
	ResultHash         string                    `bson:"result_hash,omitempty"`
	ResultSizeBytes    int64                     `bson:"result_size_bytes"`
	Summary            any                       `bson:"summary"`
	ErrorMessage       string                    `bson:"error_message,omitempty"`
	AggregateSnapshot  aggregateSnapshotDocument `bson:"aggregate_snapshot"`
	VersionSnapshot    versionSnapshotDocument   `bson:"version_snapshot"`
}

type aggregateSnapshotDocument struct {
	AggregateID     string     `bson:"aggregate_id"`
	Name            string     `bson:"name"`
	ActiveVersionID string     `bson:"active_version_id"`
	CreatedAt       time.Time  `bson:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at"`
	ArchivedAt      *time.Time `bson:"archived_at,omitempty"`
}

type versionSnapshotDocument struct {
	VersionID   string    `bson:"version_id"`
	AggregateID string    `bson:"aggregate_id"`
	Version     int       `bson:"version"`
	SpecHash    string    `bson:"spec_hash"`
	CreatedAt   time.Time `bson:"created_at"`
}

type runItemMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
	ItemID        string    `bson:"item_id"`
	RunID         string    `bson:"run_id"`
	Ordinal       int       `bson:"ordinal"`
	Payload       any       `bson:"payload"`
}

var _ aggregateservice.Repository = (*mongoRepository)(nil)

func NewMongoRepository(database *mongo.Database) (aggregateservice.Repository, error) {
	if database == nil {
		return nil, oops.In("aggregate_repository").New("mongodb database is nil")
	}
	return &mongoRepository{
		aggregates: database.Collection("aggregates"),
		versions:   database.Collection("aggregate_versions"),
		runs:       database.Collection("aggregate_runs"),
		items:      database.Collection("aggregate_run_items"),
	}, nil
}

func (r *mongoRepository) CreateAggregateWithVersion(ctx context.Context, aggregate aggregateservice.Aggregate, version aggregateservice.Version) (aggregateservice.Detail, error) {
	errb := oops.In("aggregate_repository").With("backend", "mongodb", "name", aggregate.Name, "aggregate_id", aggregate.ID, "version_id", version.ID)
	aggregateDocument := aggregateToMongoDocument(aggregate)
	version.AggregateID = aggregate.ID
	versionDocument, err := versionToMongoDocument(version)
	if err != nil {
		return aggregateservice.Detail{}, errb.Wrap(err)
	}
	if _, err := r.aggregates.InsertOne(ctx, aggregateDocument); err != nil {
		return aggregateservice.Detail{}, errb.Wrapf(err, "create aggregate mongodb document")
	}
	if _, err := r.versions.InsertOne(ctx, versionDocument); err != nil {
		return aggregateservice.Detail{}, errb.Wrapf(err, "create aggregate version mongodb document")
	}
	return aggregateservice.Detail{Aggregate: mongoDocumentToAggregate(aggregateDocument), ActiveVersion: mongoDocumentToVersion(versionDocument)}, nil
}

func (r *mongoRepository) ListAggregates(ctx context.Context) ([]aggregateservice.Detail, error) {
	errb := oops.In("aggregate_repository").With("backend", "mongodb")
	cursor, err := r.aggregates.Find(ctx, bson.D{{Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "list aggregate mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []aggregateMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode aggregate mongodb documents")
	}
	out := make([]aggregateservice.Detail, 0, len(documents))
	for _, document := range documents {
		version, err := r.getVersionDocumentByID(ctx, document.ActiveVersionID)
		if err != nil {
			return nil, errb.With("aggregate_id", document.AggregateID).Wrap(err)
		}
		out = append(out, aggregateservice.Detail{
			Aggregate:     mongoDocumentToAggregate(document),
			ActiveVersion: mongoDocumentToVersion(version),
		})
	}
	return out, nil
}

func (r *mongoRepository) GetAggregate(ctx context.Context, name string) (aggregateservice.Detail, error) {
	document, err := r.getAggregateDocument(ctx, name)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	version, err := r.getVersionDocumentByID(ctx, document.ActiveVersionID)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	versions, err := r.listVersionDocuments(ctx, document.AggregateID)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	return aggregateservice.Detail{
		Aggregate:     mongoDocumentToAggregate(document),
		ActiveVersion: mongoDocumentToVersion(version),
		Versions:      mongoDocumentsToVersions(versions),
	}, nil
}

func (r *mongoRepository) GetAggregateVersion(ctx context.Context, name string, ref aggregateservice.VersionRef) (aggregateservice.Detail, error) {
	aggregateDocument, err := r.getAggregateDocument(ctx, name)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	versionDocument, err := r.getVersionDocumentByID(ctx, aggregateDocument.ActiveVersionID)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	if strings.TrimSpace(ref.Version) != "" && strings.TrimSpace(ref.Version) != "latest" {
		versionNumber, parseErr := strconv.Atoi(ref.Version)
		if parseErr != nil {
			return aggregateservice.Detail{}, oops.In("aggregate_repository").With("name", name, "version", ref.Version).Wrapf(parseErr, "parse aggregate version")
		}
		versionDocument, err = r.getVersionDocumentByNumber(ctx, aggregateDocument.AggregateID, versionNumber)
		if err != nil {
			return aggregateservice.Detail{}, err
		}
	}
	if strings.TrimSpace(ref.SpecHash) != "" {
		versionDocument, err = r.getVersionDocumentBySpecHash(ctx, aggregateDocument.AggregateID, ref.SpecHash)
		if err != nil {
			return aggregateservice.Detail{}, err
		}
	}
	versions, err := r.listVersionDocuments(ctx, aggregateDocument.AggregateID)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	return aggregateservice.Detail{
		Aggregate:     mongoDocumentToAggregate(aggregateDocument),
		ActiveVersion: mongoDocumentToVersion(versionDocument),
		Versions:      mongoDocumentsToVersions(versions),
	}, nil
}

func (r *mongoRepository) ListAggregateVersions(ctx context.Context, name string) ([]aggregateservice.Version, error) {
	aggregateDocument, err := r.getAggregateDocument(ctx, name)
	if err != nil {
		return nil, err
	}
	documents, err := r.listVersionDocuments(ctx, aggregateDocument.AggregateID)
	if err != nil {
		return nil, err
	}
	return mongoDocumentsToVersions(documents), nil
}

func (r *mongoRepository) AddAggregateVersion(ctx context.Context, name string, version aggregateservice.Version, now time.Time) (aggregateservice.Detail, error) {
	errb := oops.In("aggregate_repository").With("backend", "mongodb", "name", name, "version_id", version.ID)
	current, err := r.getAggregateDocument(ctx, name)
	if err != nil {
		return aggregateservice.Detail{}, err
	}
	version.AggregateID = current.AggregateID
	versionDocument, err := versionToMongoDocument(version)
	if err != nil {
		return aggregateservice.Detail{}, errb.Wrap(err)
	}
	if _, err := r.versions.InsertOne(ctx, versionDocument); err != nil {
		return aggregateservice.Detail{}, errb.Wrapf(err, "create aggregate version mongodb document")
	}
	result, err := r.aggregates.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: current.ID}, {Key: "revision", Value: current.Revision}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "active_version_id", Value: version.ID}, {Key: "updated_at", Value: now}}}, {Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}}},
	)
	if err != nil {
		return aggregateservice.Detail{}, errb.Wrapf(err, "update aggregate active version")
	}
	if result.MatchedCount == 0 {
		return aggregateservice.Detail{}, storagemongodb.NewRevisionConflictError("aggregates", current.ID, current.Revision)
	}
	current.ActiveVersionID = version.ID
	current.UpdatedAt = now
	current.Revision++
	return aggregateservice.Detail{Aggregate: mongoDocumentToAggregate(current), ActiveVersion: mongoDocumentToVersion(versionDocument)}, nil
}

func (r *mongoRepository) ArchiveAggregate(ctx context.Context, name string, archivedAt time.Time) error {
	errb := oops.In("aggregate_repository").With("backend", "mongodb", "name", name)
	result, err := r.aggregates.UpdateOne(ctx,
		bson.D{{Key: "name", Value: name}, {Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "archived_at", Value: archivedAt}, {Key: "updated_at", Value: archivedAt}}}, {Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}}},
	)
	if err != nil {
		return errb.Wrapf(err, "archive aggregate mongodb document")
	}
	if result.MatchedCount == 0 {
		return errb.Errorf("aggregate not found: %s", name)
	}
	return nil
}

func (r *mongoRepository) HasRunAlias(ctx context.Context, alias string) (bool, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return false, nil
	}
	count, err := r.runs.CountDocuments(ctx, bson.D{{Key: "alias", Value: alias}}, options.Count().SetLimit(1))
	if err != nil {
		return false, oops.In("aggregate_repository").With("backend", "mongodb", "alias", alias).Wrapf(err, "check aggregate run alias")
	}
	return count > 0, nil
}

func (r *mongoRepository) CreateRun(ctx context.Context, run aggregateservice.Run, items []aggregateservice.RunItem) (aggregateservice.RunDetail, error) {
	errb := oops.In("aggregate_repository").With("backend", "mongodb", "run_id", run.ID, "alias", run.Alias, "aggregate_id", run.AggregateID)
	aggregateDocument, err := r.getAggregateDocumentByID(ctx, run.AggregateID)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrapf(err, "load aggregate run aggregate")
	}
	versionDocument, err := r.getVersionDocumentByID(ctx, run.AggregateVersionID)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrapf(err, "load aggregate run version")
	}
	runDocument, err := runToMongoDocument(run, aggregateDocument, versionDocument)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrap(err)
	}
	itemDocuments := make([]runItemMongoDocument, 0, len(items))
	itemValues := make([]any, 0, len(items))
	for _, item := range items {
		itemDocument, err := runItemToMongoDocument(item)
		if err != nil {
			return aggregateservice.RunDetail{}, errb.With("ordinal", item.Ordinal).Wrap(err)
		}
		itemDocuments = append(itemDocuments, itemDocument)
		itemValues = append(itemValues, itemDocument)
	}
	if len(itemValues) > 0 {
		if _, err := r.items.InsertMany(ctx, itemValues, options.InsertMany().SetOrdered(true)); err != nil {
			cleanupErr := r.deleteRunItems(ctx, run.ID)
			return aggregateservice.RunDetail{}, oops.Join(
				errb.Wrapf(err, "create aggregate run item mongodb documents"),
				cleanupErr,
			)
		}
	}
	if _, err := r.runs.InsertOne(ctx, runDocument); err != nil {
		cleanupErr := r.deleteRunItems(ctx, run.ID)
		return aggregateservice.RunDetail{}, oops.Join(
			errb.Wrapf(err, "create aggregate run mongodb document"),
			cleanupErr,
		)
	}
	return runDetailFromMongoDocuments(runDocument, aggregateDocument, versionDocument, itemDocuments), nil
}

func (r *mongoRepository) deleteRunItems(ctx context.Context, runID string) error {
	if _, err := r.items.DeleteMany(ctx, bson.D{{Key: "run_id", Value: runID}}); err != nil {
		return oops.In("aggregate_repository").With("backend", "mongodb", "run_id", runID).Wrapf(err, "rollback aggregate run items")
	}
	return nil
}

func (r *mongoRepository) ListRuns(ctx context.Context, filter aggregateservice.RunHistoryFilter) ([]aggregateservice.Run, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query := bson.D{}
	if strings.TrimSpace(filter.Name) != "" {
		query = append(query, bson.E{Key: "aggregate_name", Value: strings.TrimSpace(filter.Name)})
	}
	if filter.Status != "" {
		query = append(query, bson.E{Key: "status", Value: string(filter.Status)})
	}
	cursor, err := r.runs.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "started_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb").Wrapf(err, "list aggregate run mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []runMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb").Wrapf(err, "decode aggregate run mongodb documents")
	}
	out := make([]aggregateservice.Run, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToRun(document))
	}
	return out, nil
}

func (r *mongoRepository) GetRun(ctx context.Context, ref string, limit int) (aggregateservice.RunDetail, error) {
	errb := oops.In("aggregate_repository").With("backend", "mongodb", "ref", ref)
	var runDocument runMongoDocument
	err := r.runs.FindOne(ctx, bson.D{{Key: "$or", Value: bson.A{bson.D{{Key: "run_id", Value: ref}}, bson.D{{Key: "alias", Value: ref}}}}}).Decode(&runDocument)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return aggregateservice.RunDetail{}, errb.Errorf("aggregate run not found: %s", ref)
	}
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrapf(err, "get aggregate run mongodb document")
	}
	aggregateDocument, err := r.getAggregateDocumentByID(ctx, runDocument.AggregateID)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrapf(err, "load aggregate run aggregate")
	}
	versionDocument, err := r.getVersionDocumentByID(ctx, runDocument.AggregateVersionID)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrapf(err, "load aggregate run version")
	}
	itemDocuments, err := r.listRunItemDocuments(ctx, runDocument.RunID, limit)
	if err != nil {
		return aggregateservice.RunDetail{}, errb.Wrap(err)
	}
	return runDetailFromMongoDocuments(runDocument, aggregateDocument, versionDocument, itemDocuments), nil
}

func (r *mongoRepository) getAggregateDocument(ctx context.Context, name string) (aggregateMongoDocument, error) {
	var document aggregateMongoDocument
	err := r.aggregates.FindOne(ctx, bson.D{{Key: "name", Value: name}, {Key: "archived_at", Value: bson.D{{Key: "$exists", Value: false}}}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return aggregateMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "name", name).Errorf("aggregate not found: %s", name)
	}
	if err != nil {
		return aggregateMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "name", name).Wrapf(err, "get aggregate mongodb document")
	}
	return document, nil
}

func (r *mongoRepository) getAggregateDocumentByID(ctx context.Context, id string) (aggregateMongoDocument, error) {
	var document aggregateMongoDocument
	err := r.aggregates.FindOne(ctx, bson.D{{Key: "aggregate_id", Value: id}}).Decode(&document)
	if err != nil {
		return aggregateMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "aggregate_id", id).Wrapf(err, "get aggregate mongodb document by id")
	}
	return document, nil
}

func (r *mongoRepository) getVersionDocumentByID(ctx context.Context, id string) (versionMongoDocument, error) {
	var document versionMongoDocument
	err := r.versions.FindOne(ctx, bson.D{{Key: "version_id", Value: id}}).Decode(&document)
	if err != nil {
		return versionMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "version_id", id).Wrapf(err, "get aggregate version mongodb document")
	}
	return document, nil
}

func (r *mongoRepository) getVersionDocumentByNumber(ctx context.Context, aggregateID string, version int) (versionMongoDocument, error) {
	var document versionMongoDocument
	err := r.versions.FindOne(ctx, bson.D{{Key: "aggregate_id", Value: aggregateID}, {Key: "version", Value: version}}).Decode(&document)
	if err != nil {
		return versionMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "aggregate_id", aggregateID, "version", version).Wrapf(err, "get aggregate version mongodb document by number")
	}
	return document, nil
}

func (r *mongoRepository) getVersionDocumentBySpecHash(ctx context.Context, aggregateID string, specHash string) (versionMongoDocument, error) {
	var document versionMongoDocument
	err := r.versions.FindOne(ctx, bson.D{{Key: "aggregate_id", Value: aggregateID}, {Key: "spec_hash", Value: specHash}}).Decode(&document)
	if err != nil {
		return versionMongoDocument{}, oops.In("aggregate_repository").With("backend", "mongodb", "aggregate_id", aggregateID, "spec_hash", specHash).Wrapf(err, "get aggregate version mongodb document by spec hash")
	}
	return document, nil
}

func (r *mongoRepository) listVersionDocuments(ctx context.Context, aggregateID string) ([]versionMongoDocument, error) {
	cursor, err := r.versions.Find(ctx, bson.D{{Key: "aggregate_id", Value: aggregateID}}, options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb", "aggregate_id", aggregateID).Wrapf(err, "list aggregate version mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []versionMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb", "aggregate_id", aggregateID).Wrapf(err, "decode aggregate version mongodb documents")
	}
	return documents, nil
}

func (r *mongoRepository) listRunItemDocuments(ctx context.Context, runID string, limit int) ([]runItemMongoDocument, error) {
	findOptions := options.Find().SetSort(bson.D{{Key: "ordinal", Value: 1}})
	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}
	cursor, err := r.items.Find(ctx, bson.D{{Key: "run_id", Value: runID}}, findOptions)
	if err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb", "run_id", runID).Wrapf(err, "load aggregate run item mongodb documents")
	}
	defer cursor.Close(ctx)
	var documents []runItemMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("aggregate_repository").With("backend", "mongodb", "run_id", runID).Wrapf(err, "decode aggregate run item mongodb documents")
	}
	return documents, nil
}

func aggregateToMongoDocument(in aggregateservice.Aggregate) aggregateMongoDocument {
	return aggregateMongoDocument{
		ID:              aggregateMongoID(in.ID),
		SchemaVersion:   storagemongodb.SchemaVersion1,
		Revision:        1,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		ArchivedAt:      in.ArchivedAt,
		AggregateID:     in.ID,
		Name:            in.Name,
		ActiveVersionID: in.ActiveVersionID,
	}
}

func versionToMongoDocument(version aggregateservice.Version) (versionMongoDocument, error) {
	spec, err := rawJSONToBSONValue(version.SpecJSON)
	if err != nil {
		return versionMongoDocument{}, err
	}
	return versionMongoDocument{
		ID:            aggregateVersionMongoID(version.ID),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     version.CreatedAt,
		UpdatedAt:     version.CreatedAt,
		VersionID:     version.ID,
		AggregateID:   version.AggregateID,
		Version:       version.Version,
		YAMLText:      version.YAMLText,
		Spec:          spec,
		SpecHash:      version.SpecHash,
		Note:          version.Note,
	}, nil
}

func runToMongoDocument(run aggregateservice.Run, aggregate aggregateMongoDocument, version versionMongoDocument) (runMongoDocument, error) {
	params, err := rawJSONToBSONValue(run.ParamsJSON)
	if err != nil {
		return runMongoDocument{}, err
	}
	stages, err := rawJSONToBSONValue(run.StagesJSON)
	if err != nil {
		return runMongoDocument{}, err
	}
	pipeline, err := rawJSONToBSONValue(run.PipelineJSON)
	if err != nil {
		return runMongoDocument{}, err
	}
	summary, err := rawJSONToBSONValue(run.SummaryJSON)
	if err != nil {
		return runMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	return runMongoDocument{
		ID:                 aggregateRunMongoID(run.ID),
		SchemaVersion:      storagemongodb.SchemaVersion1,
		Revision:           1,
		CreatedAt:          now,
		UpdatedAt:          now,
		RunID:              run.ID,
		Alias:              strings.TrimSpace(run.Alias),
		AggregateID:        run.AggregateID,
		AggregateVersionID: run.AggregateVersionID,
		AggregateName:      run.AggregateName,
		Version:            run.Version,
		SpecHash:           run.SpecHash,
		Params:             params,
		Stages:             stages,
		Pipeline:           pipeline,
		StartedAt:          run.StartedAt,
		FinishedAt:         run.FinishedAt,
		Status:             string(run.Status),
		ResultCount:        run.ResultCount,
		ResultHash:         run.ResultHash,
		ResultSizeBytes:    run.ResultSizeBytes,
		Summary:            summary,
		ErrorMessage:       run.ErrorMessage,
		AggregateSnapshot:  aggregateSnapshotFromMongoDocument(aggregate),
		VersionSnapshot:    versionSnapshotFromMongoDocument(version),
	}, nil
}

func runItemToMongoDocument(item aggregateservice.RunItem) (runItemMongoDocument, error) {
	payload, err := rawJSONToBSONValue(item.PayloadJSON)
	if err != nil {
		return runItemMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	return runItemMongoDocument{
		ID:            aggregateRunItemMongoID(item.RunID, item.Ordinal),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		ItemID:        item.ID,
		RunID:         item.RunID,
		Ordinal:       item.Ordinal,
		Payload:       payload,
	}, nil
}

func mongoDocumentToAggregate(document aggregateMongoDocument) aggregateservice.Aggregate {
	return aggregateservice.Aggregate{
		ID:              document.AggregateID,
		Name:            document.Name,
		ActiveVersionID: document.ActiveVersionID,
		CreatedAt:       document.CreatedAt,
		UpdatedAt:       document.UpdatedAt,
		ArchivedAt:      document.ArchivedAt,
	}
}

func mongoDocumentToVersion(document versionMongoDocument) aggregateservice.Version {
	return aggregateservice.Version{
		ID:          document.VersionID,
		AggregateID: document.AggregateID,
		Version:     document.Version,
		YAMLText:    document.YAMLText,
		SpecJSON:    bsonValueToRawJSON(document.Spec),
		SpecHash:    document.SpecHash,
		CreatedAt:   document.CreatedAt,
		Note:        document.Note,
	}
}

func mongoDocumentsToVersions(documents []versionMongoDocument) []aggregateservice.Version {
	out := make([]aggregateservice.Version, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToVersion(document))
	}
	return out
}

func mongoDocumentToRun(document runMongoDocument) aggregateservice.Run {
	return aggregateservice.Run{
		ID:                 document.RunID,
		Alias:              document.Alias,
		AggregateID:        document.AggregateID,
		AggregateVersionID: document.AggregateVersionID,
		AggregateName:      document.AggregateName,
		Version:            document.Version,
		SpecHash:           document.SpecHash,
		ParamsJSON:         bsonValueToRawJSON(document.Params),
		StagesJSON:         bsonValueToRawJSON(document.Stages),
		PipelineJSON:       bsonValueToRawJSON(document.Pipeline),
		StartedAt:          document.StartedAt,
		FinishedAt:         document.FinishedAt,
		Status:             aggregateservice.RunStatus(document.Status),
		ResultCount:        document.ResultCount,
		ResultHash:         document.ResultHash,
		ResultSizeBytes:    document.ResultSizeBytes,
		SummaryJSON:        bsonValueToRawJSON(document.Summary),
		ErrorMessage:       document.ErrorMessage,
	}
}

func mongoDocumentToRunItem(document runItemMongoDocument) aggregateservice.RunItem {
	return aggregateservice.RunItem{
		ID:          document.ItemID,
		RunID:       document.RunID,
		Ordinal:     document.Ordinal,
		PayloadJSON: bsonValueToRawJSON(document.Payload),
	}
}

func runDetailFromMongoDocuments(run runMongoDocument, aggregate aggregateMongoDocument, version versionMongoDocument, items []runItemMongoDocument) aggregateservice.RunDetail {
	detail := aggregateservice.RunDetail{
		Run:       mongoDocumentToRun(run),
		Aggregate: mongoDocumentToAggregate(aggregate),
		Version:   mongoDocumentToVersion(version),
		Items:     make([]aggregateservice.RunItem, 0, len(items)),
	}
	for _, item := range items {
		detail.Items = append(detail.Items, mongoDocumentToRunItem(item))
	}
	return detail
}

func aggregateSnapshotFromMongoDocument(document aggregateMongoDocument) aggregateSnapshotDocument {
	return aggregateSnapshotDocument{
		AggregateID:     document.AggregateID,
		Name:            document.Name,
		ActiveVersionID: document.ActiveVersionID,
		CreatedAt:       document.CreatedAt,
		UpdatedAt:       document.UpdatedAt,
		ArchivedAt:      document.ArchivedAt,
	}
}

func versionSnapshotFromMongoDocument(document versionMongoDocument) versionSnapshotDocument {
	return versionSnapshotDocument{
		VersionID:   document.VersionID,
		AggregateID: document.AggregateID,
		Version:     document.Version,
		SpecHash:    document.SpecHash,
		CreatedAt:   document.CreatedAt,
	}
}

func rawJSONToBSONValue(raw json.RawMessage) (any, error) {
	normalized := raw
	if len(normalized) == 0 {
		normalized = json.RawMessage(`{}`)
	}
	var value any
	if err := json.Unmarshal(normalized, &value); err != nil {
		return nil, oops.In("aggregate_repository").Wrapf(err, "decode json payload")
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

func aggregateMongoID(aggregateID string) string {
	return strings.Join([]string{"aggregates", aggregateID}, ":")
}

func aggregateVersionMongoID(versionID string) string {
	return strings.Join([]string{"aggregate_versions", versionID}, ":")
}

func aggregateRunMongoID(runID string) string {
	return strings.Join([]string{"aggregate_runs", runID}, ":")
}

func aggregateRunItemMongoID(runID string, ordinal int) string {
	return strings.Join([]string{"aggregate_run_items", runID, strconv.Itoa(ordinal)}, ":")
}
