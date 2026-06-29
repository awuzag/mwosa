package cli

import (
	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/spf13/cobra"
)

type intradayFlags struct {
	SecurityType string
	At           string
	Limit        int
}

type orderbookFlags struct {
	SecurityType string
}

type tradesFlags struct {
	SecurityType string
	At           string
	Limit        int
}

type constituentsFlags struct {
	Limit int
}

func registerMarketDataCommands(roots commandRoots, opts *Options) {
	roots.Get.AddCommand(newGetIntradayCommand(opts))
	roots.Get.AddCommand(newGetOrderbookCommand(opts))
	roots.List.AddCommand(newListTradesCommand(opts))
	roots.List.AddCommand(newListConstituentsCommand(opts))
}

func newGetIntradayCommand(opts *Options) *cobra.Command {
	flags := intradayFlags{SecurityType: string(provider.SecurityTypeStock)}
	cmd := &cobra.Command{
		Use:   "intraday <symbol>",
		Short: "Fetch provider intraday bars for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Intraday.Get(cmd.Context(), handler.GetIntradayRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
				At:             flags.At,
				Limit:          flags.Limit,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn")
	cmd.Flags().StringVar(&flags.At, "at", flags.At, "provider-neutral time anchor in HHMMSS or HH:MM:SS form")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of intraday bars to return")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	return cmd
}

func newGetOrderbookCommand(opts *Options) *cobra.Command {
	flags := orderbookFlags{SecurityType: string(provider.SecurityTypeStock)}
	cmd := &cobra.Command{
		Use:   "orderbook <symbol>",
		Short: "Fetch a provider orderbook snapshot for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Orderbooks.Get(cmd.Context(), handler.GetOrderbookRequest{
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

func newListTradesCommand(opts *Options) *cobra.Command {
	flags := tradesFlags{SecurityType: string(provider.SecurityTypeStock)}
	cmd := &cobra.Command{
		Use:   "trades <symbol>",
		Short: "List recent market trade prints for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Trades.List(cmd.Context(), handler.ListTradesRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
				At:             flags.At,
				Limit:          flags.Limit,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn")
	cmd.Flags().StringVar(&flags.At, "at", flags.At, "provider-neutral time anchor in HHMMSS or HH:MM:SS form")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of market trades to return")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
	return cmd
}

func newListConstituentsCommand(opts *Options) *cobra.Command {
	flags := constituentsFlags{}
	cmd := &cobra.Command{
		Use:   "constituents <symbol>",
		Short: "List composition constituents for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Compositions.List(cmd.Context(), handler.ListConstituentsRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityTypeETF,
				Symbol:         args[0],
				Limit:          flags.Limit,
			})
		}),
	}
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum number of constituents to return")
	return cmd
}
