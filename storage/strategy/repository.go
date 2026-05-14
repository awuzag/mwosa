package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
)

type repository struct {
	database *storage.Database
}

var _ strategyservice.Repository = (*repository)(nil)

func NewRepository(database *storage.Database) (strategyservice.Repository, error) {
	if database == nil {
		return nil, oops.In("strategy_repository").New("strategy repository database is nil")
	}
	return &repository{database: database}, nil
}

func (r *repository) CreateStrategyWithVersion(ctx context.Context, in strategyservice.Strategy, version strategyservice.StrategyVersion) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "begin strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	strategyRow := strategyToRow(in)
	if _, err := tx.NewInsert().Model(&strategyRow).Exec(ctx); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "create strategy sqlite row")
	}
	versionRow := strategyVersionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "create strategy version sqlite row")
	}
	if err := tx.Commit(); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "commit strategy transaction")
	}
	committed = true

	return strategyservice.StrategyDetail{
		Strategy:      strategyFromRow(&strategyRow),
		ActiveVersion: strategyVersionFromRow(&versionRow),
	}, nil
}

func (r *repository) ListStrategies(ctx context.Context) ([]strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository")
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	var rows []storage.StrategyRow
	if err := client.NewSelect().
		Model(&rows).
		Where("archived_at IS NULL").
		Order("name ASC").
		Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list strategy sqlite rows")
	}

	details := make([]strategyservice.StrategyDetail, 0, len(rows))
	for i := range rows {
		version, err := r.getStrategyVersionByID(ctx, rows[i].ActiveVersionID)
		if err != nil {
			return nil, errb.With("strategy_id", rows[i].ID, "version_id", rows[i].ActiveVersionID).Wrapf(err, "load active strategy version")
		}
		details = append(details, strategyservice.StrategyDetail{
			Strategy:      strategyFromRow(&rows[i]),
			ActiveVersion: version,
		})
	}
	return details, nil
}

func (r *repository) GetStrategy(ctx context.Context, name string) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}

	var row storage.StrategyRow
	if err := client.NewSelect().
		Model(&row).
		Where("name = ?", name).
		Where("archived_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return strategyservice.StrategyDetail{}, errb.Errorf("strategy not found: %s", name)
		}
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "get strategy sqlite row")
	}
	version, err := r.getStrategyVersionByID(ctx, row.ActiveVersionID)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.With("strategy_id", row.ID, "version_id", row.ActiveVersionID).Wrapf(err, "load active strategy version")
	}
	return strategyservice.StrategyDetail{
		Strategy:      strategyFromRow(&row),
		ActiveVersion: version,
	}, nil
}

func (r *repository) GetStrategyVersion(ctx context.Context, name string, ref strategyservice.StrategyVersionRef) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("name", name, "version", ref.Version, "spec_hash", ref.SpecHash)
	detail, err := r.GetStrategy(ctx, name)
	if err != nil {
		return strategyservice.StrategyDetail{}, err
	}
	if strings.TrimSpace(ref.Version) == "" && strings.TrimSpace(ref.SpecHash) == "" {
		return detail, nil
	}
	if strings.TrimSpace(ref.Version) == "latest" && strings.TrimSpace(ref.SpecHash) == "" {
		return detail, nil
	}

	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}
	var versionRow storage.StrategyVersionRow
	query := client.NewSelect().
		Model(&versionRow).
		Where("strategy_id = ?", detail.Strategy.ID)
	if strings.TrimSpace(ref.SpecHash) != "" {
		query = query.Where("spec_hash = ?", ref.SpecHash)
	} else {
		version, err := strconv.Atoi(ref.Version)
		if err != nil {
			return strategyservice.StrategyDetail{}, errb.Wrapf(err, "parse strategy version")
		}
		query = query.Where("version = ?", version)
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return strategyservice.StrategyDetail{}, errb.Errorf("strategy version not found: %s", name)
		}
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "get strategy version sqlite row")
	}
	detail.ActiveVersion = strategyVersionFromRow(&versionRow)
	return detail, nil
}

