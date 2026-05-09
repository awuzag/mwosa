package handler

import (
	"context"
	"strconv"

	migrationcore "github.com/ev3rlit/mwosa/migration"
)

type Migration struct {
	runner migrationcore.Runner
}

func NewMigration(runner migrationcore.Runner) Migration {
	return Migration{runner: runner}
}

func (h Migration) Status(ctx context.Context) (MigrationStatusOutput, error) {
	result, err := h.runner.Status(ctx)
	if err != nil {
		return MigrationStatusOutput{}, err
	}
	return MigrationStatusOutput{Result: result}, nil
}

func (h Migration) Apply(ctx context.Context) (MigrationApplyOutput, error) {
	result, err := h.runner.Apply(ctx)
	if err != nil {
		return MigrationApplyOutput{}, err
	}
	return MigrationApplyOutput{Result: result}, nil
}

type MigrationStatusOutput struct {
	Result migrationcore.StatusResult
}

func (o MigrationStatusOutput) JSONValue() any {
	return o.Result
}

func (o MigrationStatusOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Migrations))
	for _, migration := range o.Result.Migrations {
		rows = append(rows, migrationTableRow(migration))
	}
	return migrationTableHeader(), rows
}

type MigrationApplyOutput struct {
	Result migrationcore.ApplyResult
}

func (o MigrationApplyOutput) JSONValue() any {
	return o.Result
}

func (o MigrationApplyOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Applied)+len(o.Result.Skipped))
	for _, migration := range o.Result.Applied {
		rows = append(rows, migrationTableRow(migration))
	}
	for _, migration := range o.Result.Skipped {
		rows = append(rows, migrationTableRow(migration))
	}
	return migrationTableHeader(), rows
}

func migrationTableHeader() []string {
	return []string{"id", "resource", "from", "to", "status", "rows"}
}

func migrationTableRow(run migrationcore.MigrationRun) []string {
	return []string{
		run.ID,
		run.Resource,
		run.FromVersion,
		run.ToVersion,
		run.Status,
		strconv.FormatInt(run.RowsMigrated, 10),
	}
}
