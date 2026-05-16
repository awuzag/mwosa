package providerauth

import (
	"context"
	stdsql "database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

type Database struct {
	path   string
	mu     sync.Mutex
	client *bun.DB
}

func NewDatabase(path string) *Database {
	return &Database{path: path}
}

func (db *Database) Client(ctx context.Context) (*bun.DB, error) {
	if db == nil || strings.TrimSpace(db.path) == "" {
		return nil, oops.In("provider_auth_database").New("provider auth sqlite database path is empty")
	}
	errb := oops.In("provider_auth_database").With("path", db.path)

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.client != nil {
		return db.client, nil
	}
	directory := filepath.Dir(db.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, errb.With("directory", directory).Wrapf(err, "create provider auth sqlite directory")
	}

	rawDB, err := stdsql.Open("sqlite", db.path)
	if err != nil {
		return nil, errb.Wrapf(err, "open provider auth sqlite database")
	}
	rawDB.SetMaxOpenConns(1)

	if err := setupDatabase(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, errb.Wrap(err)
	}

	client := bun.NewDB(rawDB, sqlitedialect.New())
	if err := setupSchema(ctx, client); err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "apply provider auth sqlite schema"),
			errb.Wrap(client.Close()),
		)
	}
	db.client = client
	return db.client, nil
}

func (db *Database) Close() error {
	if db == nil {
		return nil
	}
	errb := oops.In("provider_auth_database").With("path", db.path)

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.client == nil {
		return nil
	}
	err := db.client.Close()
	db.client = nil
	if err != nil {
		return errb.Wrapf(err, "close provider auth sqlite database")
	}
	return nil
}

func setupDatabase(ctx context.Context, db *stdsql.DB) error {
	errb := oops.In("provider_auth_database")
	for _, statement := range []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return errb.With("statement", statement).Wrapf(err, "configure provider auth sqlite database")
		}
	}
	return nil
}

func setupSchema(ctx context.Context, db *bun.DB) error {
	errb := oops.In("provider_auth_database")
	if _, err := db.NewCreateTable().
		Model((*TokenRow)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return errb.With("table", "provider_auth_tokens").Wrapf(err, "create provider auth tokens table")
	}
	if _, err := db.NewCreateIndex().
		Model((*TokenRow)(nil)).
		Index("provider_auth_tokens_scope_key_unique").
		Column("provider_id", "auth_scope", "environment", "app_key_hash").
		Unique().
		IfNotExists().
		Exec(ctx); err != nil {
		return errb.With("index", "provider_auth_tokens_scope_key_unique").Wrapf(err, "create provider auth token unique index")
	}
	return nil
}
