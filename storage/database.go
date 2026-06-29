package storage

import (
	"context"
	stdsql "database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

type Backend string

const (
	BackendSQLite   Backend = "sqlite"
	BackendPostgres Backend = "postgres"
)

type DatabaseConfig struct {
	Backend Backend
	Path    string
	URL     string
}

type SQLDatabase struct {
	backend    Backend
	path       string
	url        string
	mu         sync.Mutex
	client     *bun.DB
	readClient *bun.DB
}

func NewDatabase(path string) *SQLDatabase {
	return NewSQLDatabase(path)
}

func NewSQLDatabase(path string) *SQLDatabase {
	return NewSQLDatabaseWithConfig(DatabaseConfig{Backend: BackendSQLite, Path: path})
}

func NewDatabaseWithConfig(config DatabaseConfig) *SQLDatabase {
	return NewSQLDatabaseWithConfig(config)
}

func NewSQLDatabaseWithConfig(config DatabaseConfig) *SQLDatabase {
	backend := config.Backend
	if backend == "" {
		backend = BackendSQLite
	}
	return &SQLDatabase{
		backend: backend,
		path:    config.Path,
		url:     config.URL,
	}
}

func (db *SQLDatabase) Client(ctx context.Context) (*bun.DB, error) {
	return db.DB(ctx)
}

func (db *SQLDatabase) Reader(ctx context.Context) (*bun.DB, error) {
	if db == nil {
		return nil, oops.In("storage_database").New("database is nil")
	}
	if db.backend == BackendPostgres {
		return db.DB(ctx)
	}
	if strings.TrimSpace(db.path) == "" {
		return nil, oops.In("storage_database").New("sqlite database path is empty")
	}
	if _, err := db.DB(ctx); err != nil {
		return nil, err
	}
	errb := oops.In("storage_database").With("path", db.path)

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.readClient != nil {
		return db.readClient, nil
	}
	client, err := openSQLiteReader(ctx, db.path)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	db.readClient = client
	return client, nil
}

func (db *SQLDatabase) DB(ctx context.Context) (*bun.DB, error) {
	if db == nil {
		return nil, oops.In("storage_database").New("database is nil")
	}
	if db.backend == "" {
		db.backend = BackendSQLite
	}
	errb := oops.In("storage_database").With("backend", db.backend, "database", db.safeLocation())

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.client != nil {
		return db.client, nil
	}

	client, err := db.open(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	if err := setupSchema(ctx, client, db.backend); err != nil {
		return nil, oops.Join(
			errb.Wrapf(err, "apply bun schema"),
			errb.Wrap(client.Close()),
		)
	}
	db.client = client
	return db.client, nil
}

func (db *SQLDatabase) Close() error {
	if db == nil {
		return nil
	}
	errb := oops.In("storage_database").With("backend", db.backend, "database", db.safeLocation())

	db.mu.Lock()
	defer db.mu.Unlock()

	var out error
	if db.readClient != nil {
		if err := db.readClient.Close(); err != nil {
			out = oops.Join(out, errb.Wrapf(err, "close read-only sqlite database"))
		}
		db.readClient = nil
	}
	if db.client == nil {
		return out
	}
	err := db.client.Close()
	db.client = nil
	if err != nil {
		out = oops.Join(out, errb.Wrapf(err, "close sqlite database"))
	}
	return out
}

func (db *SQLDatabase) open(ctx context.Context) (*bun.DB, error) {
	switch db.backend {
	case BackendSQLite:
		return openSQLiteWriter(ctx, db.path)
	case BackendPostgres:
		return openPostgres(ctx, db.url)
	default:
		return nil, oops.In("storage_database").With("backend", db.backend).New("unsupported database backend")
	}
}

func (db *SQLDatabase) safeLocation() string {
	if db == nil {
		return ""
	}
	switch db.backend {
	case BackendPostgres:
		if strings.TrimSpace(db.url) == "" {
			return ""
		}
		return "<configured>"
	default:
		return db.path
	}
}

func openSQLiteWriter(ctx context.Context, path string) (*bun.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, oops.In("storage_database").New("sqlite database path is empty")
	}
	errb := oops.In("storage_database").With("path", path)
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, errb.With("directory", directory).Wrapf(err, "create sqlite database directory")
	}
	rawDB, err := stdsql.Open("sqlite", path)
	if err != nil {
		return nil, errb.Wrapf(err, "open sqlite database")
	}
	rawDB.SetMaxOpenConns(1)
	if err := setupSQLiteWriter(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, errb.Wrap(err)
	}
	return bun.NewDB(rawDB, sqlitedialect.New()), nil
}

