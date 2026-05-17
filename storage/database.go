package storage

import (
	"context"
	stdsql "database/sql"
	"net/url"
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
	path       string
	mu         sync.Mutex
	client     *bun.DB
	readClient *bun.DB
}

func NewDatabase(path string) *Database {
	return &Database{path: path}
}

func (db *Database) Client(ctx context.Context) (*bun.DB, error) {
	return db.DB(ctx)
}

func (db *Database) Reader(ctx context.Context) (*bun.DB, error) {
	if db == nil || strings.TrimSpace(db.path) == "" {
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
	rawDB, err := stdsql.Open("sqlite", sqliteReadOnlyDSN(db.path))
	if err != nil {
		return nil, errb.Wrapf(err, "open read-only sqlite database")
	}
	rawDB.SetMaxOpenConns(4)

	if err := setupReadDatabase(ctx, rawDB); err != nil {
		_ = rawDB.Close()
		return nil, errb.Wrap(err)
	}

	db.readClient = bun.NewDB(rawDB, sqlitedialect.New())
	return db.readClient, nil
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

func setupReadDatabase(ctx context.Context, db *stdsql.DB) error {
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

func setupDatabase(ctx context.Context, db *stdsql.DB) error {
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

func setupSchema(ctx context.Context, db *bun.DB) error {
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
			return errb.With("table", table.name).Wrapf(err, "create sqlite table")
		}
	}
	if err := ensureStrategyVersionColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureBacktestRunColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureBacktestExperimentCaseColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureBacktestResultColumns(ctx, db); err != nil {
		return errb.Wrap(err)
	}
	if err := ensureBacktestWalkForwardStepColumns(ctx, db); err != nil {
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

func ensureStrategyVersionColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "spec_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "spec_hash", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE strategy_versions ADD COLUMN "+column.name+" "+column.definition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", "strategy_versions", "column", column.name).Wrapf(err, "ensure strategy version sqlite column")
		}
	}
	return nil
}

func ensureBacktestExperimentCaseColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "strategy_hash", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "run_hash", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "engine_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "indicator_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "metric_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "data_fingerprint", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE backtest_experiment_cases ADD COLUMN "+column.name+" "+column.definition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", "backtest_experiment_cases", "column", column.name).Wrapf(err, "ensure backtest experiment case sqlite column")
		}
	}
	return nil
}

func ensureBacktestRunColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "engine_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "indicator_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "metric_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE backtest_runs ADD COLUMN "+column.name+" "+column.definition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", "backtest_runs", "column", column.name).Wrapf(err, "ensure backtest run sqlite column")
		}
	}
	return nil
}

func ensureBacktestResultColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "engine_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "indicator_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "metric_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE backtest_results ADD COLUMN "+column.name+" "+column.definition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", "backtest_results", "column", column.name).Wrapf(err, "ensure backtest result sqlite column")
		}
	}
	return nil
}

func ensureBacktestWalkForwardStepColumns(ctx context.Context, db *bun.DB) error {
	errb := oops.In("storage_database")
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "strategy_hash", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "run_hash", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "engine_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "indicator_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "metric_registry_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "data_fingerprint", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "test_metrics_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE backtest_walk_forward_steps ADD COLUMN "+column.name+" "+column.definition); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return errb.With("table", "backtest_walk_forward_steps", "column", column.name).Wrapf(err, "ensure backtest walk-forward step sqlite column")
		}
	}
	return nil
}
