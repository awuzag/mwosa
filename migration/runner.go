package migration

import (
	"context"
	"time"

	"github.com/samber/oops"
)

const (
	StatusPending = "pending"
	StatusApplied = "applied"
	StatusSkipped = "skipped"
)

type Definition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Resource    string   `json:"resource"`
	FromVersion string   `json:"from_version"`
	ToVersion   string   `json:"to_version"`
	Description string   `json:"description"`
	Executor    Executor `json:"-"`
}

type Executor interface {
	Apply(ctx context.Context) (int64, error)
}

type ExecutorFunc func(ctx context.Context) (int64, error)

func (f ExecutorFunc) Apply(ctx context.Context) (int64, error) {
	return f(ctx)
}

type MigrationRun struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Resource     string `json:"resource"`
	FromVersion  string `json:"from_version"`
	ToVersion    string `json:"to_version"`
	Status       string `json:"status"`
	RowsMigrated int64  `json:"rows_migrated"`
	AppliedAtMS  int64  `json:"applied_at_ms"`
	UpdatedAtMS  int64  `json:"updated_at_ms"`
}

type StatusResult struct {
	Migrations []MigrationRun `json:"migrations"`
}

type ApplyResult struct {
	Applied []MigrationRun `json:"applied"`
	Skipped []MigrationRun `json:"skipped"`
}

type Store interface {
	GetRun(ctx context.Context, id string) (MigrationRun, bool, error)
	RecordApplied(ctx context.Context, definition Definition, rowsMigrated int64, appliedAt time.Time) (MigrationRun, error)
}

type Runner struct {
	store       Store
	definitions []Definition
}

func NewRunner(store Store, definitions []Definition) (Runner, error) {
	if store == nil {
		return Runner{}, oops.In("migration_runner").New("migration store is nil")
	}
	seen := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		if definition.ID == "" {
			return Runner{}, oops.In("migration_runner").New("migration id is empty")
		}
		if seen[definition.ID] {
			return Runner{}, oops.In("migration_runner").With("migration", definition.ID).New("migration id is duplicated")
		}
		if definition.Executor == nil {
			return Runner{}, oops.In("migration_runner").With("migration", definition.ID).New("migration executor is nil")
		}
		seen[definition.ID] = true
	}
	return Runner{
		store:       store,
		definitions: append([]Definition(nil), definitions...),
	}, nil
}

func (r Runner) Status(ctx context.Context) (StatusResult, error) {
	runs := make([]MigrationRun, 0, len(r.definitions))
	for _, definition := range r.definitions {
		run, ok, err := r.store.GetRun(ctx, definition.ID)
		if err != nil {
			return StatusResult{}, oops.In("migration_runner").With("migration", definition.ID).Wrapf(err, "load migration status")
		}
		if !ok {
			run = MigrationRun{
				ID:          definition.ID,
				Name:        definition.Name,
				Resource:    definition.Resource,
				FromVersion: definition.FromVersion,
				ToVersion:   definition.ToVersion,
				Status:      StatusPending,
			}
		}
		runs = append(runs, run)
	}
	return StatusResult{Migrations: runs}, nil
}

func (r Runner) Apply(ctx context.Context) (ApplyResult, error) {
	result := ApplyResult{}
	for _, definition := range r.definitions {
		existing, ok, err := r.store.GetRun(ctx, definition.ID)
		if err != nil {
			return ApplyResult{}, oops.In("migration_runner").With("migration", definition.ID).Wrapf(err, "load migration status")
		}
		if ok && existing.Status == StatusApplied {
			existing.Status = StatusSkipped
			result.Skipped = append(result.Skipped, existing)
			continue
		}

		rowsMigrated, err := definition.Executor.Apply(ctx)
		if err != nil {
			return ApplyResult{}, oops.In("migration_runner").With("migration", definition.ID).Wrapf(err, "apply migration")
		}
		run, err := r.store.RecordApplied(ctx, definition, rowsMigrated, time.Now().UTC())
		if err != nil {
			return ApplyResult{}, oops.In("migration_runner").With("migration", definition.ID).Wrapf(err, "record migration")
		}
		result.Applied = append(result.Applied, run)
	}
	return result, nil
}
