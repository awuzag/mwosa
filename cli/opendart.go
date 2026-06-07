package cli

import (
	"fmt"
	"strings"
	"time"

	kisclient "github.com/awuzag/kis"
	provider "github.com/awuzag/mwosa/providers/core"
	opendartprovider "github.com/awuzag/mwosa/providers/opendart"
	"github.com/awuzag/mwosa/storage/companyevent"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/opendartcompany"
	"github.com/awuzag/mwosa/storage/providerraw"
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

type opendartFilingDocumentFlags struct {
	IncludePayload bool
}

type companyEventFlags struct {
	From      string
	To        string
	EventType string
	Limit     int
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
	Provider      provider.ProviderID          `json:"provider"`
	Operation     provider.OperationID         `json:"operation"`
	Companies     opendartcompany.UpsertResult `json:"companies"`
	IdentityGraph companyidentity.UpsertResult `json:"identity_graph"`
	Snapshot      providerraw.WriteResult      `json:"raw_snapshot"`
}

type opendartFilingOutput struct {
	Result opendartprovider.FilingResult
}

type opendartFilingDocumentOutput struct {
	Result opendartprovider.FilingDocument
}

type companyInspectOutput struct {
	Result companyidentity.InspectResult
}

type companyIdentifiersOutput struct {
	Result companyidentity.InspectResult
}

type companyEventsOutput struct {
	Events []companyevent.Event
}

func registerOpenDARTCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListProviderAPIsCommand(opts))
	roots.List.AddCommand(newListFilingsCommand(opts))
	roots.List.AddCommand(newListEventsCommand(opts))
	roots.Get.AddCommand(newGetCompanyIdentifiersCommand(opts))
	roots.Get.AddCommand(newGetFilingCommand(opts))
	roots.Inspect.AddCommand(newInspectCompanyCommand(opts))
	roots.Search.AddCommand(newSearchCompaniesCommand(opts))
	roots.Sync.AddCommand(newSyncCompaniesCommand(opts))
	roots.Sync.AddCommand(newSyncEventsCommand(opts))
}

func newListProviderAPIsCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider-apis <provider>",
		Short:             "List provider-native APIs known to mwosa diagnostics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runResult(opts, func(_ *cobra.Command, args []string) (any, error) {
			if provider.ProviderID(args[0]) == provider.ProviderKIS {
				rows := make([]kisAPIOutputRow, 0, len(kisclient.RawOperations()))
				for _, operation := range kisclient.RawOperations() {
					rows = append(rows, kisAPIOutputRow{
						Group:            operation.Group,
						APIID:            operation.OperationID,
						Method:           operation.Method,
						Endpoint:         operation.Endpoint,
						Description:      firstNonEmpty(operation.Description, operation.Summary),
						CanonicalSupport: kisRawCanonicalSupport(operation.RoleHint),
					})
				}
				return kisAPIListOutput{Services: rows}, nil
			}
			if provider.ProviderID(args[0]) != provider.ProviderOpenDART {
				return nil, oops.In("cli").
					With("provider", args[0]).
					New("provider API catalog is only available for kis or opendart")
			}
			rows := make([]opendartAPIOutputRow, 0, len(opendartprovider.ServiceCatalog()))
			for _, service := range opendartprovider.ServiceCatalog() {
				rows = append(rows, opendartAPIOutputRow{
					Category:         service.Category,
					Group:            string(service.Group),
					APIID:            string(service.Operation),
					Description:      service.Description,
					CanonicalSupport: service.CanonicalSupport,
				})
			}
			return opendartAPIListOutput{Services: rows}, nil
		}),
	}
}

type companyEventsSyncOutput struct {
	Company companyidentity.Company                  `json:"company"`
	Events  companyevent.UpsertResult                `json:"events"`
	Source  opendartprovider.CompanyEventBatchResult `json:"source"`
}

