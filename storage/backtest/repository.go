package backtest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	backtestservice "github.com/ev3rlit/mwosa/service/backtest"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type repository struct {
	database *storage.Database
}

var _ backtestservice.StrategyRepository = (*repository)(nil)

func NewRepository(database *storage.Database) (backtestservice.StrategyRepository, error) {
	if database == nil {
		return nil, oops.In("backtest_strategy_repository").New("backtest strategy repository database is nil")
	}
	return &repository{database: database}, nil
}

func (r *repository) CreateStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	strategyRow := strategyToRow(in)
	if _, err := tx.NewInsert().Model(&strategyRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create backtest strategy sqlite row")
	}
	versionRow := versionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create backtest strategy version sqlite row")
	}
	if err := tx.Commit(); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit backtest strategy transaction")
	}
	committed = true

	return detailFromRows(&strategyRow, &versionRow)
}

func (r *repository) ListStrategies(ctx context.Context) ([]backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository")
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	var rows []storage.BacktestStrategyRow
	if err := client.NewSelect().
		Model(&rows).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list backtest strategy sqlite rows")
	}

	details := make([]backtestservice.SavedStrategyDetail, 0, len(rows))
	for i := range rows {
		version, err := r.getVersionByID(ctx, rows[i].ActiveVersionID)
		if err != nil {
			return nil, errb.With("strategy_id", rows[i].ID, "version_id", rows[i].ActiveVersionID).Wrapf(err, "load active backtest strategy version")
		}
		detail, err := detailFromRows(&rows[i], &version)
		if err != nil {
			return nil, errb.With("strategy_id", rows[i].ID).Wrap(err)
		}
		details = append(details, detail)
	}
	return details, nil
}

func (r *repository) GetStrategy(ctx context.Context, name string) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}

	var row storage.BacktestStrategyRow
	if err := client.NewSelect().
		Model(&row).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedStrategyDetail{}, errb.Errorf("backtest strategy not found: %s", name)
		}
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "get backtest strategy sqlite row")
	}
	version, err := r.getVersionByID(ctx, row.ActiveVersionID)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.With("strategy_id", row.ID, "version_id", row.ActiveVersionID).Wrapf(err, "load active backtest strategy version")
	}
	return detailFromRows(&row, &version)
}

func (r *repository) AddStrategyVersion(ctx context.Context, name string, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", name, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin update backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var strategyRow storage.BacktestStrategyRow
	if err := tx.NewSelect().
		Model(&strategyRow).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backtestservice.SavedStrategyDetail{}, errb.Errorf("backtest strategy not found: %s", name)
		}
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "load backtest strategy for version update")
	}
	version.StrategyID = strategyRow.ID
	versionRow := versionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create updated backtest strategy version")
	}
	if _, err := tx.NewUpdate().
		Model((*storage.BacktestStrategyRow)(nil)).
		Set("active_version_id = ?", version.ID).
		Set("updated_at = ?", now).
		Where("id = ?", strategyRow.ID).
		Exec(ctx); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "update active backtest strategy version")
	}
	strategyRow.ActiveVersionID = version.ID
	strategyRow.UpdatedAt = now
	if err := tx.Commit(); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit update backtest strategy transaction")
	}
	committed = true

	return detailFromRows(&strategyRow, &versionRow)
}

func (r *repository) UpsertStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "begin upsert backtest strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var strategyRow storage.BacktestStrategyRow
	err = tx.NewSelect().
		Model(&strategyRow).
		Where("name = ?", in.Name).
		Where("deleted_at IS NULL").
		Scan(ctx)
	switch {
	case err == nil:
		activeVersion, err := r.getVersionByIDTx(ctx, tx, strategyRow.ActiveVersionID)
		if err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.With("active_version_id", strategyRow.ActiveVersionID).Wrapf(err, "load active backtest strategy version")
		}
		version.StrategyID = strategyRow.ID
		version.Version = activeVersion.Version + 1
		versionRow := versionToRow(version)
		if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy version")
		}
		if _, err := tx.NewUpdate().
			Model((*storage.BacktestStrategyRow)(nil)).
			Set("active_version_id = ?", version.ID).
			Set("updated_at = ?", now).
			Where("id = ?", strategyRow.ID).
			Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "update active backtest strategy version")
		}
		strategyRow.ActiveVersionID = version.ID
		strategyRow.UpdatedAt = now
		if err := tx.Commit(); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit upsert backtest strategy transaction")
		}
		committed = true
		return detailFromRows(&strategyRow, &versionRow)
	case errors.Is(err, sql.ErrNoRows):
		version.Version = 1
		version.StrategyID = in.ID
		in.ActiveVersionID = version.ID
		strategyRow = strategyToRow(in)
		versionRow := versionToRow(version)
		if _, err := tx.NewInsert().Model(&strategyRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy sqlite row")
		}
		if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy version sqlite row")
		}
		if err := tx.Commit(); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "commit upsert backtest strategy transaction")
		}
		committed = true
		return detailFromRows(&strategyRow, &versionRow)
	default:
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "load backtest strategy for upsert")
	}
}

