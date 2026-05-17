package financialmetric

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ev3rlit/mwosa/packages/financialmetrics"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type Repository struct {
	database *storage.Database
}

type CalculateOptions struct {
	WindowYears int
	Period      financials.PeriodType
}

type Query struct {
	WindowYears int
	Period      financials.PeriodType
}

type UpsertResult struct {
	MetricsCalculated int `json:"metrics_calculated" csv:"metrics_calculated"`
	MetricsWritten    int `json:"metrics_written" csv:"metrics_written"`
	Uncomputable      int `json:"uncomputable" csv:"uncomputable"`
}

type Metric struct {
	CompanyID          int64                 `json:"company_id" csv:"company_id"`
	InstrumentID       int64                 `json:"instrument_id,omitempty" csv:"instrument_id"`
	StatementID        int64                 `json:"statement_id,omitempty" csv:"statement_id"`
	Metric             string                `json:"metric" csv:"metric"`
	FiscalYear         string                `json:"fiscal_year" csv:"fiscal_year"`
	FiscalPeriod       financials.PeriodType `json:"fiscal_period,omitempty" csv:"fiscal_period"`
	AsOfDate           string                `json:"as_of_date,omitempty" csv:"as_of_date"`
	ValueDecimal       string                `json:"value_decimal,omitempty" csv:"value_decimal"`
	ValueBP            *int64                `json:"value_bp,omitempty" csv:"value_bp"`
	ValueMinor         *int64                `json:"value_minor,omitempty" csv:"value_minor"`
	FormulaVersion     string                `json:"formula_version" csv:"formula_version"`
	UncomputableReason string                `json:"uncomputable_reason,omitempty" csv:"uncomputable_reason"`
	Provenance         map[string]any        `json:"provenance,omitempty" csv:"-"`
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("financial_metric_repository").New("financial metric repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) CalculateAndUpsert(ctx context.Context, company companyidentity.InspectResult, options CalculateOptions) (UpsertResult, error) {
	errb := oops.In("financial_metric_repository").With("company_id", company.Company.ID, "window_years", options.WindowYears, "period", options.Period)
	if company.Company.ID == 0 {
		return UpsertResult{}, errb.New("financial metric calculation requires canonical company")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	values, err := selectAccountValues(ctx, client, company.Company.ID, options.Period)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	if len(values) == 0 {
		return UpsertResult{}, errb.New("financial metric calculation requires stored financial line items")
	}
	metrics := financialmetrics.Calculate(values, options.WindowYears)
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, errb.Wrapf(err, "begin financial metric sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	nowMS := time.Now().UTC().UnixMilli()
	result := UpsertResult{MetricsCalculated: len(metrics)}
	for _, metric := range metrics {
		if metric.UncomputableReason != "" {
			result.Uncomputable++
		}
		if err := upsertMetric(ctx, tx, company.Company.ID, metric, nowMS); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.MetricsWritten++
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, errb.Wrapf(err, "commit financial metric sqlite transaction")
	}
	committed = true
	return result, nil
}

func (r Repository) ListMetrics(ctx context.Context, company companyidentity.InspectResult, query Query) ([]Metric, error) {
	errb := oops.In("financial_metric_repository").With("company_id", company.Company.ID, "window_years", query.WindowYears, "period", query.Period)
	if company.Company.ID == 0 {
		return nil, errb.New("financial metric query requires canonical company")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows := make([]storage.FinancialMetricV1Row, 0)
	selectQuery := client.NewSelect().
		Model(&rows).
		Where("company_id = ?", company.Company.ID).
		Order("fiscal_year DESC", "metric ASC")
	if query.Period != "" {
		selectQuery = selectQuery.Where("fiscal_period = ?", string(query.Period))
	}
	if query.WindowYears > 0 {
		selectQuery = selectQuery.Limit(query.WindowYears * 10)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "select financial metrics")
	}
	out := make([]Metric, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToMetric(row))
	}
	return out, nil
}

func selectAccountValues(ctx context.Context, db *bun.DB, companyID int64, period financials.PeriodType) ([]financialmetrics.AccountValue, error) {
	var rows []struct {
		StatementID      int64  `bun:"statement_id"`
		InstrumentID     int64  `bun:"instrument_id"`
		Provider         string `bun:"provider"`
		ProviderGroup    string `bun:"provider_group"`
		Operation        string `bun:"operation"`
		ReportCode       string `bun:"report_code"`
		FsDiv            string `bun:"fs_div"`
		RceptNo          string `bun:"rcept_no"`
		FiscalYear       string `bun:"fiscal_year"`
		FiscalPeriod     string `bun:"fiscal_period"`
		AsOfDate         string `bun:"as_of_date"`
		CanonicalAccount string `bun:"canonical_account"`
		AmountMinor      *int64 `bun:"amount_minor"`
	}
	query := db.NewSelect().
		TableExpr("financial_line_item_v1 AS l").
		ColumnExpr("s.id AS statement_id").
		ColumnExpr("s.instrument_id").
		ColumnExpr("s.provider").
		ColumnExpr("s.provider_group").
		ColumnExpr("s.operation").
		ColumnExpr("s.report_code").
		ColumnExpr("s.fs_div").
		ColumnExpr("s.rcept_no").
		ColumnExpr("s.fiscal_year").
		ColumnExpr("s.fiscal_period").
		ColumnExpr("s.reported_at AS as_of_date").
		ColumnExpr("l.canonical_account").
		ColumnExpr("l.amount_minor").
		Join("JOIN financial_statement_v1 AS s ON s.id = l.statement_id").
		Where("s.company_id = ?", companyID).
		Where("l.canonical_account != ''").
		Where("l.amount_minor IS NOT NULL")
	if period != "" {
		query = query.Where("s.fiscal_period = ?", string(period))
	}
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, oops.In("financial_metric_repository").With("company_id", companyID).Wrapf(err, "select financial metric account values")
	}
	values := make([]financialmetrics.AccountValue, 0, len(rows))
	for _, row := range rows {
		if row.AmountMinor == nil {
			continue
		}
		asOfDate := strings.TrimSpace(row.AsOfDate)
		if asOfDate == "" {
			asOfDate = inferredAsOfDate(row.FiscalYear, row.ReportCode)
		}
		values = append(values, financialmetrics.AccountValue{
			StatementID:      row.StatementID,
			InstrumentID:     row.InstrumentID,
			Provider:         row.Provider,
			ProviderGroup:    row.ProviderGroup,
			Operation:        row.Operation,
			ReportCode:       row.ReportCode,
			FsDiv:            row.FsDiv,
			RceptNo:          row.RceptNo,
			FiscalYear:       row.FiscalYear,
			FiscalPeriod:     row.FiscalPeriod,
			AsOfDate:         asOfDate,
			CanonicalAccount: row.CanonicalAccount,
			AmountMinor:      *row.AmountMinor,
		})
	}
	return values, nil
}

func upsertMetric(ctx context.Context, tx bun.Tx, companyID int64, metric financialmetrics.Metric, nowMS int64) error {
	provenance, err := json.Marshal(metric.Provenance)
	if err != nil {
		return oops.In("financial_metric_repository").With("company_id", companyID, "metric", metric.Metric).Wrapf(err, "encode financial metric provenance")
	}
	row := storage.FinancialMetricV1Row{
		CompanyID:          companyID,
		InstrumentID:       metric.InstrumentID,
		StatementID:        metric.StatementID,
		Metric:             metric.Metric,
		FiscalYear:         metric.FiscalYear,
		FiscalPeriod:       metric.FiscalPeriod,
		AsOfDate:           metric.AsOfDate,
		ValueDecimal:       metric.ValueDecimal,
		ValueBP:            metric.ValueBP,
		ValueMinor:         metric.ValueMinor,
		FormulaVersion:     metric.FormulaVersion,
		ProvenanceJSON:     string(provenance),
		UncomputableReason: metric.UncomputableReason,
		CreatedAtMS:        nowMS,
		UpdatedAtMS:        nowMS,
	}
	if row.Metric == "" || row.FiscalYear == "" || row.FormulaVersion == "" {
		return oops.In("financial_metric_repository").With("company_id", companyID, "metric", metric.Metric).New("financial metric missing natural key")
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (company_id, instrument_id, metric, fiscal_year, fiscal_period, as_of_date, formula_version) DO UPDATE").
		Set("statement_id = EXCLUDED.statement_id").
		Set("value_decimal = EXCLUDED.value_decimal").
		Set("value_bp = EXCLUDED.value_bp").
		Set("value_minor = EXCLUDED.value_minor").
		Set("provenance_json = EXCLUDED.provenance_json").
		Set("uncomputable_reason = EXCLUDED.uncomputable_reason").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return oops.In("financial_metric_repository").With("company_id", companyID, "metric", row.Metric, "fiscal_year", row.FiscalYear).Wrapf(err, "upsert financial metric")
	}
	return nil
}

func rowToMetric(row storage.FinancialMetricV1Row) Metric {
	provenance := make(map[string]any)
	if strings.TrimSpace(row.ProvenanceJSON) != "" {
		_ = json.Unmarshal([]byte(row.ProvenanceJSON), &provenance)
	}
	return Metric{
		CompanyID:          row.CompanyID,
		InstrumentID:       row.InstrumentID,
		StatementID:        row.StatementID,
		Metric:             row.Metric,
		FiscalYear:         row.FiscalYear,
		FiscalPeriod:       financials.PeriodType(row.FiscalPeriod),
		AsOfDate:           row.AsOfDate,
		ValueDecimal:       row.ValueDecimal,
		ValueBP:            row.ValueBP,
		ValueMinor:         row.ValueMinor,
		FormulaVersion:     row.FormulaVersion,
		UncomputableReason: row.UncomputableReason,
		Provenance:         provenance,
	}
}

func inferredAsOfDate(fiscalYear string, reportCode string) string {
	switch strings.TrimSpace(reportCode) {
	case "11012":
		return fiscalYear + "-03-31"
	case "11013":
		return fiscalYear + "-06-30"
	case "11014":
		return fiscalYear + "-09-30"
	default:
		return fiscalYear + "-12-31"
	}
}

var _ = provider.ProviderOpenDART
