package backtest

import (
	"slices"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	KindStrategy    = "Strategy"
	KindBacktestRun = "BacktestRun"

	FillNextOpen = "next_open"

	SizingPercentOfEquity = "percent_of_equity"
)

func Compile(strategy StrategySpec, run BacktestRunSpec, registry IndicatorRegistry) (StrategyPlan, error) {
	errb := oops.In("backtest_compiler").With("strategy", strategy.Name, "run", run.Name)
	metricRegistry, err := DefaultMetricRegistry()
	if err != nil {
		return StrategyPlan{}, errb.Wrapf(err, "create metric registry")
	}
	if err := validateStrategy(strategy, registry); err != nil {
		return StrategyPlan{}, errb.Wrap(err)
	}
	selectedMetrics, err := validateRun(run, strategy.Name, metricRegistry)
	if err != nil {
		return StrategyPlan{}, errb.Wrap(err)
	}

	from, err := time.Parse(time.DateOnly, run.Data.From)
	if err != nil {
		return StrategyPlan{}, errb.With("from", run.Data.From).Wrapf(err, "parse backtest from date")
	}
	to, err := time.Parse(time.DateOnly, run.Data.To)
	if err != nil {
		return StrategyPlan{}, errb.With("to", run.Data.To).Wrapf(err, "parse backtest to date")
	}
	if to.Before(from) {
		return StrategyPlan{}, errb.With("from", run.Data.From, "to", run.Data.To).New("backtest to date must be on or after from date")
	}

	normalizedUniverse := NormalizeUniverseSpecWithData(run.Universe, run.Data)
	universePlan, err := CompileUniverseSpec(normalizedUniverse, UniverseDataWindow{
		Market: run.Data.Market,
		From:   from,
		To:     to,
	}, DefaultUniverseSelectorRegistry())
	if err != nil {
		return StrategyPlan{}, errb.Wrap(err)
	}
	symbols := append([]string(nil), universePlan.StaticSymbols...)
	slices.Sort(symbols)
	benchmark := run.Benchmark
	if benchmark.Market == "" {
		benchmark.Market = run.Data.Market
	}
	if benchmark.SecurityType == "" {
		benchmark.SecurityType = run.Data.SecurityType
	}

	return StrategyPlan{
		StrategyName:    strategy.Name,
		RunName:         run.Name,
		Symbols:         symbols,
		Instruments:     instrumentsFromStaticSymbols(symbols, run.Data),
		From:            from,
		To:              to,
		Timeframe:       run.Data.Timeframe,
		Market:          run.Data.Market,
		Benchmark:       benchmark,
		InitialCash:     run.Portfolio.InitialCash,
		Currency:        withDefault(run.Portfolio.Currency, "KRW"),
		Fill:            run.Execution.Fill,
		Commission:      run.Execution.Commission,
		Slippage:        run.Execution.Slippage,
		Indicators:      cloneIndicators(strategy.Indicators),
		Entry:           strategy.Entry,
		Exit:            strategy.Exit,
		Sizing:          strategy.Sizing,
		Risk:            strategy.Risk,
		Report:          run.Report,
		SelectedMetrics: selectedMetrics,
		Universe:        universePlan,
		metricRegistry:  metricRegistry,
		registry:        registry,
	}, nil
}

func ValidateStrategySpec(strategy StrategySpec, registry IndicatorRegistry) error {
	return validateStrategy(strategy, registry)
}

func validateStrategy(strategy StrategySpec, registry IndicatorRegistry) error {
	errb := oops.In("backtest_strategy_spec").With("strategy", strategy.Name)
	if strategy.Kind != KindStrategy {
		return errb.With("kind", strategy.Kind).New("strategy kind must be Strategy")
	}
	if strategy.SchemaVersion != SchemaVersion {
		return errb.With("schema_version", strategy.SchemaVersion).New("unsupported strategy schema version")
	}
	if strings.TrimSpace(strategy.Name) == "" {
		return errb.New("strategy name is required")
	}
	if strategy.Sizing.Type != SizingPercentOfEquity {
		return errb.With("sizing_type", strategy.Sizing.Type).New("unsupported sizing type")
	}
	if strategy.Sizing.Value <= 0 {
		return errb.With("sizing_value", strategy.Sizing.Value).New("sizing value must be positive")
	}
	for alias, indicator := range strategy.Indicators {
		if strings.TrimSpace(alias) == "" {
			return errb.New("indicator alias is empty")
		}
		if err := validateIndicator(indicator, registry); err != nil {
			return errb.With("indicator_alias", alias).Wrap(err)
		}
	}
	if err := validateRule(strategy.Entry, strategy.Indicators, registry); err != nil {
		return errb.Wrapf(err, "validate entry rule")
	}
	if err := validateRule(strategy.Exit, strategy.Indicators, registry); err != nil {
		return errb.Wrapf(err, "validate exit rule")
	}
	return nil
}

