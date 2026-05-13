package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/storage"
	dailybarstorage "github.com/ev3rlit/mwosa/storage/dailybar"
)

func TestStrategyLifecycleStoresJQSourceAndScreensFixtureData(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedStrategyDailyBars(t, ctx, databasePath)

	strategyFile := filepath.Join(t.TempDir(), "strategy.jq")
	if err := os.WriteFile(strategyFile, []byte(`map(select(.symbol == "069500"))`), 0o644); err != nil {
		t.Fatalf("write jq file: %v", err)
	}

	var createOut bytes.Buffer
	createCmd := NewRootCommand(BuildInfo{})
	createCmd.SetOut(&createOut)
	createCmd.SetErr(&createOut)
	if err := executeForTest(t, ctx, createCmd,
		"--database", databasePath,
		"--output", "json",
		"create", "strategy", "etf-lowvol",
		"--engine", "jq",
		"--input", "etf_daily_metrics",
		"--jq-file", strategyFile,
	); err != nil {
		t.Fatalf("create strategy: %v\n%s", err, createOut.String())
	}
	if strings.Contains(createOut.String(), strategyFile) {
		t.Fatalf("create output should not store jq file path:\n%s", createOut.String())
	}
	if !strings.Contains(createOut.String(), `"query_text": "map(select(.symbol == \"069500\"))"`) {
		t.Fatalf("create output should include stored jq source:\n%s", createOut.String())
	}

	var updateOut bytes.Buffer
	updateCmd := NewRootCommand(BuildInfo{})
	updateCmd.SetOut(&updateOut)
	updateCmd.SetErr(&updateOut)
	if err := executeForTest(t, ctx, updateCmd,
		"--database", databasePath,
		"--output", "json",
		"update", "strategy", "etf-lowvol",
		"--jq", `map(select(.symbol == "123456"))`,
	); err != nil {
		t.Fatalf("update strategy: %v\n%s", err, updateOut.String())
	}
	if !strings.Contains(updateOut.String(), `"version": 2`) {
		t.Fatalf("update output should create version 2:\n%s", updateOut.String())
	}

	var screenOut bytes.Buffer
	screenCmd := NewRootCommand(BuildInfo{})
	screenCmd.SetOut(&screenOut)
	screenCmd.SetErr(&screenOut)
	if err := executeForTest(t, ctx, screenCmd,
		"--database", databasePath,
		"--output", "json",
		"screen", "strategy", "etf-lowvol",
		"--alias", "close-watch",
	); err != nil {
		t.Fatalf("screen strategy: %v\n%s", err, screenOut.String())
	}
	for _, want := range []string{`"alias": "close-watch"`, `"result_count": 1`, `"symbol": "123456"`} {
		if !strings.Contains(screenOut.String(), want) {
			t.Fatalf("screen output missing %q in:\n%s", want, screenOut.String())
		}
	}

	var historyOut bytes.Buffer
	historyCmd := NewRootCommand(BuildInfo{})
	historyCmd.SetOut(&historyOut)
	historyCmd.SetErr(&historyOut)
	if err := executeForTest(t, ctx, historyCmd,
		"--database", databasePath,
		"--output", "json",
		"history", "screen",
	); err != nil {
		t.Fatalf("history screen: %v\n%s", err, historyOut.String())
	}
	if !strings.Contains(historyOut.String(), `"alias": "close-watch"`) {
		t.Fatalf("history output should include alias:\n%s", historyOut.String())
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "screen", "close-watch",
	); err != nil {
		t.Fatalf("inspect screen by alias: %v\n%s", err, inspectOut.String())
	}
	if !strings.Contains(inspectOut.String(), `"payload"`) || !strings.Contains(inspectOut.String(), `"123456"`) {
		t.Fatalf("inspect screen should include stored row payload:\n%s", inspectOut.String())
	}

	var deleteOut bytes.Buffer
	deleteCmd := NewRootCommand(BuildInfo{})
	deleteCmd.SetOut(&deleteOut)
	deleteCmd.SetErr(&deleteOut)
	if err := executeForTest(t, ctx, deleteCmd,
		"--database", databasePath,
		"--output", "json",
		"delete", "strategy", "etf-lowvol",
	); err != nil {
		t.Fatalf("delete strategy: %v\n%s", err, deleteOut.String())
	}
	if !strings.Contains(deleteOut.String(), `"deleted": true`) {
		t.Fatalf("delete output should confirm soft delete:\n%s", deleteOut.String())
	}

	var listOut bytes.Buffer
	listCmd := NewRootCommand(BuildInfo{})
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	if err := executeForTest(t, ctx, listCmd,
		"--database", databasePath,
		"--output", "json",
		"list", "strategies",
	); err != nil {
		t.Fatalf("list strategies: %v\n%s", err, listOut.String())
	}
	if strings.Contains(listOut.String(), `"name": "etf-lowvol"`) {
		t.Fatalf("soft-deleted strategy should be hidden from list:\n%s", listOut.String())
	}
}

