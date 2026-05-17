package storage

import "github.com/uptrace/bun"

type CompanyFactV1Row struct {
	bun.BaseModel `bun:"table:company_fact_v1,alias:company_fact_v1"`

	ID                             int64  `bun:"id,pk,autoincrement"`
	CompanyID                      int64  `bun:"company_id,notnull"`
	InstrumentID                   int64  `bun:"instrument_id"`
	Provider                       string `bun:"provider,notnull"`
	ProviderGroup                  string `bun:"provider_group,notnull"`
	Operation                      string `bun:"operation,notnull"`
	ProviderCompanyIdentifierType  string `bun:"provider_company_identifier_type,notnull,default:''"`
	ProviderCompanyIdentifierValue string `bun:"provider_company_identifier_value,notnull,default:''"`
	FactType                       string `bun:"fact_type,notnull"`
	FiscalYear                     string `bun:"fiscal_year,notnull,default:''"`
	ReportCode                     string `bun:"report_code,notnull,default:''"`
	RceptNo                        string `bun:"rcept_no,notnull,default:''"`
	FactDate                       string `bun:"fact_date,notnull,default:''"`
	Key                            string `bun:"key,notnull"`
	ValueText                      string `bun:"value_text,notnull,default:''"`
	ValueNumber                    string `bun:"value_number,notnull,default:''"`
	CurrencyCode                   string `bun:"currency_code,notnull,default:''"`
	RawJSON                        string `bun:"raw_json,notnull,default:'{}'"`
	CreatedAtMS                    int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS                    int64  `bun:"updated_at_ms,notnull"`
}
