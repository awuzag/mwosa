package backtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	"github.com/samber/oops"
)

type Engine struct {
	feed StreamingFeed
}

func NewEngine(feed StreamingFeed) (Engine, error) {
	if feed == nil {
		return Engine{}, oops.In("backtest_engine").New("feed is nil")
	}
	return Engine{feed: feed}, nil
}

type Result struct {
	RunName         string               `json:"run_name"`
	StrategyName    string               `json:"strategy_name"`
	Symbols         []string             `json:"symbols"`
	Instruments     []InstrumentIdentity `json:"instruments,omitempty"`
	Period          Period               `json:"period"`
	Market          string               `json:"market"`
	Timeframe       string               `json:"timeframe"`
	Currency        string               `json:"currency"`
	Execution       ExecutionAssumption  `json:"execution"`
	InitialCash     float64              `json:"initial_cash"`
	Benchmark       BenchmarkSpec        `json:"benchmark,omitempty"`
	FinalEquity     float64              `json:"-"`
	TotalReturn     float64              `json:"-"`
	MaxDrawdown     float64              `json:"-"`
	TradeCount      int                  `json:"-"`
	WinRate         float64              `json:"-"`
	AverageTradeRet float64              `json:"-"`
	RealizedPnL     float64              `json:"-"`
	Metrics         Metrics              `json:"metrics"`
	SelectedMetrics []string             `json:"-"`
	Trades          []Trade              `json:"trades"`
	EquityCurve     []EquityPoint        `json:"equity_curve"`
	RiskEvents      []Event              `json:"risk_events,omitempty"`
	ExecutionEvents []Event              `json:"execution_events,omitempty"`
	Universe        UniverseExplain      `json:"universe"`
	UnfilledCount   int                  `json:"-"`
	ResultHash      string               `json:"result_hash"`
	BenchmarkBars   []Bar                `json:"-"`
}

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type ExecutionAssumption struct {
	Fill       string   `json:"fill"`
	Commission CostSpec `json:"commission,omitempty"`
	Slippage   CostSpec `json:"slippage,omitempty"`
}

type EquityPoint struct {
	Time           time.Time `json:"time"`
	Cash           float64   `json:"cash"`
	PositionsValue float64   `json:"positions_value"`
	Equity         float64   `json:"equity"`
}

type Event struct {
	Time   time.Time `json:"time,omitempty"`
	Layer  string    `json:"layer"`
	Type   string    `json:"type"`
	Symbol string    `json:"symbol,omitempty"`
	Side   Side      `json:"side,omitempty"`
	Reason string    `json:"reason"`
}

