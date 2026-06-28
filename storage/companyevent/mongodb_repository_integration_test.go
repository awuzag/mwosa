//go:build integration

package companyevent

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

type eventRepositoryContract interface {
	UpsertEvents(ctx context.Context, company companyidentity.InspectResult, events []EventInput) (UpsertResult, error)
	ListEvents(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Event, error)
}

func TestMongoCompanyEventRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_company_event_contract_test",
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

	sqliteCompany := seedSQLiteEventCompany(t, ctx, sqliteDatabase)
	mongoCompany := seedMongoEventCompany(t, ctx, runtime)

	sqliteRepository, err := NewRepository(sqliteDatabase)
	require.NoError(t, err)
	mongoRepository, err := NewMongoRepository(runtime.Database())
	require.NoError(t, err)

	assertCompanyEventRepositoryContract(t, sqliteRepository, sqliteCompany)
	assertCompanyEventRepositoryContract(t, mongoRepository, mongoCompany)
	assertMongoCompanyEventDocumentShape(t, runtime)
}

func assertCompanyEventRepositoryContract(t *testing.T, repository eventRepositoryContract, company companyidentity.InspectResult) {
	t.Helper()

	ctx := context.Background()
	amount := int64(1000000)
	result, err := repository.UpsertEvents(ctx, company, []EventInput{
		{
			EventType:   "convertible_bond_issuance",
			EventDate:   "2025-03-31",
			RceptDt:     "2025-04-01",
			RceptNo:     "20250401000001",
			Provider:    provider.ProviderOpenDART,
			Group:       provider.GroupOpenDARTMaterialEvents,
			Operation:   provider.OperationOpenDARTCvbdIsDecsn,
			Title:       "전환사채권 발행결정",
			AmountMinor: &amount,
			ValueText:   "1000000",
			Raw: map[string]any{
				"rcept_no": "20250401000001",
				"version":  "first",
			},
		},
		{
			EventType: "company_merger",
			RceptDt:   "2025-05-03",
			RceptNo:   "20250503000001",
			Provider:  provider.ProviderOpenDART,
			Group:     provider.GroupOpenDARTMaterialEvents,
			Operation: provider.OperationOpenDARTCmpMgDecsn,
			Title:     "회사합병 결정",
			ValueText: "합병",
			Raw: map[string]any{
				"rcept_no": "20250503000001",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{EventsWritten: 2}, result)

	updatedAmount := int64(2000000)
	updated, err := repository.UpsertEvents(ctx, company, []EventInput{
		{
			EventType:   "convertible_bond_issuance",
			EventDate:   "2025-03-31",
			RceptDt:     "2025-04-01",
			RceptNo:     "20250401000001",
			Provider:    provider.ProviderOpenDART,
			Group:       provider.GroupOpenDARTMaterialEvents,
			Operation:   provider.OperationOpenDARTCvbdIsDecsn,
			Title:       "전환사채권 발행결정",
			AmountMinor: &updatedAmount,
			ValueText:   "2000000",
			Raw: map[string]any{
				"rcept_no": "20250401000001",
				"version":  "updated",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, UpsertResult{EventsWritten: 1}, updated)

	events, err := repository.ListEvents(ctx, company, Query{
		Provider:  provider.ProviderOpenDART,
		EventType: "convertible_bond_issuance",
		From:      "2025-01-01",
		To:        "2025-04-30",
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, company.Instruments[0].InstrumentID, events[0].InstrumentID)
	require.Equal(t, "convertible_bond_issuance", events[0].EventType)
	require.Equal(t, provider.GroupOpenDARTMaterialEvents, events[0].Group)
	require.Equal(t, provider.OperationOpenDARTCvbdIsDecsn, events[0].Operation)
	require.Equal(t, int64(2000000), *events[0].AmountMinor)
	require.Equal(t, "updated", events[0].Raw["version"])

	limited, err := repository.ListEvents(ctx, company, Query{Provider: provider.ProviderOpenDART, Limit: 1})
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Equal(t, "company_merger", limited[0].EventType)

	fallbackDate, err := repository.ListEvents(ctx, company, Query{From: "2025-05-01", To: "2025-05-31"})
	require.NoError(t, err)
	require.Len(t, fallbackDate, 1)
	require.Equal(t, "2025-05-03", fallbackDate[0].RceptDt)
	require.Empty(t, fallbackDate[0].EventDate)
}

func seedSQLiteEventCompany(t *testing.T, ctx context.Context, database *storage.Database) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	return seedEventCompany(t, ctx, repository)
}

func seedMongoEventCompany(t *testing.T, ctx context.Context, runtime *storagemongodb.Runtime) companyidentity.InspectResult {
	t.Helper()

	repository, err := companyidentity.NewMongoRepository(runtime.Database())
	require.NoError(t, err)
	return seedEventCompany(t, ctx, repository)
}

func seedEventCompany(t *testing.T, ctx context.Context, repository interface {
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

func assertMongoCompanyEventDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var event struct {
		ID            string `bson:"_id"`
		SchemaVersion string `bson:"schema_version"`
		Revision      int64  `bson:"revision"`
		Company       bson.M `bson:"company"`
		Instrument    bson.M `bson:"instrument"`
		EffectiveDate string `bson:"effective_date"`
		Raw           bson.M `bson:"raw"`
		UpdatedAt     any    `bson:"updated_at"`
	}
	require.NoError(t, runtime.Database().
		Collection("company_events").
		FindOne(context.Background(), bson.D{{Key: "event_type", Value: "convertible_bond_issuance"}}).
		Decode(&event))
	require.NotEmpty(t, event.ID)
	require.Equal(t, "1.0.0", event.SchemaVersion)
	require.GreaterOrEqual(t, event.Revision, int64(2))
	require.Equal(t, "삼성전자", event.Company["name"])
	require.Equal(t, "005930", event.Instrument["symbol"])
	require.Equal(t, "2025-03-31", event.EffectiveDate)
	require.Equal(t, "updated", event.Raw["version"])
	require.IsType(t, bson.DateTime(0), event.UpdatedAt)
}
