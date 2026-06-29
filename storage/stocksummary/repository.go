package stocksummary

import (
	"context"
	"strconv"
	"strings"

	"github.com/awuzag/mwosa/packages/financialscores"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/service/research"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyevent"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialmetric"
	"github.com/awuzag/mwosa/storage/financialstatement"
	"github.com/awuzag/mwosa/storage/valuation"
	"github.com/samber/oops"
)

const (
	SectionAll        = "all"
	SectionProfile    = "profile"
	SectionInvestment = "investment"
	SectionFinancials = "financials"
	SectionScores     = "scores"
	SectionDividends  = "dividends"
	SectionFacts      = "facts"
	SectionEvents     = "events"
)

var defaultSections = []string{SectionProfile, SectionInvestment, SectionFinancials, SectionScores, SectionDividends, SectionEvents}

type Repository struct {
	database *storage.SQLDatabase
}

type Query struct {
	Sections []string
	AsOf     string
	Window   int
	Period   financials.PeriodType
}

type Summary struct {
	Sections    []string                       `json:"sections"`
	Company     companyidentity.Company        `json:"company"`
	Instrument  companyidentity.InstrumentLink `json:"instrument,omitempty"`
	Identifiers []companyidentity.Identifier   `json:"identifiers,omitempty"`
	Valuation   []valuation.Snapshot           `json:"valuation,omitempty"`
	Metrics     []financialmetric.Metric       `json:"metrics,omitempty"`
	Scores      *financialscores.Scores        `json:"fundamental_scores,omitempty"`
	Dividends   []companyfact.Fact             `json:"dividends,omitempty"`
	Facts       []companyfact.Fact             `json:"facts,omitempty"`
	Events      []companyevent.Event           `json:"events,omitempty"`
	Missing     []MissingSection               `json:"missing,omitempty"`
	Profile     research.StockResearchProfile  `json:"research_profile,omitempty"`
	Report      StockInspectReport             `json:"report,omitempty"`
}

type MissingSection struct {
	Section string `json:"section" csv:"section"`
	Reason  string `json:"reason" csv:"reason"`
}

func NewRepository(database *storage.SQLDatabase) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("stock_summary_repository").New("stock summary repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) Inspect(ctx context.Context, query string, options Query) (Summary, error) {
	errb := oops.In("stock_summary_repository").With("query", query)
	identityRepository, err := companyidentity.NewRepository(r.database)
	if err != nil {
		return Summary{}, errb.Wrap(err)
	}
	company, err := identityRepository.Inspect(ctx, query)
	if err != nil {
		return Summary{}, errb.Wrap(err)
	}
	sections := normalizeSections(options.Sections)
	out := Summary{
		Sections:    sectionList(sections),
		Company:     company.Company,
		Instrument:  issuerInstrument(company.Instruments),
		Identifiers: company.Identifiers,
	}
	var snapshots []valuation.Snapshot
	if hasSection(sections, SectionInvestment) || hasSection(sections, SectionScores) {
		snapshots, err = listValuation(ctx, r.database, company, options.AsOf)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		if hasSection(sections, SectionInvestment) {
			out.Valuation = snapshots
			if len(snapshots) == 0 {
				out.Missing = append(out.Missing, MissingSection{Section: SectionInvestment, Reason: "stored valuation snapshot not found"})
			}
		}
	}
	var metrics []financialmetric.Metric
	var statements []financials.Statement
	if hasSection(sections, SectionFinancials) || hasSection(sections, SectionScores) {
		metrics, err = listMetrics(ctx, r.database, company, options.Window, options.Period)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		if hasSection(sections, SectionFinancials) {
			out.Metrics = metrics
			if len(metrics) == 0 {
				out.Missing = append(out.Missing, MissingSection{Section: SectionFinancials, Reason: "stored financial metrics not found"})
			}
			statements, err = listStatements(ctx, r.database, company, options.Period)
			if err != nil {
				return Summary{}, errb.Wrap(err)
			}
			if len(statements) == 0 {
				out.Missing = append(out.Missing, MissingSection{Section: "statements", Reason: "stored financial statements not found"})
			}
		}
	}
	if hasSection(sections, SectionScores) {
		scores := buildScores(metrics, snapshots)
		if scores.HasSignal() {
			out.Scores = &scores
		} else {
			out.Missing = append(out.Missing, MissingSection{Section: SectionScores, Reason: "stored score inputs not found"})
		}
	}
	if hasSection(sections, SectionDividends) {
		dividends, err := listDividends(ctx, r.database, company, options.Window)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		out.Dividends = dividends
		if len(dividends) == 0 {
			out.Missing = append(out.Missing, MissingSection{Section: SectionDividends, Reason: "stored dividend facts not found"})
		}
	}
	if hasSection(sections, SectionFacts) {
		facts, err := listFacts(ctx, r.database, company, options.Window)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		out.Facts = facts
		if len(facts) == 0 {
			out.Missing = append(out.Missing, MissingSection{Section: SectionFacts, Reason: "stored company facts not found"})
		}
	}
	reportFacts := out.Facts
	if !hasSection(sections, SectionFacts) && (hasSection(sections, SectionDividends) || hasSection(sections, SectionEvents)) {
		reportFacts, err = listFacts(ctx, r.database, company, options.Window)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
	}
	if hasSection(sections, SectionEvents) {
		events, err := listEvents(ctx, r.database, company)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		out.Events = events
		if len(events) == 0 {
			out.Missing = append(out.Missing, MissingSection{Section: SectionEvents, Reason: "stored company events not found"})
		}
	}
	out.Profile = buildResearchProfile(out)
	out.Report = BuildStockInspectReport(out, statements, reportFacts)
	return out, nil
}

