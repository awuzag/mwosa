package macro

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type repository struct {
	database *storage.Database
}

var _ macroservice.ReadRepository = (*repository)(nil)
var _ macroservice.WriteRepository = (*repository)(nil)

func NewRepository(database *storage.Database) (macroservice.ReadRepository, macroservice.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("macro_repository").New("macro repository database is nil")
	}
	repo := &repository{database: database}
	return repo, repo, nil
}

func (r *repository) QueryIndicators(ctx context.Context, query macroservice.IndicatorQuery) ([]macrorole.Indicator, error) {
	errb := oops.In("macro_repository").With("provider", query.ProviderID, "preset", query.Preset, "indicator_id", query.IndicatorID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	sqlQuery, args := indicatorSelectSQL(query)
	rows, err := client.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, errb.Wrapf(err, "query macro indicators sqlite")
	}
	defer rows.Close()

	indicators := make([]macrorole.Indicator, 0)
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		indicators = append(indicators, indicator)
	}
	if err := rows.Err(); err != nil {
		return nil, errb.Wrapf(err, "iterate macro indicators sqlite")
	}
	return indicators, nil
}

func (r *repository) QueryObservations(ctx context.Context, query macroservice.ObservationQuery) ([]macrorole.Observation, error) {
	errb := oops.In("macro_repository").With("indicator_id", query.IndicatorID, "from", query.From, "to", query.To)
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	sqlQuery, args := observationSelectSQL(query)
	rows, err := client.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, errb.Wrapf(err, "query macro observations sqlite")
	}
	defer rows.Close()

	observations := make([]macrorole.Observation, 0)
	for rows.Next() {
		observation, err := scanObservation(rows)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, errb.Wrapf(err, "iterate macro observations sqlite")
	}
	return observations, nil
}

