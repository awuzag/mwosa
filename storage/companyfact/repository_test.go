package companyfact

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsertsAndListsFacts(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	company := seedCompany(t, ctx, database)

	repository, err := NewRepository(database)
	require.NoError(t, err)
	result, err := repository.UpsertFacts(ctx, company, []FactInput{
		{
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTPeriodicReport,
			Operation:    provider.OperationOpenDARTAlotMatter,
			FactType:     FactTypeDividend,
			FiscalYear:   "2025",
			ReportCode:   "11011",
			RceptNo:      "20260330000001",
			FactDate:     "2025-12-31",
			Key:          "thstrm:현금배당금총액:보통주",
			ValueText:    "9,809,437,000,000",
			ValueNumber:  "9809437000000",
			CurrencyCode: "KRW",
			Raw: map[string]any{
				"corp_code": "00126380",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{FactsWritten: 1}, result)

	facts, err := repository.ListFacts(ctx, company, Query{FactType: FactTypeDividend, FiscalYear: "2025"})
	require.NoError(t, err)
	require.Len(t, facts, 1)
	require.Equal(t, "005930", company.Instruments[0].Symbol)
	require.Equal(t, company.Instruments[0].InstrumentID, facts[0].InstrumentID)
	require.Equal(t, "dart_corp_code", facts[0].ProviderCompanyIdentifierType)
	require.Equal(t, "00126380", facts[0].ProviderCompanyIdentifierValue)
	require.Equal(t, "9809437000000", facts[0].ValueNumber)
	require.Equal(t, "00126380", facts[0].Raw["corp_code"])
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
