package instrument

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	coreinstrument "github.com/awuzag/mwosa/providers/core/instrument"
	instrumentservice "github.com/awuzag/mwosa/service/instrument"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	collection *mongo.Collection
}

type instrumentMongoDocument struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	SchemaVersion string        `bson:"schema_version"`
	Revision      int64         `bson:"revision"`
	CreatedAt     time.Time     `bson:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at"`

	InstrumentKey string                  `bson:"instrument_key"`
	MarketKey     string                  `bson:"market_key"`
	SecurityType  string                  `bson:"security_type"`
	Symbol        string                  `bson:"symbol"`
	ISIN          string                  `bson:"isin,omitempty"`
	Name          string                  `bson:"name,omitempty"`
	ExchangeCode  string                  `bson:"exchange_code,omitempty"`
	CountryCode   string                  `bson:"country_code,omitempty"`
	Timezone      string                  `bson:"timezone,omitempty"`
	Sources       []instrumentMongoSource `bson:"sources"`
	Extensions    map[string]string       `bson:"extensions,omitempty"`
}

type instrumentMongoSource struct {
	Provider       string `bson:"provider"`
	ProviderGroup  string `bson:"provider_group"`
	Operation      string `bson:"operation"`
	ProviderSymbol string `bson:"provider_symbol"`
}

var _ instrumentservice.Repository = (*mongoRepository)(nil)

func NewMongoRepository(database *mongo.Database) (instrumentservice.Repository, error) {
	if database == nil {
		return nil, oops.In("instrument_repository").New("mongodb database is nil")
	}
	return &mongoRepository{collection: database.Collection("instruments")}, nil
}

func (r *mongoRepository) UpsertInstruments(ctx context.Context, instruments []coreinstrument.Instrument) (instrumentservice.WriteResult, error) {
	errb := oops.In("instrument_repository").With("backend", "mongodb", "instruments", len(instruments))
	for _, item := range instruments {
		if err := validateInstrumentKey(item); err != nil {
			return instrumentservice.WriteResult{}, errb.Wrap(err)
		}
		document := instrumentToMongoDocument(item)
		if err := r.replaceDocument(ctx, document); err != nil {
			return instrumentservice.WriteResult{}, errb.With("instrument_key", document.InstrumentKey).Wrap(err)
		}
	}
	return instrumentservice.WriteResult{
		InstrumentsWritten: len(instruments),
		RowsAffected:       len(instruments),
	}, nil
}

func (r *mongoRepository) replaceDocument(ctx context.Context, next instrumentMongoDocument) error {
	var current instrumentMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "instrument_key", Value: next.InstrumentKey}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		next.ID = bson.NewObjectID()
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("instrument_repository").With("instrument_key", next.InstrumentKey).Wrapf(insertErr, "insert instrument mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("instrument_repository").With("instrument_key", next.InstrumentKey).Wrapf(err, "read instrument mongodb document")
	}

	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	next.Sources = replaceSource(current.Sources, next.Sources[0])
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "instrument_key", Value: next.InstrumentKey}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("instrument_repository").With("instrument_key", next.InstrumentKey, "revision", current.Revision).Wrapf(err, "replace instrument mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("instruments", next.InstrumentKey, current.Revision)
	}
	return nil
}

