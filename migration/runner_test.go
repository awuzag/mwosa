package migration

import (
	"context"
	"testing"
	"time"
)

func TestRunnerStatusAndApplyRegisteredMigrations(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	executor := &fakeExecutor{rowsMigrated: 2}
	runner, err := NewRunner(store, []Definition{{
		ID:          "daily_bar_v1_to_v2",
		Name:        "Daily bar v1 to v2",
		Resource:    "daily_bar",
		FromVersion: "1",
		ToVersion:   "2.0.0",
		Executor:    executor,
	}})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	status, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.Migrations) != 1 {
		t.Fatalf("status migrations len = %d, want 1", len(status.Migrations))
	}
	if status.Migrations[0].ID != "daily_bar_v1_to_v2" || status.Migrations[0].Status != StatusPending {
		t.Fatalf("status migration = %+v, want pending daily_bar_v1_to_v2", status.Migrations[0])
	}

	applied, err := runner.Apply(ctx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", executor.calls)
	}
	if len(applied.Applied) != 1 || applied.Applied[0].RowsMigrated != 2 {
		t.Fatalf("applied = %+v, want one applied run with rows=2", applied)
	}

	skipped, err := runner.Apply(ctx)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls after skip = %d, want 1", executor.calls)
	}
	if len(skipped.Skipped) != 1 || skipped.Skipped[0].Status != StatusSkipped {
		t.Fatalf("skipped = %+v, want one skipped run", skipped)
	}
}

type fakeStore struct {
	runs map[string]MigrationRun
}

func newFakeStore() *fakeStore {
	return &fakeStore{runs: map[string]MigrationRun{}}
}

func (s *fakeStore) GetRun(_ context.Context, id string) (MigrationRun, bool, error) {
	run, ok := s.runs[id]
	return run, ok, nil
}

func (s *fakeStore) RecordApplied(_ context.Context, definition Definition, rowsMigrated int64, appliedAt time.Time) (MigrationRun, error) {
	run := MigrationRun{
		ID:           definition.ID,
		Name:         definition.Name,
		Resource:     definition.Resource,
		FromVersion:  definition.FromVersion,
		ToVersion:    definition.ToVersion,
		Status:       StatusApplied,
		RowsMigrated: rowsMigrated,
		AppliedAtMS:  appliedAt.UnixMilli(),
		UpdatedAtMS:  appliedAt.UnixMilli(),
	}
	s.runs[definition.ID] = run
	return run, nil
}

type fakeExecutor struct {
	rowsMigrated int64
	calls        int
}

func (e *fakeExecutor) Apply(context.Context) (int64, error) {
	e.calls++
	return e.rowsMigrated, nil
}
