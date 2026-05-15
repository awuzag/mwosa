package backtest

import (
	"math"

	"github.com/samber/oops"
)

type indicatorRuntime interface {
	Add(Bar) (float64, error)
}

func newIndicatorRuntime(spec IndicatorSpec) (indicatorRuntime, error) {
	if spec.ID == "macd" {
		return newMACDRuntime(spec)
	}
	if spec.ID == "stochastic" {
		return newStochasticRuntime(spec)
	}
	if spec.ID == "adx" || spec.ID == "di_plus" || spec.ID == "di_minus" {
		return newDirectionalMovementRuntime(spec)
	}
	window := int(spec.Params["window"])
	if window <= 0 {
		return nil, oops.In("backtest_indicator_runtime").With("indicator", spec.ID).New("indicator requires positive window")
	}
	switch spec.ID {
	case "sma":
		return &smaRuntime{spec: spec, window: window}, nil
	case "ema":
		return &emaRuntime{spec: spec, window: window, multiplier: 2 / float64(window+1)}, nil
	case "wma":
		return &wmaRuntime{spec: spec, window: window}, nil
	case "hma":
		halfWindow := max(1, window/2)
		smoothingWindow := max(1, int(math.Sqrt(float64(window))))
		return &hmaRuntime{
			spec:      spec,
			fast:      wmaValueRuntime{window: halfWindow},
			slow:      wmaValueRuntime{window: window},
			smoothing: wmaValueRuntime{window: smoothingWindow},
		}, nil
	case "kama":
		if err := validateKAMAParams(spec); err != nil {
			return nil, err
		}
		fastWindow := int(spec.Params["fast_window"])
		if fastWindow == 0 {
			fastWindow = 2
		}
		slowWindow := int(spec.Params["slow_window"])
		if slowWindow == 0 {
			slowWindow = 30
		}
		return &kamaRuntime{
			spec:   spec,
			window: window,
			fastSC: 2 / float64(fastWindow+1),
			slowSC: 2 / float64(slowWindow+1),
		}, nil
	case "rsi":
		return &rsiRuntime{spec: spec, window: window}, nil
	case "roc":
		return &rocRuntime{spec: spec, window: window}, nil
	case "donchian_high":
		return &extremeRuntime{spec: spec, window: window, better: func(candidate, current float64) bool { return candidate > current }}, nil
	case "donchian_low":
		return &extremeRuntime{spec: spec, window: window, better: func(candidate, current float64) bool { return candidate < current }}, nil
	case "bollinger_middle":
		return &bollingerRuntime{spec: spec, window: window, band: "middle"}, nil
	case "bollinger_upper":
		return &bollingerRuntime{spec: spec, window: window, band: "upper"}, nil
	case "bollinger_lower":
		return &bollingerRuntime{spec: spec, window: window, band: "lower"}, nil
	case "keltner_middle", "keltner_upper", "keltner_lower":
		atrWindow := int(spec.Params["atr_window"])
		if atrWindow == 0 {
			atrWindow = window
		}
		if atrWindow < 0 {
			return nil, oops.In("backtest_indicator_runtime").With("indicator", spec.ID, "atr_window", atrWindow).New("keltner indicator atr_window must not be negative")
		}
		return &keltnerRuntime{
			spec: spec,
			ema:  newEMAValueRuntime(window),
			atr:  atrRuntime{spec: IndicatorSpec{ID: "atr"}, window: atrWindow},
			band: keltnerBand(spec.ID),
		}, nil
	case "atr":
		return &atrRuntime{spec: spec, window: window}, nil
	case "natr":
		return &natrRuntime{atr: atrRuntime{spec: spec, window: window}}, nil
	case "zscore":
		return &zscoreRuntime{spec: spec, window: window}, nil
	case "correlation", "beta":
		if err := validatePairedReturnIndicatorSpec(spec, spec.ID); err != nil {
			return nil, err
		}
		return &pairedReturnRuntime{spec: spec, window: window}, nil
	default:
		return nil, oops.In("backtest_indicator_runtime").With("indicator", spec.ID).New("indicator is not registered for rolling runtime")
	}
}

type stochasticRuntime struct {
	kWindow int
	dWindow int
	output  string
	bars    []Bar
	kValues []float64
}

