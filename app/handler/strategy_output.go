package handler

import (
	"fmt"
	"strings"

	universecore "github.com/ev3rlit/mwosa/packages/universe"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	universeservice "github.com/ev3rlit/mwosa/service/universe"
)

type DeleteStrategyResult struct {
	Name    string `json:"name" csv:"name"`
	Deleted bool   `json:"deleted" csv:"deleted"`
}

type strategySummary struct {
	Name         string `json:"name" csv:"name"`
	Engine       string `json:"engine" csv:"engine"`
	InputDataset string `json:"input_dataset" csv:"input_dataset"`
	Version      int    `json:"version" csv:"version"`
	QueryHash    string `json:"query_hash" csv:"query_hash"`
	SpecHash     string `json:"spec_hash" csv:"spec_hash"`
}

type StrategyDetailOutput struct {
	Detail strategyservice.StrategyDetail
}

func (o StrategyDetailOutput) JSONValue() any {
	return o.Detail
}

func (o StrategyDetailOutput) NDJSONRows() any {
	return o.Detail
}

func (o StrategyDetailOutput) CSVRows() any {
	return []strategySummary{strategySummaryFromDetail(o.Detail)}
}

func (o StrategyDetailOutput) TableRows() ([]string, [][]string) {
	detail := o.Detail
	return []string{"name", "engine", "input", "version", "query_hash", "spec_hash"}, [][]string{{
		detail.Strategy.Name,
		string(detail.Strategy.Engine),
		detail.ActiveVersion.InputDataset,
		fmt.Sprint(detail.ActiveVersion.Version),
		detail.ActiveVersion.QueryHash,
		detail.ActiveVersion.SpecHash,
	}}
}

type StrategyListOutput []strategyservice.StrategyDetail

func (o StrategyListOutput) JSONValue() any {
	return []strategyservice.StrategyDetail(o)
}

func (o StrategyListOutput) NDJSONRows() any {
	return []strategyservice.StrategyDetail(o)
}

func (o StrategyListOutput) CSVRows() any {
	rows := make([]strategySummary, 0, len(o))
	for _, detail := range o {
		rows = append(rows, strategySummaryFromDetail(detail))
	}
	return rows
}

func (o StrategyListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, detail := range o {
		rows = append(rows, []string{
			detail.Strategy.Name,
			string(detail.Strategy.Engine),
			detail.ActiveVersion.InputDataset,
			fmt.Sprint(detail.ActiveVersion.Version),
			detail.ActiveVersion.QueryHash,
			detail.ActiveVersion.SpecHash,
		})
	}
	return []string{"name", "engine", "input", "version", "query_hash", "spec_hash"}, rows
}

func strategySummaryFromDetail(detail strategyservice.StrategyDetail) strategySummary {
	return strategySummary{
		Name:         detail.Strategy.Name,
		Engine:       string(detail.Strategy.Engine),
		InputDataset: detail.ActiveVersion.InputDataset,
		Version:      detail.ActiveVersion.Version,
		QueryHash:    detail.ActiveVersion.QueryHash,
		SpecHash:     detail.ActiveVersion.SpecHash,
	}
}

func (r DeleteStrategyResult) CSVRows() any {
	return []DeleteStrategyResult{r}
}

func (r DeleteStrategyResult) TableRows() ([]string, [][]string) {
	return []string{"name", "deleted"}, [][]string{{r.Name, fmt.Sprint(r.Deleted)}}
}

type ScreenRunHistoryOutput []strategyservice.ScreenRun

func (o ScreenRunHistoryOutput) JSONValue() any {
	return []strategyservice.ScreenRun(o)
}

func (o ScreenRunHistoryOutput) NDJSONRows() any {
	return []strategyservice.ScreenRun(o)
}

func (o ScreenRunHistoryOutput) CSVRows() any {
	return []strategyservice.ScreenRun(o)
}

func (o ScreenRunHistoryOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, run := range o {
		rows = append(rows, []string{
			run.ID,
			run.Alias,
			string(run.Status),
			run.InputDataset,
			fmt.Sprint(run.ResultCount),
			run.StartedAt.Format(timeLayout),
		})
	}
	return []string{"id", "alias", "status", "input", "results", "started"}, rows
}

type ScreenRunDetailOutput struct {
	Detail strategyservice.ScreenRunDetail
}

func (o ScreenRunDetailOutput) JSONValue() any {
	return o.Detail
}

func (o ScreenRunDetailOutput) NDJSONRows() any {
	return o.Detail.Items
}

func (o ScreenRunDetailOutput) CSVRows() any {
	return o.Detail.Items
}

func (o ScreenRunDetailOutput) TableRows() ([]string, [][]string) {
	detail := o.Detail
	rows := make([][]string, 0, len(detail.Items))
	for _, item := range detail.Items {
		rows = append(rows, []string{fmt.Sprint(item.Ordinal), item.Symbol, string(item.PayloadJSON)})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", "", fmt.Sprintf("screen %s %s with %d results", detail.Run.ID, detail.Run.Status, detail.Run.ResultCount)})
	}
	return []string{"ordinal", "symbol", "payload"}, rows
}

