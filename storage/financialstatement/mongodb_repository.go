package financialstatement

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
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

type statementMongoDocument struct {
	ID                             string                  `bson:"_id"`
	SchemaVersion                  string                  `bson:"schema_version"`
	Revision                       int64                   `bson:"revision"`
	CreatedAt                      time.Time               `bson:"created_at"`
	UpdatedAt                      time.Time               `bson:"updated_at"`
	StatementID                    int64                   `bson:"statement_id"`
	CompanyID                      int64                   `bson:"company_id"`
	InstrumentID                   int64                   `bson:"instrument_id"`
	Company                        companySnapshotDocument `bson:"company"`
	Instrument                     instrumentSnapshot      `bson:"instrument"`
	Provider                       string                  `bson:"provider"`
	ProviderGroup                  string                  `bson:"provider_group"`
	Operation                      string                  `bson:"operation"`
	ProviderCompanyIdentifierType  string                  `bson:"provider_company_identifier_type,omitempty"`
	ProviderCompanyIdentifierValue string                  `bson:"provider_company_identifier_value,omitempty"`
	RceptNo                        string                  `bson:"rcept_no,omitempty"`
	FiscalYear                     string                  `bson:"fiscal_year"`
	FiscalPeriod                   string                  `bson:"fiscal_period,omitempty"`
	ReportCode                     string                  `bson:"report_code,omitempty"`
	FsDiv                          string                  `bson:"fs_div,omitempty"`
	StatementType                  string                  `bson:"statement_type"`
	ReportedAt                     string                  `bson:"reported_at,omitempty"`
	SourcePayloadRef               string                  `bson:"source_payload_ref,omitempty"`
	LineItems                      []lineItemMongoDocument `bson:"line_items"`
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

type lineItemMongoDocument struct {
	AccountID        string            `bson:"account_id,omitempty"`
	AccountName      string            `bson:"account_name"`
	CanonicalAccount string            `bson:"canonical_account,omitempty"`
	AmountMinor      *int64            `bson:"amount_minor,omitempty"`
	CurrencyCode     string            `bson:"currency_code,omitempty"`
	Unit             string            `bson:"unit,omitempty"`
	RawAmount        string            `bson:"raw_amount,omitempty"`
	PeriodName       string            `bson:"period_name,omitempty"`
	Ord              int               `bson:"ord"`
	Extensions       map[string]string `bson:"extensions,omitempty"`
	CreatedAt        time.Time         `bson:"created_at"`
	UpdatedAt        time.Time         `bson:"updated_at"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("financial_statement_repository").New("mongodb database is nil")
	}
	return MongoRepository{collection: database.Collection("financial_statements")}, nil
}

func (r MongoRepository) UpsertStatements(ctx context.Context, company companyidentity.InspectResult, statements []financialsrole.Statement) (UpsertResult, error) {
	errb := oops.In("financial_statement_repository").With("backend", "mongodb", "company_id", company.Company.ID, "statements", len(statements))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("financial statement upsert requires canonical company")
	}
	var result UpsertResult
	for _, bucket := range statementBuckets(statements) {
		next, err := statementToMongoDocument(company, bucket)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if err := r.replaceDocument(ctx, next); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.StatementsWritten++
		result.LineItemsWritten += len(bucket.Lines)
	}
	return result, nil
}

func (r MongoRepository) ListStatements(ctx context.Context, company companyidentity.InspectResult, query Query) ([]financialsrole.Statement, error) {
	errb := oops.In("financial_statement_repository").With("backend", "mongodb", "company_id", company.Company.ID, "fiscal_year", query.FiscalYear, "period", query.Period, "statement", query.Statement)
	if company.Company.ID == 0 {
		return nil, errb.New("financial statement query requires canonical company")
	}
	filter := bson.D{{Key: "company_id", Value: company.Company.ID}}
	if query.FiscalYear != "" {
		filter = append(filter, bson.E{Key: "fiscal_year", Value: strings.TrimSpace(query.FiscalYear)})
	}
	if query.Period != "" {
		filter = append(filter, bson.E{Key: "fiscal_period", Value: string(query.Period)})
	}
	if query.Statement != "" {
		filter = append(filter, bson.E{Key: "statement_type", Value: string(query.Statement)})
	}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "fiscal_year", Value: -1}, {Key: "report_code", Value: -1}, {Key: "statement_type", Value: 1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "select financial statements")
	}
	defer cursor.Close(ctx)
	var documents []statementMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode financial statements")
	}
	out := make([]financialsrole.Statement, 0, len(documents))
	for _, document := range documents {
		out = append(out, mongoDocumentToStatement(document, company, query.Limit))
	}
	return out, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next statementMongoDocument) error {
	var current statementMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("financial_statement_repository").With("backend", "mongodb", "statement_id", next.StatementID).Wrapf(insertErr, "insert financial statement mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("financial_statement_repository").With("backend", "mongodb", "statement_id", next.StatementID).Wrapf(err, "read financial statement mongodb document")
	}
	next.StatementID = current.StatementID
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	next.LineItems = mergeLineItems(current.LineItems, next.LineItems)
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("financial_statement_repository").With("backend", "mongodb", "statement_id", next.StatementID, "revision", current.Revision).Wrapf(err, "replace financial statement mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("financial_statements", next.ID, current.Revision)
	}
	return nil
}

func statementToMongoDocument(company companyidentity.InspectResult, bucket bucket) (statementMongoDocument, error) {
	statement := bucket.Statement
	instrument := firstInstrument(company.Instruments)
	identifierType, identifierValue := providerCompanyIdentifier(company.Identifiers)
	reportCode := firstNonEmpty(statement.Extensions["reprt_code"], statement.FiscalPeriod)
	fsDiv := firstNonEmpty(statement.Extensions["fs_div"])
	rceptNo := firstNonEmpty(statement.Extensions["rcept_no"])
	providerID := string(statement.Provider)
	providerGroup := string(statement.Group)
	operation := string(statement.Operation)
	fiscalYear := strings.TrimSpace(statement.FiscalYear)
	statementType := string(statement.Statement)
	if providerID == "" || providerGroup == "" || operation == "" || fiscalYear == "" || statementType == "" {
		return statementMongoDocument{}, oops.In("financial_statement_repository").With("company_id", company.Company.ID).New("financial statement missing natural key")
	}
	now := storagemongodb.ISOTimeNow()
	id := financialStatementMongoID(company.Company.ID, instrument.InstrumentID, providerID, providerGroup, operation, rceptNo, fiscalYear, reportCode, fsDiv, statementType)
	lineItems := make([]lineItemMongoDocument, 0, len(bucket.Lines))
	for _, line := range bucket.Lines {
		item, err := lineItemToMongoDocument(line, now)
		if err != nil {
			return statementMongoDocument{}, err
		}
		lineItems = append(lineItems, item)
	}
	sortLineItems(lineItems)
	return statementMongoDocument{
		ID:                             id,
		SchemaVersion:                  storagemongodb.SchemaVersion1,
		Revision:                       1,
		CreatedAt:                      now,
		UpdatedAt:                      now,
		StatementID:                    stableStatementID(id),
		CompanyID:                      company.Company.ID,
		InstrumentID:                   instrument.InstrumentID,
		Company:                        companySnapshotFromInspect(company),
		Instrument:                     instrumentSnapshotFromLink(instrument),
		Provider:                       providerID,
		ProviderGroup:                  providerGroup,
		Operation:                      operation,
		ProviderCompanyIdentifierType:  identifierType,
		ProviderCompanyIdentifierValue: identifierValue,
		RceptNo:                        rceptNo,
		FiscalYear:                     fiscalYear,
		FiscalPeriod:                   string(statement.Period),
		ReportCode:                     reportCode,
		FsDiv:                          fsDiv,
		StatementType:                  statementType,
		ReportedAt:                     strings.TrimSpace(statement.ReportedAt),
		SourcePayloadRef:               strings.TrimSpace(statement.Extensions["source_payload_ref"]),
		LineItems:                      lineItems,
	}, nil
}

func lineItemToMongoDocument(line financialsrole.LineItem, now time.Time) (lineItemMongoDocument, error) {
	extensions := cloneMap(line.Extensions)
	if extensions == nil {
		extensions = map[string]string{}
	}
	accountName := strings.TrimSpace(line.AccountName)
	if accountName == "" {
		return lineItemMongoDocument{}, oops.In("financial_statement_repository").With("account_id", line.AccountID).New("financial line item missing account name")
	}
	return lineItemMongoDocument{
		AccountID:        strings.TrimSpace(line.AccountID),
		AccountName:      accountName,
		CanonicalAccount: canonicalAccount(line),
		AmountMinor:      parseAmountMinor(line.Value),
		CurrencyCode:     strings.TrimSpace(line.Currency),
		Unit:             strings.TrimSpace(line.Unit),
		RawAmount:        strings.TrimSpace(line.Value),
		PeriodName:       strings.TrimSpace(line.Extensions["thstrm_nm"]),
		Ord:              parseOrd(line.Extensions["ord"]),
		Extensions:       extensions,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func mongoDocumentToStatement(document statementMongoDocument, company companyidentity.InspectResult, limit int) financialsrole.Statement {
	lines := append([]lineItemMongoDocument(nil), document.LineItems...)
	sortLineItems(lines)
	if limit > 0 && len(lines) > limit {
		lines = lines[:limit]
	}
	statement := financialsrole.Statement{
		Statement:    financialsrole.StatementType(document.StatementType),
		Symbol:       firstInstrumentSymbol(company.Instruments),
		Name:         company.Company.Name,
		FiscalYear:   document.FiscalYear,
		FiscalPeriod: document.ReportCode,
		Period:       financialsrole.PeriodType(document.FiscalPeriod),
		ReportedAt:   document.ReportedAt,
		Lines:        make([]financialsrole.LineItem, 0, len(lines)),
		Extensions: map[string]string{
			"provider_company_identifier_type":  document.ProviderCompanyIdentifierType,
			"provider_company_identifier_value": document.ProviderCompanyIdentifierValue,
			"rcept_no":                          document.RceptNo,
			"reprt_code":                        document.ReportCode,
			"fs_div":                            document.FsDiv,
			"statement_id":                      strconv.FormatInt(document.StatementID, 10),
		},
		Provider:     provider.ProviderID(document.Provider),
		Group:        provider.GroupID(document.ProviderGroup),
		Operation:    provider.OperationID(document.Operation),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
	}
	for _, line := range lines {
		statement.Lines = append(statement.Lines, mongoLineItemToCanonical(line))
		if statement.Currency == "" && line.CurrencyCode != "" {
			statement.Currency = line.CurrencyCode
		}
	}
	return statement
}

func mongoLineItemToCanonical(document lineItemMongoDocument) financialsrole.LineItem {
	extensions := cloneMap(document.Extensions)
	if extensions == nil {
		extensions = map[string]string{}
	}
	if document.CanonicalAccount != "" {
		extensions["canonical_account"] = document.CanonicalAccount
	}
	return financialsrole.LineItem{
		AccountID:   document.AccountID,
		AccountName: document.AccountName,
		Value:       document.RawAmount,
		Currency:    document.CurrencyCode,
		Unit:        document.Unit,
		Extensions:  extensions,
	}
}

func mergeLineItems(current []lineItemMongoDocument, next []lineItemMongoDocument) []lineItemMongoDocument {
	items := append([]lineItemMongoDocument(nil), current...)
	for _, incoming := range next {
		replaced := false
		for i := range items {
			if sameLineItem(items[i], incoming) {
				incoming.CreatedAt = items[i].CreatedAt
				items[i] = incoming
				replaced = true
				break
			}
		}
		if !replaced {
			items = append(items, incoming)
		}
	}
	sortLineItems(items)
	return items
}

func sameLineItem(left lineItemMongoDocument, right lineItemMongoDocument) bool {
	return left.AccountID == right.AccountID &&
		left.AccountName == right.AccountName &&
		left.PeriodName == right.PeriodName &&
		left.Ord == right.Ord
}

func sortLineItems(items []lineItemMongoDocument) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Ord != items[j].Ord {
			return items[i].Ord < items[j].Ord
		}
		return items[i].AccountName < items[j].AccountName
	})
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

func firstInstrument(links []companyidentity.InstrumentLink) companyidentity.InstrumentLink {
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

func financialStatementMongoID(parts ...any) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, "financial_statements")
	for _, part := range parts {
		values = append(values, strings.TrimSpace(toString(part)))
	}
	return strings.Join(values, ":")
}

func stableStatementID(value string) int64 {
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
