package handler

import (
	"encoding/json"
	"fmt"

	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
)

type AggregateValidationOutput struct {
	Spec  aggregateservice.Spec `json:"spec"`
	Valid bool                  `json:"valid" csv:"valid"`
	View  string                `json:"-"`
}

func (o AggregateValidationOutput) JSONValue() any {
	if o.View == "raw" {
		return o.Spec
	}
	return o
}

func (o AggregateValidationOutput) TableRows() ([]string, [][]string) {
	return []string{"name", "valid", "stages", "output"}, [][]string{{
		o.Spec.Name,
		fmt.Sprint(o.Valid),
		fmt.Sprint(len(o.Spec.Pipeline)),
		o.Spec.Output.From,
	}}
}

type aggregateSummary struct {
	Name     string `json:"name" csv:"name"`
	Version  int    `json:"version" csv:"version"`
	SpecHash string `json:"spec_hash" csv:"spec_hash"`
	Archived string `json:"archived,omitempty" csv:"archived"`
}

type AggregateDetailOutput struct {
	Detail aggregateservice.Detail
	View   string
}

func (o AggregateDetailOutput) JSONValue() any {
	if o.View == "versions" {
		return o.Detail.Versions
	}
	return o.Detail
}

func (o AggregateDetailOutput) NDJSONRows() any {
	return o.Detail
}

func (o AggregateDetailOutput) CSVRows() any {
	return []aggregateSummary{aggregateSummaryFromDetail(o.Detail)}
}

func (o AggregateDetailOutput) TableRows() ([]string, [][]string) {
	if o.View == "versions" {
		rows := make([][]string, 0, len(o.Detail.Versions))
		for _, version := range o.Detail.Versions {
			rows = append(rows, []string{fmt.Sprint(version.Version), version.ID, version.SpecHash, version.CreatedAt.Format(timeLayout), version.Note})
		}
		return []string{"version", "id", "spec_hash", "created_at", "note"}, rows
	}
	summary := aggregateSummaryFromDetail(o.Detail)
	return []string{"name", "version", "spec_hash", "archived"}, [][]string{{
		summary.Name,
		fmt.Sprint(summary.Version),
		summary.SpecHash,
		summary.Archived,
	}}
}

type AggregateListOutput []aggregateservice.Detail

func (o AggregateListOutput) JSONValue() any {
	return []aggregateservice.Detail(o)
}

func (o AggregateListOutput) NDJSONRows() any {
	return []aggregateservice.Detail(o)
}

func (o AggregateListOutput) CSVRows() any {
	rows := make([]aggregateSummary, 0, len(o))
	for _, detail := range o {
		rows = append(rows, aggregateSummaryFromDetail(detail))
	}
	return rows
}

func (o AggregateListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, detail := range o {
		summary := aggregateSummaryFromDetail(detail)
		rows = append(rows, []string{summary.Name, fmt.Sprint(summary.Version), summary.SpecHash, summary.Archived})
	}
	return []string{"name", "version", "spec_hash", "archived"}, rows
}

func aggregateSummaryFromDetail(detail aggregateservice.Detail) aggregateSummary {
	archived := ""
	if detail.Aggregate.ArchivedAt != nil {
		archived = detail.Aggregate.ArchivedAt.Format(timeLayout)
	}
	return aggregateSummary{
		Name:     detail.Aggregate.Name,
		Version:  detail.ActiveVersion.Version,
		SpecHash: detail.ActiveVersion.SpecHash,
		Archived: archived,
	}
}

type AggregatePlanOutput struct {
	Plan aggregateservice.Plan
	View string
}

func (o AggregatePlanOutput) JSONValue() any {
	switch o.View {
	case "stages":
		return o.Plan.Stages
	case "pipeline":
		return o.Plan.Stages
	case "raw":
		return o.Plan
	}
	return o.Plan
}

func (o AggregatePlanOutput) TableRows() ([]string, [][]string) {
	switch o.View {
	case "stages":
		rows := make([][]string, 0, len(o.Plan.Stages))
		for _, stage := range o.Plan.Stages {
			rows = append(rows, []string{stage.Name, string(stage.Type), stage.From, stage.Collection, stage.Dataset, stage.Provider, stage.Operation})
		}
		return []string{"name", "type", "from", "collection", "dataset", "provider", "operation"}, rows
	case "pipeline":
		rows := make([][]string, 0, len(o.Plan.Stages))
		for _, stage := range o.Plan.Stages {
			rows = append(rows, []string{stage.Name, string(stage.Type), fmt.Sprint(len(stage.Pipeline)), stage.Query})
		}
		return []string{"name", "type", "mongo_stages", "jq_query"}, rows
	}
	return []string{"name", "params", "stages", "output", "spec_hash"}, [][]string{{
		o.Plan.Name,
		fmt.Sprint(len(o.Plan.Params)),
		fmt.Sprint(len(o.Plan.Stages)),
		o.Plan.Output.From,
		o.Plan.SpecHash,
	}}
}