func (r *repository) AddStrategyVersion(ctx context.Context, name string, engine strategyservice.Engine, version strategyservice.StrategyVersion, now time.Time) (strategyservice.StrategyDetail, error) {
	errb := oops.In("strategy_repository").With("name", name, "engine", engine, "version_id", version.ID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "begin update strategy transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var strategyRow storage.StrategyRow
	if err := tx.NewSelect().
		Model(&strategyRow).
		Where("name = ?", name).
		Where("archived_at IS NULL").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return strategyservice.StrategyDetail{}, errb.Errorf("strategy not found: %s", name)
		}
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "load strategy for version update")
	}
	version.StrategyID = strategyRow.ID
	versionRow := strategyVersionToRow(version)
	if _, err := tx.NewInsert().Model(&versionRow).Exec(ctx); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "create updated strategy version")
	}
	if _, err := tx.NewUpdate().
		Model((*storage.StrategyRow)(nil)).
		Set("active_version_id = ?", version.ID).
		Set("engine = ?", string(engine)).
		Set("updated_at = ?", now).
		Where("id = ?", strategyRow.ID).
		Exec(ctx); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "update active strategy version")
	}
	strategyRow.ActiveVersionID = version.ID
	strategyRow.Engine = string(engine)
	strategyRow.UpdatedAt = now
	if err := tx.Commit(); err != nil {
		return strategyservice.StrategyDetail{}, errb.Wrapf(err, "commit update strategy transaction")
	}
	committed = true

	return strategyservice.StrategyDetail{
		Strategy:      strategyFromRow(&strategyRow),
		ActiveVersion: strategyVersionFromRow(&versionRow),
	}, nil
}

func (r *repository) ArchiveStrategy(ctx context.Context, name string, archivedAt time.Time) error {
	errb := oops.In("strategy_repository").With("name", name)
	client, err := r.database.Client(ctx)
	if err != nil {
		return errb.Wrap(err)
	}
	result, err := client.NewUpdate().
		Model((*storage.StrategyRow)(nil)).
		Set("archived_at = ?", archivedAt).
		Set("updated_at = ?", archivedAt).
		Where("name = ?", name).
		Where("archived_at IS NULL").
		Exec(ctx)
	if err != nil {
		return errb.Wrapf(err, "archive strategy sqlite row")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return errb.Wrapf(err, "read archive strategy affected rows")
	}
	if affected == 0 {
		return errb.Errorf("strategy not found: %s", name)
	}
	return nil
}

func (r *repository) CreateScreenRun(ctx context.Context, run strategyservice.ScreenRun, items []strategyservice.ScreenRunItem) (strategyservice.ScreenRunDetail, error) {
	errb := oops.In("strategy_repository").With("screen_run_id", run.ID, "alias", run.Alias, "strategy_id", run.StrategyID)
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "begin screen run transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	runRow := screenRunToRow(run)
	if _, err := tx.NewInsert().Model(&runRow).Exec(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "create screen run sqlite row")
	}
	itemRows := make([]storage.ScreenRunItemRow, 0, len(items))
	for _, item := range items {
		row := screenRunItemToRow(item)
		if _, err := tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return strategyservice.ScreenRunDetail{}, errb.With("ordinal", item.Ordinal).Wrapf(err, "create screen run item sqlite row")
		}
		itemRows = append(itemRows, row)
	}

	var strategyRow storage.StrategyRow
	if err := tx.NewSelect().Model(&strategyRow).Where("id = ?", run.StrategyID).Scan(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy")
	}
	var versionRow storage.StrategyVersionRow
	if err := tx.NewSelect().Model(&versionRow).Where("id = ?", run.StrategyVersionID).Scan(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "load screen run strategy version")
	}
	if err := tx.Commit(); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "commit screen run transaction")
	}
	committed = true

	return screenRunDetailFromRows(&runRow, &strategyRow, &versionRow, itemRows), nil
}