func normalizeSections(sections []string) map[string]struct{} {
	if len(sections) == 0 {
		sections = defaultSections
	}
	out := make(map[string]struct{})
	for _, section := range sections {
		for _, part := range strings.Split(section, ",") {
			trimmed := strings.ToLower(strings.TrimSpace(part))
			if trimmed == "" {
				continue
			}
			if trimmed == SectionAll {
				for _, defaultSection := range defaultSections {
					out[defaultSection] = struct{}{}
				}
				continue
			}
			out[trimmed] = struct{}{}
		}
	}
	return out
}

func hasSection(sections map[string]struct{}, section string) bool {
	_, ok := sections[section]
	return ok
}

func sectionList(sections map[string]struct{}) []string {
	ordered := []string{SectionProfile, SectionInvestment, SectionFinancials, SectionScores, SectionDividends, SectionFacts, SectionEvents}
	out := make([]string, 0, len(sections))
	for _, section := range ordered {
		if hasSection(sections, section) {
			out = append(out, section)
		}
	}
	return out
}

func buildScores(metrics []financialmetric.Metric, snapshots []valuation.Snapshot) financialscores.Scores {
	inputMetrics := make(map[string]financialscores.Metric, len(metrics))
	for _, metric := range metrics {
		inputMetrics[metric.Metric] = financialscores.Metric{
			ValueBP:            metric.ValueBP,
			UncomputableReason: metric.UncomputableReason,
		}
	}
	var inputValuation *financialscores.Valuation
	if len(snapshots) > 0 {
		inputValuation = &financialscores.Valuation{
			PerBP:           snapshots[0].PerBP,
			PbrBP:           snapshots[0].PbrBP,
			PsrBP:           snapshots[0].PsrBP,
			DividendYieldBP: snapshots[0].DividendYieldBP,
		}
	}
	return financialscores.Calculate(financialscores.Input{Metrics: inputMetrics, Valuation: inputValuation})
}

func listValuation(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult, asOf string) ([]valuation.Snapshot, error) {
	repository, err := valuation.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListSnapshots(ctx, company, valuation.Query{AsOf: asOf})
}

func listMetrics(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult, window int, period financials.PeriodType) ([]financialmetric.Metric, error) {
	repository, err := financialmetric.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListMetrics(ctx, company, financialmetric.Query{WindowYears: window, Period: period})
}

func listStatements(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult, period financials.PeriodType) ([]financials.Statement, error) {
	repository, err := financialstatement.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListStatements(ctx, company, financialstatement.Query{Period: period})
}

