package backtest

import (
	"math"
	"slices"
	"time"

	"github.com/samber/oops"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Position struct {
	Symbol    string    `json:"symbol"`
	Quantity  float64   `json:"quantity"`
	AvgPrice  float64   `json:"avg_price"`
	EntryTime time.Time `json:"entry_time,omitempty"`
	PeakPrice float64   `json:"peak_price,omitempty"`
}

type Trade struct {
	Time        time.Time `json:"time"`
	Symbol      string    `json:"symbol"`
	Side        Side      `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Notional    float64   `json:"notional"`
	Commission  float64   `json:"commission"`
	Tax         float64   `json:"tax,omitempty"`
	ExchangeFee float64   `json:"exchange_fee,omitempty"`
	TotalCost   float64   `json:"total_cost,omitempty"`
	SlippageBps float64   `json:"slippage_bps,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	RealizedPnL float64   `json:"realized_pnl,omitempty"`
	Return      float64   `json:"return,omitempty"`
}

type orderIntent struct {
	Symbol               string
	Side                 Side
	Amount               float64
	Quantity             float64
	Reason               string
	CarryBars            int
	SlippageReference    float64
	SlippageReferenceSet bool
}

type fill struct {
	Trade Trade
}

type executionOutcome struct {
	Trade     *Trade
	Remaining *orderIntent
	Events    []Event
}

type executionPriceDecision struct {
	Price         float64
	BlockedReason string
	Events        []Event
}

type riskReview struct {
	Approved []orderIntent
	Events   []Event
}

type positionSizer interface {
	Size([]orderIntent, portfolio, map[string]float64, StrategyPlan) []orderIntent
}

type riskManager interface {
	Review(time.Time, []orderIntent, portfolio, map[string]float64, StrategyPlan) riskReview
}

type executionModel interface {
	Execute(orderIntent, Bar, portfolio, StrategyPlan) (executionOutcome, error)
}

type percentOfEquitySizer struct{}

type limitRiskManager struct{}

type barExecutionModel struct{}

type portfolio struct {
	cash      float64
	positions map[string]Position
	trades    []Trade
}

func newPortfolio(initialCash float64) portfolio {
	return portfolio{
		cash:      initialCash,
		positions: make(map[string]Position),
	}
}

func (p portfolio) hasPosition(symbol string) bool {
	position, ok := p.positions[symbol]
	return ok && position.Quantity > 0
}

func (p portfolio) positionCount() int {
	count := 0
	for _, position := range p.positions {
		if position.Quantity > 0 {
			count++
		}
	}
	return count
}

func (p portfolio) equity(prices map[string]float64) float64 {
	total := p.cash
	for symbol, position := range p.positions {
		price := prices[symbol]
		total += position.Quantity * price
	}
	return total
}

func (p portfolio) positionsValue(prices map[string]float64) float64 {
	var total float64
	for symbol, position := range p.positions {
		total += position.Quantity * prices[symbol]
	}
	return total
}

func (p portfolio) snapshots(prices map[string]float64) []PositionSnapshot {
	equity := p.equity(prices)
	symbols := make([]string, 0, len(p.positions))
	for symbol, position := range p.positions {
		if position.Quantity <= 0 {
			continue
		}
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	out := make([]PositionSnapshot, 0, len(symbols))
	for _, symbol := range symbols {
		position := p.positions[symbol]
		price := prices[symbol]
		marketValue := position.Quantity * price
		weightPct := 0.0
		if equity > 0 {
			weightPct = marketValue / equity * 100
		}
		out = append(out, PositionSnapshot{
			Symbol:      symbol,
			Quantity:    position.Quantity,
			AvgPrice:    position.AvgPrice,
			PeakPrice:   position.PeakPrice,
			MarketPrice: price,
			MarketValue: marketValue,
			WeightPct:   weightPct,
			EntryTime:   position.EntryTime,
		})
	}
	return out
}

func (p *portfolio) apply(trade Trade) error {
	errb := oops.In("backtest_portfolio").With("symbol", trade.Symbol, "side", trade.Side)
	switch trade.Side {
	case SideBuy:
		cost := trade.Notional + trade.totalCost()
		if cost > p.cash+1e-9 {
			return errb.With("cash", p.cash, "cost", cost).New("buy fill exceeds cash")
		}
		position := p.positions[trade.Symbol]
		wasOpen := position.Quantity > 0
		totalQuantity := position.Quantity + trade.Quantity
		totalCost := position.Quantity*position.AvgPrice + trade.Notional
		position.Symbol = trade.Symbol
		position.Quantity = totalQuantity
		position.AvgPrice = totalCost / totalQuantity
		if !wasOpen {
			position.EntryTime = trade.Time
			position.PeakPrice = trade.Price
		} else if trade.Price > position.PeakPrice {
			position.PeakPrice = trade.Price
		}
		if position.PeakPrice <= 0 {
			position.PeakPrice = position.AvgPrice
		}
		p.cash -= cost
		p.positions[trade.Symbol] = position
	case SideSell:
		position := p.positions[trade.Symbol]
		if position.Quantity < trade.Quantity {
			return errb.With("position_quantity", position.Quantity, "fill_quantity", trade.Quantity).New("sell fill exceeds position")
		}
		p.cash += trade.Notional - trade.totalCost()
		position.Quantity -= trade.Quantity
		if position.Quantity <= 1e-9 {
			delete(p.positions, trade.Symbol)
		} else {
			p.positions[trade.Symbol] = position
		}
	default:
		return errb.New("unsupported fill side")
	}
	p.trades = append(p.trades, trade)
	return nil
}

func (p *portfolio) updateTrailingPeaks(currentTime time.Time, bars map[string]Bar, fill string) {
	for symbol, position := range p.positions {
		bar, ok := bars[symbol]
		if !ok || position.Quantity <= 0 {
			continue
		}
		if position.EntryTime.Equal(currentTime) && fill != FillNextOpen {
			continue
		}
		peak := position.PeakPrice
		if peak <= 0 {
			peak = position.AvgPrice
		}
		if bar.High <= peak {
			continue
		}
		position.PeakPrice = bar.High
		p.positions[symbol] = position
	}
}

func (t Trade) totalCost() float64 {
	if t.TotalCost > 0 {
		return t.TotalCost
	}
	return t.Commission + t.Tax + t.ExchangeFee
}

func (percentOfEquitySizer) Size(intents []orderIntent, p portfolio, prices map[string]float64, plan StrategyPlan) []orderIntent {
	equity := p.equity(prices)
	out := make([]orderIntent, 0, len(intents))
	for _, intent := range intents {
		if plan.OrderType == OrderTypeRebalance && intent.Side == SideBuy {
			next, ok := rebalanceIntent(intent, p, prices, equity, plan)
			if ok {
				out = append(out, next)
			}
			continue
		}
		if intent.Side == SideSell {
			if position := p.positions[intent.Symbol]; position.Quantity > 0 && intent.Quantity == 0 {
				intent.Quantity = position.Quantity
			}
			out = append(out, intent)
			continue
		}
		intent.Amount = equity * plan.Sizing.Value / 100
		out = append(out, intent)
	}
	return out
}

func rebalanceIntent(intent orderIntent, p portfolio, prices map[string]float64, equity float64, plan StrategyPlan) (orderIntent, bool) {
	price := prices[intent.Symbol]
	if price <= 0 || equity <= 0 {
		return orderIntent{}, false
	}
	targetValue := equity * plan.Sizing.Value / 100
	currentValue := p.positions[intent.Symbol].Quantity * price
	delta := targetValue - currentValue
	if math.Abs(delta) < price {
		return orderIntent{}, false
	}
	if currentValue > 0 {
		intent.Reason = "rebalance"
	}
	if delta > 0 {
		intent.Amount = delta
		return intent, true
	}
	intent.Side = SideSell
	intent.Amount = 0
	intent.Quantity = math.Floor(math.Abs(delta) / price)
	if intent.Quantity <= 0 {
		return orderIntent{}, false
	}
	return intent, true
}

func (limitRiskManager) Review(currentTime time.Time, intents []orderIntent, p portfolio, prices map[string]float64, plan StrategyPlan) riskReview {
	equity := p.equity(prices)
	out := riskReview{Approved: make([]orderIntent, 0, len(intents))}
	for _, intent := range intents {
		if intent.Side == SideSell {
			out.Approved = append(out.Approved, intent)
			continue
		}
		if plan.Risk.MaxPositions > 0 && !p.hasPosition(intent.Symbol) && p.positionCount() >= plan.Risk.MaxPositions {
			out.Events = append(out.Events, Event{
				Time:   currentTime,
				Layer:  "risk",
				Type:   "rejected",
				Symbol: intent.Symbol,
				Side:   intent.Side,
				Reason: "max_positions",
			})
			continue
		}
		if plan.Risk.MaxSymbolWeightPct > 0 && equity > 0 {
			if intent.Amount/equity*100 > plan.Risk.MaxSymbolWeightPct {
				out.Events = append(out.Events, Event{
					Time:   currentTime,
					Layer:  "risk",
					Type:   "rejected",
					Symbol: intent.Symbol,
					Side:   intent.Side,
					Reason: "max_symbol_weight_pct",
				})
				continue
			}
		}
		out.Approved = append(out.Approved, intent)
	}
	return out
}

func (barExecutionModel) Execute(intent orderIntent, bar Bar, p portfolio, plan StrategyPlan) (executionOutcome, error) {
	errb := oops.In("backtest_execution_model").With("symbol", intent.Symbol, "side", intent.Side)
	if bar.isNoTradeBar() {
		return executionOutcome{Events: []Event{deferredBarEvent(bar, intent, "deferred_no_trade_bar", "no_trade_bar")}}, nil
	}
	priceDecision, err := executionPrice(intent, bar, p, plan)
	if err != nil {
		return executionOutcome{}, errb.Wrap(err)
	}
	if priceDecision.BlockedReason != "" {
		events := append([]Event(nil), priceDecision.Events...)
		events = append(events, unfilledEvent(bar.Time, intent, priceDecision.BlockedReason))
		return executionOutcome{Events: events}, nil
	}
	price := priceDecision.Price
	if price <= 0 {
		return executionOutcome{}, errb.With("price", price).New("invalid market data bar: fill price must be positive")
	}
	if blockedReason := slippageBlockedReason(intent, plan); blockedReason != "" {
		return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, blockedReason)}}, nil
	}
	if intent.Side == SideBuy {
		basePrice := price
		var slippageBps float64
		if !isParticipationSlippage(plan.Slippage) {
			price, slippageBps = applySlippage(basePrice, SideBuy, 0, bar, plan.Slippage, intent)
			price = applyTickSize(price, SideBuy, plan.TickSize)
		}
		desiredQuantity := roundQuantityToLot(intent.Amount/price, plan.LotSize)
		if desiredQuantity <= 0 {
			return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "sized_quantity_is_zero")}}, nil
		}
		quantity, partial := capQuantityByLiquidity(desiredQuantity, price, bar, plan.Liquidity)
		if quantity <= 0 {
			return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "liquidity_cap_quantity_is_zero")}}, nil
		}
		quantity = roundQuantityToLot(quantity, plan.LotSize)
		partial = partial || quantity < desiredQuantity
		if quantity <= 0 {
			return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "lot_size_quantity_is_zero")}}, nil
		}
		if isParticipationSlippage(plan.Slippage) {
			price, slippageBps = applySlippage(basePrice, SideBuy, quantity, bar, plan.Slippage, intent)
			price = applyTickSize(price, SideBuy, plan.TickSize)
		}
		notional := quantity * price
		commission := executionCost(SideBuy, notional, quantity, plan.Commission)
		tax := executionCost(SideBuy, notional, quantity, plan.Tax)
		exchangeFee := executionCost(SideBuy, notional, quantity, plan.ExchangeFee)
		totalCost := commission + tax + exchangeFee
		if notional+totalCost > p.cash {
			quantity = affordableBuyQuantity(p.cash, price, quantity, plan)
			quantity = roundQuantityToLot(quantity, plan.LotSize)
			if isParticipationSlippage(plan.Slippage) {
				price, slippageBps = applySlippage(basePrice, SideBuy, quantity, bar, plan.Slippage, intent)
				price = applyTickSize(price, SideBuy, plan.TickSize)
			}
			notional = quantity * price
			commission = executionCost(SideBuy, notional, quantity, plan.Commission)
			tax = executionCost(SideBuy, notional, quantity, plan.Tax)
			exchangeFee = executionCost(SideBuy, notional, quantity, plan.ExchangeFee)
			totalCost = commission + tax + exchangeFee
		}
		if quantity <= 0 {
			return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "insufficient_cash")}}, nil
		}
		trade := Trade{
			Time:        bar.Time,
			Symbol:      intent.Symbol,
			Side:        SideBuy,
			Quantity:    quantity,
			Price:       price,
			Notional:    notional,
			Commission:  commission,
			Tax:         tax,
			ExchangeFee: exchangeFee,
			TotalCost:   totalCost,
			SlippageBps: slippageBps,
			Reason:      intent.Reason,
		}
		var remaining *orderIntent
		if partial {
			remaining = remainingIntent(intent, desiredQuantity, quantity, price, plan)
		}
		events := fillEvents(bar.Time, intent, trade, partial)
		if partial {
			events = append(events, partialFillRemainderEvent(bar.Time, intent, desiredQuantity, quantity, price, plan, remaining)...)
		}
		return executionOutcome{
			Trade:     &trade,
			Remaining: remaining,
			Events:    append(priceDecision.Events, events...),
		}, nil
	}

	position := p.positions[intent.Symbol]
	if position.Quantity <= 0 {
		return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "position_not_open")}}, nil
	}
	basePrice := price
	var slippageBps float64
	if !isParticipationSlippage(plan.Slippage) {
		price, slippageBps = applySlippage(basePrice, SideSell, 0, bar, plan.Slippage, intent)
		price = applyTickSize(price, SideSell, plan.TickSize)
	}
	desiredQuantity := intent.Quantity
	if desiredQuantity <= 0 || desiredQuantity > position.Quantity {
		desiredQuantity = position.Quantity
	}
	desiredQuantity = roundQuantityToLot(desiredQuantity, plan.LotSize)
	if desiredQuantity <= 0 {
		return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "lot_size_quantity_is_zero")}}, nil
	}
	quantity, partial := capQuantityByLiquidity(desiredQuantity, price, bar, plan.Liquidity)
	if quantity <= 0 {
		return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "liquidity_cap_quantity_is_zero")}}, nil
	}
	quantity = roundQuantityToLot(quantity, plan.LotSize)
	partial = partial || quantity < desiredQuantity
	if quantity <= 0 {
		return executionOutcome{Events: []Event{unfilledEvent(bar.Time, intent, "lot_size_quantity_is_zero")}}, nil
	}
	if isParticipationSlippage(plan.Slippage) {
		price, slippageBps = applySlippage(basePrice, SideSell, quantity, bar, plan.Slippage, intent)
		price = applyTickSize(price, SideSell, plan.TickSize)
	}
	notional := quantity * price
	commission := executionCost(SideSell, notional, quantity, plan.Commission)
	tax := executionCost(SideSell, notional, quantity, plan.Tax)
	exchangeFee := executionCost(SideSell, notional, quantity, plan.ExchangeFee)
	totalCost := commission + tax + exchangeFee
	costBasis := quantity * position.AvgPrice
	realizedPnL := notional - totalCost - costBasis
	tradeReturn := 0.0
	if costBasis > 0 {
		tradeReturn = realizedPnL / costBasis
	}
	trade := Trade{
		Time:        bar.Time,
		Symbol:      intent.Symbol,
		Side:        SideSell,
		Quantity:    quantity,
		Price:       price,
		Notional:    notional,
		Commission:  commission,
		Tax:         tax,
		ExchangeFee: exchangeFee,
		TotalCost:   totalCost,
		SlippageBps: slippageBps,
		Reason:      intent.Reason,
		RealizedPnL: realizedPnL,
		Return:      tradeReturn,
	}
	var remaining *orderIntent
	if partial {
		remaining = remainingIntent(intent, desiredQuantity, quantity, price, plan)
	}
	events := fillEvents(bar.Time, intent, trade, partial)
	if partial {
		events = append(events, partialFillRemainderEvent(bar.Time, intent, desiredQuantity, quantity, price, plan, remaining)...)
	}
	return executionOutcome{
		Trade:     &trade,
		Remaining: remaining,
		Events:    append(priceDecision.Events, events...),
	}, nil
}

