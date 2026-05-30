package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/storage"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
)

func TestBacktestValidateAndRunUseStoredDailyBars(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())

	var validateOut bytes.Buffer
	validateCmd := NewRootCommand(BuildInfo{})
	validateCmd.SetOut(&validateOut)
	validateCmd.SetErr(&validateOut)
	if err := executeForTest(t, ctx, validateCmd,
		"--database", databasePath,
		"--output", "json",
		"validate", "backtest", yamlPath,
	); err != nil {
		t.Fatalf("backtest validate: %v\n%s", err, validateOut.String())
	}
	for _, want := range []string{`"valid": true`, `"strategy_name": "sma-cross"`, `"fill": "next_open"`} {
		if !strings.Contains(validateOut.String(), want) {
			t.Fatalf("validate output missing %q in:\n%s", want, validateOut.String())
		}
	}

	var runOut bytes.Buffer
	runCmd := NewRootCommand(BuildInfo{})
	runCmd.SetOut(&runOut)
	runCmd.SetErr(&runOut)
	if err := executeForTest(t, ctx, runCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
	); err != nil {
		t.Fatalf("backtest run: %v\n%s", err, runOut.String())
	}

	var result struct {
		StrategyName string `json:"strategy_name"`
		RunName      string `json:"run_name"`
		Symbols      []string
		Execution    struct {
			Fill string `json:"fill"`
		} `json:"execution"`
		Metrics struct {
			TradeCount int `json:"trade_count"`
		} `json:"metrics"`
		Trades      []map[string]any `json:"trades"`
		EquityCurve []map[string]any `json:"equity_curve"`
		ResultHash  string           `json:"result_hash"`
	}
	if err := json.Unmarshal(runOut.Bytes(), &result); err != nil {
		t.Fatalf("run output should be json: %v\n%s", err, runOut.String())
	}
	if result.StrategyName != "sma-cross" || result.RunName != "sma-cross-run" {
		t.Fatalf("unexpected result identity: %#v", result)
	}
	if len(result.Symbols) != 1 || result.Symbols[0] != "069500" {
		t.Fatalf("symbols = %#v, want [069500]", result.Symbols)
	}
	if result.Execution.Fill != "next_open" {
		t.Fatalf("fill = %q, want next_open", result.Execution.Fill)
	}
	if result.Metrics.TradeCount != 2 || len(result.Trades) != 2 || len(result.EquityCurve) == 0 || result.ResultHash == "" {
		t.Fatalf("run output missing expected result payload:\n%s", runOut.String())
	}
	if result.ResultHash == "" {
		t.Fatalf("run output should include result hash:\n%s", runOut.String())
	}

	var listOut bytes.Buffer
	listCmd := NewRootCommand(BuildInfo{})
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	if err := executeForTest(t, ctx, listCmd,
		"--database", databasePath,
		"--output", "json",
		"list", "backtest", "runs",
	); err != nil {
		t.Fatalf("backtest run list: %v\n%s", err, listOut.String())
	}
	var savedRuns []struct {
		ID              string `json:"id"`
		RunName         string `json:"run_name"`
		StrategyHash    string `json:"strategy_hash"`
		RunHash         string `json:"run_hash"`
		DataFingerprint string `json:"data_fingerprint"`
		ResultHash      string `json:"result_hash"`
	}
	if err := json.Unmarshal(listOut.Bytes(), &savedRuns); err != nil {
		t.Fatalf("list output should be json: %v\n%s", err, listOut.String())
	}
	if len(savedRuns) != 1 || savedRuns[0].RunName != "sma-cross-run" || savedRuns[0].ResultHash != result.ResultHash || savedRuns[0].StrategyHash == "" || savedRuns[0].RunHash == "" || savedRuns[0].DataFingerprint == "" {
		t.Fatalf("saved run list missing expected run metadata:\n%#v\n%s", savedRuns, listOut.String())
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "table",
		"inspect", "backtest-run", savedRuns[0].ID,
		"--view", "events",
	); err != nil {
		t.Fatalf("inspect saved backtest run events: %v\n%s", err, inspectOut.String())
	}
	if !strings.Contains(inspectOut.String(), "portfolio_mutation") || !strings.Contains(inspectOut.String(), "total_cost") {
		t.Fatalf("inspect saved run events should reuse event view:\n%s", inspectOut.String())
	}
}

func TestBacktestRunJSONUsesSelectedMetrics(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAMLWithMetricSelection())

	var runOut bytes.Buffer
	runCmd := NewRootCommand(BuildInfo{})
	runCmd.SetOut(&runOut)
	runCmd.SetErr(&runOut)
	if err := executeForTest(t, ctx, runCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
	); err != nil {
		t.Fatalf("backtest run: %v\n%s", err, runOut.String())
	}

	var result map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &result); err != nil {
		t.Fatalf("run output should be json: %v\n%s", err, runOut.String())
	}
	metrics, ok := result["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("metrics should be object:\n%s", runOut.String())
	}
	if _, ok := metrics["trade_count"]; ok {
		t.Fatalf("excluded metric trade_count should not be present:\n%s", runOut.String())
	}
	if _, ok := metrics["average_trade_return"]; !ok {
		t.Fatalf("included metric average_trade_return should be present:\n%s", runOut.String())
	}
	if _, ok := result["trade_count"]; ok {
		t.Fatalf("metric trade_count should not be duplicated at top-level:\n%s", runOut.String())
	}
}

