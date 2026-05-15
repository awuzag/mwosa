package backtest

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/samber/oops"
)

type ruleContext struct {
	symbol         string
	index          int
	bars           []Bar
	series         map[string][]Bar
	activeSymbols  []string
	currentBars    map[string]Bar
	currentIndexes map[string]int
	indicators     map[string]map[string][]float64
	plan           StrategyPlan
	portfolio      portfolio
	prices         map[string]float64
	equity         []EquityPoint
}

func evaluateRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	switch rule.Operator {
	case "all":
		for _, child := range rule.Rules {
			matched, err := evaluateRule(child, ctx)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	case "any":
		for _, child := range rule.Rules {
			matched, err := evaluateRule(child, ctx)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	case "not":
		if rule.Rule == nil {
			return false, errb.New("not rule requires a child rule")
		}
		matched, err := evaluateRule(*rule.Rule, ctx)
		return !matched, err
	case "gt", "gte", "lt", "lte", "eq":
		left, leftOK, err := currentValue(rule.Args[0], ctx)
		if err != nil || !leftOK {
			return false, err
		}
		right, rightOK, err := currentValue(rule.Args[1], ctx)
		if err != nil || !rightOK {
			return false, err
		}
		return compare(rule.Operator, left, right), nil
	case "between":
		value, valueOK, err := currentValue(rule.Args[0], ctx)
		if err != nil || !valueOK {
			return false, err
		}
		lower, lowerOK, err := currentValue(rule.Args[1], ctx)
		if err != nil || !lowerOK {
			return false, err
		}
		upper, upperOK, err := currentValue(rule.Args[2], ctx)
		if err != nil || !upperOK {
			return false, err
		}
		return value >= lower && value <= upper, nil
	case "crosses_above", "crosses_below":
		left, leftPrevious, leftOK, err := currentAndPrevious(rule.Args[0], ctx)
		if err != nil || !leftOK {
			return false, err
		}
		right, rightPrevious, rightOK, err := currentAndPrevious(rule.Args[1], ctx)
		if err != nil || !rightOK {
			return false, err
		}
		if rule.Operator == "crosses_above" {
			return leftPrevious <= rightPrevious && left > right, nil
		}
		return leftPrevious >= rightPrevious && left < right, nil
	case "rising", "falling":
		return evaluateTrendRule(rule, ctx)
	case "slope":
		return evaluateSlopeRule(rule, ctx)
	case "for_n_bars":
		return evaluateForNBarsRule(rule, ctx)
	case "bars_since":
		return evaluateBarsSinceRule(rule, ctx)
	case "cooldown":
		return evaluateCooldownRule(rule, ctx)
	case "changed":
		return evaluateChangedRule(rule, ctx)
	case "new_high", "new_low":
		return evaluateNewExtremeRule(rule, ctx)
	case "position_exists":
		return ctx.portfolio.hasPosition(ctx.symbol), nil
	case "weekly", "monthly", "first_trading_day":
		return evaluateCalendarRule(rule, ctx)
	case "target_weight_changed":
		return evaluateTargetWeightChangedRule(rule, ctx)
	case "stop_loss", "take_profit", "time_stop", "trailing_stop", "volatility_stop":
		return evaluateStopRule(rule, ctx)
	default:
		return false, errb.New("unsupported rule operator")
	}
}

func compare(operator string, left float64, right float64) bool {
	switch operator {
	case "gt":
		return left > right
	case "gte":
		return left >= right
	case "lt":
		return left < right
	case "lte":
		return left <= right
	case "eq":
		return left == right
	default:
		return false
	}
}

func currentAndPrevious(expr ValueExpr, ctx ruleContext) (float64, float64, bool, error) {
	current, currentOK, err := currentValue(expr, ctx)
	if err != nil || !currentOK {
		return 0, 0, false, err
	}
	previous, previousOK, err := previousValue(expr, ctx)
	if err != nil || !previousOK {
		return 0, 0, false, err
	}
	return current, previous, true, nil
}

func currentValue(expr ValueExpr, ctx ruleContext) (float64, bool, error) {
	return valueAt(expr, ctx, ctx.index)
}

func previousValue(expr ValueExpr, ctx ruleContext) (float64, bool, error) {
	if ctx.index == 0 {
		return 0, false, nil
	}
	return valueAt(expr, ctx, ctx.index-1)
}

func valueAt(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	errb := oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol)
	if index < 0 || index >= len(ctx.bars) {
		return 0, false, nil
	}
	if strings.TrimSpace(expr.Timeframe) != "" {
		return timeframeValueAt(expr, ctx, index)
	}
	switch expr.Kind {
	case "price":
		value, ok := priceValue(ctx.bars[index], expr.Price)
		return value, ok, nil
	case "value":
		return expr.Value, true, nil
	case "ref":
		seriesBySymbol, ok := ctx.indicators["ref:"+expr.Ref]
		if !ok {
			return 0, false, errb.With("ref", expr.Ref).New("indicator ref series is missing")
		}
		value, ok := finiteSeriesValue(seriesBySymbol[ctx.symbol], index)
		return value, ok, nil
	case "position":
		if index != ctx.index {
			return 0, false, nil
		}
		return positionValue(expr.Position, ctx)
	case "portfolio":
		if index != ctx.index {
			return 0, false, nil
		}
		return portfolioValue(expr.Portfolio, ctx)
	case "indicator":
		if expr.Indicator == nil {
			return 0, false, errb.New("indicator expression requires indicator spec")
		}
		seriesBySymbol, ok := ctx.indicators[indicatorKey(*expr.Indicator)]
		if !ok {
			return 0, false, errb.With("indicator", expr.Indicator.ID).New("indicator series is missing")
		}
		value, ok := finiteSeriesValue(seriesBySymbol[ctx.symbol], index)
		return value, ok, nil
	case "rank", "percentile", "universe_rank":
		return crossSectionalRankValue(expr, ctx, index)
	case "relative_strength":
		return relativeStrengthValue(expr, ctx, index)
	case "spread", "ratio":
		return pairValueAt(expr, ctx, index)
	case "add", "sub", "mul", "div", "min", "max", "abs":
		return arithmeticValueAt(expr, ctx, index)
	default:
		return 0, false, errb.New("unsupported value expression kind")
	}
}

