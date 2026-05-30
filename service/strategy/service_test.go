package strategy

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	universecore "github.com/awuzag/mwosa/packages/universe"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
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

func TestDailyBarDatasetReaderEnrichesStockDailyMetrics(t *testing.T) {
	ctx := context.Background()
	per := int64(120000)
	reader, err := NewDailyBarDatasetReaderWithFundamentals(
		fakeDailyReadRepository{bars: []dailybar.Bar{
			{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				TradingDate:  "2026-05-16",
				Symbol:       "005930",
				Close:        "70000",
			},
		}},
		fakeFundamentalsRepository{items: map[string]Fundamentals{
			"005930": {
				Symbol: "005930",
				Metrics: map[string]FundamentalMetric{
					"roe": {FiscalYear: "2025", ValueBP: int64Ptr(1800)},
				},
				Valuation: &FundamentalValuation{AsOfDate: "2026-05-16", PerBP: &per},
				Facts: map[string]FundamentalFact{
					"audit_opinion": {
						FactType:   "audit_opinion",
						FiscalYear: "2025",
						Key:        "audit_opinion",
						ValueText:  "적정",
					},
				},
				Events: []FundamentalEvent{
					{
						EventType: "company_merger",
						EventDate: "2026-05-10",
						RceptNo:   "20260510000123",
						Title:     "합병 결정",
					},
				},
			},
		}},
		provider.MarketKRX,
	)
	require.NoError(t, err)

	dataset, err := reader.ReadDataset(ctx, "stock_daily_metrics")
	require.NoError(t, err)
	require.Len(t, dataset.Records, 1)
	var row map[string]any
	require.NoError(t, json.Unmarshal(dataset.Records[0], &row))
	require.Equal(t, "005930", row["symbol"])
	require.Contains(t, row, "financial_metrics")
	require.Contains(t, row, "valuation")
	require.Contains(t, row, "fundamental_scores")
	require.Contains(t, row, "company_facts")
	require.Contains(t, row, "company_events")

	scores, ok := row["fundamental_scores"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "fundamentals-score/v1", scores["score_version"])
	assert.Equal(t, float64(72), scores["quality_score"])
	assert.Equal(t, float64(82), scores["valuation_score"])
	assert.Contains(t, scores["uncomputable"], "growth_score")
}

func TestServiceComparesScreenStrategiesWithoutRecordingRuns(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryStrategyRepository()
	service, err := NewService(repo, fakeDatasetReader{})
	require.NoError(t, err)
	service.SetPipelineExecutor(fakePipelineExecutor{})

	_, err = service.Upsert(ctx, UpsertStrategyRequest{
		Name: "return-rank",
		Spec: yamlPipelineSpec("return-rank", "069500"),
	})
	require.NoError(t, err)
	_, err = service.Upsert(ctx, UpsertStrategyRequest{
		Name: "mdd-rank",
		Spec: yamlPipelineSpec("mdd-rank", "102110"),
	})
	require.NoError(t, err)

	result, err := service.CompareScreenStrategies(ctx, CompareScreenStrategiesRequest{
		Names: []string{"return-rank", "mdd-rank"},
		AsOf:  "2026-05-06",
		TopN:  5,
	})
	require.NoError(t, err)

	assert.Equal(t, "2026-05-06", result.AsOf)
	require.Len(t, result.Strategies, 2)
	assert.Equal(t, "2026-05-06", result.Strategies[0].DataAsOf)
	assert.Equal(t, []string{"069500"}, result.Strategies[0].TopSymbols)
	assert.NotNil(t, result.Strategies[0].Metrics.AverageReturn20D)
	require.Len(t, result.Overlaps, 1)
	assert.Equal(t, 0, result.Overlaps[0].Count)
	assert.Empty(t, repo.runs)
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

type fakeDailyReadRepository struct {
	bars []dailybar.Bar
}

func (r fakeDailyReadRepository) QueryDailyBars(_ context.Context, query daily.Query) ([]dailybar.Bar, error) {
	out := make([]dailybar.Bar, 0, len(r.bars))
	for _, bar := range r.bars {
		if query.SecurityType != "" && bar.SecurityType != query.SecurityType {
			continue
		}
		out = append(out, bar)
	}
	return out, nil
}

func (r fakeDailyReadRepository) SummarizeDailyBarStorage(context.Context, daily.Query) (daily.StorageSummaryResult, error) {
	return daily.StorageSummaryResult{}, nil
}

func (r fakeDailyReadRepository) QueryDailyBarCoverage(context.Context, daily.Query) (daily.CoverageResult, error) {
	return daily.CoverageResult{}, nil
}

type fakeFundamentalsRepository struct {
	items map[string]Fundamentals
}

func (r fakeFundamentalsRepository) ListLatestFundamentals(context.Context, FundamentalsQuery) (map[string]Fundamentals, error) {
	return r.items, nil
}

func int64Ptr(value int64) *int64 {
	return &value
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
		Rows:               []json.RawMessage{json.RawMessage(`{"symbol":"` + symbol + `","return_20d":0.12,"max_dd_20d":-0.04,"traded_amount":1000000}`)},
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
