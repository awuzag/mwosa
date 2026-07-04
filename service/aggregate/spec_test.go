package aggregate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSpecValidatesAggregateYAML(t *testing.T) {
	spec, err := LoadSpecBytes(context.Background(), []byte(`
kind: Aggregate
schema_version: 1
name: krx-candidates
params:
  as_of:
    type: date
    default: "2026-07-01"
  limit:
    type: int
    default: 20
pipeline:
  - name: universe
    type: local_collection
    collection: instruments
    filter:
      market_key: krx
      trade_date: "${params.as_of}"
  - name: ranked
    type: aggregate
    from: universe
    pipeline:
      - $sort:
          traded_amount: -1
      - $limit: "${params.limit}"
output:
  from: ranked
  default_format: table
  columns:
    - key: symbol
      title: 코드
`))
	require.NoError(t, err)
	require.NoError(t, ValidateSpec(spec))

	params, err := ApplyParams(spec, []string{"limit=5"})
	require.NoError(t, err)
	assert.Equal(t, "2026-07-01", params["as_of"])
	assert.Equal(t, int64(5), params["limit"])
}

func TestApplyParamsRejectsUnknownAndBadTypes(t *testing.T) {
	spec := minimalSpec()

	_, err := ApplyParams(spec, []string{"missing=value"})
	require.ErrorContains(t, err, "unknown aggregate param")

	_, err = ApplyParams(spec, []string{"limit=not-int"})
	require.ErrorContains(t, err, "parse aggregate param")
}

func TestResolvePlaceholdersPreservesScalarTypes(t *testing.T) {
	spec := minimalSpec()
	params, err := ApplyParams(spec, []string{"limit=7"})
	require.NoError(t, err)

	value, err := ResolveTemplateValue("${params.limit}", TemplateContext{Params: params, Each: map[string]any{"symbol": "005930"}})
	require.NoError(t, err)
	assert.Equal(t, int64(7), value)

	value, err = ResolveTemplateValue(map[string]any{
		"symbol": "${each.symbol}",
		"label":  "watch-${each.symbol}-${params.as_of}",
	}, TemplateContext{Params: params, Each: map[string]any{"symbol": "005930"}})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"symbol": "005930", "label": "watch-005930-2026-07-01"}, value)
}

func TestValidateSpecRejectsUnknownSourceAndBadAlias(t *testing.T) {
	spec := minimalSpec()
	spec.Pipeline[0].Type = "sqlite"
	require.ErrorContains(t, ValidateSpec(spec), "unsupported aggregate stage type")

	spec = minimalSpec()
	spec.Pipeline = append(spec.Pipeline, StageSpec{Name: "bad", Type: StageAggregate, From: "missing"})
	require.ErrorContains(t, ValidateSpec(spec), "unknown aggregate stage input")
}

func TestMongoPipelinePolicyAllowsReadStagesAndBlocksSideEffects(t *testing.T) {
	require.NoError(t, ValidateMongoPipeline([]map[string]any{
		{"$lookup": map[string]any{"from": "universe", "localField": "symbol", "foreignField": "symbol", "as": "rows"}},
		{"$group": map[string]any{"_id": "$symbol"}},
		{"$setWindowFields": map[string]any{"sortBy": map[string]any{"date": 1}}},
	}, map[string]string{"universe": "aggregate_tmp_run_universe"}))

	err := ValidateMongoPipeline([]map[string]any{{"$merge": "daily_bars"}}, nil)
	require.ErrorContains(t, err, "blocked mongodb aggregation stage")

	err = ValidateMongoPipeline([]map[string]any{{"$lookup": map[string]any{"from": "not_a_stage"}}}, map[string]string{"universe": "aggregate_tmp_run_universe"})
	require.ErrorContains(t, err, "unknown lookup source")
}

