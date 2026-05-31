package cli

import (
	"strings"

	"github.com/awuzag/mwosa/app/handler"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/macro"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type macroFlags struct {
	Preset string
	From   string
	To     string
}

func registerMacroCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListMacroIndicatorsCommand(opts))
	roots.Get.AddCommand(newGetMacroCommand(opts))
	roots.Sync.AddCommand(newSyncMacroCommand(opts))
}

func newListMacroIndicatorsCommand(opts *Options) *cobra.Command {
	flags := macroFlags{}
	cmd := &cobra.Command{
		Use:   "macro-indicators",
		Short: "List macro indicator metadata",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, opts.Provider != "" || opts.PreferProvider != "")
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Macro.ListIndicators(cmd.Context(), handler.ListMacroIndicatorsRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Preset:         macro.Preset(strings.TrimSpace(flags.Preset)),
			})
		}),
	}
	cmd.Flags().StringVar(&flags.Preset, "preset", flags.Preset, "macro indicator preset, e.g. key-statistics")
	return cmd
}

func newGetMacroCommand(opts *Options) *cobra.Command {
	flags := macroFlags{}
	cmd := &cobra.Command{
		Use:   "macro <indicator-id>",
		Short: "Read stored macro observations",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Macro.Get(cmd.Context(), handler.GetMacroRequest{
				IndicatorID: strings.TrimSpace(args[0]),
				From:        flags.From,
				To:          flags.To,
			})
		}),
	}
	addMacroRangeFlags(cmd, &flags)
	return cmd
}

func newSyncMacroCommand(opts *Options) *cobra.Command {
	flags := macroFlags{}
	cmd := &cobra.Command{
		Use:   "macro <key-statistics|indicator-id>",
		Short: "Fetch and store macro indicator metadata or observations",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			target := strings.TrimSpace(args[0])
			if target == "" {
				return nil, oops.In("cli").New("sync macro requires target")
			}
			runtime, err := newAppRuntime(opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Macro.Sync(cmd.Context(), handler.SyncMacroRequest{
				ProviderID:     provider.ProviderID(opts.Provider),
				PreferProvider: provider.ProviderID(opts.PreferProvider),
				Target:         target,
				From:           flags.From,
				To:             flags.To,
			})
		}),
	}
	addMacroRangeFlags(cmd, &flags)
	return cmd
}

func addMacroRangeFlags(cmd *cobra.Command, flags *macroFlags) {
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "start period, e.g. YYYY-MM")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "end period, e.g. YYYY-MM")
}
