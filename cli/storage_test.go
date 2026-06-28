package cli

import (
	"testing"
)

func TestStorageCommandsAreRegistered(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	for _, args := range [][]string{
		{"init", "storage"},
		{"doctor", "storage"},
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

func TestInitStorageRequiresMongoDBURI(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetArgs([]string{"--config", t.TempDir() + "/config.json", "init", "storage", "--mongodb-database", "mwosa"})

	err := cmd.Execute()

	if err == nil {
		t.Fatal("init storage error = nil, want missing URI error")
	}
}
