package handler

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	financialsservice "github.com/awuzag/mwosa/service/financials"
)

type Financials struct {
	service financialsservice.Service
}

func NewFinancials(service financialsservice.Service) Financials {
	return Financials{service: service}
}

type GetFinancialsRequest struct {
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

func (h Financials) Get(ctx context.Context, req GetFinancialsRequest) (FinancialStatementsOutput, error) {
	result, err := h.service.Get(ctx, financialsservice.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
		FiscalYear:     req.FiscalYear,
		Period:         req.Period,
		Statement:      req.Statement,
		Limit:          req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return FinancialStatementsOutput(result.Statements), nil
}
