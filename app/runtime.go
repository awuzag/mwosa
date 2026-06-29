package app

import (
	"context"
	"path/filepath"
	"time"

	appconfig "github.com/awuzag/mwosa/app/config"
	"github.com/awuzag/mwosa/app/handler"
	migrationcore "github.com/awuzag/mwosa/migration"
	"github.com/awuzag/mwosa/providers/builtin"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/composition"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/awuzag/mwosa/providers/core/instrument"
	"github.com/awuzag/mwosa/providers/core/intradaybar"
	"github.com/awuzag/mwosa/providers/core/macro"
	"github.com/awuzag/mwosa/providers/core/orderbook"
	"github.com/awuzag/mwosa/providers/core/quote"
	"github.com/awuzag/mwosa/providers/core/trades"
	kisprovider "github.com/awuzag/mwosa/providers/kis"
	backtestservice "github.com/awuzag/mwosa/service/backtest"
	compositionservice "github.com/awuzag/mwosa/service/composition"
	"github.com/awuzag/mwosa/service/daily"
	financialsservice "github.com/awuzag/mwosa/service/financials"
	indexservice "github.com/awuzag/mwosa/service/index"
	instrumentservice "github.com/awuzag/mwosa/service/instrument"
	intradayservice "github.com/awuzag/mwosa/service/intraday"
	macroservice "github.com/awuzag/mwosa/service/macro"
	orderbookservice "github.com/awuzag/mwosa/service/orderbook"
	providerservice "github.com/awuzag/mwosa/service/providers"
	quoteservice "github.com/awuzag/mwosa/service/quote"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	tradesservice "github.com/awuzag/mwosa/service/trades"
	universeservice "github.com/awuzag/mwosa/service/universe"
	"github.com/awuzag/mwosa/storage"
	backteststorage "github.com/awuzag/mwosa/storage/backtest"
	compositionstorage "github.com/awuzag/mwosa/storage/composition"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
	indexbarstorage "github.com/awuzag/mwosa/storage/indexbar"
	instrumentstorage "github.com/awuzag/mwosa/storage/instrument"
	macrostorage "github.com/awuzag/mwosa/storage/macro"
	migrationstorage "github.com/awuzag/mwosa/storage/migration"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/awuzag/mwosa/storage/providerauth"
	strategystorage "github.com/awuzag/mwosa/storage/strategy"
	strategyfundamentalsstorage "github.com/awuzag/mwosa/storage/strategyfundamentals"
	"github.com/samber/oops"
)

type Options struct {
	DatabaseBackend      string
	Database             string
	DatabaseURL          string
	ProviderAuthDatabase string
	Market               provider.Market
	ProviderID           provider.ProviderID
	PreferProvider       provider.ProviderID
	ProviderConfig       provider.Config
	ActivateProviders    bool
}

type Runtime struct {
	Storage   StorageRuntime
	Providers ProviderRuntime
	Services  ServiceRuntime
	Handlers  Handlers
}

type StorageRuntime struct {
	Database             *storage.Database
	MongoDB              *storagemongodb.Runtime
	ProviderAuthDatabase *providerauth.Database
	Compositions         compositionservice.Repository
	DailyBars            DailyBarStorage
	IndexBars            IndexBarStorage
	Macro                MacroStorage
	Instruments          instrumentservice.Repository
	Migrations           migrationcore.Store
	Strategies           strategyservice.Repository
	BacktestStrategies   backtestservice.StrategyRepository
}

type DailyBarStorage struct {
	Reader daily.ReadRepository
	Writer daily.WriteRepository
}

type IndexBarStorage struct {
	Reader indexservice.ReadRepository
	Writer indexservice.WriteRepository
}

type MacroStorage struct {
	Reader macroservice.ReadRepository
	Writer macroservice.WriteRepository
}

type ProviderRuntime struct {
	Registry     *provider.Registry
	Router       *provider.Router
	Compositions composition.Router
	DailyBars    dailybar.Router
	IndexBars    indexbar.Router
	Financials   financials.Router
	Macro        macro.Router
	Quotes       quote.Router
	Instruments  instrument.Router
	Intraday     intradaybar.Router
	Orderbooks   orderbook.Router
	Trades       trades.Router
}

