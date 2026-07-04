package app

import (
	"context"
	"reflect"
	"time"

	appconfig "github.com/awuzag/mwosa/app/config"
	migrationcore "github.com/awuzag/mwosa/migration"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
	migrationstorage "github.com/awuzag/mwosa/storage/migration"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type RepositoryName string

const (
	RepositoryDailyBars          RepositoryName = "dailybar"
	RepositoryIndexBars          RepositoryName = "indexbar"
	RepositoryMacro              RepositoryName = "macro"
	RepositoryInstruments        RepositoryName = "instrument"
	RepositoryCompositions       RepositoryName = "composition"
	RepositoryAggregates         RepositoryName = "aggregate"
	RepositoryStrategies         RepositoryName = "strategy"
	RepositoryFundamentals       RepositoryName = "strategyfundamentals"
	RepositoryBacktestStrategies RepositoryName = "backtest"
	RepositoryProviderRaw        RepositoryName = "providerraw"
	RepositoryMigrations         RepositoryName = "migration"
)

type RepositoryFactoryConfig struct {
	DefaultBackend string
	// RepositoryBackends is an internal assembly hook for tests and staged
	// backend work. It is not a user-facing config or CLI override surface.
	RepositoryBackends map[RepositoryName]string
	Registry           *repositoryregistry.Registry
	SQLDatabase        func() (*storage.SQLDatabase, error)
	MongoDatabase      func() (*mongo.Database, error)
}

type RepositoryFactory interface {
	Backend(RepositoryName) (string, error)
	Initialize(*StorageRuntime) error
}

type repositoryFactory struct {
	defaultBackend     string
	repositoryBackends map[RepositoryName]string
	registry           *repositoryregistry.Registry
	sqlDatabase        func() (*storage.SQLDatabase, error)
	mongoDatabase      func() (*mongo.Database, error)
}

func NewRepositoryFactory(config RepositoryFactoryConfig) (RepositoryFactory, error) {
	if config.Registry == nil {
		return nil, oops.In("repository_factory").New("repository registry is nil")
	}
	defaultBackend := config.DefaultBackend
	if defaultBackend == "" {
		defaultBackend = appconfig.DatabaseBackendSQLite
	}
	if err := validateRepositoryBackend(defaultBackend); err != nil {
		return nil, err
	}
	for repository, backend := range config.RepositoryBackends {
		if backend == "" {
			continue
		}
		if err := validateRepositoryBackend(backend); err != nil {
			return nil, oops.In("repository_factory").With("repository", repository).Wrap(err)
		}
	}
	return repositoryFactory{
		defaultBackend:     defaultBackend,
		repositoryBackends: cloneRepositoryBackends(config.RepositoryBackends),
		registry:           config.Registry,
		sqlDatabase:        config.SQLDatabase,
		mongoDatabase:      config.MongoDatabase,
	}, nil
}

func (f repositoryFactory) Backend(repository RepositoryName) (string, error) {
	if backend := f.repositoryBackends[repository]; backend != "" {
		return backend, nil
	}
	return f.defaultBackend, nil
}

func (f repositoryFactory) Initialize(runtime *StorageRuntime) error {
	if runtime == nil {
		return oops.In("repository_factory").New("storage runtime is nil")
	}
	runtimeValue := reflect.ValueOf(runtime).Elem()
	runtimeType := runtimeValue.Type()
	for i := 0; i < runtimeType.NumField(); i++ {
		field := runtimeType.Field(i)
		repository, ok := repositoryFromField(field)
		if !ok {
			continue
		}
		values, err := f.create(repository)
		if err != nil {
			return oops.In("repository_factory").
				With("repository", repository, "field", field.Name).
				Wrapf(err, "create repository")
		}
		if err := assignRepositoryField(runtimeValue.Field(i), values); err != nil {
			return oops.In("repository_factory").
				With("repository", repository, "field", field.Name).
				Wrapf(err, "assign repository")
		}
	}
	// Migration execution composes a migration store with repository-specific
	// executors, so it stays in the app assembly path instead of the repository
	// registry. Unsupported backends still fail explicitly at execution time.
	migrationRunner, err := f.migrationRunner(runtime.DailyBars.Writer)
	if err != nil {
		return oops.In("repository_factory").Wrapf(err, "create migration runner")
	}
	runtime.Migration = migrationRunner
	return nil
}

func (f repositoryFactory) create(repository RepositoryName) ([]reflect.Value, error) {
	backend, err := f.Backend(repository)
	if err != nil {
		return nil, err
	}
	creator, err := f.registry.Creator(repositoryregistry.Name(repository), repositoryregistry.Backend(backend))
	if err != nil {
		return nil, err
	}
	result, err := creator(repositoryregistry.BuildContext{
		Name:     repositoryregistry.Name(repository),
		Backend:  repositoryregistry.Backend(backend),
		Resolver: repositoryResolver{factory: f, repository: repository},
	})
	if err != nil {
		return nil, err
	}
	return reflectValuesFromResult(repository, backend, result)
}

type repositoryResolver struct {
	factory    repositoryFactory
	repository RepositoryName
}

func (r repositoryResolver) Resolve(backend repositoryregistry.Backend) (any, error) {
	switch backend {
	case repositoryregistry.BackendSQLite, repositoryregistry.BackendPostgres:
		return r.factory.sqlDatabaseFor(r.repository)
	case repositoryregistry.BackendMongoDB:
		return r.factory.mongoDatabaseFor(r.repository)
	default:
		return nil, unsupportedRepositoryBackend(string(backend), r.repository)
	}
}

func reflectValuesFromResult(repository RepositoryName, backend string, result repositoryregistry.Result) ([]reflect.Value, error) {
	values := make([]reflect.Value, 0, len(result.Values))
	for index, value := range result.Values {
		if value == nil {
			return nil, oops.In("repository_factory").
				With("backend", backend, "repository", repository, "index", index).
				New("repository creator returned nil value")
		}
		values = append(values, reflect.ValueOf(value))
	}
	return values, nil
}

func assignRepositoryField(field reflect.Value, values []reflect.Value) error {
	switch len(values) {
	case 0:
		return oops.In("repository_factory").New("repository constructor returned no repository values")
	case 1:
		return setReflectValue(field, values[0])
	default:
		if field.Kind() != reflect.Struct {
			return oops.In("repository_factory").New("multi-value repository constructor requires struct field")
		}
		if len(values) > field.NumField() {
			return oops.In("repository_factory").New("repository constructor returned too many values")
		}
		for i, value := range values {
			if err := setReflectValue(field.Field(i), value); err != nil {
				return err
			}
		}
		return nil
	}
}

func setReflectValue(field reflect.Value, value reflect.Value) error {
	if !field.CanSet() {
		return oops.In("repository_factory").New("repository field cannot be set")
	}
	if value.Type().AssignableTo(field.Type()) {
		field.Set(value)
		return nil
	}
	return oops.In("repository_factory").
		With("field_type", field.Type().String(), "value_type", value.Type().String()).
		New("repository value type mismatch")
}

func repositoryFromField(field reflect.StructField) (RepositoryName, bool) {
	repository := field.Tag.Get("repository")
	if repository == "" {
		return "", false
	}
	return RepositoryName(repository), true
}

func (f repositoryFactory) migrationRunner(writer daily.WriteRepository) (migrationcore.Runner, error) {
	backend, err := f.Backend(RepositoryMigrations)
	if err != nil {
		return migrationcore.Runner{}, err
	}
	switch backend {
	case appconfig.DatabaseBackendSQLite, appconfig.DatabaseBackendPostgres:
		database, err := f.sqlDatabaseFor(RepositoryMigrations)
		if err != nil {
			return migrationcore.Runner{}, err
		}
		store, err := migrationstorage.NewRepository(database)
		if err != nil {
			return migrationcore.Runner{}, err
		}
		dailyBarMigration, err := migrationstorage.NewDailyBarV1ToV2Executor(database, writer)
		if err != nil {
			return migrationcore.Runner{}, err
		}
		dailyBarExtensionCleanup, err := migrationstorage.NewDailyBarV2ExtensionCleanupExecutor(database)
		if err != nil {
			return migrationcore.Runner{}, err
		}
		return migrationcore.NewRunner(store, []migrationcore.Definition{
			migrationstorage.NewDailyBarV1ToV2Definition(dailyBarMigration),
			migrationstorage.NewDailyBarV2ExtensionCleanupDefinition(dailyBarExtensionCleanup),
		})
	case appconfig.DatabaseBackendMongoDB:
		store := unsupportedMigrationStore{backend: backend}
		executor := unsupportedMigrationExecutor{backend: backend}
		return migrationcore.NewRunner(store, []migrationcore.Definition{
			migrationstorage.NewDailyBarV1ToV2Definition(executor),
			migrationstorage.NewDailyBarV2ExtensionCleanupDefinition(executor),
		})
	default:
		return migrationcore.Runner{}, unsupportedRepositoryBackend(backend, RepositoryMigrations)
	}
}

func (f repositoryFactory) sqlDatabaseFor(repository RepositoryName) (*storage.SQLDatabase, error) {
	if f.sqlDatabase == nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			New("sql database factory is nil")
	}
	database, err := f.sqlDatabase()
	if err != nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			Wrapf(err, "create sql database handle")
	}
	if database == nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			New("sql database is nil")
	}
	return database, nil
}

