package indexbar

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type RangeQuerySupport string
type FetchMode string

const (
	RangeQueryUnsupported RangeQuerySupport = "unsupported"
	RangeQuerySupported   RangeQuerySupport = "supported"

	FetchModeDirect FetchMode = "direct"
	FetchModeBatch  FetchMode = "batch"
)

type Profile struct {
	Markets       []provider.Market
	Group         provider.GroupID
	Operations    []provider.OperationID
	AuthScope     provider.CredentialScope
	RangeQuery    RangeQuerySupport
	Freshness     provider.Freshness
	Compatibility provider.Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
	FetchMode     FetchMode
}

func (p Profile) RoleProfile() provider.RoleProfile {
	return provider.RoleProfile{
		Role:          provider.RoleIndexBar,
		Markets:       p.Markets,
		Group:         p.Group,
		Operations:    p.Operations,
		AuthScope:     p.AuthScope,
		Freshness:     p.Freshness,
		Compatibility: p.Compatibility,
		RequiresAuth:  p.RequiresAuth,
		Priority:      p.Priority,
		Limitations:   p.Limitations,
		FetchMode:     string(p.FetchMode),
	}
}

type FetchInput struct {
	Market    provider.Market
	IndexCode string
	From      string
	To        string
}

type BatchFetchInput struct {
	Market    provider.Market
	IndexCode string
	From      string
	To        string
}

type Bar struct {
	Provider  provider.ProviderID  `json:"provider" csv:"-"`
	Group     provider.GroupID     `json:"provider_group" csv:"-"`
	Operation provider.OperationID `json:"operation" csv:"-"`
	Market    provider.Market      `json:"market" csv:"market"`

	IndexCode string `json:"index_code" csv:"index_code"`
	Name      string `json:"name,omitempty" csv:"name"`
	Family    string `json:"family,omitempty" csv:"family"`

	TradingDate string `json:"trading_date" csv:"date"`
	Currency    string `json:"currency,omitempty" csv:"currency"`

	Open        string `json:"open_value,omitempty" csv:"open"`
	High        string `json:"high_value,omitempty" csv:"high"`
	Low         string `json:"low_value,omitempty" csv:"low"`
	Close       string `json:"close_value,omitempty" csv:"close"`
	Change      string `json:"change_value,omitempty" csv:"change"`
	ChangeRate  string `json:"change_rate,omitempty" csv:"change_rate"`
	Volume      string `json:"volume,omitempty" csv:"volume"`
	TradedValue string `json:"traded_amount,omitempty" csv:"traded_amount"`
	MarketCap   string `json:"market_cap,omitempty" csv:"market_cap"`

	Extensions map[string]string `json:"extensions,omitempty" csv:"-"`
}

type FetchResult struct {
	Bars       []Bar
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type BatchFetchResult struct {
	Bars       []Bar
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type Fetcher interface {
	provider.RoleProvider
	FetchIndexBars(ctx context.Context, input FetchInput) (FetchResult, error)
	IndexBarProfile() Profile
}

type BatchFetcher interface {
	FetchIndexBatch(ctx context.Context, input BatchFetchInput) (BatchFetchResult, error)
}

type FetchFunc func(context.Context, FetchInput) (FetchResult, error)
type BatchFetchFunc func(context.Context, BatchFetchInput) (BatchFetchResult, error)

type Fetch struct {
	profile    Profile
	fetch      FetchFunc
	batchFetch BatchFetchFunc
}

func NewFetch(profile Profile, fetch FetchFunc) Fetch {
	if profile.FetchMode == "" {
		profile.FetchMode = FetchModeDirect
	}
	return Fetch{profile: profile, fetch: fetch}
}

func NewBatchFetch(profile Profile, fetch FetchFunc, batchFetch BatchFetchFunc) Fetch {
	profile.FetchMode = FetchModeBatch
	return Fetch{profile: profile, fetch: fetch, batchFetch: batchFetch}
}

func (f Fetch) FetchIndexBars(ctx context.Context, input FetchInput) (FetchResult, error) {
	if f.fetch == nil {
		return FetchResult{}, oops.In("provider_role").With("role", provider.RoleIndexBar).New("indexbar fetch role is not configured")
	}
	return f.fetch(ctx, input)
}

func (f Fetch) FetchIndexBatch(ctx context.Context, input BatchFetchInput) (BatchFetchResult, error) {
	if f.batchFetch == nil {
		return BatchFetchResult{}, oops.In("provider_role").With("role", provider.RoleIndexBar).New("indexbar batch fetch role is not configured")
	}
	return f.batchFetch(ctx, input)
}

func (f Fetch) IndexBarProfile() Profile {
	return f.profile
}

func (f Fetch) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.profile.RoleProfile(),
		Impl:    f,
	}
}
