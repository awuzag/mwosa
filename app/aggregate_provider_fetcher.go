package app

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	quoterole "github.com/awuzag/mwosa/providers/core/quote"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	"github.com/samber/oops"
)

type aggregateProviderFetcher struct {
	quotes quoterole.Router
}

func newAggregateProviderFetcher(runtime ProviderRuntime) aggregateservice.ProviderFetcher {
	if runtime.Quotes == nil {
		return nil
	}
	return aggregateProviderFetcher{quotes: runtime.Quotes}
}

func (f aggregateProviderFetcher) FetchProvider(ctx context.Context, req aggregateservice.ProviderFetchRequest) (aggregateservice.ProviderFetchResult, error) {
	switch provider.Role(req.Role) {
	case provider.RoleQuote:
		return f.fetchQuote(ctx, req)
	default:
		return aggregateservice.ProviderFetchResult{}, oops.In("aggregate_provider_fetcher").With("role", req.Role).Errorf("unsupported aggregate provider role: %s", req.Role)
	}
}

func (f aggregateProviderFetcher) fetchQuote(ctx context.Context, req aggregateservice.ProviderFetchRequest) (aggregateservice.ProviderFetchResult, error) {
	snapshotter, err := f.quotes.RouteQuoteSnapshot(ctx, quoterole.RouteInput{
		ProviderID:   provider.ProviderID(req.Provider),
		Market:       provider.Market(req.Request["market"]),
		SecurityType: provider.SecurityType(req.Request["security_type"]),
		Symbol:       req.Request["symbol"],
	})
	if err != nil {
		return aggregateservice.ProviderFetchResult{}, err
	}
	result, err := snapshotter.FetchQuoteSnapshot(ctx, quoterole.SnapshotInput{
		Market:       provider.Market(req.Request["market"]),
		SecurityType: provider.SecurityType(req.Request["security_type"]),
		Symbol:       req.Request["symbol"],
	})
	if err != nil {
		return aggregateservice.ProviderFetchResult{}, err
	}
	return aggregateservice.ProviderFetchResult{
		Provider: string(result.Provider.ID),
		Role:     string(provider.RoleQuote),
		Payload: map[string]any{
			"symbol": result.Symbol,
			"price":  result.Price,
		},
	}, nil
}
