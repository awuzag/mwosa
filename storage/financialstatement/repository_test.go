package financialstatement

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	financialsrole "github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsertsAndListsStatements(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
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

	repository, err := NewRepository(database)
	require.NoError(t, err)
	result, err := repository.UpsertStatements(ctx, company, []financialsrole.Statement{
		{
			Statement:    financialsrole.StatementTypeSummary,
			Symbol:       "005930",
			Name:         "삼성전자",
			FiscalYear:   "2025",
			FiscalPeriod: "11011",
			Period:       financialsrole.PeriodTypeAnnual,
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTFinancials,
			Operation:    provider.OperationOpenDARTSinglAcntAll,
			Extensions: map[string]string{
				"reprt_code": "11011",
				"fs_div":     "CFS",
			},
			Lines: []financialsrole.LineItem{
				{
					AccountID:   "ifrs-full_Revenue",
					AccountName: "매출액",
					Value:       "1,000",
					Currency:    "KRW",
					Extensions: map[string]string{
						"sj_div":     "IS",
						"reprt_code": "11011",
						"fs_div":     "CFS",
						"rcept_no":   "20260330000001",
						"thstrm_nm":  "제 56 기",
						"ord":        "1",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{StatementsWritten: 1, LineItemsWritten: 1}, result)

	statements, err := repository.ListStatements(ctx, company, Query{
		FiscalYear: "2025",
		Period:     financialsrole.PeriodTypeAnnual,
		Statement:  financialsrole.StatementTypeIncomeStatement,
	})
	require.NoError(t, err)
	require.Len(t, statements, 1)
	require.Equal(t, financialsrole.StatementTypeIncomeStatement, statements[0].Statement)
	require.Equal(t, "005930", statements[0].Symbol)
	require.Equal(t, "revenue", statements[0].Lines[0].Extensions["canonical_account"])
	require.Equal(t, "1,000", statements[0].Lines[0].Value)
}
