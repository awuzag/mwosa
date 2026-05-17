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
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			repository, err := providerraw.NewRepository(runtime.Storage.Database)
			if err != nil {
				return nil, err
			}
			snapshots, err := repository.ListSnapshots(cmd.Context(), providerraw.Query{
				Provider:       provider.ProviderID(opts.Provider),
				Group:          provider.GroupID(flags.Group),
				Operation:      provider.OperationID(flags.Operation),
				From:           flags.From,
				To:             flags.To,
				Limit:          flags.Limit,
				IncludePayload: flags.IncludePayload,
			})
			if err != nil {
				return nil, err
			}
			if len(snapshots) == 0 {
				return nil, oops.In("cli").With("provider", opts.Provider, "operation", flags.Operation).New("stored provider raw snapshots not found")
			}
			return providerRawSnapshotsOutput{Snapshots: snapshots}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.Group, "group", flags.Group, "provider group filter")
	cmd.Flags().StringVar(&flags.Operation, "operation", flags.Operation, "provider operation/api id filter")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "base date lower bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "base date upper bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum snapshots to return")
	cmd.Flags().BoolVar(&flags.IncludePayload, "include-payload", flags.IncludePayload, "include decoded provider-native payload in JSON/NDJSON output")
	return cmd
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