func openSQLiteReader(ctx context.Context, path string) (*bun.DB, error) {
	rawDB, err := stdsql.Open("sqlite", sqliteReadOnlyDSN(path))
	if err != nil {
		return nil, oops.In("storage_database").Wrapf(err, "open read-only sqlite database")
	}
	rawDB.SetMaxOpenConns(4)
	if err := setupSQLiteReader(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, err
	}
	return bun.NewDB(rawDB, sqlitedialect.New()), nil
}

func openPostgres(ctx context.Context, dsn string) (*bun.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, oops.In("storage_database").New("postgres database URL is empty")
	}
	rawDB := stdsql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(postgresDSNWithApplicationName(dsn))))
	rawDB.SetMaxOpenConns(16)
	rawDB.SetMaxIdleConns(4)
	rawDB.SetConnMaxLifetime(30 * time.Minute)
	if err := setupPostgres(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, err
	}
	return bun.NewDB(rawDB, pgdialect.New()), nil
}

func setupSQLiteReader(ctx context.Context, db *stdsql.DB) error {
	errb := oops.In("storage_database")
	for _, statement := range []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA foreign_keys = ON`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return errb.With("statement", statement).Wrapf(err, "configure read-only sqlite database")
		}
	}
	return nil
}

func sqliteReadOnlyDSN(path string) string {
	dsn := url.URL{Scheme: "file", Path: path}
	query := dsn.Query()
	query.Set("mode", "ro")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func setupSQLiteWriter(ctx context.Context, db *stdsql.DB) error {
	errb := oops.In("storage_database")
	for _, statement := range []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return errb.With("statement", statement).Wrapf(err, "configure sqlite database")
		}
	}
	return nil
}

func setupPostgres(ctx context.Context, db *stdsql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return oops.In("storage_database").Wrapf(err, "ping postgres database")
	}
	return nil
}

func postgresDSNWithApplicationName(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" {
		return dsn
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("application_name")) == "" {
		query.Set("application_name", "mwosa")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func setupSchema(ctx context.Context, db *bun.DB, backend Backend) error {
	errb := oops.In("storage_database")
	tables := []struct {
		name  string
		model any
	}{
		{name: "daily_bar", model: (*DailyBarV1Row)(nil)},
		{name: "market_v2", model: (*MarketV2Row)(nil)},
		{name: "instrument_v2", model: (*InstrumentV2Row)(nil)},
		{name: "instrument_source_v1", model: (*InstrumentSourceV1Row)(nil)},
		{name: "instrument_extension_v1", model: (*InstrumentExtensionV1Row)(nil)},
		{name: "provider_source_v2", model: (*ProviderSourceV2Row)(nil)},
		{name: "daily_bar_v2", model: (*DailyBarV2Row)(nil)},
		{name: "daily_bar_extension_v2", model: (*DailyBarExtensionV2Row)(nil)},
		{name: "composition_observation_v1", model: (*CompositionObservationV1Row)(nil)},
		{name: "composition_member_v1", model: (*CompositionMemberV1Row)(nil)},
		{name: "index_v1", model: (*IndexV1Row)(nil)},
		{name: "index_source_v1", model: (*IndexSourceV1Row)(nil)},
		{name: "index_bar_v1", model: (*IndexBarV1Row)(nil)},
		{name: "index_bar_extension_v1", model: (*IndexBarExtensionV1Row)(nil)},
		{name: "macro_indicator", model: (*MacroIndicatorRow)(nil)},
		{name: "macro_observation", model: (*MacroObservationRow)(nil)},
		{name: "macro_indicator_source", model: (*MacroIndicatorSourceRow)(nil)},
		{name: "macro_indicator_provider_doc", model: (*MacroIndicatorProviderDocRow)(nil)},
		{name: "migration_runs", model: (*MigrationRunRow)(nil)},
		{name: "strategies", model: (*StrategyRow)(nil)},
		{name: "strategy_versions", model: (*StrategyVersionRow)(nil)},
		{name: "screen_runs", model: (*ScreenRunRow)(nil)},
		{name: "screen_run_items", model: (*ScreenRunItemRow)(nil)},
		{name: "backtest_strategies", model: (*BacktestStrategyRow)(nil)},
		{name: "backtest_strategy_versions", model: (*BacktestStrategyVersionRow)(nil)},
		{name: "backtest_runs", model: (*BacktestRunRow)(nil)},
		{name: "backtest_experiments", model: (*BacktestExperimentRow)(nil)},
		{name: "backtest_experiment_cases", model: (*BacktestExperimentCaseRow)(nil)},
		{name: "backtest_results", model: (*BacktestResultRow)(nil)},
		{name: "backtest_metric_summaries", model: (*BacktestMetricSummaryRow)(nil)},
		{name: "backtest_walk_forward_steps", model: (*BacktestWalkForwardStepRow)(nil)},
		{name: "provider_raw_snapshots", model: (*ProviderRawSnapshotRow)(nil)},
		{name: "opendart_companies", model: (*OpenDARTCompanyRow)(nil)},
		{name: "company_v1", model: (*CompanyV1Row)(nil)},
		{name: "company_identifier_v1", model: (*CompanyIdentifierV1Row)(nil)},
		{name: "instrument_company_link_v1", model: (*InstrumentCompanyLinkV1Row)(nil)},
		{name: "financial_statement_v1", model: (*FinancialStatementV1Row)(nil)},
		{name: "financial_line_item_v1", model: (*FinancialLineItemV1Row)(nil)},
		{name: "financial_metric_v1", model: (*FinancialMetricV1Row)(nil)},
		{name: "valuation_snapshot_v1", model: (*ValuationSnapshotV1Row)(nil)},
		{name: "company_fact_v1", model: (*CompanyFactV1Row)(nil)},
		{name: "company_event_v1", model: (*CompanyEventV1Row)(nil)},
	}
	for _, table := range tables {
		if _, err := db.NewCreateTable().
			Model(table.model).
			IfNotExists().
			Exec(ctx); err != nil {
			return errb.With("backend", backend, "table", table.name).Wrapf(err, "create table")
		}
	}
	if err := ensureBackendSchemaCompatibility(ctx, db, backend); err != nil {
		return errb.Wrap(err)
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
			name:    "instrument_source_v1_natural_key",
			model:   (*InstrumentSourceV1Row)(nil),
			columns: []string{"provider", "provider_group", "operation", "provider_symbol"},
			unique:  true,
		},
		{
			name:    "idx_instrument_source_v1_instrument",
			model:   (*InstrumentSourceV1Row)(nil),
			columns: []string{"instrument_id"},
		},
		{
			name:    "idx_instrument_extension_v1_key_value",
			model:   (*InstrumentExtensionV1Row)(nil),
			columns: []string{"key", "value"},
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
			name:    "composition_observation_v1_natural_key",
			model:   (*CompositionObservationV1Row)(nil),
			columns: []string{"source_id", "subject_instrument_id", "as_of_date", "observed_at_ms"},
			unique:  true,
		},
		{
			name:    "idx_composition_observation_v1_subject_as_of",
			model:   (*CompositionObservationV1Row)(nil),
			columns: []string{"subject_instrument_id", "as_of_date"},
		},
		{
			name:    "composition_member_v1_ordinal_unique",
			model:   (*CompositionMemberV1Row)(nil),
			columns: []string{"composition_id", "ordinal"},
			unique:  true,
		},
		{
			name:    "idx_composition_member_v1_member",
			model:   (*CompositionMemberV1Row)(nil),
			columns: []string{"member_instrument_id"},
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
			name:    "macro_indicator_provider_source_unique",
			model:   (*MacroIndicatorRow)(nil),
			columns: []string{"provider", "source_code"},
			unique:  true,
		},
		{
			name:    "idx_macro_indicator_preset",
			model:   (*MacroIndicatorRow)(nil),
			columns: []string{"provider", "preset"},
		},
		{
			name:    "idx_macro_indicator_category",
			model:   (*MacroIndicatorRow)(nil),
			columns: []string{"category", "frequency"},
		},
		{
			name:    "idx_macro_observation_period",
			model:   (*MacroObservationRow)(nil),
			columns: []string{"indicator_id", "period"},
		},
		{
			name:    "idx_macro_indicator_source_provider",
			model:   (*MacroIndicatorSourceRow)(nil),
			columns: []string{"provider", "source_code"},
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
			name:    "idx_strategy_versions_spec_hash",
			model:   (*StrategyVersionRow)(nil),
			columns: []string{"spec_hash"},
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
			name:    "idx_backtest_strategies_deleted_at",
			model:   (*BacktestStrategyRow)(nil),
			columns: []string{"deleted_at"},
		},
		{
			name:    "backtest_strategy_versions_strategy_version_unique",
			model:   (*BacktestStrategyVersionRow)(nil),
			columns: []string{"strategy_id", "version"},
			unique:  true,
		},
		{
			name:    "idx_backtest_strategy_versions_spec_hash",
			model:   (*BacktestStrategyVersionRow)(nil),
			columns: []string{"spec_hash"},
		},
		{
			name:    "idx_backtest_runs_name_created",
			model:   (*BacktestRunRow)(nil),
			columns: []string{"run_name", "created_at"},
		},
		{
			name:    "idx_backtest_runs_result_hash",
			model:   (*BacktestRunRow)(nil),
			columns: []string{"result_hash"},
		},
		{
			name:    "idx_backtest_experiments_name_created",
			model:   (*BacktestExperimentRow)(nil),
			columns: []string{"name", "created_at"},
		},
		{
			name:    "idx_backtest_experiments_spec_hash",
			model:   (*BacktestExperimentRow)(nil),
			columns: []string{"spec_hash"},
		},
		{
			name:    "idx_backtest_experiment_cases_experiment",
			model:   (*BacktestExperimentCaseRow)(nil),
			columns: []string{"experiment_id", "rank"},
		},
		{
			name:    "idx_backtest_results_case",
			model:   (*BacktestResultRow)(nil),
			columns: []string{"experiment_case_id"},
			unique:  true,
		},
		{
			name:    "idx_backtest_metric_summaries_case_metric",
			model:   (*BacktestMetricSummaryRow)(nil),
			columns: []string{"experiment_case_id", "metric"},
			unique:  true,
		},
		{
			name:    "idx_backtest_walk_forward_steps_experiment",
			model:   (*BacktestWalkForwardStepRow)(nil),
			columns: []string{"experiment_id", "step_index"},
			unique:  true,
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
		{
			name:    "opendart_companies_corp_code_unique",
			model:   (*OpenDARTCompanyRow)(nil),
			columns: []string{"corp_code"},
			unique:  true,
		},
		{
			name:    "idx_opendart_companies_stock_code",
			model:   (*OpenDARTCompanyRow)(nil),
			columns: []string{"stock_code"},
		},
		{
			name:    "idx_opendart_companies_corp_name",
			model:   (*OpenDARTCompanyRow)(nil),
			columns: []string{"corp_name"},
		},
		{
			name:    "idx_company_v1_name",
			model:   (*CompanyV1Row)(nil),
			columns: []string{"name"},
		},
		{
			name:    "company_identifier_v1_natural_key",
			model:   (*CompanyIdentifierV1Row)(nil),
			columns: []string{"provider", "provider_group", "operation", "identifier_type", "identifier_value", "valid_from"},
			unique:  true,
		},
		{
			name:    "idx_company_identifier_v1_company",
			model:   (*CompanyIdentifierV1Row)(nil),
			columns: []string{"company_id"},
		},
		{
			name:    "idx_company_identifier_v1_type_value",
			model:   (*CompanyIdentifierV1Row)(nil),
			columns: []string{"identifier_type", "identifier_value"},
		},
		{
			name:    "instrument_company_link_v1_natural_key",
			model:   (*InstrumentCompanyLinkV1Row)(nil),
			columns: []string{"instrument_id", "company_id", "relation_type", "valid_from"},
			unique:  true,
		},
		{
			name:    "idx_instrument_company_link_v1_company",
			model:   (*InstrumentCompanyLinkV1Row)(nil),
			columns: []string{"company_id"},
		},
		{
			name:    "financial_statement_v1_natural_key",
			model:   (*FinancialStatementV1Row)(nil),
			columns: []string{"company_id", "instrument_id", "provider", "provider_group", "operation", "rcept_no", "fiscal_year", "report_code", "fs_div", "statement_type"},
			unique:  true,
		},
		{
			name:    "idx_financial_statement_v1_company_year",
			model:   (*FinancialStatementV1Row)(nil),
			columns: []string{"company_id", "fiscal_year", "fiscal_period"},
		},
		{
			name:    "financial_line_item_v1_natural_key",
			model:   (*FinancialLineItemV1Row)(nil),
			columns: []string{"statement_id", "account_id", "account_name", "period_name", "ord"},
			unique:  true,
		},
		{
			name:    "idx_financial_line_item_v1_statement",
			model:   (*FinancialLineItemV1Row)(nil),
			columns: []string{"statement_id"},
		},
		{
			name:    "idx_financial_line_item_v1_canonical",
			model:   (*FinancialLineItemV1Row)(nil),
			columns: []string{"canonical_account"},
		},
		{
			name:    "financial_metric_v1_natural_key",
			model:   (*FinancialMetricV1Row)(nil),
			columns: []string{"company_id", "instrument_id", "metric", "fiscal_year", "fiscal_period", "as_of_date", "formula_version"},
			unique:  true,
		},
		{
			name:    "idx_financial_metric_v1_company_metric",
			model:   (*FinancialMetricV1Row)(nil),
			columns: []string{"company_id", "metric", "fiscal_year", "fiscal_period"},
		},
		{
			name:    "idx_financial_metric_v1_instrument_metric",
			model:   (*FinancialMetricV1Row)(nil),
			columns: []string{"instrument_id", "metric", "as_of_date"},
		},
		{
			name:    "valuation_snapshot_v1_natural_key",
			model:   (*ValuationSnapshotV1Row)(nil),
			columns: []string{"company_id", "instrument_id", "as_of_date", "metric_source_version"},
			unique:  true,
		},
		{
			name:    "idx_valuation_snapshot_v1_instrument_date",
			model:   (*ValuationSnapshotV1Row)(nil),
			columns: []string{"instrument_id", "as_of_date"},
		},
		{
			name:    "company_fact_v1_natural_key",
			model:   (*CompanyFactV1Row)(nil),
			columns: []string{"company_id", "instrument_id", "provider", "provider_group", "operation", "fact_type", "fiscal_year", "report_code", "rcept_no", "key"},
			unique:  true,
		},
		{
			name:    "idx_company_fact_v1_company_type",
			model:   (*CompanyFactV1Row)(nil),
			columns: []string{"company_id", "fact_type", "fiscal_year"},
		},
		{
			name:    "idx_company_fact_v1_instrument_type_date",
			model:   (*CompanyFactV1Row)(nil),
			columns: []string{"instrument_id", "fact_type", "fact_date"},
		},
		{
			name:    "company_event_v1_natural_key",
			model:   (*CompanyEventV1Row)(nil),
			columns: []string{"company_id", "instrument_id", "provider", "provider_group", "operation", "event_type", "rcept_no", "title"},
			unique:  true,
		},
		{
			name:    "idx_company_event_v1_company_date",
			model:   (*CompanyEventV1Row)(nil),
			columns: []string{"company_id", "event_date", "rcept_dt"},
		},
		{
			name:    "idx_company_event_v1_instrument_type_date",
			model:   (*CompanyEventV1Row)(nil),
			columns: []string{"instrument_id", "event_type", "event_date"},
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

func ensureMacroProviderDocJSONValidation(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	statements := []struct {
		name string
		sql  string
	}{
		{
			name: "macro_indicator_provider_doc_json_valid_insert",
			sql: `CREATE TRIGGER IF NOT EXISTS macro_indicator_provider_doc_json_valid_insert
BEFORE INSERT ON macro_indicator_provider_doc
WHEN json_valid(NEW.document_json) = 0
BEGIN
	SELECT RAISE(ABORT, 'macro_indicator_provider_doc.document_json must be valid JSON');
END`,
		},
		{
			name: "macro_indicator_provider_doc_json_valid_update",
			sql: `CREATE TRIGGER IF NOT EXISTS macro_indicator_provider_doc_json_valid_update
BEFORE UPDATE OF document_json ON macro_indicator_provider_doc
WHEN json_valid(NEW.document_json) = 0
BEGIN
	SELECT RAISE(ABORT, 'macro_indicator_provider_doc.document_json must be valid JSON');
END`,
		},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.sql); err != nil {
			return errb.With("trigger", statement.name).Wrapf(err, "create macro provider doc json validation trigger")
		}
	}
	return nil
}

func ensureBackendSchemaCompatibility(ctx context.Context, db *bun.DB, backend Backend) error {
	switch backend {
	case BackendSQLite:
		return ensureSQLiteSchemaCompatibility(ctx, db)
	case BackendPostgres:
		return ensurePostgresSchemaCompatibility(ctx, db)
	default:
		return oops.In("storage_database").With("backend", backend).New("unsupported database backend")
	}
}

func ensureSQLiteSchemaCompatibility(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	if err := ensureSQLiteStrategyVersionColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureSQLiteBacktestRunColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureSQLiteBacktestExperimentCaseColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureSQLiteBacktestResultColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureSQLiteBacktestWalkForwardStepColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureMacroProviderDocJSONValidation(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	return nil
}

func ensurePostgresSchemaCompatibility(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		for _, column := range table.columns {
			statement := "ALTER TABLE " + table.name + " ADD COLUMN IF NOT EXISTS " + column.name + " " + column.postgresDefinition
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return errb.With("table", table.name, "column", column.name).Wrapf(err, "ensure postgres column")
			}
		}
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_indicator_provider_doc ALTER COLUMN document_json DROP DEFAULT`); err != nil {
		return errb.With("table", "macro_indicator_provider_doc", "column", "document_json").Wrapf(err, "drop postgres document default")
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_indicator_provider_doc ALTER COLUMN document_json TYPE jsonb USING document_json::jsonb`); err != nil {
		return errb.With("table", "macro_indicator_provider_doc", "column", "document_json").Wrapf(err, "ensure postgres jsonb document column")
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_indicator_provider_doc ALTER COLUMN document_json SET DEFAULT '{}'::jsonb`); err != nil {
		return errb.With("table", "macro_indicator_provider_doc", "column", "document_json").Wrapf(err, "ensure postgres jsonb document default")
	}
	return nil
}