func validateRun(run BacktestRunSpec, strategyName string, metricRegistry MetricRegistry) ([]string, error) {
	errb := oops.In("backtest_run_spec").With("run", run.Name)
	if run.Kind != KindBacktestRun {
		return nil, errb.With("kind", run.Kind).New("backtest run kind must be BacktestRun")
	}
	if run.SchemaVersion != SchemaVersion {
		return nil, errb.With("schema_version", run.SchemaVersion).New("unsupported backtest run schema version")
	}
	if strings.TrimSpace(run.Name) == "" {
		return nil, errb.New("backtest run name is required")
	}
	if run.Strategy.Name != strategyName {
		return nil, errb.With("strategy_name", run.Strategy.Name, "expected_strategy_name", strategyName).New("backtest run strategy reference does not match strategy")
	}
	if len(run.Universe.Symbols) == 0 && len(run.Universe.Pipeline) == 0 {
		return nil, errb.New("backtest run universe requires symbols or pipeline")
	}
	if run.Portfolio.InitialCash <= 0 {
		return nil, errb.With("initial_cash", run.Portfolio.InitialCash).New("initial cash must be positive")
	}
	if run.Execution.Fill != FillNextOpen {
		return nil, errb.With("fill", run.Execution.Fill).New("unsupported execution fill")
	}
	if run.Execution.Commission.Type != "" && run.Execution.Commission.Type != "bps" {
		return nil, errb.With("commission_type", run.Execution.Commission.Type).New("unsupported commission type")
	}
	if run.Execution.Slippage.Type != "" && run.Execution.Slippage.Type != "bps" {
		return nil, errb.With("slippage_type", run.Execution.Slippage.Type).New("unsupported slippage type")
	}
	if strings.TrimSpace(run.Data.Market) == "" {
		return nil, errb.New("data market is required")
	}
	if run.Data.Timeframe != "1d" {
		return nil, errb.With("timeframe", run.Data.Timeframe).New("only 1d timeframe is supported")
	}
	if strings.TrimSpace(run.Benchmark.Name) != "" && strings.TrimSpace(run.Benchmark.Symbol) == "" {
		return nil, errb.New("benchmark symbol is required when benchmark is configured")
	}
	selectedMetrics, err := resolveMetricSelection(run.Report, strings.TrimSpace(run.Benchmark.Symbol) != "", metricRegistry)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	return selectedMetrics, nil
}

func validateRule(rule RuleExpr, indicators map[string]IndicatorSpec, registry IndicatorRegistry) error {
	errb := oops.In("backtest_rule").With("operator", rule.Operator)
	switch rule.Operator {
	case "all", "any":
		if len(rule.Rules) == 0 {
			return errb.New("logical rule requires child rules")
		}
		for _, child := range rule.Rules {
			if err := validateRule(child, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "not":
		if rule.Rule == nil {
			return errb.New("not rule requires a child rule")
		}
		return validateRule(*rule.Rule, indicators, registry)
	case "gt", "gte", "lt", "lte", "eq", "crosses_above", "crosses_below":
		if len(rule.Args) != 2 {
			return errb.New("comparison rule requires two args")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	default:
		return errb.New("unsupported rule operator")
	}
	return nil
}

func validateValue(expr ValueExpr, indicators map[string]IndicatorSpec, registry IndicatorRegistry) error {
	errb := oops.In("backtest_value_expr").With("kind", expr.Kind)
	switch expr.Kind {
	case "price":
		if !isPriceField(expr.Price) {
			return errb.With("price", expr.Price).New("unsupported price field")
		}
	case "value":
	case "ref":
		if _, ok := indicators[expr.Ref]; !ok {
			return errb.With("ref", expr.Ref).New("indicator ref is unknown")
		}
	case "indicator":
		if expr.Indicator == nil {
			return errb.New("indicator expression requires indicator spec")
		}
		return validateIndicator(*expr.Indicator, registry)
	default:
		return errb.New("unsupported value expression kind")
	}
	return nil
}

func validateIndicator(indicator IndicatorSpec, registry IndicatorRegistry) error {
	errb := oops.In("backtest_indicator").With("indicator", indicator.ID)
	definition, ok := registry.Definition(indicator.ID)
	if !ok {
		return errb.New("indicator is not registered")
	}
	if definition.Validate != nil {
		return definition.Validate(indicator)
	}
	return nil
}

func cloneIndicators(in map[string]IndicatorSpec) map[string]IndicatorSpec {
	out := make(map[string]IndicatorSpec, len(in))
	for key, value := range in {
		params := make(map[string]float64, len(value.Params))
		for paramKey, paramValue := range value.Params {
			params[paramKey] = paramValue
		}
		value.Params = params
		out[key] = value
	}
	return out
}

func withDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
