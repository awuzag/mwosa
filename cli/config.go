package cli

import (
	"fmt"
	"net/url"
	"strings"

	appconfig "github.com/awuzag/mwosa/app/config"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type configInspectResult struct {
	ConfigFile           configFileInspect     `json:"config_file"`
	Database             databaseInspect       `json:"database"`
	DatabaseFile         databaseFileInspect   `json:"database_file"`
	ProviderAuthDatabase providerAuthInspect   `json:"provider_auth_database"`
	DataDir              dataDirectoryInspect  `json:"data_directory"`
	App                  appConfigInspect      `json:"app"`
	Providers            []providerInspectItem `json:"providers"`
}

type configFileInspect struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Exists  bool   `json:"exists"`
	Created bool   `json:"created"`
}

type databaseFileInspect struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type databaseInspect struct {
	Backend       string              `json:"backend"`
	BackendSource string              `json:"backend_source"`
	Path          databasePathInspect `json:"path"`
	URL           databaseURLInspect  `json:"url"`
}

type databasePathInspect struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

type databaseURLInspect struct {
	Value      string `json:"value,omitempty"`
	Source     string `json:"source"`
	URLEnv     string `json:"url_env,omitempty"`
	Configured bool   `json:"configured"`
}

type providerAuthInspect struct {
	Path string `json:"path"`
}

type dataDirectoryInspect struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type appConfigInspect struct {
	Market            string `json:"market"`
	PreferredProvider string `json:"preferred_provider,omitempty"`
}

type providerInspectItem struct {
	ID      string              `json:"id"`
	Enabled bool                `json:"enabled"`
	Groups  []providerGroupItem `json:"groups"`
	Auth    map[string]bool     `json:"auth"`
}

type providerGroupItem struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type configSetResult struct {
	ConfigFile string `json:"config_file"`
	Setting    string `json:"setting"`
	Value      string `json:"value"`
}

type configUseDatabaseResult struct {
	ConfigFile string `json:"config_file"`
	Backend    string `json:"backend"`
	Path       string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
	URLEnv     string `json:"url_env,omitempty"`
}

func newConfigCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mwosa config file",
	}
	cmd.AddCommand(newConfigSetCommand(opts))
	cmd.AddCommand(newConfigUseDatabaseCommand(opts))
	return cmd
}

func registerConfigCommands(roots commandRoots, opts *Options) {
	roots.Inspect.AddCommand(newInspectConfigCommand(opts))
}

func newConfigSetCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "set <path> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			resolved, err := appconfig.SetValue(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, args[0], args[1])
			if err != nil {
				return nil, oops.In("cli").Wrapf(err, "set config")
			}
			opts.Config = resolved.ConfigPath
			opts.DatabaseBackend = resolved.DatabaseBackend
			opts.Database = resolved.DatabasePath
			opts.DatabaseURL = resolved.DatabaseURL
			opts.ProviderAuthDatabase = resolved.ProviderAuthDatabasePath
			opts.ProviderConfig = resolved.ProviderConfig
			opts.ConfigState = resolved
			opts.configLoaded = true
			return configSetResult{
				ConfigFile: resolved.ConfigPath,
				Setting:    args[0],
				Value:      maskedConfigSetValue(args[0], args[1]),
			}, nil
		}),
	}
}

func newConfigUseDatabaseCommand(opts *Options) *cobra.Command {
	flags := struct {
		Path   string
		URL    string
		URLEnv string
	}{}
	cmd := &cobra.Command{
		Use:   "use-database <sqlite|postgres>",
		Short: "Select the database backend",
		Args:  cobra.ExactArgs(1),
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			backend := strings.TrimSpace(args[0])
			if strings.TrimSpace(flags.URL) != "" && strings.TrimSpace(flags.URLEnv) != "" {
				return nil, oops.In("cli").New("config use-database accepts only one of --url or --url-env")
			}
			resolved, err := appconfig.UseDatabase(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, appconfig.DatabaseConfig{
				Backend: backend,
				Path:    flags.Path,
				URL:     flags.URL,
				URLEnv:  flags.URLEnv,
			})
			if err != nil {
				return nil, oops.In("cli").Wrapf(err, "select database backend")
			}
			opts.Config = resolved.ConfigPath
			opts.DatabaseBackend = resolved.DatabaseBackend
			opts.Database = resolved.DatabasePath
			opts.DatabaseURL = resolved.DatabaseURL
			opts.ProviderAuthDatabase = resolved.ProviderAuthDatabasePath
			opts.ProviderConfig = resolved.ProviderConfig
			opts.ConfigState = resolved
			opts.configLoaded = true
			return configUseDatabaseResult{
				ConfigFile: resolved.ConfigPath,
				Backend:    resolved.DatabaseBackend,
				Path:       pathForBackend(resolved),
				URL:        urlForBackend(resolved),
				URLEnv:     urlEnvForBackend(resolved),
			}, nil
		}),
	}
	cmd.Flags().StringVar(&flags.Path, "path", flags.Path, "SQLite database path")
	cmd.Flags().StringVar(&flags.URL, "url", flags.URL, "PostgreSQL database URL")
	cmd.Flags().StringVar(&flags.URLEnv, "url-env", flags.URLEnv, "environment variable that contains the PostgreSQL database URL")
	return cmd
}

func newInspectConfigCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Inspect resolved config and data paths",
		Args:  cobra.NoArgs,
		RunE: runJSONResult(func(cmd *cobra.Command, _ []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			return configInspectFromResolved(opts.ConfigState), nil
		}),
	}
}

func configInspectFromResolved(resolved appconfig.Resolved) configInspectResult {
	result := configInspectResult{
		ConfigFile: configFileInspect{
			Path:    resolved.ConfigPath,
			Source:  string(resolved.ConfigPathSource),
			Exists:  resolved.ConfigFileExists,
			Created: resolved.ConfigFileCreated,
		},
		Database: databaseInspect{
			Backend:       resolved.DatabaseBackend,
			BackendSource: string(resolved.DatabaseBackendSource),
			Path: databasePathInspect{
				Value:  resolved.DatabasePath,
				Source: string(resolved.DatabasePathSource),
			},
			URL: databaseURLInspect{
				Value:      maskedDatabaseURL(resolved.DatabaseURL),
				Source:     string(resolved.DatabaseURLSource),
				URLEnv:     resolved.DatabaseURLEnv,
				Configured: strings.TrimSpace(resolved.DatabaseURL) != "",
			},
		},
		DatabaseFile: databaseFileInspect{
			Path:   resolved.DatabasePath,
			Source: string(resolved.DatabasePathSource),
		},
		ProviderAuthDatabase: providerAuthInspect{
			Path: resolved.ProviderAuthDatabasePath,
		},
		DataDir: dataDirectoryInspect{
			Path:   resolved.DataDirectory,
			Exists: resolved.DataDirectoryExists,
		},
		App: appConfigInspect{
			Market:            resolved.File.App.Market,
			PreferredProvider: resolved.File.App.PreferredProvider,
		},
	}
	for _, item := range resolved.File.Providers {
		result.Providers = append(result.Providers, providerInspectFromConfig(item))
	}
	return result
}

func providerInspectFromConfig(config provider.Config) providerInspectItem {
	enabled, ok := config.Bool("enabled")
	if !ok {
		enabled = true
	}
	item := providerInspectItem{
		ID:      config.String("id"),
		Enabled: enabled,
		Auth:    map[string]bool{},
	}
	if auth, ok := config.Lookup("auth"); ok {
		if values, ok := auth.(map[string]any); ok {
			for key, value := range values {
				item.Auth[key] = fmt.Sprint(value) != ""
			}
		}
	}
	if groups, ok := config.Lookup("groups"); ok {
		if values, ok := groups.(map[string]any); ok {
			for key, value := range values {
				group := providerGroupItem{ID: key, Enabled: true}
				if config, ok := value.(map[string]any); ok {
					if enabled, ok := provider.Config(config).Bool("enabled"); ok {
						group.Enabled = enabled
					}
				}
				item.Groups = append(item.Groups, group)
			}
		}
	}
	return item
}

func maskedConfigSetValue(path string, value string) string {
	if isSecretConfigPath(path) && value != "" {
		return "<configured>"
	}
	return value
}

func isSecretConfigPath(path string) bool {
	return path == "app.database.url" || strings.HasPrefix(path, "providers.") && strings.Contains(path, ".auth.")
}

func pathForBackend(resolved appconfig.Resolved) string {
	if resolved.DatabaseBackend == appconfig.DatabaseBackendSQLite {
		return resolved.DatabasePath
	}
	return ""
}

func urlForBackend(resolved appconfig.Resolved) string {
	if resolved.DatabaseBackend == appconfig.DatabaseBackendPostgres {
		return maskedDatabaseURL(resolved.DatabaseURL)
	}
	return ""
}

func urlEnvForBackend(resolved appconfig.Resolved) string {
	if resolved.DatabaseBackend == appconfig.DatabaseBackendPostgres {
		return resolved.DatabaseURLEnv
	}
	return ""
}

func maskedDatabaseURL(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return "<configured>"
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if _, ok := parsed.User.Password(); ok {
			parsed.User = url.UserPassword(username, "xxxxx")
		}
	}
	return parsed.String()
}
