package trades

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
		Role:          provider.RoleTrades,
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

type ListInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	At           string
	Limit        int
}

type Trade struct {
	Provider     provider.ProviderID   `json:"provider" csv:"-"`
	Group        provider.GroupID      `json:"provider_group" csv:"-"`
	Operation    provider.OperationID  `json:"operation" csv:"-"`
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`

	Symbol             string            `json:"symbol" csv:"symbol"`
	Time               string            `json:"time" csv:"time"`
	Price              string            `json:"price" csv:"price"`
	Volume             string            `json:"volume" csv:"volume"`
	AccumulatedVolume  string            `json:"accumulated_volume,omitempty" csv:"accumulated_volume"`
	Ask                string            `json:"ask_price,omitempty" csv:"ask_price"`
	Bid                string            `json:"bid_price,omitempty" csv:"bid_price"`
	PreviousChange     string            `json:"previous_change,omitempty" csv:"previous_change"`
	PreviousChangeSign string            `json:"previous_change_sign,omitempty" csv:"-"`
	PreviousChangeRate string            `json:"previous_change_rate,omitempty" csv:"-"`
	Strength           string            `json:"trade_strength,omitempty" csv:"trade_strength"`
	Extensions         map[string]string `json:"extensions,omitempty" csv:"-"`
}

type ListResult struct {
	Trades     []Trade
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type Lister interface {
	provider.RoleProvider
	ListMarketTrades(ctx context.Context, input ListInput) (ListResult, error)
	TradesProfile() Profile
}

type ListFunc func(context.Context, ListInput) (ListResult, error)

type List struct {
	profile Profile
	list    ListFunc
}

func NewList(profile Profile, list ListFunc) List {
	return List{profile: profile, list: list}
}

func (l List) ListMarketTrades(ctx context.Context, input ListInput) (ListResult, error) {
	if l.list == nil {
		return ListResult{}, oops.In("provider_role").With("role", provider.RoleTrades).New("market trades list role is not configured")
	}
	return l.list(ctx, input)
}

func (l List) TradesProfile() Profile {
	return l.profile
}

func (l List) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: l.profile.RoleProfile(),
		Impl:    l,
	}
}