func (r *mongoRepository) SearchInstruments(ctx context.Context, query instrumentservice.Query) (coreinstrument.SearchResult, error) {
	errb := oops.In("instrument_repository").With("backend", "mongodb", "provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "query", query.Query, "limit", query.Limit)
	if strings.TrimSpace(query.Query) == "" {
		return coreinstrument.SearchResult{}, errb.New("search instruments requires query")
	}
	documents, err := r.queryDocuments(ctx, query, false)
	if err != nil {
		return coreinstrument.SearchResult{}, errb.Wrap(err)
	}
	return mongoDocumentsToSearchResult(documents, query.ProviderID), nil
}

func (r *mongoRepository) ListInstruments(ctx context.Context, query instrumentservice.Query) (coreinstrument.SearchResult, error) {
	errb := oops.In("instrument_repository").With("backend", "mongodb", "provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "limit", query.Limit)
	documents, err := r.listDocuments(ctx, query)
	if err != nil {
		return coreinstrument.SearchResult{}, errb.Wrap(err)
	}
	return mongoDocumentsToSearchResult(documents, query.ProviderID), nil
}

func (r *mongoRepository) InspectInstrument(ctx context.Context, query instrumentservice.Query) (coreinstrument.Instrument, error) {
	symbol := strings.TrimSpace(query.Symbol)
	errb := oops.In("instrument_repository").With("backend", "mongodb", "provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "symbol", symbol)
	if symbol == "" {
		return coreinstrument.Instrument{}, errb.New("inspect instrument requires symbol")
	}
	documents, err := r.queryDocuments(ctx, query, true)
	if err != nil {
		return coreinstrument.Instrument{}, errb.Wrap(err)
	}
	if len(documents) == 0 {
		return coreinstrument.Instrument{}, &instrumentservice.NotFoundError{
			Query:        symbol,
			Market:       query.Market,
			SecurityType: query.SecurityType,
		}
	}
	return mongoDocumentToCanonical(documents[0], firstMatchingSource(documents[0], query.ProviderID)), nil
}

func (r *mongoRepository) listDocuments(ctx context.Context, query instrumentservice.Query) ([]instrumentMongoDocument, error) {
	filter := instrumentMongoBaseFilter(query)
	findOptions := options.Find().SetSort(bson.D{{Key: "symbol", Value: 1}, {Key: "sources.provider", Value: 1}, {Key: "sources.operation", Value: 1}})
	if query.Limit > 0 {
		findOptions.SetLimit(int64(query.Limit))
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, oops.In("instrument_repository").With("backend", "mongodb").Wrapf(err, "list instruments mongodb")
	}
	defer cursor.Close(ctx)
	var documents []instrumentMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("instrument_repository").With("backend", "mongodb").Wrapf(err, "decode instruments mongodb")
	}
	return documents, nil
}

func (r *mongoRepository) queryDocuments(ctx context.Context, query instrumentservice.Query, exact bool) ([]instrumentMongoDocument, error) {
	filter := instrumentMongoFilter(query, exact)
	limit := int64(query.Limit)
	if limit <= 0 {
		limit = 25
	}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetLimit(limit).SetSort(bson.D{{Key: "symbol", Value: 1}, {Key: "sources.provider", Value: 1}, {Key: "sources.operation", Value: 1}}))
	if err != nil {
		return nil, oops.In("instrument_repository").With("backend", "mongodb").Wrapf(err, "query instruments mongodb")
	}
	defer cursor.Close(ctx)
	var documents []instrumentMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("instrument_repository").With("backend", "mongodb").Wrapf(err, "decode instruments mongodb")
	}
	return rankInstrumentMongoDocuments(documents, query, exact), nil
}

func instrumentMongoBaseFilter(query instrumentservice.Query) bson.D {
	filter := bson.D{{Key: "market_key", Value: string(withDefaultMarket(query.Market))}}
	if query.SecurityType != "" {
		filter = append(filter, bson.E{Key: "security_type", Value: string(query.SecurityType)})
	}
	if query.ProviderID != "" {
		filter = append(filter, bson.E{Key: "sources.provider", Value: string(query.ProviderID)})
	}
	return filter
}

func instrumentToMongoDocument(item coreinstrument.Instrument) instrumentMongoDocument {
	now := storagemongodb.ISOTimeNow()
	market := string(withDefaultMarket(item.Market))
	symbol := canonicalSymbol(item)
	return instrumentMongoDocument{
		InstrumentKey: instrumentMongoKey(market, string(item.SecurityType), symbol),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		MarketKey:     market,
		SecurityType:  string(item.SecurityType),
		Symbol:        symbol,
		ISIN:          strings.TrimSpace(item.ISIN),
		Name:          strings.TrimSpace(item.Name),
		ExchangeCode:  strings.TrimSpace(item.ExchangeCode),
		CountryCode:   firstNonEmpty(item.CountryCode, countryCode(provider.Market(market))),
		Timezone:      firstNonEmpty(item.Timezone, marketTimezone(provider.Market(market))),
		Sources: []instrumentMongoSource{
			{
				Provider:       string(item.Provider),
				ProviderGroup:  string(item.Group),
				Operation:      string(item.Operation),
				ProviderSymbol: providerSymbol(item),
			},
		},
		Extensions: sanitizedExtensions(item.Extensions),
	}
}

func instrumentMongoFilter(query instrumentservice.Query, exact bool) bson.D {
	filter := instrumentMongoBaseFilter(query)
	if exact {
		needle := strings.TrimSpace(query.Symbol)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			exactRegexFilter("symbol", needle),
			exactRegexFilter("isin", needle),
			exactRegexFilter("sources.provider_symbol", needle),
			exactRegexFilter("extensions.alias", needle),
			exactRegexFilter("extensions.k_ticker", needle),
		}})
		return filter
	}
	needle := strings.TrimSpace(query.Query)
	filter = append(filter, bson.E{Key: "$or", Value: bson.A{
		exactRegexFilter("symbol", needle),
		exactRegexFilter("isin", needle),
		exactRegexFilter("sources.provider_symbol", needle),
		containsRegexFilter("name", needle),
		containsRegexFilter("extensions.issueName", needle),
		containsRegexFilter("extensions.issueEnglishName", needle),
		containsRegexFilter("extensions.listingDate", needle),
		containsRegexFilter("extensions.alias", needle),
		containsRegexFilter("extensions.k_ticker", needle),
	}})
	return filter
}

