package storage

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
	return db.DB(ctx)
}

func (db *Database) DB(ctx context.Context) (*bun.DB, error) {
	if db == nil || strings.TrimSpace(db.path) == "" {
		return nil, oops.In("storage_database").New("sqlite database path is empty")
	}
	errb := oops.In("storage_database").With("path", db.path)

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.client != nil {
		return db.client, nil
	}
	directory := filepath.Dir(db.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, errb.With("directory", directory).Wrapf(err, "create sqlite database directory")
	}

	rawDB, err := stdsql.Open("sqlite", db.path)
	if err != nil {
		return nil, errb.Wrapf(err, "open sqlite database")
	}
	rawDB.SetMaxOpenConns(1)

	if err := setupDatabase(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, errb.Wrap(err)
	}

	client := bun.NewDB(rawDB, sqlitedialect.New())
	if err := setupSchema(ctx, client); err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "apply sqlite bun schema"),
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
	errb := oops.In("storage_database").With("path", db.path)

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.client == nil {
		return nil
	}
	err := db.client.Close()
	db.client = nil
	if err != nil {
		return errb.Wrapf(err, "close sqlite database")
	}
	return nil
}

func setupDatabase(ctx context.Context, db *stdsql.DB) error {
	errb := oops.In("storage_database")
	for _, statement := range []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return errb.With("statement", statement).Wrapf(err, "configure sqlite database")
		}
	}
	return nil
}

