package backtest

import (
	"context"
	"encoding/json"
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

func NormalizeUniverseSpecWithData(spec UniverseSpec, data DataSpec) UniverseSpec {
	if len(spec.Pipeline) == 0 && len(spec.Symbols) > 0 {
		params := map[string]any{"symbols": append([]string(nil), spec.Symbols...)}
		if fields := identityFields(data); len(fields) > 0 {
			params["fields"] = fields
		}
		spec.Pipeline = []universe.StepSpec{{ID: "source.symbols", Params: params}}
		return spec
	}
	spec = NormalizeUniverseSpec(spec)
	spec.Pipeline = migrateLegacySecurityType(spec.Pipeline, data)
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
			Market:       bar.Market,
			SecurityType: bar.SecurityType,
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

func migrateLegacySecurityType(pipeline []universe.StepSpec, data DataSpec) []universe.StepSpec {
	if data.SecurityType == "" && data.Market == "" {
		return pipeline
	}
	out := make([]universe.StepSpec, 0, len(pipeline))
	for _, step := range pipeline {
		next := step
		if next.Params == nil {
			next.Params = map[string]any{}
		}
		switch next.ID {
		case "source.daily_bars":
			if data.Market != "" {
				if _, ok := next.Params["market"]; !ok {
					next.Params["market"] = data.Market
				}
			}
			if data.SecurityType != "" {
				if _, ok := next.Params["security_type"]; !ok {
					next.Params["security_type"] = data.SecurityType
				}
			}
		case "source.symbols":
			fields := mapParamOrEmpty(next.Params, "fields")
			for key, value := range identityFields(data) {
				if _, ok := fields[key]; !ok {
					fields[key] = value
				}
			}
			if len(fields) > 0 {
				next.Params["fields"] = fields
			}
		case "combine.union", "combine.intersect", "combine.difference", "combine.concat":
			next.Params["pipelines"] = migrateSubPipelines(next.Params["pipelines"], data)
		}
		out = append(out, next)
	}
	return out
}

func migrateSubPipelines(raw any, data DataSpec) any {
	items, ok := raw.([]any)
	if !ok {
		return raw
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		rawSteps, ok := object["pipeline"].([]any)
		if !ok {
			out = append(out, item)
			continue
		}
		steps := make([]universe.StepSpec, 0, len(rawSteps))
		for _, rawStep := range rawSteps {
			payload, err := json.Marshal(rawStep)
			if err != nil {
				continue
			}
			var step universe.StepSpec
			err = json.Unmarshal(payload, &step)
			if err == nil {
				steps = append(steps, step)
			}
		}
		migrated := migrateLegacySecurityType(steps, data)
		nextSteps := make([]any, 0, len(migrated))
		for _, step := range migrated {
			nextSteps = append(nextSteps, map[string]any{"id": step.ID, "name": step.Name, "params": step.Params, "pipeline": step.Pipeline})
		}
		next := make(map[string]any, len(object))
		for key, value := range object {
			next[key] = value
		}
		next["pipeline"] = nextSteps
		out = append(out, next)
	}
	return out
}

func identityFields(data DataSpec) map[string]any {
	fields := map[string]any{}
	if data.Market != "" {
		fields["market"] = data.Market
	}
	if data.SecurityType != "" {
		fields["security_type"] = data.SecurityType
	}
	return fields
}

func mapParamOrEmpty(params map[string]any, key string) map[string]any {
	if raw, ok := params[key].(map[string]any); ok {
		out := make(map[string]any, len(raw))
		for k, v := range raw {
			out[k] = v
		}
		return out
	}
	return map[string]any{}
}

func instrumentsFromStaticSymbols(symbols []string, data DataSpec) []InstrumentIdentity {
	out := make([]InstrumentIdentity, 0, len(symbols))
	for _, symbol := range symbols {
		out = append(out, InstrumentIdentity{
			Symbol:       symbol,
			Market:       data.Market,
			SecurityType: data.SecurityType,
		})
	}
	return out
}
