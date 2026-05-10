package backtest

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/samber/oops"
)

type ruleContext struct {
	symbol     string
	index      int
	bars       []Bar
	indicators map[string]map[string][]float64
	plan       StrategyPlan
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
	default:
		return 0, false, errb.New("unsupported value expression kind")
	}
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
	collectInlineIndicators(plan.Entry, out)
	collectInlineIndicators(plan.Exit, out)
	return out
}

func collectInlineIndicators(rule RuleExpr, out map[string]IndicatorSpec) {
	for _, arg := range rule.Args {
		if arg.Kind == "indicator" && arg.Indicator != nil {
			out[indicatorKey(*arg.Indicator)] = *arg.Indicator
		}
	}
	for _, child := range rule.Rules {
		collectInlineIndicators(child, out)
	}
	if rule.Rule != nil {
		collectInlineIndicators(*rule.Rule, out)
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
		ID     string             `json:"id"`
		Source ValueExpr          `json:"source"`
		Params map[string]float64 `json:"params,omitempty"`
		Output string             `json:"output,omitempty"`
	}{
		ID:     spec.ID,
		Source: spec.Source,
		Params: params,
		Output: spec.Output,
	})
	return "indicator:" + string(payload)
}
