package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
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

func TestScreenStockReadsFinancialMetricAndValuationDataset(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStrategyStockFundamentals(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"screen", "stock",
		"--jq", `map(select(.financial_metrics.roe.value_bp == 1800 and .valuation.per_bp == 120000 and .fundamental_scores.quality_score == 72 and .fundamental_scores.valuation_score == 82 and .company_facts.audit_opinion.value_text == "적정" and .company_events[0].event_type == "company_merger"))`,
	); err != nil {
		t.Fatalf("screen stock: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"input_dataset": "stock_daily_metrics"`, `"result_count": 1`, `"symbol": "005930"`, `"financial_metrics"`, `"valuation"`, `"fundamental_scores"`, `"company_facts"`, `"company_events"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("screen stock output missing %q in:\n%s", want, out.String())
		}
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

func TestUpdateScreenStrategyStoresYAMLPipelineAndScreensByName(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedStrategyDailyBars(t, ctx, databasePath)

	yamlPath := filepath.Join(t.TempDir(), "screen-strategy.yaml")
	if err := os.WriteFile(yamlPath, []byte(`kind: ScreenStrategy
schema_version: 1
name: etf-uptrend
engine: yaml_pipeline
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
		t.Fatalf("write screen strategy yaml: %v", err)
	}

	var updateOut bytes.Buffer
	updateCmd := NewRootCommand(BuildInfo{})
	updateCmd.SetOut(&updateOut)
	updateCmd.SetErr(&updateOut)
	if err := executeForTest(t, ctx, updateCmd,
		"--database", databasePath,
		"--output", "json",
		"update", "screen", "strategy", "etf-uptrend",
		"--file", yamlPath,
	); err != nil {
		t.Fatalf("update screen strategy: %v\n%s", err, updateOut.String())
	}
	for _, want := range []string{`"engine": "yaml_pipeline"`, `"spec_hash": "sha256:`, `"input_dataset": "screen_pipeline"`} {
		if !strings.Contains(updateOut.String(), want) {
			t.Fatalf("update screen strategy output missing %q in:\n%s", want, updateOut.String())
		}
	}

	var screenOut bytes.Buffer
	screenCmd := NewRootCommand(BuildInfo{})
	screenCmd.SetOut(&screenOut)
	screenCmd.SetErr(&screenOut)
	if err := executeForTest(t, ctx, screenCmd,
		"--database", databasePath,
		"--output", "json",
		"screen", "strategy", "etf-uptrend",
		"--alias", "yaml-uptrend",
	); err != nil {
		t.Fatalf("screen yaml strategy: %v\n%s", err, screenOut.String())
	}
	for _, want := range []string{`"alias": "yaml-uptrend"`, `"result_count": 1`, `"symbol": "069500"`, `"data_as_of": "2024-04-16"`} {
		if !strings.Contains(screenOut.String(), want) {
			t.Fatalf("screen yaml strategy output missing %q in:\n%s", want, screenOut.String())
		}
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, ctx, inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "screen", "yaml-uptrend",
	); err != nil {
		t.Fatalf("inspect yaml screen: %v\n%s", err, inspectOut.String())
	}
	if !strings.Contains(inspectOut.String(), `"payload"`) || !strings.Contains(inspectOut.String(), `"069500"`) {
		t.Fatalf("inspect yaml screen should include stored row payload:\n%s", inspectOut.String())
	}

	yamlSecondPath := filepath.Join(t.TempDir(), "screen-strategy-second.yaml")
	if err := os.WriteFile(yamlSecondPath, []byte(strings.ReplaceAll(string(mustReadFile(t, yamlPath)), "name: etf-uptrend", "name: etf-mdd")), 0o644); err != nil {
		t.Fatalf("write second screen strategy yaml: %v", err)
	}

	var updateSecondOut bytes.Buffer
	updateSecondCmd := NewRootCommand(BuildInfo{})
	updateSecondCmd.SetOut(&updateSecondOut)
	updateSecondCmd.SetErr(&updateSecondOut)
	if err := executeForTest(t, ctx, updateSecondCmd,
		"--database", databasePath,
		"--output", "json",
		"update", "screen", "strategy", "etf-mdd",
		"--file", yamlSecondPath,
	); err != nil {
		t.Fatalf("update second screen strategy: %v\n%s", err, updateSecondOut.String())
	}

	var compareOut bytes.Buffer
	compareCmd := NewRootCommand(BuildInfo{})
	compareCmd.SetOut(&compareOut)
	compareCmd.SetErr(&compareOut)
	if err := executeForTest(t, ctx, compareCmd,
		"--database", databasePath,
		"--output", "json",
		"compare", "screen", "strategies", "etf-uptrend", "etf-mdd",
		"--as-of", "2024-04-16",
	); err != nil {
		t.Fatalf("compare screen strategies: %v\n%s", err, compareOut.String())
	}
	for _, want := range []string{`"strategies": [`, `"strategy_name": "etf-uptrend"`, `"overlaps": [`} {
		if !strings.Contains(compareOut.String(), want) {
			t.Fatalf("compare screen strategies output missing %q in:\n%s", want, compareOut.String())
		}
	}
}

func TestInspectMarketRegimeAndStrategySetExposeStabilityJSON(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedMarketRegimeDailyBars(t, ctx, databasePath)

	dir := t.TempDir()
	regimePath := filepath.Join(dir, "regime.yaml")
	if err := os.WriteFile(regimePath, []byte(`kind: MarketRegime
schema_version: 1
name: us-growth-regime
spec:
  benchmark:
    symbol: "379810"
    market: krx
    security_type: etf
  evaluation:
    lookback_days: 10
    confirm_days: 7
  rules:
    - regime: uptrend
      when:
        return_20d_gte: 0.03
        close_above_ma20: true
        ma20_above_ma60: true
    - regime: sideways
      when:
        return_20d_between: [-0.03, 0.03]
`), 0o644); err != nil {
		t.Fatalf("write regime yaml: %v", err)
	}
	strategySetPath := filepath.Join(dir, "strategy-set.yaml")
	if err := os.WriteFile(strategySetPath, []byte(`kind: StrategySet
schema_version: 1
name: etf-swing-by-regime
spec:
  regime: us-growth-regime
  regime_file: regime.yaml
  routes:
    uptrend:
      strategy: nasdaq-uptrend-swing
      version: latest
      min_confidence: 0.7
`), 0o644); err != nil {
		t.Fatalf("write strategy set yaml: %v", err)
	}

	var regimeOut bytes.Buffer
	regimeCmd := NewRootCommand(BuildInfo{})
	regimeCmd.SetOut(&regimeOut)
	regimeCmd.SetErr(&regimeOut)
	if err := executeForTest(t, ctx, regimeCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "market-regime", regimePath,
		"--as-of", "2024-03-10",
	); err != nil {
		t.Fatalf("inspect market regime json: %v\n%s", err, regimeOut.String())
	}
	for _, want := range []string{`"confidence": 1`, `"stable_days": 10`, `"transitions": 0`, `"recent_regimes": [`, `"evidence": [`, `"code": "return_20d_gte"`} {
		if !strings.Contains(regimeOut.String(), want) {
			t.Fatalf("market regime output missing %q in:\n%s", want, regimeOut.String())
		}
	}

	var tableOut bytes.Buffer
	tableCmd := NewRootCommand(BuildInfo{})
	tableCmd.SetOut(&tableOut)
	tableCmd.SetErr(&tableOut)
	if err := executeForTest(t, ctx, tableCmd,
		"--database", databasePath,
		"--output", "table",
		"inspect", "market-regime", regimePath,
		"--as-of", "2024-03-10",
	); err != nil {
		t.Fatalf("inspect market regime table: %v\n%s", err, tableOut.String())
	}
	for _, want := range []string{"confidence", "stable_days", "transitions", "uptrend"} {
		if !strings.Contains(tableOut.String(), want) {
			t.Fatalf("market regime table missing %q in:\n%s", want, tableOut.String())
		}
	}

	var strategySetOut bytes.Buffer
	strategySetCmd := NewRootCommand(BuildInfo{})
	strategySetCmd.SetOut(&strategySetOut)
	strategySetCmd.SetErr(&strategySetOut)
	if err := executeForTest(t, ctx, strategySetCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "strategy-set", strategySetPath,
		"--as-of", "2024-03-10",
	); err != nil {
		t.Fatalf("inspect strategy set json: %v\n%s", err, strategySetOut.String())
	}
	for _, want := range []string{`"selected_route": {`, `"min_confidence": 0.7`, `"confidence": 1`, `"evidence": [`} {
		if !strings.Contains(strategySetOut.String(), want) {
			t.Fatalf("strategy set output missing %q in:\n%s", want, strategySetOut.String())
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

func seedStrategyStockFundamentals(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close stock seed database: %v", err)
		}
	})
	_, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		t.Fatalf("new daily bar repositories: %v", err)
	}
	if _, err := writer.UpsertDailyBars(ctx, []dailybar.Bar{
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupStockPrice,
			Operation:    provider.OperationGetStockPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			Symbol:       "005930",
			Name:         "삼성전자",
			TradingDate:  "2026-05-16",
			Close:        "70000",
			MarketCap:    "1000000000",
		},
	}); err != nil {
		t.Fatalf("seed stock daily bars: %v", err)
	}
	companyRepository, err := companyidentity.NewRepository(database)
	if err != nil {
		t.Fatalf("company identity repository: %v", err)
	}
	_, err = companyRepository.UpsertCompanies(ctx, []companyidentity.CompanyInput{
		{
			Name:        "삼성전자",
			LegalName:   "삼성전자",
			CountryCode: "KR",
			Identifiers: []companyidentity.IdentifierInput{
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
					IdentifierValue: "00126380",
					Primary:         true,
					Confidence:      1,
				},
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeKRXStockCode,
					IdentifierValue: "005930",
					Confidence:      1,
				},
			},
			InstrumentRef: companyidentity.InstrumentRef{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				Symbol:       "005930",
				Name:         "삼성전자",
				RelationType: companyidentity.RelationTypeIssuer,
			},
		},
	})
	if err != nil {
		t.Fatalf("upsert stock company identity: %v", err)
	}
	company, err := companyRepository.Inspect(ctx, "005930")
	if err != nil {
		t.Fatalf("inspect stock company identity: %v", err)
	}
	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("database client: %v", err)
	}
	nowMS := time.Now().UTC().UnixMilli()
	roe := int64(1800)
	metric := storage.FinancialMetricV1Row{
		CompanyID:      company.Company.ID,
		InstrumentID:   company.Instruments[0].InstrumentID,
		Metric:         "roe",
		FiscalYear:     "2025",
		FiscalPeriod:   string(financials.PeriodTypeAnnual),
		AsOfDate:       "2025-12-31",
		ValueBP:        &roe,
		FormulaVersion: "financialmetrics/v1",
		ProvenanceJSON: "{}",
		CreatedAtMS:    nowMS,
		UpdatedAtMS:    nowMS,
	}
	if _, err := client.NewInsert().Model(&metric).Exec(ctx); err != nil {
		t.Fatalf("insert stock financial metric: %v", err)
	}
	per := int64(120000)
	marketCap := int64(1000000000)
	valuation := storage.ValuationSnapshotV1Row{
		CompanyID:           company.Company.ID,
		InstrumentID:        company.Instruments[0].InstrumentID,
		AsOfDate:            "2026-05-16",
		SourcePriceDate:     "2026-05-16",
		MarketCapMinor:      &marketCap,
		PerBP:               &per,
		MetricSourceVersion: "valuation/v1",
		ProvenanceJSON:      "{}",
		UncomputableJSON:    "{}",
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
	}
	if _, err := client.NewInsert().Model(&valuation).Exec(ctx); err != nil {
		t.Fatalf("insert stock valuation: %v", err)
	}
	fact := storage.CompanyFactV1Row{
		CompanyID:                      company.Company.ID,
		InstrumentID:                   company.Instruments[0].InstrumentID,
		Provider:                       string(provider.ProviderOpenDART),
		ProviderGroup:                  string(provider.GroupOpenDARTPeriodicReport),
		Operation:                      string(provider.OperationOpenDARTAuditOpinion),
		ProviderCompanyIdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
		ProviderCompanyIdentifierValue: "00126380",
		FactType:                       "audit_opinion",
		FiscalYear:                     "2025",
		ReportCode:                     "11011",
		RceptNo:                        "20260331000123",
		FactDate:                       "2025-12-31",
		Key:                            "audit_opinion",
		ValueText:                      "적정",
		RawJSON:                        "{}",
		CreatedAtMS:                    nowMS,
		UpdatedAtMS:                    nowMS,
	}
	if _, err := client.NewInsert().Model(&fact).Exec(ctx); err != nil {
		t.Fatalf("insert stock company fact: %v", err)
	}
	event := storage.CompanyEventV1Row{
		CompanyID:     company.Company.ID,
		InstrumentID:  company.Instruments[0].InstrumentID,
		EventType:     "company_merger",
		EventDate:     "2026-05-10",
		RceptDt:       "20260510",
		RceptNo:       "20260510000123",
		Provider:      string(provider.ProviderOpenDART),
		ProviderGroup: string(provider.GroupOpenDARTMaterialEvents),
		Operation:     string(provider.OperationOpenDARTCmpMgDecsn),
		Title:         "합병 결정",
		RawJSON:       "{}",
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if _, err := client.NewInsert().Model(&event).Exec(ctx); err != nil {
		t.Fatalf("insert stock company event: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close seeded stock database: %v", err)
	}
}

func seedMarketRegimeDailyBars(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close market regime seed database: %v", err)
		}
	})
	_, writer, err := dailybarstorage.NewRepositories(database)
	if err != nil {
		t.Fatalf("new daily bar repositories: %v", err)
	}
	startDate := mustParseDateForTest(t, "2024-01-01")
	bars := make([]dailybar.Bar, 0, 70)
	for i := 0; i < 70; i++ {
		closePrice := strconv.Itoa(100 + i)
		bars = append(bars, dailybar.Bar{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "379810",
			Name:         "KODEX US NASDAQ 100 TR",
			TradingDate:  startDate.AddDate(0, 0, i).Format(time.DateOnly),
			Open:         closePrice,
			High:         closePrice,
			Low:          closePrice,
			Close:        closePrice,
		})
	}
	if _, err := writer.UpsertDailyBars(ctx, bars); err != nil {
		t.Fatalf("seed market regime daily bars: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close seeded market regime database: %v", err)
	}
}

func mustParseDateForTest(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatalf("parse date %s: %v", value, err)
	}
	return parsed
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return data
}