func timeframeValueAt(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	timeframe, err := ParseTimeframe(expr.Timeframe)
	if err != nil {
		return 0, false, oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol, "timeframe", expr.Timeframe).Wrap(err)
	}
	if index < 0 || index >= len(ctx.bars) {
		return 0, false, nil
	}
	anchor := ctx.bars[index].Time
	bars, ok := ctx.series[ctx.symbol]
	if !ok {
		bars = ctx.bars
	}
	targetIndex := -1
	for candidate := len(bars) - 1; candidate >= 0; candidate-- {
		bar := bars[candidate]
		if bar.Time.After(anchor) {
			continue
		}
		if withDefault(bar.Timeframe, Timeframe1Day) != timeframe.ID {
			continue
		}
		targetIndex = candidate
		break
	}
	if targetIndex < 0 {
		return 0, false, nil
	}
	scoped := expr
	scoped.Timeframe = ""
	next := ctx
	next.bars = bars
	next.index = targetIndex
	return valueAt(scoped, next, targetIndex)
}

type crossSectionalValue struct {
	symbol string
	value  float64
}

func crossSectionalRankValue(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	values, ok, err := crossSectionalValues(expr, ctx, index)
	if err != nil || !ok {
		return 0, false, err
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value == values[j].value {
			return values[i].symbol < values[j].symbol
		}
		return values[i].value > values[j].value
	})
	for rank, item := range values {
		if item.symbol != ctx.symbol {
			continue
		}
		if expr.Kind == "percentile" {
			if len(values) == 1 {
				return 100, true, nil
			}
			return float64(len(values)-rank-1) / float64(len(values)-1) * 100, true, nil
		}
		return float64(rank + 1), true, nil
	}
	return 0, false, nil
}

