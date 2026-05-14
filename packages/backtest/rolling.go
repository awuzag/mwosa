package backtest

import (
	"math"

	"github.com/samber/oops"
)

type indicatorRuntime interface {
	Add(Bar) (float64, error)
}

func newIndicatorRuntime(spec IndicatorSpec) (indicatorRuntime, error) {
	window := int(spec.Params["window"])
	if window <= 0 {
		return nil, oops.In("backtest_indicator_runtime").With("indicator", spec.ID).New("indicator requires positive window")
	}
	switch spec.ID {
	case "sma":
		return &smaRuntime{spec: spec, window: window}, nil
	case "rsi":
		return &rsiRuntime{spec: spec, window: window}, nil
	case "donchian_high":
		return &extremeRuntime{spec: spec, window: window, better: func(candidate, current float64) bool { return candidate > current }}, nil
	case "donchian_low":
		return &extremeRuntime{spec: spec, window: window, better: func(candidate, current float64) bool { return candidate < current }}, nil
	case "atr":
		return &atrRuntime{spec: spec, window: window}, nil
	case "natr":
		return &natrRuntime{atr: atrRuntime{spec: spec, window: window}}, nil
	default:
		return nil, oops.In("backtest_indicator_runtime").With("indicator", spec.ID).New("indicator is not registered for rolling runtime")
	}
}

type smaRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
	sum    float64
}

func (r *smaRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	r.values = append(r.values, value)
	r.sum += value
	if len(r.values) > r.window {
		r.sum -= r.values[0]
		r.values = r.values[1:]
	}
	if len(r.values) < r.window {
		return math.NaN(), nil
	}
	return r.sum / float64(r.window), nil
}

type rsiRuntime struct {
	spec    IndicatorSpec
	window  int
	seen    int
	prev    float64
	gainSum float64
	lossSum float64
	avgGain float64
	avgLoss float64
	ready   bool
}

func (r *rsiRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	r.seen++
	if r.seen == 1 {
		r.prev = value
		return math.NaN(), nil
	}
	change := value - r.prev
	r.prev = value
	gain := 0.0
	loss := 0.0
	if change >= 0 {
		gain = change
	} else {
		loss = -change
	}
	if !r.ready {
		r.gainSum += gain
		r.lossSum += loss
		if r.seen <= r.window {
			return math.NaN(), nil
		}
		r.avgGain = r.gainSum / float64(r.window)
		r.avgLoss = r.lossSum / float64(r.window)
		r.ready = true
		return rsiValue(r.avgGain, r.avgLoss), nil
	}
	r.avgGain = (r.avgGain*float64(r.window-1) + gain) / float64(r.window)
	r.avgLoss = (r.avgLoss*float64(r.window-1) + loss) / float64(r.window)
	return rsiValue(r.avgGain, r.avgLoss), nil
}

type extremeRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
	better func(candidate, current float64) bool
}

func (r *extremeRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	r.values = append(r.values, value)
	if len(r.values) > r.window {
		r.values = r.values[1:]
	}
	if len(r.values) < r.window {
		return math.NaN(), nil
	}
	current := r.values[0]
	for _, candidate := range r.values[1:] {
		if r.better(candidate, current) {
			current = candidate
		}
	}
	return current, nil
}

type atrRuntime struct {
	spec      IndicatorSpec
	window    int
	prevClose float64
	seen      int
	values    []float64
	sum       float64
}

func (r *atrRuntime) Add(bar Bar) (float64, error) {
	tr := bar.High - bar.Low
	if r.seen > 0 {
		tr = math.Max(tr, math.Abs(bar.High-r.prevClose))
		tr = math.Max(tr, math.Abs(bar.Low-r.prevClose))
	}
	r.prevClose = bar.Close
	r.seen++
	r.values = append(r.values, tr)
	r.sum += tr
	if len(r.values) > r.window {
		r.sum -= r.values[0]
		r.values = r.values[1:]
	}
	if len(r.values) < r.window {
		return math.NaN(), nil
	}
	return r.sum / float64(r.window), nil
}

type natrRuntime struct {
	atr atrRuntime
}

func (r *natrRuntime) Add(bar Bar) (float64, error) {
	atr, err := r.atr.Add(bar)
	if err != nil || math.IsNaN(atr) || math.IsInf(atr, 0) {
		return atr, err
	}
	if bar.Close <= 0 {
		return math.NaN(), nil
	}
	return atr / bar.Close * 100, nil
}
