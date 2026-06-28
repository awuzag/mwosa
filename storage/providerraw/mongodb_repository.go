package providerraw

import (
	"context"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
}

type rawSnapshotMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	Source           rawSnapshotMongoSource `bson:"source"`
	CanonicalSupport string                 `bson:"canonical_support,omitempty"`
	RowCount         int                    `bson:"row_count"`
	Payload          any                    `bson:"payload"`
}

type rawSnapshotMongoSource struct {
	Provider      string `bson:"provider"`
	ProviderGroup string `bson:"provider_group"`
	Operation     string `bson:"operation"`
	BaseDate      string `bson:"base_date"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("provider_raw_repository").New("mongodb database is nil")
	}
	return MongoRepository{collection: database.Collection("provider_raw_snapshots")}, nil
}

func (r MongoRepository) UpsertSnapshot(ctx context.Context, snapshot Snapshot) (WriteResult, error) {
	errb := oops.In("provider_raw_repository").With(
		"backend", "mongodb",
		"provider", snapshot.Provider,
		"group", snapshot.Group,
		"operation", snapshot.Operation,
		"base_date", snapshot.BaseDate,
	)
	if snapshot.Provider == "" || snapshot.Group == "" || snapshot.Operation == "" {
		return WriteResult{}, errb.New("provider raw snapshot missing natural key")
	}
	document, err := snapshotToMongoDocument(snapshot)
	if err != nil {
		return WriteResult{}, errb.Wrap(err)
	}
	if _, err := bson.Marshal(bson.M{"payload": document.Payload}); err != nil {
		return WriteResult{}, errb.Wrapf(err, "encode provider raw snapshot bson payload")
	}
	update := bson.D{
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: document.ID},
			{Key: "created_at", Value: document.CreatedAt},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "schema_version", Value: document.SchemaVersion},
			{Key: "updated_at", Value: document.UpdatedAt},
			{Key: "source", Value: document.Source},
			{Key: "canonical_support", Value: document.CanonicalSupport},
			{Key: "row_count", Value: document.RowCount},
			{Key: "payload", Value: document.Payload},
		}},
		{Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}},
	}
	if _, err := r.collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: document.ID}}, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return WriteResult{}, errb.With("id", document.ID).Wrapf(err, "upsert provider raw snapshot mongodb document")
	}
	return WriteResult{
		Provider:         snapshot.Provider,
		Group:            snapshot.Group,
		Operation:        snapshot.Operation,
		BaseDate:         document.Source.BaseDate,
		CanonicalSupport: snapshot.CanonicalSupport,
		RowCount:         snapshot.RowCount,
		RowsAffected:     1,
	}, nil
}

func (r MongoRepository) ListSnapshots(ctx context.Context, query Query) ([]SnapshotRecord, error) {
	errb := oops.In("provider_raw_repository").With(
		"backend", "mongodb",
		"provider", query.Provider,
		"group", query.Group,
		"operation", query.Operation,
		"from", query.From,
		"to", query.To,
	)
	filter, err := rawSnapshotMongoFilter(query)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	findOptions := options.Find().SetSort(bson.D{
		{Key: "source.base_date", Value: -1},
		{Key: "source.provider", Value: 1},
		{Key: "source.provider_group", Value: 1},
		{Key: "source.operation", Value: 1},
	})
	if query.Limit > 0 {
		findOptions.SetLimit(int64(query.Limit))
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errb.Wrapf(err, "query provider raw snapshots mongodb")
	}
	defer cursor.Close(ctx)
	var documents []rawSnapshotMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode provider raw snapshots mongodb")
	}
	records := make([]SnapshotRecord, 0, len(documents))
	for _, document := range documents {
		records = append(records, mongoDocumentToSnapshotRecord(document, query.IncludePayload))
	}
	return records, nil
}

func snapshotToMongoDocument(snapshot Snapshot) (rawSnapshotMongoDocument, error) {
	baseDate, err := parseBaseDate(snapshot.BaseDate)
	if err != nil {
		return rawSnapshotMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	source := rawSnapshotMongoSource{
		Provider:      string(snapshot.Provider),
		ProviderGroup: string(snapshot.Group),
		Operation:     string(snapshot.Operation),
		BaseDate:      formatBaseDate(baseDate),
	}
	return rawSnapshotMongoDocument{
		ID:               rawSnapshotMongoID(source),
		SchemaVersion:    storagemongodb.SchemaVersion1,
		Revision:         1,
		CreatedAt:        now,
		UpdatedAt:        now,
		Source:           source,
		CanonicalSupport: strings.TrimSpace(snapshot.CanonicalSupport),
		RowCount:         snapshot.RowCount,
		Payload:          snapshot.Rows,
	}, nil
}

func rawSnapshotMongoFilter(query Query) (bson.D, error) {
	filter := bson.D{}
	if strings.TrimSpace(string(query.Provider)) != "" {
		filter = append(filter, bson.E{Key: "source.provider", Value: strings.TrimSpace(string(query.Provider))})
	}
	if strings.TrimSpace(string(query.Group)) != "" {
		filter = append(filter, bson.E{Key: "source.provider_group", Value: strings.TrimSpace(string(query.Group))})
	}
	if strings.TrimSpace(string(query.Operation)) != "" {
		filter = append(filter, bson.E{Key: "source.operation", Value: strings.TrimSpace(string(query.Operation))})
	}
	rangeFilter := bson.D{}
	if strings.TrimSpace(query.From) != "" {
		from, err := normalizeRawSnapshotBaseDate(query.From)
		if err != nil {
			return nil, err
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$gte", Value: from})
	}
	if strings.TrimSpace(query.To) != "" {
		to, err := normalizeRawSnapshotBaseDate(query.To)
		if err != nil {
			return nil, err
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$lte", Value: to})
	}
	if len(rangeFilter) > 0 {
		filter = append(filter, bson.E{Key: "source.base_date", Value: rangeFilter})
	}
	return filter, nil
}

func mongoDocumentToSnapshotRecord(document rawSnapshotMongoDocument, includePayload bool) SnapshotRecord {
	var payload any
	if includePayload {
		payload = document.Payload
	}
	return SnapshotRecord{
		Provider:         provider.ProviderID(document.Source.Provider),
		Group:            provider.GroupID(document.Source.ProviderGroup),
		Operation:        provider.OperationID(document.Source.Operation),
		BaseDate:         document.Source.BaseDate,
		CanonicalSupport: document.CanonicalSupport,
		RowCount:         document.RowCount,
		Payload:          payload,
		CreatedAtMS:      document.CreatedAt.UTC().UnixMilli(),
		UpdatedAtMS:      document.UpdatedAt.UTC().UnixMilli(),
	}
}

func normalizeRawSnapshotBaseDate(value string) (string, error) {
	baseDate, err := parseBaseDate(value)
	if err != nil {
		return "", err
	}
	return formatBaseDate(baseDate), nil
}

func rawSnapshotMongoID(source rawSnapshotMongoSource) string {
	return strings.Join([]string{
		"provider_raw_snapshots",
		source.Provider,
		source.ProviderGroup,
		source.Operation,
		source.BaseDate,
	}, ":")
}
