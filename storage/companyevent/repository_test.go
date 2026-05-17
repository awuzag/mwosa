package companyevent

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsertsAndListsEvents(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	company := seedCompany(t, ctx, database)

	repository, err := NewRepository(database)
	require.NoError(t, err)
	amount := int64(1000000)
	result, err := repository.UpsertEvents(ctx, company, []EventInput{
		{
			EventType:   "convertible_bond_issuance",
			EventDate:   "2025-03-31",
			RceptDt:     "2025-04-01",
			RceptNo:     "20250401000001",
			Provider:    provider.ProviderOpenDART,
			Group:       provider.GroupOpenDARTDisclosure,
			Operation:   "cvbdIsDecsn",
			Title:       "전환사채권 발행결정",
			AmountMinor: &amount,
			ValueText:   "1000000",
			Raw: map[string]any{
				"rcept_no": "20250401000001",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{EventsWritten: 1}, result)

	events, err := repository.ListEvents(ctx, company, Query{Provider: provider.ProviderOpenDART, From: "2025-01-01"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "convertible_bond_issuance", events[0].EventType)
	require.Equal(t, company.Instruments[0].InstrumentID, events[0].InstrumentID)
	require.Equal(t, int64(1000000), *events[0].AmountMinor)
	require.Equal(t, "20250401000001", events[0].Raw["rcept_no"])
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
