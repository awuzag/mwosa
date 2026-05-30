package instrument

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type Profile struct {
	Markets       []provider.Market
	SecurityTypes []provider.SecurityType
	Group         provider.GroupID
	Operations    []provider.OperationID
	AuthScope     provider.CredentialScope
	Freshness     provider.Freshness
	Compatibility provider.Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
}

func (p Profile) RoleProfile() provider.RoleProfile {
	return provider.RoleProfile{
		Role:          provider.RoleInstrument,
		Markets:       p.Markets,
		SecurityTypes: p.SecurityTypes,
		Group:         p.Group,
		Operations:    p.Operations,
		AuthScope:     p.AuthScope,
		Freshness:     p.Freshness,
		Compatibility: p.Compatibility,
		RequiresAuth:  p.RequiresAuth,
		Priority:      p.Priority,
		Limitations:   p.Limitations,
	}
}

type SearchInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Query        string
	Limit        int
	AsOf         string
}

type Instrument struct {
	Provider     provider.ProviderID   `json:"provider"`
	Group        provider.GroupID      `json:"provider_group"`
	Operation    provider.OperationID  `json:"operation"`
	Market       provider.Market       `json:"market"`
	SecurityType provider.SecurityType `json:"security_type"`

	SecurityCode string `json:"security_code"`
	ISIN         string `json:"isin"`
	Name         string `json:"name"`
	ExchangeCode string `json:"exchange_code"`
	CountryCode  string `json:"country_code"`
	Timezone     string `json:"timezone"`

	Extensions map[string]string `json:"extensions,omitempty"`
}

type SearchResult struct {
	Instruments []Instrument           `json:"instruments"`
	Provider    provider.Identity      `json:"provider_identity"`
	Group       provider.GroupID       `json:"provider_group"`
	Operations  []provider.OperationID `json:"operations"`
	TotalCount  int                    `json:"total_count"`
}

type Searcher interface {
	provider.RoleProvider
	SearchInstruments(ctx context.Context, input SearchInput) (SearchResult, error)
	InstrumentSearchProfile() Profile
}

type SearchFunc func(context.Context, SearchInput) (SearchResult, error)

type Search struct {
	profile Profile
	search  SearchFunc
}

func NewSearch(profile Profile, search SearchFunc) Search {
	return Search{profile: profile, search: search}
}

func (s Search) SearchInstruments(ctx context.Context, input SearchInput) (SearchResult, error) {
	if s.search == nil {
		return SearchResult{}, oops.In("provider_role").With("role", provider.RoleInstrument).New("instrument search role is not configured")
	}
	return s.search(ctx, input)
}

func (s Search) InstrumentSearchProfile() Profile {
	return s.profile
}

func (s Search) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: s.profile.RoleProfile(),
		Impl:    s,
	}
}
