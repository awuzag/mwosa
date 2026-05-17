package cli

import (
	"strconv"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/storage/providerraw"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type providerRawSnapshotFlags struct {
	Group          string
	Operation      string
	From           string
	To             string
	Limit          int
	IncludePayload bool
}

type providerRawSnapshotsOutput struct {
	Snapshots []providerraw.SnapshotRecord `json:"snapshots"`
}

func registerProviderRawCommands(roots commandRoots, opts *Options) {
	roots.Get.AddCommand(newGetProviderRawSnapshotsCommand(opts))
	roots.Get.AddCommand(newGetProviderRawCommand(opts))
}

func newGetProviderRawSnapshotsCommand(opts *Options) *cobra.Command {
	flags := providerRawSnapshotFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "provider-raw-snapshots",
		Short: "Read stored provider-native raw snapshots",
		Long: strings.TrimSpace(`Read stored provider-native raw snapshots.

This reads provider_raw_snapshots from local SQLite. It is an escape hatch for
provider APIs that are not yet canonicalized, while keeping canonical analysis
tables separate from provider-native payloads.`),
		Args: cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			return readProviderRawSnapshots(cmd, opts, providerraw.Query{
				Provider:       provider.ProviderID(opts.Provider),
				Group:          provider.GroupID(flags.Group),
				Operation:      provider.OperationID(flags.Operation),
				From:           flags.From,
				To:             flags.To,
				Limit:          flags.Limit,
				IncludePayload: flags.IncludePayload,
			})
		}),
	}
	addProviderRawSnapshotFlags(cmd, &flags)
	return cmd
}

func newGetProviderRawCommand(opts *Options) *cobra.Command {
	flags := providerRawSnapshotFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "provider-raw [provider] [operation]",
		Short: "Read stored provider-native raw payload snapshots",
		Long: strings.TrimSpace(`Read stored provider-native raw payload snapshots.

This is a friendlier alias over provider_raw_snapshots for canonicalization
escape hatches. It does not call the provider live; it only reads snapshots that
previous sync commands have already stored locally.`),
		Args: cobra.MaximumNArgs(2),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			query, err := providerRawQueryFromArgs(opts, flags, args)
			if err != nil {
				return nil, err
			}
			return readProviderRawSnapshots(cmd, opts, query)
		}),
	}
	addProviderRawSnapshotFlags(cmd, &flags)
	return cmd
}

func addProviderRawSnapshotFlags(cmd *cobra.Command, flags *providerRawSnapshotFlags) {
	cmd.Flags().StringVar(&flags.Group, "group", flags.Group, "provider group filter")
	cmd.Flags().StringVar(&flags.Operation, "operation", flags.Operation, "provider operation/api id filter")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "base date lower bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "base date upper bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum snapshots to return")
	cmd.Flags().BoolVar(&flags.IncludePayload, "include-payload", flags.IncludePayload, "include decoded provider-native payload in JSON/NDJSON output")
}

func providerRawQueryFromArgs(opts *Options, flags providerRawSnapshotFlags, args []string) (providerraw.Query, error) {
	providerID := strings.TrimSpace(opts.Provider)
	operation := strings.TrimSpace(flags.Operation)
	switch len(args) {
	case 0:
	case 1:
		if providerID == "" {
			providerID = strings.TrimSpace(args[0])
		} else if operation == "" {
			operation = strings.TrimSpace(args[0])
		} else if operation != strings.TrimSpace(args[0]) {
			return providerraw.Query{}, oops.In("cli").With("operation", operation, "arg", args[0]).New("provider raw operation flag conflicts with positional operation")
		}
	case 2:
		if providerID != "" && providerID != strings.TrimSpace(args[0]) {
			return providerraw.Query{}, oops.In("cli").With("provider", providerID, "arg", args[0]).New("provider raw provider flag conflicts with positional provider")
		}
		providerID = strings.TrimSpace(args[0])
		if operation != "" && operation != strings.TrimSpace(args[1]) {
			return providerraw.Query{}, oops.In("cli").With("operation", operation, "arg", args[1]).New("provider raw operation flag conflicts with positional operation")
		}
		operation = strings.TrimSpace(args[1])
	}
	return providerraw.Query{
		Provider:       provider.ProviderID(providerID),
		Group:          provider.GroupID(flags.Group),
		Operation:      provider.OperationID(operation),
		From:           flags.From,
		To:             flags.To,
		Limit:          flags.Limit,
		IncludePayload: flags.IncludePayload,
	}, nil
}

func readProviderRawSnapshots(cmd *cobra.Command, opts *Options, query providerraw.Query) (result any, err error) {
	runtime, err := newAppRuntime(opts, false)
	if err != nil {
		return nil, err
	}
	defer closeAppRuntime(runtime, &err)

	repository, err := providerraw.NewRepository(runtime.Storage.Database)
	if err != nil {
		return nil, err
	}
	snapshots, err := repository.ListSnapshots(cmd.Context(), query)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, oops.In("cli").With("provider", query.Provider, "operation", query.Operation).New("stored provider raw snapshots not found")
	}
	return providerRawSnapshotsOutput{Snapshots: snapshots}, nil
}

func (o providerRawSnapshotsOutput) JSONValue() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) NDJSONRows() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) CSVRows() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Snapshots))
	for _, snapshot := range o.Snapshots {
		rows = append(rows, []string{
			string(snapshot.Provider),
			string(snapshot.Group),
			string(snapshot.Operation),
			snapshot.BaseDate,
			snapshot.CanonicalSupport,
			strconv.Itoa(snapshot.RowCount),
		})
	}
	return []string{"provider", "group", "operation", "base_date", "canonical_support", "row_count"}, rows
}
