package core_test

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/samber/oops"
)

func TestRegistryRegisterConfiguredRegistersBuilderDecision(t *testing.T) {
	registry := provider.NewRegistry()
	err := registry.RegisterConfigured(provider.RegisterOptions{}, provider.Config{}, fakeBuilder{
		id: provider.ProviderID("fake"),
		decision: provider.RegistrationDecision{
			Register: true,
			Reason:   "test builder",
		},
		provider: newFakeProvider(provider.ProviderID("fake")),
	})
	if err != nil {
		t.Fatalf("RegisterConfigured error = %v", err)
	}

	fetcher, err := dailybar.NewRouter(provider.NewRouter(registry)).RouteDailyBars(context.Background(), dailybar.RouteInput{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
	})
	if err != nil {
		t.Fatalf("RouteDailyBars error = %v", err)
	}
	if fetcher.DailyBarProfile().Group != provider.GroupID("fakeGroup") {
		t.Fatalf("route returned unexpected fake profile")
	}
}

func TestRegistryRegisterConfiguredSkipsBuilderDecision(t *testing.T) {
	registry := provider.NewRegistry()
	err := registry.RegisterConfigured(provider.RegisterOptions{}, provider.Config{}, fakeBuilder{
		id: provider.ProviderID("fake"),
		decision: provider.RegistrationDecision{
			Register: false,
			Reason:   "not configured",
		},
		provider: newFakeProvider(provider.ProviderID("fake")),
	})
	if err != nil {
		t.Fatalf("RegisterConfigured error = %v", err)
	}

	_, err = dailybar.NewRouter(provider.NewRouter(registry)).RouteDailyBars(context.Background(), dailybar.RouteInput{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
	})
	if !provider.IsNoProvider(err) {
		t.Fatalf("RouteDailyBars error = %v, want ErrNoProvider", err)
	}
}

func TestRegistryRegisterConfiguredWrapsBuilderError(t *testing.T) {
	registry := provider.NewRegistry()
	err := registry.RegisterConfigured(provider.RegisterOptions{}, provider.Config{}, fakeBuilder{
		id: provider.ProviderID("fake"),
		decision: provider.RegistrationDecision{
			Register: true,
			Reason:   "forced in test",
		},
		err: oops.New("fake build failed"),
	})
	if err == nil {
		t.Fatal("RegisterConfigured error = nil, want builder error")
	}
	for _, want := range []string{"fake", "forced in test", "fake build failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

type fakeBuilder struct {
	id       provider.ProviderID
	decision provider.RegistrationDecision
	provider provider.IdentityProvider
	err      error
}

func (b fakeBuilder) ID() provider.ProviderID {
	return b.id
}

func (b fakeBuilder) DefaultConfig() provider.Config {
	return provider.Config{"id": string(b.id)}
}

func (b fakeBuilder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{ProviderID: b.id}
}

func (b fakeBuilder) Decide(provider.RegisterOptions, provider.Config) provider.RegistrationDecision {
	return b.decision
}

func (b fakeBuilder) Build(provider.Config) (provider.IdentityProvider, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.provider, nil
}

type fakeProvider struct {
	provider.Identity

	dailybar.Fetcher
}

func newFakeProvider(id provider.ProviderID) *fakeProvider {
	p := &fakeProvider{
		Identity: provider.Identity{
			ID: id,
		},
	}
	p.Fetcher = dailybar.NewFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Group:         provider.GroupID("fakeGroup"),
		Operations:    []provider.OperationID{provider.OperationID("fakeOperation")},
		Freshness:     provider.FreshnessDaily,
		Compatibility: provider.Compatibility{
			DataLatency: provider.DataLatencyEndOfDay,
		},
		Priority: 10,
	}, func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{
			Provider: p.Identity,
			Group:    provider.GroupID("fakeGroup"),
		}, nil
	})
	return p
}