func relativeStrengthValue(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	values, ok, err := crossSectionalValues(expr, ctx, index)
	if err != nil || !ok {
		return 0, false, err
	}
	var current float64
	var sum float64
	var found bool
	for _, item := range values {
		sum += item.value
		if item.symbol == ctx.symbol {
			current = item.value
			found = true
		}
	}
	if !found || len(values) == 0 {
		return 0, false, nil
	}
	average := sum / float64(len(values))
	if average == 0 {
		return 0, false, nil
	}
	return current/average - 1, true, nil
}

func crossSectionalValues(expr ValueExpr, ctx ruleContext, index int) ([]crossSectionalValue, bool, error) {
	errb := oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol)
	if len(expr.Args) != 1 {
		return nil, false, errb.New("cross-sectional value expression requires one arg")
	}
	if index != ctx.index {
		return nil, false, nil
	}
	values := make([]crossSectionalValue, 0, len(ctx.activeSymbols))
	for _, symbol := range ctx.activeSymbols {
		if _, ok := ctx.currentBars[symbol]; !ok {
			continue
		}
		symbolIndex, ok := ctx.currentIndexes[symbol]
		if !ok {
			continue
		}
		value, valueOK, err := valueAtForSymbol(expr.Args[0], ctx, symbol, symbolIndex)
		if err != nil {
			return nil, false, err
		}
		if !valueOK || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		values = append(values, crossSectionalValue{symbol: symbol, value: value})
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	return values, true, nil
}

func valueAtForSymbol(expr ValueExpr, ctx ruleContext, symbol string, index int) (float64, bool, error) {
	if isCrossSectionalValueKind(expr.Kind) {
		return 0, false, oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", symbol).New("cross-sectional value expression cannot be nested")
	}
	bars, ok := ctx.series[symbol]
	if !ok || index < 0 || index >= len(bars) {
		return 0, false, nil
	}
	next := ctx
	next.symbol = symbol
	next.index = index
	next.bars = bars
	return valueAt(expr, next, index)
}

func pairValueAt(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	if len(expr.Args) != 2 {
		return 0, false, oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol).New("pair value expression requires two args")
	}
	left, leftOK, err := valueAt(expr.Args[0], ctx, index)
	if err != nil || !leftOK {
		return 0, false, err
	}
	right, rightOK, err := valueAt(expr.Args[1], ctx, index)
	if err != nil || !rightOK {
		return 0, false, err
	}
	switch expr.Kind {
	case "spread":
		return left - right, true, nil
	case "ratio":
		if right == 0 {
			return 0, false, nil
		}
		return left / right, true, nil
	default:
		return 0, false, oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol).New("unsupported pair value expression kind")
	}
}

func arithmeticValueAt(expr ValueExpr, ctx ruleContext, index int) (float64, bool, error) {
	values := make([]float64, 0, len(expr.Args))
	for _, arg := range expr.Args {
		value, ok, err := valueAt(arg, ctx, index)
		if err != nil || !ok {
			return 0, false, err
		}
		values = append(values, value)
	}
	switch expr.Kind {
	case "add":
		var out float64
		for _, value := range values {
			out += value
		}
		return out, true, nil
	case "sub":
		out := values[0]
		for _, value := range values[1:] {
			out -= value
		}
		return out, true, nil
	case "mul":
		out := 1.0
		for _, value := range values {
			out *= value
		}
		return out, true, nil
	case "div":
		out := values[0]
		for _, value := range values[1:] {
			if value == 0 {
				return 0, false, nil
			}
			out /= value
		}
		return out, true, nil
	case "min":
		out := values[0]
		for _, value := range values[1:] {
			out = math.Min(out, value)
		}
		return out, true, nil
	case "max":
		out := values[0]
		for _, value := range values[1:] {
			out = math.Max(out, value)
		}
		return out, true, nil
	case "abs":
		return math.Abs(values[0]), true, nil
	default:
		return 0, false, oops.In("backtest_value_evaluator").With("kind", expr.Kind, "symbol", ctx.symbol).New("unsupported arithmetic value expression kind")
	}
}

func evaluateTrendRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 2 {
		return false, errb.New("trend rule requires value expression and bar count")
	}
	bars, ok, err := currentValue(rule.Args[1], ctx)
	if err != nil || !ok {
		return false, err
	}
	count := int(bars)
	if count < 2 {
		return false, errb.With("bars", bars).New("trend rule requires at least two bars")
	}
	if ctx.index-count+1 < 0 {
		return false, nil
	}
	previous, previousOK, err := valueAt(rule.Args[0], ctx, ctx.index-count+1)
	if err != nil || !previousOK {
		return false, err
	}
	for index := ctx.index - count + 2; index <= ctx.index; index++ {
		current, currentOK, err := valueAt(rule.Args[0], ctx, index)
		if err != nil || !currentOK {
			return false, err
		}
		if rule.Operator == "rising" && current <= previous {
			return false, nil
		}
		if rule.Operator == "falling" && current >= previous {
			return false, nil
		}
		previous = current
	}
	return true, nil
}

func evaluateSlopeRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 3 {
		return false, errb.New("slope rule requires value expression, bar count, and minimum slope")
	}
	bars, barsOK, err := currentValue(rule.Args[1], ctx)
	if err != nil || !barsOK {
		return false, err
	}
	count := int(bars)
	if count < 1 {
		return false, errb.With("bars", bars).New("slope rule requires positive bar count")
	}
	startIndex := ctx.index - count
	if startIndex < 0 {
		return false, nil
	}
	start, startOK, err := valueAt(rule.Args[0], ctx, startIndex)
	if err != nil || !startOK {
		return false, err
	}
	current, currentOK, err := currentValue(rule.Args[0], ctx)
	if err != nil || !currentOK {
		return false, err
	}
	minimum, minimumOK, err := currentValue(rule.Args[2], ctx)
	if err != nil || !minimumOK {
		return false, err
	}
	return (current-start)/float64(count) > minimum, nil
}

func evaluateForNBarsRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if rule.Rule == nil {
		return false, errb.New("for_n_bars rule requires child rule")
	}
	if len(rule.Args) != 1 {
		return false, errb.New("for_n_bars rule requires bar count arg")
	}
	bars, ok, err := currentValue(rule.Args[0], ctx)
	if err != nil || !ok {
		return false, err
	}
	count := int(bars)
	if count < 1 {
		return false, errb.With("bars", bars).New("for_n_bars rule requires positive bar count")
	}
	startIndex := ctx.index - count + 1
	if startIndex < 0 {
		return false, nil
	}
	for index := startIndex; index <= ctx.index; index++ {
		next := ctx
		next.index = index
		matched, err := evaluateRule(*rule.Rule, next)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}

func evaluateBarsSinceRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if rule.Rule == nil {
		return false, errb.New("bars_since rule requires child rule")
	}
	if len(rule.Args) != 1 {
		return false, errb.New("bars_since rule requires bar count arg")
	}
	bars, ok, err := currentValue(rule.Args[0], ctx)
	if err != nil || !ok {
		return false, err
	}
	count := int(bars)
	if count < 1 {
		return false, errb.With("bars", bars).New("bars_since rule requires positive bar count")
	}
	startIndex := ctx.index - count
	if startIndex < 0 {
		startIndex = 0
	}
	for index := ctx.index - 1; index >= startIndex; index-- {
		next := ctx
		next.index = index
		matched, err := evaluateRule(*rule.Rule, next)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func evaluateCooldownRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if rule.Rule == nil {
		return false, errb.New("cooldown rule requires child rule")
	}
	if len(rule.Args) != 1 {
		return false, errb.New("cooldown rule requires bar count arg")
	}
	matched, err := evaluateRule(*rule.Rule, ctx)
	if err != nil || !matched {
		return matched, err
	}
	barsSince := RuleExpr{Operator: "bars_since", Rule: rule.Rule, Args: rule.Args}
	recent, err := evaluateBarsSinceRule(barsSince, ctx)
	if err != nil {
		return false, err
	}
	return !recent, nil
}

func evaluateChangedRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 1 {
		return false, errb.New("changed rule requires value expression")
	}
	current, previous, ok, err := currentAndPrevious(rule.Args[0], ctx)
	if err != nil || !ok {
		return false, err
	}
	return current != previous, nil
}

func evaluateNewExtremeRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 2 {
		return false, errb.New("new extreme rule requires value expression and window")
	}
	windowValue, ok, err := currentValue(rule.Args[1], ctx)
	if err != nil || !ok {
		return false, err
	}
	window := int(windowValue)
	if window < 2 {
		return false, errb.With("window", windowValue).New("new extreme rule requires at least two bars")
	}
	startIndex := ctx.index - window + 1
	if startIndex < 0 {
		return false, nil
	}
	current, currentOK, err := currentValue(rule.Args[0], ctx)
	if err != nil || !currentOK {
		return false, err
	}
	for index := startIndex; index < ctx.index; index++ {
		value, valueOK, err := valueAt(rule.Args[0], ctx, index)
		if err != nil || !valueOK {
			return false, err
		}
		if rule.Operator == "new_high" && current <= value {
			return false, nil
		}
		if rule.Operator == "new_low" && current >= value {
			return false, nil
		}
	}
	return true, nil
}

func evaluateCalendarRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 0 || len(rule.Rules) != 0 || rule.Rule != nil {
		return false, errb.New("calendar rule does not accept args or children")
	}
	if ctx.index < 0 || ctx.index >= len(ctx.bars) {
		return false, errb.With("index", ctx.index).New("calendar rule index is out of range")
	}
	if ctx.index == 0 {
		return true, nil
	}
	current := ctx.bars[ctx.index].Time
	previous := ctx.bars[ctx.index-1].Time
	switch rule.Operator {
	case "weekly":
		currentYear, currentWeek := current.ISOWeek()
		previousYear, previousWeek := previous.ISOWeek()
		return currentYear != previousYear || currentWeek != previousWeek, nil
	case "monthly", "first_trading_day":
		return current.Year() != previous.Year() || current.Month() != previous.Month(), nil
	default:
		return false, errb.New("unsupported calendar rule operator")
	}
}

func evaluateTargetWeightChangedRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) > 1 {
		return false, errb.New("target_weight_changed rule accepts at most one threshold arg")
	}
	threshold := 0.0
	if len(rule.Args) == 1 {
		value, ok, err := currentValue(rule.Args[0], ctx)
		if err != nil || !ok {
			return false, err
		}
		if value < 0 {
			return false, errb.With("threshold", value).New("target_weight_changed threshold must not be negative")
		}
		threshold = value
	}
	current, ok := portfolioSymbolWeightPct(ctx.portfolio, ctx.prices, ctx.symbol)
	if !ok {
		return false, nil
	}
	return math.Abs(current-ctx.plan.Sizing.Value) > threshold, nil
}

func evaluateStopRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	switch rule.Operator {
	case "stop_loss":
		threshold, ok, err := stopThreshold(rule, ctx)
		if err != nil || !ok {
			return false, err
		}
		value, valueOK, err := positionValue("unrealized_return", ctx)
		if err != nil || !valueOK {
			return false, err
		}
		return value <= -math.Abs(threshold), nil
	case "take_profit":
		threshold, ok, err := stopThreshold(rule, ctx)
		if err != nil || !ok {
			return false, err
		}
		value, valueOK, err := positionValue("unrealized_return", ctx)
		if err != nil || !valueOK {
			return false, err
		}
		return value >= threshold, nil
	case "time_stop":
		threshold, ok, err := stopThreshold(rule, ctx)
		if err != nil || !ok {
			return false, err
		}
		value, valueOK, err := positionValue("holding_bars", ctx)
		if err != nil || !valueOK {
			return false, err
		}
		return value >= threshold, nil
	case "trailing_stop":
		threshold, ok, err := stopThreshold(rule, ctx)
		if err != nil || !ok {
			return false, err
		}
		value, valueOK, err := positionValue("drawdown_from_entry", ctx)
		if err != nil || !valueOK {
			return false, err
		}
		return value <= -math.Abs(threshold), nil
	case "volatility_stop":
		return evaluateVolatilityStopRule(rule, ctx)
	default:
		return false, errb.New("unsupported stop rule operator")
	}
}

