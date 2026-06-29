package companyidentity

import (
	"context"
	"database/sql"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

const (
	IdentifierTypeDARTCorpCode = "dart_corp_code"
	IdentifierTypeKRXStockCode = "krx_stock_code"
	RelationTypeIssuer         = "issuer"
)

type Repository struct {
	database *storage.SQLDatabase
}

type CompanyInput struct {
	Name          string
	LegalName     string
	EnglishName   string
	CountryCode   string
	Identifiers   []IdentifierInput
	InstrumentRef InstrumentRef
}

type IdentifierInput struct {
	Provider        provider.ProviderID
	Group           provider.GroupID
	Operation       provider.OperationID
	IdentifierType  string
	IdentifierValue string
	Primary         bool
	Confidence      float64
	SourceUpdatedAt string
}

type InstrumentRef struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	Name         string
	RelationType string
}

type UpsertResult struct {
	CompaniesWritten   int `json:"companies_written" csv:"companies_written"`
	IdentifiersWritten int `json:"identifiers_written" csv:"identifiers_written"`
	InstrumentsLinked  int `json:"instruments_linked" csv:"instruments_linked"`
}

type Company struct {
	ID          int64  `json:"id" csv:"id"`
	Name        string `json:"name" csv:"name"`
	LegalName   string `json:"legal_name,omitempty" csv:"legal_name"`
	EnglishName string `json:"english_name,omitempty" csv:"english_name"`
	CountryCode string `json:"country_code,omitempty" csv:"country_code"`
}

type Identifier struct {
	CompanyID       int64                `json:"company_id" csv:"company_id"`
	Provider        provider.ProviderID  `json:"provider" csv:"provider"`
	Group           provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation       provider.OperationID `json:"operation" csv:"operation"`
	IdentifierType  string               `json:"identifier_type" csv:"identifier_type"`
	IdentifierValue string               `json:"identifier_value" csv:"identifier_value"`
	Primary         bool                 `json:"primary" csv:"primary"`
	Confidence      float64              `json:"confidence" csv:"confidence"`
	SourceUpdatedAt string               `json:"source_updated_at,omitempty" csv:"source_updated_at"`
}

type InstrumentLink struct {
	CompanyID    int64                 `json:"company_id" csv:"company_id"`
	InstrumentID int64                 `json:"instrument_id" csv:"instrument_id"`
	Market       provider.Market       `json:"market" csv:"market"`
	SecurityType provider.SecurityType `json:"security_type" csv:"security_type"`
	Symbol       string                `json:"symbol" csv:"symbol"`
	Name         string                `json:"name,omitempty" csv:"name"`
	RelationType string                `json:"relation_type" csv:"relation_type"`
}

type InspectResult struct {
	Company     Company          `json:"company"`
	Identifiers []Identifier     `json:"identifiers"`
	Instruments []InstrumentLink `json:"instruments"`
}

