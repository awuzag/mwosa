package stocksummary

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyevent"
	"github.com/ev3rlit/mwosa/storage/companyfact"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/ev3rlit/mwosa/storage/financialstatement"
	"github.com/stretchr/testify/require"
)

func TestRepositoryInspectBuildsStoredStockSummary(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	company := seedCompany(t, ctx, database)
	require.NoError(t, seedSummaryData(ctx, database, company))

	repository, err := NewRepository(database)
	require.NoError(t, err)
	summary, err := repository.Inspect(ctx, "005930", Query{
		Sections: []string{SectionAll},
		AsOf:     "latest",
		Window:   3,
		Period:   financials.PeriodTypeAnnual,
	})
	require.NoError(t, err)
	require.Equal(t, []string{SectionProfile, SectionInvestment, SectionFinancials, SectionScores, SectionDividends, SectionEvents}, summary.Sections)
	require.Equal(t, "삼성전자", summary.Company.Name)
	require.Equal(t, "005930", summary.Instrument.Symbol)
	require.Len(t, summary.Valuation, 1)
	require.Equal(t, int64(120000), *summary.Valuation[0].PerBP)
	require.Len(t, summary.Metrics, 1)
	require.Equal(t, "roe", summary.Metrics[0].Metric)
	require.NotNil(t, summary.Scores)
	require.Equal(t, 72, *summary.Scores.QualityScore)
	require.Equal(t, 82, *summary.Scores.ValuationScore)
	require.Equal(t, "no usable growth metrics", summary.Scores.Uncomputable["growth_score"])
	require.Len(t, summary.Dividends, 1)
	require.Equal(t, "9809437000000", summary.Dividends[0].ValueNumber)
	require.Empty(t, summary.Facts)
	require.Len(t, summary.Events, 1)
	require.Equal(t, "company_merger", summary.Events[0].EventType)
	require.Empty(t, summary.Missing)
	require.Equal(t, "삼성전자", summary.Profile.Company.Name)
	require.NotNil(t, summary.Profile.FinancialProfile)
	require.Equal(t, "roe", summary.Profile.FinancialProfile.Metrics[0].Metric)
	require.NotNil(t, summary.Profile.ValuationProfile)
	require.Equal(t, int64(120000), *summary.Profile.ValuationProfile.Snapshots[0].PerBP)
	require.NotNil(t, summary.Profile.ScreenCandidate)
	require.Equal(t, "005930", summary.Profile.ScreenCandidate.Symbol)
	require.Equal(t, "company_fact_v1", summary.Profile.CapitalPolicyProfile.Dividends[0].Source.SourceTable)
	require.Nil(t, summary.Profile.GovernanceProfile)
	require.NotEmpty(t, summary.Report.Tables)
	require.Equal(t, "overview", summary.Report.Tables[0].ID)
}

func TestRepositoryInspectReadsFactsOnlyWhenRequested(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	company := seedCompany(t, ctx, database)
	require.NoError(t, seedSummaryData(ctx, database, company))

	repository, err := NewRepository(database)
	require.NoError(t, err)
	summary, err := repository.Inspect(ctx, "005930", Query{
		Sections: []string{SectionFacts},
		Window:   3,
		Period:   financials.PeriodTypeAnnual,
	})
	require.NoError(t, err)
	require.Equal(t, []string{SectionFacts}, summary.Sections)
	require.Len(t, summary.Facts, 1)
	require.Equal(t, "audit_opinion", summary.Facts[0].FactType)
	require.NotNil(t, summary.Profile.GovernanceProfile)
}

func TestRepositoryInspectMarksMissingOptionalSections(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	seedCompany(t, ctx, database)

	repository, err := NewRepository(database)
	require.NoError(t, err)
	summary, err := repository.Inspect(ctx, "005930", Query{
		Sections: []string{SectionInvestment, SectionFinancials, SectionDividends},
		AsOf:     "latest",
		Window:   3,
		Period:   financials.PeriodTypeAnnual,
	})
	require.NoError(t, err)
	require.Len(t, summary.Missing, 4)
	require.Equal(t, SectionInvestment, summary.Missing[0].Section)
	require.Equal(t, "statements", summary.Missing[2].Section)
}

func seedCompany(t *testing.T, ctx context.Context, database *storage.Database) companyidentity.InspectResult {
	t.Helper()
	companyRepository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
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
	require.NoError(t, err)
	company, err := companyRepository.Inspect(ctx, "005930")
	require.NoError(t, err)
	return company
}

func seedSummaryData(ctx context.Context, database *storage.Database, company companyidentity.InspectResult) error {
	client, err := database.Client(ctx)
	if err != nil {
		return err
	}
	nowMS := time.Now().UTC().UnixMilli()
	per := int64(120000)
	roe := int64(1800)
	marketCap := int64(1000000000)
	closePrice := int64(700000)
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
		return err
	}
	snapshot := storage.ValuationSnapshotV1Row{
		CompanyID:           company.Company.ID,
		InstrumentID:        company.Instruments[0].InstrumentID,
		AsOfDate:            "2026-05-16",
		SourcePriceDate:     "2026-05-16",
		MarketCapMinor:      &marketCap,
		ClosePriceMinor:     &closePrice,
		PerBP:               &per,
		MetricSourceVersion: "valuation/v1",
		ProvenanceJSON:      "{}",
		UncomputableJSON:    "{}",
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
	}
	if _, err := client.NewInsert().Model(&snapshot).Exec(ctx); err != nil {
		return err
	}
	statementRepository, err := financialstatement.NewRepository(database)
	if err != nil {
		return err
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
		return err
	}
	factRepository, err := companyfact.NewRepository(database)
	if err != nil {
		return err
	}
	_, err = factRepository.UpsertFacts(ctx, company, []companyfact.FactInput{
		{
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTPeriodicReport,
			Operation:    provider.OperationOpenDARTAlotMatter,
			FactType:     companyfact.FactTypeDividend,
			FiscalYear:   "2025",
			ReportCode:   "11011",
			RceptNo:      "20260330000001",
			FactDate:     "2025-12-31",
			Key:          "thstrm:현금배당금총액:보통주",
			ValueText:    "9,809,437,000,000",
			ValueNumber:  "9809437000000",
			CurrencyCode: "KRW",
			Raw:          map[string]string{"corp_code": "00126380"},
		},
		{
			Provider:   provider.ProviderOpenDART,
			Group:      provider.GroupOpenDARTPeriodicReport,
			Operation:  provider.OperationOpenDARTAuditOpinion,
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
	if err != nil {
		return err
	}
	eventRepository, err := companyevent.NewRepository(database)
	if err != nil {
		return err
	}
	amountMinor := int64(1000000000)
	_, err = eventRepository.UpsertEvents(ctx, company, []companyevent.EventInput{
		{
			EventType:   "company_merger",
			EventDate:   "2026-03-01",
			RceptNo:     "20260126000001",
			Provider:    provider.ProviderOpenDART,
			Group:       provider.GroupOpenDARTMaterialEvents,
			Operation:   provider.OperationOpenDARTCmpMgDecsn,
			Title:       "회사합병 결정",
			AmountMinor: &amountMinor,
			ValueText:   "합병상대회사=테스트합병대상",
			Raw:         map[string]string{"corp_code": "00126380"},
		},
	})
	return err
}
