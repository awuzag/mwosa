package backtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceUpsertsSavedStrategyFromCanonicalSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	repo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(&recordingDailyBarRepository{}, repo)
	require.NoError(t, err)

	detail, err := service.UpsertStrategy(context.Background(), SaveStrategyRequest{
		Name:     "sma-cross",
		YAMLPath: path,
	})
	require.NoError(t, err)

	assert.Equal(t, "sma-cross", detail.Strategy.Name)
	assert.Equal(t, 1, detail.ActiveVersion.Version)
	assert.Equal(t, core.SchemaVersion, detail.ActiveVersion.SchemaVersion)
	assert.NotEmpty(t, detail.ActiveVersion.SpecHash)
	assert.Equal(t, "sma", detail.Spec.Indicators["trend"].ID)

	var stored core.StrategySpec
	require.NoError(t, json.Unmarshal(repo.versions[0].SpecJSON, &stored))
	assert.Equal(t, core.KindStrategy, stored.Kind)
	assert.Equal(t, "sma-cross", stored.Name)
	assert.Equal(t, "crosses_above", stored.Entry.Operator)
}

func TestServiceRejectsSavedStrategyNameMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))

	service, err := NewServiceWithRepository(&recordingDailyBarRepository{}, newMemoryBacktestStrategyRepository())
	require.NoError(t, err)

	_, err = service.UpsertStrategy(context.Background(), SaveStrategyRequest{
		Name:     "other-name",
		YAMLPath: path,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strategy name mismatch")
	assert.Contains(t, err.Error(), "other-name")
	assert.Contains(t, err.Error(), "sma-cross")
}

func TestServiceSavedStrategyLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sma-cross.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML()), 0o644))
	updatedPath := filepath.Join(t.TempDir(), "sma-cross-updated.yaml")
	require.NoError(t, os.WriteFile(updatedPath, []byte(sampleUpdatedStrategyYAML()), 0o644))

	repo := newMemoryBacktestStrategyRepository()
	service, err := NewServiceWithRepository(&recordingDailyBarRepository{}, repo)
	require.NoError(t, err)

	created, err := service.UpsertStrategy(context.Background(), SaveStrategyRequest{Name: "sma-cross", YAMLPath: path})
	require.NoError(t, err)
	assert.Equal(t, 1, created.ActiveVersion.Version)

	listed, err := service.ListStrategies(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "sma-cross", listed[0].Strategy.Name)

	inspected, err := service.InspectStrategy(context.Background(), "sma-cross")
	require.NoError(t, err)
	assert.Equal(t, "sma-cross", inspected.Spec.Name)

	updated, err := service.UpsertStrategy(context.Background(), SaveStrategyRequest{Name: "sma-cross", YAMLPath: updatedPath})
	require.NoError(t, err)
	assert.Equal(t, 2, updated.ActiveVersion.Version)
	assert.Equal(t, 25.0, updated.Spec.Sizing.Value)
	assert.NotEqual(t, created.ActiveVersion.ID, updated.ActiveVersion.ID)

	require.NoError(t, service.DeleteStrategy(context.Background(), "sma-cross"))
	listed, err = service.ListStrategies(context.Background())
	require.NoError(t, err)
	assert.Empty(t, listed)
	_, err = service.InspectStrategy(context.Background(), "sma-cross")
	require.Error(t, err)
}

type memoryBacktestStrategyRepository struct {
	strategies  map[string]SavedStrategy
	versions    []SavedStrategyVersion
	evaluations []SavedEvaluationDetail
}

func newMemoryBacktestStrategyRepository() *memoryBacktestStrategyRepository {
	return &memoryBacktestStrategyRepository{strategies: map[string]SavedStrategy{}}
}

func (r *memoryBacktestStrategyRepository) CreateStrategyWithVersion(_ context.Context, strategy SavedStrategy, version SavedStrategyVersion) (SavedStrategyDetail, error) {
	r.strategies[strategy.Name] = strategy
	r.versions = append(r.versions, version)
	return detailFromMemory(strategy, version)
}

