package cli

import "github.com/spf13/cobra"

type commandRoots struct {
	Inspect  *cobra.Command
	List     *cobra.Command
	Search   *cobra.Command
	Create   *cobra.Command
	Update   *cobra.Command
	Delete   *cobra.Command
	Screen   *cobra.Command
	History  *cobra.Command
	Run      *cobra.Command
	Compare  *cobra.Command
	Rank     *cobra.Command
	Calc     *cobra.Command
	Migrate  *cobra.Command
	Get      *cobra.Command
	Ensure   *cobra.Command
	Sync     *cobra.Command
	Backfill *cobra.Command
	Login    *cobra.Command
	Logout   *cobra.Command
	Validate *cobra.Command
	Doctor   *cobra.Command
	Enable   *cobra.Command
	Disable  *cobra.Command
	Prefer   *cobra.Command
}

func newInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect mwosa resources and local state",
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List mwosa resources",
	}
}

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search",
		Short: "Search local and provider-backed resources",
	}
}

func newCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create mwosa resources",
	}
}

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update mwosa resources",
	}
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete mwosa resources",
	}
}

func newScreenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "screen",
		Short: "Run screening workflows",
	}
}

func newHistoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List mwosa execution history",
	}
}

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run mwosa workflows",
	}
}

func newCompareCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "compare",
		Short: "Compare mwosa resources",
	}
}

func newRankCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rank",
		Short: "Rank mwosa resources",
	}
}

func newCalcCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "calc",
		Short: "Calculate derived mwosa resources",
	}
}

func newMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Manage local data migrations",
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Read source-like data from local storage",
	}
}

func newEnsureCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ensure",
		Short: "Fetch missing data and store it locally",
	}
}

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Refresh provider-backed data batches",
	}
}

func newBackfillCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Collect historical data ranges",
	}
}

func newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Register credentials for a resource",
	}
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials for a resource",
	}
}

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate local configuration and resources",
	}
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose local configuration and resources",
	}
}

func newEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable a resource",
	}
}

func newDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable a resource",
	}
}

func newPreferCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prefer",
		Short: "Set resource preference",
	}
}
