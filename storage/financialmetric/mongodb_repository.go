package financialmetric

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/awuzag/mwosa/packages/financialmetrics"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage/companyidentity"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection          *mongo.Collection
	statementCollection *mongo.Collection
}

type metricMongoDocument struct {
	ID                 string                  `bson:"_id"`
	SchemaVersion      string                  `bson:"schema_version"`
	Revision           int64                   `bson:"revision"`
	CreatedAt          time.Time               `bson:"created_at"`
	UpdatedAt          time.Time               `bson:"updated_at"`
	CompanyID          int64                   `bson:"company_id"`
	InstrumentID       int64                   `bson:"instrument_id"`
	StatementID        int64                   `bson:"statement_id,omitempty"`
	Company            companySnapshotDocument `bson:"company"`
	Instrument         instrumentSnapshot      `bson:"instrument"`
	Metric             string                  `bson:"metric"`
	FiscalYear         string                  `bson:"fiscal_year"`
	FiscalPeriod       string                  `bson:"fiscal_period,omitempty"`
	AsOfDate           string                  `bson:"as_of_date,omitempty"`
	ValueDecimal       string                  `bson:"value_decimal,omitempty"`
	ValueBP            *int64                  `bson:"value_bp,omitempty"`
	ValueMinor         *int64                  `bson:"value_minor,omitempty"`
	FormulaVersion     string                  `bson:"formula_version"`
	UncomputableReason string                  `bson:"uncomputable_reason,omitempty"`
	Provenance         map[string]any          `bson:"provenance,omitempty"`
}

type statementMetricDocument struct {
	StatementID   int64                    `bson:"statement_id"`
	InstrumentID  int64                    `bson:"instrument_id"`
	Provider      string                   `bson:"provider"`
	ProviderGroup string                   `bson:"provider_group"`
	Operation     string                   `bson:"operation"`
	RceptNo       string                   `bson:"rcept_no"`
	FiscalYear    string                   `bson:"fiscal_year"`
	FiscalPeriod  string                   `bson:"fiscal_period"`
	ReportCode    string                   `bson:"report_code"`
	FsDiv         string                   `bson:"fs_div"`
	ReportedAt    string                   `bson:"reported_at"`
	LineItems     []lineItemMetricDocument `bson:"line_items"`
}

type lineItemMetricDocument struct {
	CanonicalAccount string `bson:"canonical_account"`
	AmountMinor      *int64 `bson:"amount_minor"`
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
		return MongoRepository{}, oops.In("financial_metric_repository").New("mongodb database is nil")
	}
	return MongoRepository{
		collection:          database.Collection("financial_metrics"),
		statementCollection: database.Collection("financial_statements"),
	}, nil
}

func (r MongoRepository) CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (UpsertResult, error) {
	errb := oops.In("financial_metric_repository").With("backend", "mongodb", "company_id", company.Company.ID, "window_years", options.WindowYears, "period", options.Period)
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("financial metric calculation requires canonical company")
	}
	values, err := r.selectAccountValues(ctx, company.Company.ID, options.Period)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	if len(values) == 0 {
		return UpsertResult{}, errb.New("financial metric calculation requires stored financial line items")
	}
	metrics := financialmetrics.Calculate(values, options.WindowYears)
	result := UpsertResult{MetricsCalculated: len(metrics)}
	for _, metric := range metrics {
		if metric.UncomputableReason != "" {
			result.Uncomputable++
		}
		document, err := metricToMongoDocument(company, metric)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if err := r.replaceDocument(ctx, document); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.MetricsWritten++
	}
	return result, nil
}

func (r MongoRepository) ListMetrics(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Metric, error) {
	errb := oops.In("financial_metric_repository").With("backend", "mongodb", "company_id", company.Company.ID, "window_years", query.WindowYears, "period", query.Period)
	if company.Company.ID == 0 {
		return nil, errb.New("financial metric query requires canonical company")
	}
	filter := bson.D{{Key: "company_id", Value: company.Company.ID}}
	if query.Period != "" {
		filter = append(filter, bson.E{Key: "fiscal_period", Value: string(query.Period)})
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "fiscal_year", Value: -1}, {Key: "metric", Value: 1}})
	if query.WindowYears > 0 {
		findOptions.SetLimit(int64(query.WindowYears * 10))
	}
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errb.Wrapf(err, "select financial metrics")
	}
	defer cursor.Close(ctx)
	var documents []metricMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode financial metrics")
	}
	out := make([]Metric, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToMetric(document))
	}
	return out, nil
}

