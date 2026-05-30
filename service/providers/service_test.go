package providers

import (
	"context"
	"sort"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
)

func TestNewServiceRequiresRegistry(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatal("NewService error = nil, want registry error")
	}
	if !strings.Contains(err.Error(), "registry") {
		t.Fatalf("error = %q, want registry context", err.Error())
	}
}

func TestListGroupsProviderRoles(t *testing.T) {
	registry := fakeRegistry{
		provider.RoleDailyBar: []provider.RoleEntry{
			{
				Provider: provider.Identity{ID: provider.ProviderID("datago")},
				Profile: provider.RoleProfile{
					Role:          provider.RoleDailyBar,
					Markets:       []provider.Market{provider.MarketKRX},
					SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
					Group:         provider.GroupSecuritiesProductPrice,
					Operations:    []provider.OperationID{provider.OperationGetETFPriceInfo},
					Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
					Priority:      50,
				},
			},
		},
		provider.RoleInstrument: []provider.RoleEntry{
			{
				Provider: provider.Identity{ID: provider.ProviderID("datago")},
				Profile: provider.RoleProfile{
					Role:          provider.RoleInstrument,
					Markets:       []provider.Market{provider.MarketKRX},
					SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
					Group:         provider.GroupSecuritiesProductPrice,
					Operations:    []provider.OperationID{provider.OperationGetETFPriceInfo},
					Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
					Priority:      50,
				},
			},
		},
	}
	service, err := NewService(registry)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.List(context.Background(), ListRequest{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
	})
	if err != nil {
		t.Fatalf("List error = %v", err)
	}

	if len(result.Providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(result.Providers))
	}
	if len(result.Providers[0].Roles) != 2 {
		t.Fatalf("roles len = %d, want 2", len(result.Providers[0].Roles))
	}
	if result.Providers[0].Roles[0].Role != provider.RoleDailyBar {
		t.Fatalf("first role = %s, want daily_bar", result.Providers[0].Roles[0].Role)
	}
}

func TestListFiltersRole(t *testing.T) {
	registry := fakeRegistry{
		provider.RoleDailyBar: []provider.RoleEntry{
			{Provider: provider.Identity{ID: provider.ProviderID("datago")}, Profile: provider.RoleProfile{Role: provider.RoleDailyBar}},
		},
		provider.RoleQuote: []provider.RoleEntry{
			{Provider: provider.Identity{ID: provider.ProviderID("kis")}, Profile: provider.RoleProfile{Role: provider.RoleQuote}},
		},
	}
	service, err := NewService(registry)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.List(context.Background(), ListRequest{Role: provider.RoleQuote})
	if err != nil {
		t.Fatalf("List error = %v", err)
	}

	if len(result.Providers) != 1 || result.Providers[0].Provider.ID != provider.ProviderID("kis") {
		t.Fatalf("providers = %+v, want kis only", result.Providers)
	}
}

func TestListUsesRegistryRolesByDefault(t *testing.T) {
	roleMacro := provider.Role("macro")
	registry := fakeRegistry{
		roleMacro: []provider.RoleEntry{
			{
				Provider: provider.Identity{ID: provider.ProviderID("macro-provider")},
				Profile:  provider.RoleProfile{Role: roleMacro},
			},
		},
	}
	service, err := NewService(registry)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.List(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("List error = %v", err)
	}

	if len(result.Providers) != 1 || result.Providers[0].Provider.ID != provider.ProviderID("macro-provider") {
		t.Fatalf("providers = %+v, want macro-provider", result.Providers)
	}
	if len(result.Providers[0].Roles) != 1 || result.Providers[0].Roles[0].Role != roleMacro {
		t.Fatalf("roles = %+v, want macro role", result.Providers[0].Roles)
	}
}

func TestInspectReportsMissingProvider(t *testing.T) {
	service, err := NewService(fakeRegistry{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Inspect(context.Background(), InspectRequest{ProviderID: provider.ProviderID("missing")})
	if !provider.IsNoProvider(err) {
		t.Fatalf("Inspect error = %v, want ErrNoProvider", err)
	}
}

type fakeRegistry map[provider.Role][]provider.RoleEntry

func (r fakeRegistry) Entries(role provider.Role) []provider.RoleEntry {
	return append([]provider.RoleEntry(nil), r[role]...)
}

func (r fakeRegistry) Roles() []provider.Role {
	roles := make([]provider.Role, 0, len(r))
	for role := range r {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i] < roles[j]
	})
	return roles
}
