package backtest

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/oops"
)

func NewTimeframeStream(stream BarStream, timeframe Timeframe) (BarStream, error) {
	if stream == nil {
		return nil, oops.In("backtest_timeframe_stream").New("bar stream is nil")
	}
	if timeframe.IsDailyNative() {
		return stream, nil
	}
	if timeframe.IsDailyResample() {
		return &resamplingBarStream{stream: stream, timeframe: timeframe}, nil
	}
	return stream, nil
}

type resamplingBarStream struct {
	stream     BarStream
	timeframe  Timeframe
	currentKey string
	builders   map[string]*barAggregate
	buffered   *BarFrame
	closed     bool
}

func (s *resamplingBarStream) Next(ctx context.Context) (BarFrame, bool, error) {
	if err := ctx.Err(); err != nil {
		return BarFrame{}, false, oops.In("backtest_timeframe_stream").Wrap(err)
	}
	for {
		frame, ok, err := s.takeFrame(ctx)
		if err != nil || !ok {
			if err != nil {
				return BarFrame{}, false, err
			}
			if len(s.builders) == 0 {
				return BarFrame{}, false, nil
			}
			out := s.flush()
			return out, true, nil
		}
		key := bucketKey(s.timeframe, frame.Time)
		if s.currentKey == "" {
			s.currentKey = key
			s.builders = make(map[string]*barAggregate, len(frame.Bars))
		}
		if key != s.currentKey {
			s.buffered = &frame
			out := s.flush()
			s.currentKey = ""
			return out, true, nil
		}
		for _, symbol := range sortedBarSymbols(frame.Bars) {
			bar := frame.Bars[symbol]
			aggregate := s.builders[symbol]
			if aggregate == nil {
				aggregate = &barAggregate{}
				s.builders[symbol] = aggregate
			}
			aggregate.add(bar)
		}
	}
}

func (s *resamplingBarStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.stream.Close()
}

func (s *resamplingBarStream) takeFrame(ctx context.Context) (BarFrame, bool, error) {
	if s.buffered != nil {
		frame := *s.buffered
		s.buffered = nil
		return frame, true, nil
	}
	return s.stream.Next(ctx)
}

func (s *resamplingBarStream) flush() BarFrame {
	out := BarFrame{Bars: make(map[string]Bar, len(s.builders))}
	for _, symbol := range sortedAggregateSymbols(s.builders) {
		aggregate := s.builders[symbol]
		bar := aggregate.bar
		bar.Timeframe = s.timeframe.ID
		if bar.Session == "" {
			bar.Session = BarSessionRegular
		}
		if aggregate.allNoTrade {
			bar.Status = BarStatusNoTrade
		} else {
			bar.Status = BarStatusOK
		}
		out.Bars[symbol] = bar
		if out.Time.IsZero() || bar.Time.After(out.Time) {
			out.Time = bar.Time
		}
	}
	s.builders = nil
	return out
}

type barAggregate struct {
	bar        Bar
	seen       bool
	allNoTrade bool
}

func (a *barAggregate) add(bar Bar) {
	noTrade := bar.Status == BarStatusNoTrade || bar.isNoTradeBar()
	if !a.seen {
		a.bar = bar
		a.seen = true
		a.allNoTrade = noTrade
		return
	}
	a.allNoTrade = a.allNoTrade && noTrade
	if bar.High > a.bar.High {
		a.bar.High = bar.High
	}
	if bar.Low < a.bar.Low {
		a.bar.Low = bar.Low
	}
	a.bar.Close = bar.Close
	a.bar.Volume += bar.Volume
	a.bar.TradedAmount += bar.TradedAmount
	a.bar.MarketCap = bar.MarketCap
	a.bar.NAV = bar.NAV
	a.bar.Time = bar.Time
}

func bucketKey(timeframe Timeframe, t time.Time) string {
	switch timeframe.ID {
	case Timeframe1Week:
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case Timeframe1Month:
		return t.Format("2006-01")
	default:
		return t.Format(time.DateOnly)
	}
}

func sortedBarSymbols(bars map[string]Bar) []string {
	symbols := make([]string, 0, len(bars))
	for symbol := range bars {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	return symbols
}

func sortedAggregateSymbols(aggregates map[string]*barAggregate) []string {
	symbols := make([]string, 0, len(aggregates))
	for symbol := range aggregates {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	return symbols
}
