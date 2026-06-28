package companyevent

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
}

type eventMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	CompanyID    int64                   `bson:"company_id"`
	InstrumentID int64                   `bson:"instrument_id"`
	Company      companySnapshotDocument `bson:"company"`
	Instrument   instrumentSnapshot      `bson:"instrument"`

	EventType     string         `bson:"event_type"`
	EventDate     string         `bson:"event_date,omitempty"`
	RceptDt       string         `bson:"rcept_dt,omitempty"`
	RceptNo       string         `bson:"rcept_no"`
	EffectiveDate string         `bson:"effective_date"`
	Provider      string         `bson:"provider"`
	ProviderGroup string         `bson:"provider_group"`
	Operation     string         `bson:"operation"`
	Title         string         `bson:"title,omitempty"`
	AmountMinor   *int64         `bson:"amount_minor,omitempty"`
	ValueText     string         `bson:"value_text,omitempty"`
	Raw           map[string]any `bson:"raw"`
}

type companySnapshotDocument struct {
	CompanyID   int64  `bson:"company_id"`
	Name        string `bson:"name"`
	LegalName   string `bson:"legal_name,omitempty"`
	EnglishName string `bson:"english_name,omitempty"`
	CountryCode string `bson:"country_code,omitempty"`
}

type instrumentSnapshot struct {
	InstrumentID int64  `bson:"instrument_id"`
	Market       string `bson:"market,omitempty"`
	SecurityType string `bson:"security_type,omitempty"`
	Symbol       string `bson:"symbol,omitempty"`
	Name         string `bson:"name,omitempty"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("company_event_repository").New("mongodb database is nil")
	}
	return MongoRepository{collection: database.Collection("company_events")}, nil
}

func (r MongoRepository) UpsertEvents(ctx context.Context, company companyidentity.InspectResult, events []EventInput) (UpsertResult, error) {
	errb := oops.In("company_event_repository").With("backend", "mongodb", "company_id", company.Company.ID, "events", len(events))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("company event upsert requires canonical company")
	}
	result := UpsertResult{}
	for _, event := range events {
		document, err := eventToMongoDocument(company, event)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if err := r.replaceDocument(ctx, document); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.EventsWritten++
	}
	return result, nil
}

func (r MongoRepository) ListEvents(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Event, error) {
	errb := oops.In("company_event_repository").With("backend", "mongodb", "company_id", company.Company.ID, "from", query.From, "to", query.To)
	if company.Company.ID == 0 {
		return nil, errb.New("company event query requires canonical company")
	}
	filter := bson.D{{Key: "company_id", Value: company.Company.ID}}
	if strings.TrimSpace(string(query.Provider)) != "" {
		filter = append(filter, bson.E{Key: "provider", Value: strings.TrimSpace(string(query.Provider))})
	}
	if strings.TrimSpace(query.EventType) != "" {
		filter = append(filter, bson.E{Key: "event_type", Value: strings.TrimSpace(query.EventType)})
	}
	dateRange := bson.D{}
	if strings.TrimSpace(query.From) != "" {
		dateRange = append(dateRange, bson.E{Key: "$gte", Value: strings.TrimSpace(query.From)})
	}
	if strings.TrimSpace(query.To) != "" {
		dateRange = append(dateRange, bson.E{Key: "$lte", Value: strings.TrimSpace(query.To)})
	}
	if len(dateRange) > 0 {
		filter = append(filter, bson.E{Key: "effective_date", Value: dateRange})
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "effective_date", Value: -1}, {Key: "rcept_no", Value: -1}})
	if query.Limit > 0 {
		findOptions.SetLimit(int64(query.Limit))
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errb.Wrapf(err, "select company events")
	}
	defer cursor.Close(ctx)
	var documents []eventMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode company events")
	}
	out := make([]Event, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToEvent(document))
	}
	return out, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next eventMongoDocument) error {
	var current eventMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("company_event_repository").With("backend", "mongodb", "id", next.ID).Wrapf(insertErr, "insert company event mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("company_event_repository").With("backend", "mongodb", "id", next.ID).Wrapf(err, "read company event mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("company_event_repository").With("backend", "mongodb", "id", next.ID, "revision", current.Revision).Wrapf(err, "replace company event mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("company_events", next.ID, current.Revision)
	}
	return nil
}

func eventToMongoDocument(company companyidentity.InspectResult, event EventInput) (eventMongoDocument, error) {
	raw, err := rawDocument(event.Raw)
	if err != nil {
		return eventMongoDocument{}, oops.In("company_event_repository").With("event_type", event.EventType, "rcept_no", event.RceptNo).Wrapf(err, "encode company event raw payload")
	}
	providerID := string(event.Provider)
	providerGroup := string(event.Group)
	operation := string(event.Operation)
	eventType := strings.TrimSpace(event.EventType)
	rceptNo := strings.TrimSpace(event.RceptNo)
	if providerID == "" || providerGroup == "" || operation == "" || eventType == "" || rceptNo == "" {
		return eventMongoDocument{}, oops.In("company_event_repository").With("company_id", company.Company.ID).New("company event missing natural key")
	}
	instrument := issuerInstrument(company.Instruments)
	eventDate := strings.TrimSpace(event.EventDate)
	rceptDt := strings.TrimSpace(event.RceptDt)
	title := strings.TrimSpace(event.Title)
	now := storagemongodb.ISOTimeNow()
	id := companyEventMongoID(company.Company.ID, instrument.InstrumentID, providerID, providerGroup, operation, eventType, rceptNo, title)
	return eventMongoDocument{
		ID:            id,
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		CompanyID:     company.Company.ID,
		InstrumentID:  instrument.InstrumentID,
		Company:       companySnapshotFromInspect(company),
		Instrument:    instrumentSnapshotFromLink(instrument),
		EventType:     eventType,
		EventDate:     eventDate,
		RceptDt:       rceptDt,
		RceptNo:       rceptNo,
		EffectiveDate: firstNonEmpty(eventDate, rceptDt),
		Provider:      providerID,
		ProviderGroup: providerGroup,
		Operation:     operation,
		Title:         title,
		AmountMinor:   event.AmountMinor,
		ValueText:     strings.TrimSpace(event.ValueText),
		Raw:           raw,
	}, nil
}

func mongoDocumentToEvent(document eventMongoDocument) Event {
	return Event{
		CompanyID:    document.CompanyID,
		InstrumentID: document.InstrumentID,
		EventType:    document.EventType,
		EventDate:    document.EventDate,
		RceptDt:      document.RceptDt,
		RceptNo:      document.RceptNo,
		Provider:     provider.ProviderID(document.Provider),
		Group:        provider.GroupID(document.ProviderGroup),
		Operation:    provider.OperationID(document.Operation),
		Title:        document.Title,
		AmountMinor:  document.AmountMinor,
		ValueText:    document.ValueText,
		Raw:          cloneMap(document.Raw),
	}
}

func rawDocument(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if string(bytes) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(bytes, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func companySnapshotFromInspect(company companyidentity.InspectResult) companySnapshotDocument {
	return companySnapshotDocument{
		CompanyID:   company.Company.ID,
		Name:        company.Company.Name,
		LegalName:   company.Company.LegalName,
		EnglishName: company.Company.EnglishName,
		CountryCode: company.Company.CountryCode,
	}
}

func instrumentSnapshotFromLink(link companyidentity.InstrumentLink) instrumentSnapshot {
	return instrumentSnapshot{
		InstrumentID: link.InstrumentID,
		Market:       string(link.Market),
		SecurityType: string(link.SecurityType),
		Symbol:       link.Symbol,
		Name:         link.Name,
	}
}

func issuerInstrument(links []companyidentity.InstrumentLink) companyidentity.InstrumentLink {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer {
			return link
		}
	}
	if len(links) == 0 {
		return companyidentity.InstrumentLink{}
	}
	return links[0]
}

func companyEventMongoID(parts ...any) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, "company_events")
	for _, part := range parts {
		values = append(values, strings.TrimSpace(toString(part)))
	}
	return strings.Join(values, ":")
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	default:
		bytes, _ := json.Marshal(typed)
		return string(bytes)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	if source == nil {
		return nil
	}
	out := make(map[K]V, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
