package cli

import (
	"fmt"
	"io"
	"runtime"

	appconfig "github.com/awuzag/mwosa/app/config"
	"github.com/awuzag/mwosa/providers/builtin"
	provider "github.com/awuzag/mwosa/providers/core"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

const (
	defaultVersion = "dev"
	schemaVersion  = "dev"
)

type BuildInfo struct {
	Version     string
	Commit      string
	Date        string
	Development bool
}

type Options struct {
	// 선택. 비어 있으면 MWOSA_CONFIG 또는 OS 기본 config 경로를 따른다.
	Config string

	// 필수. 명령 결과를 출력할 형식이다.
	Output OutputMode

	// 선택. 비어 있으면 provider router 가 요청에 맞는 provider 를 고른다.
	Provider string

	// 선택. 비어 있으면 provider router 의 기본 우선순위를 따른다.
	PreferProvider string

	// 필수. provider routing 과 storage query 에 사용할 시장 ID 다.
	Market string

	// 선택. 비어 있으면 config/env/default 기준으로 database backend 를 결정한다.
	DatabaseBackend string

	// 선택. SQLite database path 다. MongoDB/PostgreSQL backend 에서는 sidecar cache 기준 경로로만 쓴다.
	Database string

	// 선택. MongoDB/PostgreSQL backend 에 사용할 database URL 이다.
	DatabaseURL string

	ProviderAuthDatabase string
	ProviderConfig       provider.Config
	ConfigState          appconfig.Resolved
	configLoaded         bool
	Development          bool
}

func (opts Options) Validate() error {
	err := validation.ValidateStruct(&opts,
		validation.Field(&opts.Output, validation.Required, validation.By(validateOutputMode)),
		validation.Field(&opts.Provider),
		validation.Field(&opts.PreferProvider),
		validation.Field(&opts.Market, validation.Required),
	)
	if err != nil {
		return err
	}
	switch opts.DatabaseBackend {
	case "":
		if opts.DatabaseURL != "" {
			return nil
		}
		if opts.Database != "" {
			return nil
		}
		return oops.In("cli").New("DatabaseURL is required for mongodb backend")
	case appconfig.DatabaseBackendSQLite:
		if opts.Database == "" {
			return oops.In("cli").New("Database is required for sqlite backend")
		}
	case appconfig.DatabaseBackendPostgres:
		if opts.DatabaseURL == "" {
			return oops.In("cli").New("DatabaseURL is required for postgres backend")
		}
	case appconfig.DatabaseBackendMongoDB:
		if opts.DatabaseURL == "" {
			return oops.In("cli").New("DatabaseURL is required for mongodb backend")
		}
	default:
		return oops.In("cli").With("backend", opts.DatabaseBackend).New("unsupported database backend")
	}
	return nil
}

func validateOutputMode(value any) error {
	mode, ok := value.(OutputMode)
	if !ok {
		return oops.In("cli").New("output mode has invalid type")
	}
	_, err := ParseOutputMode(string(mode))
	return err
}

func NewRootCommand(build BuildInfo) *cobra.Command {
	opts := Options{
		Output:      DefaultOutputMode,
		Market:      string(provider.MarketKRX),
		Development: build.Development,
	}

	cmd := &cobra.Command{
		Use:           "mwosa",
		Short:         "Investment research CLI for provider-backed market data",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if skipConfigLoadForCompletion(cmd) || skipConfigLoadForConfigMutation(cmd) {
				return nil
			}
			return loadConfig(&opts)
		},
	}

	cmd.PersistentFlags().StringVar(
		&opts.Config,
		"config",
		opts.Config,
		"config file path",
	)
	cmd.PersistentFlags().VarP(
		&opts.Output,
		"output",
		"o",
		OutputModeHelp(),
	)
	cmd.PersistentFlags().StringVar(
		&opts.Provider,
		"provider",
		opts.Provider,
		"force a provider by id",
	)
	cmd.PersistentFlags().StringVar(
		&opts.PreferProvider,
		"prefer-provider",
		opts.PreferProvider,
		"prefer a provider when multiple candidates match",
	)
	cmd.PersistentFlags().StringVar(
		&opts.Market,
		"market",
		opts.Market,
		"market id",
	)
	cmd.PersistentFlags().StringVar(
		&opts.DatabaseBackend,
		"database-backend",
		opts.DatabaseBackend,
		"database backend: mongodb, sqlite, or postgres",
	)
	cmd.PersistentFlags().StringVar(
		&opts.Database,
		"database",
		opts.Database,
		"SQLite database path",
	)
	cmd.PersistentFlags().StringVar(
		&opts.DatabaseURL,
		"database-url",
		opts.DatabaseURL,
		"MongoDB/PostgreSQL database URL",
	)
	registerRootCompletions(cmd)

	initCommand := newInitCommand()
	inspectCommand := newInspectCommand()
	listCommand := newListCommand()
	searchCommand := newSearchCommand()
	createCommand := newCreateCommand()
	updateCommand := newUpdateCommand()
	deleteCommand := newDeleteCommand()
	screenCommand := newScreenCommand()
	historyCommand := newHistoryCommand()
	fetchCommand := newFetchCommand()
	runCommand := newRunCommand()
	compareCommand := newCompareCommand()
	rankCommand := newRankCommand()
	calcCommand := newCalcCommand()
	migrateCommand := newMigrateCommand()
	getCommand := newGetCommand()
	ensureCommand := newEnsureCommand()
	syncCommand := newSyncCommand()
	backfillCommand := newBackfillCommand()
	loginCommand := newLoginCommand()
	logoutCommand := newLogoutCommand()
	validateCommand := newValidateCommand()
	doctorCommand := newDoctorCommand()
	enableCommand := newEnableCommand()
	disableCommand := newDisableCommand()
	preferCommand := newPreferCommand()
	roots := commandRoots{
		Init:     initCommand,
		Inspect:  inspectCommand,
		List:     listCommand,
		Search:   searchCommand,
		Create:   createCommand,
		Update:   updateCommand,
		Delete:   deleteCommand,
		Screen:   screenCommand,
		History:  historyCommand,
		Fetch:    fetchCommand,
		Run:      runCommand,
		Compare:  compareCommand,
		Rank:     rankCommand,
		Calc:     calcCommand,
		Migrate:  migrateCommand,
		Get:      getCommand,
		Ensure:   ensureCommand,
		Sync:     syncCommand,
		Backfill: backfillCommand,
		Login:    loginCommand,
		Logout:   logoutCommand,
		Validate: validateCommand,
		Doctor:   doctorCommand,
		Enable:   enableCommand,
		Disable:  disableCommand,
		Prefer:   preferCommand,
	}
	registerConfigCommands(roots, &opts)
	registerDailyCommands(roots, &opts)
	registerIndexCommands(roots, &opts)
	registerMacroCommands(roots, &opts)
	registerFinancialsCommands(roots, &opts)
	registerInstrumentCommands(roots, &opts)
	registerStockCommands(roots, &opts)
	registerMarketDataCommands(roots, &opts)
	registerQuoteCommands(roots, &opts)
	registerAggregateCommands(roots, &opts)
	registerStrategyCommands(roots, &opts)
	registerBacktestCommands(roots, &opts)
	registerProviderCommands(roots, &opts)
	registerProviderRawCommands(roots, &opts)
	registerMigrationCommands(roots, &opts)
	registerKRXCommands(roots, &opts)
	registerOpenDARTCommands(roots, &opts)
	registerStorageCommands(roots, &opts)

	cmd.AddCommand(newCompletionCommand())
	cmd.AddCommand(newVersionCommand(build))
	cmd.AddCommand(initCommand)
	cmd.AddCommand(inspectCommand)
	cmd.AddCommand(newConfigCommand(&opts))
	cmd.AddCommand(createCommand)
	cmd.AddCommand(listCommand)
	cmd.AddCommand(searchCommand)
	cmd.AddCommand(updateCommand)
	cmd.AddCommand(deleteCommand)
	cmd.AddCommand(screenCommand)
	cmd.AddCommand(historyCommand)
	cmd.AddCommand(fetchCommand)
	cmd.AddCommand(runCommand)
	cmd.AddCommand(compareCommand)
	cmd.AddCommand(rankCommand)
	cmd.AddCommand(calcCommand)
	cmd.AddCommand(migrateCommand)
	cmd.AddCommand(loginCommand)
	cmd.AddCommand(logoutCommand)
	cmd.AddCommand(validateCommand)
	cmd.AddCommand(doctorCommand)
	cmd.AddCommand(enableCommand)
	cmd.AddCommand(disableCommand)
	cmd.AddCommand(preferCommand)
	cmd.AddCommand(getCommand)
	cmd.AddCommand(ensureCommand)
	cmd.AddCommand(syncCommand)
	cmd.AddCommand(backfillCommand)

	return cmd
}

