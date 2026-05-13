package universe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniversePipelineSelectorFamilies(t *testing.T) {
	ctx := ExecutionContext{
		SelectionTime: date("2024-01-06"),
		From:          date("2024-01-02"),
		To:            date("2024-01-08"),
		DailyBars: []Bar{
			{Time: date("2024-01-02"), Symbol: "069500", Open: 10, High: 11, Low: 9, Close: 10, Volume: 100, TradedAmount: 1000},
			{Time: date("2024-01-03"), Symbol: "069500", Open: 10, High: 12, Low: 9, Close: 12, Volume: 200, TradedAmount: 2000},
			{Time: date("2024-01-04"), Symbol: "102110", Open: 20, High: 22, Low: 19, Close: 21, Volume: 300, TradedAmount: 3000},
			{Time: date("2024-01-05"), Symbol: "102110", Open: 21, High: 23, Low: 20, Close: 23, Volume: 400, TradedAmount: 4000},
			{Time: date("2024-01-06"), Symbol: "252670", Open: 30, High: 31, Low: 29, Close: 30, Volume: 500, TradedAmount: 5000},
		},
		Metadata: map[string]Candidate{
			"069500": {Symbol: "069500", Fields: map[string]any{"asset_class": "equity", "security_type": "etf", "market": "krx"}},
			"102110": {Symbol: "102110", Fields: map[string]any{"asset_class": "bond", "security_type": "etf", "market": "krx"}},
		},
		Watchlists: map[string][]Candidate{
			"core": {{Symbol: "069500", Fields: map[string]any{"watch": "core"}}},
		},
		SavedScreens: map[string][]Candidate{
			"screen": {{Symbol: "102110", Fields: map[string]any{"score": 9.0}}},
		},
		Files: map[string][]Candidate{
			"fixture.csv": {{Symbol: "252670", Fields: map[string]any{"score": 1.0}}},
		},
	}
	spec := PipelineSpec{Pipeline: []StepSpec{
		{ID: "combine.union", Params: map[string]any{"pipelines": []any{
			map[string]any{"name": "daily", "pipeline": []any{
				map[string]any{"id": "source.daily_bars", "params": map[string]any{"market": "krx", "security_type": "etf"}},
				map[string]any{"id": "transform.window_metrics", "params": map[string]any{"metrics": map[string]any{
					"return_2d": map[string]any{"id": "return", "params": map[string]any{"window": 2}},
				}}},
				map[string]any{"id": "transform.latest_per_symbol"},
				map[string]any{"id": "transform.join_metadata"},
				map[string]any{"id": "transform.normalize_fields"},
				map[string]any{"id": "transform.indicator", "params": map[string]any{"id": "sma", "field": "close", "window": 2, "output": "sma_2"}},
				map[string]any{"id": "filter.has_daily_bars", "params": map[string]any{"min_count": 2}},
				map[string]any{"id": "filter.field", "params": map[string]any{"field": "traded_amount", "op": "gte", "value": 1000}},
				map[string]any{"id": "filter.expr", "params": map[string]any{"all": []any{
					map[string]any{"gte": []any{map[string]any{"field": "return_2d"}, map[string]any{"value": 0}}},
				}}},
				map[string]any{"id": "filter.security_type", "params": map[string]any{"value": "etf"}},
				map[string]any{"id": "filter.market", "params": map[string]any{"value": "krx"}},
				map[string]any{"id": "filter.tags", "params": map[string]any{"include": []any{"liquid"}}},
				map[string]any{"id": "rank.weighted", "params": map[string]any{"fields": map[string]any{"return_2d": 1.0, "traded_amount": 0.001}, "order": "desc"}},
				map[string]any{"id": "rank.percentile", "params": map[string]any{"field": "score", "top": 1.0}},
				map[string]any{"id": "rank.group_top_n", "params": map[string]any{"group_field": "asset_class", "n": 1, "rank_field": "score", "order": "desc"}},
			}},
			map[string]any{"name": "watch", "pipeline": []any{
				map[string]any{"id": "source.watchlist", "params": map[string]any{"name": "core"}},
				map[string]any{"id": "transform.tag", "params": map[string]any{"tags": []any{"manual"}}},
			}},
			map[string]any{"name": "screen", "pipeline": []any{
				map[string]any{"id": "source.saved_screen", "params": map[string]any{"name": "screen"}},
			}},
			map[string]any{"name": "file", "pipeline": []any{
				map[string]any{"id": "source.file", "params": map[string]any{"path": "fixture.csv"}},
			}},
			map[string]any{"name": "inline", "pipeline": []any{
				map[string]any{"id": "source.inline", "params": map[string]any{"rows": []any{map[string]any{"symbol": "069500", "score": 3.0}}}},
			}},
		}}},
		{ID: "transform.distinct"},
		{ID: "filter.exclude_symbols", Params: map[string]any{"symbols": []any{"252670"}, "reason": "leveraged_or_inverse"}},
		{ID: "rank.round_robin", Params: map[string]any{"group_field": "asset_class", "limit": 10}},
		{ID: "limit.per_group", Params: map[string]any{"group_field": "asset_class", "count": 1}},
		{ID: "limit.max_count", Params: map[string]any{"count": 3}},
		{ID: "debug.snapshot", Params: map[string]any{"fields": []any{"score", "return_2d"}}},
		{ID: "debug.assert_count", Params: map[string]any{"min": 1, "max": 3}},
		{ID: "debug.sample", Params: map[string]any{"count": 2}},
		{ID: "limit.min_count", Params: map[string]any{"count": 1}},
	}}

	plan, err := Compile(spec, testDataWindow(), DefaultSelectorRegistry())
	require.NoError(t, err)
	snapshot, err := ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)

	assert.Equal(t, date("2024-01-06"), snapshot.Time)
	assert.NotEmpty(t, snapshot.Symbols)
	assert.LessOrEqual(t, len(snapshot.Symbols), 2)
	assert.NotContains(t, snapshot.Symbols, "252670")
	assert.NotEmpty(t, snapshot.Steps)
	assert.NotEmpty(t, snapshot.Decisions)
}