func listDividends(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult, window int) ([]companyfact.Fact, error) {
	repository, err := companyfact.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListFacts(ctx, company, companyfact.Query{FactType: companyfact.FactTypeDividend, WindowYears: window})
}

func listFacts(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult, window int) ([]companyfact.Fact, error) {
	repository, err := companyfact.NewRepository(database)
	if err != nil {
		return nil, err
	}
	facts, err := repository.ListFacts(ctx, company, companyfact.Query{WindowYears: window})
	if err != nil {
		return nil, err
	}
	out := make([]companyfact.Fact, 0, len(facts))
	for _, fact := range facts {
		if fact.FactType == companyfact.FactTypeDividend {
			continue
		}
		out = append(out, fact)
	}
	return out, nil
}

func listEvents(ctx context.Context, database *storage.SQLDatabase, company companyidentity.InspectResult) ([]companyevent.Event, error) {
	repository, err := companyevent.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListEvents(ctx, company, companyevent.Query{Limit: 20})
}

func issuerInstrument(links []companyidentity.InstrumentLink) companyidentity.InstrumentLink {
	for _, link := range links {
		if link.RelationType == companyidentity.RelationTypeIssuer {
			return link
		}
	}
	if len(links) == 0 {
		return companyidentity.InstrumentLink{}
	}
	return links[0]
}

func Int64PtrString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func buildResearchProfile(summary Summary) research.StockResearchProfile {
	profile := research.StockResearchProfile{
		Sections: summary.Sections,
		Company: research.CompanyIdentity{
			CompanyID:   summary.Company.ID,
			Name:        summary.Company.Name,
			LegalName:   summary.Company.LegalName,
			CountryCode: summary.Company.CountryCode,
		},
		Missing: researchMissing(summary.Missing),
	}
	if summary.Instrument.InstrumentID != 0 || summary.Instrument.Symbol != "" {
		profile.Company.Instrument = &research.InstrumentIdentity{
			InstrumentID: summary.Instrument.InstrumentID,
			Market:       string(summary.Instrument.Market),
			SecurityType: string(summary.Instrument.SecurityType),
			Symbol:       summary.Instrument.Symbol,
			Name:         summary.Instrument.Name,
		}
	}
	if len(summary.Identifiers) > 0 {
		profile.Company.Identifiers = researchIdentifiers(summary.Identifiers)
	}
	if len(summary.Metrics) > 0 {
		profile.FinancialProfile = &research.FinancialProfile{Metrics: researchMetrics(summary.Metrics)}
	}
	if len(summary.Valuation) > 0 {
		profile.ValuationProfile = &research.ValuationProfile{Snapshots: researchValuations(summary.Valuation)}
	}
	if len(summary.Dividends) > 0 {
		profile.CapitalPolicyProfile = &research.CapitalPolicyProfile{Dividends: researchFacts(summary.Dividends, "company_fact_v1")}
	}
	if len(summary.Facts) > 0 {
		profile.GovernanceProfile = &research.GovernanceProfile{Facts: researchFacts(summary.Facts, "company_fact_v1")}
	}
	if len(summary.Events) > 0 {
		profile.DisclosureEventTimeline = &research.DisclosureEventTimeline{Events: researchEvents(summary.Events, "company_event_v1")}
	}
	candidate := researchScreenCandidate(summary)
	if candidate != nil {
		profile.ScreenCandidate = candidate
	}
	return profile
}

func researchIdentifiers(identifiers []companyidentity.Identifier) []research.CompanyIdentifier {
	out := make([]research.CompanyIdentifier, 0, len(identifiers))
	for _, identifier := range identifiers {
		out = append(out, research.CompanyIdentifier{
			Type:       identifier.IdentifierType,
			Value:      identifier.IdentifierValue,
			Primary:    identifier.Primary,
			Confidence: identifier.Confidence,
			Source: research.SourceRef{
				Provider:  string(identifier.Provider),
				Group:     string(identifier.Group),
				Operation: string(identifier.Operation),
			},
		})
	}
	return out
}

