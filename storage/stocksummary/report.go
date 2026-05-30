package stocksummary

import (
	"sort"
	"strconv"
	"strings"

	"github.com/awuzag/mwosa/packages/financialmetrics"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage/companyevent"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/financialmetric"
)

type StockInspectReport struct {
	Tables []ReportTable `json:"tables,omitempty"`
}

type ReportTable struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Header []string   `json:"header"`
	Rows   [][]string `json:"rows"`
}

type OverviewTable = ReportTable
type InvestmentMetricsTable = ReportTable
type FinancialSummaryTable = ReportTable
type PeriodMetricTable = ReportTable
type FinancialStatementTable = ReportTable
type DividendSummaryTable = ReportTable
type RiskSummaryTable = ReportTable
type MissingTable = ReportTable

type statementValue struct {
	Label    string
	Value    string
	Currency string
	Unit     string
	Ord      int
}

func BuildStockInspectReport(summary Summary, statements []financials.Statement, facts []companyfact.Fact) StockInspectReport {
	sections := sectionsSet(summary.Sections)
	statementValues := collectStatementValues(statements)
	metricValues := collectMetricValues(summary.Metrics)
	tables := make([]ReportTable, 0, 10)

	if hasReportSection(sections, SectionProfile) {
		tables = append(tables, buildOverviewTable(summary))
	}
	if hasReportSection(sections, SectionInvestment) || hasReportSection(sections, SectionScores) {
		tables = append(tables, buildInvestmentMetricsTable(summary, metricValues))
	}
	if hasReportSection(sections, SectionFinancials) {
		tables = append(tables,
			buildFinancialSummaryTable(statementValues, metricValues),
			buildPeriodMetricTable("profitability", "수익성 추이", []metricRowSpec{
				{Label: "매출", Account: "revenue", Statement: financials.StatementTypeIncomeStatement},
				{Label: "영업이익", Account: "operating_income", Statement: financials.StatementTypeIncomeStatement},
				{Label: "순이익", Account: "net_income", Statement: financials.StatementTypeIncomeStatement},
				{Label: "영업이익률", Metric: financialmetrics.MetricOperatingMargin, Percent: true},
				{Label: "순이익률", Metric: financialmetrics.MetricNetMargin, Percent: true},
			}, statementValues, metricValues),
			buildPeriodMetricTable("growth", "성장성 추이", []metricRowSpec{
				{Label: "매출 성장률", Metric: financialmetrics.MetricRevenueGrowthYoY, Percent: true},
				{Label: "영업이익 성장률", Metric: financialmetrics.MetricOperatingIncomeGrowthYoY, Percent: true},
				{Label: "순이익 성장률", Metric: financialmetrics.MetricNetIncomeGrowthYoY, Percent: true},
			}, statementValues, metricValues),
			buildPeriodMetricTable("stability", "안정성 추이", []metricRowSpec{
				{Label: "부채비율", Metric: financialmetrics.MetricDebtToEquity, Percent: true},
				{Label: "유동비율", Metric: financialmetrics.MetricCurrentRatio, Percent: true},
				{Label: "ROE", Metric: financialmetrics.MetricROE, Percent: true},
			}, statementValues, metricValues),
		)
		tables = append(tables, buildStatementTables(statementValues)...)
	}
	if hasReportSection(sections, SectionDividends) {
		tables = append(tables, buildDividendSummaryTable(summary, facts))
	}
	if hasReportSection(sections, SectionEvents) || len(facts) > 0 {
		tables = append(tables, buildRiskSummaryTable(summary, facts))
	}

	return StockInspectReport{Tables: dropEmptyTables(tables)}
}

func sectionsSet(sections []string) map[string]struct{} {
	out := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		out[section] = struct{}{}
	}
	return out
}

func hasReportSection(sections map[string]struct{}, section string) bool {
	_, ok := sections[section]
	return ok
}

func buildOverviewTable(summary Summary) OverviewTable {
	snapshot := latestSnapshot(summary)
	return OverviewTable{
		ID:     "overview",
		Title:  "회사 개요",
		Header: []string{"기업명", "종목코드", "시장", "대표이사", "상장일", "발행주식수", "시가총액", "기준일"},
		Rows: [][]string{{
			firstNonEmpty(summary.Company.Name, summary.Company.LegalName),
			summary.Instrument.Symbol,
			string(summary.Instrument.Market),
			"",
			"",
			formatIntPtr(snapshot.SharesOutstanding),
			formatIntPtr(snapshot.MarketCapMinor),
			firstNonEmpty(snapshot.AsOfDate, latestMetricDate(summary.Metrics)),
		}},
	}
}

