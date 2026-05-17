package storage

import "github.com/uptrace/bun"

type FinancialStatementV1Row struct {
	bun.BaseModel `bun:"table:financial_statement_v1,alias:financial_statement_v1"`

	ID                             int64  `bun:"id,pk,autoincrement"`
	CompanyID                      int64  `bun:"company_id,notnull"`
	InstrumentID                   int64  `bun:"instrument_id"`
	Provider                       string `bun:"provider,notnull"`
	ProviderGroup                  string `bun:"provider_group,notnull"`
	Operation                      string `bun:"operation,notnull"`
	ProviderCompanyIdentifierType  string `bun:"provider_company_identifier_type,notnull,default:''"`
	ProviderCompanyIdentifierValue string `bun:"provider_company_identifier_value,notnull,default:''"`
	RceptNo                        string `bun:"rcept_no,notnull,default:''"`
	FiscalYear                     string `bun:"fiscal_year,notnull"`
	FiscalPeriod                   string `bun:"fiscal_period,notnull,default:''"`
	ReportCode                     string `bun:"report_code,notnull,default:''"`
	FsDiv                          string `bun:"fs_div,notnull,default:''"`
	StatementType                  string `bun:"statement_type,notnull"`
	ReportedAt                     string `bun:"reported_at,notnull,default:''"`
	SourcePayloadRef               string `bun:"source_payload_ref,notnull,default:''"`
	CreatedAtMS                    int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS                    int64  `bun:"updated_at_ms,notnull"`
}

type FinancialLineItemV1Row struct {
	bun.BaseModel `bun:"table:financial_line_item_v1,alias:financial_line_item_v1"`

	ID               int64  `bun:"id,pk,autoincrement"`
	StatementID      int64  `bun:"statement_id,notnull"`
	AccountID        string `bun:"account_id,notnull,default:''"`
	AccountName      string `bun:"account_name,notnull"`
	CanonicalAccount string `bun:"canonical_account,notnull,default:''"`
	AmountMinor      *int64 `bun:"amount_minor"`
	CurrencyCode     string `bun:"currency_code,notnull,default:''"`
	Unit             string `bun:"unit,notnull,default:''"`
	RawAmount        string `bun:"raw_amount,notnull,default:''"`
	PeriodName       string `bun:"period_name,notnull,default:''"`
	Ord              int    `bun:"ord,notnull,default:0"`
	ExtensionsJSON   string `bun:"extensions_json,notnull,default:'{}'"`
	CreatedAtMS      int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS      int64  `bun:"updated_at_ms,notnull"`
}
