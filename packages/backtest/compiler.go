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
	KindEvaluation  = "Evaluation"

	FillSameClose    = "same_close"
	FillNextOpen     = "next_open"
	FillNextClose    = "next_close"
	FillIntrabarOHLC = "intrabar_ohlc"

	OrderTypeMarket       = "market"
	OrderTypeLimit        = "limit"
	OrderTypeStop         = "stop"
	OrderTypeStopLimit    = "stop_limit"
	OrderTypeTrailingStop = "trailing_stop"
	OrderTypeRebalance    = "rebalance"

	IntrabarAmbiguityConservative     = "conservative"
	IntrabarAmbiguityOptimistic       = "optimistic"
	IntrabarAmbiguityOpenHighLowClose = "open_high_low_close"
	IntrabarAmbiguityOpenLowHighClose = "open_low_high_close"
	TimeInForceDay                    = "day"
	TimeInForceGTC                    = "gtc"
	TimeInForceIOC                    = "ioc"
	TimeInForceCancelOnRebalance      = "cancel_on_rebalance"
	PartialFillCarry                  = "carry"
	PartialFillCancel                 = "cancel"
	PartialFillExpireAfterNBars       = "expire_after_n_bars"

	CostTypeNone          = "none"
	CostTypeBPS           = "bps"
	CostTypeFixedBPS      = "fixed_bps"
	CostTypeFixedAmount   = "fixed_amount"
	CostTypeFixedPerUnit  = "fixed_per_unit"
	CostTypeATR           = "atr"
	CostTypeSpreadProxy   = "spread_proxy"
	CostTypeParticipation = "participation"
	CostTypeVolatility    = "volatility"

	SizingPercentOfEquity = "percent_of_equity"
)

func Compile(strategy StrategySpec, run BacktestRunSpec, registry IndicatorRegistry) (StrategyPlan, error) {
	strategy = NormalizeStrategySpec(strategy)
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
		StrategyName:            strategy.Name,
		RunName:                 run.Name,
		Symbols:                 symbols,
		Instruments:             instrumentsFromStaticSymbols(symbols, run.Data),
		From:                    from,
		To:                      to,
		Timeframe:               run.Data.Timeframe,
		Market:                  run.Data.Market,
		Benchmark:               benchmark,
		InitialCash:             run.Portfolio.InitialCash,
		Currency:                withDefault(run.Portfolio.Currency, "KRW"),
		Fill:                    run.Execution.Fill,
		OrderType:               withDefault(run.Execution.OrderType, OrderTypeMarket),
		LimitPrice:              run.Execution.LimitPrice,
		StopPrice:               run.Execution.StopPrice,
		TrailingStopPct:         run.Execution.TrailingStopPct,
		IntrabarAmbiguityPolicy: withDefault(run.Execution.IntrabarAmbiguityPolicy, IntrabarAmbiguityConservative),
		TimeInForce:             withDefault(run.Execution.TimeInForce, TimeInForceDay),
		LotSize:                 run.Execution.LotSize,
		TickSize:                run.Execution.TickSize,
		Commission:              run.Execution.Commission,
		Tax:                     run.Execution.Tax,
		ExchangeFee:             run.Execution.ExchangeFee,
		Slippage:                run.Execution.Slippage,
		Liquidity:               run.Execution.Liquidity,
		PartialFill:             normalizePartialFill(run.Execution.PartialFill),
		Indicators:              cloneIndicators(strategy.Indicators),
		Entry:                   strategy.Entry,
		Exit:                    strategy.Exit,
		Entries:                 cloneRules(strategy.Entries),
		Exits:                   cloneRules(strategy.Exits),
		Rebalance:               cloneRules(strategy.Rebalance),
		Stops:                   cloneRules(strategy.Stops),
		Sizing:                  strategy.Sizing,
		Risk:                    strategy.Risk,
		Report:                  run.Report,
		SelectedMetrics:         selectedMetrics,
		Universe:                universePlan,
		metricRegistry:          metricRegistry,
		registry:                registry,
	}, nil
}

