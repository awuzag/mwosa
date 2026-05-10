package providerraw

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
)

type Repository struct {
	database *storage.Database
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

type WriteResult struct {
	Provider         provider.ProviderID  `json:"provider" csv:"provider"`
	Group            provider.GroupID     `json:"provider_group" csv:"group"`
	Operation        provider.OperationID `json:"api_id" csv:"api_id"`
	BaseDate         string               `json:"base_date" csv:"base_date"`
	CanonicalSupport string               `json:"canonical_support" csv:"canonical_support"`
	RowCount         int                  `json:"row_count" csv:"row_count"`
	RowsAffected     int64                `json:"rows_affected" csv:"rows_affected"`
}

func NewRepository(database *storage.Database) (Repository, error) {
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
