//go:build integration

package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
)

func TestPostgresDatabaseBootstrapsSchemaAndJSONB(t *testing.T) {
	ctx := context.Background()
	postgres := integrationtest.StartPostgres(t)
	database := NewDatabaseWithConfig(DatabaseConfig{
		Backend: BackendPostgres,
		URL:     postgres.DSN,
	})
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("postgres client: %v", err)
	}
	reader, err := database.Reader(ctx)
	if err != nil {
		t.Fatalf("postgres reader: %v", err)
	}
	if reader != client {
		t.Fatal("postgres reader should reuse the writer client")
	}

	requirePostgresRelation(t, client, "macro_indicator")
	requirePostgresRelation(t, client, "idx_macro_observation_period")
	requirePostgresColumnType(t, client, "macro_indicator_provider_doc", "document_json", "jsonb")

	if err := setupSchema(ctx, client, BackendPostgres); err != nil {
		t.Fatalf("second schema setup: %v", err)
	}

	var tableCount int
	if err := reader.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&tableCount); err != nil {
		t.Fatalf("reader query table count: %v", err)
	}
	if tableCount == 0 {
		t.Fatal("postgres schema table count = 0")
	}

	_, err = client.ExecContext(ctx, `INSERT INTO macro_indicator_provider_doc (indicator_id, provider, schema_version, document_json, updated_at_ms) VALUES ('probe', 'ecos', '1.0.0', '{bad', 1)`)
	if err == nil {
		t.Fatal("invalid macro provider document JSON insert succeeded")
	}
}

func requirePostgresRelation(t *testing.T, db queryRower, name string) {
	t.Helper()

	var relation sql.NullString
	if err := db.QueryRowContext(context.Background(), `SELECT to_regclass(?)::text`, "public."+name).Scan(&relation); err != nil {
		t.Fatalf("query postgres relation %s: %v", name, err)
	}
	if !relation.Valid || relation.String == "" {
		t.Fatalf("postgres relation %s was not created", name)
	}
}

func requirePostgresColumnType(t *testing.T, db queryRower, table string, column string, want string) {
	t.Helper()

	var got string
	if err := db.QueryRowContext(
		context.Background(),
		`SELECT udt_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = ? AND column_name = ?`,
		table,
		column,
	).Scan(&got); err != nil {
		t.Fatalf("query postgres column type %s.%s: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("postgres column type %s.%s = %q, want %q", table, column, got, want)
	}
}