func ValidateStrategySpec(strategy StrategySpec, registry IndicatorRegistry) error {
	strategy = NormalizeStrategySpec(strategy)
	return validateStrategy(strategy, registry)
}

func NormalizeStrategySpec(strategy StrategySpec) StrategySpec {
	if len(strategy.Entries) == 0 && !strategy.Entry.Empty() {
		strategy.Entries = []RuleExpr{strategy.Entry}
	}
	if len(strategy.Exits) == 0 && !strategy.Exit.Empty() {
		strategy.Exits = []RuleExpr{strategy.Exit}
	}
	if strategy.Entry.Empty() {
		strategy.Entry = combineRoleRules(strategy.Entries)
	}
	if strategy.Exit.Empty() {
		strategy.Exit = combineRoleRules(strategy.Exits)
	}
	return strategy
}

func combineRoleRules(rules []RuleExpr) RuleExpr {
	switch len(rules) {
	case 0:
		return RuleExpr{}
	case 1:
		return rules[0]
	default:
		return RuleExpr{Operator: "any", Rules: cloneRules(rules)}
	}
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
	if len(strategy.Entries) == 0 {
		return errb.New("strategy requires entry or entries")
	}
	if err := validateRuleList("entry", strategy.Entries, strategy.Indicators, registry); err != nil {
		return errb.Wrap(err)
	}
	if err := validateRuleList("exit", strategy.Exits, strategy.Indicators, registry); err != nil {
		return errb.Wrap(err)
	}
	if err := validateRuleList("rebalance", strategy.Rebalance, strategy.Indicators, registry); err != nil {
		return errb.Wrap(err)
	}
	if err := validateRuleList("stop", strategy.Stops, strategy.Indicators, registry); err != nil {
		return errb.Wrap(err)
	}
	return nil
}

