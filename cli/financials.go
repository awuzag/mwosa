package cli

import (
	"strconv"
	"strings"

	"github.com/awuzag/mwosa/app/handler"
	metricscore "github.com/awuzag/mwosa/packages/financialmetrics"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	opendartprovider "github.com/awuzag/mwosa/providers/opendart"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialmetric"
	"github.com/awuzag/mwosa/storage/financialstatement"
	"github.com/awuzag/mwosa/storage/valuation"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type financialsFlags struct {
	SecurityType string
	FiscalYear   string
	Period       string
	Statement    string
	Limit        int
	From         string
	To           string
	Window       string
	AsOf         string
	FactType     string
}

func registerFinancialsCommands(roots commandRoots, opts *Options) {
	getFinancials := newGetFinancialsCommand(opts)
	getFinancials.AddCommand(newGetFinancialStatementsCommand(opts))
	getFinancials.AddCommand(newGetFinancialMetricsCommand(opts))
	getFinancials.AddCommand(newGetFinancialValuationCommand(opts))
	getFinancials.AddCommand(newGetFinancialDividendsCommand(opts))
	getFinancials.AddCommand(newGetFinancialHealthCommand(opts))
	getFinancials.AddCommand(newGetFinancialFactsCommand(opts))
	roots.Get.AddCommand(getFinancials)
	roots.Sync.AddCommand(newSyncFinancialsCommand(opts))
	roots.Calc.AddCommand(newCalcFinancialsCommand(opts))
}

func newGetFinancialsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		SecurityType: string(provider.SecurityTypeStock),
		Period:       string(financials.PeriodTypeAnnual),
	}
	cmd := &cobra.Command{
		Use:   "financials <company>",
		Short: "Fetch provider-backed financial statements by company name or KRX code",
		Long: `Fetch provider-backed financial statements by company name or KRX code.

With --provider opendart, <company> may be an OpenDART corp_code or a listed-company
stock_code. stock_code is resolved to corp_code before OpenDART financial API calls;
corp_code and stock_code remain separate fields in output extensions.`,
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Financials.Get(cmd.Context(), handler.GetFinancialsRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
				FiscalYear:     flags.FiscalYear,
				Period:         financials.PeriodType(flags.Period),
				Statement:      financials.StatementType(flags.Statement),
				Limit:          flags.Limit,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn, elw")
	cmd.Flags().StringVar(&flags.FiscalYear, "year", flags.FiscalYear, "fiscal year, for example 2025")
	cmd.Flags().StringVar(&flags.Period, "period", flags.Period, "financial period: annual, quarter")
	cmd.Flags().StringVar(&flags.Statement, "statement", flags.Statement, "statement type: summary, income_statement, balance_sheet, cash_flow; empty fetches all")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of statement rows to fetch")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	mustRegisterFlagCompletion(cmd, "period", completeFinancialPeriods)
	mustRegisterFlagCompletion(cmd, "statement", completeFinancialStatementTypes)
	return cmd
}

func newGetFinancialStatementsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		SecurityType: string(provider.SecurityTypeStock),
		Period:       string(financials.PeriodTypeAnnual),
	}
	cmd := &cobra.Command{
		Use:   "statements <company>",
		Short: "Read stored canonical financial statements",
		Long: strings.TrimSpace(`Read stored canonical financial statements.

This reads financial_statement_v1 and financial_line_item_v1 from local SQLite.
The legacy shortcut get financials <company> still fetches provider-backed data
through the router.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			statementRepository, err := financialstatement.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			statements, err := statementRepository.ListStatements(cmd.Context(), company, financialstatement.Query{
				FiscalYear: flags.FiscalYear,
				Period:     financials.PeriodType(flags.Period),
				Statement:  financials.StatementType(flags.Statement),
				Limit:      flags.Limit,
			})
			if err != nil {
				return nil, err
			}
			if len(statements) == 0 {
				return nil, oops.In("cli").With("company", args[0], "year", flags.FiscalYear).New("stored financial statements not found")
			}
			return handler.FinancialStatementsOutput(statements), nil
		}),
	}
	addFinancialStatementFlags(cmd, &flags)
	return cmd
}

func newSyncFinancialsCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "financials",
		Short: "Sync provider-backed financial resources",
	}
	cmd.AddCommand(newSyncFinancialStatementsCommand(opts))
	cmd.AddCommand(newSyncFinancialDividendsCommand(opts))
	cmd.AddCommand(newSyncFinancialFactsCommand(opts))
	return cmd
}

func newCalcFinancialsCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "financials",
		Short: "Calculate financial derived data",
	}
	cmd.AddCommand(newCalcFinancialMetricsCommand(opts))
	cmd.AddCommand(newCalcFinancialValuationCommand(opts))
	return cmd
}

type financialStatementsSyncOutput struct {
	Company    companyidentity.Company         `json:"company"`
	Statements financialstatement.UpsertResult `json:"statements"`
}

func newSyncFinancialStatementsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		SecurityType: string(provider.SecurityTypeStock),
		Period:       string(financials.PeriodTypeAnnual),
	}
	cmd := &cobra.Command{
		Use:   "statements <company>",
		Short: "Fetch provider-backed financial statements and store canonical rows",
		Long: strings.TrimSpace(`Fetch provider-backed financial statements and store canonical rows.

The company must already exist in the canonical company identity graph. For
OpenDART, run sync companies --provider opendart first so corp_code and stock_code
are available as identifiers.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			years, err := requestedFiscalYears(flags)
			if err != nil {
				return nil, err
			}
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			statementRepository, err := financialstatement.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			allStatements := make([]financials.Statement, 0)
			for _, year := range years {
				output, err := runtime.Handlers.Financials.Get(cmd.Context(), handler.GetFinancialsRequest{
					ProviderID:     provider.ProviderID(opts.Provider),
					PreferProvider: provider.ProviderID(opts.PreferProvider),
					Market:         provider.Market(opts.Market),
					SecurityType:   provider.SecurityType(flags.SecurityType),
					Symbol:         args[0],
					FiscalYear:     year,
					Period:         financials.PeriodType(flags.Period),
					Statement:      financials.StatementType(flags.Statement),
					Limit:          flags.Limit,
				})
				if err != nil {
					return nil, err
				}
				allStatements = append(allStatements, []financials.Statement(output)...)
			}
			upsert, err := statementRepository.UpsertStatements(cmd.Context(), company, allStatements)
			if err != nil {
				return nil, err
			}
			return financialStatementsSyncOutput{Company: company.Company, Statements: upsert}, nil
		}),
	}
	addFinancialStatementFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "first fiscal year to sync, for example 2023")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "last fiscal year to sync, for example 2025")
	return cmd
}

func newSyncFinancialDividendsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{}
	cmd := &cobra.Command{
		Use:   "dividends <company>",
		Short: "Fetch OpenDART dividend facts and store canonical company facts",
		Long: strings.TrimSpace(`Fetch OpenDART dividend matters and store canonical company facts.

This writes company_fact_v1 rows with fact_type=dividend. The company must already
exist in the canonical company identity graph so the OpenDART corp_code can be
read as an identifier rather than treated as the canonical key.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			years, err := requestedFiscalYears(flags)
			if err != nil {
				return nil, err
			}
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "sync financials dividends"); err != nil {
				return nil, err
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			runtime, database, err := newSQLAppRuntime(opts, "financials")
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)
			companyRepository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			corpCode, err := dartCorpCode(company)
			if err != nil {
				return nil, err
			}
			factRepository, err := companyfact.NewRepository(database)
			if err != nil {
				return nil, err
			}
			allFacts := make([]companyfact.FactInput, 0)
			var source opendartprovider.CompanyFactResult
			for _, year := range years {
				source, err = p.FetchDividendFacts(cmd.Context(), opendartprovider.DividendFactRequest{
					CorpCode:   corpCode,
					FiscalYear: year,
					ReportCode: "11011",
				})
				if err != nil {
					return nil, err
				}
				allFacts = append(allFacts, openDARTFactsToStorage(source.Facts)...)
			}
			upsert, err := factRepository.UpsertFacts(cmd.Context(), company, allFacts)
			if err != nil {
				return nil, err
			}
			source.Facts = nil
			source.TotalCount = len(allFacts)
			return financialDividendsSyncOutput{Company: company.Company, Facts: upsert, Source: source}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.FiscalYear, "year", flags.FiscalYear, "fiscal year, for example 2025")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "first fiscal year to sync, for example 2023")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "last fiscal year to sync, for example 2025")
	return cmd
}

func newSyncFinancialFactsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{}
	cmd := &cobra.Command{
		Use:   "facts <company>",
		Short: "Fetch OpenDART periodic report facts and store canonical company facts",
		Long: strings.TrimSpace(`Fetch OpenDART periodic report facts and store canonical company facts.

With --provider opendart, this currently canonicalizes dividends, treasury stock,
major shareholders, major shareholder changes, employee status, and audit opinion
rows into company_fact_v1. The company must already exist in the canonical company
identity graph so OpenDART corp_code is used as a provider identifier.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			years, err := requestedFiscalYears(flags)
			if err != nil {
				return nil, err
			}
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "sync financials facts"); err != nil {
				return nil, err
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			runtime, database, err := newSQLAppRuntime(opts, "financials")
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)
			companyRepository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			corpCode, err := dartCorpCode(company)
			if err != nil {
				return nil, err
			}
			factRepository, err := companyfact.NewRepository(database)
			if err != nil {
				return nil, err
			}
			allFacts := make([]companyfact.FactInput, 0)
			var source opendartprovider.CompanyFactBatchResult
			for _, year := range years {
				source, err = p.FetchPeriodicFacts(cmd.Context(), opendartprovider.PeriodicFactRequest{
					CorpCode:   corpCode,
					FiscalYear: year,
					ReportCode: "11011",
				})
				if err != nil {
					return nil, err
				}
				allFacts = append(allFacts, openDARTFactsToStorage(source.Facts)...)
			}
			upsert, err := factRepository.UpsertFacts(cmd.Context(), company, allFacts)
			if err != nil {
				return nil, err
			}
			source.Facts = nil
			source.TotalCount = len(allFacts)
			return financialFactsSyncOutput{Company: company.Company, Facts: upsert, Source: source}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.FiscalYear, "year", flags.FiscalYear, "fiscal year, for example 2025")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "first fiscal year to sync, for example 2023")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "last fiscal year to sync, for example 2025")
	return cmd
}

