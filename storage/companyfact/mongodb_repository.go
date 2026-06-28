package companyfact

import (
	"context"
	"encoding/json"
	"sort"
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

type factMongoDocument struct {
	ID                             string                  `bson:"_id"`
	SchemaVersion                  string                  `bson:"schema_version"`
	Revision                       int64                   `bson:"revision"`
	CreatedAt                      time.Time               `bson:"created_at"`
	UpdatedAt                      time.Time               `bson:"updated_at"`
	FactID                         int64                   `bson:"fact_id"`
	CompanyID                      int64                   `bson:"company_id"`
	InstrumentID                   int64                   `bson:"instrument_id"`
	Company                        companySnapshotDocument `bson:"company"`
	Instrument                     instrumentSnapshot      `bson:"instrument"`
	Provider                       string                  `bson:"provider"`
	ProviderGroup                  string                  `bson:"provider_group"`
	Operation                      string                  `bson:"operation"`
	ProviderCompanyIdentifierType  string                  `bson:"provider_company_identifier_type,omitempty"`
	ProviderCompanyIdentifierValue string                  `bson:"provider_company_identifier_value,omitempty"`
	FactType                       string                  `bson:"fact_type"`
	FiscalYear                     string                  `bson:"fiscal_year,omitempty"`
	ReportCode                     string                  `bson:"report_code,omitempty"`
	RceptNo                        string                  `bson:"rcept_no,omitempty"`
	FactDate                       string                  `bson:"fact_date,omitempty"`
	Key                            string                  `bson:"key"`
	ValueText                      string                  `bson:"value_text,omitempty"`
	ValueNumber                    string                  `bson:"value_number,omitempty"`
	CurrencyCode                   string                  `bson:"currency_code,omitempty"`
	Raw                            map[string]any          `bson:"raw"`
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
		return MongoRepository{}, oops.In("company_fact_repository").New("mongodb database is nil")
	}
	return MongoRepository{collection: database.Collection("company_facts")}, nil
}

func (r MongoRepository) UpsertFacts(ctx context.Context, company companyidentity.InspectResult, facts []FactInput) (UpsertResult, error) {
	errb := oops.In("company_fact_repository").With("backend", "mongodb", "company_id", company.Company.ID, "facts", len(facts))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("company fact upsert requires canonical company")
	}
	result := UpsertResult{}
	for _, fact := range facts {
		document, err := factToMongoDocument(company, fact)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if err := r.replaceDocument(ctx, document); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.FactsWritten++
	}
	return result, nil
}

func (r MongoRepository) ListFacts(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Fact, error) {
	errb := oops.In("company_fact_repository").With("backend", "mongodb", "company_id", company.Company.ID, "fact_type", query.FactType, "fiscal_year", query.FiscalYear)
	if company.Company.ID == 0 {
		return nil, errb.New("company fact query requires canonical company")
	}
	filter := bson.D{{Key: "company_id", Value: company.Company.ID}}
	if strings.TrimSpace(query.FactType) != "" {
		filter = append(filter, bson.E{Key: "fact_type", Value: strings.TrimSpace(query.FactType)})
	}
	if strings.TrimSpace(query.FiscalYear) != "" {
		filter = append(filter, bson.E{Key: "fiscal_year", Value: strings.TrimSpace(query.FiscalYear)})
	}
	if strings.TrimSpace(query.From) != "" {
		filter = append(filter, bson.E{Key: "fact_date", Value: bson.D{{Key: "$gte", Value: strings.TrimSpace(query.From)}}})
	}
	if strings.TrimSpace(query.To) != "" {
		filter = appendFactDateUpperBound(filter, strings.TrimSpace(query.To))
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}, {Key: "fact_date", Value: -1}, {Key: "key", Value: 1}})
	if query.Limit > 0 && query.WindowYears <= 0 {
		findOptions.SetLimit(int64(query.Limit))
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errb.Wrapf(err, "select company facts")
	}
	defer cursor.Close(ctx)
	var documents []factMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode company facts")
	}
	documents = filterMongoWindowYears(documents, query.WindowYears)
	if query.Limit > 0 && len(documents) > query.Limit {
		documents = documents[:query.Limit]
	}
	out := make([]Fact, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToFact(document))
	}
	return out, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next factMongoDocument) error {
	var current factMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("company_fact_repository").With("backend", "mongodb", "id", next.ID).Wrapf(insertErr, "insert company fact mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("company_fact_repository").With("backend", "mongodb", "id", next.ID).Wrapf(err, "read company fact mongodb document")
	}
	next.FactID = current.FactID
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("company_fact_repository").With("backend", "mongodb", "id", next.ID, "revision", current.Revision).Wrapf(err, "replace company fact mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("company_facts", next.ID, current.Revision)
	}
	return nil
}

