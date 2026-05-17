package composition

import (
	"context"
	"encoding/json"

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
		Role:          provider.RoleComposition,
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
	Limit        int
}

type SourceRef struct {
	Provider  provider.ProviderID  `json:"provider"`
	Group     provider.GroupID     `json:"provider_group"`
	Operation provider.OperationID `json:"operation"`
}

type InstrumentRef struct {
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`
	Symbol       string                `json:"symbol" csv:"symbol"`
	ISIN         string                `json:"isin,omitempty" csv:"-"`
	Name         string                `json:"name,omitempty" csv:"name"`
}

type DecimalValue struct {
	Value string `json:"value,omitempty" csv:"-"`
}

type MoneyValue struct {
	Currency string `json:"currency,omitempty" csv:"-"`
	Value    string `json:"value,omitempty" csv:"-"`
}

type Composition struct {
	Source       SourceRef           `json:"source"`
	Subject      InstrumentRef       `json:"subject"`
	AsOfDate     string              `json:"as_of_date,omitempty"`
	ObservedAtMS int64               `json:"observed_at_ms,omitempty"`
	Members      []CompositionMember `json:"members"`
}

type CompositionMember struct {
	Instrument InstrumentRef `json:"instrument"`
	Weight     DecimalValue  `json:"weight,omitempty"`
	Quantity   DecimalValue  `json:"quantity,omitempty"`
	Valuation  MoneyValue    `json:"valuation,omitempty"`
}

func (m CompositionMember) MarshalJSON() ([]byte, error) {
	type member struct {
		Instrument InstrumentRef `json:"instrument"`
		Weight     *DecimalValue `json:"weight,omitempty"`
		Quantity   *DecimalValue `json:"quantity,omitempty"`
		Valuation  *MoneyValue   `json:"valuation,omitempty"`
	}
	out := member{Instrument: m.Instrument}
	if m.Weight.Value != "" {
		out.Weight = &m.Weight
	}
	if m.Quantity.Value != "" {
		out.Quantity = &m.Quantity
	}
	if m.Valuation.Value != "" || m.Valuation.Currency != "" {
		out.Valuation = &m.Valuation
	}
	return json.Marshal(out)
}

type QuoteObservation struct {
	Instrument   InstrumentRef `json:"instrument"`
	ObservedAtMS int64         `json:"observed_at_ms,omitempty"`
	Price        MoneyValue    `json:"price,omitempty"`
	Change       MoneyValue    `json:"change,omitempty"`
	ChangeRate   DecimalValue  `json:"change_rate,omitempty"`
	Volume       DecimalValue  `json:"volume,omitempty"`
}

type ListResult struct {
	Composition       Composition
	QuoteObservations []QuoteObservation
	Provider          provider.Identity
	Group             provider.GroupID
	Operation         provider.OperationID
	TotalCount        int
}

type Lister interface {
	provider.RoleProvider
	ListConstituents(ctx context.Context, input ListInput) (ListResult, error)
	CompositionProfile() Profile
}

type ListFunc func(context.Context, ListInput) (ListResult, error)

type List struct {
	profile Profile
	list    ListFunc
}

func NewList(profile Profile, list ListFunc) List {
	return List{profile: profile, list: list}
}

func (l List) ListConstituents(ctx context.Context, input ListInput) (ListResult, error) {
	if l.list == nil {
		return ListResult{}, oops.In("provider_role").With("role", provider.RoleComposition).New("composition list role is not configured")
	}
	return l.list(ctx, input)
}

func (l List) CompositionProfile() Profile {
	return l.profile
}

func (l List) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: l.profile.RoleProfile(),
		Impl:    l,
	}
}
