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
	return NewIndicatorRegistry(SMA(), RSI(), DonchianHigh(), DonchianLow())
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

func RSI() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "rsi",
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, "rsi"); err != nil {
				return err
			}
			window := int(spec.Params["window"])
			if window <= 0 {
				return oops.In("backtest_indicator_registry").With("indicator", spec.ID).New("rsi indicator requires positive window")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			window := int(spec.Params["window"])
			values := nanSeries(len(bars))
			if window <= 0 {
				return nil, oops.In("backtest_indicator_registry").With("indicator", spec.ID).New("rsi indicator requires positive window")
			}
			if len(bars) <= window {
				return values, nil
			}
			var gainSum float64
			var lossSum float64
			for i := 1; i <= window; i++ {
				current, ok := priceValue(bars[i], spec.Source.Price)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				previous, _ := priceValue(bars[i-1], spec.Source.Price)
				change := current - previous
				if change >= 0 {
					gainSum += change
				} else {
					lossSum -= change
				}
			}
			avgGain := gainSum / float64(window)
			avgLoss := lossSum / float64(window)
			values[window] = rsiValue(avgGain, avgLoss)
			for i := window + 1; i < len(bars); i++ {
				current, ok := priceValue(bars[i], spec.Source.Price)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				previous, _ := priceValue(bars[i-1], spec.Source.Price)
				change := current - previous
				gain := 0.0
				loss := 0.0
				if change >= 0 {
					gain = change
				} else {
					loss = -change
				}
				avgGain = (avgGain*float64(window-1) + gain) / float64(window)
				avgLoss = (avgLoss*float64(window-1) + loss) / float64(window)
				values[i] = rsiValue(avgGain, avgLoss)
			}
			return values, nil
		},
	}
}

func DonchianHigh() IndicatorDefinition {
	return rollingExtremeIndicator("donchian_high", func(candidate, current float64) bool {
		return candidate > current
	})
}

func DonchianLow() IndicatorDefinition {
	return rollingExtremeIndicator("donchian_low", func(candidate, current float64) bool {
		return candidate < current
	})
}

func rollingExtremeIndicator(id string, better func(candidate, current float64) bool) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, id); err != nil {
				return err
			}
			window := int(spec.Params["window"])
			if window <= 0 {
				return oops.In("backtest_indicator_registry").With("indicator", spec.ID).New(id + " indicator requires positive window")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			window := int(spec.Params["window"])
			values := nanSeries(len(bars))
			if window <= 0 {
				return nil, oops.In("backtest_indicator_registry").With("indicator", spec.ID).New(id + " indicator requires positive window")
			}
			for i := range bars {
				if i < window-1 {
					continue
				}
				current, ok := priceValue(bars[i-window+1], spec.Source.Price)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				for j := i - window + 2; j <= i; j++ {
					candidate, ok := priceValue(bars[j], spec.Source.Price)
					if !ok {
						return nil, unsupportedIndicatorPrice(spec)
					}
					if better(candidate, current) {
						current = candidate
					}
				}
				values[i] = current
			}
			return values, nil
		},
	}
}

func validatePriceIndicatorSpec(spec IndicatorSpec, id string) error {
	errb := oops.In("backtest_indicator_registry").With("indicator", id)
	if spec.Source.Kind != "price" {
		return errb.With("source_kind", spec.Source.Kind).New(id + " indicator source must be price")
	}
	if !isPriceField(spec.Source.Price) {
		return errb.With("price", spec.Source.Price).New(id + " indicator source price is unsupported")
	}
	return nil
}

func nanSeries(length int) []float64 {
	values := make([]float64, length)
	for i := range values {
		values[i] = math.NaN()
	}
	return values
}

func rsiValue(avgGain float64, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50
		}
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}

func unsupportedIndicatorPrice(spec IndicatorSpec) error {
	return oops.In("backtest_indicator_registry").With("indicator", spec.ID, "price", spec.Source.Price).New("unsupported price field")
}
