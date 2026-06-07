//go:build integration

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

	"github.com/awuzag/mwosa/internal/integrationtest"
)

func TestPostgresConfigCLIUsesEnvURLAndKeepsProviderAuthSidecar(t *testing.T) {
	postgres := integrationtest.StartPostgres(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sqlitePath := filepath.Join(tempDir, "mwosa.db")
	root := integrationRepoRoot(t)
	env := []string{
		"MWOSA_DATABASE_URL=" + postgres.DSN,
	}

	sqliteResult := runMWOSAIntegration(t, root, env,
		"--config", configPath,
		"config", "use-database", "sqlite", "--path", sqlitePath,
	)
	requireIntegrationOutputDoesNotLeakSecret(t, sqliteResult)

	postgresResult := runMWOSAIntegration(t, root, env,
		"--config", configPath,
		"config", "use-database", "postgres", "--url-env", "MWOSA_DATABASE_URL",
	)
	requireIntegrationOutputDoesNotLeakSecret(t, postgresResult)
	postgresJSON := decodeIntegrationJSONObject(t, postgresResult.Stdout)
	requireIntegrationString(t, postgresJSON, "backend", "postgres")
	requireIntegrationString(t, postgresJSON, "url_env", "MWOSA_DATABASE_URL")

	inspectResult := runMWOSAIntegration(t, root, env,
		"--config", configPath,
		"inspect", "config",
	)
	requireIntegrationOutputDoesNotLeakSecret(t, inspectResult)
	inspectJSON := decodeIntegrationJSONObject(t, inspectResult.Stdout)
	database := requireIntegrationObject(t, inspectJSON, "database")
	requireIntegrationString(t, database, "backend", "postgres")
	requireIntegrationString(t, database, "backend_source", "config_file")
	databaseURL := requireIntegrationObject(t, database, "url")
	requireIntegrationString(t, databaseURL, "source", "env")
	requireIntegrationString(t, databaseURL, "url_env", "MWOSA_DATABASE_URL")
	if configured, ok := databaseURL["configured"].(bool); !ok || !configured {
		t.Fatalf("database.url.configured = %#v, want true", databaseURL["configured"])
	}
	if value := integrationStringValue(databaseURL["value"]); !strings.Contains(value, ":xxxxx@") {
		t.Fatalf("database.url.value = %q, want masked password", value)
	}

	providerAuth := requireIntegrationObject(t, inspectJSON, "provider_auth_database")
	requireIntegrationString(t, providerAuth, "path", filepath.Join(tempDir, "provider-token-cache.sqlite"))

	listResult := runMWOSAIntegration(t, root, env,
		"--config", configPath,
		"--output", "json",
		"list", "macro-indicators",
	)
	requireIntegrationOutputDoesNotLeakSecret(t, listResult)
	indicators := decodeIntegrationJSONArray(t, listResult.Stdout)
	if len(indicators) != 0 {
		t.Fatalf("postgres macro indicator rows = %d, want empty fresh database", len(indicators))
	}

	revertResult := runMWOSAIntegration(t, root, env,
		"--config", configPath,
		"config", "use-database", "sqlite", "--path", sqlitePath,
	)
	requireIntegrationOutputDoesNotLeakSecret(t, revertResult)
	revertJSON := decodeIntegrationJSONObject(t, revertResult.Stdout)
	requireIntegrationString(t, revertJSON, "backend", "sqlite")
	requireIntegrationString(t, revertJSON, "path", sqlitePath)
	if got := integrationStringValue(revertJSON["url"]); got != "" {
		t.Fatalf("sqlite revert url = %q, want empty", got)
	}
}

type integrationCommandResult struct {
	Stdout string
	Stderr string
}

func runMWOSAIntegration(t *testing.T, root string, env []string, args ...string) integrationCommandResult {
	t.Helper()

	commandArgs := append([]string{"run", "./cmd/mwosa"}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	command := exec.CommandContext(ctx, "go", commandArgs...)
	command.Dir = root
	command.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	result := integrationCommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("mwosa command timed out: go %s", strings.Join(commandArgs, " "))
	}
	if err != nil {
		t.Fatalf(
			"mwosa command failed: go %s\nstderr:\n%s\nstdout:\n%s",
			strings.Join(commandArgs, " "),
			redactIntegrationSecrets(result.Stderr),
			redactIntegrationSecrets(result.Stdout),
		)
	}
	return result
}

func requireIntegrationOutputDoesNotLeakSecret(t *testing.T, result integrationCommandResult) {
	t.Helper()
	if strings.Contains(result.Stdout, integrationtest.PostgresPassword) {
		t.Fatal("stdout leaked postgres password")
	}
	if strings.Contains(result.Stderr, integrationtest.PostgresPassword) {
		t.Fatal("stderr leaked postgres password")
	}
}

func decodeIntegrationJSONObject(t *testing.T, output string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("stdout is not a JSON object: %v\n%s", err, redactIntegrationSecrets(output))
	}
	return value
}

func decodeIntegrationJSONArray(t *testing.T, output string) []map[string]any {
	t.Helper()
	var value []map[string]any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("stdout is not a JSON array: %v\n%s", err, redactIntegrationSecrets(output))
	}
	return value
}

func requireIntegrationObject(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not a JSON object: %#v", key, object[key])
	}
	return value
}

func requireIntegrationString(t *testing.T, object map[string]any, key string, want string) {
	t.Helper()
	if got := integrationStringValue(object[key]); got != want {
		t.Fatalf("%s = %q, want %q in %#v", key, got, want, object)
	}
}

func redactIntegrationSecrets(value string) string {
	return strings.ReplaceAll(value, integrationtest.PostgresPassword, "<redacted>")
}

func integrationStringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func integrationRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
