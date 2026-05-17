package cli

import (
	"fmt"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	opendartprovider "github.com/ev3rlit/mwosa/providers/opendart"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/opendartcompany"
	"github.com/ev3rlit/mwosa/storage/providerraw"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type opendartCompanySyncFlags struct {
	ListedOnly bool
}

type opendartCompanySearchFlags struct {
	ListedOnly bool
	Limit      int
}

type opendartFilingFlags struct {
	CorpCode   string
	From       string
	To         string
	LastReport bool
	PageNo     string
	PageCount  string
}

type opendartAPIListOutput struct {
	Services []opendartAPIOutputRow `json:"services"`
}

type opendartAPIOutputRow struct {
	Category         string `json:"category" csv:"category"`
	Group            string `json:"provider_group" csv:"group"`
	APIID            string `json:"api_id" csv:"api_id"`
	Description      string `json:"description" csv:"description"`
	CanonicalSupport string `json:"canonical_support" csv:"canonical_support"`
}

type opendartCompanySearchOutput struct {
	Companies []opendartprovider.Company `json:"companies"`
}

type opendartCompanySyncOutput struct {
	Provider  provider.ProviderID          `json:"provider"`
	Operation provider.OperationID         `json:"operation"`
	Companies opendartcompany.UpsertResult `json:"companies"`
	Snapshot  providerraw.WriteResult      `json:"raw_snapshot"`
}

type opendartFilingOutput struct {
	Result opendartprovider.FilingResult
}

func registerOpenDARTCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListProviderAPIsCommand(opts))
	roots.List.AddCommand(newListFilingsCommand(opts))
	roots.Search.AddCommand(newSearchCompaniesCommand(opts))
	roots.Sync.AddCommand(newSyncCompaniesCommand(opts))
}

func newListProviderAPIsCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider-apis <provider>",
		Short:             "List provider-native APIs known to mwosa diagnostics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runResult(opts, func(_ *cobra.Command, args []string) (any, error) {
			if provider.ProviderID(args[0]) != provider.ProviderOpenDART {
				return nil, oops.In("cli").
					With("provider", args[0]).
					New("provider API catalog is only available for opendart")
			}
			rows := []opendartAPIOutputRow{
				{
					Category:         "disclosure",
					Group:            string(provider.GroupOpenDARTDisclosure),
					APIID:            string(provider.OperationOpenDARTCorpCode),
					Description:      "OpenDART company registry: corp_code, corp_name, corp_eng_name, stock_code, modify_date",
					CanonicalSupport: "company_registry",
				},
				{
					Category:         "disclosure",
					Group:            string(provider.GroupOpenDARTDisclosure),
					APIID:            string(provider.OperationOpenDARTList),
					Description:      "OpenDART disclosure search by corp_code and date range",
					CanonicalSupport: "filings",
				},
				{
					Category:         "financial",
					Group:            string(provider.GroupOpenDARTFinancials),
					APIID:            string(provider.OperationOpenDARTSinglAcntAll),
					Description:      "OpenDART single-company full financial statements; stock_code is resolved to corp_code",
					CanonicalSupport: "financials",
				},
			}
			return opendartAPIListOutput{Services: rows}, nil
		}),
	}
}

func newSyncCompaniesCommand(opts *Options) *cobra.Command {
	flags := opendartCompanySyncFlags{}
	cmd := &cobra.Command{
		Use:   "companies",
		Short: "Sync provider-backed company registry",
		Long: strings.TrimSpace(`Sync a provider-backed company registry.

With --provider opendart, this fetches OpenDART corpCode.xml and stores corp_code,
corp_name, corp_eng_name, stock_code, and modify_date. OpenDART corp_code is not a
KRX stock_code; stock_code is stored only as a listed-company mapping.`),
		Args: cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "sync companies"); err != nil {
				return nil, err
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			registryResult, err := p.FetchCompanies(cmd.Context(), flags.ListedOnly)
			if err != nil {
				return nil, err
			}
			database := storage.NewDatabase(opts.Database)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			companyRepository, err := opendartcompany.NewRepository(database)
			if err != nil {
				return nil, err
			}
			companyResult, err := companyRepository.UpsertCompanies(cmd.Context(), registryResult.Companies)
			if err != nil {
				return nil, err
			}
			rawRepository, err := providerraw.NewRepository(database)
			if err != nil {
				return nil, err
			}
			snapshot, err := rawRepository.UpsertSnapshot(cmd.Context(), providerraw.Snapshot{
				Provider:         registryResult.Provider,
				Group:            registryResult.Group,
				Operation:        registryResult.Operation,
				BaseDate:         time.Now().Format("2006-01-02"),
				CanonicalSupport: "company_registry",
				Rows:             registryResult.Companies,
				RowCount:         registryResult.TotalCount,
			})
			if err != nil {
				return nil, err
			}
			return opendartCompanySyncOutput{
				Provider:  provider.ProviderOpenDART,
				Operation: provider.OperationOpenDARTCorpCode,
				Companies: companyResult,
				Snapshot:  snapshot,
			}, nil
		}),
	}
	cmd.Flags().BoolVar(&flags.ListedOnly, "listed-only", flags.ListedOnly, "store only companies with OpenDART stock_code")
	return cmd
}

