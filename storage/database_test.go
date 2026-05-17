package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseOpensLazilyAndReusesClient(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "mwosa.db")
	database := NewDatabase(dbPath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Dir(dbPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("database directory exists before first use: %v", err)
	}

	first, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("first client: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("database file after first use: %v", err)
	}

	second, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("second client: %v", err)
	}
	if first != second {
		t.Fatal("database returned a new client before close")
	}
}

func TestDatabaseRejectsEmptyPath(t *testing.T) {
	if _, err := NewDatabase("").Client(context.Background()); err == nil {
		t.Fatal("empty database path error is nil")
	}
}

func TestDatabaseReaderOpensReadOnlyClient(t *testing.T) {
	database := NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	writer, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("writer client: %v", err)
	}
	reader, err := database.Reader(context.Background())
	if err != nil {
		t.Fatalf("reader client: %v", err)
	}
	if writer == reader {
		t.Fatal("reader should use a separate sqlite client")
	}
	second, err := database.Reader(context.Background())
	if err != nil {
		t.Fatalf("second reader client: %v", err)
	}
	if reader != second {
		t.Fatal("database returned a new reader before close")
	}

	var count int
	if err := reader.QueryRowContext(context.Background(), "SELECT count(*) FROM sqlite_schema").Scan(&count); err != nil {
		t.Fatalf("reader query schema: %v", err)
	}
	if _, err := reader.ExecContext(context.Background(), "CREATE TABLE reader_write_probe (id INTEGER)"); err == nil {
		t.Fatal("reader write succeeded")
	}
}

func TestDatabaseConfiguresBusyTimeout(t *testing.T) {
	database := NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	writer, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("writer client: %v", err)
	}
	reader, err := database.Reader(context.Background())
	if err != nil {
		t.Fatalf("reader client: %v", err)
	}

	if got := sqlitePragmaInt(t, writer, "busy_timeout"); got != 5000 {
		t.Fatalf("writer busy_timeout = %d, want 5000", got)
	}
	if got := sqlitePragmaInt(t, reader, "busy_timeout"); got != 5000 {
		t.Fatalf("reader busy_timeout = %d, want 5000", got)
	}
}

func TestDatabaseReaderReadsDuringWritableTransaction(t *testing.T) {
	ctx := context.Background()
	database := NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	writer, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("writer client: %v", err)
	}
	if _, err := writer.ExecContext(ctx, "CREATE TABLE read_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatalf("create read probe table: %v", err)
	}
	if _, err := writer.ExecContext(ctx, "INSERT INTO read_probe (id, value) VALUES (1, 'committed')"); err != nil {
		t.Fatalf("insert read probe row: %v", err)
	}
	reader, err := database.Reader(ctx)
	if err != nil {
		t.Fatalf("reader client: %v", err)
	}

	tx, err := writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin writer transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, "UPDATE read_probe SET value = 'pending' WHERE id = 1"); err != nil {
		t.Fatalf("update read probe row in transaction: %v", err)
	}

	var got string
	if err := reader.QueryRowContext(ctx, "SELECT value FROM read_probe WHERE id = 1").Scan(&got); err != nil {
		t.Fatalf("reader query during writer transaction: %v", err)
	}
	if got != "committed" {
		t.Fatalf("reader value during writer transaction = %q, want committed snapshot", got)
	}
}

