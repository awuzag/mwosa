package cli

import (
	"os"

	"github.com/awuzag/mwosa/app/handler"
	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	"github.com/spf13/cobra"
)

type aggregateFlags struct {
	File     string
	View     string
	Version  string
	SpecHash string
	Alias    string
	Params   []string
	Name     string
	Status   string
	Limit    int
}

func registerAggregateCommands(roots commandRoots, opts *Options) {
	roots.Update.AddCommand(newUpdateAggregateCommand(opts))
	roots.Validate.AddCommand(newValidateAggregateCommand(opts))
	roots.List.AddCommand(newListAggregatesCommand(opts))
	roots.Inspect.AddCommand(newInspectAggregateCommand(opts))
	roots.Inspect.AddCommand(newInspectAggregatePlanCommand(opts))
	roots.Run.AddCommand(newRunAggregateCommand(opts))
	roots.History.AddCommand(newHistoryAggregateCommand(opts))
	roots.Inspect.AddCommand(newInspectAggregateRunCommand(opts))
	roots.Delete.AddCommand(newDeleteAggregateCommand(opts))
}

func newUpdateAggregateCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{}
	cmd := &cobra.Command{
		Use:   "aggregate <name>",
		Short: "Create or update a saved Aggregate resource from YAML",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.Upsert(cmd.Context(), handler.UpsertAggregateRequest{Name: args[0], Path: flags.File})
		}),
	}
	cmd.Flags().StringVar(&flags.File, "file", flags.File, "path to an Aggregate YAML file")
	mustMarkFlagFilename(cmd, "file", "yaml", "yml")
	return cmd
}

func newValidateAggregateCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{View: "summary"}
	cmd := &cobra.Command{
		Use:   "aggregate <yaml>",
		Short: "Validate an Aggregate YAML file without executing it",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			spec, err := aggregateservice.LoadSpecFile(cmd.Context(), args[0])
			if err != nil {
				return nil, err
			}
			if err := aggregateservice.ValidateSpec(spec); err != nil {
				return nil, err
			}
			return handler.AggregateValidationOutput{Spec: spec, Valid: true, View: flags.View}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.View, "view", flags.View, "validation view: summary or raw")
	return cmd
}

func newListAggregatesCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "aggregates",
		Short: "List saved Aggregate resources",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.List(cmd.Context(), handler.ListAggregatesRequest{})
		}),
	}
}

func newInspectAggregateCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{View: "summary"}
	cmd := &cobra.Command{
		Use:   "aggregate <name>",
		Short: "Inspect a saved Aggregate resource",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.Inspect(cmd.Context(), handler.InspectAggregateRequest{Name: args[0], Version: flags.Version, SpecHash: flags.SpecHash, View: flags.View})
		}),
	}
	addAggregateVersionFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.View, "view", flags.View, "inspect view: summary, versions, or raw")
	return cmd
}

func newInspectAggregatePlanCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{View: "summary"}
	cmd := &cobra.Command{
		Use:   "aggregate-plan <name|yaml>",
		Short: "Inspect an Aggregate execution plan without running stages",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			req := handler.PlanAggregateRequest{Ref: args[0], Version: flags.Version, SpecHash: flags.SpecHash, Params: flags.Params, View: flags.View}
			if info, statErr := os.Stat(args[0]); statErr == nil && !info.IsDir() {
				req.Path = args[0]
			}
			return runtime.Handlers.Aggregates.Plan(cmd.Context(), req)
		}),
	}
	addAggregateVersionFlags(cmd, &flags)
	cmd.Flags().StringArrayVar(&flags.Params, "param", flags.Params, "runtime parameter override, key=value")
	cmd.Flags().StringVar(&flags.View, "view", flags.View, "plan view: summary, stages, pipeline, or raw")
	return cmd
}

func newRunAggregateCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{}
	cmd := &cobra.Command{
		Use:   "aggregate <name>",
		Short: "Run a saved Aggregate resource",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, true)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.Run(cmd.Context(), handler.RunAggregateRequest{Name: args[0], Alias: flags.Alias, Version: flags.Version, SpecHash: flags.SpecHash, Params: flags.Params})
		}),
	}
	addAggregateVersionFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.Alias, "alias", flags.Alias, "optional aggregate run alias")
	cmd.Flags().StringArrayVar(&flags.Params, "param", flags.Params, "runtime parameter override, key=value")
	return cmd
}

func newHistoryAggregateCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "aggregate",
		Short: "List Aggregate run history",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.History(cmd.Context(), handler.AggregateHistoryRequest{Name: flags.Name, Status: aggregateservice.RunStatus(flags.Status), Limit: flags.Limit})
		}),
	}
	cmd.Flags().StringVar(&flags.Name, "name", flags.Name, "aggregate name")
	cmd.Flags().StringVar(&flags.Status, "status", flags.Status, "run status: succeeded or failed")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum runs to list")
	return cmd
}

func newInspectAggregateRunCommand(opts *Options) *cobra.Command {
	flags := aggregateFlags{View: "summary", Limit: 50}
	cmd := &cobra.Command{
		Use:   "aggregate-run <id|alias>",
		Short: "Inspect an Aggregate run",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.InspectRun(cmd.Context(), handler.InspectAggregateRunRequest{Ref: args[0], Limit: flags.Limit, View: flags.View})
		}),
	}
	cmd.Flags().StringVar(&flags.View, "view", flags.View, "run view: summary, stages, params, pipeline, items, or raw")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum items to include")
	return cmd
}

func newDeleteAggregateCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "aggregate <name>",
		Short: "Soft delete a saved Aggregate resource",
		Args:  cobra.ExactArgs(1),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (any, error) {
			runtime, err := newAppRuntime(cmd.Context(), opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(cmd.Context(), runtime, &err)
			return runtime.Handlers.Aggregates.Delete(cmd.Context(), handler.DeleteAggregateRequest{Name: args[0]})
		}),
	}
}

func addAggregateVersionFlags(cmd *cobra.Command, flags *aggregateFlags) {
	cmd.Flags().StringVar(&flags.Version, "version", flags.Version, "aggregate version number or latest")
	cmd.Flags().StringVar(&flags.SpecHash, "spec-hash", flags.SpecHash, "aggregate spec hash")
}