func (r MongoRepository) selectAccountValues(ctx context.Context, companyID int64, period financials.PeriodType) ([]financialmetrics.AccountValue, error) {
	filter := bson.D{{Key: "company_id", Value: companyID}}
	if period != "" {
		filter = append(filter, bson.E{Key: "fiscal_period", Value: string(period)})
	}
	cursor, err := r.statementCollection.Find(ctx, filter)
	if err != nil {
		return nil, oops.In("financial_metric_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "select financial metric source statements")
	}
	defer cursor.Close(ctx)
	var documents []statementMetricDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("financial_metric_repository").With("backend", "mongodb", "company_id", companyID).Wrapf(err, "decode financial metric source statements")
	}
	values := make([]financialmetrics.AccountValue, 0)
	for _, document := range documents {
		asOfDate := strings.TrimSpace(document.ReportedAt)
		if asOfDate == "" {
			asOfDate = inferredAsOfDate(document.FiscalYear, document.ReportCode)
		}
		for _, line := range document.LineItems {
			if line.AmountMinor == nil || strings.TrimSpace(line.CanonicalAccount) == "" {
				continue
			}
			values = append(values, financialmetrics.AccountValue{
				StatementID:      document.StatementID,
				InstrumentID:     document.InstrumentID,
				Provider:         document.Provider,
				ProviderGroup:    document.ProviderGroup,
				Operation:        document.Operation,
				ReportCode:       document.ReportCode,
				FsDiv:            document.FsDiv,
				RceptNo:          document.RceptNo,
				FiscalYear:       document.FiscalYear,
				FiscalPeriod:     document.FiscalPeriod,
				AsOfDate:         asOfDate,
				CanonicalAccount: line.CanonicalAccount,
				AmountMinor:      *line.AmountMinor,
			})
		}
	}
	return values, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next metricMongoDocument) error {
	var current metricMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("financial_metric_repository").With("backend", "mongodb", "id", next.ID).Wrapf(insertErr, "insert financial metric mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("financial_metric_repository").With("backend", "mongodb", "id", next.ID).Wrapf(err, "read financial metric mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("financial_metric_repository").With("backend", "mongodb", "id", next.ID, "revision", current.Revision).Wrapf(err, "replace financial metric mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("financial_metrics", next.ID, current.Revision)
	}
	return nil
}

func metricToMongoDocument(company companyidentity.InspectResult, metric financialmetrics.Metric) (metricMongoDocument, error) {
	if metric.Metric == "" || metric.FiscalYear == "" || metric.FormulaVersion == "" {
		return metricMongoDocument{}, oops.In("financial_metric_repository").With("company_id", company.Company.ID, "metric", metric.Metric).New("financial metric missing natural key")
	}
	now := storagemongodb.ISOTimeNow()
	instrument := firstInstrument(company.Instruments, metric.InstrumentID)
	id := financialMetricMongoID(company.Company.ID, metric.InstrumentID, metric.Metric, metric.FiscalYear, metric.FiscalPeriod, metric.AsOfDate, metric.FormulaVersion)
	provenance := cloneMap(metric.Provenance)
	if provenance == nil {
		provenance = map[string]any{}
	}
	return metricMongoDocument{
		ID:                 id,
		SchemaVersion:      storagemongodb.SchemaVersion1,
		Revision:           1,
		CreatedAt:          now,
		UpdatedAt:          now,
		CompanyID:          company.Company.ID,
		InstrumentID:       metric.InstrumentID,
		StatementID:        metric.StatementID,
		Company:            companySnapshotFromInspect(company),
		Instrument:         instrumentSnapshotFromLink(instrument),
		Metric:             metric.Metric,
		FiscalYear:         metric.FiscalYear,
		FiscalPeriod:       metric.FiscalPeriod,
		AsOfDate:           metric.AsOfDate,
		ValueDecimal:       metric.ValueDecimal,
		ValueBP:            metric.ValueBP,
		ValueMinor:         metric.ValueMinor,
		FormulaVersion:     metric.FormulaVersion,
		UncomputableReason: metric.UncomputableReason,
		Provenance:         provenance,
	}, nil
}

func mongoDocumentToMetric(document metricMongoDocument) Metric {
	return Metric{
		CompanyID:          document.CompanyID,
		InstrumentID:       document.InstrumentID,
		StatementID:        document.StatementID,
		Metric:             document.Metric,
		FiscalYear:         document.FiscalYear,
		FiscalPeriod:       financials.PeriodType(document.FiscalPeriod),
		AsOfDate:           document.AsOfDate,
		ValueDecimal:       document.ValueDecimal,
		ValueBP:            document.ValueBP,
		ValueMinor:         document.ValueMinor,
		FormulaVersion:     document.FormulaVersion,
		UncomputableReason: document.UncomputableReason,
		Provenance:         cloneMap(document.Provenance),
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

func firstInstrument(links []companyidentity.InstrumentLink, instrumentID int64) companyidentity.InstrumentLink {
	for _, link := range links {
		if instrumentID != 0 && link.InstrumentID == instrumentID {
			return link
		}
	}
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer {
			return link
		}
	}
	if len(links) > 0 {
		return links[0]
	}
	return companyidentity.InstrumentLink{}
}

func financialMetricMongoID(parts ...any) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, "financial_metrics")
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
