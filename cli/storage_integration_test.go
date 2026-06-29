//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
)

func TestInitAndDoctorStorageMongoDB(t *testing.T) {
	server := integrationtest.StartMongoDB(t)
	t.Setenv("MWOSA_DATABASE_URL", server.URI)
	configPath := t.TempDir() + "/config.json"
	databaseName := "mwosa_cli_test"

	initCmd := NewRootCommand(BuildInfo{})
	var initOut bytes.Buffer
	initCmd.SetOut(&initOut)
	initCmd.SetErr(&initOut)
	initCmd.SetArgs([]string{
		"--config", configPath,
		"init", "storage",
		"--mongodb-database", databaseName,
	})
	requireExecute(t, initCmd, &initOut)

	var initResult storageInitResult
	if err := json.Unmarshal(initOut.Bytes(), &initResult); err != nil {
		t.Fatalf("decode init storage output: %v\n%s", err, initOut.String())
	}
	if initResult.Status != "ok" || initResult.Database != databaseName {
		t.Fatalf("init storage result = %#v, want ok %s", initResult, databaseName)
	}
	if initResult.Collections != len(storagemongodb.CollectionSpecs()) {
		t.Fatalf("init collections = %d, want %d", initResult.Collections, len(storagemongodb.CollectionSpecs()))
	}

	idempotentInitCmd := NewRootCommand(BuildInfo{})
	var idempotentInitOut bytes.Buffer
	idempotentInitCmd.SetOut(&idempotentInitOut)
	idempotentInitCmd.SetErr(&idempotentInitOut)
	idempotentInitCmd.SetArgs([]string{
		"--config", configPath,
		"init", "storage",
		"--mongodb-database", databaseName,
	})
	requireExecute(t, idempotentInitCmd, &idempotentInitOut)

	doctorCmd := NewRootCommand(BuildInfo{})
	var doctorOut bytes.Buffer
	doctorCmd.SetOut(&doctorOut)
	doctorCmd.SetErr(&doctorOut)
	doctorCmd.SetArgs([]string{
		"--config", configPath,
		"doctor", "storage",
		"--mongodb-database", databaseName,
	})
	requireExecute(t, doctorCmd, &doctorOut)

	var doctorResult storageDoctorResult
	if err := json.Unmarshal(doctorOut.Bytes(), &doctorResult); err != nil {
		t.Fatalf("decode doctor storage output: %v\n%s", err, doctorOut.String())
	}
	if doctorResult.Ping.Status != "ok" {
		t.Fatalf("doctor ping = %#v, want ok", doctorResult.Ping)
	}
	if doctorResult.Server.Version == "" {
		t.Fatalf("doctor server version is empty: %#v", doctorResult.Server)
	}
	for _, spec := range storagemongodb.CollectionSpecs() {
		status, ok := doctorResult.Collections[spec.Name]
		if !ok {
			t.Fatalf("doctor missing collection %s in %#v", spec.Name, doctorResult.Collections)
		}
		if status.Status != "ok" {
			t.Fatalf("doctor collection %s = %#v, want ok", spec.Name, status)
		}
		if !status.Validator {
			t.Fatalf("doctor collection %s missing validator in %#v", spec.Name, status)
		}
		if !status.Indexes["_id_"] {
			t.Fatalf("doctor collection %s missing _id_ index in %#v", spec.Name, status.Indexes)
		}
		for _, index := range spec.Indexes {
			if !status.Indexes[index.Name] {
				t.Fatalf("doctor collection %s missing index %s in %#v", spec.Name, index.Name, status.Indexes)
			}
		}
	}
}

func requireExecute(t *testing.T, cmd interface {
	ExecuteContext(context.Context) error
}, out *bytes.Buffer) {
	t.Helper()

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute command: %v\n%s", err, out.String())
	}
}
