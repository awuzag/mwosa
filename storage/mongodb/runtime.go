package mongodb

import (
	"context"
	"time"

	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Runtime struct {
	config   Config
	client   *mongo.Client
	database *mongo.Database
}

type CheckStatus struct {
	Ping        PingStatus                  `json:"ping"`
	Server      ServerStatus                `json:"server"`
	Collections map[string]CollectionStatus `json:"collections"`
}

type PingStatus struct {
	Status string `json:"status"`
}

type ServerStatus struct {
	Version string `json:"version"`
}

type CollectionStatus struct {
	Status    string          `json:"status"`
	Validator bool            `json:"validator"`
	Indexes   map[string]bool `json:"indexes"`
}

func NewRuntime(ctx context.Context, config Config) (*Runtime, error) {
	config, err := config.WithDefaults()
	if err != nil {
		return nil, err
	}
	errb := oops.In("mongodb_runtime").With("database", config.Database)

	client, err := mongo.Connect(options.Client().ApplyURI(config.URI).SetAppName(config.AppName))
	if err != nil {
		return nil, errb.Wrapf(err, "connect mongodb")
	}
	runtime := &Runtime{
		config:   config,
		client:   client,
		database: client.Database(config.Database),
	}
	if err := runtime.Ping(ctx); err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "ping mongodb"),
			runtime.Close(ctx),
		)
	}
	return runtime, nil
}

func (r *Runtime) Client() *mongo.Client {
	if r == nil {
		return nil
	}
	return r.client
}

func (r *Runtime) Database() *mongo.Database {
	if r == nil {
		return nil
	}
	return r.database
}

func (r *Runtime) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return oops.In("mongodb_runtime").New("mongodb runtime is not initialized")
	}
	pingCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()
	if err := r.client.Ping(pingCtx, readpref.Primary()); err != nil {
		return oops.In("mongodb_runtime").With("database", r.config.Database).Wrapf(err, "ping mongodb")
	}
	return nil
}

func (r *Runtime) Init(ctx context.Context) error {
	if r == nil || r.database == nil {
		return oops.In("mongodb_runtime").New("mongodb runtime is not initialized")
	}
	initCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()
	return EnsureCollections(initCtx, r.database)
}

func (r *Runtime) Check(ctx context.Context) (CheckStatus, error) {
	if r == nil || r.database == nil {
		return CheckStatus{}, oops.In("mongodb_runtime").New("mongodb runtime is not initialized")
	}
	checkCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	if err := r.Ping(checkCtx); err != nil {
		return CheckStatus{}, err
	}
	var server struct {
		Version string `bson:"version"`
	}
	if err := r.database.RunCommand(checkCtx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&server); err != nil {
		return CheckStatus{}, oops.In("mongodb_runtime").Wrapf(err, "read mongodb server info")
	}
	existingCollections, err := collectionNameSet(checkCtx, r.database)
	if err != nil {
		return CheckStatus{}, oops.In("mongodb_runtime").Wrapf(err, "list mongodb collections")
	}
	validators, err := collectionValidatorSet(checkCtx, r.database)
	if err != nil {
		return CheckStatus{}, oops.In("mongodb_runtime").Wrapf(err, "read mongodb collection validators")
	}
	collections := map[string]CollectionStatus{}
	for _, spec := range CollectionSpecs() {
		if !existingCollections[spec.Name] {
			collections[spec.Name] = CollectionStatus{Status: "missing_collection", Validator: false, Indexes: map[string]bool{}}
			continue
		}
		indexes, err := collectionIndexes(checkCtx, r.database.Collection(spec.Name))
		if err != nil {
			return CheckStatus{}, oops.In("mongodb_runtime").With("collection", spec.Name).Wrapf(err, "read mongodb collection indexes")
		}
		status := "ok"
		if !validators[spec.Name] {
			status = "missing_validator"
		} else {
			for _, index := range spec.Indexes {
				if !indexes[index.Name] {
					status = "missing_index"
					break
				}
			}
		}
		collections[spec.Name] = CollectionStatus{Status: status, Validator: validators[spec.Name], Indexes: indexes}
	}
	return CheckStatus{
		Ping:        PingStatus{Status: "ok"},
		Server:      ServerStatus{Version: server.Version},
		Collections: collections,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil || r.client == nil {
		return nil
	}
	if err := r.client.Disconnect(ctx); err != nil {
		return oops.In("mongodb_runtime").With("database", r.config.Database).Wrapf(err, "close mongodb client")
	}
	return nil
}

func UpsertByID(ctx context.Context, collection *mongo.Collection, id string, document any) (*mongo.UpdateResult, error) {
	if collection == nil {
		return nil, oops.In("mongodb_repository").New("mongodb collection is nil")
	}
	result, err := collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, document, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, oops.In("mongodb_repository").With("collection", collection.Name(), "id", id).Wrapf(err, "upsert mongodb document")
	}
	return result, nil
}

func UpdateWithRevision(ctx context.Context, collection *mongo.Collection, id string, revision int64, set bson.D) (*mongo.UpdateResult, error) {
	if collection == nil {
		return nil, oops.In("mongodb_repository").New("mongodb collection is nil")
	}
	update := bson.D{
		{Key: "$set", Value: append(set, bson.E{Key: "updated_at", Value: ISOTimeNow()})},
		{Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}},
	}
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}, {Key: "revision", Value: revision}}, update)
	if err != nil {
		return nil, oops.In("mongodb_repository").With("collection", collection.Name(), "id", id, "revision", revision).Wrapf(err, "update mongodb document")
	}
	if result.MatchedCount == 0 {
		return nil, NewRevisionConflictError(collection.Name(), id, revision)
	}
	return result, nil
}

func ISOTimeNow() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

func collectionIndexes(ctx context.Context, collection *mongo.Collection) (map[string]bool, error) {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	indexes := map[string]bool{}
	for cursor.Next(ctx) {
		var item struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		indexes[item.Name] = true
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return indexes, nil
}

func collectionNameSet(ctx context.Context, database *mongo.Database) (map[string]bool, error) {
	names, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(names))
	for _, name := range names {
		out[name] = true
	}
	return out, nil
}

func collectionValidatorSet(ctx context.Context, database *mongo.Database) (map[string]bool, error) {
	cursor, err := database.ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := map[string]bool{}
	for cursor.Next(ctx) {
		var item struct {
			Name    string `bson:"name"`
			Options struct {
				Validator bson.Raw `bson:"validator"`
			} `bson:"options"`
		}
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		out[item.Name] = len(item.Options.Validator) > 0
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