func newSyncEventsCommand(opts *Options) *cobra.Command {
	flags := companyEventFlags{}
	cmd := &cobra.Command{
		Use:   "events <company>",
		Short: "Fetch provider-backed material events and store canonical rows",
		Long: strings.TrimSpace(`Fetch provider-backed material events and store canonical rows.

With --provider opendart, this currently canonicalizes default, bank-management,
lawsuit, capital increase/reduction, business/asset transfer, CB/BW/EB,
merger/division, stock exchange/transfer, and treasury-stock decision APIs into
company_event_v1. Additional material event APIs remain separate until each
operation has an explicit mapping.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "sync events"); err != nil {
				return nil, err
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			database := newStorageDatabase(opts)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
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
			source, err := p.FetchMaterialEvents(cmd.Context(), opendartprovider.EventRequest{
				CorpCode: corpCode,
				From:     flags.From,
				To:       flags.To,
			})
			if err != nil {
				return nil, err
			}
			eventRepository, err := companyevent.NewRepository(database)
			if err != nil {
				return nil, err
			}
			upsert, err := eventRepository.UpsertEvents(cmd.Context(), company, openDARTEventsToStorage(source.Events))
			if err != nil {
				return nil, err
			}
			source.Events = nil
			return companyEventsSyncOutput{Company: company.Company, Events: upsert, Source: source}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "event filing start date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "event filing end date, YYYYMMDD or YYYY-MM-DD")
	return cmd
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
			database := newStorageDatabase(opts)
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
			identityRepository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			identityResult, err := identityRepository.UpsertCompanies(cmd.Context(), openDARTCompaniesToCanonical(registryResult.Companies))
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
				Provider:      provider.ProviderOpenDART,
				Operation:     provider.OperationOpenDARTCorpCode,
				Companies:     companyResult,
				IdentityGraph: identityResult,
				Snapshot:      snapshot,
			}, nil
		}),
	}
	cmd.Flags().BoolVar(&flags.ListedOnly, "listed-only", flags.ListedOnly, "store only companies with OpenDART stock_code")
	return cmd
}

func newGetCompanyIdentifiersCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "company-identifiers <company>",
		Short: "List canonical company identifiers",
		Long: strings.TrimSpace(`List canonical company identifiers.

The query is resolved from local company_v1 and company_identifier_v1 rows. OpenDART
corp_code and KRX stock_code are both identifiers, not canonical company keys.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			database := newStorageDatabase(opts)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			repository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			inspect, err := repository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			return companyIdentifiersOutput{Result: inspect}, nil
		}),
	}
	return cmd
}

func newInspectCompanyCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "company <company>",
		Short: "Inspect one canonical company",
		Long: strings.TrimSpace(`Inspect one canonical company.

The query is resolved from local company_v1 and company_identifier_v1 rows. Use
sync companies --provider opendart to populate the initial Korean listed-company
identity graph from corpCode.xml.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			database := newStorageDatabase(opts)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			repository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			inspect, err := repository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			return companyInspectOutput{Result: inspect}, nil
		}),
	}
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
			database := newStorageDatabase(opts)
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

func newGetFilingCommand(opts *Options) *cobra.Command {
	flags := opendartFilingDocumentFlags{}
	cmd := &cobra.Command{
		Use:   "filing <rcept-no>",
		Short: "Fetch provider-backed filing document metadata",
		Long: strings.TrimSpace(`Fetch provider-backed filing document metadata.

With --provider opendart, this calls document.xml by rcept_no. Binary payload is
omitted by default so table, csv, json, and ndjson output remain safe for normal
stdout pipelines. Use --include-payload with json or ndjson to include the file
body as base64.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			if err := requireOpenDARTProvider(opts, "get filing"); err != nil {
				return nil, err
			}
			p, err := buildOpenDARTProvider(opts)
			if err != nil {
				return nil, err
			}
			result, err := p.FetchFilingDocument(cmd.Context(), opendartprovider.FilingDocumentRequest{ReceiptNo: args[0]})
			if err != nil {
				return nil, err
			}
			if !flags.IncludePayload {
				result.PayloadBase64 = ""
			}
			return opendartFilingDocumentOutput{Result: result}, nil
		}),
	}
	cmd.Flags().BoolVar(&flags.IncludePayload, "include-payload", flags.IncludePayload, "include base64 file payload in json or ndjson output")
	return cmd
}

