package handler

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	quoterole "github.com/awuzag/mwosa/providers/core/quote"
	"github.com/awuzag/mwosa/service/quote"
)

type Quote struct {
	service quote.Service
}

func NewQuote(service quote.Service) Quote {
	return Quote{service: service}
}

type GetQuoteRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
}

func (h Quote) Get(ctx context.Context, req GetQuoteRequest) (QuoteOutput, error) {
	result, err := h.service.Get(ctx, quote.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return QuoteOutput{}, err
	}
	return QuoteOutput{Result: result}, nil
}

type QuoteOutput struct {
	Result quoterole.SnapshotResult
}

func (o QuoteOutput) JSONValue() any {
	return o.Result
}

func (o QuoteOutput) NDJSONRows() any {
	return []quoterole.SnapshotResult{o.Result}
}

func (o QuoteOutput) CSVRows() any {
	return []quoteOutputRow{quoteOutputRowFromResult(o.Result)}
}

func (o QuoteOutput) TableRows() ([]string, [][]string) {
	row := quoteOutputRowFromResult(o.Result)
	return []string{"provider", "symbol", "price"}, [][]string{{
		row.Provider,
		row.Symbol,
		row.Price,
	}}
}

type quoteOutputRow struct {
	Provider string `json:"provider" csv:"provider"`
	Symbol   string `json:"symbol" csv:"symbol"`
	Price    string `json:"price" csv:"price"`
}

func quoteOutputRowFromResult(result quoterole.SnapshotResult) quoteOutputRow {
	return quoteOutputRow{
		Provider: string(result.Provider.ID),
		Symbol:   result.Symbol,
		Price:    result.Price,
	}
}
