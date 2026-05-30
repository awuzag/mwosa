package cli

import (
	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/spf13/cobra"
)

type instrumentFlags struct {
	SecurityType string
	Limit        int
}

func registerInstrumentCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListInstrumentsCommand(opts))
	roots.Search.AddCommand(newSearchInstrumentsCommand(opts))
	roots.Sync.AddCommand(newSyncInstrumentsCommand(opts))
	roots.Inspect.AddCommand(newInspectInstrumentCommand(opts))
}

func newSearchInstrumentsCommand(opts *Options) *cobra.Command {
	cmd := newListInstrumentsCommand(opts)
	cmd.Use = "instruments <query>"
	cmd.Short = "Search stored instruments"
	return cmd
}

func newListInstrumentsCommand(opts *Options) *cobra.Command {
	flags := instrumentFlags{
		SecurityType: string(provider.SecurityTypeStock),
		Limit:        25,
	}
	cmd := &cobra.Command{
		Use:   "instruments <query>",
		Short: "Search stored instruments, falling back to provider search",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Instruments.List(cmd.Context(), handler.ListInstrumentsRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Query:          args[0],
				Limit:          flags.Limit,
			})
		}),
	}
	addInstrumentFlags(cmd, &flags)
	return cmd
}

func newSyncInstrumentsCommand(opts *Options) *cobra.Command {
	flags := instrumentFlags{SecurityType: string(provider.SecurityTypeStock)}
	asOf := ""
	cmd := &cobra.Command{
		Use:   "instruments",
		Short: "Fetch provider instrument master and store it locally",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Instruments.Sync(cmd.Context(), handler.SyncInstrumentsRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				AsOf:           asOf,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock")
	cmd.Flags().StringVar(&asOf, "as-of", asOf, "base date to collect, YYYYMMDD or YYYY-MM-DD")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	return cmd
}

func newInspectInstrumentCommand(opts *Options) *cobra.Command {
	flags := instrumentFlags{SecurityType: string(provider.SecurityTypeStock)}
	cmd := &cobra.Command{
		Use:   "instrument <symbol>",
		Short: "Inspect one provider instrument",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Instruments.Inspect(cmd.Context(), handler.InspectInstrumentRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn, elw")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	return cmd
}

func addInstrumentFlags(cmd *cobra.Command, flags *instrumentFlags) {
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn, elw")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of instruments to return")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
}
