//go:build integration

package valuation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
	"github.com/awuzag/mwosa/storage/financialstatement"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type valuationRepositoryContract interface {
	CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (Snapshot, error)
	ListSnapshots(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Snapshot, error)
}

func TestMongoValuationRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_valuation_contract_test",
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

	sqliteCompany := seedValuationSQLiteCompany(t, ctx, sqliteDatabase)
	mongoCompany := seedValuationMongoCompany(t, ctx, runtime)
	seedValuationSQLiteInputs(t, ctx, sqliteDatabase, sqliteCompany)
	seedValuationMongoInputs(t, ctx, runtime, mongoCompany)

	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertValuationRepositoryContract(t, sqliteRepository, sqliteCompany)
	assertValuationRepositoryContract(t, mongoRepository, mongoCompany)
	assertMongoValuationSnapshotDocumentShape(t, runtime)
}

func assertValuationRepositoryContract(t *testing.T, repository valuationRepositoryContract, company companyidentity.InspectResult) {
	t.Helper()

	ctx := context.Background()
	snapshot, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{AsOf: "latest"})
	require.NoError(t, err)
	require.Equal(t, "2026-05-16", snapshot.AsOfDate)
	require.Equal(t, int64(100000), *snapshot.PerBP)
	require.Equal(t, int64(20000), *snapshot.PbrBP)
	require.Equal(t, int64(5000), *snapshot.PsrBP)
	require.Equal(t, int64(10), *snapshot.EpsMinor)
	require.Equal(t, int64(50), *snapshot.BpsMinor)
	require.Equal(t, int64(500), *snapshot.DividendYieldBP)
	require.NotContains(t, snapshot.Uncomputable, "dividend_yield")

	updated, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{AsOf: "latest"})
	require.NoError(t, err)
	require.Equal(t, snapshot.AsOfDate, updated.AsOfDate)
	require.Equal(t, snapshot.PerBP, updated.PerBP)

	snapshots, err := repository.ListSnapshots(ctx, company, Query{AsOf: "latest"})
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, snapshot.AsOfDate, snapshots[0].AsOfDate)
	require.Equal(t, int64(500), *snapshots[0].DividendYieldBP)
}

func seedValuationSQLiteCompany(t *testing.T, ctx context.Context, database *storage.SQLDatabase) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	return seedValuationCompany(t, ctx, repository)
}

func seedValuationMongoCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	return seedValuationCompany(t, ctx, repository)
}

func seedValuationCompany(t *testing.T, ctx context.Context, repository interface {
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

func seedValuationSQLiteInputs(t *testing.T, ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult) {
	t.Helper()

	statementRepository, err := financialstatement.NewRepository(database)
	require.NoError(t, err)
	seedValuationStatements(t, ctx, statementRepository, company)
	require.NoError(t, insertPriceFixture(ctx, database, company.Instruments[0].InstrumentID))
	require.NoError(t, insertDividendFixture(ctx, database, company))
}

func seedValuationMongoInputs(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime, company companyidentity.InspectResult) {
	t.Helper()

	statementRepository, err := financialstatement.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	seedValuationStatements(t, ctx, statementRepository, company)

	_, writer, err := dailybarstorage.NewMongoRepositories(runtime.Database())
	require.NoError(t, err)
	_, err = writer.UpsertDailyBars(ctx, []dailybar.Bar{
		{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupStockPrice,
			Operation:    provider.OperationGetStockPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			Symbol:       "005930",
			Name:         "삼성전자",
			TradingDate:  "2026-05-16",
			Currency:     "KRW",
			Close:        "1000000",
			MarketCap:    "1000000",
		},
	})
	require.NoError(t, err)
	seedValuationMongoDividend(t, ctx, runtime, company)
}

func seedValuationStatements(t *testing.T, ctx context.Context, repository interface {
	UpsertStatements(context.Context, companyidentity.InspectResult, []financialsrole.Statement) (financialstatement.UpsertResult, error)
}, company companyidentity.InspectResult) {
	t.Helper()

	_, err := repository.UpsertStatements(ctx, company, []financialsrole.Statement{
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
				valuationLine("ifrs-full_Revenue", "매출액", "2000000", "IS"),
				valuationLine("ifrs-full_ProfitLoss", "당기순이익", "100000", "IS"),
				valuationLine("ifrs-full_Equity", "자본총계", "500000", "BS"),
			},
		},
	})
	require.NoError(t, err)
}

func seedValuationMongoDividend(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime, company companyidentity.InspectResult) {
	t.Helper()

	repository, err := companyfact.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	_, err = repository.UpsertFacts(ctx, company, []companyfact.FactInput{
		{
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTPeriodicReport,
			Operation:    provider.OperationOpenDARTAlotMatter,
			FactType:     companyfact.FactTypeDividend,
			FiscalYear:   "2025",
			ReportCode:   "11011",
			RceptNo:      "20260330000001",
			FactDate:     "2025-12-31",
			Key:          "thstrm:현금배당금총액:보통주",
			ValueText:    "50,000",
			ValueNumber:  "50000",
			CurrencyCode: "KRW",
			Raw:          map[string]string{"corp_code": "00126380"},
		},
	})
	require.NoError(t, err)
}

func assertMongoValuationSnapshotDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var snapshot struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Company       bson.M `bson:"company"`
		Instrument    bson.M `bson:"instrument"`
		DividendYield int64  `bson:"dividend_yield_bp"`
		Uncomputable  bson.M `bson:"uncomputable"`
		UpdatedAt     any    `bson:"updated_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("valuation_snapshots").
		FindOne(context.Background(), bson.D{{Key: "as_of_date", Value: "2026-05-16"}}).
		Decode(&snapshot))
	require.NotEmpty(t, snapshot.ID)
	require.Equal(t, "1.0.0", snapshot.SchemaVersion)
	require.GreaterOrEqual(t, snapshot.Revision, int64(2))
	require.Equal(t, "삼성전자", snapshot.Company["name"])
	require.Equal(t, "005930", snapshot.Instrument["symbol"])
	require.Equal(t, int64(500), snapshot.DividendYield)
	require.NotContains(t, snapshot.Uncomputable, "dividend_yield")
	require.IsType(t, bson.DateTime(0), snapshot.UpdatedAt)
}
