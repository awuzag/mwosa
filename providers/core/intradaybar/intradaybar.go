package intradaybar

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
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
		Role:          provider.RoleIntradayBar,
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

type FetchInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	At           string
	Limit        int
}

type Bar struct {
	Provider     provider.ProviderID   `json:"provider" csv:"-"`
	Group        provider.GroupID      `json:"provider_group" csv:"-"`
	Operation    provider.OperationID  `json:"operation" csv:"-"`
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`

	TradingDate string `json:"trading_date" csv:"date"`
	Time        string `json:"time" csv:"time"`
	Symbol      string `json:"symbol" csv:"symbol"`
	Currency    string `json:"currency" csv:"-"`

	Open        string `json:"opening_price,omitempty" csv:"open"`
	High        string `json:"highest_price,omitempty" csv:"high"`
	Low         string `json:"lowest_price,omitempty" csv:"low"`
	Close       string `json:"closing_price,omitempty" csv:"close"`
	Volume      string `json:"traded_volume,omitempty" csv:"volume"`
	TradedValue string `json:"traded_amount,omitempty" csv:"traded_amount"`

	Extensions map[string]string `json:"extensions,omitempty" csv:"-"`
}

type FetchResult struct {
	Bars       []Bar
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type Fetcher interface {
	provider.RoleProvider
	FetchIntradayBars(ctx context.Context, input FetchInput) (FetchResult, error)
	IntradayBarProfile() Profile
}

type FetchFunc func(context.Context, FetchInput) (FetchResult, error)

type Fetch struct {
	profile Profile
	fetch   FetchFunc
}

func NewFetch(profile Profile, fetch FetchFunc) Fetch {
	return Fetch{profile: profile, fetch: fetch}
}

func (f Fetch) FetchIntradayBars(ctx context.Context, input FetchInput) (FetchResult, error) {
	if f.fetch == nil {
		return FetchResult{}, oops.In("provider_role").With("role", provider.RoleIntradayBar).New("intraday bar fetch role is not configured")
	}
	return f.fetch(ctx, input)
}

func (f Fetch) IntradayBarProfile() Profile {
	return f.profile
}

func (f Fetch) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.profile.RoleProfile(),
		Impl:    f,
	}
}
