package cli

import (
	"fmt"
	"strings"

	appconfig "github.com/ev3rlit/mwosa/app/config"
	"github.com/ev3rlit/mwosa/providers/builtin"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type providerLoginResult struct {
	ConfigFile string                     `json:"config_file"`
	Provider   string                     `json:"provider"`
	Enabled    bool                       `json:"enabled"`
	Fields     []providerLoginFieldResult `json:"fields"`
}

type providerLoginFieldResult struct {
	Path       string `json:"path"`
	Configured bool   `json:"configured"`
	Secret     bool   `json:"secret"`
}

type providerActionResult struct {
	ConfigFile string `json:"config_file"`
	Provider   string `json:"provider"`
	Action     string `json:"action"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Preferred  bool   `json:"preferred,omitempty"`
}

type providerDoctorResult struct {
	ConfigFile string                   `json:"config_file"`
	Providers  []providerDoctorProvider `json:"providers"`
}

type providerDoctorProvider struct {
	ID      string                `json:"id"`
	Enabled bool                  `json:"enabled"`
	Status  string                `json:"status"`
	Fields  []providerDoctorField `json:"fields"`
	Issues  []providerDoctorIssue `json:"issues"`
}

type providerDoctorField struct {
	Path       string `json:"path"`
	Required   bool   `json:"required"`
	Secret     bool   `json:"secret"`
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"`
}

type providerDoctorIssue struct {
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

func registerProviderCommands(roots commandRoots, opts *Options) {
	roots.List.AddCommand(newListProvidersCommand(opts))
	roots.Inspect.AddCommand(newInspectProviderCommand(opts))
	roots.Login.AddCommand(newLoginProviderCommand(opts))
	roots.Logout.AddCommand(newLogoutProviderCommand(opts))
	roots.Validate.AddCommand(newValidateProviderCommand(opts))
	roots.Doctor.AddCommand(newDoctorProviderCommand(opts))
	roots.Enable.AddCommand(newEnableProviderCommand(opts))
	roots.Disable.AddCommand(newDisableProviderCommand(opts))
	roots.Prefer.AddCommand(newPreferProviderCommand(opts))
}

func newListProvidersCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List configured and available providers",
		Args:  cobra.NoArgs,
		RunE: runJSONResult(func(cmd *cobra.Command, _ []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			return providerDoctorResult{
				ConfigFile: opts.ConfigState.ConfigPath,
				Providers:  doctorProviders(builtin.Builders(), opts.ProviderConfig),
			}, nil
		}),
	}
}

func newInspectProviderCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider <name>",
		Short:             "Inspect provider configuration and readiness",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			builder, err := providerBuilder(provider.ProviderID(args[0]))
			if err != nil {
				return nil, err
			}
			return providerDoctorResult{
				ConfigFile: opts.ConfigState.ConfigPath,
				Providers:  []providerDoctorProvider{doctorProvider(builder, opts.ProviderConfig)},
			}, nil
		}),
	}
}

func newLoginProviderCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Register provider credentials",
	}
	for _, builder := range builtin.Builders() {
		cmd.AddCommand(newLoginProviderIDCommand(opts, builder))
	}
	return cmd
}

func newLoginProviderIDCommand(opts *Options, builder provider.ProviderBuilder) *cobra.Command {
	spec := builder.ConfigSpec()
	values := make(map[string]*string, len(spec.Fields))
	cmd := &cobra.Command{
		Use:   string(builder.ID()),
		Short: fmt.Sprintf("Register %s provider credentials", builder.ID()),
		Args:  cobra.NoArgs,
		RunE: runJSONResult(func(cmd *cobra.Command, _ []string) (any, error) {
			updates := map[string]string{
				providerSettingPath(builder.ID(), "enabled"): "true",
			}
			fields := make([]providerLoginFieldResult, 0, len(spec.Fields))
			for _, field := range spec.Fields {
				value, hasFlag := values[field.Path]
				rawValue := ""
				flagChanged := false
				if hasFlag {
					rawValue = strings.TrimSpace(*value)
					flagChanged = cmd.Flags().Changed(field.Flag)
				}
				if field.Required && rawValue == "" {
					return nil, oops.In("cli").
						With("provider", builder.ID(), "setting", field.Path, "flag", field.Flag).
						Errorf("provider %s requires --%s", builder.ID(), field.Flag)
				}
				if hasFlag && (rawValue != "" || flagChanged) {
					updates[providerSettingPath(builder.ID(), field.Path)] = rawValue
				}
				fields = append(fields, providerLoginFieldResult{
					Path:       field.Path,
					Configured: rawValue != "",
					Secret:     field.Secret,
				})
			}
			resolved, err := appconfig.SetValues(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, updates)
			if err != nil {
				return nil, oops.In("cli").With("provider", builder.ID()).Wrapf(err, "login provider")
			}
			applyResolvedConfig(opts, resolved)
			return providerLoginResult{
				ConfigFile: resolved.ConfigPath,
				Provider:   string(builder.ID()),
				Enabled:    true,
				Fields:     fields,
			}, nil
		}),
	}
	for _, field := range spec.Fields {
		field := field
		if field.Flag == "" {
			continue
		}
		values[field.Path] = new(string)
		cmd.Flags().StringVar(values[field.Path], field.Flag, "", field.Description)
	}
	return cmd
}

func newLogoutProviderCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider <name>",
		Short:             "Remove provider credentials",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			builder, err := providerBuilder(provider.ProviderID(args[0]))
			if err != nil {
				return nil, err
			}
			updates := map[string]string{}
			for _, field := range builder.ConfigSpec().Fields {
				if field.Secret || strings.HasPrefix(field.Path, "auth.") {
					updates[providerSettingPath(builder.ID(), field.Path)] = ""
				}
			}
			resolved, err := appconfig.SetValues(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, updates)
			if err != nil {
				return nil, oops.In("cli").With("provider", builder.ID()).Wrapf(err, "logout provider")
			}
			applyResolvedConfig(opts, resolved)
			return providerActionResult{
				ConfigFile: resolved.ConfigPath,
				Provider:   string(builder.ID()),
				Action:     "logout",
			}, nil
		}),
	}
}

func newValidateProviderCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider [name]",
		Short:             "Validate provider configuration",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			builders := builtin.Builders()
			if len(args) == 1 {
				builder, err := providerBuilder(provider.ProviderID(args[0]))
				if err != nil {
					return nil, err
				}
				builders = []provider.ProviderBuilder{builder}
			}
			return providerDoctorResult{
				ConfigFile: opts.ConfigState.ConfigPath,
				Providers:  doctorProviders(builders, opts.ProviderConfig),
			}, nil
		}),
	}
}

func newDoctorProviderCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider <name>",
		Short:             "Diagnose provider configuration and client construction",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			if err := loadConfig(opts); err != nil {
				return nil, err
			}
			builder, err := providerBuilder(provider.ProviderID(args[0]))
			if err != nil {
				return nil, err
			}
			return providerDoctorResult{
				ConfigFile: opts.ConfigState.ConfigPath,
				Providers:  []providerDoctorProvider{doctorProvider(builder, opts.ProviderConfig)},
			}, nil
		}),
	}
}

func newEnableProviderCommand(opts *Options) *cobra.Command {
	return newProviderEnabledCommand(opts, "enable", "Enable a provider", true)
}

func newDisableProviderCommand(opts *Options) *cobra.Command {
	return newProviderEnabledCommand(opts, "disable", "Disable a provider", false)
}

func newProviderEnabledCommand(opts *Options, action string, short string, enabled bool) *cobra.Command {
	return &cobra.Command{
		Use:               "provider <name>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			builder, err := providerBuilder(provider.ProviderID(args[0]))
			if err != nil {
				return nil, err
			}
			resolved, err := appconfig.SetValue(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, providerSettingPath(builder.ID(), "enabled"), fmt.Sprint(enabled))
			if err != nil {
				return nil, oops.In("cli").With("provider", builder.ID()).Wrapf(err, "%s provider", action)
			}
			applyResolvedConfig(opts, resolved)
			return providerActionResult{
				ConfigFile: resolved.ConfigPath,
				Provider:   string(builder.ID()),
				Action:     action,
				Enabled:    &enabled,
			}, nil
		}),
	}
}

func newPreferProviderCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:               "provider <name>",
		Short:             "Prefer a provider when multiple providers match",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProviderIDs,
		RunE: runJSONResult(func(cmd *cobra.Command, args []string) (any, error) {
			builder, err := providerBuilder(provider.ProviderID(args[0]))
			if err != nil {
				return nil, err
			}
			resolved, err := appconfig.SetValue(appconfig.Options{
				ConfigPath:       opts.Config,
				Market:           opts.Market,
				Development:      opts.Development,
				ProviderDefaults: providerDefaults(),
			}, "app.preferred_provider", string(builder.ID()))
			if err != nil {
				return nil, oops.In("cli").With("provider", builder.ID()).Wrapf(err, "prefer provider")
			}
			applyResolvedConfig(opts, resolved)
			return providerActionResult{
				ConfigFile: resolved.ConfigPath,
				Provider:   string(builder.ID()),
				Action:     "prefer",
				Preferred:  true,
			}, nil
		}),
	}
}

func applyResolvedConfig(opts *Options, resolved appconfig.Resolved) {
	opts.Config = resolved.ConfigPath
	opts.Database = resolved.DatabasePath
	opts.ProviderAuthDatabase = resolved.ProviderAuthDatabasePath
	opts.ProviderConfig = resolved.ProviderConfig
	opts.ConfigState = resolved
	opts.PreferProvider = resolved.File.App.PreferredProvider
	opts.configLoaded = true
}

func providerBuilder(id provider.ProviderID) (provider.ProviderBuilder, error) {
	for _, builder := range builtin.Builders() {
		if builder.ID() == id {
			return builder, nil
		}
	}
	return nil, oops.In("cli").With("provider", id).Errorf("unknown provider: %s", id)
}

func doctorProviders(builders []provider.ProviderBuilder, config provider.Config) []providerDoctorProvider {
	result := make([]providerDoctorProvider, 0, len(builders))
	for _, builder := range builders {
		result = append(result, doctorProvider(builder, config))
	}
	return result
}

func doctorProvider(builder provider.ProviderBuilder, config provider.Config) providerDoctorProvider {
	spec := builder.ConfigSpec()
	result := providerDoctorProvider{
		ID:      string(builder.ID()),
		Enabled: providerEnabled(config, builder.ID()),
		Status:  "ok",
		Fields:  make([]providerDoctorField, 0, len(spec.Fields)),
		Issues:  []providerDoctorIssue{},
	}
	if !result.Enabled {
		result.Status = "disabled"
		result.Issues = append(result.Issues, providerDoctorIssue{
			Severity: "warning",
			Path:     providerSettingPath(builder.ID(), "enabled"),
			Message:  "provider is disabled",
		})
	}
	for _, field := range spec.Fields {
		value, source := providerFieldValue(config, builder.ID(), field)
		configured := strings.TrimSpace(value) != ""
		result.Fields = append(result.Fields, providerDoctorField{
			Path:       field.Path,
			Required:   field.Required,
			Secret:     field.Secret,
			Configured: configured,
			Source:     source,
		})
		if result.Enabled && field.Required && !configured {
			result.Status = "error"
			result.Issues = append(result.Issues, providerDoctorIssue{
				Severity: "error",
				Path:     providerSettingPath(builder.ID(), field.Path),
				Message:  fmt.Sprintf("required provider config is missing; run `mwosa login provider %s --%s VALUE`", builder.ID(), field.Flag),
			})
		}
	}
	if result.Enabled && result.Status == "ok" {
		if _, err := builder.Build(config); err != nil {
			result.Status = "error"
			result.Issues = append(result.Issues, providerDoctorIssue{
				Severity: "error",
				Message:  err.Error(),
			})
		}
	}
	return result
}

func providerEnabled(config provider.Config, id provider.ProviderID) bool {
	enabled, ok := config.Bool("providers", string(id), "enabled")
	return !ok || enabled
}

func providerFieldValue(config provider.Config, id provider.ProviderID, field provider.ConfigField) (string, string) {
	path := append([]string{"providers", string(id)}, splitDotPath(field.Path)...)
	if value := strings.TrimSpace(config.String(path...)); value != "" {
		return value, "config_file"
	}
	for _, env := range field.Env {
		if value := strings.TrimSpace(config.Env(env)); value != "" {
			return value, "env:" + env
		}
	}
	return "", ""
}

func providerSettingPath(id provider.ProviderID, path string) string {
	return "providers." + string(id) + "." + path
}

func splitDotPath(path string) []string {
	raw := strings.Split(path, ".")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
