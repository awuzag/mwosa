//go:build integration

package companyfact

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type factRepositoryContract interface {
	UpsertFacts(ctx context.Context, company companyidentity.InspectResult, facts []FactInput) (UpsertResult, error)
	ListFacts(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Fact, error)
}

func TestMongoCompanyFactRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_company_fact_contract_test",
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

	sqliteCompany := seedSQLiteFactCompany(t, ctx, sqliteDatabase)
	mongoCompany := seedMongoFactCompany(t, ctx, runtime)

	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertCompanyFactRepositoryContract(t, sqliteRepository, sqliteCompany)
	assertCompanyFactRepositoryContract(t, mongoRepository, mongoCompany)
	assertMongoCompanyFactDocumentShape(t, runtime)
}

func assertCompanyFactRepositoryContract(t *testing.T, repository factRepositoryContract, company companyidentity.InspectResult) {
	t.Helper()

	ctx := context.Background()
	result, err := repository.UpsertFacts(ctx, company, []FactInput{
		{
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTPeriodicReport,
			Operation:    provider.OperationOpenDARTAlotMatter,
			FactType:     FactTypeDividend,
			FiscalYear:   "2024",
			ReportCode:   "11011",
			RceptNo:      "20250330000001",
			FactDate:     "2024-12-31",
			Key:          "thstrm:현금배당금총액:보통주",
			ValueText:    "48,000",
			ValueNumber:  "48000",
			CurrencyCode: "KRW",
			Raw: map[string]any{
				"corp_code": "00126380",
				"year":      "2024",
			},
		},
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
			ValueText:    "50,000",
			ValueNumber:  "50000",
			CurrencyCode: "KRW",
			Raw: map[string]any{
				"corp_code": "00126380",
				"year":      "2025",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{FactsWritten: 2}, result)

	updated, err := repository.UpsertFacts(ctx, company, []FactInput{
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
			ValueText:    "51,000",
			ValueNumber:  "51000",
			CurrencyCode: "KRW",
			Raw: map[string]any{
				"corp_code": "00126380",
				"year":      "2025",
				"revision":  "updated",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{FactsWritten: 1}, updated)

	facts, err := repository.ListFacts(ctx, company, Query{FactType: FactTypeDividend, FiscalYear: "2025"})
	require.NoError(t, err)
	require.Len(t, facts, 1)
	require.Equal(t, company.Instruments[0].InstrumentID, facts[0].InstrumentID)
	require.Equal(t, "dart_corp_code", facts[0].ProviderCompanyIdentifierType)
	require.Equal(t, "00126380", facts[0].ProviderCompanyIdentifierValue)
	require.Equal(t, "51000", facts[0].ValueNumber)
	require.Equal(t, "updated", facts[0].Raw["revision"])

	windowed, err := repository.ListFacts(ctx, company, Query{FactType: FactTypeDividend, WindowYears: 1})
	require.NoError(t, err)
	require.Len(t, windowed, 1)
	require.Equal(t, "2025", windowed[0].FiscalYear)

	limited, err := repository.ListFacts(ctx, company, Query{FactType: FactTypeDividend, Limit: 1})
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Equal(t, "2025", limited[0].FiscalYear)
}

func seedSQLiteFactCompany(t *testing.T, ctx context.Context, database *storage.SQLDatabase) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	return seedFactCompany(t, ctx, repository)
}

func seedMongoFactCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	return seedFactCompany(t, ctx, repository)
}

func seedFactCompany(t *testing.T, ctx context.Context, repository interface {
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

func assertMongoCompanyFactDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var fact struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Company       bson.M `bson:"company"`
		Instrument    bson.M `bson:"instrument"`
		Raw           bson.M `bson:"raw"`
		UpdatedAt     any    `bson:"updated_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("company_facts").
		FindOne(context.Background(), bson.D{{Key: "fact_type", Value: FactTypeDividend}, {Key: "fiscal_year", Value: "2025"}}).
		Decode(&fact))
	require.NotEmpty(t, fact.ID)
	require.Equal(t, "1.0.0", fact.SchemaVersion)
	require.GreaterOrEqual(t, fact.Revision, int64(2))
	require.Equal(t, "삼성전자", fact.Company["name"])
	require.Equal(t, "005930", fact.Instrument["symbol"])
	require.Equal(t, "updated", fact.Raw["revision"])
	require.IsType(t, bson.DateTime(0), fact.UpdatedAt)
}
