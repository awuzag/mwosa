package backtest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	backtestservice "github.com/ev3rlit/mwosa/service/backtest"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositorySavedStrategyLifecycle(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	repository, err := NewRepository(database)
	require.NoError(t, err)

	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	spec := repositoryTestSpec(50)
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)
	created, err := repository.UpsertStrategyWithVersion(ctx, backtestservice.SavedStrategy{
		ID:              "strategy-1",
		Name:            "sma-cross",
		ActiveVersionID: "version-1",
		CreatedAt:       now,
		UpdatedAt:       now,
	}, backtestservice.SavedStrategyVersion{
		ID:            "version-1",
		StrategyID:    "strategy-1",
		Version:       1,
		SchemaVersion: core.SchemaVersion,
		SpecJSON:      specJSON,
		SpecHash:      "hash-1",
		CreatedAt:     now,
	}, now)
	require.NoError(t, err)
	assert.Equal(t, "sma-cross", created.Strategy.Name)
	assert.Equal(t, 1, created.ActiveVersion.Version)
	assert.Equal(t, spec.Name, created.Spec.Name)

	listed, err := repository.ListStrategies(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	inspected, err := repository.GetStrategy(ctx, "sma-cross")
	require.NoError(t, err)
	assert.JSONEq(t, string(specJSON), string(inspected.ActiveVersion.SpecJSON))

	updatedSpec := repositoryTestSpec(25)
	updatedJSON, err := json.Marshal(updatedSpec)
	require.NoError(t, err)
	updated, err := repository.UpsertStrategyWithVersion(ctx, backtestservice.SavedStrategy{
		ID:        "strategy-unused",
		Name:      "sma-cross",
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}, backtestservice.SavedStrategyVersion{
		ID:            "version-2",
		SchemaVersion: core.SchemaVersion,
		SpecJSON:      updatedJSON,
		SpecHash:      "hash-2",
		CreatedAt:     now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 2, updated.ActiveVersion.Version)
	assert.Equal(t, 25.0, updated.Spec.Sizing.Value)

	require.NoError(t, repository.DeleteStrategy(ctx, "sma-cross", now.Add(2*time.Minute)))
	listed, err = repository.ListStrategies(ctx)
	require.NoError(t, err)
	assert.Empty(t, listed)
	_, err = repository.GetStrategy(ctx, "sma-cross")
	require.Error(t, err)
}

func repositoryTestSpec(sizing float64) core.StrategySpec {
	return core.StrategySpec{
		Kind:          core.KindStrategy,
		SchemaVersion: core.SchemaVersion,
		Name:          "sma-cross",
		Indicators: map[string]core.IndicatorSpec{
			"trend": {
				ID:     "sma",
				Source: core.ValueExpr{Kind: "price", Price: "close"},
				Params: map[string]float64{"window": 2},
			},
		},
		Entry:  core.RuleExpr{Operator: "gt", Args: []core.ValueExpr{{Kind: "price", Price: "close"}, {Kind: "ref", Ref: "trend"}}},
		Exit:   core.RuleExpr{Operator: "lt", Args: []core.ValueExpr{{Kind: "price", Price: "close"}, {Kind: "ref", Ref: "trend"}}},
		Sizing: core.SizingSpec{Type: core.SizingPercentOfEquity, Value: sizing},
	}
}
