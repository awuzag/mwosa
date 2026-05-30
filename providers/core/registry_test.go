package core_test

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/spec"
)

type registryTestProvider struct {
	provider.Identity

	dailybar.Fetcher
}

type namedRoleFieldProvider struct {
	provider.Identity

	DailyBars dailybar.Fetcher
}

type explicitRegistrationProvider struct {
	provider.Identity

	registrations []provider.RoleRegistration
}

func TestRegisterProviderCollectsEmbeddedRoleFields(t *testing.T) {
	registry := provider.NewRegistry()
	p := &registryTestProvider{
		Identity: provider.Identity{
			ID:          provider.ProviderID("test"),
			DisplayName: "test",
		},
		Fetcher: newRegistryTestDailyBarFetcher(),
	}

	if err := registry.RegisterProvider(p); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	entries := registry.Entries(provider.RoleDailyBar)
	if len(entries) != 1 {
		t.Fatalf("dailybar entries len = %d, want 1", len(entries))
	}
	if entries[0].Provider.ID != provider.ProviderID("test") {
		t.Fatalf("provider id = %s, want test", entries[0].Provider.ID)
	}
	if entries[0].Profile.Compatibility.DataLatency != provider.DataLatencyPreviousBusinessDay {
		t.Fatalf("data latency = %s, want previous_business_day", entries[0].Profile.Compatibility.DataLatency)
	}
	if _, ok := entries[0].Impl.(dailybar.Fetcher); !ok {
		t.Fatalf("impl type = %T, want dailybar.Fetcher", entries[0].Impl)
	}
}

func TestRegisterProviderUsesExplicitRoleRegistrations(t *testing.T) {
	registry := provider.NewRegistry()
	p := &explicitRegistrationProvider{
		Identity: provider.Identity{
			ID:          provider.ProviderID("test"),
			DisplayName: "test",
		},
		registrations: []provider.RoleRegistration{
			newRegistryTestDailyBarFetcher().RoleRegistration(),
		},
	}

	if err := registry.RegisterProvider(p); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	entries := registry.Entries(provider.RoleDailyBar)
	if len(entries) != 1 {
		t.Fatalf("dailybar entries len = %d, want 1", len(entries))
	}
	if entries[0].Provider.ID != provider.ProviderID("test") {
		t.Fatalf("provider id = %s, want test", entries[0].Provider.ID)
	}
}

func (p *explicitRegistrationProvider) RoleRegistrations() []provider.RoleRegistration {
	return p.registrations
}

func TestRegistryRolesReturnsRegisteredRolesOnce(t *testing.T) {
	registry := provider.NewRegistry()
	identity := provider.Identity{ID: provider.ProviderID("test")}
	if err := registry.Register(identity,
		provider.RoleRegistration{
			Profile: provider.RoleProfile{
				Role:          provider.Role("macro"),
				Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyEndOfDay},
			},
			Impl: struct{}{},
		},
		provider.RoleRegistration{
			Profile: provider.RoleProfile{
				Role:          provider.RoleDailyBar,
				Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
			},
			Impl: struct{}{},
		},
		provider.RoleRegistration{
			Profile: provider.RoleProfile{
				Role:          provider.RoleDailyBar,
				Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
			},
			Impl: struct{}{},
		},
	); err != nil {
		t.Fatalf("register roles: %v", err)
	}

	roles := registry.Roles()
	if len(roles) != 2 {
		t.Fatalf("roles len = %d, want 2: %v", len(roles), roles)
	}
	if roles[0] != provider.RoleDailyBar || roles[1] != provider.Role("macro") {
		t.Fatalf("roles = %v, want sorted unique roles", roles)
	}
}

func TestRegisterProviderIgnoresNamedRoleFields(t *testing.T) {
	registry := provider.NewRegistry()
	p := &namedRoleFieldProvider{
		Identity: provider.Identity{
			ID:          provider.ProviderID("test"),
			DisplayName: "test",
		},
		DailyBars: newRegistryTestDailyBarFetcher(),
	}

	err := registry.RegisterProvider(p)
	if err == nil {
		t.Fatal("register provider error = nil, want no embedded role fields error")
	}
	if !strings.Contains(err.Error(), "provider has no role fields") {
		t.Fatalf("error = %q, want no role fields context", err.Error())
	}
}

func TestRegisterProviderRejectsNilEmbeddedRoleField(t *testing.T) {
	registry := provider.NewRegistry()
	p := &registryTestProvider{
		Identity: provider.Identity{
			ID:          provider.ProviderID("test"),
			DisplayName: "test",
		},
	}

	err := registry.RegisterProvider(p)
	if err == nil {
		t.Fatal("register provider error = nil, want nil role field error")
	}
	if !strings.Contains(err.Error(), "embedded role field is nil") {
		t.Fatalf("error = %q, want role field context", err.Error())
	}
}

func newRegistryTestDailyBarFetcher() dailybar.Fetcher {
	return spec.PreviousBusinessDayDailyBar(func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{}, nil
	}).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupID("testGroup")).
		Operations(provider.OperationID("testOperation")).
		RequiresAuth(provider.CredentialScopeDataGo).
		RangeQuery(dailybar.RangeQuerySupported).
		MustBuild()
}