func TestUniverseCombineIntersectDifferenceConcat(t *testing.T) {
	registry := DefaultSelectorRegistry()
	ctx := ExecutionContext{}

	intersect := PipelineSpec{Pipeline: []StepSpec{{ID: "combine.intersect", Params: map[string]any{"pipelines": []any{
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"A", "B"}}}}},
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"B", "C"}}}}},
	}}}}}
	plan, err := Compile(intersect, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err := ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"B"}, snapshot.Symbols)

	difference := PipelineSpec{Pipeline: []StepSpec{{ID: "combine.difference", Params: map[string]any{"pipelines": []any{
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"A", "B"}}}}},
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"B"}}}}},
	}}}}}
	plan, err = Compile(difference, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err = ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, snapshot.Symbols)

	concat := PipelineSpec{Pipeline: []StepSpec{{ID: "combine.concat", Params: map[string]any{"pipelines": []any{
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"A"}}}}},
		map[string]any{"pipeline": []any{map[string]any{"id": "source.symbols", "params": map[string]any{"symbols": []any{"A"}}}}},
	}}}}}
	plan, err = Compile(concat, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err = ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "A"}, snapshot.Symbols)
}

func TestUniverseRemainingSelectorsAndScheduleVariants(t *testing.T) {
	registry := DefaultSelectorRegistry()
	ctx := ExecutionContext{
		SelectionTime: date("2024-01-10"),
		From:          date("2024-01-01"),
		To:            date("2024-01-10"),
		DailyBars: []Bar{
			{Time: date("2024-01-01"), Symbol: "A", Open: 10, High: 11, Low: 9, Close: 10, Volume: 100, TradedAmount: 1000},
			{Time: date("2024-01-02"), Symbol: "A", Open: 10, High: 12, Low: 8, Close: 12, Volume: 200, TradedAmount: 2000},
			{Time: date("2024-01-03"), Symbol: "B", Open: 20, High: 22, Low: 19, Close: 21, Volume: 300, TradedAmount: 3000},
		},
		Metadata: map[string]Candidate{
			"A": {Symbol: "A", Fields: map[string]any{"listed_at": "2023-01-01", "score": 1.0}},
			"B": {Symbol: "B", Fields: map[string]any{"listed_at": "2024-01-09", "score": 2.0}},
		},
		ScreenStrategies: map[string][]Candidate{
			"momentum": {
				{Symbol: "A", Fields: map[string]any{"score": 1.0, "volume": 1000.0, "traded_amount": 10000.0, "group": "x"}},
				{Symbol: "B", Fields: map[string]any{"score": 2.0, "volume": 1.0, "traded_amount": 1.0, "group": "y"}},
			},
		},
	}
	spec := PipelineSpec{Pipeline: []StepSpec{
		{ID: "source.screen_strategy", Params: map[string]any{"name": "momentum"}},
		{ID: "filter.include_symbols", Params: map[string]any{"symbols": []any{"A", "B"}}},
		{ID: "filter.liquidity", Params: map[string]any{"min_volume": 10, "min_traded_amount": 100}},
		{ID: "filter.listing_age", Params: map[string]any{"min_days": 5}},
		{ID: "rank.by_field", Params: map[string]any{"field": "score", "order": "desc"}},
		{ID: "limit.count", Params: map[string]any{"count": 1}},
	}}
	plan, err := Compile(spec, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err := ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, snapshot.Symbols)

	instrumentSpec := PipelineSpec{Pipeline: []StepSpec{{ID: "source.instrument_master"}}}
	plan, err = Compile(instrumentSpec, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err = ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, snapshot.Symbols)

	watchlistSpec := PipelineSpec{Pipeline: []StepSpec{{ID: "source.watchlist", Params: map[string]any{"symbols": []any{"A"}}}}}
	plan, err = Compile(watchlistSpec, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err = ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, snapshot.Symbols)

	failSpec := PipelineSpec{Pipeline: []StepSpec{
		{ID: "source.symbols", Params: map[string]any{"symbols": []any{"A", "B"}}},
		{ID: "limit.max_count", Params: map[string]any{"count": 1, "fail": true}},
	}}
	plan, err = Compile(failSpec, testDataWindow(), registry)
	require.NoError(t, err)
	_, err = ExecutePipeline(context.Background(), plan, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "above maximum")

	for _, schedule := range []ScheduleSpec{
		{Frequency: ScheduleWeekly},
		{Frequency: ScheduleMonthly},
		{Frequency: ScheduleCustomCalendar, Dates: []string{"2024-01-02", "2024-01-09"}},
	} {
		plan, err = Compile(PipelineSpec{
			Schedule: schedule,
			Pipeline: []StepSpec{{ID: "source.symbols", Params: map[string]any{"symbols": []any{"A"}}}},
		}, DataWindow{From: date("2024-01-01"), To: date("2024-01-10")}, registry)
		require.NoError(t, err)
		explain, err := BuildSnapshots(context.Background(), plan, ExecutionContext{From: date("2024-01-01"), To: date("2024-01-10")})
		require.NoError(t, err)
		assert.NotEmpty(t, explain.Snapshots)
	}
}

