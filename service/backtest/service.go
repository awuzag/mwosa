package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	"github.com/awuzag/mwosa/packages/hashutil"
	"github.com/awuzag/mwosa/packages/idgen"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	strategyservice "github.com/awuzag/mwosa/service/strategy"
	universeservice "github.com/awuzag/mwosa/service/universe"
	"github.com/samber/oops"
)

type DailyBarRepository interface {
	QueryDailyBars(ctx context.Context, query daily.Query) ([]dailybar.Bar, error)
	StreamDailyBars(ctx context.Context, query daily.Query) (daily.BarStream, error)
}

type StrategyRepository interface {
	ListStrategies(ctx context.Context) ([]SavedStrategyDetail, error)
	GetStrategy(ctx context.Context, name string) (SavedStrategyDetail, error)
	UpsertStrategyWithVersion(ctx context.Context, strategy SavedStrategy, version SavedStrategyVersion, now time.Time) (SavedStrategyDetail, error)
	DeleteStrategy(ctx context.Context, name string, deletedAt time.Time) error
}

type EvaluationRepository interface {
	SaveEvaluation(ctx context.Context, experiment SavedExperiment, cases []SavedExperimentCase, steps []SavedWalkForwardStep, now time.Time) (SavedEvaluationDetail, error)
	ListEvaluations(ctx context.Context) ([]SavedEvaluationSummary, error)
	GetEvaluation(ctx context.Context, ref string) (SavedEvaluationDetail, error)
}

type BacktestRunRepository interface {
	SaveRun(ctx context.Context, run SavedBacktestRun, now time.Time) (SavedBacktestRunDetail, error)
	ListRuns(ctx context.Context) ([]SavedBacktestRun, error)
	GetRun(ctx context.Context, ref string) (SavedBacktestRunDetail, error)
}

type ScreenRepository interface {
	GetScreenRun(ctx context.Context, ref string) (strategyservice.ScreenRunDetail, error)
}

type ScreenRunner interface {
	Screen(ctx context.Context, req strategyservice.ScreenStrategyRequest) (strategyservice.ScreenRunDetail, error)
}

type Service struct {
	reader       DailyBarRepository
	strategies   StrategyRepository
	evaluations  EvaluationRepository
	runs         BacktestRunRepository
	screenRepo   ScreenRepository
	screenRunner ScreenRunner
	universe     universeservice.Runner
	registry     core.IndicatorRegistry
}

type SavedStrategy struct {
	ID              string     `json:"id" csv:"id"`
	Name            string     `json:"name" csv:"name"`
	ActiveVersionID string     `json:"active_version_id" csv:"active_version_id"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" csv:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" csv:"deleted_at"`
}

type SavedStrategyVersion struct {
	ID            string          `json:"id" csv:"id"`
	StrategyID    string          `json:"strategy_id" csv:"strategy_id"`
	Version       int             `json:"version" csv:"version"`
	SchemaVersion int             `json:"schema_version" csv:"schema_version"`
	SpecJSON      json.RawMessage `json:"spec_json" csv:"-"`
	SpecHash      string          `json:"spec_hash" csv:"spec_hash"`
	CreatedAt     time.Time       `json:"created_at" csv:"created_at"`
}

type SavedStrategyDetail struct {
	Strategy      SavedStrategy        `json:"strategy"`
	ActiveVersion SavedStrategyVersion `json:"active_version"`
	Spec          core.StrategySpec    `json:"spec"`
}

type SavedExperiment struct {
	ID               string          `json:"id" csv:"id"`
	Name             string          `json:"name" csv:"name"`
	StrategyName     string          `json:"strategy_name" csv:"strategy_name"`
	BaseRunName      string          `json:"base_run_name" csv:"base_run_name"`
	SchemaVersion    int             `json:"schema_version" csv:"schema_version"`
	SpecJSON         json.RawMessage `json:"spec_json" csv:"-"`
	SpecHash         string          `json:"spec_hash" csv:"spec_hash"`
	StrategySpecHash string          `json:"strategy_spec_hash" csv:"strategy_spec_hash"`
	DataFrom         string          `json:"data_from" csv:"data_from"`
	DataTo           string          `json:"data_to" csv:"data_to"`
	CreatedAt        time.Time       `json:"created_at" csv:"created_at"`
}

type SavedExperimentCase struct {
	ID                string          `json:"id" csv:"id"`
	ExperimentID      string          `json:"experiment_id" csv:"experiment_id"`
	CaseID            string          `json:"case_id" csv:"case_id"`
	CaseName          string          `json:"case_name" csv:"case_name"`
	RunName           string          `json:"run_name" csv:"run_name"`
	PeriodFrom        string          `json:"period_from" csv:"period_from"`
	PeriodTo          string          `json:"period_to" csv:"period_to"`
	ParameterJSON     json.RawMessage `json:"parameter_json" csv:"-"`
	RegimeTagsJSON    json.RawMessage `json:"regime_tags_json" csv:"-"`
	Status            string          `json:"status" csv:"status"`
	PassedConstraints bool            `json:"passed_constraints" csv:"passed_constraints"`
	Rank              int             `json:"rank,omitempty" csv:"rank"`
	Objective         string          `json:"objective,omitempty" csv:"objective"`
	ObjectiveValue    float64         `json:"objective_value" csv:"objective_value"`
	ResultJSON        json.RawMessage `json:"result_json" csv:"-"`
	MetricsJSON       json.RawMessage `json:"metrics_json" csv:"-"`
	StrategyHash      string          `json:"strategy_hash" csv:"strategy_hash"`
	RunHash           string          `json:"run_hash" csv:"run_hash"`
	EngineVersion     string          `json:"engine_version" csv:"engine_version"`
	IndicatorRegistry string          `json:"indicator_registry_version" csv:"indicator_registry_version"`
	MetricRegistry    string          `json:"metric_registry_version" csv:"metric_registry_version"`
	DataFingerprint   string          `json:"data_fingerprint" csv:"data_fingerprint"`
	ResultHash        string          `json:"result_hash" csv:"result_hash"`
	CreatedAt         time.Time       `json:"created_at" csv:"created_at"`
}

