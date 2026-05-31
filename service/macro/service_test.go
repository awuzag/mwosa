package macro

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	"github.com/samber/oops"
)

func TestSyncPresetStoresIndicatorsAndDocuments(t *testing.T) {
	reader := &fakeRepository{}
	writer := &fakeRepository{}
	fetcher := &fakeFetcher{
		indicators: []macrorole.Indicator{macroIndicatorFixture()},
	}
	service, err := NewService(reader, writer, fakeRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.SyncPreset(context.Background(), SyncPresetRequest{
		ProviderID: provider.ProviderECOS,
		Preset:     macrorole.PresetKeyStatistics,
	})
	if err != nil {
		t.Fatalf("SyncPreset error = %v", err)
	}
	if result.ProviderID != provider.ProviderECOS || result.IndicatorsFetched != 1 || result.IndicatorsStored != 1 || result.DocumentsStored != 1 {
		t.Fatalf("result = %+v", result)
	}
	if len(writer.indicators) != 1 || writer.indicators[0].ProviderDoc == nil {
		t.Fatalf("stored indicators = %+v", writer.indicators)
	}
}

func TestGetObservationsRejectsEmptySuccess(t *testing.T) {
	reader := &fakeRepository{}
	service, err := NewReadService(reader)
	if err != nil {
		t.Fatalf("NewReadService error = %v", err)
	}

	_, err = service.GetObservations(context.Background(), GetObservationsRequest{
		IndicatorID: "ecos.base-rate",
		From:        "2024-04",
		To:          "2024-04",
	})
	if err == nil {
		t.Fatal("GetObservations error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "macro observations not found") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestSyncObservationsUsesStoredIndicatorSourceCode(t *testing.T) {
	reader := &fakeRepository{
		indicators: []macrorole.Indicator{macroIndicatorFixture()},
	}
	writer := &fakeRepository{}
	fetcher := &fakeFetcher{
		observations: []macrorole.Observation{{
			IndicatorID: "ecos.base-rate",
			Period:      "2024-04",
			Value:       "3.50",
			CollectedAt: "2024-04-12T00:00:00Z",
		}},
	}
	service, err := NewService(reader, writer, fakeRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.SyncObservations(context.Background(), SyncObservationsRequest{
		ProviderID:  provider.ProviderECOS,
		IndicatorID: "ecos.base-rate",
		From:        "2024-04",
		To:          "2024-04",
	})
	if err != nil {
		t.Fatalf("SyncObservations error = %v", err)
	}
	if result.ObservationsFetched != 1 || result.ObservationsStored != 1 {
		t.Fatalf("result = %+v", result)
	}
	if fetcher.lastObservationInput.SourceCode != "722Y001" {
		t.Fatalf("source code = %q, want 722Y001", fetcher.lastObservationInput.SourceCode)
	}
	if len(writer.observations) != 1 {
		t.Fatalf("stored observations = %+v", writer.observations)
	}
}

func TestUnsupportedPresetReturnsError(t *testing.T) {
	service, err := NewReadService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewReadService error = %v", err)
	}
	_, err = service.ListIndicators(context.Background(), ListIndicatorsRequest{
		Preset: macrorole.Preset("all"),
	})
	if err == nil {
		t.Fatal("ListIndicators error = nil")
	}
	if !strings.Contains(err.Error(), "unsupported macro preset") {
		t.Fatalf("error = %q", err.Error())
	}
}

type fakeRouter struct {
	fetcher macrorole.Fetcher
	err     error
}

func (r fakeRouter) RouteMacro(context.Context, macrorole.RouteInput) (macrorole.Fetcher, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.fetcher, nil
}

type fakeFetcher struct {
	indicators           []macrorole.Indicator
	observations         []macrorole.Observation
	lastObservationInput macrorole.ObservationInput
}

func (f *fakeFetcher) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.MacroProfile().RoleProfile(),
		Impl:    f,
	}
}

func (f *fakeFetcher) MacroProfile() macrorole.Profile {
	return macrorole.Profile{
		Group:      provider.GroupECOSKeyStatistics,
		Operations: []provider.OperationID{provider.OperationECOSKeyStatistics},
		Compatibility: provider.Compatibility{
			DataLatency: provider.DataLatencyHistorical,
		},
	}
}

func (f *fakeFetcher) FetchMacroIndicators(context.Context, macrorole.IndicatorInput) (macrorole.IndicatorResult, error) {
	return macrorole.IndicatorResult{
		Indicators: f.indicators,
		Provider:   provider.Identity{ID: provider.ProviderECOS},
		Group:      provider.GroupECOSKeyStatistics,
		Operation:  provider.OperationECOSKeyStatistics,
		TotalCount: len(f.indicators),
	}, nil
}

func (f *fakeFetcher) FetchMacroObservations(_ context.Context, input macrorole.ObservationInput) (macrorole.ObservationResult, error) {
	f.lastObservationInput = input
	return macrorole.ObservationResult{
		Observations: f.observations,
		Provider:     provider.Identity{ID: provider.ProviderECOS},
		Group:        provider.GroupECOSKeyStatistics,
		Operation:    provider.OperationECOSKeyStatistics,
		TotalCount:   len(f.observations),
	}, nil
}

type fakeRepository struct {
	indicators   []macrorole.Indicator
	observations []macrorole.Observation
	err          error
}

func (r *fakeRepository) QueryIndicators(context.Context, IndicatorQuery) ([]macrorole.Indicator, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]macrorole.Indicator(nil), r.indicators...), nil
}

