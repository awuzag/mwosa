package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/storage"
	dailybarstorage "github.com/ev3rlit/mwosa/storage/dailybar"
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

func requireWriteFile(t *testing.T, path string, payload string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
