package handler

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
)

type Macro struct {
	reader    macroservice.ReadService
	collector macroservice.Service
}

func NewMacro(reader macroservice.ReadService, collector macroservice.Service) Macro {
	return Macro{reader: reader, collector: collector}
}

type ListMacroIndicatorsRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Preset         macrorole.Preset
}

type GetMacroRequest struct {
	IndicatorID string
	From        string
	To          string
}

type SyncMacroRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Target         string
	From           string
	To             string
}

func (h Macro) ListIndicators(ctx context.Context, req ListMacroIndicatorsRequest) (MacroIndicatorsOutput, error) {
	result, err := h.collector.ListIndicators(ctx, macroservice.ListIndicatorsRequest{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Preset:         req.Preset,
	})
	if err != nil {
		return nil, err
	}
	return MacroIndicatorsOutput(result.Indicators), nil
}

func (h Macro) Get(ctx context.Context, req GetMacroRequest) (MacroObservationsOutput, error) {
	result, err := h.reader.GetObservations(ctx, macroservice.GetObservationsRequest{
		IndicatorID: req.IndicatorID,
		From:        req.From,
		To:          req.To,
	})
	if err != nil {
		return MacroObservationsOutput{}, err
	}
	return MacroObservationsOutput{Result: result}, nil
}

func (h Macro) Sync(ctx context.Context, req SyncMacroRequest) (any, error) {
	if req.Target == string(macrorole.PresetKeyStatistics) {
		result, err := h.collector.SyncPreset(ctx, macroservice.SyncPresetRequest{
			ProviderID:     req.ProviderID,
			PreferProvider: req.PreferProvider,
			Preset:         macrorole.PresetKeyStatistics,
		})
		if err != nil {
			return MacroSyncPresetOutput{}, err
		}
		return MacroSyncPresetOutput{Result: result}, nil
	}
	result, err := h.collector.SyncObservations(ctx, macroservice.SyncObservationsRequest{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		IndicatorID:    req.Target,
		From:           req.From,
		To:             req.To,
	})
	if err != nil {
		return MacroSyncObservationsOutput{}, err
	}
	return MacroSyncObservationsOutput{Result: result}, nil
}