func (r *repository) UpsertIndicators(ctx context.Context, indicators []macrorole.Indicator) (macroservice.IndicatorWriteResult, error) {
	errb := oops.In("macro_repository").With("indicators", len(indicators))
	client, err := r.database.Client(ctx)
	if err != nil {
		return macroservice.IndicatorWriteResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return macroservice.IndicatorWriteResult{}, errb.Wrapf(err, "begin macro indicator sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result := macroservice.IndicatorWriteResult{}
	nowMS := time.Now().UTC().UnixMilli()
	for _, indicator := range indicators {
		indicator = normalizeIndicator(indicator)
		indicatorErrb := errb.With("provider", indicator.Provider, "source_code", indicator.SourceCode, "indicator_id", indicator.ID)
		if err := validateIndicator(indicator); err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
		}
		if err := upsertIndicator(ctx, tx, indicator, nowMS); err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
		}
		result.IndicatorsWritten++
		result.RowsAffected++
		if err := upsertSource(ctx, tx, indicator, nowMS); err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
		}
		result.SourcesWritten++
		result.RowsAffected++
		if indicator.ProviderDoc != nil {
			if err := upsertProviderDocument(ctx, tx, indicator, nowMS); err != nil {
				return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
			}
			result.DocumentsWritten++
			result.RowsAffected++
		}
	}

	if err := tx.Commit(); err != nil {
		return macroservice.IndicatorWriteResult{}, errb.Wrapf(err, "commit macro indicator sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r *repository) UpsertObservations(ctx context.Context, observations []macrorole.Observation) (macroservice.ObservationWriteResult, error) {
	errb := oops.In("macro_repository").With("observations", len(observations))
	client, err := r.database.Client(ctx)
	if err != nil {
		return macroservice.ObservationWriteResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return macroservice.ObservationWriteResult{}, errb.Wrapf(err, "begin macro observation sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result := macroservice.ObservationWriteResult{}
	nowMS := time.Now().UTC().UnixMilli()
	for _, observation := range observations {
		observation = normalizeObservation(observation)
		observationErrb := errb.With("indicator_id", observation.IndicatorID, "period", observation.Period, "revision", observation.Revision)
		if err := validateObservation(observation); err != nil {
			return macroservice.ObservationWriteResult{}, observationErrb.Wrap(err)
		}
		row := storage.MacroObservationRow{
			IndicatorID: observation.IndicatorID,
			Period:      observation.Period,
			Revision:    observation.Revision,
			Value:       observation.Value,
			PublishedAt: observation.PublishedAt,
			CollectedAt: observation.CollectedAt,
			CreatedAtMS: nowMS,
			UpdatedAtMS: nowMS,
		}
		if _, err := tx.NewInsert().
			Model(&row).
			On("CONFLICT (indicator_id, period, revision) DO UPDATE").
			Set("value = EXCLUDED.value").
			Set("published_at = EXCLUDED.published_at").
			Set("collected_at = EXCLUDED.collected_at").
			Set("updated_at_ms = EXCLUDED.updated_at_ms").
			Exec(ctx); err != nil {
			return macroservice.ObservationWriteResult{}, observationErrb.Wrapf(err, "upsert macro observation sqlite row")
		}
		result.ObservationsWritten++
		result.RowsAffected++
	}

	if err := tx.Commit(); err != nil {
		return macroservice.ObservationWriteResult{}, errb.Wrapf(err, "commit macro observation sqlite transaction")
	}
	committed = true
	return result, nil
}

func upsertIndicator(ctx context.Context, tx bun.Tx, indicator macrorole.Indicator, nowMS int64) error {
	row := storage.MacroIndicatorRow{
		IndicatorID:  indicator.ID,
		Preset:       string(indicator.Preset),
		Provider:     string(indicator.Provider),
		SourceCode:   indicator.SourceCode,
		Name:         indicator.Name,
		FriendlyName: indicator.FriendlyName,
		Category:     indicator.Category,
		Frequency:    string(indicator.Frequency),
		Unit:         indicator.Unit,
		Scale:        indicator.Scale,
		Active:       indicator.Active,
		CreatedAtMS:  nowMS,
		UpdatedAtMS:  nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (indicator_id) DO UPDATE").
		Set("preset = EXCLUDED.preset").
		Set("provider = EXCLUDED.provider").
		Set("source_code = EXCLUDED.source_code").
		Set("name = EXCLUDED.name").
		Set("friendly_name = EXCLUDED.friendly_name").
		Set("category = EXCLUDED.category").
		Set("frequency = EXCLUDED.frequency").
		Set("unit = EXCLUDED.unit").
		Set("scale = EXCLUDED.scale").
		Set("active = EXCLUDED.active").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("macro_repository").With("indicator_id", indicator.ID).Wrapf(err, "upsert macro indicator sqlite row")
	}
	return nil
}

func upsertSource(ctx context.Context, tx bun.Tx, indicator macrorole.Indicator, nowMS int64) error {
	row := storage.MacroIndicatorSourceRow{
		IndicatorID: indicator.ID,
		Provider:    string(indicator.Provider),
		SourceCode:  indicator.SourceCode,
		SourceName:  indicator.SourceName,
		SourceURL:   indicator.SourceURL,
		CreatedAtMS: nowMS,
		UpdatedAtMS: nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (indicator_id, provider, source_code) DO UPDATE").
		Set("source_name = EXCLUDED.source_name").
		Set("source_url = EXCLUDED.source_url").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("macro_repository").With("indicator_id", indicator.ID, "provider", indicator.Provider, "source_code", indicator.SourceCode).Wrapf(err, "upsert macro indicator source sqlite row")
	}
	return nil
}

func upsertProviderDocument(ctx context.Context, tx bun.Tx, indicator macrorole.Indicator, nowMS int64) error {
	document := indicator.ProviderDoc
	schemaVersion := strings.TrimSpace(document.SchemaVersion)
	if schemaVersion == "" {
		schemaVersion = storage.MacroProviderDocSchemaVersion
	}
	documentJSON, err := encodeDocument(document.Document)
	if err != nil {
		return err
	}
	row := storage.MacroIndicatorProviderDocRow{
		IndicatorID:   indicator.ID,
		Provider:      string(indicator.Provider),
		SchemaVersion: schemaVersion,
		DocumentJSON:  documentJSON,
		UpdatedAt:     strings.TrimSpace(document.UpdatedAt),
		UpdatedAtMS:   nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (indicator_id, provider, schema_version) DO UPDATE").
		Set("document_json = EXCLUDED.document_json").
		Set("updated_at = EXCLUDED.updated_at").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("macro_repository").With("indicator_id", indicator.ID, "provider", indicator.Provider, "schema_version", schemaVersion).Wrapf(err, "upsert macro indicator provider document sqlite row")
	}
	return nil
}

func indicatorSelectSQL(query macroservice.IndicatorQuery) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`SELECT
	i.indicator_id,
	i.preset,
	i.provider,
	i.source_code,
	COALESCE(s.source_name, ''),
	COALESCE(s.source_url, ''),
	i.name,
	i.friendly_name,
	i.category,
	i.frequency,
	i.unit,
	i.scale,
	i.active
FROM macro_indicator AS i
LEFT JOIN macro_indicator_source AS s
	ON s.indicator_id = i.indicator_id
	AND s.provider = i.provider
	AND s.source_code = i.source_code
WHERE 1 = 1`)
	args := make([]any, 0)
	if query.ProviderID != "" {
		builder.WriteString(" AND i.provider = ?")
		args = append(args, string(query.ProviderID))
	}
	if query.Preset != "" {
		builder.WriteString(" AND i.preset = ?")
		args = append(args, string(query.Preset))
	}
	if query.IndicatorID != "" {
		builder.WriteString(" AND i.indicator_id = ?")
		args = append(args, strings.TrimSpace(query.IndicatorID))
	}
	builder.WriteString(" ORDER BY i.provider, i.preset, i.category, i.indicator_id")
	return builder.String(), args
}

func observationSelectSQL(query macroservice.ObservationQuery) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`SELECT
	o.indicator_id,
	COALESCE(i.provider, ''),
	COALESCE(i.source_code, ''),
	o.period,
	o.value,
	o.published_at,
	o.collected_at,
	o.revision
FROM macro_observation AS o
LEFT JOIN macro_indicator AS i ON i.indicator_id = o.indicator_id
WHERE o.indicator_id = ?`)
	args := []any{strings.TrimSpace(query.IndicatorID)}
	if query.From != "" {
		builder.WriteString(" AND o.period >= ?")
		args = append(args, strings.TrimSpace(query.From))
	}
	if query.To != "" {
		builder.WriteString(" AND o.period <= ?")
		args = append(args, strings.TrimSpace(query.To))
	}
	builder.WriteString(" ORDER BY o.period, o.revision")
	return builder.String(), args
}