func buildInvestmentMetricsTable(summary Summary, metrics map[string]map[string]financialmetric.Metric) InvestmentMetricsTable {
	snapshot := latestSnapshot(summary)
	latestYear := latestMetricYear(metrics)
	roe := metricValue(metrics[financialmetrics.MetricROE][latestYear], true)
	return InvestmentMetricsTable{
		ID:     "investment_metrics",
		Title:  "투자 지표",
		Header: []string{"PER", "PSR", "PBR", "EPS", "BPS", "ROE", "배당수익률"},
		Rows: [][]string{{
			formatMultipleBP(snapshot.PerBP),
			formatMultipleBP(snapshot.PsrBP),
			formatMultipleBP(snapshot.PbrBP),
			formatIntPtr(snapshot.EpsMinor),
			formatIntPtr(snapshot.BpsMinor),
			roe,
			formatPercentBP(snapshot.DividendYieldBP),
		}},
	}
}

func buildFinancialSummaryTable(statements map[financials.StatementType]map[string]map[string]statementValue, metrics map[string]map[string]financialmetric.Metric) FinancialSummaryTable {
	year := latestYear(statementYears(statements), latestMetricYear(metrics))
	return FinancialSummaryTable{
		ID:     "financial_summary",
		Title:  "재무 요약",
		Header: []string{"매출", "영업이익", "순이익", "영업이익률", "순이익률", "부채비율", "유동비율"},
		Rows: [][]string{{
			accountValue(statements, financials.StatementTypeIncomeStatement, year, "revenue"),
			accountValue(statements, financials.StatementTypeIncomeStatement, year, "operating_income"),
			accountValue(statements, financials.StatementTypeIncomeStatement, year, "net_income"),
			metricValue(metrics[financialmetrics.MetricOperatingMargin][year], true),
			metricValue(metrics[financialmetrics.MetricNetMargin][year], true),
			metricValue(metrics[financialmetrics.MetricDebtToEquity][year], true),
			metricValue(metrics[financialmetrics.MetricCurrentRatio][year], true),
		}},
	}
}

type metricRowSpec struct {
	Label     string
	Statement financials.StatementType
	Account   string
	Metric    string
	Percent   bool
}

func buildPeriodMetricTable(id string, title string, specs []metricRowSpec, statements map[financials.StatementType]map[string]map[string]statementValue, metrics map[string]map[string]financialmetric.Metric) PeriodMetricTable {
	years := mergeYears(statementYears(statements), metricYears(metrics))
	header := append([]string{"항목"}, years...)
	rows := make([][]string, 0, len(specs))
	for _, spec := range specs {
		row := []string{spec.Label}
		for _, year := range years {
			if spec.Metric != "" {
				row = append(row, metricValue(metrics[spec.Metric][year], spec.Percent))
				continue
			}
			row = append(row, accountValue(statements, spec.Statement, year, spec.Account))
		}
		rows = append(rows, row)
	}
	return PeriodMetricTable{ID: id, Title: title, Header: header, Rows: rows}
}

func buildStatementTables(statements map[financials.StatementType]map[string]map[string]statementValue) []ReportTable {
	specs := []struct {
		Type  financials.StatementType
		ID    string
		Title string
	}{
		{financials.StatementTypeIncomeStatement, "income_statement", "손익계산서"},
		{financials.StatementTypeBalanceSheet, "balance_sheet", "재무상태표"},
		{financials.StatementTypeCashFlow, "cash_flow", "현금흐름표"},
	}
	tables := make([]ReportTable, 0, len(specs))
	for _, spec := range specs {
		byYear := statements[spec.Type]
		years := yearsFromStatementType(byYear)
		if len(years) == 0 {
			continue
		}
		accounts := orderedAccounts(byYear)
		rows := make([][]string, 0, len(accounts))
		for _, account := range accounts {
			label := labelForAccount(byYear, account)
			row := []string{label}
			for _, year := range years {
				row = append(row, accountValue(statements, spec.Type, year, account))
			}
			rows = append(rows, row)
		}
		tables = append(tables, FinancialStatementTable{
			ID:     spec.ID,
			Title:  spec.Title,
			Header: append([]string{"항목"}, years...),
			Rows:   rows,
		})
	}
	return tables
}

