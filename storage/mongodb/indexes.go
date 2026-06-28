package mongodb

import (
	"context"
	"errors"

	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type CollectionSpec struct {
	Name      string
	Validator bson.D
	Indexes   []IndexSpec
}

type IndexSpec struct {
	Name   string
	Keys   bson.D
	Unique bool
	Sparse bool
}

func CollectionSpecs() []CollectionSpec {
	return []CollectionSpec{
		{
			Name:      "storage_metadata",
			Validator: baseValidator([]string{"storage_kind"}),
			Indexes: []IndexSpec{
				{
					Name:   "storage_metadata_storage_kind_unique",
					Keys:   bson.D{{Key: "storage_kind", Value: 1}},
					Unique: true,
				},
			},
		},
		{
			Name:      "markets",
			Validator: baseValidator([]string{"code", "timezone"}),
			Indexes: []IndexSpec{
				{
					Name:   "markets_code_unique",
					Keys:   bson.D{{Key: "code", Value: 1}},
					Unique: true,
				},
			},
		},
		{
			Name:      "instruments",
			Validator: baseValidator([]string{"market_key", "security_type", "symbol", "sources"}),
			Indexes: []IndexSpec{
				{
					Name:   "instruments_market_security_symbol_unique",
					Keys:   bson.D{{Key: "market_key", Value: 1}, {Key: "security_type", Value: 1}, {Key: "symbol", Value: 1}},
					Unique: true,
				},
				{
					Name:   "instruments_source_provider_symbol_unique",
					Keys:   bson.D{{Key: "sources.provider", Value: 1}, {Key: "sources.provider_group", Value: 1}, {Key: "sources.operation", Value: 1}, {Key: "sources.provider_symbol", Value: 1}},
					Unique: true,
				},
			},
		},
		{
			Name:      "daily_bars",
			Validator: baseValidator([]string{"instrument_key", "market_key", "security_type", "symbol", "trading_date", "source"}),
			Indexes: []IndexSpec{
				{
					Name:   "daily_bars_instrument_source_date_unique",
					Keys:   bson.D{{Key: "instrument_key", Value: 1}, {Key: "source.provider", Value: 1}, {Key: "source.provider_group", Value: 1}, {Key: "source.operation", Value: 1}, {Key: "trading_date", Value: 1}},
					Unique: true,
				},
				{
					Name: "daily_bars_market_symbol_date",
					Keys: bson.D{{Key: "market_key", Value: 1}, {Key: "security_type", Value: 1}, {Key: "symbol", Value: 1}, {Key: "trading_date", Value: 1}},
				},
			},
		},
		{
			Name:      "indexes",
			Validator: baseValidator([]string{"market_key", "index_code", "sources"}),
			Indexes: []IndexSpec{
				{
					Name:   "indexes_market_code_unique",
					Keys:   bson.D{{Key: "market_key", Value: 1}, {Key: "index_code", Value: 1}},
					Unique: true,
				},
				{
					Name:   "indexes_source_provider_symbol_unique",
					Keys:   bson.D{{Key: "sources.provider", Value: 1}, {Key: "sources.provider_group", Value: 1}, {Key: "sources.operation", Value: 1}, {Key: "sources.provider_symbol", Value: 1}},
					Unique: true,
				},
			},
		},
		{
			Name:      "index_bars",
			Validator: baseValidator([]string{"index_key", "market_key", "index_code", "trading_date", "source"}),
			Indexes: []IndexSpec{
				{
					Name:   "index_bars_index_source_date_unique",
					Keys:   bson.D{{Key: "index_key", Value: 1}, {Key: "source.provider", Value: 1}, {Key: "source.provider_group", Value: 1}, {Key: "source.operation", Value: 1}, {Key: "trading_date", Value: 1}},
					Unique: true,
				},
				{
					Name: "index_bars_market_code_date",
					Keys: bson.D{{Key: "market_key", Value: 1}, {Key: "index_code", Value: 1}, {Key: "trading_date", Value: 1}},
				},
			},
		},
		{
			Name:      "compositions",
			Validator: baseValidator([]string{"subject_key", "subject", "source", "as_of_date", "observed_at_ms", "members"}),
			Indexes: []IndexSpec{
				{
					Name:   "compositions_subject_source_asof_observed_unique",
					Keys:   bson.D{{Key: "subject_key", Value: 1}, {Key: "source.provider", Value: 1}, {Key: "source.provider_group", Value: 1}, {Key: "source.operation", Value: 1}, {Key: "as_of_date", Value: 1}, {Key: "observed_at_ms", Value: 1}},
					Unique: true,
				},
				{
					Name: "compositions_subject_asof",
					Keys: bson.D{{Key: "subject.market", Value: 1}, {Key: "subject.security_type", Value: 1}, {Key: "subject.symbol", Value: 1}, {Key: "as_of_date", Value: -1}, {Key: "observed_at_ms", Value: -1}},
				},
				{
					Name: "compositions_member_instrument",
					Keys: bson.D{{Key: "members.instrument_key", Value: 1}},
				},
			},
		},
		{
			Name:      "provider_raw_snapshots",
			Validator: baseValidator([]string{"source", "payload"}),
			Indexes: []IndexSpec{
				{
					Name:   "provider_raw_snapshots_source_date_unique",
					Keys:   bson.D{{Key: "source.provider", Value: 1}, {Key: "source.provider_group", Value: 1}, {Key: "source.operation", Value: 1}, {Key: "source.base_date", Value: 1}},
					Unique: true,
				},
			},
		},
		{
			Name:      "macro_indicators",
			Validator: baseValidator([]string{"indicator_id", "provider", "source_code", "name", "sources"}),
			Indexes: []IndexSpec{
				{
					Name:   "macro_indicators_indicator_id_unique",
					Keys:   bson.D{{Key: "indicator_id", Value: 1}},
					Unique: true,
				},
				{
					Name:   "macro_indicators_provider_source_unique",
					Keys:   bson.D{{Key: "provider", Value: 1}, {Key: "source_code", Value: 1}},
					Unique: true,
				},
				{
					Name: "macro_indicators_preset_category",
					Keys: bson.D{{Key: "preset", Value: 1}, {Key: "category", Value: 1}, {Key: "indicator_id", Value: 1}},
				},
			},
		},
		{
			Name:      "macro_observations",
			Validator: baseValidator([]string{"indicator_id", "period", "observation_revision", "value", "collected_at"}),
			Indexes: []IndexSpec{
				{
					Name:   "macro_observations_indicator_period_revision_unique",
					Keys:   bson.D{{Key: "indicator_id", Value: 1}, {Key: "period", Value: 1}, {Key: "observation_revision", Value: 1}},
					Unique: true,
				},
				{
					Name: "macro_observations_indicator_period",
					Keys: bson.D{{Key: "indicator_id", Value: 1}, {Key: "period", Value: 1}},
				},
			},
		},
		{
			Name:      "companies",
			Validator: baseValidator([]string{"company_id", "name", "identifiers", "instruments"}),
			Indexes: []IndexSpec{
				{
					Name:   "companies_company_id_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}},
					Unique: true,
				},
				{
					Name:   "companies_identifier_unique",
					Keys:   bson.D{{Key: "identifiers.provider", Value: 1}, {Key: "identifiers.provider_group", Value: 1}, {Key: "identifiers.operation", Value: 1}, {Key: "identifiers.identifier_type", Value: 1}, {Key: "identifiers.identifier_value", Value: 1}, {Key: "identifiers.valid_from", Value: 1}},
					Unique: true,
				},
				{
					Name: "companies_name",
					Keys: bson.D{{Key: "name", Value: 1}, {Key: "legal_name", Value: 1}, {Key: "english_name", Value: 1}},
				},
				{
					Name: "companies_instrument_key",
					Keys: bson.D{{Key: "instruments.instrument_key", Value: 1}},
				},
			},
		},
		{
			Name:      "financial_statements",
			Validator: baseValidator([]string{"company_id", "provider", "provider_group", "operation", "fiscal_year", "statement_type", "line_items"}),
			Indexes: []IndexSpec{
				{
					Name:   "financial_statements_natural_key_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}, {Key: "instrument_id", Value: 1}, {Key: "provider", Value: 1}, {Key: "provider_group", Value: 1}, {Key: "operation", Value: 1}, {Key: "rcept_no", Value: 1}, {Key: "fiscal_year", Value: 1}, {Key: "report_code", Value: 1}, {Key: "fs_div", Value: 1}, {Key: "statement_type", Value: 1}},
					Unique: true,
				},
				{
					Name: "financial_statements_company_period",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}, {Key: "statement_type", Value: 1}},
				},
				{
					Name: "financial_statements_provider_identifier",
					Keys: bson.D{{Key: "provider_company_identifier_type", Value: 1}, {Key: "provider_company_identifier_value", Value: 1}},
				},
			},
		},
		{
			Name:      "financial_metrics",
			Validator: baseValidator([]string{"company_id", "metric", "fiscal_year", "formula_version", "provenance"}),
			Indexes: []IndexSpec{
				{
					Name:   "financial_metrics_natural_key_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}, {Key: "instrument_id", Value: 1}, {Key: "metric", Value: 1}, {Key: "fiscal_year", Value: 1}, {Key: "fiscal_period", Value: 1}, {Key: "as_of_date", Value: 1}, {Key: "formula_version", Value: 1}},
					Unique: true,
				},
				{
					Name: "financial_metrics_company_metric",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "metric", Value: 1}, {Key: "fiscal_year", Value: -1}},
				},
				{
					Name: "financial_metrics_instrument_metric",
					Keys: bson.D{{Key: "instrument_id", Value: 1}, {Key: "metric", Value: 1}, {Key: "as_of_date", Value: -1}},
				},
			},
		},
		{
			Name:      "valuation_snapshots",
			Validator: baseValidator([]string{"company_id", "instrument_id", "as_of_date", "source_price_date", "metric_source_version", "provenance", "uncomputable"}),
			Indexes: []IndexSpec{
				{
					Name:   "valuation_snapshots_natural_key_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}, {Key: "instrument_id", Value: 1}, {Key: "as_of_date", Value: 1}, {Key: "metric_source_version", Value: 1}},
					Unique: true,
				},
				{
					Name: "valuation_snapshots_company_asof",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "as_of_date", Value: -1}},
				},
				{
					Name: "valuation_snapshots_instrument_asof",
					Keys: bson.D{{Key: "instrument_id", Value: 1}, {Key: "as_of_date", Value: -1}},
				},
			},
		},
		{
			Name:      "company_facts",
			Validator: baseValidator([]string{"company_id", "provider", "provider_group", "operation", "fact_type", "key", "raw"}),
			Indexes: []IndexSpec{
				{
					Name:   "company_facts_natural_key_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}, {Key: "instrument_id", Value: 1}, {Key: "provider", Value: 1}, {Key: "provider_group", Value: 1}, {Key: "operation", Value: 1}, {Key: "fact_type", Value: 1}, {Key: "fiscal_year", Value: 1}, {Key: "report_code", Value: 1}, {Key: "rcept_no", Value: 1}, {Key: "key", Value: 1}},
					Unique: true,
				},
				{
					Name: "company_facts_company_type_period",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "fact_type", Value: 1}, {Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}, {Key: "fact_date", Value: -1}},
				},
				{
					Name: "company_facts_provider_identifier",
					Keys: bson.D{{Key: "provider_company_identifier_type", Value: 1}, {Key: "provider_company_identifier_value", Value: 1}},
				},
			},
		},
		{
			Name:      "company_events",
			Validator: baseValidator([]string{"company_id", "provider", "provider_group", "operation", "event_type", "rcept_no", "effective_date", "raw"}),
			Indexes: []IndexSpec{
				{
					Name:   "company_events_natural_key_unique",
					Keys:   bson.D{{Key: "company_id", Value: 1}, {Key: "instrument_id", Value: 1}, {Key: "provider", Value: 1}, {Key: "provider_group", Value: 1}, {Key: "operation", Value: 1}, {Key: "event_type", Value: 1}, {Key: "rcept_no", Value: 1}, {Key: "title", Value: 1}},
					Unique: true,
				},
				{
					Name: "company_events_company_effective_date",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "effective_date", Value: -1}, {Key: "rcept_no", Value: -1}},
				},
				{
					Name: "company_events_company_type_date",
					Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "event_type", Value: 1}, {Key: "effective_date", Value: -1}},
				},
			},
		},
		{
			Name:      "screen_strategies",
			Validator: baseValidator([]string{"strategy_id", "name", "engine", "active_version_id", "versions"}),
			Indexes: []IndexSpec{
				{
					Name:   "screen_strategies_name_unique",
					Keys:   bson.D{{Key: "name", Value: 1}},
					Unique: true,
				},
				{
					Name:   "screen_strategies_strategy_id_unique",
					Keys:   bson.D{{Key: "strategy_id", Value: 1}},
					Unique: true,
				},
				{
					Name: "screen_strategies_archived_at",
					Keys: bson.D{{Key: "archived_at", Value: 1}},
				},
			},
		},
		{
			Name:      "screen_runs",
			Validator: baseValidator([]string{"run_id", "strategy_id", "strategy_version_id", "status", "started_at"}),
			Indexes: []IndexSpec{
				{
					Name:   "screen_runs_run_id_unique",
					Keys:   bson.D{{Key: "run_id", Value: 1}},
					Unique: true,
				},
				{
					Name:   "screen_runs_alias_unique",
					Keys:   bson.D{{Key: "alias", Value: 1}},
					Unique: true,
					Sparse: true,
				},
				{
					Name: "screen_runs_started_at",
					Keys: bson.D{{Key: "started_at", Value: -1}},
				},
				{
					Name: "screen_runs_strategy_started",
					Keys: bson.D{{Key: "strategy_id", Value: 1}, {Key: "started_at", Value: -1}},
				},
			},
		},
		{
			Name:      "screen_run_items",
			Validator: baseValidator([]string{"run_id", "ordinal", "payload"}),
			Indexes: []IndexSpec{
				{
					Name:   "screen_run_items_run_ordinal_unique",
					Keys:   bson.D{{Key: "run_id", Value: 1}, {Key: "ordinal", Value: 1}},
					Unique: true,
				},
				{
					Name: "screen_run_items_symbol",
					Keys: bson.D{{Key: "symbol", Value: 1}},
				},
			},
		},
		{
			Name:      "backtest_strategies",
			Validator: baseValidator([]string{"strategy_id", "name", "active_version_id", "versions"}),
			Indexes: []IndexSpec{
				{
					Name:   "backtest_strategies_name_unique",
					Keys:   bson.D{{Key: "name", Value: 1}},
					Unique: true,
				},
				{
					Name:   "backtest_strategies_strategy_id_unique",
					Keys:   bson.D{{Key: "strategy_id", Value: 1}},
					Unique: true,
				},
				{
					Name: "backtest_strategies_deleted_at",
					Keys: bson.D{{Key: "deleted_at", Value: 1}},
				},
			},
		},
		{
			Name:      "backtest_runs",
			Validator: baseValidator([]string{"run_id", "run_name", "strategy_name", "run_hash", "result_hash", "result", "metrics"}),
			Indexes: []IndexSpec{
				{
					Name:   "backtest_runs_run_id_unique",
					Keys:   bson.D{{Key: "run_id", Value: 1}},
					Unique: true,
				},
				{
					Name:   "backtest_runs_run_hash_unique",
					Keys:   bson.D{{Key: "run_hash", Value: 1}},
					Unique: true,
				},
				{
					Name: "backtest_runs_result_hash",
					Keys: bson.D{{Key: "result_hash", Value: 1}},
				},
				{
					Name: "backtest_runs_run_name_created",
					Keys: bson.D{{Key: "run_name", Value: 1}, {Key: "created_at", Value: -1}},
				},
			},
		},
		{
			Name:      "backtest_experiments",
			Validator: baseValidator([]string{"experiment_id", "name", "strategy_name", "base_run_key", "spec_hash", "spec", "walk_forward_steps"}),
			Indexes: []IndexSpec{
				{
					Name:   "backtest_experiments_experiment_id_unique",
					Keys:   bson.D{{Key: "experiment_id", Value: 1}},
					Unique: true,
				},
				{
					Name: "backtest_experiments_name_created",
					Keys: bson.D{{Key: "name", Value: 1}, {Key: "created_at", Value: -1}},
				},
				{
					Name: "backtest_experiments_spec_hash",
					Keys: bson.D{{Key: "spec_hash", Value: 1}},
				},
			},
		},
		{
			Name:      "backtest_experiment_cases",
			Validator: baseValidator([]string{"experiment_id", "case_id", "run_name", "parameters", "result", "metrics"}),
			Indexes: []IndexSpec{
				{
					Name:   "backtest_experiment_cases_case_id_unique",
					Keys:   bson.D{{Key: "experiment_id", Value: 1}, {Key: "case_id", Value: 1}},
					Unique: true,
				},
				{
					Name: "backtest_experiment_cases_experiment_rank",
					Keys: bson.D{{Key: "experiment_id", Value: 1}, {Key: "rank", Value: 1}, {Key: "case_id", Value: 1}},
				},
				{
					Name: "backtest_experiment_cases_result_hash",
					Keys: bson.D{{Key: "result_hash", Value: 1}},
				},
			},
		},
	}
}

