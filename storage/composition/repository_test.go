package composition

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	compositionrole "github.com/ev3rlit/mwosa/providers/core/composition"
	compositionservice "github.com/ev3rlit/mwosa/service/composition"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/stretchr/testify/require"
)

func TestRepositoryStoresAndReadsCompositionAggregate(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	repository, err := NewRepository(database)
	require.NoError(t, err)

	aggregate := sampleComposition()

	writeResult, err := repository.UpsertComposition(context.Background(), aggregate)
	require.NoError(t, err)
	require.Equal(t, 1, writeResult.CompositionsStored)
	require.Equal(t, 2, writeResult.MembersStored)

	got, err := repository.GetComposition(context.Background(), compositionservice.Query{
		ProviderID:   provider.ProviderKIS,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		AsOfDate:     "2026-05-17",
	})
	require.NoError(t, err)
	require.Equal(t, provider.ProviderKIS, got.Source.Provider)
	require.Equal(t, provider.GroupKISQuote, got.Source.Group)
	require.Equal(t, provider.OperationKISETFComponentStockPrice, got.Source.Operation)
	require.Equal(t, "069500", got.Subject.Symbol)
	require.Equal(t, "KODEX 200", got.Subject.Name)
	require.Equal(t, "2026-05-17", got.AsOfDate)
	require.Equal(t, int64(1779010800000), got.ObservedAtMS)
	require.Len(t, got.Members, 2)
	require.Equal(t, "005930", got.Members[0].Instrument.Symbol)
	require.Equal(t, "삼성전자", got.Members[0].Instrument.Name)
	require.Equal(t, "28.15", got.Members[0].Weight.Value)
	require.Equal(t, "25893", got.Members[0].Quantity.Value)
	require.Equal(t, "1942000000", got.Members[0].Valuation.Value)
	require.Equal(t, "000660", got.Members[1].Instrument.Symbol)

	requireTableCount(t, database, "composition_observation_v1", 1)
	requireTableCount(t, database, "composition_member_v1", 2)
	requireNoRawColumn(t, database, "composition_observation_v1")
	requireNoRawColumn(t, database, "composition_member_v1")
}

func TestRepositoryUpsertReplacesMembersAndAllowsEmptyComposition(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	repository, err := NewRepository(database)
	require.NoError(t, err)

	aggregate := sampleComposition()
	_, err = repository.UpsertComposition(context.Background(), aggregate)
	require.NoError(t, err)
	aggregate.Members = nil
	_, err = repository.UpsertComposition(context.Background(), aggregate)
	require.NoError(t, err)

	got, err := repository.GetComposition(context.Background(), compositionservice.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		AsOfDate:     "2026-05-17",
	})
	require.NoError(t, err)
	require.Empty(t, got.Members)
	requireTableCount(t, database, "composition_observation_v1", 1)
	requireTableCount(t, database, "composition_member_v1", 0)
}

func TestRepositoryValidatesCompositionKey(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	repository, err := NewRepository(database)
	require.NoError(t, err)

	_, err = repository.UpsertComposition(context.Background(), compositionrole.Composition{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing sqlite key")
}

func sampleComposition() compositionrole.Composition {
	return compositionrole.Composition{
		Source: compositionrole.SourceRef{
			Provider:  provider.ProviderKIS,
			Group:     provider.GroupKISQuote,
			Operation: provider.OperationKISETFComponentStockPrice,
		},
		Subject: compositionrole.InstrumentRef{
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			ISIN:         "KR7069500007",
			Name:         "KODEX 200",
		},
		AsOfDate:     "2026-05-17",
		ObservedAtMS: 1779010800000,
		Members: []compositionrole.CompositionMember{
			{
				Instrument: compositionrole.InstrumentRef{
					Market:       provider.MarketKRX,
					SecurityType: provider.SecurityTypeStock,
					Symbol:       "005930",
					Name:         "삼성전자",
				},
				Weight:    compositionrole.DecimalValue{Value: "28.15"},
				Quantity:  compositionrole.DecimalValue{Value: "25893"},
				Valuation: compositionrole.MoneyValue{Currency: "KRW", Value: "1942000000"},
			},
			{
				Instrument: compositionrole.InstrumentRef{
					Market:       provider.MarketKRX,
					SecurityType: provider.SecurityTypeStock,
					Symbol:       "000660",
					Name:         "SK하이닉스",
				},
				Weight: compositionrole.DecimalValue{Value: "5.5"},
			},
		},
	}
}

func requireTableCount(t *testing.T, database *storage.Database, table string, want int) {
	t.Helper()
	client, err := database.Client(context.Background())
	require.NoError(t, err)
	var got int
	require.NoError(t, client.QueryRowContext(context.Background(), "SELECT count(*) FROM "+table).Scan(&got))
	require.Equal(t, want, got)
}

func requireNoRawColumn(t *testing.T, database *storage.Database, table string) {
	t.Helper()
	client, err := database.Client(context.Background())
	require.NoError(t, err)
	rows, err := client.QueryContext(context.Background(), `PRAGMA table_info('`+table+`')`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var pk int
		require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk))
		require.NotContains(t, name, "raw")
		require.NotContains(t, name, "kis")
	}
	require.NoError(t, rows.Err())
}
