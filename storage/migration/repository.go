package migration

import (
	"context"
	"database/sql"
	"errors"
	"time"

	migrationcore "github.com/ev3rlit/mwosa/migration"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
)

type Repository struct {
	database *storage.Database
}

var _ migrationcore.Store = (*Repository)(nil)

func NewRepository(database *storage.Database) (migrationcore.Store, error) {
	if database == nil {
		return nil, oops.In("migration_repository").New("migration repository database is nil")
	}
	return &Repository{database: database}, nil
}

func (r *Repository) GetRun(ctx context.Context, id string) (migrationcore.MigrationRun, bool, error) {
	errb := oops.In("migration_repository").With("migration", id)
	client, err := r.database.Client(ctx)
	if err != nil {
		return migrationcore.MigrationRun{}, false, errb.Wrap(err)
	}
	var row storage.MigrationRunRow
	if err := client.NewSelect().
		Model(&row).
		Where("id = ?", id).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return migrationcore.MigrationRun{}, false, nil
		}
		return migrationcore.MigrationRun{}, false, errb.Wrapf(err, "select migration run")
	}
	return migrationRunFromRow(row), true, nil
}

func (r *Repository) RecordApplied(ctx context.Context, definition migrationcore.Definition, rowsMigrated int64, appliedAt time.Time) (migrationcore.MigrationRun, error) {
	errb := oops.In("migration_repository").With("migration", definition.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return migrationcore.MigrationRun{}, errb.Wrap(err)
	}
	appliedAtMS := appliedAt.UTC().UnixMilli()
	row := storage.MigrationRunRow{
		ID:           definition.ID,
		Name:         definition.Name,
		Resource:     definition.Resource,
		FromVersion:  definition.FromVersion,
		ToVersion:    definition.ToVersion,
		Status:       migrationcore.StatusApplied,
		RowsMigrated: rowsMigrated,
		AppliedAtMS:  appliedAtMS,
		UpdatedAtMS:  appliedAtMS,
	}
	if _, err := client.NewInsert().
		Model(&row).
		On("CONFLICT (id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("resource = EXCLUDED.resource").
		Set("from_version = EXCLUDED.from_version").
		Set("to_version = EXCLUDED.to_version").
		Set("status = EXCLUDED.status").
		Set("rows_migrated = EXCLUDED.rows_migrated").
		Set("applied_at_ms = EXCLUDED.applied_at_ms").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return migrationcore.MigrationRun{}, errb.Wrapf(err, "record migration run")
	}
	return migrationRunFromRow(row), nil
}

func migrationRunFromRow(row storage.MigrationRunRow) migrationcore.MigrationRun {
	return migrationcore.MigrationRun{
		ID:           row.ID,
		Name:         row.Name,
		Resource:     row.Resource,
		FromVersion:  row.FromVersion,
		ToVersion:    row.ToVersion,
		Status:       row.Status,
		RowsMigrated: row.RowsMigrated,
		AppliedAtMS:  row.AppliedAtMS,
		UpdatedAtMS:  row.UpdatedAtMS,
	}
}