func TestBacktestSavedRunCompare(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	dir := t.TempDir()
	leftPath := filepath.Join(dir, "left.yaml")
	rightPath := filepath.Join(dir, "right.yaml")
	requireWriteFile(t, leftPath, sampleBacktestYAML())
	requireWriteFile(t, rightPath, strings.Replace(strings.Replace(sampleBacktestYAML(), "name: sma-cross-run", "name: sma-cross-run-quarter", 1), "value: 50", "value: 25", 1))

	leftID := runSavedBacktestSummary(t, ctx, databasePath, leftPath).ID
	rightID := runSavedBacktestSummary(t, ctx, databasePath, rightPath).ID
	if leftID == "" || rightID == "" || leftID == rightID {
		t.Fatalf("saved run ids should be distinct: left=%q right=%q", leftID, rightID)
	}

	var compareOut bytes.Buffer
	compareCmd := NewRootCommand(BuildInfo{})
	compareCmd.SetOut(&compareOut)
	compareCmd.SetErr(&compareOut)
	if err := executeForTest(t, ctx, compareCmd,
		"--database", databasePath,
		"--output", "json",
		"compare", "backtest-runs", leftID, rightID,
	); err != nil {
		t.Fatalf("compare saved backtest runs: %v\n%s", err, compareOut.String())
	}
	var comparison struct {
		SameDataFingerprint bool `json:"same_data_fingerprint"`
		SameResultHash      bool `json:"same_result_hash"`
		Metrics             []struct {
			Metric       string  `json:"metric"`
			LeftValue    float64 `json:"left_value"`
			RightValue   float64 `json:"right_value"`
			Delta        float64 `json:"delta"`
			LeftPresent  bool    `json:"left_present"`
			RightPresent bool    `json:"right_present"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(compareOut.Bytes(), &comparison); err != nil {
		t.Fatalf("compare output should be json: %v\n%s", err, compareOut.String())
	}
	if !comparison.SameDataFingerprint || comparison.SameResultHash || len(comparison.Metrics) == 0 {
		t.Fatalf("unexpected comparison summary: %#v\n%s", comparison, compareOut.String())
	}
	var sawTotalReturn bool
	for _, metric := range comparison.Metrics {
		if metric.Metric == "total_return" {
			sawTotalReturn = true
			if !metric.LeftPresent || !metric.RightPresent || metric.LeftValue == metric.RightValue || metric.Delta == 0 {
				t.Fatalf("total_return diff should compare both runs: %#v\n%s", metric, compareOut.String())
			}
		}
	}
	if !sawTotalReturn {
		t.Fatalf("comparison should include total_return:\n%s", compareOut.String())
	}

	var csvOut bytes.Buffer
	csvCmd := NewRootCommand(BuildInfo{})
	csvCmd.SetOut(&csvOut)
	csvCmd.SetErr(&csvOut)
	if err := executeForTest(t, ctx, csvCmd,
		"--database", databasePath,
		"--output", "csv",
		"compare", "backtest-runs", leftID, rightID,
	); err != nil {
		t.Fatalf("compare saved backtest runs csv: %v\n%s", err, csvOut.String())
	}
	if !strings.Contains(csvOut.String(), "metric") || !strings.Contains(csvOut.String(), "total_return") || !strings.Contains(csvOut.String(), "delta") {
		t.Fatalf("compare csv should include metric deltas:\n%s", csvOut.String())
	}
}

func TestBacktestRunViewsSelectResultViewModels(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())

	var summaryOut bytes.Buffer
	summaryCmd := NewRootCommand(BuildInfo{})
	summaryCmd.SetOut(&summaryOut)
	summaryCmd.SetErr(&summaryOut)
	if err := executeForTest(t, ctx, summaryCmd,
		"--database", databasePath,
		"--output", "table",
		"run", "backtest", yamlPath,
		"--view", "summary",
	); err != nil {
		t.Fatalf("backtest run summary view: %v\n%s", err, summaryOut.String())
	}
	if !strings.Contains(summaryOut.String(), "result_hash") && !strings.Contains(summaryOut.String(), "hash") {
		t.Fatalf("summary table should include hash:\n%s", summaryOut.String())
	}

	var tradesOut bytes.Buffer
	tradesCmd := NewRootCommand(BuildInfo{})
	tradesCmd.SetOut(&tradesOut)
	tradesCmd.SetErr(&tradesOut)
	if err := executeForTest(t, ctx, tradesCmd,
		"--database", databasePath,
		"--output", "csv",
		"run", "backtest", yamlPath,
		"--view", "trades",
	); err != nil {
		t.Fatalf("backtest run trades view: %v\n%s", err, tradesOut.String())
	}
	if !strings.Contains(tradesOut.String(), "symbol") || !strings.Contains(tradesOut.String(), "069500") {
		t.Fatalf("trades csv should include trade rows:\n%s", tradesOut.String())
	}

	var tradesJSONOut bytes.Buffer
	tradesJSONCmd := NewRootCommand(BuildInfo{})
	tradesJSONCmd.SetOut(&tradesJSONOut)
	tradesJSONCmd.SetErr(&tradesJSONOut)
	if err := executeForTest(t, ctx, tradesJSONCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
		"--view", "trades",
	); err != nil {
		t.Fatalf("backtest run trades json view: %v\n%s", err, tradesJSONOut.String())
	}
	var tradeRows []struct {
		Time   string  `json:"time"`
		Symbol string  `json:"symbol"`
		Return float64 `json:"return"`
	}
	if err := json.Unmarshal(tradesJSONOut.Bytes(), &tradeRows); err != nil {
		t.Fatalf("trades json view should use view rows: %v\n%s", err, tradesJSONOut.String())
	}
	if len(tradeRows) == 0 || tradeRows[0].Symbol != "069500" || strings.Contains(tradeRows[0].Time, "T") {
		t.Fatalf("trades json view should use date-only view rows:\n%#v\n%s", tradeRows, tradesJSONOut.String())
	}

	var equityOut bytes.Buffer
	equityCmd := NewRootCommand(BuildInfo{})
	equityCmd.SetOut(&equityOut)
	equityCmd.SetErr(&equityOut)
	if err := executeForTest(t, ctx, equityCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
		"--view", "equity",
	); err != nil {
		t.Fatalf("backtest run equity view: %v\n%s", err, equityOut.String())
	}
	var equity []map[string]any
	if err := json.Unmarshal(equityOut.Bytes(), &equity); err != nil {
		t.Fatalf("equity view should be json array: %v\n%s", err, equityOut.String())
	}
	if len(equity) == 0 || equity[0]["equity"] == nil {
		t.Fatalf("equity view should include equity points:\n%s", equityOut.String())
	}

	var ordersOut bytes.Buffer
	ordersCmd := NewRootCommand(BuildInfo{})
	ordersCmd.SetOut(&ordersOut)
	ordersCmd.SetErr(&ordersOut)
	if err := executeForTest(t, ctx, ordersCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
		"--view", "orders",
	); err != nil {
		t.Fatalf("backtest run orders view: %v\n%s", err, ordersOut.String())
	}
	var orders []map[string]any
	if err := json.Unmarshal(ordersOut.Bytes(), &orders); err != nil {
		t.Fatalf("orders view should be json array: %v\n%s", err, ordersOut.String())
	}
	if len(orders) == 0 || orders[0]["symbol"] != "069500" || orders[0]["side"] == nil {
		t.Fatalf("orders view should include order intent rows:\n%s", ordersOut.String())
	}

	var fillsOut bytes.Buffer
	fillsCmd := NewRootCommand(BuildInfo{})
	fillsCmd.SetOut(&fillsOut)
	fillsCmd.SetErr(&fillsOut)
	if err := executeForTest(t, ctx, fillsCmd,
		"--database", databasePath,
		"--output", "csv",
		"run", "backtest", yamlPath,
		"--view", "fills",
	); err != nil {
		t.Fatalf("backtest run fills view: %v\n%s", err, fillsOut.String())
	}
	if !strings.Contains(fillsOut.String(), "type") || !strings.Contains(fillsOut.String(), "fill") || !strings.Contains(fillsOut.String(), "total_cost") {
		t.Fatalf("fills csv should include fill rows:\n%s", fillsOut.String())
	}

	costsYAMLPath := filepath.Join(t.TempDir(), "sma-cross-costs.yaml")
	requireWriteFile(t, costsYAMLPath, strings.Replace(sampleBacktestYAML(), `commission:
    type: bps
    value: 0`, `commission:
    type: bps
    value: 10
  tax:
    type: bps
    sell_value: 10
  exchange_fee:
    type: fixed_amount
    value: 1`, 1))
	var eventsOut bytes.Buffer
	eventsCmd := NewRootCommand(BuildInfo{})
	eventsCmd.SetOut(&eventsOut)
	eventsCmd.SetErr(&eventsOut)
	if err := executeForTest(t, ctx, eventsCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", costsYAMLPath,
		"--view", "events",
	); err != nil {
		t.Fatalf("backtest run events view: %v\n%s", err, eventsOut.String())
	}
	var events []map[string]any
	if err := json.Unmarshal(eventsOut.Bytes(), &events); err != nil {
		t.Fatalf("events view should be json array: %v\n%s", err, eventsOut.String())
	}
	var sawCost bool
	for _, event := range events {
		if event["type"] == "cost" && event["total_cost"] != nil {
			sawCost = true
			break
		}
	}
	if !sawCost {
		t.Fatalf("events view should include cost component rows:\n%s", eventsOut.String())
	}

	positionsYAMLPath := filepath.Join(t.TempDir(), "always-long.yaml")
	requireWriteFile(t, positionsYAMLPath, sampleBacktestAlwaysLongYAML())
	var positionsOut bytes.Buffer
	positionsCmd := NewRootCommand(BuildInfo{})
	positionsCmd.SetOut(&positionsOut)
	positionsCmd.SetErr(&positionsOut)
	if err := executeForTest(t, ctx, positionsCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", positionsYAMLPath,
		"--view", "positions",
	); err != nil {
		t.Fatalf("backtest run positions view: %v\n%s", err, positionsOut.String())
	}
	var positions []map[string]any
	if err := json.Unmarshal(positionsOut.Bytes(), &positions); err != nil {
		t.Fatalf("positions view should be json array: %v\n%s", err, positionsOut.String())
	}
	if len(positions) != 1 || positions[0]["symbol"] != "069500" || positions[0]["market_value"] == nil {
		t.Fatalf("positions view should include final position rows:\n%s", positionsOut.String())
	}
}

func TestBacktestInspectUniverseOutputsSummaryJSONByDefault(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "backtest-universe", yamlPath,
	); err != nil {
		t.Fatalf("inspect backtest universe: %v\n%s", err, out.String())
	}

	var result struct {
		Mode            string   `json:"mode"`
		Schedule        string   `json:"schedule"`
		SelectedSymbols []string `json:"selected_symbols"`
		Steps           []struct {
			ID          string `json:"id"`
			OutputCount int    `json:"output_count"`
		} `json:"steps"`
		Snapshots []map[string]any `json:"snapshots"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("inspect output should be json: %v\n%s", err, out.String())
	}
	if result.Mode != "pipeline" || result.Schedule != "once" {
		t.Fatalf("unexpected universe identity: %#v", result)
	}
	if len(result.SelectedSymbols) != 1 || result.SelectedSymbols[0] != "069500" {
		t.Fatalf("selected symbols = %#v, want [069500]", result.SelectedSymbols)
	}
	if len(result.Steps) != 1 || result.Steps[0].ID != "source.symbols" || result.Steps[0].OutputCount != 1 {
		t.Fatalf("unexpected steps: %#v", result.Steps)
	}
	if len(result.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(result.Snapshots))
	}
	if strings.Contains(out.String(), `"candidates":`) || strings.Contains(out.String(), `"decisions":`) {
		t.Fatalf("default universe inspect should stay compact; use --view raw for full explain:\n%s", out.String())
	}
}