func executionPrice(intent orderIntent, bar Bar, p portfolio, plan StrategyPlan) (executionPriceDecision, error) {
	switch plan.OrderType {
	case OrderTypeLimit:
		return limitExecutionPrice(intent.Side, bar, plan.LimitPrice)
	case OrderTypeStop:
		return stopExecutionPrice(intent, bar, plan.StopPrice)
	case OrderTypeStopLimit:
		return stopLimitExecutionPrice(intent, bar, plan)
	case OrderTypeTrailingStop:
		return trailingStopExecutionPrice(intent, bar, p, plan)
	default:
		return marketExecutionPrice(intent, bar, plan)
	}
}

func marketExecutionPrice(intent orderIntent, bar Bar, plan StrategyPlan) (executionPriceDecision, error) {
	switch plan.Fill {
	case FillIntrabarOHLC:
		return executionPriceDecision{
			Price: bar.Open,
			Events: []Event{intrabarPathEvent(
				bar.Time,
				intent,
				withDefault(plan.IntrabarAmbiguityPolicy, IntrabarAmbiguityConservative),
				bar.Open,
			)},
		}, nil
	default:
		price, err := fillPrice(plan.Fill, bar)
		return executionPriceDecision{Price: price}, err
	}
}

func limitExecutionPrice(side Side, bar Bar, limitPrice float64) (executionPriceDecision, error) {
	if limitPrice <= 0 {
		return executionPriceDecision{}, oops.In("backtest_execution_model").With("limit_price", limitPrice).New("limit price must be positive")
	}
	switch side {
	case SideBuy:
		if bar.Low > limitPrice {
			return executionPriceDecision{BlockedReason: "limit_not_reached"}, nil
		}
		return executionPriceDecision{Price: limitPrice}, nil
	case SideSell:
		if bar.High < limitPrice {
			return executionPriceDecision{BlockedReason: "limit_not_reached"}, nil
		}
		return executionPriceDecision{Price: limitPrice}, nil
	default:
		return executionPriceDecision{}, oops.In("backtest_execution_model").With("side", side).New("unsupported limit order side")
	}
}

