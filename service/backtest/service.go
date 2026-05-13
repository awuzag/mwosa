package backtest

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	"github.com/ev3rlit/mwosa/packages/hashutil"
	"github.com/ev3rlit/mwosa/packages/idgen"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	universeservice "github.com/ev3rlit/mwosa/service/universe"
	"github.com/samber/oops"
)

type DailyBarRepository interface {
	QueryDailyBars(ctx context.Context, query daily.Query) ([]dailybar.Bar, error)
}

type StrategyRepository interface {
	ListStrategies(ctx context.Context) ([]SavedStrategyDetail, error)
	GetStrategy(ctx context.Context, name string) (SavedStrategyDetail, error)
	UpsertStrategyWithVersion(ctx context.Context, strategy SavedStrategy, version SavedStrategyVersion, now time.Time) (SavedStrategyDetail, error)
	DeleteStrategy(ctx context.Context, name string, deletedAt time.Time) error
}

type ScreenRepository interface {
	GetScreenRun(ctx context.Context, ref string) (strategyservice.ScreenRunDetail, error)
}

type ScreenRunner interface {
	Screen(ctx context.Context, req strategyservice.ScreenStrategyRequest) (strategyservice.ScreenRunDetail, error)
}

type Service struct {
	reader       DailyBarRepository
	strategies   StrategyRepository
	screenRepo   ScreenRepository
	screenRunner ScreenRunner
	universe     universeservice.Runner
	registry     core.IndicatorRegistry
}

type SavedStrategy struct {
	ID              string     `json:"id" csv:"id"`
	Name            string     `json:"name" csv:"name"`
	ActiveVersionID string     `json:"active_version_id" csv:"active_version_id"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" csv:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" csv:"deleted_at"`
}

type SavedStrategyVersion struct {
	ID            string          `json:"id" csv:"id"`
	StrategyID    string          `json:"strategy_id" csv:"strategy_id"`
	Version       int             `json:"version" csv:"version"`
	SchemaVersion int             `json:"schema_version" csv:"schema_version"`
	SpecJSON      json.RawMessage `json:"spec_json" csv:"-"`
	SpecHash      string          `json:"spec_hash" csv:"spec_hash"`
	CreatedAt     time.Time       `json:"created_at" csv:"created_at"`
}

type SavedStrategyDetail struct {
	Strategy      SavedStrategy        `json:"strategy"`
	ActiveVersion SavedStrategyVersion `json:"active_version"`
	Spec          core.StrategySpec    `json:"spec"`
}

type SaveStrategyRequest struct {
	Name     string
	YAMLPath string
}

type ValidationResult struct {
	Valid        bool                 `json:"valid"`
	StrategyName string               `json:"strategy_name"`
	RunName      string               `json:"run_name"`
	Symbols      []string             `json:"symbols"`
	Period       core.Period          `json:"period"`
	Market       string               `json:"market"`
	SecurityType string               `json:"security_type"`
	Timeframe    string               `json:"timeframe"`
	Currency     string               `json:"currency"`
	Execution    ExecutionSummary     `json:"execution"`
	Metrics      []string             `json:"metrics"`
	Indicators   map[string]string    `json:"indicators,omitempty"`
	Universe     core.UniverseExplain `json:"universe"`
}

type ExecutionSummary struct {
	Fill       string        `json:"fill"`
	Commission core.CostSpec `json:"commission,omitempty"`
	Slippage   core.CostSpec `json:"slippage,omitempty"`
}

func NewService(reader DailyBarRepository) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, nil, nil, nil, registry)
}

func NewServiceWithRepository(reader DailyBarRepository, strategies StrategyRepository) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, strategies, nil, nil, registry)
}

func NewServiceWithUniverseSources(reader DailyBarRepository, strategies StrategyRepository, screenRepo ScreenRepository, screenRunner ScreenRunner) (Service, error) {
	registry, err := core.DefaultIndicatorRegistry()
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrapf(err, "create indicator registry")
	}
	return newService(reader, strategies, screenRepo, screenRunner, registry)
}

func NewServiceWithRegistry(reader DailyBarRepository, registry core.IndicatorRegistry) (Service, error) {
	return newService(reader, nil, nil, nil, registry)
}

func newService(reader DailyBarRepository, strategies StrategyRepository, screenRepo ScreenRepository, screenRunner ScreenRunner, registry core.IndicatorRegistry) (Service, error) {
	if reader == nil {
		return Service{}, oops.In("backtest_service").New("daily bar reader is nil")
	}
	universeRunner, err := universeservice.NewRunner(reader, screenRepo, screenRunner)
	if err != nil {
		return Service{}, oops.In("backtest_service").Wrap(err)
	}
	return Service{reader: reader, strategies: strategies, screenRepo: screenRepo, screenRunner: screenRunner, universe: universeRunner, registry: registry}, nil
}

