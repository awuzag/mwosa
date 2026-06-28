//go:build integration

package financialstatement

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type repositoryContract interface {
	UpsertStatements(ctx context.Context, company companyidentity.InspectResult, statements []financialsrole.Statement) (UpsertResult, error)
	ListStatements(ctx context.Context, company companyidentity.InspectResult, query Query) ([]financialsrole.Statement, error)
}

func TestMongoFinancialStatementRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_financial_statement_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	sqliteDatabase := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, sqliteDatabase.Close())
	})

	sqliteCompany := seedSQLiteCompany(t, ctx, sqliteDatabase)
	mongoCompany := seedMongoCompany(t, ctx, runtime)

	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertFinancialStatementRepositoryContract(t, sqliteRepository, sqliteCompany)
	assertFinancialStatementRepositoryContract(t, mongoRepository, mongoCompany)
	assertMongoFinancialStatementDocumentShape(t, runtime)
}

func assertFinancialStatementRepositoryContract(t *testing.T, repository repositoryContract, company companyidentity.InspectResult) {
	t.Helper()

	ctx := context.Background()
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
			ReportedAt:   "2026-03-30",
			Extensions: map[string]string{
				"reprt_code":         "11011",
				"fs_div":             "CFS",
				"source_payload_ref": "provider_raw_snapshots:opendart:2025",
			},
			Lines: []financialsrole.LineItem{
				{
					AccountID:   "ifrs-full_Revenue",
					AccountName: "매출액",
					Value:       "1,000",
					Currency:    "KRW",
					Unit:        "KRW",
					Extensions: map[string]string{
						"sj_div":     "IS",
						"reprt_code": "11011",
						"fs_div":     "CFS",
						"rcept_no":   "20260330000001",
						"thstrm_nm":  "제 56 기",
						"ord":        "1",
					},
				},
				{
					AccountID:   "ifrs-full_ProfitLoss",
					AccountName: "당기순이익",
					Value:       "200",
					Currency:    "KRW",
					Unit:        "KRW",
					Extensions: map[string]string{
						"sj_div":     "IS",
						"reprt_code": "11011",
						"fs_div":     "CFS",
						"rcept_no":   "20260330000001",
						"thstrm_nm":  "제 56 기",
						"ord":        "2",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{StatementsWritten: 1, LineItemsWritten: 2}, result)

	statements, err := repository.ListStatements(ctx, company, Query{
		FiscalYear: "2025",
		Period:     financialsrole.PeriodTypeAnnual,
		Statement:  financialsrole.StatementTypeIncomeStatement,
	})
	require.NoError(t, err)
	require.Len(t, statements, 1)
	require.Equal(t, financialsrole.StatementTypeIncomeStatement, statements[0].Statement)
	require.Equal(t, "005930", statements[0].Symbol)
	require.Equal(t, "2025", statements[0].FiscalYear)
	require.Equal(t, "11011", statements[0].FiscalPeriod)
	require.Equal(t, provider.ProviderOpenDART, statements[0].Provider)
	require.Equal(t, provider.GroupOpenDARTFinancials, statements[0].Group)
	require.Equal(t, provider.OperationOpenDARTSinglAcntAll, statements[0].Operation)
	require.Len(t, statements[0].Lines, 2)
	require.Equal(t, "revenue", statements[0].Lines[0].Extensions["canonical_account"])
	require.Equal(t, "net_income", statements[0].Lines[1].Extensions["canonical_account"])

	limited, err := repository.ListStatements(ctx, company, Query{
		FiscalYear: "2025",
		Period:     financialsrole.PeriodTypeAnnual,
		Statement:  financialsrole.StatementTypeIncomeStatement,
		Limit:      1,
	})
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Len(t, limited[0].Lines, 1)
	require.Equal(t, "매출액", limited[0].Lines[0].AccountName)

	updated, err := repository.UpsertStatements(ctx, company, []financialsrole.Statement{
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
					Value:       "1,500",
					Currency:    "KRW",
					Unit:        "KRW",
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
	require.Equal(t, UpsertResult{StatementsWritten: 1, LineItemsWritten: 1}, updated)

	afterUpdate, err := repository.ListStatements(ctx, company, Query{
		FiscalYear: "2025",
		Period:     financialsrole.PeriodTypeAnnual,
		Statement:  financialsrole.StatementTypeIncomeStatement,
	})
	require.NoError(t, err)
	require.Equal(t, "1,500", afterUpdate[0].Lines[0].Value)
}

func seedSQLiteCompany(t *testing.T, ctx context.Context, database *storage.Database) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	return seedCompany(t, ctx, repository)
}

func seedMongoCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	return seedCompany(t, ctx, repository)
}

func seedCompany(t *testing.T, ctx context.Context, repository interface {
	UpsertCompanies(context.Context, []companyidentity.CompanyInput) (companyidentity.UpsertResult, error)
	Inspect(context.Context, string) (companyidentity.InspectResult, error)
}) companyidentity.InspectResult {
	t.Helper()

	_, err := repository.UpsertCompanies(ctx, []companyidentity.CompanyInput{
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

func assertMongoFinancialStatementDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var statement struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Company       bson.M   `bson:"company"`
		Instrument    bson.M   `bson:"instrument"`
		LineItems     []bson.M `bson:"line_items"`
	}
	require.NoError(t, runtime.Database().
		Collection("financial_statements").
		FindOne(context.Background(), bson.D{{Key: "provider_company_identifier_value", Value: "00126380"}}).
		Decode(&statement))
	require.Equal(t, "1.0.0", statement.SchemaVersion)
	require.GreaterOrEqual(t, statement.Revision, int64(2))
	require.Equal(t, "삼성전자", statement.Company["name"])
	require.Equal(t, "005930", statement.Instrument["symbol"])
	require.Len(t, statement.LineItems, 2)
	require.Equal(t, "매출액", statement.LineItems[0]["account_name"])
	require.IsType(t, int64(0), statement.LineItems[0]["amount_minor"])
}
