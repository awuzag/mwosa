package valuation

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/awuzag/mwosa/packages/financialmetrics"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const mongoDefaultPriceScale = 4

type MongoRepository struct {
	collection          *mongo.Collection
	dailyBarCollection  *mongo.Collection
	statementCollection *mongo.Collection
	factCollection      *mongo.Collection
}

type snapshotMongoDocument struct {
	ID                  string                  `bson:"_id"`
	SchemaVersion       string                  `bson:"schema_version"`
	Revision            int64                   `bson:"revision"`
	CreatedAt           time.Time               `bson:"created_at"`
	UpdatedAt           time.Time               `bson:"updated_at"`
	CompanyID           int64                   `bson:"company_id"`
	InstrumentID        int64                   `bson:"instrument_id"`
	Company             companySnapshotDocument `bson:"company"`
	Instrument          instrumentSnapshot      `bson:"instrument"`
	AsOfDate            string                  `bson:"as_of_date"`
	SourcePriceDate     string                  `bson:"source_price_date"`
	MarketCapMinor      *int64                  `bson:"market_cap_minor,omitempty"`
	ClosePriceMinor     *int64                  `bson:"close_price_minor,omitempty"`
	SharesOutstanding   *int64                  `bson:"shares_outstanding,omitempty"`
	PerBP               *int64                  `bson:"per_bp,omitempty"`
	PbrBP               *int64                  `bson:"pbr_bp,omitempty"`
	PsrBP               *int64                  `bson:"psr_bp,omitempty"`
	EpsMinor            *int64                  `bson:"eps_minor,omitempty"`
	BpsMinor            *int64                  `bson:"bps_minor,omitempty"`
	DividendYieldBP     *int64                  `bson:"dividend_yield_bp,omitempty"`
	MetricSourceVersion string                  `bson:"metric_source_version"`
	Provenance          map[string]any          `bson:"provenance"`
	Uncomputable        map[string]string       `bson:"uncomputable"`
}

type valuationDailyBarDocument struct {
	TradingDate string                   `bson:"trading_date"`
	Source      valuationDailyBarSource  `bson:"source"`
	Prices      valuationDailyBarPrices  `bson:"prices"`
	Volumes     valuationDailyBarVolumes `bson:"volumes"`
}

type valuationDailyBarSource struct {
	Provider      string `bson:"provider"`
	ProviderGroup string `bson:"provider_group"`
	Operation     string `bson:"operation"`
}

type valuationDailyBarPrices struct {
	Close string `bson:"close"`
}

type valuationDailyBarVolumes struct {
	MarketCap string `bson:"market_cap"`
}

type statementAccountDocument struct {
	StatementID   int64                   `bson:"statement_id"`
	Provider      string                  `bson:"provider"`
	ProviderGroup string                  `bson:"provider_group"`
	Operation     string                  `bson:"operation"`
	RceptNo       string                  `bson:"rcept_no"`
	FiscalYear    string                  `bson:"fiscal_year"`
	FiscalPeriod  string                  `bson:"fiscal_period"`
	ReportCode    string                  `bson:"report_code"`
	FsDiv         string                  `bson:"fs_div"`
	LineItems     []statementLineDocument `bson:"line_items"`
}

type statementLineDocument struct {
	CanonicalAccount string `bson:"canonical_account"`
	AmountMinor      *int64 `bson:"amount_minor"`
}

type dividendFactDocument struct {
	FactID        int64  `bson:"fact_id"`
	ValueNumber   string `bson:"value_number"`
	FiscalYear    string `bson:"fiscal_year"`
	ReportCode    string `bson:"report_code"`
	RceptNo       string `bson:"rcept_no"`
	FactDate      string `bson:"fact_date"`
	Key           string `bson:"key"`
	Provider      string `bson:"provider"`
	ProviderGroup string `bson:"provider_group"`
	Operation     string `bson:"operation"`
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
		return MongoRepository{}, oops.In("valuation_repository").New("mongodb database is nil")
	}
	return MongoRepository{
		collection:          database.Collection("valuation_snapshots"),
		dailyBarCollection:  database.Collection("daily_bars"),
		statementCollection: database.Collection("financial_statements"),
		factCollection:      database.Collection("company_facts"),
	}, nil
}

func (r MongoRepository) CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (Snapshot, error) {
	errb := oops.In("valuation_repository").With("backend", "mongodb", "company_id", company.Company.ID, "as_of", options.AsOf)
	if company.Company.ID == 0 {
		return Snapshot{}, errb.New("valuation calculation requires canonical company")
	}
	instrument := issuerInstrument(company.Instruments)
	if instrument.InstrumentID == 0 {
		return Snapshot{}, errb.New("valuation calculation requires issuer instrument link")
	}
	price, err := r.selectPrice(ctx, instrument, options.AsOf)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	accounts, err := r.selectLatestAccounts(ctx, company.Company.ID)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	if len(accounts) == 0 {
		return Snapshot{}, errb.New("valuation calculation requires stored financial line items")
	}
	dividend, hasDividend, err := r.selectLatestDividend(ctx, company.Company.ID)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	snapshot := buildSnapshot(company.Company.ID, price, accounts, dividend, hasDividend)
	if err := r.replaceDocument(ctx, snapshotToMongoDocument(company, instrument, snapshot)); err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	return snapshot, nil
}