func TestBacktestValidateJSONIncludesUniverseSummaryAndRawView(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"validate", "backtest", yamlPath,
	); err != nil {
		t.Fatalf("backtest validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"universe": {`) || !strings.Contains(out.String(), `"selected_symbols": [`) {
		t.Fatalf("validate output should include universe summary:\n%s", out.String())
	}
	if strings.Contains(out.String(), `"decisions":`) {
		t.Fatalf("default validate output should not include full universe decisions:\n%s", out.String())
	}

	var rawOut bytes.Buffer
	rawCmd := NewRootCommand(BuildInfo{})
	rawCmd.SetOut(&rawOut)
	rawCmd.SetErr(&rawOut)
	if err := executeForTest(t, ctx, rawCmd,
		"--database", databasePath,
		"--output", "json",
		"validate", "backtest", yamlPath,
		"--view", "raw",
	); err != nil {
		t.Fatalf("backtest validate raw: %v\n%s", err, rawOut.String())
	}
	if !strings.Contains(rawOut.String(), `"decisions":`) {
		t.Fatalf("raw validate output should include full universe decisions:\n%s", rawOut.String())
	}
}

func TestEvaluationValidateRunInspectCompareAndRank(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "evaluation.yaml")
	requireWriteFile(t, yamlPath, sampleEvaluationYAML())

	var validateOut bytes.Buffer
	validateCmd := NewRootCommand(BuildInfo{})
	validateCmd.SetOut(&validateOut)
	validateCmd.SetErr(&validateOut)
	if err := executeForTest(t, ctx, validateCmd,
		"--database", databasePath,
		"--output", "json",
		"validate", "evaluation", yamlPath,
	); err != nil {
		t.Fatalf("evaluation validate: %v\n%s", err, validateOut.String())
	}
	if !strings.Contains(validateOut.String(), `"case_count": 2`) {
		t.Fatalf("validate output should include generated case count:\n%s", validateOut.String())
	}

	var runOut bytes.Buffer
	runCmd := NewRootCommand(BuildInfo{})
	runCmd.SetOut(&runOut)
	runCmd.SetErr(&runOut)
	if err := executeForTest(t, ctx, runCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "evaluation", yamlPath,
		"--parallelism", "2",
	); err != nil {
		t.Fatalf("evaluation run: %v\n%s", err, runOut.String())
	}
	var runResult struct {
		Experiment struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"experiment"`
		Cases []struct {
			CaseID            string         `json:"case_id"`
			PassedConstraints bool           `json:"passed_constraints"`
			Metrics           map[string]any `json:"metrics"`
			Result            struct {
				DataFingerprint string `json:"data_fingerprint"`
				ResultHash      string `json:"result_hash"`
			} `json:"result"`
		} `json:"cases"`
		Ranking []struct {
			Rank int `json:"rank"`
		} `json:"ranking"`
	}
	if err := json.Unmarshal(runOut.Bytes(), &runResult); err != nil {
		t.Fatalf("run output should be json: %v\n%s", err, runOut.String())
	}
	if runResult.Experiment.Name != "sma-cross-evaluation" || runResult.Experiment.ID == "" {
		t.Fatalf("unexpected experiment identity: %#v", runResult.Experiment)
	}
	if len(runResult.Cases) != 2 || len(runResult.Ranking) == 0 || runResult.Ranking[0].Rank != 1 {
		t.Fatalf("unexpected evaluation result:\n%s", runOut.String())
	}
	if _, ok := runResult.Cases[0].Metrics["calmar"]; !ok {
		t.Fatalf("research metrics should include calmar:\n%s", runOut.String())
	}
	if runResult.Cases[0].Result.DataFingerprint == "" || runResult.Cases[0].Result.ResultHash == "" {
		t.Fatalf("evaluation case result should include data/result hashes:\n%s", runOut.String())
	}

	for _, command := range [][]string{
		{"list", "evaluations"},
		{"inspect", "evaluation", "sma-cross-evaluation"},
		{"compare", "evaluation", "sma-cross-evaluation"},
		{"rank", "evaluation", "sma-cross-evaluation", "--objective", "calmar"},
	} {
		var out bytes.Buffer
		cmd := NewRootCommand(BuildInfo{})
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		args := append([]string{"--database", databasePath, "--output", "json"}, command...)
		if err := executeForTest(t, ctx, cmd, args...); err != nil {
			t.Fatalf("%v: %v\n%s", command, err, out.String())
		}
		if !strings.Contains(out.String(), "sma-cross-evaluation") && command[0] != "rank" {
			t.Fatalf("%v output should mention evaluation:\n%s", command, out.String())
		}
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-evaluation",
	); err != nil {
		t.Fatalf("inspect evaluation hashes: %v\n%s", err, inspectOut.String())
	}
	for _, want := range []string{`"strategy_hash": "sha256:`, `"run_hash": "sha256:`, `"data_fingerprint": "sha256:`, `"result_hash": "sha256:`} {
		if !strings.Contains(inspectOut.String(), want) {
			t.Fatalf("inspect evaluation should include %q:\n%s", want, inspectOut.String())
		}
	}

	var summaryOut bytes.Buffer
	summaryCmd := NewRootCommand(BuildInfo{})
	summaryCmd.SetOut(&summaryOut)
	summaryCmd.SetErr(&summaryOut)
	if err := executeForTest(t, ctx, summaryCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-evaluation",
		"--view", "summary",
	); err != nil {
		t.Fatalf("inspect evaluation summary view: %v\n%s", err, summaryOut.String())
	}
	var summary struct {
		Name                 string `json:"name"`
		CaseCount            int    `json:"case_count"`
		WalkForwardStepCount int    `json:"walk_forward_step_count"`
		BestCaseID           string `json:"best_case_id"`
	}
	if err := json.Unmarshal(summaryOut.Bytes(), &summary); err != nil {
		t.Fatalf("summary view should be json object: %v\n%s", err, summaryOut.String())
	}
	if summary.Name != "sma-cross-evaluation" || summary.CaseCount != 2 || summary.WalkForwardStepCount != 0 || summary.BestCaseID == "" {
		t.Fatalf("unexpected evaluation summary view: %#v\n%s", summary, summaryOut.String())
	}

	var casesCSV bytes.Buffer
	casesCmd := NewRootCommand(BuildInfo{})
	casesCmd.SetOut(&casesCSV)
	casesCmd.SetErr(&casesCSV)
	if err := executeForTest(t, ctx, casesCmd,
		"--database", databasePath,
		"--output", "csv",
		"compare", "evaluation", "sma-cross-evaluation",
		"--view", "cases",
	); err != nil {
		t.Fatalf("compare evaluation cases csv view: %v\n%s", err, casesCSV.String())
	}
	if !strings.Contains(casesCSV.String(), "case_id") || !strings.Contains(casesCSV.String(), "strategy_hash") || strings.Contains(casesCSV.String(), "walk_forward") {
		t.Fatalf("cases csv view should include case rows only:\n%s", casesCSV.String())
	}

	var regimeOut bytes.Buffer
	regimeCmd := NewRootCommand(BuildInfo{})
	regimeCmd.SetOut(&regimeOut)
	regimeCmd.SetErr(&regimeOut)
	if err := executeForTest(t, ctx, regimeCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-evaluation",
		"--view", "regime",
	); err != nil {
		t.Fatalf("inspect evaluation regime view: %v\n%s", err, regimeOut.String())
	}
	var regimes []struct {
		Tag            string             `json:"tag"`
		CaseCount      int                `json:"case_count"`
		BestCaseID     string             `json:"best_case_id"`
		AverageMetrics map[string]float64 `json:"average_metrics"`
	}
	if err := json.Unmarshal(regimeOut.Bytes(), &regimes); err != nil {
		t.Fatalf("regime view should be json array: %v\n%s", err, regimeOut.String())
	}
	if len(regimes) == 0 || regimes[0].Tag == "" || regimes[0].CaseCount == 0 || regimes[0].BestCaseID == "" || len(regimes[0].AverageMetrics) == 0 {
		t.Fatalf("regime view should include split rows with metrics: %#v\n%s", regimes, regimeOut.String())
	}

	var regimeTable bytes.Buffer
	regimeTableCmd := NewRootCommand(BuildInfo{})
	regimeTableCmd.SetOut(&regimeTable)
	regimeTableCmd.SetErr(&regimeTable)
	if err := executeForTest(t, ctx, regimeTableCmd,
		"--database", databasePath,
		"--output", "table",
		"compare", "evaluation", "sma-cross-evaluation",
		"--view", "regime",
	); err != nil {
		t.Fatalf("compare evaluation regime table view: %v\n%s", err, regimeTable.String())
	}
	if !strings.Contains(regimeTable.String(), "tag") || !strings.Contains(regimeTable.String(), "best_case") || !strings.Contains(regimeTable.String(), "avg_objective") {
		t.Fatalf("regime table should include split columns:\n%s", regimeTable.String())
	}

	var robustnessOut bytes.Buffer
	robustnessCmd := NewRootCommand(BuildInfo{})
	robustnessCmd.SetOut(&robustnessOut)
	robustnessCmd.SetErr(&robustnessOut)
	if err := executeForTest(t, ctx, robustnessCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-evaluation",
		"--view", "robustness",
	); err != nil {
		t.Fatalf("inspect evaluation robustness view: %v\n%s", err, robustnessOut.String())
	}
	var robustness struct {
		ParameterSensitivity []struct {
			Parameter      string  `json:"parameter"`
			BestValue      string  `json:"best_value"`
			BestCaseID     string  `json:"best_case_id"`
			ObjectiveRange float64 `json:"objective_range"`
		} `json:"parameter_sensitivity"`
		TopNStability struct {
			PeriodCount    int     `json:"period_count"`
			AverageOverlap float64 `json:"average_overlap"`
		} `json:"top_n_stability"`
	}
	if err := json.Unmarshal(robustnessOut.Bytes(), &robustness); err != nil {
		t.Fatalf("robustness view should be json object: %v\n%s", err, robustnessOut.String())
	}
	if len(robustness.ParameterSensitivity) == 0 || robustness.ParameterSensitivity[0].Parameter == "" || robustness.ParameterSensitivity[0].BestCaseID == "" {
		t.Fatalf("robustness view should include parameter sensitivity: %#v\n%s", robustness, robustnessOut.String())
	}

	var robustnessTable bytes.Buffer
	robustnessTableCmd := NewRootCommand(BuildInfo{})
	robustnessTableCmd.SetOut(&robustnessTable)
	robustnessTableCmd.SetErr(&robustnessTable)
	if err := executeForTest(t, ctx, robustnessTableCmd,
		"--database", databasePath,
		"--output", "table",
		"compare", "evaluation", "sma-cross-evaluation",
		"--view", "robustness",
	); err != nil {
		t.Fatalf("compare evaluation robustness table view: %v\n%s", err, robustnessTable.String())
	}
	if !strings.Contains(robustnessTable.String(), "parameter") || !strings.Contains(robustnessTable.String(), "objective_range") || !strings.Contains(robustnessTable.String(), "top_n_overlap") {
		t.Fatalf("robustness table should include robustness columns:\n%s", robustnessTable.String())
	}

	var invalidOut bytes.Buffer
	invalidCmd := NewRootCommand(BuildInfo{})
	invalidCmd.SetOut(&invalidOut)
	invalidCmd.SetErr(&invalidOut)
	if err := executeForTest(t, ctx, invalidCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-evaluation",
		"--view", "nope",
	); err == nil {
		t.Fatalf("invalid evaluation view should fail:\n%s", invalidOut.String())
	}
}

func TestEvaluationWalkForwardOutputsStepHashes(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "evaluation-walk-forward.yaml")
	requireWriteFile(t, yamlPath, sampleWalkForwardEvaluationYAML())

	var runOut bytes.Buffer
	runCmd := NewRootCommand(BuildInfo{})
	runCmd.SetOut(&runOut)
	runCmd.SetErr(&runOut)
	if err := executeForTest(t, ctx, runCmd,
		"--database", databasePath,
		"--output", "json",
		"run", "evaluation", yamlPath,
		"--parallelism", "2",
	); err != nil {
		t.Fatalf("evaluation walk-forward run: %v\n%s", err, runOut.String())
	}
	var runResult struct {
		WalkForward []struct {
			DataFingerprint string `json:"data_fingerprint"`
			ResultHash      string `json:"result_hash"`
		} `json:"walk_forward"`
	}
	if err := json.Unmarshal(runOut.Bytes(), &runResult); err != nil {
		t.Fatalf("walk-forward run output should be json: %v\n%s", err, runOut.String())
	}
	if len(runResult.WalkForward) == 0 || runResult.WalkForward[0].DataFingerprint == "" || runResult.WalkForward[0].ResultHash == "" {
		t.Fatalf("walk-forward run output should include step data/result hashes:\n%s", runOut.String())
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-walk-forward",
	); err != nil {
		t.Fatalf("inspect walk-forward evaluation: %v\n%s", err, inspectOut.String())
	}
	for _, want := range []string{`"walk_forward": [`, `"strategy_hash": "sha256:`, `"run_hash": "sha256:`, `"data_fingerprint": "sha256:`, `"result_hash": "sha256:`} {
		if !strings.Contains(inspectOut.String(), want) {
			t.Fatalf("inspect walk-forward evaluation should include %q:\n%s", want, inspectOut.String())
		}
	}

	var walkForwardOut bytes.Buffer
	walkForwardCmd := NewRootCommand(BuildInfo{})
	walkForwardCmd.SetOut(&walkForwardOut)
	walkForwardCmd.SetErr(&walkForwardOut)
	if err := executeForTest(t, ctx, walkForwardCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-walk-forward",
		"--view", "walk_forward",
	); err != nil {
		t.Fatalf("inspect walk-forward view: %v\n%s", err, walkForwardOut.String())
	}
	var steps []struct {
		StepIndex       int    `json:"step_index"`
		StrategyHash    string `json:"strategy_hash"`
		RunHash         string `json:"run_hash"`
		DataFingerprint string `json:"data_fingerprint"`
		ResultHash      string `json:"result_hash"`
	}
	if err := json.Unmarshal(walkForwardOut.Bytes(), &steps); err != nil {
		t.Fatalf("walk-forward view should be json array: %v\n%s", err, walkForwardOut.String())
	}
	if len(steps) == 0 || steps[0].StepIndex == 0 || steps[0].StrategyHash == "" || steps[0].RunHash == "" || steps[0].DataFingerprint == "" || steps[0].ResultHash == "" {
		t.Fatalf("walk-forward view should include step hash rows: %#v\n%s", steps, walkForwardOut.String())
	}

	var robustnessOut bytes.Buffer
	robustnessCmd := NewRootCommand(BuildInfo{})
	robustnessCmd.SetOut(&robustnessOut)
	robustnessCmd.SetErr(&robustnessOut)
	if err := executeForTest(t, ctx, robustnessCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "evaluation", "sma-cross-walk-forward",
		"--view", "robustness",
	); err != nil {
		t.Fatalf("inspect walk-forward robustness view: %v\n%s", err, robustnessOut.String())
	}
	var robustness struct {
		OutOfSampleDegradation *struct {
			StepCount             int     `json:"step_count"`
			AverageTrainObjective float64 `json:"average_train_objective"`
			AverageTestObjective  float64 `json:"average_test_objective"`
		} `json:"out_of_sample_degradation"`
	}
	if err := json.Unmarshal(robustnessOut.Bytes(), &robustness); err != nil {
		t.Fatalf("walk-forward robustness view should be json object: %v\n%s", err, robustnessOut.String())
	}
	if robustness.OutOfSampleDegradation == nil || robustness.OutOfSampleDegradation.StepCount == 0 {
		t.Fatalf("walk-forward robustness should include out-of-sample degradation:\n%s", robustnessOut.String())
	}

	var walkForwardTable bytes.Buffer
	walkForwardTableCmd := NewRootCommand(BuildInfo{})
	walkForwardTableCmd.SetOut(&walkForwardTable)
	walkForwardTableCmd.SetErr(&walkForwardTable)
	if err := executeForTest(t, ctx, walkForwardTableCmd,
		"--database", databasePath,
		"--output", "table",
		"inspect", "evaluation", "sma-cross-walk-forward",
		"--view", "walk_forward",
	); err != nil {
		t.Fatalf("inspect walk-forward table view: %v\n%s", err, walkForwardTable.String())
	}
	if !strings.Contains(walkForwardTable.String(), "step") || !strings.Contains(walkForwardTable.String(), "strategy_hash") || !strings.Contains(walkForwardTable.String(), "result_hash") {
		t.Fatalf("walk-forward table should include step hash columns:\n%s", walkForwardTable.String())
	}
}

func TestEvaluationValidateSupportsExpandingRandomSearch(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "evaluation-random-search.yaml")
	requireWriteFile(t, yamlPath, sampleRandomSearchEvaluationYAML())

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"validate", "evaluation", yamlPath,
	); err != nil {
		t.Fatalf("validate random search evaluation: %v\n%s", err, out.String())
	}
	var result struct {
		Name      string `json:"name"`
		CaseCount int    `json:"case_count"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("validate output should be json: %v\n%s", err, out.String())
	}
	if result.Name != "sma-cross-random-search" || result.CaseCount != 12 {
		t.Fatalf("unexpected random search validation: %#v\n%s", result, out.String())
	}
}

