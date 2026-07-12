package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/service/daily"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	"github.com/spf13/cobra"
)

func TestRenderBarsTableShowsPriceFieldsWithoutProviderMetadata(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeTable, handler.DailyBarsOutput{dailyBarForOutputTest()})
	if err != nil {
		t.Fatalf("render bars table: %v", err)
	}

	got := out.String()
	for _, want := range []string{"date", "symbol", "name", "open", "high", "low", "close", "change", "2026-04-24", "069500", "KODEX 200", "97000", "99000", "96000", "98000", "-500"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("table output should be rendered, not tab-delimited:\n%s", got)
	}
	for _, unwanted := range []string{"┌", "┬", "┐", "│", "├", "┼", "┤", "└", "┴", "┘"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("table output should not include box border %q:\n%s", unwanted, got)
		}
	}
	for _, unwanted := range []string{"provider", "group", "operation", "datago", "securitiesProductPrice", "getETFPriceInfo"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("table output should not include %q:\n%s", unwanted, got)
		}
	}
}

func TestRenderBarsCSVShowsPriceFieldsWithoutProviderMetadata(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.DailyBarsOutput{dailyBarForOutputTest()})
	if err != nil {
		t.Fatalf("render bars csv: %v", err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "date,symbol,name,open,high,low,close,change\n") {
		t.Fatalf("csv header = %q", got)
	}
	for _, unwanted := range []string{"provider", "group", "operation", "datago", "securitiesProductPrice", "getETFPriceInfo"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("csv output should not include %q:\n%s", unwanted, got)
		}
	}
}

func TestRenderCollectResultCSVUsesServiceCSVContract(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.CollectResultOutput{Result: daily.CollectResult{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		ProviderID:   provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Dates:        daily.DateList{"2026-04-24", "2026-04-27"},
		BarsFetched:  10,
		BarsStored:   8,
		RowsAffected: 6,
	}})
	if err != nil {
		t.Fatalf("render collect result csv: %v", err)
	}

	want := "market,security_type,provider,group,dates,fetched,stored,rows_affected\nkrx,etf,datago,securitiesProductPrice,2,10,8,6\n"
	if got := out.String(); got != want {
		t.Fatalf("csv output = %q, want %q", got, want)
	}
}

func TestRenderDailyCoverageOutputsUseFlatRows(t *testing.T) {
	var tableOut bytes.Buffer
	err := Render(&tableOut, OutputModeTable, handler.DailyStorageSummaryOutput{Result: daily.StorageSummaryResult{
		RecordType:   "daily_bar",
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbols:      924,
		Bars:         451220,
		From:         "2024-05-02",
		To:           "2026-05-15",
		Dates:        497,
	}})
	if err != nil {
		t.Fatalf("render storage summary table: %v", err)
	}
	for _, want := range []string{"record_type", "security_type", "symbols", "daily_bar", "etf", "451220", "2024-05-02", "2026-05-15"} {
		if !strings.Contains(tableOut.String(), want) {
			t.Fatalf("storage table missing %q in:\n%s", want, tableOut.String())
		}
	}

	var csvOut bytes.Buffer
	err = Render(&csvOut, OutputModeCSV, handler.DailyCoverageOutput{Result: daily.CoverageResult{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		Name:         "KODEX 200",
		From:         "2024-04-15",
		To:           "2024-04-16",
		Bars:         2,
		Dates:        2,
	}})
	if err != nil {
		t.Fatalf("render coverage csv: %v", err)
	}
	wantCSV := "market,security_type,symbol,name,from,to,bars,dates\nkrx,etf,069500,KODEX 200,2024-04-15,2024-04-16,2,2\n"
	if got := csvOut.String(); got != wantCSV {
		t.Fatalf("coverage csv = %q, want %q", got, wantCSV)
	}
}

func TestRenderFinancialStatementsTableFlattensStatementLines(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeTable, handler.FinancialStatementsOutput{financialStatementForOutputTest()})
	if err != nil {
		t.Fatalf("render financial statements table: %v", err)
	}

	got := out.String()
	for _, want := range []string{"statement", "year", "account", "income_statement", "2025", "Revenue", "1000", "KRW"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"provider", "group", "operation", "fakeProvider"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("table output should not include %q:\n%s", unwanted, got)
		}
	}
}

func TestRenderFinancialStatementsCSVFlattensStatementLines(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.FinancialStatementsOutput{financialStatementForOutputTest()})
	if err != nil {
		t.Fatalf("render financial statements csv: %v", err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "statement,symbol,fiscal_year,fiscal_period,period,account_id,account_name,value,currency,unit\n") {
		t.Fatalf("csv header = %q", got)
	}
	if !strings.Contains(got, "income_statement,005930,2025,FY,annual,ifrs_Revenue,Revenue,1000,KRW,원\n") {
		t.Fatalf("csv output missing financial statement line:\n%s", got)
	}
}

