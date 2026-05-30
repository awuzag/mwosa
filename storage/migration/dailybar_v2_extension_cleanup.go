package migration

import (
	"context"

	migrationcore "github.com/awuzag/mwosa/migration"
	"github.com/samber/oops"
)

const DailyBarV2ExtensionCleanupID = "daily_bar_v2_extension_cleanup"

type DailyBarV2ExtensionCleanupExecutor struct {
	database clientProvider
}

func NewDailyBarV2ExtensionCleanupExecutor(database clientProvider) (DailyBarV2ExtensionCleanupExecutor, error) {
	if database == nil {
		return DailyBarV2ExtensionCleanupExecutor{}, oops.In("daily_bar_v2_extension_cleanup_migration").New("database is nil")
	}
	return DailyBarV2ExtensionCleanupExecutor{database: database}, nil
}

func NewDailyBarV2ExtensionCleanupDefinition(executor migrationcore.Executor) migrationcore.Definition {
	return migrationcore.Definition{
		ID:          DailyBarV2ExtensionCleanupID,
		Name:        "Daily bar v2 extension cleanup",
		Resource:    "daily_bar_extension",
		FromVersion: "2.0.0",
		ToVersion:   "2.0.1",
		Description: "Remove promoted daily bar extension duplicates and redundant extension index",
		Executor:    executor,
	}
}

func (e DailyBarV2ExtensionCleanupExecutor) Apply(ctx context.Context) (int64, error) {
	errb := oops.In("daily_bar_v2_extension_cleanup_migration")
	client, err := e.database.Client(ctx)
	if err != nil {
		return 0, errb.Wrap(err)
	}
	result, err := client.ExecContext(ctx, `
DELETE FROM daily_bar_extension_v2
WHERE key IN ('nav', 'stLstgCnt', 'nPptTotAmt')`)
	if err != nil {
		return 0, errb.Wrapf(err, "delete promoted daily bar extension rows")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errb.Wrapf(err, "read deleted daily bar extension rows")
	}
	if _, err := client.ExecContext(ctx, `DROP INDEX IF EXISTS idx_daily_bar_extension_v2_bar`); err != nil {
		return 0, errb.Wrapf(err, "drop redundant daily bar extension index")
	}
	return rowsAffected, nil
}