func TestEvaluationRunSupportsWeightedObjective(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "evaluation-weighted.yaml")
	requireWriteFile(t, yamlPath, sampleWeightedEvaluationYAML())

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"run", "evaluation", yamlPath,
	); err != nil {
		t.Fatalf("run weighted evaluation: %v\n%s", err, out.String())
	}
	var result struct {
		Ranking []struct {
			CaseID         string         `json:"case_id"`
			Objective      string         `json:"objective"`
			ObjectiveValue float64        `json:"objective_value"`
			Metrics        map[string]any `json:"metrics"`
		} `json:"ranking"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("weighted run output should be json: %v\n%s", err, out.String())
	}
	if len(result.Ranking) == 0 || result.Ranking[0].Objective != "weighted_score" {
		t.Fatalf("weighted evaluation ranking should use weighted_score: %#v\n%s", result.Ranking, out.String())
	}
	if _, ok := result.Ranking[0].Metrics["cagr"]; !ok {
		t.Fatalf("weighted evaluation should include cagr metric:\n%s", out.String())
	}
	if _, ok := result.Ranking[0].Metrics["turnover"]; !ok {
		t.Fatalf("weighted evaluation should include turnover metric:\n%s", out.String())
	}
}

func TestBacktestInspectUniverseNestedCommandSupportsTableOutput(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedBacktestDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "table",
		"inspect", "backtest", "universe", yamlPath,
	); err != nil {
		t.Fatalf("inspect backtest universe table: %v\n%s", err, out.String())
	}
	for _, want := range []string{"schedule", "once", "symbols", "policy", "hold"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("table output missing %q in:\n%s", want, out.String())
		}
	}
}

func seedBacktestDailyBars(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close seed database: %v", err)
		}
	})
	_, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		t.Fatalf("new daily bar repositories: %v", err)
	}
	bars := []dailybar.Bar{
		backtestDailyBar("2024-01-02", "10", "10", "10", "10"),
		backtestDailyBar("2024-01-03", "9", "9", "9", "9"),
		backtestDailyBar("2024-01-04", "12", "12", "12", "12"),
		backtestDailyBar("2024-01-05", "13", "13", "11", "11"),
		backtestDailyBar("2024-01-08", "10", "10", "10", "10"),
	}
	if _, err := writer.UpsertDailyBars(ctx, bars); err != nil {
		t.Fatalf("seed daily bars: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close seeded database: %v", err)
	}
}

type savedBacktestSummaryForTest struct {
	ID              string `json:"id"`
	RunName         string `json:"run_name"`
	ResultHash      string `json:"result_hash"`
	DataFingerprint string `json:"data_fingerprint"`
}

func runSavedBacktestSummary(t *testing.T, ctx context.Context, databasePath string, yamlPath string) savedBacktestSummaryForTest {
	t.Helper()
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"run", "backtest", yamlPath,
		"--view", "summary",
	); err != nil {
		t.Fatalf("run saved backtest summary: %v\n%s", err, out.String())
	}
	var summary savedBacktestSummaryForTest
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("summary output should be json: %v\n%s", err, out.String())
	}
	return summary
}

func backtestDailyBar(date string, open string, high string, low string, closePrice string) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		TradingDate:  date,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        closePrice,
		Volume:       "1000",
	}
}

func sampleBacktestYAML() string {
	return `
kind: Strategy
schema_version: 1
name: sma-cross
indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 2
entry:
  crosses_above:
    - price: close
    - ref: trend
exit:
  crosses_below:
    - price: close
    - ref: trend
sizing:
  type: percent_of_equity
  value: 50
risk:
  max_positions: 1
  max_symbol_weight_pct: 60
---
kind: BacktestRun
schema_version: 1
name: sma-cross-run
strategy:
  name: sma-cross
data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-02
  to: 2024-01-08
universe:
  symbols: ["069500"]
portfolio:
  initial_cash: 10000
  currency: KRW
execution:
  fill: next_open
  commission:
    type: bps
    value: 0
  slippage:
    type: bps
    value: 0
report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count
`
}

func sampleBacktestYAMLWithMetricSelection() string {
	return strings.Replace(sampleBacktestYAML(), `report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count`, `report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count`, 1)
}

func sampleEvaluationYAML() string {
	return strings.Replace(sampleBacktestYAML(), `report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-cross-evaluation
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-08
parameters:
  indicators.trend.params.window: [2, 3]
metrics:
  preset: research
constraints:
  max_drawdown_lte: 1
  min_trade_count_gte: 1
ranking:
  objective: calmar
  order: desc`, 1)
}

func sampleWalkForwardEvaluationYAML() string {
	return strings.Replace(sampleBacktestYAML(), `report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-cross-walk-forward
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-05
parameters:
  indicators.trend.params.window: [2, 3]
  sizing.value: [25, 50]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc
walk_forward:
  train:
    days: 2
  test:
    days: 1
  step:
    days: 1
  select:
    objective: calmar
    order: desc`, 1)
}

func sampleRandomSearchEvaluationYAML() string {
	return strings.Replace(sampleBacktestYAML(), `report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-cross-random-search
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: expanding
  from: 2024-01-02
  to: 2024-01-05
  window:
    days: 2
  step:
    days: 1
search:
  mode: random
  seed: 7
  samples: 4
  parameters:
    indicators.trend.params.window:
      min: 2
      max: 5
      step: 1
    sizing.value:
      values: [25, 50]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc`, 1)
}

func sampleWeightedEvaluationYAML() string {
	return strings.Replace(sampleBacktestYAML(), `report:
  metrics:
    - total_return
    - max_drawdown
    - trade_count`, `---
kind: Evaluation
schema_version: 1
name: sma-cross-weighted
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-run
periods:
  mode: explicit
  from: 2024-01-02
  to: 2024-01-08
parameters:
  indicators.trend.params.window: [2, 3]
metrics:
  preset: core
ranking:
  objective: weighted_score
  order: desc
  weights:
    cagr: 1
    turnover: -0.01`, 1)
}

func sampleBacktestAlwaysLongYAML() string {
	return strings.Replace(sampleBacktestYAML(), `indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 2
entry:
  crosses_above:
    - price: close
    - ref: trend
exit:
  crosses_below:
    - price: close
    - ref: trend`, `entry:
  gt:
    - price: close
    - value: 0
exit:
  lt:
    - price: close
    - value: 0`, 1)
}

func requireWriteFile(t *testing.T, path string, payload string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
