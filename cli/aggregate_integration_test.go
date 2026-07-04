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

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
