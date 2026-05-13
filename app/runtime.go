package app

import (
	"github.com/ev3rlit/mwosa/app/handler"
	migrationcore "github.com/ev3rlit/mwosa/migration"
	"github.com/ev3rlit/mwosa/providers/builtin"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/quote"
	backtestservice "github.com/ev3rlit/mwosa/service/backtest"
	"github.com/ev3rlit/mwosa/service/daily"
	financialsservice "github.com/ev3rlit/mwosa/service/financials"
	providerservice "github.com/ev3rlit/mwosa/service/providers"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	universeservice "github.com/ev3rlit/mwosa/service/universe"
	"github.com/ev3rlit/mwosa/storage"
	backteststorage "github.com/ev3rlit/mwosa/storage/backtest"
	dailybarstorage "github.com/ev3rlit/mwosa/storage/dailybar"
	migrationstorage "github.com/ev3rlit/mwosa/storage/migration"
	strategystorage "github.com/ev3rlit/mwosa/storage/strategy"
	"github.com/samber/oops"
)

type Options struct {
	Database          string
	Market            provider.Market
	ProviderID        provider.ProviderID
	PreferProvider    provider.ProviderID
	ProviderConfig    provider.Config
	ActivateProviders bool
}

type Runtime struct {
	Storage   StorageRuntime
	Providers ProviderRuntime
	Services  ServiceRuntime
	Handlers  Handlers
}

type StorageRuntime struct {
	Database           *storage.Database
	DailyBars          DailyBarStorage
	Migrations         migrationcore.Store
	Strategies         strategyservice.Repository
	BacktestStrategies backtestservice.StrategyRepository
}

type DailyBarStorage struct {
	Reader daily.ReadRepository
	Writer daily.WriteRepository
}

type ProviderRuntime struct {
	Registry    *provider.Registry
	Router      *provider.Router
	DailyBars   dailybar.Router
	Financials  financials.Router
	Quotes      quote.Router
	Instruments instrument.Router
}

type ServiceRuntime struct {
	Backtest   backtestservice.Service
	Daily      DailyServices
	Financials financialsservice.Service
	Providers  providerservice.Service
	Strategy   strategyservice.Service
}

type Handlers struct {
	Backtest   handler.Backtest
	Daily      handler.Daily
	Financials handler.Financials
	Migration  handler.Migration
	Strategy   handler.Strategy
}

type DailyServices struct {
	Reader    daily.ReadService
	Collector daily.Service
}

func NewRuntime(opts Options) (*Runtime, error) {
	return NewRuntimeWithProviderBuilders(opts, builtin.Builders()...)
}

func NewRuntimeWithProviderBuilders(opts Options, builders ...provider.ProviderBuilder) (*Runtime, error) {
	errb := oops.In("app_runtime")

	database := storage.NewDatabase(opts.Database)
	reader, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		return nil, errb.Wrapf(err, "create daily bar repositories")
	}
	strategyRepository, err := strategystorage.NewRepository(database)
	if err != nil {
		return nil, errb.Wrapf(err, "create strategy repository")
	}
	backtestStrategyRepository, err := backteststorage.NewRepository(database)
	if err != nil {
		return nil, errb.Wrapf(err, "create backtest strategy repository")
	}
	migrationStore, err := migrationstorage.NewRepository(database)
	if err != nil {
		return nil, errb.Wrapf(err, "create migration repository")
	}
	dailyBarMigration, err := migrationstorage.NewDailyBarV1ToV2Executor(database, writer)
	if err != nil {
		return nil, errb.Wrapf(err, "create daily bar migration")
	}
	dailyBarExtensionCleanup, err := migrationstorage.NewDailyBarV2ExtensionCleanupExecutor(database)
	if err != nil {
		return nil, errb.Wrapf(err, "create daily bar extension cleanup migration")
	}
	migrationRunner, err := migrationcore.NewRunner(migrationStore, []migrationcore.Definition{
		migrationstorage.NewDailyBarV1ToV2Definition(dailyBarMigration),
		migrationstorage.NewDailyBarV2ExtensionCleanupDefinition(dailyBarExtensionCleanup),
	})
	if err != nil {
		return nil, errb.Wrapf(err, "create migration runner")
	}

	registry := provider.NewRegistry()
	if opts.ActivateProviders {
		config := opts.ProviderConfig
		if config == nil {
			config = provider.ConfigFromEnv()
		}
		if err := registry.RegisterConfigured(provider.RegisterOptions{
			ProviderID:     opts.ProviderID,
			PreferProvider: opts.PreferProvider,
		}, config, builders...); err != nil {
			return nil, oops.Join(
				errb.Wrapf(err, "register configured providers"),
				database.Close(),
			)
		}
	}

	coreRouter := provider.NewRouter(registry)
	providerRuntime := ProviderRuntime{
		Registry:    registry,
		Router:      coreRouter,
		DailyBars:   dailybar.NewRouter(coreRouter),
		Financials:  financials.NewRouter(coreRouter),
		Quotes:      quote.NewRouter(coreRouter),
		Instruments: instrument.NewRouter(coreRouter),
	}

	dailyReader, err := daily.NewReadService(reader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily read service"),
			database.Close(),
		)
	}
	dailyCollector, err := daily.NewService(reader, writer, providerRuntime.DailyBars)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily collect service"),
			database.Close(),
		)
	}
	providersService, err := providerservice.NewService(registry)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create providers service"),
			database.Close(),
		)
	}
	financialsService, err := financialsservice.NewService(providerRuntime.Financials)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create financials service"),
			database.Close(),
		)
	}
	datasetReader, err := strategyservice.NewDailyBarDatasetReader(reader, opts.Market)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy dataset reader"),
			database.Close(),
		)
	}
	strategyService, err := strategyservice.NewService(strategyRepository, datasetReader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy service"),
			database.Close(),
		)
	}
	universeRunner, err := universeservice.NewRunner(reader, strategyRepository, strategyService)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create universe service"),
			database.Close(),
		)
	}
	backtestService, err := backtestservice.NewServiceWithUniverseSources(reader, backtestStrategyRepository, strategyRepository, strategyService)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create backtest service"),
			database.Close(),
		)
	}
	backtestHandler := handler.NewBacktest(backtestService)
	dailyHandler := handler.NewDaily(dailyReader, dailyCollector)
	financialsHandler := handler.NewFinancials(financialsService)
	strategyHandler := handler.NewStrategy(strategyService, universeRunner)
	migrationHandler := handler.NewMigration(migrationRunner)

	return &Runtime{
		Storage: StorageRuntime{
			Database: database,
			DailyBars: DailyBarStorage{
				Reader: reader,
				Writer: writer,
			},
			Migrations:         migrationStore,
			Strategies:         strategyRepository,
			BacktestStrategies: backtestStrategyRepository,
		},
		Providers: providerRuntime,
		Services: ServiceRuntime{
			Backtest: backtestService,
			Daily: DailyServices{
				Reader:    dailyReader,
				Collector: dailyCollector,
			},
			Financials: financialsService,
			Providers:  providersService,
			Strategy:   strategyService,
		},
		Handlers: Handlers{
			Backtest:   backtestHandler,
			Daily:      dailyHandler,
			Financials: financialsHandler,
			Migration:  migrationHandler,
			Strategy:   strategyHandler,
		},
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.Storage.Database == nil {
		return nil
	}
	return r.Storage.Database.Close()
}
