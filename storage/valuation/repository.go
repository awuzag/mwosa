package valuation

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/awuzag/mwosa/packages/financialmetrics"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type Repository struct {
	database *storage.SQLDatabase
}

type CalculateOptions struct {
	AsOf string
}

type Query struct {
	AsOf string
}

type UpsertResult struct {
	SnapshotsWritten int `json:"snapshots_written" csv:"snapshots_written"`
}

type Snapshot struct {
	CompanyID           int64             `json:"company_id" csv:"company_id"`
	InstrumentID        int64             `json:"instrument_id" csv:"instrument_id"`
	AsOfDate            string            `json:"as_of_date" csv:"as_of_date"`
	SourcePriceDate     string            `json:"source_price_date" csv:"source_price_date"`
	MarketCapMinor      *int64            `json:"market_cap_minor,omitempty" csv:"market_cap_minor"`
	ClosePriceMinor     *int64            `json:"close_price_minor,omitempty" csv:"close_price_minor"`
	SharesOutstanding   *int64            `json:"shares_outstanding,omitempty" csv:"shares_outstanding"`
	PerBP               *int64            `json:"per_bp,omitempty" csv:"per_bp"`
	PbrBP               *int64            `json:"pbr_bp,omitempty" csv:"pbr_bp"`
	PsrBP               *int64            `json:"psr_bp,omitempty" csv:"psr_bp"`
	EpsMinor            *int64            `json:"eps_minor,omitempty" csv:"eps_minor"`
	BpsMinor            *int64            `json:"bps_minor,omitempty" csv:"bps_minor"`
	DividendYieldBP     *int64            `json:"dividend_yield_bp,omitempty" csv:"dividend_yield_bp"`
	MetricSourceVersion string            `json:"metric_source_version" csv:"metric_source_version"`
	Provenance          map[string]any    `json:"provenance,omitempty" csv:"-"`
	Uncomputable        map[string]string `json:"uncomputable,omitempty" csv:"-"`
}

type pricePoint struct {
	InstrumentID    int64
	TradingDate     int
	ClosePriceMinor int64
	MarketCapMinor  int64
	PriceScale      int
	Provider        string
	ProviderGroup   string
	Operation       string
}

type accountValue struct {
	StatementID      int64
	CanonicalAccount string
	AmountMinor      int64
	FiscalYear       string
	FiscalPeriod     string
	ReportCode       string
	FsDiv            string
	RceptNo          string
	Provider         string
	ProviderGroup    string
	Operation        string
}

type dividendValue struct {
	FactID      int64
	AmountMinor int64
	FiscalYear  string
	ReportCode  string
	RceptNo     string
	FactDate    string
	Key         string
	Provider    string
	Group       string
	Operation   string
}

func NewRepository(database *storage.SQLDatabase) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("valuation_repository").New("valuation repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (Snapshot, error) {
	errb := oops.In("valuation_repository").With("company_id", company.Company.ID, "as_of", options.AsOf)
	if company.Company.ID == 0 {
		return Snapshot{}, errb.New("valuation calculation requires canonical company")
	}
	instrumentID := issuerInstrumentID(company.Instruments)
	if instrumentID == 0 {
		return Snapshot{}, errb.New("valuation calculation requires issuer instrument link")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	price, err := selectPrice(ctx, client, instrumentID, options.AsOf)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	accounts, err := selectLatestAccounts(ctx, client, company.Company.ID)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	if len(accounts) == 0 {
		return Snapshot{}, errb.New("valuation calculation requires stored financial line items")
	}
	dividend, hasDividend, err := selectLatestDividend(ctx, client, company.Company.ID)
	if err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	snapshot := buildSnapshot(company.Company.ID, price, accounts, dividend, hasDividend)
	if err := upsertSnapshot(ctx, client, snapshot); err != nil {
		return Snapshot{}, errb.Wrap(err)
	}
	return snapshot, nil
}

func (r Repository) ListSnapshots(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Snapshot, error) {
	errb := oops.In("valuation_repository").With("company_id", company.Company.ID, "as_of", query.AsOf)
	if company.Company.ID == 0 {
		return nil, errb.New("valuation query requires canonical company")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows := make([]storage.ValuationSnapshotV1Row, 0)
	selectQuery := client.NewSelect().
		Model(&rows).
		Where("company_id = ?", company.Company.ID).
		Order("as_of_date DESC")
	if strings.TrimSpace(query.AsOf) != "" && strings.TrimSpace(query.AsOf) != "latest" {
		selectQuery = selectQuery.Where("as_of_date = ?", strings.TrimSpace(query.AsOf))
	} else {
		selectQuery = selectQuery.Limit(1)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "select valuation snapshots")
	}
	out := make([]Snapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToSnapshot(row))
	}
	return out, nil
}

