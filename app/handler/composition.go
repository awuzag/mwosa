package handler

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	compositionrole "github.com/awuzag/mwosa/providers/core/composition"
	"github.com/awuzag/mwosa/service/composition"
)

type Composition struct {
	service composition.Service
}

func NewComposition(service composition.Service) Composition {
	return Composition{service: service}
}

type ListConstituentsRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	Limit          int
}

func (h Composition) List(ctx context.Context, req ListConstituentsRequest) (ConstituentsOutput, error) {
	result, err := h.service.List(ctx, composition.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
		Limit:          req.Limit,
	})
	if err != nil {
		return ConstituentsOutput{}, err
	}
	return ConstituentsOutput{Result: result}, nil
}

type ConstituentsOutput struct {
	Result compositionrole.ListResult
}

func (o ConstituentsOutput) JSONValue() any {
	return o.Result.Composition
}

func (o ConstituentsOutput) NDJSONRows() any {
	return o.Result.Composition.Members
}

func (o ConstituentsOutput) CSVRows() any {
	rows := make([]constituentOutputRow, 0, len(o.Result.Composition.Members))
	for _, member := range o.Result.Composition.Members {
		rows = append(rows, constituentOutputRowFromMember(o.Result.Composition.Subject, member))
	}
	return rows
}

func (o ConstituentsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Composition.Members))
	for _, member := range o.Result.Composition.Members {
		row := constituentOutputRowFromMember(o.Result.Composition.Subject, member)
		rows = append(rows, []string{row.SubjectSymbol, row.Symbol, row.Name, row.Weight, row.Valuation, row.Currency, row.Quantity})
	}
	return []string{"subject_symbol", "symbol", "name", "weight", "valuation", "currency", "quantity"}, rows
}

type constituentOutputRow struct {
	SubjectSymbol string `json:"subject_symbol" csv:"subject_symbol"`
	Symbol        string `json:"symbol" csv:"symbol"`
	Name          string `json:"name" csv:"name"`
	Weight        string `json:"weight,omitempty" csv:"weight"`
	Valuation     string `json:"valuation,omitempty" csv:"valuation"`
	Currency      string `json:"currency,omitempty" csv:"currency"`
	Quantity      string `json:"quantity,omitempty" csv:"quantity"`
}

func constituentOutputRowFromMember(subject compositionrole.InstrumentRef, member compositionrole.CompositionMember) constituentOutputRow {
	return constituentOutputRow{
		SubjectSymbol: subject.Symbol,
		Symbol:        member.Instrument.Symbol,
		Name:          member.Instrument.Name,
		Weight:        member.Weight.Value,
		Valuation:     member.Valuation.Value,
		Currency:      member.Valuation.Currency,
		Quantity:      member.Quantity.Value,
	}
}
