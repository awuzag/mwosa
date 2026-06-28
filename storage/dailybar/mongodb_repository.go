package dailybar

import (
	"context"
	"sort"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	coredailybar "github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	database   *mongo.Database
	collection *mongo.Collection
}

type dailyBarMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	InstrumentKey string               `bson:"instrument_key"`
	MarketKey     string               `bson:"market_key"`
	SecurityType  string               `bson:"security_type"`
	Symbol        string               `bson:"symbol"`
	ISIN          string               `bson:"isin,omitempty"`
	Name          string               `bson:"name,omitempty"`
	TradingDate   string               `bson:"trading_date"`
	Currency      string               `bson:"currency,omitempty"`
	Source        dailyBarMongoSource  `bson:"source"`
	Prices        dailyBarMongoPrices  `bson:"prices,omitempty"`
	Volumes       dailyBarMongoVolumes `bson:"volumes,omitempty"`
	Extensions    map[string]string    `bson:"extensions,omitempty"`
}

type dailyBarMongoSource struct {
	Provider      string `bson:"provider"`
	ProviderGroup string `bson:"provider_group"`
	Operation     string `bson:"operation"`
}

type dailyBarMongoPrices struct {
	Open       string `bson:"open,omitempty"`
	High       string `bson:"high,omitempty"`
	Low        string `bson:"low,omitempty"`
	Close      string `bson:"close,omitempty"`
	Change     string `bson:"change,omitempty"`
	ChangeRate string `bson:"change_rate,omitempty"`
}

type dailyBarMongoVolumes struct {
	Volume      string `bson:"volume,omitempty"`
	TradedValue string `bson:"traded_value,omitempty"`
	MarketCap   string `bson:"market_cap,omitempty"`
}

type mongoBarStream struct {
	bars []coredailybar.Bar
	next int
}

var _ daily.ReadRepository = (*mongoRepository)(nil)
var _ daily.StreamRepository = (*mongoRepository)(nil)
var _ daily.WriteRepository = (*mongoRepository)(nil)

func NewMongoRepositories(database *mongo.Database) (daily.ReadRepository, daily.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("dailybar_repository").New("mongodb database is nil")
	}
	repository := &mongoRepository{
		database:   database,
		collection: database.Collection("daily_bars"),
	}
	return repository, repository, nil
}

func (r *mongoRepository) UpsertDailyBars(ctx context.Context, bars []coredailybar.Bar) (daily.WriteResult, error) {
	errb := oops.In("dailybar_repository").With("backend", "mongodb", "bars", len(bars))
	for _, bar := range bars {
		if err := validateBarKey(bar); err != nil {
			return daily.WriteResult{}, errb.Wrap(err)
		}
		doc, err := dailyBarToMongoDocument(bar)
		if err != nil {
			return daily.WriteResult{}, errb.Wrap(err)
		}
		filter := bson.D{{Key: "_id", Value: doc.ID}}
		update := bson.D{
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "_id", Value: doc.ID},
				{Key: "created_at", Value: doc.CreatedAt},
			}},
			{Key: "$set", Value: bson.D{
				{Key: "schema_version", Value: doc.SchemaVersion},
				{Key: "updated_at", Value: doc.UpdatedAt},
				{Key: "instrument_key", Value: doc.InstrumentKey},
				{Key: "market_key", Value: doc.MarketKey},
				{Key: "security_type", Value: doc.SecurityType},
				{Key: "symbol", Value: doc.Symbol},
				{Key: "isin", Value: doc.ISIN},
				{Key: "name", Value: doc.Name},
				{Key: "trading_date", Value: doc.TradingDate},
				{Key: "currency", Value: doc.Currency},
				{Key: "source", Value: doc.Source},
				{Key: "prices", Value: doc.Prices},
				{Key: "volumes", Value: doc.Volumes},
				{Key: "extensions", Value: doc.Extensions},
			}},
			{Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}},
		}
		if _, err := r.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
			return daily.WriteResult{}, errb.With("id", doc.ID).Wrapf(err, "upsert daily bar mongodb document")
		}
	}
	return daily.WriteResult{
		BarsWritten:  len(bars),
		RowsAffected: len(bars),
	}, nil
}