func TestUniverseExpressionOperatorsAndWindowMetrics(t *testing.T) {
	registry := DefaultSelectorRegistry()
	ctx := ExecutionContext{SelectionTime: date("2024-01-05")}
	spec := PipelineSpec{Pipeline: []StepSpec{
		{ID: "source.inline", Params: map[string]any{"rows": []any{
			map[string]any{"symbol": "A", "close": 10.0, "trading_date": "2024-01-01"},
			map[string]any{"symbol": "A", "close": 5.0, "trading_date": "2024-01-02"},
			map[string]any{"symbol": "A", "close": 8.0, "trading_date": "2024-01-03"},
			map[string]any{"symbol": "B", "close": 10.0, "trading_date": "2024-01-03"},
		}}},
		{ID: "transform.window_metrics", Params: map[string]any{"metrics": map[string]any{
			"avg_close": map[string]any{"id": "average", "params": map[string]any{"field": "close", "window": 3}},
			"mdd":       map[string]any{"id": "max_drawdown", "params": map[string]any{"field": "close", "window": 3}},
		}}},
		{ID: "transform.latest_per_symbol"},
		{ID: "filter.expr", Params: map[string]any{"any": []any{
			map[string]any{"between": []any{map[string]any{"field": "avg_close"}, []any{7.0, 8.0}}},
			map[string]any{"not": map[string]any{"eq": []any{map[string]any{"field": "close"}, map[string]any{"value": 10.0}}}},
		}}},
	}}
	plan, err := Compile(spec, testDataWindow(), registry)
	require.NoError(t, err)
	snapshot, err := ExecutePipeline(context.Background(), plan, ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, snapshot.Symbols)
	assert.Contains(t, snapshot.Candidates[0].Fields, "mdd")
}

