package app

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateRawFetcherDispatchesRegisteredRawProvider(t *testing.T) {
	registry := provider.NewRegistry()
	rawProvider := &fakeAggregateRawProvider{
		Identity: provider.Identity{ID: provider.ProviderKIS, DisplayName: "KIS"},
	}
	require.NoError(t, registry.RegisterProvider(rawProvider))

	fetcher := newAggregateRawFetcher(ProviderRuntime{Registry: registry})
	require.NotNil(t, fetcher)

	result, err := fetcher.FetchRaw(context.Background(), aggregateservice.RawFetchRequest{
		Provider:  "kis",
		Operation: "inquire-daily-itemchartprice",
		Input:     map[string]string{"FID_INPUT_ISCD": "005930"},
		Context:   map[string]any{"symbol": "005930"},
	})
	require.NoError(t, err)

	assert.Equal(t, "kis", result.Provider)
	assert.Equal(t, "quote", result.Group)
	assert.Equal(t, "inquire-daily-itemchartprice", result.Operation)
	assert.Equal(t, "/raw", result.Endpoint)
	assert.Equal(t, 1, result.RowCount)
	assert.Equal(t, "2026-07-01", result.BaseDate)
	assert.Equal(t, provider.OperationID("inquire-daily-itemchartprice"), rawProvider.lastInput.OperationID)
	assert.Equal(t, "005930", rawProvider.lastInput.Input["FID_INPUT_ISCD"])
	assert.Equal(t, "005930", rawProvider.lastInput.Context["symbol"])
}

func TestAggregateRawFetcherReturnsUnsupportedForProviderWithoutRawAdapter(t *testing.T) {
	registry := provider.NewRegistry()
	require.NoError(t, registry.RegisterProvider(&fakeAggregateIdentityProvider{
		Identity: provider.Identity{ID: provider.ProviderDataGo, DisplayName: "DataGo"},
	}))

	fetcher := newAggregateRawFetcher(ProviderRuntime{Registry: registry})
	require.NotNil(t, fetcher)

	_, err := fetcher.FetchRaw(context.Background(), aggregateservice.RawFetchRequest{
		Provider:  "datago",
		Operation: "getETFPriceInfo",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider capability")
	assert.Contains(t, err.Error(), "provider=datago")
	assert.Contains(t, err.Error(), "operation=getETFPriceInfo")
	assert.Contains(t, err.Error(), "provider does not implement provider_raw adapter")
}

func TestAggregateRawFetcherReturnsUnsupportedForUnregisteredProvider(t *testing.T) {
	fetcher := newAggregateRawFetcher(ProviderRuntime{Registry: provider.NewRegistry()})
	require.NotNil(t, fetcher)

	_, err := fetcher.FetchRaw(context.Background(), aggregateservice.RawFetchRequest{
		Provider:  "missing",
		Operation: "raw-op",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "provider=missing"), err.Error())
	assert.True(t, strings.Contains(err.Error(), "provider is not registered"), err.Error())
}

type fakeAggregateRawProvider struct {
	provider.Identity

	lastInput provider.RawFetchInput
}

func (p *fakeAggregateRawProvider) RoleRegistrations() []provider.RoleRegistration {
	return fakeAggregateRoleRegistrations(p)
}

func (p *fakeAggregateRawProvider) FetchProviderRaw(_ context.Context, input provider.RawFetchInput) (provider.RawFetchResult, error) {
	p.lastInput = input
	return provider.RawFetchResult{
		Provider:  p.ID,
		Group:     provider.GroupKISQuote,
		Operation: input.OperationID,
		Endpoint:  "/raw",
		Response:  map[string]any{"ok": true},
		RowCount:  1,
		BaseDate:  "2026-07-01",
	}, nil
}

type fakeAggregateIdentityProvider struct {
	provider.Identity
}

func (p *fakeAggregateIdentityProvider) RoleRegistrations() []provider.RoleRegistration {
	return fakeAggregateRoleRegistrations(p)
}

func fakeAggregateRoleRegistrations(identity provider.IdentityProvider) []provider.RoleRegistration {
	return []provider.RoleRegistration{
		{
			Profile: provider.RoleProfile{
				Role: provider.RoleCompanyRegistry,
				Compatibility: provider.Compatibility{
					DataLatency: provider.DataLatencyHistorical,
				},
			},
			Impl: identity,
		},
	}
}
