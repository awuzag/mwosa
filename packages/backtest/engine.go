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

const EngineVersion = "backtest-engine/v1"

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
	Timeframes      TimeframeMetadata    `json:"timeframes"`
	Runtime         RuntimeMetadata      `json:"runtime"`
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
	Positions       []PositionSnapshot   `json:"positions,omitempty"`
	EquityCurve     []EquityPoint        `json:"equity_curve"`
	DataEvents      []Event              `json:"data_events,omitempty"`
	RiskEvents      []Event              `json:"risk_events,omitempty"`
	ExecutionEvents []Event              `json:"execution_events,omitempty"`
	Universe        UniverseExplain      `json:"universe"`
	UnfilledCount   int                  `json:"-"`
	DataFingerprint string               `json:"data_fingerprint"`
	ResultHash      string               `json:"result_hash"`
	BenchmarkBars   []Bar                `json:"-"`
}

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type TimeframeMetadata struct {
	Requested string                 `json:"requested"`
	Source    string                 `json:"source"`
	Execution string                 `json:"execution"`
	Resample  ResamplePolicyMetadata `json:"resample"`
	Warmup    WarmupPolicyMetadata   `json:"warmup"`
}

type ResamplePolicyMetadata struct {
	Enabled  bool   `json:"enabled"`
	Source   string `json:"source,omitempty"`
	Target   string `json:"target,omitempty"`
	Policy   string `json:"policy,omitempty"`
	Boundary string `json:"boundary,omitempty"`
}

type WarmupPolicyMetadata struct {
	Bars   int    `json:"bars"`
	Policy string `json:"policy"`
}

type RuntimeMetadata struct {
	EngineVersion            string `json:"engine_version"`
	IndicatorRegistryVersion string `json:"indicator_registry_version"`
	MetricRegistryVersion    string `json:"metric_registry_version"`
}

type ExecutionAssumption struct {
	Fill                    string          `json:"fill"`
	OrderType               string          `json:"order_type,omitempty"`
	LimitPrice              float64         `json:"limit_price,omitempty"`
	StopPrice               float64         `json:"stop_price,omitempty"`
	TrailingStopPct         float64         `json:"trailing_stop_pct,omitempty"`
	IntrabarAmbiguityPolicy string          `json:"intrabar_ambiguity_policy,omitempty"`
	TimeInForce             string          `json:"time_in_force,omitempty"`
	LotSize                 float64         `json:"lot_size,omitempty"`
	TickSize                float64         `json:"tick_size,omitempty"`
	Commission              CostSpec        `json:"commission,omitempty"`
	Tax                     CostSpec        `json:"tax,omitempty"`
	ExchangeFee             CostSpec        `json:"exchange_fee,omitempty"`
	Slippage                CostSpec        `json:"slippage,omitempty"`
	Liquidity               LiquiditySpec   `json:"liquidity,omitempty"`
	PartialFill             PartialFillSpec `json:"partial_fill,omitempty"`
}

type EquityPoint struct {
	Time           time.Time `json:"time"`
	Cash           float64   `json:"cash"`
	PositionsValue float64   `json:"positions_value"`
	Equity         float64   `json:"equity"`
}

type PositionSnapshot struct {
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	AvgPrice    float64   `json:"avg_price"`
	PeakPrice   float64   `json:"peak_price,omitempty"`
	MarketPrice float64   `json:"market_price"`
	MarketValue float64   `json:"market_value"`
	WeightPct   float64   `json:"weight_pct"`
	EntryTime   time.Time `json:"entry_time,omitempty"`
}

