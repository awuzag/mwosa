package indexbar

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	coreindexbar "github.com/awuzag/mwosa/providers/core/indexbar"
	indexservice "github.com/awuzag/mwosa/service/index"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	indexes   *mongo.Collection
	indexBars *mongo.Collection
}

type indexMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	MarketKey    string             `bson:"market_key"`
	IndexCode    string             `bson:"index_code"`
	Name         string             `bson:"name,omitempty"`
	Family       string             `bson:"family,omitempty"`
	CountryCode  string             `bson:"country_code,omitempty"`
	CurrencyCode string             `bson:"currency_code,omitempty"`
	Timezone     string             `bson:"timezone,omitempty"`
	IndexType    string             `bson:"index_type,omitempty"`
	Sources      []indexMongoSource `bson:"sources"`
	Extensions   map[string]string  `bson:"extensions,omitempty"`
}

type indexBarMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	IndexKey     string               `bson:"index_key"`
	MarketKey    string               `bson:"market_key"`
	IndexCode    string               `bson:"index_code"`
	Name         string               `bson:"name,omitempty"`
	Family       string               `bson:"family,omitempty"`
	CountryCode  string               `bson:"country_code,omitempty"`
	CurrencyCode string               `bson:"currency_code,omitempty"`
	Timezone     string               `bson:"timezone,omitempty"`
	IndexType    string               `bson:"index_type,omitempty"`
	TradingDate  string               `bson:"trading_date"`
	Source       indexMongoSource     `bson:"source"`
	Values       indexBarMongoValues  `bson:"values,omitempty"`
	Volumes      indexBarMongoVolumes `bson:"volumes,omitempty"`
	Extensions   map[string]string    `bson:"extensions,omitempty"`
}

type indexMongoSource struct {
	Provider       string `bson:"provider"`
	ProviderGroup  string `bson:"provider_group"`
	Operation      string `bson:"operation"`
	ProviderSymbol string `bson:"provider_symbol"`
}

type indexBarMongoValues struct {
	Open       string `bson:"open,omitempty"`
	High       string `bson:"high,omitempty"`
	Low        string `bson:"low,omitempty"`
	Close      string `bson:"close,omitempty"`
	Change     string `bson:"change,omitempty"`
	ChangeRate string `bson:"change_rate,omitempty"`
}

type indexBarMongoVolumes struct {
	Volume      string `bson:"volume,omitempty"`
	TradedValue string `bson:"traded_value,omitempty"`
	MarketCap   string `bson:"market_cap,omitempty"`
}

var _ indexservice.ReadRepository = (*mongoRepository)(nil)
var _ indexservice.WriteRepository = (*mongoRepository)(nil)

func NewMongoRepositories(database *mongo.Database) (indexservice.ReadRepository, indexservice.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("indexbar_repository").New("mongodb database is nil")
	}
	repository := &mongoRepository{
		indexes:   database.Collection("indexes"),
		indexBars: database.Collection("index_bars"),
	}
	return repository, repository, nil
}

func (r *mongoRepository) UpsertIndexBars(ctx context.Context, bars []coreindexbar.Bar) (indexservice.WriteResult, error) {
	errb := oops.In("indexbar_repository").With("backend", "mongodb", "bars", len(bars))
	for _, bar := range bars {
		barErrb := errb.With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation, "market", bar.Market, "index_code", bar.IndexCode, "date", bar.TradingDate)
		if err := validateBarKey(bar); err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		indexDocument, barDocument, err := indexBarToMongoDocuments(bar)
		if err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		if err := r.replaceIndexDocument(ctx, indexDocument); err != nil {
			return indexservice.WriteResult{}, barErrb.With("id", indexDocument.ID).Wrap(err)
		}
		if err := r.upsertBarDocument(ctx, barDocument); err != nil {
			return indexservice.WriteResult{}, barErrb.With("id", barDocument.ID).Wrap(err)
		}
	}
	return indexservice.WriteResult{
		BarsWritten:  len(bars),
		RowsAffected: len(bars),
	}, nil
}

func (r *mongoRepository) QueryIndexBars(ctx context.Context, query indexservice.Query) ([]coreindexbar.Bar, error) {
	documents, err := r.queryBarDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	bars := make([]coreindexbar.Bar, 0, len(documents))
	for _, document := range documents {
		bars = append(bars, indexBarMongoDocumentToCanonical(document))
	}
	return bars, nil
}