func NewRepository(database *storage.SQLDatabase) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("company_identity_repository").New("company identity repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertCompanies(ctx context.Context, companies []CompanyInput) (UpsertResult, error) {
	errb := oops.In("company_identity_repository").With("companies", len(companies))
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, errb.Wrapf(err, "begin company identity sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var result UpsertResult
	nowMS := time.Now().UTC().UnixMilli()
	for _, input := range companies {
		companyID, err := resolveCompanyID(ctx, tx, input.Identifiers)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		company, err := upsertCompany(ctx, tx, companyID, input, nowMS)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.CompaniesWritten++
		for _, identifier := range input.Identifiers {
			if err := upsertIdentifier(ctx, tx, company.ID, identifier, nowMS); err != nil {
				return UpsertResult{}, errb.Wrap(err)
			}
			result.IdentifiersWritten++
		}
		if strings.TrimSpace(input.InstrumentRef.Symbol) != "" {
			linked, err := upsertInstrumentLink(ctx, tx, company.ID, input.InstrumentRef, nowMS)
			if err != nil {
				return UpsertResult{}, errb.Wrap(err)
			}
			if linked {
				result.InstrumentsLinked++
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, errb.Wrapf(err, "commit company identity sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r Repository) Inspect(ctx context.Context, query string) (InspectResult, error) {
	trimmed := strings.TrimSpace(query)
	errb := oops.In("company_identity_repository").With("query", trimmed)
	if trimmed == "" {
		return InspectResult{}, errb.New("inspect company requires query")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	company, err := findCompany(ctx, client, trimmed)
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	identifiers, err := listIdentifiers(ctx, client, company.ID)
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	links, err := listInstrumentLinks(ctx, client, company.ID)
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	return InspectResult{Company: company, Identifiers: identifiers, Instruments: links}, nil
}

func resolveCompanyID(ctx context.Context, tx bun.Tx, identifiers []IdentifierInput) (int64, error) {
	for _, identifier := range identifiers {
		identifierType := strings.TrimSpace(identifier.IdentifierType)
		identifierValue := strings.TrimSpace(identifier.IdentifierValue)
		if identifierType == "" || identifierValue == "" {
			continue
		}
		var row storage.CompanyIdentifierV1Row
		err := tx.NewSelect().
			Model(&row).
			Where("identifier_type = ?", identifierType).
			Where("identifier_value = ?", identifierValue).
			Order("primary_flag DESC", "id ASC").
			Limit(1).
			Scan(ctx)
		if err == nil {
			return row.CompanyID, nil
		}
		if err != sql.ErrNoRows {
			return 0, oops.In("company_identity_repository").With("identifier_type", identifierType, "identifier_value", identifierValue).Wrapf(err, "select company identifier")
		}
	}
	return 0, nil
}

func upsertCompany(ctx context.Context, tx bun.Tx, companyID int64, input CompanyInput, nowMS int64) (storage.CompanyV1Row, error) {
	name := firstNonEmpty(input.Name, input.LegalName, input.EnglishName)
	if name == "" {
		return storage.CompanyV1Row{}, oops.In("company_identity_repository").New("company identity row missing name")
	}
	row := storage.CompanyV1Row{
		ID:          companyID,
		Name:        name,
		LegalName:   strings.TrimSpace(input.LegalName),
		EnglishName: strings.TrimSpace(input.EnglishName),
		CountryCode: firstNonEmpty(input.CountryCode, "KR"),
		CreatedAtMS: nowMS,
		UpdatedAtMS: nowMS,
	}
	if companyID > 0 {
		if _, err := tx.NewUpdate().
			Model(&row).
			Column("name", "legal_name", "english_name", "country_code", "updated_at_ms").
			WherePK().
			Exec(ctx); err != nil {
			return storage.CompanyV1Row{}, oops.In("company_identity_repository").With("company_id", companyID).Wrapf(err, "update company")
		}
		return row, nil
	}
	if _, err := tx.NewInsert().Model(&row).Returning("id").Exec(ctx); err != nil {
		return storage.CompanyV1Row{}, oops.In("company_identity_repository").Wrapf(err, "insert company")
	}
	return row, nil
}

func upsertIdentifier(ctx context.Context, tx bun.Tx, companyID int64, input IdentifierInput, nowMS int64) error {
	identifierType := strings.TrimSpace(input.IdentifierType)
	identifierValue := strings.TrimSpace(input.IdentifierValue)
	if identifierType == "" || identifierValue == "" {
		return oops.In("company_identity_repository").With("company_id", companyID).New("company identifier missing type or value")
	}
	confidence := input.Confidence
	if confidence == 0 {
		confidence = 1
	}
	row := storage.CompanyIdentifierV1Row{
		CompanyID:       companyID,
		Provider:        string(input.Provider),
		ProviderGroup:   string(input.Group),
		Operation:       string(input.Operation),
		IdentifierType:  identifierType,
		IdentifierValue: identifierValue,
		PrimaryFlag:     input.Primary,
		Confidence:      confidence,
		SourceUpdatedAt: strings.TrimSpace(input.SourceUpdatedAt),
		CreatedAtMS:     nowMS,
		UpdatedAtMS:     nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation, identifier_type, identifier_value, valid_from) DO UPDATE").
		Set("company_id = EXCLUDED.company_id").
		Set("valid_to = EXCLUDED.valid_to").
		Set("primary_flag = EXCLUDED.primary_flag").
		Set("confidence = EXCLUDED.confidence").
		Set("source_updated_at = EXCLUDED.source_updated_at").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("company_identity_repository").With("company_id", companyID, "identifier_type", identifierType, "identifier_value", identifierValue).Wrapf(err, "upsert company identifier")
	}
	return nil
}

func upsertInstrumentLink(ctx context.Context, tx bun.Tx, companyID int64, input InstrumentRef, nowMS int64) (bool, error) {
	market, err := ensureMarket(ctx, tx, withDefaultMarket(input.Market), nowMS)
	if err != nil {
		return false, err
	}
	instrument, err := ensureInstrument(ctx, tx, market.ID, input, nowMS)
	if err != nil {
		return false, err
	}
	relationType := firstNonEmpty(input.RelationType, RelationTypeIssuer)
	row := storage.InstrumentCompanyLinkV1Row{
		InstrumentID: instrument.ID,
		CompanyID:    companyID,
		RelationType: relationType,
		CreatedAtMS:  nowMS,
		UpdatedAtMS:  nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (instrument_id, company_id, relation_type, valid_from) DO UPDATE").
		Set("valid_to = EXCLUDED.valid_to").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return false, oops.In("company_identity_repository").With("company_id", companyID, "instrument_id", instrument.ID, "relation_type", relationType).Wrapf(err, "upsert instrument company link")
	}
	return true, nil
}

func ensureMarket(ctx context.Context, tx bun.Tx, market provider.Market, nowMS int64) (storage.MarketV2Row, error) {
	code := string(withDefaultMarket(market))
	row := storage.MarketV2Row{
		Code:               code,
		Timezone:           marketTimezone(provider.Market(code)),
		RegularOpenMinute:  marketRegularOpenMinute(provider.Market(code)),
		RegularCloseMinute: marketRegularCloseMinute(provider.Market(code)),
		CreatedAtMS:        nowMS,
		UpdatedAtMS:        nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (code) DO UPDATE").
		Set("timezone = EXCLUDED.timezone").
		Set("regular_open_minute = EXCLUDED.regular_open_minute").
		Set("regular_close_minute = EXCLUDED.regular_close_minute").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("company_identity_repository").With("market", code).Wrapf(err, "upsert market")
	}
	var stored storage.MarketV2Row
	if err := tx.NewSelect().Model(&stored).Where("code = ?", code).Limit(1).Scan(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("company_identity_repository").With("market", code).Wrapf(err, "select market")
	}
	return stored, nil
}

func ensureInstrument(ctx context.Context, tx bun.Tx, marketID int64, input InstrumentRef, nowMS int64) (storage.InstrumentV2Row, error) {
	securityType := input.SecurityType
	if securityType == "" {
		securityType = provider.SecurityTypeStock
	}
	symbol := strings.TrimSpace(input.Symbol)
	row := storage.InstrumentV2Row{
		MarketID:     marketID,
		SecurityType: string(securityType),
		Symbol:       symbol,
		Name:         strings.TrimSpace(input.Name),
		CurrencyCode: "KRW",
		PriceScale:   4,
		CreatedAtMS:  nowMS,
		UpdatedAtMS:  nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (market_id, security_type, symbol) DO UPDATE").
		Set("name = CASE WHEN EXCLUDED.name != '' THEN EXCLUDED.name ELSE name END").
		Set("currency_code = EXCLUDED.currency_code").
		Set("price_scale = EXCLUDED.price_scale").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("company_identity_repository").With("market_id", marketID, "security_type", securityType, "symbol", symbol).Wrapf(err, "upsert linked instrument")
	}
	var stored storage.InstrumentV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("market_id = ?", marketID).
		Where("security_type = ?", string(securityType)).
		Where("symbol = ?", symbol).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("company_identity_repository").With("market_id", marketID, "security_type", securityType, "symbol", symbol).Wrapf(err, "select linked instrument")
	}
	return stored, nil
}

type queryDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func findCompany(ctx context.Context, db queryDB, query string) (Company, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT c.id, c.name, c.legal_name, c.english_name, c.country_code
FROM company_v1 AS c
LEFT JOIN company_identifier_v1 AS ci ON ci.company_id = c.id
WHERE ci.identifier_value = ?
   OR c.name LIKE ?
   OR c.legal_name LIKE ?
   OR c.english_name LIKE ?
ORDER BY CASE WHEN ci.identifier_value = ? THEN 0 WHEN c.name = ? THEN 1 ELSE 2 END, c.name ASC
LIMIT 2`, query, "%"+query+"%", "%"+query+"%", "%"+query+"%", query, query)
	if err != nil {
		return Company{}, oops.In("company_identity_repository").With("query", query).Wrapf(err, "query company")
	}
	defer rows.Close()
	matches := make([]Company, 0, 2)
	for rows.Next() {
		var company Company
		if err := rows.Scan(&company.ID, &company.Name, &company.LegalName, &company.EnglishName, &company.CountryCode); err != nil {
			return Company{}, oops.In("company_identity_repository").Wrapf(err, "scan company")
		}
		matches = append(matches, company)
	}
	if err := rows.Err(); err != nil {
		return Company{}, oops.In("company_identity_repository").Wrapf(err, "iterate companies")
	}
	if len(matches) == 0 {
		return Company{}, oops.In("company_identity_repository").With("query", query).New("company not found")
	}
	if len(matches) > 1 {
		return Company{}, oops.In("company_identity_repository").With("query", query, "matches", len(matches)).New("company query matched multiple companies")
	}
	return matches[0], nil
}

func listIdentifiers(ctx context.Context, db queryDB, companyID int64) ([]Identifier, error) {
	rows, err := db.QueryContext(ctx, `SELECT company_id, provider, provider_group, operation, identifier_type, identifier_value, primary_flag, confidence, source_updated_at
FROM company_identifier_v1
WHERE company_id = ?
ORDER BY primary_flag DESC, identifier_type ASC, provider ASC`, companyID)
	if err != nil {
		return nil, oops.In("company_identity_repository").With("company_id", companyID).Wrapf(err, "query company identifiers")
	}
	defer rows.Close()
	identifiers := make([]Identifier, 0)
	for rows.Next() {
		var item Identifier
		if err := rows.Scan(&item.CompanyID, &item.Provider, &item.Group, &item.Operation, &item.IdentifierType, &item.IdentifierValue, &item.Primary, &item.Confidence, &item.SourceUpdatedAt); err != nil {
			return nil, oops.In("company_identity_repository").Wrapf(err, "scan company identifier")
		}
		identifiers = append(identifiers, item)
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("company_identity_repository").Wrapf(err, "iterate company identifiers")
	}
	return identifiers, nil
}

func listInstrumentLinks(ctx context.Context, db queryDB, companyID int64) ([]InstrumentLink, error) {
	rows, err := db.QueryContext(ctx, `SELECT l.company_id, l.instrument_id, m.code, i.security_type, i.symbol, i.name, l.relation_type
FROM instrument_company_link_v1 AS l
JOIN instrument_v2 AS i ON i.id = l.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE l.company_id = ?
ORDER BY l.relation_type ASC, m.code ASC, i.symbol ASC`, companyID)
	if err != nil {
		return nil, oops.In("company_identity_repository").With("company_id", companyID).Wrapf(err, "query instrument company links")
	}
	defer rows.Close()
	links := make([]InstrumentLink, 0)
	for rows.Next() {
		var item InstrumentLink
		if err := rows.Scan(&item.CompanyID, &item.InstrumentID, &item.Market, &item.SecurityType, &item.Symbol, &item.Name, &item.RelationType); err != nil {
			return nil, oops.In("company_identity_repository").Wrapf(err, "scan instrument company link")
		}
		links = append(links, item)
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("company_identity_repository").Wrapf(err, "iterate instrument company links")
	}
	return links, nil
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func marketTimezone(market provider.Market) string {
	if market == provider.MarketKRX {
		return "Asia/Seoul"
	}
	return "UTC"
}

func marketRegularOpenMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 9 * 60
	}
	return 0
}

func marketRegularCloseMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 15*60 + 30
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