type Event struct {
	Time        time.Time `json:"time,omitempty"`
	Layer       string    `json:"layer"`
	Type        string    `json:"type"`
	Symbol      string    `json:"symbol,omitempty"`
	Side        Side      `json:"side,omitempty"`
	Reason      string    `json:"reason"`
	Timeframe   string    `json:"timeframe,omitempty"`
	Session     string    `json:"session,omitempty"`
	Status      string    `json:"status,omitempty"`
	Amount      float64   `json:"amount,omitempty"`
	Quantity    float64   `json:"quantity,omitempty"`
	Price       float64   `json:"price,omitempty"`
	Notional    float64   `json:"notional,omitempty"`
	Commission  float64   `json:"commission,omitempty"`
	Tax         float64   `json:"tax,omitempty"`
	ExchangeFee float64   `json:"exchange_fee,omitempty"`
	TotalCost   float64   `json:"total_cost,omitempty"`
	SlippageBps float64   `json:"slippage_bps,omitempty"`
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
	dataEvents := make([]Event, 0)
	riskEvents := make([]Event, 0)
	executionEvents := make([]Event, 0)
	latestPrices := map[string]float64{}
	sizer := percentOfEquitySizer{}
	risk := limitRiskManager{}
	execution := barExecutionModel{}
	series := make(map[string][]Bar)
	frameCount := 0
	var lastFrameTime time.Time

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
		if frame.Time.IsZero() {
			return Result{}, errb.New("historical bar frame time is required")
		}
		if !lastFrameTime.IsZero() && !frame.Time.After(lastFrameTime) {
			return Result{}, errb.
				With("previous_time", lastFrameTime.Format(time.RFC3339), "current_time", frame.Time.Format(time.RFC3339)).
				New("historical bar frame time must be strictly increasing")
		}
		lastFrameTime = frame.Time
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
				event := Event{
					Time:   currentTime,
					Layer:  "execution",
					Type:   "deferred_no_bar",
					Symbol: intent.Symbol,
					Side:   intent.Side,
					Reason: "no_bar_for_pending_order",
				}
				executionEvents = append(executionEvents, event)
				dataEvents = append(dataEvents, dataIssueEventsFromExecutionEvents([]Event{event})...)
				nextPending = append(nextPending, intent)
				continue
			}
			outcome, err := execution.Execute(intent, bar, portfolio, plan)
			if err != nil {
				return Result{}, errb.Wrap(err)
			}
			executionEvents = append(executionEvents, outcome.Events...)
			dataEvents = append(dataEvents, dataIssueEventsFromExecutionEvents(outcome.Events)...)
			if outcome.Trade == nil {
				if shouldCarryPendingOrder(plan, outcome) {
					nextPending = append(nextPending, intent)
				}
				continue
			}
			if err := portfolio.apply(*outcome.Trade); err != nil {
				return Result{}, errb.Wrap(err)
			}
			executionEvents = append(executionEvents, portfolioMutationEvent(outcome.Trade))
			if outcome.Remaining != nil {
				nextPending = append(nextPending, *outcome.Remaining)
			}
		}
		pending = nextPending

		closePrices := closePrices(currentBars)
		latestPrices = closePrices
		nextIntents, err := evaluateSignals(plan, portfolio, activeSymbols, currentBars, currentIndexes, series, indicatorValues, curve)
		if err != nil {
			return Result{}, errb.Wrap(err)
		}
		nextIntents = append(nextIntents, universeRemovalIntents(plan, portfolio, activeSymbols)...)
		nextIntents = attachSlippageReferences(plan, nextIntents, currentIndexes, indicatorValues)
		sizedIntents := sizer.Size(nextIntents, portfolio, closePrices, plan)
		review := risk.Review(currentTime, sizedIntents, portfolio, closePrices, plan)
		riskEvents = append(riskEvents, review.Events...)
		var cancelEvents []Event
		pending, cancelEvents = cancelPendingOnRebalance(currentTime, pending, review.Approved, closePrices, plan)
		executionEvents = append(executionEvents, cancelEvents...)
		if plan.Fill == FillSameClose {
			executionEvents = append(executionEvents, orderIntentEvents(currentTime, review.Approved)...)
			for _, intent := range review.Approved {
				bar, ok := currentBars[intent.Symbol]
				if !ok {
					event := unfilledEvent(currentTime, intent, "no_bar_for_same_close_order")
					executionEvents = append(executionEvents, event)
					dataEvents = append(dataEvents, Event{
						Time:   currentTime,
						Layer:  "data",
						Type:   "data_issue",
						Symbol: intent.Symbol,
						Side:   intent.Side,
						Reason: "missing_bar_for_same_close_order",
					})
					continue
				}
				outcome, err := execution.Execute(intent, bar, portfolio, plan)
				if err != nil {
					return Result{}, errb.Wrap(err)
				}
				executionEvents = append(executionEvents, outcome.Events...)
				dataEvents = append(dataEvents, dataIssueEventsFromExecutionEvents(outcome.Events)...)
				if outcome.Trade == nil {
					continue
				}
				if err := portfolio.apply(*outcome.Trade); err != nil {
					return Result{}, errb.Wrap(err)
				}
				executionEvents = append(executionEvents, portfolioMutationEvent(outcome.Trade))
				if outcome.Remaining != nil {
					pending = append(pending, *outcome.Remaining)
				}
			}
		} else {
			var accepted []orderIntent
			pending, accepted = appendNewPendingOrders(pending, review.Approved)
			executionEvents = append(executionEvents, orderIntentEvents(currentTime, accepted)...)
		}

		positionsValue := portfolio.positionsValue(closePrices)
		curve = append(curve, EquityPoint{
			Time:           currentTime,
			Cash:           portfolio.cash,
			PositionsValue: positionsValue,
			Equity:         portfolio.cash + positionsValue,
		})
		portfolio.updateTrailingPeaks(currentTime, currentBars, plan.Fill)
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
		Market:     plan.Market,
		Timeframe:  plan.Timeframe,
		Timeframes: plan.timeframeMetadata(),
		Runtime:    plan.runtimeMetadata(),
		Currency:   plan.Currency,
		Execution: ExecutionAssumption{
			Fill:                    plan.Fill,
			OrderType:               plan.OrderType,
			LimitPrice:              plan.LimitPrice,
			StopPrice:               plan.StopPrice,
			TrailingStopPct:         plan.TrailingStopPct,
			IntrabarAmbiguityPolicy: plan.IntrabarAmbiguityPolicy,
			TimeInForce:             plan.TimeInForce,
			LotSize:                 plan.LotSize,
			TickSize:                plan.TickSize,
			Commission:              plan.Commission,
			Tax:                     plan.Tax,
			ExchangeFee:             plan.ExchangeFee,
			Slippage:                plan.Slippage,
			Liquidity:               plan.Liquidity,
			PartialFill:             plan.PartialFill,
		},
		Benchmark:       plan.Benchmark,
		InitialCash:     plan.InitialCash,
		TradeCount:      len(portfolio.trades),
		SelectedMetrics: append([]string(nil), plan.SelectedMetrics...),
		Trades:          append([]Trade(nil), portfolio.trades...),
		Positions:       portfolio.snapshots(latestPrices),
		EquityCurve:     curve,
		DataEvents:      dataEvents,
		RiskEvents:      riskEvents,
		ExecutionEvents: executionEvents,
		Universe:        plan.universeExplainForResult(),
		UnfilledCount:   executionIssueCount(executionEvents),
		DataFingerprint: dataFingerprint(series),
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
		Timeframe:   p.Timeframe,
		Benchmark:   p.Benchmark,
		WarmupBars:  maxIndicatorLookback(p),
	}
}

