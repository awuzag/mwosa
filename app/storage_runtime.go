package app

import (
	"context"
	"path/filepath"
	"time"

	appconfig "github.com/awuzag/mwosa/app/config"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/awuzag/mwosa/storage/providerauth"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	repositorybuiltin "github.com/awuzag/mwosa/storage/repositoryregistry/builtin"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewStorageRuntime(opts Options) (StorageRuntime, error) {
	backend := normalizedDatabaseBackend(opts)
	runtime := StorageRuntime{
		Backend:              backend,
		ProviderAuthDatabase: providerauth.NewDatabase(providerAuthDatabasePath(opts)),
	}
	if requiresSQLDatabase(backend, opts.RepositoryBackends) {
		runtime.SQLDatabase = storage.NewSQLDatabaseWithConfig(sqlDatabaseConfigFromOptions(opts))
	}

	if requiresMongoDBRuntime(backend, opts.RepositoryBackends) {
		mongoRuntime, err := storagemongodb.NewRuntime(context.Background(), storagemongodb.Config{
			URI: opts.DatabaseURL,
		})
		if err != nil {
			return StorageRuntime{}, oops.Join(
				oops.In("app_storage_runtime").Wrapf(err, "create mongodb runtime"),
				runtime.Close(),
			)
		}
		runtime.MongoDB = mongoRuntime
	}

	registry := repositoryregistry.New()
	if err := repositorybuiltin.Register(registry); err != nil {
		return StorageRuntime{}, oops.Join(
			oops.In("app_storage_runtime").Wrapf(err, "register builtin repositories"),
			runtime.Close(),
		)
	}
	factory, err := NewRepositoryFactory(RepositoryFactoryConfig{
		DefaultBackend:     backend,
		RepositoryBackends: opts.RepositoryBackends,
		Registry:           registry,
		SQLDatabase: func() (*storage.SQLDatabase, error) {
			if runtime.SQLDatabase == nil {
				return nil, oops.In("app_storage_runtime").New("sql database runtime is not initialized")
			}
			return runtime.SQLDatabase, nil
		},
		MongoDatabase: func() (*mongo.Database, error) {
			if runtime.MongoDB == nil {
				return nil, oops.In("app_storage_runtime").New("mongodb runtime is not initialized")
			}
			return runtime.MongoDB.Database(), nil
		},
	})
	if err != nil {
		return StorageRuntime{}, oops.Join(err, runtime.Close())
	}
	runtime, err = initializeStorageRepositories(runtime, factory)
	if err != nil {
		return StorageRuntime{}, oops.Join(
			oops.In("app_storage_runtime").Wrapf(err, "create repositories"),
			runtime.Close(),
		)
	}
	return runtime, nil
}

func initializeStorageRepositories(runtime StorageRuntime, factory RepositoryFactory) (StorageRuntime, error) {
	if err := factory.Initialize(&runtime); err != nil {
		return StorageRuntime{}, err
	}
	return runtime, nil
}

func (r StorageRuntime) Close() error {
	return oops.Join(
		r.SQLDatabase.Close(),
		closeMongoDBRuntime(r.MongoDB),
		r.ProviderAuthDatabase.Close(),
	)
}

func (r StorageRuntime) RequireSQLDatabase(feature string) (*storage.SQLDatabase, error) {
	if r.SQLDatabase == nil {
		return nil, oops.In("app_storage_runtime").
			With("backend", r.Backend, "feature", feature).
			New("unsupported storage repository")
	}
	return r.SQLDatabase, nil
}

func normalizedDatabaseBackend(opts Options) string {
	if opts.DatabaseBackend == "" {
		return appconfig.DatabaseBackendSQLite
	}
	return opts.DatabaseBackend
}

func sqlDatabaseConfigFromOptions(opts Options) storage.DatabaseConfig {
	backend := storage.BackendSQLite
	if opts.DatabaseBackend == appconfig.DatabaseBackendPostgres {
		backend = storage.BackendPostgres
	}
	return storage.DatabaseConfig{
		Backend: backend,
		Path:    opts.Database,
		URL:     opts.DatabaseURL,
	}
}

func requiresMongoDBRuntime(defaultBackend string, repositoryBackends map[RepositoryName]string) bool {
	if defaultBackend == appconfig.DatabaseBackendMongoDB {
		return true
	}
	for _, backend := range repositoryBackends {
		if backend == appconfig.DatabaseBackendMongoDB {
			return true
		}
	}
	return false
}

func requiresSQLDatabase(defaultBackend string, repositoryBackends map[RepositoryName]string) bool {
	if defaultBackend == appconfig.DatabaseBackendSQLite || defaultBackend == appconfig.DatabaseBackendPostgres {
		return true
	}
	for _, backend := range repositoryBackends {
		if backend == appconfig.DatabaseBackendSQLite || backend == appconfig.DatabaseBackendPostgres {
			return true
		}
	}
	return false
}

func providerAuthDatabasePath(opts Options) string {
	if opts.ProviderAuthDatabase != "" {
		return opts.ProviderAuthDatabase
	}
	return filepath.Join(filepath.Dir(opts.Database), appconfig.ProviderAuthDatabaseFileName)
}

func closeMongoDBRuntime(runtime *storagemongodb.Runtime) error {
	if runtime == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runtime.Close(ctx)
}