func selectPrice(ctx context.Context, db *bun.DB, instrumentID int64, asOf string) (pricePoint, error) {
	var row struct {
		InstrumentID    int64         `bun:"instrument_id"`
		TradingDate     int           `bun:"trading_date"`
		ClosePriceMinor sql.NullInt64 `bun:"close_price"`
		MarketCapMinor  sql.NullInt64 `bun:"market_cap_minor"`
		PriceScale      int           `bun:"price_scale"`
		Provider        string        `bun:"provider"`
		ProviderGroup   string        `bun:"provider_group"`
		Operation       string        `bun:"operation"`
	}
	query := db.NewSelect().
		TableExpr("daily_bar_v2 AS b").
		ColumnExpr("b.instrument_id").
		ColumnExpr("b.trading_date").
		ColumnExpr("b.close_price").
		ColumnExpr("b.market_cap_minor").
		ColumnExpr("i.price_scale").
		ColumnExpr("s.provider").
		ColumnExpr("s.provider_group").
		ColumnExpr("s.operation").
		Join("JOIN instrument_v2 AS i ON i.id = b.instrument_id").
		Join("JOIN provider_source_v2 AS s ON s.id = b.source_id").
		Where("b.instrument_id = ?", instrumentID).
		Where("b.close_price IS NOT NULL").
		Where("b.market_cap_minor IS NOT NULL").
		Order("b.trading_date DESC").
		Limit(1)
	if strings.TrimSpace(asOf) != "" && strings.TrimSpace(asOf) != "latest" {
		date, err := parseDateInt(asOf)
		if err != nil {
			return pricePoint{}, err
		}
		query = query.Where("b.trading_date <= ?", date)
	}
	if err := query.Scan(ctx, &row); err != nil {
		return pricePoint{}, oops.In("valuation_repository").With("instrument_id", instrumentID, "as_of", asOf).Wrapf(err, "select valuation price point")
	}
	if !row.ClosePriceMinor.Valid || row.ClosePriceMinor.Int64 <= 0 {
		return pricePoint{}, oops.In("valuation_repository").With("instrument_id", instrumentID, "as_of", asOf).New("valuation price point missing positive close price")
	}
	if !row.MarketCapMinor.Valid || row.MarketCapMinor.Int64 <= 0 {
		return pricePoint{}, oops.In("valuation_repository").With("instrument_id", instrumentID, "as_of", asOf).New("valuation price point missing positive market cap")
	}
	return pricePoint{
		InstrumentID:    row.InstrumentID,
		TradingDate:     row.TradingDate,
		ClosePriceMinor: row.ClosePriceMinor.Int64,
		MarketCapMinor:  row.MarketCapMinor.Int64,
		PriceScale:      row.PriceScale,
		Provider:        row.Provider,
		ProviderGroup:   row.ProviderGroup,
		Operation:       row.Operation,
	}, nil
}

