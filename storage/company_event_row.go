package storage

import "github.com/uptrace/bun"

type CompanyEventV1Row struct {
	bun.BaseModel `bun:"table:company_event_v1,alias:company_event_v1"`

	ID            int64  `bun:"id,pk,autoincrement"`
	CompanyID     int64  `bun:"company_id,notnull"`
	InstrumentID  int64  `bun:"instrument_id"`
	EventType     string `bun:"event_type,notnull"`
	EventDate     string `bun:"event_date,notnull,default:''"`
	RceptDt       string `bun:"rcept_dt,notnull,default:''"`
	RceptNo       string `bun:"rcept_no,notnull,default:''"`
	Provider      string `bun:"provider,notnull"`
	ProviderGroup string `bun:"provider_group,notnull"`
	Operation     string `bun:"operation,notnull"`
	Title         string `bun:"title,notnull,default:''"`
	AmountMinor   *int64 `bun:"amount_minor"`
	ValueText     string `bun:"value_text,notnull,default:''"`
	RawJSON       string `bun:"raw_json,notnull,default:'{}'"`
	CreatedAtMS   int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS   int64  `bun:"updated_at_ms,notnull"`
}
