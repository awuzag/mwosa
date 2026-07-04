package app

import (
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
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
	"github.com/samber/oops"
)

func newServiceRuntime(opts Options, storageRuntime StorageRuntime, providerRuntime ProviderRuntime) (ServiceRuntime, error) {
	errb := oops.In("app_service_runtime")

	dailyBars := storageRuntime.DailyBars
	indexBars := storageRuntime.IndexBars
	macroRepositories := storageRuntime.Macro
	instrumentRepository := storageRuntime.Instruments
	compositionRepository := storageRuntime.Compositions
	strategyRepository := storageRuntime.Strategies
	fundamentalsRepository := storageRuntime.Fundamentals
	backtestStrategyRepository := storageRuntime.BacktestStrategies
	aggregateRepository := storageRuntime.Aggregates

	dailyReader, err := daily.NewReadService(dailyBars.Reader)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create daily read service")
	}
	dailyCollector, err := daily.NewService(dailyBars.Reader, dailyBars.Writer, providerRuntime.DailyBars)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create daily collect service")
	}
	indexReaderService, err := indexservice.NewReadService(indexBars.Reader)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create index read service")
	}
	indexCollector, err := indexservice.NewService(indexBars.Reader, indexBars.Writer, providerRuntime.IndexBars)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create index collect service")
	}
	macroReaderService, err := macroservice.NewReadService(macroRepositories.Reader)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create macro read service")
	}
	macroCollector, err := macroservice.NewService(macroRepositories.Reader, macroRepositories.Writer, providerRuntime.Macro)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create macro collect service")
	}
	providersService, err := providerservice.NewService(providerRuntime.Registry)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create providers service")
	}
	financialsService, err := financialsservice.NewService(providerRuntime.Financials)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create financials service")
	}
	instrumentService, err := instrumentservice.NewService(providerRuntime.Instruments, instrumentservice.WithRepository(instrumentRepository))
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create instrument service")
	}
	quoteService, err := quoteservice.NewService(providerRuntime.Quotes)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create quote service")
	}
	intradayService, err := intradayservice.NewService(providerRuntime.Intraday)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create intraday service")
	}
	orderbookService, err := orderbookservice.NewService(providerRuntime.Orderbooks)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create orderbook service")
	}
	tradesService, err := tradesservice.NewService(providerRuntime.Trades)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create trades service")
	}
	compositionService, err := compositionservice.NewService(providerRuntime.Compositions, compositionservice.WithRepository(compositionRepository))
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create composition service")
	}
	datasetReader, err := strategyservice.NewDailyBarDatasetReaderWithFundamentals(dailyBars.Reader, fundamentalsRepository, opts.Market)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create strategy dataset reader")
	}
	strategyService, err := strategyservice.NewService(strategyRepository, datasetReader)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create strategy service")
	}
	universeRunner, err := universeservice.NewRunner(dailyBars.Reader, strategyRepository, strategyService)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create universe service")
	}
	strategyService.SetPipelineExecutor(universeRunner)
	backtestReader, ok := dailyBars.Reader.(backtestservice.DailyBarRepository)
	if !ok {
		return ServiceRuntime{}, errb.New("daily bar repository does not support streaming reads")
	}
	backtestService, err := backtestservice.NewServiceWithUniverseSources(backtestReader, backtestStrategyRepository, strategyRepository, strategyService)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create backtest service")
	}
	aggregateOptions := []aggregateservice.Option{}
	if storageRuntime.MongoDB != nil {
		executorOptions := []aggregateservice.MongoExecutorOption{}
		if providerFetcher := newAggregateProviderFetcher(providerRuntime); providerFetcher != nil {
			executorOptions = append(executorOptions, aggregateservice.WithProviderFetcher(providerFetcher))
		}
		if rawFetcher := newAggregateRawFetcher(providerRuntime); rawFetcher != nil {
			executorOptions = append(executorOptions, aggregateservice.WithRawFetcher(rawFetcher))
		}
		aggregateExecutor, err := aggregateservice.NewMongoExecutor(storageRuntime.MongoDB.Database(), executorOptions...)
		if err != nil {
			return ServiceRuntime{}, errb.Wrapf(err, "create aggregate executor")
		}
		aggregateOptions = append(aggregateOptions, aggregateservice.WithExecutor(aggregateExecutor))
	}
	aggregateService, err := aggregateservice.NewService(aggregateRepository, aggregateOptions...)
	if err != nil {
		return ServiceRuntime{}, errb.Wrapf(err, "create aggregate service")
	}

	return ServiceRuntime{
		Aggregates:   aggregateService,
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
		Migration:   storageRuntime.Migration,
		Universe:    universeRunner,
	}, nil
}