func (p StrategyPlan) TimeframeMetadata() TimeframeMetadata {
	return p.timeframeMetadata()
}

func (p StrategyPlan) RuntimeMetadata() RuntimeMetadata {
	return p.runtimeMetadata()
}

func (p StrategyPlan) runtimeMetadata() RuntimeMetadata {
	return RuntimeMetadata{
		EngineVersion:            EngineVersion,
		IndicatorRegistryVersion: p.registry.Version(),
		MetricRegistryVersion:    p.metricRegistry.Version(),
	}
}

func (p StrategyPlan) timeframeMetadata() TimeframeMetadata {
	timeframe := p.Timeframe
	if timeframe == "" {
		timeframe = Timeframe1Day
	}
	parsed, err := ParseTimeframe(timeframe)
	if err != nil {
		return TimeframeMetadata{
			Requested: timeframe,
			Source:    timeframe,
			Execution: timeframe,
			Warmup: WarmupPolicyMetadata{
				Bars:   maxIndicatorLookback(p),
				Policy: "indicator_lookback_bars",
			},
		}
	}
	source := parsed.ID
	resample := ResamplePolicyMetadata{Enabled: false}
	if parsed.IsDailyResample() {
		source = Timeframe1Day
		resample = ResamplePolicyMetadata{
			Enabled:  true,
			Source:   Timeframe1Day,
			Target:   parsed.ID,
			Policy:   "ohlcv_last_close_sum_volume",
			Boundary: resampleBoundary(parsed),
		}
	}
	return TimeframeMetadata{
		Requested: parsed.ID,
		Source:    source,
		Execution: parsed.ID,
		Resample:  resample,
		Warmup: WarmupPolicyMetadata{
			Bars:   maxIndicatorLookback(p),
			Policy: "indicator_lookback_bars",
		},
	}
}