func TestRenderStrategyDetailJSONUsesServiceShape(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeJSON, handler.StrategyDetailOutput{Detail: strategyDetailForOutputTest()})
	if err != nil {
		t.Fatalf("render strategy detail json: %v", err)
	}

	var parsed strategyservice.StrategyDetail
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("strategy detail json should decode: %v\n%s", err, out.String())
	}
	if parsed.Strategy.Name != "momentum" || parsed.ActiveVersion.QueryHash != "hash-1" {
		t.Fatalf("strategy detail json = %#v", parsed)
	}
}

func TestRenderStrategyListNDJSONWritesOneDetailPerLine(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeNDJSON, handler.StrategyListOutput{strategyDetailForOutputTest()})
	if err != nil {
		t.Fatalf("render strategy list ndjson: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("ndjson line count = %d, output:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], `"strategy"`) || !strings.Contains(lines[0], `"active_version"`) {
		t.Fatalf("ndjson line missing strategy detail shape:\n%s", out.String())
	}
}

func TestRenderScreenResultCSVUsesItems(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.ScreenResultOutput{Result: strategyservice.ScreenResult{
		QueryHash:          "query-hash",
		InputDataset:       "etf_daily_metrics",
		InputSchemaVersion: 1,
		ResultCount:        1,
		Items: []strategyservice.ScreenResultItem{{
			Ordinal: 1,
			Symbol:  "069500",
		}},
	}})
	if err != nil {
		t.Fatalf("render screen result csv: %v", err)
	}

	want := "ordinal,symbol\n1,069500\n"
	if got := out.String(); got != want {
		t.Fatalf("screen result csv = %q, want %q", got, want)
	}
}

func TestRenderScreenResultCSVFlattensPayloadRows(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.ScreenResultOutput{Result: strategyservice.ScreenResult{
		QueryHash:          "query-hash",
		InputDataset:       "etf_daily_metrics",
		InputSchemaVersion: 1,
		ResultCount:        1,
		Items: []strategyservice.ScreenResultItem{{
			Ordinal:     0,
			Symbol:      "069500",
			PayloadJSON: json.RawMessage(`{"symbol":"069500","name":"KODEX 200","latest_close":35120,"return_from_first_open_pct":12.5}`),
		}},
	}})
	if err != nil {
		t.Fatalf("render screen result flat csv: %v", err)
	}

	want := "ordinal,symbol,name,latest_close,return_from_first_open_pct\n0,069500,KODEX 200,35120,12.5\n"
	if got := out.String(); got != want {
		t.Fatalf("screen result flat csv = %q, want %q", got, want)
	}
}

func TestRenderScreenRunCSVFlattensPayloadRows(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeCSV, handler.ScreenRunDetailOutput{Detail: strategyservice.ScreenRunDetail{
		Items: []strategyservice.ScreenRunItem{{
			Ordinal:     0,
			Symbol:      "069500",
			PayloadJSON: json.RawMessage(`{"symbol":"069500","first_date":"2025-01-02","latest_date":"2026-05-14","avg_traded_amount_20d":1000}`),
		}},
	}})
	if err != nil {
		t.Fatalf("render screen run flat csv: %v", err)
	}

	want := "ordinal,symbol,first_date,latest_date,avg_traded_amount_20d\n0,069500,2025-01-02,2026-05-14,1000\n"
	if got := out.String(); got != want {
		t.Fatalf("screen run flat csv = %q, want %q", got, want)
	}
}

func TestRenderScreenRunHistoryTableUsesSummaryColumns(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeTable, handler.ScreenRunHistoryOutput{{
		ID:           "run-1",
		Alias:        "open",
		Status:       strategyservice.ScreenRunSucceeded,
		InputDataset: "etf_daily_metrics",
		ResultCount:  3,
		StartedAt:    time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC),
	}})
	if err != nil {
		t.Fatalf("render screen run history table: %v", err)
	}
	got := out.String()
	for _, want := range []string{"id", "alias", "status", "input", "results", "started", "run-1", "open", "succeeded", "3", "2026-05-05T01:02:03Z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("screen history table missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteTableRendersAlignedColumns(t *testing.T) {
	var out bytes.Buffer

	err := writeTable(&out, []string{"kind", "name"}, [][]string{{"etf", "한국 ETF"}})
	if err != nil {
		t.Fatalf("write table: %v", err)
	}

	got := out.String()
	for _, want := range []string{"kind", "name", "etf", "한국 ETF"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"┌", "┬", "┐", "│", "├", "┼", "┤", "└", "┴", "┘"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("table output should not include box border %q:\n%s", unwanted, got)
		}
	}
}