type SavedWalkForwardStep struct {
	ID                    string          `json:"id" csv:"id"`
	ExperimentID          string          `json:"experiment_id" csv:"experiment_id"`
	StepIndex             int             `json:"step_index" csv:"step_index"`
	TrainFrom             string          `json:"train_from" csv:"train_from"`
	TrainTo               string          `json:"train_to" csv:"train_to"`
	TestFrom              string          `json:"test_from" csv:"test_from"`
	TestTo                string          `json:"test_to" csv:"test_to"`
	SelectedParameterJSON json.RawMessage `json:"selected_parameter_json" csv:"-"`
	TrainCaseID           string          `json:"train_case_id" csv:"train_case_id"`
	TestCaseID            string          `json:"test_case_id" csv:"test_case_id"`
	TrainObjective        float64         `json:"train_objective" csv:"train_objective"`
	TestMetricsJSON       json.RawMessage `json:"test_metrics_json" csv:"-"`
	StrategyHash          string          `json:"strategy_hash" csv:"strategy_hash"`
	RunHash               string          `json:"run_hash" csv:"run_hash"`
	EngineVersion         string          `json:"engine_version" csv:"engine_version"`
	IndicatorRegistry     string          `json:"indicator_registry_version" csv:"indicator_registry_version"`
	MetricRegistry        string          `json:"metric_registry_version" csv:"metric_registry_version"`
	DataFingerprint       string          `json:"data_fingerprint" csv:"data_fingerprint"`
	ResultHash            string          `json:"result_hash" csv:"result_hash"`
	CreatedAt             time.Time       `json:"created_at" csv:"created_at"`
}

type SavedBacktestRun struct {
	ID                string          `json:"id" csv:"id"`
	RunName           string          `json:"run_name" csv:"run_name"`
	StrategyName      string          `json:"strategy_name" csv:"strategy_name"`
	Market            string          `json:"market" csv:"market"`
	Timeframe         string          `json:"timeframe" csv:"timeframe"`
	PeriodFrom        string          `json:"period_from" csv:"period_from"`
	PeriodTo          string          `json:"period_to" csv:"period_to"`
	StrategyHash      string          `json:"strategy_hash" csv:"strategy_hash"`
	RunHash           string          `json:"run_hash" csv:"run_hash"`
	EngineVersion     string          `json:"engine_version" csv:"engine_version"`
	IndicatorRegistry string          `json:"indicator_registry_version" csv:"indicator_registry_version"`
	MetricRegistry    string          `json:"metric_registry_version" csv:"metric_registry_version"`
	DataFingerprint   string          `json:"data_fingerprint" csv:"data_fingerprint"`
	ResultHash        string          `json:"result_hash" csv:"result_hash"`
	ResultJSON        json.RawMessage `json:"result_json,omitempty" csv:"-"`
	MetricsJSON       json.RawMessage `json:"metrics_json,omitempty" csv:"-"`
	CreatedAt         time.Time       `json:"created_at" csv:"created_at"`
}

type SavedBacktestRunDetail struct {
	Run    SavedBacktestRun `json:"run"`
	Result core.Result      `json:"result"`
}

type BacktestRunComparison struct {
	Left                SavedBacktestRun        `json:"left"`
	Right               SavedBacktestRun        `json:"right"`
	SameStrategyHash    bool                    `json:"same_strategy_hash"`
	SameRunHash         bool                    `json:"same_run_hash"`
	SameRuntime         bool                    `json:"same_runtime"`
	SameDataFingerprint bool                    `json:"same_data_fingerprint"`
	SameResultHash      bool                    `json:"same_result_hash"`
	Metrics             []BacktestRunMetricDiff `json:"metrics"`
}

type BacktestRunMetricDiff struct {
	Metric       string  `json:"metric" csv:"metric"`
	LeftValue    float64 `json:"left_value" csv:"left_value"`
	RightValue   float64 `json:"right_value" csv:"right_value"`
	Delta        float64 `json:"delta" csv:"delta"`
	DeltaPct     float64 `json:"delta_pct" csv:"delta_pct"`
	LeftPresent  bool    `json:"left_present" csv:"left_present"`
	RightPresent bool    `json:"right_present" csv:"right_present"`
}

type SavedEvaluationSummary struct {
	Experiment SavedExperiment      `json:"experiment"`
	CaseCount  int                  `json:"case_count"`
	BestCase   *SavedExperimentCase `json:"best_case,omitempty"`
}

type SavedEvaluationDetail struct {
	Experiment  SavedExperiment        `json:"experiment"`
	Cases       []SavedExperimentCase  `json:"cases"`
	WalkForward []SavedWalkForwardStep `json:"walk_forward,omitempty"`
}

type SaveStrategyRequest struct {
	Name     string
	YAMLPath string
}

type RunEvaluationOptions struct {
	Parallelism int
}

type ValidationResult struct {
	Valid        bool                      `json:"valid"`
	StrategyName string                    `json:"strategy_name"`
	RunName      string                    `json:"run_name"`
	Symbols      []string                  `json:"symbols"`
	Instruments  []core.InstrumentIdentity `json:"instruments,omitempty"`
	Period       core.Period               `json:"period"`
	Market       string                    `json:"market"`
	Timeframe    string                    `json:"timeframe"`
	Timeframes   core.TimeframeMetadata    `json:"timeframes"`
	Runtime      core.RuntimeMetadata      `json:"runtime"`
	Currency     string                    `json:"currency"`
	Execution    ExecutionSummary          `json:"execution"`
	Metrics      []string                  `json:"metrics"`
	Indicators   map[string]string         `json:"indicators,omitempty"`
	Universe     core.UniverseExplain      `json:"universe"`
}

type EvaluationValidationResult struct {
	Valid            bool     `json:"valid"`
	Name             string   `json:"name"`
	StrategyName     string   `json:"strategy_name"`
	BaseRunName      string   `json:"base_run_name"`
	CaseCount        int      `json:"case_count"`
	WalkForwardSteps int      `json:"walk_forward_steps"`
	Parallelism      int      `json:"parallelism"`
	Metrics          []string `json:"metrics"`
}

type EvaluationRunResult struct {
	Experiment  SavedExperiment              `json:"experiment"`
	Cases       []core.EvaluationCaseResult  `json:"cases"`
	Ranking     []core.EvaluationCaseResult  `json:"ranking"`
	RegimeSplit []core.RegimeSplitResult     `json:"regime_split,omitempty"`
	Robustness  core.RobustnessReport        `json:"robustness,omitempty"`
	WalkForward []core.WalkForwardStepResult `json:"walk_forward,omitempty"`
}

type ExecutionSummary struct {
	Fill                    string               `json:"fill"`
	OrderType               string               `json:"order_type,omitempty"`
	LimitPrice              float64              `json:"limit_price,omitempty"`
	StopPrice               float64              `json:"stop_price,omitempty"`
	TrailingStopPct         float64              `json:"trailing_stop_pct,omitempty"`
	IntrabarAmbiguityPolicy string               `json:"intrabar_ambiguity_policy,omitempty"`
	TimeInForce             string               `json:"time_in_force,omitempty"`
	LotSize                 float64              `json:"lot_size,omitempty"`
	TickSize                float64              `json:"tick_size,omitempty"`
	Commission              core.CostSpec        `json:"commission,omitempty"`
	Tax                     core.CostSpec        `json:"tax,omitempty"`
	ExchangeFee             core.CostSpec        `json:"exchange_fee,omitempty"`
	Slippage                core.CostSpec        `json:"slippage,omitempty"`
	Liquidity               core.LiquiditySpec   `json:"liquidity,omitempty"`
	PartialFill             core.PartialFillSpec `json:"partial_fill,omitempty"`
}