func researchMetrics(metrics []financialmetric.Metric) []research.FinancialMetric {
	out := make([]research.FinancialMetric, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, research.FinancialMetric{
			Metric:             metric.Metric,
			FiscalYear:         metric.FiscalYear,
			FiscalPeriod:       string(metric.FiscalPeriod),
			AsOfDate:           metric.AsOfDate,
			ValueDecimal:       metric.ValueDecimal,
			ValueBP:            metric.ValueBP,
			ValueMinor:         metric.ValueMinor,
			FormulaVersion:     metric.FormulaVersion,
			UncomputableReason: metric.UncomputableReason,
		})
	}
	return out
}

func researchValuations(snapshots []valuation.Snapshot) []research.ValuationSnapshot {
	out := make([]research.ValuationSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, research.ValuationSnapshot{
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
			Uncomputable:        snapshot.Uncomputable,
		})
	}
	return out
}

func researchFacts(facts []companyfact.Fact, sourceTable string) []research.CompanyFact {
	out := make([]research.CompanyFact, 0, len(facts))
	for _, fact := range facts {
		out = append(out, research.CompanyFact{
			FactType:     fact.FactType,
			FiscalYear:   fact.FiscalYear,
			FactDate:     fact.FactDate,
			Key:          fact.Key,
			ValueText:    fact.ValueText,
			ValueNumber:  fact.ValueNumber,
			CurrencyCode: fact.CurrencyCode,
			Source: research.SourceRef{
				Provider:    string(fact.Provider),
				Group:       string(fact.Group),
				Operation:   string(fact.Operation),
				ReportCode:  fact.ReportCode,
				ReceiptNo:   fact.RceptNo,
				SourceTable: sourceTable,
			},
		})
	}
	return out
}

func researchEvents(events []companyevent.Event, sourceTable string) []research.DisclosureEvent {
	out := make([]research.DisclosureEvent, 0, len(events))
	for _, event := range events {
		out = append(out, research.DisclosureEvent{
			EventType:   event.EventType,
			EventDate:   event.EventDate,
			RceptDt:     event.RceptDt,
			Title:       event.Title,
			AmountMinor: event.AmountMinor,
			ValueText:   event.ValueText,
			Source: research.SourceRef{
				Provider:    string(event.Provider),
				Group:       string(event.Group),
				Operation:   string(event.Operation),
				ReceiptNo:   event.RceptNo,
				SourceTable: sourceTable,
			},
		})
	}
	return out
}

func researchMissing(missing []MissingSection) []research.MissingSection {
	out := make([]research.MissingSection, 0, len(missing))
	for _, section := range missing {
		out = append(out, research.MissingSection{Section: section.Section, Reason: section.Reason})
	}
	return out
}

func researchScreenCandidate(summary Summary) *research.ScreenCandidate {
	symbol := summary.Instrument.Symbol
	if symbol == "" && len(summary.Metrics) == 0 && len(summary.Valuation) == 0 && len(summary.Facts) == 0 && len(summary.Events) == 0 {
		return nil
	}
	candidate := research.ScreenCandidate{Symbol: symbol}
	if len(summary.Metrics) > 0 {
		candidate.Metrics = make(map[string]research.FinancialMetric, len(summary.Metrics))
		for _, metric := range researchMetrics(summary.Metrics) {
			candidate.Metrics[metric.Metric] = metric
		}
	}
	if len(summary.Valuation) > 0 {
		valuation := researchValuations(summary.Valuation)[0]
		candidate.Valuation = &valuation
	}
	if len(summary.Facts) > 0 {
		facts := researchFacts(summary.Facts, "company_fact_v1")
		candidate.Facts = make(map[string]research.CompanyFact, len(facts))
		for _, fact := range facts {
			candidate.Facts[researchFactKey(fact)] = fact
		}
	}
	if len(summary.Events) > 0 {
		candidate.Events = researchEvents(summary.Events, "company_event_v1")
	}
	return &candidate
}

func researchFactKey(fact research.CompanyFact) string {
	factType := strings.TrimSpace(fact.FactType)
	key := strings.TrimSpace(fact.Key)
	if key == "" || key == factType {
		return factType
	}
	return factType + "." + key
}
