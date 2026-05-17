package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyfact"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
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
	for _, want := range []string{"investment", "dividend_yield_bp", "500"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock investment table missing %q in:\n%s", want, out.String())
		}
	}
}

func TestInspectStockScoresTableIncludesFundamentalScores(t *testing.T) {
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
		"--section", "scores",
	); err != nil {
		t.Fatalf("inspect stock scores table: %v\n%s", err, out.String())
	}
	for _, want := range []string{"scores", "quality_score", "72", "valuation_score", "100", "growth_score", "no usable growth metrics"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("inspect stock scores table missing %q in:\n%s", want, out.String())
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