func (r MongoRepository) ListSnapshots(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Snapshot, error) {
	errb := oops.In("valuation_repository").With("backend", "mongodb", "company_id", company.Company.ID, "as_of", query.AsOf)
	if company.Company.ID == 0 {
		return nil, errb.New("valuation query requires canonical company")
	}
	filter := bson.D{{Key: "company_id", Value: company.Company.ID}}
	if strings.TrimSpace(query.AsOf) != "" && strings.TrimSpace(query.AsOf) != "latest" {
		filter = append(filter, bson.E{Key: "as_of_date", Value: strings.TrimSpace(query.AsOf)})
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "as_of_date", Value: -1}})
	if strings.TrimSpace(query.AsOf) == "" || strings.TrimSpace(query.AsOf) == "latest" {
		findOptions.SetLimit(1)
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errb.Wrapf(err, "select valuation snapshots")
	}
	defer cursor.Close(ctx)
	var documents []snapshotMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode valuation snapshots")
	}
	out := make([]Snapshot, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToSnapshot(document))
	}
	return out, nil
}

func (r MongoRepository) selectPrice(ctx context.Context, instrument companyidentity.InstrumentLink, asOf string) (pricePoint, error) {
	filter := bson.D{
		{Key: "market_key", Value: string(instrument.Market)},
		{Key: "security_type", Value: string(instrument.SecurityType)},
		{Key: "symbol", Value: strings.TrimSpace(instrument.Symbol)},
		{Key: "prices.close", Value: bson.D{{Key: "$ne", Value: ""}}},
		{Key: "volumes.market_cap", Value: bson.D{{Key: "$ne", Value: ""}}},
	}
	if strings.TrimSpace(asOf) != "" && strings.TrimSpace(asOf) != "latest" {
		date, err := parseDateInt(asOf)
		if err != nil {
			return pricePoint{}, err
		}
		filter = append(filter, bson.E{Key: "trading_date", Value: bson.D{{Key: "$lte", Value: formatDateInt(date)}}})
	}
	var document valuationDailyBarDocument
	err := r.dailyBarCollection.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "trading_date", Value: -1}})).Decode(&document)
	if err != nil {
		return pricePoint{}, oops.In("valuation_repository").With("backend", "mongodb", "instrument_id", instrument.InstrumentID, "as_of", asOf).Wrapf(err, "select valuation price point")
	}
	closePrice, err := parseMinorString(document.Prices.Close)
	if err != nil || closePrice <= 0 {
		return pricePoint{}, oops.In("valuation_repository").With("backend", "mongodb", "instrument_id", instrument.InstrumentID, "as_of", asOf).New("valuation price point missing positive close price")
	}
	marketCap, err := parseMinorString(document.Volumes.MarketCap)
	if err != nil || marketCap <= 0 {
		return pricePoint{}, oops.In("valuation_repository").With("backend", "mongodb", "instrument_id", instrument.InstrumentID, "as_of", asOf).New("valuation price point missing positive market cap")
	}
	tradingDate, err := parseDateInt(document.TradingDate)
	if err != nil {
		return pricePoint{}, err
	}
	return pricePoint{
		InstrumentID:    instrument.InstrumentID,
		TradingDate:     tradingDate,
		ClosePriceMinor: closePrice,
		MarketCapMinor:  marketCap,
		PriceScale:      mongoDefaultPriceScale,
		Provider:        document.Source.Provider,
		ProviderGroup:   document.Source.ProviderGroup,
		Operation:       document.Source.Operation,
	}, nil
}