func newListEventsCommand(opts *Options) *cobra.Command {
	flags := companyEventFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "events <company>",
		Short: "List stored canonical company events",
		Long: strings.TrimSpace(`List stored canonical company events.

This reads company_event_v1 from local SQLite. Use sync events --provider
opendart to fetch canonicalized OpenDART material events first.`),
		Args: cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			database := newStorageDatabase(opts)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			companyRepository, err := companyidentity.NewRepository(database)
			if err != nil {
				return nil, err
			}
			company, err := companyRepository.Inspect(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			eventRepository, err := companyevent.NewRepository(database)
			if err != nil {
				return nil, err
			}
			events, err := eventRepository.ListEvents(cmd.Context(), company, companyevent.Query{
				Provider:  provider.ProviderID(opts.Provider),
				EventType: flags.EventType,
				From:      flags.From,
				To:        flags.To,
				Limit:     flags.Limit,
			})
			if err != nil {
				return nil, err
			}
			if len(events) == 0 {
				return nil, oops.In("cli").With("company", args[0], "from", flags.From, "provider", opts.Provider).New("stored company events not found")
			}
			return companyEventsOutput{Events: events}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "event date lower bound, YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "event date upper bound, YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.EventType, "event-type", flags.EventType, "event type filter")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum event rows to return")
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

func openDARTCompaniesToCanonical(companies []opendartprovider.Company) []companyidentity.CompanyInput {
	inputs := make([]companyidentity.CompanyInput, 0, len(companies))
	for _, company := range companies {
		identifiers := []companyidentity.IdentifierInput{
			{
				Provider:        provider.ProviderOpenDART,
				Group:           provider.GroupOpenDARTDisclosure,
				Operation:       provider.OperationOpenDARTCorpCode,
				IdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
				IdentifierValue: strings.TrimSpace(company.CorpCode),
				Primary:         true,
				Confidence:      1,
				SourceUpdatedAt: strings.TrimSpace(company.ModifyDate),
			},
		}
		stockCode := strings.TrimSpace(company.StockCode)
		if stockCode != "" {
			identifiers = append(identifiers, companyidentity.IdentifierInput{
				Provider:        provider.ProviderOpenDART,
				Group:           provider.GroupOpenDARTDisclosure,
				Operation:       provider.OperationOpenDARTCorpCode,
				IdentifierType:  companyidentity.IdentifierTypeKRXStockCode,
				IdentifierValue: stockCode,
				Primary:         false,
				Confidence:      1,
				SourceUpdatedAt: strings.TrimSpace(company.ModifyDate),
			})
		}
		input := companyidentity.CompanyInput{
			Name:        strings.TrimSpace(company.CorpName),
			LegalName:   strings.TrimSpace(company.CorpName),
			EnglishName: strings.TrimSpace(company.CorpEngName),
			CountryCode: "KR",
			Identifiers: identifiers,
		}
		if stockCode != "" {
			input.InstrumentRef = companyidentity.InstrumentRef{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				Symbol:       stockCode,
				Name:         strings.TrimSpace(company.CorpName),
				RelationType: companyidentity.RelationTypeIssuer,
			}
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func (o companyInspectOutput) JSONValue() any {
	return o.Result
}

func (o companyInspectOutput) TableRows() ([]string, [][]string) {
	rows := [][]string{
		{"company", "id", fmt.Sprint(o.Result.Company.ID), ""},
		{"company", "name", o.Result.Company.Name, ""},
		{"company", "legal_name", o.Result.Company.LegalName, ""},
		{"company", "english_name", o.Result.Company.EnglishName, ""},
		{"company", "country_code", o.Result.Company.CountryCode, ""},
	}
	for _, identifier := range o.Result.Identifiers {
		rows = append(rows, []string{
			"identifier",
			identifier.IdentifierType,
			identifier.IdentifierValue,
			string(identifier.Provider) + "/" + string(identifier.Group) + "/" + string(identifier.Operation),
		})
	}
	for _, link := range o.Result.Instruments {
		rows = append(rows, []string{
			"instrument",
			link.RelationType,
			string(link.Market) + "/" + string(link.SecurityType) + "/" + link.Symbol,
			link.Name,
		})
	}
	return []string{"section", "key", "value", "source"}, rows
}

func (o companyIdentifiersOutput) JSONValue() any {
	return o.Result.Identifiers
}

func (o companyIdentifiersOutput) NDJSONRows() any {
	return o.Result.Identifiers
}

func (o companyIdentifiersOutput) CSVRows() any {
	return o.Result.Identifiers
}

func (o companyIdentifiersOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Identifiers))
	for _, identifier := range o.Result.Identifiers {
		rows = append(rows, []string{
			identifier.IdentifierType,
			identifier.IdentifierValue,
			string(identifier.Provider),
			string(identifier.Group),
			string(identifier.Operation),
			fmt.Sprint(identifier.Primary),
			fmt.Sprintf("%.2f", identifier.Confidence),
		})
	}
	return []string{"identifier_type", "identifier_value", "provider", "group", "operation", "primary", "confidence"}, rows
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

func (o opendartFilingDocumentOutput) JSONValue() any {
	return o.Result
}

func (o opendartFilingDocumentOutput) NDJSONRows() any {
	return []opendartprovider.FilingDocument{o.Result}
}

func (o opendartFilingDocumentOutput) CSVRows() any {
	return []opendartprovider.FilingDocument{o.Result}
}

func (o opendartFilingDocumentOutput) TableRows() ([]string, [][]string) {
	return []string{"provider", "operation", "rcept_no", "content_type", "size_bytes", "sha256"}, [][]string{{
		string(o.Result.Provider),
		string(o.Result.Operation),
		o.Result.ReceiptNo,
		o.Result.ContentType,
		fmt.Sprint(o.Result.SizeBytes),
		o.Result.SHA256,
	}}
}

func (o companyEventsOutput) JSONValue() any {
	return o.Events
}

func (o companyEventsOutput) NDJSONRows() any {
	return o.Events
}

func (o companyEventsOutput) CSVRows() any {
	return o.Events
}

func (o companyEventsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Events))
	for _, event := range o.Events {
		rows = append(rows, []string{
			event.EventDate,
			event.EventType,
			event.Title,
			event.ValueText,
			event.RceptNo,
			string(event.Provider) + "/" + string(event.Group) + "/" + string(event.Operation),
		})
	}
	return []string{"event_date", "event_type", "title", "value_text", "rcept_no", "source"}, rows
}

func (o companyEventsSyncOutput) TableRows() ([]string, [][]string) {
	return []string{"company_id", "company", "events_written", "provider", "sources"}, [][]string{{
		fmt.Sprint(o.Company.ID),
		o.Company.Name,
		fmt.Sprint(o.Events.EventsWritten),
		string(o.Source.Provider),
		fmt.Sprint(len(o.Source.Sources)),
	}}
}

func openDARTEventsToStorage(events []opendartprovider.CompanyEvent) []companyevent.EventInput {
	out := make([]companyevent.EventInput, 0, len(events))
	for _, event := range events {
		out = append(out, companyevent.EventInput{
			EventType:   event.EventType,
			EventDate:   event.EventDate,
			RceptDt:     event.RceptDt,
			RceptNo:     event.RceptNo,
			Provider:    event.Provider,
			Group:       event.Group,
			Operation:   event.Operation,
			Title:       event.Title,
			AmountMinor: event.AmountMinor,
			ValueText:   event.ValueText,
			Raw:         event.Raw,
		})
	}
	return out
}