func validateRuleList(role string, rules []RuleExpr, indicators map[string]IndicatorSpec, registry IndicatorRegistry) error {
	for index, rule := range rules {
		if rule.Empty() {
			return oops.In("backtest_rule").With("role", role, "index", index).New("strategy rule is empty")
		}
		if err := validateRule(rule, indicators, registry); err != nil {
			return oops.In("backtest_rule").With("role", role, "index", index).Wrap(err)
		}
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
	switch run.Execution.Fill {
	case FillSameClose, FillNextOpen, FillNextClose, FillIntrabarOHLC:
	default:
		return nil, errb.With("fill", run.Execution.Fill).New("unsupported execution fill")
	}
	orderType := withDefault(run.Execution.OrderType, OrderTypeMarket)
	switch orderType {
	case OrderTypeMarket, OrderTypeRebalance:
	case OrderTypeLimit:
		if run.Execution.LimitPrice <= 0 {
			return nil, errb.With("limit_price", run.Execution.LimitPrice).New("limit order requires positive limit_price")
		}
	case OrderTypeStop:
		if run.Execution.StopPrice <= 0 {
			return nil, errb.With("stop_price", run.Execution.StopPrice).New("stop order requires positive stop_price")
		}
	case OrderTypeStopLimit:
		if run.Execution.StopPrice <= 0 {
			return nil, errb.With("stop_price", run.Execution.StopPrice).New("stop-limit order requires positive stop_price")
		}
		if run.Execution.LimitPrice <= 0 {
			return nil, errb.With("limit_price", run.Execution.LimitPrice).New("stop-limit order requires positive limit_price")
		}
	case OrderTypeTrailingStop:
		if run.Execution.TrailingStopPct <= 0 || run.Execution.TrailingStopPct >= 100 {
			return nil, errb.With("trailing_stop_pct", run.Execution.TrailingStopPct).New("trailing stop order requires trailing_stop_pct between 0 and 100")
		}
	default:
		return nil, errb.With("order_type", run.Execution.OrderType).New("unsupported execution order type")
	}
	intrabarPolicy := withDefault(run.Execution.IntrabarAmbiguityPolicy, IntrabarAmbiguityConservative)
	switch intrabarPolicy {
	case IntrabarAmbiguityConservative, IntrabarAmbiguityOptimistic, IntrabarAmbiguityOpenHighLowClose, IntrabarAmbiguityOpenLowHighClose:
	default:
		return nil, errb.With("intrabar_ambiguity_policy", run.Execution.IntrabarAmbiguityPolicy).New("unsupported intrabar ambiguity policy")
	}
	timeInForce := withDefault(run.Execution.TimeInForce, TimeInForceDay)
	switch timeInForce {
	case TimeInForceDay, TimeInForceGTC, TimeInForceIOC, TimeInForceCancelOnRebalance:
	default:
		return nil, errb.With("time_in_force", run.Execution.TimeInForce).New("unsupported time in force")
	}
	if err := validateCostSpec("commission", run.Execution.Commission); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := validateCostSpec("tax", run.Execution.Tax); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := validateCostSpec("exchange_fee", run.Execution.ExchangeFee); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := validateSlippageSpec(run.Execution.Slippage); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := validateLiquidity(run.Execution.Liquidity); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := validatePartialFill(run.Execution.PartialFill); err != nil {
		return nil, errb.Wrap(err)
	}
	if run.Execution.LotSize < 0 {
		return nil, errb.With("lot_size", run.Execution.LotSize).New("lot_size must not be negative")
	}
	if run.Execution.TickSize < 0 {
		return nil, errb.With("tick_size", run.Execution.TickSize).New("tick_size must not be negative")
	}
	if strings.TrimSpace(run.Data.Market) == "" {
		return nil, errb.New("data market is required")
	}
	if _, err := ParseTimeframe(run.Data.Timeframe); err != nil {
		return nil, errb.Wrap(err)
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

func validateCostSpec(name string, spec CostSpec) error {
	errb := oops.In("backtest_execution_spec").With("cost", name)
	switch withDefault(spec.Type, CostTypeBPS) {
	case CostTypeNone, CostTypeBPS, CostTypeFixedBPS, CostTypeFixedAmount, CostTypeFixedPerUnit:
	default:
		return errb.With("type", spec.Type).New("unsupported cost type")
	}
	if spec.Value < 0 || spec.BuyValue < 0 || spec.SellValue < 0 || spec.MinFee < 0 {
		return errb.With("value", spec.Value, "buy_value", spec.BuyValue, "sell_value", spec.SellValue, "min_fee", spec.MinFee).New("cost values must not be negative")
	}
	return nil
}

func validateSlippageSpec(spec CostSpec) error {
	errb := oops.In("backtest_execution_spec")
	switch withDefault(spec.Type, CostTypeBPS) {
	case CostTypeNone, CostTypeBPS, CostTypeFixedBPS, CostTypeFixedAmount, CostTypeSpreadProxy, CostTypeParticipation, CostTypeVolatility:
	case CostTypeATR:
		if spec.Window <= 0 {
			return errb.With("window", spec.Window).New("atr slippage requires positive window")
		}
	default:
		return errb.With("slippage_type", spec.Type).New("unsupported slippage type")
	}
	if spec.Value < 0 || spec.BuyValue < 0 || spec.SellValue < 0 {
		return errb.With("value", spec.Value, "buy_value", spec.BuyValue, "sell_value", spec.SellValue).New("slippage values must not be negative")
	}
	if spec.MinFee != 0 {
		return errb.With("min_fee", spec.MinFee).New("slippage min_fee is not supported")
	}
	return nil
}

func validateLiquidity(spec LiquiditySpec) error {
	errb := oops.In("backtest_execution_spec")
	if spec.MaxParticipationRate < 0 || spec.MaxParticipationRate > 1 {
		return errb.With("max_participation_rate", spec.MaxParticipationRate).New("max participation rate must be between 0 and 1")
	}
	if spec.VolumeCap < 0 {
		return errb.With("volume_cap", spec.VolumeCap).New("volume cap must not be negative")
	}
	if spec.TradedAmountCap < 0 {
		return errb.With("traded_amount_cap", spec.TradedAmountCap).New("traded amount cap must not be negative")
	}
	return nil
}

func validatePartialFill(spec PartialFillSpec) error {
	errb := oops.In("backtest_execution_spec")
	policy := withDefault(spec.Policy, PartialFillCarry)
	switch policy {
	case PartialFillCarry, PartialFillCancel, PartialFillExpireAfterNBars:
	default:
		return errb.With("partial_fill_policy", spec.Policy).New("unsupported partial fill policy")
	}
	if spec.ExpireAfterNBars < 0 {
		return errb.With("expire_after_n_bars", spec.ExpireAfterNBars).New("expire_after_n_bars must not be negative")
	}
	return nil
}

func normalizePartialFill(spec PartialFillSpec) PartialFillSpec {
	spec.Policy = withDefault(spec.Policy, PartialFillCarry)
	return spec
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
	case "between", "slope":
		if len(rule.Args) != 3 {
			return errb.New("rule requires three args")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "rising", "falling":
		if len(rule.Args) != 2 {
			return errb.New("trend rule requires two args")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "for_n_bars", "bars_since", "cooldown":
		if rule.Rule == nil {
			return errb.New("temporal rule requires child rule")
		}
		if len(rule.Args) != 1 {
			return errb.New("temporal rule requires one bar count arg")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
		return validateRule(*rule.Rule, indicators, registry)
	case "changed":
		if len(rule.Args) != 1 {
			return errb.New("changed rule requires one arg")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "new_high", "new_low":
		if len(rule.Args) != 2 {
			return errb.New("new extreme rule requires two args")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "position_exists":
		if len(rule.Args) != 0 || len(rule.Rules) != 0 || rule.Rule != nil {
			return errb.New("position_exists rule does not accept args or children")
		}
	case "weekly", "monthly", "first_trading_day":
		if len(rule.Args) != 0 || len(rule.Rules) != 0 || rule.Rule != nil {
			return errb.New("calendar rule does not accept args or children")
		}
	case "target_weight_changed":
		if len(rule.Args) > 1 {
			return errb.New("target_weight_changed rule accepts at most one arg")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "stop_loss", "take_profit", "time_stop", "trailing_stop":
		if len(rule.Args) != 1 {
			return errb.New("stop rule requires one threshold arg")
		}
		for _, arg := range rule.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "volatility_stop":
		if len(rule.Args) != 2 {
			return errb.New("volatility_stop rule requires volatility and multiplier args")
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
	if err := validateValueTimeframe(expr); err != nil {
		return errb.Wrap(err)
	}
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
	case "position":
		switch expr.Position {
		case "holding_bars", "unrealized_return", "drawdown_from_entry":
		default:
			return errb.With("position", expr.Position).New("unsupported position value")
		}
	case "portfolio":
		switch expr.Portfolio {
		case "cash_pct", "exposure_pct", "position_count", "portfolio_drawdown":
		default:
			return errb.With("portfolio", expr.Portfolio).New("unsupported portfolio value")
		}
	case "indicator":
		if expr.Indicator == nil {
			return errb.New("indicator expression requires indicator spec")
		}
		return validateIndicator(*expr.Indicator, registry)
	case "rank", "percentile", "universe_rank", "relative_strength":
		if len(expr.Args) != 1 {
			return errb.With("arg_count", len(expr.Args)).New("cross-sectional value expression requires one arg")
		}
		if containsCrossSectionalValue(expr.Args[0]) {
			return errb.New("cross-sectional value expression cannot be nested")
		}
		for _, arg := range expr.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "spread", "ratio":
		if len(expr.Args) != 2 {
			return errb.With("arg_count", len(expr.Args)).New("pair value expression requires two args")
		}
		for _, arg := range expr.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "add", "sub", "mul", "div", "min", "max":
		if len(expr.Args) < 2 {
			return errb.With("arg_count", len(expr.Args)).New("arithmetic value expression requires at least two args")
		}
		for _, arg := range expr.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	case "abs":
		if len(expr.Args) != 1 {
			return errb.With("arg_count", len(expr.Args)).New("abs value expression requires one arg")
		}
		for _, arg := range expr.Args {
			if err := validateValue(arg, indicators, registry); err != nil {
				return errb.Wrap(err)
			}
		}
	default:
		return errb.New("unsupported value expression kind")
	}
	return nil
}

func validateValueTimeframe(expr ValueExpr) error {
	timeframe := strings.TrimSpace(expr.Timeframe)
	if timeframe == "" {
		return nil
	}
	errb := oops.In("backtest_value_expr").With("kind", expr.Kind, "timeframe", expr.Timeframe)
	if _, err := ParseTimeframe(timeframe); err != nil {
		return errb.Wrap(err)
	}
	switch {
	case expr.Kind == "position" || expr.Kind == "portfolio":
		return errb.New("position and portfolio value expressions cannot be timeframe scoped")
	case isCrossSectionalValueKind(expr.Kind):
		return errb.New("cross-sectional value expressions cannot be timeframe scoped")
	default:
		return nil
	}
}

func isCrossSectionalValueKind(kind string) bool {
	switch kind {
	case "rank", "percentile", "universe_rank", "relative_strength":
		return true
	default:
		return false
	}
}

func containsCrossSectionalValue(expr ValueExpr) bool {
	if isCrossSectionalValueKind(expr.Kind) {
		return true
	}
	for _, arg := range expr.Args {
		if containsCrossSectionalValue(arg) {
			return true
		}
	}
	return false
}

func validateIndicator(indicator IndicatorSpec, registry IndicatorRegistry) error {
	errb := oops.In("backtest_indicator").With("indicator", indicator.ID)
	if !indicator.Source.Empty() {
		if err := validateValueTimeframe(indicator.Source); err != nil {
			return errb.With("field", "source").Wrap(err)
		}
	}
	if !indicator.Compare.Empty() {
		if err := validateValueTimeframe(indicator.Compare); err != nil {
			return errb.With("field", "compare").Wrap(err)
		}
	}
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
		value.Compare = cloneValue(value.Compare)
		out[key] = value
	}
	return out
}

func cloneRules(in []RuleExpr) []RuleExpr {
	out := make([]RuleExpr, 0, len(in))
	for _, rule := range in {
		out = append(out, cloneRule(rule))
	}
	return out
}

func cloneRule(in RuleExpr) RuleExpr {
	out := RuleExpr{
		Operator: in.Operator,
		Args:     cloneValues(in.Args),
	}
	if len(in.Rules) > 0 {
		out.Rules = cloneRules(in.Rules)
	}
	if in.Rule != nil {
		child := cloneRule(*in.Rule)
		out.Rule = &child
	}
	return out
}

func cloneValues(in []ValueExpr) []ValueExpr {
	out := make([]ValueExpr, 0, len(in))
	for _, value := range in {
		out = append(out, cloneValue(value))
	}
	return out
}

func cloneValue(in ValueExpr) ValueExpr {
	out := in
	if in.Indicator != nil {
		indicator := *in.Indicator
		indicator.Params = make(map[string]float64, len(in.Indicator.Params))
		for key, value := range in.Indicator.Params {
			indicator.Params[key] = value
		}
		indicator.Compare = cloneValue(in.Indicator.Compare)
		out.Indicator = &indicator
	}
	if len(in.Args) > 0 {
		out.Args = cloneValues(in.Args)
	}
	return out
}

func withDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
