package backtest

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	EvaluationPeriodExplicit    = "explicit"
	EvaluationPeriodYearly      = "yearly"
	EvaluationPeriodRolling     = "rolling"
	EvaluationPeriodExpanding   = "expanding"
	EvaluationPeriodWalkForward = "walk_forward"

	RankingOrderAsc  = "asc"
	RankingOrderDesc = "desc"

	SearchModeBounded  = "bounded"
	SearchModeBayesian = "bayesian"
	SearchModeRandom   = "random"

	EvaluationObjectiveWeightedScore = "weighted_score"
)

type EvaluationSearchOptimizer interface {
	ParameterSets(EvaluationSearchSpec) ([]map[string]any, error)
}

type EvaluationSearchOptimizerFunc func(EvaluationSearchSpec) ([]map[string]any, error)

func (f EvaluationSearchOptimizerFunc) ParameterSets(spec EvaluationSearchSpec) ([]map[string]any, error) {
	return f(spec)
}

type EvaluationSearchOptimizerRegistry map[string]EvaluationSearchOptimizer

func DefaultEvaluationSearchOptimizerRegistry() EvaluationSearchOptimizerRegistry {
	return EvaluationSearchOptimizerRegistry{
		SearchModeBounded: EvaluationSearchOptimizerFunc(boundedParameterSearch),
		SearchModeRandom:  EvaluationSearchOptimizerFunc(randomParameterSearch),
	}
}

func (r EvaluationSearchOptimizerRegistry) ParameterSets(spec EvaluationSearchSpec) ([]map[string]any, error) {
	mode := strings.ToLower(strings.TrimSpace(spec.Mode))
	errb := oops.In("backtest_evaluation_parameters").With("mode", mode)
	if mode == "" {
		return nil, errb.New("parameter search mode is required")
	}
	optimizer, ok := r.optimizer(mode)
	if !ok {
		return nil, errb.Errorf("parameter search optimizer is not registered: mode=%s", mode)
	}
	sets, err := optimizer.ParameterSets(spec)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	if len(sets) == 0 {
		return nil, errb.New("parameter search optimizer produced no parameter sets")
	}
	return sets, nil
}

func (r EvaluationSearchOptimizerRegistry) optimizer(mode string) (EvaluationSearchOptimizer, bool) {
	for candidate, optimizer := range r {
		if strings.ToLower(strings.TrimSpace(candidate)) != mode || optimizer == nil {
			continue
		}
		return optimizer, true
	}
	return nil, false
}

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
	ObjectiveValue    float64            `json:"objective_value"`
}

type RegimeSplitResult struct {
	Tag              string  `json:"tag"`
	CaseCount        int     `json:"case_count"`
	PassedCount      int     `json:"passed_count"`
	Objective        string  `json:"objective,omitempty"`
	BestCaseID       string  `json:"best_case_id,omitempty"`
	BestObjective    float64 `json:"best_objective"`
	AverageObjective float64 `json:"average_objective"`
	AverageMetrics   Metrics `json:"average_metrics,omitempty"`
}

type RobustnessReport struct {
	ParameterSensitivity   []ParameterSensitivityResult  `json:"parameter_sensitivity,omitempty"`
	TopNStability          TopNStabilityResult           `json:"top_n_stability,omitempty"`
	OutOfSampleDegradation *OutOfSampleDegradationResult `json:"out_of_sample_degradation,omitempty"`
}

type ParameterSensitivityResult struct {
	Parameter      string                      `json:"parameter"`
	CaseCount      int                         `json:"case_count"`
	ValueCount     int                         `json:"value_count"`
	Objective      string                      `json:"objective,omitempty"`
	BestValue      string                      `json:"best_value,omitempty"`
	BestCaseID     string                      `json:"best_case_id,omitempty"`
	BestObjective  float64                     `json:"best_objective"`
	WorstValue     string                      `json:"worst_value,omitempty"`
	WorstObjective float64                     `json:"worst_objective"`
	ObjectiveRange float64                     `json:"objective_range"`
	Values         []ParameterSensitivityValue `json:"values,omitempty"`
}

type ParameterSensitivityValue struct {
	Value            string  `json:"value"`
	CaseCount        int     `json:"case_count"`
	PassedCount      int     `json:"passed_count"`
	AverageObjective float64 `json:"average_objective"`
	BestObjective    float64 `json:"best_objective"`
	BestCaseID       string  `json:"best_case_id,omitempty"`
}