func resampleBoundary(timeframe Timeframe) string {
	switch timeframe.ID {
	case Timeframe1Week:
		return "iso_week"
	case Timeframe1Month:
		return "calendar_month"
	default:
		return ""
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
		if spec.ID == "macd" {
			window = int(spec.Params["slow_window"]) + int(spec.Params["signal_window"])
		}
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

func evaluateSignals(plan StrategyPlan, portfolio portfolio, activeSymbols []string, currentBars map[string]Bar, currentIndexes map[string]int, series map[string][]Bar, indicators map[string]map[string][]float64, equity []EquityPoint) ([]orderIntent, error) {
	intents := make([]orderIntent, 0)
	prices := closePrices(currentBars)
	for _, symbol := range activeSymbols {
		if _, ok := currentBars[symbol]; !ok {
			continue
		}
		ctx := ruleContext{
			symbol:         symbol,
			index:          currentIndexes[symbol],
			bars:           series[symbol],
			series:         series,
			activeSymbols:  activeSymbols,
			currentBars:    currentBars,
			currentIndexes: currentIndexes,
			indicators:     indicators,
			plan:           plan,
			portfolio:      portfolio,
			prices:         prices,
			equity:         equity,
		}
		if portfolio.hasPosition(symbol) {
			rule, matched, err := firstMatchingRule(plan.Stops, ctx)
			if err != nil {
				return nil, err
			}
			if matched {
				intents = append(intents, orderIntent{Symbol: symbol, Side: SideSell, Reason: exitReason(rule)})
				continue
			}
			rule, matched, err = firstMatchingRule(plan.Exits, ctx)
			if err != nil {
				return nil, err
			}
			if matched {
				intents = append(intents, orderIntent{Symbol: symbol, Side: SideSell, Reason: exitReason(rule)})
				continue
			}
			if plan.OrderType == OrderTypeRebalance {
				rules := plan.Rebalance
				if len(rules) == 0 {
					rules = plan.Entries
				}
				_, matched, err := firstMatchingRule(rules, ctx)
				if err != nil {
					return nil, err
				}
				if matched {
					intents = append(intents, orderIntent{Symbol: symbol, Side: SideBuy, Reason: "rebalance"})
				}
			}
			continue
		}
		_, matched, err := firstMatchingRule(plan.Entries, ctx)
		if err != nil {
			return nil, err
		}
		if matched {
			intents = append(intents, orderIntent{Symbol: symbol, Side: SideBuy, Reason: "entry"})
		}
	}
	return intents, nil
}

func firstMatchingRule(rules []RuleExpr, ctx ruleContext) (RuleExpr, bool, error) {
	for _, rule := range rules {
		matched, err := evaluateRule(rule, ctx)
		if err != nil {
			return RuleExpr{}, false, err
		}
		if matched {
			return rule, true, nil
		}
	}
	return RuleExpr{}, false, nil
}

func exitReason(rule RuleExpr) string {
	switch rule.Operator {
	case "stop_loss", "take_profit", "time_stop", "trailing_stop", "volatility_stop":
		return rule.Operator
	default:
		return "exit"
	}
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

func dataIssueEventsFromExecutionEvents(events []Event) []Event {
	out := make([]Event, 0)
	for _, event := range events {
		reason := ""
		switch event.Type {
		case "deferred_no_bar":
			reason = "missing_bar"
		case "deferred_no_trade_bar":
			reason = "no_trade_bar"
		default:
			continue
		}
		out = append(out, Event{
			Time:      event.Time,
			Layer:     "data",
			Type:      "data_issue",
			Symbol:    event.Symbol,
			Side:      event.Side,
			Reason:    reason,
			Timeframe: event.Timeframe,
			Session:   event.Session,
			Status:    event.Status,
			Price:     event.Price,
		})
	}
	return out
}

func shouldCarryPendingOrder(plan StrategyPlan, outcome executionOutcome) bool {
	if outcome.Remaining != nil {
		return true
	}
	if len(outcome.Events) == 0 {
		return false
	}
	lastEvent := outcome.Events[len(outcome.Events)-1]
	if isDeferredExecutionEvent(lastEvent) {
		return true
	}
	if lastEvent.Type == "unfilled" && lastEvent.Reason == "slippage_atr_not_ready" {
		return false
	}
	return (plan.TimeInForce == TimeInForceGTC || plan.TimeInForce == TimeInForceCancelOnRebalance) &&
		lastEvent.Type == "unfilled"
}

func attachSlippageReferences(plan StrategyPlan, intents []orderIntent, currentIndexes map[string]int, indicators map[string]map[string][]float64) []orderIntent {
	if !isATRSlippage(plan.Slippage) || len(intents) == 0 {
		return intents
	}
	seriesBySymbol := indicators[indicatorKey(slippageATRIndicatorSpec(plan.Slippage))]
	for i := range intents {
		index, ok := currentIndexes[intents[i].Symbol]
		if !ok {
			continue
		}
		value, ok := finiteSeriesValue(seriesBySymbol[intents[i].Symbol], index)
		if !ok {
			continue
		}
		intents[i].SlippageReference = value
		intents[i].SlippageReferenceSet = true
	}
	return intents
}

func appendNewPendingOrders(pending []orderIntent, intents []orderIntent) ([]orderIntent, []orderIntent) {
	seen := make(map[orderKey]struct{}, len(pending))
	for _, intent := range pending {
		seen[newOrderKey(intent)] = struct{}{}
	}
	accepted := make([]orderIntent, 0, len(intents))
	for _, intent := range intents {
		key := newOrderKey(intent)
		if _, ok := seen[key]; ok {
			continue
		}
		pending = append(pending, intent)
		accepted = append(accepted, intent)
		seen[key] = struct{}{}
	}
	return pending, accepted
}

func cancelPendingOnRebalance(currentTime time.Time, pending []orderIntent, intents []orderIntent, prices map[string]float64, plan StrategyPlan) ([]orderIntent, []Event) {
	if plan.TimeInForce != TimeInForceCancelOnRebalance || len(pending) == 0 {
		return pending, nil
	}
	rebalanceSymbols := make(map[string]struct{})
	for _, intent := range intents {
		if intent.Reason == "rebalance" {
			rebalanceSymbols[intent.Symbol] = struct{}{}
		}
	}
	if len(rebalanceSymbols) == 0 {
		return pending, nil
	}

	kept := pending[:0]
	events := make([]Event, 0, len(pending))
	for _, intent := range pending {
		if _, ok := rebalanceSymbols[intent.Symbol]; !ok {
			kept = append(kept, intent)
			continue
		}
		events = append(events, pendingOrderCancelledEvent(currentTime, intent, prices, "cancel_on_rebalance"))
	}
	return kept, events
}

func pendingOrderCancelledEvent(currentTime time.Time, intent orderIntent, prices map[string]float64, reason string) Event {
	amount := intent.Amount
	if amount <= 0 && intent.Quantity > 0 {
		amount = intent.Quantity * prices[intent.Symbol]
	}
	return Event{
		Time:     currentTime,
		Layer:    "execution",
		Type:     "order_cancelled",
		Symbol:   intent.Symbol,
		Side:     intent.Side,
		Reason:   reason,
		Quantity: intent.Quantity,
		Amount:   amount,
	}
}

type orderKey struct {
	symbol string
	side   Side
}

func newOrderKey(intent orderIntent) orderKey {
	return orderKey{symbol: intent.Symbol, side: intent.Side}
}

func orderIntentEvents(currentTime time.Time, intents []orderIntent) []Event {
	events := make([]Event, 0, len(intents))
	for _, intent := range intents {
		events = append(events, Event{
			Time:     currentTime,
			Layer:    "execution",
			Type:     "order_intent",
			Symbol:   intent.Symbol,
			Side:     intent.Side,
			Reason:   intent.Reason,
			Amount:   intent.Amount,
			Quantity: intent.Quantity,
		})
	}
	return events
}

func portfolioMutationEvent(trade *Trade) Event {
	return Event{
		Time:        trade.Time,
		Layer:       "portfolio",
		Type:        "portfolio_mutation",
		Symbol:      trade.Symbol,
		Side:        trade.Side,
		Reason:      trade.Reason,
		Quantity:    trade.Quantity,
		Price:       trade.Price,
		Notional:    trade.Notional,
		Commission:  trade.Commission,
		Tax:         trade.Tax,
		ExchangeFee: trade.ExchangeFee,
		TotalCost:   trade.totalCost(),
		SlippageBps: trade.SlippageBps,
	}
}

func executionIssueCount(events []Event) int {
	count := 0
	for _, event := range events {
		if event.Type == "unfilled" || isDeferredExecutionEvent(event) {
			count++
		}
	}
	return count
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

func dataFingerprint(series map[string][]Bar) string {
	payload, _ := json.Marshal(series)
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
