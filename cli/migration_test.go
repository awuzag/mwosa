package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
)

func TestMigrateApplyMigratesDailyBarV1Rows(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	database := storage.NewDatabase(databasePath)
	seedCLIDailyBarV1Row(t, ctx, database)
	if err := database.Close(); err != nil {
		t.Fatalf("close seed database: %v", err)
	}

	applyCmd := NewRootCommand(BuildInfo{})
	var applyOut bytes.Buffer
	applyCmd.SetOut(&applyOut)
	applyCmd.SetErr(&applyOut)
	if err := executeForTest(t, ctx, applyCmd,
		"--database", databasePath,
		"--output", "json",
		"migrate", "apply",
	); err != nil {
		t.Fatalf("execute migrate apply: %v\n%s", err, applyOut.String())
	}

	var applyResult struct {
		Applied []migrationRunForTest `json:"applied"`
		Skipped []struct {
			ID string `json:"id"`
		} `json:"skipped"`
	}
	if err := json.Unmarshal(applyOut.Bytes(), &applyResult); err != nil {
		t.Fatalf("parse migrate apply json: %v\n%s", err, applyOut.String())
	}
	if len(applyResult.Applied) != 2 {
		t.Fatalf("applied len = %d, want 2: %#v", len(applyResult.Applied), applyResult)
	}
	v1Migration := findMigrationForTest(t, applyResult.Applied, "daily_bar_v1_to_v2")
	if v1Migration.Status != "applied" || v1Migration.RowsMigrated != 1 {
		t.Fatalf("daily_bar_v1_to_v2 migration = %+v, want applied rows=1", v1Migration)
	}
	cleanupMigration := findMigrationForTest(t, applyResult.Applied, "daily_bar_v2_extension_cleanup")
	if cleanupMigration.Status != "applied" {
		t.Fatalf("daily_bar_v2_extension_cleanup migration = %+v, want applied", cleanupMigration)
	}
	if len(applyResult.Skipped) != 0 {
		t.Fatalf("skipped len = %d, want 0", len(applyResult.Skipped))
	}

	statusCmd := NewRootCommand(BuildInfo{})
	var statusOut bytes.Buffer
	statusCmd.SetOut(&statusOut)
	statusCmd.SetErr(&statusOut)
	if err := executeForTest(t, ctx, statusCmd,
		"--database", databasePath,
		"--output", "json",
		"migrate", "status",
	); err != nil {
		t.Fatalf("execute migrate status: %v\n%s", err, statusOut.String())
	}
	var statusResult struct {
		Migrations []migrationRunForTest `json:"migrations"`
	}
	if err := json.Unmarshal(statusOut.Bytes(), &statusResult); err != nil {
		t.Fatalf("parse migrate status json: %v\n%s", err, statusOut.String())
	}
	if len(statusResult.Migrations) != 2 {
		t.Fatalf("status migrations len = %d, want 2: %#v", len(statusResult.Migrations), statusResult)
	}
	statusV1Migration := findMigrationForTest(t, statusResult.Migrations, "daily_bar_v1_to_v2")
	if statusV1Migration.Status != "applied" || statusV1Migration.RowsMigrated != 1 {
		t.Fatalf("status daily_bar_v1_to_v2 migration = %+v, want applied rows=1", statusV1Migration)
	}
}

type migrationRunForTest struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	RowsMigrated int64  `json:"rows_migrated"`
}

func findMigrationForTest(t *testing.T, migrations []migrationRunForTest, id string) migrationRunForTest {
	t.Helper()
	for _, migration := range migrations {
		if migration.ID == id {
			return migration
		}
	}
	t.Fatalf("migration %q was not found in %#v", id, migrations)
	return migrationRunForTest{}
}

func seedCLIDailyBarV1Row(t *testing.T, ctx context.Context, database *storage.Database) {
	t.Helper()
	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	row := storage.DailyBarV1Row{
		Provider:                     string(provider.ProviderDataGo),
		ProviderGroup:                string(provider.GroupSecuritiesProductPrice),
		Operation:                    string(provider.OperationGetETFPriceInfo),
		Market:                       string(provider.MarketKRX),
		SecurityType:                 string(provider.SecurityTypeETF),
		Symbol:                       "069500",
		ISIN:                         "KR7069500007",
		Name:                         "KODEX 200",
		TradingDate:                  "2024-04-15",
		Currency:                     "KRW",
		OpeningPrice:                 "35000",
		HighestPrice:                 "35200",
		LowestPrice:                  "34900",
		ClosingPrice:                 "35120",
		PriceChangeFromPreviousClose: "120",
		TradedVolume:                 "1000",
		TradedAmount:                 "35120000",
		MarketCapitalization:         "1000000000",
		ExtensionsJSON:               `{"nav":"35155.1"}`,
		CreatedAt:                    time.Now().UTC(),
		UpdatedAt:                    time.Now().UTC(),
	}
	if _, err := client.NewInsert().Model(&row).Exec(ctx); err != nil {
		t.Fatalf("insert v1 row: %v", err)
	}
}
