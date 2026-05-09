package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

const (
	AppName          = "mwosa"
	ConfigFileName   = "config.json"
	DatabaseFileName = "mwosa.db"

	ConfigPathEnv   = "MWOSA_CONFIG"
	DatabasePathEnv = "MWOSA_DATABASE"
)

var (
	defaultConfigPath   string
	defaultDatabasePath string
)

type Source string

const (
	SourceFlag       Source = "flag"
	SourceEnv        Source = "env"
	SourceConfigFile Source = "config_file"
	SourceDefault    Source = "default"
)

type Options struct {
	ConfigPath       string
	DatabasePath     string
	Market           string
	Development      bool
	ProviderDefaults []ProviderDefault
}

type ProviderDefault interface {
	ID() provider.ProviderID
	DefaultConfig() provider.Config
}

type File struct {
	App       AppConfig         `json:"app"`
	Providers []provider.Config `json:"providers"`
}

type AppConfig struct {
	Market            string         `json:"market"`
	PreferredProvider string         `json:"preferred_provider,omitempty"`
	Database          DatabaseConfig `json:"database"`
}

type DatabaseConfig struct {
	Path string `json:"path"`
}

type Resolved struct {
	ConfigPath          string
	ConfigPathSource    Source
	ConfigFileExists    bool
	ConfigFileCreated   bool
	DatabasePath        string
	DatabasePathSource  Source
	DataDirectory       string
	DataDirectoryExists bool
	ProviderConfig      provider.Config
	File                File
}

func LoadOrCreate(opts Options) (Resolved, error) {
	errb := oops.In("app_config")

	configPath, configSource, err := resolveConfigPath(opts.ConfigPath, opts.Development)
	if err != nil {
		return Resolved{}, errb.Wrap(err)
	}

	defaultDatabasePath, err := defaultDatabasePathFor(opts.Development)
	if err != nil {
		return Resolved{}, errb.Wrap(err)
	}

	cfg, existed, err := readOrCreateConfigFile(configPath, defaultFile(opts, defaultDatabasePath))
	if err != nil {
		return Resolved{}, errb.With("path", configPath).Wrap(err)
	}

	changed := applyDefaults(&cfg, opts, defaultDatabasePath)
	if changed || !existed {
		if err := writeConfigFile(configPath, cfg); err != nil {
			return Resolved{}, errb.With("path", configPath).Wrap(err)
		}
	}

	databasePath, databaseSource, err := resolveDatabasePath(opts.DatabasePath, cfg.App.Database.Path, defaultDatabasePath)
	if err != nil {
		return Resolved{}, errb.Wrap(err)
	}
	dataDirectory := filepath.Dir(databasePath)
	_, dataDirErr := os.Stat(dataDirectory)
	_, configFileErr := os.Stat(configPath)

	return Resolved{
		ConfigPath:          configPath,
		ConfigPathSource:    configSource,
		ConfigFileExists:    configFileErr == nil,
		ConfigFileCreated:   !existed,
		DatabasePath:        databasePath,
		DatabasePathSource:  databaseSource,
		DataDirectory:       dataDirectory,
		DataDirectoryExists: dataDirErr == nil,
		ProviderConfig:      providerConfigFromFile(cfg),
		File:                cfg,
	}, nil
}

func SetValue(opts Options, settingPath string, rawValue string) (Resolved, error) {
	return SetValues(opts, map[string]string{settingPath: rawValue})
}

func SetValues(opts Options, values map[string]string) (Resolved, error) {
	errb := oops.In("app_config")
	configPath, _, err := resolveConfigPath(opts.ConfigPath, opts.Development)
	if err != nil {
		return Resolved{}, errb.Wrap(err)
	}
	defaultDatabasePath, err := defaultDatabasePathFor(opts.Development)
	if err != nil {
		return Resolved{}, errb.Wrap(err)
	}
	cfg, _, err := readOrCreateConfigFile(configPath, defaultFile(opts, defaultDatabasePath))
	if err != nil {
		return Resolved{}, errb.With("path", configPath).Wrap(err)
	}
	applyDefaults(&cfg, opts, defaultDatabasePath)
	for settingPath, rawValue := range values {
		if err := setConfigValue(&cfg, settingPath, rawValue, opts.ProviderDefaults); err != nil {
			return Resolved{}, errb.With("setting", settingPath).Wrap(err)
		}
	}
	if err := writeConfigFile(configPath, cfg); err != nil {
		return Resolved{}, errb.With("path", configPath).Wrap(err)
	}
	return LoadOrCreate(Options{
		ConfigPath:       configPath,
		Market:           opts.Market,
		Development:      opts.Development,
		ProviderDefaults: opts.ProviderDefaults,
	})
}

func DefaultConfigPath() (string, error) {
	if strings.TrimSpace(defaultConfigPath) != "" {
		return absolutePath(defaultConfigPath)
	}
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName, ConfigFileName), nil
}

func developmentConfigPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", oops.In("app_config").Wrapf(err, "resolve working directory")
	}
	return filepath.Join(workingDirectory, ".mwosa", ConfigFileName), nil
}

