package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
)

func TestLoadOrCreateCreatesConfigWithAppAndProviderDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mwosa", "config.json")
	databasePath := filepath.Join(t.TempDir(), "data", "mwosa.db")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	defaultDatabasePath, err := DefaultDatabasePath()
	if err != nil {
		t.Fatalf("DefaultDatabasePath error = %v", err)
	}

	resolved, err := LoadOrCreate(Options{
		ConfigPath:   configPath,
		DatabasePath: databasePath,
		Market:       string(provider.MarketKRX),
		ProviderDefaults: []ProviderDefault{
			fakeProviderDefault{
				id: provider.ProviderID("fake"),
				config: provider.Config{
					"id":      "fake",
					"enabled": true,
					"auth": map[string]any{
						"token": "",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("LoadOrCreate error = %v", err)
	}
	if resolved.ConfigPath != configPath {
		t.Fatalf("ConfigPath = %q, want %q", resolved.ConfigPath, configPath)
	}
	if resolved.DatabasePath != databasePath {
		t.Fatalf("DatabasePath = %q, want %q", resolved.DatabasePath, databasePath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse generated config: %v", err)
	}
	if cfg.App.Database.Path != defaultDatabasePath {
		t.Fatalf("generated database path = %q, want %q", cfg.App.Database.Path, defaultDatabasePath)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].String("id") != "fake" {
		t.Fatalf("generated providers = %#v, want fake provider", cfg.Providers)
	}
}

func TestDefaultPathsCanBeOverriddenByDevelopmentBuild(t *testing.T) {
	previousConfigPath := defaultConfigPath
	previousDatabasePath := defaultDatabasePath
	t.Cleanup(func() {
		defaultConfigPath = previousConfigPath
		defaultDatabasePath = previousDatabasePath
	})

	projectDir := t.TempDir()
	defaultConfigPath = filepath.Join(projectDir, ".mwosa", "config.json")
	defaultDatabasePath = filepath.Join(projectDir, ".mwosa", "mwosa.db")

	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath error = %v", err)
	}
	databasePath, err := DefaultDatabasePath()
	if err != nil {
		t.Fatalf("DefaultDatabasePath error = %v", err)
	}

	if configPath != filepath.Join(projectDir, ".mwosa", "config.json") {
		t.Fatalf("config path = %q, want project-local config path", configPath)
	}
	if databasePath != filepath.Join(projectDir, ".mwosa", "mwosa.db") {
		t.Fatalf("database path = %q, want project-local database path", databasePath)
	}
}

func TestLoadOrCreateUsesProjectLocalDefaultsForDevelopment(t *testing.T) {
	previousWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectDir := t.TempDir()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	projectDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("get project working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousWorkingDirectory); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	resolved, err := LoadOrCreate(Options{
		Development: true,
		Market:      string(provider.MarketKRX),
	})
	if err != nil {
		t.Fatalf("LoadOrCreate error = %v", err)
	}

	if resolved.ConfigPath != filepath.Join(projectDir, ".mwosa", "config.json") {
		t.Fatalf("config path = %q, want project-local config", resolved.ConfigPath)
	}
	if resolved.DatabasePath != filepath.Join(projectDir, ".mwosa", "mwosa.db") {
		t.Fatalf("database path = %q, want project-local database", resolved.DatabasePath)
	}
	if resolved.ConfigPathSource != SourceDefault || resolved.DatabasePathSource != SourceConfigFile {
		t.Fatalf("sources = %s/%s, want default/config_file", resolved.ConfigPathSource, resolved.DatabasePathSource)
	}
}

func TestLoadOrCreateMergesNewProviderDefaultsWithoutOverwritingExistingValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	existing := File{
		App: AppConfig{
			Market: "krx",
			Database: DatabaseConfig{
				Path: filepath.Join(t.TempDir(), "custom.db"),
			},
		},
		Providers: []provider.Config{
			{
				"id":      "fake",
				"enabled": false,
				"auth": map[string]any{
					"token": "keep-me",
				},
			},
		},
	}
	writeTestConfig(t, configPath, existing)

	_, err := LoadOrCreate(Options{
		ConfigPath: configPath,
		ProviderDefaults: []ProviderDefault{
			fakeProviderDefault{
				id: provider.ProviderID("fake"),
				config: provider.Config{
					"id":      "fake",
					"enabled": true,
					"auth": map[string]any{
						"token": "",
					},
					"groups": map[string]any{
						"core": map[string]any{
							"enabled": true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("LoadOrCreate error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read merged config file: %v", err)
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse merged config: %v", err)
	}
	if got, _ := cfg.Providers[0].Bool("enabled"); got {
		t.Fatal("existing enabled value was overwritten")
	}
	if got := cfg.Providers[0].String("auth", "token"); got != "keep-me" {
		t.Fatalf("auth token = %q, want keep-me", got)
	}
	if _, ok := cfg.Providers[0].Lookup("groups", "core", "enabled"); !ok {
		t.Fatalf("new nested provider default was not merged: %#v", cfg.Providers[0])
	}
}

func TestLoadOrCreateBuildsProviderConfigFromProviderArrayAndEnv(t *testing.T) {
	t.Setenv("MWOSA_FAKE_TOKEN", "env-token")
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, configPath, File{
		App: AppConfig{
			Market: string(provider.MarketKRX),
			Database: DatabaseConfig{
				Path: filepath.Join(t.TempDir(), "mwosa.db"),
			},
		},
		Providers: []provider.Config{
			{
				"id": "fake",
				"auth": map[string]any{
					"token": "file-token",
				},
			},
		},
	})

	resolved, err := LoadOrCreate(Options{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadOrCreate error = %v", err)
	}
	if got := resolved.ProviderConfig.String("providers", "fake", "auth", "token"); got != "file-token" {
		t.Fatalf("provider config token = %q, want file-token", got)
	}
	if got := resolved.ProviderConfig.Env("MWOSA_FAKE_TOKEN"); got != "env-token" {
		t.Fatalf("provider env token = %q, want env-token", got)
	}
}

func TestSetValueUpdatesAppConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	resolved, err := SetValue(Options{ConfigPath: configPath}, "app.database.path", filepath.Join(t.TempDir(), "custom.db"))
	if err != nil {
		t.Fatalf("SetValue error = %v", err)
	}
	if resolved.File.App.Database.Path == "" {
		t.Fatal("database path was not written")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config file: %v", err)
	}
	if cfg.App.Database.Path != resolved.File.App.Database.Path {
		t.Fatalf("database path = %q, want %q", cfg.App.Database.Path, resolved.File.App.Database.Path)
	}
}

func TestSetValueUpdatesProviderConfigByProviderID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	resolved, err := SetValue(Options{
		ConfigPath: configPath,
		ProviderDefaults: []ProviderDefault{
			fakeProviderDefault{
				id: provider.ProviderID("fake"),
				config: provider.Config{
					"id":      "fake",
					"enabled": true,
					"auth": map[string]any{
						"token": "",
					},
				},
			},
		},
	}, "providers.fake.auth.token", "secret")
	if err != nil {
		t.Fatalf("SetValue error = %v", err)
	}
	if got := resolved.ProviderConfig.String("providers", "fake", "auth", "token"); got != "secret" {
		t.Fatalf("provider token = %q, want secret", got)
	}
}

func TestSetValueParsesProviderEnabledAsBool(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	resolved, err := SetValue(Options{ConfigPath: configPath}, "providers.newsource.enabled", "false")
	if err != nil {
		t.Fatalf("SetValue error = %v", err)
	}
	enabled, ok := resolved.ProviderConfig.Bool("providers", "newsource", "enabled")
	if !ok || enabled {
		t.Fatalf("enabled = %t, %t, want false, true", enabled, ok)
	}
}

func writeTestConfig(t *testing.T, path string, cfg File) {
	t.Helper()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

type fakeProviderDefault struct {
	id     provider.ProviderID
	config provider.Config
}

func (d fakeProviderDefault) ID() provider.ProviderID {
	return d.id
}

func (d fakeProviderDefault) DefaultConfig() provider.Config {
	return d.config
}