type financialMetricsOutput struct {
	Metrics []financialmetric.Metric
}

type financialValuationOutput struct {
	Snapshots []valuation.Snapshot
}

type financialFactsOutput struct {
	Facts []companyfact.Fact
}

type financialHealthOutput struct {
	Items   []financialHealthItem    `json:"items"`
	Missing []financialHealthMissing `json:"missing,omitempty"`
}

type financialHealthItem struct {
	Category            string `json:"category" csv:"category"`
	Metric              string `json:"metric,omitempty" csv:"metric"`
	FactType            string `json:"fact_type,omitempty" csv:"fact_type"`
	FiscalYear          string `json:"fiscal_year,omitempty" csv:"fiscal_year"`
	FiscalPeriod        string `json:"fiscal_period,omitempty" csv:"fiscal_period"`
	AsOfDate            string `json:"as_of_date,omitempty" csv:"as_of_date"`
	ValueDecimal        string `json:"value_decimal,omitempty" csv:"value_decimal"`
	ValueBP             *int64 `json:"value_bp,omitempty" csv:"value_bp"`
	ValueMinor          *int64 `json:"value_minor,omitempty" csv:"value_minor"`
	ValueText           string `json:"value_text,omitempty" csv:"value_text"`
	ValueNumber         string `json:"value_number,omitempty" csv:"value_number"`
	CurrencyCode        string `json:"currency_code,omitempty" csv:"currency_code"`
	Status              string `json:"status" csv:"status"`
	UncomputableReason  string `json:"uncomputable_reason,omitempty" csv:"uncomputable_reason"`
	FormulaVersion      string `json:"formula_version,omitempty" csv:"formula_version"`
	Source              string `json:"source" csv:"source"`
	Provider            string `json:"provider,omitempty" csv:"provider"`
	ProviderGroup       string `json:"provider_group,omitempty" csv:"provider_group"`
	Operation           string `json:"operation,omitempty" csv:"operation"`
	RceptNo             string `json:"rcept_no,omitempty" csv:"rcept_no"`
	ProviderFiscalLabel string `json:"provider_fiscal_label,omitempty" csv:"provider_fiscal_label"`
}