type ServiceRuntime struct {
	Backtest     backtestservice.Service
	Compositions compositionservice.Service
	Daily        DailyServices
	Index        IndexServices
	Macro        MacroServices
	Financials   financialsservice.Service
	Instruments  instrumentservice.Service
	Intraday     intradayservice.Service
	Orderbooks   orderbookservice.Service
	Providers    providerservice.Service
	Quotes       quoteservice.Service
	Strategy     strategyservice.Service
	Trades       tradesservice.Service
}

type Handlers struct {
	Backtest     handler.Backtest
	Compositions handler.Composition
	Daily        handler.Daily
	Index        handler.Index
	Macro        handler.Macro
	Financials   handler.Financials
	Instruments  handler.Instrument
	Intraday     handler.Intraday
	Migration    handler.Migration
	Orderbooks   handler.Orderbook
	Quotes       handler.Quote
	Strategy     handler.Strategy
	Trades       handler.Trades
}

type DailyServices struct {
	Reader    daily.ReadService
	Collector daily.Service
}

type IndexServices struct {
	Reader    indexservice.ReadService
	Collector indexservice.Service
}

type MacroServices struct {
	Reader    macroservice.ReadService
	Collector macroservice.Service
}

func NewRuntime(opts Options) (*Runtime, error) {
	return NewRuntimeWithProviderBuilders(opts, builtin.Builders()...)
}

