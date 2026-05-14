package cli

import (
	"github.com/ev3rlit/mwosa/app/handler"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/spf13/cobra"
)

func registerStrategyCommands(roots commandRoots, opts *Options) {
	roots.Create.AddCommand(newCreateStrategyCommand(opts))
	roots.List.AddCommand(newListStrategiesCommand(opts))
	roots.Update.AddCommand(newUpdateStrategyCommand(opts))
	roots.Update.AddCommand(newUpdateScreenCommand(opts))
	roots.Delete.AddCommand(newDeleteStrategyCommand(opts))
	roots.Screen.AddCommand(newScreenETFCommand(opts))
	roots.Screen.AddCommand(newScreenStrategyCommand(opts))
	roots.Screen.AddCommand(newScreenPipelineCommand(opts))
	roots.Compare.AddCommand(newCompareScreenCommand(opts))
	roots.History.AddCommand(newHistoryScreenCommand(opts))
	roots.Inspect.AddCommand(newInspectStrategyCommand(opts))
	roots.Inspect.AddCommand(newInspectScreenCommand(opts))
	roots.Inspect.AddCommand(newInspectScreenPipelineCommand(opts))
	roots.Inspect.AddCommand(newInspectMarketRegimeCommand(opts))
	roots.Inspect.AddCommand(newInspectStrategySetCommand(opts))
}

func newCreateStrategyCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{Engine: string(strategyservice.EngineJQ)}
	cmd := &cobra.Command{
		Use:   "strategy <name>",
		Short: "Create a saved screening strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			queryText, err := resolveJQSource(flags)
			if err != nil {
				return nil, err
			}
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.Create(cmd.Context(), handler.CreateStrategyRequest{
				Name:         args[0],
				Engine:       strategyservice.Engine(flags.Engine),
				InputDataset: flags.Input,
				QueryText:    queryText,
			})
		}),
	}
	addStrategySourceFlags(cmd, &flags, true)
	return cmd
}

func newListStrategiesCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategies",
		Short: "List saved screening strategies",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.List(cmd.Context(), handler.ListStrategiesRequest{})
		}),
	}
}

func newUpdateStrategyCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{}
	cmd := &cobra.Command{
		Use:   "strategy <name>",
		Short: "Create a new version of a saved screening strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			queryText, err := resolveJQSource(flags)
			if err != nil {
				return nil, err
			}
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.Update(cmd.Context(), handler.UpdateStrategyRequest{
				Name:      args[0],
				QueryText: queryText,
			})
		}),
	}
	addJQFlags(cmd, &flags)
	return cmd
}

func newUpdateScreenCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screen",
		Short: "Update saved screen resources",
	}
	cmd.AddCommand(newUpdateScreenStrategyCommand(opts))
	return cmd
}

func newUpdateScreenStrategyCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{}
	cmd := &cobra.Command{
		Use:   "strategy <name>",
		Short: "Create or update a saved screen strategy from YAML",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.UpsertScreenStrategy(cmd.Context(), handler.UpsertScreenStrategyRequest{
				Name: args[0],
				Path: flags.File,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.File, "file", flags.File, "path to a ScreenStrategy or ScreenRun YAML file")
	mustMarkFlagFilename(cmd, "file", "yaml", "yml")
	return cmd
}

func newDeleteStrategyCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategy <name>",
		Short: "Soft delete a saved screening strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.Delete(cmd.Context(), handler.DeleteStrategyRequest{Name: args[0]})
		}),
	}
}

func newScreenETFCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{Input: "etf_daily_metrics"}
	cmd := &cobra.Command{
		Use:     "etf",
		Aliases: []string{"etfs"},
		Short:   "Run an inline jq screen against stored ETF daily records",
		Args:    cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			queryText, err := resolveJQSource(flags)
			if err != nil {
				return nil, err
			}
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.ScreenJQ(cmd.Context(), handler.ScreenJQRequest{
				InputDataset: flags.Input,
				QueryText:    queryText,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.Input, "input", flags.Input, "input dataset name")
	addJQFlags(cmd, &flags)
	return cmd
}

func newScreenStrategyCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{}
	cmd := &cobra.Command{
		Use:   "strategy <name>",
		Short: "Run a saved screening strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.Screen(cmd.Context(), handler.ScreenStrategyRequest{
				Name:     args[0],
				Alias:    flags.Alias,
				Version:  flags.Version,
				SpecHash: flags.SpecHash,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.Alias, "alias", flags.Alias, "optional screen run alias")
	cmd.Flags().StringVar(&flags.Version, "version", flags.Version, "strategy version number or latest")
	cmd.Flags().StringVar(&flags.SpecHash, "spec-hash", flags.SpecHash, "strategy spec hash to run")
	return cmd
}

func newScreenPipelineCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline <yaml>",
		Short: "Run a YAML screen universe pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.ScreenPipeline(cmd.Context(), handler.ScreenPipelineRequest{Path: args[0]})
		}),
	}
	mustMarkScreenPipelineYAML(cmd)
	return cmd
}

func newCompareScreenCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screen",
		Short: "Compare saved screen resources",
	}
	cmd.AddCommand(newCompareScreenStrategiesCommand(opts))
	return cmd
}

func newCompareScreenStrategiesCommand(opts *Options) *cobra.Command {
	var asOf string
	var topN int
	cmd := &cobra.Command{
		Use:   "strategies <name> <name> [name...]",
		Short: "Compare saved screen strategies without recording screen history",
		Args:  cobra.MinimumNArgs(2),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.CompareScreenStrategies(cmd.Context(), handler.CompareScreenStrategiesRequest{
				Names: args,
				AsOf:  asOf,
				TopN:  topN,
			})
		}),
	}
	cmd.Flags().StringVar(&asOf, "as-of", asOf, "override YAML pipeline strategy as_of date in YYYY-MM-DD")
	cmd.Flags().IntVar(&topN, "top", 10, "top symbol count used for overlap")
	return cmd
}

func newHistoryScreenCommand(opts *Options) *cobra.Command {
	flags := strategySourceFlags{History: 50}
	cmd := &cobra.Command{
		Use:   "screen",
		Short: "List saved screening runs",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.History(cmd.Context(), handler.ScreenHistoryRequest{Limit: flags.History})
		}),
	}
	cmd.Flags().IntVar(&flags.History, "limit", flags.History, "maximum number of screen runs to list")
	return cmd
}

func newInspectStrategyCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategy <name>",
		Short: "Inspect a saved screening strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.Inspect(cmd.Context(), handler.InspectStrategyRequest{Name: args[0]})
		}),
	}
}

func newInspectScreenCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "screen <screen-id-or-alias>",
		Short: "Inspect a saved screening run",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.InspectScreen(cmd.Context(), handler.InspectScreenRequest{Ref: args[0]})
		}),
	}
}

func newInspectScreenPipelineCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screen-pipeline <yaml>",
		Short: "Inspect a YAML screen universe pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.InspectScreenPipeline(cmd.Context(), handler.InspectScreenPipelineRequest{Path: args[0]})
		}),
	}
	mustMarkScreenPipelineYAML(cmd)
	return cmd
}

func newInspectMarketRegimeCommand(opts *Options) *cobra.Command {
	var asOf string
	cmd := &cobra.Command{
		Use:   "market-regime <yaml>",
		Short: "Inspect a YAML market regime model",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.InspectMarketRegime(cmd.Context(), handler.InspectMarketRegimeRequest{Path: args[0], AsOf: asOf})
		}),
	}
	cmd.Flags().StringVar(&asOf, "as-of", asOf, "regime calculation date in YYYY-MM-DD")
	mustMarkScreenPipelineYAML(cmd)
	return cmd
}

func newInspectStrategySetCommand(opts *Options) *cobra.Command {
	var asOf string
	cmd := &cobra.Command{
		Use:   "strategy-set <yaml>",
		Short: "Inspect a YAML strategy set route by market regime",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Strategy.InspectStrategySet(cmd.Context(), handler.InspectStrategySetRequest{Path: args[0], AsOf: asOf})
		}),
	}
	cmd.Flags().StringVar(&asOf, "as-of", asOf, "strategy set routing date in YYYY-MM-DD")
	mustMarkScreenPipelineYAML(cmd)
	return cmd
}

func mustMarkScreenPipelineYAML(cmd *cobra.Command) {
	cmd.ValidArgsFunction = cobra.FixedCompletions([]string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt)
}
