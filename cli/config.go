package cli

import (
	"fmt"
	"strings"

	appconfig "github.com/ev3rlit/mwosa/app/config"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type configInspectResult struct {
	ConfigFile   configFileInspect     `json:"config_file"`
	DatabaseFile databaseFileInspect   `json:"database_file"`
	DataDir      dataDirectoryInspect  `json:"data_directory"`
	App          appConfigInspect      `json:"app"`
	Providers    []providerInspectItem `json:"providers"`
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

func newConfigCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mwosa config file",
	}
	cmd.AddCommand(newConfigSetCommand(opts))
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
			opts.Database = resolved.DatabasePath
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
		DatabaseFile: databaseFileInspect{
			Path:   resolved.DatabasePath,
			Source: string(resolved.DatabasePathSource),
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
	return strings.HasPrefix(path, "providers.") && strings.HasSuffix(path, ".service_key")
}
