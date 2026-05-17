package cli

import (
	"strconv"
	"strings"

	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/storage/stocksummary"
	"github.com/spf13/cobra"
)

type stockInspectFlags struct {
	Section string
	AsOf    string
	Window  string
	Period  string
}

type stockInspectOutput struct {
	Summary          stocksummary.Summary
	RawTableSections map[string]struct{}
}

func registerStockCommands(roots commandRoots, opts *Options) {
	roots.Inspect.AddCommand(newInspectStockCommand(opts))
}

func newInspectStockCommand(opts *Options) *cobra.Command {
	flags := stockInspectFlags{
		Section: "profile,investment,financials,scores,dividends,events",
		AsOf:    "latest",
		Window:  "3y",
		Period:  string(financials.PeriodTypeAnnual),
	}
	cmd := &cobra.Command{
		Use:   "stock <symbol-or-company>",
		Short: "Inspect a stored stock profile with financial analysis sections",
		Long: strings.TrimSpace(`Inspect a stored stock profile with financial analysis sections.

This command reads canonical local storage only. Use sync companies, sync
financials statements, calc financials metrics, calc financials valuation, and
sync financials facts or sync events to populate the underlying sections.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			windowYears, err := parseWindowYears(flags.Window)
			if err != nil {
				return nil, err
			}
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			repository, err := stocksummary.NewRepository(runtime.Storage.Database)
			if err != nil {
				return nil, err
			}
			summary, err := repository.Inspect(cmd.Context(), args[0], stocksummary.Query{
				Sections: strings.Split(flags.Section, ","),
				AsOf:     flags.AsOf,
				Window:   windowYears,
				Period:   financials.PeriodType(flags.Period),
			})
			if err != nil {
				return nil, err
			}
			return stockInspectOutput{
				Summary:          summary,
				RawTableSections: stockInspectRawTableSections(flags.Section),
			}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.Section, "section", flags.Section, "comma-separated sections: profile,investment,financials,scores,dividends,events,all; use facts explicitly for large fact rows")
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "valuation date, YYYY-MM-DD or latest")
	cmd.Flags().StringVar(&flags.Window, "window", flags.Window, "financial metric/dividend window, for example 3y")
	cmd.Flags().StringVar(&flags.Period, "period", flags.Period, "financial period: annual, quarter")
	mustRegisterFlagCompletion(cmd, "period", completeFinancialPeriods)
	return cmd
}

func (o stockInspectOutput) JSONValue() any {
	return o.Summary
}

func (o stockInspectOutput) TableBlocks() []TableBlock {
	if len(o.RawTableSections) > 0 {
		return o.rawTableBlocks()
	}
	blocks := make([]TableBlock, 0, len(o.Summary.Report.Tables))
	for _, table := range o.Summary.Report.Tables {
		blocks = append(blocks, TableBlock{
			Title:  table.Title,
			Header: table.Header,
			Rows:   table.Rows,
		})
	}
	if len(blocks) > 0 {
		return blocks
	}
	header, rows := o.TableRows()
	return []TableBlock{{Header: header, Rows: rows}}
}

func (o stockInspectOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0)
	sections := make(map[string]struct{}, len(o.Summary.Sections))
	for _, section := range o.Summary.Sections {
		sections[section] = struct{}{}
	}
	if _, ok := sections[stocksummary.SectionProfile]; ok {
		rows = append(rows,
			[]string{"profile", "company_id", strconv.FormatInt(o.Summary.Company.ID, 10), ""},
			[]string{"profile", "name", o.Summary.Company.Name, ""},
			[]string{"profile", "legal_name", o.Summary.Company.LegalName, ""},
			[]string{"profile", "country_code", o.Summary.Company.CountryCode, ""},
		)
		if o.Summary.Instrument.InstrumentID != 0 {
			rows = append(rows, []string{
				"profile",
				"instrument",
				string(o.Summary.Instrument.Market) + "/" + string(o.Summary.Instrument.SecurityType) + "/" + o.Summary.Instrument.Symbol,
				"instrument_company_link_v1",
			})
		}
		for _, identifier := range o.Summary.Identifiers {
			rows = append(rows, []string{
				"profile",
				identifier.IdentifierType,
				identifier.IdentifierValue,
				string(identifier.Provider) + "/" + string(identifier.Group) + "/" + string(identifier.Operation),
			})
		}
	}
	if _, ok := sections[stocksummary.SectionInvestment]; ok {
		for _, snapshot := range o.Summary.Valuation {
			rows = append(rows,
				[]string{"investment", "as_of", snapshot.AsOfDate, "valuation_snapshot_v1"},
				[]string{"investment", "source_price_date", snapshot.SourcePriceDate, "valuation_snapshot_v1"},
				[]string{"investment", "market_cap_minor", stocksummary.Int64PtrString(snapshot.MarketCapMinor), "valuation_snapshot_v1"},
				[]string{"investment", "close_price_minor", stocksummary.Int64PtrString(snapshot.ClosePriceMinor), "valuation_snapshot_v1"},
				[]string{"investment", "per_bp", stocksummary.Int64PtrString(snapshot.PerBP), "valuation_snapshot_v1"},
				[]string{"investment", "pbr_bp", stocksummary.Int64PtrString(snapshot.PbrBP), "valuation_snapshot_v1"},
				[]string{"investment", "psr_bp", stocksummary.Int64PtrString(snapshot.PsrBP), "valuation_snapshot_v1"},
				[]string{"investment", "eps_minor", stocksummary.Int64PtrString(snapshot.EpsMinor), "valuation_snapshot_v1"},
				[]string{"investment", "bps_minor", stocksummary.Int64PtrString(snapshot.BpsMinor), "valuation_snapshot_v1"},
				[]string{"investment", "dividend_yield_bp", stocksummary.Int64PtrString(snapshot.DividendYieldBP), "valuation_snapshot_v1"},
			)
		}
	}
	if _, ok := sections[stocksummary.SectionFinancials]; ok {
		for _, metric := range o.Summary.Metrics {
			value := metric.ValueDecimal
			if value == "" {
				value = stocksummary.Int64PtrString(metric.ValueBP)
			}
			if value == "" {
				value = stocksummary.Int64PtrString(metric.ValueMinor)
			}
			source := "financial_metric_v1"
			if metric.UncomputableReason != "" {
				source = "uncomputable: " + metric.UncomputableReason
			}
			rows = append(rows, []string{
				"financials",
				metric.Metric + ":" + metric.FiscalYear,
				value,
				source,
			})
		}
	}
	if _, ok := sections[stocksummary.SectionScores]; ok && o.Summary.Scores != nil {
		rows = appendScoreRow(rows, "valuation_score", o.Summary.Scores.ValuationScore, o.Summary.Scores.ScoreVersion)
		rows = appendScoreRow(rows, "quality_score", o.Summary.Scores.QualityScore, o.Summary.Scores.ScoreVersion)
		rows = appendScoreRow(rows, "growth_score", o.Summary.Scores.GrowthScore, o.Summary.Scores.ScoreVersion)
		for key, reason := range o.Summary.Scores.Uncomputable {
			rows = append(rows, []string{"scores", key, reason, "uncomputable"})
		}
	}
	if _, ok := sections[stocksummary.SectionDividends]; ok {
		for _, fact := range o.Summary.Dividends {
			rows = append(rows, []string{
				"dividends",
				fact.FiscalYear + ":" + fact.Key,
				firstNonEmpty(fact.ValueNumber, fact.ValueText),
				string(fact.Provider) + "/" + string(fact.Group) + "/" + string(fact.Operation),
			})
		}
	}
	if _, ok := sections[stocksummary.SectionFacts]; ok {
		for _, fact := range o.Summary.Facts {
			rows = append(rows, []string{
				"facts",
				fact.FactType + ":" + fact.FiscalYear + ":" + fact.Key,
				firstNonEmpty(fact.ValueNumber, fact.ValueText),
				string(fact.Provider) + "/" + string(fact.Group) + "/" + string(fact.Operation),
			})
		}
	}
	if _, ok := sections[stocksummary.SectionEvents]; ok {
		for _, event := range o.Summary.Events {
			rows = append(rows, []string{
				"events",
				event.EventType + ":" + event.RceptNo,
				firstNonEmpty(event.Title, event.ValueText),
				string(event.Provider) + "/" + string(event.Group) + "/" + string(event.Operation),
			})
		}
	}
	for _, missing := range o.Summary.Missing {
		rows = append(rows, []string{missing.Section, "missing", missing.Reason, ""})
	}
	return []string{"section", "key", "value", "source"}, rows
}

func (o stockInspectOutput) rawTableBlocks() []TableBlock {
	blocks := make([]TableBlock, 0, len(o.RawTableSections)+1)
	if _, ok := o.RawTableSections[stocksummary.SectionFacts]; ok {
		rows := make([][]string, 0, len(o.Summary.Facts))
		for _, fact := range o.Summary.Facts {
			rows = append(rows, []string{
				fact.FiscalYear,
				fact.FactType,
				fact.Key,
				fact.ValueText,
				fact.ValueNumber,
				fact.FactDate,
				fact.RceptNo,
				string(fact.Provider) + "/" + string(fact.Group) + "/" + string(fact.Operation),
			})
		}
		blocks = append(blocks, TableBlock{
			Title:  "facts 상세",
			Header: []string{"year", "fact_type", "key", "value_text", "value_number", "fact_date", "rcept_no", "source"},
			Rows:   rows,
		})
	}
	if _, ok := o.RawTableSections[stocksummary.SectionDividends]; ok {
		rows := make([][]string, 0, len(o.Summary.Dividends))
		for _, fact := range o.Summary.Dividends {
			rows = append(rows, []string{
				fact.FiscalYear,
				fact.Key,
				fact.ValueText,
				fact.ValueNumber,
				fact.FactDate,
				fact.RceptNo,
				string(fact.Provider) + "/" + string(fact.Group) + "/" + string(fact.Operation),
			})
		}
		blocks = append(blocks, TableBlock{
			Title:  "dividends 상세",
			Header: []string{"year", "key", "value_text", "value_number", "fact_date", "rcept_no", "source"},
			Rows:   rows,
		})
	}
	if _, ok := o.RawTableSections[stocksummary.SectionEvents]; ok {
		rows := make([][]string, 0, len(o.Summary.Events))
		for _, event := range o.Summary.Events {
			rows = append(rows, []string{
				event.EventDate,
				event.EventType,
				event.Title,
				event.ValueText,
				event.RceptNo,
				string(event.Provider) + "/" + string(event.Group) + "/" + string(event.Operation),
			})
		}
		blocks = append(blocks, TableBlock{
			Title:  "events 상세",
			Header: []string{"event_date", "event_type", "title", "value_text", "rcept_no", "source"},
			Rows:   rows,
		})
	}
	if len(o.Summary.Missing) > 0 {
		rows := make([][]string, 0, len(o.Summary.Missing))
		for _, missing := range o.Summary.Missing {
			rows = append(rows, []string{missing.Section, missing.Reason})
		}
		blocks = append(blocks, TableBlock{
			Title:  "누락",
			Header: []string{"section", "reason"},
			Rows:   rows,
		})
	}
	return blocks
}

func stockInspectRawTableSections(value string) map[string]struct{} {
	parts := strings.Split(value, ",")
	out := make(map[string]struct{})
	for _, part := range parts {
		section := strings.ToLower(strings.TrimSpace(part))
		switch section {
		case stocksummary.SectionFacts, stocksummary.SectionDividends, stocksummary.SectionEvents:
			out[section] = struct{}{}
		case "", stocksummary.SectionAll:
			return nil
		default:
			return nil
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func appendScoreRow(rows [][]string, key string, value *int, source string) [][]string {
	if value == nil {
		return rows
	}
	return append(rows, []string{"scores", key, strconv.Itoa(*value), source})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
