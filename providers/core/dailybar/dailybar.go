package dailybar

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type RangeQuerySupport string

const (
	RangeQueryUnsupported RangeQuerySupport = "unsupported"
	RangeQuerySupported   RangeQuerySupport = "supported"
)

type Profile struct {
	Markets       []provider.Market
	SecurityTypes []provider.SecurityType
	Group         provider.GroupID
	Operations    []provider.OperationID
	AuthScope     provider.CredentialScope
	RangeQuery    RangeQuerySupport
	Freshness     provider.Freshness
	Compatibility provider.Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
}

func (p Profile) RoleProfile() provider.RoleProfile {
	return provider.RoleProfile{
		Role:          provider.RoleDailyBar,
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
	From         string
	To           string
	Limit        int
	Workers      int
}

type PageFetchInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	From         string
	To           string
	PageNo       int
	PageSize     int
}

type Bar struct {
	Provider     provider.ProviderID   `json:"provider" csv:"-"`
	Group        provider.GroupID      `json:"provider_group" csv:"-"`
	Operation    provider.OperationID  `json:"operation" csv:"-"`
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`

	TradingDate string `json:"trading_date" csv:"date"`
	Symbol      string `json:"symbol" csv:"symbol"`
	ISIN        string `json:"isin,omitempty" csv:"-"`
	Name        string `json:"name,omitempty" csv:"name"`
	Currency    string `json:"currency" csv:"-"`

	Open        string `json:"opening_price,omitempty" csv:"open"`
	High        string `json:"highest_price,omitempty" csv:"high"`
	Low         string `json:"lowest_price,omitempty" csv:"low"`
	Close       string `json:"closing_price,omitempty" csv:"close"`
	Change      string `json:"price_change_from_previous_close,omitempty" csv:"change"`
	ChangeRate  string `json:"price_change_rate_from_previous_close,omitempty" csv:"-"`
	Volume      string `json:"traded_volume,omitempty" csv:"-"`
	TradedValue string `json:"traded_amount,omitempty" csv:"-"`
	MarketCap   string `json:"market_capitalization,omitempty" csv:"-"`

	Extensions map[string]string `json:"extensions,omitempty" csv:"-"`
}

type FetchResult struct {
	Bars       []Bar
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type PageFetchResult struct {
	Bars       []Bar
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	PageNo     int
	PageSize   int
	TotalCount int
}

type Fetcher interface {
	provider.RoleProvider
	FetchDailyBars(ctx context.Context, input FetchInput) (FetchResult, error)
	DailyBarProfile() Profile
}

type PageFetcher interface {
	FetchDailyBarsPage(ctx context.Context, input PageFetchInput) (PageFetchResult, error)
}

type FetchFunc func(context.Context, FetchInput) (FetchResult, error)
type PageFetchFunc func(context.Context, PageFetchInput) (PageFetchResult, error)

type Fetch struct {
	profile   Profile
	fetch     FetchFunc
	pageFetch PageFetchFunc
}

func NewFetch(profile Profile, fetch FetchFunc) Fetch {
	return Fetch{profile: profile, fetch: fetch}
}

func NewPagedFetch(profile Profile, fetch FetchFunc, pageFetch PageFetchFunc) Fetch {
	return Fetch{profile: profile, fetch: fetch, pageFetch: pageFetch}
}

func (f Fetch) FetchDailyBars(ctx context.Context, input FetchInput) (FetchResult, error) {
	if f.fetch == nil {
		return FetchResult{}, oops.In("provider_role").With("role", provider.RoleDailyBar).New("dailybar fetch role is not configured")
	}
	return f.fetch(ctx, input)
}

func (f Fetch) FetchDailyBarsPage(ctx context.Context, input PageFetchInput) (PageFetchResult, error) {
	if f.pageFetch == nil {
		return PageFetchResult{}, oops.In("provider_role").With("role", provider.RoleDailyBar).New("dailybar page fetch role is not configured")
	}
	return f.pageFetch(ctx, input)
}

func (f Fetch) DailyBarProfile() Profile {
	return f.profile
}

func (f Fetch) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.profile.RoleProfile(),
		Impl:    f,
	}
}