func skipConfigLoadForConfigMutation(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.CommandPath() {
	case "mwosa config set", "mwosa config use-database":
		return true
	default:
		return false
	}
}

func loadConfig(opts *Options) error {
	if opts == nil {
		return oops.In("cli").New("cli options are nil")
	}
	if opts.configLoaded {
		return nil
	}
	resolved, err := appconfig.LoadOrCreate(appconfig.Options{
		ConfigPath:       opts.Config,
		DatabaseBackend:  opts.DatabaseBackend,
		DatabasePath:     opts.Database,
		DatabaseURL:      opts.DatabaseURL,
		Market:           opts.Market,
		Development:      opts.Development,
		ProviderDefaults: providerDefaults(),
	})
	if err != nil {
		return oops.In("cli").Wrapf(err, "load config")
	}
	opts.Config = resolved.ConfigPath
	opts.DatabaseBackend = resolved.DatabaseBackend
	opts.Database = resolved.DatabasePath
	opts.DatabaseURL = resolved.DatabaseURL
	opts.ProviderAuthDatabase = resolved.ProviderAuthDatabasePath
	if opts.PreferProvider == "" {
		opts.PreferProvider = resolved.File.App.PreferredProvider
	}
	opts.ProviderConfig = resolved.ProviderConfig
	opts.ConfigState = resolved
	opts.configLoaded = true
	return nil
}

func providerDefaults() []appconfig.ProviderDefault {
	builders := builtin.Builders()
	defaults := make([]appconfig.ProviderDefault, 0, len(builders))
	for _, builder := range builders {
		defaults = append(defaults, builder)
	}
	return defaults
}

func newVersionCommand(build BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print mwosa build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			writeVersion(cmd.OutOrStdout(), normalizeBuildInfo(build))
			return nil
		},
	}
}

func normalizeBuildInfo(build BuildInfo) BuildInfo {
	if build.Version == "" {
		build.Version = defaultVersion
	}
	if build.Commit == "" {
		build.Commit = "unknown"
	}
	if build.Date == "" {
		build.Date = "unknown"
	}
	return build
}

func writeVersion(w io.Writer, build BuildInfo) {
	_, _ = fmt.Fprintf(w, "mwosa %s\n", build.Version)
	_, _ = fmt.Fprintf(w, "schema %s\n", schemaVersion)
	_, _ = fmt.Fprintf(w, "commit %s\n", build.Commit)
	_, _ = fmt.Fprintf(w, "built %s\n", build.Date)
	_, _ = fmt.Fprintf(w, "go %s\n", runtime.Version())
}
