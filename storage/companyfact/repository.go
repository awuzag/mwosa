package companyfact

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

const FactTypeDividend = "dividend"

type Repository struct {
	database *storage.Database
}

type FactInput struct {
	Provider                       provider.ProviderID
	Group                          provider.GroupID
	Operation                      provider.OperationID
	ProviderCompanyIdentifierType  string
	ProviderCompanyIdentifierValue string
	FactType                       string
	FiscalYear                     string
	ReportCode                     string
	RceptNo                        string
	FactDate                       string
	Key                            string
	ValueText                      string
	ValueNumber                    string
	CurrencyCode                   string
	Raw                            any
}

type Fact struct {
	CompanyID                      int64                `json:"company_id" csv:"company_id"`
	InstrumentID                   int64                `json:"instrument_id" csv:"instrument_id"`
	Provider                       provider.ProviderID  `json:"provider" csv:"provider"`
	Group                          provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation                      provider.OperationID `json:"operation" csv:"operation"`
	ProviderCompanyIdentifierType  string               `json:"provider_company_identifier_type" csv:"provider_company_identifier_type"`
	ProviderCompanyIdentifierValue string               `json:"provider_company_identifier_value" csv:"provider_company_identifier_value"`
	FactType                       string               `json:"fact_type" csv:"fact_type"`
	FiscalYear                     string               `json:"fiscal_year" csv:"fiscal_year"`
	ReportCode                     string               `json:"report_code" csv:"report_code"`
	RceptNo                        string               `json:"rcept_no" csv:"rcept_no"`
	FactDate                       string               `json:"fact_date" csv:"fact_date"`
	Key                            string               `json:"key" csv:"key"`
	ValueText                      string               `json:"value_text" csv:"value_text"`
	ValueNumber                    string               `json:"value_number" csv:"value_number"`
	CurrencyCode                   string               `json:"currency_code,omitempty" csv:"currency_code"`
	Raw                            map[string]any       `json:"raw,omitempty" csv:"-"`
}

type Query struct {
	FactType    string
	FiscalYear  string
	WindowYears int
	From        string
	To          string
	Limit       int
}

