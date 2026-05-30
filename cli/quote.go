package cli

import (
	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/spf13/cobra"
)

type quoteFlags struct {
	SecurityType string
}

func registerQuoteCommands(roots commandRoots, opts *Options) {
	roots.Get.AddCommand(newGetQuoteCommand(opts))
}

func newGetQuoteCommand(opts *Options) *cobra.Command {
	flags := quoteFlags{SecurityType: string(provider.SecurityTypeStock)}
	cmd := &cobra.Command{
		Use:   "quote <symbol>",
		Short: "Fetch a provider quote snapshot for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Quotes.Get(cmd.Context(), handler.GetQuoteRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	return cmd
}