func NewService(reader DailyBarRepository) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, nil, nil, nil, registry)
}

func NewServiceWithRepository(reader DailyBarRepository, strategies StrategyRepository) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, strategies, nil, nil, registry)
}

func NewServiceWithUniverseSources(reader DailyBarRepository, strategies StrategyRepository, screenRepo ScreenRepository, screenRunner ScreenRunner) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, strategies, screenRepo, screenRunner, registry)
}

func NewServiceWithRegistry(reader DailyBarRepository, registry core.IndicatorRegistry) (Service, error) {
	return newService(reader, nil, nil, nil, registry)
}

func newService(reader DailyBarRepository, strategies StrategyRepository, screenRepo ScreenRepository, screenRunner ScreenRunner, registry core.IndicatorRegistry) (Service, error) {
	if reader == nil {
		return Service{}, oops.In("backtest_service").New("daily bar reader is nil")
	}
	universeRunner, err := universeservice.NewRunner(reader, screenRepo, screenRunner)
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrap(err)
	}
	var evaluations EvaluationRepository
	var runs BacktestRunRepository
	if strategies != nil {
		if repository, ok := strategies.(EvaluationRepository); ok {
			evaluations = repository
		}
		if repository, ok := strategies.(BacktestRunRepository); ok {
			runs = repository
		}
	}
	return Service{reader: reader, strategies: strategies, evaluations: evaluations, runs: runs, screenRepo: screenRepo, screenRunner: screenRunner, universe: universeRunner, registry: registry}, nil
}

func (s Service) Validate(ctx context.Context, path string) (ValidationResult, error) {
	_, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return ValidationResult{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return ValidationResult{}, err
	}
	return validationResultFromPlan(plan), nil
}

func (s Service) Run(ctx context.Context, path string) (core.Result, error) {
	_, result, err := s.runFile(ctx, path)
	return result, err
}

func (s Service) RunAndSave(ctx context.Context, path string) (SavedBacktestRunDetail, error) {
	if s.runs == nil {
		return SavedBacktestRunDetail{}, oops.In("backtest_run_service").New("backtest run repository is nil")
	}
	bundle, result, err := s.runFile(ctx, path)
	if err != nil {
		return SavedBacktestRunDetail{}, err
	}
	run, err := savedBacktestRun(bundle, result)
	if err != nil {
		return SavedBacktestRunDetail{}, err
	}
	detail, err := s.runs.SaveRun(ctx, run, time.Now())
	if err != nil {
		return SavedBacktestRunDetail{}, err
	}
	detail.Result = result
	return detail, nil
}

func (s Service) ListRuns(ctx context.Context) ([]SavedBacktestRun, error) {
	if s.runs == nil {
		return nil, oops.In("backtest_run_service").New("backtest run repository is nil")
	}
	return s.runs.ListRuns(ctx)
}

func (s Service) InspectRun(ctx context.Context, ref string) (SavedBacktestRunDetail, error) {
	if strings.TrimSpace(ref) == "" {
		return SavedBacktestRunDetail{}, oops.In("backtest_run_service").New("inspect backtest run requires name, id, or result hash")
	}
	if s.runs == nil {
		return SavedBacktestRunDetail{}, oops.In("backtest_run_service").With("ref", ref).New("backtest run repository is nil")
	}
	return s.runs.GetRun(ctx, ref)
}

func (s Service) CompareRuns(ctx context.Context, leftRef string, rightRef string) (BacktestRunComparison, error) {
	if strings.TrimSpace(leftRef) == "" || strings.TrimSpace(rightRef) == "" {
		return BacktestRunComparison{}, oops.In("backtest_run_service").New("compare backtest runs requires two refs")
	}
	left, err := s.InspectRun(ctx, leftRef)
	if err != nil {
		return BacktestRunComparison{}, err
	}
	right, err := s.InspectRun(ctx, rightRef)
	if err != nil {
		return BacktestRunComparison{}, err
	}
	metrics, err := compareRunMetrics(left, right)
	if err != nil {
		return BacktestRunComparison{}, err
	}
	return BacktestRunComparison{
		Left:                publicSavedRun(left.Run),
		Right:               publicSavedRun(right.Run),
		SameStrategyHash:    left.Run.StrategyHash == right.Run.StrategyHash,
		SameRunHash:         left.Run.RunHash == right.Run.RunHash,
		SameRuntime:         sameRunRuntime(left.Run, right.Run),
		SameDataFingerprint: left.Run.DataFingerprint == right.Run.DataFingerprint,
		SameResultHash:      left.Run.ResultHash == right.Run.ResultHash,
		Metrics:             metrics,
	}, nil
}

func (s Service) runFile(ctx context.Context, path string) (Bundle, core.Result, error) {
	bundle, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return Bundle{}, core.Result{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return Bundle{}, core.Result{}, err
	}
	engine, err := core.NewEngine(newDailyBarFeed(s.reader))
	if err != nil {
		return Bundle{}, core.Result{}, oops.In("backtest_service").With("strategy", bundle.Strategy.Name, "run", bundle.Run.Name).Wrap(err)
	}
	result, err := engine.Run(ctx, plan)
	if err != nil {
		return Bundle{}, core.Result{}, err
	}
	return bundle, result, nil
}

func (s Service) InspectUniverse(ctx context.Context, path string) (core.UniverseExplain, error) {
	_, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return core.UniverseExplain{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return core.UniverseExplain{}, err
	}
	return plan.UniverseExplain, nil
}

func (s Service) ValidateEvaluation(ctx context.Context, path string) (EvaluationValidationResult, error) {
	plan, err := s.compileEvaluationFile(ctx, path)
	if err != nil {
		return EvaluationValidationResult{}, err
	}
	metrics, err := selectedEvaluationMetrics(plan)
	if err != nil {
		return EvaluationValidationResult{}, err
	}
	return EvaluationValidationResult{
		Valid:            true,
		Name:             plan.Name,
		StrategyName:     plan.Strategy.Name,
		BaseRunName:      plan.BaseRun.Name,
		CaseCount:        len(plan.Cases),
		WalkForwardSteps: len(plan.WalkForward),
		Parallelism:      resolveEvaluationParallelism(plan.Spec, RunEvaluationOptions{}),
		Metrics:          metrics,
	}, nil
}

