package financials

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
)

func TestNewServiceRequiresRouter(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatal("NewService error = nil, want router error")
	}
	if !strings.Contains(err.Error(), "router") {
		t.Fatalf("error = %q, want router context", err.Error())
	}
}

func TestGetRoutesAndFetchesFinancialStatements(t *testing.T) {
	var gotFetch financialsrole.FetchInput
	fetcher := financialsrole.NewFetch(financialsrole.Profile{}, func(_ context.Context, input financialsrole.FetchInput) (financialsrole.FetchResult, error) {
		gotFetch = input
		return financialsrole.FetchResult{
			Statements: []financialsrole.Statement{
				{
					Statement:  financialsrole.StatementTypeIncomeStatement,
					Symbol:     input.Symbol,
					FiscalYear: input.FiscalYear,
					Period:     input.Period,
					Lines: []financialsrole.LineItem{
						{AccountName: "Revenue", Value: "1000"},
					},
				},
			},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
			Group:    provider.GroupID("fakeGroup"),
		}, nil
	})
	router := &fakeFinancialsRouter{fetcher: fetcher}
	service, err := NewService(router)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Get(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		FiscalYear:   "2025",
		Period:       financialsrole.PeriodTypeAnnual,
		Statement:    financialsrole.StatementTypeIncomeStatement,
	})
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}

	if router.gotRoute.ProviderID != provider.ProviderID("fake") || router.gotRoute.Symbol != "005930" {
		t.Fatalf("route input = %+v, want provider and symbol", router.gotRoute)
	}
	if gotFetch.Symbol != "005930" || gotFetch.FiscalYear != "2025" || gotFetch.Statement != financialsrole.StatementTypeIncomeStatement {
		t.Fatalf("fetch input = %+v, want symbol, year, and statement", gotFetch)
	}
	if len(result.Statements) != 1 || result.Statements[0].Lines[0].Value != "1000" {
		t.Fatalf("result = %+v, want one statement line", result)
	}
}

func TestGetRequiresSymbol(t *testing.T) {
	service, err := NewService(&fakeFinancialsRouter{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Get(context.Background(), Request{})
	if err == nil {
		t.Fatal("Get error = nil, want symbol error")
	}
	if !strings.Contains(err.Error(), "requires symbol") {
		t.Fatalf("error = %q, want symbol context", err.Error())
	}
}

func TestGetRejectsUnsupportedPeriod(t *testing.T) {
	service, err := NewService(&fakeFinancialsRouter{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Get(context.Background(), Request{Symbol: "005930", Period: financialsrole.PeriodType("monthly")})
	if err == nil {
		t.Fatal("Get error = nil, want period error")
	}
	if !strings.Contains(err.Error(), "unsupported financials period") {
		t.Fatalf("error = %q, want period context", err.Error())
	}
}

func TestGetReportsNotFoundWhenProviderReturnsNoStatements(t *testing.T) {
	fetcher := financialsrole.NewFetch(financialsrole.Profile{}, func(context.Context, financialsrole.FetchInput) (financialsrole.FetchResult, error) {
		return financialsrole.FetchResult{}, nil
	})
	service, err := NewService(&fakeFinancialsRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Get(context.Background(), Request{Symbol: "005930"})
	if err == nil {
		t.Fatal("Get error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "financial statements not found") {
		t.Fatalf("error = %q, want not found context", err.Error())
	}
}

type fakeFinancialsRouter struct {
	fetcher  financialsrole.Fetcher
	gotRoute financialsrole.RouteInput
}

func (r *fakeFinancialsRouter) RouteFinancialStatements(_ context.Context, input financialsrole.RouteInput) (financialsrole.Fetcher, error) {
	r.gotRoute = input
	return r.fetcher, nil
}