func (r *repository) ListScreenRuns(ctx context.Context, limit int) ([]strategyservice.ScreenRun, error) {
	errb := oops.In("strategy_repository").With("limit", limit)
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	var rows []storage.ScreenRunRow
	if err := client.NewSelect().
		Model(&rows).
		Order("started_at DESC").
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "list screen runs sqlite rows")
	}
	runs := make([]strategyservice.ScreenRun, 0, len(rows))
	for i := range rows {
		runs = append(runs, screenRunFromRow(&rows[i]))
	}
	return runs, nil
}

func (r *repository) GetScreenRun(ctx context.Context, ref string) (strategyservice.ScreenRunDetail, error) {
	errb := oops.In("strategy_repository").With("ref", ref)
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.ScreenRunDetail{}, errb.Wrap(err)
	}

	var runRow storage.ScreenRunRow
	if err := client.NewSelect().
		Model(&runRow).
		Where("id = ? OR alias = ?", ref, ref).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return strategyservice.ScreenRunDetail{}, errb.Errorf("screen run not found: %s", ref)
		}
		return strategyservice.ScreenRunDetail{}, errb.Wrapf(err, "get screen run sqlite row")
	}
	var strategyRow storage.StrategyRow
	if err := client.NewSelect().Model(&strategyRow).Where("id = ?", runRow.StrategyID).Scan(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.With("strategy_id", runRow.StrategyID).Wrapf(err, "load screen run strategy")
	}
	var versionRow storage.StrategyVersionRow
	if err := client.NewSelect().Model(&versionRow).Where("id = ?", runRow.StrategyVersionID).Scan(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.With("strategy_version_id", runRow.StrategyVersionID).Wrapf(err, "load screen run strategy version")
	}
	var itemRows []storage.ScreenRunItemRow
	if err := client.NewSelect().
		Model(&itemRows).
		Where("screen_run_id = ?", runRow.ID).
		Order("ordinal ASC").
		Scan(ctx); err != nil {
		return strategyservice.ScreenRunDetail{}, errb.With("screen_run_id", runRow.ID).Wrapf(err, "load screen run items")
	}
	return screenRunDetailFromRows(&runRow, &strategyRow, &versionRow, itemRows), nil
}

func (r *repository) getStrategyVersionByID(ctx context.Context, id string) (strategyservice.StrategyVersion, error) {
	client, err := r.database.Client(ctx)
	if err != nil {
		return strategyservice.StrategyVersion{}, err
	}
	var row storage.StrategyVersionRow
	if err := client.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		return strategyservice.StrategyVersion{}, err
	}
	return strategyVersionFromRow(&row), nil
}

func strategyToRow(in strategyservice.Strategy) storage.StrategyRow {
	return storage.StrategyRow{
		ID:              in.ID,
		Name:            in.Name,
		Engine:          string(in.Engine),
		ActiveVersionID: in.ActiveVersionID,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		ArchivedAt:      in.ArchivedAt,
	}
}

func strategyVersionToRow(version strategyservice.StrategyVersion) storage.StrategyVersionRow {
	return storage.StrategyVersionRow{
		ID:                 version.ID,
		StrategyID:         version.StrategyID,
		Version:            version.Version,
		QueryText:          version.QueryText,
		QueryHash:          version.QueryHash,
		InputDataset:       version.InputDataset,
		InputSchemaVersion: version.InputSchemaVersion,
		ParamsJSON:         string(normalizeRawMessage(version.ParamsJSON)),
		SpecJSON:           string(normalizeRawMessage(version.SpecJSON)),
		SpecHash:           version.SpecHash,
		CreatedAt:          version.CreatedAt,
		Note:               version.Note,
	}
}

func screenRunToRow(run strategyservice.ScreenRun) storage.ScreenRunRow {
	return storage.ScreenRunRow{
		ID:                 run.ID,
		Alias:              strings.TrimSpace(run.Alias),
		StrategyID:         run.StrategyID,
		StrategyVersionID:  run.StrategyVersionID,
		QueryHash:          run.QueryHash,
		InputDataset:       run.InputDataset,
		InputSchemaVersion: run.InputSchemaVersion,
		ParamsJSON:         string(normalizeRawMessage(run.ParamsJSON)),
		DataFrom:           run.DataFrom,
		DataTo:             run.DataTo,
		DataAsOf:           run.DataAsOf,
		StartedAt:          run.StartedAt,
		FinishedAt:         run.FinishedAt,
		Status:             string(run.Status),
		ResultCount:        run.ResultCount,
		ResultHash:         run.ResultHash,
		ResultSizeBytes:    run.ResultSizeBytes,
		SummaryJSON:        string(normalizeRawMessage(run.SummaryJSON)),
		ErrorMessage:       run.ErrorMessage,
	}
}