func stopExecutionPrice(intent orderIntent, bar Bar, stopPrice float64) (executionPriceDecision, error) {
	if stopPrice <= 0 {
		return executionPriceDecision{}, oops.In("backtest_execution_model").With("stop_price", stopPrice).New("stop price must be positive")
	}
	if !stopTriggered(intent.Side, bar, stopPrice) {
		return executionPriceDecision{BlockedReason: "stop_not_triggered"}, nil
	}
	return executionPriceDecision{
		Price:  stopPrice,
		Events: []Event{stopTriggeredEvent(bar.Time, intent, stopPrice)},
	}, nil
}

func stopLimitExecutionPrice(intent orderIntent, bar Bar, plan StrategyPlan) (executionPriceDecision, error) {
	decision, err := stopExecutionPrice(intent, bar, plan.StopPrice)
	if err != nil || decision.BlockedReason != "" {
		return decision, err
	}
	limitDecision, err := limitExecutionPrice(intent.Side, bar, plan.LimitPrice)
	if err != nil {
		return executionPriceDecision{}, err
	}
	limitDecision.Events = append(decision.Events, limitDecision.Events...)
	if limitDecision.BlockedReason != "" {
		return limitDecision, nil
	}
	if stopLimitIsAmbiguous(intent.Side, bar, plan.StopPrice, plan.LimitPrice, plan.IntrabarAmbiguityPolicy) {
		return executionPriceDecision{
			BlockedReason: "intrabar_ambiguous",
			Events:        decision.Events,
		}, nil
	}
	return limitDecision, nil
}

