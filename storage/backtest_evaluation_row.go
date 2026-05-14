package storage

import (
	"time"

	"github.com/uptrace/bun"
)

type BacktestExperimentRow struct {
	bun.BaseModel `bun:"table:backtest_experiments,alias:backtest_experiment"`

	ID               string    `bun:"id,pk"`
	Name             string    `bun:"name,notnull"`
	StrategyName     string    `bun:"strategy_name,notnull"`
	BaseRunName      string    `bun:"base_run_name,notnull"`
	SchemaVersion    int       `bun:"schema_version,notnull"`
	SpecJSON         string    `bun:"spec_json,notnull"`
	SpecHash         string    `bun:"spec_hash,notnull"`
	StrategySpecHash string    `bun:"strategy_spec_hash,notnull"`
	DataFrom         string    `bun:"data_from,notnull"`
	DataTo           string    `bun:"data_to,notnull"`
	CreatedAt        time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}

type BacktestExperimentCaseRow struct {
	bun.BaseModel `bun:"table:backtest_experiment_cases,alias:backtest_experiment_case"`

	ID                string    `bun:"id,pk"`
	ExperimentID      string    `bun:"experiment_id,notnull"`
	CaseID            string    `bun:"case_id,notnull"`
	CaseName          string    `bun:"case_name,notnull"`
	RunName           string    `bun:"run_name,notnull"`
	PeriodFrom        string    `bun:"period_from,notnull"`
	PeriodTo          string    `bun:"period_to,notnull"`
	ParameterJSON     string    `bun:"parameter_json,notnull"`
	RegimeTagsJSON    string    `bun:"regime_tags_json,notnull"`
	Status            string    `bun:"status,notnull"`
	PassedConstraints bool      `bun:"passed_constraints,notnull"`
	Rank              int       `bun:"rank,notnull"`
	Objective         string    `bun:"objective,nullzero"`
	ObjectiveValue    float64   `bun:"objective_value,nullzero"`
	ResultHash        string    `bun:"result_hash,notnull"`
	CreatedAt         time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}

type BacktestResultRow struct {
	bun.BaseModel `bun:"table:backtest_results,alias:backtest_result"`

	ID               string    `bun:"id,pk"`
	ExperimentCaseID string    `bun:"experiment_case_id,notnull"`
	ResultJSON       string    `bun:"result_json,notnull"`
	ResultHash       string    `bun:"result_hash,notnull"`
	CreatedAt        time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}

type BacktestMetricSummaryRow struct {
	bun.BaseModel `bun:"table:backtest_metric_summaries,alias:backtest_metric_summary"`

	ID               string    `bun:"id,pk"`
	ExperimentCaseID string    `bun:"experiment_case_id,notnull"`
	Metric           string    `bun:"metric,notnull"`
	Value            float64   `bun:"value,notnull"`
	CreatedAt        time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}

type BacktestWalkForwardStepRow struct {
	bun.BaseModel `bun:"table:backtest_walk_forward_steps,alias:backtest_walk_forward_step"`

	ID                    string    `bun:"id,pk"`
	ExperimentID          string    `bun:"experiment_id,notnull"`
	StepIndex             int       `bun:"step_index,notnull"`
	TrainFrom             string    `bun:"train_from,notnull"`
	TrainTo               string    `bun:"train_to,notnull"`
	TestFrom              string    `bun:"test_from,notnull"`
	TestTo                string    `bun:"test_to,notnull"`
	SelectedParameterJSON string    `bun:"selected_parameter_json,notnull"`
	TrainCaseID           string    `bun:"train_case_id,notnull"`
	TestCaseID            string    `bun:"test_case_id,notnull"`
	TrainObjective        float64   `bun:"train_objective,notnull"`
	ResultHash            string    `bun:"result_hash,notnull"`
	CreatedAt             time.Time `bun:"created_at,notnull,default:CURRENT_TIMESTAMP"`
}
