//go:build integration

package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestRuntimeInitializesCollectionsAndChecksStatus(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := NewRuntime(ctx, Config{
		URI:      server.URI,
		Database: "mwosa_runtime_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})

	require.NotNil(t, runtime.Client())
	require.NotNil(t, runtime.Database())
	require.NoError(t, runtime.Ping(ctx))
	require.NoError(t, runtime.Init(ctx))

	status, err := runtime.Check(ctx)
	require.NoError(t, err)
	require.Equal(t, "ok", status.Ping.Status)
	require.NotEmpty(t, status.Server.Version)
	require.Equal(t, "ok", status.Collections["daily_bars"].Status)
	require.True(t, status.Collections["daily_bars"].Validator)
	require.True(t, status.Collections["daily_bars"].Indexes["daily_bars_instrument_source_date_unique"])
}

func TestRuntimeValidatesDailyBarsUpsertAndRevisionConflict(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := NewRuntime(ctx, Config{
		URI:      server.URI,
		Database: "mwosa_daily_bars_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	collection := runtime.Database().Collection("daily_bars")
	now := time.Date(2026, 6, 28, 3, 34, 56, 789000000, time.UTC)
	fields, err := NewDocumentFields("daily_bars:krx:etf:069500:20260628:datago:securitiesProductPrice:getETFPriceInfo", "1.0.0", now)
	require.NoError(t, err)

	document := bson.D{
		{Key: "_id", Value: fields.ID},
		{Key: "schema_version", Value: fields.SchemaVersion},
		{Key: "revision", Value: fields.Revision},
		{Key: "created_at", Value: fields.CreatedAt},
		{Key: "updated_at", Value: fields.UpdatedAt},
		{Key: "instrument_key", Value: "krx:etf:069500"},
		{Key: "market_key", Value: "krx"},
		{Key: "security_type", Value: "etf"},
		{Key: "symbol", Value: "069500"},
		{Key: "trading_date", Value: "2026-06-28"},
		{Key: "source", Value: bson.D{
			{Key: "provider", Value: "datago"},
			{Key: "provider_group", Value: "securitiesProductPrice"},
			{Key: "operation", Value: "getETFPriceInfo"},
		}},
		{Key: "prices", Value: bson.D{{Key: "close", Value: int64(10000)}}},
		{Key: "volumes", Value: bson.D{{Key: "traded", Value: int64(100)}}},
		{Key: "extensions", Value: bson.D{{Key: "nav", Value: "10001"}}},
	}

	result, err := UpsertByID(ctx, collection, fields.ID, document)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.UpsertedCount)

	result, err = UpsertByID(ctx, collection, fields.ID, document)
	require.NoError(t, err)
	require.Equal(t, int64(0), result.UpsertedCount)

	var stored struct {
		CreatedAt time.Time `bson:"created_at"`
		UpdatedAt time.Time `bson:"updated_at"`
	}
	require.NoError(t, collection.FindOne(ctx, bson.D{{Key: "_id", Value: fields.ID}}).Decode(&stored))
	require.True(t, stored.CreatedAt.Equal(now))
	require.Equal(t, "2026-06-28T03:34:56.789Z", ISOTime(stored.CreatedAt).String())

	update, err := UpdateWithRevision(ctx, collection, fields.ID, 1, bson.D{{Key: "prices.close", Value: int64(11000)}})
	require.NoError(t, err)
	require.Equal(t, int64(1), update.ModifiedCount)

	_, err = UpdateWithRevision(ctx, collection, fields.ID, 1, bson.D{{Key: "prices.close", Value: int64(12000)}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "revision")

	invalid := bson.D{
		{Key: "_id", Value: "invalid"},
		{Key: "schema_version", Value: "1.0.0"},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
	}
	_, err = collection.InsertOne(ctx, invalid)
	require.Error(t, err)
	require.False(t, mongo.IsDuplicateKeyError(err))
}

func TestRuntimeValidatorRejectsInvalidCommonFieldTypes(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := NewRuntime(ctx, Config{
		URI:      server.URI,
		Database: "mwosa_common_fields_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	collection := runtime.Database().Collection("storage_metadata")
	now := time.Date(2026, 6, 28, 3, 34, 56, 789000000, time.UTC)
	_, err = collection.InsertOne(ctx, bson.D{
		{Key: "_id", Value: "storage_metadata:valid"},
		{Key: "schema_version", Value: SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "collected_at", Value: now},
		{Key: "source_updated_at", Value: now},
		{Key: "deleted_at", Value: now},
		{Key: "storage_kind", Value: "mongodb"},
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		field string
		value any
	}{
		{name: "revision_string", field: "revision", value: "1"},
		{name: "created_at_string", field: "created_at", value: "2026-06-28T03:34:56.789Z"},
		{name: "updated_at_string", field: "updated_at", value: "2026-06-28T03:34:56.789Z"},
		{name: "collected_at_string", field: "collected_at", value: "2026-06-28T03:34:56.789Z"},
		{name: "source_updated_at_string", field: "source_updated_at", value: "2026-06-28T03:34:56.789Z"},
		{name: "deleted_at_string", field: "deleted_at", value: "2026-06-28T03:34:56.789Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := bson.D{
				{Key: "_id", Value: "storage_metadata:" + tt.name},
				{Key: "schema_version", Value: SchemaVersion1},
				{Key: "revision", Value: int64(1)},
				{Key: "created_at", Value: now},
				{Key: "updated_at", Value: now},
				{Key: "storage_kind", Value: "mongodb:" + tt.name},
			}
			document = setDocumentValue(document, tt.field, tt.value)

			_, err := collection.InsertOne(ctx, document)
			require.Error(t, err)
		})
	}
}

func setDocumentValue(document bson.D, key string, value any) bson.D {
	for i := range document {
		if document[i].Key == key {
			document[i].Value = value
			return document
		}
	}
	return append(document, bson.E{Key: key, Value: value})
}
