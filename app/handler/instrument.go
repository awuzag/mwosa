package handler

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
	instrumentrole "github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/service/instrument"
)

type Instrument struct {
	service instrument.Service
}

func NewInstrument(service instrument.Service) Instrument {
	return Instrument{service: service}
}

type ListInstrumentsRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Query          string
	Limit          int
}

type InspectInstrumentRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
}

func (h Instrument) List(ctx context.Context, req ListInstrumentsRequest) (InstrumentsOutput, error) {
	result, err := h.service.Search(ctx, instrument.SearchRequest{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Query:          req.Query,
		Limit:          req.Limit,
	})
	if err != nil {
		return InstrumentsOutput{}, err
	}
	return InstrumentsOutput{Result: result}, nil
}

func (h Instrument) Inspect(ctx context.Context, req InspectInstrumentRequest) (InstrumentOutput, error) {
	result, err := h.service.Inspect(ctx, instrument.InspectRequest{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return InstrumentOutput{}, err
	}
	return InstrumentOutput{Result: result}, nil
}

type InstrumentsOutput struct {
	Result instrumentrole.SearchResult
}

func (o InstrumentsOutput) JSONValue() any {
	return o.Result
}

func (o InstrumentsOutput) NDJSONRows() any {
	return o.Result.Instruments
}

func (o InstrumentsOutput) CSVRows() any {
	rows := make([]instrumentOutputRow, 0, len(o.Result.Instruments))
	for _, item := range o.Result.Instruments {
		rows = append(rows, instrumentOutputRowFromInstrument(item))
	}
	return rows
}

func (o InstrumentsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Instruments))
	for _, item := range o.Result.Instruments {
		row := instrumentOutputRowFromInstrument(item)
		rows = append(rows, []string{row.Provider, row.SecurityType, row.SecurityCode, row.ISIN, row.Name})
	}
	return []string{"provider", "security_type", "symbol", "isin", "name"}, rows
}

type InstrumentOutput struct {
	Result instrument.InspectResult
}

func (o InstrumentOutput) JSONValue() any {
	return o.Result
}

func (o InstrumentOutput) NDJSONRows() any {
	return []instrumentrole.Instrument{o.Result.Instrument}
}

func (o InstrumentOutput) CSVRows() any {
	return []instrumentOutputRow{instrumentOutputRowFromInstrument(o.Result.Instrument)}
}

func (o InstrumentOutput) TableRows() ([]string, [][]string) {
	row := instrumentOutputRowFromInstrument(o.Result.Instrument)
	return []string{"provider", "security_type", "symbol", "isin", "name"}, [][]string{{
		row.Provider,
		row.SecurityType,
		row.SecurityCode,
		row.ISIN,
		row.Name,
	}}
}

type instrumentOutputRow struct {
	Provider     string `json:"provider" csv:"provider"`
	SecurityType string `json:"security_type" csv:"security_type"`
	SecurityCode string `json:"security_code" csv:"security_code"`
	ISIN         string `json:"isin" csv:"isin"`
	Name         string `json:"name" csv:"name"`
}

func instrumentOutputRowFromInstrument(item instrumentrole.Instrument) instrumentOutputRow {
	return instrumentOutputRow{
		Provider:     string(item.Provider),
		SecurityType: string(item.SecurityType),
		SecurityCode: item.SecurityCode,
		ISIN:         item.ISIN,
		Name:         item.Name,
	}
}
