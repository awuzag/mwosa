package backtest

import "time"

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

type StrategyRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type DataSpec struct {
	Market       string `json:"market" yaml:"market"`
	SecurityType string `json:"security_type" yaml:"security_type"`
	Timeframe    string `json:"timeframe" yaml:"timeframe"`
	From         string `json:"from" yaml:"from"`
	To           string `json:"to" yaml:"to"`
}

type UniverseSpec struct {
	Symbols []string `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Ref     string   `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type BenchmarkSpec struct {
	Symbol string `json:"symbol,omitempty" yaml:"symbol,omitempty"`
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
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
	From            time.Time
	To              time.Time
	Timeframe       string
	Market          string
	SecurityType    string
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
	metricRegistry  MetricRegistry
	registry        IndicatorRegistry
}