func NewRuntimeWithProviderBuilders(opts Options, builders ...provider.ProviderBuilder) (*Runtime, error) {
	errb := oops.In("app_runtime")

	database := storageDatabaseFromOptions(opts)
	mongoRuntime, err := mongodbRuntimeFromOptions(opts)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create mongodb runtime"),
			database.Close(),
		)
	}
	providerAuthDatabase := providerauth.NewDatabase(providerAuthDatabasePath(opts))
	closeStorage := func() error {
		return oops.Join(
			database.Close(),
			closeMongoDBRuntime(mongoRuntime),
			providerAuthDatabase.Close(),
		)
	}
	tokenCache, err := providerauth.NewRepository(providerAuthDatabase)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create provider auth token repository"),
			closeStorage(),
		)
	}
	reader, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily bar repositories"),
			closeStorage(),
		)
	}
	indexReader, indexWriter, err := indexbarstorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create index bar repository"),
			closeStorage(),
		)
	}
	macroReader, macroWriter, err := macrostorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create macro repository"),
			closeStorage(),
		)
	}
	instrumentRepository, err := instrumentRepositoryFromOptions(database, mongoRuntime)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create instrument repository"),
			closeStorage(),
		)
	}
	compositionRepository, err := compositionstorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create composition repository"),
			closeStorage(),
		)
	}
	strategyRepository, err := strategystorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy repository"),
			closeStorage(),
		)
	}
	backtestStrategyRepository, err := backteststorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create backtest strategy repository"),
			closeStorage(),
		)
	}
	migrationStore, err := migrationstorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create migration repository"),
			closeStorage(),
		)
	}
	dailyBarMigration, err := migrationstorage.NewDailyBarV1ToV2Executor(database, writer)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily bar migration"),
			closeStorage(),
		)
	}
	dailyBarExtensionCleanup, err := migrationstorage.NewDailyBarV2ExtensionCleanupExecutor(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily bar extension cleanup migration"),
			closeStorage(),
		)
	}
	migrationRunner, err := migrationcore.NewRunner(migrationStore, []migrationcore.Definition{
		migrationstorage.NewDailyBarV1ToV2Definition(dailyBarMigration),
		migrationstorage.NewDailyBarV2ExtensionCleanupDefinition(dailyBarExtensionCleanup),
	})
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create migration runner"),
			closeStorage(),
		)
	}

	registry := provider.NewRegistry()
	if opts.ActivateProviders {
		config := opts.ProviderConfig
		if config == nil {
			config = provider.ConfigFromEnv()
		}
		builders = withKISTokenCache(builders, tokenCache)
		if err := registry.RegisterConfigured(provider.RegisterOptions{
			ProviderID:     opts.ProviderID,
			PreferProvider: opts.PreferProvider,
		}, config, builders...); err != nil {
			return nil, oops.Join(
				errb.Wrapf(err, "register configured providers"),
				closeStorage(),
			)
		}
	}

	coreRouter := provider.NewRouter(registry)
	providerRuntime := ProviderRuntime{
		Registry:     registry,
		Router:       coreRouter,
		Compositions: composition.NewRouter(coreRouter),
		DailyBars:    dailybar.NewRouter(coreRouter),
		IndexBars:    indexbar.NewRouter(coreRouter),
		Financials:   financials.NewRouter(coreRouter),
		Macro:        macro.NewRouter(coreRouter),
		Quotes:       quote.NewRouter(coreRouter),
		Instruments:  instrument.NewRouter(coreRouter),
		Intraday:     intradaybar.NewRouter(coreRouter),
		Orderbooks:   orderbook.NewRouter(coreRouter),
		Trades:       trades.NewRouter(coreRouter),
	}

	dailyReader, err := daily.NewReadService(reader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily read service"),
			closeStorage(),
		)
	}
	dailyCollector, err := daily.NewService(reader, writer, providerRuntime.DailyBars)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create daily collect service"),
			closeStorage(),
		)
	}
	indexReaderService, err := indexservice.NewReadService(indexReader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create index read service"),
			closeStorage(),
		)
	}
	indexCollector, err := indexservice.NewService(indexReader, indexWriter, providerRuntime.IndexBars)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create index collect service"),
			closeStorage(),
		)
	}
	macroReaderService, err := macroservice.NewReadService(macroReader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create macro read service"),
			closeStorage(),
		)
	}
	macroCollector, err := macroservice.NewService(macroReader, macroWriter, providerRuntime.Macro)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create macro collect service"),
			closeStorage(),
		)
	}
	providersService, err := providerservice.NewService(registry)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create providers service"),
			closeStorage(),
		)
	}
	financialsService, err := financialsservice.NewService(providerRuntime.Financials)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create financials service"),
			closeStorage(),
		)
	}
	instrumentService, err := instrumentservice.NewService(providerRuntime.Instruments, instrumentservice.WithRepository(instrumentRepository))
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create instrument service"),
			closeStorage(),
		)
	}
	quoteService, err := quoteservice.NewService(providerRuntime.Quotes)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create quote service"),
			closeStorage(),
		)
	}
	intradayService, err := intradayservice.NewService(providerRuntime.Intraday)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create intraday service"),
			closeStorage(),
		)
	}
	orderbookService, err := orderbookservice.NewService(providerRuntime.Orderbooks)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create orderbook service"),
			closeStorage(),
		)
	}
	tradesService, err := tradesservice.NewService(providerRuntime.Trades)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create trades service"),
			closeStorage(),
		)
	}
	compositionService, err := compositionservice.NewService(providerRuntime.Compositions, compositionservice.WithRepository(compositionRepository))
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create composition service"),
			closeStorage(),
		)
	}
	fundamentalsRepository, err := strategyfundamentalsstorage.NewRepository(database)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy fundamentals repository"),
			closeStorage(),
		)
	}
	datasetReader, err := strategyservice.NewDailyBarDatasetReaderWithFundamentals(reader, fundamentalsRepository, opts.Market)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy dataset reader"),
			closeStorage(),
		)
	}
	strategyService, err := strategyservice.NewService(strategyRepository, datasetReader)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create strategy service"),
			closeStorage(),
		)
	}
	universeRunner, err := universeservice.NewRunner(reader, strategyRepository, strategyService)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create universe service"),
			closeStorage(),
		)
	}
	strategyService.SetPipelineExecutor(universeRunner)
	backtestReader, ok := reader.(backtestservice.DailyBarRepository)
	if !ok {
		return nil, oops.Join(
			errb.New("daily bar repository does not support streaming reads"),
			closeStorage(),
		)
	}
	backtestService, err := backtestservice.NewServiceWithUniverseSources(backtestReader, backtestStrategyRepository, strategyRepository, strategyService)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create backtest service"),
			closeStorage(),
		)
	}
	backtestHandler := handler.NewBacktest(backtestService)
	dailyHandler := handler.NewDaily(dailyReader, dailyCollector)
	indexHandler := handler.NewIndex(indexReaderService, indexCollector)
	macroHandler := handler.NewMacro(macroReaderService, macroCollector)
	financialsHandler := handler.NewFinancials(financialsService)
	strategyHandler := handler.NewStrategy(strategyService, universeRunner)
	instrumentHandler := handler.NewInstrument(instrumentService)
	quoteHandler := handler.NewQuote(quoteService)
	intradayHandler := handler.NewIntraday(intradayService)
	orderbookHandler := handler.NewOrderbook(orderbookService)
	tradesHandler := handler.NewTrades(tradesService)
	compositionHandler := handler.NewComposition(compositionService)
	migrationHandler := handler.NewMigration(migrationRunner)

	return &Runtime{
		Storage: StorageRuntime{
			Database:             database,
			MongoDB:              mongoRuntime,
			ProviderAuthDatabase: providerAuthDatabase,
			Compositions:         compositionRepository,
			DailyBars: DailyBarStorage{
				Reader: reader,
				Writer: writer,
			},
			IndexBars: IndexBarStorage{
				Reader: indexReader,
				Writer: indexWriter,
			},
			Macro: MacroStorage{
				Reader: macroReader,
				Writer: macroWriter,
			},
			Instruments:        instrumentRepository,
			Migrations:         migrationStore,
			Strategies:         strategyRepository,
			BacktestStrategies: backtestStrategyRepository,
		},
		Providers: providerRuntime,
		Services: ServiceRuntime{
			Backtest:     backtestService,
			Compositions: compositionService,
			Daily: DailyServices{
				Reader:    dailyReader,
				Collector: dailyCollector,
			},
			Index: IndexServices{
				Reader:    indexReaderService,
				Collector: indexCollector,
			},
			Macro: MacroServices{
				Reader:    macroReaderService,
				Collector: macroCollector,
			},
			Financials:  financialsService,
			Instruments: instrumentService,
			Intraday:    intradayService,
			Orderbooks:  orderbookService,
			Providers:   providersService,
			Quotes:      quoteService,
			Strategy:    strategyService,
			Trades:      tradesService,
		},
		Handlers: Handlers{
			Backtest:     backtestHandler,
			Compositions: compositionHandler,
			Daily:        dailyHandler,
			Index:        indexHandler,
			Macro:        macroHandler,
			Financials:   financialsHandler,
			Instruments:  instrumentHandler,
			Intraday:     intradayHandler,
			Migration:    migrationHandler,
			Orderbooks:   orderbookHandler,
			Quotes:       quoteHandler,
			Strategy:     strategyHandler,
			Trades:       tradesHandler,
		},
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	return oops.Join(
		r.Storage.Database.Close(),
		closeMongoDBRuntime(r.Storage.MongoDB),
		r.Storage.ProviderAuthDatabase.Close(),
	)
}

