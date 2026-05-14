package backtest

import (
	"time"

	"github.com/ev3rlit/mwosa/packages/universe"
)

const SchemaVersion = 1

type StrategySpec struct {
	Kind          string                   `json:"kind" yaml:"kind"`
	SchemaVersion int                      `json:"schema_version" yaml:"schema_version"`
	Name          string                   `json:"name" yaml:"name"`
	Description   string                   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags          []string                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	Indicators    map[string]IndicatorSpec `json:"indicators,omitempty" yaml:"indicators,omitempty"`
	Entry         RuleExpr                 `json:"entry" yaml:"entry"`
	Exit          RuleExpr                 `json:"exit" yaml:"exit"`
	Sizing        SizingSpec               `json:"sizing" yaml:"sizing"`
	Risk          RiskSpec                 `json:"risk,omitempty" yaml:"risk,omitempty"`
}

type BacktestRunSpec struct {
	Kind          string        `json:"kind" yaml:"kind"`
	SchemaVersion int           `json:"schema_version" yaml:"schema_version"`
	Name          string        `json:"name" yaml:"name"`
	Strategy      StrategyRef   `json:"strategy" yaml:"strategy"`
	Data          DataSpec      `json:"data" yaml:"data"`
	Universe      UniverseSpec  `json:"universe" yaml:"universe"`
	Benchmark     BenchmarkSpec `json:"benchmark,omitempty" yaml:"benchmark,omitempty"`
	Portfolio     PortfolioSpec `json:"portfolio" yaml:"portfolio"`
	Execution     ExecutionSpec `json:"execution" yaml:"execution"`
	Report        ReportSpec    `json:"report,omitempty" yaml:"report,omitempty"`
}

type EvaluationSpec struct {
	Kind          string                  `json:"kind" yaml:"kind"`
	SchemaVersion int                     `json:"schema_version" yaml:"schema_version"`
	Name          string                  `json:"name" yaml:"name"`
	Strategy      StrategyRef             `json:"strategy" yaml:"strategy"`
	BaseRun       EvaluationBaseRunRef    `json:"base_run" yaml:"base_run"`
	Periods       EvaluationPeriodsSpec   `json:"periods" yaml:"periods"`
	Parameters    map[string][]any        `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Metrics       MetricSelectionSpec     `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Constraints   EvaluationConstraintSet `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Ranking       EvaluationRankingSpec   `json:"ranking,omitempty" yaml:"ranking,omitempty"`
	Regime        EvaluationRegimeSpec    `json:"regime,omitempty" yaml:"regime,omitempty"`
	Execution     EvaluationExecutionSpec `json:"execution,omitempty" yaml:"execution,omitempty"`
	WalkForward   WalkForwardSpec         `json:"walk_forward,omitempty" yaml:"walk_forward,omitempty"`
}

type EvaluationBaseRunRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type EvaluationPeriodsSpec struct {
	Mode       string                 `json:"mode" yaml:"mode"`
	From       string                 `json:"from,omitempty" yaml:"from,omitempty"`
	To         string                 `json:"to,omitempty" yaml:"to,omitempty"`
	Items      []EvaluationPeriodSpec `json:"items,omitempty" yaml:"items,omitempty"`
	Window     DurationSpec           `json:"window,omitempty" yaml:"window,omitempty"`
	Step       DurationSpec           `json:"step,omitempty" yaml:"step,omitempty"`
	WindowDays int                    `json:"window_days,omitempty" yaml:"window_days,omitempty"`
	StepDays   int                    `json:"step_days,omitempty" yaml:"step_days,omitempty"`
}

type EvaluationPeriodSpec struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
}

type DurationSpec struct {
	Years  int `json:"years,omitempty" yaml:"years,omitempty"`
	Months int `json:"months,omitempty" yaml:"months,omitempty"`
	Days   int `json:"days,omitempty" yaml:"days,omitempty"`
}

type EvaluationConstraintSet struct {
	MaxDrawdownLTE   *float64 `json:"max_drawdown_lte,omitempty" yaml:"max_drawdown_lte,omitempty"`
	MinCAGRGTE       *float64 `json:"min_cagr_gte,omitempty" yaml:"min_cagr_gte,omitempty"`
	MaxTurnoverLTE   *float64 `json:"max_turnover_lte,omitempty" yaml:"max_turnover_lte,omitempty"`
	MinTradeCountGTE *float64 `json:"min_trade_count_gte,omitempty" yaml:"min_trade_count_gte,omitempty"`
}

type EvaluationRankingSpec struct {
	Objective string `json:"objective,omitempty" yaml:"objective,omitempty"`
	Order     string `json:"order,omitempty" yaml:"order,omitempty"`
}

type EvaluationRegimeSpec struct {
	Benchmark BenchmarkSpec `json:"benchmark,omitempty" yaml:"benchmark,omitempty"`
}

type EvaluationExecutionSpec struct {
	Parallelism int `json:"parallelism,omitempty" yaml:"parallelism,omitempty"`
}

type WalkForwardSpec struct {
	Train  DurationSpec             `json:"train,omitempty" yaml:"train,omitempty"`
	Test   DurationSpec             `json:"test,omitempty" yaml:"test,omitempty"`
	Step   DurationSpec             `json:"step,omitempty" yaml:"step,omitempty"`
	Select WalkForwardSelectionSpec `json:"select,omitempty" yaml:"select,omitempty"`
}

type WalkForwardSelectionSpec struct {
	Objective   string                  `json:"objective,omitempty" yaml:"objective,omitempty"`
	Order       string                  `json:"order,omitempty" yaml:"order,omitempty"`
	Constraints EvaluationConstraintSet `json:"constraints,omitempty" yaml:"constraints,omitempty"`
}

type StrategyRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type DataSpec struct {
	Market       string `json:"market" yaml:"market"`
	SecurityType string `json:"security_type,omitempty" yaml:"security_type,omitempty"`
	Timeframe    string `json:"timeframe" yaml:"timeframe"`
	From         string `json:"from" yaml:"from"`
	To           string `json:"to" yaml:"to"`
}

type UniverseSpec struct {
	Symbols        []string              `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Ref            string                `json:"ref,omitempty" yaml:"ref,omitempty"`
	Schedule       universe.ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Pipeline       []universe.StepSpec   `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	PositionPolicy string                `json:"position_policy,omitempty" yaml:"position_policy,omitempty"`
}

type UniverseScheduleSpec = universe.ScheduleSpec
type UniverseSelectorStepSpec = universe.StepSpec

type BenchmarkSpec struct {
	Symbol       string `json:"symbol,omitempty" yaml:"symbol,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	Market       string `json:"market,omitempty" yaml:"market,omitempty"`
	SecurityType string `json:"security_type,omitempty" yaml:"security_type,omitempty"`
}

