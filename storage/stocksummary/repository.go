package stocksummary

import (
	"context"
	"strconv"
	"strings"

	"github.com/ev3rlit/mwosa/packages/financialscores"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/companyevent"
	"github.com/ev3rlit/mwosa/storage/companyfact"
	"github.com/ev3rlit/mwosa/storage/companyidentity"
	"github.com/ev3rlit/mwosa/storage/financialmetric"
	"github.com/ev3rlit/mwosa/storage/valuation"
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

type Repository struct {
	database *storage.Database
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
}

type MissingSection struct {
	Section string `json:"section" csv:"section"`
	Reason  string `json:"reason" csv:"reason"`
}

func NewRepository(database *storage.Database) (Repository, error) {
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
	return out, nil
}

func normalizeSections(sections []string) map[string]struct{} {
	if len(sections) == 0 {
		sections = []string{SectionProfile, SectionInvestment, SectionFinancials, SectionScores, SectionDividends, SectionFacts, SectionEvents}
	}
	out := make(map[string]struct{})
	for _, section := range sections {
		for _, part := range strings.Split(section, ",") {
			trimmed := strings.ToLower(strings.TrimSpace(part))
			if trimmed == "" {
				continue
			}
			if trimmed == SectionAll {
				out[SectionProfile] = struct{}{}
				out[SectionInvestment] = struct{}{}
				out[SectionFinancials] = struct{}{}
				out[SectionScores] = struct{}{}
				out[SectionDividends] = struct{}{}
				out[SectionFacts] = struct{}{}
				out[SectionEvents] = struct{}{}
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

func listValuation(ctx context.Context, database *storage.Database, company companyidentity.InspectResult, asOf string) ([]valuation.Snapshot, error) {
	repository, err := valuation.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListSnapshots(ctx, company, valuation.Query{AsOf: asOf})
}

func listMetrics(ctx context.Context, database *storage.Database, company companyidentity.InspectResult, window int, period financials.PeriodType) ([]financialmetric.Metric, error) {
	repository, err := financialmetric.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListMetrics(ctx, company, financialmetric.Query{WindowYears: window, Period: period})
}

func listDividends(ctx context.Context, database *storage.Database, company companyidentity.InspectResult, window int) ([]companyfact.Fact, error) {
	repository, err := companyfact.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return repository.ListFacts(ctx, company, companyfact.Query{FactType: companyfact.FactTypeDividend, WindowYears: window})
}

func listFacts(ctx context.Context, database *storage.Database, company companyidentity.InspectResult, window int) ([]companyfact.Fact, error) {
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

func listEvents(ctx context.Context, database *storage.Database, company companyidentity.InspectResult) ([]companyevent.Event, error) {
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
