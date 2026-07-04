//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAggregateCLISmokeMongoDB(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI})
	if err != nil {
		t.Fatalf("create mongodb runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Fatalf("close mongodb runtime: %v", err)
		}
	})
	if _, err := runtime.Database().Collection("candidate_source").InsertMany(ctx, []any{
		bson.M{"symbol": "005930", "market": "krx", "traded_amount": 1000, "change_pct": 3.421},
		bson.M{"symbol": "000660", "market": "krx", "traded_amount": 2000, "change_pct": 2.1},
	}); err != nil {
		t.Fatalf("seed candidate_source: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	yamlPath := filepath.Join(t.TempDir(), "aggregate.yaml")
	writeFile(t, yamlPath, `
kind: Aggregate
schema_version: 1
name: cli-krx-candidates
params:
  market:
    type: string
    default: krx
  limit:
    type: int
    default: 2
pipeline:
  - name: source
    type: local_collection
    collection: candidate_source
    filter:
      market: "${params.market}"
  - name: ranked
    type: aggregate
    from: source
    pipeline:
      - $sort:
          traded_amount: -1
      - $limit: "${params.limit}"
  - name: shaped
    type: jq
    from: ranked
    query: |
      map({symbol, traded_amount, change_pct})
output:
  from: shaped
  default_format: table
  columns:
    - key: ordinal
      title: "#"
      format: integer
    - key: symbol
      title: 코드
    - key: traded_amount
      title: 거래대금
      format: number
      precision: 0
`)

	validateOut := runAggregateCLI(t, configPath, server.URI, "validate", "aggregate", yamlPath, "-o", "json")
	if !strings.Contains(validateOut, `"valid": true`) {
		t.Fatalf("validate output missing valid=true:\n%s", validateOut)
	}

	updateOut := runAggregateCLI(t, configPath, server.URI, "update", "aggregate", "cli-krx-candidates", "--file", yamlPath, "-o", "json")
	if !strings.Contains(updateOut, `"name": "cli-krx-candidates"`) {
		t.Fatalf("update output missing aggregate name:\n%s", updateOut)
	}

	versionsOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate", "cli-krx-candidates", "--view", "versions", "-o", "json")
	if !strings.Contains(versionsOut, `"version": 1`) {
		t.Fatalf("inspect versions output missing version 1:\n%s", versionsOut)
	}

	planOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-plan", "cli-krx-candidates", "--param", "limit=1", "-o", "json")
	if !strings.Contains(planOut, `"limit": 1`) {
		t.Fatalf("plan output missing limit override:\n%s", planOut)
	}

	planStagesOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-plan", "cli-krx-candidates", "--param", "limit=1", "--view", "stages", "-o", "json")
	if !strings.Contains(planStagesOut, `"name": "ranked"`) || !strings.Contains(planStagesOut, `"type": "aggregate"`) {
		t.Fatalf("plan stages output missing aggregate stage:\n%s", planStagesOut)
	}

	runOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "cli-krx-candidates", "--alias", "cli-latest", "--param", "limit=1", "-o", "json")
	var rows []map[string]any
	if err := json.Unmarshal([]byte(runOut), &rows); err != nil {
		t.Fatalf("decode run rows: %v\n%s", err, runOut)
	}
	if len(rows) != 1 || rows[0]["symbol"] != "000660" {
		t.Fatalf("run rows = %#v, want top 000660", rows)
	}

	historyOut := runAggregateCLI(t, configPath, server.URI, "history", "aggregate", "--name", "cli-krx-candidates", "-o", "json")
	if !strings.Contains(historyOut, `"alias": "cli-latest"`) {
		t.Fatalf("history output missing run alias:\n%s", historyOut)
	}

	inspectOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-run", "cli-latest", "--limit", "10", "-o", "json")
	if !strings.Contains(inspectOut, `"status": "succeeded"`) {
		t.Fatalf("inspect run output missing succeeded status:\n%s", inspectOut)
	}

	inspectStagesOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-run", "cli-latest", "--view", "stages", "-o", "json")
	if !strings.Contains(inspectStagesOut, `"collection": "aggregate_tmp_`) || !strings.Contains(inspectStagesOut, `"name": "source"`) {
		t.Fatalf("inspect run stages output missing materialized stage context:\n%s", inspectStagesOut)
	}

	inspectParamsOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-run", "cli-latest", "--view", "params", "-o", "json")
	if !strings.Contains(inspectParamsOut, `"limit": 1`) {
		t.Fatalf("inspect run params output missing limit:\n%s", inspectParamsOut)
	}

	csvOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "cli-krx-candidates", "--alias", "cli-csv", "--param", "limit=1", "-o", "csv")
	if !strings.Contains(csvOut, "symbol") || !strings.Contains(csvOut, "000660") {
		t.Fatalf("csv output missing expected row:\n%s", csvOut)
	}

	ndjsonOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "cli-krx-candidates", "--alias", "cli-ndjson", "--param", "limit=1", "-o", "ndjson")
	lines := strings.Split(strings.TrimSpace(ndjsonOut), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], `"symbol":"000660"`) {
		t.Fatalf("ndjson output missing expected single row:\n%s", ndjsonOut)
	}
}