func (r *mongoRepository) replaceIndexDocument(ctx context.Context, next indexMongoDocument) error {
	var current indexMongoDocument
	err := r.indexes.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, insertErr := r.indexes.InsertOne(ctx, next); insertErr != nil {
			return oops.In("indexbar_repository").With("id", next.ID).Wrapf(insertErr, "insert index mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("indexbar_repository").With("id", next.ID).Wrapf(err, "read index mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	next.Sources = replaceIndexSource(current.Sources, next.Sources[0])
	result, err := r.indexes.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("indexbar_repository").With("id", next.ID, "revision", current.Revision).Wrapf(err, "replace index mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("indexes", next.ID, current.Revision)
	}
	return nil
}

func (r *mongoRepository) upsertBarDocument(ctx context.Context, document indexBarMongoDocument) error {
	filter := bson.D{{Key: "_id", Value: document.ID}}
	update := bson.D{
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: document.ID},
			{Key: "created_at", Value: document.CreatedAt},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "schema_version", Value: document.SchemaVersion},
			{Key: "updated_at", Value: document.UpdatedAt},
			{Key: "index_key", Value: document.IndexKey},
			{Key: "market_key", Value: document.MarketKey},
			{Key: "index_code", Value: document.IndexCode},
			{Key: "name", Value: document.Name},
			{Key: "family", Value: document.Family},
			{Key: "country_code", Value: document.CountryCode},
			{Key: "currency_code", Value: document.CurrencyCode},
			{Key: "timezone", Value: document.Timezone},
			{Key: "index_type", Value: document.IndexType},
			{Key: "trading_date", Value: document.TradingDate},
			{Key: "source", Value: document.Source},
			{Key: "values", Value: document.Values},
			{Key: "volumes", Value: document.Volumes},
			{Key: "extensions", Value: document.Extensions},
		}},
		{Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}},
	}
	if _, err := r.indexBars.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return oops.In("indexbar_repository").With("id", document.ID).Wrapf(err, "upsert index bar mongodb document")
	}
	return nil
}

func (r *mongoRepository) queryBarDocuments(ctx context.Context, query indexservice.Query) ([]indexBarMongoDocument, error) {
	filter, err := indexBarMongoFilter(query)
	if err != nil {
		return nil, err
	}
	cursor, err := r.indexBars.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "trading_date", Value: 1},
		{Key: "index_code", Value: 1},
		{Key: "source.provider", Value: 1},
		{Key: "source.provider_group", Value: 1},
	}))
	if err != nil {
		return nil, oops.In("indexbar_repository").With("backend", "mongodb").Wrapf(err, "query index bars mongodb")
	}
	defer cursor.Close(ctx)
	var documents []indexBarMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("indexbar_repository").With("backend", "mongodb").Wrapf(err, "decode index bars mongodb")
	}
	return sortedIndexBarMongoDocuments(documents), nil
}

