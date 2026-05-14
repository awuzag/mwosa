package storage

import (
	"time"

	"github.com/uptrace/bun"
)

type StrategyRow struct {
	bun.BaseModel `bun:"table:strategies,alias:strategy"`

	ID              string     `bun:"id,pk"`
	Name            string     `bun:"name,notnull,unique"`
	Engine          string     `bun:"engine,notnull"`
	ActiveVersionID string     `bun:"active_version_id,notnull"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:CURRENT_TIMESTAMP"`
	ArchivedAt      *time.Time `bun:"archived_at,nullzero"`
}

type StrategyVersionRow struct {
	bun.BaseModel `bun:"table:strategy_versions,alias:strategy_version"`

	ID                 string    `bun:"id,pk"`
	StrategyID         string    `bun:"strategy_id,notnull"`
	Version            int       `bun:"version,notnull"`
	QueryText          string    `bun:"query_text,notnull"`
	QueryHash          string    `bun:"query_hash,notnull"`
	InputDataset       string    `bun:"input_dataset,notnull"`
	InputSchemaVersion int       `bun:"input_schema_version,notnull"`
	ParamsJSON         string    `bun:"params_json,notnull,default:'{}'"`
	SpecJSON           string    `bun:"spec_json,notnull,default:'{}'"`
	SpecHash           string    `bun:"spec_hash,notnull,default:''"`
	CreatedAt          time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
	Note               string    `bun:"note,notnull,default:''"`
}

type ScreenRunRow struct {
	bun.BaseModel `bun:"table:screen_runs,alias:screen_run"`

	ID                 string     `bun:"id,pk"`
	Alias              string     `bun:"alias,nullzero"`
	StrategyID         string     `bun:"strategy_id,notnull"`
	StrategyVersionID  string     `bun:"strategy_version_id,notnull"`
	QueryHash          string     `bun:"query_hash,notnull"`
	InputDataset       string     `bun:"input_dataset,notnull"`
	InputSchemaVersion int        `bun:"input_schema_version,notnull"`
	ParamsJSON         string     `bun:"params_json,notnull,default:'{}'"`
	DataFrom           string     `bun:"data_from,notnull,default:''"`
	DataTo             string     `bun:"data_to,notnull,default:''"`
	DataAsOf           string     `bun:"data_as_of,notnull,default:''"`
	StartedAt          time.Time  `bun:"started_at,notnull,default:CURRENT_TIMESTAMP"`
	FinishedAt         *time.Time `bun:"finished_at,nullzero"`
	Status             string     `bun:"status,notnull"`
	ResultCount        int        `bun:"result_count,notnull"`
	ResultHash         string     `bun:"result_hash,notnull,default:''"`
	ResultSizeBytes    int64      `bun:"result_size_bytes,notnull"`
	SummaryJSON        string     `bun:"summary_json,notnull,default:'{}'"`
	ErrorMessage       string     `bun:"error_message,notnull,default:''"`
}

type ScreenRunItemRow struct {
	bun.BaseModel `bun:"table:screen_run_items,alias:screen_run_item"`

	ID          string `bun:"id,pk"`
	ScreenRunID string `bun:"screen_run_id,notnull"`
	Ordinal     int    `bun:"ordinal,notnull"`
	Symbol      string `bun:"symbol,notnull,default:''"`
	PayloadJSON string `bun:"payload_json,notnull"`
}