func scanIndicator(rows *sql.Rows) (macrorole.Indicator, error) {
	var indicator macrorole.Indicator
	var providerID string
	var group string
	var operation string
	var preset string
	var frequency string
	if err := rows.Scan(
		&indicator.ID,
		&preset,
		&providerID,
		&indicator.SourceCode,
		&indicator.SourceName,
		&indicator.SourceURL,
		&indicator.Name,
		&indicator.FriendlyName,
		&indicator.Category,
		&frequency,
		&indicator.Unit,
		&indicator.Scale,
		&indicator.Active,
	); err != nil {
		return macrorole.Indicator{}, oops.In("macro_repository").Wrapf(err, "scan macro indicator row")
	}
	indicator.Preset = macrorole.Preset(preset)
	indicator.Provider = provider.ProviderID(providerID)
	indicator.Group = provider.GroupID(group)
	indicator.Operation = provider.OperationID(operation)
	indicator.Frequency = macrorole.Frequency(frequency)
	return indicator, nil
}

func scanObservation(rows *sql.Rows) (macrorole.Observation, error) {
	var observation macrorole.Observation
	var providerID string
	if err := rows.Scan(
		&observation.IndicatorID,
		&providerID,
		&observation.SourceCode,
		&observation.Period,
		&observation.Value,
		&observation.PublishedAt,
		&observation.CollectedAt,
		&observation.Revision,
	); err != nil {
		return macrorole.Observation{}, oops.In("macro_repository").Wrapf(err, "scan macro observation row")
	}
	observation.Provider = provider.ProviderID(providerID)
	return observation, nil
}

func normalizeIndicator(indicator macrorole.Indicator) macrorole.Indicator {
	indicator.ID = strings.TrimSpace(indicator.ID)
	indicator.Provider = provider.ProviderID(strings.TrimSpace(string(indicator.Provider)))
	indicator.Group = provider.GroupID(strings.TrimSpace(string(indicator.Group)))
	indicator.Operation = provider.OperationID(strings.TrimSpace(string(indicator.Operation)))
	indicator.SourceCode = strings.TrimSpace(indicator.SourceCode)
	indicator.SourceName = strings.TrimSpace(indicator.SourceName)
	indicator.SourceURL = strings.TrimSpace(indicator.SourceURL)
	indicator.Name = strings.TrimSpace(indicator.Name)
	indicator.FriendlyName = strings.TrimSpace(indicator.FriendlyName)
	indicator.Category = strings.TrimSpace(indicator.Category)
	indicator.Frequency = macrorole.Frequency(strings.TrimSpace(string(indicator.Frequency)))
	indicator.Unit = strings.TrimSpace(indicator.Unit)
	indicator.Scale = strings.TrimSpace(indicator.Scale)
	return indicator
}

func normalizeObservation(observation macrorole.Observation) macrorole.Observation {
	observation.IndicatorID = strings.TrimSpace(observation.IndicatorID)
	observation.Provider = provider.ProviderID(strings.TrimSpace(string(observation.Provider)))
	observation.SourceCode = strings.TrimSpace(observation.SourceCode)
	observation.Period = strings.TrimSpace(observation.Period)
	observation.Value = strings.TrimSpace(observation.Value)
	observation.PublishedAt = strings.TrimSpace(observation.PublishedAt)
	observation.CollectedAt = strings.TrimSpace(observation.CollectedAt)
	return observation
}

func validateIndicator(indicator macrorole.Indicator) error {
	if indicator.ID == "" || indicator.Provider == "" || indicator.SourceCode == "" || indicator.Name == "" {
		return oops.In("macro_repository").With(
			"indicator_id", indicator.ID,
			"provider", indicator.Provider,
			"source_code", indicator.SourceCode,
			"name", indicator.Name,
		).New("macro indicator missing sqlite key")
	}
	return nil
}

func validateObservation(observation macrorole.Observation) error {
	if observation.IndicatorID == "" || observation.Period == "" || observation.Value == "" || observation.CollectedAt == "" {
		return oops.In("macro_repository").With(
			"indicator_id", observation.IndicatorID,
			"period", observation.Period,
			"value", observation.Value,
			"collected_at", observation.CollectedAt,
		).New("macro observation missing sqlite key")
	}
	return nil
}

func encodeDocument(document map[string]any) (string, error) {
	if len(document) == 0 {
		return "{}", nil
	}
	bytes, err := json.Marshal(document)
	if err != nil {
		return "", oops.In("macro_repository").Wrapf(err, "encode macro provider document")
	}
	if !json.Valid(bytes) {
		return "", oops.In("macro_repository").New("macro provider document is not valid JSON")
	}
	return string(bytes), nil
}