func (s Service) Validate(ctx context.Context, path string) (ValidationResult, error) {
	_, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return ValidationResult{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return ValidationResult{}, err
	}
	return validationResultFromPlan(plan), nil
}

func (s Service) Run(ctx context.Context, path string) (core.Result, error) {
	bundle, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return core.Result{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return core.Result{}, err
	}
	bars, err := s.loadBars(ctx, plan)
	if err != nil {
		return core.Result{}, oops.In("backtest_service").With("strategy", bundle.Strategy.Name, "run", bundle.Run.Name).Wrap(err)
	}
	engine, err := core.NewEngine(core.NewMemoryFeed(bars))
	if err != nil {
		return core.Result{}, oops.In("backtest_service").With("strategy", bundle.Strategy.Name, "run", bundle.Run.Name).Wrap(err)
	}
	return engine.Run(ctx, plan)
}

func (s Service) InspectUniverse(ctx context.Context, path string) (core.UniverseExplain, error) {
	_, plan, err := s.compileFile(ctx, path)
	if err != nil {
		return core.UniverseExplain{}, err
	}
	plan, err = s.resolveUniverse(ctx, path, plan)
	if err != nil {
		return core.UniverseExplain{}, err
	}
	return plan.UniverseExplain, nil
}

func (s Service) UpsertStrategy(ctx context.Context, req SaveStrategyRequest) (SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_service").With("name", req.Name, "yaml_path", req.YAMLPath)
	if s.strategies == nil {
		return SavedStrategyDetail{}, errb.New("backtest strategy repository is nil")
	}
	spec, specJSON, specHash, err := s.loadCanonicalStrategy(ctx, req)
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrap(err)
	}
	strategyID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrapf(err, "generate backtest strategy id")
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return SavedStrategyDetail{}, errb.Wrapf(err, "generate backtest strategy version id")
	}
	now := time.Now()
	strategy := SavedStrategy{
		ID:              strategyID,
		Name:            spec.Name,
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := SavedStrategyVersion{
		ID:            versionID,
		StrategyID:    strategyID,
		Version:       1,
		SchemaVersion: spec.SchemaVersion,
		SpecJSON:      specJSON,
		SpecHash:      specHash,
		CreatedAt:     now,
	}
	return s.strategies.UpsertStrategyWithVersion(ctx, strategy, version, now)
}

func (s Service) ListStrategies(ctx context.Context) ([]SavedStrategyDetail, error) {
	if s.strategies == nil {
		return nil, oops.In("backtest_strategy_service").New("backtest strategy repository is nil")
	}
	return s.strategies.ListStrategies(ctx)
}

func (s Service) InspectStrategy(ctx context.Context, name string) (SavedStrategyDetail, error) {
	if strings.TrimSpace(name) == "" {
		return SavedStrategyDetail{}, oops.In("backtest_strategy_service").New("inspect backtest strategy requires name")
	}
	if s.strategies == nil {
		return SavedStrategyDetail{}, oops.In("backtest_strategy_service").With("name", name).New("backtest strategy repository is nil")
	}
	return s.strategies.GetStrategy(ctx, name)
}

func (s Service) DeleteStrategy(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return oops.In("backtest_strategy_service").New("delete backtest strategy requires name")
	}
	if s.strategies == nil {
		return oops.In("backtest_strategy_service").With("name", name).New("backtest strategy repository is nil")
	}
	return s.strategies.DeleteStrategy(ctx, name, time.Now())
}

func (s Service) loadCanonicalStrategy(ctx context.Context, req SaveStrategyRequest) (core.StrategySpec, json.RawMessage, string, error) {
	errb := oops.In("backtest_strategy_service").With("name", req.Name, "yaml_path", req.YAMLPath)
	if strings.TrimSpace(req.Name) == "" {
		return core.StrategySpec{}, nil, "", errb.New("backtest strategy name is required")
	}
	strategy, err := LoadStrategyFile(ctx, req.YAMLPath)
	if err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrap(err)
	}
	if strategy.Name != req.Name {
		return core.StrategySpec{}, nil, "", errb.With("yaml_name", strategy.Name).Errorf("backtest strategy name mismatch: cli=%s yaml=%s", req.Name, strategy.Name)
	}
	if err := core.ValidateStrategySpec(strategy, s.registry); err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrap(err)
	}
	specJSON, err := json.Marshal(strategy)
	if err != nil {
		return core.StrategySpec{}, nil, "", errb.Wrapf(err, "encode canonical backtest strategy spec")
	}
	return strategy, specJSON, hashutil.SHA256(specJSON), nil
}

func (s Service) compileFile(ctx context.Context, path string) (Bundle, core.StrategyPlan, error) {
	bundle, err := LoadFile(ctx, path)
	if err != nil {
		return Bundle{}, core.StrategyPlan{}, err
	}
	plan, err := core.Compile(bundle.Strategy, bundle.Run, s.registry)
	if err != nil {
		return Bundle{}, core.StrategyPlan{}, err
	}
	return bundle, plan, nil
}

