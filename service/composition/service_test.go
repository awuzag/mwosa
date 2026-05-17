package composition

import (
	"context"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	compositionrole "github.com/ev3rlit/mwosa/providers/core/composition"
	"github.com/stretchr/testify/require"
)

func TestListRoutesAndFetchesConstituents(t *testing.T) {
	var gotList compositionrole.ListInput
	lister := compositionrole.NewList(compositionrole.Profile{}, func(_ context.Context, input compositionrole.ListInput) (compositionrole.ListResult, error) {
		gotList = input
		return compositionrole.ListResult{
			Composition: compositionrole.Composition{
				Subject: compositionrole.InstrumentRef{Symbol: input.Symbol},
				Members: []compositionrole.CompositionMember{
					{Instrument: compositionrole.InstrumentRef{Symbol: "005930"}},
				},
			},
		}, nil
	})
	router := &fakeCompositionRouter{lister: lister}
	service, err := NewService(router)
	require.NoError(t, err)

	result, err := service.List(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		Limit:        10,
	})

	require.NoError(t, err)
	require.Equal(t, provider.ProviderID("fake"), router.gotRoute.ProviderID)
	require.Equal(t, 10, gotList.Limit)
	require.Len(t, result.Composition.Members, 1)
}

type fakeCompositionRouter struct {
	lister   compositionrole.Lister
	gotRoute compositionrole.RouteInput
}

func (r *fakeCompositionRouter) RouteComposition(_ context.Context, input compositionrole.RouteInput) (compositionrole.Lister, error) {
	r.gotRoute = input
	return r.lister, nil
}
