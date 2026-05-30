package companyidentity

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsertsCompanyIdentifiersAndInstrumentLink(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	repository, err := NewRepository(database)
	require.NoError(t, err)

	result, err := repository.UpsertCompanies(ctx, []CompanyInput{
		{
			Name:        "삼성전자",
			LegalName:   "삼성전자",
			EnglishName: "SAMSUNG ELECTRONICS CO,.LTD",
			CountryCode: "KR",
			Identifiers: []IdentifierInput{
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  IdentifierTypeDARTCorpCode,
					IdentifierValue: "00126380",
					Primary:         true,
					Confidence:      1,
					SourceUpdatedAt: "20240101",
				},
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  IdentifierTypeKRXStockCode,
					IdentifierValue: "005930",
					Confidence:      1,
					SourceUpdatedAt: "20240101",
				},
			},
			InstrumentRef: InstrumentRef{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				Symbol:       "005930",
				Name:         "삼성전자",
				RelationType: RelationTypeIssuer,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{CompaniesWritten: 1, IdentifiersWritten: 2, InstrumentsLinked: 1}, result)

	inspect, err := repository.Inspect(ctx, "005930")
	require.NoError(t, err)
	require.Equal(t, "삼성전자", inspect.Company.Name)
	require.Len(t, inspect.Identifiers, 2)
	require.Len(t, inspect.Instruments, 1)
	require.Equal(t, "005930", inspect.Instruments[0].Symbol)
	require.Equal(t, RelationTypeIssuer, inspect.Instruments[0].RelationType)
}
