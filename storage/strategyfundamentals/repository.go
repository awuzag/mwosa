package strategyfundamentals

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
)

const (
	defaultFactLimit  = 40
	defaultEventLimit = 20
)

type Repository struct {
	database *storage.Database
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("strategy_fundamentals_repository").New("strategy fundamentals repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) ListLatestFundamentals(ctx context.Context, query strategyservice.FundamentalsQuery) (map[string]strategyservice.Fundamentals, error) {
	errb := oops.In("strategy_fundamentals_repository").With("market", query.Market, "security_type", query.SecurityType)
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	out := map[string]strategyservice.Fundamentals{}
	if err := r.loadMetrics(ctx, client, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadValuations(ctx, client, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadFacts(ctx, client, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	if err := r.loadEvents(ctx, client, query, out); err != nil {
		return nil, errb.Wrap(err)
	}
	return out, nil
}

func (r Repository) loadMetrics(ctx context.Context, client queryer, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	rows, err := client.QueryContext(ctx, `
SELECT
	i.symbol,
	fm.metric,
	fm.fiscal_year,
	fm.fiscal_period,
	fm.as_of_date,
	fm.value_decimal,
	fm.value_bp,
	fm.value_minor,
	fm.formula_version,
	fm.uncomputable_reason
FROM financial_metric_v1 AS fm
JOIN company_v1 AS c ON c.id = fm.company_id
JOIN instrument_company_link_v1 AS l ON l.company_id = c.id AND l.relation_type = 'issuer'
JOIN instrument_v2 AS i ON i.id = l.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE m.code = ? AND i.security_type = ?
ORDER BY i.symbol ASC, fm.metric ASC, fm.fiscal_year DESC, fm.as_of_date DESC, fm.id DESC
`, market(query.Market), securityType(query.SecurityType))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("market", query.Market, "security_type", query.SecurityType).Wrapf(err, "query financial metric screen fundamentals")
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	for rows.Next() {
		var row metricRow
		if err := rows.Scan(
			&row.Symbol,
			&row.Metric,
			&row.FiscalYear,
			&row.FiscalPeriod,
			&row.AsOfDate,
			&row.ValueDecimal,
			&row.ValueBP,
			&row.ValueMinor,
			&row.FormulaVersion,
			&row.UncomputableReason,
		); err != nil {
			return oops.In("strategy_fundamentals_repository").Wrapf(err, "scan financial metric screen fundamentals")
		}
		key := row.Symbol + "\x00" + row.Metric
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item := out[row.Symbol]
		item.Symbol = row.Symbol
		if item.Metrics == nil {
			item.Metrics = map[string]strategyservice.FundamentalMetric{}
		}
		item.Metrics[row.Metric] = strategyservice.FundamentalMetric{
			FiscalYear:         row.FiscalYear,
			FiscalPeriod:       row.FiscalPeriod,
			AsOfDate:           row.AsOfDate,
			ValueDecimal:       row.ValueDecimal,
			ValueBP:            int64Ptr(row.ValueBP),
			ValueMinor:         int64Ptr(row.ValueMinor),
			FormulaVersion:     row.FormulaVersion,
			UncomputableReason: row.UncomputableReason,
		}
		out[row.Symbol] = item
	}
	if err := rows.Err(); err != nil {
		return oops.In("strategy_fundamentals_repository").Wrapf(err, "iterate financial metric screen fundamentals")
	}
	return nil
}

func (r Repository) loadValuations(ctx context.Context, client queryer, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	rows, err := client.QueryContext(ctx, `
SELECT
	i.symbol,
	v.as_of_date,
	v.source_price_date,
	v.market_cap_minor,
	v.close_price_minor,
	v.shares_outstanding,
	v.per_bp,
	v.pbr_bp,
	v.psr_bp,
	v.eps_minor,
	v.bps_minor,
	v.dividend_yield_bp,
	v.metric_source_version,
	v.uncomputable_json
FROM valuation_snapshot_v1 AS v
JOIN instrument_v2 AS i ON i.id = v.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE m.code = ? AND i.security_type = ?
ORDER BY i.symbol ASC, v.as_of_date DESC, v.id DESC
`, market(query.Market), securityType(query.SecurityType))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("market", query.Market, "security_type", query.SecurityType).Wrapf(err, "query valuation screen fundamentals")
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	for rows.Next() {
		var row valuationRow
		if err := rows.Scan(
			&row.Symbol,
			&row.AsOfDate,
			&row.SourcePriceDate,
			&row.MarketCapMinor,
			&row.ClosePriceMinor,
			&row.SharesOutstanding,
			&row.PerBP,
			&row.PbrBP,
			&row.PsrBP,
			&row.EpsMinor,
			&row.BpsMinor,
			&row.DividendYieldBP,
			&row.MetricSourceVersion,
			&row.UncomputableJSON,
		); err != nil {
			return oops.In("strategy_fundamentals_repository").Wrapf(err, "scan valuation screen fundamentals")
		}
		if _, ok := seen[row.Symbol]; ok {
			continue
		}
		seen[row.Symbol] = struct{}{}
		item := out[row.Symbol]
		item.Symbol = row.Symbol
		item.Valuation = &strategyservice.FundamentalValuation{
			AsOfDate:            row.AsOfDate,
			SourcePriceDate:     row.SourcePriceDate,
			MarketCapMinor:      int64Ptr(row.MarketCapMinor),
			ClosePriceMinor:     int64Ptr(row.ClosePriceMinor),
			SharesOutstanding:   int64Ptr(row.SharesOutstanding),
			PerBP:               int64Ptr(row.PerBP),
			PbrBP:               int64Ptr(row.PbrBP),
			PsrBP:               int64Ptr(row.PsrBP),
			EpsMinor:            int64Ptr(row.EpsMinor),
			BpsMinor:            int64Ptr(row.BpsMinor),
			DividendYieldBP:     int64Ptr(row.DividendYieldBP),
			MetricSourceVersion: row.MetricSourceVersion,
			Uncomputable:        uncomputableMap(row.UncomputableJSON),
		}
		out[row.Symbol] = item
	}
	if err := rows.Err(); err != nil {
		return oops.In("strategy_fundamentals_repository").Wrapf(err, "iterate valuation screen fundamentals")
	}
	return nil
}

func (r Repository) loadFacts(ctx context.Context, client queryer, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	rows, err := client.QueryContext(ctx, `
SELECT
	i.symbol,
	f.fact_type,
	f.fiscal_year,
	f.report_code,
	f.rcept_no,
	f.fact_date,
	f.key,
	f.value_text,
	f.value_number,
	f.currency_code,
	f.provider,
	f.provider_group,
	f.operation
FROM company_fact_v1 AS f
JOIN company_v1 AS c ON c.id = f.company_id
JOIN instrument_company_link_v1 AS l ON l.company_id = c.id AND l.relation_type = 'issuer'
JOIN instrument_v2 AS i ON i.id = l.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE m.code = ? AND i.security_type = ?
ORDER BY i.symbol ASC, f.fact_type ASC, f.key ASC, f.fiscal_year DESC, f.report_code DESC, f.fact_date DESC, f.id DESC
`, market(query.Market), securityType(query.SecurityType))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("market", query.Market, "security_type", query.SecurityType).Wrapf(err, "query company fact screen fundamentals")
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	counts := map[string]int{}
	for rows.Next() {
		var row factRow
		if err := rows.Scan(
			&row.Symbol,
			&row.FactType,
			&row.FiscalYear,
			&row.ReportCode,
			&row.RceptNo,
			&row.FactDate,
			&row.Key,
			&row.ValueText,
			&row.ValueNumber,
			&row.CurrencyCode,
			&row.Provider,
			&row.Group,
			&row.Operation,
		); err != nil {
			return oops.In("strategy_fundamentals_repository").Wrapf(err, "scan company fact screen fundamentals")
		}
		if counts[row.Symbol] >= defaultFactLimit {
			continue
		}
		key := factMapKey(row)
		seenKey := row.Symbol + "\x00" + key
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}
		counts[row.Symbol]++

		item := out[row.Symbol]
		item.Symbol = row.Symbol
		if item.Facts == nil {
			item.Facts = map[string]strategyservice.FundamentalFact{}
		}
		item.Facts[key] = strategyservice.FundamentalFact{
			FactType:     row.FactType,
			FiscalYear:   row.FiscalYear,
			ReportCode:   row.ReportCode,
			RceptNo:      row.RceptNo,
			FactDate:     row.FactDate,
			Key:          row.Key,
			ValueText:    row.ValueText,
			ValueNumber:  row.ValueNumber,
			CurrencyCode: row.CurrencyCode,
			Provider:     row.Provider,
			Group:        row.Group,
			Operation:    row.Operation,
		}
		out[row.Symbol] = item
	}
	if err := rows.Err(); err != nil {
		return oops.In("strategy_fundamentals_repository").Wrapf(err, "iterate company fact screen fundamentals")
	}
	return nil
}

func (r Repository) loadEvents(ctx context.Context, client queryer, query strategyservice.FundamentalsQuery, out map[string]strategyservice.Fundamentals) error {
	rows, err := client.QueryContext(ctx, `
SELECT
	i.symbol,
	e.event_type,
	e.event_date,
	e.rcept_dt,
	e.rcept_no,
	e.title,
	e.amount_minor,
	e.value_text,
	e.provider,
	e.provider_group,
	e.operation
FROM company_event_v1 AS e
JOIN company_v1 AS c ON c.id = e.company_id
JOIN instrument_company_link_v1 AS l ON l.company_id = c.id AND l.relation_type = 'issuer'
JOIN instrument_v2 AS i ON i.id = l.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE m.code = ? AND i.security_type = ?
ORDER BY i.symbol ASC, COALESCE(NULLIF(e.event_date, ''), e.rcept_dt) DESC, e.id DESC
`, market(query.Market), securityType(query.SecurityType))
	if err != nil {
		return oops.In("strategy_fundamentals_repository").With("market", query.Market, "security_type", query.SecurityType).Wrapf(err, "query company event screen fundamentals")
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var row eventRow
		if err := rows.Scan(
			&row.Symbol,
			&row.EventType,
			&row.EventDate,
			&row.RceptDt,
			&row.RceptNo,
			&row.Title,
			&row.AmountMinor,
			&row.ValueText,
			&row.Provider,
			&row.Group,
			&row.Operation,
		); err != nil {
			return oops.In("strategy_fundamentals_repository").Wrapf(err, "scan company event screen fundamentals")
		}
		if counts[row.Symbol] >= defaultEventLimit {
			continue
		}
		counts[row.Symbol]++

		item := out[row.Symbol]
		item.Symbol = row.Symbol
		item.Events = append(item.Events, strategyservice.FundamentalEvent{
			EventType:   row.EventType,
			EventDate:   row.EventDate,
			RceptDt:     row.RceptDt,
			RceptNo:     row.RceptNo,
			Title:       row.Title,
			AmountMinor: int64Ptr(row.AmountMinor),
			ValueText:   row.ValueText,
			Provider:    row.Provider,
			Group:       row.Group,
			Operation:   row.Operation,
		})
		out[row.Symbol] = item
	}
	if err := rows.Err(); err != nil {
		return oops.In("strategy_fundamentals_repository").Wrapf(err, "iterate company event screen fundamentals")
	}
	return nil
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type metricRow struct {
	Symbol             string
	Metric             string
	FiscalYear         string
	FiscalPeriod       string
	AsOfDate           string
	ValueDecimal       string
	ValueBP            sql.NullInt64
	ValueMinor         sql.NullInt64
	FormulaVersion     string
	UncomputableReason string
}

type valuationRow struct {
	Symbol              string
	AsOfDate            string
	SourcePriceDate     string
	MarketCapMinor      sql.NullInt64
	ClosePriceMinor     sql.NullInt64
	SharesOutstanding   sql.NullInt64
	PerBP               sql.NullInt64
	PbrBP               sql.NullInt64
	PsrBP               sql.NullInt64
	EpsMinor            sql.NullInt64
	BpsMinor            sql.NullInt64
	DividendYieldBP     sql.NullInt64
	MetricSourceVersion string
	UncomputableJSON    string
}

type factRow struct {
	Symbol       string
	FactType     string
	FiscalYear   string
	ReportCode   string
	RceptNo      string
	FactDate     string
	Key          string
	ValueText    string
	ValueNumber  string
	CurrencyCode string
	Provider     string
	Group        string
	Operation    string
}

type eventRow struct {
	Symbol      string
	EventType   string
	EventDate   string
	RceptDt     string
	RceptNo     string
	Title       string
	AmountMinor sql.NullInt64
	ValueText   string
	Provider    string
	Group       string
	Operation   string
}

func market(value provider.Market) string {
	if value == "" {
		return string(provider.MarketKRX)
	}
	return string(value)
}

func securityType(value provider.SecurityType) string {
	if value == "" {
		return string(provider.SecurityTypeStock)
	}
	return string(value)
}

func int64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

func uncomputableMap(raw string) map[string]string {
	out := map[string]string{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func factMapKey(row factRow) string {
	factType := strings.TrimSpace(row.FactType)
	key := strings.TrimSpace(row.Key)
	if key == "" || key == factType {
		return factType
	}
	return factType + "." + key
}