func providerAuthDatabasePath(opts Options) string {
	if opts.ProviderAuthDatabase != "" {
		return opts.ProviderAuthDatabase
	}
	return filepath.Join(filepath.Dir(opts.Database), appconfig.ProviderAuthDatabaseFileName)
}

func storageDatabaseFromOptions(opts Options) *storage.Database {
	backend := storage.Backend(opts.DatabaseBackend)
	if backend == "" {
		backend = storage.BackendSQLite
	}
	return storage.NewDatabaseWithConfig(storage.DatabaseConfig{
		Backend: backend,
		Path:    opts.Database,
		URL:     opts.DatabaseURL,
	})
}

func mongodbRuntimeFromOptions(opts Options) (*storagemongodb.Runtime, error) {
	if opts.DatabaseBackend != appconfig.DatabaseBackendMongoDB {
		return nil, nil
	}
	return storagemongodb.NewRuntime(context.Background(), storagemongodb.Config{
		URI: opts.DatabaseURL,
	})
}

func closeMongoDBRuntime(runtime *storagemongodb.Runtime) error {
	if runtime == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runtime.Close(ctx)
}

func instrumentRepositoryFromOptions(database *storage.Database, mongoRuntime *storagemongodb.Runtime) (instrumentservice.Repository, error) {
	if mongoRuntime != nil {
		return instrumentstorage.NewMongoRepository(mongoRuntime.Database())
	}
	return instrumentstorage.NewRepository(database)
}

type kisTokenCacheBuilder interface {
	WithTokenCache(kisprovider.TokenCache) provider.ProviderBuilder
}

func withKISTokenCache(builders []provider.ProviderBuilder, tokenCache kisprovider.TokenCache) []provider.ProviderBuilder {
	if tokenCache == nil {
		return builders
	}
	copied := make([]provider.ProviderBuilder, 0, len(builders))
	for _, builder := range builders {
		if builder != nil && builder.ID() == provider.ProviderKIS {
			if typed, ok := builder.(kisTokenCacheBuilder); ok {
				copied = append(copied, typed.WithTokenCache(tokenCache))
				continue
			}
		}
		copied = append(copied, builder)
	}
	return copied
}
