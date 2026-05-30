package financials

import (
	"context"
	"fmt"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/samber/oops"
)

type Router interface {
	RouteFinancialStatements(ctx context.Context, input financialsrole.RouteInput) (financialsrole.Fetcher, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("financials_service").New("financials service router is nil")
	}
	return Service{router: router}, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	FiscalYear     string
	Period         financialsrole.PeriodType
	Statement      financialsrole.StatementType
	Limit          int
}

type NotFoundError struct {
	Symbol       string
	Market       provider.Market
	SecurityType provider.SecurityType
	FiscalYear   string
	Period       financialsrole.PeriodType
	Statement    financialsrole.StatementType
}

func (e *NotFoundError) Error() string {
	parts := []string{"financial statements not found"}
	if e.Market != "" {
		parts = append(parts, fmt.Sprintf("market=%s", e.Market))
	}
	if e.SecurityType != "" {
		parts = append(parts, fmt.Sprintf("security_type=%s", e.SecurityType))
	}
	if e.Symbol != "" {
		parts = append(parts, fmt.Sprintf("symbol=%s", e.Symbol))
	}
	if e.FiscalYear != "" {
		parts = append(parts, fmt.Sprintf("fiscal_year=%s", e.FiscalYear))
	}
	if e.Period != "" {
		parts = append(parts, fmt.Sprintf("period=%s", e.Period))
	}
	if e.Statement != "" {
		parts = append(parts, fmt.Sprintf("statement=%s", e.Statement))
	}
	return strings.Join(parts, " ")
}

func (s Service) Get(ctx context.Context, req Request) (financialsrole.FetchResult, error) {
	req.Symbol = strings.TrimSpace(req.Symbol)
	req.FiscalYear = strings.TrimSpace(req.FiscalYear)
	errb := oops.In("financials_service").With(
		"provider", req.ProviderID,
		"prefer_provider", req.PreferProvider,
		"market", req.Market,
		"security_type", req.SecurityType,
		"symbol", req.Symbol,
		"fiscal_year", req.FiscalYear,
		"period", req.Period,
		"statement", req.Statement,
	)
	if req.Symbol == "" {
		return financialsrole.FetchResult{}, errb.New("financials request requires symbol")
	}
	if req.Period != "" && req.Period != financialsrole.PeriodTypeAnnual && req.Period != financialsrole.PeriodTypeQuarter {
		return financialsrole.FetchResult{}, errb.Errorf("unsupported financials period: %s", req.Period)
	}
	if req.Statement != "" &&
		req.Statement != financialsrole.StatementTypeSummary &&
		req.Statement != financialsrole.StatementTypeIncomeStatement &&
		req.Statement != financialsrole.StatementTypeBalanceSheet &&
		req.Statement != financialsrole.StatementTypeCashFlow {
		return financialsrole.FetchResult{}, errb.Errorf("unsupported financial statement type: %s", req.Statement)
	}
	if req.Limit < 0 {
		return financialsrole.FetchResult{}, errb.Errorf("financials limit must not be negative: %d", req.Limit)
	}
	if s.router == nil {
		return financialsrole.FetchResult{}, errb.New("financials service router is nil")
	}

	fetcher, err := s.router.RouteFinancialStatements(ctx, financialsrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return financialsrole.FetchResult{}, errb.Wrapf(err, "route financial statements")
	}

	result, err := fetcher.FetchFinancialStatements(ctx, financialsrole.FetchInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
		FiscalYear:   req.FiscalYear,
		Period:       req.Period,
		Statement:    req.Statement,
		Limit:        req.Limit,
	})
	if err != nil {
		return financialsrole.FetchResult{}, errb.Wrapf(err, "fetch financial statements")
	}
	if len(result.Statements) == 0 {
		return financialsrole.FetchResult{}, errb.Wrap(&NotFoundError{
			Symbol:       req.Symbol,
			Market:       req.Market,
			SecurityType: req.SecurityType,
			FiscalYear:   req.FiscalYear,
			Period:       req.Period,
			Statement:    req.Statement,
		})
	}
	return result, nil
}
