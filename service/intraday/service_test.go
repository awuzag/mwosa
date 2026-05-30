package intraday

import (
	"context"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/intradaybar"
	"github.com/stretchr/testify/require"
)

func TestGetRoutesAndFetchesIntradayBars(t *testing.T) {
	var gotFetch intradaybar.FetchInput
	fetcher := intradaybar.NewFetch(intradaybar.Profile{}, func(_ context.Context, input intradaybar.FetchInput) (intradaybar.FetchResult, error) {
		gotFetch = input
		return intradaybar.FetchResult{
			Bars: []intradaybar.Bar{{Symbol: input.Symbol, Time: "14:12:00"}},
		}, nil
	})
	router := &fakeIntradayRouter{fetcher: fetcher}
	service, err := NewService(router)
	require.NoError(t, err)

	result, err := service.Get(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		At:           "141200",
		Limit:        1,
	})

	require.NoError(t, err)
	require.Equal(t, provider.ProviderID("fake"), router.gotRoute.ProviderID)
	require.Equal(t, "005930", router.gotRoute.Symbol)
	require.Equal(t, "141200", gotFetch.At)
	require.Len(t, result.Bars, 1)
}

type fakeIntradayRouter struct {
	fetcher  intradaybar.Fetcher
	gotRoute intradaybar.RouteInput
}

func (r *fakeIntradayRouter) RouteIntradayBars(_ context.Context, input intradaybar.RouteInput) (intradaybar.Fetcher, error) {
	r.gotRoute = input
	return r.fetcher, nil
}
