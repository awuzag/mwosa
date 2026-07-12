//go:build integration

package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/awuzag/mwosa/testing/fixtures/krxmonthly"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAggregateCLIRunsKRXMonthlyArchive(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	databaseName := "mwosa_aggregate_krx_monthly_fixture_test"
	databaseURL, err := storagemongodb.URIWithDatabase(server.URI, databaseName)
	require.NoError(t, err)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: databaseName,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runtime.Close(context.Background()))
	})
	require.NoError(t, runtime.Init(ctx))

	archivePath := filepath.Join("..", "testdata", "aggregate", "krx", "krx-stock-daily-2026-06.zip")
	first, err := krxmonthly.LoadArchive(ctx, runtime.Database(), archivePath)
	require.NoError(t, err)
	require.Equal(t, 42, first.Manifest.SnapshotCount)
	require.Equal(t, 58144, first.Manifest.TotalRows)
	require.Len(t, first.Manifest.TradingDates, 21)
	require.Equal(t, "726ffedab13468cf7c018c239e93de827d3c7d0d321a9664755821fab1b77165", first.Manifest.DatasetSHA256)
	require.Equal(t, int64(42), first.Bulk.UpsertedCount)
	require.Equal(t, first.Manifest.TotalRows, first.Bulk.RowCount)

	second, err := krxmonthly.LoadArchive(ctx, runtime.Database(), archivePath)
	require.NoError(t, err)
	require.Equal(t, int64(42), second.Bulk.MatchedCount)
	require.Zero(t, second.Bulk.UpsertedCount)

	snapshotCount, err := runtime.Database().Collection("provider_raw_snapshots").CountDocuments(ctx, bson.D{})
	require.NoError(t, err)
	require.Equal(t, int64(42), snapshotCount)

	configPath := filepath.Join(t.TempDir(), "config.json")
	specPath := filepath.Join("..", "testdata", "aggregate", "krx", "krx-monthly-replay.aggregate.yaml")
	validateOut := runAggregateCLI(t, configPath, databaseURL, "validate", "aggregate", specPath, "-o", "json")
	require.Contains(t, validateOut, `"valid": true`)

	updateOut := runAggregateCLI(t, configPath, databaseURL, "update", "aggregate", "krx-monthly-fixture-replay", "--file", specPath, "-o", "json")
	require.Contains(t, updateOut, `"name": "krx-monthly-fixture-replay"`)

	runOut := runAggregateCLI(t, configPath, databaseURL, "run", "aggregate", "krx-monthly-fixture-replay", "--alias", "krx-monthly-fixture", "-o", "json")
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(runOut), &rows))

	stagesOut := runAggregateCLI(t, configPath, databaseURL, "inspect", "aggregate-run", "krx-monthly-fixture", "--view", "stages", "-o", "json")
	var stages []struct {
		Name       string `json:"name"`
		Rows       int    `json:"rows"`
		Status     string `json:"status"`
		Collection string `json:"collection"`
	}
	require.NoError(t, json.Unmarshal([]byte(stagesOut), &stages))
	t.Logf("aggregate stages: %+v", stages)
	requireStageRows(t, stages, "snapshots", 42)
	requireStageRows(t, stages, "normalized", 58144)
	requireStageRows(t, stages, "monthly", 30)
	requireStageRows(t, stages, "derived", 30)
	require.Len(t, rows, 30)
	markets := map[string]bool{}
	symbols := map[string]bool{}
	require.IsType(t, float64(0), rows[0]["total_traded_amount"])
	previousTradedAmount := rows[0]["total_traded_amount"].(float64)
	for _, row := range rows {
		market, ok := row["market"].(string)
		require.True(t, ok)
		markets[market] = true
		symbol, ok := row["symbol"].(string)
		require.True(t, ok)
		require.False(t, symbols[symbol], "duplicate monthly symbol: %s", symbol)
		symbols[symbol] = true
		require.IsType(t, float64(0), row["trading_days"])
		require.GreaterOrEqual(t, row["trading_days"].(float64), float64(15))
		require.Equal(t, "2026-06-30", row["last_date"])
		require.IsType(t, float64(0), row["total_traded_amount"])
		tradedAmount := row["total_traded_amount"].(float64)
		require.IsType(t, float64(0), row["total_traded_amount_100m"])
		require.InDelta(t, tradedAmount/100000000, row["total_traded_amount_100m"].(float64), 0.0000001)
		require.LessOrEqual(t, tradedAmount, previousTradedAmount)
		previousTradedAmount = tradedAmount
		require.NotNil(t, row["return_pct"])
	}
	require.True(t, markets["KOSPI"])
	require.True(t, markets["KOSDAQ"])

	collectionNames, err := runtime.Database().ListCollectionNames(ctx, bson.D{})
	require.NoError(t, err)
	for _, name := range collectionNames {
		require.False(t, strings.HasPrefix(name, "aggregate_tmp_"), "legacy temporary collection found: %s", name)
	}
}

func requireStageRows(t *testing.T, stages []struct {
	Name       string `json:"name"`
	Rows       int    `json:"rows"`
	Status     string `json:"status"`
	Collection string `json:"collection"`
}, name string, rows int) {
	t.Helper()
	for _, stage := range stages {
		if stage.Name != name {
			continue
		}
		require.Equal(t, "succeeded", stage.Status)
		require.Equal(t, rows, stage.Rows)
		require.Equal(t, "aggregate_stage_items", stage.Collection)
		return
	}
	t.Fatalf("aggregate stage not found: %s", name)
}