func trailingStopExecutionPrice(intent orderIntent, bar Bar, p portfolio, plan StrategyPlan) (executionPriceDecision, error) {
	if intent.Side == SideBuy {
		return marketExecutionPrice(intent, bar, plan)
	}
	if plan.TrailingStopPct <= 0 || plan.TrailingStopPct >= 100 {
		return executionPriceDecision{}, oops.In("backtest_execution_model").With("trailing_stop_pct", plan.TrailingStopPct).New("trailing stop pct must be between 0 and 100")
	}
	position := p.positions[intent.Symbol]
	if position.Quantity <= 0 {
		return executionPriceDecision{BlockedReason: "position_not_open"}, nil
	}
	peak := position.PeakPrice
	if peak <= 0 {
		peak = position.AvgPrice
	}
	stopPrice := peak * (1 - plan.TrailingStopPct/100)
	events := []Event{trailingStopUpdatedEvent(bar.Time, intent, peak, stopPrice)}
	if !stopTriggered(intent.Side, bar, stopPrice) {
		return executionPriceDecision{BlockedReason: "trailing_stop_not_triggered", Events: events}, nil
	}
	events = append(events, trailingStopTriggeredEvent(bar.Time, intent, stopPrice))
	return executionPriceDecision{Price: stopPrice, Events: events}, nil
}