func buildDividendSummaryTable(summary Summary, facts []companyfact.Fact) DividendSummaryTable {
	dividends := summary.Dividends
	if len(dividends) == 0 {
		for _, fact := range facts {
			if fact.FactType == companyfact.FactTypeDividend {
				dividends = append(dividends, fact)
			}
		}
	}
	latestTotal := latestFactValue(dividends, "현금배당금총액", true, false)
	latestDPS := latestFactValue(dividends, "주당 현금배당금", true, false)
	payoutRatio := latestFactValue(dividends, "현금배당성향", false, true)
	snapshot := latestSnapshot(summary)
	dividendYield := firstNonEmpty(formatPercentBP(snapshot.DividendYieldBP), latestFactValue(dividends, "현금배당수익률", false, true))
	frequency := ""
	if len(dividends) > 0 {
		frequency = "연간"
	}
	return DividendSummaryTable{
		ID:     "dividend_summary",
		Title:  "배당 요약",
		Header: []string{"최근 주당배당금", "배당수익률", "배당성향", "연간 배당총액", "배당 빈도"},
		Rows: [][]string{{
			latestDPS,
			dividendYield,
			payoutRatio,
			latestTotal,
			frequency,
		}},
	}
}

func buildRiskSummaryTable(summary Summary, facts []companyfact.Fact) RiskSummaryTable {
	return RiskSummaryTable{
		ID:     "risk_summary",
		Title:  "리스크 요약",
		Header: []string{"감사의견", "소송", "증자/감자", "합병/분할", "최근 이벤트"},
		Rows: [][]string{{
			auditOpinion(facts),
			eventSummary(summary.Events, "lawsuit"),
			eventSummary(summary.Events, "capital"),
			eventSummary(summary.Events, "merger", "division"),
			recentEvent(summary.Events),
		}},
	}
}

func buildMissingTable(summary Summary) MissingTable {
	rows := make([][]string, 0, len(summary.Missing))
	for _, missing := range summary.Missing {
		rows = append(rows, []string{missing.Section, "missing", missing.Reason})
	}
	for _, snapshot := range summary.Valuation {
		keys := sortedStringKeys(snapshot.Uncomputable)
		for _, key := range keys {
			rows = append(rows, []string{"investment:" + key, "unavailable", snapshot.Uncomputable[key]})
		}
	}
	for _, metric := range summary.Metrics {
		if strings.TrimSpace(metric.UncomputableReason) == "" {
			continue
		}
		rows = append(rows, []string{metric.Metric + ":" + metric.FiscalYear, "uncomputable", metric.UncomputableReason})
	}
	if summary.Scores != nil {
		keys := sortedStringKeys(summary.Scores.Uncomputable)
		for _, key := range keys {
			rows = append(rows, []string{"score:" + key, "uncomputable", summary.Scores.Uncomputable[key]})
		}
	}
	return MissingTable{
		ID:     "missing",
		Title:  "누락/계산 불가",
		Header: []string{"구분", "상태", "이유"},
		Rows:   rows,
	}
}

func dropEmptyTables(tables []ReportTable) []ReportTable {
	out := tables[:0]
	for _, table := range tables {
		if len(table.Header) == 0 || len(table.Rows) == 0 {
			continue
		}
		out = append(out, normalizeReportTable(table))
	}
	return out
}

func normalizeReportTable(table ReportTable) ReportTable {
	for rowIndex := range table.Rows {
		for columnIndex := range table.Rows[rowIndex] {
			if strings.TrimSpace(table.Rows[rowIndex][columnIndex]) == "" {
				table.Rows[rowIndex][columnIndex] = "-"
			}
		}
	}
	return table
}

func collectMetricValues(metrics []financialmetric.Metric) map[string]map[string]financialmetric.Metric {
	out := make(map[string]map[string]financialmetric.Metric)
	for _, metric := range metrics {
		if out[metric.Metric] == nil {
			out[metric.Metric] = make(map[string]financialmetric.Metric)
		}
		out[metric.Metric][metric.FiscalYear] = metric
	}
	return out
}

func collectStatementValues(statements []financials.Statement) map[financials.StatementType]map[string]map[string]statementValue {
	out := make(map[financials.StatementType]map[string]map[string]statementValue)
	for _, statement := range statements {
		if statement.Statement == "" || statement.FiscalYear == "" {
			continue
		}
		if out[statement.Statement] == nil {
			out[statement.Statement] = make(map[string]map[string]statementValue)
		}
		if out[statement.Statement][statement.FiscalYear] == nil {
			out[statement.Statement][statement.FiscalYear] = make(map[string]statementValue)
		}
		for index, line := range statement.Lines {
			account := strings.TrimSpace(line.Extensions["canonical_account"])
			if account == "" {
				continue
			}
			if !allowedStatementAccount(statement.Statement, account) {
				continue
			}
			if _, exists := out[statement.Statement][statement.FiscalYear][account]; exists {
				continue
			}
			out[statement.Statement][statement.FiscalYear][account] = statementValue{
				Label:    firstNonEmpty(canonicalAccountLabel(account), line.AccountName, account),
				Value:    firstNonEmpty(line.Value, line.Extensions["raw_amount"]),
				Currency: strings.TrimSpace(line.Currency),
				Unit:     strings.TrimSpace(line.Unit),
				Ord:      parseIntDefault(line.Extensions["ord"], index+1),
			}
		}
	}
	return out
}

