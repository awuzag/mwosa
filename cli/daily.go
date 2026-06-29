package cli

import (
	"context"

	"github.com/awuzag/mwosa/app"
	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type dailyFlags struct {
	SecurityType string
	From         string
	To           string
	AsOf         string
	Workers      int
}

func registerDailyCommands(roots commandRoots, opts *Options) {
	roots.Inspect.AddCommand(newInspectStorageCommand(opts))
	roots.Inspect.AddCommand(newInspectCoverageCommand(opts))
	roots.Get.AddCommand(newGetDailyCommand(opts))
	roots.Ensure.AddCommand(newEnsureDailyCommand(opts))
	roots.Sync.AddCommand(newSyncDailyCommand(opts))
	roots.Backfill.AddCommand(newBackfillDailyCommand(opts))
}

func newInspectStorageCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Summarize local daily bar storage coverage",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.StorageSummary(cmd.Context(), handler.DailyStorageSummaryRequest{
				Market:       provider.Market(opts.Market),
				SecurityType: provider.SecurityType(flags.SecurityType),
			})
		}),
	}
	addSecurityTypeFlag(cmd, &flags)
	return cmd
}

func newInspectCoverageCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "coverage <symbol>",
		Short: "Inspect local daily bar coverage for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.Coverage(cmd.Context(), handler.DailyCoverageRequest{
				Market:       provider.Market(opts.Market),
				SecurityType: provider.SecurityType(flags.SecurityType),
				Symbol:       args[0],
			})
		}),
	}
	addSecurityTypeFlag(cmd, &flags)
	return cmd
}

func newGetDailyCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "daily <symbol>",
		Short: "Read stored daily bars for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.Get(cmd.Context(), handler.GetDailyRequest{
				Market:       provider.Market(opts.Market),
				SecurityType: provider.SecurityType(flags.SecurityType),
				Symbol:       args[0],
				From:         flags.From,
				To:           flags.To,
				AsOf:         flags.AsOf,
			})
		}),
	}
	addDailyRangeFlags(cmd, &flags)
	return cmd
}

func newEnsureDailyCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "daily <symbol>",
		Short: "Fetch missing daily bars for a symbol and store them locally",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.Ensure(cmd.Context(), handler.EnsureDailyRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				Symbol:         args[0],
				From:           flags.From,
				To:             flags.To,
				AsOf:           flags.AsOf,
			})
		}),
	}
	addDailyRangeFlags(cmd, &flags)
	return cmd
}

func newSyncDailyCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Collect one provider daily batch for a date",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.Sync(cmd.Context(), handler.SyncDailyRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				AsOf:           flags.AsOf,
			})
		}),
	}
	addSecurityTypeFlag(cmd, &flags)
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "trading date to collect, YYYYMMDD or YYYY-MM-DD")
	return cmd
}

func newBackfillDailyCommand(opts *Options) *cobra.Command {
	flags := dailyFlags{SecurityType: string(provider.SecurityTypeETF)}
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Collect provider daily batches for a date range",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)

			return runtime.Handlers.Daily.Backfill(cmd.Context(), handler.BackfillDailyRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Market:         provider.Market(opts.Market),
				SecurityType:   provider.SecurityType(flags.SecurityType),
				From:           flags.From,
				To:             flags.To,
				Workers:        flags.Workers,
				Progress:       cmd.ErrOrStderr(),
			})
		}),
	}
	addSecurityTypeFlag(cmd, &flags)
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "start trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "end trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().IntVar(&flags.Workers, "workers", 1, "number of workers for page-based daily providers")
	return cmd
}

func addDailyRangeFlags(cmd *cobra.Command, flags *dailyFlags) {
	addSecurityTypeFlag(cmd, flags)
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "start trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "end trading date, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.AsOf, "as-of", flags.AsOf, "single trading date, YYYYMMDD or YYYY-MM-DD")
}

func addSecurityTypeFlag(cmd *cobra.Command, flags *dailyFlags) {
	cmd.Flags().StringVar(&flags.SecurityType, "security-type", flags.SecurityType, "security type: stock, etf, etn, elw")
	mustRegisterFlagCompletion(cmd, "security-type", completeSecurityTypes)
}

func newAppRuntime(ctx context.Context, opts *Options, activateProviders bool) (*app.Runtime, error) {
	if opts == nil {
		return nil, oops.In("cli").New("cli options are nil")
	}
	if err := loadConfig(opts); err != nil {
		return nil, err
	}
	if err := opts.Validate(); err != nil {
		return nil, oops.In("cli").Wrapf(err, "validate cli options")
	}
	return app.NewRuntime(ctx, app.Options{
		DatabaseBackend:      opts.DatabaseBackend,
		Database:             opts.Database,
		DatabaseURL:          opts.DatabaseURL,
		ProviderAuthDatabase: opts.ProviderAuthDatabase,
		Market:               provider.Market(opts.Market),
		ProviderID:           provider.ProviderID(opts.Provider),
		PreferProvider:       provider.ProviderID(opts.PreferProvider),
		ProviderConfig:       opts.ProviderConfig,
		ActivateProviders:    activateProviders,
	})
}

func closeAppRuntime(ctx context.Context, runtime *app.Runtime, err *error) {
	if runtime == nil {
		return
	}
	*err = oops.Join(*err, runtime.Close(ctx))
}