func newSearchCompaniesCommand(opts *Options) *cobra.Command {
	flags := opendartCompanySearchFlags{Limit: 20}
	cmd := &cobra.Command{
		Use:   "companies <query>",
		Short: "Search a provider-backed company registry",
		Long: strings.TrimSpace(`Search a provider-backed company registry.

With --provider opendart, the query matches local OpenDART corp_code, stock_code,
corp_name, or corp_eng_name. OpenDART corp_code is the disclosure identifier and
stock_code is the listed-company KRX mapping.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "search companies"); err != nil {
				return nil, err
			}
			database := storage.NewDatabase(opts.Database)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			repository, err := opendartcompany.NewRepository(database)
			if err != nil {
				return nil, err
			}
			companies, err := repository.Search(cmd.Context(), args[0], flags.ListedOnly, flags.Limit)
			if err != nil {
				return nil, err
			}
			return opendartCompanySearchOutput{Companies: companies}, nil
		}),
	}
	cmd.Flags().BoolVar(&flags.ListedOnly, "listed-only", flags.ListedOnly, "return only rows with OpenDART stock_code")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum company rows to return")
	return cmd
}

func newListFilingsCommand(opts *Options) *cobra.Command {
	flags := opendartFilingFlags{
		PageNo:    "1",
		PageCount: "10",
	}
	cmd := &cobra.Command{
		Use:   "filings [corp-code-or-stock-code]",
		Short: "List provider-backed filings",
		Long: strings.TrimSpace(`List provider-backed filings.

With --provider opendart, the positional argument may be an OpenDART corp_code or
a listed-company stock_code. stock_code is resolved to corp_code before querying
OpenDART. Use --corp-code to bypass stock_code resolution.`),
		Args: cobra.MaximumNArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "list filings"); err != nil {
				return nil, err
			}
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			if strings.TrimSpace(identifier) == "" && strings.TrimSpace(flags.CorpCode) == "" {
				return nil, oops.In("cli").New("list filings requires an identifier or --corp-code")
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			result, err := p.FetchFilings(cmd.Context(), opendartprovider.FilingRequest{
				Identifier: identifier,
				CorpCode:   flags.CorpCode,
				From:       flags.From,
				To:         flags.To,
				LastReport: flags.LastReport,
				PageNo:     flags.PageNo,
				PageCount:  flags.PageCount,
			})
			if err != nil {
				return nil, err
			}
			return opendartFilingOutput{Result: result}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.CorpCode, "corp-code", flags.CorpCode, "OpenDART corp_code; bypasses stock_code resolution")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "filing start date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "filing end date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().BoolVar(&flags.LastReport, "last-report", flags.LastReport, "request only final reports")
	cmd.Flags().StringVar(&flags.PageNo, "page-no", flags.PageNo, "OpenDART page number")
	cmd.Flags().StringVar(&flags.PageCount, "page-count", flags.PageCount, "OpenDART page size, max 100")
	return cmd
}

func requireOpenDARTProvider(opts *Options, command string) error {
	selected := ""
	if opts != nil {
		selected = strings.TrimSpace(opts.Provider)
	}
	errb := oops.In("cli").With("command", command, "provider", selected)
	if selected == "" {
		return errb.Errorf("%s requires --provider opendart", command)
	}
	if provider.ProviderID(selected) != provider.ProviderOpenDART {
		return errb.Errorf("%s does not support provider=%s; use --provider opendart", command, selected)
	}
	return nil
}

func buildOpenDARTProvider(opts *Options) (*opendartprovider.Provider, error) {
	builder := opendartprovider.NewBuilder()
	instance, err := builder.Build(opts.ProviderConfig)
	if err != nil {
		return nil, oops.In("cli").With("provider", provider.ProviderOpenDART).Wrapf(err, "build OpenDART provider")
	}
	p, ok := instance.(*opendartprovider.Provider)
	if !ok {
		return nil, oops.In("cli").With("provider", provider.ProviderOpenDART).New("configured OpenDART provider has unexpected type")
	}
	return p, nil
}

func (o opendartAPIListOutput) JSONValue() any {
	return o.Services
}

func (o opendartAPIListOutput) NDJSONRows() any {
	return o.Services
}

func (o opendartAPIListOutput) CSVRows() any {
	return o.Services
}

func (o opendartAPIListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Services))
	for _, service := range o.Services {
		rows = append(rows, []string{service.Category, service.Group, service.APIID, service.Description, service.CanonicalSupport})
	}
	return []string{"category", "group", "api_id", "description", "canonical_support"}, rows
}

func (o opendartCompanySearchOutput) JSONValue() any {
	return o.Companies
}

func (o opendartCompanySearchOutput) NDJSONRows() any {
	return o.Companies
}

func (o opendartCompanySearchOutput) CSVRows() any {
	return o.Companies
}

func (o opendartCompanySearchOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Companies))
	for _, company := range o.Companies {
		rows = append(rows, []string{company.CorpCode, company.StockCode, company.CorpName, company.CorpEngName, company.ModifyDate})
	}
	return []string{"corp_code", "stock_code", "corp_name", "corp_eng_name", "modify_date"}, rows
}

func (o opendartCompanySyncOutput) TableRows() ([]string, [][]string) {
	return []string{"provider", "operation", "companies", "listed", "rows_affected", "snapshot_rows"}, [][]string{{
		string(o.Provider),
		string(o.Operation),
		fmt.Sprint(o.Companies.TotalCount),
		fmt.Sprint(o.Companies.ListedCount),
		fmt.Sprint(o.Companies.RowsAffected),
		fmt.Sprint(o.Snapshot.RowCount),
	}}
}

func (o opendartFilingOutput) JSONValue() any {
	return o.Result
}

func (o opendartFilingOutput) NDJSONRows() any {
	return o.Result.Items
}

func (o opendartFilingOutput) CSVRows() any {
	return o.Result.Items
}

func (o opendartFilingOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Items))
	for _, item := range o.Result.Items {
		rows = append(rows, []string{item.CorpCode, item.StockCode, item.CorpName, item.Report, item.ReceiptNo, item.ReceiptAt, item.Remark})
	}
	return []string{"corp_code", "stock_code", "corp_name", "report_nm", "rcept_no", "rcept_dt", "rm"}, rows
}
