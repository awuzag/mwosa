package storage

import (
	"time"

	"github.com/uptrace/bun"
)

type BacktestRunRow struct {
	bun.BaseModel `bun:"table:backtest_runs,alias:backtest_run"`

	ID                       string    `bun:"id,pk"`
	RunName                  string    `bun:"run_name,notnull"`
	StrategyName             string    `bun:"strategy_name,notnull"`
	Market                   string    `bun:"market,notnull"`
	Timeframe                string    `bun:"timeframe,notnull"`
	PeriodFrom               string    `bun:"period_from,notnull"`
	PeriodTo                 string    `bun:"period_to,notnull"`
	StrategyHash             string    `bun:"strategy_hash,notnull"`
	RunHash                  string    `bun:"run_hash,notnull"`
	EngineVersion            string    `bun:"engine_version,notnull,default:''"`
	IndicatorRegistryVersion string    `bun:"indicator_registry_version,notnull,default:''"`
	MetricRegistryVersion    string    `bun:"metric_registry_version,notnull,default:''"`
	DataFingerprint          string    `bun:"data_fingerprint,notnull"`
	ResultHash               string    `bun:"result_hash,notnull"`
	ResultJSON               string    `bun:"result_json,notnull"`
	MetricsJSON              string    `bun:"metrics_json,notnull"`
	CreatedAt                time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}
