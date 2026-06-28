//go:build integration

package strategyfundamentals

import (
	"context"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoRepositoryListsLatestFundamentalsBySymbol(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_strategy_fundamentals_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	company := seedMongoCompany(t, ctx, runtime)
	seedMongoFundamentals(t, ctx, runtime, company)

	repository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	result, err := repository.ListLatestFundamentals(ctx, strategyservice.FundamentalsQuery{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
	})
	require.NoError(t, err)
	item, ok := result["005930"]
	require.True(t, ok)
	require.Equal(t, "005930", item.Symbol)
	require.Equal(t, int64(1900), *item.Metrics["roe"].ValueBP)
	require.Equal(t, "2025", item.Metrics["roe"].FiscalYear)
	require.NotNil(t, item.Valuation)
	require.Equal(t, int64(120000), *item.Valuation.PerBP)
	require.Equal(t, "2026-05-16", item.Valuation.AsOfDate)
	require.Equal(t, "적정", item.Facts["audit_opinion"].ValueText)
	require.Equal(t, "2025", item.Facts["audit_opinion"].FiscalYear)
	require.Len(t, item.Events, 1)
	require.Equal(t, "company_merger", item.Events[0].EventType)
	require.Equal(t, "2026-05-10", item.Events[0].EventDate)
}

func seedMongoCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
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

func seedMongoFundamentals(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime, company companyidentity.InspectResult) {
	t.Helper()

	now := time.Now().UTC()
	instrument := company.Instruments[0]
	oldROE := int64(1700)
	newROE := int64(1900)
	_, err := runtime.Database().Collection("financial_metrics").InsertMany(ctx, []any{
		bson.D{
			{Key: "_id", Value: "financial_metrics:test:roe:2024"},
			{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
			{Key: "revision", Value: int64(1)},
			{Key: "created_at", Value: now},
			{Key: "updated_at", Value: now},
			{Key: "company_id", Value: company.Company.ID},
			{Key: "instrument_id", Value: instrument.InstrumentID},
			{Key: "instrument", Value: strategyFundamentalsTestInstrument(instrument)},
			{Key: "metric", Value: "roe"},
			{Key: "fiscal_year", Value: "2024"},
			{Key: "fiscal_period", Value: string(financials.PeriodTypeAnnual)},
			{Key: "as_of_date", Value: "2024-12-31"},
			{Key: "value_bp", Value: oldROE},
			{Key: "formula_version", Value: "financialmetrics/v1"},
			{Key: "provenance", Value: bson.D{}},
		},
		bson.D{
			{Key: "_id", Value: "financial_metrics:test:roe:2025"},
			{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
			{Key: "revision", Value: int64(1)},
			{Key: "created_at", Value: now},
			{Key: "updated_at", Value: now},
			{Key: "company_id", Value: company.Company.ID},
			{Key: "instrument_id", Value: instrument.InstrumentID},
			{Key: "instrument", Value: strategyFundamentalsTestInstrument(instrument)},
			{Key: "metric", Value: "roe"},
			{Key: "fiscal_year", Value: "2025"},
			{Key: "fiscal_period", Value: string(financials.PeriodTypeAnnual)},
			{Key: "as_of_date", Value: "2025-12-31"},
			{Key: "value_bp", Value: newROE},
			{Key: "formula_version", Value: "financialmetrics/v1"},
			{Key: "provenance", Value: bson.D{}},
		},
	})
	require.NoError(t, err)

	per := int64(120000)
	_, err = runtime.Database().Collection("valuation_snapshots").InsertOne(ctx, bson.D{
		{Key: "_id", Value: "valuation_snapshots:test:2026-05-16"},
		{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "company_id", Value: company.Company.ID},
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "instrument", Value: strategyFundamentalsTestInstrument(instrument)},
		{Key: "as_of_date", Value: "2026-05-16"},
		{Key: "source_price_date", Value: "2026-05-16"},
		{Key: "per_bp", Value: per},
		{Key: "metric_source_version", Value: "valuation/v1"},
		{Key: "provenance", Value: bson.D{}},
		{Key: "uncomputable", Value: bson.D{}},
	})
	require.NoError(t, err)

	_, err = runtime.Database().Collection("company_facts").InsertOne(ctx, bson.D{
		{Key: "_id", Value: "company_facts:test:audit_opinion"},
		{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "company_id", Value: company.Company.ID},
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "instrument", Value: strategyFundamentalsTestInstrument(instrument)},
		{Key: "provider", Value: string(provider.ProviderOpenDART)},
		{Key: "provider_group", Value: string(provider.GroupOpenDARTPeriodicReport)},
		{Key: "operation", Value: string(provider.OperationOpenDARTAuditOpinion)},
		{Key: "fact_type", Value: "audit_opinion"},
		{Key: "fiscal_year", Value: "2025"},
		{Key: "report_code", Value: "11011"},
		{Key: "rcept_no", Value: "20260331000123"},
		{Key: "fact_date", Value: "2025-12-31"},
		{Key: "key", Value: "audit_opinion"},
		{Key: "value_text", Value: "적정"},
		{Key: "raw", Value: bson.D{}},
	})
	require.NoError(t, err)

	_, err = runtime.Database().Collection("company_events").InsertOne(ctx, bson.D{
		{Key: "_id", Value: "company_events:test:merger"},
		{Key: "schema_version", Value: storagemongodb.SchemaVersion1},
		{Key: "revision", Value: int64(1)},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "company_id", Value: company.Company.ID},
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "instrument", Value: strategyFundamentalsTestInstrument(instrument)},
		{Key: "provider", Value: string(provider.ProviderOpenDART)},
		{Key: "provider_group", Value: string(provider.GroupOpenDARTMaterialEvents)},
		{Key: "operation", Value: string(provider.OperationOpenDARTCmpMgDecsn)},
		{Key: "event_type", Value: "company_merger"},
		{Key: "event_date", Value: "2026-05-10"},
		{Key: "rcept_dt", Value: "20260510"},
		{Key: "rcept_no", Value: "20260510000123"},
		{Key: "effective_date", Value: "2026-05-10"},
		{Key: "title", Value: "합병 결정"},
		{Key: "raw", Value: bson.D{}},
	})
	require.NoError(t, err)
}

func strategyFundamentalsTestInstrument(instrument companyidentity.InstrumentLink) bson.D {
	return bson.D{
		{Key: "instrument_id", Value: instrument.InstrumentID},
		{Key: "market", Value: string(instrument.Market)},
		{Key: "security_type", Value: string(instrument.SecurityType)},
		{Key: "symbol", Value: instrument.Symbol},
		{Key: "name", Value: instrument.Name},
	}
}
