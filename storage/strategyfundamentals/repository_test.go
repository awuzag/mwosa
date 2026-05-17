package strategyfundamentals

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/stretchr/testify/require"
)

func TestRepositoryListsLatestFundamentalsBySymbol(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	company := seedCompany(t, ctx, database)
	require.NoError(t, seedFundamentals(ctx, database, company))

	repository, err := NewRepository(database)
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

func seedFundamentals(ctx context.Context, database *storage.Database, company companyidentity.InspectResult) error {
	client, err := database.Client(ctx)
	if err != nil {
		return err
	}
	nowMS := time.Now().UTC().UnixMilli()
	oldROE := int64(1700)
	newROE := int64(1900)
	for _, row := range []storage.FinancialMetricV1Row{
		{
			CompanyID:      company.Company.ID,
			InstrumentID:   company.Instruments[0].InstrumentID,
			Metric:         "roe",
			FiscalYear:     "2024",
			FiscalPeriod:   string(financials.PeriodTypeAnnual),
			AsOfDate:       "2024-12-31",
			ValueBP:        &oldROE,
			FormulaVersion: "financialmetrics/v1",
			ProvenanceJSON: "{}",
			CreatedAtMS:    nowMS,
			UpdatedAtMS:    nowMS,
		},
		{
			CompanyID:      company.Company.ID,
			InstrumentID:   company.Instruments[0].InstrumentID,
			Metric:         "roe",
			FiscalYear:     "2025",
			FiscalPeriod:   string(financials.PeriodTypeAnnual),
			AsOfDate:       "2025-12-31",
			ValueBP:        &newROE,
			FormulaVersion: "financialmetrics/v1",
			ProvenanceJSON: "{}",
			CreatedAtMS:    nowMS,
			UpdatedAtMS:    nowMS,
		},
	} {
		if _, err := client.NewInsert().Model(&row).Exec(ctx); err != nil {
			return err
		}
	}
	per := int64(120000)
	marketCap := int64(1000000000)
	snapshot := storage.ValuationSnapshotV1Row{
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
	if _, err := client.NewInsert().Model(&snapshot).Exec(ctx); err != nil {
		return err
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
		return err
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
	_, err = client.NewInsert().Model(&event).Exec(ctx)
	return err
}