func newStochasticRuntime(spec IndicatorSpec) (indicatorRuntime, error) {
	errb := oops.In("backtest_indicator_runtime").With("indicator", "stochastic")
	kWindow := int(spec.Params["k_window"])
	if kWindow == 0 {
		kWindow = int(spec.Params["window"])
	}
	dWindow := int(spec.Params["d_window"])
	if dWindow == 0 {
		dWindow = 3
	}
	if kWindow <= 0 || dWindow <= 0 {
		return nil, errb.New("stochastic indicator requires positive k_window and d_window")
	}
	output := withDefault(spec.Output, "k")
	switch output {
	case "k", "d", "signal":
	default:
		return nil, errb.With("output", spec.Output).New("unsupported stochastic output")
	}
	return &stochasticRuntime{kWindow: kWindow, dWindow: dWindow, output: output}, nil
}

func (r *stochasticRuntime) Add(bar Bar) (float64, error) {
	r.bars = append(r.bars, bar)
	if len(r.bars) > r.kWindow {
		r.bars = r.bars[1:]
	}
	if len(r.bars) < r.kWindow {
		return math.NaN(), nil
	}
	lowest := r.bars[0].Low
	highest := r.bars[0].High
	for _, candidate := range r.bars[1:] {
		lowest = math.Min(lowest, candidate.Low)
		highest = math.Max(highest, candidate.High)
	}
	k := 50.0
	if highest != lowest {
		k = (bar.Close - lowest) / (highest - lowest) * 100
	}
	r.kValues = append(r.kValues, k)
	if len(r.kValues) > r.dWindow {
		r.kValues = r.kValues[1:]
	}
	switch r.output {
	case "k":
		return k, nil
	case "d", "signal":
		if len(r.kValues) < r.dWindow {
			return math.NaN(), nil
		}
		var sum float64
		for _, value := range r.kValues {
			sum += value
		}
		return sum / float64(r.dWindow), nil
	default:
		return math.NaN(), nil
	}
}

type directionalMovementRuntime struct {
	window          int
	output          string
	seen            int
	prev            Bar
	trValues        []float64
	plusDMValues    []float64
	minusDMValues   []float64
	smoothedTR      float64
	smoothedPlusDM  float64
	smoothedMinusDM float64
	dxValues        []float64
	adx             float64
	readyDM         bool
	readyADX        bool
}

func newDirectionalMovementRuntime(spec IndicatorSpec) (indicatorRuntime, error) {
	if err := validateDirectionalMovementParams(spec, spec.ID); err != nil {
		return nil, err
	}
	return &directionalMovementRuntime{
		window: int(spec.Params["window"]),
		output: directionalMovementOutput(spec, spec.ID),
	}, nil
}

func (r *directionalMovementRuntime) Add(bar Bar) (float64, error) {
	if r.seen == 0 {
		r.prev = bar
		r.seen = 1
		return math.NaN(), nil
	}
	tr, plusDM, minusDM := directionalMovementComponents(r.prev, bar)
	r.prev = bar
	r.seen++

	if !r.readyDM {
		r.trValues = append(r.trValues, tr)
		r.plusDMValues = append(r.plusDMValues, plusDM)
		r.minusDMValues = append(r.minusDMValues, minusDM)
		if len(r.trValues) < r.window {
			return math.NaN(), nil
		}
		r.smoothedTR = sumFloat64(r.trValues)
		r.smoothedPlusDM = sumFloat64(r.plusDMValues)
		r.smoothedMinusDM = sumFloat64(r.minusDMValues)
		r.readyDM = true
	} else {
		window := float64(r.window)
		r.smoothedTR = r.smoothedTR - r.smoothedTR/window + tr
		r.smoothedPlusDM = r.smoothedPlusDM - r.smoothedPlusDM/window + plusDM
		r.smoothedMinusDM = r.smoothedMinusDM - r.smoothedMinusDM/window + minusDM
	}

	plusDI, minusDI, dx := directionalMovementValues(r.smoothedTR, r.smoothedPlusDM, r.smoothedMinusDM)
	switch r.output {
	case "di_plus":
		return plusDI, nil
	case "di_minus":
		return minusDI, nil
	}
	if !r.readyADX {
		r.dxValues = append(r.dxValues, dx)
		if len(r.dxValues) < r.window {
			return math.NaN(), nil
		}
		r.adx = sumFloat64(r.dxValues) / float64(r.window)
		r.readyADX = true
		return r.adx, nil
	}
	r.adx = (r.adx*float64(r.window-1) + dx) / float64(r.window)
	return r.adx, nil
}