type UpsertResult struct {
	FactsWritten int `json:"facts_written" csv:"facts_written"`
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("company_fact_repository").New("company fact repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertFacts(ctx context.Context, company companyidentity.InspectResult, facts []FactInput) (UpsertResult, error) {
	errb := oops.In("company_fact_repository").With("company_id", company.Company.ID, "facts", len(facts))
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("company fact upsert requires canonical company")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, errb.Wrapf(err, "begin company fact sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	nowMS := time.Now().UTC().UnixMilli()
	result := UpsertResult{}
	for _, fact := range facts {
		if err := upsertFact(ctx, tx, company, fact, nowMS); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.FactsWritten++
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, errb.Wrapf(err, "commit company fact sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r Repository) ListFacts(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Fact, error) {
	errb := oops.In("company_fact_repository").With("company_id", company.Company.ID, "fact_type", query.FactType, "fiscal_year", query.FiscalYear)
	if company.Company.ID == 0 {
		return nil, errb.New("company fact query requires canonical company")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows := make([]storage.CompanyFactV1Row, 0)
	selectQuery := client.NewSelect().
		Model(&rows).
		Where("company_id = ?", company.Company.ID).
		Order("fiscal_year DESC", "report_code DESC", "fact_date DESC", "key ASC")
	if strings.TrimSpace(query.FactType) != "" {
		selectQuery = selectQuery.Where("fact_type = ?", strings.TrimSpace(query.FactType))
	}
	if strings.TrimSpace(query.FiscalYear) != "" {
		selectQuery = selectQuery.Where("fiscal_year = ?", strings.TrimSpace(query.FiscalYear))
	}
	if strings.TrimSpace(query.From) != "" {
		selectQuery = selectQuery.Where("fact_date >= ?", strings.TrimSpace(query.From))
	}
	if strings.TrimSpace(query.To) != "" {
		selectQuery = selectQuery.Where("fact_date <= ?", strings.TrimSpace(query.To))
	}
	if query.Limit > 0 && query.WindowYears <= 0 {
		selectQuery = selectQuery.Limit(query.Limit)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "select company facts")
	}
	rows = filterWindowYears(rows, query.WindowYears)
	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
	}
	out := make([]Fact, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToFact(row))
	}
	return out, nil
}

func upsertFact(ctx context.Context, tx bun.Tx, company companyidentity.InspectResult, fact FactInput, nowMS int64) error {
	identifierType, identifierValue := fact.ProviderCompanyIdentifierType, fact.ProviderCompanyIdentifierValue
	if strings.TrimSpace(identifierType) == "" || strings.TrimSpace(identifierValue) == "" {
		identifierType, identifierValue = providerCompanyIdentifier(company.Identifiers)
	}
	rawJSON, err := json.Marshal(fact.Raw)
	if err != nil {
		return oops.In("company_fact_repository").With("fact_type", fact.FactType, "key", fact.Key).Wrapf(err, "encode company fact raw payload")
	}
	row := storage.CompanyFactV1Row{
		CompanyID:                      company.Company.ID,
		InstrumentID:                   issuerInstrumentID(company.Instruments),
		Provider:                       string(fact.Provider),
		ProviderGroup:                  string(fact.Group),
		Operation:                      string(fact.Operation),
		ProviderCompanyIdentifierType:  strings.TrimSpace(identifierType),
		ProviderCompanyIdentifierValue: strings.TrimSpace(identifierValue),
		FactType:                       strings.TrimSpace(fact.FactType),
		FiscalYear:                     strings.TrimSpace(fact.FiscalYear),
		ReportCode:                     strings.TrimSpace(fact.ReportCode),
		RceptNo:                        strings.TrimSpace(fact.RceptNo),
		FactDate:                       strings.TrimSpace(fact.FactDate),
		Key:                            strings.TrimSpace(fact.Key),
		ValueText:                      strings.TrimSpace(fact.ValueText),
		ValueNumber:                    strings.TrimSpace(fact.ValueNumber),
		CurrencyCode:                   strings.TrimSpace(fact.CurrencyCode),
		RawJSON:                        string(rawJSON),
		CreatedAtMS:                    nowMS,
		UpdatedAtMS:                    nowMS,
	}
	if row.RawJSON == "null" {
		row.RawJSON = "{}"
	}
	if row.Provider == "" || row.ProviderGroup == "" || row.Operation == "" || row.FactType == "" || row.Key == "" {
		return oops.In("company_fact_repository").With("company_id", row.CompanyID).New("company fact missing natural key")
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (company_id, instrument_id, provider, provider_group, operation, fact_type, fiscal_year, report_code, rcept_no, key) DO UPDATE").
		Set("provider_company_identifier_type = EXCLUDED.provider_company_identifier_type").
		Set("provider_company_identifier_value = EXCLUDED.provider_company_identifier_value").
		Set("fact_date = EXCLUDED.fact_date").
		Set("value_text = EXCLUDED.value_text").
		Set("value_number = EXCLUDED.value_number").
		Set("currency_code = EXCLUDED.currency_code").
		Set("raw_json = EXCLUDED.raw_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("company_fact_repository").With("company_id", row.CompanyID, "fact_type", row.FactType, "key", row.Key).Wrapf(err, "upsert company fact")
	}
	return nil
}

func rowToFact(row storage.CompanyFactV1Row) Fact {
	raw := map[string]any{}
	if strings.TrimSpace(row.RawJSON) != "" {
		_ = json.Unmarshal([]byte(row.RawJSON), &raw)
	}
	return Fact{
		CompanyID:                      row.CompanyID,
		InstrumentID:                   row.InstrumentID,
		Provider:                       provider.ProviderID(row.Provider),
		Group:                          provider.GroupID(row.ProviderGroup),
		Operation:                      provider.OperationID(row.Operation),
		ProviderCompanyIdentifierType:  row.ProviderCompanyIdentifierType,
		ProviderCompanyIdentifierValue: row.ProviderCompanyIdentifierValue,
		FactType:                       row.FactType,
		FiscalYear:                     row.FiscalYear,
		ReportCode:                     row.ReportCode,
		RceptNo:                        row.RceptNo,
		FactDate:                       row.FactDate,
		Key:                            row.Key,
		ValueText:                      row.ValueText,
		ValueNumber:                    row.ValueNumber,
		CurrencyCode:                   row.CurrencyCode,
		Raw:                            raw,
	}
}

func filterWindowYears(rows []storage.CompanyFactV1Row, windowYears int) []storage.CompanyFactV1Row {
	if windowYears <= 0 || len(rows) == 0 {
		return rows
	}
	years := make(map[string]struct{})
	for _, row := range rows {
		year := strings.TrimSpace(row.FiscalYear)
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
	filtered := rows[:0]
	for _, row := range rows {
		if _, ok := allowed[row.FiscalYear]; ok {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func providerCompanyIdentifier(identifiers []companyidentity.Identifier) (string, string) {
	for _, identifier := range identifiers {
		if identifier.IdentifierType == companyidentity.IdentifierTypeDARTCorpCode {
			return identifier.IdentifierType, identifier.IdentifierValue
		}
	}
	for _, identifier := range identifiers {
		if identifier.Primary {
			return identifier.IdentifierType, identifier.IdentifierValue
		}
	}
	if len(identifiers) == 0 {
		return "", ""
	}
	return identifiers[0].IdentifierType, identifiers[0].IdentifierValue
}

func issuerInstrumentID(links []companyidentity.InstrumentLink) int64 {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer {
			return link.InstrumentID
		}
	}
	if len(links) == 0 {
		return 0
	}
	return links[0].InstrumentID
}