func TestAggregateCLIOutputsPriorityCandidateBoard(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{URI: server.URI})
	if err != nil {
		t.Fatalf("create mongodb runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Fatalf("close mongodb runtime: %v", err)
		}
	})
	seedPriorityCandidateFixture(t, ctx, runtime)

	configPath := filepath.Join(t.TempDir(), "config.json")
	yamlPath := filepath.Join(t.TempDir(), "priority-candidate-board-fixture.aggregate.yaml")
	writeFile(t, yamlPath, priorityCandidateBoardYAML)

	validateOut := runAggregateCLI(t, configPath, server.URI, "validate", "aggregate", yamlPath, "-o", "json")
	if !strings.Contains(validateOut, `"valid": true`) {
		t.Fatalf("validate output missing valid=true:\n%s", validateOut)
	}

	updateOut := runAggregateCLI(t, configPath, server.URI, "update", "aggregate", "priority-candidate-board-fixture", "--file", yamlPath, "-o", "json")
	if !strings.Contains(updateOut, `"name": "priority-candidate-board-fixture"`) {
		t.Fatalf("update output missing aggregate name:\n%s", updateOut)
	}

	planOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-plan", "priority-candidate-board-fixture", "--view", "stages", "-o", "json")
	for _, want := range []string{`"name": "universe"`, `"name": "board"`, `"name": "ranked_candidates"`} {
		if !strings.Contains(planOut, want) {
			t.Fatalf("plan output missing %q:\n%s", want, planOut)
		}
	}

	tableOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "priority-candidate-board-fixture", "--alias", "priority-table", "-o", "table")
	for _, want := range []string{"코드", "시총(조)", "거래대금(억)", "RSI", "ADX", "ATR%", "추세", "메모/라벨", "000660", "SK하이닉스", "52주 고점 근접"} {
		if !strings.Contains(tableOut, want) {
			t.Fatalf("table output missing %q:\n%s", want, tableOut)
		}
	}
	if !strings.Contains(tableOut, "3.21%") || !strings.Contains(tableOut, "180.00") || !strings.Contains(tableOut, "2100") {
		t.Fatalf("table output missing formatted numeric values:\n%s", tableOut)
	}

	jsonOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "priority-candidate-board-fixture", "--alias", "priority-json", "-o", "json")
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &rows); err != nil {
		t.Fatalf("decode json rows: %v\n%s", err, jsonOut)
	}
	if len(rows) != 2 {
		t.Fatalf("json rows len = %d, want 2: %#v", len(rows), rows)
	}
	if rows[0]["symbol"] != "000660" || rows[0]["trend"] != "강세" || rows[0]["label"] != "52주 고점 근접" {
		t.Fatalf("first json row = %#v, want SK hynix priority candidate", rows[0])
	}

	csvOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "priority-candidate-board-fixture", "--alias", "priority-csv", "-o", "csv")
	for _, want := range []string{"symbol", "000660", "SK하이닉스", "52주 고점 근접"} {
		if !strings.Contains(csvOut, want) {
			t.Fatalf("csv output missing %q:\n%s", want, csvOut)
		}
	}

	ndjsonOut := runAggregateCLI(t, configPath, server.URI, "run", "aggregate", "priority-candidate-board-fixture", "--alias", "priority-ndjson", "-o", "ndjson")
	lines := strings.Split(strings.TrimSpace(ndjsonOut), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"symbol":"000660"`) || !strings.Contains(lines[1], `"symbol":"005930"`) {
		t.Fatalf("ndjson output missing expected ordered rows:\n%s", ndjsonOut)
	}

	historyOut := runAggregateCLI(t, configPath, server.URI, "history", "aggregate", "--name", "priority-candidate-board-fixture", "-o", "json")
	if !strings.Contains(historyOut, `"alias": "priority-table"`) || !strings.Contains(historyOut, `"status": "succeeded"`) {
		t.Fatalf("history output missing succeeded priority run:\n%s", historyOut)
	}

	inspectOut := runAggregateCLI(t, configPath, server.URI, "inspect", "aggregate-run", "priority-table", "--view", "stages", "-o", "json")
	for _, want := range []string{`"name": "universe"`, `"rows": 2`, `"name": "ranked_candidates"`} {
		if !strings.Contains(inspectOut, want) {
			t.Fatalf("inspect stages output missing %q:\n%s", want, inspectOut)
		}
	}
}

func seedPriorityCandidateFixture(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) {
	t.Helper()
	candidates := runtime.Database().Collection("aggregate_priority_candidates")
	valuations := runtime.Database().Collection("aggregate_priority_valuation_snapshots")
	if _, err := candidates.InsertMany(ctx, []any{
		bson.M{
			"fixture":             "priority-candidate-e2e",
			"as_of_date":          "2026-07-01",
			"symbol":              "005930",
			"name":                "삼성전자",
			"change_pct":          2.34,
			"traded_amount":       123400000000,
			"relative_volume_20d": 1.8,
			"high_52w_pct":        87.4,
			"close_position_pct":  72.1,
			"rsi_14":              61.2,
			"adx_14":              24.7,
			"atr_pct_14":          2.8,
			"trend":               "상승",
			"label":               "거래대금 확대",
		},
		bson.M{
			"fixture":             "priority-candidate-e2e",
			"as_of_date":          "2026-07-01",
			"symbol":              "000660",
			"name":                "SK하이닉스",
			"change_pct":          3.21,
			"traded_amount":       210000000000,
			"relative_volume_20d": 2.4,
			"high_52w_pct":        95.2,
			"close_position_pct":  88.6,
			"rsi_14":              68.5,
			"adx_14":              31.2,
			"atr_pct_14":          3.1,
			"trend":               "강세",
			"label":               "52주 고점 근접",
		},
	}); err != nil {
		t.Fatalf("seed priority candidates: %v", err)
	}
	if _, err := valuations.InsertMany(ctx, []any{
		bson.M{
			"fixture":          "priority-candidate-e2e",
			"as_of_date":       "2026-07-01",
			"symbol":           "005930",
			"market_cap_minor": 420000000000000,
		},
		bson.M{
			"fixture":          "priority-candidate-e2e",
			"as_of_date":       "2026-07-01",
			"symbol":           "000660",
			"market_cap_minor": 180000000000000,
		},
	}); err != nil {
		t.Fatalf("seed priority valuations: %v", err)
	}
}

func runAggregateCLI(t *testing.T, configPath string, uri string, args ...string) string {
	t.Helper()
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	fullArgs := []string{"--config", configPath, "--database-backend", "mongodb", "--database-url", uri}
	fullArgs = append(fullArgs, args...)
	cmd.SetArgs(fullArgs)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, out.String())
	}
	return out.String()
}

const priorityCandidateBoardYAML = `
kind: Aggregate
schema_version: 1
name: priority-candidate-board-fixture
params:
  as_of:
    type: date
    default: "2026-07-01"
  fixture:
    type: string
    default: priority-candidate-e2e
  limit:
    type: int
    default: 10
pipeline:
  - name: universe
    type: local_collection
    collection: aggregate_priority_candidates
    filter:
      fixture: "${params.fixture}"
      as_of_date: "${params.as_of}"
  - name: valuation
    type: local_collection
    collection: aggregate_priority_valuation_snapshots
    filter:
      fixture: "${params.fixture}"
      as_of_date: "${params.as_of}"
  - name: board
    type: aggregate
    from: universe
    pipeline:
      - $lookup:
          from: valuation
          localField: symbol
          foreignField: symbol
          as: valuation
      - $addFields:
          market_cap_trillion:
            $divide:
              - { $ifNull: [{ $first: "$valuation.market_cap_minor" }, 0] }
              - 1000000000000
          turnover_100m:
            $divide:
              - { $ifNull: ["$traded_amount", 0] }
              - 100000000
      - $sort:
          turnover_100m: -1
          change_pct: -1
      - $limit: "${params.limit}"
  - name: ranked_candidates
    type: jq
    from: board
    query: |
      map({
        symbol,
        name,
        change_pct,
        market_cap_trillion,
        turnover_100m,
        relative_volume_20d,
        high_52w_pct,
        close_position_pct,
        rsi_14,
        adx_14,
        atr_pct_14,
        trend,
        label
      })
output:
  from: ranked_candidates
  default_format: table
  sort:
    - field: turnover_100m
      order: desc
    - field: change_pct
      order: desc
  columns:
    - key: ordinal
      title: "#"
      format: integer
      align: right
    - key: symbol
      title: 코드
      align: center
    - key: name
      title: 종목
      align: left
    - key: change_pct
      title: 등락%
      format: percent
      precision: 2
      align: right
    - key: market_cap_trillion
      title: 시총(조)
      format: number
      precision: 2
      align: right
    - key: turnover_100m
      title: 거래대금(억)
      format: number
      precision: 0
      align: right
    - key: relative_volume_20d
      title: 거래량x20
      format: number
      precision: 1
      align: right
    - key: high_52w_pct
      title: 52주고점%
      format: percent
      precision: 1
      align: right
    - key: close_position_pct
      title: 종가위치%
      format: percent
      precision: 1
      align: right
    - key: rsi_14
      title: RSI
      format: number
      precision: 1
      align: right
    - key: adx_14
      title: ADX
      format: number
      precision: 1
      align: right
    - key: atr_pct_14
      title: ATR%
      format: percent
      precision: 1
      align: right
    - key: trend
      title: 추세
      align: center
    - key: label
      title: 메모/라벨
      align: left
`

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
