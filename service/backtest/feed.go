package backtest

import (
	"context"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/service/daily"
	"github.com/samber/oops"
)

type dailyBarFeed struct {
	reader DailyBarRepository
}

func newDailyBarFeed(reader DailyBarRepository) core.StreamingFeed {
	return dailyBarFeed{reader: reader}
}

func (f dailyBarFeed) Open(ctx context.Context, request core.DataRequest) (core.BarStream, error) {
	errb := oops.In("backtest_dailybar_feed").With(
		"from", request.From.Format(time.DateOnly),
		"to", request.To.Format(time.DateOnly),
	)
	timeframe, err := core.ParseTimeframe(defaultTimeframe(request.Timeframe))
	if err != nil {
		return nil, errb.Wrap(err)
	}
	if !timeframe.IsDailyCompatible() {
		return nil, errb.With("timeframe", timeframe.ID).New("canonical daily bar feed supports only 1d, 1w, and 1mo timeframes")
	}
	instruments, err := dataInstrumentsFromRequest(request)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	market, err := singleMarket(instruments)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	stream, err := f.reader.StreamDailyBars(ctx, daily.Query{
		Market: provider.Market(market),
		From:   request.From.Format(time.DateOnly),
		To:     request.To.Format(time.DateOnly),
	})
	if err != nil {
		return nil, errb.With("market", market).Wrapf(err, "open canonical daily bar stream")
	}
	return core.NewTimeframeStream(newDailyBarCursorFrameStream(stream, instruments), timeframe)
}

type dailyBarCursorFrameStream struct {
	stream   daily.BarStream
	allowed  map[string]struct{}
	required map[string]core.InstrumentIdentity
	seen     map[string]struct{}
	buffered *core.Bar
	closed   bool
}

func newDailyBarCursorFrameStream(stream daily.BarStream, instruments []core.InstrumentIdentity) *dailyBarCursorFrameStream {
	allowed := make(map[string]struct{}, len(instruments))
	required := make(map[string]core.InstrumentIdentity, len(instruments))
	for _, instrument := range instruments {
		key := instrumentKey(instrument.Market, instrument.SecurityType, instrument.Symbol)
		allowed[key] = struct{}{}
		required[key] = instrument
	}
	return &dailyBarCursorFrameStream{
		stream:   stream,
		allowed:  allowed,
		required: required,
		seen:     make(map[string]struct{}, len(instruments)),
	}
}

func (s *dailyBarCursorFrameStream) Next(ctx context.Context) (core.BarFrame, bool, error) {
	if err := ctx.Err(); err != nil {
		return core.BarFrame{}, false, oops.In("backtest_dailybar_feed").Wrap(err)
	}
	first, ok, err := s.takeNextAllowed(ctx)
	if err != nil || !ok {
		return core.BarFrame{}, false, err
	}
	frame := core.BarFrame{Time: first.Time, Bars: map[string]core.Bar{first.Symbol: first}}
	for {
		bar, ok, err := s.takeNextAllowed(ctx)
		if err != nil {
			return core.BarFrame{}, false, err
		}
		if !ok {
			return frame, true, nil
		}
		if !bar.Time.Equal(frame.Time) {
			s.buffered = &bar
			return frame, true, nil
		}
		frame.Bars[bar.Symbol] = bar
	}
}

func (s *dailyBarCursorFrameStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.stream.Close()
}

func (s *dailyBarCursorFrameStream) takeNextAllowed(ctx context.Context) (core.Bar, bool, error) {
	if s.buffered != nil {
		bar := *s.buffered
		s.buffered = nil
		return bar, true, nil
	}
	for {
		row, ok, err := s.stream.Next(ctx)
		if err != nil {
			return core.Bar{}, false, oops.In("backtest_dailybar_feed").Wrap(err)
		}
		if !ok {
			if err := s.validateSeen(); err != nil {
				return core.Bar{}, false, err
			}
			return core.Bar{}, false, nil
		}
		key := instrumentKey(string(row.Market), string(row.SecurityType), row.Symbol)
		if _, ok := s.allowed[key]; !ok {
			continue
		}
		bar, err := canonicalDailyBarToBacktestBar(row)
		if err != nil {
			return core.Bar{}, false, oops.In("backtest_dailybar_feed").With("symbol", row.Symbol, "trading_date", row.TradingDate).Wrap(err)
		}
		s.seen[key] = struct{}{}
		return bar, true, nil
	}
}

func (s *dailyBarCursorFrameStream) validateSeen() error {
	for key, instrument := range s.required {
		if _, ok := s.seen[key]; ok {
			continue
		}
		return oops.In("backtest_dailybar_feed").
			With("symbol", instrument.Symbol, "instrument_market", instrument.Market, "instrument_security_type", instrument.SecurityType).
			Errorf("canonical daily bars not found for backtest symbol: symbol=%s market=%s security_type=%s", instrument.Symbol, instrument.Market, instrument.SecurityType)
	}
	return nil
}

func dataInstrumentsFromRequest(request core.DataRequest) ([]core.InstrumentIdentity, error) {
	instruments := append([]core.InstrumentIdentity(nil), request.Instruments...)
	if len(instruments) == 0 {
		for _, symbol := range request.Symbols {
			instruments = append(instruments, core.InstrumentIdentity{Symbol: symbol})
		}
	}
	if request.Benchmark.Symbol != "" {
		instruments = append(instruments, core.InstrumentIdentity{
			Symbol:       request.Benchmark.Symbol,
			Market:       request.Benchmark.Market,
			SecurityType: request.Benchmark.SecurityType,
		})
	}
	return normalizeDataInstruments(instruments, "")
}

func singleMarket(instruments []core.InstrumentIdentity) (string, error) {
	market := ""
	for _, instrument := range instruments {
		if market == "" {
			market = instrument.Market
			continue
		}
		if instrument.Market != market {
			return "", oops.In("backtest_dailybar_feed").
				With("market", market, "other_market", instrument.Market).
				New("repository-backed backtest feed requires one market per stream request")
		}
	}
	return market, nil
}

func instrumentKey(market string, securityType string, symbol string) string {
	return market + "\x00" + securityType + "\x00" + symbol
}

func defaultTimeframe(value string) string {
	if value == "" {
		return core.Timeframe1Day
	}
	return value
}