func (e Engine) Run(ctx context.Context, plan StrategyPlan) (result Result, runErr error) {
	errb := oops.In("backtest_engine").With("strategy", plan.StrategyName, "run", plan.RunName)
	stream, err := e.feed.Open(ctx, plan.DataRequest())
	if err != nil {
		return Result{}, errb.Wrapf(err, "open historical bar stream")
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			if runErr != nil {
				runErr = oops.Join(runErr, errb.Wrapf(closeErr, "close historical bar stream"))
				return
			}
			runErr = errb.Wrapf(closeErr, "close historical bar stream")
		}
	}()

	indicatorValues := make(map[string]map[string][]float64)
	indicatorRuntimes, err := newIndicatorRuntimes(plan)
	if err != nil {
		return Result{}, errb.Wrap(err)
	}

	portfolio := newPortfolio(plan.InitialCash)
	pending := make([]orderIntent, 0)
	curve := make([]EquityPoint, 0)
	riskEvents := make([]Event, 0)
	executionEvents := make([]Event, 0)
	sizer := percentOfEquitySizer{}
	risk := limitRiskManager{}
	execution := nextOpenExecutionModel{}
	series := make(map[string][]Bar)
	frameCount := 0

	for {
		if err := ctx.Err(); err != nil {
			return Result{}, errb.Wrap(err)
		}
		frame, ok, err := stream.Next(ctx)
		if err != nil {
			return Result{}, errb.Wrapf(err, "read historical bar frame")
		}
		if !ok {
			break
		}
		frameCount++
		currentTime := frame.Time
		currentBars := frame.Bars
		if len(currentBars) == 0 {
			continue
		}
		currentIndexes, err := appendFrameBars(plan, series, indicatorValues, indicatorRuntimes, frame)
		if err != nil {
			return Result{}, errb.Wrap(err)
		}
		activeSymbols := plan.activeSymbolsAt(currentTime)

		nextPending := pending[:0]
		for _, intent := range pending {
			bar, ok := currentBars[intent.Symbol]
			if !ok {
				executionEvents = append(executionEvents, Event{
					Time:   currentTime,
					Layer:  "execution",
					Type:   "deferred_no_bar",
					Symbol: intent.Symbol,
					Side:   intent.Side,
					Reason: "no_bar_for_pending_order",
				})
				nextPending = append(nextPending, intent)
				continue
			}
			fill, ok, event, err := execution.Execute(intent, bar, portfolio, plan)
			if err != nil {
				return Result{}, errb.Wrap(err)
			}
			if !ok {
				executionEvents = append(executionEvents, event)
				if isDeferredExecutionEvent(event) {
					nextPending = append(nextPending, intent)
				}
				continue
			}
			if err := portfolio.apply(fill.Trade); err != nil {
				return Result{}, errb.Wrap(err)
			}
		}
		pending = nextPending

		closePrices := closePrices(currentBars)
		nextIntents, err := evaluateSignals(plan, portfolio, activeSymbols, currentBars, currentIndexes, series, indicatorValues)
		if err != nil {
			return Result{}, errb.Wrap(err)
		}
		nextIntents = append(nextIntents, universeRemovalIntents(plan, portfolio, activeSymbols)...)
		sizedIntents := sizer.Size(nextIntents, portfolio, closePrices, plan)
		review := risk.Review(currentTime, sizedIntents, portfolio, closePrices, plan)
		riskEvents = append(riskEvents, review.Events...)
		pending = appendNewPendingOrders(pending, review.Approved)

		positionsValue := portfolio.positionsValue(closePrices)
		curve = append(curve, EquityPoint{
			Time:           currentTime,
			Cash:           portfolio.cash,
			PositionsValue: positionsValue,
			Equity:         portfolio.cash + positionsValue,
		})
	}
	if frameCount == 0 {
		return Result{}, errb.New("historical feed returned no bars")
	}

	for _, intent := range pending {
		executionEvents = append(executionEvents, Event{
			Layer:  "execution",
			Type:   "unfilled",
			Symbol: intent.Symbol,
			Side:   intent.Side,
			Reason: "no_next_bar_for_pending_order",
		})
	}

	result = Result{
		RunName:      plan.RunName,
		StrategyName: plan.StrategyName,
		Symbols:      plan.resultSymbols(),
		Instruments:  plan.resultInstruments(),
		Period: Period{
			From: plan.From,
			To:   plan.To,
		},
		Market:    plan.Market,
		Timeframe: plan.Timeframe,
		Currency:  plan.Currency,
		Execution: ExecutionAssumption{
			Fill:       plan.Fill,
			Commission: plan.Commission,
			Slippage:   plan.Slippage,
		},
		Benchmark:       plan.Benchmark,
		InitialCash:     plan.InitialCash,
		TradeCount:      len(portfolio.trades),
		SelectedMetrics: append([]string(nil), plan.SelectedMetrics...),
		Trades:          append([]Trade(nil), portfolio.trades...),
		EquityCurve:     curve,
		RiskEvents:      riskEvents,
		ExecutionEvents: executionEvents,
		Universe:        plan.universeExplainForResult(),
		UnfilledCount:   len(executionEvents),
		BenchmarkBars:   append([]Bar(nil), series[plan.Benchmark.Symbol]...),
	}
	if len(curve) > 0 {
		result.FinalEquity = curve[len(curve)-1].Equity
		result.TotalReturn = result.FinalEquity/plan.InitialCash - 1
		result.MaxDrawdown = maxDrawdown(curve)
	}
	result.RealizedPnL, result.WinRate, result.AverageTradeRet = tradeStats(result.Trades)
	result.Metrics, err = calculateSelectedMetrics(plan, result, series)
	if err != nil {
		return Result{}, errb.Wrap(err)
	}
	result.ResultHash = resultHash(result)
	return result, nil
}

func (p StrategyPlan) DataSymbols() []string {
	out := append([]string(nil), p.resultSymbols()...)
	if p.Benchmark.Symbol != "" && !slices.Contains(out, p.Benchmark.Symbol) {
		out = append(out, p.Benchmark.Symbol)
	}
	return out
}

func (p StrategyPlan) DataRequest() DataRequest {
	return DataRequest{
		Symbols:     p.DataSymbols(),
		Instruments: p.resultInstruments(),
		From:        p.From,
		To:          p.To,
		Benchmark:   p.Benchmark,
		WarmupBars:  maxIndicatorLookback(p),
	}
}

func (p StrategyPlan) resultInstruments() []InstrumentIdentity {
	seen := map[string]struct{}{}
	out := make([]InstrumentIdentity, 0, len(p.Instruments))
	for _, instrument := range p.Instruments {
		key := instrumentKey(instrument)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, instrument)
	}
	slices.SortFunc(out, func(a, b InstrumentIdentity) int {
		if a.Symbol < b.Symbol {
			return -1
		}
		if a.Symbol > b.Symbol {
			return 1
		}
		if a.Market < b.Market {
			return -1
		}
		if a.Market > b.Market {
			return 1
		}
		if a.SecurityType < b.SecurityType {
			return -1
		}
		if a.SecurityType > b.SecurityType {
			return 1
		}
		return 0
	})
	return out
}

