package strategy

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	universecore "github.com/ev3rlit/mwosa/packages/universe"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceStoresAndRunsYAMLPipelineStrategyByVersionAndHash(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryStrategyRepository()
	service, err := NewService(repo, fakeDatasetReader{})
	require.NoError(t, err)
	service.SetPipelineExecutor(fakePipelineExecutor{})

	v1, err := service.Upsert(ctx, UpsertStrategyRequest{
		Name: "etf-uptrend",
		Spec: yamlPipelineSpec("etf-uptrend", "069500"),
	})
	require.NoError(t, err)
	v2, err := service.Upsert(ctx, UpsertStrategyRequest{
		Name: "etf-uptrend",
		Spec: yamlPipelineSpec("etf-uptrend", "102110"),
	})
	require.NoError(t, err)
	require.Equal(t, 2, v2.ActiveVersion.Version)
	require.NotEqual(t, v1.ActiveVersion.SpecHash, v2.ActiveVersion.SpecHash)

	latest, err := service.Screen(ctx, ScreenStrategyRequest{Name: "etf-uptrend", Alias: "latest"})
	require.NoError(t, err)
	require.Len(t, latest.Items, 1)
	assert.Equal(t, "102110", latest.Items[0].Symbol)
	assert.Equal(t, v2.ActiveVersion.ID, latest.Run.StrategyVersionID)

	pinned, err := service.Screen(ctx, ScreenStrategyRequest{Name: "etf-uptrend", Alias: "pinned", SpecHash: v1.ActiveVersion.SpecHash})
	require.NoError(t, err)
	require.Len(t, pinned.Items, 1)
	assert.Equal(t, "069500", pinned.Items[0].Symbol)
	assert.Equal(t, v1.ActiveVersion.ID, pinned.Run.StrategyVersionID)
	assert.Equal(t, "screen_pipeline", pinned.Run.InputDataset)
	assert.Equal(t, "2024-04-16", pinned.Run.DataAsOf)
}

func TestServiceScreenStrategySpecHashIsDeterministic(t *testing.T) {
	spec := yamlPipelineSpec("etf-uptrend", "069500")
	_, first, err := canonicalStrategySpecPayload(spec)
	require.NoError(t, err)
	_, second, err := canonicalStrategySpecPayload(spec)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.True(t, strings.HasPrefix(first, "sha256:"))
}

func TestServiceJQStrategyStillScreensDataset(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryStrategyRepository()
	service, err := NewService(repo, fakeDatasetReader{records: []json.RawMessage{
		json.RawMessage(`{"symbol":"069500","score":1}`),
		json.RawMessage(`{"symbol":"102110","score":2}`),
	}})
	require.NoError(t, err)

	_, err = service.Create(ctx, CreateStrategyRequest{
		Name:         "jq-leaders",
		Engine:       EngineJQ,
		InputDataset: "etf_daily_metrics",
		QueryText:    `map(select(.symbol == "102110"))`,
	})
	require.NoError(t, err)
	detail, err := service.Screen(ctx, ScreenStrategyRequest{Name: "jq-leaders", Alias: "jq"})
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, "102110", detail.Items[0].Symbol)
	assert.NotEmpty(t, detail.StrategyVersion.SpecHash)
}

type memoryStrategyRepository struct {
	strategies map[string]Strategy
	versions   []StrategyVersion
	runs       []ScreenRunDetail
}

func newMemoryStrategyRepository() *memoryStrategyRepository {
	return &memoryStrategyRepository{strategies: map[string]Strategy{}}
}

func (r *memoryStrategyRepository) CreateStrategyWithVersion(_ context.Context, strategy Strategy, version StrategyVersion) (StrategyDetail, error) {
	r.strategies[strategy.Name] = strategy
	r.versions = append(r.versions, version)
	return StrategyDetail{Strategy: strategy, ActiveVersion: version}, nil
}

func (r *memoryStrategyRepository) ListStrategies(context.Context) ([]StrategyDetail, error) {
	out := make([]StrategyDetail, 0, len(r.strategies))
	for _, strategy := range r.strategies {
		if strategy.ArchivedAt != nil {
			continue
		}
		out = append(out, StrategyDetail{Strategy: strategy, ActiveVersion: r.versionByID(strategy.ActiveVersionID)})
	}
	return out, nil
}

func (r *memoryStrategyRepository) GetStrategy(_ context.Context, name string) (StrategyDetail, error) {
	strategy, ok := r.strategies[name]
	if !ok || strategy.ArchivedAt != nil {
		return StrategyDetail{}, oops.In("memory_strategy_repository").Errorf("strategy not found: %s", name)
	}
	return StrategyDetail{Strategy: strategy, ActiveVersion: r.versionByID(strategy.ActiveVersionID)}, nil
}

