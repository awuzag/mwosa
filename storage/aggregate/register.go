package aggregate

import (
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Register(registry *repositoryregistry.Registry) error {
	for _, backend := range []repositoryregistry.Backend{repositoryregistry.BackendSQLite, repositoryregistry.BackendPostgres} {
		if err := registry.Register(repositoryregistry.NameAggregate, backend, createUnsupportedRepository); err != nil {
			return err
		}
	}
	return registry.Register(repositoryregistry.NameAggregate, repositoryregistry.BackendMongoDB, createMongoRepository)
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

func createUnsupportedRepository(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	return repositoryregistry.One(unsupportedRepository{backend: string(ctx.Backend)}), nil
}