func stopTriggered(side Side, bar Bar, stopPrice float64) bool {
	switch side {
	case SideBuy:
		return bar.High >= stopPrice
	case SideSell:
		return bar.Low <= stopPrice
	default:
		return false
	}
}

func stopLimitIsAmbiguous(side Side, bar Bar, stopPrice float64, limitPrice float64, policy string) bool {
	switch policy {
	case IntrabarAmbiguityOptimistic:
		return false
	case IntrabarAmbiguityOpenHighLowClose:
		return side == SideSell && limitPrice > stopPrice && bar.Open > stopPrice
	case IntrabarAmbiguityOpenLowHighClose:
		return side == SideBuy && limitPrice < stopPrice && bar.Open < stopPrice
	}
	switch side {
	case SideBuy:
		if limitPrice >= stopPrice || bar.Open >= stopPrice {
			return false
		}
		return bar.Low <= limitPrice && bar.High >= stopPrice
	case SideSell:
		if limitPrice <= stopPrice || bar.Open <= stopPrice {
			return false
		}
		return bar.High >= limitPrice && bar.Low <= stopPrice
	default:
		return false
	}
}

func fillPrice(fill string, bar Bar) (float64, error) {
	switch fill {
	case FillSameClose, FillNextClose:
		return bar.Close, nil
	case FillNextOpen:
		return bar.Open, nil
	default:
		return 0, oops.In("backtest_execution_model").With("fill", fill).New("unsupported execution fill")
	}
}

