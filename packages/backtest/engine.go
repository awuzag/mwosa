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
	feed Feed
}

func NewEngine(feed Feed) (Engine, error) {
	if feed == nil {
		return Engine{}, oops.In("backtest_engine").New("feed is nil")
	}
	return Engine{feed: feed}, nil
}

type Result struct {
	RunName         string              `json:"run_name"`
	StrategyName    string              `json:"strategy_name"`
	Symbols         []string            `json:"symbols"`
	Period          Period              `json:"period"`
	Market          string              `json:"market"`
	SecurityType    string              `json:"security_type"`
	Timeframe       string              `json:"timeframe"`
	Currency        string              `json:"currency"`
	Execution       ExecutionAssumption `json:"execution"`
	InitialCash     float64             `json:"initial_cash"`
	FinalEquity     float64             `json:"final_equity"`
	TotalReturn     float64             `json:"total_return"`
	MaxDrawdown     float64             `json:"max_drawdown"`
	TradeCount      int                 `json:"trade_count"`
	WinRate         float64             `json:"win_rate"`
	AverageTradeRet float64             `json:"average_trade_return"`
	RealizedPnL     float64             `json:"realized_pnl"`
	Metrics         Metrics             `json:"metrics"`
	Trades          []Trade             `json:"trades"`
	EquityCurve     []EquityPoint       `json:"equity_curve"`
	RiskEvents      []Event             `json:"risk_events,omitempty"`
	ExecutionEvents []Event             `json:"execution_events,omitempty"`
	UnfilledCount   int                 `json:"unfilled_count,omitempty"`
	ResultHash      string              `json:"result_hash"`
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

type Metrics struct {
	TotalReturn        float64 `json:"total_return"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	TradeCount         int     `json:"trade_count"`
	WinRate            float64 `json:"win_rate"`
	AverageTradeReturn float64 `json:"average_trade_return"`
	RealizedPnL        float64 `json:"realized_pnl"`
	UnfilledCount      int     `json:"unfilled_count"`
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

func (e Engine) Run(ctx context.Context, plan StrategyPlan) (Result, error) {
	errb := oops.In("backtest_engine").With("strategy", plan.StrategyName, "run", plan.RunName)
	bars, err := e.feed.Bars(ctx, DataRequest{Symbols: plan.Symbols, From: plan.From, To: plan.To})
	if err != nil {
		return Result{}, errb.Wrapf(err, "load historical bars")
	}
	if len(bars) == 0 {
		return Result{}, errb.New("historical feed returned no bars")
	}
	if err := validateBars(bars); err != nil {
		return Result{}, errb.Wrap(err)
	}

	series := groupBars(bars)
	indicatorValues, err := calculateIndicators(plan, series)
	if err != nil {
		return Result{}, errb.Wrap(err)
	}

	times := orderedTimes(bars)
	portfolio := newPortfolio(plan.InitialCash)
	pending := make([]orderIntent, 0)
	curve := make([]EquityPoint, 0, len(times))
	riskEvents := make([]Event, 0)
	executionEvents := make([]Event, 0)
	sizer := percentOfEquitySizer{}
	risk := limitRiskManager{}
	execution := nextOpenExecutionModel{}

	for _, currentTime := range times {
		if err := ctx.Err(); err != nil {
			return Result{}, errb.Wrap(err)
		}
		currentBars, currentIndexes := barsAt(series, currentTime)
		if len(currentBars) == 0 {
			continue
		}

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
		nextIntents, err := evaluateSignals(plan, portfolio, currentBars, currentIndexes, series, indicatorValues)
		if err != nil {
			return Result{}, errb.Wrap(err)
		}
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

	for _, intent := range pending {
		executionEvents = append(executionEvents, Event{
			Layer:  "execution",
			Type:   "unfilled",
			Symbol: intent.Symbol,
			Side:   intent.Side,
			Reason: "no_next_bar_for_pending_order",
		})
	}

	result := Result{
		RunName:      plan.RunName,
		StrategyName: plan.StrategyName,
		Symbols:      append([]string(nil), plan.Symbols...),
		Period: Period{
			From: plan.From,
			To:   plan.To,
		},
		Market:       plan.Market,
		SecurityType: plan.SecurityType,
		Timeframe:    plan.Timeframe,
		Currency:     plan.Currency,
		Execution: ExecutionAssumption{
			Fill:       plan.Fill,
			Commission: plan.Commission,
			Slippage:   plan.Slippage,
		},
		InitialCash:     plan.InitialCash,
		TradeCount:      len(portfolio.trades),
		Trades:          append([]Trade(nil), portfolio.trades...),
		EquityCurve:     curve,
		RiskEvents:      riskEvents,
		ExecutionEvents: executionEvents,
		UnfilledCount:   len(executionEvents),
	}
	if len(curve) > 0 {
		result.FinalEquity = curve[len(curve)-1].Equity
		result.TotalReturn = result.FinalEquity/plan.InitialCash - 1
		result.MaxDrawdown = maxDrawdown(curve)
	}
	result.RealizedPnL, result.WinRate, result.AverageTradeRet = tradeStats(result.Trades)
	result.Metrics = Metrics{
		TotalReturn:        result.TotalReturn,
		MaxDrawdown:        result.MaxDrawdown,
		TradeCount:         result.TradeCount,
		WinRate:            result.WinRate,
		AverageTradeReturn: result.AverageTradeRet,
		RealizedPnL:        result.RealizedPnL,
		UnfilledCount:      result.UnfilledCount,
	}
	result.ResultHash = resultHash(result)
	return result, nil
}

func evaluateSignals(plan StrategyPlan, portfolio portfolio, currentBars map[string]Bar, currentIndexes map[string]int, series map[string][]Bar, indicators map[string]map[string][]float64) ([]orderIntent, error) {
	intents := make([]orderIntent, 0)
	for _, symbol := range plan.Symbols {
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