func (s Service) RunEvaluation(ctx context.Context, path string, options ...RunEvaluationOptions) (EvaluationRunResult, error) {
	if s.evaluations == nil {
		return EvaluationRunResult{}, oops.In("backtest_evaluation_service").New("backtest evaluation repository is nil")
	}
	plan, err := s.compileEvaluationFile(ctx, path)
	if err != nil {
		return EvaluationRunResult{}, err
	}
	option := RunEvaluationOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	if option.Parallelism < 0 {
		return EvaluationRunResult{}, oops.In("backtest_evaluation_service").With("parallelism", option.Parallelism).New("evaluation parallelism must not be negative")
	}
	parallelism := resolveEvaluationParallelism(plan.Spec, option)
	caseResults, err := s.runEvaluationCases(ctx, path, plan.Cases, plan.Spec.Constraints, plan.Spec.Regime, parallelism)
	if err != nil {
		return EvaluationRunResult{}, err
	}
	ranking, err := core.RankEvaluationResults(caseResults, plan.Spec.Ranking)
	if err != nil {
		return EvaluationRunResult{}, err
	}
	applyRanks(caseResults, ranking)

	walkForward, err := s.runWalkForward(ctx, path, plan, parallelism)
	if err != nil {
		return EvaluationRunResult{}, err
	}

	experiment, cases, steps, err := savedEvaluationRows(plan, caseResults, ranking, walkForward)
	if err != nil {
		return EvaluationRunResult{}, err
	}
	detail, err := s.evaluations.SaveEvaluation(ctx, experiment, cases, steps, time.Now())
	if err != nil {
		return EvaluationRunResult{}, err
	}
	return EvaluationRunResult{
		Experiment:  detail.Experiment,
		Cases:       caseResults,
		Ranking:     ranking,
		RegimeSplit: core.BuildRegimeSplit(caseResults),
		Robustness:  core.BuildRobustnessReport(caseResults, walkForward, plan.Spec.Ranking, 3),
		WalkForward: walkForward,
	}, nil
}

func (s Service) ListEvaluations(ctx context.Context) ([]SavedEvaluationSummary, error) {
	if s.evaluations == nil {
		return nil, oops.In("backtest_evaluation_service").New("backtest evaluation repository is nil")
	}
	return s.evaluations.ListEvaluations(ctx)
}

func (s Service) InspectEvaluation(ctx context.Context, ref string) (SavedEvaluationDetail, error) {
	if strings.TrimSpace(ref) == "" {
		return SavedEvaluationDetail{}, oops.In("backtest_evaluation_service").New("inspect evaluation requires name or id")
	}
	if s.evaluations == nil {
		return SavedEvaluationDetail{}, oops.In("backtest_evaluation_service").With("ref", ref).New("backtest evaluation repository is nil")
	}
	return s.evaluations.GetEvaluation(ctx, ref)
}

func (s Service) CompareEvaluation(ctx context.Context, ref string) (SavedEvaluationDetail, error) {
	return s.InspectEvaluation(ctx, ref)
}

func (s Service) RankEvaluation(ctx context.Context, ref string, objective string) ([]SavedExperimentCase, error) {
	detail, err := s.InspectEvaluation(ctx, ref)
	if err != nil {
		return nil, err
	}
	out := append([]SavedExperimentCase(nil), detail.Cases...)
	if strings.TrimSpace(objective) == "" {
		objective = core.MetricCalmar
	}
	for i := range out {
		metrics := core.Metrics{}
		if len(out[i].MetricsJSON) > 0 {
			if err := json.Unmarshal(out[i].MetricsJSON, &metrics); err != nil {
				return nil, oops.In("backtest_evaluation_service").With("case_id", out[i].CaseID).Wrapf(err, "decode saved metrics")
			}
		}
		value, ok := metrics[objective]
		if !ok {
			return nil, oops.In("backtest_evaluation_service").With("objective", objective, "case_id", out[i].CaseID).New("ranking objective metric is missing")
		}
		out[i].Objective = objective
		out[i].ObjectiveValue = value
	}
	sortSavedCases(out, "desc")
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

func (s Service) UpsertStrategy(ctx context.Context, req SaveStrategyRequest) (SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_service").With("name", req.Name, "yaml_path", req.YAMLPath)
	if s.strategies == nil {
		return SavedStrategyDetail{}, errb.New("backtest strategy repository is nil")
	}
	spec, specJSON, specHash, err := s.loadCanonicalStrategy(ctx, req)
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrap(err)
	}
	strategyID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrapf(err, "generate backtest strategy id")
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrapf(err, "generate backtest strategy version id")
	}
	now := time.Now()
	strategy := SavedStrategy{
		ID:              strategyID,
		Name:            spec.Name,
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := SavedStrategyVersion{
		ID:            versionID,
		StrategyID:    strategyID,
		Version:       1,
		SchemaVersion: spec.SchemaVersion,
		SpecJSON:      specJSON,
		SpecHash:      specHash,
		CreatedAt:     now,
	}
	return s.strategies.UpsertStrategyWithVersion(ctx, strategy, version, now)
}

func (s Service) ListStrategies(ctx context.Context) ([]SavedStrategyDetail, error) {
	if s.strategies == nil {
		return nil, oops.In("backtest_strategy_service").New("backtest strategy repository is nil")
	}
	return s.strategies.ListStrategies(ctx)
}

func (s Service) InspectStrategy(ctx context.Context, name string) (SavedStrategyDetail, error) {
	if strings.TrimSpace(name) == "" {
		return SavedStrategyDetail{}, oops.In("backtest_strategy_service").New("inspect backtest strategy requires name")
	}
	if s.strategies == nil {
		return SavedStrategyDetail{}, oops.In("backtest_strategy_service").With("name", name).New("backtest strategy repository is nil")
	}
	return s.strategies.GetStrategy(ctx, name)
}

func (s Service) DeleteStrategy(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return oops.In("backtest_strategy_service").New("delete backtest strategy requires name")
	}
	if s.strategies == nil {
		return oops.In("backtest_strategy_service").With("name", name).New("backtest strategy repository is nil")
	}
	return s.strategies.DeleteStrategy(ctx, name, time.Now())
}

