package spec

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type CompatibilitySource interface {
	BuildCompatibility() (provider.Compatibility, error)
}

type CompatibilityBuilder struct {
	compatibility provider.Compatibility
}

func Realtime() CompatibilityBuilder {
	return CompatibilityBuilder{
		compatibility: provider.Compatibility{
			DataLatency:         provider.DataLatencyRealtime,
			CurrentDaySupported: true,
		},
	}
}

func EndOfDay() CompatibilityBuilder {
	return CompatibilityBuilder{
		compatibility: provider.Compatibility{
			DataLatency: provider.DataLatencyEndOfDay,
		},
	}
}

func PreviousBusinessDay() CompatibilityBuilder {
	return CompatibilityBuilder{
		compatibility: provider.Compatibility{
			DataLatency:         provider.DataLatencyPreviousBusinessDay,
			LagBusinessDays:     1,
			CurrentDaySupported: false,
		},
	}
}

func Historical() CompatibilityBuilder {
	return CompatibilityBuilder{
		compatibility: provider.Compatibility{
			DataLatency:         provider.DataLatencyHistorical,
			CurrentDaySupported: false,
		},
	}
}

func (b CompatibilityBuilder) LagBusinessDays(days int) CompatibilityBuilder {
	b.compatibility.LagBusinessDays = days
	return b
}

func (b CompatibilityBuilder) CurrentDay() CompatibilityBuilder {
	b.compatibility.CurrentDaySupported = true
	return b
}

func (b CompatibilityBuilder) NoCurrentDay() CompatibilityBuilder {
	b.compatibility.CurrentDaySupported = false
	return b
}

func (b CompatibilityBuilder) Notes(notes ...string) CompatibilityBuilder {
	b.compatibility.Notes = append(b.compatibility.Notes, notes...)
	return b
}

func (b CompatibilityBuilder) BuildCompatibility() (provider.Compatibility, error) {
	compatibility := b.compatibility
	if err := ValidateCompatibility(compatibility); err != nil {
		return provider.Compatibility{}, err
	}
	return compatibility, nil
}

func (b CompatibilityBuilder) MustBuildCompatibility() provider.Compatibility {
	compatibility, err := b.BuildCompatibility()
	if err != nil {
		panic(err)
	}
	return compatibility
}

func ValidateCompatibility(compatibility provider.Compatibility) error {
	errb := oops.In("provider_spec").With("data_latency", compatibility.DataLatency)
	if compatibility.DataLatency == "" {
		return errb.New("provider compatibility requires data latency")
	}
	if compatibility.LagBusinessDays < 0 {
		return errb.With("lag_business_days", compatibility.LagBusinessDays).New("provider compatibility lag business days must not be negative")
	}

	switch compatibility.DataLatency {
	case provider.DataLatencyRealtime:
		if compatibility.LagBusinessDays != 0 {
			return errb.With("lag_business_days", compatibility.LagBusinessDays).New("realtime compatibility must not have business-day lag")
		}
		if !compatibility.CurrentDaySupported {
			return errb.New("realtime compatibility must support current trading-day data")
		}
	case provider.DataLatencyPreviousBusinessDay:
		if compatibility.LagBusinessDays <= 0 {
			return errb.With("lag_business_days", compatibility.LagBusinessDays).New("previous-business-day compatibility requires positive business-day lag")
		}
		if compatibility.CurrentDaySupported {
			return errb.New("previous-business-day compatibility must not support current trading-day data")
		}
	case provider.DataLatencyEndOfDay, provider.DataLatencyHistorical:
	default:
		return errb.New("provider compatibility has unknown data latency")
	}

	return nil
}