type PortfolioSpec struct {
	InitialCash float64 `json:"initial_cash" yaml:"initial_cash"`
	Currency    string  `json:"currency,omitempty" yaml:"currency,omitempty"`
}

type ExecutionSpec struct {
	Fill       string   `json:"fill" yaml:"fill"`
	Commission CostSpec `json:"commission,omitempty" yaml:"commission,omitempty"`
	Slippage   CostSpec `json:"slippage,omitempty" yaml:"slippage,omitempty"`
}

type CostSpec struct {
	Type  string  `json:"type,omitempty" yaml:"type,omitempty"`
	Value float64 `json:"value,omitempty" yaml:"value,omitempty"`
}

type ReportSpec struct {
	Metrics MetricSelectionSpec `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

type MetricSelectionSpec struct {
	Preset  string   `json:"preset,omitempty" yaml:"preset,omitempty"`
	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

type SizingSpec struct {
	Type  string  `json:"type" yaml:"type"`
	Value float64 `json:"value" yaml:"value"`
}

type RiskSpec struct {
	MaxPositions       int     `json:"max_positions,omitempty" yaml:"max_positions,omitempty"`
	MaxSymbolWeightPct float64 `json:"max_symbol_weight_pct,omitempty" yaml:"max_symbol_weight_pct,omitempty"`
}

type IndicatorSpec struct {
	ID     string             `json:"id" yaml:"id"`
	Source ValueExpr          `json:"source" yaml:"source"`
	Params map[string]float64 `json:"params,omitempty" yaml:"params,omitempty"`
	Output string             `json:"output,omitempty" yaml:"output,omitempty"`
}

type RuleExpr struct {
	Operator string      `json:"operator"`
	Rules    []RuleExpr  `json:"rules,omitempty"`
	Rule     *RuleExpr   `json:"rule,omitempty"`
	Args     []ValueExpr `json:"args,omitempty"`
}

type ValueExpr struct {
	Kind      string         `json:"kind"`
	Price     string         `json:"price,omitempty"`
	Value     float64        `json:"value,omitempty"`
	Ref       string         `json:"ref,omitempty"`
	Indicator *IndicatorSpec `json:"indicator,omitempty"`
}

type StrategyPlan struct {
	StrategyName    string
	RunName         string
	Symbols         []string
	Instruments     []InstrumentIdentity
	From            time.Time
	To              time.Time
	Timeframe       string
	Market          string
	Benchmark       BenchmarkSpec
	InitialCash     float64
	Currency        string
	Fill            string
	Commission      CostSpec
	Slippage        CostSpec
	Indicators      map[string]IndicatorSpec
	Entry           RuleExpr
	Exit            RuleExpr
	Sizing          SizingSpec
	Risk            RiskSpec
	Report          ReportSpec
	SelectedMetrics []string
	Universe        UniversePlan
	UniverseExplain UniverseExplain
	metricRegistry  MetricRegistry
	registry        IndicatorRegistry
}

type InstrumentIdentity struct {
	Symbol       string `json:"symbol"`
	Market       string `json:"market,omitempty"`
	SecurityType string `json:"security_type,omitempty"`
}
