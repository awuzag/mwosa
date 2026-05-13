package universe

import (
	"math"

	"github.com/samber/oops"
)

type IndicatorSpec struct {
	ID     string
	Field  string
	Window int
}

type IndicatorRegistry struct {
	defs map[string]IndicatorDefinition
}

type IndicatorDefinition struct {
	ID        string
	Calculate func(IndicatorSpec, []Bar) ([]float64, error)
}

func NewIndicatorRegistry(definitions ...IndicatorDefinition) (IndicatorRegistry, error) {
	registry := IndicatorRegistry{defs: make(map[string]IndicatorDefinition, len(definitions))}
	for _, definition := range definitions {
		if definition.ID == "" {
			return IndicatorRegistry{}, oops.In("universe_indicator_registry").New("indicator id is empty")
		}
		if definition.Calculate == nil {
			return IndicatorRegistry{}, oops.In("universe_indicator_registry").With("indicator", definition.ID).New("indicator calculator is nil")
		}
		if _, exists := registry.defs[definition.ID]; exists {
			return IndicatorRegistry{}, oops.In("universe_indicator_registry").With("indicator", definition.ID).New("indicator id is duplicated")
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
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			if spec.Window <= 0 {
				return nil, oops.In("universe_indicator_registry").With("indicator", spec.ID).New("sma indicator requires positive window")
			}
			values := nanSeries(len(bars))
			var sum float64
			for i, bar := range bars {
				value, ok := priceValue(bar, spec.Field)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				sum += value
				if i >= spec.Window {
					previous, _ := priceValue(bars[i-spec.Window], spec.Field)
					sum -= previous
				}
				if i >= spec.Window-1 {
					values[i] = sum / float64(spec.Window)
				}
			}
			return values, nil
		},
	}
}

func RSI() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "rsi",
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			if spec.Window <= 0 {
				return nil, oops.In("universe_indicator_registry").With("indicator", spec.ID).New("rsi indicator requires positive window")
			}
			values := nanSeries(len(bars))
			if len(bars) <= spec.Window {
				return values, nil
			}
			var gainSum float64
			var lossSum float64
			for i := 1; i <= spec.Window; i++ {
				current, ok := priceValue(bars[i], spec.Field)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				previous, _ := priceValue(bars[i-1], spec.Field)
				change := current - previous
				if change >= 0 {
					gainSum += change
				} else {
					lossSum -= change
				}
			}
			avgGain := gainSum / float64(spec.Window)
			avgLoss := lossSum / float64(spec.Window)
			values[spec.Window] = rsiValue(avgGain, avgLoss)
			for i := spec.Window + 1; i < len(bars); i++ {
				current, ok := priceValue(bars[i], spec.Field)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				previous, _ := priceValue(bars[i-1], spec.Field)
				change := current - previous
				gain := 0.0
				loss := 0.0
				if change >= 0 {
					gain = change
				} else {
					loss = -change
				}
				avgGain = (avgGain*float64(spec.Window-1) + gain) / float64(spec.Window)
				avgLoss = (avgLoss*float64(spec.Window-1) + loss) / float64(spec.Window)
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
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			if spec.Window <= 0 {
				return nil, oops.In("universe_indicator_registry").With("indicator", spec.ID).New(id + " indicator requires positive window")
			}
			values := nanSeries(len(bars))
			for i := range bars {
				if i < spec.Window-1 {
					continue
				}
				current, ok := priceValue(bars[i-spec.Window+1], spec.Field)
				if !ok {
					return nil, unsupportedIndicatorPrice(spec)
				}
				for j := i - spec.Window + 2; j <= i; j++ {
					candidate, ok := priceValue(bars[j], spec.Field)
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

func priceValue(bar Bar, field string) (float64, bool) {
	switch field {
	case "open":
		return bar.Open, true
	case "high":
		return bar.High, true
	case "low":
		return bar.Low, true
	case "close", "":
		return bar.Close, true
	case "volume":
		return bar.Volume, true
	case "traded_amount":
		return bar.TradedAmount, true
	default:
		return 0, false
	}
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
	return oops.In("universe_indicator_registry").With("indicator", spec.ID, "field", spec.Field).New("unsupported price field")
}