func indexBarToMongoDocuments(bar coreindexbar.Bar) (indexMongoDocument, indexBarMongoDocument, error) {
	row, err := indexBarToRow(bar, 1, 1, storagemongodb.ISOTimeNow().UnixMilli())
	if err != nil {
		return indexMongoDocument{}, indexBarMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	market := string(withDefaultMarket(bar.Market))
	indexCode := strings.TrimSpace(bar.IndexCode)
	source := indexMongoSource{
		Provider:       string(bar.Provider),
		ProviderGroup:  string(bar.Group),
		Operation:      string(bar.Operation),
		ProviderSymbol: indexCode,
	}
	currencyCode := normalizedCurrencyCode(bar.Currency)
	timezone := marketTimezone(provider.Market(market))
	indexType := "price"
	indexKey := indexMongoKey(market, indexCode)
	indexDocument := indexMongoDocument{
		ID:            indexMongoID(market, indexCode),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		MarketKey:     market,
		IndexCode:     indexCode,
		Name:          strings.TrimSpace(bar.Name),
		Family:        strings.TrimSpace(bar.Family),
		CountryCode:   countryCode(provider.Market(market)),
		CurrencyCode:  currencyCode,
		Timezone:      timezone,
		IndexType:     indexType,
		Sources:       []indexMongoSource{source},
	}
	barDocument := indexBarMongoDocument{
		ID:            indexBarMongoID(market, indexCode, formatTradingDate(row.TradingDate), source),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IndexKey:      indexKey,
		MarketKey:     market,
		IndexCode:     indexCode,
		Name:          strings.TrimSpace(bar.Name),
		Family:        strings.TrimSpace(bar.Family),
		CountryCode:   countryCode(provider.Market(market)),
		CurrencyCode:  currencyCode,
		Timezone:      timezone,
		IndexType:     indexType,
		TradingDate:   formatTradingDate(row.TradingDate),
		Source:        source,
		Values: indexBarMongoValues{
			Open:       bar.Open,
			High:       bar.High,
			Low:        bar.Low,
			Close:      bar.Close,
			Change:     bar.Change,
			ChangeRate: bar.ChangeRate,
		},
		Volumes: indexBarMongoVolumes{
			Volume:      bar.Volume,
			TradedValue: bar.TradedValue,
			MarketCap:   bar.MarketCap,
		},
		Extensions: copyStringMap(bar.Extensions),
	}
	return indexDocument, barDocument, nil
}

func indexBarMongoDocumentToCanonical(document indexBarMongoDocument) coreindexbar.Bar {
	extensions := make(map[string]string)
	if document.Timezone != "" {
		extensions["index.timezone"] = document.Timezone
	}
	if document.IndexType != "" {
		extensions["index.type"] = document.IndexType
	}
	for key, value := range document.Extensions {
		extensions[key] = value
	}
	if len(extensions) == 0 {
		extensions = nil
	}
	return coreindexbar.Bar{
		Provider:    provider.ProviderID(document.Source.Provider),
		Group:       provider.GroupID(document.Source.ProviderGroup),
		Operation:   provider.OperationID(document.Source.Operation),
		Market:      provider.Market(document.MarketKey),
		IndexCode:   document.IndexCode,
		Name:        document.Name,
		Family:      document.Family,
		TradingDate: document.TradingDate,
		Currency:    document.CurrencyCode,
		Open:        document.Values.Open,
		High:        document.Values.High,
		Low:         document.Values.Low,
		Close:       document.Values.Close,
		Change:      document.Values.Change,
		ChangeRate:  document.Values.ChangeRate,
		Volume:      document.Volumes.Volume,
		TradedValue: document.Volumes.TradedValue,
		MarketCap:   document.Volumes.MarketCap,
		Extensions:  extensions,
	}
}

func indexBarMongoFilter(query indexservice.Query) (bson.D, error) {
	filter := bson.D{{Key: "market_key", Value: string(withDefaultMarket(query.Market))}}
	if strings.TrimSpace(query.IndexCode) != "" {
		filter = append(filter, bson.E{Key: "index_code", Value: strings.TrimSpace(query.IndexCode)})
	}
	rangeFilter := bson.D{}
	if strings.TrimSpace(query.From) != "" {
		from, err := normalizeIndexMongoTradingDate(query.From)
		if err != nil {
			return nil, err
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$gte", Value: from})
	}
	if strings.TrimSpace(query.To) != "" {
		to, err := normalizeIndexMongoTradingDate(query.To)
		if err != nil {
			return nil, err
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$lte", Value: to})
	}
	if len(rangeFilter) > 0 {
		filter = append(filter, bson.E{Key: "trading_date", Value: rangeFilter})
	}
	return filter, nil
}

func normalizeIndexMongoTradingDate(value string) (string, error) {
	date, err := parseTradingDate(value)
	if err != nil {
		return "", err
	}
	return formatTradingDate(date), nil
}

func replaceIndexSource(sources []indexMongoSource, next indexMongoSource) []indexMongoSource {
	replaced := false
	out := make([]indexMongoSource, 0, len(sources)+1)
	for _, source := range sources {
		if sameIndexSource(source, next) {
			out = append(out, next)
			replaced = true
			continue
		}
		out = append(out, source)
	}
	if !replaced {
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Provider, out[i].ProviderGroup, out[i].Operation, out[i].ProviderSymbol}, "\x00")
		right := strings.Join([]string{out[j].Provider, out[j].ProviderGroup, out[j].Operation, out[j].ProviderSymbol}, "\x00")
		return left < right
	})
	return out
}

func sameIndexSource(left indexMongoSource, right indexMongoSource) bool {
	return left.Provider == right.Provider &&
		left.ProviderGroup == right.ProviderGroup &&
		left.Operation == right.Operation &&
		left.ProviderSymbol == right.ProviderSymbol
}

func indexMongoKey(market string, indexCode string) string {
	return strings.Join([]string{market, indexCode}, ":")
}

func indexMongoID(market string, indexCode string) string {
	return strings.Join([]string{"indexes", market, indexCode}, ":")
}

func indexBarMongoID(market string, indexCode string, tradingDate string, source indexMongoSource) string {
	return strings.Join([]string{
		"index_bars",
		market,
		indexCode,
		tradingDate,
		source.Provider,
		source.ProviderGroup,
		source.Operation,
	}, ":")
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func sortedIndexBarMongoDocuments(documents []indexBarMongoDocument) []indexBarMongoDocument {
	copied := append([]indexBarMongoDocument(nil), documents...)
	sort.Slice(copied, func(i, j int) bool {
		if copied[i].TradingDate != copied[j].TradingDate {
			return copied[i].TradingDate < copied[j].TradingDate
		}
		if copied[i].IndexCode != copied[j].IndexCode {
			return copied[i].IndexCode < copied[j].IndexCode
		}
		return copied[i].Source.Provider < copied[j].Source.Provider
	})
	return copied
}
