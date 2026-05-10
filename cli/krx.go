package cli

import (
	"fmt"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	krxprovider "github.com/ev3rlit/mwosa/providers/krx"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/providerraw"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type krxRawFlags struct {
	AsOf string
}

type krxAPIListOutput struct {
	Services []krxAPIOutputRow `json:"services"`
}

type krxAPIOutputRow struct {
	Category         string `json:"category" csv:"category"`
	Group            string `json:"provider_group" csv:"group"`
	APIID            string `json:"api_id" csv:"api_id"`
	CanonicalSupport string `json:"canonical_support" csv:"canonical_support"`
}

type krxRawOutput struct {
	Result krxprovider.RawResult
}

func registerKRXCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListKRXAPIsCommand(opts))
	roots.Get.AddCommand(newGetKRXCommand(opts))
	roots.Sync.AddCommand(newSyncKRXCommand(opts))
}

func newListKRXAPIsCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "krx-apis",
		Short: "List KRX OPEN API services known to mwosa",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(_ *cobra.Command, _ []string) (any, error) {
			rows := make([]krxAPIOutputRow, 0, len(krxprovider.ServiceCatalog()))
			for _, service := range krxprovider.ServiceCatalog() {
				rows = append(rows, krxAPIOutputRow{
					Category:         service.Category,
					Group:            string(service.Group),
					APIID:            string(service.Operation),
					CanonicalSupport: krxCanonicalSupportLabel(service.Operation),
				})
			}
			return krxAPIListOutput{Services: rows}, nil
		}),
	}
}

func newGetKRXCommand(opts *Options) *cobra.Command {
	flags := krxRawFlags{}
	cmd := &cobra.Command{
		Use:               "krx <api-id>",
		Short:             "Fetch a provider-native KRX OPEN API response",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeKRXAPIIDs,
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			if strings.TrimSpace(flags.AsOf) == "" {
				return nil, oops.In("cli").New("get krx requires --as-of")
			}
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			result, err := fetchKRXRaw(cmd, opts, args[0], flags.AsOf)
			if err != nil {
				return nil, err
			}
			return krxRawOutput{Result: result}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "base date to query, YYYYMMDD or YYYY-MM-DD")
	return cmd
}

func newSyncKRXCommand(opts *Options) *cobra.Command {
	flags := krxRawFlags{}
	cmd := &cobra.Command{
		Use:               "krx <api-id>",
		Short:             "Fetch and store a provider-native KRX OPEN API snapshot",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeKRXAPIIDs,
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if strings.TrimSpace(flags.AsOf) == "" {
				return nil, oops.In("cli").New("sync krx requires --as-of")
			}
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			rawResult, err := fetchKRXRaw(cmd, opts, args[0], flags.AsOf)
			if err != nil {
				return nil, err
			}
			database := storage.NewDatabase(opts.Database)
			defer func() {
				err = oops.Join(err, database.Close())
			}()
			repository, err := providerraw.NewRepository(database)
			if err != nil {
				return nil, err
			}
			writeResult, err := repository.UpsertSnapshot(cmd.Context(), providerraw.Snapshot{
				Provider:         rawResult.Provider,
				Group:            rawResult.Group,
				Operation:        rawResult.APIID,
				BaseDate:         rawResult.BaseDate,
				CanonicalSupport: rawResult.Canonical,
				Rows:             rawResult.Rows,
				RowCount:         rawResult.RowCount,
			})
			if err != nil {
				return nil, err
			}
			return writeResult, nil
		}),
	}
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "base date to fetch and store, YYYYMMDD or YYYY-MM-DD")
	return cmd
}

func fetchKRXRaw(cmd *cobra.Command, opts *Options, apiID string, asOf string) (krxprovider.RawResult, error) {
	builder := krxprovider.NewBuilder()
	instance, err := builder.Build(opts.ProviderConfig)
	if err != nil {
		return krxprovider.RawResult{}, oops.In("cli").With("provider", provider.ProviderKRX).Wrapf(err, "build KRX provider")
	}
	krxProvider, ok := instance.(*krxprovider.Provider)
	if !ok {
		return krxprovider.RawResult{}, oops.In("cli").With("provider", provider.ProviderKRX).New("configured KRX provider has unexpected type")
	}
	return krxProvider.FetchRaw(cmd.Context(), krxprovider.RawRequest{
		APIID:    provider.OperationID(apiID),
		BaseDate: asOf,
	})
}

func (o krxAPIListOutput) JSONValue() any {
	return o.Services
}

func (o krxAPIListOutput) NDJSONRows() any {
	return o.Services
}

func (o krxAPIListOutput) CSVRows() any {
	return o.Services
}

func (o krxAPIListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Services))
	for _, service := range o.Services {
		rows = append(rows, []string{service.Category, service.Group, service.APIID, service.CanonicalSupport})
	}
	return []string{"category", "group", "api_id", "canonical_support"}, rows
}

func (o krxRawOutput) JSONValue() any {
	return o.Result
}

func (o krxRawOutput) NDJSONRows() any {
	return o.Result.Rows
}

func (o krxRawOutput) CSVRows() any {
	return o.Result.Rows
}

func (o krxRawOutput) TableRows() ([]string, [][]string) {
	return []string{"provider", "group", "api_id", "base_date", "rows", "canonical_support"}, [][]string{{
		string(o.Result.Provider),
		string(o.Result.Group),
		string(o.Result.APIID),
		o.Result.BaseDate,
		fmt.Sprint(o.Result.RowCount),
		o.Result.Canonical,
	}}
}

func (o krxRawOutput) SyncSnapshot() providerraw.Snapshot {
	return providerraw.Snapshot{
		Provider:         o.Result.Provider,
		Group:            o.Result.Group,
		Operation:        o.Result.APIID,
		BaseDate:         o.Result.BaseDate,
		CanonicalSupport: o.Result.Canonical,
		Rows:             o.Result.Rows,
		RowCount:         o.Result.RowCount,
	}
}

func completeKRXAPIIDs(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	values := make([]cobra.Completion, 0, len(krxprovider.ServiceCatalog()))
	for _, service := range krxprovider.ServiceCatalog() {
		values = append(values, cobra.CompletionWithDesc(string(service.Operation), string(service.Group)))
	}
	return values, cobra.ShellCompDirectiveNoFileComp
}

func krxCanonicalSupportLabel(apiID provider.OperationID) string {
	switch apiID {
	case provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd:
		return "daily_bar,instrument"
	case provider.OperationStockByddTrd, provider.OperationKOSDAQByddTrd, provider.OperationKONEXByddTrd:
		return "daily_bar"
	case provider.OperationStockIssueBaseInfo, provider.OperationKOSDAQIssueBaseInfo, provider.OperationKONEXIssueBaseInfo:
		return "instrument"
	default:
		return "raw_only"
	}
}