func (s Service) loadCanonicalStrategy(ctx context.Context, req SaveStrategyRequest) (core.StrategySpec, json.RawMessage, string, error) {
	errb := oops.In("backtest_strategy_service").With("name", req.Name, "yaml_path", req.YAMLPath)
	if strings.TrimSpace(req.Name) == "" {
		return core.StrategySpec{}, nil, "", errb.New("backtest strategy name is required")
	}
	strategy, err := LoadStrategyFile(ctx, req.YAMLPath)
	if err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrap(err)
	}
	if strategy.Name != req.Name {
		return core.StrategySpec{}, nil, "", errb.With("yaml_name", strategy.Name).Errorf("backtest strategy name mismatch: cli=%s yaml=%s", req.Name, strategy.Name)
	}
	if err := core.ValidateStrategySpec(strategy, s.registry); err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrap(err)
	}
	specJSON, err := json.Marshal(strategy)
	if err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrapf(err, "encode canonical backtest strategy spec")
	}
	return strategy, specJSON, hashutil.SHA256(specJSON), nil
}

func (s Service) compileFile(ctx context.Context, path string) (Bundle, core.StrategyPlan, error) {
	bundle, err := LoadFile(ctx, path)
	if err != nil {
		return Bundle{}, core.StrategyPlan{}, err
	}
	plan, err := core.Compile(bundle.Strategy, bundle.Run, s.registry)
	if err != nil {
		return Bundle{}, core.StrategyPlan{}, err
	}
	return bundle, plan, nil
}

func (s Service) compileEvaluationFile(ctx context.Context, path string) (core.EvaluationPlan, error) {
	bundle, err := LoadEvaluationFile(ctx, path)
	if err != nil {
		return core.EvaluationPlan{}, err
	}
	plan, err := core.CompileEvaluation(bundle.Strategy, bundle.Run, bundle.Evaluation, s.registry)
	if err != nil {
		return core.EvaluationPlan{}, err
	}
	return plan, nil
}

func (s Service) runEvaluationCase(ctx context.Context, yamlPath string, evalCase core.EvaluationCasePlan, regime core.EvaluationRegimeSpec) (core.Result, []string, error) {
	plan, err := s.resolveUniverse(ctx, yamlPath, evalCase.Plan)
	if err != nil {
		return core.Result{}, nil, err
	}
	engine, err := core.NewEngine(newDailyBarFeed(s.reader))
	if err != nil {
		return core.Result{}, nil, oops.In("backtest_evaluation_service").With("case", evalCase.Name).Wrap(err)
	}
	result, err := engine.Run(ctx, plan)
	if err != nil {
		return core.Result{}, nil, oops.In("backtest_evaluation_service").With("case", evalCase.Name).Wrap(err)
	}
	return result, core.RegimeTagsWithSpec(result, result.BenchmarkBars, regime), nil
}

func (s Service) runEvaluationCases(ctx context.Context, yamlPath string, cases []core.EvaluationCasePlan, constraints core.EvaluationConstraintSet, regime core.EvaluationRegimeSpec, parallelism int) ([]core.EvaluationCaseResult, error) {
	if len(cases) == 0 {
		return nil, nil
	}
	if parallelism <= 0 {
		return nil, oops.In("backtest_evaluation_service").With("parallelism", parallelism).New("evaluation parallelism must be positive")
	}
	if parallelism > len(cases) {
		parallelism = len(cases)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]core.EvaluationCaseResult, len(cases))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	setError := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if firstErr == nil {
			firstErr = err
			cancel()
		}
	}

	for worker := 0; worker < parallelism; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				evalCase := cases[index]
				result, tags, err := s.runEvaluationCase(ctx, yamlPath, evalCase, regime)
				if err != nil {
					setError(err)
					continue
				}
				caseConstraints := core.EvaluateConstraints(result.Metrics, constraints)
				results[index] = core.EvaluationCaseResult{
					CaseID:            evalCase.ID,
					CaseName:          evalCase.Name,
					Period:            evalCase.Period,
					Parameters:        evalCase.Parameters,
					Result:            result,
					Metrics:           result.Metrics,
					RegimeTags:        tags,
					ConstraintResults: caseConstraints,
					PassedConstraints: core.ConstraintsPassed(caseConstraints),
				}
			}
		}()
	}

	for index := range cases {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			if firstErr != nil {
				return nil, firstErr
			}
			return nil, oops.In("backtest_evaluation_service").Wrap(ctx.Err())
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, oops.In("backtest_evaluation_service").Wrap(err)
	}
	return results, nil
}

func resolveEvaluationParallelism(spec core.EvaluationSpec, option RunEvaluationOptions) int {
	if option.Parallelism > 0 {
		return option.Parallelism
	}
	if spec.Execution.Parallelism > 0 {
		return spec.Execution.Parallelism
	}
	return 1
}

func (s Service) runWalkForward(ctx context.Context, yamlPath string, plan core.EvaluationPlan, parallelism int) ([]core.WalkForwardStepResult, error) {
	out := make([]core.WalkForwardStepResult, 0, len(plan.WalkForward))
	for _, step := range plan.WalkForward {
		trainResults, err := s.runEvaluationCases(ctx, yamlPath, step.Cases, core.WalkForwardConstraints(plan.Spec), plan.Spec.Regime, parallelism)
		if err != nil {
			return nil, err
		}
		ranking, err := core.RankEvaluationResults(trainResults, core.ResolveWalkForwardSelection(plan.Spec))
		if err != nil {
			return nil, err
		}
		if len(ranking) == 0 {
			return nil, oops.In("backtest_walk_forward").With("step", step.Index).New("walk-forward train step has no passing parameter set")
		}
		selected := ranking[0]
		testCase, ok := matchingTestCase(step.TestCases, selected.Parameters)
		if !ok {
			return nil, oops.In("backtest_walk_forward").With("step", step.Index).New("walk-forward selected parameters did not match a test case")
		}
		testResult, _, err := s.runEvaluationCase(ctx, yamlPath, testCase, plan.Spec.Regime)
		if err != nil {
			return nil, err
		}
		out = append(out, core.WalkForwardStepResult{
			Index:              step.Index,
			Train:              step.Train,
			Test:               step.Test,
			SelectedParameters: selected.Parameters,
			TrainCaseID:        selected.CaseID,
			TestCaseID:         testCase.ID,
			TrainObjective:     selected.ObjectiveValue,
			TestMetrics:        testResult.Metrics,
			Runtime:            testResult.Runtime,
			DataFingerprint:    testResult.DataFingerprint,
			ResultHash:         testResult.ResultHash,
		})
	}
	return out, nil
}

