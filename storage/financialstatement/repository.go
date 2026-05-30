package financialstatement

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type Repository struct {
	database *storage.Database
}

type UpsertResult struct {
	StatementsWritten int `json:"statements_written" csv:"statements_written"`
	LineItemsWritten  int `json:"line_items_written" csv:"line_items_written"`
}

type Query struct {
	FiscalYear string
	Period     financialsrole.PeriodType
	Statement  financialsrole.StatementType
	Limit      int
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("financial_statement_repository").New("financial statement repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertStatements(ctx context.Context, company companyidentity.InspectResult, statements []financialsrole.Statement) (UpsertResult, error) {
	errb := oops.In("financial_statement_repository").With("company_id", company.Company.ID, "statements", len(statements))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("financial statement upsert requires canonical company")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, errb.Wrapf(err, "begin financial statement sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	nowMS := time.Now().UTC().UnixMilli()
	var result UpsertResult
	for _, bucket := range statementBuckets(statements) {
		statement, err := upsertStatement(ctx, tx, company, bucket, nowMS)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.StatementsWritten++
		for _, line := range bucket.Lines {
			if err := upsertLineItem(ctx, tx, statement.ID, line, nowMS); err != nil {
				return UpsertResult{}, errb.Wrap(err)
			}
			result.LineItemsWritten++
		}
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, errb.Wrapf(err, "commit financial statement sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r Repository) ListStatements(ctx context.Context, company companyidentity.InspectResult, query Query) ([]financialsrole.Statement, error) {
	errb := oops.In("financial_statement_repository").With("company_id", company.Company.ID, "fiscal_year", query.FiscalYear, "period", query.Period, "statement", query.Statement)
	if company.Company.ID == 0 {
		return nil, errb.New("financial statement query requires canonical company")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	statementRows, err := selectStatements(ctx, client, company.Company.ID, query)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	out := make([]financialsrole.Statement, 0, len(statementRows))
	for _, row := range statementRows {
		lines, err := selectLines(ctx, client, row.ID, query.Limit)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		out = append(out, rowToStatement(row, company, lines))
	}
	return out, nil
}

type bucket struct {
	Statement financialsrole.Statement
	Lines     []financialsrole.LineItem
}

func statementBuckets(statements []financialsrole.Statement) []bucket {
	seen := make(map[string]int)
	buckets := make([]bucket, 0)
	for _, statement := range statements {
		for _, line := range statement.Lines {
			statementType := statementTypeForLine(line, statement.Statement)
			reportCode := firstNonEmpty(line.Extensions["reprt_code"], statement.Extensions["reprt_code"], statement.FiscalPeriod)
			fsDiv := firstNonEmpty(line.Extensions["fs_div"], statement.Extensions["fs_div"])
			rceptNo := firstNonEmpty(line.Extensions["rcept_no"], statement.Extensions["rcept_no"])
			key := strings.Join([]string{
				string(statement.Provider),
				string(statement.Group),
				string(statement.Operation),
				statement.FiscalYear,
				reportCode,
				fsDiv,
				string(statementType),
				rceptNo,
			}, "\x00")
			index, ok := seen[key]
			if !ok {
				copyStatement := statement
				copyStatement.Statement = statementType
				copyStatement.FiscalPeriod = reportCode
				copyStatement.Extensions = cloneMap(statement.Extensions)
				if copyStatement.Extensions == nil {
					copyStatement.Extensions = make(map[string]string)
				}
				copyStatement.Extensions["reprt_code"] = reportCode
				copyStatement.Extensions["fs_div"] = fsDiv
				copyStatement.Extensions["rcept_no"] = rceptNo
				copyStatement.Lines = nil
				buckets = append(buckets, bucket{Statement: copyStatement})
				index = len(buckets) - 1
				seen[key] = index
			}
			buckets[index].Lines = append(buckets[index].Lines, line)
		}
	}
	return buckets
}

func upsertStatement(ctx context.Context, tx bun.Tx, company companyidentity.InspectResult, bucket bucket, nowMS int64) (storage.FinancialStatementV1Row, error) {
	statement := bucket.Statement
	instrumentID := firstInstrumentID(company.Instruments)
	identifierType, identifierValue := providerCompanyIdentifier(company.Identifiers)
	reportCode := firstNonEmpty(statement.Extensions["reprt_code"], statement.FiscalPeriod)
	fsDiv := firstNonEmpty(statement.Extensions["fs_div"])
	rceptNo := firstNonEmpty(statement.Extensions["rcept_no"])
	row := storage.FinancialStatementV1Row{
		CompanyID:                      company.Company.ID,
		InstrumentID:                   instrumentID,
		Provider:                       string(statement.Provider),
		ProviderGroup:                  string(statement.Group),
		Operation:                      string(statement.Operation),
		ProviderCompanyIdentifierType:  identifierType,
		ProviderCompanyIdentifierValue: identifierValue,
		RceptNo:                        rceptNo,
		FiscalYear:                     strings.TrimSpace(statement.FiscalYear),
		FiscalPeriod:                   string(statement.Period),
		ReportCode:                     reportCode,
		FsDiv:                          fsDiv,
		StatementType:                  string(statement.Statement),
		ReportedAt:                     strings.TrimSpace(statement.ReportedAt),
		SourcePayloadRef:               strings.TrimSpace(statement.Extensions["source_payload_ref"]),
		CreatedAtMS:                    nowMS,
		UpdatedAtMS:                    nowMS,
	}
	if row.Provider == "" || row.ProviderGroup == "" || row.Operation == "" || row.FiscalYear == "" || row.StatementType == "" {
		return storage.FinancialStatementV1Row{}, oops.In("financial_statement_repository").With("company_id", row.CompanyID).New("financial statement missing natural key")
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (company_id, instrument_id, provider, provider_group, operation, rcept_no, fiscal_year, report_code, fs_div, statement_type) DO UPDATE").
		Set("provider_company_identifier_type = EXCLUDED.provider_company_identifier_type").
		Set("provider_company_identifier_value = EXCLUDED.provider_company_identifier_value").
		Set("fiscal_period = EXCLUDED.fiscal_period").
		Set("reported_at = EXCLUDED.reported_at").
		Set("source_payload_ref = EXCLUDED.source_payload_ref").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.FinancialStatementV1Row{}, oops.In("financial_statement_repository").With("company_id", row.CompanyID, "statement_type", row.StatementType).Wrapf(err, "upsert financial statement")
	}
	var stored storage.FinancialStatementV1Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("company_id = ?", row.CompanyID).
		Where("instrument_id = ?", row.InstrumentID).
		Where("provider = ?", row.Provider).
		Where("provider_group = ?", row.ProviderGroup).
		Where("operation = ?", row.Operation).
		Where("rcept_no = ?", row.RceptNo).
		Where("fiscal_year = ?", row.FiscalYear).
		Where("report_code = ?", row.ReportCode).
		Where("fs_div = ?", row.FsDiv).
		Where("statement_type = ?", row.StatementType).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.FinancialStatementV1Row{}, oops.In("financial_statement_repository").With("company_id", row.CompanyID, "statement_type", row.StatementType).Wrapf(err, "select financial statement")
	}
	return stored, nil
}

func upsertLineItem(ctx context.Context, tx bun.Tx, statementID int64, line financialsrole.LineItem, nowMS int64) error {
	extensionsJSON, err := json.Marshal(line.Extensions)
	if err != nil {
		return oops.In("financial_statement_repository").With("statement_id", statementID, "account_id", line.AccountID).Wrapf(err, "encode financial line item extensions")
	}
	row := storage.FinancialLineItemV1Row{
		StatementID:      statementID,
		AccountID:        strings.TrimSpace(line.AccountID),
		AccountName:      strings.TrimSpace(line.AccountName),
		CanonicalAccount: canonicalAccount(line),
		AmountMinor:      parseAmountMinor(line.Value),
		CurrencyCode:     strings.TrimSpace(line.Currency),
		Unit:             strings.TrimSpace(line.Unit),
		RawAmount:        strings.TrimSpace(line.Value),
		PeriodName:       strings.TrimSpace(line.Extensions["thstrm_nm"]),
		Ord:              parseOrd(line.Extensions["ord"]),
		ExtensionsJSON:   string(extensionsJSON),
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
	}
	if row.AccountName == "" {
		return oops.In("financial_statement_repository").With("statement_id", statementID, "account_id", row.AccountID).New("financial line item missing account name")
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (statement_id, account_id, account_name, period_name, ord) DO UPDATE").
		Set("canonical_account = EXCLUDED.canonical_account").
		Set("amount_minor = EXCLUDED.amount_minor").
		Set("currency_code = EXCLUDED.currency_code").
		Set("unit = EXCLUDED.unit").
		Set("raw_amount = EXCLUDED.raw_amount").
		Set("extensions_json = EXCLUDED.extensions_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("financial_statement_repository").With("statement_id", statementID, "account_id", row.AccountID, "account_name", row.AccountName).Wrapf(err, "upsert financial line item")
	}
	return nil
}

func selectStatements(ctx context.Context, db *bun.DB, companyID int64, query Query) ([]storage.FinancialStatementV1Row, error) {
	rows := make([]storage.FinancialStatementV1Row, 0)
	selectQuery := db.NewSelect().
		Model(&rows).
		Where("company_id = ?", companyID).
		Order("fiscal_year DESC", "report_code DESC", "statement_type ASC")
	if query.FiscalYear != "" {
		selectQuery = selectQuery.Where("fiscal_year = ?", strings.TrimSpace(query.FiscalYear))
	}
	if query.Period != "" {
		selectQuery = selectQuery.Where("fiscal_period = ?", string(query.Period))
	}
	if query.Statement != "" {
		selectQuery = selectQuery.Where("statement_type = ?", string(query.Statement))
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, oops.In("financial_statement_repository").With("company_id", companyID).Wrapf(err, "select financial statements")
	}
	return rows, nil
}

func selectLines(ctx context.Context, db *bun.DB, statementID int64, limit int) ([]storage.FinancialLineItemV1Row, error) {
	rows := make([]storage.FinancialLineItemV1Row, 0)
	selectQuery := db.NewSelect().
		Model(&rows).
		Where("statement_id = ?", statementID).
		Order("ord ASC", "account_name ASC")
	if limit > 0 {
		selectQuery = selectQuery.Limit(limit)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, oops.In("financial_statement_repository").With("statement_id", statementID).Wrapf(err, "select financial line items")
	}
	return rows, nil
}

func rowToStatement(row storage.FinancialStatementV1Row, company companyidentity.InspectResult, lines []storage.FinancialLineItemV1Row) financialsrole.Statement {
	statement := financialsrole.Statement{
		Statement:    financialsrole.StatementType(row.StatementType),
		Symbol:       firstInstrumentSymbol(company.Instruments),
		Name:         company.Company.Name,
		FiscalYear:   row.FiscalYear,
		FiscalPeriod: row.ReportCode,
		Period:       financialsrole.PeriodType(row.FiscalPeriod),
		ReportedAt:   row.ReportedAt,
		Lines:        make([]financialsrole.LineItem, 0, len(lines)),
		Extensions: map[string]string{
			"provider_company_identifier_type":  row.ProviderCompanyIdentifierType,
			"provider_company_identifier_value": row.ProviderCompanyIdentifierValue,
			"rcept_no":                          row.RceptNo,
			"reprt_code":                        row.ReportCode,
			"fs_div":                            row.FsDiv,
			"statement_id":                      strconv.FormatInt(row.ID, 10),
		},
		Provider:     provider.ProviderID(row.Provider),
		Group:        provider.GroupID(row.ProviderGroup),
		Operation:    provider.OperationID(row.Operation),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
	}
	for _, line := range lines {
		statement.Lines = append(statement.Lines, rowToLine(line))
		if statement.Currency == "" && line.CurrencyCode != "" {
			statement.Currency = line.CurrencyCode
		}
	}
	return statement
}

func rowToLine(row storage.FinancialLineItemV1Row) financialsrole.LineItem {
	extensions := make(map[string]string)
	if strings.TrimSpace(row.ExtensionsJSON) != "" {
		_ = json.Unmarshal([]byte(row.ExtensionsJSON), &extensions)
	}
	if row.CanonicalAccount != "" {
		extensions["canonical_account"] = row.CanonicalAccount
	}
	return financialsrole.LineItem{
		AccountID:   row.AccountID,
		AccountName: row.AccountName,
		Value:       row.RawAmount,
		Currency:    row.CurrencyCode,
		Unit:        row.Unit,
		Extensions:  extensions,
	}
}

func providerCompanyIdentifier(identifiers []companyidentity.Identifier) (string, string) {
	for _, identifier := range identifiers {
		if identifier.IdentifierType == companyidentity.IdentifierTypeDARTCorpCode {
			return identifier.IdentifierType, identifier.IdentifierValue
		}
	}
	if len(identifiers) == 0 {
		return "", ""
	}
	return identifiers[0].IdentifierType, identifiers[0].IdentifierValue
}

func firstInstrumentID(links []companyidentity.InstrumentLink) int64 {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer && link.InstrumentID != 0 {
			return link.InstrumentID
		}
	}
	if len(links) > 0 {
		return links[0].InstrumentID
	}
	return 0
}

func firstInstrumentSymbol(links []companyidentity.InstrumentLink) string {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer && strings.TrimSpace(link.Symbol) != "" {
			return link.Symbol
		}
	}
	if len(links) > 0 {
		return links[0].Symbol
	}
	return ""
}

func statementTypeForLine(line financialsrole.LineItem, fallback financialsrole.StatementType) financialsrole.StatementType {
	switch strings.ToUpper(strings.TrimSpace(line.Extensions["sj_div"])) {
	case "BS":
		return financialsrole.StatementTypeBalanceSheet
	case "IS", "CIS":
		return financialsrole.StatementTypeIncomeStatement
	case "CF":
		return financialsrole.StatementTypeCashFlow
	default:
		if fallback != "" && fallback != financialsrole.StatementTypeSummary {
			return fallback
		}
		return financialsrole.StatementTypeSummary
	}
}

func canonicalAccount(line financialsrole.LineItem) string {
	id := strings.ToLower(strings.TrimSpace(line.AccountID))
	name := strings.TrimSpace(line.AccountName)
	switch {
	case strings.Contains(id, "revenue") || name == "매출액" || strings.Contains(name, "수익(매출액)"):
		return "revenue"
	case strings.Contains(id, "operatingincome") || name == "영업이익":
		return "operating_income"
	case strings.Contains(id, "profitloss") || name == "당기순이익" || name == "분기순이익" || name == "반기순이익":
		return "net_income"
	case strings.Contains(id, "assets") && (name == "자산총계" || strings.Contains(strings.ToLower(id), "assets")):
		return "total_assets"
	case strings.Contains(id, "liabilities") && (name == "부채총계" || strings.Contains(strings.ToLower(id), "liabilities")):
		return "total_liabilities"
	case strings.Contains(id, "equity") && (name == "자본총계" || strings.Contains(strings.ToLower(id), "equity")):
		return "equity"
	case strings.Contains(id, "cashflowsfromusedinoperatingactivities") || strings.Contains(name, "영업활동"):
		return "operating_cashflow"
	default:
		return ""
	}
}

func parseAmountMinor(value string) *int64 {
	trimmed := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if trimmed == "" || trimmed == "-" {
		return nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseOrd(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
