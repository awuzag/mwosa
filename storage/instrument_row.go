package storage

import "github.com/uptrace/bun"

type InstrumentV2Row struct {
	bun.BaseModel `bun:"table:instrument_v2,alias:instrument_v2"`

	ID           int64  `bun:"id,pk,autoincrement"`
	MarketID     int64  `bun:"market_id,notnull"`
	SecurityType string `bun:"security_type,notnull"`
	Symbol       string `bun:"symbol,notnull"`
	ISIN         string `bun:"isin,notnull,default:''"`
	Name         string `bun:"name,notnull,default:''"`
	CurrencyCode string `bun:"currency_code,notnull,default:'KRW'"`
	PriceScale   int    `bun:"price_scale,notnull"`
	CreatedAtMS  int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS  int64  `bun:"updated_at_ms,notnull"`
}

type InstrumentSourceV1Row struct {
	bun.BaseModel `bun:"table:instrument_source_v1,alias:instrument_source_v1"`

	ID             int64  `bun:"id,pk,autoincrement"`
	Provider       string `bun:"provider,notnull"`
	ProviderGroup  string `bun:"provider_group,notnull"`
	Operation      string `bun:"operation,notnull"`
	ProviderSymbol string `bun:"provider_symbol,notnull"`
	InstrumentID   int64  `bun:"instrument_id,notnull"`
	CreatedAtMS    int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS    int64  `bun:"updated_at_ms,notnull"`
}

type InstrumentExtensionV1Row struct {
	bun.BaseModel `bun:"table:instrument_extension_v1,alias:instrument_extension_v1"`

	InstrumentID int64  `bun:"instrument_id,pk,notnull"`
	SourceID     int64  `bun:"source_id,pk,notnull"`
	Key          string `bun:"key,pk,notnull"`
	Value        string `bun:"value,notnull"`
}