func selectLatestAccounts(ctx context.Context, db *bun.DB, companyID int64) (map[string]accountValue, error) {
	var rows []struct {
		StatementID      int64         `bun:"statement_id"`
		CanonicalAccount string        `bun:"canonical_account"`
		AmountMinor      sql.NullInt64 `bun:"amount_minor"`
		FiscalYear       string        `bun:"fiscal_year"`
		FiscalPeriod     string        `bun:"fiscal_period"`
		ReportCode       string        `bun:"report_code"`
		FsDiv            string        `bun:"fs_div"`
		RceptNo          string        `bun:"rcept_no"`
		Provider         string        `bun:"provider"`
		ProviderGroup    string        `bun:"provider_group"`
		Operation        string        `bun:"operation"`
	}
	if err := db.NewSelect().
		TableExpr("financial_line_item_v1 AS l").
		ColumnExpr("s.id AS statement_id").
		ColumnExpr("l.canonical_account").
		ColumnExpr("l.amount_minor").
		ColumnExpr("s.fiscal_year").
		ColumnExpr("s.fiscal_period").
		ColumnExpr("s.report_code").
		ColumnExpr("s.fs_div").
		ColumnExpr("s.rcept_no").
		ColumnExpr("s.provider").
		ColumnExpr("s.provider_group").
		ColumnExpr("s.operation").
		Join("JOIN financial_statement_v1 AS s ON s.id = l.statement_id").
		Where("s.company_id = ?", companyID).
		Where("l.canonical_account IN (?)", bun.In([]string{"revenue", "net_income", "equity"})).
		Where("l.amount_minor IS NOT NULL").
		Order("s.fiscal_year DESC", "s.report_code DESC").
		Scan(ctx, &rows); err != nil {
		return nil, oops.In("valuation_repository").With("company_id", companyID).Wrapf(err, "select valuation financial accounts")
	}
	out := make(map[string]accountValue)
	for _, row := range rows {
		if _, exists := out[row.CanonicalAccount]; exists || !row.AmountMinor.Valid {
			continue
		}
		out[row.CanonicalAccount] = accountValue{
			StatementID:      row.StatementID,
			CanonicalAccount: row.CanonicalAccount,
			AmountMinor:      row.AmountMinor.Int64,
			FiscalYear:       row.FiscalYear,
			FiscalPeriod:     row.FiscalPeriod,
			ReportCode:       row.ReportCode,
			FsDiv:            row.FsDiv,
			RceptNo:          row.RceptNo,
			Provider:         row.Provider,
			ProviderGroup:    row.ProviderGroup,
			Operation:        row.Operation,
		}
	}
	return out, nil
}

func selectLatestDividend(ctx context.Context, db *bun.DB, companyID int64) (dividendValue, bool, error) {
	var rows []struct {
		ID            int64  `bun:"id"`
		ValueNumber   string `bun:"value_number"`
		FiscalYear    string `bun:"fiscal_year"`
		ReportCode    string `bun:"report_code"`
		RceptNo       string `bun:"rcept_no"`
		FactDate      string `bun:"fact_date"`
		Key           string `bun:"key"`
		Provider      string `bun:"provider"`
		ProviderGroup string `bun:"provider_group"`
		Operation     string `bun:"operation"`
	}
	if err := db.NewSelect().
		TableExpr("company_fact_v1 AS f").
		ColumnExpr("f.id").
		ColumnExpr("f.value_number").
		ColumnExpr("f.fiscal_year").
		ColumnExpr("f.report_code").
		ColumnExpr("f.rcept_no").
		ColumnExpr("f.fact_date").
		ColumnExpr("f.key").
		ColumnExpr("f.provider").
		ColumnExpr("f.provider_group").
		ColumnExpr("f.operation").
		Where("f.company_id = ?", companyID).
		Where("f.fact_type = ?", "dividend").
		Where("f.value_number != ''").
		Order("f.fiscal_year DESC", "f.report_code DESC", "f.fact_date DESC", "f.id DESC").
		Scan(ctx, &rows); err != nil {
		return dividendValue{}, false, oops.In("valuation_repository").With("company_id", companyID).Wrapf(err, "select valuation dividend facts")
	}
	for _, row := range rows {
		if !isCashDividendTotalKey(row.Key) {
			continue
		}
		amount, err := strconv.ParseInt(strings.TrimSpace(row.ValueNumber), 10, 64)
		if err != nil || amount <= 0 {
			continue
		}
		return dividendValue{
			FactID:      row.ID,
			AmountMinor: amount,
			FiscalYear:  row.FiscalYear,
			ReportCode:  row.ReportCode,
			RceptNo:     row.RceptNo,
			FactDate:    row.FactDate,
			Key:         row.Key,
			Provider:    row.Provider,
			Group:       row.ProviderGroup,
			Operation:   row.Operation,
		}, true, nil
	}
	return dividendValue{}, false, nil
}

