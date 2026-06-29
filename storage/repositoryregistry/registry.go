package repositoryregistry

import (
	"sort"

	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Name string

const (
	NameDailyBar             Name = "dailybar"
	NameIndexBar             Name = "indexbar"
	NameMacro                Name = "macro"
	NameInstrument           Name = "instrument"
	NameComposition          Name = "composition"
	NameStrategy             Name = "strategy"
	NameStrategyFundamentals Name = "strategyfundamentals"
	NameBacktest             Name = "backtest"
	NameProviderRaw          Name = "providerraw"
)

type Backend string

const (
	BackendSQLite   Backend = "sqlite"
	BackendPostgres Backend = "postgres"
	BackendMongoDB  Backend = "mongodb"
)

type Resolver interface {
	SQL() (*storage.SQLDatabase, error)
	Mongo() (*mongo.Database, error)
}

type BuildContext struct {
	Name     Name
	Backend  Backend
	Resolver Resolver
}

type Creator func(BuildContext) (Result, error)

type Result struct {
	Values []any
}

func One(value any) Result {
	return Result{Values: []any{value}}
}

func Pair(first any, second any) Result {
	return Result{Values: []any{first, second}}
}

type Registry struct {
	creators map[Name]map[Backend]Creator
}

func New() *Registry {
	return &Registry{
		creators: make(map[Name]map[Backend]Creator),
	}
}

func (r *Registry) Register(name Name, backend Backend, creator Creator) error {
	errb := oops.In("repository_registry").With("repository", name, "backend", backend)
	if r == nil {
		return errb.New("repository registry is nil")
	}
	if name == "" {
		return errb.New("repository name is empty")
	}
	if backend == "" {
		return errb.New("repository backend is empty")
	}
	if creator == nil {
		return errb.New("repository creator is nil")
	}
	if r.creators == nil {
		r.creators = make(map[Name]map[Backend]Creator)
	}
	if r.creators[name] == nil {
		r.creators[name] = make(map[Backend]Creator)
	}
	if _, exists := r.creators[name][backend]; exists {
		return errb.New("repository creator is already registered")
	}
	r.creators[name][backend] = creator
	return nil
}

func (r *Registry) Creator(name Name, backend Backend) (Creator, error) {
	errb := oops.In("repository_registry").With("repository", name, "backend", backend)
	if r == nil {
		return nil, errb.New("repository registry is nil")
	}
	if name == "" {
		return nil, errb.New("repository name is empty")
	}
	if backend == "" {
		return nil, errb.New("repository backend is empty")
	}
	byBackend, ok := r.creators[name]
	if !ok {
		return nil, errb.New("repository is not registered")
	}
	creator, ok := byBackend[backend]
	if !ok {
		return nil, errb.With("registered_backends", registeredBackends(byBackend)).New("repository backend is not registered")
	}
	return creator, nil
}

func registeredBackends(in map[Backend]Creator) []string {
	out := make([]string, 0, len(in))
	for backend := range in {
		out = append(out, string(backend))
	}
	sort.Strings(out)
	return out
}