func (s Service) resolveUniverse(ctx context.Context, yamlPath string, plan core.StrategyPlan) (core.StrategyPlan, error) {
	explain, err := s.universe.Explain(ctx, universeservice.ContextRequest{
		YAMLPath: yamlPath,
		Market:   plan.Market,
		From:     plan.From,
		To:       plan.To,
		Pipeline: plan.Universe.Pipeline,
	}, plan.Universe.Core)
	if err != nil {
		return core.StrategyPlan{}, oops.In("backtest_service").With("run", plan.RunName).Wrap(err)
	}
	explain.PositionPolicy = plan.Universe.PositionPolicy
	plan.UniverseExplain = explain
	plan.Symbols = symbolsFromUniverseExplain(explain)
	plan.Instruments = instrumentsFromUniverseExplain(explain, plan.Market)
	return plan, nil
}

func symbolsFromUniverseExplain(explain core.UniverseExplain) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, snapshot := range explain.Snapshots {
		for _, symbol := range snapshot.Symbols {
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			out = append(out, symbol)
		}
	}
	return out
}

func instrumentsFromUniverseExplain(explain core.UniverseExplain, defaultMarket string) []core.InstrumentIdentity {
	seen := map[string]struct{}{}
	out := make([]core.InstrumentIdentity, 0)
	for _, snapshot := range explain.Snapshots {
		for _, candidate := range snapshot.Candidates {
			instrument := core.InstrumentIdentity{
				Symbol:       candidate.Symbol,
				Market:       stringField(candidate.Fields, "market", defaultMarket),
				SecurityType: stringField(candidate.Fields, "security_type", ""),
			}
			key := instrument.Market + "\x00" + instrument.SecurityType + "\x00" + instrument.Symbol
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, instrument)
		}
	}
	return out
}

func normalizeDataInstruments(instruments []core.InstrumentIdentity, defaultMarket string) ([]core.InstrumentIdentity, error) {
	seen := map[string]struct{}{}
	out := make([]core.InstrumentIdentity, 0, len(instruments))
	symbolIdentities := map[string]string{}
	for _, instrument := range instruments {
		instrument.Market = withDefault(instrument.Market, defaultMarket)
		if strings.TrimSpace(instrument.Symbol) == "" {
			return nil, oops.In("backtest_service").New("backtest instrument symbol is required")
		}
		if strings.TrimSpace(instrument.Market) == "" {
			return nil, oops.In("backtest_service").With("symbol", instrument.Symbol).New("backtest instrument market is required")
		}
		if strings.TrimSpace(instrument.SecurityType) == "" {
			return nil, oops.In("backtest_service").With("symbol", instrument.Symbol, "market", instrument.Market).New("backtest instrument security_type is required")
		}
		key := instrument.Market + "\x00" + instrument.SecurityType + "\x00" + instrument.Symbol
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		symbolKey := instrument.Symbol
		identityKey := instrument.Market + "/" + instrument.SecurityType
		if existing, ok := symbolIdentities[symbolKey]; ok && existing != identityKey {
			return nil, oops.In("backtest_service").With("symbol", instrument.Symbol).New("backtest engine requires unique symbol per instrument identity")
		}
		symbolIdentities[symbolKey] = identityKey
		out = append(out, instrument)
	}
	return out, nil
}

func canonicalDailyBarToBacktestBar(row dailybar.Bar) (core.Bar, error) {
	errb := oops.In("backtest_dailybar_mapper").With("symbol", row.Symbol, "trading_date", row.TradingDate)
	tradingDate, err := time.Parse(time.DateOnly, row.TradingDate)
	if err != nil {
		return core.Bar{}, errb.Wrapf(err, "parse trading date")
	}
	open, err := requiredFloat(row.Open, "opening_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	high, err := requiredFloat(row.High, "highest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	low, err := requiredFloat(row.Low, "lowest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	closePrice, err := requiredFloat(row.Close, "closing_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	volume, err := optionalFloat(row.Volume, "traded_volume")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	tradedAmount, err := optionalFloat(row.TradedValue, "traded_amount")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	marketCap, err := optionalFloat(row.MarketCap, "market_capitalization")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	nav, err := optionalFloat(row.Extensions["nav"], "extensions.nav")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	adjustedClose, err := optionalFloat(row.Extensions["adjusted_close"], "extensions.adjusted_close")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	status := core.BarStatusOK
	if open == 0 && high == 0 && low == 0 && closePrice > 0 && volume == 0 && tradedAmount == 0 {
		status = core.BarStatusNoTrade
	}
	return core.Bar{
		Time:          tradingDate,
		Symbol:        row.Symbol,
		Market:        string(row.Market),
		SecurityType:  string(row.SecurityType),
		Timeframe:     core.Timeframe1Day,
		Session:       core.BarSessionRegular,
		Status:        status,
		Open:          open,
		High:          high,
		Low:           low,
		Close:         closePrice,
		AdjustedClose: adjustedClose,
		Volume:        volume,
		TradedAmount:  tradedAmount,
		MarketCap:     marketCap,
		NAV:           nav,
	}, nil
}

func requiredFloat(value string, field string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, oops.In("backtest_dailybar_mapper").With("field", field).New("daily bar numeric field is required")
	}
	return optionalFloat(value, field)
}

func optionalFloat(value string, field string) (float64, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, oops.In("backtest_dailybar_mapper").With("field", field, "value", value).Wrapf(err, "parse daily bar numeric field")
	}
	return parsed, nil
}

func validationResultFromPlan(plan core.StrategyPlan) ValidationResult {
	indicators := make(map[string]string, len(plan.Indicators))
	for alias, indicator := range plan.Indicators {
		indicators[alias] = indicator.ID
	}
	return ValidationResult{
		Valid:        true,
		StrategyName: plan.StrategyName,
		RunName:      plan.RunName,
		Symbols:      append([]string(nil), plan.Symbols...),
		Instruments:  append([]core.InstrumentIdentity(nil), plan.Instruments...),
		Period:       core.Period{From: plan.From, To: plan.To},
		Market:       plan.Market,
		Timeframe:    plan.Timeframe,
		Timeframes:   plan.TimeframeMetadata(),
		Runtime:      plan.RuntimeMetadata(),
		Currency:     plan.Currency,
		Metrics:      append([]string(nil), plan.SelectedMetrics...),
		Execution: ExecutionSummary{
			Fill:                    plan.Fill,
			OrderType:               plan.OrderType,
			LimitPrice:              plan.LimitPrice,
			StopPrice:               plan.StopPrice,
			TrailingStopPct:         plan.TrailingStopPct,
			IntrabarAmbiguityPolicy: plan.IntrabarAmbiguityPolicy,
			TimeInForce:             plan.TimeInForce,
			LotSize:                 plan.LotSize,
			TickSize:                plan.TickSize,
			Commission:              plan.Commission,
			Tax:                     plan.Tax,
			ExchangeFee:             plan.ExchangeFee,
			Slippage:                plan.Slippage,
			Liquidity:               plan.Liquidity,
			PartialFill:             plan.PartialFill,
		},
		Indicators: indicators,
		Universe:   plan.UniverseExplain,
	}
}

