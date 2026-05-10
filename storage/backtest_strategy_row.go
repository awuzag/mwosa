package storage

import (
	"time"

	"github.com/uptrace/bun"
)

type BacktestStrategyRow struct {
	bun.BaseModel `bun:"table:backtest_strategies,alias:backtest_strategy"`

	ID              string     `bun:"id,pk"`
	Name            string     `bun:"name,notnull,unique"`
	ActiveVersionID string     `bun:"active_version_id,notnull"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:CURRENT_TIMESTAMP"`
	DeletedAt       *time.Time `bun:"deleted_at,nullzero"`
}

type BacktestStrategyVersionRow struct {
	bun.BaseModel `bun:"table:backtest_strategy_versions,alias:backtest_strategy_version"`

	ID            string    `bun:"id,pk"`
	StrategyID    string    `bun:"strategy_id,notnull"`
	Version       int       `bun:"version,notnull"`
	SchemaVersion int       `bun:"schema_version,notnull"`
	SpecJSON      string    `bun:"spec_json,notnull"`
	SpecHash      string    `bun:"spec_hash,notnull"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}
