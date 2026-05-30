package handler

import (
	"fmt"

	"github.com/awuzag/mwosa/providers/core/indexbar"
	indexservice "github.com/awuzag/mwosa/service/index"
)

type IndexBarsOutput []indexbar.Bar

func (o IndexBarsOutput) JSONValue() any {
	if len(o) == 1 {
		return o[0]
	}
	return []indexbar.Bar(o)
}

func (o IndexBarsOutput) NDJSONRows() any {
	return []indexbar.Bar(o)
}

func (o IndexBarsOutput) CSVRows() any {
	return []indexbar.Bar(o)
}

func (o IndexBarsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, bar := range o {
		rows = append(rows, []string{
			string(bar.Market),
			bar.IndexCode,
			bar.Name,
			bar.TradingDate,
			bar.Open,
			bar.High,
			bar.Low,
			bar.Close,
			bar.Change,
			bar.ChangeRate,
			bar.Volume,
			bar.TradedValue,
			string(bar.Provider),
			string(bar.Group),
			string(bar.Operation),
		})
	}
	return []string{"market", "index_code", "name", "date", "open", "high", "low", "close", "change", "change_rate", "volume", "traded_amount", "provider", "group", "operation"}, rows
}

type IndexCollectResultOutput struct {
	Result indexservice.CollectResult
}

func (o IndexCollectResultOutput) JSONValue() any {
	return o.Result
}

func (o IndexCollectResultOutput) NDJSONRows() any {
	return []indexservice.CollectResult{o.Result}
}

func (o IndexCollectResultOutput) CSVRows() any {
	return []indexservice.CollectResult{o.Result}
}

func (o IndexCollectResultOutput) TableRows() ([]string, [][]string) {
	return []string{"market", "provider", "group", "index_code", "dates", "fetched", "stored", "rows_affected"}, [][]string{{
		string(o.Result.Market),
		string(o.Result.ProviderID),
		string(o.Result.Group),
		o.Result.IndexCode,
		fmt.Sprint(len(o.Result.Dates)),
		fmt.Sprint(o.Result.BarsFetched),
		fmt.Sprint(o.Result.BarsStored),
		fmt.Sprint(o.Result.RowsAffected),
	}}
}
