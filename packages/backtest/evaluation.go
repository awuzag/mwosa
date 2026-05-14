package backtest

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	EvaluationPeriodExplicit = "explicit"
	EvaluationPeriodYearly   = "yearly"
	EvaluationPeriodRolling  = "rolling"

	RankingOrderAsc  = "asc"
	RankingOrderDesc = "desc"
)

type EvaluationPlan struct {
	Name        string
	Spec        EvaluationSpec
	Strategy    StrategySpec
	BaseRun     BacktestRunSpec
	Cases       []EvaluationCasePlan
	WalkForward []WalkForwardStepPlan
}

type EvaluationCasePlan struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Period     Period          `json:"period"`
	Parameters map[string]any  `json:"parameters,omitempty"`
	Strategy   StrategySpec    `json:"-"`
	Run        BacktestRunSpec `json:"-"`
	Plan       StrategyPlan    `json:"-"`
}

type EvaluationCaseResult struct {
	CaseID            string             `json:"case_id"`
	CaseName          string             `json:"case_name"`
	Period            Period             `json:"period"`
	Parameters        map[string]any     `json:"parameters,omitempty"`
	Result            Result             `json:"result"`
	Metrics           Metrics            `json:"metrics"`
	RegimeTags        []string           `json:"regime_tags,omitempty"`
	ConstraintResults []ConstraintResult `json:"constraint_results,omitempty"`
	PassedConstraints bool               `json:"passed_constraints"`
	Rank              int                `json:"rank,omitempty"`
	Objective         string             `json:"objective,omitempty"`
	ObjectiveValue    float64            `json:"objective_value,omitempty"`
}

type ConstraintResult struct {
	ID      string  `json:"id"`
	Passed  bool    `json:"passed"`
	Actual  float64 `json:"actual"`
	Limit   float64 `json:"limit"`
	Message string  `json:"message"`
}

type WalkForwardStepPlan struct {
	Index      int                  `json:"index"`
	Train      Period               `json:"train"`
	Test       Period               `json:"test"`
	Parameters []map[string]any     `json:"parameters"`
	Cases      []EvaluationCasePlan `json:"-"`
	TestCases  []EvaluationCasePlan `json:"-"`
}

type WalkForwardStepResult struct {
	Index              int            `json:"index"`
	Train              Period         `json:"train"`
	Test               Period         `json:"test"`
	SelectedParameters map[string]any `json:"selected_parameters,omitempty"`
	TrainCaseID        string         `json:"train_case_id"`
	TestCaseID         string         `json:"test_case_id"`
	TrainObjective     float64        `json:"train_objective"`
	TestMetrics        Metrics        `json:"test_metrics"`
	ResultHash         string         `json:"result_hash"`
}

type EvaluationResult struct {
	Name        string                  `json:"name"`
	Strategy    string                  `json:"strategy"`
	BaseRun     string                  `json:"base_run"`
	Cases       []EvaluationCaseResult  `json:"cases"`
	Ranking     []EvaluationCaseResult  `json:"ranking"`
	WalkForward []WalkForwardStepResult `json:"walk_forward,omitempty"`
}

func CompileEvaluation(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec, registry IndicatorRegistry) (EvaluationPlan, error) {
	errb := oops.In("backtest_evaluation").With("evaluation", evaluation.Name)
	if err := validateEvaluationSpec(strategy, baseRun, evaluation); err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	periods, err := evaluationPeriods(evaluation.Periods, baseRun.Data)
	if err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	parameterSets, err := parameterGrid(evaluation.Parameters)
	if err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	cases, err := buildEvaluationCases(strategy, baseRun, evaluation, periods, parameterSets, registry, "case")
	if err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	walkForward, err := buildWalkForwardSteps(strategy, baseRun, evaluation, parameterSets, registry)
	if err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	return EvaluationPlan{
		Name:        evaluation.Name,
		Spec:        evaluation,
		Strategy:    strategy,
		BaseRun:     baseRun,
		Cases:       cases,
		WalkForward: walkForward,
	}, nil
}

