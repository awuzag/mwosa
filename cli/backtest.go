package cli

import (
	"github.com/ev3rlit/mwosa/app/handler"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

func registerBacktestCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListBacktestCommand(opts))
	roots.List.AddCommand(newListEvaluationsCommand(opts))
	roots.Inspect.AddCommand(newInspectBacktestCommand(opts))
	roots.Inspect.AddCommand(newInspectBacktestUniverseCommand(opts))
	roots.Inspect.AddCommand(newInspectEvaluationCommand(opts))
	roots.Update.AddCommand(newUpdateBacktestCommand(opts))
	roots.Delete.AddCommand(newDeleteBacktestCommand(opts))
	roots.Validate.AddCommand(newValidateBacktestCommand(opts))
	roots.Validate.AddCommand(newValidateEvaluationCommand(opts))
	roots.Run.AddCommand(newRunBacktestCommand(opts))
	roots.Run.AddCommand(newRunEvaluationCommand(opts))
	roots.Compare.AddCommand(newCompareEvaluationCommand(opts))
	roots.Rank.AddCommand(newRankEvaluationCommand(opts))
}

func newListBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "List backtest resources",
	}
	cmd.AddCommand(newListBacktestStrategiesCommand(opts))
	return cmd
}

func newListEvaluationsCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "evaluations",
		Short: "List saved backtest evaluations",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.ListEvaluations(cmd.Context(), handler.ListEvaluationsRequest{})
		}),
	}
}

func newListBacktestStrategiesCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategies",
		Short: "List saved backtest strategies",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.ListStrategies(cmd.Context(), handler.ListBacktestStrategiesRequest{})
		}),
	}
}

func newInspectBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "Inspect backtest resources",
	}
	cmd.AddCommand(newInspectBacktestStrategyCommand(opts))
	cmd.AddCommand(newInspectBacktestUniverseNestedCommand(opts))
	return cmd
}

func newInspectEvaluationCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "evaluation <name|id>",
		Short: "Inspect a saved backtest evaluation",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.InspectEvaluation(cmd.Context(), handler.InspectEvaluationRequest{Ref: args[0]})
		}),
	}
}

func newInspectBacktestStrategyCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategy <name>",
		Short: "Inspect a saved backtest strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.InspectStrategy(cmd.Context(), handler.InspectBacktestStrategyRequest{Name: args[0]})
		}),
	}
}

func newInspectBacktestUniverseNestedCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "universe <yaml>",
		Short: "Inspect a YAML backtest universe pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.InspectUniverse(cmd.Context(), handler.InspectBacktestUniverseRequest{Path: args[0]})
		}),
	}
	mustMarkBacktestYAML(cmd)
	return cmd
}

func newInspectBacktestUniverseCommand(opts *Options) *cobra.Command {
	cmd := newInspectBacktestUniverseNestedCommand(opts)
	cmd.Use = "backtest-universe <yaml>"
	return cmd
}

func newUpdateBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "Update backtest resources",
	}
	cmd.AddCommand(newUpdateBacktestStrategyCommand(opts))
	return cmd
}

func newUpdateBacktestStrategyCommand(opts *Options) *cobra.Command {
	var yamlFile string
	cmd := &cobra.Command{
		Use:   "strategy <name>",
		Short: "Create or update a saved backtest strategy from YAML",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			if yamlFile == "" {
				return nil, oops.In("cli_backtest").New("--yaml-file is required")
			}
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.UpdateStrategy(cmd.Context(), handler.UpdateBacktestStrategyRequest{
				Name:     args[0],
				YAMLPath: yamlFile,
			})
		}),
	}
	cmd.Flags().StringVar(&yamlFile, "yaml-file", "", "YAML file containing a Strategy document")
	return cmd
}

func newDeleteBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "Delete backtest resources",
	}
	cmd.AddCommand(newDeleteBacktestStrategyCommand(opts))
	return cmd
}

func newDeleteBacktestStrategyCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "strategy <name>",
		Short: "Soft delete a saved backtest strategy",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.DeleteStrategy(cmd.Context(), handler.DeleteBacktestStrategyRequest{Name: args[0]})
		}),
	}
}

func newValidateBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest <yaml>",
		Short: "Validate a YAML backtest strategy and run spec",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.Validate(cmd.Context(), handler.ValidateBacktestRequest{Path: args[0]})
		}),
	}
	mustMarkBacktestYAML(cmd)
	return cmd
}

func newValidateEvaluationCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evaluation <yaml>",
		Short: "Validate a YAML backtest evaluation spec",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.ValidateEvaluation(cmd.Context(), handler.ValidateEvaluationRequest{Path: args[0]})
		}),
	}
	mustMarkBacktestYAML(cmd)
	return cmd
}

func newRunBacktestCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest <yaml>",
		Short: "Run a YAML backtest against stored canonical daily bars",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.Run(cmd.Context(), handler.RunBacktestRequest{Path: args[0]})
		}),
	}
	mustMarkBacktestYAML(cmd)
	return cmd
}

func newRunEvaluationCommand(opts *Options) *cobra.Command {
	var parallelism int
	cmd := &cobra.Command{
		Use:   "evaluation <yaml>",
		Short: "Run a YAML backtest evaluation against stored canonical daily bars",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.RunEvaluation(cmd.Context(), handler.RunEvaluationRequest{Path: args[0], Parallelism: parallelism})
		}),
	}
	cmd.Flags().IntVar(&parallelism, "parallelism", 0, "bounded worker count for evaluation cases; overrides YAML execution.parallelism when positive")
	mustMarkBacktestYAML(cmd)
	return cmd
}

func newCompareEvaluationCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "evaluation <name|id>",
		Short: "Compare saved backtest evaluation cases",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.CompareEvaluation(cmd.Context(), handler.CompareEvaluationRequest{Ref: args[0]})
		}),
	}
}

func newRankEvaluationCommand(opts *Options) *cobra.Command {
	var objective string
	cmd := &cobra.Command{
		Use:   "evaluation <name|id>",
		Short: "Rank saved backtest evaluation cases",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Backtest.RankEvaluation(cmd.Context(), handler.RankEvaluationRequest{Ref: args[0], Objective: objective})
		}),
	}
	cmd.Flags().StringVar(&objective, "objective", "calmar", "metric objective for ranking")
	return cmd
}

func mustMarkBacktestYAML(cmd *cobra.Command) {
	cmd.ValidArgsFunction = cobra.FixedCompletions([]string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt)
}
