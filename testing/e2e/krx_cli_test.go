//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	krxE2EEnabledEnv     = "KRX_E2E"
	krxE2EAuthKeyEnv     = "MWOSA_KRX_AUTH_KEY"
	krxE2ETimeoutEnv     = "KRX_E2E_TIMEOUT"
	defaultKRXE2EAsOf    = "20240415"
	defaultKRXE2ETimeout = 120 * time.Second
)

func TestKRXCLILiveRawAndCanonicalDailyFlow(t *testing.T) {
	config := loadKRXCLIE2EConfig(t)

	rawGet := runMWOSA(t, config,
		"get", "krx", "etf_bydd_trd",
		"--as-of", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireEmptyStderr(t, rawGet)
	rawGetJSON := decodeJSONObject(t, rawGet.Stdout)
	requireString(t, rawGetJSON, "provider", "krx")
	requireString(t, rawGetJSON, "api_id", "etf_bydd_trd")
	requireNumberGreaterThanZero(t, rawGetJSON, "row_count")
	requireRows(t, rawGetJSON, "rows")

	rawSync := runMWOSA(t, config,
		"sync", "krx", "etf_bydd_trd",
		"--as-of", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireEmptyStderr(t, rawSync)
	rawSyncJSON := decodeJSONObject(t, rawSync.Stdout)
	requireString(t, rawSyncJSON, "provider", "krx")
	requireString(t, rawSyncJSON, "api_id", "etf_bydd_trd")
	requireNumberGreaterThanZero(t, rawSyncJSON, "row_count")
	requireNumberGreaterThanZero(t, rawSyncJSON, "rows_affected")

	etfSync := runMWOSA(t, config,
		"sync", "daily",
		"--provider", "krx",
		"--security-type", "etf",
		"--as-of", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireEmptyStderr(t, etfSync)
	etfSyncJSON := decodeJSONObject(t, etfSync.Stdout)
	requireString(t, etfSyncJSON, "provider", "krx")
	requireString(t, etfSyncJSON, "security_type", "etf")
	requireNumberGreaterThanZero(t, etfSyncJSON, "bars_stored")

	stockBackfill := runMWOSA(t, config,
		"backfill", "daily",
		"--provider", "krx",
		"--security-type", "stock",
		"--from", defaultKRXE2EAsOf,
		"--to", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireStderrContains(t, stockBackfill, "backfill daily: fetched pages")
	requireStdoutDoesNotContain(t, stockBackfill, "backfill daily: fetched pages")
	stockBackfillJSON := decodeJSONObject(t, stockBackfill.Stdout)
	requireString(t, stockBackfillJSON, "provider", "krx")
	requireString(t, stockBackfillJSON, "security_type", "stock")
	requireNumberGreaterThanZero(t, stockBackfillJSON, "bars_stored")

	etfDaily := runMWOSA(t, config,
		"get", "daily", "069500",
		"--security-type", "etf",
		"--as-of", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireEmptyStderr(t, etfDaily)
	etfRows := decodeJSONArray(t, etfDaily.Stdout)
	requireDailyBar(t, etfRows, "069500", "etf")

	stockDaily := runMWOSA(t, config,
		"get", "daily", "005930",
		"--security-type", "stock",
		"--as-of", defaultKRXE2EAsOf,
		"-o", "json",
	)
	requireEmptyStderr(t, stockDaily)
	stockRows := decodeJSONArray(t, stockDaily.Stdout)
	requireDailyBar(t, stockRows, "005930", "stock")
}

type krxCLIE2EConfig struct {
	RootDir      string
	ConfigPath   string
	DatabasePath string
	AuthKey      string
	Timeout      time.Duration
}

type commandResult struct {
	Stdout string
	Stderr string
}

func loadKRXCLIE2EConfig(t *testing.T) krxCLIE2EConfig {
	t.Helper()
	if os.Getenv(krxE2EEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run live KRX CLI e2e tests", krxE2EEnabledEnv)
	}
	authKey := strings.TrimSpace(os.Getenv(krxE2EAuthKeyEnv))
	if authKey == "" {
		t.Skipf("set %s to run live KRX CLI e2e tests", krxE2EAuthKeyEnv)
	}
	tempDir := t.TempDir()
	return krxCLIE2EConfig{
		RootDir:      repoRoot(t),
		ConfigPath:   filepath.Join(tempDir, "config.json"),
		DatabasePath: filepath.Join(tempDir, "mwosa.db"),
		AuthKey:      authKey,
		Timeout:      envDurationDefault(t, krxE2ETimeoutEnv, defaultKRXE2ETimeout),
	}
}

func runMWOSA(t *testing.T, config krxCLIE2EConfig, args ...string) commandResult {
	t.Helper()
	commandArgs := append([]string{
		"run", "./cmd/mwosa",
		"--config", config.ConfigPath,
		"--database", config.DatabasePath,
	}, args...)

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	command := exec.CommandContext(ctx, "go", commandArgs...)
	command.Dir = config.RootDir
	command.Env = append(os.Environ(),
		"MWOSA_CONFIG="+config.ConfigPath,
		"MWOSA_DATABASE="+config.DatabasePath,
		krxE2EAuthKeyEnv+"="+config.AuthKey,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	result := commandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	requireNoSecretLeak(t, config.AuthKey, result)
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("mwosa command timed out: go %s", strings.Join(commandArgs, " "))
	}
	if err != nil {
		t.Fatalf("mwosa command failed: go %s\nstderr:\n%s\nstdout:\n%s", strings.Join(commandArgs, " "), result.Stderr, result.Stdout)
	}
	return result
}

func decodeJSONObject(t *testing.T, output string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("stdout is not a JSON object: %v\n%s", err, output)
	}
	return value
}

func decodeJSONArray(t *testing.T, output string) []map[string]any {
	t.Helper()
	var value []map[string]any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("stdout is not a JSON array: %v\n%s", err, output)
	}
	if len(value) == 0 {
		t.Fatalf("stdout JSON array is empty:\n%s", output)
	}
	return value
}

func requireRows(t *testing.T, object map[string]any, key string) {
	t.Helper()
	rows, ok := object[key].([]any)
	if !ok {
		t.Fatalf("%s is not a JSON array: %#v", key, object[key])
	}
	if len(rows) == 0 {
		t.Fatalf("%s is empty", key)
	}
	first, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("%s[0] is not a JSON object: %#v", key, rows[0])
	}
	if strings.TrimSpace(stringValue(first["ISU_CD"])) == "" || strings.TrimSpace(stringValue(first["ISU_NM"])) == "" {
		t.Fatalf("%s[0] missing required KRX fields: %#v", key, first)
	}
}

func requireDailyBar(t *testing.T, rows []map[string]any, symbol string, securityType string) {
	t.Helper()
	row := rows[0]
	requireString(t, row, "provider", "krx")
	requireString(t, row, "security_type", securityType)
	requireString(t, row, "symbol", symbol)
	if strings.TrimSpace(stringValue(row["trading_date"])) == "" {
		t.Fatalf("daily row missing trading_date: %#v", row)
	}
	if strings.TrimSpace(stringValue(row["closing_price"])) == "" {
		t.Fatalf("daily row missing closing_price: %#v", row)
	}
}

func requireString(t *testing.T, object map[string]any, key string, want string) {
	t.Helper()
	got := stringValue(object[key])
	if got != want {
		t.Fatalf("%s = %q, want %q in %#v", key, got, want, object)
	}
}

func requireNumberGreaterThanZero(t *testing.T, object map[string]any, key string) {
	t.Helper()
	value, ok := object[key].(float64)
	if !ok {
		t.Fatalf("%s is not a JSON number: %#v", key, object[key])
	}
	if value <= 0 {
		t.Fatalf("%s = %v, want > 0 in %#v", key, value, object)
	}
}

func requireEmptyStderr(t *testing.T, result commandResult) {
	t.Helper()
	if strings.TrimSpace(result.Stderr) != "" {
		t.Fatalf("stderr should be empty for machine-readable result commands:\n%s", result.Stderr)
	}
}

func requireStderrContains(t *testing.T, result commandResult, want string) {
	t.Helper()
	if !strings.Contains(result.Stderr, want) {
		t.Fatalf("stderr missing %q:\n%s", want, result.Stderr)
	}
}

func requireStdoutDoesNotContain(t *testing.T, result commandResult, value string) {
	t.Helper()
	if strings.Contains(result.Stdout, value) {
		t.Fatalf("stdout should not contain diagnostic text %q:\n%s", value, result.Stdout)
	}
}

func requireNoSecretLeak(t *testing.T, secret string, result commandResult) {
	t.Helper()
	if secret == "" {
		return
	}
	if strings.Contains(result.Stdout, secret) {
		t.Fatal("stdout leaked KRX auth key")
	}
	if strings.Contains(result.Stderr, secret) {
		t.Fatal("stderr leaked KRX auth key")
	}
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func envDurationDefault(t *testing.T, name string, fallback time.Duration) time.Duration {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	timeout, err := time.ParseDuration(value)
	if err != nil {
		t.Fatalf("parse %s=%q: %v", name, value, err)
	}
	if timeout <= 0 {
		t.Fatalf("%s must be positive: %s", name, value)
	}
	return timeout
}
