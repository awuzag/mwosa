package stocksummary

import (
	"context"

	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage/companyevent"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialmetric"
	"github.com/awuzag/mwosa/storage/financialstatement"
	"github.com/awuzag/mwosa/storage/valuation"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoRepository struct {
	identity   companyidentity.MongoRepository
	valuation  valuation.MongoRepository
	metrics    financialmetric.MongoRepository
	statements financialstatement.MongoRepository
	facts      companyfact.MongoRepository
	events     companyevent.MongoRepository
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("stock_summary_repository").New("mongodb database is nil")
	}
	identityRepository, err := companyidentity.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	valuationRepository, err := valuation.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	metricRepository, err := financialmetric.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	statementRepository, err := financialstatement.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	factRepository, err := companyfact.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	eventRepository, err := companyevent.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("stock_summary_repository").Wrap(err)
	}
	return MongoRepository{
		identity:   identityRepository,
		valuation:  valuationRepository,
		metrics:    metricRepository,
		statements: statementRepository,
		facts:      factRepository,
		events:     eventRepository,
	}, nil
}

func (r MongoRepository) Inspect(ctx context.Context, query string, options Query) (Summary, error) {
	errb := oops.In("stock_summary_repository").With("backend", "mongodb", "query", query)
	company, err := r.identity.Inspect(ctx, query)
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
		snapshots, err = r.valuation.ListSnapshots(ctx, company, valuation.Query{AsOf: options.AsOf})
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
		metrics, err = r.metrics.ListMetrics(ctx, company, financialmetric.Query{WindowYears: options.Window, Period: options.Period})
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		if hasSection(sections, SectionFinancials) {
			out.Metrics = metrics
			if len(metrics) == 0 {
				out.Missing = append(out.Missing, MissingSection{Section: SectionFinancials, Reason: "stored financial metrics not found"})
			}
			statements, err = r.statements.ListStatements(ctx, company, financialstatement.Query{Period: options.Period})
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
		dividends, err := r.facts.ListFacts(ctx, company, companyfact.Query{FactType: companyfact.FactTypeDividend, WindowYears: options.Window})
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
		out.Dividends = dividends
		if len(dividends) == 0 {
			out.Missing = append(out.Missing, MissingSection{Section: SectionDividends, Reason: "stored dividend facts not found"})
		}
	}
	if hasSection(sections, SectionFacts) {
		facts, err := r.listFacts(ctx, company, options.Window)
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
		reportFacts, err = r.listFacts(ctx, company, options.Window)
		if err != nil {
			return Summary{}, errb.Wrap(err)
		}
	}
	if hasSection(sections, SectionEvents) {
		events, err := r.events.ListEvents(ctx, company, companyevent.Query{Limit: 20})
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

func (r MongoRepository) listFacts(ctx context.Context, company companyidentity.InspectResult, window int) ([]companyfact.Fact, error) {
	facts, err := r.facts.ListFacts(ctx, company, companyfact.Query{WindowYears: window})
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
