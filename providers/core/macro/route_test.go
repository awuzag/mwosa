package macro_test

import (
	"context"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/macro"
)

type fakeCoreRouter struct {
	candidate provider.RouteCandidate
	err       error
	input     provider.RouteInput
}

func (r *fakeCoreRouter) Route(_ context.Context, input provider.RouteInput) (provider.RouteCandidate, error) {
	r.input = input
	if r.err != nil {
		return provider.RouteCandidate{}, r.err
	}
	return r.candidate, nil
}

func (r *fakeCoreRouter) Plan(_ context.Context, input provider.RouteInput) (provider.RoutePlan, error) {
	r.input = input
	if r.err != nil {
		return provider.RoutePlan{}, r.err
	}
	return provider.RoutePlan{Candidates: []provider.RouteCandidate{r.candidate}}, nil
}

func TestRouteMacroUsesMacroRole(t *testing.T) {
	fetcher := macro.NewFetch(macro.Profile{
		Group:      provider.GroupECOSKeyStatistics,
		Operations: []provider.OperationID{provider.OperationECOSKeyStatistics},
		Compatibility: provider.Compatibility{
			DataLatency: provider.DataLatencyHistorical,
		},
	}, func(context.Context, macro.IndicatorInput) (macro.IndicatorResult, error) {
		return macro.IndicatorResult{}, nil
	}, func(context.Context, macro.ObservationInput) (macro.ObservationResult, error) {
		return macro.ObservationResult{}, nil
	})
	core := &fakeCoreRouter{
		candidate: provider.RouteCandidate{
			Provider: provider.Identity{ID: provider.ProviderECOS},
			Profile:  fetcher.MacroProfile().RoleProfile(),
			Impl:     fetcher,
		},
	}
	router := macro.NewRouter(core)

	got, err := router.RouteMacro(context.Background(), macro.RouteInput{
		ProviderID:  provider.ProviderECOS,
		Preset:      macro.PresetKeyStatistics,
		IndicatorID: "ecos.base-rate",
	})
	if err != nil {
		t.Fatalf("RouteMacro error = %v", err)
	}
	if got.MacroProfile().Group != provider.GroupECOSKeyStatistics {
		t.Fatalf("group = %s, want %s", got.MacroProfile().Group, provider.GroupECOSKeyStatistics)
	}
	if core.input.Role != provider.RoleMacro || core.input.ProviderID != provider.ProviderECOS || core.input.Symbol != "ecos.base-rate" {
		t.Fatalf("core route input = %+v", core.input)
	}
}

func TestFetchRejectsMissingCallables(t *testing.T) {
	fetcher := macro.NewFetch(macro.Profile{}, nil, nil)
	if _, err := fetcher.FetchMacroIndicators(context.Background(), macro.IndicatorInput{}); err == nil {
		t.Fatal("FetchMacroIndicators error = nil")
	}
	if _, err := fetcher.FetchMacroObservations(context.Background(), macro.ObservationInput{}); err == nil {
		t.Fatal("FetchMacroObservations error = nil")
	}
}
