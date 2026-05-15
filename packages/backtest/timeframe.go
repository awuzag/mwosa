package backtest

import (
	"strconv"
	"strings"

	"github.com/samber/oops"
)

const (
	Timeframe1Min     = "1m"
	Timeframe5Min     = "5m"
	Timeframe15Min    = "15m"
	Timeframe30Min    = "30m"
	Timeframe1Hour    = "1h"
	Timeframe1Day     = "1d"
	Timeframe1Week    = "1w"
	Timeframe1Month   = "1mo"
	TimeframeCustomID = "custom"
)

type Timeframe struct {
	ID     string `json:"id"`
	Step   int    `json:"step,omitempty"`
	Unit   string `json:"unit"`
	Custom bool   `json:"custom,omitempty"`
}

func ParseTimeframe(value string) (Timeframe, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	errb := oops.In("backtest_timeframe").With("timeframe", value)
	if normalized == "" {
		return Timeframe{}, errb.New("timeframe is required")
	}
	if normalized == TimeframeCustomID {
		return Timeframe{ID: normalized, Unit: TimeframeCustomID, Custom: true}, nil
	}
	switch {
	case strings.HasSuffix(normalized, "mo"):
		step, err := parseTimeframeStep(normalized, "mo")
		if err != nil {
			return Timeframe{}, errb.Wrap(err)
		}
		if step != 1 {
			return Timeframe{}, errb.New("unsupported monthly timeframe")
		}
		return Timeframe{ID: normalized, Step: step, Unit: "month"}, nil
	case strings.HasSuffix(normalized, "m"):
		step, err := parseTimeframeStep(normalized, "m")
		if err != nil {
			return Timeframe{}, errb.Wrap(err)
		}
		switch step {
		case 1, 5, 15, 30:
			return Timeframe{ID: normalized, Step: step, Unit: "minute"}, nil
		default:
			return Timeframe{}, errb.New("unsupported minute timeframe")
		}
	case strings.HasSuffix(normalized, "h"):
		step, err := parseTimeframeStep(normalized, "h")
		if err != nil {
			return Timeframe{}, errb.Wrap(err)
		}
		if step != 1 {
			return Timeframe{}, errb.New("unsupported hourly timeframe")
		}
		return Timeframe{ID: normalized, Step: step, Unit: "hour"}, nil
	case strings.HasSuffix(normalized, "d"):
		step, err := parseTimeframeStep(normalized, "d")
		if err != nil {
			return Timeframe{}, errb.Wrap(err)
		}
		if step != 1 {
			return Timeframe{}, errb.New("unsupported daily timeframe")
		}
		return Timeframe{ID: normalized, Step: step, Unit: "day"}, nil
	case strings.HasSuffix(normalized, "w"):
		step, err := parseTimeframeStep(normalized, "w")
		if err != nil {
			return Timeframe{}, errb.Wrap(err)
		}
		if step != 1 {
			return Timeframe{}, errb.New("unsupported weekly timeframe")
		}
		return Timeframe{ID: normalized, Step: step, Unit: "week"}, nil
	default:
		return Timeframe{}, errb.New("unsupported timeframe")
	}
}

func MustTimeframeID(value string) (string, error) {
	timeframe, err := ParseTimeframe(value)
	if err != nil {
		return "", err
	}
	return timeframe.ID, nil
}

func (t Timeframe) IsDailyNative() bool {
	return t.ID == "" || t.ID == Timeframe1Day
}

func (t Timeframe) IsDailyResample() bool {
	return t.ID == Timeframe1Week || t.ID == Timeframe1Month
}

func (t Timeframe) IsDailyCompatible() bool {
	return t.IsDailyNative() || t.IsDailyResample()
}

func parseTimeframeStep(value string, suffix string) (int, error) {
	raw := strings.TrimSuffix(value, suffix)
	step, err := strconv.Atoi(raw)
	if err != nil || step <= 0 {
		return 0, oops.In("backtest_timeframe").With("timeframe", value).New("timeframe step must be a positive integer")
	}
	return step, nil
}