func setConfigValue(cfg *File, settingPath string, rawValue string, defaults []ProviderDefault) error {
	parts := splitSettingPath(settingPath)
	if len(parts) < 2 {
		return oops.In("app_config").With("setting", settingPath).New("config setting path must include at least two segments")
	}
	switch parts[0] {
	case "app":
		return setAppConfigValue(cfg, parts[1:], rawValue)
	case "providers":
		return setProviderConfigValue(cfg, parts[1:], rawValue, defaults)
	default:
		return oops.In("app_config").With("setting", settingPath).New("config setting path must start with app or providers")
	}
}

func setAppConfigValue(cfg *File, parts []string, rawValue string) error {
	switch strings.Join(parts, ".") {
	case "market":
		cfg.App.Market = strings.TrimSpace(rawValue)
	case "preferred_provider":
		cfg.App.PreferredProvider = strings.TrimSpace(rawValue)
	case "database.path":
		cfg.App.Database.Path = strings.TrimSpace(rawValue)
	default:
		return oops.In("app_config").With("setting", "app."+strings.Join(parts, ".")).New("unsupported app config setting")
	}
	return nil
}

func setProviderConfigValue(cfg *File, parts []string, rawValue string, defaults []ProviderDefault) error {
	if len(parts) < 2 {
		return oops.In("app_config").With("setting", "providers."+strings.Join(parts, ".")).New("provider config setting must include provider id and field")
	}
	providerID := provider.ProviderID(parts[0])
	providerConfig := ensureProviderConfig(cfg, providerID, defaults)
	value, err := parseConfigValue(parts[len(parts)-1], rawValue)
	if err != nil {
		return err
	}
	setMapValue(providerConfig, parts[1:], value)
	return nil
}

func ensureProviderConfig(cfg *File, providerID provider.ProviderID, defaults []ProviderDefault) provider.Config {
	index := providerConfigIndex(cfg.Providers, providerID)
	if index >= 0 {
		return cfg.Providers[index]
	}
	providerConfig := provider.Config{"id": string(providerID)}
	for _, item := range defaults {
		if item != nil && item.ID() == providerID {
			providerConfig = copyConfig(item.DefaultConfig())
			break
		}
	}
	if strings.TrimSpace(providerConfig.String("id")) == "" {
		providerConfig["id"] = string(providerID)
	}
	cfg.Providers = append(cfg.Providers, providerConfig)
	return providerConfig
}

func parseConfigValue(key string, rawValue string) (any, error) {
	if key == "enabled" {
		value, err := strconv.ParseBool(strings.TrimSpace(rawValue))
		if err != nil {
			return nil, oops.In("app_config").With("value", rawValue).Wrapf(err, "parse bool config value")
		}
		return value, nil
	}
	return strings.TrimSpace(rawValue), nil
}

func setMapValue(target map[string]any, parts []string, value any) {
	if len(parts) == 1 {
		target[parts[0]] = value
		return
	}
	next, ok := target[parts[0]].(map[string]any)
	if !ok {
		next = map[string]any{}
		target[parts[0]] = next
	}
	setMapValue(next, parts[1:], value)
}

func splitSettingPath(path string) []string {
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

func DefaultDatabasePath() (string, error) {
	if strings.TrimSpace(defaultDatabasePath) != "" {
		return absolutePath(defaultDatabasePath)
	}
	base, err := dataBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName, DatabaseFileName), nil
}

func developmentDatabasePath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", oops.In("app_config").Wrapf(err, "resolve working directory")
	}
	return filepath.Join(workingDirectory, ".mwosa", DatabaseFileName), nil
}

func defaultConfigPathFor(development bool) (string, error) {
	if development {
		return developmentConfigPath()
	}
	return DefaultConfigPath()
}

func defaultDatabasePathFor(development bool) (string, error) {
	if development {
		return developmentDatabasePath()
	}
	return DefaultDatabasePath()
}

func resolveConfigPath(flagPath string, development bool) (string, Source, error) {
	if strings.TrimSpace(flagPath) != "" {
		path, err := absolutePath(flagPath)
		return path, SourceFlag, err
	}
	if envPath := strings.TrimSpace(os.Getenv(ConfigPathEnv)); envPath != "" {
		path, err := absolutePath(envPath)
		return path, SourceEnv, err
	}
	path, err := defaultConfigPathFor(development)
	if err != nil {
		return "", "", err
	}
	return path, SourceDefault, nil
}

func resolveDatabasePath(flagPath, configPath, defaultPath string) (string, Source, error) {
	if strings.TrimSpace(flagPath) != "" {
		path, err := absolutePath(flagPath)
		return path, SourceFlag, err
	}
	if envPath := strings.TrimSpace(os.Getenv(DatabasePathEnv)); envPath != "" {
		path, err := absolutePath(envPath)
		return path, SourceEnv, err
	}
	if strings.TrimSpace(configPath) != "" {
		path, err := absolutePath(configPath)
		return path, SourceConfigFile, err
	}
	return defaultPath, SourceDefault, nil
}