func (r *fakeRepository) QueryObservations(context.Context, ObservationQuery) ([]macrorole.Observation, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]macrorole.Observation(nil), r.observations...), nil
}

func (r *fakeRepository) UpsertIndicators(_ context.Context, indicators []macrorole.Indicator) (IndicatorWriteResult, error) {
	if r.err != nil {
		return IndicatorWriteResult{}, r.err
	}
	r.indicators = append([]macrorole.Indicator(nil), indicators...)
	result := IndicatorWriteResult{IndicatorsWritten: len(indicators), SourcesWritten: len(indicators), RowsAffected: len(indicators) * 2}
	for _, indicator := range indicators {
		if indicator.ProviderDoc != nil {
			result.DocumentsWritten++
			result.RowsAffected++
		}
	}
	return result, nil
}

func (r *fakeRepository) UpsertObservations(_ context.Context, observations []macrorole.Observation) (ObservationWriteResult, error) {
	if r.err != nil {
		return ObservationWriteResult{}, r.err
	}
	r.observations = append([]macrorole.Observation(nil), observations...)
	return ObservationWriteResult{ObservationsWritten: len(observations), RowsAffected: len(observations)}, nil
}

func macroIndicatorFixture() macrorole.Indicator {
	return macrorole.Indicator{
		ID:           "ecos.base-rate",
		Preset:       macrorole.PresetKeyStatistics,
		Provider:     provider.ProviderECOS,
		SourceCode:   "722Y001",
		Name:         "Bank of Korea base rate",
		FriendlyName: "base rate",
		Category:     "rates",
		Frequency:    macrorole.FrequencyMonthly,
		Unit:         "%",
		Scale:        "percent",
		Active:       true,
		ProviderDoc: &macrorole.ProviderDocument{
			Provider:      provider.ProviderECOS,
			SchemaVersion: "1.0.0",
			Document: map[string]any{
				"stat_code": "722Y001",
			},
		},
	}
}

func TestRouteErrorsAreSurfaced(t *testing.T) {
	service, err := NewService(&fakeRepository{}, &fakeRepository{}, fakeRouter{err: oops.New("route failed")})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}
	_, err = service.SyncPreset(context.Background(), SyncPresetRequest{Preset: macrorole.PresetKeyStatistics})
	if err == nil {
		t.Fatal("SyncPreset error = nil")
	}
	if !strings.Contains(err.Error(), "route failed") {
		t.Fatalf("error = %q", err.Error())
	}
}
