package app

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
)

type aggregateRawFetcher struct {
	registry *provider.Registry
}

func newAggregateRawFetcher(runtime ProviderRuntime) aggregateservice.RawFetcher {
	if runtime.Registry == nil {
		return nil
	}
	return aggregateRawFetcher{registry: runtime.Registry}
}

func (f aggregateRawFetcher) FetchRaw(ctx context.Context, req aggregateservice.RawFetchRequest) (aggregateservice.RawFetchResult, error) {
	providerID := provider.ProviderID(req.Provider)
	instance, ok := f.registry.Provider(providerID)
	if !ok {
		return aggregateservice.RawFetchResult{}, provider.NewUnsupported(provider.UnsupportedError{
			Capability:  provider.Role("provider_raw"),
			ProviderID:  providerID,
			OperationID: provider.OperationID(req.Operation),
			Reason:      "provider is not registered",
		})
	}
	rawFetcher, ok := instance.(provider.RawFetcher)
	if !ok {
		return aggregateservice.RawFetchResult{}, provider.NewUnsupported(provider.UnsupportedError{
			Capability:  provider.Role("provider_raw"),
			ProviderID:  providerID,
			OperationID: provider.OperationID(req.Operation),
			Reason:      "provider does not implement provider_raw adapter",
		})
	}
	result, err := rawFetcher.FetchProviderRaw(ctx, provider.RawFetchInput{
		OperationID: provider.OperationID(req.Operation),
		Input:       req.Input,
		Context:     req.Context,
	})
	if err != nil {
		return aggregateservice.RawFetchResult{}, err
	}
	return aggregateservice.RawFetchResult{
		Provider:  string(result.Provider),
		Group:     string(result.Group),
		Operation: string(result.Operation),
		Endpoint:  result.Endpoint,
		Response:  result.Response,
		RowCount:  result.RowCount,
		BaseDate:  result.BaseDate,
	}, nil
}
