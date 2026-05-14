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
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
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
