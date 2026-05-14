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
	instruments, err := dataInstrumentsFromRequest(request)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	streams := make([]*dailyBarInstrumentStream, 0, len(instruments))
	for _, instrument := range instruments {
		stream, err := f.reader.StreamDailyBars(ctx, daily.Query{
			Market:       provider.Market(instrument.Market),
			SecurityType: provider.SecurityType(instrument.SecurityType),
			Symbol:       instrument.Symbol,
			From:         request.From.Format(time.DateOnly),
			To:           request.To.Format(time.DateOnly),
		})
		if err != nil {
			closeDailyBarStreams(streams)
			return nil, errb.With("symbol", instrument.Symbol, "instrument_market", instrument.Market, "instrument_security_type", instrument.SecurityType).Wrapf(err, "open canonical daily bar stream")
		}
		next := &dailyBarInstrumentStream{identity: instrument, stream: stream}
		if err := next.advance(ctx); err != nil {
			closeDailyBarStreams(append(streams, next))
			return nil, errb.With("symbol", instrument.Symbol, "instrument_market", instrument.Market, "instrument_security_type", instrument.SecurityType).Wrap(err)
		}
		if !next.hasCurrent {
			closeDailyBarStreams(append(streams, next))
			return nil, errb.With("symbol", instrument.Symbol, "instrument_market", instrument.Market, "instrument_security_type", instrument.SecurityType).Errorf("canonical daily bars not found for backtest symbol: symbol=%s market=%s security_type=%s", instrument.Symbol, instrument.Market, instrument.SecurityType)
		}
		streams = append(streams, next)
	}
	return &dailyBarFrameStream{streams: streams}, nil
}

type dailyBarFrameStream struct {
	streams []*dailyBarInstrumentStream
	closed  bool
}

func (s *dailyBarFrameStream) Next(ctx context.Context) (core.BarFrame, bool, error) {
	if err := ctx.Err(); err != nil {
		return core.BarFrame{}, false, oops.In("backtest_dailybar_feed").Wrap(err)
	}
	minTime, ok := s.nextTime()
	if !ok {
		return core.BarFrame{}, false, nil
	}
	frame := core.BarFrame{Time: minTime, Bars: map[string]core.Bar{}}
	for _, stream := range s.streams {
		if !stream.hasCurrent || !stream.current.Time.Equal(minTime) {
			continue
		}
		frame.Bars[stream.current.Symbol] = stream.current
		if err := stream.advance(ctx); err != nil {
			return core.BarFrame{}, false, err
		}
	}
	return frame, true, nil
}

func (s *dailyBarFrameStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return closeDailyBarStreams(s.streams)
}

func (s *dailyBarFrameStream) nextTime() (time.Time, bool) {
	var out time.Time
	found := false
	for _, stream := range s.streams {
		if !stream.hasCurrent {
			continue
		}
		if !found || stream.current.Time.Before(out) {
			out = stream.current.Time
			found = true
		}
	}
	return out, found
}

type dailyBarInstrumentStream struct {
	identity   core.InstrumentIdentity
	stream     daily.BarStream
	current    core.Bar
	hasCurrent bool
}

func (s *dailyBarInstrumentStream) advance(ctx context.Context) error {
	row, ok, err := s.stream.Next(ctx)
	if err != nil {
		return oops.In("backtest_dailybar_feed").With("symbol", s.identity.Symbol).Wrap(err)
	}
	if !ok {
		s.hasCurrent = false
		return nil
	}
	bar, err := canonicalDailyBarToBacktestBar(row)
	if err != nil {
		return oops.In("backtest_dailybar_feed").With("symbol", s.identity.Symbol, "trading_date", row.TradingDate).Wrap(err)
	}
	s.current = bar
	s.hasCurrent = true
	return nil
}

func closeDailyBarStreams(streams []*dailyBarInstrumentStream) error {
	var out error
	for _, stream := range streams {
		if stream == nil || stream.stream == nil {
			continue
		}
		if err := stream.stream.Close(); err != nil {
			if out == nil {
				out = err
			} else {
				out = oops.Join(out, err)
			}
		}
	}
	return out
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
