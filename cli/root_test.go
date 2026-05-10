package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{
		Version: "test-version",
		Commit:  "abc123",
		Date:    "2026-04-26",
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", t.TempDir() + "/config.json", "version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"mwosa test-version",
		"schema dev",
		"commit abc123",
		"built 2026-04-26",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output missing %q in:\n%s", want, got)
		}
	}
}

func TestRootHelpHasOutputFlag(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}

	got := out.String()
	for _, want := range []string{"--config", "--output"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output should include %s flag:\n%s", want, got)
		}
	}
}

func TestMarketDataCommandsAreRegistered(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	for _, args := range [][]string{
		{"get", "intraday", "005930"},
		{"get", "orderbook", "005930"},
		{"list", "trades", "005930"},
	} {
		found, _, err := cmd.Find(args)
		if err != nil {
			t.Fatalf("find %v: %v", args, err)
		}
		if found == nil || found.Use == "" {
			t.Fatalf("find %v returned no command", args)
		}
	}
}

func TestCompletionBashGeneratesScriptWithoutConfigLoad(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "completion", "bash"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute completion bash: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{"__start_mwosa", "complete -o default"} {
		if !strings.Contains(got, want) {
			t.Fatalf("bash completion output missing %q in:\n%s", want, got)
		}
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("completion should not create config file, stat error = %v", err)
	}
}

func TestCompletionRejectsUnsupportedShell(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"completion", "xonsh"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute completion xonsh error = nil, want unsupported shell error")
	}
	if !strings.Contains(err.Error(), "unsupported completion shell: xonsh") {
		t.Fatalf("unsupported shell error = %q", err.Error())
	}
}

func TestCompletionProtocolSuggestsSupportedShells(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{cobra.ShellCompRequestCmd, "completion", ""})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute completion protocol: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"bash\tGenerate Bash completion script",
		"zsh\tGenerate Zsh completion script",
		"fish\tGenerate Fish completion script",
		"powershell\tGenerate PowerShell completion script",
		":4",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("completion protocol output missing %q in:\n%s", want, got)
		}
	}
}

func TestCompletionProtocolCompletesOutputFlagValues(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{cobra.ShellCompRequestCmd, "version", "--output", ""})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute output flag completion: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"table\tHuman-readable table",
		"json\tMachine-readable JSON",
		"ndjson\tNewline-delimited JSON",
		"csv\tComma-separated values",
		":4",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output flag completion missing %q in:\n%s", want, got)
		}
	}
}

func TestInspectConfigCreatesAndPrintsResolvedPaths(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "inspect", "config"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect config: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	got := out.String()
	for _, want := range []string{`"config_file"`, configPath, `"providers"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("inspect config output missing %q in:\n%s", want, got)
		}
	}
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("inspect config output should be json: %v\n%s", err, got)
	}
}

func TestInspectConfigAlwaysWritesJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "--output", "csv", "inspect", "config"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect config: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("inspect config output should ignore tabular output modes: %v\n%s", err, out.String())
	}
	if _, ok := parsed["config_file"]; !ok {
		t.Fatalf("inspect config json missing config_file: %#v", parsed)
	}
}

func TestConfigSetUpdatesProviderConfigAndMasksSecretOutput(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--config", configPath,
		"config", "set", "providers.datago.auth.service_key", "secret-key",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute config set: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "secret-key") {
		t.Fatalf("config set output should mask secret:\n%s", out.String())
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("config set output should be json: %v\n%s", err, out.String())
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	var cfg struct {
		Providers []provider.Config `json:"providers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config file: %v", err)
	}
	if len(cfg.Providers) == 0 {
		t.Fatal("provider config was not written")
	}
	if got := cfg.Providers[0].String("auth", "service_key"); got != "secret-key" {
		t.Fatalf("service key = %q, want secret-key", got)
	}
}

func TestLoginProviderDataGoWritesProviderConfigAndMasksOutput(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--config", configPath,
		"login", "provider", "datago", "--service-key", "secret-key",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute login provider datago: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "secret-key") {
		t.Fatalf("login provider output should not include secret:\n%s", out.String())
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("login provider output should be json: %v\n%s", err, out.String())
	}
	if got := result["provider"]; got != "datago" {
		t.Fatalf("login provider provider = %v, want datago", got)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	var cfg struct {
		Providers []provider.Config `json:"providers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config file: %v", err)
	}
	if len(cfg.Providers) == 0 {
		t.Fatal("provider config was not written")
	}
	if got := cfg.Providers[0].String("auth", "service_key"); got != "secret-key" {
		t.Fatalf("service key = %q, want secret-key", got)
	}
}

func TestValidateProviderReportsMissingDataGoServiceKey(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--config", configPath,
		"validate", "provider", "datago",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute validate provider datago: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{`"status": "error"`, "providers.datago.groups.securitiesProductPrice.auth.service_key", "providers.datago.groups.stockPrice.auth.service_key"} {
		if !strings.Contains(got, want) {
			t.Fatalf("validate provider output missing %q in:\n%s", want, got)
		}
	}
}

func TestValidateProviderReportsConfiguredDataGo(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	addCmd := NewRootCommand(BuildInfo{})
	addCmd.SetOut(&bytes.Buffer{})
	addCmd.SetErr(&bytes.Buffer{})
	addCmd.SetArgs([]string{
		"--config", configPath,
		"login", "provider", "datago", "--service-key", "secret-key",
	})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("execute login provider datago: %v", err)
	}

	validateCmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	validateCmd.SetOut(&out)
	validateCmd.SetErr(&out)
	validateCmd.SetArgs([]string{
		"--config", configPath,
		"validate", "provider", "datago",
	})
	if err := validateCmd.Execute(); err != nil {
		t.Fatalf("execute validate provider datago: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{`"status": "ok"`, `"configured": true`, `"source": "config_file"`, `"issues": []`} {
		if !strings.Contains(got, want) {
			t.Fatalf("validate provider output missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "secret-key") {
		t.Fatalf("validate provider output should not include secret:\n%s", got)
	}
}

func TestOptionsValidateTreatsProviderFlagsAsOptional(t *testing.T) {
	opts := Options{
		Output:   OutputModeTable,
		Market:   "krx",
		Database: t.TempDir() + "/mwosa.db",
	}

	if err := opts.Validate(); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
}

func TestOptionsValidateRequiresCoreOptions(t *testing.T) {
	opts := Options{
		Output: OutputMode("xml"),
	}

	err := opts.Validate()
	if err == nil {
		t.Fatal("Validate error = nil, want validation errors")
	}
	for _, want := range []string{"Output", "Market", "Database"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("validation error missing %q in %q", want, err.Error())
		}
	}
}
