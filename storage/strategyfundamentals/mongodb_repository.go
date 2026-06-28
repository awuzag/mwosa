package strategyfundamentals

import (
	"context"
	"strings"

	strategyservice "github.com/awuzag/mwosa/service/strategy"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	metrics    *mongo.Collection
	valuations *mongo.Collection
	facts      *mongo.Collection
	events     *mongo.Collection
}

type strategyInstrumentDocument struct {
	Market       string `bson:"market,omitempty"`
	SecurityType string `bson:"security_type,omitempty"`
	Symbol       string `bson:"symbol,omitempty"`
}

type strategyMetricDocument struct {
	Instrument         strategyInstrumentDocument `bson:"instrument"`
	Metric             string                     `bson:"metric"`
	FiscalYear         string                     `bson:"fiscal_year"`
	FiscalPeriod       string                     `bson:"fiscal_period,omitempty"`
	AsOfDate           string                     `bson:"as_of_date,omitempty"`
	ValueDecimal       string                     `bson:"value_decimal,omitempty"`
	ValueBP            *int64                     `bson:"value_bp,omitempty"`
	ValueMinor         *int64                     `bson:"value_minor,omitempty"`
	FormulaVersion     string                     `bson:"formula_version"`
	UncomputableReason string                     `bson:"uncomputable_reason,omitempty"`
}

type strategyValuationDocument struct {
	Instrument          strategyInstrumentDocument `bson:"instrument"`
	AsOfDate            string                     `bson:"as_of_date"`
	SourcePriceDate     string                     `bson:"source_price_date"`
	MarketCapMinor      *int64                     `bson:"market_cap_minor,omitempty"`
	ClosePriceMinor     *int64                     `bson:"close_price_minor,omitempty"`
	SharesOutstanding   *int64                     `bson:"shares_outstanding,omitempty"`
	PerBP               *int64                     `bson:"per_bp,omitempty"`
	PbrBP               *int64                     `bson:"pbr_bp,omitempty"`
	PsrBP               *int64                     `bson:"psr_bp,omitempty"`
	EpsMinor            *int64                     `bson:"eps_minor,omitempty"`
	BpsMinor            *int64                     `bson:"bps_minor,omitempty"`
	DividendYieldBP     *int64                     `bson:"dividend_yield_bp,omitempty"`
	MetricSourceVersion string                     `bson:"metric_source_version"`
	Uncomputable        map[string]string          `bson:"uncomputable,omitempty"`
}

type strategyFactDocument struct {
	Instrument   strategyInstrumentDocument `bson:"instrument"`
	FactType     string                     `bson:"fact_type"`
	FiscalYear   string                     `bson:"fiscal_year,omitempty"`
	ReportCode   string                     `bson:"report_code,omitempty"`
	RceptNo      string                     `bson:"rcept_no,omitempty"`
	FactDate     string                     `bson:"fact_date,omitempty"`
	Key          string                     `bson:"key"`
	ValueText    string                     `bson:"value_text,omitempty"`
	ValueNumber  string                     `bson:"value_number,omitempty"`
	CurrencyCode string                     `bson:"currency_code,omitempty"`
	Provider     string                     `bson:"provider"`
	Group        string                     `bson:"provider_group"`
	Operation    string                     `bson:"operation"`
}

