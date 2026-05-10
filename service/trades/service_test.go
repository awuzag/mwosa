package trades

import (
	"context"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	tradesrole "github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/stretchr/testify/require"
)

func TestListRoutesAndFetchesMarketTrades(t *testing.T) {
	var gotList tradesrole.ListInput
	lister := tradesrole.NewList(tradesrole.Profile{}, func(_ context.Context, input tradesrole.ListInput) (tradesrole.ListResult, error) {
		gotList = input
		return tradesrole.ListResult{
			Trades: []tradesrole.Trade{{Symbol: input.Symbol, Time: "14:12:00"}},
		}, nil
	})
	router := &fakeTradesRouter{lister: lister}
	service, err := NewService(router)
	require.NoError(t, err)

	result, err := service.List(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		At:           "141200",
		Limit:        1,
	})

	require.NoError(t, err)
	require.Equal(t, provider.ProviderID("fake"), router.gotRoute.ProviderID)
	require.Equal(t, "141200", gotList.At)
	require.Len(t, result.Trades, 1)
}

type fakeTradesRouter struct {
	lister   tradesrole.Lister
	gotRoute tradesrole.RouteInput
}

func (r *fakeTradesRouter) RouteMarketTrades(_ context.Context, input tradesrole.RouteInput) (tradesrole.Lister, error) {
	r.gotRoute = input
	return r.lister, nil
}
