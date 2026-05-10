package backtest

import (
	"math"

	"github.com/samber/oops"
)

type IndicatorRegistry struct {
	defs map[string]IndicatorDefinition
}

type IndicatorDefinition struct {
	ID        string
	Validate  func(IndicatorSpec) error
	Calculate func(IndicatorSpec, []Bar) ([]float64, error)
}

func NewIndicatorRegistry(definitions ...IndicatorDefinition) (IndicatorRegistry, error) {
	registry := IndicatorRegistry{defs: make(map[string]IndicatorDefinition, len(definitions))}
	for _, definition := range definitions {
		if definition.ID == "" {
			return IndicatorRegistry{}, oops.In("backtest_indicator_registry").New("indicator id is empty")
		}
		if definition.Calculate == nil {
			return IndicatorRegistry{}, oops.In("backtest_indicator_registry").With("indicator", definition.ID).New("indicator calculator is nil")
		}
		if _, exists := registry.defs[definition.ID]; exists {
			return IndicatorRegistry{}, oops.In("backtest_indicator_registry").With("indicator", definition.ID).New("indicator id is duplicated")
		}
		registry.defs[definition.ID] = definition
	}
	return registry, nil
}

func DefaultIndicatorRegistry() (IndicatorRegistry, error) {
	return NewIndicatorRegistry(SMA())
}

func (r IndicatorRegistry) Definition(id string) (IndicatorDefinition, bool) {
	definition, ok := r.defs[id]
	return definition, ok
}

func SMA() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "sma",
		Validate: func(spec IndicatorSpec) error {
			errb := oops.In("backtest_indicator_registry").With("indicator", spec.ID)
			window := int(spec.Params["window"])
			if window <= 0 {
				return errb.New("sma indicator requires positive window")
			}
			if spec.Source.Kind != "price" {
				return errb.With("source_kind", spec.Source.Kind).New("sma indicator source must be price")
			}
			if !isPriceField(spec.Source.Price) {
				return errb.With("price", spec.Source.Price).New("sma indicator source price is unsupported")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			window := int(spec.Params["window"])
			values := make([]float64, len(bars))
			for i := range values {
				values[i] = math.NaN()
			}
			if window <= 0 {
				return nil, oops.In("backtest_indicator_registry").With("indicator", spec.ID).New("sma indicator requires positive window")
			}
			var sum float64
			for i, bar := range bars {
				value, ok := priceValue(bar, spec.Source.Price)
				if !ok {
					return nil, oops.In("backtest_indicator_registry").With("indicator", spec.ID, "price", spec.Source.Price).New("unsupported price field")
				}
				sum += value
				if i >= window {
					previous, _ := priceValue(bars[i-window], spec.Source.Price)
					sum -= previous
				}
				if i >= window-1 {
					values[i] = sum / float64(window)
				}
			}
			return values, nil
		},
	}
}
