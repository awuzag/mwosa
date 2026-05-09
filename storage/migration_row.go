package storage

import "github.com/uptrace/bun"

type MigrationRunRow struct {
	bun.BaseModel `bun:"table:migration_runs,alias:migration_runs"`

	ID           string `bun:"id,pk,notnull"`
	Name         string `bun:"name,notnull"`
	Resource     string `bun:"resource,notnull"`
	FromVersion  string `bun:"from_version,notnull"`
	ToVersion    string `bun:"to_version,notnull"`
	Status       string `bun:"status,notnull"`
	RowsMigrated int64  `bun:"rows_migrated,notnull"`
	AppliedAtMS  int64  `bun:"applied_at_ms,notnull"`
	UpdatedAtMS  int64  `bun:"updated_at_ms,notnull"`
}
