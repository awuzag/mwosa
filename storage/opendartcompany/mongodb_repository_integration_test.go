//go:build integration

package opendartcompany

import (
	"context"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
	opendartprovider "github.com/awuzag/mwosa/providers/opendart"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoOpenDARTCompanyRepositoryStoresCompaniesInCanonicalCollection(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_opendart_company_contract_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	repository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	result, err := repository.UpsertCompanies(ctx, []opendartprovider.Company{
		{
			CorpCode:    "00126380",
			CorpName:    "삼성전자",
			CorpEngName: "SAMSUNG ELECTRONICS CO,.LTD",
			StockCode:   "005930",
			ModifyDate:  "20250101",
		},
		{
			CorpCode:    "00164779",
			CorpName:    "현대자동차",
			CorpEngName: "HYUNDAI MOTOR COMPANY",
			StockCode:   "005380",
			ModifyDate:  "20250102",
		},
		{
			CorpCode:    "00999999",
			CorpName:    "비상장테스트",
			CorpEngName: "UNLISTED TEST",
			ModifyDate:  "20250103",
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{RowsAffected: 3, TotalCount: 3, ListedCount: 2}, result)

	updated, err := repository.UpsertCompanies(ctx, []opendartprovider.Company{
		{
			CorpCode:    "00126380",
			CorpName:    "삼성전자",
			CorpEngName: "Samsung Electronics",
			StockCode:   "005930",
			ModifyDate:  "20260101",
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{RowsAffected: 1, TotalCount: 1, ListedCount: 1}, updated)

	byStockCode, err := repository.Search(ctx, "005930", false, 20)
	require.NoError(t, err)
	require.Len(t, byStockCode, 1)
	require.Equal(t, "00126380", byStockCode[0].CorpCode)
	require.Equal(t, "Samsung Electronics", byStockCode[0].CorpEngName)
	require.Equal(t, "20260101", byStockCode[0].ModifyDate)

	byName, err := repository.Search(ctx, "자동차", false, 20)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	require.Equal(t, "00164779", byName[0].CorpCode)

	listedOnly, err := repository.Search(ctx, "테스트", true, 20)
	require.NoError(t, err)
	require.Empty(t, listedOnly)

	var company struct {
		ID            string    `bson:"_id"`
		SchemaVersion string    `bson:"schema_version"`
		Revision      int64     `bson:"revision"`
		Identifiers   []bson.M  `bson:"identifiers"`
		Instruments   []bson.M  `bson:"instruments"`
		CreatedAt     time.Time `bson:"created_at"`
		UpdatedAt     time.Time `bson:"updated_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("companies").
		FindOne(ctx, bson.D{{Key: "identifiers.identifier_value", Value: "00126380"}}).
		Decode(&company))
	require.Equal(t, "1.0.0", company.SchemaVersion)
	require.GreaterOrEqual(t, company.Revision, int64(2))
	require.NotEmpty(t, company.CreatedAt)
	require.NotEmpty(t, company.UpdatedAt)
	require.Len(t, company.Identifiers, 2)
	require.Len(t, company.Instruments, 1)
	require.Equal(t, "005930", company.Instruments[0]["symbol"])
}