func (r *mongoRepository) QueryDailyBars(ctx context.Context, query daily.Query) ([]coredailybar.Bar, error) {
	documents, err := r.queryDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	bars := make([]coredailybar.Bar, 0, len(documents))
	for _, document := range documents {
		bars = append(bars, dailyBarMongoDocumentToCanonical(document))
	}
	return bars, nil
}

func (r *mongoRepository) StreamDailyBars(ctx context.Context, query daily.Query) (daily.BarStream, error) {
	bars, err := r.QueryDailyBars(ctx, query)
	if err != nil {
		return nil, err
	}
	return &mongoBarStream{bars: bars}, nil
}

func (r *mongoRepository) SummarizeDailyBarStorage(ctx context.Context, query daily.Query) (daily.StorageSummaryResult, error) {
	documents, err := r.queryDocuments(ctx, query)
	if err != nil {
		return daily.StorageSummaryResult{}, err
	}
	symbols := map[string]bool{}
	dates := map[string]bool{}
	var from string
	var to string
	for _, document := range documents {
		symbols[document.Symbol] = true
		dates[document.TradingDate] = true
		if from == "" || document.TradingDate < from {
			from = document.TradingDate
		}
		if to == "" || document.TradingDate > to {
			to = document.TradingDate
		}
	}
	return daily.StorageSummaryResult{
		RecordType:   "daily_bar",
		Market:       marketWithDefault(query.Market),
		SecurityType: query.SecurityType,
		Symbols:      len(symbols),
		Bars:         len(documents),
		From:         from,
		To:           to,
		Dates:        len(dates),
	}, nil
}

func (r *mongoRepository) QueryDailyBarCoverage(ctx context.Context, query daily.Query) (daily.CoverageResult, error) {
	documents, err := r.queryDocuments(ctx, query)
	if err != nil {
		return daily.CoverageResult{}, err
	}
	dates := map[string]bool{}
	var from string
	var to string
	name := ""
	for _, document := range documents {
		if name == "" {
			name = document.Name
		}
		dates[document.TradingDate] = true
		if from == "" || document.TradingDate < from {
			from = document.TradingDate
		}
		if to == "" || document.TradingDate > to {
			to = document.TradingDate
		}
	}
	return daily.CoverageResult{
		Market:       marketWithDefault(query.Market),
		SecurityType: query.SecurityType,
		Symbol:       query.Symbol,
		Name:         name,
		From:         from,
		To:           to,
		Bars:         len(documents),
		Dates:        len(dates),
	}, nil
}

func (r *mongoRepository) queryDocuments(ctx context.Context, query daily.Query) ([]dailyBarMongoDocument, error) {
	filter, err := dailyBarMongoFilter(query)
	if err != nil {
		return nil, err
	}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "trading_date", Value: 1},
		{Key: "symbol", Value: 1},
		{Key: "source.provider", Value: 1},
		{Key: "source.provider_group", Value: 1},
	}))
	if err != nil {
		return nil, oops.In("dailybar_repository").With("backend", "mongodb").Wrapf(err, "query daily bars mongodb")
	}
	defer cursor.Close(ctx)
	var documents []dailyBarMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("dailybar_repository").With("backend", "mongodb").Wrapf(err, "decode daily bars mongodb")
	}
	return documents, nil
}

func (s *mongoBarStream) Next(ctx context.Context) (coredailybar.Bar, bool, error) {
	if err := ctx.Err(); err != nil {
		return coredailybar.Bar{}, false, oops.In("dailybar_repository").Wrap(err)
	}
	if s.next >= len(s.bars) {
		return coredailybar.Bar{}, false, nil
	}
	bar := s.bars[s.next]
	s.next++
	return bar, true, nil
}

func (s *mongoBarStream) Close() error {
	return nil
}

