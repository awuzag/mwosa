package backtest

import (
	"math"

	"github.com/samber/oops"
)

const (
	DefaultIndicatorRegistryVersion = "default-indicators/v1"
	CustomIndicatorRegistryVersion  = "custom-indicators"
)

type IndicatorRegistry struct {
	version string
	defs    map[string]IndicatorDefinition
}

type IndicatorDefinition struct {
	ID        string
	Validate  func(IndicatorSpec) error
	Calculate func(IndicatorSpec, []Bar) ([]float64, error)
}

func NewIndicatorRegistry(definitions ...IndicatorDefinition) (IndicatorRegistry, error) {
	registry := IndicatorRegistry{
		version: CustomIndicatorRegistryVersion,
		defs:    make(map[string]IndicatorDefinition, len(definitions)),
	}
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
	registry, err := NewIndicatorRegistry(
		SMA(),
		EMA(),
		WMA(),
		HMA(),
		KAMA(),
		RSI(),
		MACD(),
		Stochastic(),
		ADX(),
		DIPlus(),
		DIMinus(),
		ROC(),
		DonchianHigh(),
		DonchianLow(),
		BollingerMiddle(),
		BollingerUpper(),
		BollingerLower(),
		KeltnerMiddle(),
		KeltnerUpper(),
		KeltnerLower(),
		ATR(),
		NATR(),
		ZScore(),
		Correlation(),
		Beta(),
	)
	if err != nil {
		return IndicatorRegistry{}, err
	}
	registry.version = DefaultIndicatorRegistryVersion
	return registry, nil
}

func (r IndicatorRegistry) Definition(id string) (IndicatorDefinition, bool) {
	definition, ok := r.defs[id]
	return definition, ok
}

func (r IndicatorRegistry) Version() string {
	if r.version == "" {
		return CustomIndicatorRegistryVersion
	}
	return r.version
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

func EMA() IndicatorDefinition {
	return priceRuntimeIndicator("ema")
}

func WMA() IndicatorDefinition {
	return priceRuntimeIndicator("wma")
}

func HMA() IndicatorDefinition {
	return priceRuntimeIndicator("hma")
}

func KAMA() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "kama",
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, "kama"); err != nil {
				return err
			}
			return validateKAMAParams(spec)
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func MACD() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "macd",
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, "macd"); err != nil {
				return err
			}
			fastWindow := int(spec.Params["fast_window"])
			slowWindow := int(spec.Params["slow_window"])
			signalWindow := int(spec.Params["signal_window"])
			errb := oops.In("backtest_indicator_registry").With("indicator", "macd")
			if fastWindow <= 0 || slowWindow <= 0 || signalWindow <= 0 {
				return errb.New("macd indicator requires positive fast_window, slow_window, and signal_window")
			}
			if fastWindow >= slowWindow {
				return errb.With("fast_window", fastWindow, "slow_window", slowWindow).New("macd indicator requires fast_window less than slow_window")
			}
			switch withDefault(spec.Output, "line") {
			case "line", "signal", "histogram":
				return nil
			default:
				return errb.With("output", spec.Output).New("unsupported macd output")
			}
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newMACDRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
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