func (r *repository) DeleteStrategy(ctx context.Context, name string, deletedAt time.Time) error {
	errb := oops.In("backtest_strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return errb.Wrap(err)
	}
	result, err := client.NewUpdate().
		Model((*storage.BacktestStrategyRow)(nil)).
		Set("deleted_at = ?", deletedAt).
		Set("updated_at = ?", deletedAt).
		Where("name = ?", name).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return errb.Wrapf(err, "delete backtest strategy sqlite row")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return errb.Wrapf(err, "read delete backtest strategy affected rows")
	}
	if affected == 0 {
		return errb.Errorf("backtest strategy not found: %s", name)
	}
	return nil
}

func (r *repository) getVersionByID(ctx context.Context, id string) (storage.BacktestStrategyVersionRow, error) {
	client, err := r.database.Client(ctx)
	if err != nil {
		return storage.BacktestStrategyVersionRow{}, err
	}
	return r.getVersionByIDTx(ctx, client, id)
}

type versionSelector interface {
	NewSelect() *bun.SelectQuery
}

func (r *repository) getVersionByIDTx(ctx context.Context, client versionSelector, id string) (storage.BacktestStrategyVersionRow, error) {
	var row storage.BacktestStrategyVersionRow
	if err := client.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		return storage.BacktestStrategyVersionRow{}, err
	}
	return row, nil
}

func strategyToRow(in backtestservice.SavedStrategy) storage.BacktestStrategyRow {
	return storage.BacktestStrategyRow{
		ID:              in.ID,
		Name:            in.Name,
		ActiveVersionID: in.ActiveVersionID,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		DeletedAt:       in.DeletedAt,
	}
}

func versionToRow(in backtestservice.SavedStrategyVersion) storage.BacktestStrategyVersionRow {
	return storage.BacktestStrategyVersionRow{
		ID:            in.ID,
		StrategyID:    in.StrategyID,
		Version:       in.Version,
		SchemaVersion: in.SchemaVersion,
		SpecJSON:      string(in.SpecJSON),
		SpecHash:      in.SpecHash,
		CreatedAt:     in.CreatedAt,
	}
}

func strategyFromRow(row *storage.BacktestStrategyRow) backtestservice.SavedStrategy {
	return backtestservice.SavedStrategy{
		ID:              row.ID,
		Name:            row.Name,
		ActiveVersionID: row.ActiveVersionID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       row.DeletedAt,
	}
}

func versionFromRow(row *storage.BacktestStrategyVersionRow) backtestservice.SavedStrategyVersion {
	return backtestservice.SavedStrategyVersion{
		ID:            row.ID,
		StrategyID:    row.StrategyID,
		Version:       row.Version,
		SchemaVersion: row.SchemaVersion,
		SpecJSON:      json.RawMessage(row.SpecJSON),
		SpecHash:      row.SpecHash,
		CreatedAt:     row.CreatedAt,
	}
}

func detailFromRows(strategy *storage.BacktestStrategyRow, version *storage.BacktestStrategyVersionRow) (backtestservice.SavedStrategyDetail, error) {
	detail := backtestservice.SavedStrategyDetail{
		Strategy:      strategyFromRow(strategy),
		ActiveVersion: versionFromRow(version),
	}
	if err := json.Unmarshal(detail.ActiveVersion.SpecJSON, &detail.Spec); err != nil {
		return backtestservice.SavedStrategyDetail{}, oops.In("backtest_strategy_repository").With("strategy_id", strategy.ID, "version_id", version.ID).Wrapf(err, "decode canonical backtest strategy spec")
	}
	if detail.Spec.Kind == "" {
		detail.Spec = core.StrategySpec{}
	}
	return detail, nil
}