func (r *memoryStrategyRepository) GetStrategyVersion(ctx context.Context, name string, ref StrategyVersionRef) (StrategyDetail, error) {
	detail, err := r.GetStrategy(ctx, name)
	if err != nil {
		return StrategyDetail{}, err
	}
	if ref.SpecHash != "" {
		for _, version := range r.versions {
			if version.StrategyID == detail.Strategy.ID && version.SpecHash == ref.SpecHash {
				detail.ActiveVersion = version
				return detail, nil
			}
		}
		return StrategyDetail{}, assert.AnError
	}
	if ref.Version != "" && ref.Version != "latest" {
		for _, version := range r.versions {
			if version.StrategyID == detail.Strategy.ID && ref.Version == strconv.Itoa(version.Version) {
				detail.ActiveVersion = version
				return detail, nil
			}
		}
		return StrategyDetail{}, assert.AnError
	}
	return detail, nil
}

func (r *memoryStrategyRepository) AddStrategyVersion(_ context.Context, name string, engine Engine, version StrategyVersion, now time.Time) (StrategyDetail, error) {
	strategy, ok := r.strategies[name]
	if !ok || strategy.ArchivedAt != nil {
		return StrategyDetail{}, assert.AnError
	}
	version.StrategyID = strategy.ID
	strategy.Engine = engine
	strategy.ActiveVersionID = version.ID
	strategy.UpdatedAt = now
	r.strategies[name] = strategy
	r.versions = append(r.versions, version)
	return StrategyDetail{Strategy: strategy, ActiveVersion: version}, nil
}

func (r *memoryStrategyRepository) ArchiveStrategy(_ context.Context, name string, archivedAt time.Time) error {
	strategy, ok := r.strategies[name]
	if !ok {
		return assert.AnError
	}
	strategy.ArchivedAt = &archivedAt
	r.strategies[name] = strategy
	return nil
}

func (r *memoryStrategyRepository) CreateScreenRun(_ context.Context, run ScreenRun, items []ScreenRunItem) (ScreenRunDetail, error) {
	detail := ScreenRunDetail{
		Run:             run,
		Strategy:        r.strategyByID(run.StrategyID),
		StrategyVersion: r.versionByID(run.StrategyVersionID),
		Items:           items,
	}
	r.runs = append(r.runs, detail)
	return detail, nil
}

func (r *memoryStrategyRepository) ListScreenRuns(_ context.Context, _ int) ([]ScreenRun, error) {
	out := make([]ScreenRun, 0, len(r.runs))
	for _, detail := range r.runs {
		out = append(out, detail.Run)
	}
	return out, nil
}

func (r *memoryStrategyRepository) GetScreenRun(_ context.Context, ref string) (ScreenRunDetail, error) {
	for _, detail := range r.runs {
		if detail.Run.ID == ref || detail.Run.Alias == ref {
			return detail, nil
		}
	}
	return ScreenRunDetail{}, assert.AnError
}

func (r *memoryStrategyRepository) versionByID(id string) StrategyVersion {
	for _, version := range r.versions {
		if version.ID == id {
			return version
		}
	}
	return StrategyVersion{}
}

func (r *memoryStrategyRepository) strategyByID(id string) Strategy {
	for _, strategy := range r.strategies {
		if strategy.ID == id {
			return strategy
		}
	}
	return Strategy{}
}

type fakeDatasetReader struct {
	records []json.RawMessage
}

func (r fakeDatasetReader) ReadDataset(_ context.Context, name string) (Dataset, error) {
	return Dataset{Name: name, SchemaVersion: 1, Records: r.records}, nil
}

type fakePipelineExecutor struct{}

func (fakePipelineExecutor) ExecuteScreenStrategyPipeline(_ context.Context, spec ScreenStrategySpec) (PipelineExecutionResult, error) {
	symbol := "069500"
	for _, step := range spec.Pipeline.Pipeline {
		if step.ID != "filter.include_symbols" {
			continue
		}
		raw, ok := step.Params["symbols"].([]any)
		if ok && len(raw) > 0 {
			if value, ok := raw[0].(string); ok {
				symbol = value
			}
		}
	}
	return PipelineExecutionResult{
		InputDataset:       "screen_pipeline",
		InputSchemaVersion: spec.SchemaVersion,
		DataAsOf:           spec.Pipeline.Data.AsOf,
		Rows:               []json.RawMessage{json.RawMessage(`{"symbol":"` + symbol + `"}`)},
	}, nil
}

func yamlPipelineSpec(name string, symbol string) ScreenStrategySpec {
	return ScreenStrategySpec{
		Kind:          KindScreenStrategy,
		SchemaVersion: 1,
		Name:          name,
		Engine:        EngineYAMLPipeline,
		Pipeline: &ScreenPipelineStrategySpec{
			Data: ScreenPipelineDataSpec{Market: "krx", SecurityType: "etf", AsOf: "2024-04-16"},
			Pipeline: []universecore.StepSpec{
				{ID: "source.daily_bars"},
				{ID: "transform.latest_per_symbol"},
				{ID: "filter.include_symbols", Params: map[string]any{"symbols": []any{symbol}}},
			},
		},
	}
}
