package aggregate

import (
	"context"
	"encoding/json"
	"time"
)

type RunStatus string

const (
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

type Aggregate struct {
	ID              string     `json:"id" csv:"id"`
	Name            string     `json:"name" csv:"name"`
	ActiveVersionID string     `json:"active_version_id" csv:"active_version_id"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" csv:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty" csv:"archived_at"`
}

type Version struct {
	ID          string          `json:"id" csv:"id"`
	AggregateID string          `json:"aggregate_id" csv:"aggregate_id"`
	Version     int             `json:"version" csv:"version"`
	YAMLText    string          `json:"yaml_text,omitempty" csv:"-"`
	SpecJSON    json.RawMessage `json:"spec_json,omitempty" csv:"-"`
	SpecHash    string          `json:"spec_hash" csv:"spec_hash"`
	CreatedAt   time.Time       `json:"created_at" csv:"created_at"`
	Note        string          `json:"note,omitempty" csv:"note"`
}

type Detail struct {
	Aggregate     Aggregate `json:"aggregate"`
	ActiveVersion Version   `json:"active_version"`
	Versions      []Version `json:"versions,omitempty"`
}

type VersionRef struct {
	Version  string
	SpecHash string
}

type Run struct {
	ID                 string          `json:"id" csv:"id"`
	Alias              string          `json:"alias,omitempty" csv:"alias"`
	AggregateID        string          `json:"aggregate_id" csv:"aggregate_id"`
	AggregateVersionID string          `json:"aggregate_version_id" csv:"aggregate_version_id"`
	AggregateName      string          `json:"aggregate_name" csv:"aggregate_name"`
	Version            int             `json:"version" csv:"version"`
	SpecHash           string          `json:"spec_hash" csv:"spec_hash"`
	ParamsJSON         json.RawMessage `json:"params,omitempty" csv:"-"`
	StagesJSON         json.RawMessage `json:"stages,omitempty" csv:"-"`
	PipelineJSON       json.RawMessage `json:"pipeline,omitempty" csv:"-"`
	StartedAt          time.Time       `json:"started_at" csv:"started_at"`
	FinishedAt         *time.Time      `json:"finished_at,omitempty" csv:"finished_at"`
	Status             RunStatus       `json:"status" csv:"status"`
	ResultCount        int             `json:"result_count" csv:"result_count"`
	ResultHash         string          `json:"result_hash" csv:"result_hash"`
	ResultSizeBytes    int64           `json:"result_size_bytes" csv:"result_size_bytes"`
	SummaryJSON        json.RawMessage `json:"summary,omitempty" csv:"-"`
	ErrorMessage       string          `json:"error_message,omitempty" csv:"error_message"`
}

type RunItem struct {
	ID          string          `json:"id" csv:"id"`
	RunID       string          `json:"run_id" csv:"run_id"`
	Ordinal     int             `json:"ordinal" csv:"ordinal"`
	PayloadJSON json.RawMessage `json:"payload" csv:"-"`
}

type RunDetail struct {
	Run       Run       `json:"run"`
	Aggregate Aggregate `json:"aggregate"`
	Version   Version   `json:"version"`
	Items     []RunItem `json:"items"`
}

type RunHistoryFilter struct {
	Name   string
	Status RunStatus
	Limit  int
}

type Repository interface {
	CreateAggregateWithVersion(ctx context.Context, aggregate Aggregate, version Version) (Detail, error)
	ListAggregates(ctx context.Context) ([]Detail, error)
	GetAggregate(ctx context.Context, name string) (Detail, error)
	GetAggregateVersion(ctx context.Context, name string, ref VersionRef) (Detail, error)
	ListAggregateVersions(ctx context.Context, name string) ([]Version, error)
	AddAggregateVersion(ctx context.Context, name string, version Version, now time.Time) (Detail, error)
	ArchiveAggregate(ctx context.Context, name string, archivedAt time.Time) error
	HasRunAlias(ctx context.Context, alias string) (bool, error)
	CreateRun(ctx context.Context, run Run, items []RunItem) (RunDetail, error)
	ListRuns(ctx context.Context, filter RunHistoryFilter) ([]Run, error)
	GetRun(ctx context.Context, ref string, limit int) (RunDetail, error)
}