type schemaRepairTable struct {
	name    string
	columns []schemaRepairColumn
}

type schemaRepairColumn struct {
	name               string
	sqliteDefinition   string
	postgresDefinition string
}

func schemaRepairColumns() []schemaRepairTable {
	return []schemaRepairTable{
		{
			name: "strategy_versions",
			columns: []schemaRepairColumn{
				{name: "spec_json", sqliteDefinition: "TEXT NOT NULL DEFAULT '{}'", postgresDefinition: "TEXT NOT NULL DEFAULT '{}'"},
				{name: "spec_hash", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
			},
		},
		{
			name: "backtest_experiment_cases",
			columns: []schemaRepairColumn{
				{name: "strategy_hash", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "run_hash", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "engine_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "indicator_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "metric_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "data_fingerprint", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
			},
		},
		{
			name: "backtest_runs",
			columns: []schemaRepairColumn{
				{name: "engine_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "indicator_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "metric_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
			},
		},
		{
			name: "backtest_results",
			columns: []schemaRepairColumn{
				{name: "engine_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "indicator_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "metric_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
			},
		},
		{
			name: "backtest_walk_forward_steps",
			columns: []schemaRepairColumn{
				{name: "strategy_hash", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "run_hash", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "engine_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "indicator_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "metric_registry_version", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "data_fingerprint", sqliteDefinition: "TEXT NOT NULL DEFAULT ''", postgresDefinition: "TEXT NOT NULL DEFAULT ''"},
				{name: "test_metrics_json", sqliteDefinition: "TEXT NOT NULL DEFAULT '{}'", postgresDefinition: "TEXT NOT NULL DEFAULT '{}'"},
			},
		},
	}
}

func ensureSQLiteColumns(ctx context.Context, db *bun.DB, table schemaRepairTable) error {
	errb := oops.In("storage_database")
	for _, column := range table.columns {
		if _, err := db.ExecContext(ctx, "ALTER TABLE "+table.name+" ADD COLUMN "+column.name+" "+column.sqliteDefinition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", table.name, "column", column.name).Wrapf(err, "ensure sqlite column")
		}
	}
	return nil
}

func ensureSQLiteStrategyVersionColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		if table.name == "strategy_versions" {
			return ensureSQLiteColumns(ctx, db, table)
		}
	}
	return errb.New("strategy_versions schema repair definition is missing")
}

func ensureSQLiteBacktestExperimentCaseColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		if table.name == "backtest_experiment_cases" {
			return ensureSQLiteColumns(ctx, db, table)
		}
	}
	return errb.New("backtest_experiment_cases schema repair definition is missing")
}

func ensureSQLiteBacktestRunColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		if table.name == "backtest_runs" {
			return ensureSQLiteColumns(ctx, db, table)
		}
	}
	return errb.New("backtest_runs schema repair definition is missing")
}

func ensureSQLiteBacktestResultColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		if table.name == "backtest_results" {
			return ensureSQLiteColumns(ctx, db, table)
		}
	}
	return errb.New("backtest_results schema repair definition is missing")
}

func ensureSQLiteBacktestWalkForwardStepColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, table := range schemaRepairColumns() {
		if table.name == "backtest_walk_forward_steps" {
			return ensureSQLiteColumns(ctx, db, table)
		}
	}
	return errb.New("backtest_walk_forward_steps schema repair definition is missing")
}