func selectedEvaluationMetrics(plan core.EvaluationPlan) ([]string, error) {
	selection := plan.Spec.Metrics
	if selection.Preset == "" && len(selection.Include) == 0 && len(selection.Exclude) == 0 {
		selection.Preset = "research"
	}
	hasBenchmark := strings.TrimSpace(plan.BaseRun.Benchmark.Symbol) != "" || strings.TrimSpace(plan.Spec.Regime.Benchmark.Symbol) != ""
	return core.ResolveMetricSelection(selection, hasBenchmark)
}

func savedEvaluationRows(plan core.EvaluationPlan, results []core.EvaluationCaseResult, ranking []core.EvaluationCaseResult, walkForward []core.WalkForwardStepResult) (SavedExperiment, []SavedExperimentCase, []SavedWalkForwardStep, error) {
	errb := oops.In("backtest_evaluation_service").With("evaluation", plan.Name)
	experimentID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedExperiment{}, nil, nil, errb.Wrapf(err, "generate evaluation id")
	}
	specJSON, err := json.Marshal(plan.Spec)
	if err != nil {
		return SavedExperiment{}, nil, nil, errb.Wrapf(err, "encode evaluation spec")
	}
	strategyJSON, err := json.Marshal(plan.Strategy)
	if err != nil {
		return SavedExperiment{}, nil, nil, errb.Wrapf(err, "encode evaluation strategy")
	}
	now := time.Now()
	experiment := SavedExperiment{
		ID:               experimentID,
		Name:             plan.Name,
		StrategyName:     plan.Strategy.Name,
		BaseRunName:      plan.BaseRun.Name,
		SchemaVersion:    plan.Spec.SchemaVersion,
		SpecJSON:         specJSON,
		SpecHash:         hashutil.SHA256(specJSON),
		StrategySpecHash: hashutil.SHA256(strategyJSON),
		DataFrom:         plan.BaseRun.Data.From,
		DataTo:           plan.BaseRun.Data.To,
		CreatedAt:        now,
	}
	rankByCase := map[string]core.EvaluationCaseResult{}
	for _, ranked := range ranking {
		rankByCase[ranked.CaseID] = ranked
	}
	planByCase := map[string]core.EvaluationCasePlan{}
	for _, evalCase := range plan.Cases {
		planByCase[evalCase.ID] = evalCase
	}
	walkForwardTestCases := map[string]core.EvaluationCasePlan{}
	for _, step := range plan.WalkForward {
		for _, evalCase := range step.TestCases {
			walkForwardTestCases[evalCase.ID] = evalCase
		}
	}
	cases := make([]SavedExperimentCase, 0, len(results))
	for _, result := range results {
		caseID, err := idgen.NewUUIDV7()
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.Wrapf(err, "generate evaluation case id")
		}
		evalCase, ok := planByCase[result.CaseID]
		if !ok {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).New("evaluation case plan is missing")
		}
		strategyHash, runHash, err := evaluationCaseSpecHashes(evalCase)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).Wrap(err)
		}
		resultJSON, err := json.Marshal(result.Result)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).Wrapf(err, "encode backtest result")
		}
		metricsJSON, err := json.Marshal(result.Metrics)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).Wrapf(err, "encode metric summary")
		}
		parameterJSON, err := json.Marshal(result.Parameters)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).Wrapf(err, "encode parameter set")
		}
		regimeJSON, err := json.Marshal(result.RegimeTags)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("case", result.CaseID).Wrapf(err, "encode regime tags")
		}
		ranked := rankByCase[result.CaseID]
		status := "failed_constraints"
		if result.PassedConstraints {
			status = "passed"
		}
		cases = append(cases, SavedExperimentCase{
			ID:                caseID,
			ExperimentID:      experimentID,
			CaseID:            result.CaseID,
			CaseName:          result.CaseName,
			RunName:           result.Result.RunName,
			PeriodFrom:        result.Period.From.Format(time.DateOnly),
			PeriodTo:          result.Period.To.Format(time.DateOnly),
			ParameterJSON:     parameterJSON,
			RegimeTagsJSON:    regimeJSON,
			Status:            status,
			PassedConstraints: result.PassedConstraints,
			Rank:              ranked.Rank,
			Objective:         ranked.Objective,
			ObjectiveValue:    ranked.ObjectiveValue,
			ResultJSON:        resultJSON,
			MetricsJSON:       metricsJSON,
			StrategyHash:      strategyHash,
			RunHash:           runHash,
			EngineVersion:     result.Result.Runtime.EngineVersion,
			IndicatorRegistry: result.Result.Runtime.IndicatorRegistryVersion,
			MetricRegistry:    result.Result.Runtime.MetricRegistryVersion,
			DataFingerprint:   result.Result.DataFingerprint,
			ResultHash:        result.Result.ResultHash,
			CreatedAt:         now,
		})
	}
	steps := make([]SavedWalkForwardStep, 0, len(walkForward))
	for _, step := range walkForward {
		stepID, err := idgen.NewUUIDV7()
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.Wrapf(err, "generate walk-forward step id")
		}
		paramsJSON, err := json.Marshal(step.SelectedParameters)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("step", step.Index).Wrapf(err, "encode selected walk-forward parameters")
		}
		testMetricsJSON, err := json.Marshal(step.TestMetrics)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("step", step.Index).Wrapf(err, "encode walk-forward test metrics")
		}
		testCase, ok := walkForwardTestCases[step.TestCaseID]
		if !ok {
			return SavedExperiment{}, nil, nil, errb.With("step", step.Index, "test_case", step.TestCaseID).New("walk-forward test case plan is missing")
		}
		strategyHash, runHash, err := evaluationCaseSpecHashes(testCase)
		if err != nil {
			return SavedExperiment{}, nil, nil, errb.With("step", step.Index, "test_case", step.TestCaseID).Wrap(err)
		}
		steps = append(steps, SavedWalkForwardStep{
			ID:                    stepID,
			ExperimentID:          experimentID,
			StepIndex:             step.Index,
			TrainFrom:             step.Train.From.Format(time.DateOnly),
			TrainTo:               step.Train.To.Format(time.DateOnly),
			TestFrom:              step.Test.From.Format(time.DateOnly),
			TestTo:                step.Test.To.Format(time.DateOnly),
			SelectedParameterJSON: paramsJSON,
			TrainCaseID:           step.TrainCaseID,
			TestCaseID:            step.TestCaseID,
			TrainObjective:        step.TrainObjective,
			TestMetricsJSON:       testMetricsJSON,
			StrategyHash:          strategyHash,
			RunHash:               runHash,
			EngineVersion:         step.Runtime.EngineVersion,
			IndicatorRegistry:     step.Runtime.IndicatorRegistryVersion,
			MetricRegistry:        step.Runtime.MetricRegistryVersion,
			DataFingerprint:       step.DataFingerprint,
			ResultHash:            step.ResultHash,
			CreatedAt:             now,
		})
	}
	return experiment, cases, steps, nil
}