func buildSnapshot(companyID int64, price pricePoint, accounts map[string]accountValue, dividend dividendValue, hasDividend bool) Snapshot {
	marketCap := price.MarketCapMinor
	closePrice := price.ClosePriceMinor
	shares := sharesOutstanding(marketCap, closePrice, price.PriceScale)
	uncomputable := make(map[string]string)
	revenue, hasRevenue := accounts["revenue"]
	netIncome, hasNetIncome := accounts["net_income"]
	equity, hasEquity := accounts["equity"]
	snapshot := Snapshot{
		CompanyID:           companyID,
		InstrumentID:        price.InstrumentID,
		AsOfDate:            formatDateInt(price.TradingDate),
		SourcePriceDate:     formatDateInt(price.TradingDate),
		MarketCapMinor:      &marketCap,
		ClosePriceMinor:     &closePrice,
		SharesOutstanding:   shares,
		MetricSourceVersion: financialmetrics.FormulaVersion,
		Provenance: map[string]any{
			"source_price_date": price.TradingDate,
			"price_provider":    price.Provider,
			"price_group":       price.ProviderGroup,
			"price_operation":   price.Operation,
		},
		Uncomputable: uncomputable,
	}
	if hasNetIncome {
		snapshot.PerBP = ratioBP(marketCap, netIncome.AmountMinor)
		snapshot.EpsMinor = perShareMinor(netIncome.AmountMinor, shares)
		addAccountProvenance(snapshot.Provenance, "net_income", netIncome)
	} else {
		uncomputable["per"] = "net_income account value is missing"
		uncomputable["eps"] = "net_income account value is missing"
	}
	if hasEquity {
		snapshot.PbrBP = ratioBP(marketCap, equity.AmountMinor)
		snapshot.BpsMinor = perShareMinor(equity.AmountMinor, shares)
		addAccountProvenance(snapshot.Provenance, "equity", equity)
	} else {
		uncomputable["pbr"] = "equity account value is missing"
		uncomputable["bps"] = "equity account value is missing"
	}
	if hasRevenue {
		snapshot.PsrBP = ratioBP(marketCap, revenue.AmountMinor)
		addAccountProvenance(snapshot.Provenance, "revenue", revenue)
	} else {
		uncomputable["psr"] = "revenue account value is missing"
	}
	if hasDividend {
		snapshot.DividendYieldBP = ratioBP(dividend.AmountMinor, marketCap)
		addDividendProvenance(snapshot.Provenance, dividend)
		if snapshot.DividendYieldBP == nil {
			uncomputable["dividend_yield"] = "cash dividend total is not positive"
		}
	} else {
		uncomputable["dividend_yield"] = "cash dividend total fact is missing"
	}
	if shares == nil {
		uncomputable["shares_outstanding"] = "market cap and close price could not derive shares outstanding"
		if hasNetIncome {
			uncomputable["eps"] = "shares_outstanding is missing"
		}
		if hasEquity {
			uncomputable["bps"] = "shares_outstanding is missing"
		}
	}
	return snapshot
}

func upsertSnapshot(ctx context.Context, db *bun.DB, snapshot Snapshot) error {
	provenance, err := json.Marshal(snapshot.Provenance)
	if err != nil {
		return oops.In("valuation_repository").With("company_id", snapshot.CompanyID).Wrapf(err, "encode valuation provenance")
	}
	uncomputable, err := json.Marshal(snapshot.Uncomputable)
	if err != nil {
		return oops.In("valuation_repository").With("company_id", snapshot.CompanyID).Wrapf(err, "encode valuation uncomputable reasons")
	}
	nowMS := time.Now().UTC().UnixMilli()
	row := storage.ValuationSnapshotV1Row{
		CompanyID:           snapshot.CompanyID,
		InstrumentID:        snapshot.InstrumentID,
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
		ProvenanceJSON:      string(provenance),
		UncomputableJSON:    string(uncomputable),
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
	}
	if _, err := db.NewInsert().
		Model(&row).
		On("CONFLICT (company_id, instrument_id, as_of_date, metric_source_version) DO UPDATE").
		Set("source_price_date = EXCLUDED.source_price_date").
		Set("market_cap_minor = EXCLUDED.market_cap_minor").
		Set("close_price_minor = EXCLUDED.close_price_minor").
		Set("shares_outstanding = EXCLUDED.shares_outstanding").
		Set("per_bp = EXCLUDED.per_bp").
		Set("pbr_bp = EXCLUDED.pbr_bp").
		Set("psr_bp = EXCLUDED.psr_bp").
		Set("eps_minor = EXCLUDED.eps_minor").
		Set("bps_minor = EXCLUDED.bps_minor").
		Set("dividend_yield_bp = EXCLUDED.dividend_yield_bp").
		Set("provenance_json = EXCLUDED.provenance_json").
		Set("uncomputable_json = EXCLUDED.uncomputable_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("valuation_repository").With("company_id", snapshot.CompanyID, "instrument_id", snapshot.InstrumentID, "as_of", snapshot.AsOfDate).Wrapf(err, "upsert valuation snapshot")
	}
	return nil
}

