package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/providerraw"
)

func TestGetProviderRawSnapshotsReadsStoredPayload(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	database := storage.NewDatabase(databasePath)
	repository, err := providerraw.NewRepository(database)
	if err != nil {
		t.Fatalf("provider raw repository: %v", err)
	}
	_, err = repository.UpsertSnapshot(ctx, providerraw.Snapshot{
		Provider:         provider.ProviderOpenDART,
		Group:            provider.GroupOpenDARTDisclosure,
		Operation:        provider.OperationOpenDARTCorpCode,
		BaseDate:         "2026-05-17",
		CanonicalSupport: "company_registry",
		Rows: []map[string]string{
			{"corp_code": "00126380", "corp_name": "삼성전자"},
		},
		RowCount: 1,
	})
	if err != nil {
		t.Fatalf("seed provider raw snapshot: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"get", "provider-raw-snapshots",
		"--operation", "corpCode",
		"--include-payload",
	); err != nil {
		t.Fatalf("get provider-raw-snapshots: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider": "opendart"`, `"operation": "corpCode"`, `"base_date": "2026-05-17"`, `"canonical_support": "company_registry"`, `"payload"`, `"corp_code": "00126380"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("provider raw snapshot output missing %q in:\n%s", want, out.String())
		}
	}
}

func TestGetProviderRawAliasReadsStoredPayloadByProviderAndOperation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	seedProviderRawSnapshot(t, ctx, databasePath)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, ctx, cmd,
		"--config", filepath.Join(dir, "config.json"),
		"--database", databasePath,
		"--output", "json",
		"get", "provider-raw", "opendart", "corpCode",
		"--include-payload",
	); err != nil {
		t.Fatalf("get provider-raw: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider": "opendart"`, `"operation": "corpCode"`, `"payload"`, `"corp_code": "00126380"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("provider raw alias output missing %q in:\n%s", want, out.String())
		}
	}
}

func TestGetProviderRawAliasRejectsConflictingProvider(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := executeForTest(t, context.Background(), cmd,
		"--database", filepath.Join(t.TempDir(), "mwosa.db"),
		"--provider", "opendart",
		"get", "provider-raw", "datago", "corpCode",
	)
	if err == nil {
		t.Fatalf("expected provider conflict error")
	}
	if !strings.Contains(err.Error(), "provider raw provider flag conflicts") {
		t.Fatalf("unexpected provider conflict error: %v", err)
	}
}

func seedProviderRawSnapshot(t *testing.T, ctx context.Context, databasePath string) {
	t.Helper()
	database := storage.NewDatabase(databasePath)
	repository, err := providerraw.NewRepository(database)
	if err != nil {
		t.Fatalf("provider raw repository: %v", err)
	}
	_, err = repository.UpsertSnapshot(ctx, providerraw.Snapshot{
		Provider:         provider.ProviderOpenDART,
		Group:            provider.GroupOpenDARTDisclosure,
		Operation:        provider.OperationOpenDARTCorpCode,
		BaseDate:         "2026-05-17",
		CanonicalSupport: "company_registry",
		Rows: []map[string]string{
			{"corp_code": "00126380", "corp_name": "삼성전자"},
		},
		RowCount: 1,
	})
	if err != nil {
		t.Fatalf("seed provider raw snapshot: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
}
