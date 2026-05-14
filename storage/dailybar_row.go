package storage

import (
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

const DailyBarV2SchemaVersion = "2.0.0"

type DailyBarV1Row struct {
	bun.BaseModel `bun:"table:daily_bar,alias:daily_bar"`

	ID                               int64     `bun:"id,pk,autoincrement"`
	Provider                         string    `bun:"provider,notnull"`
	ProviderGroup                    string    `bun:"provider_group,notnull"`
	Operation                        string    `bun:"operation,notnull"`
	Market                           string    `bun:"market,notnull"`
	SecurityType                     string    `bun:"security_type,notnull"`
	Symbol                           string    `bun:"symbol,notnull"`
	ISIN                             string    `bun:"isin,notnull,default:''"`
	Name                             string    `bun:"name,notnull,default:''"`
	TradingDate                      string    `bun:"trading_date,notnull"`
	Currency                         string    `bun:"currency,notnull,default:''"`
	OpeningPrice                     string    `bun:"opening_price,notnull,default:''"`
	HighestPrice                     string    `bun:"highest_price,notnull,default:''"`
	LowestPrice                      string    `bun:"lowest_price,notnull,default:''"`
	ClosingPrice                     string    `bun:"closing_price,notnull,default:''"`
	PriceChangeFromPreviousClose     string    `bun:"price_change_from_previous_close,notnull,default:''"`
	PriceChangeRateFromPreviousClose string    `bun:"price_change_rate_from_previous_close,notnull,default:''"`
	TradedVolume                     string    `bun:"traded_volume,notnull,default:''"`
	TradedAmount                     string    `bun:"traded_amount,notnull,default:''"`
	MarketCapitalization             string    `bun:"market_capitalization,notnull,default:''"`
	ExtensionsJSON                   string    `bun:"extensions_json,notnull,default:'{}'"`
	CreatedAt                        time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt                        time.Time `bun:"updated_at,notnull,default:CURRENT_TIMESTAMP"`
}

type DailyBarRow = DailyBarV1Row

type MarketV2Row struct {
	bun.BaseModel `bun:"table:market_v2,alias:market_v2"`

	ID                 int64  `bun:"id,pk,autoincrement"`
	Code               string `bun:"code,notnull"`
	Timezone           string `bun:"timezone,notnull"`
	RegularOpenMinute  int    `bun:"regular_open_minute,notnull"`
	RegularCloseMinute int    `bun:"regular_close_minute,notnull"`
	CreatedAtMS        int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS        int64  `bun:"updated_at_ms,notnull"`
}

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

type ProviderSourceV2Row struct {
	bun.BaseModel `bun:"table:provider_source_v2,alias:provider_source_v2"`

	ID            int64  `bun:"id,pk,autoincrement"`
	Provider      string `bun:"provider,notnull"`
	ProviderGroup string `bun:"provider_group,notnull"`
	Operation     string `bun:"operation,notnull"`
	CreatedAtMS   int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS   int64  `bun:"updated_at_ms,notnull"`
}

type DailyBarV2Row struct {
	bun.BaseModel `bun:"table:daily_bar_v2,alias:daily_bar_v2"`

	SchemaVersion string `bun:"schema_version,notnull"`
	InstrumentID  int64  `bun:"instrument_id,pk,notnull"`
	SourceID      int64  `bun:"source_id,pk,notnull"`
	TradingDate   int    `bun:"trading_date,pk,notnull"`

	OpenPrice    sql.NullInt64 `bun:"open_price"`
	HighPrice    sql.NullInt64 `bun:"high_price"`
	LowPrice     sql.NullInt64 `bun:"low_price"`
	ClosePrice   sql.NullInt64 `bun:"close_price"`
	ChangePrice  sql.NullInt64 `bun:"change_price"`
	ChangeRateBP sql.NullInt64 `bun:"change_rate_bp"`

	Volume            sql.NullInt64 `bun:"volume"`
	TradedAmountMinor sql.NullInt64 `bun:"traded_amount_minor"`
	MarketCapMinor    sql.NullInt64 `bun:"market_cap_minor"`

	NAVPrice      sql.NullInt64 `bun:"nav_price"`
	ListedShares  sql.NullInt64 `bun:"listed_shares"`
	NetAssetMinor sql.NullInt64 `bun:"net_asset_minor"`

	CreatedAtMS int64 `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64 `bun:"updated_at_ms,notnull"`
}

type DailyBarExtensionV2Row struct {
	bun.BaseModel `bun:"table:daily_bar_extension_v2,alias:daily_bar_extension_v2"`

	InstrumentID int64  `bun:"instrument_id,pk,notnull"`
	SourceID     int64  `bun:"source_id,pk,notnull"`
	TradingDate  int    `bun:"trading_date,pk,notnull"`
	Key          string `bun:"key,pk,notnull"`
	Value        string `bun:"value,notnull"`
}