func directionalMovementComponents(prev Bar, current Bar) (float64, float64, float64) {
	tr := current.High - current.Low
	tr = math.Max(tr, math.Abs(current.High-prev.Close))
	tr = math.Max(tr, math.Abs(current.Low-prev.Close))

	upMove := current.High - prev.High
	downMove := prev.Low - current.Low
	plusDM := 0.0
	if upMove > downMove && upMove > 0 {
		plusDM = upMove
	}
	minusDM := 0.0
	if downMove > upMove && downMove > 0 {
		minusDM = downMove
	}
	return tr, plusDM, minusDM
}

func directionalMovementValues(smoothedTR float64, smoothedPlusDM float64, smoothedMinusDM float64) (float64, float64, float64) {
	if smoothedTR == 0 {
		return 0, 0, 0
	}
	plusDI := 100 * smoothedPlusDM / smoothedTR
	minusDI := 100 * smoothedMinusDM / smoothedTR
	diSum := plusDI + minusDI
	if diSum == 0 {
		return plusDI, minusDI, 0
	}
	return plusDI, minusDI, 100 * math.Abs(plusDI-minusDI) / diSum
}

func sumFloat64(values []float64) float64 {
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum
}

func keltnerBand(id string) string {
	switch id {
	case "keltner_middle":
		return "middle"
	case "keltner_upper":
		return "upper"
	case "keltner_lower":
		return "lower"
	default:
		return ""
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

type emaRuntime struct {
	spec       IndicatorSpec
	window     int
	values     []float64
	sum        float64
	ema        float64
	multiplier float64
	ready      bool
}

func (r *emaRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	if !r.ready {
		r.values = append(r.values, value)
		r.sum += value
		if len(r.values) < r.window {
			return math.NaN(), nil
		}
		r.ema = r.sum / float64(r.window)
		r.ready = true
		return r.ema, nil
	}
	r.ema = (value-r.ema)*r.multiplier + r.ema
	return r.ema, nil
}

type wmaRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
}

func (r *wmaRuntime) Add(bar Bar) (float64, error) {
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
	var weighted float64
	var totalWeight float64
	for index, candidate := range r.values {
		weight := float64(index + 1)
		weighted += candidate * weight
		totalWeight += weight
	}
	return weighted / totalWeight, nil
}

type wmaValueRuntime struct {
	window int
	values []float64
}

func (r *wmaValueRuntime) Add(value float64) (float64, bool) {
	r.values = append(r.values, value)
	if len(r.values) > r.window {
		r.values = r.values[1:]
	}
	if len(r.values) < r.window {
		return math.NaN(), false
	}
	var weighted float64
	var totalWeight float64
	for index, candidate := range r.values {
		weight := float64(index + 1)
		weighted += candidate * weight
		totalWeight += weight
	}
	return weighted / totalWeight, true
}

type hmaRuntime struct {
	spec      IndicatorSpec
	fast      wmaValueRuntime
	slow      wmaValueRuntime
	smoothing wmaValueRuntime
}

func (r *hmaRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	fast, fastOK := r.fast.Add(value)
	slow, slowOK := r.slow.Add(value)
	if !fastOK || !slowOK {
		return math.NaN(), nil
	}
	hmaInput := 2*fast - slow
	smoothed, smoothedOK := r.smoothing.Add(hmaInput)
	if !smoothedOK {
		return math.NaN(), nil
	}
	return smoothed, nil
}

type kamaRuntime struct {
	spec   IndicatorSpec
	window int
	fastSC float64
	slowSC float64
	values []float64
	kama   float64
	ready  bool
}

func (r *kamaRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	r.values = append(r.values, value)
	if len(r.values) < r.window {
		return math.NaN(), nil
	}
	if !r.ready {
		var sum float64
		for _, candidate := range r.values[:r.window] {
			sum += candidate
		}
		r.kama = sum / float64(r.window)
		r.ready = true
		return r.kama, nil
	}
	if len(r.values) > r.window+1 {
		r.values = r.values[1:]
	}
	change := math.Abs(r.values[len(r.values)-1] - r.values[len(r.values)-r.window-1])
	var volatility float64
	for i := len(r.values) - r.window; i < len(r.values); i++ {
		volatility += math.Abs(r.values[i] - r.values[i-1])
	}
	efficiencyRatio := 0.0
	if volatility != 0 {
		efficiencyRatio = change / volatility
	}
	smoothingConstant := math.Pow(efficiencyRatio*(r.fastSC-r.slowSC)+r.slowSC, 2)
	r.kama = r.kama + smoothingConstant*(value-r.kama)
	return r.kama, nil
}

type macdRuntime struct {
	spec   IndicatorSpec
	fast   emaValueRuntime
	slow   emaValueRuntime
	signal emaValueRuntime
	output string
}

func newMACDRuntime(spec IndicatorSpec) (indicatorRuntime, error) {
	fastWindow := int(spec.Params["fast_window"])
	slowWindow := int(spec.Params["slow_window"])
	signalWindow := int(spec.Params["signal_window"])
	errb := oops.In("backtest_indicator_runtime").With("indicator", "macd")
	if fastWindow <= 0 || slowWindow <= 0 || signalWindow <= 0 {
		return nil, errb.New("macd indicator requires positive fast_window, slow_window, and signal_window")
	}
	if fastWindow >= slowWindow {
		return nil, errb.With("fast_window", fastWindow, "slow_window", slowWindow).New("macd indicator requires fast_window less than slow_window")
	}
	output := withDefault(spec.Output, "line")
	switch output {
	case "line", "signal", "histogram":
	default:
		return nil, errb.With("output", spec.Output).New("unsupported macd output")
	}
	return &macdRuntime{
		spec:   spec,
		fast:   newEMAValueRuntime(fastWindow),
		slow:   newEMAValueRuntime(slowWindow),
		signal: newEMAValueRuntime(signalWindow),
		output: output,
	}, nil
}

func (r *macdRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	fast, fastOK := r.fast.Add(value)
	slow, slowOK := r.slow.Add(value)
	if !fastOK || !slowOK {
		return math.NaN(), nil
	}
	line := fast - slow
	signal, signalOK := r.signal.Add(line)
	switch r.output {
	case "line":
		return line, nil
	case "signal":
		if !signalOK {
			return math.NaN(), nil
		}
		return signal, nil
	case "histogram":
		if !signalOK {
			return math.NaN(), nil
		}
		return line - signal, nil
	default:
		return math.NaN(), nil
	}
}

type emaValueRuntime struct {
	window     int
	values     []float64
	sum        float64
	ema        float64
	multiplier float64
	ready      bool
}

func newEMAValueRuntime(window int) emaValueRuntime {
	return emaValueRuntime{window: window, multiplier: 2 / float64(window+1)}
}

func (r *emaValueRuntime) Add(value float64) (float64, bool) {
	if !r.ready {
		r.values = append(r.values, value)
		r.sum += value
		if len(r.values) < r.window {
			return math.NaN(), false
		}
		r.ema = r.sum / float64(r.window)
		r.ready = true
		return r.ema, true
	}
	r.ema = (value-r.ema)*r.multiplier + r.ema
	return r.ema, true
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

type rocRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
}

func (r *rocRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	r.values = append(r.values, value)
	if len(r.values) <= r.window {
		return math.NaN(), nil
	}
	previous := r.values[len(r.values)-r.window-1]
	if previous == 0 {
		return math.NaN(), nil
	}
	return (value/previous - 1) * 100, nil
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

type bollingerRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
	band   string
}

func (r *bollingerRuntime) Add(bar Bar) (float64, error) {
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
	mean, standardDeviation := meanAndPopulationStdDev(r.values)
	multiplier := r.spec.Params["multiplier"]
	if multiplier == 0 {
		multiplier = 2
	}
	switch r.band {
	case "middle":
		return mean, nil
	case "upper":
		return mean + standardDeviation*multiplier, nil
	case "lower":
		return mean - standardDeviation*multiplier, nil
	default:
		return math.NaN(), nil
	}
}

type keltnerRuntime struct {
	spec IndicatorSpec
	ema  emaValueRuntime
	atr  atrRuntime
	band string
}

func (r *keltnerRuntime) Add(bar Bar) (float64, error) {
	value, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	middle, middleOK := r.ema.Add(value)
	atr, err := r.atr.Add(bar)
	if err != nil {
		return 0, err
	}
	if !middleOK || math.IsNaN(atr) || math.IsInf(atr, 0) {
		return math.NaN(), nil
	}
	multiplier := r.spec.Params["multiplier"]
	if multiplier == 0 {
		multiplier = 2
	}
	switch r.band {
	case "middle":
		return middle, nil
	case "upper":
		return middle + atr*multiplier, nil
	case "lower":
		return middle - atr*multiplier, nil
	default:
		return math.NaN(), nil
	}
}

func meanAndPopulationStdDev(values []float64) (float64, float64) {
	var sum float64
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	return mean, math.Sqrt(variance / float64(len(values)))
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

type zscoreRuntime struct {
	spec   IndicatorSpec
	window int
	values []float64
}

func (r *zscoreRuntime) Add(bar Bar) (float64, error) {
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
	mean, standardDeviation := meanAndPopulationStdDev(r.values)
	if standardDeviation == 0 {
		return 0, nil
	}
	return (value - mean) / standardDeviation, nil
}

type pairedReturnRuntime struct {
	spec            IndicatorSpec
	window          int
	seen            bool
	previousSource  float64
	previousCompare float64
	sourceReturns   []float64
	compareReturns  []float64
}

func (r *pairedReturnRuntime) Add(bar Bar) (float64, error) {
	source, ok := priceValue(bar, r.spec.Source.Price)
	if !ok {
		return 0, unsupportedIndicatorPrice(r.spec)
	}
	compareValue, ok := priceValue(bar, r.spec.Compare.Price)
	if !ok {
		return 0, oops.In("backtest_indicator_registry").With("indicator", r.spec.ID, "price", r.spec.Compare.Price).New("unsupported compare price field")
	}
	if !r.seen {
		r.previousSource = source
		r.previousCompare = compareValue
		r.seen = true
		return math.NaN(), nil
	}
	if r.previousSource <= 0 || r.previousCompare <= 0 {
		r.previousSource = source
		r.previousCompare = compareValue
		return math.NaN(), nil
	}
	sourceReturn := source/r.previousSource - 1
	compareReturn := compareValue/r.previousCompare - 1
	r.previousSource = source
	r.previousCompare = compareValue
	r.sourceReturns = append(r.sourceReturns, sourceReturn)
	r.compareReturns = append(r.compareReturns, compareReturn)
	if len(r.sourceReturns) > r.window {
		r.sourceReturns = r.sourceReturns[1:]
		r.compareReturns = r.compareReturns[1:]
	}
	if len(r.sourceReturns) < r.window {
		return math.NaN(), nil
	}
	covariance, sourceVariance, compareVariance := pairedReturnMoments(r.sourceReturns, r.compareReturns)
	switch r.spec.ID {
	case "correlation":
		if sourceVariance == 0 || compareVariance == 0 {
			return 0, nil
		}
		return covariance / math.Sqrt(sourceVariance*compareVariance), nil
	case "beta":
		if compareVariance == 0 {
			return 0, nil
		}
		return covariance / compareVariance, nil
	default:
		return math.NaN(), nil
	}
}

func pairedReturnMoments(sourceReturns []float64, compareReturns []float64) (float64, float64, float64) {
	sourceMean, _ := meanAndPopulationStdDev(sourceReturns)
	compareMean, _ := meanAndPopulationStdDev(compareReturns)
	var covariance float64
	var sourceVariance float64
	var compareVariance float64
	for i := range sourceReturns {
		sourceDiff := sourceReturns[i] - sourceMean
		compareDiff := compareReturns[i] - compareMean
		covariance += sourceDiff * compareDiff
		sourceVariance += sourceDiff * sourceDiff
		compareVariance += compareDiff * compareDiff
	}
	length := float64(len(sourceReturns))
	return covariance / length, sourceVariance / length, compareVariance / length
}
