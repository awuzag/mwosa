package app

import (
	"context"
	"path/filepath"

	appconfig "github.com/awuzag/mwosa/app/config"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/awuzag/mwosa/storage/providerauth"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewStorageRuntime(ctx context.Context, opts Options) (StorageRuntime, error) {
	ctx = runtimeContext(ctx)
	backend := normalizedDatabaseBackend(opts)
	runtime := StorageRuntime{
		Backend:              backend,
		ProviderAuthDatabase: providerauth.NewDatabase(providerAuthDatabasePath(opts)),
	}
	if requiresSQLDatabase(backend, opts.RepositoryBackends) {
		runtime.SQLDatabase = storage.NewSQLDatabaseWithConfig(sqlDatabaseConfigFromOptions(opts))
	}

	if requiresMongoDBRuntime(backend, opts.RepositoryBackends) {
		mongoRuntime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
			URI: opts.DatabaseURL,
		})
		if err != nil {
			return StorageRuntime{}, oops.Join(
				oops.In("app_storage_runtime").Wrapf(err, "create mongodb runtime"),
				runtime.Close(ctx),
			)
		}
		runtime.MongoDB = mongoRuntime
	}

	registry := repositoryregistry.New()
	if err := registerStorageRepositories(registry); err != nil {
		return StorageRuntime{}, oops.Join(
			oops.In("app_storage_runtime").Wrapf(err, "register storage repositories"),
			runtime.Close(ctx),
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
		return StorageRuntime{}, oops.Join(err, runtime.Close(ctx))
	}
	runtime, err = initializeStorageRepositories(runtime, factory)
	if err != nil {
		return StorageRuntime{}, oops.Join(
			oops.In("app_storage_runtime").Wrapf(err, "create repositories"),
			runtime.Close(ctx),
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

func (r StorageRuntime) Close(ctx context.Context) error {
	return oops.Join(
		r.SQLDatabase.Close(),
		closeMongoDBRuntime(ctx, r.MongoDB),
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

func closeMongoDBRuntime(ctx context.Context, runtime *storagemongodb.Runtime) error {
	if runtime == nil {
		return nil
	}
	shutdownCtx, cancel := shutdownContext(ctx)
	defer cancel()
	return runtime.Close(shutdownCtx)
}
