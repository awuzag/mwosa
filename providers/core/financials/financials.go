package financials

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type StatementType string
type PeriodType string

const (
	StatementTypeSummary         StatementType = "summary"
	StatementTypeIncomeStatement StatementType = "income_statement"
	StatementTypeBalanceSheet    StatementType = "balance_sheet"
	StatementTypeCashFlow        StatementType = "cash_flow"

	PeriodTypeAnnual  PeriodType = "annual"
	PeriodTypeQuarter PeriodType = "quarter"
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
		Role:          provider.RoleFinancials,
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
	FiscalYear   string
	Period       PeriodType
	Statement    StatementType
	Limit        int
}

type Statement struct {
	Statement    StatementType         `json:"statement" csv:"statement"`
	Symbol       string                `json:"symbol" csv:"symbol"`
	Name         string                `json:"name,omitempty" csv:"name"`
	FiscalYear   string                `json:"fiscal_year,omitempty" csv:"fiscal_year"`
	FiscalPeriod string                `json:"fiscal_period,omitempty" csv:"fiscal_period"`
	Period       PeriodType            `json:"period,omitempty" csv:"period"`
	ReportedAt   string                `json:"reported_at,omitempty" csv:"reported_at"`
	Currency     string                `json:"currency,omitempty" csv:"currency"`
	Unit         string                `json:"unit,omitempty" csv:"unit"`
	Lines        []LineItem            `json:"lines" csv:"-"`
	Extensions   map[string]string     `json:"extensions,omitempty" csv:"-"`
	Provider     provider.ProviderID   `json:"provider" csv:"-"`
	Group        provider.GroupID      `json:"provider_group" csv:"-"`
	Operation    provider.OperationID  `json:"operation" csv:"-"`
	Market       provider.Market       `json:"market" csv:"-"`
	SecurityType provider.SecurityType `json:"security_type" csv:"-"`
}

type LineItem struct {
	AccountID   string            `json:"account_id,omitempty" csv:"account_id"`
	AccountName string            `json:"account_name" csv:"account_name"`
	Value       string            `json:"value" csv:"value"`
	Currency    string            `json:"currency,omitempty" csv:"currency"`
	Unit        string            `json:"unit,omitempty" csv:"unit"`
	Extensions  map[string]string `json:"extensions,omitempty" csv:"-"`
}

type FetchResult struct {
	Statements []Statement
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type Fetcher interface {
	provider.RoleProvider
	FetchFinancialStatements(ctx context.Context, input FetchInput) (FetchResult, error)
	FinancialsProfile() Profile
}

type FetchFunc func(context.Context, FetchInput) (FetchResult, error)

type Fetch struct {
	profile Profile
	fetch   FetchFunc
}

func NewFetch(profile Profile, fetch FetchFunc) Fetch {
	return Fetch{profile: profile, fetch: fetch}
}

func (f Fetch) FetchFinancialStatements(ctx context.Context, input FetchInput) (FetchResult, error) {
	if f.fetch == nil {
		return FetchResult{}, oops.In("provider_role").With("role", provider.RoleFinancials).New("financials fetch role is not configured")
	}
	return f.fetch(ctx, input)
}

func (f Fetch) FinancialsProfile() Profile {
	return f.profile
}

func (f Fetch) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.profile.RoleProfile(),
		Impl:    f,
	}
}
