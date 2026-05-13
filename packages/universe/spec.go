package universe

import "time"

type PipelineSpec struct {
	Symbols  []string     `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Ref      string       `json:"ref,omitempty" yaml:"ref,omitempty"`
	Schedule ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Pipeline []StepSpec   `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
}

type ScheduleSpec struct {
	Frequency      string   `json:"frequency,omitempty" yaml:"frequency,omitempty"`
	Anchor         string   `json:"anchor,omitempty" yaml:"anchor,omitempty"`
	LookbackPolicy string   `json:"lookback_policy,omitempty" yaml:"lookback_policy,omitempty"`
	Dates          []string `json:"dates,omitempty" yaml:"dates,omitempty"`
}

type StepSpec struct {
	ID       string         `json:"id" yaml:"id"`
	Name     string         `json:"name,omitempty" yaml:"name,omitempty"`
	Params   map[string]any `json:"params,omitempty" yaml:"params,omitempty"`
	Pipeline []StepSpec     `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
}

type Bar struct {
	Time         time.Time `json:"time"`
	Symbol       string    `json:"symbol"`
	Market       string    `json:"market,omitempty"`
	SecurityType string    `json:"security_type,omitempty"`
	Open         float64   `json:"open"`
	High         float64   `json:"high"`
	Low          float64   `json:"low"`
	Close        float64   `json:"close"`
	Volume       float64   `json:"volume,omitempty"`
	TradedAmount float64   `json:"traded_amount,omitempty"`
}
