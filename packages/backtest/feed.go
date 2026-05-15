package backtest

import (
	"context"
	"slices"
	"time"

	"github.com/samber/oops"
)

const (
	BarSessionRegular = "regular"

	BarStatusOK      = "ok"
	BarStatusNoTrade = "no_trade"
	BarStatusMissing = "missing"
)

type Bar struct {
	Time          time.Time `json:"time"`
	Symbol        string    `json:"symbol"`
	Market        string    `json:"market,omitempty"`
	SecurityType  string    `json:"security_type,omitempty"`
	Timeframe     string    `json:"timeframe,omitempty"`
	Session       string    `json:"session,omitempty"`
	Status        string    `json:"status,omitempty"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Close         float64   `json:"close"`
	AdjustedClose float64   `json:"adjusted_close,omitempty"`
	Volume        float64   `json:"volume,omitempty"`
	TradedAmount  float64   `json:"traded_amount,omitempty"`
	MarketCap     float64   `json:"market_cap,omitempty"`
	NAV           float64   `json:"nav,omitempty"`
}

type DataRequest struct {
	Symbols     []string
	Instruments []InstrumentIdentity
	From        time.Time
	To          time.Time
	Timeframe   string
	Benchmark   BenchmarkSpec
	WarmupBars  int
}

type BarFrame struct {
	Time time.Time      `json:"time"`
	Bars map[string]Bar `json:"bars"`
}

type BarStream interface {
	Next(ctx context.Context) (BarFrame, bool, error)
	Close() error
}

type StreamingFeed interface {
	Open(ctx context.Context, request DataRequest) (BarStream, error)
}

type MemoryFeed struct {
	bars []Bar
}

func NewMemoryFeed(bars []Bar) MemoryFeed {
	copied := append([]Bar(nil), bars...)
	sortBars(copied)
	return MemoryFeed{bars: copied}
}

func (f MemoryFeed) Open(ctx context.Context, request DataRequest) (BarStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, oops.In("backtest_memory_feed").Wrap(err)
	}
	timeframe, err := ParseTimeframe(withDefault(request.Timeframe, Timeframe1Day))
	if err != nil {
		return nil, oops.In("backtest_memory_feed").Wrap(err)
	}
	sourceTimeframe := sourceTimeframeForRequest(timeframe)
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
		out = append(out, normalizeBarMetadata(bar, sourceTimeframe))
	}
	sortBars(out)
	return NewTimeframeStream(&memoryBarStream{bars: out}, timeframe)
}

type memoryBarStream struct {
	bars   []Bar
	offset int
}

func (s *memoryBarStream) Next(ctx context.Context) (BarFrame, bool, error) {
	if err := ctx.Err(); err != nil {
		return BarFrame{}, false, oops.In("backtest_memory_feed").Wrap(err)
	}
	if s.offset >= len(s.bars) {
		return BarFrame{}, false, nil
	}
	current := s.bars[s.offset].Time
	frame := BarFrame{
		Time: current,
		Bars: make(map[string]Bar),
	}
	for s.offset < len(s.bars) && s.bars[s.offset].Time.Equal(current) {
		bar := s.bars[s.offset]
		frame.Bars[bar.Symbol] = bar
		s.offset++
	}
	return frame, true, nil
}

func (s *memoryBarStream) Close() error {
	return nil
}

func normalizeBarMetadata(bar Bar, timeframe string) Bar {
	if bar.Timeframe == "" {
		bar.Timeframe = timeframe
	}
	if bar.Session == "" {
		bar.Session = BarSessionRegular
	}
	if bar.Status == "" {
		if bar.isNoTradeBar() {
			bar.Status = BarStatusNoTrade
		} else {
			bar.Status = BarStatusOK
		}
	}
	return bar
}

func sourceTimeframeForRequest(timeframe Timeframe) string {
	if timeframe.IsDailyResample() {
		return Timeframe1Day
	}
	return withDefault(timeframe.ID, Timeframe1Day)
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
	case "open", "high", "low", "close", "adjusted_close", "volume", "amount", "traded_amount", "market_cap", "nav":
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
	case "adjusted_close":
		if bar.AdjustedClose != 0 {
			return bar.AdjustedClose, true
		}
		return bar.Close, true
	case "volume":
		return bar.Volume, true
	case "amount", "traded_amount":
		return bar.TradedAmount, true
	case "market_cap":
		return bar.MarketCap, true
	case "nav":
		return bar.NAV, true
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
	if bar.Open < 0 || bar.High < 0 || bar.Low < 0 || bar.Close < 0 || bar.AdjustedClose < 0 {
		return errb.With("open", bar.Open, "high", bar.High, "low", bar.Low, "close", bar.Close, "adjusted_close", bar.AdjustedClose).New("invalid market data bar: price must not be negative")
	}
	if bar.Volume < 0 || bar.TradedAmount < 0 || bar.MarketCap < 0 || bar.NAV < 0 {
		return errb.With("volume", bar.Volume, "traded_amount", bar.TradedAmount, "market_cap", bar.MarketCap, "nav", bar.NAV).New("invalid market data bar: numeric market data fields must not be negative")
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