type TopNStabilityResult struct {
	TopN           int                 `json:"top_n"`
	PeriodCount    int                 `json:"period_count"`
	ComparedPairs  int                 `json:"compared_pairs"`
	AverageOverlap float64             `json:"average_overlap"`
	Pairs          []TopNStabilityPair `json:"pairs,omitempty"`
}

type TopNStabilityPair struct {
	LeftPeriod  string  `json:"left_period"`
	RightPeriod string  `json:"right_period"`
	Overlap     float64 `json:"overlap"`
}

type OutOfSampleDegradationResult struct {
	Objective             string  `json:"objective,omitempty"`
	StepCount             int     `json:"step_count"`
	AverageTrainObjective float64 `json:"average_train_objective"`
	AverageTestObjective  float64 `json:"average_test_objective"`
	AverageDegradation    float64 `json:"average_degradation"`
	DegradationPct        float64 `json:"degradation_pct"`
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
	Index              int             `json:"index"`
	Train              Period          `json:"train"`
	Test               Period          `json:"test"`
	SelectedParameters map[string]any  `json:"selected_parameters,omitempty"`
	TrainCaseID        string          `json:"train_case_id"`
	TestCaseID         string          `json:"test_case_id"`
	TrainObjective     float64         `json:"train_objective"`
	TestMetrics        Metrics         `json:"test_metrics"`
	Runtime            RuntimeMetadata `json:"runtime"`
	DataFingerprint    string          `json:"data_fingerprint"`
	ResultHash         string          `json:"result_hash"`
}

type EvaluationResult struct {
	Name        string                  `json:"name"`
	Strategy    string                  `json:"strategy"`
	BaseRun     string                  `json:"base_run"`
	Cases       []EvaluationCaseResult  `json:"cases"`
	Ranking     []EvaluationCaseResult  `json:"ranking"`
	RegimeSplit []RegimeSplitResult     `json:"regime_split,omitempty"`
	WalkForward []WalkForwardStepResult `json:"walk_forward,omitempty"`
}

func CompileEvaluation(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec, registry IndicatorRegistry) (EvaluationPlan, error) {
	return CompileEvaluationWithSearchOptimizers(strategy, baseRun, evaluation, registry, DefaultEvaluationSearchOptimizerRegistry())
}