func screenRunItemToRow(item strategyservice.ScreenRunItem) storage.ScreenRunItemRow {
	return storage.ScreenRunItemRow{
		ID:          item.ID,
		ScreenRunID: item.ScreenRunID,
		Ordinal:     item.Ordinal,
		Symbol:      item.Symbol,
		PayloadJSON: string(item.PayloadJSON),
	}
}

func normalizeRawMessage(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}

func strategyFromRow(row *storage.StrategyRow) strategyservice.Strategy {
	return strategyservice.Strategy{
		ID:              row.ID,
		Name:            row.Name,
		Engine:          strategyservice.Engine(row.Engine),
		ActiveVersionID: row.ActiveVersionID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		ArchivedAt:      row.ArchivedAt,
	}
}

func strategyVersionFromRow(row *storage.StrategyVersionRow) strategyservice.StrategyVersion {
	specHash := row.SpecHash
	if specHash == "" {
		specHash = row.QueryHash
	}
	return strategyservice.StrategyVersion{
		ID:                 row.ID,
		StrategyID:         row.StrategyID,
		Version:            row.Version,
		QueryText:          row.QueryText,
		QueryHash:          row.QueryHash,
		InputDataset:       row.InputDataset,
		InputSchemaVersion: row.InputSchemaVersion,
		ParamsJSON:         json.RawMessage(row.ParamsJSON),
		SpecJSON:           json.RawMessage(row.SpecJSON),
		SpecHash:           specHash,
		CreatedAt:          row.CreatedAt,
		Note:               row.Note,
	}
}

func screenRunFromRow(row *storage.ScreenRunRow) strategyservice.ScreenRun {
	return strategyservice.ScreenRun{
		ID:                 row.ID,
		Alias:              row.Alias,
		StrategyID:         row.StrategyID,
		StrategyVersionID:  row.StrategyVersionID,
		QueryHash:          row.QueryHash,
		InputDataset:       row.InputDataset,
		InputSchemaVersion: row.InputSchemaVersion,
		ParamsJSON:         json.RawMessage(row.ParamsJSON),
		DataFrom:           row.DataFrom,
		DataTo:             row.DataTo,
		DataAsOf:           row.DataAsOf,
		StartedAt:          row.StartedAt,
		FinishedAt:         row.FinishedAt,
		Status:             strategyservice.ScreenRunStatus(row.Status),
		ResultCount:        row.ResultCount,
		ResultHash:         row.ResultHash,
		ResultSizeBytes:    row.ResultSizeBytes,
		SummaryJSON:        json.RawMessage(row.SummaryJSON),
		ErrorMessage:       row.ErrorMessage,
	}
}

func screenRunItemFromRow(row *storage.ScreenRunItemRow) strategyservice.ScreenRunItem {
	return strategyservice.ScreenRunItem{
		ID:          row.ID,
		ScreenRunID: row.ScreenRunID,
		Ordinal:     row.Ordinal,
		Symbol:      row.Symbol,
		PayloadJSON: json.RawMessage(row.PayloadJSON),
	}
}

func screenRunDetailFromRows(run *storage.ScreenRunRow, strategy *storage.StrategyRow, version *storage.StrategyVersionRow, items []storage.ScreenRunItemRow) strategyservice.ScreenRunDetail {
	detail := strategyservice.ScreenRunDetail{
		Run:             screenRunFromRow(run),
		Strategy:        strategyFromRow(strategy),
		StrategyVersion: strategyVersionFromRow(version),
		Items:           make([]strategyservice.ScreenRunItem, 0, len(items)),
	}
	for i := range items {
		detail.Items = append(detail.Items, screenRunItemFromRow(&items[i]))
	}
	return detail
}
