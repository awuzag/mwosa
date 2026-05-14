package cli

import (
	"strings"

	"github.com/ev3rlit/mwosa/app/handler"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type indexFlags struct {
	From string
	To   string
	AsOf string
}

func registerIndexCommands(roots commandRoots, opts *Options) {
	roots.Get.AddCommand(newGetIndexCommand(opts))
	roots.Sync.AddCommand(newSyncIndexCommand(opts))
}

func newGetIndexCommand(opts *Options) *cobra.Command {
	flags := indexFlags{}
	cmd := &cobra.Command{
		Use:   "index <index-code>",
		Short: "Fetch or read canonical index bars",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if strings.TrimSpace(flags.AsOf) == "" && strings.TrimSpace(flags.From) == "" && strings.TrimSpace(flags.To) == "" {
				return nil, oops.In("cli").New("get index requires --as-of or --from/--to")
			}
			runtime, err := newAppRuntime(opts, opts.Provider != "" || opts.PreferProvider != "")
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Index.Get(cmd.Context(), handler.GetIndexRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				IndexCode:      normalizeIndexCodeArg(args[0]),
				From:           flags.From,
				To:             flags.To,
				AsOf:           flags.AsOf,
			})
		}),
	}
	addIndexRangeFlags(cmd, &flags)
	return cmd
}

func newSyncIndexCommand(opts *Options) *cobra.Command {
	flags := indexFlags{}
	cmd := &cobra.Command{
		Use:   "index [index-code]",
		Short: "Fetch and store canonical index bars for a date",
		Args:  cobra.MaximumNArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if strings.TrimSpace(flags.AsOf) == "" {
				return nil, oops.In("cli").New("sync index requires --as-of")
			}
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			indexCode := ""
			if len(args) > 0 {
				indexCode = normalizeIndexCodeArg(args[0])
			}
			return runtime.Handlers.Index.Sync(cmd.Context(), handler.SyncIndexRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				IndexCode:      indexCode,
				AsOf:           flags.AsOf,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "trading date to collect, YYYYMMDD or YYYY-MM-DD")
	return cmd
}

func addIndexRangeFlags(cmd *cobra.Command, flags *indexFlags) {
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "start trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "end trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "single trading date, YYYYMMDD or YYYY-MM-DD")
}

func normalizeIndexCodeArg(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", "/", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(value)))
}