func (r MongoRepository) selectLatestAccounts(ctx context.Context, companyID int64) (map[string]accountValue, error) {
	cursor, err := r.statementCollection.Find(ctx,
		bson.D{{Key: "company_id", Value: companyID}},
		options.Find().SetSort(bson.D{{Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}}),
	)
	if err != nil {
		return nil, oops.In("valuation_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "select valuation financial accounts")
	}
	defer cursor.Close(ctx)
	var documents []statementAccountDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("valuation_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "decode valuation financial accounts")
	}
	out := make(map[string]accountValue)
	for _, document := range documents {
		for _, line := range document.LineItems {
			account := strings.TrimSpace(line.CanonicalAccount)
			if _, exists := out[account]; exists || line.AmountMinor == nil || !valuationAccount(account) {
				continue
			}
			out[account] = accountValue{
				StatementID:      document.StatementID,
				CanonicalAccount: account,
				AmountMinor:      *line.AmountMinor,
				FiscalYear:       document.FiscalYear,
				FiscalPeriod:     document.FiscalPeriod,
				ReportCode:       document.ReportCode,
				FsDiv:            document.FsDiv,
				RceptNo:          document.RceptNo,
				Provider:         document.Provider,
				ProviderGroup:    document.ProviderGroup,
				Operation:        document.Operation,
			}
		}
	}
	return out, nil
}

func (r MongoRepository) selectLatestDividend(ctx context.Context, companyID int64) (dividendValue, bool, error) {
	cursor, err := r.factCollection.Find(ctx,
		bson.D{{Key: "company_id", Value: companyID}, {Key: "fact_type", Value: "dividend"}, {Key: "value_number", Value: bson.D{{Key: "$ne", Value: ""}}}},
		options.Find().SetSort(bson.D{{Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}, {Key: "fact_date", Value: -1}, {Key: "fact_id", Value: -1}}),
	)
	if err != nil {
		return dividendValue{}, false, oops.In("valuation_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "select valuation dividend facts")
	}
	defer cursor.Close(ctx)
	var documents []dividendFactDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return dividendValue{}, false, oops.In("valuation_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "decode valuation dividend facts")
	}
	for _, document := range documents {
		if !isCashDividendTotalKey(document.Key) {
			continue
		}
		amount, err := strconv.ParseInt(strings.TrimSpace(document.ValueNumber), 10, 64)
		if err != nil || amount <= 0 {
			continue
		}
		return dividendValue{
			FactID:      document.FactID,
			AmountMinor: amount,
			FiscalYear:  document.FiscalYear,
			ReportCode:  document.ReportCode,
			RceptNo:     document.RceptNo,
			FactDate:    document.FactDate,
			Key:         document.Key,
			Provider:    document.Provider,
			Group:       document.ProviderGroup,
			Operation:   document.Operation,
		}, true, nil
	}
	return dividendValue{}, false, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next snapshotMongoDocument) error {
	var current snapshotMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("valuation_repository").With("backend", "mongodb", "id", next.ID).Wrapf(insertErr, "insert valuation snapshot mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("valuation_repository").With("backend", "mongodb", "id", next.ID).Wrapf(err, "read valuation snapshot mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("valuation_repository").With("backend", "mongodb", "id", next.ID, "revision", current.Revision).Wrapf(err, "replace valuation snapshot mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("valuation_snapshots", next.ID, current.Revision)
	}
	return nil
}

func snapshotToMongoDocument(company companyidentity.InspectResult, instrument companyidentity.InstrumentLink, snapshot Snapshot) snapshotMongoDocument {
	now := storagemongodb.ISOTimeNow()
	return snapshotMongoDocument{
		ID:                  valuationSnapshotMongoID(snapshot.CompanyID, snapshot.InstrumentID, snapshot.AsOfDate, snapshot.MetricSourceVersion),
		SchemaVersion:       storagemongodb.SchemaVersion1,
		Revision:            1,
		CreatedAt:           now,
		UpdatedAt:           now,
		CompanyID:           snapshot.CompanyID,
		InstrumentID:        snapshot.InstrumentID,
		Company:             companySnapshotFromInspect(company),
		Instrument:          instrumentSnapshotFromLink(instrument),
		AsOfDate:            snapshot.AsOfDate,
		SourcePriceDate:     snapshot.SourcePriceDate,
		MarketCapMinor:      snapshot.MarketCapMinor,
		ClosePriceMinor:     snapshot.ClosePriceMinor,
		SharesOutstanding:   snapshot.SharesOutstanding,
		PerBP:               snapshot.PerBP,
		PbrBP:               snapshot.PbrBP,
		PsrBP:               snapshot.PsrBP,
		EpsMinor:            snapshot.EpsMinor,
		BpsMinor:            snapshot.BpsMinor,
		DividendYieldBP:     snapshot.DividendYieldBP,
		MetricSourceVersion: snapshot.MetricSourceVersion,
		Provenance:          cloneMap(snapshot.Provenance),
		Uncomputable:        cloneMap(snapshot.Uncomputable),
	}
}

func mongoDocumentToSnapshot(document snapshotMongoDocument) Snapshot {
	return Snapshot{
		CompanyID:           document.CompanyID,
		InstrumentID:        document.InstrumentID,
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
		Provenance:          cloneMap(document.Provenance),
		Uncomputable:        cloneMap(document.Uncomputable),
	}
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
		if link.RelationType == companyidentity.RelationTypeIssuer && link.InstrumentID != 0 {
			return link
		}
	}
	if len(links) > 0 {
		return links[0]
	}
	return companyidentity.InstrumentLink{}
}

func valuationSnapshotMongoID(parts ...any) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, "valuation_snapshots")
	for _, part := range parts {
		values = append(values, strings.TrimSpace(toString(part)))
	}
	return strings.Join(values, ":")
}

func parseMinorString(value string) (int64, error) {
	normalized := strings.NewReplacer(",", "", "_", "", " ", "").Replace(strings.TrimSpace(value))
	if normalized == "" {
		return 0, oops.In("valuation_repository").New("minor amount is empty")
	}
	return strconv.ParseInt(normalized, 10, 64)
}

func valuationAccount(account string) bool {
	switch account {
	case "revenue", "net_income", "equity":
		return true
	default:
		return false
	}
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

var _ = financialmetrics.FormulaVersion