func capQuantityByLiquidity(desiredQuantity float64, price float64, bar Bar, spec LiquiditySpec) (float64, bool) {
	quantity := desiredQuantity
	if spec.MaxParticipationRate > 0 {
		quantity = math.Min(quantity, math.Floor(bar.Volume*spec.MaxParticipationRate))
	}
	if spec.VolumeCap > 0 {
		quantity = math.Min(quantity, math.Floor(spec.VolumeCap))
	}
	if spec.TradedAmountCap > 0 && price > 0 {
		quantity = math.Min(quantity, math.Floor(spec.TradedAmountCap/price))
	}
	return quantity, quantity < desiredQuantity
}

func applySlippage(price float64, side Side, quantity float64, bar Bar, spec CostSpec, intent orderIntent) (float64, float64) {
	value := sideCostValue(spec, side)
	switch withDefault(spec.Type, CostTypeBPS) {
	case CostTypeNone:
		return price, 0
	case CostTypeFixedAmount:
		if side == SideSell {
			adjusted := math.Max(0, price-value)
			return adjusted, effectiveSlippageBps(price, price-adjusted)
		}
		adjusted := price + value
		return adjusted, effectiveSlippageBps(price, adjusted-price)
	case CostTypeSpreadProxy:
		spread := math.Max(0, bar.High-bar.Low)
		amount := spread * value
		if amount <= 0 {
			return price, 0
		}
		if side == SideSell {
			adjusted := math.Max(0, price-amount)
			return adjusted, effectiveSlippageBps(price, price-adjusted)
		}
		adjusted := price + amount
		return adjusted, effectiveSlippageBps(price, adjusted-price)
	case CostTypeATR:
		amount := intent.SlippageReference * value
		if amount <= 0 {
			return price, 0
		}
		if side == SideSell {
			adjusted := math.Max(0, price-amount)
			return adjusted, effectiveSlippageBps(price, price-adjusted)
		}
		adjusted := price + amount
		return adjusted, effectiveSlippageBps(price, adjusted-price)
	case CostTypeVolatility:
		if price <= 0 || value <= 0 || bar.High <= bar.Low {
			return price, 0
		}
		bps := (bar.High - bar.Low) / price * 10000 * value
		if side == SideSell {
			return price * (1 - bps/10000), bps
		}
		return price * (1 + bps/10000), bps
	case CostTypeParticipation:
		if bar.Volume <= 0 || quantity <= 0 || value <= 0 {
			return price, 0
		}
		bps := value * quantity / bar.Volume
		if side == SideSell {
			return price * (1 - bps/10000), bps
		}
		return price * (1 + bps/10000), bps
	default:
		if side == SideSell {
			return price * (1 - value/10000), value
		}
		return price * (1 + value/10000), value
	}
}

func isParticipationSlippage(spec CostSpec) bool {
	return withDefault(spec.Type, CostTypeBPS) == CostTypeParticipation
}