func stopThreshold(rule RuleExpr, ctx ruleContext) (float64, bool, error) {
	if len(rule.Args) != 1 {
		return 0, false, oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol).New("stop rule requires one threshold arg")
	}
	return currentValue(rule.Args[0], ctx)
}

func evaluateVolatilityStopRule(rule RuleExpr, ctx ruleContext) (bool, error) {
	errb := oops.In("backtest_rule_evaluator").With("operator", rule.Operator, "symbol", ctx.symbol)
	if len(rule.Args) != 2 {
		return false, errb.New("volatility_stop rule requires volatility and multiplier args")
	}
	volatility, volatilityOK, err := currentValue(rule.Args[0], ctx)
	if err != nil || !volatilityOK {
		return false, err
	}
	multiplier, multiplierOK, err := currentValue(rule.Args[1], ctx)
	if err != nil || !multiplierOK {
		return false, err
	}
	if volatility < 0 {
		return false, errb.With("volatility", volatility).New("volatility_stop volatility must not be negative")
	}
	if multiplier < 0 {
		return false, errb.With("multiplier", multiplier).New("volatility_stop multiplier must not be negative")
	}
	position, ok := ctx.portfolio.positions[ctx.symbol]
	if !ok || position.Quantity <= 0 || position.AvgPrice <= 0 {
		return false, nil
	}
	price, ok := ctx.prices[ctx.symbol]
	if !ok || price <= 0 {
		return false, nil
	}
	stopPrice := position.AvgPrice - volatility*multiplier
	return price <= stopPrice, nil
}

func positionValue(field string, ctx ruleContext) (float64, bool, error) {
	position, ok := ctx.portfolio.positions[ctx.symbol]
	if !ok || position.Quantity <= 0 {
		return 0, false, nil
	}
	price, ok := ctx.prices[ctx.symbol]
	if !ok || price <= 0 {
		return 0, false, nil
	}
	switch field {
	case "holding_bars":
		return float64(holdingBars(position.EntryTime, ctx.bars, ctx.index)), true, nil
	case "unrealized_return":
		if position.AvgPrice <= 0 {
			return 0, false, nil
		}
		return price/position.AvgPrice - 1, true, nil
	case "drawdown_from_entry":
		peak := positionPeakSinceEntry(position.EntryTime, ctx.bars, ctx.index)
		if peak <= 0 {
			return 0, false, nil
		}
		return price/peak - 1, true, nil
	default:
		return 0, false, oops.In("backtest_value_evaluator").With("position", field, "symbol", ctx.symbol).New("unsupported position value")
	}
}

func portfolioValue(field string, ctx ruleContext) (float64, bool, error) {
	equity := ctx.portfolio.equity(ctx.prices)
	switch field {
	case "cash_pct":
		if equity <= 0 {
			return 0, false, nil
		}
		return ctx.portfolio.cash / equity * 100, true, nil
	case "exposure_pct":
		if equity <= 0 {
			return 0, false, nil
		}
		return ctx.portfolio.positionsValue(ctx.prices) / equity * 100, true, nil
	case "position_count":
		return float64(ctx.portfolio.positionCount()), true, nil
	case "portfolio_drawdown":
		return portfolioDrawdown(ctx.portfolio, ctx.prices, ctx.equity), true, nil
	default:
		return 0, false, oops.In("backtest_value_evaluator").With("portfolio", field, "symbol", ctx.symbol).New("unsupported portfolio value")
	}
}