func TestRenderTableUsesPerColumnAlignment(t *testing.T) {
	var out bytes.Buffer

	err := Render(&out, OutputModeTable, alignedTableOutput{
		header:     []string{"name", "value"},
		rows:       [][]string{{"a", "12"}, {"long", "3"}},
		alignments: []string{"left", "right"},
	})
	if err != nil {
		t.Fatalf("render aligned table: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "a        12") || !strings.Contains(got, "long      3") {
		t.Fatalf("table output should right align second column:\n%s", got)
	}
}

func TestRunResultUsesResultDefaultOutputWhenFlagIsUnchanged(t *testing.T) {
	opts := Options{Output: OutputModeTable}
	cmd := &cobra.Command{}
	cmd.Flags().VarP(&opts.Output, "output", "o", "output")
	var out bytes.Buffer
	cmd.SetOut(&out)

	run := runResult(&opts, func(*cobra.Command, []string) (any, error) {
		return preferredJSONOutput{}, nil
	})
	if err := run(cmd, nil); err != nil {
		t.Fatalf("run result: %v", err)
	}
	if !strings.Contains(out.String(), `"mode": "json"`) {
		t.Fatalf("run result should use preferred json output:\n%s", out.String())
	}
}

func TestRunResultKeepsExplicitOutputFlag(t *testing.T) {
	opts := Options{Output: OutputModeTable}
	cmd := &cobra.Command{}
	cmd.Flags().VarP(&opts.Output, "output", "o", "output")
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatalf("set output flag: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)

	run := runResult(&opts, func(*cobra.Command, []string) (any, error) {
		return preferredJSONOutput{}, nil
	})
	if err := run(cmd, nil); err != nil {
		t.Fatalf("run result: %v", err)
	}
	if !strings.Contains(out.String(), "mode") || !strings.Contains(out.String(), "table") {
		t.Fatalf("explicit table output should win:\n%s", out.String())
	}
}

type preferredJSONOutput struct{}

func (preferredJSONOutput) DefaultOutputMode() string {
	return "json"
}

func (preferredJSONOutput) JSONValue() any {
	return map[string]string{"mode": "json"}
}

func (preferredJSONOutput) TableRows() ([]string, [][]string) {
	return []string{"mode"}, [][]string{{"table"}}
}

type alignedTableOutput struct {
	header     []string
	rows       [][]string
	alignments []string
}

func (o alignedTableOutput) TableRows() ([]string, [][]string) {
	return o.header, o.rows
}

func (o alignedTableOutput) TableAlignments() []string {
	return o.alignments
}

func dailyBarForOutputTest() dailybar.Bar {
	return dailybar.Bar{
		Provider:    provider.ProviderDataGo,
		Group:       provider.GroupSecuritiesProductPrice,
		Operation:   provider.OperationGetETFPriceInfo,
		Symbol:      "069500",
		Name:        "KODEX 200",
		TradingDate: "2026-04-24",
		Open:        "97000",
		High:        "99000",
		Low:         "96000",
		Close:       "98000",
		Change:      "-500",
		Volume:      "16267003",
	}
}

func financialStatementForOutputTest() financials.Statement {
	return financials.Statement{
		Provider:     provider.ProviderID("fakeProvider"),
		Group:        provider.GroupID("fakeGroup"),
		Operation:    provider.OperationID("fakeOperation"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Statement:    financials.StatementTypeIncomeStatement,
		Symbol:       "005930",
		FiscalYear:   "2025",
		FiscalPeriod: "FY",
		Period:       financials.PeriodTypeAnnual,
		Currency:     "KRW",
		Unit:         "원",
		Lines: []financials.LineItem{
			{AccountID: "ifrs_Revenue", AccountName: "Revenue", Value: "1000"},
		},
	}
}

func strategyDetailForOutputTest() strategyservice.StrategyDetail {
	createdAt := time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC)
	return strategyservice.StrategyDetail{
		Strategy: strategyservice.Strategy{
			ID:              "strategy-1",
			Name:            "momentum",
			Engine:          strategyservice.EngineJQ,
			ActiveVersionID: "version-1",
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		},
		ActiveVersion: strategyservice.StrategyVersion{
			ID:                 "version-1",
			StrategyID:         "strategy-1",
			Version:            1,
			QueryText:          ".[]",
			QueryHash:          "hash-1",
			InputDataset:       "etf_daily_metrics",
			InputSchemaVersion: 1,
			CreatedAt:          createdAt,
		},
	}
}