func maxIndicatorLookback(plan StrategyPlan) int {
	maximum := 0
	for _, spec := range collectIndicatorKeys(plan) {
		window := int(spec.Params["window"])
		if spec.ID == "rsi" {
			window++
		}
		if window > maximum {
			maximum = window
		}
	}
	return maximum
}

func instrumentKey(instrument InstrumentIdentity) string {
	return instrument.Market + "\x00" + instrument.SecurityType + "\x00" + instrument.Symbol
}

func evaluateSignals(plan StrategyPlan, portfolio portfolio, activeSymbols []string, currentBars map[string]Bar, currentIndexes map[string]int, series map[string][]Bar, indicators map[string]map[string][]float64) ([]orderIntent, error) {
	intents := make([]orderIntent, 0)
	for _, symbol := range activeSymbols {
		if _, ok := currentBars[symbol]; !ok {
			continue
		}
		ctx := ruleContext{
			symbol:     symbol,
			index:      currentIndexes[symbol],
			bars:       series[symbol],
			indicators: indicators,
			plan:       plan,
		}
		if portfolio.hasPosition(symbol) {
			matched, err := evaluateRule(plan.Exit, ctx)
			if err != nil {
				return nil, err
			}
			if matched {
				intents = append(intents, orderIntent{Symbol: symbol, Side: SideSell, Reason: "exit"})
			}
			continue
		}
		matched, err := evaluateRule(plan.Entry, ctx)
		if err != nil {
			return nil, err
		}
		if matched {
			intents = append(intents, orderIntent{Symbol: symbol, Side: SideBuy, Reason: "entry"})
		}
	}
	return intents, nil
}

func newIndicatorRuntimes(plan StrategyPlan) (map[string]map[string]indicatorRuntime, error) {
	out := make(map[string]map[string]indicatorRuntime)
	for key, spec := range collectIndicatorKeys(plan) {
		if _, ok := plan.registry.Definition(spec.ID); !ok {
			return nil, oops.In("backtest_indicators").With("indicator", spec.ID).New("indicator is not registered")
		}
		out[key] = map[string]indicatorRuntime{}
	}
	return out, nil
}

func appendFrameBars(plan StrategyPlan, series map[string][]Bar, indicators map[string]map[string][]float64, runtimes map[string]map[string]indicatorRuntime, frame BarFrame) (map[string]int, error) {
	indexes := make(map[string]int, len(frame.Bars))
	for symbol, bar := range frame.Bars {
		if err := bar.validate(); err != nil {
			return nil, err
		}
		series[symbol] = append(series[symbol], bar)
		index := len(series[symbol]) - 1
		indexes[symbol] = index
		for key, spec := range collectIndicatorKeys(plan) {
			if indicators[key] == nil {
				indicators[key] = map[string][]float64{}
			}
			runtime := runtimes[key][symbol]
			if runtime == nil {
				next, err := newIndicatorRuntime(spec)
				if err != nil {
					return nil, oops.In("backtest_indicators").With("indicator", spec.ID, "symbol", symbol).Wrap(err)
				}
				runtime = next
				runtimes[key][symbol] = runtime
			}
			value, err := runtime.Add(bar)
			if err != nil {
				return nil, oops.In("backtest_indicators").With("indicator", spec.ID, "symbol", symbol).Wrap(err)
			}
			indicators[key][symbol] = append(indicators[key][symbol], value)
		}
	}
	return indexes, nil
}

func universeRemovalIntents(plan StrategyPlan, p portfolio, activeSymbols []string) []orderIntent {
	if plan.Universe.PositionPolicy != UniversePositionPolicyLiquidate {
		return nil
	}
	active := make(map[string]struct{}, len(activeSymbols))
	for _, symbol := range activeSymbols {
		active[symbol] = struct{}{}
	}
	out := make([]orderIntent, 0)
	for symbol, position := range p.positions {
		if position.Quantity <= 0 {
			continue
		}
		if _, ok := active[symbol]; ok {
			continue
		}
		out = append(out, orderIntent{Symbol: symbol, Side: SideSell, Reason: "universe_removed"})
	}
	return out
}

func (p StrategyPlan) activeSymbolsAt(currentTime time.Time) []string {
	if len(p.UniverseExplain.Snapshots) == 0 {
		return append([]string(nil), p.Symbols...)
	}
	var active []string
	for _, snapshot := range p.UniverseExplain.Snapshots {
		if snapshot.Time.After(currentTime) {
			continue
		}
		active = snapshot.Symbols
	}
	return append([]string(nil), active...)
}