func CompileEvaluationWithSearchOptimizers(strategy StrategySpec, baseRun BacktestRunSpec, evaluation EvaluationSpec, registry IndicatorRegistry, optimizers EvaluationSearchOptimizerRegistry) (EvaluationPlan, error) {
	errb := oops.In("backtest_evaluation").With("evaluation", evaluation.Name)
	if err := validateEvaluationSpec(strategy, baseRun, evaluation); err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	evaluation = evaluationWithRequiredMetricDependencies(evaluation)
	periods, err := evaluationPeriods(evaluation.Periods, baseRun.Data)
	if err != nil {
		return EvaluationPlan{}, errb.Wrap(err)
	}
	parameterSets, err := evaluationParameterSetsWithSearchOptimizers(evaluation, optimizers)
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
		value, err := rankingObjectiveValue(result.Metrics, ranking)
		if err != nil {
			return nil, oops.In("backtest_evaluation").With("objective", objective, "case", result.CaseID).Wrap(err)
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
	if constraints.MaxExposureLTE != nil {
		out = append(out, compareLTE("max_exposure_lte", metrics[MetricExposure], *constraints.MaxExposureLTE))
	}
	if constraints.MaxUnfilledCountLTE != nil {
		out = append(out, compareLTE("max_unfilled_count_lte", metrics[MetricUnfilledCount], *constraints.MaxUnfilledCountLTE))
	}
	if constraints.MaxDataIssueCountLTE != nil {
		out = append(out, compareLTE("max_data_issue_count_lte", metrics[MetricDataIssueCount], *constraints.MaxDataIssueCountLTE))
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
	return RegimeTagsWithSpec(result, benchmark, EvaluationRegimeSpec{})
}

func RegimeTagsWithSpec(result Result, benchmark []Bar, spec EvaluationRegimeSpec) []string {
	returnThreshold := withDefaultPositive(spec.ReturnThreshold, 0.05)
	volatilityThreshold := withDefaultPositive(spec.VolatilityThreshold, 0.2)
	tags := make([]string, 0, 2)
	if len(benchmark) >= 2 && benchmark[0].Close > 0 {
		ret := benchmark[len(benchmark)-1].Close/benchmark[0].Close - 1
		switch {
		case ret > returnThreshold:
			tags = append(tags, "bull")
		case ret < -returnThreshold:
			tags = append(tags, "bear")
		default:
			tags = append(tags, "sideways")
		}
	} else {
		switch {
		case result.TotalReturn > returnThreshold:
			tags = append(tags, "bull")
		case result.TotalReturn < -returnThreshold:
			tags = append(tags, "bear")
		default:
			tags = append(tags, "sideways")
		}
	}
	if volatility(result) >= volatilityThreshold {
		tags = append(tags, "high_vol")
	} else {
		tags = append(tags, "low_vol")
	}
	return tags
}

func withDefaultPositive(value float64, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func BuildRegimeSplit(results []EvaluationCaseResult) []RegimeSplitResult {
	type aggregate struct {
		result          RegimeSplitResult
		objectiveTotal  float64
		metricTotals    Metrics
		metricCounts    map[string]int
		bestRank        int
		bestHasRank     bool
		bestInitialized bool
	}
	aggregates := map[string]*aggregate{}
	for _, result := range results {
		tags := result.RegimeTags
		if len(tags) == 0 {
			tags = []string{"unclassified"}
		}
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				tag = "unclassified"
			}
			item := aggregates[tag]
			if item == nil {
				item = &aggregate{
					result:       RegimeSplitResult{Tag: tag, Objective: result.Objective, AverageMetrics: Metrics{}},
					metricTotals: Metrics{},
					metricCounts: map[string]int{},
				}
				aggregates[tag] = item
			}
			if item.result.Objective == "" {
				item.result.Objective = result.Objective
			}
			item.result.CaseCount++
			if result.PassedConstraints {
				item.result.PassedCount++
			}
			item.objectiveTotal += result.ObjectiveValue
			for metric, value := range result.Metrics {
				item.metricTotals[metric] += value
				item.metricCounts[metric]++
			}
			if betterRegimeCase(result, item.result.BestCaseID, item.result.BestObjective, item.bestRank, item.bestHasRank, item.bestInitialized) {
				item.result.BestCaseID = result.CaseID
				item.result.BestObjective = result.ObjectiveValue
				item.bestRank = result.Rank
				item.bestHasRank = result.Rank > 0
				item.bestInitialized = true
			}
		}
	}
	tags := make([]string, 0, len(aggregates))
	for tag := range aggregates {
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	out := make([]RegimeSplitResult, 0, len(tags))
	for _, tag := range tags {
		item := aggregates[tag]
		if item.result.CaseCount > 0 {
			item.result.AverageObjective = item.objectiveTotal / float64(item.result.CaseCount)
		}
		metricIDs := make([]string, 0, len(item.metricTotals))
		for metric := range item.metricTotals {
			metricIDs = append(metricIDs, metric)
		}
		slices.Sort(metricIDs)
		for _, metric := range metricIDs {
			count := item.metricCounts[metric]
			if count > 0 {
				item.result.AverageMetrics[metric] = item.metricTotals[metric] / float64(count)
			}
		}
		out = append(out, item.result)
	}
	return out
}

func BuildRobustnessReport(results []EvaluationCaseResult, walkForward []WalkForwardStepResult, ranking EvaluationRankingSpec, topN int) RobustnessReport {
	if topN <= 0 {
		topN = 3
	}
	return RobustnessReport{
		ParameterSensitivity:   buildParameterSensitivity(results, ranking),
		TopNStability:          buildTopNStability(results, ranking, topN),
		OutOfSampleDegradation: buildOutOfSampleDegradation(walkForward, ranking),
	}
}

func buildParameterSensitivity(results []EvaluationCaseResult, ranking EvaluationRankingSpec) []ParameterSensitivityResult {
	type valueAggregate struct {
		value          string
		caseCount      int
		passedCount    int
		objectiveTotal float64
		bestCaseID     string
		bestObjective  float64
		bestSet        bool
	}
	type parameterAggregate struct {
		parameter string
		values    map[string]*valueAggregate
	}
	parameters := map[string]*parameterAggregate{}
	for _, result := range results {
		objective := robustnessCaseObjective(result, ranking)
		for parameter, rawValue := range result.Parameters {
			parameter = strings.TrimSpace(parameter)
			if parameter == "" {
				continue
			}
			value := parameterValueKey(rawValue)
			parameterAgg := parameters[parameter]
			if parameterAgg == nil {
				parameterAgg = &parameterAggregate{parameter: parameter, values: map[string]*valueAggregate{}}
				parameters[parameter] = parameterAgg
			}
			valueAgg := parameterAgg.values[value]
			if valueAgg == nil {
				valueAgg = &valueAggregate{value: value}
				parameterAgg.values[value] = valueAgg
			}
			valueAgg.caseCount++
			if result.PassedConstraints {
				valueAgg.passedCount++
			}
			valueAgg.objectiveTotal += objective
			if !valueAgg.bestSet || robustnessBetterObjective(objective, result.CaseID, valueAgg.bestObjective, valueAgg.bestCaseID, ranking) {
				valueAgg.bestSet = true
				valueAgg.bestObjective = objective
				valueAgg.bestCaseID = result.CaseID
			}
		}
	}
	parameterNames := make([]string, 0, len(parameters))
	for parameter := range parameters {
		parameterNames = append(parameterNames, parameter)
	}
	slices.Sort(parameterNames)
	out := make([]ParameterSensitivityResult, 0, len(parameterNames))
	for _, parameter := range parameterNames {
		parameterAgg := parameters[parameter]
		valueKeys := make([]string, 0, len(parameterAgg.values))
		for value := range parameterAgg.values {
			valueKeys = append(valueKeys, value)
		}
		slices.Sort(valueKeys)
		values := make([]ParameterSensitivityValue, 0, len(valueKeys))
		result := ParameterSensitivityResult{
			Parameter:  parameter,
			ValueCount: len(valueKeys),
			Objective:  withDefault(strings.TrimSpace(ranking.Objective), MetricCalmar),
		}
		for _, value := range valueKeys {
			valueAgg := parameterAgg.values[value]
			averageObjective := 0.0
			if valueAgg.caseCount > 0 {
				averageObjective = valueAgg.objectiveTotal / float64(valueAgg.caseCount)
			}
			values = append(values, ParameterSensitivityValue{
				Value:            value,
				CaseCount:        valueAgg.caseCount,
				PassedCount:      valueAgg.passedCount,
				AverageObjective: averageObjective,
				BestObjective:    valueAgg.bestObjective,
				BestCaseID:       valueAgg.bestCaseID,
			})
			result.CaseCount += valueAgg.caseCount
			if result.BestValue == "" || robustnessBetterObjective(averageObjective, value, result.BestObjective, result.BestValue, ranking) {
				result.BestValue = value
				result.BestObjective = averageObjective
				result.BestCaseID = valueAgg.bestCaseID
			}
			if result.WorstValue == "" || robustnessWorseObjective(averageObjective, value, result.WorstObjective, result.WorstValue, ranking) {
				result.WorstValue = value
				result.WorstObjective = averageObjective
			}
		}
		result.ObjectiveRange = math.Abs(result.BestObjective - result.WorstObjective)
		result.Values = values
		out = append(out, result)
	}
	return out
}

func buildTopNStability(results []EvaluationCaseResult, ranking EvaluationRankingSpec, topN int) TopNStabilityResult {
	type periodGroup struct {
		label string
		cases []EvaluationCaseResult
	}
	periods := map[string]*periodGroup{}
	for _, result := range results {
		label := periodLabel(result.Period)
		group := periods[label]
		if group == nil {
			group = &periodGroup{label: label}
			periods[label] = group
		}
		group.cases = append(group.cases, result)
	}
	labels := make([]string, 0, len(periods))
	for label := range periods {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	selected := make([]map[string]struct{}, 0, len(labels))
	for _, label := range labels {
		cases := append([]EvaluationCaseResult(nil), periods[label].cases...)
		sortRobustnessCases(cases, ranking)
		limit := topN
		if len(cases) < limit {
			limit = len(cases)
		}
		keys := map[string]struct{}{}
		for i := 0; i < limit; i++ {
			keys[parameterSetKey(cases[i].Parameters)] = struct{}{}
		}
		selected = append(selected, keys)
	}
	out := TopNStabilityResult{TopN: topN, PeriodCount: len(labels)}
	for i := 1; i < len(labels); i++ {
		overlap := setOverlapRatio(selected[i-1], selected[i])
		out.Pairs = append(out.Pairs, TopNStabilityPair{
			LeftPeriod:  labels[i-1],
			RightPeriod: labels[i],
			Overlap:     overlap,
		})
		out.AverageOverlap += overlap
		out.ComparedPairs++
	}
	if out.ComparedPairs > 0 {
		out.AverageOverlap /= float64(out.ComparedPairs)
	}
	return out
}

func buildOutOfSampleDegradation(walkForward []WalkForwardStepResult, ranking EvaluationRankingSpec) *OutOfSampleDegradationResult {
	if len(walkForward) == 0 {
		return nil
	}
	objective := withDefault(strings.TrimSpace(ranking.Objective), MetricCalmar)
	out := OutOfSampleDegradationResult{Objective: objective}
	for _, step := range walkForward {
		testObjective, ok := step.TestMetrics[objective]
		if !ok {
			continue
		}
		out.StepCount++
		out.AverageTrainObjective += step.TrainObjective
		out.AverageTestObjective += testObjective
		out.AverageDegradation += step.TrainObjective - testObjective
	}
	if out.StepCount == 0 {
		return nil
	}
	out.AverageTrainObjective /= float64(out.StepCount)
	out.AverageTestObjective /= float64(out.StepCount)
	out.AverageDegradation /= float64(out.StepCount)
	if out.AverageTrainObjective != 0 {
		out.DegradationPct = out.AverageDegradation / math.Abs(out.AverageTrainObjective)
	}
	return &out
}

func robustnessCaseObjective(result EvaluationCaseResult, ranking EvaluationRankingSpec) float64 {
	if result.Objective != "" || result.ObjectiveValue != 0 {
		return result.ObjectiveValue
	}
	value, err := rankingObjectiveValue(result.Metrics, ranking)
	if err != nil {
		return 0
	}
	return value
}

func sortRobustnessCases(results []EvaluationCaseResult, ranking EvaluationRankingSpec) {
	slices.SortFunc(results, func(a, b EvaluationCaseResult) int {
		aValue := robustnessCaseObjective(a, ranking)
		bValue := robustnessCaseObjective(b, ranking)
		if aValue == bValue {
			return strings.Compare(a.CaseID, b.CaseID)
		}
		if strings.TrimSpace(ranking.Order) == RankingOrderAsc {
			if aValue < bValue {
				return -1
			}
			return 1
		}
		if aValue > bValue {
			return -1
		}
		return 1
	})
}

func robustnessBetterObjective(candidate float64, candidateTie string, best float64, bestTie string, ranking EvaluationRankingSpec) bool {
	if candidate == best {
		return candidateTie < bestTie
	}
	if strings.TrimSpace(ranking.Order) == RankingOrderAsc {
		return candidate < best
	}
	return candidate > best
}

func robustnessWorseObjective(candidate float64, candidateTie string, worst float64, worstTie string, ranking EvaluationRankingSpec) bool {
	if candidate == worst {
		return candidateTie < worstTie
	}
	if strings.TrimSpace(ranking.Order) == RankingOrderAsc {
		return candidate > worst
	}
	return candidate < worst
}

func parameterValueKey(value any) string {
	switch typed := normalizeParameterValue(value).(type) {
	case string:
		return typed
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return strings.TrimSpace(strconv.FormatFloat(0, 'f', -1, 64))
		}
		return string(payload)
	}
}

func parameterSetKey(parameters map[string]any) string {
	if len(parameters) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+parameterValueKey(parameters[key]))
	}
	return strings.Join(parts, ";")
}

