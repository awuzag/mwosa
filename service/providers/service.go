package providers

import (
	"context"
	"sort"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type Registry interface {
	Entries(role provider.Role) []provider.RoleEntry
	Roles() []provider.Role
}

type Service struct {
	registry Registry
	roles    []provider.Role
}

func NewService(registry Registry, roles ...provider.Role) (Service, error) {
	if registry == nil {
		return Service{}, oops.In("providers_service").New("providers service registry is nil")
	}
	return Service{
		registry: registry,
		roles:    append([]provider.Role(nil), roles...),
	}, nil
}

type ListRequest struct {
	ProviderID   provider.ProviderID
	Role         provider.Role
	Market       provider.Market
	SecurityType provider.SecurityType
}

type InspectRequest struct {
	ProviderID provider.ProviderID
}

type ListResult struct {
	Providers []ProviderSummary `json:"providers"`
}

type ProviderSummary struct {
	Provider provider.Identity `json:"provider"`
	Roles    []RoleSummary     `json:"roles"`
}

type RoleSummary struct {
	Role          provider.Role            `json:"role"`
	Markets       []provider.Market        `json:"markets,omitempty"`
	SecurityTypes []provider.SecurityType  `json:"security_types,omitempty"`
	Group         provider.GroupID         `json:"provider_group,omitempty"`
	Operations    []provider.OperationID   `json:"operations,omitempty"`
	AuthScope     provider.CredentialScope `json:"auth_scope,omitempty"`
	Freshness     provider.Freshness       `json:"freshness,omitempty"`
	Compatibility provider.Compatibility   `json:"compatibility"`
	RequiresAuth  bool                     `json:"requires_auth"`
	Priority      int                      `json:"priority,omitempty"`
	Limitations   []string                 `json:"limitations,omitempty"`
	FetchMode     string                   `json:"fetch_mode,omitempty"`
}

func (s Service) List(ctx context.Context, req ListRequest) (ListResult, error) {
	errb := oops.In("providers_service").With("provider", req.ProviderID, "role", req.Role, "market", req.Market, "security_type", req.SecurityType)
	if err := ctx.Err(); err != nil {
		return ListResult{}, errb.Wrap(err)
	}
	if s.registry == nil {
		return ListResult{}, errb.New("providers service registry is nil")
	}

	byProvider := make(map[provider.ProviderID]int)
	summaries := make([]ProviderSummary, 0)
	for _, role := range s.rolesForRequest(req) {
		for _, entry := range s.registry.Entries(role) {
			if !matchesListRequest(req, entry) {
				continue
			}
			index, ok := byProvider[entry.Provider.ID]
			if !ok {
				index = len(summaries)
				byProvider[entry.Provider.ID] = index
				summaries = append(summaries, ProviderSummary{
					Provider: entry.Provider,
				})
			}
			summaries[index].Roles = append(summaries[index].Roles, roleSummary(entry.Profile))
		}
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].Provider.ID < summaries[j].Provider.ID
	})
	for i := range summaries {
		sort.SliceStable(summaries[i].Roles, func(left, right int) bool {
			return summaries[i].Roles[left].Role < summaries[i].Roles[right].Role
		})
	}
	return ListResult{Providers: summaries}, nil
}

func (s Service) Inspect(ctx context.Context, req InspectRequest) (ProviderSummary, error) {
	errb := oops.In("providers_service").With("provider", req.ProviderID)
	if req.ProviderID == "" {
		return ProviderSummary{}, errb.New("inspect provider requires provider id")
	}
	result, err := s.List(ctx, ListRequest{ProviderID: req.ProviderID})
	if err != nil {
		return ProviderSummary{}, errb.Wrap(err)
	}
	if len(result.Providers) == 0 {
		return ProviderSummary{}, errb.Wrapf(provider.ErrNoProvider, "provider=%s", req.ProviderID)
	}
	return result.Providers[0], nil
}

func (s Service) rolesForRequest(req ListRequest) []provider.Role {
	if req.Role != "" {
		return []provider.Role{req.Role}
	}
	if len(s.roles) > 0 {
		return append([]provider.Role(nil), s.roles...)
	}
	return s.registry.Roles()
}

func matchesListRequest(req ListRequest, entry provider.RoleEntry) bool {
	if req.ProviderID != "" && entry.Provider.ID != req.ProviderID {
		return false
	}
	if req.Market != "" && !containsMarket(entry.Profile.Markets, req.Market) {
		return false
	}
	if req.SecurityType != "" && !containsSecurityType(entry.Profile.SecurityTypes, req.SecurityType) {
		return false
	}
	return true
}

func roleSummary(profile provider.RoleProfile) RoleSummary {
	return RoleSummary{
		Role:          profile.Role,
		Markets:       append([]provider.Market(nil), profile.Markets...),
		SecurityTypes: append([]provider.SecurityType(nil), profile.SecurityTypes...),
		Group:         profile.Group,
		Operations:    append([]provider.OperationID(nil), profile.Operations...),
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   append([]string(nil), profile.Limitations...),
		FetchMode:     profile.FetchMode,
	}
}

func containsMarket(values []provider.Market, target provider.Market) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsSecurityType(values []provider.SecurityType, target provider.SecurityType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
