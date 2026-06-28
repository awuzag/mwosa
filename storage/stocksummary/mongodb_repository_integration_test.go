//go:build integration

package stocksummary

import (
	"context"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage/companyevent"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialstatement"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoRepositoryInspectBuildsStoredStockSummary(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_stock_summary_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	company := seedMongoSummaryCompany(t, ctx, runtime)
	seedMongoSummaryData(t, ctx, runtime, company)

	repository, err := NewMongoRepository(runtime.Database())
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
	require.Len(t, summary.Dividends, 1)
	require.Equal(t, "9809437000000", summary.Dividends[0].ValueNumber)
	require.Empty(t, summary.Facts)
	require.Len(t, summary.Events, 1)
	require.Equal(t, "company_merger", summary.Events[0].EventType)
	require.Empty(t, summary.Missing)
	require.Equal(t, "삼성전자", summary.Profile.Company.Name)
	require.NotNil(t, summary.Profile.FinancialProfile)
	require.NotNil(t, summary.Profile.ValuationProfile)
	require.NotNil(t, summary.Profile.ScreenCandidate)
	require.Equal(t, "005930", summary.Profile.ScreenCandidate.Symbol)
	require.NotEmpty(t, summary.Report.Tables)
	require.Equal(t, "overview", summary.Report.Tables[0].ID)
}

func seedMongoSummaryCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	_, err = repository.UpsertCompanies(ctx, []companyidentity.CompanyInput{
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
	company, err := repository.Inspect(ctx, "005930")
	require.NoError(t, err)
	return company
}

func seedMongoSummaryData(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime, company companyidentity.InspectResult) {
	t.Helper()

	now := time.Now().UTC()
	instrument := company.Instruments[0]
	per := int64(120000)
	roe := int64(1800)
	marketCap := int64(1000000000)
	closePrice := int64(700000)
	_, err := runtime.Database().Collection("financial_metrics").InsertOne(ctx, bson.D{
		{Key: "_id", Value: "financial_metrics:stocksummary:roe"},
		{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "company_id", Value: company.Company.ID},
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "instrument", Value: stockSummaryTestInstrument(instrument)},
		{Key: "metric", Value: "roe"},
		{Key: "fiscal_year", Value: "2025"},
		{Key: "fiscal_period", Value: string(financials.PeriodTypeAnnual)},
		{Key: "as_of_date", Value: "2025-12-31"},
		{Key: "value_bp", Value: roe},
		{Key: "formula_version", Value: "financialmetrics/v1"},
		{Key: "provenance", Value: bson.D{}},
	})
	require.NoError(t, err)
	_, err = runtime.Database().Collection("valuation_snapshots").InsertOne(ctx, bson.D{
		{Key: "_id", Value: "valuation_snapshots:stocksummary:2026-05-16"},
		{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "company_id", Value: company.Company.ID},
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "instrument", Value: stockSummaryTestInstrument(instrument)},
		{Key: "as_of_date", Value: "2026-05-16"},
		{Key: "source_price_date", Value: "2026-05-16"},
		{Key: "market_cap_minor", Value: marketCap},
		{Key: "close_price_minor", Value: closePrice},
		{Key: "per_bp", Value: per},
		{Key: "metric_source_version", Value: "valuation/v1"},
		{Key: "provenance", Value: bson.D{}},
		{Key: "uncomputable", Value: bson.D{}},
	})
	require.NoError(t, err)

	statementRepository, err := financialstatement.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	_, err = statementRepository.UpsertStatements(ctx, company, []financials.Statement{
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
				{AccountID: "ifrs-full_Revenue", AccountName: "매출액", Value: "1,000", Currency: "KRW", Extensions: map[string]string{"sj_div": "IS", "ord": "1"}},
				{AccountID: "dart_OperatingIncomeLoss", AccountName: "영업이익", Value: "150", Currency: "KRW", Extensions: map[string]string{"sj_div": "IS", "ord": "2"}},
				{AccountID: "ifrs-full_ProfitLoss", AccountName: "당기순이익", Value: "90", Currency: "KRW", Extensions: map[string]string{"sj_div": "IS", "ord": "3"}},
			},
		},
	})
	require.NoError(t, err)

	factRepository, err := companyfact.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
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
	})
	require.NoError(t, err)

	eventRepository, err := companyevent.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
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
	require.NoError(t, err)
}

func stockSummaryTestInstrument(instrument companyidentity.InstrumentLink) bson.D {
	return bson.D{
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "market", Value: string(instrument.Market)},
		{Key: "security_type", Value: string(instrument.SecurityType)},
		{Key: "symbol", Value: instrument.Symbol},
		{Key: "name", Value: instrument.Name},
	}
}
