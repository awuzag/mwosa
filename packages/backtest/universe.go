package backtest

import (
	"context"
	"strings"
	"time"

	"github.com/ev3rlit/mwosa/packages/universe"
	"github.com/samber/oops"
)

const (
	UniversePositionPolicyHold      = "hold"
	UniversePositionPolicyLiquidate = "liquidate"
)

const (
	UniverseScheduleOnce           = universe.ScheduleOnce
	UniverseScheduleDaily          = universe.ScheduleDaily
	UniverseScheduleWeekly         = universe.ScheduleWeekly
	UniverseScheduleMonthly        = universe.ScheduleMonthly
	UniverseScheduleCustomCalendar = universe.ScheduleCustomCalendar
)

type UniverseDataWindow = universe.DataWindow
type UniverseCandidate = universe.Candidate
type UniverseDecision = universe.Decision
type UniverseStepSummary = universe.StepSummary
type UniverseSnapshot = universe.Snapshot
type UniverseExplain = universe.Explain
type UniverseSelectorRegistry = universe.SelectorRegistry
type UniverseSelectorDefinition = universe.SelectorDefinition

type UniverseExecutionContext struct {
	SelectionTime    time.Time
	From             time.Time
	To               time.Time
	Market           string
	SecurityType     string
	DailyBars        []Bar
	Metadata         map[string]UniverseCandidate
	Watchlists       map[string][]UniverseCandidate
	SavedScreens     map[string][]UniverseCandidate
	ScreenStrategies map[string][]UniverseCandidate
	Files            map[string][]UniverseCandidate
}

func DefaultUniverseSelectorRegistry() UniverseSelectorRegistry {
	return universe.DefaultSelectorRegistry()
}

func NewUniverseSelectorRegistry(definitions ...UniverseSelectorDefinition) (UniverseSelectorRegistry, error) {
	return universe.NewSelectorRegistry(definitions...)
}

type UniversePlan struct {
	Core           universe.Plan
	Mode           string                `json:"mode"`
	Schedule       universe.ScheduleSpec `json:"schedule"`
	Pipeline       []universe.StepSpec   `json:"pipeline"`
	PositionPolicy string                `json:"position_policy"`
	StaticSymbols  []string              `json:"static_symbols,omitempty"`
}

func CompileUniverseSpec(spec UniverseSpec, window UniverseDataWindow, registry UniverseSelectorRegistry) (UniversePlan, error) {
	errb := oops.In("backtest_universe")
	normalized := NormalizeUniverseSpec(spec)
	corePlan, err := universe.Compile(universe.PipelineSpec{
		Symbols:  normalized.Symbols,
		Ref:      normalized.Ref,
		Schedule: normalized.Schedule,
		Pipeline: normalized.Pipeline,
	}, window, registry)
	if err != nil {
		return UniversePlan{}, errb.Wrap(err)
	}
	policy := strings.TrimSpace(normalized.PositionPolicy)
	if policy == "" {
		policy = UniversePositionPolicyHold
	}
	if policy != UniversePositionPolicyHold && policy != UniversePositionPolicyLiquidate {
		return UniversePlan{}, errb.With("position_policy", policy).New("unsupported universe position policy")
	}
	return UniversePlan{
		Core:           corePlan,
		Mode:           corePlan.Mode,
		Schedule:       corePlan.Schedule,
		Pipeline:       corePlan.Pipeline,
		PositionPolicy: policy,
		StaticSymbols:  append([]string(nil), corePlan.StaticSymbols...),
	}, nil
}

func NormalizeUniverseSpec(spec UniverseSpec) UniverseSpec {
	coreSpec := universe.Normalize(universe.PipelineSpec{
		Symbols:  spec.Symbols,
		Ref:      spec.Ref,
		Schedule: spec.Schedule,
		Pipeline: spec.Pipeline,
	})
	spec.Symbols = coreSpec.Symbols
	spec.Ref = coreSpec.Ref
	spec.Schedule = coreSpec.Schedule
	spec.Pipeline = coreSpec.Pipeline
	return spec
}

func ExecuteUniversePipeline(ctx context.Context, plan UniversePlan, execCtx UniverseExecutionContext) (UniverseSnapshot, error) {
	return universe.ExecutePipeline(ctx, plan.Core, toCoreUniverseExecutionContext(execCtx))
}

func BuildUniverseSnapshots(ctx context.Context, plan UniversePlan, execCtx UniverseExecutionContext) (UniverseExplain, error) {
	explain, err := universe.BuildSnapshots(ctx, plan.Core, toCoreUniverseExecutionContext(execCtx))
	if err != nil {
		return UniverseExplain{}, err
	}
	explain.PositionPolicy = plan.PositionPolicy
	return explain, nil
}

func toCoreUniverseExecutionContext(execCtx UniverseExecutionContext) universe.ExecutionContext {
	return universe.ExecutionContext{
		SelectionTime:    execCtx.SelectionTime,
		From:             execCtx.From,
		To:               execCtx.To,
		Market:           execCtx.Market,
		SecurityType:     execCtx.SecurityType,
		DailyBars:        universeBarsFromBacktestBars(execCtx.DailyBars),
		Metadata:         execCtx.Metadata,
		Watchlists:       execCtx.Watchlists,
		SavedScreens:     execCtx.SavedScreens,
		ScreenStrategies: execCtx.ScreenStrategies,
		Files:            execCtx.Files,
	}
}

func universeBarsFromBacktestBars(bars []Bar) []universe.Bar {
	out := make([]universe.Bar, 0, len(bars))
	for _, bar := range bars {
		out = append(out, universe.Bar{
			Time:         bar.Time,
			Symbol:       bar.Symbol,
			Open:         bar.Open,
			High:         bar.High,
			Low:          bar.Low,
			Close:        bar.Close,
			Volume:       bar.Volume,
			TradedAmount: bar.TradedAmount,
		})
	}
	return out
}