func dailyBarToMongoDocument(bar coredailybar.Bar) (dailyBarMongoDocument, error) {
	tradingDate, err := parseTradingDate(bar.TradingDate)
	if err != nil {
		return dailyBarMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	market := string(marketWithDefault(bar.Market))
	securityType := string(bar.SecurityType)
	symbol := strings.TrimSpace(bar.Symbol)
	source := dailyBarMongoSource{
		Provider:      string(bar.Provider),
		ProviderGroup: string(bar.Group),
		Operation:     string(bar.Operation),
	}
	return dailyBarMongoDocument{
		ID:            dailyBarMongoID(market, securityType, symbol, formatTradingDate(tradingDate), source),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		InstrumentKey: strings.Join([]string{market, securityType, symbol}, ":"),
		MarketKey:     market,
		SecurityType:  securityType,
		Symbol:        symbol,
		ISIN:          bar.ISIN,
		Name:          bar.Name,
		TradingDate:   formatTradingDate(tradingDate),
		Currency:      normalizedCurrencyCode(bar.Currency),
		Source:        source,
		Prices: dailyBarMongoPrices{
			Open:       bar.Open,
			High:       bar.High,
			Low:        bar.Low,
			Close:      bar.Close,
			Change:     bar.Change,
			ChangeRate: bar.ChangeRate,
		},
		Volumes: dailyBarMongoVolumes{
			Volume:      bar.Volume,
			TradedValue: bar.TradedValue,
			MarketCap:   bar.MarketCap,
		},
		Extensions: bar.Extensions,
	}, nil
}

func dailyBarMongoDocumentToCanonical(document dailyBarMongoDocument) coredailybar.Bar {
	return coredailybar.Bar{
		Provider:     provider.ProviderID(document.Source.Provider),
		Group:        provider.GroupID(document.Source.ProviderGroup),
		Operation:    provider.OperationID(document.Source.Operation),
		Market:       provider.Market(document.MarketKey),
		SecurityType: provider.SecurityType(document.SecurityType),
		Symbol:       document.Symbol,
		ISIN:         document.ISIN,
		Name:         document.Name,
		TradingDate:  document.TradingDate,
		Currency:     document.Currency,
		Open:         document.Prices.Open,
		High:         document.Prices.High,
		Low:          document.Prices.Low,
		Close:        document.Prices.Close,
		Change:       document.Prices.Change,
		ChangeRate:   document.Prices.ChangeRate,
		Volume:       document.Volumes.Volume,
		TradedValue:  document.Volumes.TradedValue,
		MarketCap:    document.Volumes.MarketCap,
		Extensions:   document.Extensions,
	}
}

func dailyBarMongoFilter(query daily.Query) (bson.D, error) {
	filter := bson.D{{Key: "market_key", Value: string(marketWithDefault(query.Market))}}
	if query.SecurityType != "" {
		filter = append(filter, bson.E{Key: "security_type", Value: string(query.SecurityType)})
	}
	if strings.TrimSpace(query.Symbol) != "" {
		filter = append(filter, bson.E{Key: "symbol", Value: strings.TrimSpace(query.Symbol)})
	}
	rangeFilter := bson.D{}
	if strings.TrimSpace(query.From) != "" {
		from, err := normalizeMongoTradingDate(query.From)
		if err != nil {
			return nil, err
		}
		rangeFilter = append(rangeFilter, bson.E{Key: "$gte", Value: from})
	}
	if strings.TrimSpace(query.To) != "" {
		to, err := normalizeMongoTradingDate(query.To)
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

func normalizeMongoTradingDate(value string) (string, error) {
	date, err := parseTradingDate(value)
	if err != nil {
		return "", err
	}
	return formatTradingDate(date), nil
}

func dailyBarMongoID(market string, securityType string, symbol string, tradingDate string, source dailyBarMongoSource) string {
	parts := []string{
		"daily_bars",
		market,
		securityType,
		symbol,
		tradingDate,
		source.Provider,
		source.ProviderGroup,
		source.Operation,
	}
	return strings.Join(parts, ":")
}

func sortedMongoDailyBarDocuments(documents []dailyBarMongoDocument) []dailyBarMongoDocument {
	copied := append([]dailyBarMongoDocument(nil), documents...)
	sort.Slice(copied, func(i, j int) bool {
		if copied[i].TradingDate != copied[j].TradingDate {
			return copied[i].TradingDate < copied[j].TradingDate
		}
		if copied[i].Symbol != copied[j].Symbol {
			return copied[i].Symbol < copied[j].Symbol
		}
		return copied[i].Source.Provider < copied[j].Source.Provider
	})
	return copied
}
