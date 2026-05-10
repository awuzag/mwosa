package storage

import "github.com/uptrace/bun"

type ProviderRawSnapshotRow struct {
	bun.BaseModel `bun:"table:provider_raw_snapshots,alias:provider_raw_snapshots"`

	ID               int64  `bun:"id,pk,autoincrement"`
	Provider         string `bun:"provider,notnull"`
	ProviderGroup    string `bun:"provider_group,notnull"`
	Operation        string `bun:"operation,notnull"`
	BaseDate         int    `bun:"base_date,notnull"`
	CanonicalSupport string `bun:"canonical_support,notnull,default:''"`
	RowCount         int    `bun:"row_count,notnull"`
	PayloadJSON      string `bun:"payload_json,notnull"`
	CreatedAtMS      int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS      int64  `bun:"updated_at_ms,notnull"`
}