func evaluationCaseSpecHashes(evalCase core.EvaluationCasePlan) (strategyHash string, runHash string, err error) {
	errb := oops.In("backtest_evaluation_service").With("case", evalCase.ID)
	strategyJSON, err := json.Marshal(evalCase.Strategy)
	if err != nil {
		return "", "", errb.Wrapf(err, "encode evaluation case strategy")
	}
	runJSON, err := json.Marshal(evalCase.Run)
	if err != nil {
		return "", "", errb.Wrapf(err, "encode evaluation case run")
	}
	return hashutil.SHA256(strategyJSON), hashutil.SHA256(runJSON), nil
}

func savedBacktestRun(bundle Bundle, result core.Result) (SavedBacktestRun, error) {
	errb := oops.In("backtest_run_service").With("run", result.RunName, "strategy", result.StrategyName)
	runID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedBacktestRun{}, errb.Wrapf(err, "generate backtest run id")
	}
	strategyJSON, err := json.Marshal(bundle.Strategy)
	if err != nil {
		return SavedBacktestRun{}, errb.Wrapf(err, "encode backtest strategy spec")
	}
	runJSON, err := json.Marshal(bundle.Run)
	if err != nil {
		return SavedBacktestRun{}, errb.Wrapf(err, "encode backtest run spec")
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return SavedBacktestRun{}, errb.Wrapf(err, "encode backtest result")
	}
	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return SavedBacktestRun{}, errb.Wrapf(err, "encode backtest metrics")
	}
	now := time.Now()
	return SavedBacktestRun{
		ID:                runID,
		RunName:           result.RunName,
		StrategyName:      result.StrategyName,
		Market:            result.Market,
		Timeframe:         result.Timeframe,
		PeriodFrom:        result.Period.From.Format(time.DateOnly),
		PeriodTo:          result.Period.To.Format(time.DateOnly),
		StrategyHash:      hashutil.SHA256(strategyJSON),
		RunHash:           hashutil.SHA256(runJSON),
		EngineVersion:     result.Runtime.EngineVersion,
		IndicatorRegistry: result.Runtime.IndicatorRegistryVersion,
		MetricRegistry:    result.Runtime.MetricRegistryVersion,
		DataFingerprint:   result.DataFingerprint,
		ResultHash:        result.ResultHash,
		ResultJSON:        resultJSON,
		MetricsJSON:       metricsJSON,
		CreatedAt:         now,
	}, nil
}

func compareRunMetrics(left SavedBacktestRunDetail, right SavedBacktestRunDetail) ([]BacktestRunMetricDiff, error) {
	leftMetrics, err := savedRunMetrics(left)
	if err != nil {
		return nil, err
	}
	rightMetrics, err := savedRunMetrics(right)
	if err != nil {
		return nil, err
	}
	metricIDs := make([]string, 0, len(leftMetrics)+len(rightMetrics))
	for metric := range leftMetrics {
		metricIDs = append(metricIDs, metric)
	}
	for metric := range rightMetrics {
		if _, ok := leftMetrics[metric]; !ok {
			metricIDs = append(metricIDs, metric)
		}
	}
	slices.Sort(metricIDs)
	out := make([]BacktestRunMetricDiff, 0, len(metricIDs))
	for _, metric := range metricIDs {
		leftValue, leftOK := leftMetrics[metric]
		rightValue, rightOK := rightMetrics[metric]
		delta := rightValue - leftValue
		deltaPct := 0.0
		if leftValue != 0 {
			deltaPct = delta / leftValue
		}
		out = append(out, BacktestRunMetricDiff{
			Metric:       metric,
			LeftValue:    leftValue,
			RightValue:   rightValue,
			Delta:        delta,
			DeltaPct:     deltaPct,
			LeftPresent:  leftOK,
			RightPresent: rightOK,
		})
	}
	return out, nil
}

func savedRunMetrics(detail SavedBacktestRunDetail) (core.Metrics, error) {
	metrics := core.Metrics{}
	if len(detail.Run.MetricsJSON) > 0 {
		if err := json.Unmarshal(detail.Run.MetricsJSON, &metrics); err != nil {
			return nil, oops.In("backtest_run_service").With("run_id", detail.Run.ID).Wrapf(err, "decode saved run metrics")
		}
	}
	for metric, value := range detail.Result.Metrics {
		if _, ok := metrics[metric]; !ok {
			metrics[metric] = value
		}
	}
	return metrics, nil
}

func publicSavedRun(run SavedBacktestRun) SavedBacktestRun {
	run.ResultJSON = nil
	run.MetricsJSON = nil
	return run
}

func sameRunRuntime(left SavedBacktestRun, right SavedBacktestRun) bool {
	return left.EngineVersion == right.EngineVersion &&
		left.IndicatorRegistry == right.IndicatorRegistry &&
		left.MetricRegistry == right.MetricRegistry
}

func applyRanks(results []core.EvaluationCaseResult, ranking []core.EvaluationCaseResult) {
	ranked := make(map[string]core.EvaluationCaseResult, len(ranking))
	for _, item := range ranking {
		ranked[item.CaseID] = item
	}
	for i := range results {
		if item, ok := ranked[results[i].CaseID]; ok {
			results[i].Rank = item.Rank
			results[i].Objective = item.Objective
			results[i].ObjectiveValue = item.ObjectiveValue
		}
	}
}

func matchingTestCase(cases []core.EvaluationCasePlan, parameters map[string]any) (core.EvaluationCasePlan, bool) {
	want, _ := json.Marshal(parameters)
	for _, evalCase := range cases {
		got, _ := json.Marshal(evalCase.Parameters)
		if string(got) == string(want) {
			return evalCase, true
		}
	}
	return core.EvaluationCasePlan{}, false
}

func sortSavedCases(cases []SavedExperimentCase, order string) {
	slices.SortFunc(cases, func(a, b SavedExperimentCase) int {
		if a.ObjectiveValue == b.ObjectiveValue {
			return strings.Compare(a.CaseID, b.CaseID)
		}
		if order == "asc" {
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
}

func stringField(fields map[string]any, key string, fallback string) string {
	value, ok := fields[key]
	if !ok {
		return fallback
	}
	text := strings.TrimSpace(valueString(value))
	if text == "" {
		return fallback
	}
	return text
}

func withDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}
