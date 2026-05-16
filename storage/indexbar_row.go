package storage

import (
	"database/sql"

	"github.com/uptrace/bun"
)

const IndexBarV1SchemaVersion = "1.0.0"

type IndexV1Row struct {
	bun.BaseModel `bun:"table:index_v1,alias:index_v1"`

	ID             int64  `bun:"id,pk,autoincrement"`
	Market         string `bun:"market,notnull"`
	IndexCode      string `bun:"index_code,notnull"`
	Name           string `bun:"name,notnull,default:''"`
	Family         string `bun:"family,notnull,default:''"`
	CountryCode    string `bun:"country_code,notnull,default:''"`
	CurrencyCode   string `bun:"currency_code,notnull,default:''"`
	Timezone       string `bun:"timezone,notnull,default:''"`
	IndexType      string `bun:"index_type,notnull,default:''"`
	ExtensionsJSON string `bun:"extensions_json,notnull,default:'{}'"`
	CreatedAtMS    int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS    int64  `bun:"updated_at_ms,notnull"`
}

type IndexSourceV1Row struct {
	bun.BaseModel `bun:"table:index_source_v1,alias:index_source_v1"`

	ID             int64  `bun:"id,pk,autoincrement"`
	Provider       string `bun:"provider,notnull"`
	ProviderGroup  string `bun:"provider_group,notnull"`
	Operation      string `bun:"operation,notnull"`
	ProviderSymbol string `bun:"provider_symbol,notnull"`
	IndexID        int64  `bun:"index_id,notnull"`
	CreatedAtMS    int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS    int64  `bun:"updated_at_ms,notnull"`
}

type IndexBarV1Row struct {
	bun.BaseModel `bun:"table:index_bar_v1,alias:index_bar_v1"`

	SchemaVersion string `bun:"schema_version,notnull"`
	IndexID       int64  `bun:"index_id,pk,notnull"`
	SourceID      int64  `bun:"source_id,pk,notnull"`
	TradingDate   int    `bun:"trading_date,pk,notnull"`

	OpenValue         sql.NullInt64 `bun:"open_value"`
	HighValue         sql.NullInt64 `bun:"high_value"`
	LowValue          sql.NullInt64 `bun:"low_value"`
	CloseValue        sql.NullInt64 `bun:"close_value"`
	ChangeValue       sql.NullInt64 `bun:"change_value"`
	ChangeRateBP      sql.NullInt64 `bun:"change_rate_bp"`
	Volume            sql.NullInt64 `bun:"volume"`
	TradedAmountMinor sql.NullInt64 `bun:"traded_amount_minor"`
	MarketCapMinor    sql.NullInt64 `bun:"market_cap_minor"`

	CreatedAtMS int64 `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64 `bun:"updated_at_ms,notnull"`
}

type IndexBarExtensionV1Row struct {
	bun.BaseModel `bun:"table:index_bar_extension_v1,alias:index_bar_extension_v1"`

	IndexID     int64  `bun:"index_id,pk,notnull"`
	SourceID    int64  `bun:"source_id,pk,notnull"`
	TradingDate int    `bun:"trading_date,pk,notnull"`
	Key         string `bun:"key,pk,notnull"`
	Value       string `bun:"value,notnull"`
}