type strategyEventDocument struct {
	Instrument    strategyInstrumentDocument `bson:"instrument"`
	EventType     string                     `bson:"event_type"`
	EventDate     string                     `bson:"event_date,omitempty"`
	RceptDt       string                     `bson:"rcept_dt,omitempty"`
	RceptNo       string                     `bson:"rcept_no"`
	EffectiveDate string                     `bson:"effective_date"`
	Title         string                     `bson:"title,omitempty"`
	AmountMinor   *int64                     `bson:"amount_minor,omitempty"`
	ValueText     string                     `bson:"value_text,omitempty"`
	Provider      string                     `bson:"provider"`
	Group         string                     `bson:"provider_group"`
	Operation     string                     `bson:"operation"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("strategy_fundamentals_repository").New("mongodb database is nil")
	}
	return MongoRepository{
		metrics:    database.Collection("financial_metrics"),
		valuations: database.Collection("valuation_snapshots"),
		facts:      database.Collection("company_facts"),
		events:     database.Collection("company_events"),
	}, nil
}

func (r MongoRepository) ListLatestFundamentals(ctx context.Context, query strategyservice.FundamentalsQuery) (map[string]strategyservice.Fundamentals, error) {
	errb := oops.In("strategy_fundamentals_repository").With("backend", "mongodb", "market", query.Market, "security_type", query.SecurityType)
	out := map[string]strategyservice.Fundamentals{}
	if err := r.loadMongoMetrics(ctx, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadMongoValuations(ctx, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadMongoFacts(ctx, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadMongoEvents(ctx, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	return out, nil
}

func (r MongoRepository) loadMongoMetrics(ctx context.Context, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	filter := strategyInstrumentFilter(query)
	cursor, err := r.metrics.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "instrument.symbol", Value: 1},
		{Key: "metric", Value: 1},
		{Key: "fiscal_year", Value: -1},
		{Key: "as_of_date", Value: -1},
		{Key: "_id", Value: -1},
	}))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "query financial metric screen fundamentals")
	}
	defer cursor.Close(ctx)
	var documents []strategyMetricDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "decode financial metric screen fundamentals")
	}
	seen := map[string]struct{}{}
	for _, document := range documents {
		symbol := strings.TrimSpace(document.Instrument.Symbol)
		key := symbol + "\x00" + document.Metric
		if symbol == "" || document.Metric == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item := out[symbol]
		item.Symbol = symbol
		if item.Metrics == nil {
			item.Metrics = map[string]strategyservice.FundamentalMetric{}
		}
		item.Metrics[document.Metric] = strategyservice.FundamentalMetric{
			Metric:             document.Metric,
			FiscalYear:         document.FiscalYear,
			FiscalPeriod:       document.FiscalPeriod,
			AsOfDate:           document.AsOfDate,
			ValueDecimal:       document.ValueDecimal,
			ValueBP:            document.ValueBP,
			ValueMinor:         document.ValueMinor,
			FormulaVersion:     document.FormulaVersion,
			UncomputableReason: document.UncomputableReason,
		}
		out[symbol] = item
	}
	return nil
}

func (r MongoRepository) loadMongoValuations(ctx context.Context, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	filter := strategyInstrumentFilter(query)
	cursor, err := r.valuations.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "instrument.symbol", Value: 1},
		{Key: "as_of_date", Value: -1},
		{Key: "_id", Value: -1},
	}))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "query valuation screen fundamentals")
	}
	defer cursor.Close(ctx)
	var documents []strategyValuationDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "decode valuation screen fundamentals")
	}
	seen := map[string]struct{}{}
	for _, document := range documents {
		symbol := strings.TrimSpace(document.Instrument.Symbol)
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		item := out[symbol]
		item.Symbol = symbol
		item.Valuation = &strategyservice.FundamentalValuation{
			AsOfDate:            document.AsOfDate,
			SourcePriceDate:     document.SourcePriceDate,
			MarketCapMinor:      document.MarketCapMinor,
			ClosePriceMinor:     document.ClosePriceMinor,
			SharesOutstanding:   document.SharesOutstanding,
			PerBP:               document.PerBP,
			PbrBP:               document.PbrBP,
			PsrBP:               document.PsrBP,
			EpsMinor:            document.EpsMinor,
			BpsMinor:            document.BpsMinor,
			DividendYieldBP:     document.DividendYieldBP,
			MetricSourceVersion: document.MetricSourceVersion,
			Uncomputable:        cloneStringMap(document.Uncomputable),
		}
		out[symbol] = item
	}
	return nil
}

func (r MongoRepository) loadMongoFacts(ctx context.Context, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	filter := strategyInstrumentFilter(query)
	cursor, err := r.facts.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "instrument.symbol", Value: 1},
		{Key: "fact_type", Value: 1},
		{Key: "key", Value: 1},
		{Key: "fiscal_year", Value: -1},
		{Key: "report_code", Value: -1},
		{Key: "fact_date", Value: -1},
		{Key: "_id", Value: -1},
	}))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "query company fact screen fundamentals")
	}
	defer cursor.Close(ctx)
	var documents []strategyFactDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "decode company fact screen fundamentals")
	}
	seen := map[string]struct{}{}
	counts := map[string]int{}
	for _, document := range documents {
		symbol := strings.TrimSpace(document.Instrument.Symbol)
		if symbol == "" || counts[symbol] >= defaultFactLimit {
			continue
		}
		key := mongoFactMapKey(document)
		seenKey := symbol + "\x00" + key
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}
		counts[symbol]++
		item := out[symbol]
		item.Symbol = symbol
		if item.Facts == nil {
			item.Facts = map[string]strategyservice.FundamentalFact{}
		}
		item.Facts[key] = strategyservice.FundamentalFact{
			FactType:     document.FactType,
			FiscalYear:   document.FiscalYear,
			ReportCode:   document.ReportCode,
			RceptNo:      document.RceptNo,
			FactDate:     document.FactDate,
			Key:          document.Key,
			ValueText:    document.ValueText,
			ValueNumber:  document.ValueNumber,
			CurrencyCode: document.CurrencyCode,
			Provider:     document.Provider,
			Group:        document.Group,
			Operation:    document.Operation,
		}
		out[symbol] = item
	}
	return nil
}

func (r MongoRepository) loadMongoEvents(ctx context.Context, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	filter := strategyInstrumentFilter(query)
	cursor, err := r.events.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "instrument.symbol", Value: 1},
		{Key: "effective_date", Value: -1},
		{Key: "_id", Value: -1},
	}))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "query company event screen fundamentals")
	}
	defer cursor.Close(ctx)
	var documents []strategyEventDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return oops.In("strategy_fundamentals_repository").With("backend", "mongodb").Wrapf(err, "decode company event screen fundamentals")
	}
	counts := map[string]int{}
	for _, document := range documents {
		symbol := strings.TrimSpace(document.Instrument.Symbol)
		if symbol == "" || counts[symbol] >= defaultEventLimit {
			continue
		}
		counts[symbol]++
		item := out[symbol]
		item.Symbol = symbol
		item.Events = append(item.Events, strategyservice.FundamentalEvent{
			EventType:   document.EventType,
			EventDate:   document.EventDate,
			RceptDt:     document.RceptDt,
			RceptNo:     document.RceptNo,
			Title:       document.Title,
			AmountMinor: document.AmountMinor,
			ValueText:   document.ValueText,
			Provider:    document.Provider,
			Group:       document.Group,
			Operation:   document.Operation,
		})
		out[symbol] = item
	}
	return nil
}

func strategyInstrumentFilter(query strategyservice.FundamentalsQuery) bson.D {
	return bson.D{
		{Key: "instrument.market", Value: market(query.Market)},
		{Key: "instrument.security_type", Value: securityType(query.SecurityType)},
	}
}

func mongoFactMapKey(document strategyFactDocument) string {
	factType := strings.TrimSpace(document.FactType)
	key := strings.TrimSpace(document.Key)
	if key == "" || key == factType {
		return factType
	}
	return factType + "." + key
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