func EnsureCollections(ctx context.Context, db *mongo.Database) error {
	if db == nil {
		return oops.In("mongodb_indexes").New("mongodb database is nil")
	}
	errb := oops.In("mongodb_indexes")
	for _, spec := range CollectionSpecs() {
		if err := ensureCollection(ctx, db, spec); err != nil {
			return errb.With("collection", spec.Name).Wrap(err)
		}
		if err := ensureIndexes(ctx, db.Collection(spec.Name), spec); err != nil {
			return errb.With("collection", spec.Name).Wrap(err)
		}
	}
	return nil
}

func ensureCollection(ctx context.Context, db *mongo.Database, spec CollectionSpec) error {
	options := options.CreateCollection().
		SetValidator(spec.Validator).
		SetValidationAction("error").
		SetValidationLevel("strict")
	if err := db.CreateCollection(ctx, spec.Name, options); err != nil {
		if isNamespaceExists(err) {
			return updateCollectionValidator(ctx, db, spec)
		}
		return oops.In("mongodb_indexes").With("collection", spec.Name).Wrapf(err, "create mongodb collection")
	}
	return nil
}

func updateCollectionValidator(ctx context.Context, db *mongo.Database, spec CollectionSpec) error {
	command := bson.D{
		{Key: "collMod", Value: spec.Name},
		{Key: "validator", Value: spec.Validator},
		{Key: "validationAction", Value: "error"},
		{Key: "validationLevel", Value: "strict"},
	}
	if err := db.RunCommand(ctx, command).Err(); err != nil {
		return oops.In("mongodb_indexes").With("collection", spec.Name).Wrapf(err, "update mongodb collection validator")
	}
	return nil
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection, spec CollectionSpec) error {
	errb := oops.In("mongodb_indexes")
	for _, index := range spec.Indexes {
		opts := options.Index().SetName(index.Name)
		if index.Unique {
			opts.SetUnique(true)
		}
		if index.Sparse {
			opts.SetSparse(true)
		}
		model := mongo.IndexModel{
			Keys:    index.Keys,
			Options: opts,
		}
		if _, err := collection.Indexes().CreateOne(ctx, model); err != nil {
			return errb.With("collection", spec.Name, "index", index.Name).Wrapf(err, "create mongodb index")
		}
	}
	return nil
}

func baseValidator(required []string) bson.D {
	allRequired := append([]string{"_id", "schema_version", "revision", "created_at", "updated_at"}, required...)
	return bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "required", Value: allRequired},
			{Key: "properties", Value: bson.D{
				{Key: "_id", Value: bson.D{{Key: "bsonType", Value: []string{"string", "objectId"}}}},
				{Key: "schema_version", Value: bson.D{{Key: "bsonType", Value: "string"}}},
				{Key: "revision", Value: bson.D{{Key: "bsonType", Value: []string{"int", "long"}}}},
				{Key: "created_at", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				{Key: "updated_at", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				{Key: "collected_at", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				{Key: "source_updated_at", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				{Key: "deleted_at", Value: bson.D{{Key: "bsonType", Value: "date"}}},
			}},
		}},
	}
}

func isNamespaceExists(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) && commandError.HasErrorCode(48)
}
