package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyfact"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/ev3rlit/mwosa/storage/valuation"
	"github.com/stretchr/testify/require"
)

func TestGetFinancialsHasCommandSurface(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	get, _, err := cmd.Find([]string{"get", "financials"})
	if err != nil {
		t.Fatalf("find get financials: %v", err)
	}
	if get == nil || get.Use != "financials <company>" {
		t.Fatalf("get financials command = %#v", get)
	}
	getStatements, _, err := cmd.Find([]string{"get", "financials", "statements"})
	if err != nil {
		t.Fatalf("find get financials statements: %v", err)
	}
	if getStatements == nil || getStatements.Use != "statements <company>" {
		t.Fatalf("get financials statements command = %#v", getStatements)
	}
	getMetrics, _, err := cmd.Find([]string{"get", "financials", "metrics"})
	if err != nil {
		t.Fatalf("find get financials metrics: %v", err)
	}
	if getMetrics == nil || getMetrics.Use != "metrics <company>" {
		t.Fatalf("get financials metrics command = %#v", getMetrics)
	}
	calcMetrics, _, err := cmd.Find([]string{"calc", "financials", "metrics"})
	if err != nil {
		t.Fatalf("find calc financials metrics: %v", err)
	}
	if calcMetrics == nil || calcMetrics.Use != "metrics <company>" {
		t.Fatalf("calc financials metrics command = %#v", calcMetrics)
	}
	getValuation, _, err := cmd.Find([]string{"get", "financials", "valuation"})
	if err != nil {
		t.Fatalf("find get financials valuation: %v", err)
	}
	if getValuation == nil || getValuation.Use != "valuation <company>" {
		t.Fatalf("get financials valuation command = %#v", getValuation)
	}
	calcValuation, _, err := cmd.Find([]string{"calc", "financials", "valuation"})
	if err != nil {
		t.Fatalf("find calc financials valuation: %v", err)
	}
	if calcValuation == nil || calcValuation.Use != "valuation <company>" {
		t.Fatalf("calc financials valuation command = %#v", calcValuation)
	}
	getDividends, _, err := cmd.Find([]string{"get", "financials", "dividends"})
	if err != nil {
		t.Fatalf("find get financials dividends: %v", err)
	}
	if getDividends == nil || getDividends.Use != "dividends <company>" {
		t.Fatalf("get financials dividends command = %#v", getDividends)
	}
	getHealth, _, err := cmd.Find([]string{"get", "financials", "health"})
	if err != nil {
		t.Fatalf("find get financials health: %v", err)
	}
	if getHealth == nil || getHealth.Use != "health <company>" {
		t.Fatalf("get financials health command = %#v", getHealth)
	}
	getFacts, _, err := cmd.Find([]string{"get", "financials", "facts"})
	if err != nil {
		t.Fatalf("find get financials facts: %v", err)
	}
	if getFacts == nil || getFacts.Use != "facts <company>" {
		t.Fatalf("get financials facts command = %#v", getFacts)
	}
	syncDividends, _, err := cmd.Find([]string{"sync", "financials", "dividends"})
	if err != nil {
		t.Fatalf("find sync financials dividends: %v", err)
	}
	if syncDividends == nil || syncDividends.Use != "dividends <company>" {
		t.Fatalf("sync financials dividends command = %#v", syncDividends)
	}
	syncFacts, _, err := cmd.Find([]string{"sync", "financials", "facts"})
	if err != nil {
		t.Fatalf("find sync financials facts: %v", err)
	}
	if syncFacts == nil || syncFacts.Use != "facts <company>" {
		t.Fatalf("sync financials facts command = %#v", syncFacts)
	}
}

func TestGetFinancialHealthReadsStoredMetricsAndAuditFacts(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	seedFinancialHealthData(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--database", databasePath,
		"--output", "json",
		"get", "financials", "health", "005930",
		"--window", "3y",
	); err != nil {
		t.Fatalf("get financials health: %v\n%s", err, out.String())
	}
	for _, want := range []string{
		`"metric": "debt_to_equity"`,
		`"category": "stability"`,
		`"metric": "interest_coverage"`,
		`"status": "uncomputable"`,
		`"fact_type": "audit_opinion"`,
		`"value_text": "적정"`,
		`"missing"`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("financial health output missing %q in:\n%s", want, out.String())
		}
	}
}

