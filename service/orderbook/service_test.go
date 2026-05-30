package orderbook

import (
	"context"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	orderbookrole "github.com/awuzag/mwosa/providers/core/orderbook"
	"github.com/stretchr/testify/require"
)

func TestGetRoutesAndFetchesOrderbookSnapshot(t *testing.T) {
	var gotSnapshot orderbookrole.SnapshotInput
	snapshotter := orderbookrole.NewSnapshot(orderbookrole.Profile{}, func(_ context.Context, input orderbookrole.SnapshotInput) (orderbookrole.SnapshotResult, error) {
		gotSnapshot = input
		return orderbookrole.SnapshotResult{
			Snapshot: orderbookrole.Snapshot{Symbol: input.Symbol},
		}, nil
	})
	router := &fakeOrderbookRouter{snapshotter: snapshotter}
	service, err := NewService(router)
	require.NoError(t, err)

	result, err := service.Get(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, provider.ProviderID("fake"), router.gotRoute.ProviderID)
	require.Equal(t, "005930", gotSnapshot.Symbol)
	require.Equal(t, "005930", result.Snapshot.Symbol)
}

type fakeOrderbookRouter struct {
	snapshotter orderbookrole.Snapshotter
	gotRoute    orderbookrole.RouteInput
}

func (r *fakeOrderbookRouter) RouteOrderbookSnapshot(_ context.Context, input orderbookrole.RouteInput) (orderbookrole.Snapshotter, error) {
	r.gotRoute = input
	return r.snapshotter, nil
}