func isATRSlippage(spec CostSpec) bool {
	return withDefault(spec.Type, CostTypeBPS) == CostTypeATR
}

func slippageBlockedReason(intent orderIntent, plan StrategyPlan) string {
	if isATRSlippage(plan.Slippage) && !intent.SlippageReferenceSet {
		return "slippage_atr_not_ready"
	}
	return ""
}

func roundQuantityToLot(quantity float64, lotSize float64) float64 {
	if quantity <= 0 {
		return 0
	}
	if lotSize <= 0 {
		return math.Floor(quantity)
	}
	return math.Floor(quantity/lotSize) * lotSize
}

func applyTickSize(price float64, side Side, tickSize float64) float64 {
	if price <= 0 || tickSize <= 0 {
		return price
	}
	switch side {
	case SideSell:
		return math.Floor(price/tickSize) * tickSize
	default:
		return math.Ceil(price/tickSize) * tickSize
	}
}

func executionCost(side Side, notional float64, quantity float64, spec CostSpec) float64 {
	value := sideCostValue(spec, side)
	var cost float64
	switch withDefault(spec.Type, CostTypeBPS) {
	case CostTypeNone:
		return 0
	case CostTypeFixedAmount:
		cost = value
	case CostTypeFixedPerUnit:
		cost = quantity * value
	default:
		cost = notional * value / 10000
	}
	if cost > 0 && spec.MinFee > 0 && cost < spec.MinFee {
		return spec.MinFee
	}
	return cost
}

func sideCostValue(spec CostSpec, side Side) float64 {
	if spec.BuyValue != 0 || spec.SellValue != 0 {
		if side == SideBuy {
			return spec.BuyValue
		}
		return spec.SellValue
	}
	return spec.Value
}

func effectiveSlippageBps(basePrice float64, slippageAmount float64) float64 {
	if basePrice <= 0 || slippageAmount <= 0 {
		return 0
	}
	return slippageAmount / basePrice * 10000
}

func affordableBuyQuantity(cash float64, price float64, maxQuantity float64, plan StrategyPlan) float64 {
	high := math.Floor(maxQuantity)
	low := 0.0
	for low < high {
		mid := math.Ceil((low + high + 1) / 2)
		notional := mid * price
		totalCost := executionCost(SideBuy, notional, mid, plan.Commission) +
			executionCost(SideBuy, notional, mid, plan.Tax) +
			executionCost(SideBuy, notional, mid, plan.ExchangeFee)
		if notional+totalCost <= cash+1e-9 {
			low = mid
			continue
		}
		high = mid - 1
	}
	return low
}

func remainingIntent(intent orderIntent, desiredQuantity float64, filledQuantity float64, price float64, plan StrategyPlan) *orderIntent {
	if filledQuantity >= desiredQuantity {
		return nil
	}
	if plan.TimeInForce == TimeInForceIOC {
		return nil
	}
	switch plan.PartialFill.Policy {
	case PartialFillCancel:
		return nil
	case PartialFillExpireAfterNBars:
		nextCarryBars := intent.CarryBars + 1
		if plan.PartialFill.ExpireAfterNBars > 0 && nextCarryBars >= plan.PartialFill.ExpireAfterNBars {
			return nil
		}
		next := intent
		next.CarryBars = nextCarryBars
		if intent.Side == SideBuy {
			next.Amount = math.Max(0, intent.Amount-filledQuantity*price)
		} else {
			next.Quantity = math.Max(0, desiredQuantity-filledQuantity)
		}
		return &next
	default:
		next := intent
		next.CarryBars++
		if intent.Side == SideBuy {
			next.Amount = math.Max(0, intent.Amount-filledQuantity*price)
		} else {
			next.Quantity = math.Max(0, desiredQuantity-filledQuantity)
		}
		return &next
	}
}

func partialFillRemainderEvent(currentTime time.Time, intent orderIntent, desiredQuantity float64, filledQuantity float64, price float64, plan StrategyPlan, remaining *orderIntent) []Event {
	if remaining != nil || filledQuantity >= desiredQuantity {
		return nil
	}
	reason := partialFillRemainderCancelReason(intent, plan)
	if reason == "" {
		return nil
	}
	quantity := math.Max(0, desiredQuantity-filledQuantity)
	return []Event{{
		Time:     currentTime,
		Layer:    "execution",
		Type:     "order_cancelled",
		Symbol:   intent.Symbol,
		Side:     intent.Side,
		Reason:   reason,
		Quantity: quantity,
		Amount:   quantity * price,
	}}
}