func (f repositoryFactory) mongoDatabaseFor(repository RepositoryName) (*mongo.Database, error) {
	if f.mongoDatabase == nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			New("mongodb database factory is nil")
	}
	database, err := f.mongoDatabase()
	if err != nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			Wrapf(err, "create mongodb database handle")
	}
	if database == nil {
		return nil, oops.In("repository_factory").
			With("backend", f.defaultBackend, "repository", repository).
			New("mongodb database is nil")
	}
	return database, nil
}

type unsupportedMigrationStore struct {
	backend string
}

func (s unsupportedMigrationStore) GetRun(context.Context, string) (migrationcore.MigrationRun, bool, error) {
	return migrationcore.MigrationRun{}, false, unsupportedRepositoryBackend(s.backend, RepositoryMigrations)
}

func (s unsupportedMigrationStore) RecordApplied(context.Context, migrationcore.Definition, int64, time.Time) (migrationcore.MigrationRun, error) {
	return migrationcore.MigrationRun{}, unsupportedRepositoryBackend(s.backend, RepositoryMigrations)
}

type unsupportedMigrationExecutor struct {
	backend string
}

func (e unsupportedMigrationExecutor) Apply(context.Context) (int64, error) {
	return 0, unsupportedRepositoryBackend(e.backend, RepositoryMigrations)
}

func validateRepositoryBackend(backend string) error {
	switch backend {
	case appconfig.DatabaseBackendSQLite, appconfig.DatabaseBackendPostgres, appconfig.DatabaseBackendMongoDB:
		return nil
	default:
		return unsupportedRepositoryBackend(backend, "")
	}
}

func unsupportedRepositoryBackend(backend string, repository RepositoryName) error {
	return oops.In("repository_factory").
		With("backend", backend, "repository", repository).
		New("unsupported storage repository")
}

func cloneRepositoryBackends(in map[RepositoryName]string) map[RepositoryName]string {
	out := make(map[RepositoryName]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
