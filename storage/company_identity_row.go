package storage

import "github.com/uptrace/bun"

type CompanyV1Row struct {
	bun.BaseModel `bun:"table:company_v1,alias:company_v1"`

	ID          int64  `bun:"id,pk,autoincrement"`
	Name        string `bun:"name,notnull"`
	LegalName   string `bun:"legal_name,notnull,default:''"`
	EnglishName string `bun:"english_name,notnull,default:''"`
	CountryCode string `bun:"country_code,notnull,default:''"`
	CreatedAtMS int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64  `bun:"updated_at_ms,notnull"`
}

type CompanyIdentifierV1Row struct {
	bun.BaseModel `bun:"table:company_identifier_v1,alias:company_identifier_v1"`

	ID              int64   `bun:"id,pk,autoincrement"`
	CompanyID       int64   `bun:"company_id,notnull"`
	Provider        string  `bun:"provider,notnull,default:''"`
	ProviderGroup   string  `bun:"provider_group,notnull,default:''"`
	Operation       string  `bun:"operation,notnull,default:''"`
	IdentifierType  string  `bun:"identifier_type,notnull"`
	IdentifierValue string  `bun:"identifier_value,notnull"`
	ValidFrom       string  `bun:"valid_from,notnull,default:''"`
	ValidTo         string  `bun:"valid_to,notnull,default:''"`
	PrimaryFlag     bool    `bun:"primary_flag,notnull,default:false"`
	Confidence      float64 `bun:"confidence,notnull,default:1"`
	SourceUpdatedAt string  `bun:"source_updated_at,notnull,default:''"`
	CreatedAtMS     int64   `bun:"created_at_ms,notnull"`
	UpdatedAtMS     int64   `bun:"updated_at_ms,notnull"`
}

type InstrumentCompanyLinkV1Row struct {
	bun.BaseModel `bun:"table:instrument_company_link_v1,alias:instrument_company_link_v1"`

	ID           int64  `bun:"id,pk,autoincrement"`
	InstrumentID int64  `bun:"instrument_id,notnull"`
	CompanyID    int64  `bun:"company_id,notnull"`
	RelationType string `bun:"relation_type,notnull"`
	ValidFrom    string `bun:"valid_from,notnull,default:''"`
	ValidTo      string `bun:"valid_to,notnull,default:''"`
	CreatedAtMS  int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS  int64  `bun:"updated_at_ms,notnull"`
}