type AggregateRunOutput struct {
	Detail aggregateservice.RunDetail
	Rows   aggregateservice.OutputRows
}

func (o AggregateRunOutput) JSONValue() any {
	if len(o.Rows.Rows) > 0 {
		return o.Rows.JSONValue()
	}
	return o.Detail
}

func (o AggregateRunOutput) NDJSONRows() any {
	if len(o.Rows.Rows) > 0 {
		return o.Rows.NDJSONRows()
	}
	return o.Detail.Items
}

func (o AggregateRunOutput) CSVRows() any {
	if len(o.Rows.Rows) > 0 {
		return o.Rows.CSVRows()
	}
	return o.Detail.Items
}

func (o AggregateRunOutput) TableRows() ([]string, [][]string) {
	if len(o.Rows.Rows) > 0 {
		return o.Rows.TableRows()
	}
	return AggregateRunDetailOutput{Detail: o.Detail}.TableRows()
}

func (o AggregateRunOutput) TableAlignments() []string {
	if len(o.Rows.Rows) > 0 {
		return o.Rows.TableAlignments()
	}
	return nil
}

type AggregateRunHistoryOutput []aggregateservice.Run

func (o AggregateRunHistoryOutput) JSONValue() any {
	return []aggregateservice.Run(o)
}

func (o AggregateRunHistoryOutput) NDJSONRows() any {
	return []aggregateservice.Run(o)
}

func (o AggregateRunHistoryOutput) CSVRows() any {
	return []aggregateservice.Run(o)
}

func (o AggregateRunHistoryOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, run := range o {
		rows = append(rows, []string{
			run.ID,
			run.Alias,
			run.AggregateName,
			string(run.Status),
			fmt.Sprint(run.ResultCount),
			run.StartedAt.Format(timeLayout),
		})
	}
	return []string{"id", "alias", "aggregate", "status", "results", "started"}, rows
}

type AggregateRunDetailOutput struct {
	Detail aggregateservice.RunDetail
	View   string
}

func (o AggregateRunDetailOutput) JSONValue() any {
	switch o.View {
	case "stages":
		return jsonRawOrString(o.Detail.Run.StagesJSON)
	case "params":
		return jsonRawOrString(o.Detail.Run.ParamsJSON)
	case "pipeline":
		return jsonRawOrString(o.Detail.Run.PipelineJSON)
	case "items":
		return o.Detail.Items
	}
	return o.Detail
}

func (o AggregateRunDetailOutput) NDJSONRows() any {
	return o.Detail.Items
}

func (o AggregateRunDetailOutput) CSVRows() any {
	return o.Detail.Items
}

func (o AggregateRunDetailOutput) TableRows() ([]string, [][]string) {
	detail := o.Detail
	switch o.View {
	case "summary":
		finished := ""
		if detail.Run.FinishedAt != nil {
			finished = detail.Run.FinishedAt.Format(timeLayout)
		}
		return []string{"id", "alias", "aggregate", "status", "results", "started", "finished", "error"}, [][]string{{
			detail.Run.ID,
			detail.Run.Alias,
			detail.Run.AggregateName,
			string(detail.Run.Status),
			fmt.Sprint(detail.Run.ResultCount),
			detail.Run.StartedAt.Format(timeLayout),
			finished,
			detail.Run.ErrorMessage,
		}}
	case "stages":
		return rawJSONTable("stages", detail.Run.StagesJSON)
	case "params":
		return rawJSONTable("params", detail.Run.ParamsJSON)
	case "pipeline":
		return rawJSONTable("pipeline", detail.Run.PipelineJSON)
	}
	rows := make([][]string, 0, len(detail.Items))
	for _, item := range detail.Items {
		rows = append(rows, []string{fmt.Sprint(item.Ordinal), string(item.PayloadJSON)})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", fmt.Sprintf("aggregate run %s %s with %d results", detail.Run.ID, detail.Run.Status, detail.Run.ResultCount)})
	}
	return []string{"ordinal", "payload"}, rows
}

type DeleteAggregateResult struct {
	Name    string `json:"name" csv:"name"`
	Deleted bool   `json:"deleted" csv:"deleted"`
}

func (r DeleteAggregateResult) CSVRows() any {
	return []DeleteAggregateResult{r}
}

func (r DeleteAggregateResult) TableRows() ([]string, [][]string) {
	return []string{"name", "deleted"}, [][]string{{r.Name, fmt.Sprint(r.Deleted)}}
}

func jsonRawOrString(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	return value
}

func rawJSONTable(label string, raw json.RawMessage) ([]string, [][]string) {
	if len(raw) == 0 {
		return []string{"view", "json"}, [][]string{{label, ""}}
	}
	return []string{"view", "json"}, [][]string{{label, string(raw)}}
}
