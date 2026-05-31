package handler

import (
	"fmt"

	"github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
)

type MacroIndicatorsOutput []macro.Indicator

func (o MacroIndicatorsOutput) JSONValue() any {
	return []macro.Indicator(o)
}

func (o MacroIndicatorsOutput) NDJSONRows() any {
	return []macro.Indicator(o)
}

func (o MacroIndicatorsOutput) CSVRows() any {
	return []macro.Indicator(o)
}

func (o MacroIndicatorsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, indicator := range o {
		rows = append(rows, []string{
			indicator.ID,
			indicator.Name,
			indicator.FriendlyName,
			indicator.Category,
			string(indicator.Frequency),
			indicator.Unit,
			indicator.Scale,
			fmt.Sprint(indicator.Active),
			string(indicator.Provider),
			indicator.SourceCode,
			indicator.SourceName,
		})
	}
	return []string{"indicator_id", "name", "friendly_name", "category", "frequency", "unit", "scale", "active", "provider", "source_code", "source_name"}, rows
}

type MacroObservationsOutput struct {
	Result macroservice.ObservationsResult
}

func (o MacroObservationsOutput) JSONValue() any {
	return o.Result.Observations
}

func (o MacroObservationsOutput) NDJSONRows() any {
	return o.Result.Observations
}

func (o MacroObservationsOutput) CSVRows() any {
	return o.Result.Observations
}

func (o MacroObservationsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Observations))
	for _, observation := range o.Result.Observations {
		rows = append(rows, []string{
			observation.IndicatorID,
			observation.Period,
			observation.Value,
			observation.PublishedAt,
			observation.CollectedAt,
			fmt.Sprint(observation.Revision),
			string(observation.Provider),
			observation.SourceCode,
		})
	}
	return []string{"indicator_id", "period", "value", "published_at", "collected_at", "revision", "provider", "source_code"}, rows
}

type MacroSyncPresetOutput struct {
	Result macroservice.SyncPresetResult
}

func (o MacroSyncPresetOutput) JSONValue() any {
	return o.Result
}

func (o MacroSyncPresetOutput) NDJSONRows() any {
	return []macroservice.SyncPresetResult{o.Result}
}

func (o MacroSyncPresetOutput) CSVRows() any {
	return []macroservice.SyncPresetResult{o.Result}
}

func (o MacroSyncPresetOutput) TableRows() ([]string, [][]string) {
	return []string{"preset", "provider", "group", "operation", "fetched", "stored", "sources", "documents", "rows_affected"}, [][]string{{
		string(o.Result.Preset),
		string(o.Result.ProviderID),
		string(o.Result.Group),
		string(o.Result.Operation),
		fmt.Sprint(o.Result.IndicatorsFetched),
		fmt.Sprint(o.Result.IndicatorsStored),
		fmt.Sprint(o.Result.SourcesStored),
		fmt.Sprint(o.Result.DocumentsStored),
		fmt.Sprint(o.Result.RowsAffected),
	}}
}

type MacroSyncObservationsOutput struct {
	Result macroservice.SyncObservationsResult
}

func (o MacroSyncObservationsOutput) JSONValue() any {
	return o.Result
}

func (o MacroSyncObservationsOutput) NDJSONRows() any {
	return []macroservice.SyncObservationsResult{o.Result}
}

func (o MacroSyncObservationsOutput) CSVRows() any {
	return []macroservice.SyncObservationsResult{o.Result}
}

func (o MacroSyncObservationsOutput) TableRows() ([]string, [][]string) {
	return []string{"indicator_id", "provider", "group", "operation", "from", "to", "fetched", "stored", "rows_affected"}, [][]string{{
		o.Result.IndicatorID,
		string(o.Result.ProviderID),
		string(o.Result.Group),
		string(o.Result.Operation),
		o.Result.From,
		o.Result.To,
		fmt.Sprint(o.Result.ObservationsFetched),
		fmt.Sprint(o.Result.ObservationsStored),
		fmt.Sprint(o.Result.RowsAffected),
	}}
}
