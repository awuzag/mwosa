package handler

import (
	"fmt"

	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
)

type DailyBarsOutput []dailybar.Bar

func (o DailyBarsOutput) JSONValue() any {
	return []dailybar.Bar(o)
}

func (o DailyBarsOutput) NDJSONRows() any {
	return []dailybar.Bar(o)
}

func (o DailyBarsOutput) CSVRows() any {
	rows := make([]dailyBarOutputRow, 0, len(o))
	for _, bar := range o {
		rows = append(rows, dailyBarOutputRowFromBar(bar))
	}
	return rows
}

func (o DailyBarsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, bar := range o {
		rows = append(rows, []string{bar.TradingDate, bar.Symbol, bar.Name, bar.Open, bar.High, bar.Low, bar.Close, bar.Change})
	}
	return []string{"date", "symbol", "name", "open", "high", "low", "close", "change"}, rows
}

type DailyStorageSummaryOutput struct {
	Result daily.StorageSummaryResult
}

func (o DailyStorageSummaryOutput) JSONValue() any {
	return o.Result
}

func (o DailyStorageSummaryOutput) NDJSONRows() any {
	return o.Result
}

func (o DailyStorageSummaryOutput) CSVRows() any {
	return []daily.StorageSummaryResult{o.Result}
}

func (o DailyStorageSummaryOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"record_type", "market", "security_type", "symbols", "bars", "from", "to", "dates"}, [][]string{{
		result.RecordType,
		string(result.Market),
		string(result.SecurityType),
		fmt.Sprint(result.Symbols),
		fmt.Sprint(result.Bars),
		result.From,
		result.To,
		fmt.Sprint(result.Dates),
	}}
}

type DailyCoverageOutput struct {
	Result daily.CoverageResult
}

func (o DailyCoverageOutput) JSONValue() any {
	return o.Result
}

func (o DailyCoverageOutput) NDJSONRows() any {
	return o.Result
}

func (o DailyCoverageOutput) CSVRows() any {
	return []daily.CoverageResult{o.Result}
}

func (o DailyCoverageOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"market", "security_type", "symbol", "name", "from", "to", "bars", "dates"}, [][]string{{
		string(result.Market),
		string(result.SecurityType),
		result.Symbol,
		result.Name,
		result.From,
		result.To,
		fmt.Sprint(result.Bars),
		fmt.Sprint(result.Dates),
	}}
}

type dailyBarOutputRow struct {
	Date   string `json:"date" csv:"date"`
	Symbol string `json:"symbol" csv:"symbol"`
	Name   string `json:"name" csv:"name"`
	Open   string `json:"open" csv:"open"`
	High   string `json:"high" csv:"high"`
	Low    string `json:"low" csv:"low"`
	Close  string `json:"close" csv:"close"`
	Change string `json:"change" csv:"change"`
}

func dailyBarOutputRowFromBar(bar dailybar.Bar) dailyBarOutputRow {
	return dailyBarOutputRow{
		Date:   bar.TradingDate,
		Symbol: bar.Symbol,
		Name:   bar.Name,
		Open:   bar.Open,
		High:   bar.High,
		Low:    bar.Low,
		Close:  bar.Close,
		Change: bar.Change,
	}
}

type CollectResultOutput struct {
	Result daily.CollectResult
}

func (o CollectResultOutput) JSONValue() any {
	return o.Result
}

func (o CollectResultOutput) NDJSONRows() any {
	return o.Result
}

func (o CollectResultOutput) CSVRows() any {
	return []daily.CollectResult{o.Result}
}

func (o CollectResultOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"market", "security_type", "provider", "group", "dates", "fetched", "stored", "rows_affected"}, [][]string{{
		string(result.Market),
		string(result.SecurityType),
		string(result.ProviderID),
		string(result.Group),
		fmt.Sprint(len(result.Dates)),
		fmt.Sprint(result.BarsFetched),
		fmt.Sprint(result.BarsStored),
		fmt.Sprint(result.RowsAffected),
	}}
}
