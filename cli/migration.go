package cli

import "github.com/spf13/cobra"

func registerMigrationCommands(roots commandRoots, opts *Options) {
	roots.Migrate.AddCommand(newMigrateStatusCommand(opts))
	roots.Migrate.AddCommand(newMigrateApplyCommand(opts))
}

func newMigrateStatusCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List local data migration status",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Migration.Status(cmd.Context())
		}),
	}
}

func newMigrateApplyCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Apply pending local data migrations",
		Args:  cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			runtime, err := newAppRuntime(opts, false)
			if err != nil {
				return nil, err
			}
			defer closeAppRuntime(runtime, &err)

			return runtime.Handlers.Migration.Apply(cmd.Context())
		}),
	}
}