func (p StrategyPlan) resultSymbols() []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(p.Symbols))
	for _, symbol := range p.Symbols {
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	for _, snapshot := range p.UniverseExplain.Snapshots {
		for _, symbol := range snapshot.Symbols {
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			out = append(out, symbol)
		}
	}
	slices.Sort(out)
	return out
}

func (p StrategyPlan) universeExplainForResult() UniverseExplain {
	explain := p.UniverseExplain
	if explain.Mode == "" {
		explain.Mode = p.Universe.Mode
	}
	if explain.Schedule == "" {
		explain.Schedule = p.Universe.Schedule.Frequency
	}
	if explain.PositionPolicy == "" {
		explain.PositionPolicy = p.Universe.PositionPolicy
	}
	if len(explain.SelectedSymbols) == 0 {
		explain.SelectedSymbols = p.resultSymbols()
	}
	return explain
}

func calculateIndicators(plan StrategyPlan, series map[string][]Bar) (map[string]map[string][]float64, error) {
	out := make(map[string]map[string][]float64)
	for key, spec := range collectIndicatorKeys(plan) {
		definition, ok := plan.registry.Definition(spec.ID)
		if !ok {
			return nil, oops.In("backtest_indicators").With("indicator", spec.ID).New("indicator is not registered")
		}
		out[key] = make(map[string][]float64, len(series))
		for symbol, bars := range series {
			values, err := definition.Calculate(spec, bars)
			if err != nil {
				return nil, oops.In("backtest_indicators").With("indicator", spec.ID, "symbol", symbol).Wrap(err)
			}
			out[key][symbol] = values
		}
	}
	return out, nil
}

func groupBars(bars []Bar) map[string][]Bar {
	out := make(map[string][]Bar)
	for _, bar := range bars {
		out[bar.Symbol] = append(out[bar.Symbol], bar)
	}
	for symbol := range out {
		sortBars(out[symbol])
	}
	return out
}

func orderedTimes(bars []Bar) []time.Time {
	seen := make(map[time.Time]struct{})
	out := make([]time.Time, 0)
	for _, bar := range bars {
		if _, ok := seen[bar.Time]; ok {
			continue
		}
		seen[bar.Time] = struct{}{}
		out = append(out, bar.Time)
	}
	slices.SortFunc(out, func(a, b time.Time) int {
		if a.Before(b) {
			return -1
		}
		if a.After(b) {
			return 1
		}
		return 0
	})
	return out
}

func barsAt(series map[string][]Bar, current time.Time) (map[string]Bar, map[string]int) {
	bars := make(map[string]Bar)
	indexes := make(map[string]int)
	for symbol, symbolBars := range series {
		for index, bar := range symbolBars {
			if bar.Time.Equal(current) {
				bars[symbol] = bar
				indexes[symbol] = index
				break
			}
		}
	}
	return bars, indexes
}

func closePrices(bars map[string]Bar) map[string]float64 {
	out := make(map[string]float64, len(bars))
	for symbol, bar := range bars {
		out[symbol] = bar.Close
	}
	return out
}

func isDeferredExecutionEvent(event Event) bool {
	switch event.Type {
	case "deferred_no_bar", "deferred_no_trade_bar":
		return true
	default:
		return false
	}
}

func appendNewPendingOrders(pending []orderIntent, intents []orderIntent) []orderIntent {
	seen := make(map[orderKey]struct{}, len(pending))
	for _, intent := range pending {
		seen[newOrderKey(intent)] = struct{}{}
	}
	for _, intent := range intents {
		key := newOrderKey(intent)
		if _, ok := seen[key]; ok {
			continue
		}
		pending = append(pending, intent)
		seen[key] = struct{}{}
	}
	return pending
}

type orderKey struct {
	symbol string
	side   Side
}

func newOrderKey(intent orderIntent) orderKey {
	return orderKey{symbol: intent.Symbol, side: intent.Side}
}

func maxDrawdown(curve []EquityPoint) float64 {
	var peak float64
	var maxDD float64
	for _, point := range curve {
		if point.Equity > peak {
			peak = point.Equity
		}
		if peak <= 0 {
			continue
		}
		drawdown := point.Equity/peak - 1
		if drawdown < maxDD {
			maxDD = drawdown
		}
	}
	return maxDD
}

func tradeStats(trades []Trade) (realizedPnL float64, winRate float64, averageReturn float64) {
	closed := 0
	winners := 0
	var returns float64
	for _, trade := range trades {
		if trade.Side != SideSell {
			continue
		}
		closed++
		realizedPnL += trade.RealizedPnL
		returns += trade.Return
		if trade.RealizedPnL > 0 {
			winners++
		}
	}
	if closed == 0 {
		return realizedPnL, 0, 0
	}
	return realizedPnL, float64(winners) / float64(closed), returns / float64(closed)
}

func resultHash(result Result) string {
	result.ResultHash = ""
	payload, _ := json.Marshal(result)
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