func setupSchema(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	tables := []struct {
		name  string
		model any
	}{
		{name: "daily_bar", model: (*DailyBarV1Row)(nil)},
		{name: "market_v2", model: (*MarketV2Row)(nil)},
		{name: "instrument_v2", model: (*InstrumentV2Row)(nil)},
		{name: "provider_source_v2", model: (*ProviderSourceV2Row)(nil)},
		{name: "daily_bar_v2", model: (*DailyBarV2Row)(nil)},
		{name: "daily_bar_extension_v2", model: (*DailyBarExtensionV2Row)(nil)},
		{name: "index_v1", model: (*IndexV1Row)(nil)},
		{name: "index_source_v1", model: (*IndexSourceV1Row)(nil)},
		{name: "index_bar_v1", model: (*IndexBarV1Row)(nil)},
		{name: "index_bar_extension_v1", model: (*IndexBarExtensionV1Row)(nil)},
		{name: "migration_runs", model: (*MigrationRunRow)(nil)},
		{name: "strategies", model: (*StrategyRow)(nil)},
		{name: "strategy_versions", model: (*StrategyVersionRow)(nil)},
		{name: "screen_runs", model: (*ScreenRunRow)(nil)},
		{name: "screen_run_items", model: (*ScreenRunItemRow)(nil)},
		{name: "provider_raw_snapshots", model: (*ProviderRawSnapshotRow)(nil)},
	}
	for _, table := range tables {
		if _, err := db.NewCreateTable().
			Model(table.model).
			IfNotExists().
			Exec(ctx); err != nil {
			return errb.With("table", table.name).Wrapf(err, "create sqlite table")
		}
	}

	indexes := []struct {
		name    string
		model   any
		columns []string
		unique  bool
	}{
		{
			name:    "daily_bar_natural_key",
			model:   (*DailyBarV1Row)(nil),
			columns: []string{"market", "security_type", "trading_date", "symbol", "provider", "provider_group"},
			unique:  true,
		},
		{
			name:    "idx_daily_bar_date",
			model:   (*DailyBarV1Row)(nil),
			columns: []string{"market", "security_type", "trading_date"},
		},
		{
			name:    "idx_daily_bar_symbol_date",
			model:   (*DailyBarV1Row)(nil),
			columns: []string{"market", "security_type", "symbol", "trading_date"},
		},
		{
			name:    "market_v2_code_unique",
			model:   (*MarketV2Row)(nil),
			columns: []string{"code"},
			unique:  true,
		},
		{
			name:    "instrument_v2_natural_key",
			model:   (*InstrumentV2Row)(nil),
			columns: []string{"market_id", "security_type", "symbol"},
			unique:  true,
		},
		{
			name:    "provider_source_v2_natural_key",
			model:   (*ProviderSourceV2Row)(nil),
			columns: []string{"provider", "provider_group", "operation"},
			unique:  true,
		},
		{
			name:    "idx_daily_bar_v2_date",
			model:   (*DailyBarV2Row)(nil),
			columns: []string{"trading_date"},
		},
		{
			name:    "idx_daily_bar_v2_instrument_date",
			model:   (*DailyBarV2Row)(nil),
			columns: []string{"instrument_id", "trading_date"},
		},
		{
			name:    "index_v1_natural_key",
			model:   (*IndexV1Row)(nil),
			columns: []string{"market", "index_code"},
			unique:  true,
		},
		{
			name:    "index_source_v1_natural_key",
			model:   (*IndexSourceV1Row)(nil),
			columns: []string{"provider", "provider_group", "operation", "provider_symbol"},
			unique:  true,
		},
		{
			name:    "idx_index_bar_v1_date",
			model:   (*IndexBarV1Row)(nil),
			columns: []string{"trading_date"},
		},
		{
			name:    "idx_index_bar_v1_index_date",
			model:   (*IndexBarV1Row)(nil),
			columns: []string{"index_id", "trading_date"},
		},
		{
			name:    "idx_migration_runs_resource",
			model:   (*MigrationRunRow)(nil),
			columns: []string{"resource"},
		},
		{
			name:    "idx_migration_runs_status",
			model:   (*MigrationRunRow)(nil),
			columns: []string{"status"},
		},
		{
			name:    "idx_strategies_archived_at",
			model:   (*StrategyRow)(nil),
			columns: []string{"archived_at"},
		},
		{
			name:    "strategy_versions_strategy_version_unique",
			model:   (*StrategyVersionRow)(nil),
			columns: []string{"strategy_id", "version"},
			unique:  true,
		},
		{
			name:    "idx_strategy_versions_query_hash",
			model:   (*StrategyVersionRow)(nil),
			columns: []string{"query_hash"},
		},
		{
			name:    "screen_runs_alias_unique",
			model:   (*ScreenRunRow)(nil),
			columns: []string{"alias"},
			unique:  true,
		},
		{
			name:    "idx_screen_runs_started_at",
			model:   (*ScreenRunRow)(nil),
			columns: []string{"started_at"},
		},
		{
			name:    "idx_screen_runs_strategy_started",
			model:   (*ScreenRunRow)(nil),
			columns: []string{"strategy_id", "started_at"},
		},
		{
			name:    "screen_run_items_run_ordinal_unique",
			model:   (*ScreenRunItemRow)(nil),
			columns: []string{"screen_run_id", "ordinal"},
			unique:  true,
		},
		{
			name:    "idx_screen_run_items_symbol",
			model:   (*ScreenRunItemRow)(nil),
			columns: []string{"symbol"},
		},
		{
			name:    "provider_raw_snapshots_natural_key",
			model:   (*ProviderRawSnapshotRow)(nil),
			columns: []string{"provider", "provider_group", "operation", "base_date"},
			unique:  true,
		},
		{
			name:    "idx_provider_raw_snapshots_operation_date",
			model:   (*ProviderRawSnapshotRow)(nil),
			columns: []string{"provider", "operation", "base_date"},
		},
	}
	for _, index := range indexes {
		query := db.NewCreateIndex().
			Model(index.model).
			Index(index.name).
			Column(index.columns...).
			IfNotExists()
		if index.unique {
			query = query.Unique()
		}
		if _, err := query.Exec(ctx); err != nil {
			return errb.With("index", index.name).Wrapf(err, "create daily_bar index")
		}
	}
	return nil
}