type ScreenResultOutput struct {
	Result strategyservice.ScreenResult
}

type ScreenPipelineOutput struct {
	Result universeservice.ScreenPipelineResult
}

type ScreenStrategyComparisonOutput struct {
	Result strategyservice.ScreenStrategyComparison
}

type MarketRegimeOutput struct {
	Result universecore.MarketRegimeResult
}

type StrategySetOutput struct {
	Result universeservice.StrategySetSelectionResult
}

func (o ScreenResultOutput) JSONValue() any {
	return o.Result
}

func (o ScreenResultOutput) NDJSONRows() any {
	return o.Result.Items
}

func (o ScreenResultOutput) CSVRows() any {
	return o.Result.Items
}

func (o ScreenResultOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	rows := make([][]string, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, []string{fmt.Sprint(item.Ordinal), item.Symbol, string(item.PayloadJSON)})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", "", fmt.Sprintf("screen %s with %d results", result.QueryHash, result.ResultCount)})
	}
	return []string{"ordinal", "symbol", "payload"}, rows
}

func (o ScreenPipelineOutput) JSONValue() any {
	return o.Result
}

func (o ScreenPipelineOutput) NDJSONRows() any {
	return o.Result.Candidates
}

func (o ScreenPipelineOutput) CSVRows() any {
	return o.Result.Candidates
}

func (o ScreenPipelineOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"kind", "market", "security_type", "as_of", "symbols", "steps"}, [][]string{{
		result.Kind,
		result.Market,
		result.SecurityType,
		result.AsOf,
		fmt.Sprint(result.ResultCount),
		fmt.Sprint(len(result.Explain.Steps)),
	}}
}

func (o ScreenStrategyComparisonOutput) JSONValue() any {
	return o.Result
}

func (o ScreenStrategyComparisonOutput) NDJSONRows() any {
	return o.Result.Strategies
}

func (o ScreenStrategyComparisonOutput) CSVRows() any {
	return o.Result.Strategies
}

func (o ScreenStrategyComparisonOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Strategies))
	for _, item := range o.Result.Strategies {
		rows = append(rows, []string{
			item.StrategyName,
			fmt.Sprint(item.Version),
			item.SpecHash,
			item.DataAsOf,
			fmt.Sprint(item.ResultCount),
			strings.Join(item.TopSymbols, ","),
			formatFloatPtr(item.Metrics.AverageReturn20D),
			formatFloatPtr(item.Metrics.MedianReturn20D),
			formatFloatPtr(item.Metrics.AverageMaxDD20D),
			formatFloatPtr(item.Metrics.MedianMaxDD20D),
			formatFloatPtr(item.Metrics.AverageTradedAmount),
			overlapSummary(o.Result.Overlaps, item.StrategyName),
		})
	}
	return []string{"strategy", "version", "spec_hash", "as_of", "count", "top_symbols", "avg_return_20d", "median_return_20d", "avg_max_dd_20d", "median_max_dd_20d", "avg_traded_amount", "top_overlap"}, rows
}

func formatFloatPtr(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.6g", *value)
}

func overlapSummary(overlaps []strategyservice.ScreenStrategyOverlap, strategyName string) string {
	parts := make([]string, 0)
	for _, overlap := range overlaps {
		switch strategyName {
		case overlap.LeftStrategy:
			parts = append(parts, fmt.Sprintf("%s:%d", overlap.RightStrategy, overlap.Count))
		case overlap.RightStrategy:
			parts = append(parts, fmt.Sprintf("%s:%d", overlap.LeftStrategy, overlap.Count))
		}
	}
	return strings.Join(parts, ",")
}

func (o MarketRegimeOutput) JSONValue() any {
	return o.Result
}

func (o MarketRegimeOutput) NDJSONRows() any {
	return o.Result
}

func (o MarketRegimeOutput) CSVRows() any {
	return []universecore.MarketRegimeResult{o.Result}
}

func (o MarketRegimeOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"name", "as_of", "benchmark", "regime", "return_20d", "ma20", "ma60"}, [][]string{{
		result.Name,
		result.AsOf,
		result.Benchmark.Symbol,
		result.Regime,
		fmt.Sprintf("%.6g", result.Metrics.Return20D),
		fmt.Sprintf("%.6g", result.Metrics.MA20),
		fmt.Sprintf("%.6g", result.Metrics.MA60),
	}}
}

func (o StrategySetOutput) JSONValue() any {
	return o.Result
}

func (o StrategySetOutput) NDJSONRows() any {
	return o.Result
}

func (o StrategySetOutput) CSVRows() any {
	return []universeservice.StrategySetSelectionResult{o.Result}
}

func (o StrategySetOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"name", "as_of", "regime", "strategy", "version", "spec_hash"}, [][]string{{
		result.Name,
		result.AsOf,
		result.Regime.Regime,
		result.SelectedRoute.Strategy,
		result.SelectedRoute.Version,
		result.SelectedRoute.SpecHash,
	}}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