func RankEvaluationResults(results []EvaluationCaseResult, ranking EvaluationRankingSpec) ([]EvaluationCaseResult, error) {
	objective := withDefault(strings.TrimSpace(ranking.Objective), MetricCalmar)
	order := withDefault(strings.TrimSpace(ranking.Order), RankingOrderDesc)
	if order != RankingOrderAsc && order != RankingOrderDesc {
		return nil, oops.In("backtest_evaluation").With("order", order).New("ranking order must be asc or desc")
	}
	out := make([]EvaluationCaseResult, 0, len(results))
	for _, result := range results {
		if !result.PassedConstraints {
			continue
		}
		value, ok := metricValue(result.Metrics, objective)
		if !ok {
			return nil, oops.In("backtest_evaluation").With("objective", objective, "case", result.CaseID).New("ranking objective metric is missing")
		}
		result.Objective = objective
		result.ObjectiveValue = value
		out = append(out, result)
	}
	slices.SortFunc(out, func(a, b EvaluationCaseResult) int {
		if a.ObjectiveValue == b.ObjectiveValue {
			return strings.Compare(a.CaseID, b.CaseID)
		}
		if order == RankingOrderAsc {
			if a.ObjectiveValue < b.ObjectiveValue {
				return -1
			}
			return 1
		}
		if a.ObjectiveValue > b.ObjectiveValue {
			return -1
		}
		return 1
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

func EvaluateConstraints(metrics Metrics, constraints EvaluationConstraintSet) []ConstraintResult {
	out := make([]ConstraintResult, 0)
	if constraints.MaxDrawdownLTE != nil {
		actual := math.Abs(metrics[MetricMaxDrawdown])
		out = append(out, compareLTE("max_drawdown_lte", actual, *constraints.MaxDrawdownLTE))
	}
	if constraints.MinCAGRGTE != nil {
		out = append(out, compareGTE("min_cagr_gte", metrics[MetricCAGR], *constraints.MinCAGRGTE))
	}
	if constraints.MaxTurnoverLTE != nil {
		out = append(out, compareLTE("max_turnover_lte", metrics[MetricTurnover], *constraints.MaxTurnoverLTE))
	}
	if constraints.MinTradeCountGTE != nil {
		out = append(out, compareGTE("min_trade_count_gte", metrics[MetricTradeCount], *constraints.MinTradeCountGTE))
	}
	return out
}

func ConstraintsPassed(results []ConstraintResult) bool {
	for _, result := range results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func RegimeTags(result Result, benchmark []Bar) []string {
	tags := make([]string, 0, 2)
	if len(benchmark) >= 2 && benchmark[0].Close > 0 {
		ret := benchmark[len(benchmark)-1].Close/benchmark[0].Close - 1
		switch {
		case ret > 0.05:
			tags = append(tags, "bull")
		case ret < -0.05:
			tags = append(tags, "bear")
		default:
			tags = append(tags, "sideways")
		}
	} else {
		switch {
		case result.TotalReturn > 0.05:
			tags = append(tags, "bull")
		case result.TotalReturn < -0.05:
			tags = append(tags, "bear")
		default:
			tags = append(tags, "sideways")
		}
	}
	if volatility(result) >= 0.2 {
		tags = append(tags, "high_vol")
	} else {
		tags = append(tags, "low_vol")
	}
	return tags
}

func validateEvaluationSpec(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec) error {
	errb := oops.In("backtest_evaluation").With("evaluation", evaluation.Name)
	if evaluation.Kind != KindEvaluation {
		return errb.With("kind", evaluation.Kind).New("evaluation kind must be Evaluation")
	}
	if evaluation.SchemaVersion != SchemaVersion {
		return errb.With("schema_version", evaluation.SchemaVersion).New("unsupported evaluation schema version")
	}
	if strings.TrimSpace(evaluation.Name) == "" {
		return errb.New("evaluation name is required")
	}
	if evaluation.Strategy.Name != "" && evaluation.Strategy.Name != strategy.Name {
		return errb.With("strategy_name", evaluation.Strategy.Name, "expected_strategy_name", strategy.Name).New("evaluation strategy reference does not match strategy")
	}
	baseRef := withDefault(evaluation.BaseRun.Ref, evaluation.BaseRun.Name)
	if baseRef != "" && baseRef != baseRun.Name {
		return errb.With("base_run", baseRef, "expected_base_run", baseRun.Name).New("evaluation base_run reference does not match BacktestRun")
	}
	if strings.TrimSpace(baseRef) == "" {
		return errb.New("evaluation base_run ref is required")
	}
	if strings.TrimSpace(evaluation.Ranking.Objective) != "" {
		if _, err := requireMetricMust(evaluation.Ranking.Objective); err != nil {
			return errb.Wrap(err)
		}
	}
	if evaluation.Execution.Parallelism < 0 {
		return errb.With("parallelism", evaluation.Execution.Parallelism).New("evaluation execution parallelism must not be negative")
	}
	return nil
}

func evaluationPeriods(spec EvaluationPeriodsSpec, data DataSpec) ([]EvaluationPeriodSpec, error) {
	mode := withDefault(strings.TrimSpace(spec.Mode), EvaluationPeriodExplicit)
	fromText := withDefault(spec.From, data.From)
	toText := withDefault(spec.To, data.To)
	from, err := time.Parse(time.DateOnly, fromText)
	if err != nil {
		return nil, oops.In("backtest_evaluation_periods").With("from", fromText).Wrapf(err, "parse evaluation from date")
	}
	to, err := time.Parse(time.DateOnly, toText)
	if err != nil {
		return nil, oops.In("backtest_evaluation_periods").With("to", toText).Wrapf(err, "parse evaluation to date")
	}
	if to.Before(from) {
		return nil, oops.In("backtest_evaluation_periods").New("evaluation to date must be on or after from date")
	}
	switch mode {
	case EvaluationPeriodExplicit:
		if len(spec.Items) == 0 {
			return []EvaluationPeriodSpec{{Name: from.Format("2006-01-02") + "_" + to.Format("2006-01-02"), From: fromText, To: toText}}, nil
		}
		return normalizeExplicitPeriods(spec.Items)
	case EvaluationPeriodYearly:
		return yearlyPeriods(from, to), nil
	case EvaluationPeriodRolling:
		window := spec.Window
		if window.Empty() && spec.WindowDays > 0 {
			window.Days = spec.WindowDays
		}
		step := spec.Step
		if step.Empty() && spec.StepDays > 0 {
			step.Days = spec.StepDays
		}
		return rollingPeriods(from, to, window, step)
	default:
		return nil, oops.In("backtest_evaluation_periods").With("mode", mode).New("unsupported evaluation periods mode")
	}
}

func normalizeExplicitPeriods(items []EvaluationPeriodSpec) ([]EvaluationPeriodSpec, error) {
	out := make([]EvaluationPeriodSpec, 0, len(items))
	for index, item := range items {
		if _, err := time.Parse(time.DateOnly, item.From); err != nil {
			return nil, oops.In("backtest_evaluation_periods").With("index", index, "from", item.From).Wrapf(err, "parse explicit period from date")
		}
		if _, err := time.Parse(time.DateOnly, item.To); err != nil {
			return nil, oops.In("backtest_evaluation_periods").With("index", index, "to", item.To).Wrapf(err, "parse explicit period to date")
		}
		if item.Name == "" {
			item.Name = "period-" + strconv.Itoa(index+1)
		}
		out = append(out, item)
	}
	return out, nil
}

func yearlyPeriods(from, to time.Time) []EvaluationPeriodSpec {
	out := make([]EvaluationPeriodSpec, 0)
	for year := from.Year(); year <= to.Year(); year++ {
		start := time.Date(year, 1, 1, 0, 0, 0, 0, from.Location())
		end := time.Date(year, 12, 31, 0, 0, 0, 0, from.Location())
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		out = append(out, EvaluationPeriodSpec{Name: strconv.Itoa(year), From: start.Format(time.DateOnly), To: end.Format(time.DateOnly)})
	}
	return out
}

func rollingPeriods(from, to time.Time, window DurationSpec, step DurationSpec) ([]EvaluationPeriodSpec, error) {
	if window.Empty() {
		return nil, oops.In("backtest_evaluation_periods").New("rolling periods require window")
	}
	if step.Empty() {
		step = window
	}
	out := make([]EvaluationPeriodSpec, 0)
	for index, start := 1, from; !start.After(to); index, start = index+1, addDuration(start, step) {
		end := addDuration(start, window).AddDate(0, 0, -1)
		if end.After(to) {
			break
		}
		out = append(out, EvaluationPeriodSpec{Name: "rolling-" + strconv.Itoa(index), From: start.Format(time.DateOnly), To: end.Format(time.DateOnly)})
	}
	if len(out) == 0 {
		return nil, oops.In("backtest_evaluation_periods").New("rolling periods produced no cases")
	}
	return out, nil
}

func buildEvaluationCases(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec, periods []EvaluationPeriodSpec, parameterSets []map[string]any, registry IndicatorRegistry, prefix string) ([]EvaluationCasePlan, error) {
	out := make([]EvaluationCasePlan, 0, len(periods)*len(parameterSets))
	for _, period := range periods {
		for paramIndex, parameters := range parameterSets {
			caseStrategy := cloneStrategy(strategy)
			caseRun := cloneRun(baseRun)
			caseRun.Name = evaluation.Name + "-" + period.Name + "-p" + strconv.Itoa(paramIndex+1)
			caseRun.Report.Metrics = mergeMetricSelection(caseRun.Report.Metrics, evaluation.Metrics)
			caseRun.Data.From = period.From
			caseRun.Data.To = period.To
			if err := applyParameters(&caseStrategy, &caseRun, parameters); err != nil {
				return nil, err
			}
			plan, err := Compile(caseStrategy, caseRun, registry)
			if err != nil {
				return nil, oops.In("backtest_evaluation").With("case", caseRun.Name).Wrap(err)
			}
			out = append(out, EvaluationCasePlan{
				ID:         prefix + "-" + strconv.Itoa(len(out)+1),
				Name:       caseRun.Name,
				Period:     Period{From: plan.From, To: plan.To},
				Parameters: cloneParameterMap(parameters),
				Strategy:   caseStrategy,
				Run:        caseRun,
				Plan:       plan,
			})
		}
	}
	return out, nil
}

func buildWalkForwardSteps(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec, parameterSets []map[string]any, registry IndicatorRegistry) ([]WalkForwardStepPlan, error) {
	if evaluation.WalkForward.Train.Empty() && evaluation.WalkForward.Test.Empty() {
		return nil, nil
	}
	errb := oops.In("backtest_walk_forward").With("evaluation", evaluation.Name)
	if evaluation.WalkForward.Train.Empty() || evaluation.WalkForward.Test.Empty() {
		return nil, errb.New("walk_forward requires train and test windows")
	}
	stepDuration := evaluation.WalkForward.Step
	if stepDuration.Empty() {
		stepDuration = evaluation.WalkForward.Test
	}
	start, err := time.Parse(time.DateOnly, withDefault(evaluation.Periods.From, baseRun.Data.From))
	if err != nil {
		return nil, errb.Wrapf(err, "parse walk-forward start date")
	}
	end, err := time.Parse(time.DateOnly, withDefault(evaluation.Periods.To, baseRun.Data.To))
	if err != nil {
		return nil, errb.Wrapf(err, "parse walk-forward end date")
	}
	out := make([]WalkForwardStepPlan, 0)
	for index, trainStart := 1, start; ; index, trainStart = index+1, addDuration(trainStart, stepDuration) {
		trainEnd := addDuration(trainStart, evaluation.WalkForward.Train).AddDate(0, 0, -1)
		testStart := trainEnd.AddDate(0, 0, 1)
		testEnd := addDuration(testStart, evaluation.WalkForward.Test).AddDate(0, 0, -1)
		if testEnd.After(end) {
			break
		}
		trainPeriod := EvaluationPeriodSpec{Name: "wf-" + strconv.Itoa(index) + "-train", From: trainStart.Format(time.DateOnly), To: trainEnd.Format(time.DateOnly)}
		testPeriod := EvaluationPeriodSpec{Name: "wf-" + strconv.Itoa(index) + "-test", From: testStart.Format(time.DateOnly), To: testEnd.Format(time.DateOnly)}
		trainCases, err := buildEvaluationCases(strategy, baseRun, evaluation, []EvaluationPeriodSpec{trainPeriod}, parameterSets, registry, "wf-"+strconv.Itoa(index)+"-train")
		if err != nil {
			return nil, err
		}
		testCases, err := buildEvaluationCases(strategy, baseRun, evaluation, []EvaluationPeriodSpec{testPeriod}, parameterSets, registry, "wf-"+strconv.Itoa(index)+"-test")
		if err != nil {
			return nil, err
		}
		out = append(out, WalkForwardStepPlan{
			Index:      index,
			Train:      Period{From: trainStart, To: trainEnd},
			Test:       Period{From: testStart, To: testEnd},
			Parameters: cloneParameterSets(parameterSets),
			Cases:      trainCases,
			TestCases:  testCases,
		})
	}
	if len(out) == 0 {
		return nil, errb.New("walk_forward produced no train/test steps")
	}
	return out, nil
}

func parameterGrid(parameters map[string][]any) ([]map[string]any, error) {
	if len(parameters) == 0 {
		return []map[string]any{{}}, nil
	}
	keys := make([]string, 0, len(parameters))
	for key, values := range parameters {
		if strings.TrimSpace(key) == "" {
			return nil, oops.In("backtest_evaluation_parameters").New("parameter path is empty")
		}
		if len(values) == 0 {
			return nil, oops.In("backtest_evaluation_parameters").With("path", key).New("parameter grid values are empty")
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := []map[string]any{{}}
	for _, key := range keys {
		next := make([]map[string]any, 0, len(out)*len(parameters[key]))
		for _, current := range out {
			for _, value := range parameters[key] {
				copied := cloneParameterMap(current)
				copied[key] = normalizeParameterValue(value)
				next = append(next, copied)
			}
		}
		out = next
	}
	return out, nil
}

func applyParameters(strategy *StrategySpec, run *BacktestRunSpec, parameters map[string]any) error {
	for path, value := range parameters {
		parts := strings.Split(path, ".")
		if len(parts) == 4 && parts[0] == "indicators" && parts[2] == "params" {
			indicator, ok := strategy.Indicators[parts[1]]
			if !ok {
				return oops.In("backtest_evaluation_parameters").With("path", path).New("indicator parameter target is unknown")
			}
			if indicator.Params == nil {
				indicator.Params = map[string]float64{}
			}
			number, err := numericParameter(value, path)
			if err != nil {
				return err
			}
			indicator.Params[parts[3]] = number
			strategy.Indicators[parts[1]] = indicator
			continue
		}
		switch path {
		case "sizing.value":
			number, err := numericParameter(value, path)
			if err != nil {
				return err
			}
			strategy.Sizing.Value = number
		case "risk.max_positions":
			number, err := numericParameter(value, path)
			if err != nil {
				return err
			}
			strategy.Risk.MaxPositions = int(number)
		case "risk.max_symbol_weight_pct":
			number, err := numericParameter(value, path)
			if err != nil {
				return err
			}
			strategy.Risk.MaxSymbolWeightPct = number
		default:
			if err := applyUniverseParameter(run, parts, path, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyUniverseParameter(run *BacktestRunSpec, parts []string, path string, value any) error {
	if len(parts) < 5 || parts[0] != "universe" || parts[1] != "pipeline" || parts[3] != "params" {
		return oops.In("backtest_evaluation_parameters").With("path", path).New("unsupported parameter path")
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil || index < 0 || index >= len(run.Universe.Pipeline) {
		return oops.In("backtest_evaluation_parameters").With("path", path).New("universe pipeline parameter index is invalid")
	}
	if run.Universe.Pipeline[index].Params == nil {
		run.Universe.Pipeline[index].Params = map[string]any{}
	}
	run.Universe.Pipeline[index].Params[strings.Join(parts[4:], ".")] = normalizeParameterValue(value)
	return nil
}

func mergeMetricSelection(base MetricSelectionSpec, override MetricSelectionSpec) MetricSelectionSpec {
	if override.Preset == "" && len(override.Include) == 0 && len(override.Exclude) == 0 {
		return base
	}
	return override
}

func metricValue(metrics Metrics, metric string) (float64, bool) {
	value, ok := metrics[metric]
	return value, ok
}

func compareLTE(id string, actual float64, limit float64) ConstraintResult {
	return ConstraintResult{ID: id, Passed: actual <= limit, Actual: actual, Limit: limit, Message: id}
}

func compareGTE(id string, actual float64, limit float64) ConstraintResult {
	return ConstraintResult{ID: id, Passed: actual >= limit, Actual: actual, Limit: limit, Message: id}
}

func (d DurationSpec) Empty() bool {
	return d.Years == 0 && d.Months == 0 && d.Days == 0
}

func addDuration(value time.Time, duration DurationSpec) time.Time {
	return value.AddDate(duration.Years, duration.Months, duration.Days)
}

func cloneStrategy(in StrategySpec) StrategySpec {
	payload, _ := json.Marshal(in)
	var out StrategySpec
	_ = json.Unmarshal(payload, &out)
	return out
}

func cloneRun(in BacktestRunSpec) BacktestRunSpec {
	payload, _ := json.Marshal(in)
	var out BacktestRunSpec
	_ = json.Unmarshal(payload, &out)
	return out
}

func cloneParameterMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = normalizeParameterValue(value)
	}
	return out
}

func cloneParameterSets(in []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneParameterMap(item))
	}
	return out
}

func normalizeParameterValue(value any) any {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint64:
		return float64(typed)
	case float32:
		return float64(typed)
	default:
		return value
	}
}

func numericParameter(value any, path string) (float64, error) {
	switch typed := normalizeParameterValue(value).(type) {
	case float64:
		return typed, nil
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, oops.In("backtest_evaluation_parameters").With("path", path, "value", typed).Wrapf(err, "parse numeric parameter")
		}
		return parsed, nil
	default:
		return 0, oops.In("backtest_evaluation_parameters").With("path", path).New("parameter value must be numeric")
	}
}

func requireMetricMust(id string) (MetricDefinition, error) {
	registry, err := DefaultMetricRegistry()
	if err != nil {
		return MetricDefinition{}, err
	}
	return requireMetric(registry, id)
}

func ResolveWalkForwardSelection(spec EvaluationSpec) EvaluationRankingSpec {
	objective := withDefault(spec.WalkForward.Select.Objective, withDefault(spec.Ranking.Objective, MetricCalmar))
	order := withDefault(spec.WalkForward.Select.Order, withDefault(spec.Ranking.Order, RankingOrderDesc))
	return EvaluationRankingSpec{Objective: objective, Order: order}
}

func WalkForwardConstraints(spec EvaluationSpec) EvaluationConstraintSet {
	if spec.WalkForward.Select.Constraints.Empty() {
		return spec.Constraints
	}
	return spec.WalkForward.Select.Constraints
}

func (c EvaluationConstraintSet) Empty() bool {
	return c.MaxDrawdownLTE == nil && c.MinCAGRGTE == nil && c.MaxTurnoverLTE == nil && c.MinTradeCountGTE == nil
}

func RunCase(ctx context.Context, engine Engine, plan StrategyPlan) (Result, error) {
	return engine.Run(ctx, plan)
}
