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

func TestStoreDelegatesToCompositionRepository(t *testing.T) {
	repository := &fakeCompositionRepository{}
	service, err := NewService(&fakeCompositionRouter{}, WithRepository(repository))
	require.NoError(t, err)

	aggregate := compositionrole.Composition{
		Source: compositionrole.SourceRef{
			Provider:  provider.ProviderKIS,
			Group:     provider.GroupKISDomesticStockQuotation,
			Operation: provider.OperationKISETFComponentStockPrice,
		},
		Subject: compositionrole.InstrumentRef{
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
		},
		AsOfDate:     "2026-05-17",
		ObservedAtMS: 1779010800000,
	}
	result, err := service.Store(context.Background(), StoreRequest{Composition: aggregate})

	require.NoError(t, err)
	require.Equal(t, "069500", repository.gotComposition.Subject.Symbol)
	require.Equal(t, 1, result.CompositionsStored)
}

func TestGetDelegatesToCompositionRepository(t *testing.T) {
	repository := &fakeCompositionRepository{
		composition: compositionrole.Composition{
			Subject: compositionrole.InstrumentRef{Symbol: "069500"},
		},
	}
	service, err := NewService(&fakeCompositionRouter{}, WithRepository(repository))
	require.NoError(t, err)

	result, err := service.Get(context.Background(), Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})

	require.NoError(t, err)
	require.Equal(t, "069500", repository.gotQuery.Symbol)
	require.Equal(t, "069500", result.Subject.Symbol)
}

type fakeCompositionRouter struct {
	lister   compositionrole.Lister
	gotRoute compositionrole.RouteInput
}

func (r *fakeCompositionRouter) RouteComposition(_ context.Context, input compositionrole.RouteInput) (compositionrole.Lister, error) {
	r.gotRoute = input
	return r.lister, nil
}

type fakeCompositionRepository struct {
	gotComposition compositionrole.Composition
	gotQuery       Query
	composition    compositionrole.Composition
}

func (r *fakeCompositionRepository) UpsertComposition(_ context.Context, aggregate compositionrole.Composition) (WriteResult, error) {
	r.gotComposition = aggregate
	return WriteResult{
		RowsAffected:       1 + len(aggregate.Members),
		CompositionsStored: 1,
		MembersStored:      len(aggregate.Members),
	}, nil
}

func (r *fakeCompositionRepository) GetComposition(_ context.Context, query Query) (compositionrole.Composition, error) {
	r.gotQuery = query
	return r.composition, nil
}
