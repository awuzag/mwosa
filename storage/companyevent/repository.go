package companyevent

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type Repository struct {
	database *storage.Database
}

type EventInput struct {
	EventType   string
	EventDate   string
	RceptDt     string
	RceptNo     string
	Provider    provider.ProviderID
	Group       provider.GroupID
	Operation   provider.OperationID
	Title       string
	AmountMinor *int64
	ValueText   string
	Raw         any
}

type Event struct {
	CompanyID    int64                `json:"company_id" csv:"company_id"`
	InstrumentID int64                `json:"instrument_id" csv:"instrument_id"`
	EventType    string               `json:"event_type" csv:"event_type"`
	EventDate    string               `json:"event_date" csv:"event_date"`
	RceptDt      string               `json:"rcept_dt" csv:"rcept_dt"`
	RceptNo      string               `json:"rcept_no" csv:"rcept_no"`
	Provider     provider.ProviderID  `json:"provider" csv:"provider"`
	Group        provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation    provider.OperationID `json:"operation" csv:"operation"`
	Title        string               `json:"title" csv:"title"`
	AmountMinor  *int64               `json:"amount_minor,omitempty" csv:"amount_minor"`
	ValueText    string               `json:"value_text" csv:"value_text"`
	Raw          map[string]any       `json:"raw,omitempty" csv:"-"`
}

type Query struct {
	Provider  provider.ProviderID
	EventType string
	From      string
	To        string
	Limit     int
}

type UpsertResult struct {
	EventsWritten int `json:"events_written" csv:"events_written"`
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("company_event_repository").New("company event repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertEvents(ctx context.Context, company companyidentity.InspectResult, events []EventInput) (UpsertResult, error) {
	errb := oops.In("company_event_repository").With("company_id", company.Company.ID, "events", len(events))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("company event upsert requires canonical company")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, errb.Wrapf(err, "begin company event sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	nowMS := time.Now().UTC().UnixMilli()
	result := UpsertResult{}
	for _, event := range events {
		if err := upsertEvent(ctx, tx, company, event, nowMS); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.EventsWritten++
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, errb.Wrapf(err, "commit company event sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r Repository) ListEvents(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Event, error) {
	errb := oops.In("company_event_repository").With("company_id", company.Company.ID, "from", query.From, "to", query.To)
	if company.Company.ID == 0 {
		return nil, errb.New("company event query requires canonical company")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows := make([]storage.CompanyEventV1Row, 0)
	selectQuery := client.NewSelect().
		Model(&rows).
		Where("company_id = ?", company.Company.ID).
		OrderExpr("COALESCE(NULLIF(event_date, ''), rcept_dt) DESC").
		Order("rcept_no DESC")
	if strings.TrimSpace(string(query.Provider)) != "" {
		selectQuery = selectQuery.Where("provider = ?", strings.TrimSpace(string(query.Provider)))
	}
	if strings.TrimSpace(query.EventType) != "" {
		selectQuery = selectQuery.Where("event_type = ?", strings.TrimSpace(query.EventType))
	}
	if strings.TrimSpace(query.From) != "" {
		selectQuery = selectQuery.Where("COALESCE(NULLIF(event_date, ''), rcept_dt) >= ?", strings.TrimSpace(query.From))
	}
	if strings.TrimSpace(query.To) != "" {
		selectQuery = selectQuery.Where("COALESCE(NULLIF(event_date, ''), rcept_dt) <= ?", strings.TrimSpace(query.To))
	}
	if query.Limit > 0 {
		selectQuery = selectQuery.Limit(query.Limit)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "select company events")
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEvent(row))
	}
	return out, nil
}

func upsertEvent(ctx context.Context, tx bun.Tx, company companyidentity.InspectResult, event EventInput, nowMS int64) error {
	rawJSON, err := json.Marshal(event.Raw)
	if err != nil {
		return oops.In("company_event_repository").With("event_type", event.EventType, "rcept_no", event.RceptNo).Wrapf(err, "encode company event raw payload")
	}
	row := storage.CompanyEventV1Row{
		CompanyID:     company.Company.ID,
		InstrumentID:  issuerInstrumentID(company.Instruments),
		EventType:     strings.TrimSpace(event.EventType),
		EventDate:     strings.TrimSpace(event.EventDate),
		RceptDt:       strings.TrimSpace(event.RceptDt),
		RceptNo:       strings.TrimSpace(event.RceptNo),
		Provider:      string(event.Provider),
		ProviderGroup: string(event.Group),
		Operation:     string(event.Operation),
		Title:         strings.TrimSpace(event.Title),
		AmountMinor:   event.AmountMinor,
		ValueText:     strings.TrimSpace(event.ValueText),
		RawJSON:       string(rawJSON),
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if row.RawJSON == "null" {
		row.RawJSON = "{}"
	}
	if row.Provider == "" || row.ProviderGroup == "" || row.Operation == "" || row.EventType == "" || row.RceptNo == "" {
		return oops.In("company_event_repository").With("company_id", row.CompanyID).New("company event missing natural key")
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (company_id, instrument_id, provider, provider_group, operation, event_type, rcept_no, title) DO UPDATE").
		Set("event_date = EXCLUDED.event_date").
		Set("rcept_dt = EXCLUDED.rcept_dt").
		Set("amount_minor = EXCLUDED.amount_minor").
		Set("value_text = EXCLUDED.value_text").
		Set("raw_json = EXCLUDED.raw_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("company_event_repository").With("company_id", row.CompanyID, "event_type", row.EventType, "rcept_no", row.RceptNo).Wrapf(err, "upsert company event")
	}
	return nil
}

func rowToEvent(row storage.CompanyEventV1Row) Event {
	raw := map[string]any{}
	if strings.TrimSpace(row.RawJSON) != "" {
		_ = json.Unmarshal([]byte(row.RawJSON), &raw)
	}
	return Event{
		CompanyID:    row.CompanyID,
		InstrumentID: row.InstrumentID,
		EventType:    row.EventType,
		EventDate:    row.EventDate,
		RceptDt:      row.RceptDt,
		RceptNo:      row.RceptNo,
		Provider:     provider.ProviderID(row.Provider),
		Group:        provider.GroupID(row.ProviderGroup),
		Operation:    provider.OperationID(row.Operation),
		Title:        row.Title,
		AmountMinor:  row.AmountMinor,
		ValueText:    row.ValueText,
		Raw:          raw,
	}
}

func issuerInstrumentID(links []companyidentity.InstrumentLink) int64 {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer {
			return link.InstrumentID
		}
	}
	if len(links) == 0 {
		return 0
	}
	return links[0].InstrumentID
}