func (r *memoryBacktestStrategyRepository) ListStrategies(_ context.Context) ([]SavedStrategyDetail, error) {
	out := make([]SavedStrategyDetail, 0, len(r.strategies))
	for _, strategy := range r.strategies {
		if strategy.DeletedAt != nil {
			continue
		}
		version := r.versionByID(strategy.ActiveVersionID)
		detail, err := detailFromMemory(strategy, version)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, nil
}

func (r *memoryBacktestStrategyRepository) GetStrategy(_ context.Context, name string) (SavedStrategyDetail, error) {
	strategy, ok := r.strategies[name]
	if !ok || strategy.DeletedAt != nil {
		return SavedStrategyDetail{}, assert.AnError
	}
	return detailFromMemory(strategy, r.versionByID(strategy.ActiveVersionID))
}

func (r *memoryBacktestStrategyRepository) AddStrategyVersion(_ context.Context, name string, version SavedStrategyVersion, now time.Time) (SavedStrategyDetail, error) {
	strategy, ok := r.strategies[name]
	if !ok || strategy.DeletedAt != nil {
		return SavedStrategyDetail{}, assert.AnError
	}
	version.StrategyID = strategy.ID
	strategy.ActiveVersionID = version.ID
	strategy.UpdatedAt = now
	r.strategies[name] = strategy
	r.versions = append(r.versions, version)
	return detailFromMemory(strategy, version)
}

func (r *memoryBacktestStrategyRepository) UpsertStrategyWithVersion(ctx context.Context, strategy SavedStrategy, version SavedStrategyVersion, now time.Time) (SavedStrategyDetail, error) {
	existing, ok := r.strategies[strategy.Name]
	if !ok || existing.DeletedAt != nil {
		version.Version = 1
		version.StrategyID = strategy.ID
		return r.CreateStrategyWithVersion(ctx, strategy, version)
	}
	version.StrategyID = existing.ID
	version.Version = r.versionByID(existing.ActiveVersionID).Version + 1
	return r.AddStrategyVersion(ctx, existing.Name, version, now)
}

func (r *memoryBacktestStrategyRepository) DeleteStrategy(_ context.Context, name string, deletedAt time.Time) error {
	strategy, ok := r.strategies[name]
	if !ok || strategy.DeletedAt != nil {
		return assert.AnError
	}
	strategy.DeletedAt = &deletedAt
	strategy.UpdatedAt = deletedAt
	r.strategies[name] = strategy
	return nil
}

func (r *memoryBacktestStrategyRepository) SaveEvaluation(_ context.Context, experiment SavedExperiment, cases []SavedExperimentCase, steps []SavedWalkForwardStep, now time.Time) (SavedEvaluationDetail, error) {
	if experiment.CreatedAt.IsZero() {
		experiment.CreatedAt = now
	}
	for index := range cases {
		if cases[index].CreatedAt.IsZero() {
			cases[index].CreatedAt = now
		}
	}
	for index := range steps {
		if steps[index].CreatedAt.IsZero() {
			steps[index].CreatedAt = now
		}
	}
	detail := SavedEvaluationDetail{Experiment: experiment, Cases: cases, WalkForward: steps}
	r.evaluations = append(r.evaluations, detail)
	return detail, nil
}

func (r *memoryBacktestStrategyRepository) ListEvaluations(context.Context) ([]SavedEvaluationSummary, error) {
	out := make([]SavedEvaluationSummary, 0, len(r.evaluations))
	for _, detail := range r.evaluations {
		var best *SavedExperimentCase
		for index := range detail.Cases {
			if detail.Cases[index].Rank == 1 {
				item := detail.Cases[index]
				best = &item
				break
			}
		}
		out = append(out, SavedEvaluationSummary{
			Experiment: detail.Experiment,
			CaseCount:  len(detail.Cases),
			BestCase:   best,
		})
	}
	return out, nil
}

func (r *memoryBacktestStrategyRepository) GetEvaluation(_ context.Context, ref string) (SavedEvaluationDetail, error) {
	for index := len(r.evaluations) - 1; index >= 0; index-- {
		detail := r.evaluations[index]
		if detail.Experiment.ID == ref || detail.Experiment.Name == ref {
			return detail, nil
		}
	}
	return SavedEvaluationDetail{}, assert.AnError
}

func (r *memoryBacktestStrategyRepository) versionByID(id string) SavedStrategyVersion {
	for _, version := range r.versions {
		if version.ID == id {
			return version
		}
	}
	return SavedStrategyVersion{}
}

func detailFromMemory(strategy SavedStrategy, version SavedStrategyVersion) (SavedStrategyDetail, error) {
	var spec core.StrategySpec
	if err := json.Unmarshal(version.SpecJSON, &spec); err != nil {
		return SavedStrategyDetail{}, err
	}
	return SavedStrategyDetail{Strategy: strategy, ActiveVersion: version, Spec: spec}, nil
}

func sampleUpdatedStrategyYAML() string {
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
      window: 3
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
  value: 25
risk:
  max_positions: 1
  max_symbol_weight_pct: 50
`
}