func TestDatabaseCreatesDailyBarIndexes(t *testing.T) {
	database := NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	indexes := sqliteIndexes(t, client, "daily_bar")

	if !indexes["daily_bar_natural_key"] {
		t.Fatal("daily_bar_natural_key unique index was not created")
	}
	for _, name := range []string{"idx_daily_bar_date", "idx_daily_bar_symbol_date"} {
		if _, ok := indexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}

	v2Indexes := sqliteIndexes(t, client, "daily_bar_v2")
	for _, name := range []string{"idx_daily_bar_v2_date", "idx_daily_bar_v2_instrument_date"} {
		if _, ok := v2Indexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}
	compositionIndexes := sqliteIndexes(t, client, "composition_observation_v1")
	for _, name := range []string{"composition_observation_v1_natural_key", "idx_composition_observation_v1_subject_as_of"} {
		if _, ok := compositionIndexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}
	compositionMemberIndexes := sqliteIndexes(t, client, "composition_member_v1")
	for _, name := range []string{"composition_member_v1_ordinal_unique", "idx_composition_member_v1_member"} {
		if _, ok := compositionMemberIndexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}
	extensionIndexes := sqliteIndexes(t, client, "daily_bar_extension_v2")
	if _, ok := extensionIndexes["idx_daily_bar_extension_v2_bar"]; ok {
		t.Fatal("idx_daily_bar_extension_v2_bar should not be created because the primary key index covers bar-prefix lookups")
	}

	migrationIndexes := sqliteIndexes(t, client, "migration_runs")
	for _, name := range []string{"idx_migration_runs_resource", "idx_migration_runs_status"} {
		if _, ok := migrationIndexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}

	rawIndexes := sqliteIndexes(t, client, "provider_raw_snapshots")
	for _, name := range []string{"provider_raw_snapshots_natural_key", "idx_provider_raw_snapshots_operation_date"} {
		if _, ok := rawIndexes[name]; !ok {
			t.Fatalf("%s index was not created", name)
		}
	}
}

func TestDatabaseMigratesBacktestRuntimeMetadataColumns(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "mwosa.db")
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE backtest_runs (id TEXT PRIMARY KEY, run_name TEXT NOT NULL, created_at TIMESTAMP NOT NULL, result_hash TEXT NOT NULL)`,
		`CREATE TABLE backtest_experiment_cases (id TEXT PRIMARY KEY, experiment_id TEXT NOT NULL, rank INTEGER NOT NULL)`,
		`CREATE TABLE backtest_results (id TEXT PRIMARY KEY, experiment_case_id TEXT NOT NULL)`,
		`CREATE TABLE backtest_walk_forward_steps (id TEXT PRIMARY KEY, experiment_id TEXT NOT NULL, step_index INTEGER NOT NULL)`,
	} {
		if _, err := raw.ExecContext(ctx, statement); err != nil {
			_ = raw.Close()
			t.Fatalf("create legacy table: %v", err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close sqlite fixture: %v", err)
	}

	database := NewDatabase(dbPath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	for _, table := range []string{"backtest_runs", "backtest_experiment_cases", "backtest_results", "backtest_walk_forward_steps"} {
		columns := sqliteColumns(t, client, table)
		for _, column := range []string{"engine_version", "indicator_registry_version", "metric_registry_version"} {
			if !columns[column] {
				t.Fatalf("%s.%s was not migrated", table, column)
			}
		}
	}
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func sqlitePragmaInt(t *testing.T, client queryRower, pragma string) int {
	t.Helper()

	var got int
	if err := client.QueryRowContext(context.Background(), "PRAGMA "+pragma).Scan(&got); err != nil {
		t.Fatalf("query sqlite pragma %s: %v", pragma, err)
	}
	return got
}

func sqliteIndexes(t *testing.T, client queryer, table string) map[string]bool {
	t.Helper()

	rows, err := client.QueryContext(context.Background(), `PRAGMA index_list('`+table+`')`)
	if err != nil {
		t.Fatalf("index list: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var seq int
		var name string
		var unique bool
		var origin string
		var partial bool
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index row: %v", err)
		}
		indexes[name] = unique
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate index rows: %v", err)
	}
	return indexes
}

func sqliteColumns(t *testing.T, client queryer, table string) map[string]bool {
	t.Helper()

	rows, err := client.QueryContext(context.Background(), `PRAGMA table_info('`+table+`')`)
	if err != nil {
		t.Fatalf("table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull bool
		var defaultValue any
		var pk bool
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info row: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info rows: %v", err)
	}
	return columns
}
