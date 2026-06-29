package strategyfundamentals

import "github.com/awuzag/mwosa/storage/repositoryregistry"

func Register(registry *repositoryregistry.Registry) error {
	for _, backend := range []repositoryregistry.Backend{repositoryregistry.BackendSQLite, repositoryregistry.BackendPostgres} {
		if err := registry.Register(repositoryregistry.NameStrategyFundamentals, backend, createSQLRepository); err != nil {
			return err
		}
	}
	return registry.Register(repositoryregistry.NameStrategyFundamentals, repositoryregistry.BackendMongoDB, createMongoRepository)
}

func createSQLRepository(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	database, err := ctx.Resolver.SQL()
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
	database, err := ctx.Resolver.Mongo()
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	repository, err := NewMongoRepository(database)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	return repositoryregistry.One(repository), nil
}