func periodLabel(period Period) string {
	if period.From.IsZero() && period.To.IsZero() {
		return "all"
	}
	return period.From.Format(time.DateOnly) + "_" + period.To.Format(time.DateOnly)
}

func setOverlapRatio(left map[string]struct{}, right map[string]struct{}) float64 {
	denominator := len(left)
	if len(right) > denominator {
		denominator = len(right)
	}
	if denominator == 0 {
		return 0
	}
	var intersection int
	for key := range left {
		if _, ok := right[key]; ok {
			intersection++
		}
	}
	return float64(intersection) / float64(denominator)
}

func betterRegimeCase(candidate EvaluationCaseResult, bestCaseID string, bestObjective float64, bestRank int, bestHasRank bool, initialized bool) bool {
	if !initialized {
		return true
	}
	candidateHasRank := candidate.Rank > 0
	if candidateHasRank || bestHasRank {
		if candidateHasRank != bestHasRank {
			return candidateHasRank
		}
		if candidate.Rank != bestRank {
			return candidate.Rank < bestRank
		}
		return candidate.CaseID < bestCaseID
	}
	if candidate.ObjectiveValue != bestObjective {
		return candidate.ObjectiveValue > bestObjective
	}
	return candidate.CaseID < bestCaseID
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
		if err := validateRankingObjective(evaluation.Ranking); err != nil {
			return errb.Wrap(err)
		}
	}
	if !evaluation.WalkForward.Select.Empty() {
		if err := validateRankingObjective(EvaluationRankingSpec{
			Objective: evaluation.WalkForward.Select.Objective,
			Order:     evaluation.WalkForward.Select.Order,
			Weights:   evaluation.WalkForward.Select.Weights,
		}); err != nil {
			return errb.Wrap(err)
		}
	}
	if strings.TrimSpace(evaluation.Periods.Mode) == EvaluationPeriodWalkForward {
		if evaluation.WalkForward.Train.Empty() || evaluation.WalkForward.Test.Empty() {
			return errb.New("walk_forward periods mode requires walk_forward train and test windows")
		}
	}
	if evaluation.Execution.Parallelism < 0 {
		return errb.With("parallelism", evaluation.Execution.Parallelism).New("evaluation execution parallelism must not be negative")
	}
	return nil
}