func TestFinancialValuationTableRowsIncludeDividendYield(t *testing.T) {
	dividendYield := int64(500)
	output := financialValuationOutput{
		Snapshots: []valuation.Snapshot{
			{
				AsOfDate:        "2026-05-16",
				SourcePriceDate: "2026-05-16",
				DividendYieldBP: &dividendYield,
			},
		},
	}

	header, rows := output.TableRows()
	require.Contains(t, header, "dividend_yield_bp")
	require.Len(t, rows, 1)
	require.Contains(t, rows[0], "500")
}

func TestGetFinancialsWithoutProviderReportsFinancialsCapability(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := executeForTest(t, context.Background(), cmd,
		"--database", filepath.Join(t.TempDir(), "mwosa.db"),
		"get", "financials", "005930",
		"--year", "2025",
	)
	if err == nil {
		t.Fatal("get financials error = nil, want no provider error")
	}
	for _, want := range []string{"financials", "no provider candidate", "symbol=005930"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

func seedFinancialHealthData(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	companyRepository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	_, err = companyRepository.UpsertCompanies(ctx, []companyidentity.CompanyInput{
		{
			Name:        "삼성전자",
			LegalName:   "삼성전자",
			CountryCode: "KR",
			Identifiers: []companyidentity.IdentifierInput{
				{
					Provider:        core.ProviderOpenDART,
					Group:           core.GroupOpenDARTDisclosure,
					Operation:       core.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
					IdentifierValue: "00126380",
					Primary:         true,
					Confidence:      1,
				},
				{
					Provider:        core.ProviderOpenDART,
					Group:           core.GroupOpenDARTDisclosure,
					Operation:       core.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeKRXStockCode,
					IdentifierValue: "005930",
					Confidence:      1,
				},
			},
			InstrumentRef: companyidentity.InstrumentRef{
				Market:       core.MarketKRX,
				SecurityType: core.SecurityTypeStock,
				Symbol:       "005930",
				Name:         "삼성전자",
				RelationType: companyidentity.RelationTypeIssuer,
			},
		},
	})
	require.NoError(t, err)
	company, err := companyRepository.Inspect(ctx, "005930")
	require.NoError(t, err)
	client, err := database.Client(ctx)
	require.NoError(t, err)
	nowMS := time.Now().UTC().UnixMilli()
	for _, row := range []storage.FinancialMetricV1Row{
		{
			CompanyID:      company.Company.ID,
			InstrumentID:   company.Instruments[0].InstrumentID,
			Metric:         "debt_to_equity",
			FiscalYear:     "2025",
			FiscalPeriod:   string(financials.PeriodTypeAnnual),
			AsOfDate:       "2025-12-31",
			ValueDecimal:   "0.42000000",
			FormulaVersion: "financialmetrics/v1",
			ProvenanceJSON: "{}",
			CreatedAtMS:    nowMS,
			UpdatedAtMS:    nowMS,
		},
		{
			CompanyID:      company.Company.ID,
			InstrumentID:   company.Instruments[0].InstrumentID,
			Metric:         "current_ratio",
			FiscalYear:     "2025",
			FiscalPeriod:   string(financials.PeriodTypeAnnual),
			AsOfDate:       "2025-12-31",
			ValueDecimal:   "1.75000000",
			FormulaVersion: "financialmetrics/v1",
			ProvenanceJSON: "{}",
			CreatedAtMS:    nowMS,
			UpdatedAtMS:    nowMS,
		},
		{
			CompanyID:          company.Company.ID,
			InstrumentID:       company.Instruments[0].InstrumentID,
			Metric:             "interest_coverage",
			FiscalYear:         "2025",
			FiscalPeriod:       string(financials.PeriodTypeAnnual),
			AsOfDate:           "2025-12-31",
			FormulaVersion:     "financialmetrics/v1",
			UncomputableReason: "interest expense account not found",
			ProvenanceJSON:     "{}",
			CreatedAtMS:        nowMS,
			UpdatedAtMS:        nowMS,
		},
	} {
		_, err = client.NewInsert().Model(&row).Exec(ctx)
		require.NoError(t, err)
	}
	factRepository, err := companyfact.NewRepository(database)
	require.NoError(t, err)
	_, err = factRepository.UpsertFacts(ctx, company, []companyfact.FactInput{
		{
			Provider:   core.ProviderOpenDART,
			Group:      core.GroupOpenDARTPeriodicReport,
			Operation:  core.OperationOpenDARTAuditOpinion,
			FactType:   "audit_opinion",
			FiscalYear: "2025",
			ReportCode: "11011",
			RceptNo:    "20260330000002",
			FactDate:   "2025-12-31",
			Key:        "당기:opinion",
			ValueText:  "적정",
			Raw:        map[string]string{"corp_code": "00126380"},
		},
	})
	require.NoError(t, err)
}