func allowedStatementAccount(statement financials.StatementType, account string) bool {
	switch statement {
	case financials.StatementTypeIncomeStatement:
		return account == "revenue" || account == "operating_income" || account == "net_income"
	case financials.StatementTypeBalanceSheet:
		return account == "total_assets" || account == "total_liabilities" || account == "equity"
	case financials.StatementTypeCashFlow:
		return account == "operating_cashflow"
	default:
		return false
	}
}

func latestSnapshot(summary Summary) struct {
	AsOfDate          string
	MarketCapMinor    *int64
	SharesOutstanding *int64
	PerBP             *int64
	PbrBP             *int64
	PsrBP             *int64
	EpsMinor          *int64
	BpsMinor          *int64
	DividendYieldBP   *int64
	Uncomputable      map[string]string
} {
	if len(summary.Valuation) == 0 {
		return struct {
			AsOfDate          string
			MarketCapMinor    *int64
			SharesOutstanding *int64
			PerBP             *int64
			PbrBP             *int64
			PsrBP             *int64
			EpsMinor          *int64
			BpsMinor          *int64
			DividendYieldBP   *int64
			Uncomputable      map[string]string
		}{}
	}
	snapshot := summary.Valuation[0]
	return struct {
		AsOfDate          string
		MarketCapMinor    *int64
		SharesOutstanding *int64
		PerBP             *int64
		PbrBP             *int64
		PsrBP             *int64
		EpsMinor          *int64
		BpsMinor          *int64
		DividendYieldBP   *int64
		Uncomputable      map[string]string
	}{
		AsOfDate:          snapshot.AsOfDate,
		MarketCapMinor:    snapshot.MarketCapMinor,
		SharesOutstanding: snapshot.SharesOutstanding,
		PerBP:             snapshot.PerBP,
		PbrBP:             snapshot.PbrBP,
		PsrBP:             snapshot.PsrBP,
		EpsMinor:          snapshot.EpsMinor,
		BpsMinor:          snapshot.BpsMinor,
		DividendYieldBP:   snapshot.DividendYieldBP,
		Uncomputable:      snapshot.Uncomputable,
	}
}

func latestMetricDate(metrics []financialmetric.Metric) string {
	for _, metric := range metrics {
		if strings.TrimSpace(metric.AsOfDate) != "" {
			return metric.AsOfDate
		}
	}
	return ""
}

func latestMetricYear(metrics map[string]map[string]financialmetric.Metric) string {
	years := metricYears(metrics)
	if len(years) == 0 {
		return ""
	}
	return years[len(years)-1]
}

func latestYear(left []string, right string) string {
	if len(left) > 0 {
		return left[len(left)-1]
	}
	return right
}

func accountValue(statements map[financials.StatementType]map[string]map[string]statementValue, statement financials.StatementType, year string, account string) string {
	if byYear := statements[statement]; byYear != nil {
		if values := byYear[year]; values != nil {
			value := values[account]
			return formatAmountWithCurrency(value.Value, value.Currency)
		}
	}
	return ""
}

func metricValue(metric financialmetric.Metric, percent bool) string {
	if strings.TrimSpace(metric.UncomputableReason) != "" {
		return ""
	}
	if metric.ValueBP != nil {
		if percent {
			return formatPercentBP(metric.ValueBP)
		}
		return formatMultipleBP(metric.ValueBP)
	}
	if metric.ValueDecimal != "" {
		return metric.ValueDecimal
	}
	return formatIntPtr(metric.ValueMinor)
}

func statementYears(statements map[financials.StatementType]map[string]map[string]statementValue) []string {
	seen := make(map[string]struct{})
	for _, byYear := range statements {
		for year := range byYear {
			seen[year] = struct{}{}
		}
	}
	return sortedYears(seen)
}

func metricYears(metrics map[string]map[string]financialmetric.Metric) []string {
	seen := make(map[string]struct{})
	for _, byYear := range metrics {
		for year := range byYear {
			seen[year] = struct{}{}
		}
	}
	return sortedYears(seen)
}

