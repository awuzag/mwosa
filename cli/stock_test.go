package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyfact"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/ev3rlit/mwosa/storage/financialstatement"
)

func TestInspectStockHasCommandSurface(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	inspectStock, _, err := cmd.Find([]string{"inspect", "stock"})
	if err != nil {
		t.Fatalf("find inspect stock: %v", err)
	}
	if inspectStock == nil || inspectStock.Use != "stock <symbol-or-company>" {
		t.Fatalf("inspect stock command = %#v", inspectStock)
	}
}

func TestInspectStockReadsStoredSummaryJSON(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"inspect", "stock", "005930",
		"--section", "profile,financials",
	); err != nil {
		t.Fatalf("inspect stock: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"sections"`, `"profile"`, `"financials"`, `"name": "삼성전자"`, `"metric": "roe"`, `"value_bp": 1800`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock output missing %q in:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), `"investment"`) || strings.Contains(out.String(), `"fundamental_scores"`) || strings.Contains(out.String(), `"dividends"`) || strings.Contains(out.String(), `"facts"`) || strings.Contains(out.String(), `"events"`) {
		t.Fatalf("inspect stock output included unrequested sections:\n%s", out.String())
	}
}

func TestInspectStockReadsFactsAndEvents(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"inspect", "stock", "005930",
		"--section", "facts,events",
	); err != nil {
		t.Fatalf("inspect stock facts/events: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"sections"`, `"facts"`, `"events"`, `"fact_type": "audit_opinion"`, `"key": "당기:opinion"`, `"value_text": "적정"`, `"event_type": "company_merger"`, `"title": "회사합병 결정"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock facts/events output missing %q in:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), `"dividends"`) || strings.Contains(out.String(), `"investment"`) {
		t.Fatalf("inspect stock facts/events output included unrequested sections:\n%s", out.String())
	}
}

func TestInspectStockAllDoesNotIncludeFacts(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"inspect", "stock", "005930",
		"--section", "all",
	); err != nil {
		t.Fatalf("inspect stock all: %v\n%s", err, out.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("inspect stock all output should be json: %v\n%s", err, out.String())
	}
	if _, ok := got["facts"]; ok {
		t.Fatalf("inspect stock all output included top-level facts:\n%s", out.String())
	}
	profile, ok := got["research_profile"].(map[string]any)
	if !ok {
		t.Fatalf("inspect stock all output missing research profile:\n%s", out.String())
	}
	if _, ok := profile["governance_profile"]; ok {
		t.Fatalf("inspect stock all output included governance profile:\n%s", out.String())
	}
	candidate, ok := profile["screen_candidate"].(map[string]any)
	if !ok {
		t.Fatalf("inspect stock all output missing screen candidate:\n%s", out.String())
	}
	if _, ok := candidate["company_facts"]; ok {
		t.Fatalf("inspect stock all output included screen candidate facts:\n%s", out.String())
	}
}

func TestInspectStockInvestmentTableIncludesDividendYield(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "table",
		"inspect", "stock", "005930",
		"--section", "investment",
	); err != nil {
		t.Fatalf("inspect stock investment table: %v\n%s", err, out.String())
	}
	for _, want := range []string{"투자 지표", "배당수익률", "5.00%"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock investment table missing %q in:\n%s", want, out.String())
		}
	}
}

func TestInspectStockAllTableUsesReportBlocks(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "table",
		"inspect", "stock", "005930",
		"--section", "all",
	); err != nil {
		t.Fatalf("inspect stock all table: %v\n%s", err, out.String())
	}
	for _, want := range []string{"회사 개요", "기업명", "종목코드", "투자 지표", "재무 요약", "수익성 추이", "성장성 추이", "손익계산서", "배당 요약", "리스크 요약", "1,000 KRW"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock all table missing %q in:\n%s", want, out.String())
		}
	}
	for _, unwanted := range []string{"section  key  value  source", "facts 상세", "dividends 상세", "누락/계산 불가"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("inspect stock all table included raw dump marker %q in:\n%s", unwanted, out.String())
		}
	}
}

func TestInspectStockFactsTableUsesRawDetailBlock(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedStockSummaryCLIData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "table",
		"inspect", "stock", "005930",
		"--section", "facts",
	); err != nil {
		t.Fatalf("inspect stock facts table: %v\n%s", err, out.String())
	}
	for _, want := range []string{"facts 상세", "fact_type", "audit_opinion", "적정"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock facts table missing %q in:\n%s", want, out.String())
		}
	}
}

