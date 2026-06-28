//go:build integration

package financialmetric

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialstatement"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type metricRepositoryContract interface {
	CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (UpsertResult, error)
	ListMetrics(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Metric, error)
}

func TestMongoFinancialMetricRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_financial_metric_contract_test",
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

	sqliteCompany := seedMetricSQLiteCompany(t, ctx, sqliteDatabase)
	mongoCompany := seedMetricMongoCompany(t, ctx, runtime)
	seedMetricStatements(t, ctx, sqliteDatabase, nil, sqliteCompany)
	seedMetricStatements(t, ctx, nil, runtime, mongoCompany)

	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertFinancialMetricRepositoryContract(t, sqliteRepository, sqliteCompany)
	assertFinancialMetricRepositoryContract(t, mongoRepository, mongoCompany)
	assertMongoFinancialMetricDocumentShape(t, runtime)
}

func assertFinancialMetricRepositoryContract(t *testing.T, repository metricRepositoryContract, company companyidentity.InspectResult) {
	t.Helper()

	ctx := context.Background()
	result, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{WindowYears: 2, Period: financialsrole.PeriodTypeAnnual})
	require.NoError(t, err)
	require.Equal(t, 20, result.MetricsCalculated)
	require.Equal(t, 20, result.MetricsWritten)
	require.Equal(t, 7, result.Uncomputable)

	updated, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{WindowYears: 2, Period: financialsrole.PeriodTypeAnnual})
	require.NoError(t, err)
	require.Equal(t, result, updated)

	metrics, err := repository.ListMetrics(ctx, company, Query{WindowYears: 2, Period: financialsrole.PeriodTypeAnnual})
	require.NoError(t, err)
	require.Len(t, metrics, 20)
	require.Contains(t, metricDecimals(metrics), "revenue_growth_yoy=0.20000000")
	require.Contains(t, metricDecimals(metrics), "operating_margin=0.15000000")
	require.Contains(t, metricDecimals(metrics), "roe=0.25000000")
}

func seedMetricSQLiteCompany(t *testing.T, ctx context.Context, database *storage.Database) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	return seedMetricCompany(t, ctx, repository)
}

func seedMetricMongoCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	return seedMetricCompany(t, ctx, repository)
}

func seedMetricCompany(t *testing.T, ctx context.Context, repository interface {
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

func seedMetricStatements(t *testing.T, ctx context.Context, sqliteDatabase *storage.Database, runtime *storagemongodb.Runtime, company companyidentity.InspectResult) {
	t.Helper()

	var repository interface {
		UpsertStatements(context.Context, companyidentity.InspectResult, []financialsrole.Statement) (financialstatement.UpsertResult, error)
	}
	if runtime != nil {
		mongoRepository, err := financialstatement.NewMongoRepository(runtime.Database())
		require.NoError(t, err)
		repository = mongoRepository
	} else {
		sqliteRepository, err := financialstatement.NewRepository(sqliteDatabase)
		require.NoError(t, err)
		repository = sqliteRepository
	}

	_, err := repository.UpsertStatements(ctx, company, []financialsrole.Statement{
		statement("2024", []financialsrole.LineItem{
			line("ifrs-full_Revenue", "매출액", "1,000", "IS"),
			line("ifrs-full_OperatingIncomeLoss", "영업이익", "100", "IS"),
			line("ifrs-full_ProfitLoss", "당기순이익", "50", "IS"),
			line("ifrs-full_Assets", "자산총계", "500", "BS"),
			line("ifrs-full_Liabilities", "부채총계", "200", "BS"),
			line("ifrs-full_Equity", "자본총계", "300", "BS"),
		}),
		statement("2025", []financialsrole.LineItem{
			line("ifrs-full_Revenue", "매출액", "1,200", "IS"),
			line("ifrs-full_OperatingIncomeLoss", "영업이익", "180", "IS"),
			line("ifrs-full_ProfitLoss", "당기순이익", "90", "IS"),
			line("ifrs-full_Assets", "자산총계", "600", "BS"),
			line("ifrs-full_Liabilities", "부채총계", "240", "BS"),
			line("ifrs-full_Equity", "자본총계", "360", "BS"),
		}),
	})
	require.NoError(t, err)
}

func assertMongoFinancialMetricDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var metric struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Company       bson.M `bson:"company"`
		Instrument    bson.M `bson:"instrument"`
		Provenance    bson.M `bson:"provenance"`
		UpdatedAt     any    `bson:"updated_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("financial_metrics").
		FindOne(context.Background(), bson.D{{Key: "metric", Value: "operating_margin"}, {Key: "fiscal_year", Value: "2025"}}).
		Decode(&metric))
	require.NotEmpty(t, metric.ID)
	require.Equal(t, "1.0.0", metric.SchemaVersion)
	require.GreaterOrEqual(t, metric.Revision, int64(2))
	require.Equal(t, "삼성전자", metric.Company["name"])
	require.Equal(t, "005930", metric.Instrument["symbol"])
	require.Equal(t, "financial-metrics/v1", metric.Provenance["formula_version"])
	require.IsType(t, bson.DateTime(0), metric.UpdatedAt)
}
