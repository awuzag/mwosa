package indexbar

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type RouteInput struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	Group          provider.GroupID
	Operation      provider.OperationID
	IndexCode      string
}

type RoutePlan struct {
	Candidates []RouteCandidate
}

type RouteCandidate struct {
	Provider provider.Identity
	Group    provider.GroupID
	Fetcher  Fetcher
	Profile  Profile
	Reason   string
}

type Router interface {
	RouteIndexBars(ctx context.Context, input RouteInput) (Fetcher, error)
	PlanIndexBars(ctx context.Context, input RouteInput) (RoutePlan, error)
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

func (r routeAdapter) RouteIndexBars(ctx context.Context, input RouteInput) (Fetcher, error) {
	errb := oops.In("indexbar_router").With("provider", input.ProviderID, "market", input.Market, "index_code", input.IndexCode)
	candidate, err := r.router.Route(ctx, toCoreRouteInput(input))
	if err != nil {
		return nil, errb.Wrap(err)
	}
	fetcher, ok := candidate.Impl.(Fetcher)
	if !ok {
		return nil, errb.With("provider", candidate.Provider.ID).New("routed indexbar implementation does not satisfy Fetcher")
	}
	return fetcher, nil
}

func (r routeAdapter) PlanIndexBars(ctx context.Context, input RouteInput) (RoutePlan, error) {
	errb := oops.In("indexbar_router").With("provider", input.ProviderID, "market", input.Market, "index_code", input.IndexCode)
	plan, err := r.router.Plan(ctx, toCoreRouteInput(input))
	if err != nil {
		return RoutePlan{}, errb.Wrap(err)
	}
	candidates := make([]RouteCandidate, 0, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		fetcher, ok := candidate.Impl.(Fetcher)
		if !ok {
			return RoutePlan{}, errb.With("provider", candidate.Provider.ID).New("routed indexbar implementation does not satisfy Fetcher")
		}
		candidates = append(candidates, RouteCandidate{
			Provider: candidate.Provider,
			Group:    candidate.Profile.Group,
			Fetcher:  fetcher,
			Profile:  fetcher.IndexBarProfile(),
			Reason:   candidate.Reason,
		})
	}
	return RoutePlan{Candidates: candidates}, nil
}

func toCoreRouteInput(input RouteInput) provider.RouteInput {
	return provider.RouteInput{
		Role:           provider.RoleIndexBar,
		ProviderID:     input.ProviderID,
		PreferProvider: input.PreferProvider,
		Market:         input.Market,
		Group:          input.Group,
		Operation:      input.Operation,
		Symbol:         input.IndexCode,
	}
}
