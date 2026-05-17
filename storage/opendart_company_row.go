package storage

import "github.com/uptrace/bun"

type OpenDARTCompanyRow struct {
	bun.BaseModel `bun:"table:opendart_companies,alias:opendart_companies"`

	ID          int64  `bun:"id,pk,autoincrement"`
	CorpCode    string `bun:"corp_code,notnull"`
	CorpName    string `bun:"corp_name,notnull"`
	CorpEngName string `bun:"corp_eng_name,notnull,default:''"`
	StockCode   string `bun:"stock_code,notnull,default:''"`
	ModifyDate  string `bun:"modify_date,notnull,default:''"`
	Listed      bool   `bun:"listed,notnull,default:false"`
	CreatedAtMS int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS int64  `bun:"updated_at_ms,notnull"`
}
