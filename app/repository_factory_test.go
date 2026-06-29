package app

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/awuzag/mwosa/app/config"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestRepositoryFactoryAllowsRepositorySpecificBackends(t *testing.T) {
	mongoDatabase := testMongoDatabase(t)
	sqlDatabase := storage.NewSQLDatabase(filepath.Join(t.TempDir(), "mwosa.db"))

	factory, err := NewRepositoryFactory(RepositoryFactoryConfig{
		DefaultBackend: appconfig.DatabaseBackendSQLite,
		RepositoryBackends: map[RepositoryName]string{
			RepositoryDailyBars: appconfig.DatabaseBackendMongoDB,
		},
		Registry: testRepositoryRegistry(t),
		SQLDatabase: func() (*storage.SQLDatabase, error) {
			return sqlDatabase, nil
		},
		MongoDatabase: func() (*mongo.Database, error) {
			return mongoDatabase, nil
		},
	})
	require.NoError(t, err)

	runtime := StorageRuntime{}
	require.NoError(t, factory.Initialize(&runtime))

	assert.Contains(t, fmt.Sprintf("%T", runtime.DailyBars.Reader), "dailybar.mongoRepository")
	assert.NotContains(t, fmt.Sprintf("%T", runtime.Instruments), "mongoRepository")
}

func TestRepositoryFactoryAllowsSQLRepositoryOverrideFromMongoDefault(t *testing.T) {
	sqlDatabase := storage.NewSQLDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	mongoCalls := 0

	factory, err := NewRepositoryFactory(RepositoryFactoryConfig{
		DefaultBackend: appconfig.DatabaseBackendMongoDB,
		RepositoryBackends: map[RepositoryName]string{
			RepositoryInstruments: appconfig.DatabaseBackendSQLite,
		},
		Registry: testRepositoryRegistry(t),
		SQLDatabase: func() (*storage.SQLDatabase, error) {
			return sqlDatabase, nil
		},
		MongoDatabase: func() (*mongo.Database, error) {
			mongoCalls++
			return testMongoDatabase(t), nil
		},
	})
	require.NoError(t, err)

	values, err := factory.(repositoryFactory).create(RepositoryInstruments)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.NotContains(t, fmt.Sprintf("%T", values[0].Interface()), "mongoRepository")
	assert.Zero(t, mongoCalls)
}

func TestRepositoryFactoryUsesMongoDailyAndInstrumentWithoutSQLBackendFallback(t *testing.T) {
	mongoDatabase := testMongoDatabase(t)
	sqlCalls := 0

	factory, err := NewRepositoryFactory(RepositoryFactoryConfig{
		DefaultBackend: appconfig.DatabaseBackendMongoDB,
		Registry:       testRepositoryRegistry(t),
		SQLDatabase: func() (*storage.SQLDatabase, error) {
			sqlCalls++
			return storage.NewSQLDatabase(filepath.Join(t.TempDir(), "unused.db")), nil
		},
		MongoDatabase: func() (*mongo.Database, error) {
			return mongoDatabase, nil
		},
	})
	require.NoError(t, err)

	runtime := StorageRuntime{}
	require.NoError(t, factory.Initialize(&runtime))

	assert.Contains(t, fmt.Sprintf("%T", runtime.DailyBars.Reader), "dailybar.mongoRepository")
	assert.Contains(t, fmt.Sprintf("%T", runtime.Instruments), "instrument.mongoRepository")
	assert.Zero(t, sqlCalls)
}

func TestNewStorageRuntimeInitializesRepositories(t *testing.T) {
	ctx := context.Background()
	runtime, err := NewStorageRuntime(ctx, Options{
		DatabaseBackend: appconfig.DatabaseBackendSQLite,
		Database:        filepath.Join(t.TempDir(), "mwosa.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(ctx))
	})

	assert.NotNil(t, runtime.DailyBars.Reader)
	assert.NotNil(t, runtime.DailyBars.Writer)
	assert.NotNil(t, runtime.IndexBars.Reader)
	assert.NotNil(t, runtime.IndexBars.Writer)
	assert.NotNil(t, runtime.Macro.Reader)
	assert.NotNil(t, runtime.Macro.Writer)
	assert.NotNil(t, runtime.Instruments)
	assert.NotNil(t, runtime.Compositions)
	assert.NotNil(t, runtime.ProviderRaw)
	assert.NotNil(t, runtime.Strategies)
	assert.NotNil(t, runtime.Fundamentals)
	assert.NotNil(t, runtime.BacktestStrategies)
}

func TestNewStorageRuntimeUsesCallerContextForMongoStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runtime, err := NewStorageRuntime(ctx, Options{
		DatabaseBackend: appconfig.DatabaseBackendMongoDB,
		Database:        filepath.Join(t.TempDir(), "mwosa.db"),
		DatabaseURL:     "mongodb://127.0.0.1:27017",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, runtime.MongoDB)
}

func testMongoDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	client, err := mongo.Connect(options.Client().
		ApplyURI("mongodb://127.0.0.1:27017").
		SetServerSelectionTimeout(time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	return client.Database("mwosa_repository_factory_test")
}

func testRepositoryRegistry(t *testing.T) *repositoryregistry.Registry {
	t.Helper()

	registry := repositoryregistry.New()
	require.NoError(t, registerStorageRepositories(registry))
	return registry
}