func validateRankingObjective(ranking EvaluationRankingSpec) error {
	objective := strings.TrimSpace(ranking.Objective)
	if objective == "" {
		return nil
	}
	if objective != EvaluationObjectiveWeightedScore {
		_, err := requireMetricMust(objective)
		return err
	}
	if len(ranking.Weights) == 0 {
		return oops.In("backtest_evaluation").With("objective", objective).New("weighted objective requires weights")
	}
	for metric, weight := range ranking.Weights {
		if strings.TrimSpace(metric) == "" {
			return oops.In("backtest_evaluation").With("objective", objective).New("weighted objective metric is empty")
		}
		if weight == 0 {
			return oops.In("backtest_evaluation").With("objective", objective, "metric", metric).New("weighted objective weight must not be zero")
		}
		if _, err := requireMetricMust(metric); err != nil {
			return err
		}
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
	case EvaluationPeriodExpanding:
		window := spec.Window
		if window.Empty() && spec.WindowDays > 0 {
			window.Days = spec.WindowDays
		}
		step := spec.Step
		if step.Empty() && spec.StepDays > 0 {
			step.Days = spec.StepDays
		}
		return expandingPeriods(from, to, window, step)
	case EvaluationPeriodWalkForward:
		return nil, nil
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

func expandingPeriods(from, to time.Time, window DurationSpec, step DurationSpec) ([]EvaluationPeriodSpec, error) {
	if window.Empty() {
		return nil, oops.In("backtest_evaluation_periods").New("expanding periods require window")
	}
	if step.Empty() {
		step = window
	}
	out := make([]EvaluationPeriodSpec, 0)
	for index, end := 1, addDuration(from, window).AddDate(0, 0, -1); !end.After(to); index, end = index+1, addDuration(end, step) {
		out = append(out, EvaluationPeriodSpec{Name: "expanding-" + strconv.Itoa(index), From: from.Format(time.DateOnly), To: end.Format(time.DateOnly)})
	}
	if len(out) == 0 {
		return nil, oops.In("backtest_evaluation_periods").New("expanding periods produced no cases")
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
			caseRun.Benchmark = evaluationCaseBenchmark(caseRun.Benchmark, evaluation.Regime.Benchmark)
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

func evaluationCaseBenchmark(runBenchmark BenchmarkSpec, regimeBenchmark BenchmarkSpec) BenchmarkSpec {
	if strings.TrimSpace(runBenchmark.Symbol) != "" || strings.TrimSpace(regimeBenchmark.Symbol) == "" {
		return runBenchmark
	}
	return regimeBenchmark
}

func evaluationParameterSets(evaluation EvaluationSpec) ([]map[string]any, error) {
	return evaluationParameterSetsWithSearchOptimizers(evaluation, DefaultEvaluationSearchOptimizerRegistry())
}

func evaluationParameterSetsWithSearchOptimizers(evaluation EvaluationSpec, optimizers EvaluationSearchOptimizerRegistry) ([]map[string]any, error) {
	if !evaluation.Search.Empty() {
		if len(evaluation.Parameters) > 0 {
			return nil, oops.In("backtest_evaluation_parameters").New("cannot use parameters and search together")
		}
		return parameterSearchWithOptimizers(evaluation.Search, optimizers)
	}
	return parameterGrid(evaluation.Parameters)
}

func parameterSearch(spec EvaluationSearchSpec) ([]map[string]any, error) {
	return parameterSearchWithOptimizers(spec, DefaultEvaluationSearchOptimizerRegistry())
}

func parameterSearchWithOptimizers(spec EvaluationSearchSpec, optimizers EvaluationSearchOptimizerRegistry) ([]map[string]any, error) {
	return optimizers.ParameterSets(spec)
}

func boundedParameterSearch(spec EvaluationSearchSpec) ([]map[string]any, error) {
	grid, err := searchParameterGrid(spec.Parameters)
	if err != nil {
		return nil, err
	}
	return parameterGrid(grid)
}

func randomParameterSearch(spec EvaluationSearchSpec) ([]map[string]any, error) {
	errb := oops.In("backtest_evaluation_parameters")
	if spec.Samples <= 0 {
		return nil, errb.With("samples", spec.Samples).New("random parameter search requires positive samples")
	}
	if len(spec.Parameters) == 0 {
		return nil, errb.New("random parameter search requires parameters")
	}
	keys := make([]string, 0, len(spec.Parameters))
	for key := range spec.Parameters {
		if strings.TrimSpace(key) == "" {
			return nil, errb.New("parameter search path is empty")
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	rng := rand.New(rand.NewSource(spec.Seed))
	out := make([]map[string]any, 0, spec.Samples)
	for index := 0; index < spec.Samples; index++ {
		item := make(map[string]any, len(keys))
		for _, key := range keys {
			value, err := randomParameterValue(rng, key, spec.Parameters[key])
			if err != nil {
				return nil, err
			}
			item[key] = normalizeParameterValue(value)
		}
		out = append(out, item)
	}
	return out, nil
}

func searchParameterGrid(parameters map[string]EvaluationSearchParameterSpec) (map[string][]any, error) {
	if len(parameters) == 0 {
		return nil, oops.In("backtest_evaluation_parameters").New("bounded parameter search requires parameters")
	}
	out := make(map[string][]any, len(parameters))
	for path, spec := range parameters {
		values, err := searchParameterValues(path, spec)
		if err != nil {
			return nil, err
		}
		out[path] = values
	}
	return out, nil
}

func searchParameterValues(path string, spec EvaluationSearchParameterSpec) ([]any, error) {
	errb := oops.In("backtest_evaluation_parameters").With("path", path)
	if strings.TrimSpace(path) == "" {
		return nil, oops.In("backtest_evaluation_parameters").New("parameter search path is empty")
	}
	if len(spec.Values) > 0 {
		out := make([]any, 0, len(spec.Values))
		for _, value := range spec.Values {
			out = append(out, normalizeParameterValue(value))
		}
		return out, nil
	}
	if spec.Min == nil || spec.Max == nil {
		return nil, errb.New("parameter search range requires min and max")
	}
	step := 1.0
	if spec.Step != nil {
		step = *spec.Step
	}
	if step <= 0 {
		return nil, errb.With("step", step).New("parameter search step must be positive")
	}
	if *spec.Max < *spec.Min {
		return nil, errb.With("min", *spec.Min, "max", *spec.Max).New("parameter search max must be greater than or equal to min")
	}
	out := make([]any, 0)
	for value := *spec.Min; value <= *spec.Max+step/1_000_000; value += step {
		if value > *spec.Max {
			value = *spec.Max
		}
		out = append(out, value)
		if value == *spec.Max {
			break
		}
	}
	return out, nil
}

func randomParameterValue(rng *rand.Rand, path string, spec EvaluationSearchParameterSpec) (any, error) {
	values, err := searchParameterValues(path, spec)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, oops.In("backtest_evaluation_parameters").With("path", path).New("parameter search values are empty")
	}
	return values[rng.Intn(len(values))], nil
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

func rankingObjectiveValue(metrics Metrics, ranking EvaluationRankingSpec) (float64, error) {
	objective := withDefault(strings.TrimSpace(ranking.Objective), MetricCalmar)
	if objective != EvaluationObjectiveWeightedScore {
		value, ok := metricValue(metrics, objective)
		if !ok {
			return 0, oops.In("backtest_evaluation").With("metric", objective).New("ranking objective metric is missing")
		}
		return value, nil
	}
	if len(ranking.Weights) == 0 {
		return 0, oops.In("backtest_evaluation").With("objective", objective).New("weighted objective requires weights")
	}
	value := 0.0
	keys := make([]string, 0, len(ranking.Weights))
	for metric := range ranking.Weights {
		keys = append(keys, metric)
	}
	slices.Sort(keys)
	for _, metric := range keys {
		metricValue, ok := metrics[metric]
		if !ok {
			return 0, oops.In("backtest_evaluation").With("metric", metric).New("weighted objective metric is missing")
		}
		value += metricValue * ranking.Weights[metric]
	}
	return value, nil
}

func evaluationWithRequiredMetricDependencies(evaluation EvaluationSpec) EvaluationSpec {
	metrics := make([]string, 0)
	if objective := withDefault(strings.TrimSpace(evaluation.Ranking.Objective), MetricCalmar); objective != EvaluationObjectiveWeightedScore {
		metrics = append(metrics, objective)
	}
	for metric := range evaluation.Ranking.Weights {
		metrics = append(metrics, metric)
	}
	if objective := strings.TrimSpace(evaluation.WalkForward.Select.Objective); objective != "" && objective != EvaluationObjectiveWeightedScore {
		metrics = append(metrics, objective)
	}
	for metric := range evaluation.WalkForward.Select.Weights {
		metrics = append(metrics, metric)
	}
	metrics = append(metrics, metricDependenciesForConstraints(evaluation.Constraints)...)
	metrics = append(metrics, metricDependenciesForConstraints(evaluation.WalkForward.Select.Constraints)...)
	if len(metrics) == 0 {
		return evaluation
	}
	evaluation.Metrics = includeMetricDependencies(evaluation.Metrics, metrics)
	return evaluation
}

func metricDependenciesForConstraints(constraints EvaluationConstraintSet) []string {
	metrics := make([]string, 0, 7)
	if constraints.MaxDrawdownLTE != nil {
		metrics = append(metrics, MetricMaxDrawdown)
	}
	if constraints.MinCAGRGTE != nil {
		metrics = append(metrics, MetricCAGR)
	}
	if constraints.MaxTurnoverLTE != nil {
		metrics = append(metrics, MetricTurnover)
	}
	if constraints.MinTradeCountGTE != nil {
		metrics = append(metrics, MetricTradeCount)
	}
	if constraints.MaxExposureLTE != nil {
		metrics = append(metrics, MetricExposure)
	}
	if constraints.MaxUnfilledCountLTE != nil {
		metrics = append(metrics, MetricUnfilledCount)
	}
	if constraints.MaxDataIssueCountLTE != nil {
		metrics = append(metrics, MetricDataIssueCount)
	}
	return metrics
}

func includeMetricDependencies(selection MetricSelectionSpec, metrics []string) MetricSelectionSpec {
	seen := make(map[string]struct{}, len(selection.Include)+len(metrics))
	for _, metric := range selection.Include {
		seen[metric] = struct{}{}
	}
	for _, metric := range metrics {
		if _, ok := seen[metric]; ok {
			continue
		}
		selection.Include = append(selection.Include, metric)
		seen[metric] = struct{}{}
	}
	if len(selection.Exclude) == 0 {
		return selection
	}
	required := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		required[metric] = struct{}{}
	}
	filtered := selection.Exclude[:0]
	for _, metric := range selection.Exclude {
		if _, ok := required[metric]; ok {
			continue
		}
		filtered = append(filtered, metric)
	}
	selection.Exclude = filtered
	return selection
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

func (s EvaluationSearchSpec) Empty() bool {
	return strings.TrimSpace(s.Mode) == "" && s.Seed == 0 && s.Samples == 0 && s.InitialSamples == 0 && strings.TrimSpace(s.Acquisition) == "" && len(s.Parameters) == 0
}

func (s WalkForwardSelectionSpec) Empty() bool {
	return strings.TrimSpace(s.Objective) == "" && strings.TrimSpace(s.Order) == "" && len(s.Weights) == 0 && s.Constraints.Empty()
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
	weights := spec.Ranking.Weights
	if len(spec.WalkForward.Select.Weights) > 0 {
		weights = spec.WalkForward.Select.Weights
	}
	return EvaluationRankingSpec{Objective: objective, Order: order, Weights: weights}
}

func WalkForwardConstraints(spec EvaluationSpec) EvaluationConstraintSet {
	if spec.WalkForward.Select.Constraints.Empty() {
		return spec.Constraints
	}
	return spec.WalkForward.Select.Constraints
}

func (c EvaluationConstraintSet) Empty() bool {
	return c.MaxDrawdownLTE == nil &&
		c.MinCAGRGTE == nil &&
		c.MaxTurnoverLTE == nil &&
		c.MinTradeCountGTE == nil &&
		c.MaxExposureLTE == nil &&
		c.MaxUnfilledCountLTE == nil &&
		c.MaxDataIssueCountLTE == nil
}

func RunCase(ctx context.Context, engine Engine, plan StrategyPlan) (Result, error) {
	return engine.Run(ctx, plan)
}