func mergeYears(groups ...[]string) []string {
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, year := range group {
			if strings.TrimSpace(year) != "" {
				seen[year] = struct{}{}
			}
		}
	}
	return sortedYears(seen)
}

func sortedYears(seen map[string]struct{}) []string {
	years := make([]string, 0, len(seen))
	for year := range seen {
		years = append(years, year)
	}
	sort.Strings(years)
	return years
}

func yearsFromStatementType(byYear map[string]map[string]statementValue) []string {
	seen := make(map[string]struct{}, len(byYear))
	for year := range byYear {
		seen[year] = struct{}{}
	}
	return sortedYears(seen)
}

func orderedAccounts(byYear map[string]map[string]statementValue) []string {
	type accountOrder struct {
		Account string
		Ord     int
	}
	seen := make(map[string]accountOrder)
	for _, values := range byYear {
		for account, value := range values {
			current, ok := seen[account]
			if !ok || value.Ord < current.Ord {
				seen[account] = accountOrder{Account: account, Ord: value.Ord}
			}
		}
	}
	orders := make([]accountOrder, 0, len(seen))
	for _, order := range seen {
		orders = append(orders, order)
	}
	sort.Slice(orders, func(i int, j int) bool {
		if orders[i].Ord == orders[j].Ord {
			return orders[i].Account < orders[j].Account
		}
		return orders[i].Ord < orders[j].Ord
	})
	accounts := make([]string, 0, len(orders))
	for _, order := range orders {
		accounts = append(accounts, order.Account)
	}
	return accounts
}

func labelForAccount(byYear map[string]map[string]statementValue, account string) string {
	for _, values := range byYear {
		if value, ok := values[account]; ok && strings.TrimSpace(value.Label) != "" {
			return value.Label
		}
	}
	return firstNonEmpty(canonicalAccountLabel(account), account)
}

func canonicalAccountLabel(account string) string {
	switch account {
	case "revenue":
		return "매출"
	case "operating_income":
		return "영업이익"
	case "net_income":
		return "순이익"
	case "total_assets":
		return "자산총계"
	case "total_liabilities":
		return "부채총계"
	case "equity":
		return "자본총계"
	case "operating_cashflow":
		return "영업현금흐름"
	default:
		return ""
	}
}

func latestFactValue(facts []companyfact.Fact, contains string, includeCurrency bool, includePercent bool) string {
	for _, fact := range facts {
		if strings.Contains(fact.Key, contains) {
			value := firstNonEmpty(fact.ValueText, fact.ValueNumber)
			if includePercent {
				return formatPercentText(value)
			}
			if includeCurrency {
				return formatAmountWithCurrency(value, fact.CurrencyCode)
			}
			return value
		}
	}
	return ""
}

func auditOpinion(facts []companyfact.Fact) string {
	fallback := ""
	for _, fact := range facts {
		if fact.FactType == "audit_opinion" {
			value := firstNonEmpty(fact.ValueText, fact.ValueNumber)
			key := strings.ToLower(fact.Key)
			if strings.Contains(key, "opinion") || strings.Contains(fact.Key, "감사의견") || looksLikeAuditOpinion(value) {
				return value
			}
			if fallback == "" {
				fallback = value
			}
		}
	}
	return fallback
}

func looksLikeAuditOpinion(value string) bool {
	for _, token := range []string{"적정", "한정", "부적정", "의견거절"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func eventSummary(events []companyevent.Event, needles ...string) string {
	for _, event := range events {
		lower := strings.ToLower(event.EventType)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				return firstNonEmpty(event.Title, event.EventType)
			}
		}
	}
	return ""
}

func recentEvent(events []companyevent.Event) string {
	if len(events) == 0 {
		return ""
	}
	return firstNonEmpty(events[0].Title, events[0].EventType)
}

func formatIntPtr(value *int64) string {
	if value == nil {
		return ""
	}
	return formatInt(*value)
}

func formatAmountWithCurrency(value string, currency string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	code := strings.TrimSpace(currency)
	if code == "" {
		return trimmed
	}
	return trimmed + " " + code
}

func formatPercentText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "%") {
		return trimmed
	}
	return trimmed + "%"
}

func formatInt(value int64) string {
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return text
	}
	var builder strings.Builder
	prefix := len(text) % 3
	if prefix == 0 {
		prefix = 3
	}
	builder.WriteString(text[:prefix])
	for i := prefix; i < len(text); i += 3 {
		builder.WriteString(",")
		builder.WriteString(text[i : i+3])
	}
	return builder.String()
}

func formatMultipleBP(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*value)/10000, 'f', 2, 64)
}

func formatPercentBP(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*value)/100, 'f', 2, 64) + "%"
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