func TestBuildSnapshotsUsesClosedDataOnly(t *testing.T) {
	spec := PipelineSpec{
		Schedule: ScheduleSpec{Frequency: ScheduleDaily},
		Pipeline: []StepSpec{
			{ID: "source.daily_bars"},
			{ID: "transform.latest_per_symbol"},
			{ID: "filter.field", Params: map[string]any{"field": "trading_date", "op": "lt", "value": "2024-01-03"}},
		},
	}
	plan, err := Compile(spec, testDataWindow(), DefaultSelectorRegistry())
	require.NoError(t, err)

	explain, err := BuildSnapshots(context.Background(), plan, ExecutionContext{
		From: date("2024-01-02"),
		To:   date("2024-01-04"),
		DailyBars: []Bar{
			{Time: date("2024-01-02"), Symbol: "A", Open: 1, High: 1, Low: 1, Close: 1},
			{Time: date("2024-01-03"), Symbol: "B", Open: 1, High: 1, Low: 1, Close: 1},
		},
	})
	require.NoError(t, err)

	require.Len(t, explain.Snapshots, 3)
	assert.Empty(t, explain.Snapshots[0].Symbols)
	assert.Equal(t, []string{"A"}, explain.Snapshots[1].Symbols)
	assert.Equal(t, []string{"A"}, explain.Snapshots[2].Symbols)
}

func TestSourceDailyBarsPreservesRowIdentityAndFiltersSecurityType(t *testing.T) {
	plan, err := Compile(PipelineSpec{Pipeline: []StepSpec{
		{ID: "source.daily_bars"},
	}}, testDataWindow(), DefaultSelectorRegistry())
	require.NoError(t, err)

	snapshot, err := ExecutePipeline(context.Background(), plan, ExecutionContext{
		SelectionTime: date("2024-01-04"),
		Market:        "krx",
		DailyBars: []Bar{
			{Time: date("2024-01-03"), Symbol: "005930", Market: "krx", SecurityType: "stock", Open: 1, High: 1, Low: 1, Close: 1},
			{Time: date("2024-01-03"), Symbol: "069500", Market: "krx", SecurityType: "etf", Open: 1, High: 1, Low: 1, Close: 1},
			{Time: date("2024-01-03"), Symbol: "580001", Market: "krx", SecurityType: "etn", Open: 1, High: 1, Low: 1, Close: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, snapshot.Candidates, 3)
	assert.Equal(t, "stock", snapshot.Candidates[0].Fields["security_type"])
	assert.Equal(t, "etf", snapshot.Candidates[1].Fields["security_type"])
	assert.Equal(t, "etn", snapshot.Candidates[2].Fields["security_type"])

	plan, err = Compile(PipelineSpec{Pipeline: []StepSpec{
		{ID: "source.daily_bars"},
		{ID: "filter.security_type", Params: map[string]any{"value": "etf"}},
	}}, testDataWindow(), DefaultSelectorRegistry())
	require.NoError(t, err)

	snapshot, err = ExecutePipeline(context.Background(), plan, ExecutionContext{
		SelectionTime: date("2024-01-04"),
		Market:        "krx",
		DailyBars: []Bar{
			{Time: date("2024-01-03"), Symbol: "005930", Market: "krx", SecurityType: "stock", Open: 1, High: 1, Low: 1, Close: 1},
			{Time: date("2024-01-03"), Symbol: "069500", Market: "krx", SecurityType: "etf", Open: 1, High: 1, Low: 1, Close: 1},
			{Time: date("2024-01-03"), Symbol: "580001", Market: "krx", SecurityType: "etn", Open: 1, High: 1, Low: 1, Close: 1},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"069500"}, snapshot.Symbols)
	require.Len(t, snapshot.Candidates, 1)
	assert.Equal(t, "etf", snapshot.Candidates[0].Fields["security_type"])
}

func testDataWindow() DataWindow {
	return DataWindow{Market: "krx", From: date("2024-01-02"), To: date("2024-01-08")}
}

func date(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
