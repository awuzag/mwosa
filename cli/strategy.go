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
	roots.Delete.AddCommand(newDeleteStrategyCommand(opts))
	roots.Screen.AddCommand(newScreenETFCommand(opts))
	roots.Screen.AddCommand(newScreenStrategyCommand(opts))
	roots.Screen.AddCommand(newScreenPipelineCommand(opts))
	roots.History.AddCommand(newHistoryScreenCommand(opts))
	roots.Inspect.AddCommand(newInspectStrategyCommand(opts))
	roots.Inspect.AddCommand(newInspectScreenCommand(opts))
	roots.Inspect.AddCommand(newInspectScreenPipelineCommand(opts))
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
				Name:  args[0],
				Alias: flags.Alias,
			})
		}),
	}
	cmd.Flags().StringVar(&flags.Alias, "alias", flags.Alias, "optional screen run alias")
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

func mustMarkScreenPipelineYAML(cmd *cobra.Command) {
	cmd.ValidArgsFunction = cobra.FixedCompletions([]string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt)
}
