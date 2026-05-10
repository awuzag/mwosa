package app

import (
	"github.com/ev3rlit/mwosa/app/handler"
	"github.com/ev3rlit/mwosa/providers/builtin"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/intradaybar"
	"github.com/ev3rlit/mwosa/providers/core/orderbook"
	"github.com/ev3rlit/mwosa/providers/core/quote"
	"github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/ev3rlit/mwosa/service/daily"
	financialsservice "github.com/ev3rlit/mwosa/service/financials"
	instrumentservice "github.com/ev3rlit/mwosa/service/instrument"
	intradayservice "github.com/ev3rlit/mwosa/service/intraday"
	orderbookservice "github.com/ev3rlit/mwosa/service/orderbook"
	providerservice "github.com/ev3rlit/mwosa/service/providers"
	quoteservice "github.com/ev3rlit/mwosa/service/quote"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	tradesservice "github.com/ev3rlit/mwosa/service/trades"
	"github.com/ev3rlit/mwosa/storage"
	dailybarstorage "github.com/ev3rlit/mwosa/storage/dailybar"
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
	Database   *storage.Database
	DailyBars  DailyBarStorage
	Strategies strategyservice.Repository
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
	Intraday    intradaybar.Router
	Orderbooks  orderbook.Router
	Trades      trades.Router
}

type ServiceRuntime struct {
	Daily       DailyServices
	Financials  financialsservice.Service
	Instruments instrumentservice.Service
	Intraday    intradayservice.Service
	Orderbooks  orderbookservice.Service
	Providers   providerservice.Service
	Quotes      quoteservice.Service
	Strategy    strategyservice.Service
	Trades      tradesservice.Service
}

type Handlers struct {
	Daily       handler.Daily
	Financials  handler.Financials
	Instruments handler.Instrument
	Intraday    handler.Intraday
	Orderbooks  handler.Orderbook
	Quotes      handler.Quote
	Strategy    handler.Strategy
	Trades      handler.Trades
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
		Intraday:    intradaybar.NewRouter(coreRouter),
		Orderbooks:  orderbook.NewRouter(coreRouter),
		Trades:      trades.NewRouter(coreRouter),
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
	instrumentService, err := instrumentservice.NewService(providerRuntime.Instruments)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create instrument service"),
			database.Close(),
		)
	}
	quoteService, err := quoteservice.NewService(providerRuntime.Quotes)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create quote service"),
			database.Close(),
		)
	}
	intradayService, err := intradayservice.NewService(providerRuntime.Intraday)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create intraday service"),
			database.Close(),
		)
	}
	orderbookService, err := orderbookservice.NewService(providerRuntime.Orderbooks)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create orderbook service"),
			database.Close(),
		)
	}
	tradesService, err := tradesservice.NewService(providerRuntime.Trades)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create trades service"),
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
	dailyHandler := handler.NewDaily(dailyReader, dailyCollector)
	financialsHandler := handler.NewFinancials(financialsService)
	instrumentHandler := handler.NewInstrument(instrumentService)
	quoteHandler := handler.NewQuote(quoteService)
	intradayHandler := handler.NewIntraday(intradayService)
	orderbookHandler := handler.NewOrderbook(orderbookService)
	tradesHandler := handler.NewTrades(tradesService)
	strategyHandler := handler.NewStrategy(strategyService)

	return &Runtime{
		Storage: StorageRuntime{
			Database: database,
			DailyBars: DailyBarStorage{
				Reader: reader,
				Writer: writer,
			},
			Strategies: strategyRepository,
		},
		Providers: providerRuntime,
		Services: ServiceRuntime{
			Daily: DailyServices{
				Reader:    dailyReader,
				Collector: dailyCollector,
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
			Daily:       dailyHandler,
			Financials:  financialsHandler,
			Instruments: instrumentHandler,
			Intraday:    intradayHandler,
			Orderbooks:  orderbookHandler,
			Quotes:      quoteHandler,
			Strategy:    strategyHandler,
			Trades:      tradesHandler,
		},
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.Storage.Database == nil {
		return nil
	}
	return r.Storage.Database.Close()
}
