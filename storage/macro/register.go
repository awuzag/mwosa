package macro

import "github.com/awuzag/mwosa/storage/repositoryregistry"

func Register(registry *repositoryregistry.Registry) error {
	for _, backend := range []repositoryregistry.Backend{repositoryregistry.BackendSQLite, repositoryregistry.BackendPostgres} {
		if err := registry.Register(repositoryregistry.NameMacro, backend, createSQLRepositories); err != nil {
			return err
		}
	}
	return registry.Register(repositoryregistry.NameMacro, repositoryregistry.BackendMongoDB, createMongoRepositories)
}

func createSQLRepositories(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	database, err := ctx.Resolver.SQL()
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	reader, writer, err := NewRepository(database)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	return repositoryregistry.Pair(reader, writer), nil
}

func createMongoRepositories(ctx repositoryregistry.BuildContext) (repositoryregistry.Result, error) {
	database, err := ctx.Resolver.Mongo()
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	reader, writer, err := NewMongoRepositories(database)
	if err != nil {
		return repositoryregistry.Result{}, err
	}
	return repositoryregistry.Pair(reader, writer), nil
}
