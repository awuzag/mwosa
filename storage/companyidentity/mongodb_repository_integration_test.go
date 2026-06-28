//go:build integration

package companyidentity

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type repositoryContract interface {
	UpsertCompanies(ctx context.Context, companies []CompanyInput) (UpsertResult, error)
	Inspect(ctx context.Context, query string) (InspectResult, error)
}

func TestMongoCompanyIdentityRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_company_identity_contract_test",
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
	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertCompanyIdentityRepositoryContract(t, sqliteRepository)
	assertCompanyIdentityRepositoryContract(t, mongoRepository)
	assertMongoCompanyIdentityDocumentShape(t, runtime)
}

func assertCompanyIdentityRepositoryContract(t *testing.T, repository repositoryContract) {
	t.Helper()

	ctx := context.Background()
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

	inspectBySymbol, err := repository.Inspect(ctx, "005930")
	require.NoError(t, err)
	require.NotZero(t, inspectBySymbol.Company.ID)
	require.Equal(t, "삼성전자", inspectBySymbol.Company.Name)
	require.Equal(t, "SAMSUNG ELECTRONICS CO,.LTD", inspectBySymbol.Company.EnglishName)
	require.Len(t, inspectBySymbol.Identifiers, 2)
	require.Len(t, inspectBySymbol.Instruments, 1)
	require.NotZero(t, inspectBySymbol.Instruments[0].InstrumentID)
	require.Equal(t, "005930", inspectBySymbol.Instruments[0].Symbol)
	require.Equal(t, RelationTypeIssuer, inspectBySymbol.Instruments[0].RelationType)

	updated, err := repository.UpsertCompanies(ctx, []CompanyInput{
		{
			Name:        "삼성전자",
			LegalName:   "삼성전자주식회사",
			EnglishName: "Samsung Electronics",
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
					SourceUpdatedAt: "20250101",
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{CompaniesWritten: 1, IdentifiersWritten: 1}, updated)

	inspectByCorpCode, err := repository.Inspect(ctx, "00126380")
	require.NoError(t, err)
	require.Equal(t, inspectBySymbol.Company.ID, inspectByCorpCode.Company.ID)
	require.Equal(t, "삼성전자주식회사", inspectByCorpCode.Company.LegalName)
	require.Len(t, inspectByCorpCode.Identifiers, 2)

	inspectByName, err := repository.Inspect(ctx, "삼성전자")
	require.NoError(t, err)
	require.Equal(t, inspectBySymbol.Company.ID, inspectByName.Company.ID)
}

func assertMongoCompanyIdentityDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var company struct {
		ID            string   `bson:"_id"`
		SchemaVersion string   `bson:"schema_version"`
		Revision      int64    `bson:"revision"`
		Identifiers   []bson.M `bson:"identifiers"`
		Instruments   []bson.M `bson:"instruments"`
	}
	require.NoError(t, runtime.Database().
		Collection("companies").
		FindOne(context.Background(), bson.D{{Key: "identifiers.identifier_value", Value: "00126380"}}).
		Decode(&company))
	require.Equal(t, "1.0.0", company.SchemaVersion)
	require.GreaterOrEqual(t, company.Revision, int64(2))
	require.Len(t, company.Identifiers, 2)
	require.Len(t, company.Instruments, 1)
	require.Equal(t, "005930", company.Instruments[0]["symbol"])
}
