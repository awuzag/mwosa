package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionSpecsIncludePriorityMongoDBModels(t *testing.T) {
	specs := CollectionSpecs()

	require.Contains(t, collectionNames(specs), "storage_metadata")
	require.Contains(t, collectionNames(specs), "instruments")
	require.Contains(t, collectionNames(specs), "daily_bars")
	require.Contains(t, collectionNames(specs), "indexes")
	require.Contains(t, collectionNames(specs), "index_bars")
	require.Contains(t, collectionNames(specs), "compositions")
	require.Contains(t, collectionNames(specs), "provider_raw_snapshots")
	require.Contains(t, collectionNames(specs), "macro_indicators")
	require.Contains(t, collectionNames(specs), "macro_observations")
	require.Contains(t, collectionNames(specs), "companies")
	require.Contains(t, collectionNames(specs), "financial_statements")
	require.Contains(t, collectionNames(specs), "financial_metrics")
	require.Contains(t, collectionNames(specs), "valuation_snapshots")
	require.Contains(t, collectionNames(specs), "company_facts")
	require.Contains(t, collectionNames(specs), "company_events")
	require.Contains(t, collectionNames(specs), "screen_strategies")
	require.Contains(t, collectionNames(specs), "screen_runs")
	require.Contains(t, collectionNames(specs), "screen_run_items")
	require.Contains(t, collectionNames(specs), "backtest_strategies")
	require.Contains(t, collectionNames(specs), "backtest_runs")
	require.Contains(t, collectionNames(specs), "backtest_experiments")
	require.Contains(t, collectionNames(specs), "backtest_experiment_cases")

	instruments := requireCollectionSpec(t, specs, "instruments")
	require.NotEmpty(t, instruments.Validator)
	require.True(t, hasUniqueIndex(instruments, "instruments_instrument_key_unique"))
	require.True(t, hasUniqueIndex(instruments, "instruments_source_provider_symbol_unique"))

	dailyBars := requireCollectionSpec(t, specs, "daily_bars")
	require.NotEmpty(t, dailyBars.Validator)
	require.True(t, hasUniqueIndex(dailyBars, "daily_bars_instrument_source_date_unique"))

	indexes := requireCollectionSpec(t, specs, "indexes")
	require.NotEmpty(t, indexes.Validator)
	require.True(t, hasUniqueIndex(indexes, "indexes_market_code_unique"))
	require.True(t, hasUniqueIndex(indexes, "indexes_source_provider_symbol_unique"))

	indexBars := requireCollectionSpec(t, specs, "index_bars")
	require.NotEmpty(t, indexBars.Validator)
	require.True(t, hasUniqueIndex(indexBars, "index_bars_index_source_date_unique"))

	compositions := requireCollectionSpec(t, specs, "compositions")
	require.NotEmpty(t, compositions.Validator)
	require.True(t, hasUniqueIndex(compositions, "compositions_subject_source_asof_observed_unique"))

	rawSnapshots := requireCollectionSpec(t, specs, "provider_raw_snapshots")
	require.NotEmpty(t, rawSnapshots.Validator)
	require.True(t, hasUniqueIndex(rawSnapshots, "provider_raw_snapshots_source_date_unique"))

	macroIndicators := requireCollectionSpec(t, specs, "macro_indicators")
	require.NotEmpty(t, macroIndicators.Validator)
	require.True(t, hasUniqueIndex(macroIndicators, "macro_indicators_indicator_id_unique"))
	require.True(t, hasUniqueIndex(macroIndicators, "macro_indicators_provider_source_unique"))

	macroObservations := requireCollectionSpec(t, specs, "macro_observations")
	require.NotEmpty(t, macroObservations.Validator)
	require.True(t, hasUniqueIndex(macroObservations, "macro_observations_indicator_period_revision_unique"))

	companies := requireCollectionSpec(t, specs, "companies")
	require.NotEmpty(t, companies.Validator)
	require.True(t, hasUniqueIndex(companies, "companies_company_id_unique"))
	require.True(t, hasUniqueIndex(companies, "companies_identifier_unique"))

	financialStatements := requireCollectionSpec(t, specs, "financial_statements")
	require.NotEmpty(t, financialStatements.Validator)
	require.True(t, hasUniqueIndex(financialStatements, "financial_statements_natural_key_unique"))

	financialMetrics := requireCollectionSpec(t, specs, "financial_metrics")
	require.NotEmpty(t, financialMetrics.Validator)
	require.True(t, hasUniqueIndex(financialMetrics, "financial_metrics_natural_key_unique"))

	valuationSnapshots := requireCollectionSpec(t, specs, "valuation_snapshots")
	require.NotEmpty(t, valuationSnapshots.Validator)
	require.True(t, hasUniqueIndex(valuationSnapshots, "valuation_snapshots_natural_key_unique"))

	companyFacts := requireCollectionSpec(t, specs, "company_facts")
	require.NotEmpty(t, companyFacts.Validator)
	require.True(t, hasUniqueIndex(companyFacts, "company_facts_natural_key_unique"))

	companyEvents := requireCollectionSpec(t, specs, "company_events")
	require.NotEmpty(t, companyEvents.Validator)
	require.True(t, hasUniqueIndex(companyEvents, "company_events_natural_key_unique"))

	screenStrategies := requireCollectionSpec(t, specs, "screen_strategies")
	require.NotEmpty(t, screenStrategies.Validator)
	require.True(t, hasUniqueIndex(screenStrategies, "screen_strategies_name_unique"))
	require.True(t, hasUniqueIndex(screenStrategies, "screen_strategies_strategy_id_unique"))

	screenRuns := requireCollectionSpec(t, specs, "screen_runs")
	require.NotEmpty(t, screenRuns.Validator)
	require.True(t, hasUniqueIndex(screenRuns, "screen_runs_run_id_unique"))

	screenRunItems := requireCollectionSpec(t, specs, "screen_run_items")
	require.NotEmpty(t, screenRunItems.Validator)
	require.True(t, hasUniqueIndex(screenRunItems, "screen_run_items_run_ordinal_unique"))

	backtestStrategies := requireCollectionSpec(t, specs, "backtest_strategies")
	require.NotEmpty(t, backtestStrategies.Validator)
	require.True(t, hasUniqueIndex(backtestStrategies, "backtest_strategies_name_unique"))
	require.True(t, hasUniqueIndex(backtestStrategies, "backtest_strategies_strategy_id_unique"))

	backtestRuns := requireCollectionSpec(t, specs, "backtest_runs")
	require.NotEmpty(t, backtestRuns.Validator)
	require.True(t, hasUniqueIndex(backtestRuns, "backtest_runs_run_id_unique"))
	require.True(t, hasUniqueIndex(backtestRuns, "backtest_runs_run_hash_unique"))

	backtestExperiments := requireCollectionSpec(t, specs, "backtest_experiments")
	require.NotEmpty(t, backtestExperiments.Validator)
	require.True(t, hasUniqueIndex(backtestExperiments, "backtest_experiments_experiment_id_unique"))

	backtestExperimentCases := requireCollectionSpec(t, specs, "backtest_experiment_cases")
	require.NotEmpty(t, backtestExperimentCases.Validator)
	require.True(t, hasUniqueIndex(backtestExperimentCases, "backtest_experiment_cases_case_id_unique"))
}

func collectionNames(specs []CollectionSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	return names
}

func requireCollectionSpec(t *testing.T, specs []CollectionSpec, name string) CollectionSpec {
	t.Helper()

	for _, spec := range specs {
		if spec.Name == name {
			return spec
		}
	}
	t.Fatalf("collection spec %q was not found", name)
	return CollectionSpec{}
}

func hasUniqueIndex(spec CollectionSpec, name string) bool {
	for _, index := range spec.Indexes {
		if index.Name == name && index.Unique {
			return true
		}
	}
	return false
}
