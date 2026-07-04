package app

import (
	"context"

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
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/awuzag/mwosa/storage/providerauth"
	"github.com/samber/oops"
)

type Options struct {
	DatabaseBackend      string
	Database             string
	DatabaseURL          string
	ProviderAuthDatabase string
	// RepositoryBackends is an internal assembly hook for tests and staged
	// migrations. CLI/config backend selection stays on DatabaseBackend.
	RepositoryBackends map[RepositoryName]string
	Market             provider.Market
	ProviderID         provider.ProviderID
	PreferProvider     provider.ProviderID
	ProviderConfig     provider.Config
	ActivateProviders  bool
}

type Runtime struct {
	Storage   StorageRuntime
	Providers ProviderRuntime
	Services  ServiceRuntime
	Handlers  Handlers
}

type StorageRuntime struct {
	Backend              string
	SQLDatabase          *storage.SQLDatabase
	MongoDB              *storagemongodb.Runtime
	ProviderAuthDatabase *providerauth.Database
	ProviderRaw          ProviderRawRepository         `repository:"providerraw"`
	Aggregates           aggregateservice.Repository   `repository:"aggregate"`
	Compositions         compositionservice.Repository `repository:"composition"`
	DailyBars            DailyBarStorage               `repository:"dailybar"`
	IndexBars            IndexBarStorage               `repository:"indexbar"`
	Macro                MacroStorage                  `repository:"macro"`
	Instruments          instrumentservice.Repository  `repository:"instrument"`
	Migration            migrationcore.Runner
	Strategies           strategyservice.Repository             `repository:"strategy"`
	Fundamentals         strategyservice.FundamentalsRepository `repository:"strategyfundamentals"`
	BacktestStrategies   backtestservice.StrategyRepository     `repository:"backtest"`
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
	Aggregates   aggregateservice.Service
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
	Migration    migrationcore.Runner
	Universe     universeservice.Runner
}

type Handlers struct {
	Backtest     handler.Backtest
	Aggregates   handler.Aggregate
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

func NewRuntime(ctx context.Context, opts Options) (*Runtime, error) {
	return NewRuntimeWithProviderBuilders(ctx, opts, builtin.Builders()...)
}

func NewRuntimeWithProviderBuilders(ctx context.Context, opts Options, builders ...provider.ProviderBuilder) (*Runtime, error) {
	errb := oops.In("app_runtime")

	storageRuntime, err := NewStorageRuntime(ctx, opts)
	if err != nil {
		return nil, errb.Wrapf(err, "create storage runtime")
	}
	providerRuntime, err := newProviderRuntime(opts, builders, storageRuntime)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create provider runtime"),
			storageRuntime.Close(ctx),
		)
	}
	serviceRuntime, err := newServiceRuntime(opts, storageRuntime, providerRuntime)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create service runtime"),
			storageRuntime.Close(ctx),
		)
	}
	handlers, err := newHandlers(serviceRuntime)
	if err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "create handler runtime"),
			storageRuntime.Close(ctx),
		)
	}

	return &Runtime{
		Storage:   storageRuntime,
		Providers: providerRuntime,
		Services:  serviceRuntime,
		Handlers:  handlers,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	return r.Storage.Close(ctx)
}