func mongoDocumentsToSearchResult(documents []instrumentMongoDocument, preferredProvider provider.ProviderID) coreinstrument.SearchResult {
	instruments := make([]coreinstrument.Instrument, 0, len(documents))
	operations := make([]provider.OperationID, 0)
	seenOperations := make(map[provider.OperationID]bool)
	var identity provider.Identity
	var group provider.GroupID
	for _, document := range documents {
		source := firstMatchingSource(document, preferredProvider)
		instruments = append(instruments, mongoDocumentToCanonical(document, source))
		if identity.ID == "" {
			identity = provider.Identity{ID: provider.ProviderID(source.Provider)}
		}
		if group == "" {
			group = provider.GroupID(source.ProviderGroup)
		}
		operation := provider.OperationID(source.Operation)
		if !seenOperations[operation] {
			seenOperations[operation] = true
			operations = append(operations, operation)
		}
	}
	return coreinstrument.SearchResult{
		Instruments: instruments,
		Provider:    identity,
		Group:       group,
		Operations:  operations,
		TotalCount:  len(instruments),
	}
}

func mongoDocumentToCanonical(document instrumentMongoDocument, source instrumentMongoSource) coreinstrument.Instrument {
	return coreinstrument.Instrument{
		Provider:     provider.ProviderID(source.Provider),
		Group:        provider.GroupID(source.ProviderGroup),
		Operation:    provider.OperationID(source.Operation),
		Market:       provider.Market(document.MarketKey),
		SecurityType: provider.SecurityType(document.SecurityType),
		SecurityCode: document.Symbol,
		ISIN:         document.ISIN,
		Name:         document.Name,
		ExchangeCode: document.ExchangeCode,
		CountryCode:  document.CountryCode,
		Timezone:     document.Timezone,
		Extensions:   document.Extensions,
	}
}

func firstMatchingSource(document instrumentMongoDocument, preferredProvider provider.ProviderID) instrumentMongoSource {
	for _, source := range document.Sources {
		if preferredProvider == "" || source.Provider == string(preferredProvider) {
			return source
		}
	}
	if len(document.Sources) > 0 {
		return document.Sources[0]
	}
	return instrumentMongoSource{}
}

func replaceSource(sources []instrumentMongoSource, next instrumentMongoSource) []instrumentMongoSource {
	out := make([]instrumentMongoSource, 0, len(sources)+1)
	for _, source := range sources {
		if sameSource(source, next) {
			continue
		}
		out = append(out, source)
	}
	out = append(out, next)
	return out
}

func sameSource(left instrumentMongoSource, right instrumentMongoSource) bool {
	return left.Provider == right.Provider &&
		left.ProviderGroup == right.ProviderGroup &&
		left.Operation == right.Operation &&
		left.ProviderSymbol == right.ProviderSymbol
}

func rankInstrumentMongoDocuments(documents []instrumentMongoDocument, query instrumentservice.Query, exact bool) []instrumentMongoDocument {
	needle := strings.TrimSpace(query.Query)
	if exact {
		needle = strings.TrimSpace(query.Symbol)
	}
	copied := append([]instrumentMongoDocument(nil), documents...)
	sort.SliceStable(copied, func(i, j int) bool {
		left := instrumentMongoRank(copied[i], needle)
		right := instrumentMongoRank(copied[j], needle)
		if left != right {
			return left < right
		}
		return copied[i].Symbol < copied[j].Symbol
	})
	return copied
}

func instrumentMongoRank(document instrumentMongoDocument, needle string) int {
	if strings.EqualFold(document.Symbol, needle) {
		return 0
	}
	if strings.EqualFold(document.ISIN, needle) {
		return 1
	}
	for _, source := range document.Sources {
		if strings.EqualFold(source.ProviderSymbol, needle) {
			return 1
		}
	}
	return 2
}

func sanitizedExtensions(extensions map[string]string) map[string]string {
	if len(extensions) == 0 {
		return nil
	}
	out := make(map[string]string, len(extensions))
	for key, value := range extensions {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func instrumentMongoKey(market string, securityType string, symbol string) string {
	return strings.Join([]string{"instruments", market, securityType, symbol}, ":")
}

func exactRegexFilter(field string, value string) bson.D {
	return bson.D{{Key: field, Value: bson.D{{Key: "$regex", Value: "^" + regexp.QuoteMeta(value) + "$"}, {Key: "$options", Value: "i"}}}}
}

func containsRegexFilter(field string, value string) bson.D {
	return bson.D{{Key: field, Value: bson.D{{Key: "$regex", Value: regexp.QuoteMeta(value)}, {Key: "$options", Value: "i"}}}}
}
