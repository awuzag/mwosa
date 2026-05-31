package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/macro"
	"github.com/awuzag/mwosa/storage"
	macrostorage "github.com/awuzag/mwosa/storage/macro"
)

func TestMacroCommandsAreRegistered(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	for _, args := range [][]string{
		{"list", "macro-indicators"},
		{"get", "macro", "ecos.base-rate"},
		{"sync", "macro", "key-statistics"},
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

func TestMacroCLIReadsStoredIndicatorsAndObservations(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	_, writer, err := macrostorage.NewRepository(database)
	if err != nil {
		t.Fatalf("new macro repository: %v", err)
	}
	if _, err := writer.UpsertIndicators(ctx, []macro.Indicator{{
		ID:           "ecos.base-rate",
		Preset:       macro.PresetKeyStatistics,
		Provider:     provider.ProviderECOS,
		SourceCode:   "722Y001",
		SourceName:   "ECOS key statistics",
		Name:         "Bank of Korea base rate",
		FriendlyName: "base rate",
		Category:     "rates",
		Frequency:    macro.FrequencyMonthly,
		Unit:         "%",
		Scale:        "percent",
		Active:       true,
	}}); err != nil {
		t.Fatalf("seed macro indicator: %v", err)
	}
	if _, err := writer.UpsertObservations(ctx, []macro.Observation{{
		IndicatorID: "ecos.base-rate",
		Provider:    provider.ProviderECOS,
		SourceCode:  "722Y001",
		Period:      "2024-04",
		Value:       "3.50",
		PublishedAt: "2024-04-11",
		CollectedAt: "2024-04-12T00:00:00Z",
	}}); err != nil {
		t.Fatalf("seed macro observation: %v", err)
	}

	var listOut bytes.Buffer
	listCmd := NewRootCommand(BuildInfo{})
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	if err := executeForTest(t, ctx, listCmd,
		"--database", databasePath,
		"-o", "json",
		"list", "macro-indicators",
		"--preset", "key-statistics",
	); err != nil {
		t.Fatalf("list macro-indicators: %v\n%s", err, listOut.String())
	}
	var indicators []struct {
		IndicatorID string `json:"indicator_id"`
		SourceCode  string `json:"source_code"`
	}
	if err := json.Unmarshal(listOut.Bytes(), &indicators); err != nil {
		t.Fatalf("decode list output: %v\n%s", err, listOut.String())
	}
	if len(indicators) != 1 || indicators[0].IndicatorID != "ecos.base-rate" || indicators[0].SourceCode != "722Y001" {
		t.Fatalf("indicators = %+v", indicators)
	}

	var getOut bytes.Buffer
	getCmd := NewRootCommand(BuildInfo{})
	getCmd.SetOut(&getOut)
	getCmd.SetErr(&getOut)
	if err := executeForTest(t, ctx, getCmd,
		"--database", databasePath,
		"-o", "table",
		"get", "macro", "ecos.base-rate",
		"--from", "2024-04",
		"--to", "2024-04",
	); err != nil {
		t.Fatalf("get macro: %v\n%s", err, getOut.String())
	}
	for _, want := range []string{"ecos.base-rate", "2024-04", "3.50"} {
		if !strings.Contains(getOut.String(), want) {
			t.Fatalf("get output missing %q in:\n%s", want, getOut.String())
		}
	}
}
