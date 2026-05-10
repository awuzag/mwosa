package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBacktestStrategyCLILifecycle(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	yamlPath := filepath.Join(t.TempDir(), "sma-cross.yaml")
	requireWriteFile(t, yamlPath, sampleBacktestYAML())
	updatedPath := filepath.Join(t.TempDir(), "sma-cross-updated.yaml")
	requireWriteFile(t, updatedPath, sampleBacktestStrategyOnlyYAML(25))

	createOut := executeBacktestStrategyCommand(t, ctx, databasePath,
		"update", "backtest", "strategy", "sma-cross", "--yaml-file", yamlPath, "-o", "json",
	)
	var created struct {
		Name          string         `json:"name"`
		Version       int            `json:"version"`
		SchemaVersion int            `json:"schema_version"`
		Indicators    map[string]any `json:"indicators"`
		SpecHash      string         `json:"spec_hash"`
	}
	require.NoError(t, json.Unmarshal(createOut, &created))
	assert.Equal(t, "sma-cross", created.Name)
	assert.Equal(t, 1, created.Version)
	assert.Equal(t, 1, created.SchemaVersion)
	assert.Contains(t, created.Indicators, "trend")
	assert.NotEmpty(t, created.SpecHash)

	listOut := executeBacktestStrategyCommand(t, ctx, databasePath,
		"list", "backtest", "strategies", "-o", "json",
	)
	var listed []struct {
		Name    string `json:"name"`
		Version int    `json:"version"`
	}
	require.NoError(t, json.Unmarshal(listOut, &listed))
	require.Len(t, listed, 1)
	assert.Equal(t, "sma-cross", listed[0].Name)

	inspectOut := executeBacktestStrategyCommand(t, ctx, databasePath,
		"inspect", "backtest", "strategy", "sma-cross", "-o", "json",
	)
	var inspected struct {
		Name string `json:"name"`
		Spec struct {
			Kind   string `json:"kind"`
			Name   string `json:"name"`
			Sizing struct {
				Value float64 `json:"value"`
			} `json:"sizing"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(inspectOut, &inspected))
	assert.Equal(t, "sma-cross", inspected.Name)
	assert.Equal(t, "Strategy", inspected.Spec.Kind)
	assert.Equal(t, "sma-cross", inspected.Spec.Name)
	assert.Equal(t, 50.0, inspected.Spec.Sizing.Value)

	updateOut := executeBacktestStrategyCommand(t, ctx, databasePath,
		"update", "backtest", "strategy", "sma-cross", "--yaml-file", updatedPath, "-o", "json",
	)
	var updated struct {
		Name    string `json:"name"`
		Version int    `json:"version"`
		Spec    struct {
			Sizing struct {
				Value float64 `json:"value"`
			} `json:"sizing"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(updateOut, &updated))
	assert.Equal(t, "sma-cross", updated.Name)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, 25.0, updated.Spec.Sizing.Value)

	deleteOut := executeBacktestStrategyCommand(t, ctx, databasePath,
		"delete", "backtest", "strategy", "sma-cross", "-o", "json",
	)
	var deleted struct {
		Name    string `json:"name"`
		Deleted bool   `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(deleteOut, &deleted))
	assert.Equal(t, "sma-cross", deleted.Name)
	assert.True(t, deleted.Deleted)

	listOut = executeBacktestStrategyCommand(t, ctx, databasePath,
		"list", "backtest", "strategies", "-o", "json",
	)
	require.NoError(t, json.Unmarshal(listOut, &listed))
	assert.Empty(t, listed)
}

func executeBacktestStrategyCommand(t *testing.T, ctx context.Context, databasePath string, args ...string) []byte {
	t.Helper()
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	require.NoError(t, executeForTest(t, ctx, cmd, append([]string{"--database", databasePath}, args...)...), out.String())
	return out.Bytes()
}

func sampleBacktestStrategyOnlyYAML(sizing float64) string {
	return `
kind: Strategy
schema_version: 1
name: sma-cross
indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 2
entry:
  crosses_above:
    - price: close
    - ref: trend
exit:
  crosses_below:
    - price: close
    - ref: trend
sizing:
  type: percent_of_equity
  value: ` + fmt.Sprintf("%.0f", sizing) + `
risk:
  max_positions: 1
  max_symbol_weight_pct: 60
`
}