func defaultFile(opts Options, databasePath string) File {
	market := strings.TrimSpace(opts.Market)
	if market == "" {
		market = string(provider.MarketKRX)
	}
	cfg := File{
		App: AppConfig{
			Market: market,
			Database: DatabaseConfig{
				Path: databasePath,
			},
		},
	}
	for _, defaults := range opts.ProviderDefaults {
		if defaults == nil {
			continue
		}
		cfg.Providers = append(cfg.Providers, copyConfig(defaults.DefaultConfig()))
	}
	return cfg
}

func readOrCreateConfigFile(path string, defaults File) (File, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults, false, nil
		}
		return File{}, false, oops.In("app_config").Wrapf(err, "read config file")
	}
	if strings.TrimSpace(string(data)) == "" {
		return File{}, true, oops.In("app_config").New("config file is empty")
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return File{}, true, oops.In("app_config").Wrapf(err, "parse config file")
	}
	return cfg, true, nil
}

func writeConfigFile(path string, cfg File) error {
	errb := oops.In("app_config").With("path", path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errb.Wrapf(err, "create config directory")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errb.Wrapf(err, "encode config file")
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errb.Wrapf(err, "write config file")
	}
	return nil
}

func applyDefaults(cfg *File, opts Options, databasePath string) bool {
	changed := false
	if strings.TrimSpace(cfg.App.Market) == "" {
		market := strings.TrimSpace(opts.Market)
		if market == "" {
			market = string(provider.MarketKRX)
		}
		cfg.App.Market = market
		changed = true
	}
	if strings.TrimSpace(cfg.App.Database.Path) == "" {
		cfg.App.Database.Path = databasePath
		changed = true
	}

	for _, defaults := range opts.ProviderDefaults {
		if defaults == nil {
			continue
		}
		defaultConfig := copyConfig(defaults.DefaultConfig())
		index := providerConfigIndex(cfg.Providers, defaults.ID())
		if index < 0 {
			cfg.Providers = append(cfg.Providers, defaultConfig)
			changed = true
			continue
		}
		if mergeMissingConfig(cfg.Providers[index], defaultConfig) {
			changed = true
		}
	}
	return changed
}

func providerConfigFromFile(cfg File) provider.Config {
	merged := provider.ConfigFromEnv()
	providers := map[string]any{}
	for _, item := range cfg.Providers {
		id := strings.TrimSpace(item.String("id"))
		if id == "" {
			continue
		}
		providers[id] = map[string]any(copyConfig(item))
	}
	merged["providers"] = providers
	return merged
}

func providerConfigIndex(configs []provider.Config, id provider.ProviderID) int {
	for index, cfg := range configs {
		if provider.ProviderID(strings.TrimSpace(cfg.String("id"))) == id {
			return index
		}
	}
	return -1
}

func mergeMissingConfig(dst provider.Config, defaults provider.Config) bool {
	changed := false
	for key, value := range defaults {
		existing, ok := dst[key]
		if !ok {
			dst[key] = copyValue(value)
			changed = true
			continue
		}
		existingMap, existingOK := existing.(map[string]any)
		defaultMap, defaultOK := value.(map[string]any)
		if existingOK && defaultOK {
			if mergeMissingMap(existingMap, defaultMap) {
				changed = true
			}
		}
	}
	return changed
}

func mergeMissingMap(dst map[string]any, defaults map[string]any) bool {
	changed := false
	for key, value := range defaults {
		existing, ok := dst[key]
		if !ok {
			dst[key] = copyValue(value)
			changed = true
			continue
		}
		existingMap, existingOK := existing.(map[string]any)
		defaultMap, defaultOK := value.(map[string]any)
		if existingOK && defaultOK {
			if mergeMissingMap(existingMap, defaultMap) {
				changed = true
			}
		}
	}
	return changed
}

func copyConfig(cfg provider.Config) provider.Config {
	copied := provider.Config{}
	for key, value := range cfg {
		copied[key] = copyValue(value)
	}
	return copied
}

func copyValue(value any) any {
	switch typed := value.(type) {
	case provider.Config:
		return copyConfig(typed)
	case map[string]any:
		copied := map[string]any{}
		for key, value := range typed {
			copied[key] = copyValue(value)
		}
		return copied
	default:
		return typed
	}
}

func absolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", oops.In("app_config").With("path", path).Wrapf(err, "resolve absolute path")
	}
	return absolute, nil
}

func configBaseDir() (string, error) {
	if runtime.GOOS == "windows" {
		return os.UserConfigDir()
	}
	if path := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); path != "" {
		return absolutePath(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", oops.In("app_config").Wrapf(err, "resolve user home directory")
	}
	return filepath.Join(home, ".config"), nil
}

func dataBaseDir() (string, error) {
	if runtime.GOOS == "windows" {
		if path := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); path != "" {
			return absolutePath(path)
		}
		return os.UserConfigDir()
	}
	if path := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); path != "" {
		return absolutePath(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", oops.In("app_config").Wrapf(err, "resolve user home directory")
	}
	return filepath.Join(home, ".local", "share"), nil
}
