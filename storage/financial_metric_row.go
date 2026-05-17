package storage

import "github.com/uptrace/bun"

type FinancialMetricV1Row struct {
	bun.BaseModel `bun:"table:financial_metric_v1,alias:financial_metric_v1"`

	ID                 int64  `bun:"id,pk,autoincrement"`
	CompanyID          int64  `bun:"company_id,notnull"`
	InstrumentID       int64  `bun:"instrument_id"`
	StatementID        int64  `bun:"statement_id"`
	Metric             string `bun:"metric,notnull"`
	FiscalYear         string `bun:"fiscal_year,notnull"`
	FiscalPeriod       string `bun:"fiscal_period,notnull,default:''"`
	AsOfDate           string `bun:"as_of_date,notnull,default:''"`
	ValueDecimal       string `bun:"value_decimal,notnull,default:''"`
	ValueBP            *int64 `bun:"value_bp"`
	ValueMinor         *int64 `bun:"value_minor"`
	FormulaVersion     string `bun:"formula_version,notnull"`
	ProvenanceJSON     string `bun:"provenance_json,notnull,default:'{}'"`
	UncomputableReason string `bun:"uncomputable_reason,notnull,default:''"`
	CreatedAtMS        int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS        int64  `bun:"updated_at_ms,notnull"`
}