func seedStockSummaryCLIData(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	companyRepository, err := companyidentity.NewRepository(database)
	if err != nil {
		t.Fatalf("company repository: %v", err)
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
		t.Fatalf("upsert company: %v", err)
	}
	company, err := companyRepository.Inspect(ctx, "005930")
	if err != nil {
		t.Fatalf("inspect company: %v", err)
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
		t.Fatalf("insert metric: %v", err)
	}
	dividendYield := int64(500)
	valuation := storage.ValuationSnapshotV1Row{
		CompanyID:           company.Company.ID,
		InstrumentID:        company.Instruments[0].InstrumentID,
		AsOfDate:            "2026-05-16",
		SourcePriceDate:     "2026-05-16",
		DividendYieldBP:     &dividendYield,
		MetricSourceVersion: "financialmetrics/v1",
		ProvenanceJSON:      "{}",
		UncomputableJSON:    "{}",
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
	}
	if _, err := client.NewInsert().Model(&valuation).Exec(ctx); err != nil {
		t.Fatalf("insert valuation: %v", err)
	}
	statementRepository, err := financialstatement.NewRepository(database)
	if err != nil {
		t.Fatalf("financial statement repository: %v", err)
	}
	if _, err := statementRepository.UpsertStatements(ctx, company, []financials.Statement{
		{
			Statement:    financials.StatementTypeSummary,
			Symbol:       "005930",
			Name:         "삼성전자",
			FiscalYear:   "2025",
			FiscalPeriod: "11011",
			Period:       financials.PeriodTypeAnnual,
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTFinancials,
			Operation:    provider.OperationOpenDARTSinglAcntAll,
			Extensions: map[string]string{
				"reprt_code": "11011",
				"fs_div":     "CFS",
			},
			Lines: []financials.LineItem{
				{
					AccountID:   "ifrs-full_Revenue",
					AccountName: "매출액",
					Value:       "1,000",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "IS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "1"},
				},
				{
					AccountID:   "dart_OperatingIncomeLoss",
					AccountName: "영업이익",
					Value:       "150",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "IS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "2"},
				},
				{
					AccountID:   "ifrs-full_ProfitLoss",
					AccountName: "당기순이익",
					Value:       "90",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "IS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "3"},
				},
				{
					AccountID:   "ifrs-full_Assets",
					AccountName: "자산총계",
					Value:       "3,000",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "BS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "1"},
				},
				{
					AccountID:   "ifrs-full_Liabilities",
					AccountName: "부채총계",
					Value:       "1,000",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "BS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "2"},
				},
				{
					AccountID:   "ifrs-full_Equity",
					AccountName: "자본총계",
					Value:       "2,000",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "BS", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "3"},
				},
				{
					AccountID:   "ifrs-full_CashFlowsFromUsedInOperatingActivities",
					AccountName: "영업활동 현금흐름",
					Value:       "120",
					Currency:    "KRW",
					Extensions:  map[string]string{"sj_div": "CF", "reprt_code": "11011", "fs_div": "CFS", "rcept_no": "20260330000001", "ord": "1"},
				},
			},
		},
	}); err != nil {
		t.Fatalf("upsert financial statements: %v", err)
	}
	facts := []storage.CompanyFactV1Row{
		{
			CompanyID:                      company.Company.ID,
			InstrumentID:                   company.Instruments[0].InstrumentID,
			Provider:                       string(provider.ProviderOpenDART),
			ProviderGroup:                  string(provider.GroupOpenDARTPeriodicReport),
			Operation:                      string(provider.OperationOpenDARTAlotMatter),
			ProviderCompanyIdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
			ProviderCompanyIdentifierValue: "00126380",
			FactType:                       companyfact.FactTypeDividend,
			FiscalYear:                     "2025",
			ReportCode:                     "11011",
			RceptNo:                        "20260330000001",
			FactDate:                       "2025-12-31",
			Key:                            "thstrm:현금배당금총액:보통주",
			ValueText:                      "9,809,437,000,000",
			ValueNumber:                    "9809437000000",
			CurrencyCode:                   "KRW",
			RawJSON:                        `{"corp_code":"00126380"}`,
			CreatedAtMS:                    nowMS,
			UpdatedAtMS:                    nowMS,
		},
		{
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
			RceptNo:                        "20260330000002",
			FactDate:                       "2025-12-31",
			Key:                            "당기:opinion",
			ValueText:                      "적정",
			RawJSON:                        `{"corp_code":"00126380"}`,
			CreatedAtMS:                    nowMS,
			UpdatedAtMS:                    nowMS,
		},
	}
	if _, err := client.NewInsert().Model(&facts).Exec(ctx); err != nil {
		t.Fatalf("insert facts: %v", err)
	}
	amountMinor := int64(1000000000)
	event := storage.CompanyEventV1Row{
		CompanyID:     company.Company.ID,
		InstrumentID:  company.Instruments[0].InstrumentID,
		EventType:     "company_merger",
		EventDate:     "2026-03-01",
		RceptNo:       "20260126000001",
		Provider:      string(provider.ProviderOpenDART),
		ProviderGroup: string(provider.GroupOpenDARTMaterialEvents),
		Operation:     string(provider.OperationOpenDARTCmpMgDecsn),
		Title:         "회사합병 결정",
		AmountMinor:   &amountMinor,
		ValueText:     "합병상대회사=테스트합병대상",
		RawJSON:       `{"corp_code":"00126380"}`,
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if _, err := client.NewInsert().Model(&event).Exec(ctx); err != nil {
		t.Fatalf("insert event: %v", err)
	}
}
