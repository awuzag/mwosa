package providerraw

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
)

type Repository struct {
	database *storage.SQLDatabase
}

type Snapshot struct {
	Provider         provider.ProviderID
	Group            provider.GroupID
	Operation        provider.OperationID
	BaseDate         string
	CanonicalSupport string
	Rows             any
	RowCount         int
}

type Query struct {
	Provider       provider.ProviderID
	Group          provider.GroupID
	Operation      provider.OperationID
	From           string
	To             string
	Limit          int
	IncludePayload bool
}

type WriteResult struct {
	Provider         provider.ProviderID  `json:"provider" csv:"provider"`
	Group            provider.GroupID     `json:"provider_group" csv:"group"`
	Operation        provider.OperationID `json:"api_id" csv:"api_id"`
	BaseDate         string               `json:"base_date" csv:"base_date"`
	CanonicalSupport string               `json:"canonical_support" csv:"canonical_support"`
	RowCount         int                  `json:"row_count" csv:"row_count"`
	RowsAffected     int64                `json:"rows_affected" csv:"rows_affected"`
}

type SnapshotRecord struct {
	ID               int64                `json:"id" csv:"id"`
	Provider         provider.ProviderID  `json:"provider" csv:"provider"`
	Group            provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation        provider.OperationID `json:"operation" csv:"operation"`
	BaseDate         string               `json:"base_date" csv:"base_date"`
	CanonicalSupport string               `json:"canonical_support" csv:"canonical_support"`
	RowCount         int                  `json:"row_count" csv:"row_count"`
	Payload          any                  `json:"payload,omitempty" csv:"-"`
	CreatedAtMS      int64                `json:"created_at_ms" csv:"created_at_ms"`
	UpdatedAtMS      int64                `json:"updated_at_ms" csv:"updated_at_ms"`
}

func NewRepository(database *storage.SQLDatabase) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("provider_raw_repository").New("provider raw repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertSnapshot(ctx context.Context, snapshot Snapshot) (WriteResult, error) {
	errb := oops.In("provider_raw_repository").With(
		"provider", snapshot.Provider,
		"group", snapshot.Group,
		"operation", snapshot.Operation,
		"base_date", snapshot.BaseDate,
	)
	if snapshot.Provider == "" || snapshot.Group == "" || snapshot.Operation == "" {
		return WriteResult{}, errb.New("provider raw snapshot missing natural key")
	}
	baseDate, err := parseBaseDate(snapshot.BaseDate)
	if err != nil {
		return WriteResult{}, errb.Wrap(err)
	}
	payload, err := json.Marshal(snapshot.Rows)
	if err != nil {
		return WriteResult{}, errb.Wrapf(err, "encode provider raw snapshot payload")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return WriteResult{}, errb.Wrap(err)
	}

	nowMS := time.Now().UTC().UnixMilli()
	row := storage.ProviderRawSnapshotRow{
		Provider:         string(snapshot.Provider),
		ProviderGroup:    string(snapshot.Group),
		Operation:        string(snapshot.Operation),
		BaseDate:         baseDate,
		CanonicalSupport: snapshot.CanonicalSupport,
		RowCount:         snapshot.RowCount,
		PayloadJSON:      string(payload),
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
	}
	result, err := client.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation, base_date) DO UPDATE").
		Set("canonical_support = EXCLUDED.canonical_support").
		Set("row_count = EXCLUDED.row_count").
		Set("payload_json = EXCLUDED.payload_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx)
	if err != nil {
		return WriteResult{}, errb.Wrapf(err, "upsert provider raw snapshot sqlite row")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return WriteResult{}, errb.Wrapf(err, "read provider raw snapshot affected rows")
	}
	return WriteResult{
		Provider:         snapshot.Provider,
		Group:            snapshot.Group,
		Operation:        snapshot.Operation,
		BaseDate:         formatBaseDate(baseDate),
		CanonicalSupport: snapshot.CanonicalSupport,
		RowCount:         snapshot.RowCount,
		RowsAffected:     affected,
	}, nil
}

func (r Repository) ListSnapshots(ctx context.Context, query Query) ([]SnapshotRecord, error) {
	errb := oops.In("provider_raw_repository").With(
		"provider", query.Provider,
		"group", query.Group,
		"operation", query.Operation,
		"from", query.From,
		"to", query.To,
	)
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows := make([]storage.ProviderRawSnapshotRow, 0)
	selectQuery := client.NewSelect().
		Model(&rows).
		Order("base_date DESC", "provider ASC", "provider_group ASC", "operation ASC")
	if strings.TrimSpace(string(query.Provider)) != "" {
		selectQuery = selectQuery.Where("provider = ?", strings.TrimSpace(string(query.Provider)))
	}
	if strings.TrimSpace(string(query.Group)) != "" {
		selectQuery = selectQuery.Where("provider_group = ?", strings.TrimSpace(string(query.Group)))
	}
	if strings.TrimSpace(string(query.Operation)) != "" {
		selectQuery = selectQuery.Where("operation = ?", strings.TrimSpace(string(query.Operation)))
	}
	if strings.TrimSpace(query.From) != "" {
		from, err := parseBaseDate(query.From)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		selectQuery = selectQuery.Where("base_date >= ?", from)
	}
	if strings.TrimSpace(query.To) != "" {
		to, err := parseBaseDate(query.To)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		selectQuery = selectQuery.Where("base_date <= ?", to)
	}
	if query.Limit > 0 {
		selectQuery = selectQuery.Limit(query.Limit)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "select provider raw snapshots")
	}
	out := make([]SnapshotRecord, 0, len(rows))
	for _, row := range rows {
		record, err := rowToSnapshotRecord(row, query.IncludePayload)
		if err != nil {
			return nil, errb.With("id", row.ID).Wrap(err)
		}
		out = append(out, record)
	}
	return out, nil
}

func rowToSnapshotRecord(row storage.ProviderRawSnapshotRow, includePayload bool) (SnapshotRecord, error) {
	var payload any
	if includePayload {
		if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
			return SnapshotRecord{}, oops.In("provider_raw_repository").With("id", row.ID).Wrapf(err, "decode provider raw snapshot payload")
		}
	}
	return SnapshotRecord{
		ID:               row.ID,
		Provider:         provider.ProviderID(row.Provider),
		Group:            provider.GroupID(row.ProviderGroup),
		Operation:        provider.OperationID(row.Operation),
		BaseDate:         formatBaseDate(row.BaseDate),
		CanonicalSupport: row.CanonicalSupport,
		RowCount:         row.RowCount,
		Payload:          payload,
		CreatedAtMS:      row.CreatedAtMS,
		UpdatedAtMS:      row.UpdatedAtMS,
	}, nil
}

func parseBaseDate(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 8 {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, oops.In("provider_raw_repository").With("base_date", value).Wrapf(err, "parse base date")
		}
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return 0, oops.In("provider_raw_repository").With("base_date", value).Wrapf(err, "parse base date")
	}
	return parsed.Year()*10000 + int(parsed.Month())*100 + parsed.Day(), nil
}

func formatBaseDate(value int) string {
	year := value / 10000
	month := value / 100 % 100
	day := value % 100
	return strconv.Itoa(year) + "-" + twoDigits(month) + "-" + twoDigits(day)
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