func partialFillRemainderCancelReason(intent orderIntent, plan StrategyPlan) string {
	if plan.TimeInForce == TimeInForceIOC {
		return "time_in_force_ioc"
	}
	switch plan.PartialFill.Policy {
	case PartialFillCancel:
		return "partial_fill_cancel"
	case PartialFillExpireAfterNBars:
		nextCarryBars := intent.CarryBars + 1
		if plan.PartialFill.ExpireAfterNBars > 0 && nextCarryBars >= plan.PartialFill.ExpireAfterNBars {
			return "partial_fill_expired"
		}
	}
	return ""
}

func fillEvents(currentTime time.Time, intent orderIntent, trade Trade, partial bool) []Event {
	events := make([]Event, 0, 5)
	if partial {
		events = append(events, Event{
			Time:        currentTime,
			Layer:       "execution",
			Type:        "partial_fill",
			Symbol:      intent.Symbol,
			Side:        intent.Side,
			Reason:      "liquidity_cap",
			Quantity:    trade.Quantity,
			Price:       trade.Price,
			Notional:    trade.Notional,
			SlippageBps: trade.SlippageBps,
		})
	}
	events = append(events, Event{
		Time:        currentTime,
		Layer:       "execution",
		Type:        "fill",
		Symbol:      intent.Symbol,
		Side:        intent.Side,
		Reason:      intent.Reason,
		Quantity:    trade.Quantity,
		Price:       trade.Price,
		Notional:    trade.Notional,
		SlippageBps: trade.SlippageBps,
	})
	if trade.Commission > 0 {
		events = append(events, Event{
			Time:       currentTime,
			Layer:      "execution",
			Type:       "cost",
			Symbol:     intent.Symbol,
			Side:       intent.Side,
			Reason:     "commission",
			Commission: trade.Commission,
			TotalCost:  trade.totalCost(),
		})
	}
	if trade.Tax > 0 {
		events = append(events, Event{
			Time:      currentTime,
			Layer:     "execution",
			Type:      "cost",
			Symbol:    intent.Symbol,
			Side:      intent.Side,
			Reason:    "tax",
			Tax:       trade.Tax,
			TotalCost: trade.totalCost(),
		})
	}
	if trade.ExchangeFee > 0 {
		events = append(events, Event{
			Time:        currentTime,
			Layer:       "execution",
			Type:        "cost",
			Symbol:      intent.Symbol,
			Side:        intent.Side,
			Reason:      "exchange_fee",
			ExchangeFee: trade.ExchangeFee,
			TotalCost:   trade.totalCost(),
		})
	}
	return events
}

func unfilledEvent(time time.Time, intent orderIntent, reason string) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   "unfilled",
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: reason,
	}
}

func stopTriggeredEvent(time time.Time, intent orderIntent, stopPrice float64) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   "stop_triggered",
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: "stop_price",
		Price:  stopPrice,
	}
}

func trailingStopUpdatedEvent(time time.Time, intent orderIntent, peakPrice float64, stopPrice float64) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   "trailing_stop_updated",
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: "trailing_stop",
		Amount: peakPrice,
		Price:  stopPrice,
	}
}

func trailingStopTriggeredEvent(time time.Time, intent orderIntent, stopPrice float64) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   "stop_triggered",
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: "trailing_stop",
		Price:  stopPrice,
	}
}

func intrabarPathEvent(time time.Time, intent orderIntent, policy string, openPrice float64) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   "intrabar_path",
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: policy,
		Price:  openPrice,
	}
}

func deferredEvent(time time.Time, intent orderIntent, eventType string, reason string) Event {
	return Event{
		Time:   time,
		Layer:  "execution",
		Type:   eventType,
		Symbol: intent.Symbol,
		Side:   intent.Side,
		Reason: reason,
	}
}

func deferredBarEvent(bar Bar, intent orderIntent, eventType string, reason string) Event {
	return Event{
		Time:      bar.Time,
		Layer:     "execution",
		Type:      eventType,
		Symbol:    intent.Symbol,
		Side:      intent.Side,
		Reason:    reason,
		Timeframe: bar.Timeframe,
		Session:   bar.Session,
		Status:    bar.Status,
		Price:     bar.Close,
	}
}