func factToMongoDocument(company companyidentity.InspectResult, fact FactInput) (factMongoDocument, error) {
	identifierType, identifierValue := fact.ProviderCompanyIdentifierType, fact.ProviderCompanyIdentifierValue
	if strings.TrimSpace(identifierType) == "" || strings.TrimSpace(identifierValue) == "" {
		identifierType, identifierValue = providerCompanyIdentifier(company.Identifiers)
	}
	raw, err := rawDocument(fact.Raw)
	if err != nil {
		return factMongoDocument{}, oops.In("company_fact_repository").With("fact_type", fact.FactType, "key", fact.Key).Wrapf(err, "encode company fact raw payload")
	}
	providerID := string(fact.Provider)
	providerGroup := string(fact.Group)
	operation := string(fact.Operation)
	factType := strings.TrimSpace(fact.FactType)
	key := strings.TrimSpace(fact.Key)
	if providerID == "" || providerGroup == "" || operation == "" || factType == "" || key == "" {
		return factMongoDocument{}, oops.In("company_fact_repository").With("company_id", company.Company.ID).New("company fact missing natural key")
	}
	instrument := issuerInstrument(company.Instruments)
	now := storagemongodb.ISOTimeNow()
	id := companyFactMongoID(company.Company.ID, instrument.InstrumentID, providerID, providerGroup, operation, factType, fact.FiscalYear, fact.ReportCode, fact.RceptNo, key)
	return factMongoDocument{
		ID:                             id,
		SchemaVersion:                  storagemongodb.SchemaVersion1,
		Revision:                       1,
		CreatedAt:                      now,
		UpdatedAt:                      now,
		FactID:                         stableFactID(id),
		CompanyID:                      company.Company.ID,
		InstrumentID:                   instrument.InstrumentID,
		Company:                        companySnapshotFromInspect(company),
		Instrument:                     instrumentSnapshotFromLink(instrument),
		Provider:                       providerID,
		ProviderGroup:                  providerGroup,
		Operation:                      operation,
		ProviderCompanyIdentifierType:  strings.TrimSpace(identifierType),
		ProviderCompanyIdentifierValue: strings.TrimSpace(identifierValue),
		FactType:                       factType,
		FiscalYear:                     strings.TrimSpace(fact.FiscalYear),
		ReportCode:                     strings.TrimSpace(fact.ReportCode),
		RceptNo:                        strings.TrimSpace(fact.RceptNo),
		FactDate:                       strings.TrimSpace(fact.FactDate),
		Key:                            key,
		ValueText:                      strings.TrimSpace(fact.ValueText),
		ValueNumber:                    strings.TrimSpace(fact.ValueNumber),
		CurrencyCode:                   strings.TrimSpace(fact.CurrencyCode),
		Raw:                            raw,
	}, nil
}

func mongoDocumentToFact(document factMongoDocument) Fact {
	return Fact{
		CompanyID:                      document.CompanyID,
		InstrumentID:                   document.InstrumentID,
		Provider:                       provider.ProviderID(document.Provider),
		Group:                          provider.GroupID(document.ProviderGroup),
		Operation:                      provider.OperationID(document.Operation),
		ProviderCompanyIdentifierType:  document.ProviderCompanyIdentifierType,
		ProviderCompanyIdentifierValue: document.ProviderCompanyIdentifierValue,
		FactType:                       document.FactType,
		FiscalYear:                     document.FiscalYear,
		ReportCode:                     document.ReportCode,
		RceptNo:                        document.RceptNo,
		FactDate:                       document.FactDate,
		Key:                            document.Key,
		ValueText:                      document.ValueText,
		ValueNumber:                    document.ValueNumber,
		CurrencyCode:                   document.CurrencyCode,
		Raw:                            cloneMap(document.Raw),
	}
}

func appendFactDateUpperBound(filter bson.D, to string) bson.D {
	for i, item := range filter {
		if item.Key != "fact_date" {
			continue
		}
		rangeFilter, ok := item.Value.(bson.D)
		if !ok {
			break
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$lte", Value: to})
		filter[i].Value = rangeFilter
		return filter
	}
	return append(filter, bson.E{Key: "fact_date", Value: bson.D{{Key: "$lte", Value: to}}})
}

func filterMongoWindowYears(documents []factMongoDocument, windowYears int) []factMongoDocument {
	if windowYears <= 0 || len(documents) == 0 {
		return documents
	}
	years := make(map[string]struct{})
	for _, document := range documents {
		year := strings.TrimSpace(document.FiscalYear)
		if year != "" {
			years[year] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(years))
	for year := range years {
		ordered = append(ordered, year)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ordered)))
	if len(ordered) > windowYears {
		ordered = ordered[:windowYears]
	}
	allowed := make(map[string]struct{}, len(ordered))
	for _, year := range ordered {
		allowed[year] = struct{}{}
	}
	filtered := documents[:0]
	for _, document := range documents {
		if _, ok := allowed[document.FiscalYear]; ok {
			filtered = append(filtered, document)
		}
	}
	return filtered
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

func companyFactMongoID(parts ...any) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, "company_facts")
	for _, part := range parts {
		values = append(values, strings.TrimSpace(toString(part)))
	}
	return strings.Join(values, ":")
}

func stableFactID(value string) int64 {
	var out int64
	for _, b := range []byte(value) {
		out = out*131 + int64(b)
		out &= 0x7fffffffffffffff
	}
	if out == 0 {
		return 1
	}
	return out
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