func TestExecuteJQRowsSuccessAndFailure(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"symbol":"005930","close":"70000"}`),
		json.RawMessage(`{"symbol":"000660","close":"130000"}`),
	}

	out, err := ExecuteJQRows(context.Background(), rows, `map({symbol, close: (.close | tonumber)})`)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.JSONEq(t, `{"symbol":"005930","close":70000}`, string(out[0]))

	_, err = ExecuteJQRows(context.Background(), rows, `map(`)
	require.ErrorContains(t, err, "execute aggregate jq")
}

func TestMongoExecutorProviderRawStageWithFakeFetcher(t *testing.T) {
	fetcher := fakeRawFetcher{}
	stageRows := map[string][]json.RawMessage{
		"universe": {
			json.RawMessage(`{"symbol":"005930"}`),
		},
	}
	executor := MongoExecutor{rawFetcher: fetcher}

	rows, err := executor.executeProviderRaw(context.Background(), StageSpec{
		Name:      "daily_windows",
		Type:      StageProviderRaw,
		Provider:  "kis",
		Operation: "inquire-daily-itemchartprice",
		Foreach:   &ForeachSpec{Stage: "universe", Field: "symbol", As: "symbol"},
		Params: map[string]any{
			"FID_INPUT_ISCD":   "${each.symbol}",
			"FID_INPUT_DATE_1": "${params.from}",
		},
	}, TemplateContext{Params: map[string]any{"from": "20250102"}}, stageRows)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.JSONEq(t, `{
		"context":{"symbol":"005930"},
		"provider":"kis",
		"provider_group":"quote",
		"operation":"inquire-daily-itemchartprice",
		"endpoint":"/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
		"row_count":1,
		"base_date":"2025-01-02",
		"payload":{"output2":[{"stck_bsop_date":"20250102"}]}
	}`, string(rows[0]))
}

func TestMongoExecutorProviderStageWithFakeFetcher(t *testing.T) {
	executor := MongoExecutor{providerFetcher: fakeProviderFetcher{}}
	stageRows := map[string][]json.RawMessage{
		"universe": {
			json.RawMessage(`{"symbol":"005930"}`),
		},
	}

	rows, err := executor.executeProvider(context.Background(), StageSpec{
		Name:     "quotes",
		Type:     StageProvider,
		Provider: "kis",
		Role:     "quote",
		Foreach:  &ForeachSpec{Stage: "universe", Field: "symbol", As: "symbol"},
		Request: map[string]any{
			"market":        "krx",
			"security_type": "stock",
			"symbol":        "${each.symbol}",
		},
	}, TemplateContext{}, stageRows)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.JSONEq(t, `{"context":{"symbol":"005930"},"provider":"kis","role":"quote","symbol":"005930","price":"75000"}`, string(rows[0]))
}

func TestFormatOutputRowsHonorsColumnsAndFormats(t *testing.T) {
	spec := minimalSpec()
	spec.Output.Columns = []OutputColumnSpec{
		{Key: "ordinal", Title: "#", Format: "integer"},
		{Key: "symbol", Title: "코드"},
		{Key: "change_pct", Title: "등락%", Format: "percent", Precision: intPtr(2)},
		{Key: "market_cap", Title: "시총(조)", Format: "number", Precision: intPtr(1)},
	}
	rows := []json.RawMessage{
		json.RawMessage(`{"symbol":"005930","change_pct":3.421,"market_cap":415.234}`),
	}

	out, err := FormatOutputRows(spec.Output, rows)
	require.NoError(t, err)
	header, table := out.TableRows()
	assert.Equal(t, []string{"#", "코드", "등락%", "시총(조)"}, header)
	assert.Equal(t, [][]string{{"1", "005930", "3.42%", "415.2"}}, table)
	assert.Equal(t, []map[string]any{{"ordinal": 1, "symbol": "005930", "change_pct": 3.421, "market_cap": 415.234}}, out.JSONValue())
}

type fakeRawFetcher struct{}

func (fakeRawFetcher) FetchRaw(_ context.Context, req RawFetchRequest) (RawFetchResult, error) {
	return RawFetchResult{
		Provider:  req.Provider,
		Group:     "quote",
		Operation: req.Operation,
		Endpoint:  "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
		Response:  map[string]any{"output2": []map[string]any{{"stck_bsop_date": req.Input["FID_INPUT_DATE_1"]}}},
		RowCount:  1,
		BaseDate:  "2025-01-02",
	}, nil
}

type fakeProviderFetcher struct{}

func (fakeProviderFetcher) FetchProvider(_ context.Context, req ProviderFetchRequest) (ProviderFetchResult, error) {
	return ProviderFetchResult{
		Provider: req.Provider,
		Role:     req.Role,
		Payload: map[string]any{
			"symbol": req.Request["symbol"],
			"price":  "75000",
		},
	}, nil
}

func minimalSpec() Spec {
	return Spec{
		Kind:          KindAggregate,
		SchemaVersion: 1,
		Name:          "krx-candidates",
		Params: map[string]ParamSpec{
			"as_of": {Type: ParamDate, Default: "2026-07-01"},
			"limit": {Type: ParamInt, Default: 20},
		},
		Pipeline: []StageSpec{
			{Name: "universe", Type: StageLocalCollection, Collection: "instruments"},
		},
		Output: OutputSpec{
			From:          "universe",
			DefaultFormat: "table",
			Columns:       []OutputColumnSpec{{Key: "symbol", Title: "코드"}},
		},
	}
}

func intPtr(value int) *int {
	return &value
}
