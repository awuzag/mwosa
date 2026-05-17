package storage

import "github.com/uptrace/bun"

type ValuationSnapshotV1Row struct {
	bun.BaseModel `bun:"table:valuation_snapshot_v1,alias:valuation_snapshot_v1"`

	ID                  int64  `bun:"id,pk,autoincrement"`
	CompanyID           int64  `bun:"company_id,notnull"`
	InstrumentID        int64  `bun:"instrument_id,notnull"`
	AsOfDate            string `bun:"as_of_date,notnull"`
	SourcePriceDate     string `bun:"source_price_date,notnull"`
	MarketCapMinor      *int64 `bun:"market_cap_minor"`
	ClosePriceMinor     *int64 `bun:"close_price_minor"`
	SharesOutstanding   *int64 `bun:"shares_outstanding"`
	PerBP               *int64 `bun:"per_bp"`
	PbrBP               *int64 `bun:"pbr_bp"`
	PsrBP               *int64 `bun:"psr_bp"`
	EpsMinor            *int64 `bun:"eps_minor"`
	BpsMinor            *int64 `bun:"bps_minor"`
	DividendYieldBP     *int64 `bun:"dividend_yield_bp"`
	MetricSourceVersion string `bun:"metric_source_version,notnull"`
	ProvenanceJSON      string `bun:"provenance_json,notnull,default:'{}'"`
	UncomputableJSON    string `bun:"uncomputable_json,notnull,default:'{}'"`
	CreatedAtMS         int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS         int64  `bun:"updated_at_ms,notnull"`
}
