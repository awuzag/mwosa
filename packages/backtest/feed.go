package backtest

import (
	"context"
	"slices"
	"time"

	"github.com/samber/oops"
)

type Bar struct {
	Time         time.Time `json:"time"`
	Symbol       string    `json:"symbol"`
	Market       string    `json:"market,omitempty"`
	SecurityType string    `json:"security_type,omitempty"`
	Open         float64   `json:"open"`
	High         float64   `json:"high"`
	Low          float64   `json:"low"`
	Close        float64   `json:"close"`
	Volume       float64   `json:"volume,omitempty"`
	TradedAmount float64   `json:"traded_amount,omitempty"`
}

type DataRequest struct {
	Symbols []string
	From    time.Time
	To      time.Time
}

type Feed interface {
	Bars(ctx context.Context, request DataRequest) ([]Bar, error)
}

type MemoryFeed struct {
	bars []Bar
}

func NewMemoryFeed(bars []Bar) MemoryFeed {
	copied := append([]Bar(nil), bars...)
	sortBars(copied)
	return MemoryFeed{bars: copied}
}

func (f MemoryFeed) Bars(ctx context.Context, request DataRequest) ([]Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, oops.In("backtest_memory_feed").Wrap(err)
	}
	symbols := make(map[string]struct{}, len(request.Symbols))
	for _, symbol := range request.Symbols {
		symbols[symbol] = struct{}{}
	}
	out := make([]Bar, 0, len(f.bars))
	for _, bar := range f.bars {
		if _, ok := symbols[bar.Symbol]; !ok {
			continue
		}
		if bar.Time.Before(request.From) || bar.Time.After(request.To) {
			continue
		}
		out = append(out, bar)
	}
	sortBars(out)
	return out, nil
}

func sortBars(bars []Bar) {
	slices.SortFunc(bars, func(a, b Bar) int {
		if a.Time.Before(b.Time) {
			return -1
		}
		if a.Time.After(b.Time) {
			return 1
		}
		if a.Symbol < b.Symbol {
			return -1
		}
		if a.Symbol > b.Symbol {
			return 1
		}
		return 0
	})
}

func isPriceField(field string) bool {
	switch field {
	case "open", "high", "low", "close", "volume":
		return true
	default:
		return false
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
	case "close":
		return bar.Close, true
	case "volume":
		return bar.Volume, true
	default:
		return 0, false
	}
}

func (bar Bar) isNoTradeBar() bool {
	return bar.Open == 0 &&
		bar.High == 0 &&
		bar.Low == 0 &&
		bar.Close > 0 &&
		bar.Volume == 0 &&
		bar.TradedAmount == 0
}

func (bar Bar) validate() error {
	errb := oops.In("backtest_bar").With("symbol", bar.Symbol, "time", bar.Time.Format(time.DateOnly))
	if bar.Open < 0 || bar.High < 0 || bar.Low < 0 || bar.Close < 0 {
		return errb.With("open", bar.Open, "high", bar.High, "low", bar.Low, "close", bar.Close).New("invalid market data bar: price must not be negative")
	}
	if bar.Volume < 0 || bar.TradedAmount < 0 {
		return errb.With("volume", bar.Volume, "traded_amount", bar.TradedAmount).New("invalid market data bar: liquidity fields must not be negative")
	}
	if bar.Close <= 0 {
		return errb.With("close", bar.Close).New("invalid market data bar: close price must be positive")
	}
	if bar.isNoTradeBar() {
		return nil
	}
	if bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 {
		return errb.With("open", bar.Open, "high", bar.High, "low", bar.Low, "volume", bar.Volume, "traded_amount", bar.TradedAmount).New("invalid market data bar: zero OHLC is only allowed for no-trade bars")
	}
	if bar.High < bar.Low {
		return errb.With("high", bar.High, "low", bar.Low).New("invalid market data bar: high price must be greater than or equal to low price")
	}
	return nil
}

func validateBars(bars []Bar) error {
	for _, bar := range bars {
		if err := bar.validate(); err != nil {
			return err
		}
	}
	return nil
}
