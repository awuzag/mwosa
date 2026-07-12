//go:build e2e

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/awuzag/mwosa/internal/integrationtest"
)

func TestAggregateLiveE2EKRXMarketSnapshot(t *testing.T) {
	if os.Getenv("MWOSA_LIVE_E2E") != "1" {
		t.Skip("set MWOSA_LIVE_E2E=1 to run live Aggregate E2E")
	}
	krxAuthKey := strings.TrimSpace(os.Getenv("MWOSA_KRX_AUTH_KEY"))
	if krxAuthKey == "" {
		t.Skip("set MWOSA_KRX_AUTH_KEY to run live KRX Aggregate E2E")
	}

	repoRoot := liveE2ERepoRoot(t)
	mongoServer := integrationtest.StartMongoDB(t)
	bin := buildLiveE2EMwosa(t, repoRoot, krxAuthKey)
	configPath := filepath.Join(t.TempDir(), "config.json")
	runner := liveE2ECLI{
		bin:        bin,
		repoRoot:   repoRoot,
		configPath: configPath,
		mongoURI:   mongoServer.URI,
		env:        liveE2EEnv(t, configPath, krxAuthKey),
		secrets:    []string{krxAuthKey},
	}
	specPath := filepath.Join(repoRoot, "examples", "aggregate", "e2e-trading-scenarios", "krx-market-snapshot.aggregate.yaml")
	alias := fmt.Sprintf("e2e-krx-market-live-%d", time.Now().UTC().UnixNano())

	initOut := runner.run(t, 2*time.Minute, "init", "storage", "--mongodb-uri", mongoServer.URI, "--mongodb-database", "mwosa", "-o", "json")
	initResult := decodeLiveE2EJSON[liveE2EStorageInit](t, "init storage", initOut, runner.secrets)
	if initResult.Status != "ok" || initResult.Backend != "mongodb" {
		t.Fatalf("init storage = %+v, want mongodb ok", initResult)
	}

	validateOut := runner.run(t, 30*time.Second, "validate", "aggregate", specPath, "-o", "json")
	validateResult := decodeLiveE2EJSON[liveE2EValidation](t, "validate aggregate", validateOut, runner.secrets)
	if !validateResult.Valid {
		t.Fatalf("validate aggregate valid = false")
	}

	updateOut := runner.run(t, 30*time.Second, "update", "aggregate", "krx-market-snapshot", "--file", specPath, "-o", "json")
	updateResult := decodeLiveE2EJSON[liveE2EAggregateDetail](t, "update aggregate", updateOut, runner.secrets)
	if updateResult.Aggregate.Name != "krx-market-snapshot" {
		t.Fatalf("updated aggregate name = %q, want krx-market-snapshot", updateResult.Aggregate.Name)
	}

	planOut := runner.run(t, 30*time.Second, "inspect", "aggregate-plan", "krx-market-snapshot", "--view", "stages", "-o", "json")
	planStages := decodeLiveE2EJSON[[]liveE2EPlanStage](t, "inspect aggregate-plan stages", planOut, runner.secrets)
	requireLiveE2EPlanStage(t, planStages, "kospi_raw", "provider_raw", "krx", "stk_bydd_trd")
	requireLiveE2EPlanStage(t, planStages, "kosdaq_raw", "provider_raw", "krx", "ksq_bydd_trd")
	requireLiveE2EPlanStage(t, planStages, "merged", "aggregate", "", "")

	runOut := runner.run(t, 3*time.Minute, "run", "aggregate", "krx-market-snapshot", "--alias", alias, "-o", "json")
	rows := decodeLiveE2EJSON[[]map[string]any](t, "run aggregate rows", runOut, runner.secrets)
	if len(rows) == 0 {
		t.Fatalf("run aggregate returned 0 rows")
	}
	markets := map[string]bool{}
	for _, row := range rows {
		if market, ok := row["market"].(string); ok {
			markets[market] = true
		}
	}
	if !markets["KOSPI"] || !markets["KOSDAQ"] {
		t.Fatalf("run aggregate markets = %v, want both KOSPI and KOSDAQ", markets)
	}

	historyOut := runner.run(t, 30*time.Second, "history", "aggregate", "--name", "krx-market-snapshot", "-o", "json")
	history := decodeLiveE2EJSON[[]liveE2ERun](t, "history aggregate", historyOut, runner.secrets)
	run := requireLiveE2ERun(t, history, alias)
	if run.Status != "succeeded" {
		t.Fatalf("run status = %q, want succeeded", run.Status)
	}
	if run.ResultCount <= 0 {
		t.Fatalf("run result_count = %d, want > 0", run.ResultCount)
	}

	stagesOut := runner.run(t, 30*time.Second, "inspect", "aggregate-run", alias, "--view", "stages", "-o", "json")
	stages := decodeLiveE2EJSON[[]liveE2EStageSummary](t, "inspect aggregate-run stages", stagesOut, runner.secrets)
	requireLiveE2ESucceededStage(t, stages, "kospi_raw", 1)
	requireLiveE2ESucceededStage(t, stages, "kosdaq_raw", 1)
	requireLiveE2ESucceededStage(t, stages, "kospi_rows", 1)
	requireLiveE2ESucceededStage(t, stages, "kosdaq_rows", 1)
	requireLiveE2ESucceededStage(t, stages, "merged", 1)
}

