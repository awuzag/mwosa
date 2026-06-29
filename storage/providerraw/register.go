package providerraw

import (
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Register(registry *repositoryregistry.Registry) error {
	for _, backend := range []repositoryregistry.Backend{repositoryregistry.BackendSQLite, repositoryregistry.BackendPostgres} {
		if err := registry.Register(repositoryregistry.NameProviderRaw, backend, createSQLRepository); err != nil {
			return err
		}
	}
	return registry.Register(repositoryregistry.NameProviderRaw, repositoryregistry.BackendMongoDB, createMongoRepository)
}

func createSQLRepository(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	database, err := repositoryregistry.Resolve[*storage.SQLDatabase](ctx)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	repository, err := NewRepository(database)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	return repositoryregistry.One(repository), nil
}

func createMongoRepository(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	database, err := repositoryregistry.Resolve[*mongo.Database](ctx)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	repository, err := NewMongoRepository(database)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	return repositoryregistry.One(repository), nil
}
