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
	Entries       []RuleExpr               `json:"entries,omitempty" yaml:"entries,omitempty"`
	Exits         []RuleExpr               `json:"exits,omitempty" yaml:"exits,omitempty"`
	Rebalance     []RuleExpr               `json:"rebalance,omitempty" yaml:"rebalance,omitempty"`
	Stops         []RuleExpr               `json:"stops,omitempty" yaml:"stops,omitempty"`
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
	Search        EvaluationSearchSpec    `json:"search,omitempty" yaml:"search,omitempty"`
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

type EvaluationSearchSpec struct {
	Mode           string                                   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Seed           int64                                    `json:"seed,omitempty" yaml:"seed,omitempty"`
	Samples        int                                      `json:"samples,omitempty" yaml:"samples,omitempty"`
	InitialSamples int                                      `json:"initial_samples,omitempty" yaml:"initial_samples,omitempty"`
	Acquisition    string                                   `json:"acquisition,omitempty" yaml:"acquisition,omitempty"`
	Parameters     map[string]EvaluationSearchParameterSpec `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type EvaluationSearchParameterSpec struct {
	Values []any    `json:"values,omitempty" yaml:"values,omitempty"`
	Min    *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max    *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Step   *float64 `json:"step,omitempty" yaml:"step,omitempty"`
}

type DurationSpec struct {
	Years  int `json:"years,omitempty" yaml:"years,omitempty"`
	Months int `json:"months,omitempty" yaml:"months,omitempty"`
	Days   int `json:"days,omitempty" yaml:"days,omitempty"`
}

type EvaluationConstraintSet struct {
	MaxDrawdownLTE       *float64 `json:"max_drawdown_lte,omitempty" yaml:"max_drawdown_lte,omitempty"`
	MinCAGRGTE           *float64 `json:"min_cagr_gte,omitempty" yaml:"min_cagr_gte,omitempty"`
	MaxTurnoverLTE       *float64 `json:"max_turnover_lte,omitempty" yaml:"max_turnover_lte,omitempty"`
	MinTradeCountGTE     *float64 `json:"min_trade_count_gte,omitempty" yaml:"min_trade_count_gte,omitempty"`
	MaxExposureLTE       *float64 `json:"max_exposure_lte,omitempty" yaml:"max_exposure_lte,omitempty"`
	MaxUnfilledCountLTE  *float64 `json:"max_unfilled_count_lte,omitempty" yaml:"max_unfilled_count_lte,omitempty"`
	MaxDataIssueCountLTE *float64 `json:"max_data_issue_count_lte,omitempty" yaml:"max_data_issue_count_lte,omitempty"`
}

type EvaluationRankingSpec struct {
	Objective string             `json:"objective,omitempty" yaml:"objective,omitempty"`
	Order     string             `json:"order,omitempty" yaml:"order,omitempty"`
	Weights   map[string]float64 `json:"weights,omitempty" yaml:"weights,omitempty"`
}

type EvaluationRegimeSpec struct {
	Benchmark           BenchmarkSpec `json:"benchmark,omitempty" yaml:"benchmark,omitempty"`
	ReturnThreshold     float64       `json:"return_threshold,omitempty" yaml:"return_threshold,omitempty"`
	VolatilityThreshold float64       `json:"volatility_threshold,omitempty" yaml:"volatility_threshold,omitempty"`
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
	Weights     map[string]float64      `json:"weights,omitempty" yaml:"weights,omitempty"`
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
	Fill                    string          `json:"fill" yaml:"fill"`
	OrderType               string          `json:"order_type,omitempty" yaml:"order_type,omitempty"`
	LimitPrice              float64         `json:"limit_price,omitempty" yaml:"limit_price,omitempty"`
	StopPrice               float64         `json:"stop_price,omitempty" yaml:"stop_price,omitempty"`
	TrailingStopPct         float64         `json:"trailing_stop_pct,omitempty" yaml:"trailing_stop_pct,omitempty"`
	IntrabarAmbiguityPolicy string          `json:"intrabar_ambiguity_policy,omitempty" yaml:"intrabar_ambiguity_policy,omitempty"`
	TimeInForce             string          `json:"time_in_force,omitempty" yaml:"time_in_force,omitempty"`
	LotSize                 float64         `json:"lot_size,omitempty" yaml:"lot_size,omitempty"`
	TickSize                float64         `json:"tick_size,omitempty" yaml:"tick_size,omitempty"`
	Commission              CostSpec        `json:"commission,omitempty" yaml:"commission,omitempty"`
	Tax                     CostSpec        `json:"tax,omitempty" yaml:"tax,omitempty"`
	ExchangeFee             CostSpec        `json:"exchange_fee,omitempty" yaml:"exchange_fee,omitempty"`
	Slippage                CostSpec        `json:"slippage,omitempty" yaml:"slippage,omitempty"`
	Liquidity               LiquiditySpec   `json:"liquidity,omitempty" yaml:"liquidity,omitempty"`
	PartialFill             PartialFillSpec `json:"partial_fill,omitempty" yaml:"partial_fill,omitempty"`
}

type CostSpec struct {
	Type      string  `json:"type,omitempty" yaml:"type,omitempty"`
	Value     float64 `json:"value,omitempty" yaml:"value,omitempty"`
	BuyValue  float64 `json:"buy_value,omitempty" yaml:"buy_value,omitempty"`
	SellValue float64 `json:"sell_value,omitempty" yaml:"sell_value,omitempty"`
	MinFee    float64 `json:"min_fee,omitempty" yaml:"min_fee,omitempty"`
	Window    int     `json:"window,omitempty" yaml:"window,omitempty"`
}

type LiquiditySpec struct {
	MaxParticipationRate float64 `json:"max_participation_rate,omitempty" yaml:"max_participation_rate,omitempty"`
	VolumeCap            float64 `json:"volume_cap,omitempty" yaml:"volume_cap,omitempty"`
	TradedAmountCap      float64 `json:"traded_amount_cap,omitempty" yaml:"traded_amount_cap,omitempty"`
}

type PartialFillSpec struct {
	Policy           string `json:"policy,omitempty" yaml:"policy,omitempty"`
	ExpireAfterNBars int    `json:"expire_after_n_bars,omitempty" yaml:"expire_after_n_bars,omitempty"`
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
	ID      string             `json:"id" yaml:"id"`
	Source  ValueExpr          `json:"source" yaml:"source"`
	Compare ValueExpr          `json:"compare,omitempty" yaml:"compare,omitempty"`
	Params  map[string]float64 `json:"params,omitempty" yaml:"params,omitempty"`
	Output  string             `json:"output,omitempty" yaml:"output,omitempty"`
}

type RuleExpr struct {
	Operator string      `json:"operator"`
	Rules    []RuleExpr  `json:"rules,omitempty"`
	Rule     *RuleExpr   `json:"rule,omitempty"`
	Args     []ValueExpr `json:"args,omitempty"`
}

func (r RuleExpr) Empty() bool {
	return r.Operator == "" && len(r.Rules) == 0 && r.Rule == nil && len(r.Args) == 0
}

type ValueExpr struct {
	Kind      string         `json:"kind"`
	Price     string         `json:"price,omitempty"`
	Value     float64        `json:"value,omitempty"`
	Ref       string         `json:"ref,omitempty"`
	Timeframe string         `json:"timeframe,omitempty"`
	Position  string         `json:"position,omitempty"`
	Portfolio string         `json:"portfolio,omitempty"`
	Indicator *IndicatorSpec `json:"indicator,omitempty"`
	Args      []ValueExpr    `json:"args,omitempty"`
}

func (v ValueExpr) Empty() bool {
	return v.Kind == "" &&
		v.Price == "" &&
		v.Value == 0 &&
		v.Ref == "" &&
		v.Timeframe == "" &&
		v.Position == "" &&
		v.Portfolio == "" &&
		v.Indicator == nil &&
		len(v.Args) == 0
}

type StrategyPlan struct {
	StrategyName            string
	RunName                 string
	Symbols                 []string
	Instruments             []InstrumentIdentity
	From                    time.Time
	To                      time.Time
	Timeframe               string
	Market                  string
	Benchmark               BenchmarkSpec
	InitialCash             float64
	Currency                string
	Fill                    string
	OrderType               string
	LimitPrice              float64
	StopPrice               float64
	TrailingStopPct         float64
	IntrabarAmbiguityPolicy string
	TimeInForce             string
	LotSize                 float64
	TickSize                float64
	Commission              CostSpec
	Tax                     CostSpec
	ExchangeFee             CostSpec
	Slippage                CostSpec
	Liquidity               LiquiditySpec
	PartialFill             PartialFillSpec
	Indicators              map[string]IndicatorSpec
	Entry                   RuleExpr
	Exit                    RuleExpr
	Sizing                  SizingSpec
	Risk                    RiskSpec
	Report                  ReportSpec
	SelectedMetrics         []string
	Universe                UniversePlan
	UniverseExplain         UniverseExplain
	metricRegistry          MetricRegistry
	registry                IndicatorRegistry
	Entries                 []RuleExpr
	Exits                   []RuleExpr
	Rebalance               []RuleExpr
	Stops                   []RuleExpr
}

type InstrumentIdentity struct {
	Symbol       string `json:"symbol"`
	Market       string `json:"market,omitempty"`
	SecurityType string `json:"security_type,omitempty"`
}
