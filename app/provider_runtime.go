package app

import (
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
	"github.com/awuzag/mwosa/storage/providerauth"
	"github.com/samber/oops"
)

func newProviderRuntime(opts Options, builders []provider.ProviderBuilder, storageRuntime StorageRuntime) (ProviderRuntime, error) {
	tokenCache, err := providerauth.NewRepository(storageRuntime.ProviderAuthDatabase)
	if err != nil {
		return ProviderRuntime{}, oops.In("app_provider_runtime").Wrapf(err, "create provider auth token repository")
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
			return ProviderRuntime{}, oops.In("app_provider_runtime").Wrapf(err, "register configured providers")
		}
	}

	coreRouter := provider.NewRouter(registry)
	return ProviderRuntime{
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
	}, nil
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
