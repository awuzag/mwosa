package instrument

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type RouteInput struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Group          provider.GroupID
	Operation      provider.OperationID
	Symbol         string
}

type RoutePlan struct {
	Candidates []RouteCandidate
}

type RouteCandidate struct {
	Provider provider.Identity
	Group    provider.GroupID
	Searcher Searcher
	Profile  Profile
	Reason   string
}

type Router interface {
	RouteInstrumentSearch(ctx context.Context, input RouteInput) (Searcher, error)
	PlanInstrumentSearch(ctx context.Context, input RouteInput) (RoutePlan, error)
}

type coreRouter interface {
	Route(context.Context, provider.RouteInput) (provider.RouteCandidate, error)
	Plan(context.Context, provider.RouteInput) (provider.RoutePlan, error)
}

type routeAdapter struct {
	router coreRouter
}

func NewRouter(router coreRouter) Router {
	return routeAdapter{router: router}
}

func (r routeAdapter) RouteInstrumentSearch(ctx context.Context, input RouteInput) (Searcher, error) {
	errb := oops.In("instrument_router").With("provider", input.ProviderID, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol)
	candidate, err := r.router.Route(ctx, toCoreRouteInput(input))
	if err != nil {
		return nil, errb.Wrap(err)
	}
	searcher, ok := candidate.Impl.(Searcher)
	if !ok {
		return nil, errb.With("provider", candidate.Provider.ID).New("routed instrument implementation does not satisfy Searcher")
	}
	return searcher, nil
}

func (r routeAdapter) PlanInstrumentSearch(ctx context.Context, input RouteInput) (RoutePlan, error) {
	errb := oops.In("instrument_router").With("provider", input.ProviderID, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol)
	plan, err := r.router.Plan(ctx, toCoreRouteInput(input))
	if err != nil {
		return RoutePlan{}, errb.Wrap(err)
	}
	candidates := make([]RouteCandidate, 0, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		searcher, ok := candidate.Impl.(Searcher)
		if !ok {
			return RoutePlan{}, errb.With("provider", candidate.Provider.ID).New("routed instrument implementation does not satisfy Searcher")
		}
		candidates = append(candidates, RouteCandidate{
			Provider: candidate.Provider,
			Group:    candidate.Profile.Group,
			Searcher: searcher,
			Profile:  searcher.InstrumentSearchProfile(),
			Reason:   candidate.Reason,
		})
	}
	return RoutePlan{Candidates: candidates}, nil
}

func toCoreRouteInput(input RouteInput) provider.RouteInput {
	return provider.RouteInput{
		Role:           provider.RoleInstrument,
		ProviderID:     input.ProviderID,
		PreferProvider: input.PreferProvider,
		Market:         input.Market,
		SecurityType:   input.SecurityType,
		Group:          input.Group,
		Operation:      input.Operation,
		Symbol:         input.Symbol,
	}
}