func (s Service) loadBars(ctx context.Context, plan core.StrategyPlan) ([]core.Bar, error) {
	errb := oops.In("backtest_service").With(
		"market", plan.Market,
		"security_type", plan.SecurityType,
		"from", plan.From.Format(time.DateOnly),
		"to", plan.To.Format(time.DateOnly),
	)
	bars := make([]core.Bar, 0)
	for _, symbol := range plan.DataSymbols() {
		rows, err := s.reader.QueryDailyBars(ctx, daily.Query{
			Market:       provider.Market(plan.Market),
			SecurityType: provider.SecurityType(plan.SecurityType),
			Symbol:       symbol,
			From:         plan.From.Format(time.DateOnly),
			To:           plan.To.Format(time.DateOnly),
		})
		if err != nil {
			return nil, errb.With("symbol", symbol).Wrapf(err, "query canonical daily bars")
		}
		if len(rows) == 0 {
			return nil, errb.With("symbol", symbol).Errorf("canonical daily bars not found for backtest symbol: symbol=%s", symbol)
		}
		for _, row := range rows {
			bar, err := canonicalDailyBarToBacktestBar(row)
			if err != nil {
				return nil, errb.With("symbol", symbol, "trading_date", row.TradingDate).Wrap(err)
			}
			bars = append(bars, bar)
		}
	}
	return bars, nil
}

func (s Service) resolveUniverse(ctx context.Context, yamlPath string, plan core.StrategyPlan) (core.StrategyPlan, error) {
	explain, err := s.universe.Explain(ctx, universeservice.ContextRequest{
		YAMLPath:     yamlPath,
		Market:       plan.Market,
		SecurityType: plan.SecurityType,
		From:         plan.From,
		To:           plan.To,
		Pipeline:     plan.Universe.Pipeline,
	}, plan.Universe.Core)
	if err != nil {
		return core.StrategyPlan{}, oops.In("backtest_service").With("run", plan.RunName).Wrap(err)
	}
	explain.PositionPolicy = plan.Universe.PositionPolicy
	plan.UniverseExplain = explain
	plan.Symbols = symbolsFromUniverseExplain(explain)
	return plan, nil
}

func symbolsFromUniverseExplain(explain core.UniverseExplain) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, snapshot := range explain.Snapshots {
		for _, symbol := range snapshot.Symbols {
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			out = append(out, symbol)
		}
	}
	return out
}

func canonicalDailyBarToBacktestBar(row dailybar.Bar) (core.Bar, error) {
	errb := oops.In("backtest_dailybar_mapper").With("symbol", row.Symbol, "trading_date", row.TradingDate)
	tradingDate, err := time.Parse(time.DateOnly, row.TradingDate)
	if err != nil {
		return core.Bar{}, errb.Wrapf(err, "parse trading date")
	}
	open, err := requiredFloat(row.Open, "opening_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	high, err := requiredFloat(row.High, "highest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	low, err := requiredFloat(row.Low, "lowest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	closePrice, err := requiredFloat(row.Close, "closing_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	volume, err := optionalFloat(row.Volume, "traded_volume")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	tradedAmount, err := optionalFloat(row.TradedValue, "traded_amount")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	return core.Bar{
		Time:         tradingDate,
		Symbol:       row.Symbol,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        closePrice,
		Volume:       volume,
		TradedAmount: tradedAmount,
	}, nil
}

func requiredFloat(value string, field string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, oops.In("backtest_dailybar_mapper").With("field", field).New("daily bar numeric field is required")
	}
	return optionalFloat(value, field)
}

func optionalFloat(value string, field string) (float64, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, oops.In("backtest_dailybar_mapper").With("field", field, "value", value).Wrapf(err, "parse daily bar numeric field")
	}
	return parsed, nil
}

func validationResultFromPlan(plan core.StrategyPlan) ValidationResult {
	indicators := make(map[string]string, len(plan.Indicators))
	for alias, indicator := range plan.Indicators {
		indicators[alias] = indicator.ID
	}
	return ValidationResult{
		Valid:        true,
		StrategyName: plan.StrategyName,
		RunName:      plan.RunName,
		Symbols:      append([]string(nil), plan.Symbols...),
		Period:       core.Period{From: plan.From, To: plan.To},
		Market:       plan.Market,
		SecurityType: plan.SecurityType,
		Timeframe:    plan.Timeframe,
		Currency:     plan.Currency,
		Metrics:      append([]string(nil), plan.SelectedMetrics...),
		Execution: ExecutionSummary{
			Fill:       plan.Fill,
			Commission: plan.Commission,
			Slippage:   plan.Slippage,
		},
		Indicators: indicators,
		Universe:   plan.UniverseExplain,
	}
}
