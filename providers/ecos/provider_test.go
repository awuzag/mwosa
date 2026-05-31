package ecos

import (
	"context"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/macro"
)

type fakeClient struct{}

func (fakeClient) FetchKeyStatistics(context.Context) ([]macro.Indicator, error) {
	return []macro.Indicator{{
		ID:           "ecos.base-rate",
		SourceCode:   "722Y001",
		SourceName:   "ECOS key statistics",
		Name:         "Bank of Korea base rate",
		FriendlyName: "base rate",
		Category:     "rates",
		Frequency:    macro.FrequencyMonthly,
		Unit:         "%",
		Scale:        "percent",
		Active:       true,
		ProviderDoc: &macro.ProviderDocument{
			SchemaVersion: "1.0.0",
			Document: map[string]any{
				"stat_code": "722Y001",
				"item_code": "0101000",
			},
		},
	}}, nil
}

func (fakeClient) FetchObservations(context.Context, ObservationRequest) ([]macro.Observation, error) {
	return []macro.Observation{{
		Period:      "2024-04",
		Value:       "3.50",
		PublishedAt: "2024-04-11",
		Revision:    0,
	}}, nil
}

func TestProviderNormalizesFakeClientRecords(t *testing.T) {
	p := NewWithClient(fakeClient{}, func() time.Time {
		return time.Date(2024, 4, 12, 3, 4, 5, 0, time.UTC)
	})

	indicators, err := p.FetchMacroIndicators(context.Background(), macro.IndicatorInput{Preset: macro.PresetKeyStatistics})
	if err != nil {
		t.Fatalf("FetchMacroIndicators error = %v", err)
	}
	if len(indicators.Indicators) != 1 {
		t.Fatalf("indicators len = %d, want 1", len(indicators.Indicators))
	}
	got := indicators.Indicators[0]
	if got.Provider != provider.ProviderECOS || got.Group != provider.GroupECOSKeyStatistics || got.Operation != provider.OperationECOSKeyStatistics || got.Preset != macro.PresetKeyStatistics {
		t.Fatalf("normalized indicator = %+v", got)
	}
	if got.ProviderDoc == nil || got.ProviderDoc.Provider != provider.ProviderECOS || got.ProviderDoc.UpdatedAt != "2024-04-12T03:04:05Z" {
		t.Fatalf("provider doc = %+v", got.ProviderDoc)
	}

	observations, err := p.FetchMacroObservations(context.Background(), macro.ObservationInput{
		IndicatorID: "ecos.base-rate",
		SourceCode:  "722Y001",
		From:        "2024-04",
		To:          "2024-04",
	})
	if err != nil {
		t.Fatalf("FetchMacroObservations error = %v", err)
	}
	obs := observations.Observations[0]
	if obs.IndicatorID != "ecos.base-rate" || obs.Provider != provider.ProviderECOS || obs.SourceCode != "722Y001" || obs.CollectedAt != "2024-04-12T03:04:05Z" {
		t.Fatalf("normalized observation = %+v", obs)
	}
}