func portfolioDrawdown(p portfolio, prices map[string]float64, curve []EquityPoint) float64 {
	current := p.equity(prices)
	peak := current
	for _, point := range curve {
		if point.Equity > peak {
			peak = point.Equity
		}
	}
	if peak <= 0 {
		return 0
	}
	return current/peak - 1
}

func portfolioSymbolWeightPct(p portfolio, prices map[string]float64, symbol string) (float64, bool) {
	equity := p.equity(prices)
	if equity <= 0 {
		return 0, false
	}
	price, ok := prices[symbol]
	if !ok || price <= 0 {
		return 0, false
	}
	position := p.positions[symbol]
	return position.Quantity * price / equity * 100, true
}

func holdingBars(entryTime time.Time, bars []Bar, currentIndex int) int {
	count := 0
	for index := 0; index <= currentIndex && index < len(bars); index++ {
		if bars[index].Time.After(entryTime) {
			count++
		}
	}
	return count
}

func positionPeakSinceEntry(entryTime time.Time, bars []Bar, currentIndex int) float64 {
	var peak float64
	for index := 0; index <= currentIndex && index < len(bars); index++ {
		if bars[index].Time.Before(entryTime) {
			continue
		}
		if bars[index].Close > peak {
			peak = bars[index].Close
		}
	}
	return peak
}

func finiteSeriesValue(series []float64, index int) (float64, bool) {
	if index < 0 || index >= len(series) {
		return 0, false
	}
	value := series[index]
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func collectIndicatorKeys(plan StrategyPlan) map[string]IndicatorSpec {
	out := make(map[string]IndicatorSpec, len(plan.Indicators))
	for alias, indicator := range plan.Indicators {
		out["ref:"+alias] = indicator
	}
	if isATRSlippage(plan.Slippage) {
		spec := slippageATRIndicatorSpec(plan.Slippage)
		out[indicatorKey(spec)] = spec
	}
	for _, rule := range plan.Entries {
		collectInlineIndicators(rule, out)
	}
	for _, rule := range plan.Exits {
		collectInlineIndicators(rule, out)
	}
	for _, rule := range plan.Rebalance {
		collectInlineIndicators(rule, out)
	}
	for _, rule := range plan.Stops {
		collectInlineIndicators(rule, out)
	}
	return out
}

func slippageATRIndicatorSpec(spec CostSpec) IndicatorSpec {
	return IndicatorSpec{
		ID:     "atr",
		Source: ValueExpr{Kind: "price", Price: "close"},
		Params: map[string]float64{"window": float64(spec.Window)},
	}
}

func collectInlineIndicators(rule RuleExpr, out map[string]IndicatorSpec) {
	for _, arg := range rule.Args {
		collectInlineIndicatorsFromValue(arg, out)
	}
	for _, child := range rule.Rules {
		collectInlineIndicators(child, out)
	}
	if rule.Rule != nil {
		collectInlineIndicators(*rule.Rule, out)
	}
}

func collectInlineIndicatorsFromValue(expr ValueExpr, out map[string]IndicatorSpec) {
	if expr.Kind == "indicator" && expr.Indicator != nil {
		out[indicatorKey(*expr.Indicator)] = *expr.Indicator
	}
	for _, arg := range expr.Args {
		collectInlineIndicatorsFromValue(arg, out)
	}
}

func indicatorKey(spec IndicatorSpec) string {
	params := make(map[string]float64, len(spec.Params))
	keys := make([]string, 0, len(spec.Params))
	for key := range spec.Params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		params[key] = spec.Params[key]
	}
	payload, _ := json.Marshal(struct {
		ID      string             `json:"id"`
		Source  ValueExpr          `json:"source"`
		Compare ValueExpr          `json:"compare,omitempty"`
		Params  map[string]float64 `json:"params,omitempty"`
		Output  string             `json:"output,omitempty"`
	}{
		ID:      spec.ID,
		Source:  spec.Source,
		Compare: spec.Compare,
		Params:  params,
		Output:  spec.Output,
	})
	return "indicator:" + string(payload)
}