func rowToSnapshot(row storage.ValuationSnapshotV1Row) Snapshot {
	provenance := make(map[string]any)
	if strings.TrimSpace(row.ProvenanceJSON) != "" {
		_ = json.Unmarshal([]byte(row.ProvenanceJSON), &provenance)
	}
	uncomputable := make(map[string]string)
	if strings.TrimSpace(row.UncomputableJSON) != "" {
		_ = json.Unmarshal([]byte(row.UncomputableJSON), &uncomputable)
	}
	return Snapshot{
		CompanyID:           row.CompanyID,
		InstrumentID:        row.InstrumentID,
		AsOfDate:            row.AsOfDate,
		SourcePriceDate:     row.SourcePriceDate,
		MarketCapMinor:      row.MarketCapMinor,
		ClosePriceMinor:     row.ClosePriceMinor,
		SharesOutstanding:   row.SharesOutstanding,
		PerBP:               row.PerBP,
		PbrBP:               row.PbrBP,
		PsrBP:               row.PsrBP,
		EpsMinor:            row.EpsMinor,
		BpsMinor:            row.BpsMinor,
		DividendYieldBP:     row.DividendYieldBP,
		MetricSourceVersion: row.MetricSourceVersion,
		Provenance:          provenance,
		Uncomputable:        uncomputable,
	}
}

func issuerInstrumentID(links []companyidentity.InstrumentLink) int64 {
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

func sharesOutstanding(marketCapMinor int64, closePriceMinor int64, priceScale int) *int64 {
	if marketCapMinor <= 0 || closePriceMinor <= 0 {
		return nil
	}
	shares := int64(math.Round(float64(marketCapMinor) * float64(pow10(priceScale)) / float64(closePriceMinor)))
	if shares <= 0 {
		return nil
	}
	return &shares
}

func ratioBP(numerator int64, denominator int64) *int64 {
	if denominator <= 0 {
		return nil
	}
	value := int64(math.Round(float64(numerator) / float64(denominator) * 10000))
	return &value
}

func perShareMinor(amountMinor int64, shares *int64) *int64 {
	if shares == nil || *shares <= 0 {
		return nil
	}
	value := int64(math.Round(float64(amountMinor) / float64(*shares)))
	return &value
}

func addAccountProvenance(provenance map[string]any, key string, account accountValue) {
	provenance[key] = map[string]any{
		"statement_id": account.StatementID,
		"fiscal_year":  account.FiscalYear,
		"report_code":  account.ReportCode,
		"fs_div":       account.FsDiv,
		"rcept_no":     account.RceptNo,
		"provider":     account.Provider,
		"group":        account.ProviderGroup,
		"operation":    account.Operation,
	}
}

func addDividendProvenance(provenance map[string]any, dividend dividendValue) {
	provenance["dividend"] = map[string]any{
		"fact_id":     dividend.FactID,
		"fiscal_year": dividend.FiscalYear,
		"report_code": dividend.ReportCode,
		"rcept_no":    dividend.RceptNo,
		"fact_date":   dividend.FactDate,
		"key":         dividend.Key,
		"provider":    dividend.Provider,
		"group":       dividend.Group,
		"operation":   dividend.Operation,
	}
}

func isCashDividendTotalKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	return strings.Contains(trimmed, "현금배당금총액")
}

func parseDateInt(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 8 && !strings.Contains(trimmed, "-") {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, oops.In("valuation_repository").With("date", value).Wrapf(err, "parse date")
		}
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return 0, oops.In("valuation_repository").With("date", value).Wrapf(err, "parse date")
	}
	return parsed.Year()*10000 + int(parsed.Month())*100 + parsed.Day(), nil
}

func formatDateInt(value int) string {
	year := value / 10000
	month := value / 100 % 100
	day := value % 100
	return strconv.Itoa(year) + "-" + twoDigits(month) + "-" + twoDigits(day)
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func pow10(scale int) int64 {
	value := int64(1)
	for i := 0; i < scale; i++ {
		value *= 10
	}
	return value
}
