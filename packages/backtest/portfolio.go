package backtest

import (
	"math"
	"time"

	"github.com/samber/oops"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Position struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	AvgPrice float64 `json:"avg_price"`
}

type Trade struct {
	Time        time.Time `json:"time"`
	Symbol      string    `json:"symbol"`
	Side        Side      `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Notional    float64   `json:"notional"`
	Commission  float64   `json:"commission"`
	SlippageBps float64   `json:"slippage_bps,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	RealizedPnL float64   `json:"realized_pnl,omitempty"`
	Return      float64   `json:"return,omitempty"`
}

type orderIntent struct {
	Symbol string
	Side   Side
	Amount float64
	Reason string
}

type fill struct {
	Trade Trade
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
	Execute(orderIntent, Bar, portfolio, StrategyPlan) (fill, bool, Event, error)
}

type percentOfEquitySizer struct{}

type limitRiskManager struct{}

type nextOpenExecutionModel struct{}

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

func (p *portfolio) apply(trade Trade) error {
	errb := oops.In("backtest_portfolio").With("symbol", trade.Symbol, "side", trade.Side)
	switch trade.Side {
	case SideBuy:
		cost := trade.Notional + trade.Commission
		if cost > p.cash+1e-9 {
			return errb.With("cash", p.cash, "cost", cost).New("buy fill exceeds cash")
		}
		position := p.positions[trade.Symbol]
		totalQuantity := position.Quantity + trade.Quantity
		totalCost := position.Quantity*position.AvgPrice + trade.Notional
		position.Symbol = trade.Symbol
		position.Quantity = totalQuantity
		position.AvgPrice = totalCost / totalQuantity
		p.cash -= cost
		p.positions[trade.Symbol] = position
	case SideSell:
		position := p.positions[trade.Symbol]
		if position.Quantity < trade.Quantity {
			return errb.With("position_quantity", position.Quantity, "fill_quantity", trade.Quantity).New("sell fill exceeds position")
		}
		p.cash += trade.Notional - trade.Commission
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

func (percentOfEquitySizer) Size(intents []orderIntent, p portfolio, prices map[string]float64, plan StrategyPlan) []orderIntent {
	equity := p.equity(prices)
	out := make([]orderIntent, 0, len(intents))
	for _, intent := range intents {
		if intent.Side == SideSell {
			out = append(out, intent)
			continue
		}
		intent.Amount = equity * plan.Sizing.Value / 100
		out = append(out, intent)
	}
	return out
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

func (nextOpenExecutionModel) Execute(intent orderIntent, bar Bar, p portfolio, plan StrategyPlan) (fill, bool, Event, error) {
	errb := oops.In("backtest_execution_model").With("symbol", intent.Symbol, "side", intent.Side)
	price := bar.Open
	if price <= 0 {
		if bar.isNoTradeBar() {
			return fill{}, false, deferredEvent(bar.Time, intent, "deferred_no_trade_bar", "no_trade_bar"), nil
		}
		return fill{}, false, Event{}, errb.With("price", price).New("invalid market data bar: fill price must be positive")
	}
	slippageBps := plan.Slippage.Value
	if intent.Side == SideBuy {
		price *= 1 + slippageBps/10000
		quantity := math.Floor(intent.Amount / price)
		if quantity <= 0 {
			return fill{}, false, unfilledEvent(bar.Time, intent, "sized_quantity_is_zero"), nil
		}
		notional := quantity * price
		commission := notional * plan.Commission.Value / 10000
		if notional+commission > p.cash {
			quantity = math.Floor(p.cash / (price * (1 + plan.Commission.Value/10000)))
			notional = quantity * price
			commission = notional * plan.Commission.Value / 10000
		}
		if quantity <= 0 {
			return fill{}, false, unfilledEvent(bar.Time, intent, "insufficient_cash"), nil
		}
		return fill{Trade: Trade{
			Time:        bar.Time,
			Symbol:      intent.Symbol,
			Side:        SideBuy,
			Quantity:    quantity,
			Price:       price,
			Notional:    notional,
			Commission:  commission,
			SlippageBps: slippageBps,
			Reason:      intent.Reason,
		}}, true, Event{}, nil
	}

	position := p.positions[intent.Symbol]
	if position.Quantity <= 0 {
		return fill{}, false, unfilledEvent(bar.Time, intent, "position_not_open"), nil
	}
	price *= 1 - slippageBps/10000
	quantity := position.Quantity
	notional := quantity * price
	commission := notional * plan.Commission.Value / 10000
	costBasis := quantity * position.AvgPrice
	realizedPnL := notional - commission - costBasis
	tradeReturn := 0.0
	if costBasis > 0 {
		tradeReturn = realizedPnL / costBasis
	}
	return fill{Trade: Trade{
		Time:        bar.Time,
		Symbol:      intent.Symbol,
		Side:        SideSell,
		Quantity:    quantity,
		Price:       price,
		Notional:    notional,
		Commission:  commission,
		SlippageBps: slippageBps,
		Reason:      intent.Reason,
		RealizedPnL: realizedPnL,
		Return:      tradeReturn,
	}}, true, Event{}, nil
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