func Stochastic() IndicatorDefinition {
	return IndicatorDefinition{
		ID: "stochastic",
		Validate: func(spec IndicatorSpec) error {
			errb := oops.In("backtest_indicator_registry").With("indicator", "stochastic")
			kWindow := int(spec.Params["k_window"])
			if kWindow == 0 {
				kWindow = int(spec.Params["window"])
			}
			dWindow := int(spec.Params["d_window"])
			if dWindow == 0 {
				dWindow = 3
			}
			if kWindow <= 0 || dWindow <= 0 {
				return errb.New("stochastic indicator requires positive k_window and d_window")
			}
			if spec.Source.Kind != "" && spec.Source.Kind != "price" {
				return errb.With("source_kind", spec.Source.Kind).New("stochastic indicator source must be price when configured")
			}
			if spec.Source.Price != "" && !isPriceField(spec.Source.Price) {
				return errb.With("price", spec.Source.Price).New("stochastic indicator source price is unsupported")
			}
			switch withDefault(spec.Output, "k") {
			case "k", "d", "signal":
				return nil
			default:
				return errb.With("output", spec.Output).New("unsupported stochastic output")
			}
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newStochasticRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func ADX() IndicatorDefinition {
	return directionalMovementIndicator("adx")
}

func DIPlus() IndicatorDefinition {
	return directionalMovementIndicator("di_plus")
}

func DIMinus() IndicatorDefinition {
	return directionalMovementIndicator("di_minus")
}

func ZScore() IndicatorDefinition {
	return priceRuntimeIndicator("zscore")
}

func Correlation() IndicatorDefinition {
	return pairedReturnIndicator("correlation")
}

func Beta() IndicatorDefinition {
	return pairedReturnIndicator("beta")
}

func ROC() IndicatorDefinition {
	return priceRuntimeIndicator("roc")
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

func ATR() IndicatorDefinition {
	return rollingRuntimeIndicator("atr")
}

func NATR() IndicatorDefinition {
	return rollingRuntimeIndicator("natr")
}

func BollingerMiddle() IndicatorDefinition {
	return bollingerIndicator("bollinger_middle")
}

func BollingerUpper() IndicatorDefinition {
	return bollingerIndicator("bollinger_upper")
}

func BollingerLower() IndicatorDefinition {
	return bollingerIndicator("bollinger_lower")
}

func KeltnerMiddle() IndicatorDefinition {
	return keltnerIndicator("keltner_middle")
}

func KeltnerUpper() IndicatorDefinition {
	return keltnerIndicator("keltner_upper")
}

func KeltnerLower() IndicatorDefinition {
	return keltnerIndicator("keltner_lower")
}

func priceRuntimeIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, id); err != nil {
				return err
			}
			window := int(spec.Params["window"])
			if window <= 0 {
				return oops.In("backtest_indicator_registry").With("indicator", id).New(id + " indicator requires positive window")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func bollingerIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, id); err != nil {
				return err
			}
			window := int(spec.Params["window"])
			if window <= 0 {
				return oops.In("backtest_indicator_registry").With("indicator", id).New(id + " indicator requires positive window")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func keltnerIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			if err := validatePriceIndicatorSpec(spec, id); err != nil {
				return err
			}
			errb := oops.In("backtest_indicator_registry").With("indicator", id)
			window := int(spec.Params["window"])
			if window <= 0 {
				return errb.New(id + " indicator requires positive window")
			}
			atrWindow := int(spec.Params["atr_window"])
			if atrWindow < 0 {
				return errb.With("atr_window", atrWindow).New(id + " indicator atr_window must not be negative")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func directionalMovementIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			return validateDirectionalMovementParams(spec, id)
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func rollingRuntimeIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			errb := oops.In("backtest_indicator_registry").With("indicator", id)
			window := int(spec.Params["window"])
			if window <= 0 {
				return errb.New(id + " indicator requires positive window")
			}
			return nil
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
}

func pairedReturnIndicator(id string) IndicatorDefinition {
	return IndicatorDefinition{
		ID: id,
		Validate: func(spec IndicatorSpec) error {
			return validatePairedReturnIndicatorSpec(spec, id)
		},
		Calculate: func(spec IndicatorSpec, bars []Bar) ([]float64, error) {
			runtime, err := newIndicatorRuntime(spec)
			if err != nil {
				return nil, err
			}
			values := nanSeries(len(bars))
			for i, bar := range bars {
				value, err := runtime.Add(bar)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}
			return values, nil
		},
	}
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

func validatePairedReturnIndicatorSpec(spec IndicatorSpec, id string) error {
	errb := oops.In("backtest_indicator_registry").With("indicator", id)
	if err := validatePriceIndicatorSpec(spec, id); err != nil {
		return err
	}
	if spec.Compare.Empty() {
		return errb.New(id + " indicator requires compare source")
	}
	if spec.Compare.Kind != "price" {
		return errb.With("compare_kind", spec.Compare.Kind).New(id + " indicator compare source must be price")
	}
	if !isPriceField(spec.Compare.Price) {
		return errb.With("compare_price", spec.Compare.Price).New(id + " indicator compare price is unsupported")
	}
	window := int(spec.Params["window"])
	if window <= 0 {
		return errb.New(id + " indicator requires positive window")
	}
	if spec.Output != "" {
		return errb.With("output", spec.Output).New(id + " indicator does not support output")
	}
	return nil
}

func validateKAMAParams(spec IndicatorSpec) error {
	errb := oops.In("backtest_indicator_registry").With("indicator", "kama")
	window := int(spec.Params["window"])
	fastWindow := int(spec.Params["fast_window"])
	if fastWindow == 0 {
		fastWindow = 2
	}
	slowWindow := int(spec.Params["slow_window"])
	if slowWindow == 0 {
		slowWindow = 30
	}
	if window <= 0 {
		return errb.New("kama indicator requires positive window")
	}
	if fastWindow <= 0 || slowWindow <= 0 {
		return errb.New("kama indicator requires positive fast_window and slow_window")
	}
	if fastWindow >= slowWindow {
		return errb.With("fast_window", fastWindow, "slow_window", slowWindow).New("kama indicator requires fast_window less than slow_window")
	}
	return nil
}

func validateDirectionalMovementParams(spec IndicatorSpec, id string) error {
	errb := oops.In("backtest_indicator_registry").With("indicator", id)
	window := int(spec.Params["window"])
	if window <= 0 {
		return errb.New(id + " indicator requires positive window")
	}
	if spec.Source.Kind != "" && spec.Source.Kind != "price" {
		return errb.With("source_kind", spec.Source.Kind).New(id + " indicator source must be price when configured")
	}
	if spec.Source.Price != "" && !isPriceField(spec.Source.Price) {
		return errb.With("price", spec.Source.Price).New(id + " indicator source price is unsupported")
	}
	switch directionalMovementOutput(spec, id) {
	case "adx", "di_plus", "di_minus":
		return nil
	default:
		return errb.With("output", spec.Output).New("unsupported directional movement output")
	}
}

func directionalMovementOutput(spec IndicatorSpec, id string) string {
	if spec.Output != "" {
		return spec.Output
	}
	return id
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
