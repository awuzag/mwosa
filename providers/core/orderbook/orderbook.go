package orderbook

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type Side string

const (
	SideAsk Side = "ask"
	SideBid Side = "bid"
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
		Role:          provider.RoleOrderbook,
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

type SnapshotInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
}

type Snapshot struct {
	Provider     provider.ProviderID   `json:"provider" csv:"-"`
	Group        provider.GroupID      `json:"provider_group" csv:"-"`
	Operation    provider.OperationID  `json:"operation" csv:"-"`
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`

	Symbol           string             `json:"symbol" csv:"symbol"`
	AcceptanceTime   string             `json:"acceptance_time,omitempty" csv:"acceptance_time"`
	Currency         string             `json:"currency" csv:"-"`
	Levels           []Level            `json:"levels" csv:"-"`
	TotalAskQuantity string             `json:"total_ask_quantity,omitempty" csv:"total_ask_quantity"`
	TotalBidQuantity string             `json:"total_bid_quantity,omitempty" csv:"total_bid_quantity"`
	Expected         ExpectedConclusion `json:"expected,omitempty" csv:"-"`
	Extensions       map[string]string  `json:"extensions,omitempty" csv:"-"`
}

type Level struct {
	Side          Side   `json:"side" csv:"side"`
	Level         int    `json:"level" csv:"level"`
	Price         string `json:"price" csv:"price"`
	Quantity      string `json:"quantity" csv:"quantity"`
	QuantityDelta string `json:"quantity_delta,omitempty" csv:"quantity_delta"`
}

type ExpectedConclusion struct {
	Price              string `json:"price,omitempty"`
	Volume             string `json:"volume,omitempty"`
	Current            string `json:"current_price,omitempty"`
	Open               string `json:"opening_price,omitempty"`
	High               string `json:"highest_price,omitempty"`
	Low                string `json:"lowest_price,omitempty"`
	PreviousClose      string `json:"previous_close,omitempty"`
	PreviousChange     string `json:"previous_change,omitempty"`
	PreviousChangeSign string `json:"previous_change_sign,omitempty"`
	PreviousChangeRate string `json:"previous_change_rate,omitempty"`
}

type SnapshotResult struct {
	Snapshot   Snapshot
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type Snapshotter interface {
	provider.RoleProvider
	FetchOrderbookSnapshot(ctx context.Context, input SnapshotInput) (SnapshotResult, error)
	OrderbookProfile() Profile
}

type SnapshotFunc func(context.Context, SnapshotInput) (SnapshotResult, error)

type SnapshotRole struct {
	profile  Profile
	snapshot SnapshotFunc
}

func NewSnapshot(profile Profile, snapshot SnapshotFunc) SnapshotRole {
	return SnapshotRole{profile: profile, snapshot: snapshot}
}

func (s SnapshotRole) FetchOrderbookSnapshot(ctx context.Context, input SnapshotInput) (SnapshotResult, error) {
	if s.snapshot == nil {
		return SnapshotResult{}, oops.In("provider_role").With("role", provider.RoleOrderbook).New("orderbook snapshot role is not configured")
	}
	return s.snapshot(ctx, input)
}

func (s SnapshotRole) OrderbookProfile() Profile {
	return s.profile
}

func (s SnapshotRole) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: s.profile.RoleProfile(),
		Impl:    s,
	}
}
