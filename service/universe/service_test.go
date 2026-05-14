package universe

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	core "github.com/ev3rlit/mwosa/packages/universe"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnerInspectScreenPipelineUsesCanonicalDailyBars(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "screen.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(`kind: ScreenRun
schema_version: 1
data:
  market: krx
  security_type: etf
  as_of: "2024-04-16"
pipeline:
  - id: source.daily_bars
  - id: transform.latest_per_symbol
  - id: filter.include_symbols
    params:
      symbols: ["069500"]
`), 0o644))

	runner, err := NewRunner(fakeDailyBarRepository{rows: []dailybar.Bar{
		screenDailyBar("069500", "2024-04-15", "35120"),
		screenDailyBar("123456", "2024-04-15", "1000"),
	}}, nil, nil)
	require.NoError(t, err)

	result, err := runner.InspectScreenPipeline(ctx, yamlPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"069500"}, result.SelectedSymbols)
	assert.Equal(t, 1, result.ResultCount)
	require.Len(t, result.Explain.Steps, 3)
	assert.Equal(t, "source.daily_bars", result.Explain.Steps[0].ID)
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, "069500", result.Candidates[0].Symbol)
}

func TestRunnerExecutesStoredScreenStrategyPipeline(t *testing.T) {
	ctx := context.Background()
	runner, err := NewRunner(fakeDailyBarRepository{rows: []dailybar.Bar{
		screenDailyBar("069500", "2024-04-15", "35120"),
		screenDailyBar("123456", "2024-04-15", "1000"),
	}}, nil, nil)
	require.NoError(t, err)

	result, err := runner.ExecuteScreenStrategyPipeline(ctx, strategyservice.ScreenStrategySpec{
		Kind:          strategyservice.KindScreenStrategy,
		SchemaVersion: 1,
		Name:          "etf-uptrend",
		Engine:        strategyservice.EngineYAMLPipeline,
		Pipeline: &strategyservice.ScreenPipelineStrategySpec{
			Data: strategyservice.ScreenPipelineDataSpec{Market: "krx", SecurityType: "etf", AsOf: "2024-04-16"},
			Pipeline: []core.StepSpec{
				{ID: "source.daily_bars"},
				{ID: "transform.latest_per_symbol"},
				{ID: "filter.include_symbols", Params: map[string]any{"symbols": []any{"069500"}}},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "screen_pipeline", result.InputDataset)
	assert.Equal(t, "2024-04-16", result.DataAsOf)
	require.Len(t, result.Rows, 1)
	assert.Contains(t, string(result.Rows[0]), `"symbol":"069500"`)
	assert.Contains(t, string(result.Rows[0]), `"security_type":"etf"`)
}

func TestRunnerDailyBarsSourceCanLoadMixedSecurityTypesAndFilter(t *testing.T) {
	ctx := context.Background()
	runner, err := NewRunner(fakeDailyBarRepository{rows: []dailybar.Bar{
		screenDailyBarWithType("005930", provider.SecurityTypeStock, "2024-04-15", "70000"),
		screenDailyBarWithType("069500", provider.SecurityTypeETF, "2024-04-15", "35120"),
		screenDailyBarWithType("580001", provider.SecurityTypeETN, "2024-04-15", "10000"),
	}}, nil, nil)
	require.NoError(t, err)

	plan, err := core.Compile(core.PipelineSpec{
		Pipeline: []core.StepSpec{
			{ID: "source.daily_bars", Params: map[string]any{"lookback_days": 5}},
			{ID: "filter.security_type", Params: map[string]any{"value": "etf"}},
		},
	}, core.DataWindow{Market: "krx", From: mustDate("2024-04-16"), To: mustDate("2024-04-16")}, core.DefaultSelectorRegistry())
	require.NoError(t, err)

	explain, err := runner.Explain(ctx, ContextRequest{
		Market:   "krx",
		From:     mustDate("2024-04-16"),
		To:       mustDate("2024-04-16"),
		Pipeline: plan.Pipeline,
	}, plan)
	require.NoError(t, err)

	assert.Equal(t, []string{"069500"}, explain.SelectedSymbols)
	require.Len(t, explain.Snapshots, 1)
	require.Len(t, explain.Snapshots[0].Candidates, 1)
	assert.Equal(t, "etf", explain.Snapshots[0].Candidates[0].Fields["security_type"])
}

func TestRunnerReturnsSelectorValidationErrors(t *testing.T) {
	ctx := context.Background()
	yamlPath := filepath.Join(t.TempDir(), "bad-screen.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(`kind: ScreenRun
schema_version: 1
data:
  market: krx
  security_type: etf
  as_of: "2024-04-16"
pipeline:
  - id: filter.mystery
`), 0o644))

	runner, err := NewRunner(fakeDailyBarRepository{}, nil, nil)
	require.NoError(t, err)

	_, err = runner.InspectScreenPipeline(ctx, yamlPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "universe selector is not registered")
	assert.Contains(t, err.Error(), "filter.mystery")
}

func TestRunnerInspectMarketRegimeAndStrategySet(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	regimePath := filepath.Join(dir, "regime.yaml")
	require.NoError(t, os.WriteFile(regimePath, []byte(`kind: MarketRegime
schema_version: 1
name: us-growth-regime
spec:
  benchmark:
    symbol: "379810"
    market: krx
    security_type: etf
  evaluation:
    lookback_days: 10
    confirm_days: 7
  rules:
    - regime: uptrend
      when:
        return_20d_gte: 0.03
        close_above_ma20: true
        ma20_above_ma60: true
    - regime: sideways
      when:
        return_20d_between: [-0.03, 0.03]
`), 0o644))
	strategySetPath := filepath.Join(dir, "strategy-set.yaml")
	require.NoError(t, os.WriteFile(strategySetPath, []byte(`kind: StrategySet
schema_version: 1
name: etf-swing-by-regime
spec:
  regime: us-growth-regime
  regime_file: regime.yaml
  routes:
    uptrend:
      strategy: nasdaq-uptrend-swing
      version: latest
      min_confidence: 0.7
    sideways:
      strategy: defensive-income-low-vol
      spec_hash: sha256:fixed
`), 0o644))

	runner, err := NewRunner(fakeDailyBarRepository{rows: marketRegimeDailyBars("379810", "2024-01-01", 70)}, nil, nil)
	require.NoError(t, err)

	regime, err := runner.InspectMarketRegime(ctx, regimePath, "2024-03-10")
	require.NoError(t, err)
	assert.Equal(t, "uptrend", regime.Regime)
	assert.Equal(t, 1.0, regime.Confidence)
	assert.Equal(t, 10, regime.StableDays)
	assert.Len(t, regime.RecentRegimes, 10)
	assert.NotEmpty(t, regime.Evidence)

	selection, err := runner.InspectStrategySet(ctx, strategySetPath, "2024-03-10")
	require.NoError(t, err)
	assert.Equal(t, "nasdaq-uptrend-swing", selection.SelectedRoute.Strategy)
	assert.Equal(t, "latest", selection.SelectedRoute.Version)
	assert.Equal(t, 1.0, selection.Regime.Confidence)
	assert.NotEmpty(t, selection.Hints)
}

type fakeDailyBarRepository struct {
	rows []dailybar.Bar
}

func (r fakeDailyBarRepository) QueryDailyBars(_ context.Context, query daily.Query) ([]dailybar.Bar, error) {
	out := make([]dailybar.Bar, 0, len(r.rows))
	for _, row := range r.rows {
		if query.SecurityType != "" && row.SecurityType != query.SecurityType {
			continue
		}
		if query.From != "" && row.TradingDate < query.From {
			continue
		}
		if query.To != "" && row.TradingDate > query.To {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func screenDailyBar(symbol string, tradingDate string, closePrice string) dailybar.Bar {
	return screenDailyBarWithType(symbol, provider.SecurityTypeETF, tradingDate, closePrice)
}

func screenDailyBarWithType(symbol string, securityType provider.SecurityType, tradingDate string, closePrice string) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderDataGo,
		Group:        provider.GroupSecuritiesProductPrice,
		Operation:    provider.OperationGetETFPriceInfo,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		Symbol:       symbol,
		TradingDate:  tradingDate,
		Close:        closePrice,
	}
}

func marketRegimeDailyBars(symbol string, start string, count int) []dailybar.Bar {
	startDate := mustDate(start)
	out := make([]dailybar.Bar, 0, count)
	for i := 0; i < count; i++ {
		closePrice := strconv.Itoa(100 + i)
		out = append(out, dailybar.Bar{
			Provider:     provider.ProviderDataGo,
			Group:        provider.GroupSecuritiesProductPrice,
			Operation:    provider.OperationGetETFPriceInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       symbol,
			TradingDate:  startDate.AddDate(0, 0, i).Format(time.DateOnly),
			Open:         closePrice,
			High:         closePrice,
			Low:          closePrice,
			Close:        closePrice,
		})
	}
	return out
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
