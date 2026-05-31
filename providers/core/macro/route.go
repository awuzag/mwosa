package macro

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type RouteInput struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Group          provider.GroupID
	Operation      provider.OperationID
	Preset         Preset
	IndicatorID    string
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
	RouteMacro(ctx context.Context, input RouteInput) (Fetcher, error)
	PlanMacro(ctx context.Context, input RouteInput) (RoutePlan, error)
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

func (r routeAdapter) RouteMacro(ctx context.Context, input RouteInput) (Fetcher, error) {
	errb := oops.In("macro_router").With("provider", input.ProviderID, "preset", input.Preset, "indicator_id", input.IndicatorID)
	candidate, err := r.router.Route(ctx, toCoreRouteInput(input))
	if err != nil {
		return nil, errb.Wrap(err)
	}
	fetcher, ok := candidate.Impl.(Fetcher)
	if !ok {
		return nil, errb.With("provider", candidate.Provider.ID).New("routed macro implementation does not satisfy Fetcher")
	}
	return fetcher, nil
}

func (r routeAdapter) PlanMacro(ctx context.Context, input RouteInput) (RoutePlan, error) {
	errb := oops.In("macro_router").With("provider", input.ProviderID, "preset", input.Preset, "indicator_id", input.IndicatorID)
	plan, err := r.router.Plan(ctx, toCoreRouteInput(input))
	if err != nil {
		return RoutePlan{}, errb.Wrap(err)
	}
	candidates := make([]RouteCandidate, 0, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		fetcher, ok := candidate.Impl.(Fetcher)
		if !ok {
			return RoutePlan{}, errb.With("provider", candidate.Provider.ID).New("routed macro implementation does not satisfy Fetcher")
		}
		candidates = append(candidates, RouteCandidate{
			Provider: candidate.Provider,
			Group:    candidate.Profile.Group,
			Fetcher:  fetcher,
			Profile:  fetcher.MacroProfile(),
			Reason:   candidate.Reason,
		})
	}
	return RoutePlan{Candidates: candidates}, nil
}

func toCoreRouteInput(input RouteInput) provider.RouteInput {
	symbol := input.IndicatorID
	if symbol == "" {
		symbol = string(input.Preset)
	}
	return provider.RouteInput{
		Role:           provider.RoleMacro,
		ProviderID:     input.ProviderID,
		PreferProvider: input.PreferProvider,
		Group:          input.Group,
		Operation:      input.Operation,
		Symbol:         symbol,
	}
}
