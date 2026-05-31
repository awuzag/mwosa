package storage

import "github.com/uptrace/bun"

const MacroProviderDocSchemaVersion = "1.0.0"

type MacroIndicatorRow struct {
	bun.BaseModel `bun:"table:macro_indicator,alias:macro_indicator"`

	IndicatorID  string `bun:"indicator_id,pk,notnull"`
	Preset       string `bun:"preset,notnull,default:''"`
	Provider     string `bun:"provider,notnull"`
	SourceCode   string `bun:"source_code,notnull"`
	Name         string `bun:"name,notnull"`
	FriendlyName string `bun:"friendly_name,notnull,default:''"`
	Category     string `bun:"category,notnull,default:''"`
	Frequency    string `bun:"frequency,notnull,default:''"`
	Unit         string `bun:"unit,notnull,default:''"`
	Scale        string `bun:"scale,notnull,default:''"`
	Active       bool   `bun:"active,notnull"`
	CreatedAtMS  int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS  int64  `bun:"updated_at_ms,notnull"`
}

type MacroObservationRow struct {
	bun.BaseModel `bun:"table:macro_observation,alias:macro_observation"`

	IndicatorID string `bun:"indicator_id,pk,notnull"`
	Period      string `bun:"period,pk,notnull"`
	Revision    int    `bun:"revision,pk,notnull"`
	Value       string `bun:"value,notnull"`
	PublishedAt string `bun:"published_at,notnull,default:''"`
	CollectedAt string `bun:"collected_at,notnull"`
	CreatedAtMS int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64  `bun:"updated_at_ms,notnull"`
}

type MacroIndicatorSourceRow struct {
	bun.BaseModel `bun:"table:macro_indicator_source,alias:macro_indicator_source"`

	IndicatorID string `bun:"indicator_id,pk,notnull"`
	Provider    string `bun:"provider,pk,notnull"`
	SourceCode  string `bun:"source_code,pk,notnull"`
	SourceName  string `bun:"source_name,notnull,default:''"`
	SourceURL   string `bun:"source_url,notnull,default:''"`
	CreatedAtMS int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64  `bun:"updated_at_ms,notnull"`
}

type MacroIndicatorProviderDocRow struct {
	bun.BaseModel `bun:"table:macro_indicator_provider_doc,alias:macro_indicator_provider_doc"`

	IndicatorID   string `bun:"indicator_id,pk,notnull"`
	Provider      string `bun:"provider,pk,notnull"`
	SchemaVersion string `bun:"schema_version,pk,notnull"`
	DocumentJSON  string `bun:"document_json,notnull,default:'{}'"`
	UpdatedAt     string `bun:"updated_at,notnull,default:''"`
	UpdatedAtMS   int64  `bun:"updated_at_ms,notnull"`
}