type financialHealthMissing struct {
	Category string `json:"category" csv:"category"`
	Metric   string `json:"metric,omitempty" csv:"metric"`
	FactType string `json:"fact_type,omitempty" csv:"fact_type"`
	Reason   string `json:"reason" csv:"reason"`
}

type financialDividendsSyncOutput struct {
	Company companyidentity.Company            `json:"company"`
	Facts   companyfact.UpsertResult           `json:"facts"`
	Source  opendartprovider.CompanyFactResult `json:"source"`
}

type financialFactsSyncOutput struct {
	Company companyidentity.Company                 `json:"company"`
	Facts   companyfact.UpsertResult                `json:"facts"`
	Source  opendartprovider.CompanyFactBatchResult `json:"source"`
}

func newCalcFinancialMetricsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		Period: string(financials.PeriodTypeAnnual),
		Window: "3y",
	}
	cmd := &cobra.Command{
		Use:   "metrics <company>",
		Short: "Calculate stored canonical financial metrics",
		Long: strings.TrimSpace(`Calculate stored canonical financial metrics.

This reads financial_statement_v1 and financial_line_item_v1, then writes
financial_metric_v1. Missing source accounts are stored as uncomputable metrics
with explicit reasons.`),
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

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			metricRepository, err := financialmetric.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			return metricRepository.CalculateAndUpsert(cmd.Context(), company, financialmetric.CalculateOptions{
				WindowYears: windowYears,
				Period:      financials.PeriodType(flags.Period),
			})
		}),
	}
	addFinancialMetricFlags(cmd, &flags)
	return cmd
}

func newGetFinancialMetricsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		Period: string(financials.PeriodTypeAnnual),
		Window: "3y",
	}
	cmd := &cobra.Command{
		Use:   "metrics <company>",
		Short: "Read stored canonical financial metrics",
		Args:  cobra.ExactArgs(1),
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

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			metricRepository, err := financialmetric.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			metrics, err := metricRepository.ListMetrics(cmd.Context(), company, financialmetric.Query{
				WindowYears: windowYears,
				Period:      financials.PeriodType(flags.Period),
			})
			if err != nil {
				return nil, err
			}
			if len(metrics) == 0 {
				return nil, oops.In("cli").With("company", args[0], "window", flags.Window).New("stored financial metrics not found")
			}
			return financialMetricsOutput{Metrics: metrics}, nil
		}),
	}
	addFinancialMetricFlags(cmd, &flags)
	return cmd
}

func newCalcFinancialValuationCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{AsOf: "latest"}
	cmd := &cobra.Command{
		Use:   "valuation <company>",
		Short: "Calculate stored canonical valuation snapshots",
		Long: strings.TrimSpace(`Calculate stored canonical valuation snapshots.

This combines the issuer instrument's daily_bar_v2 price/market cap with stored
financial statement line items, then writes valuation_snapshot_v1.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			repository, err := valuation.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			snapshot, err := repository.CalculateAndUpsert(cmd.Context(), company, valuation.CalculateOptions{AsOf: flags.AsOf})
			if err != nil {
				return nil, err
			}
			return financialValuationOutput{Snapshots: []valuation.Snapshot{snapshot}}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "valuation date, YYYY-MM-DD or latest")
	return cmd
}

func newGetFinancialValuationCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{AsOf: "latest"}
	cmd := &cobra.Command{
		Use:   "valuation <company>",
		Short: "Read stored canonical valuation snapshots",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			repository, err := valuation.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			snapshots, err := repository.ListSnapshots(cmd.Context(), company, valuation.Query{AsOf: flags.AsOf})
			if err != nil {
				return nil, err
			}
			if len(snapshots) == 0 {
				return nil, oops.In("cli").With("company", args[0], "as_of", flags.AsOf).New("stored financial valuation snapshots not found")
			}
			return financialValuationOutput{Snapshots: snapshots}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "valuation date, YYYY-MM-DD or latest")
	return cmd
}

func newGetFinancialHealthCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{
		Period: string(financials.PeriodTypeAnnual),
		Window: "3y",
	}
	cmd := &cobra.Command{
		Use:   "health <company>",
		Short: "Read stored financial health metrics and audit facts",
		Long: strings.TrimSpace(`Read stored financial health metrics and audit facts.

This command reads provider-neutral financial_metric_v1 rows plus audit opinion
facts from company_fact_v1. Run calc financials metrics and sync financials facts
first to populate the underlying canonical data.`),
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

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			metricRepository, err := financialmetric.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			metrics, err := metricRepository.ListMetrics(cmd.Context(), company, financialmetric.Query{
				WindowYears: windowYears,
				Period:      financials.PeriodType(flags.Period),
			})
			if err != nil {
				return nil, err
			}
			factRepository, err := companyfact.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			auditFacts, err := factRepository.ListFacts(cmd.Context(), company, companyfact.Query{
				FactType:    "audit_opinion",
				WindowYears: windowYears,
			})
			if err != nil {
				return nil, err
			}
			output := buildFinancialHealthOutput(metrics, auditFacts)
			if len(output.Items) == 0 {
				return nil, oops.In("cli").With("company", args[0], "window", flags.Window, "period", flags.Period).New("stored financial health inputs not found")
			}
			return output, nil
		}),
	}
	addFinancialMetricFlags(cmd, &flags)
	return cmd
}

func newGetFinancialDividendsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{Window: "3y"}
	cmd := &cobra.Command{
		Use:   "dividends <company>",
		Short: "Read stored canonical dividend facts",
		Args:  cobra.ExactArgs(1),
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

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			repository, err := companyfact.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			facts, err := repository.ListFacts(cmd.Context(), company, companyfact.Query{
				FactType:    companyfact.FactTypeDividend,
				WindowYears: windowYears,
			})
			if err != nil {
				return nil, err
			}
			if len(facts) == 0 {
				return nil, oops.In("cli").With("company", args[0], "window", flags.Window).New("stored financial dividend facts not found")
			}
			return financialFactsOutput{Facts: facts}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.Window, "window", flags.Window, "dividend window, for example 3y")
	return cmd
}

func newGetFinancialFactsCommand(opts *Options) *cobra.Command {
	flags := financialsFlags{FactType: ""}
	cmd := &cobra.Command{
		Use:   "facts <company>",
		Short: "Read stored canonical company financial facts",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			companyRepository, err := companyidentity.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			repository, err := companyfact.NewRepository(runtime.Storage.SQLDatabase)
			if err != nil {
				return nil, err
			}
			facts, err := repository.ListFacts(cmd.Context(), company, companyfact.Query{
				FactType:   flags.FactType,
				FiscalYear: flags.FiscalYear,
				From:       flags.From,
				To:         flags.To,
				Limit:      flags.Limit,
			})
			if err != nil {
				return nil, err
			}
			if len(facts) == 0 {
				return nil, oops.In("cli").With("company", args[0], "year", flags.FiscalYear, "fact_type", flags.FactType).New("stored financial facts not found")
			}
			return financialFactsOutput{Facts: facts}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.FiscalYear, "year", flags.FiscalYear, "fiscal year, for example 2025")
	cmd.Flags().StringVar(&flags.FactType, "fact-type", flags.FactType, "fact type filter, for example dividend")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "fact date lower bound, YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "fact date upper bound, YYYY-MM-DD")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum fact rows to return")
	return cmd
}

func addFinancialStatementFlags(cmd *cobra.Command, flags *financialsFlags) {
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn, elw")
	cmd.Flags().StringVar(&flags.FiscalYear, "year", flags.FiscalYear, "fiscal year, for example 2025")
	cmd.Flags().StringVar(&flags.Period, "period", flags.Period, "financial period: annual, quarter")
	cmd.Flags().StringVar(&flags.Statement, "statement", flags.Statement, "statement type: summary, income_statement, balance_sheet, cash_flow; empty fetches all")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of statement rows to fetch")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	mustRegisterFlagCompletion(cmd, "period", completeFinancialPeriods)
	mustRegisterFlagCompletion(cmd, "statement", completeFinancialStatementTypes)
}

func addFinancialMetricFlags(cmd *cobra.Command, flags *financialsFlags) {
	cmd.Flags().StringVar(&flags.Period, "period", flags.Period, "financial period: annual, quarter")
	cmd.Flags().StringVar(&flags.Window, "window", flags.Window, "metric window, for example 3y")
	mustRegisterFlagCompletion(cmd, "period", completeFinancialPeriods)
}

func requestedFiscalYears(flags financialsFlags) ([]string, error) {
	if strings.TrimSpace(flags.FiscalYear) != "" {
		return []string{strings.TrimSpace(flags.FiscalYear)}, nil
	}
	if strings.TrimSpace(flags.From) == "" && strings.TrimSpace(flags.To) == "" {
		return nil, oops.In("cli").New("sync financials requires --year or --from/--to")
	}
	from, err := strconv.Atoi(strings.TrimSpace(flags.From))
	if err != nil {
		return nil, oops.In("cli").With("from", flags.From).Wrapf(err, "parse financials --from year")
	}
	to, err := strconv.Atoi(strings.TrimSpace(flags.To))
	if err != nil {
		return nil, oops.In("cli").With("to", flags.To).Wrapf(err, "parse financials --to year")
	}
	if from > to {
		return nil, oops.In("cli").With("from", flags.From, "to", flags.To).New("financials --from must be before or equal to --to")
	}
	years := make([]string, 0, to-from+1)
	for year := from; year <= to; year++ {
		years = append(years, strconv.Itoa(year))
	}
	return years, nil
}

func parseWindowYears(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 3, nil
	}
	trimmed = strings.TrimSuffix(trimmed, "Y")
	trimmed = strings.TrimSuffix(trimmed, "y")
	years, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, oops.In("cli").With("window", value).Wrapf(err, "parse financials --window")
	}
	if years <= 0 {
		return 0, oops.In("cli").With("window", value).New("financials --window must be positive")
	}
	return years, nil
}

func (o financialMetricsOutput) JSONValue() any {
	return o.Metrics
}

func (o financialMetricsOutput) NDJSONRows() any {
	return o.Metrics
}

func (o financialMetricsOutput) CSVRows() any {
	return o.Metrics
}

func (o financialMetricsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Metrics))
	for _, metric := range o.Metrics {
		value := metric.ValueDecimal
		if value == "" && metric.ValueBP != nil {
			value = strconv.FormatInt(*metric.ValueBP, 10)
		}
		if value == "" && metric.ValueMinor != nil {
			value = strconv.FormatInt(*metric.ValueMinor, 10)
		}
		rows = append(rows, []string{
			metric.FiscalYear,
			string(metric.FiscalPeriod),
			metric.Metric,
			value,
			int64PointerString(metric.ValueBP),
			metric.UncomputableReason,
		})
	}
	return []string{"year", "period", "metric", "value_decimal", "value_bp", "uncomputable_reason"}, rows
}

func (o financialValuationOutput) JSONValue() any {
	return o.Snapshots
}

func (o financialValuationOutput) NDJSONRows() any {
	return o.Snapshots
}

func (o financialValuationOutput) CSVRows() any {
	return o.Snapshots
}

func (o financialValuationOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Snapshots))
	for _, snapshot := range o.Snapshots {
		rows = append(rows, []string{
			snapshot.AsOfDate,
			snapshot.SourcePriceDate,
			int64PointerString(snapshot.MarketCapMinor),
			int64PointerString(snapshot.ClosePriceMinor),
			int64PointerString(snapshot.SharesOutstanding),
			int64PointerString(snapshot.PerBP),
			int64PointerString(snapshot.PbrBP),
			int64PointerString(snapshot.PsrBP),
			int64PointerString(snapshot.EpsMinor),
			int64PointerString(snapshot.BpsMinor),
			int64PointerString(snapshot.DividendYieldBP),
		})
	}
	return []string{"as_of", "price_date", "market_cap", "close", "shares", "per_bp", "pbr_bp", "psr_bp", "eps", "bps", "dividend_yield_bp"}, rows
}

func int64PointerString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func (o financialFactsOutput) JSONValue() any {
	return o.Facts
}

func (o financialFactsOutput) NDJSONRows() any {
	return o.Facts
}

func (o financialFactsOutput) CSVRows() any {
	return o.Facts
}

func (o financialFactsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Facts))
	for _, fact := range o.Facts {
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
	return []string{"year", "fact_type", "key", "value_text", "value_number", "fact_date", "rcept_no", "source"}, rows
}

func (o financialHealthOutput) JSONValue() any {
	return o
}

func (o financialHealthOutput) NDJSONRows() any {
	return o.Items
}

func (o financialHealthOutput) CSVRows() any {
	return o.Items
}

func (o financialHealthOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Items))
	for _, item := range o.Items {
		value := item.ValueDecimal
		if value == "" && item.ValueBP != nil {
			value = strconv.FormatInt(*item.ValueBP, 10)
		}
		if value == "" && item.ValueMinor != nil {
			value = strconv.FormatInt(*item.ValueMinor, 10)
		}
		if value == "" {
			value = item.ValueText
		}
		if value == "" {
			value = item.ValueNumber
		}
		name := item.Metric
		if name == "" {
			name = item.FactType
		}
		rows = append(rows, []string{
			item.Category,
			name,
			item.FiscalYear,
			item.FiscalPeriod,
			value,
			int64PointerString(item.ValueBP),
			item.Status,
			item.UncomputableReason,
			item.Source,
		})
	}
	return []string{"category", "item", "year", "period", "value", "value_bp", "status", "reason", "source"}, rows
}

func (o financialDividendsSyncOutput) TableRows() ([]string, [][]string) {
	return []string{"company_id", "company", "facts_written", "provider", "operation"}, [][]string{{
		strconv.FormatInt(o.Company.ID, 10),
		o.Company.Name,
		strconv.Itoa(o.Facts.FactsWritten),
		string(o.Source.Provider),
		string(o.Source.Operation),
	}}
}

func (o financialFactsSyncOutput) TableRows() ([]string, [][]string) {
	return []string{"company_id", "company", "facts_written", "provider", "sources"}, [][]string{{
		strconv.FormatInt(o.Company.ID, 10),
		o.Company.Name,
		strconv.Itoa(o.Facts.FactsWritten),
		string(o.Source.Provider),
		strconv.Itoa(len(o.Source.Sources)),
	}}
}

func dartCorpCode(company companyidentity.InspectResult) (string, error) {
	for _, identifier := range company.Identifiers {
		if identifier.IdentifierType == companyidentity.IdentifierTypeDARTCorpCode && strings.TrimSpace(identifier.IdentifierValue) != "" {
			return strings.TrimSpace(identifier.IdentifierValue), nil
		}
	}
	return "", oops.In("cli").With("company_id", company.Company.ID).New("canonical company is missing dart_corp_code identifier")
}

func buildFinancialHealthOutput(metrics []financialmetric.Metric, auditFacts []companyfact.Fact) financialHealthOutput {
	seenMetrics := map[string]struct{}{}
	items := make([]financialHealthItem, 0, len(metrics)+len(auditFacts))
	for _, metric := range metrics {
		category, ok := financialHealthMetricCategory(metric.Metric)
		if !ok {
			continue
		}
		seenMetrics[metric.Metric] = struct{}{}
		status := "available"
		if strings.TrimSpace(metric.UncomputableReason) != "" {
			status = "uncomputable"
		}
		items = append(items, financialHealthItem{
			Category:           category,
			Metric:             metric.Metric,
			FiscalYear:         metric.FiscalYear,
			FiscalPeriod:       string(metric.FiscalPeriod),
			AsOfDate:           metric.AsOfDate,
			ValueDecimal:       metric.ValueDecimal,
			ValueBP:            metric.ValueBP,
			ValueMinor:         metric.ValueMinor,
			Status:             status,
			UncomputableReason: metric.UncomputableReason,
			FormulaVersion:     metric.FormulaVersion,
			Source:             "financial_metric_v1",
		})
	}
	for _, fact := range auditFacts {
		items = append(items, financialHealthItem{
			Category:            "audit",
			FactType:            fact.FactType,
			FiscalYear:          fact.FiscalYear,
			AsOfDate:            fact.FactDate,
			ValueText:           fact.ValueText,
			ValueNumber:         fact.ValueNumber,
			CurrencyCode:        fact.CurrencyCode,
			Status:              "available",
			Source:              "company_fact_v1",
			Provider:            string(fact.Provider),
			ProviderGroup:       string(fact.Group),
			Operation:           string(fact.Operation),
			RceptNo:             fact.RceptNo,
			ProviderFiscalLabel: fact.Key,
		})
	}
	missing := make([]financialHealthMissing, 0)
	for _, metric := range financialHealthMetrics() {
		if _, ok := seenMetrics[metric]; ok {
			continue
		}
		category, _ := financialHealthMetricCategory(metric)
		missing = append(missing, financialHealthMissing{
			Category: category,
			Metric:   metric,
			Reason:   "stored financial metric not found",
		})
	}
	if len(auditFacts) == 0 {
		missing = append(missing, financialHealthMissing{
			Category: "audit",
			FactType: "audit_opinion",
			Reason:   "stored audit opinion fact not found",
		})
	}
	return financialHealthOutput{Items: items, Missing: missing}
}

func financialHealthMetrics() []string {
	return []string{
		metricscore.MetricDebtToEquity,
		metricscore.MetricCurrentRatio,
		metricscore.MetricInterestCoverage,
		metricscore.MetricROE,
		metricscore.MetricROA,
		metricscore.MetricOperatingMargin,
		metricscore.MetricNetMargin,
	}
}

func financialHealthMetricCategory(metric string) (string, bool) {
	switch metric {
	case metricscore.MetricDebtToEquity, metricscore.MetricCurrentRatio, metricscore.MetricInterestCoverage:
		return "stability", true
	case metricscore.MetricROE, metricscore.MetricROA, metricscore.MetricOperatingMargin, metricscore.MetricNetMargin:
		return "profitability", true
	default:
		return "", false
	}
}

func openDARTFactsToStorage(facts []opendartprovider.CompanyFact) []companyfact.FactInput {
	out := make([]companyfact.FactInput, 0, len(facts))
	for _, fact := range facts {
		out = append(out, companyfact.FactInput{
			Provider:                       fact.Provider,
			Group:                          fact.Group,
			Operation:                      fact.Operation,
			ProviderCompanyIdentifierType:  fact.ProviderCompanyIdentifierType,
			ProviderCompanyIdentifierValue: fact.ProviderCompanyIdentifierValue,
			FactType:                       fact.FactType,
			FiscalYear:                     fact.FiscalYear,
			ReportCode:                     fact.ReportCode,
			RceptNo:                        fact.RceptNo,
			FactDate:                       fact.FactDate,
			Key:                            fact.Key,
			ValueText:                      fact.ValueText,
			ValueNumber:                    fact.ValueNumber,
			CurrencyCode:                   fact.CurrencyCode,
			Raw:                            fact.Raw,
		})
	}
	return out
}