func TestScreenETFExecutesInlineJQWithoutSavedStrategy(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedStrategyDailyBars(t, ctx, databasePath)

	var screenOut bytes.Buffer
	screenCmd := NewRootCommand(BuildInfo{})
	screenCmd.SetOut(&screenOut)
	screenCmd.SetErr(&screenOut)
	if err := executeForTest(t, ctx, screenCmd,
		"--database", databasePath,
		"--output", "json",
		"screen", "etfs",
		"--jq", `map(select(.symbol == "069500"))`,
	); err != nil {
		t.Fatalf("screen etfs inline jq: %v\n%s", err, screenOut.String())
	}
	for _, want := range []string{`"input_dataset": "etf_daily_metrics"`, `"result_count": 1`, `"symbol": "069500"`} {
		if !strings.Contains(screenOut.String(), want) {
			t.Fatalf("screen etfs output missing %q in:\n%s", want, screenOut.String())
		}
	}

	var historyOut bytes.Buffer
	historyCmd := NewRootCommand(BuildInfo{})
	historyCmd.SetOut(&historyOut)
	historyCmd.SetErr(&historyOut)
	if err := executeForTest(t, ctx, historyCmd,
		"--database", databasePath,
		"--output", "json",
		"history", "screen",
	); err != nil {
		t.Fatalf("history screen after inline jq: %v\n%s", err, historyOut.String())
	}
	if strings.Contains(historyOut.String(), `"result_count"`) {
		t.Fatalf("inline jq screen should not create saved screen history:\n%s", historyOut.String())
	}
}

func TestScreenPipelineAndInspectScreenPipelineOutputExplainJSON(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedStrategyDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "screen-pipeline.yaml")
	if err := os.WriteFile(yamlPath, []byte(`kind: ScreenRun
schema_version: 1
data:
  market: krx
  security_type: etf
  as_of: "2024-04-16"
pipeline:
  - id: source.daily_bars
  - id: transform.latest_per_symbol
  - id: filter.include_symbols
    params:
      symbols: ["069500"]
`), 0o644); err != nil {
		t.Fatalf("write screen pipeline yaml: %v", err)
	}

	var screenOut bytes.Buffer
	screenCmd := NewRootCommand(BuildInfo{})
	screenCmd.SetOut(&screenOut)
	screenCmd.SetErr(&screenOut)
	if err := executeForTest(t, ctx, screenCmd,
		"--database", databasePath,
		"--output", "json",
		"screen", "pipeline", yamlPath,
	); err != nil {
		t.Fatalf("screen pipeline: %v\n%s", err, screenOut.String())
	}
	for _, want := range []string{`"kind": "ScreenRun"`, `"selected_symbols": [`, `"069500"`, `"explain": {`} {
		if !strings.Contains(screenOut.String(), want) {
			t.Fatalf("screen pipeline output missing %q in:\n%s", want, screenOut.String())
		}
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "screen-pipeline", yamlPath,
	); err != nil {
		t.Fatalf("inspect screen-pipeline: %v\n%s", err, inspectOut.String())
	}
	for _, want := range []string{`"result_count": 1`, `"source.daily_bars"`, `"decisions": [`} {
		if !strings.Contains(inspectOut.String(), want) {
			t.Fatalf("inspect screen-pipeline output missing %q in:\n%s", want, inspectOut.String())
		}
	}
}

func seedStrategyDailyBars(t *testing.T, ctx context.Context, databasePath string) {
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
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			Name:         "KODEX 200",
			TradingDate:  "2024-04-15",
			Close:        "35120",
		},
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "123456",
			Name:         "OTHER ETF",
			TradingDate:  "2024-04-15",
			Close:        "1000",
		},
	}
	if _, err := writer.UpsertDailyBars(ctx, bars); err != nil {
		t.Fatalf("seed daily bars: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close seeded database: %v", err)
	}
}