type liveE2ECLI struct {
	bin        string
	repoRoot   string
	configPath string
	mongoURI   string
	env        []string
	secrets    []string
}

func (c liveE2ECLI) run(t *testing.T, timeout time.Duration, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"--config", c.configPath, "--database-backend", "mongodb", "--database-url", c.mongoURI}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.bin, fullArgs...)
	cmd.Dir = c.repoRoot
	cmd.Env = c.env
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if containsLiveE2ESecret(output, c.secrets) {
		t.Fatalf("mwosa %s output contains a credential value", publicLiveE2EArgs(fullArgs))
	}
	if err != nil {
		t.Fatalf("mwosa %s failed: %v\n%s", publicLiveE2EArgs(fullArgs), err, redactLiveE2EOutput(output, c.secrets))
	}
	return output
}

func buildLiveE2EMwosa(t *testing.T, repoRoot string, krxAuthKey string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mwosa")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/mwosa")
	cmd.Dir = repoRoot
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	secrets := []string{krxAuthKey}
	if containsLiveE2ESecret(output, secrets) {
		t.Fatalf("go build output contains a credential value")
	}
	if err != nil {
		t.Fatalf("build mwosa test binary: %v\n%s", err, redactLiveE2EOutput(output, secrets))
	}
	return bin
}

func liveE2ERepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve live e2e repo root")
	}
	return filepath.Dir(filepath.Dir(file))
}

func liveE2EEnv(t *testing.T, configPath string, krxAuthKey string) []string {
	t.Helper()
	env := []string{
		"HOME=" + t.TempDir(),
		"MWOSA_CONFIG=" + configPath,
		"MWOSA_KRX_AUTH_KEY=" + krxAuthKey,
		"MWOSA_LIVE_E2E=1",
	}
	for _, key := range []string{
		"PATH",
		"TMPDIR",
		"MWOSA_KRX_BASE_URL",
		"MWOSA_KRX_SAMPLE_BASE_URL",
		"MWOSA_KRX_USE_SAMPLE",
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
		"http_proxy",
		"https_proxy",
		"no_proxy",
		"SSL_CERT_FILE",
		"SSL_CERT_DIR",
	} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func decodeLiveE2EJSON[T any](t *testing.T, label string, output string, secrets []string) T {
	t.Helper()
	var value T
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("decode %s JSON: %v\n%s", label, err, redactLiveE2EOutput(output, secrets))
	}
	return value
}

func requireLiveE2EPlanStage(t *testing.T, stages []liveE2EPlanStage, name string, stageType string, provider string, operation string) {
	t.Helper()
	for _, stage := range stages {
		if stage.Name != name {
			continue
		}
		if stage.Type != stageType {
			t.Fatalf("plan stage %s type = %q, want %q", name, stage.Type, stageType)
		}
		if provider != "" && stage.Provider != provider {
			t.Fatalf("plan stage %s provider = %q, want %q", name, stage.Provider, provider)
		}
		if operation != "" && stage.Operation != operation {
			t.Fatalf("plan stage %s operation = %q, want %q", name, stage.Operation, operation)
		}
		return
	}
	t.Fatalf("plan stage %s not found", name)
}

func requireLiveE2ERun(t *testing.T, runs []liveE2ERun, alias string) liveE2ERun {
	t.Helper()
	for _, run := range runs {
		if run.Alias == alias {
			return run
		}
	}
	t.Fatalf("aggregate run alias %s not found in history", alias)
	return liveE2ERun{}
}

func requireLiveE2ESucceededStage(t *testing.T, stages []liveE2EStageSummary, name string, minRows int) {
	t.Helper()
	for _, stage := range stages {
		if stage.Name != name {
			continue
		}
		if stage.Status != "succeeded" {
			t.Fatalf("stage %s status = %q, want succeeded", name, stage.Status)
		}
		if stage.Rows < minRows {
			t.Fatalf("stage %s rows = %d, want >= %d", name, stage.Rows, minRows)
		}
		return
	}
	t.Fatalf("stage %s not found", name)
}

func containsLiveE2ESecret(output string, secrets []string) bool {
	for _, secret := range secrets {
		if secret != "" && strings.Contains(output, secret) {
			return true
		}
	}
	return false
}

func redactLiveE2EOutput(output string, secrets []string) string {
	redacted := output
	for _, secret := range secrets {
		if secret != "" {
			redacted = strings.ReplaceAll(redacted, secret, "<redacted>")
		}
	}
	return redacted
}

func publicLiveE2EArgs(args []string) string {
	out := make([]string, 0, len(args))
	hideNext := false
	for _, arg := range args {
		if hideNext {
			out = append(out, "<redacted>")
			hideNext = false
			continue
		}
		out = append(out, arg)
		switch arg {
		case "--config", "--database-url", "--mongodb-uri":
			hideNext = true
		}
	}
	return strings.Join(out, " ")
}

type liveE2EStorageInit struct {
	Status  string `json:"status"`
	Backend string `json:"backend"`
}

type liveE2EValidation struct {
	Valid bool `json:"valid"`
}

type liveE2EAggregateDetail struct {
	Aggregate struct {
		Name string `json:"name"`
	} `json:"aggregate"`
}

type liveE2EPlanStage struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	Operation string `json:"operation"`
}

type liveE2ERun struct {
	Alias       string `json:"alias"`
	Status      string `json:"status"`
	ResultCount int    `json:"result_count"`
}

type liveE2EStageSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Rows   int    `json:"rows"`
}
