package universe

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	ScheduleOnce           = "once"
	ScheduleDaily          = "daily"
	ScheduleWeekly         = "weekly"
	ScheduleMonthly        = "monthly"
	ScheduleCustomCalendar = "custom_calendar"
)

type DataWindow struct {
	Market       string
	SecurityType string
	From         time.Time
	To           time.Time
}

type Plan struct {
	Mode          string       `json:"mode"`
	Schedule      ScheduleSpec `json:"schedule"`
	Pipeline      []StepSpec   `json:"pipeline"`
	StaticSymbols []string     `json:"static_symbols,omitempty"`
	window        DataWindow
	registry      SelectorRegistry
}

type Candidate struct {
	Symbol string         `json:"symbol"`
	Fields map[string]any `json:"fields,omitempty"`
	Tags   []string       `json:"tags,omitempty"`
}

type Decision struct {
	Time   time.Time      `json:"time,omitempty"`
	Symbol string         `json:"symbol"`
	StepID string         `json:"step_id"`
	Action string         `json:"action"`
	Reason string         `json:"reason,omitempty"`
	Values map[string]any `json:"values,omitempty"`
}

type StepSummary struct {
	ID          string         `json:"id"`
	Name        string         `json:"name,omitempty"`
	InputCount  int            `json:"input_count"`
	OutputCount int            `json:"output_count"`
	Reason      string         `json:"reason,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
}

type Snapshot struct {
	Time       time.Time     `json:"time"`
	Symbols    []string      `json:"symbols"`
	Candidates []Candidate   `json:"candidates,omitempty"`
	Decisions  []Decision    `json:"decisions,omitempty"`
	Steps      []StepSummary `json:"steps"`
}

type Explain struct {
	Mode            string        `json:"mode"`
	Schedule        string        `json:"schedule"`
	PositionPolicy  string        `json:"position_policy"`
	SelectedSymbols []string      `json:"selected_symbols"`
	Steps           []StepSummary `json:"steps"`
	Decisions       []Decision    `json:"decisions,omitempty"`
	Snapshots       []Snapshot    `json:"snapshots,omitempty"`
}

type ExecutionContext struct {
	SelectionTime    time.Time
	From             time.Time
	To               time.Time
	Market           string
	SecurityType     string
	DailyBars        []Bar
	Metadata         map[string]Candidate
	Watchlists       map[string][]Candidate
	SavedScreens     map[string][]Candidate
	ScreenStrategies map[string][]Candidate
	Files            map[string][]Candidate
	closedDataOnly   bool
}

type SelectorRegistry struct {
	defs map[string]SelectorDefinition
}

type SelectorDefinition struct {
	ID         string
	Kind       string
	AllowEmpty bool
	Validate   func(StepSpec) error
	Execute    func(context.Context, StepSpec, []Candidate, ExecutionContext, SelectorRegistry) ([]Candidate, StepSummary, []Decision, error)
}

func NewSelectorRegistry(definitions ...SelectorDefinition) (SelectorRegistry, error) {
	registry := SelectorRegistry{defs: make(map[string]SelectorDefinition, len(definitions))}
	for _, definition := range definitions {
		if strings.TrimSpace(definition.ID) == "" {
			return SelectorRegistry{}, oops.In("universe_registry").New("universe selector id is empty")
		}
		if definition.Execute == nil {
			return SelectorRegistry{}, oops.In("universe_registry").With("selector", definition.ID).New("universe selector executor is nil")
		}
		if _, ok := registry.defs[definition.ID]; ok {
			return SelectorRegistry{}, oops.In("universe_registry").With("selector", definition.ID).New("universe selector id is duplicated")
		}
		registry.defs[definition.ID] = definition
	}
	return registry, nil
}

func DefaultSelectorRegistry() SelectorRegistry {
	registry, err := NewSelectorRegistry(defaultUniverseSelectors()...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r SelectorRegistry) Definition(id string) (SelectorDefinition, bool) {
	definition, ok := r.defs[id]
	return definition, ok
}

func CompilePipelineSpec(spec PipelineSpec, window DataWindow, registry SelectorRegistry) (Plan, error) {
	errb := oops.In("universe")
	normalized := NormalizePipelineSpec(spec)
	if len(normalized.Pipeline) == 0 {
		return Plan{}, errb.New("universe pipeline requires at least one selector")
	}
	schedule, err := normalizeUniverseSchedule(normalized.Schedule)
	if err != nil {
		return Plan{}, errb.Wrap(err)
	}
	if err := validateUniversePipeline(normalized.Pipeline, registry); err != nil {
		return Plan{}, errb.Wrap(err)
	}
	staticSymbols := staticSymbolsFromSource(normalized.Pipeline)
	return Plan{
		Mode:          "pipeline",
		Schedule:      schedule,
		Pipeline:      normalized.Pipeline,
		StaticSymbols: staticSymbols,
		window:        window,
		registry:      registry,
	}, nil
}

func Compile(spec PipelineSpec, window DataWindow, registry SelectorRegistry) (Plan, error) {
	return CompilePipelineSpec(spec, window, registry)
}

func NormalizePipelineSpec(spec PipelineSpec) PipelineSpec {
	out := spec
	if len(out.Pipeline) == 0 && len(out.Symbols) > 0 {
		params := map[string]any{"symbols": append([]string(nil), out.Symbols...)}
		out.Pipeline = []StepSpec{{ID: "source.symbols", Params: params}}
	}
	return out
}

func Normalize(spec PipelineSpec) PipelineSpec {
	return NormalizePipelineSpec(spec)
}

func normalizeUniverseSchedule(schedule ScheduleSpec) (ScheduleSpec, error) {
	if strings.TrimSpace(schedule.Frequency) == "" {
		schedule.Frequency = ScheduleOnce
	}
	switch schedule.Frequency {
	case ScheduleOnce, ScheduleDaily, ScheduleWeekly, ScheduleMonthly, ScheduleCustomCalendar:
	default:
		return ScheduleSpec{}, oops.In("universe_schedule").With("frequency", schedule.Frequency).New("unsupported universe schedule frequency")
	}
	if strings.TrimSpace(schedule.LookbackPolicy) == "" {
		schedule.LookbackPolicy = "closed_bars_only"
	}
	if schedule.LookbackPolicy != "closed_bars_only" {
		return ScheduleSpec{}, oops.In("universe_schedule").With("lookback_policy", schedule.LookbackPolicy).New("unsupported universe lookback policy")
	}
	return schedule, nil
}

func validateUniversePipeline(pipeline []StepSpec, registry SelectorRegistry) error {
	for index, step := range pipeline {
		errb := oops.In("universe").With("selector", step.ID, "index", index)
		definition, ok := registry.Definition(step.ID)
		if !ok {
			return errb.Errorf("universe selector is not registered: selector=%s", step.ID)
		}
		if index == 0 && definition.Kind != "source" && definition.Kind != "combine" {
			return errb.New("universe pipeline must start with source or combine selector")
		}
		if definition.Validate != nil {
			if err := definition.Validate(step); err != nil {
				return errb.Wrap(err)
			}
		}
		if definition.Kind == "combine" {
			for _, sub := range subPipelinesForValidation(step) {
				if err := validateUniversePipeline(sub, registry); err != nil {
					return errb.Wrap(err)
				}
			}
		}
	}
	return nil
}

func subPipelinesForValidation(step StepSpec) [][]StepSpec {
	raw, ok := step.Params["pipelines"].([]any)
	if !ok {
		return nil
	}
	out := make([][]StepSpec, 0, len(raw))
	for _, item := range raw {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rawSteps, ok := object["pipeline"].([]any)
		if !ok {
			continue
		}
		steps := make([]StepSpec, 0, len(rawSteps))
		for _, rawStep := range rawSteps {
			step, err := stepFromAny(rawStep)
			if err == nil {
				steps = append(steps, step)
			}
		}
		out = append(out, steps)
	}
	return out
}

func staticSymbolsFromSource(pipeline []StepSpec) []string {
	if len(pipeline) != 1 || pipeline[0].ID != "source.symbols" {
		return nil
	}
	symbols, _ := stringSliceParam(pipeline[0].Params, "symbols")
	slices.Sort(symbols)
	return symbols
}

func ExecutePipeline(ctx context.Context, plan Plan, execCtx ExecutionContext) (Snapshot, error) {
	return executeUniversePipeline(ctx, plan.Pipeline, nil, execCtx, plan.registry)
}

func executeUniversePipeline(ctx context.Context, pipeline []StepSpec, input []Candidate, execCtx ExecutionContext, registry SelectorRegistry) (Snapshot, error) {
	candidates := cloneCandidates(input)
	steps := make([]StepSummary, 0, len(pipeline))
	decisions := make([]Decision, 0)
	for _, step := range pipeline {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, oops.In("universe").Wrap(err)
		}
		definition, ok := registry.Definition(step.ID)
		if !ok {
			return Snapshot{}, oops.In("universe").With("selector", step.ID).Errorf("universe selector is not registered: selector=%s", step.ID)
		}
		out, summary, stepDecisions, err := definition.Execute(ctx, step, candidates, execCtx, registry)
		if err != nil {
			return Snapshot{}, oops.In("universe").With("selector", step.ID).Wrap(err)
		}
		summary.ID = step.ID
		summary.Name = step.Name
		if summary.InputCount == 0 {
			summary.InputCount = len(candidates)
		}
		summary.OutputCount = len(out)
		steps = append(steps, summary)
		decisions = append(decisions, stepDecisions...)
		candidates = out
	}
	sortCandidates(candidates)
	return Snapshot{
		Time:       execCtx.SelectionTime,
		Symbols:    symbolsFromCandidates(candidates),
		Candidates: candidates,
		Decisions:  decisions,
		Steps:      steps,
	}, nil
}

func BuildSnapshots(ctx context.Context, plan Plan, execCtx ExecutionContext) (Explain, error) {
	dates, err := universeSelectionDates(plan, execCtx)
	if err != nil {
		return Explain{}, err
	}
	snapshots := make([]Snapshot, 0, len(dates))
	for _, selectionTime := range dates {
		selectionCtx := execCtx
		selectionCtx.SelectionTime = selectionTime
		selectionCtx.closedDataOnly = true
		snapshot, err := ExecutePipeline(ctx, plan, selectionCtx)
		if err != nil {
			return Explain{}, err
		}
		snapshots = append(snapshots, snapshot)
	}
	selected := []string(nil)
	steps := []StepSummary(nil)
	decisions := []Decision(nil)
	if len(snapshots) > 0 {
		selected = append([]string(nil), snapshots[len(snapshots)-1].Symbols...)
		steps = append([]StepSummary(nil), snapshots[len(snapshots)-1].Steps...)
		decisions = append([]Decision(nil), snapshots[len(snapshots)-1].Decisions...)
	}
	return Explain{
		Mode:            plan.Mode,
		Schedule:        plan.Schedule.Frequency,
		SelectedSymbols: selected,
		Steps:           steps,
		Decisions:       decisions,
		Snapshots:       snapshots,
	}, nil
}

func universeSelectionDates(plan Plan, execCtx ExecutionContext) ([]time.Time, error) {
	from := execCtx.From
	if from.IsZero() {
		from = plan.window.From
	}
	to := execCtx.To
	if to.IsZero() {
		to = plan.window.To
	}
	if to.Before(from) {
		return nil, oops.In("universe_schedule").New("universe schedule to date is before from date")
	}
	switch plan.Schedule.Frequency {
	case ScheduleOnce:
		return []time.Time{from}, nil
	case ScheduleDaily:
		return dailyDates(from, to), nil
	case ScheduleWeekly:
		return anchoredWeekDates(from, to), nil
	case ScheduleMonthly:
		return anchoredMonthDates(from, to), nil
	case ScheduleCustomCalendar:
		out := make([]time.Time, 0, len(plan.Schedule.Dates))
		for _, value := range plan.Schedule.Dates {
			parsed, err := time.Parse(time.DateOnly, value)
			if err != nil {
				return nil, oops.In("universe_schedule").With("date", value).Wrapf(err, "parse custom calendar date")
			}
			if parsed.Before(from) || parsed.After(to) {
				continue
			}
			out = append(out, parsed)
		}
		if len(out) == 0 {
			return nil, oops.In("universe_schedule").New("custom calendar has no dates in backtest range")
		}
		return out, nil
	default:
		return nil, oops.In("universe_schedule").With("frequency", plan.Schedule.Frequency).New("unsupported universe schedule frequency")
	}
}

func dailyDates(from, to time.Time) []time.Time {
	out := make([]time.Time, 0)
	for current := from; !current.After(to); current = current.AddDate(0, 0, 1) {
		out = append(out, current)
	}
	return out
}

func anchoredWeekDates(from, to time.Time) []time.Time {
	days := dailyDates(from, to)
	out := make([]time.Time, 0)
	seen := map[string]struct{}{}
	for _, day := range days {
		year, week := day.ISOWeek()
		key := strconv.Itoa(year) + "-" + strconv.Itoa(week)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, day)
	}
	return out
}

func anchoredMonthDates(from, to time.Time) []time.Time {
	days := dailyDates(from, to)
	out := make([]time.Time, 0)
	seen := map[string]struct{}{}
	for _, day := range days {
		key := day.Format("2006-01")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, day)
	}
	return out
}

func defaultUniverseSelectors() []SelectorDefinition {
	return []SelectorDefinition{
		sourceSelector("source.symbols", sourceSymbols),
		sourceSelector("source.daily_bars", sourceDailyBars),
		sourceSelector("source.instrument_master", sourceInstrumentMaster),
		sourceSelector("source.saved_screen", sourceSavedScreen),
		sourceSelector("source.screen_strategy", sourceScreenStrategy),
		sourceSelector("source.watchlist", sourceWatchlist),
		sourceSelector("source.file", sourceFile),
		sourceSelector("source.inline", sourceInline),
		transformSelector("transform.latest_per_symbol", transformLatestPerSymbol),
		transformSelector("transform.window_metrics", transformWindowMetrics),
		transformSelector("transform.indicator", transformIndicator),
		transformSelector("transform.join_metadata", transformJoinMetadata),
		transformSelector("transform.normalize_fields", transformNormalizeFields),
		transformSelector("transform.distinct", transformDistinct),
		transformSelector("transform.tag", transformTag),
		filterSelector("filter.field", filterField),
		filterSelector("filter.expr", filterExpr),
		filterSelector("filter.has_daily_bars", filterHasDailyBars),
		filterSelector("filter.listing_age", filterListingAge),
		filterSelector("filter.liquidity", filterLiquidity),
		filterSelector("filter.exclude_symbols", filterExcludeSymbols),
		filterSelector("filter.include_symbols", filterIncludeSymbols),
		filterSelector("filter.security_type", filterSecurityType),
		filterSelector("filter.market", filterMarket),
		filterSelector("filter.tags", filterTags),
		rankSelector("rank.by_field", rankByField),
		rankSelector("rank.weighted", rankWeighted),
		rankSelector("rank.percentile", rankPercentile),
		rankSelector("rank.group_top_n", rankGroupTopN),
		rankSelector("rank.round_robin", rankRoundRobin),
		transformSelector("limit.count", limitCount),
		transformSelector("limit.per_group", limitPerGroup),
		transformSelector("limit.min_count", limitMinCount),
		transformSelector("limit.max_count", limitMaxCount),
		sourceSelector("combine.union", combineUnion),
		sourceSelector("combine.intersect", combineIntersect),
		sourceSelector("combine.difference", combineDifference),
		sourceSelector("combine.concat", combineConcat),
		transformSelector("debug.snapshot", debugSnapshot),
		transformSelector("debug.assert_count", debugAssertCount),
		transformSelector("debug.sample", debugSample),
	}
}

type selectorFunc func(context.Context, StepSpec, []Candidate, ExecutionContext, SelectorRegistry) ([]Candidate, StepSummary, []Decision, error)

func sourceSelector(id string, fn selectorFunc) SelectorDefinition {
	return SelectorDefinition{ID: id, Kind: strings.Split(id, ".")[0], AllowEmpty: true, Validate: validateUniverseStepParams, Execute: fn}
}

func transformSelector(id string, fn selectorFunc) SelectorDefinition {
	return SelectorDefinition{ID: id, Kind: strings.Split(id, ".")[0], Validate: validateUniverseStepParams, Execute: fn}
}

func filterSelector(id string, fn selectorFunc) SelectorDefinition {
	return SelectorDefinition{ID: id, Kind: strings.Split(id, ".")[0], Validate: validateUniverseStepParams, Execute: fn}
}

func rankSelector(id string, fn selectorFunc) SelectorDefinition {
	return SelectorDefinition{ID: id, Kind: strings.Split(id, ".")[0], Validate: validateUniverseStepParams, Execute: fn}
}

func validateUniverseStepParams(step StepSpec) error {
	switch step.ID {
	case "source.symbols":
		if _, ok := stringSliceParam(step.Params, "symbols"); !ok {
			return oops.In("universe").New("source.symbols requires symbols")
		}
	case "rank.by_field":
		if _, ok := stringParam(step.Params, "field"); !ok {
			return oops.In("universe").New("rank.by_field requires field")
		}
		if order := stringParamDefault(step.Params, "order", "desc"); order != "asc" && order != "desc" {
			return oops.In("universe").With("order", order).New("rank order must be asc or desc")
		}
	case "filter.field":
		if _, ok := stringParam(step.Params, "field"); !ok {
			return oops.In("universe").New("filter.field requires field")
		}
	case "combine.union", "combine.intersect", "combine.difference", "combine.concat":
		if _, ok := step.Params["pipelines"]; !ok {
			return oops.In("universe").New("combine selector requires pipelines")
		}
	}
	return nil
}

func sourceSymbols(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	symbols, ok := stringSliceParam(step.Params, "symbols")
	if !ok || len(symbols) == 0 {
		return nil, StepSummary{}, nil, oops.In("universe").New("source.symbols requires symbols")
	}
	out := make([]Candidate, 0, len(symbols))
	decisions := make([]Decision, 0, len(symbols))
	for _, symbol := range symbols {
		candidate := newCandidate(symbol)
		out = append(out, candidate)
		decisions = append(decisions, includeDecision(execCtx.SelectionTime, step.ID, symbol, "source.symbols"))
	}
	return out, StepSummary{InputCount: 0, OutputCount: len(out)}, decisions, nil
}

func sourceDailyBars(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	lookback := intParamDefault(step.Params, "lookback_days", 0)
	from := execCtx.From
	if lookback > 0 && !execCtx.SelectionTime.IsZero() {
		from = execCtx.SelectionTime.AddDate(0, 0, -lookback)
	}
	out := make([]Candidate, 0)
	for _, bar := range closedBars(execCtx.DailyBars, execCtx.SelectionTime, execCtx.closedDataOnly) {
		if !from.IsZero() && bar.Time.Before(from) {
			continue
		}
		fields := barFields(bar)
		if market := stringParamDefault(step.Params, "market", execCtx.Market); market != "" {
			fields["market"] = market
		}
		if securityType := stringParamDefault(step.Params, "security_type", execCtx.SecurityType); securityType != "" {
			fields["security_type"] = securityType
		}
		out = append(out, Candidate{Symbol: bar.Symbol, Fields: fields, Tags: []string{"liquid"}})
	}
	return out, StepSummary{Reason: "canonical daily bars"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "daily_bar_row"), nil
}

func sourceInstrumentMaster(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	out := make([]Candidate, 0, len(execCtx.Metadata))
	for symbol, candidate := range execCtx.Metadata {
		if strings.TrimSpace(candidate.Symbol) == "" {
			candidate.Symbol = symbol
		}
		out = append(out, cloneCandidate(candidate))
	}
	if len(out) == 0 {
		latest := latestBarsBySymbol(closedBars(execCtx.DailyBars, execCtx.SelectionTime, execCtx.closedDataOnly))
		for symbol, bar := range latest {
			out = append(out, Candidate{Symbol: symbol, Fields: barFields(bar)})
		}
	}
	return out, StepSummary{Reason: "instrument metadata"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "instrument_master"), nil
}

func sourceSavedScreen(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	name := stringParamDefault(step.Params, "name", stringParamDefault(step.Params, "ref", ""))
	rows := execCtx.SavedScreens[name]
	if rows == nil {
		return nil, StepSummary{}, nil, oops.In("universe").With("name", name).New("saved screen is not available")
	}
	out := cloneCandidates(rows)
	return out, StepSummary{Reason: "saved_screen:" + name}, includeDecisions(execCtx.SelectionTime, step.ID, out, "saved_screen"), nil
}

func sourceScreenStrategy(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	name := stringParamDefault(step.Params, "name", "")
	rows := execCtx.ScreenStrategies[name]
	if rows == nil {
		return nil, StepSummary{}, nil, oops.In("universe").With("name", name).New("screen strategy result is not available")
	}
	out := cloneCandidates(rows)
	return out, StepSummary{Reason: "screen_strategy:" + name}, includeDecisions(execCtx.SelectionTime, step.ID, out, "screen_strategy"), nil
}

func sourceWatchlist(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	if symbols, ok := stringSliceParam(step.Params, "symbols"); ok {
		out := candidatesFromSymbols(symbols)
		return out, StepSummary{Reason: "inline_watchlist"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "watchlist"), nil
	}
	name := stringParamDefault(step.Params, "name", "")
	rows := execCtx.Watchlists[name]
	if rows == nil {
		return nil, StepSummary{}, nil, oops.In("universe").With("name", name).New("watchlist is not available")
	}
	out := cloneCandidates(rows)
	return out, StepSummary{Reason: "watchlist:" + name}, includeDecisions(execCtx.SelectionTime, step.ID, out, "watchlist"), nil
}

func sourceFile(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	path := stringParamDefault(step.Params, "path", "")
	rows := execCtx.Files[path]
	if rows == nil {
		return nil, StepSummary{}, nil, oops.In("universe").With("path", path).New("source file rows are not available")
	}
	out := cloneCandidates(rows)
	return out, StepSummary{Reason: "file:" + path}, includeDecisions(execCtx.SelectionTime, step.ID, out, "file"), nil
}

func sourceInline(_ context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	raw, ok := step.Params["rows"]
	if !ok {
		return nil, StepSummary{}, nil, oops.In("universe").New("source.inline requires rows")
	}
	out, err := candidatesFromAny(raw)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	return out, StepSummary{Reason: "inline rows"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "inline"), nil
}

func transformLatestPerSymbol(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	latest := map[string]Candidate{}
	for _, candidate := range input {
		current, ok := latest[candidate.Symbol]
		if !ok || compareField(candidate, current, "trading_date") > 0 {
			latest[candidate.Symbol] = cloneCandidate(candidate)
		}
	}
	out := make([]Candidate, 0, len(latest))
	for _, candidate := range latest {
		out = append(out, candidate)
	}
	sortCandidates(out)
	return out, StepSummary{Reason: "latest row per symbol"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "latest_per_symbol"), nil
}

func transformWindowMetrics(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	metrics, ok := mapParam(step.Params, "metrics")
	if !ok {
		return nil, StepSummary{}, nil, oops.In("universe").New("transform.window_metrics requires metrics")
	}
	series := candidatesBySymbol(input)
	out := cloneCandidates(input)
	latestIndex := map[string]int{}
	for i, candidate := range out {
		latestIndex[candidate.Symbol] = i
	}
	for output, raw := range metrics {
		spec, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		metricID := stringParamDefault(spec, "id", "")
		params, _ := mapParam(spec, "params")
		window := intParamDefault(params, "window", 1)
		field := stringParamDefault(params, "field", "close")
		for symbol, rows := range series {
			sortCandidatesByField(rows, "trading_date", "asc")
			value, valueOK := windowMetric(metricID, rows, field, window)
			if !valueOK {
				continue
			}
			index := latestIndex[symbol]
			ensureFields(&out[index])
			out[index].Fields[output] = value
		}
	}
	return out, StepSummary{Reason: "window metrics"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "window_metrics"), nil
}

func transformIndicator(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	id := stringParamDefault(step.Params, "id", "")
	field := stringParamDefault(step.Params, "field", "close")
	window := intParamDefault(step.Params, "window", 1)
	output := stringParamDefault(step.Params, "output", id+"_"+strconv.Itoa(window))
	if id == "" {
		return nil, StepSummary{}, nil, oops.In("universe").New("transform.indicator requires id")
	}
	indicatorRegistry, err := DefaultIndicatorRegistry()
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	definition, ok := indicatorRegistry.Definition(id)
	if !ok {
		return nil, StepSummary{}, nil, oops.In("universe").With("indicator", id).New("indicator is not registered")
	}
	out := cloneCandidates(input)
	series := candidatesBySymbol(out)
	for symbol, rows := range series {
		sortCandidatesByField(rows, "trading_date", "asc")
		bars := barsFromCandidates(rows)
		values, err := definition.Calculate(IndicatorSpec{ID: id, Field: field, Window: window}, bars)
		if err != nil {
			return nil, StepSummary{}, nil, err
		}
		if len(values) == 0 {
			continue
		}
		value := values[len(values)-1]
		for i := range out {
			if out[i].Symbol == symbol {
				ensureFields(&out[i])
				out[i].Fields[output] = value
			}
		}
	}
	return out, StepSummary{Reason: "indicator:" + id}, includeDecisions(execCtx.SelectionTime, step.ID, out, "indicator"), nil
}

func transformJoinMetadata(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	out := cloneCandidates(input)
	for i := range out {
		meta, ok := execCtx.Metadata[out[i].Symbol]
		if !ok {
			continue
		}
		ensureFields(&out[i])
		for key, value := range meta.Fields {
			out[i].Fields[key] = value
		}
		out[i].Tags = mergeTags(out[i].Tags, meta.Tags)
	}
	return out, StepSummary{Reason: "join metadata"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "join_metadata"), nil
}

func transformNormalizeFields(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	out := cloneCandidates(input)
	for i := range out {
		for key, value := range out[i].Fields {
			out[i].Fields[key] = normalizeScalar(value)
		}
	}
	return out, StepSummary{Reason: "normalize fields"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "normalize_fields"), nil
}

func transformDistinct(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	by := stringParamDefault(step.Params, "by", "symbol")
	seen := map[string]struct{}{}
	out := make([]Candidate, 0, len(input))
	decisions := make([]Decision, 0)
	for _, candidate := range input {
		key := candidate.Symbol
		if by != "symbol" {
			key = valueString(candidate.Fields[by])
		}
		if _, ok := seen[key]; ok {
			decisions = append(decisions, excludeDecision(execCtx.SelectionTime, step.ID, candidate.Symbol, "duplicate"))
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cloneCandidate(candidate))
	}
	return out, StepSummary{Reason: "distinct by " + by}, decisions, nil
}

func transformTag(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	tags, _ := stringSliceParam(step.Params, "tags")
	if tag := stringParamDefault(step.Params, "tag", ""); tag != "" {
		tags = append(tags, tag)
	}
	out := cloneCandidates(input)
	for i := range out {
		out[i].Tags = mergeTags(out[i].Tags, tags)
	}
	return out, StepSummary{Reason: "tag"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "tag"), nil
}

func filterField(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	field := stringParamDefault(step.Params, "field", "")
	op := stringParamDefault(step.Params, "op", "eq")
	value := step.Params["value"]
	return filterCandidates(step.ID, execCtx.SelectionTime, input, field+" "+op+" "+valueString(value), func(candidate Candidate) bool {
		return compareAny(candidate.Fields[field], op, value)
	})
}

func filterExpr(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "expr", func(candidate Candidate) bool {
		return evalUniverseExpr(step.Params, candidate)
	})
}

func filterHasDailyBars(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	minCount := intParamDefault(step.Params, "min_count", 1)
	counts := map[string]int{}
	for _, bar := range closedBars(execCtx.DailyBars, execCtx.SelectionTime, execCtx.closedDataOnly) {
		counts[bar.Symbol]++
	}
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "has_daily_bars", func(candidate Candidate) bool {
		return counts[candidate.Symbol] >= minCount
	})
}

func filterListingAge(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	minDays := intParamDefault(step.Params, "min_days", 0)
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "listing_age", func(candidate Candidate) bool {
		listedAt, ok := dateField(candidate, "listed_at")
		if !ok {
			listedAt, ok = earliestBarDate(execCtx.DailyBars, candidate.Symbol)
		}
		if !ok || execCtx.SelectionTime.IsZero() {
			return false
		}
		return int(execCtx.SelectionTime.Sub(listedAt).Hours()/24) >= minDays
	})
}

func filterLiquidity(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	minAmount := floatParamDefault(step.Params, "min_traded_amount", math.Inf(-1))
	minVolume := floatParamDefault(step.Params, "min_volume", math.Inf(-1))
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "liquidity", func(candidate Candidate) bool {
		return floatField(candidate, "traded_amount") >= minAmount && floatField(candidate, "volume") >= minVolume
	})
}

func filterExcludeSymbols(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	symbols, _ := stringSliceParam(step.Params, "symbols")
	blocked := set(symbols)
	reason := stringParamDefault(step.Params, "reason", "excluded_symbol")
	return filterCandidates(step.ID, execCtx.SelectionTime, input, reason, func(candidate Candidate) bool {
		_, ok := blocked[candidate.Symbol]
		return !ok
	})
}

func filterIncludeSymbols(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	symbols, _ := stringSliceParam(step.Params, "symbols")
	allowed := set(symbols)
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "include_symbols", func(candidate Candidate) bool {
		_, ok := allowed[candidate.Symbol]
		return ok
	})
}

func filterSecurityType(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	value := stringParamDefault(step.Params, "value", stringParamDefault(step.Params, "security_type", ""))
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "security_type="+value, func(candidate Candidate) bool {
		return valueString(candidate.Fields["security_type"]) == value
	})
}

func filterMarket(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	value := stringParamDefault(step.Params, "value", stringParamDefault(step.Params, "market", ""))
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "market="+value, func(candidate Candidate) bool {
		return valueString(candidate.Fields["market"]) == value
	})
}

func filterTags(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	include, _ := stringSliceParam(step.Params, "include")
	exclude, _ := stringSliceParam(step.Params, "exclude")
	return filterCandidates(step.ID, execCtx.SelectionTime, input, "tags", func(candidate Candidate) bool {
		for _, tag := range include {
			if !slices.Contains(candidate.Tags, tag) {
				return false
			}
		}
		for _, tag := range exclude {
			if slices.Contains(candidate.Tags, tag) {
				return false
			}
		}
		return true
	})
}

func rankByField(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	field := stringParamDefault(step.Params, "field", "")
	order := stringParamDefault(step.Params, "order", "desc")
	out := cloneCandidates(input)
	sortCandidatesByField(out, field, order)
	if limit := intParamDefault(step.Params, "limit", 0); limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, StepSummary{Reason: "rank by " + field, Fields: map[string]any{"field": field, "order": order}}, includeDecisions(execCtx.SelectionTime, step.ID, out, "ranked"), nil
}

func rankWeighted(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	fields, ok := mapParam(step.Params, "fields")
	if !ok {
		return nil, StepSummary{}, nil, oops.In("universe").New("rank.weighted requires fields")
	}
	out := cloneCandidates(input)
	for i := range out {
		var score float64
		for field, rawWeight := range fields {
			score += floatField(out[i], field) * toFloat(rawWeight)
		}
		ensureFields(&out[i])
		out[i].Fields["score"] = score
	}
	sortCandidatesByField(out, "score", stringParamDefault(step.Params, "order", "desc"))
	if limit := intParamDefault(step.Params, "limit", 0); limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, StepSummary{Reason: "weighted rank"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "weighted_rank"), nil
}

func rankPercentile(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	field := stringParamDefault(step.Params, "field", "score")
	top := floatParamDefault(step.Params, "top", 0)
	bottom := floatParamDefault(step.Params, "bottom", 0)
	out := cloneCandidates(input)
	order := "desc"
	if bottom > 0 {
		order = "asc"
		top = bottom
	}
	if top <= 0 || top > 1 {
		top = 1
	}
	sortCandidatesByField(out, field, order)
	keep := int(math.Ceil(float64(len(out)) * top))
	if keep < len(out) {
		out = out[:keep]
	}
	return out, StepSummary{Reason: "percentile " + field}, includeDecisions(execCtx.SelectionTime, step.ID, out, "percentile"), nil
}

func rankGroupTopN(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	groupField := stringParamDefault(step.Params, "group_field", "group")
	rankField := stringParamDefault(step.Params, "rank_field", stringParamDefault(step.Params, "field", "score"))
	n := intParamDefault(step.Params, "n", intParamDefault(step.Params, "count", 1))
	order := stringParamDefault(step.Params, "order", "desc")
	groups := groupCandidates(input, groupField)
	out := make([]Candidate, 0)
	for _, rows := range groups {
		sortCandidatesByField(rows, rankField, order)
		if n < len(rows) {
			rows = rows[:n]
		}
		out = append(out, rows...)
	}
	sortCandidates(out)
	return out, StepSummary{Reason: "group top n"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "group_top_n"), nil
}

func rankRoundRobin(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	groupField := stringParamDefault(step.Params, "group_field", "group")
	limit := intParamDefault(step.Params, "limit", len(input))
	groups := groupCandidates(input, groupField)
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
		sortCandidates(groups[key])
	}
	sort.Strings(keys)
	out := make([]Candidate, 0, len(input))
	for len(out) < limit {
		added := false
		for _, key := range keys {
			rows := groups[key]
			if len(rows) == 0 {
				continue
			}
			out = append(out, rows[0])
			groups[key] = rows[1:]
			added = true
			if len(out) >= limit {
				break
			}
		}
		if !added {
			break
		}
	}
	return out, StepSummary{Reason: "round robin"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "round_robin"), nil
}

func limitCount(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	count := intParamDefault(step.Params, "count", len(input))
	out := cloneCandidates(input)
	if count < len(out) {
		out = out[:count]
	}
	return out, StepSummary{Reason: "limit count"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "limit_count"), nil
}

func limitPerGroup(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	groupField := stringParamDefault(step.Params, "group_field", "group")
	count := intParamDefault(step.Params, "count", 1)
	groups := map[string]int{}
	out := make([]Candidate, 0, len(input))
	for _, candidate := range input {
		key := valueString(candidate.Fields[groupField])
		if groups[key] >= count {
			continue
		}
		groups[key]++
		out = append(out, cloneCandidate(candidate))
	}
	return out, StepSummary{Reason: "limit per group"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "limit_per_group"), nil
}

func limitMinCount(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	count := intParamDefault(step.Params, "count", intParamDefault(step.Params, "min", 0))
	if len(input) < count {
		return nil, StepSummary{}, nil, oops.In("universe").With("count", len(input), "min", count).New("universe candidate count is below minimum")
	}
	out := cloneCandidates(input)
	return out, StepSummary{Reason: "min count"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "min_count"), nil
}

func limitMaxCount(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	count := intParamDefault(step.Params, "count", intParamDefault(step.Params, "max", len(input)))
	fail := boolParamDefault(step.Params, "fail", false)
	if len(input) > count && fail {
		return nil, StepSummary{}, nil, oops.In("universe").With("count", len(input), "max", count).New("universe candidate count is above maximum")
	}
	out := cloneCandidates(input)
	if count < len(out) {
		out = out[:count]
	}
	return out, StepSummary{Reason: "max count"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "max_count"), nil
}

func combineUnion(ctx context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, registry SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	pipelines, err := subPipelines(step)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	seen := map[string]Candidate{}
	decisions := make([]Decision, 0)
	for _, pipeline := range pipelines {
		snapshot, err := executeUniversePipeline(ctx, pipeline.Pipeline, nil, execCtx, registry)
		if err != nil {
			return nil, StepSummary{}, nil, err
		}
		decisions = append(decisions, snapshot.Decisions...)
		for _, candidate := range snapshot.Candidates {
			if existing, ok := seen[candidate.Symbol]; ok {
				seen[candidate.Symbol] = mergeCandidate(existing, candidate)
				continue
			}
			seen[candidate.Symbol] = cloneCandidate(candidate)
		}
	}
	out := mapCandidates(seen)
	return out, StepSummary{Reason: "combine union"}, decisions, nil
}

func combineIntersect(ctx context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, registry SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	pipelines, err := subPipelines(step)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	counts := map[string]int{}
	values := map[string]Candidate{}
	decisions := make([]Decision, 0)
	for _, pipeline := range pipelines {
		snapshot, err := executeUniversePipeline(ctx, pipeline.Pipeline, nil, execCtx, registry)
		if err != nil {
			return nil, StepSummary{}, nil, err
		}
		decisions = append(decisions, snapshot.Decisions...)
		for _, candidate := range snapshot.Candidates {
			counts[candidate.Symbol]++
			values[candidate.Symbol] = mergeCandidate(values[candidate.Symbol], candidate)
		}
	}
	out := make([]Candidate, 0)
	for symbol, count := range counts {
		if count == len(pipelines) {
			out = append(out, values[symbol])
		}
	}
	sortCandidates(out)
	return out, StepSummary{Reason: "combine intersect"}, decisions, nil
}

func combineDifference(ctx context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, registry SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	pipelines, err := subPipelines(step)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	if len(pipelines) == 0 {
		return nil, StepSummary{}, nil, oops.In("universe").New("combine.difference requires pipelines")
	}
	first, err := executeUniversePipeline(ctx, pipelines[0].Pipeline, nil, execCtx, registry)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	blocked := map[string]struct{}{}
	decisions := append([]Decision(nil), first.Decisions...)
	for _, pipeline := range pipelines[1:] {
		snapshot, err := executeUniversePipeline(ctx, pipeline.Pipeline, nil, execCtx, registry)
		if err != nil {
			return nil, StepSummary{}, nil, err
		}
		decisions = append(decisions, snapshot.Decisions...)
		for _, symbol := range snapshot.Symbols {
			blocked[symbol] = struct{}{}
		}
	}
	out := make([]Candidate, 0)
	for _, candidate := range first.Candidates {
		if _, ok := blocked[candidate.Symbol]; !ok {
			out = append(out, candidate)
		}
	}
	return out, StepSummary{Reason: "combine difference"}, decisions, nil
}

func combineConcat(ctx context.Context, step StepSpec, _ []Candidate, execCtx ExecutionContext, registry SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	pipelines, err := subPipelines(step)
	if err != nil {
		return nil, StepSummary{}, nil, err
	}
	out := make([]Candidate, 0)
	decisions := make([]Decision, 0)
	for _, pipeline := range pipelines {
		snapshot, err := executeUniversePipeline(ctx, pipeline.Pipeline, nil, execCtx, registry)
		if err != nil {
			return nil, StepSummary{}, nil, err
		}
		out = append(out, snapshot.Candidates...)
		decisions = append(decisions, snapshot.Decisions...)
	}
	return out, StepSummary{Reason: "combine concat"}, decisions, nil
}

func debugSnapshot(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	fields, _ := stringSliceParam(step.Params, "fields")
	summary := StepSummary{Reason: "debug snapshot", Fields: map[string]any{"fields": fields, "sample": input[:min(len(input), 5)]}}
	out := cloneCandidates(input)
	return out, summary, includeDecisions(execCtx.SelectionTime, step.ID, out, "debug_snapshot"), nil
}

func debugAssertCount(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	minCount := intParamDefault(step.Params, "min", math.MinInt)
	maxCount := intParamDefault(step.Params, "max", math.MaxInt)
	if len(input) < minCount || len(input) > maxCount {
		return nil, StepSummary{}, nil, oops.In("universe").With("count", len(input), "min", minCount, "max", maxCount).New("debug assert count failed")
	}
	out := cloneCandidates(input)
	return out, StepSummary{Reason: "assert count"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "assert_count"), nil
}

func debugSample(_ context.Context, step StepSpec, input []Candidate, execCtx ExecutionContext, _ SelectorRegistry) ([]Candidate, StepSummary, []Decision, error) {
	count := intParamDefault(step.Params, "count", len(input))
	out := cloneCandidates(input)
	if count < len(out) {
		out = out[:count]
	}
	return out, StepSummary{Reason: "debug sample"}, includeDecisions(execCtx.SelectionTime, step.ID, out, "sample"), nil
}

type subPipeline struct {
	Name     string
	Pipeline []StepSpec
}

func subPipelines(step StepSpec) ([]subPipeline, error) {
	raw, ok := step.Params["pipelines"]
	if !ok {
		return nil, oops.In("universe").New("combine selector requires pipelines")
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, oops.In("universe").New("combine pipelines must be sequence")
	}
	out := make([]subPipeline, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, oops.In("universe").New("combine pipeline must be mapping")
		}
		name := stringParamDefault(object, "name", "")
		stepsRaw, ok := object["pipeline"].([]any)
		if !ok {
			return nil, oops.In("universe").New("combine pipeline requires pipeline sequence")
		}
		steps := make([]StepSpec, 0, len(stepsRaw))
		for _, rawStep := range stepsRaw {
			step, err := stepFromAny(rawStep)
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
		}
		out = append(out, subPipeline{Name: name, Pipeline: steps})
	}
	return out, nil
}

func stepFromAny(raw any) (StepSpec, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return StepSpec{}, oops.In("universe").Wrapf(err, "encode sub pipeline step")
	}
	var step StepSpec
	if err := json.Unmarshal(payload, &step); err != nil {
		return StepSpec{}, oops.In("universe").Wrapf(err, "decode sub pipeline step")
	}
	return step, nil
}

func filterCandidates(stepID string, at time.Time, input []Candidate, reason string, keep func(Candidate) bool) ([]Candidate, StepSummary, []Decision, error) {
	out := make([]Candidate, 0, len(input))
	decisions := make([]Decision, 0, len(input))
	for _, candidate := range input {
		if keep(candidate) {
			out = append(out, cloneCandidate(candidate))
			decisions = append(decisions, includeDecision(at, stepID, candidate.Symbol, reason))
		} else {
			decisions = append(decisions, excludeDecision(at, stepID, candidate.Symbol, reason))
		}
	}
	return out, StepSummary{Reason: reason}, decisions, nil
}

func evalUniverseExpr(expr map[string]any, candidate Candidate) bool {
	if raw, ok := expr["all"].([]any); ok {
		for _, item := range raw {
			child, _ := item.(map[string]any)
			if !evalUniverseExpr(child, candidate) {
				return false
			}
		}
		return true
	}
	if raw, ok := expr["any"].([]any); ok {
		for _, item := range raw {
			child, _ := item.(map[string]any)
			if evalUniverseExpr(child, candidate) {
				return true
			}
		}
		return false
	}
	if raw, ok := expr["not"].(map[string]any); ok {
		return !evalUniverseExpr(raw, candidate)
	}
	for _, op := range []string{"gt", "gte", "lt", "lte", "eq", "between"} {
		if raw, ok := expr[op].([]any); ok && len(raw) == 2 {
			return compareAny(exprValue(raw[0], candidate), op, exprValue(raw[1], candidate))
		}
	}
	return false
}

func exprValue(raw any, candidate Candidate) any {
	object, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	if field, ok := stringParam(object, "field"); ok {
		return candidate.Fields[field]
	}
	if value, ok := object["value"]; ok {
		return value
	}
	return raw
}

func windowMetric(id string, rows []Candidate, field string, window int) (float64, bool) {
	if window <= 0 || len(rows) < window {
		return 0, false
	}
	windowRows := rows[len(rows)-window:]
	switch id {
	case "return":
		first := floatField(windowRows[0], field)
		last := floatField(windowRows[len(windowRows)-1], field)
		if first == 0 {
			return 0, false
		}
		return last/first - 1, true
	case "average":
		var sum float64
		for _, row := range windowRows {
			sum += floatField(row, field)
		}
		return sum / float64(len(windowRows)), true
	case "max_drawdown":
		peak := math.Inf(-1)
		maxDD := 0.0
		for _, row := range windowRows {
			value := floatField(row, field)
			if value > peak {
				peak = value
			}
			if peak > 0 {
				dd := value/peak - 1
				if dd < maxDD {
					maxDD = dd
				}
			}
		}
		return maxDD, true
	default:
		return 0, false
	}
}

func closedBars(bars []Bar, selection time.Time, closedOnly bool) []Bar {
	if selection.IsZero() || !closedOnly {
		return append([]Bar(nil), bars...)
	}
	out := make([]Bar, 0, len(bars))
	for _, bar := range bars {
		if bar.Time.Before(selection) {
			out = append(out, bar)
		}
	}
	return out
}

func barFields(bar Bar) map[string]any {
	return map[string]any{
		"trading_date":  bar.Time.Format(time.DateOnly),
		"time":          bar.Time.Format(time.DateOnly),
		"open":          bar.Open,
		"high":          bar.High,
		"low":           bar.Low,
		"close":         bar.Close,
		"volume":        bar.Volume,
		"traded_amount": bar.TradedAmount,
	}
}

func latestBarsBySymbol(bars []Bar) map[string]Bar {
	out := map[string]Bar{}
	for _, bar := range bars {
		current, ok := out[bar.Symbol]
		if !ok || bar.Time.After(current.Time) {
			out[bar.Symbol] = bar
		}
	}
	return out
}

func candidatesFromAny(raw any) ([]Candidate, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, oops.In("universe").New("candidate rows must be sequence")
	}
	out := make([]Candidate, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, oops.In("universe").New("candidate row must be mapping")
		}
		symbol := valueString(object["symbol"])
		if strings.TrimSpace(symbol) == "" {
			return nil, oops.In("universe").New("candidate row requires symbol")
		}
		fields := make(map[string]any, len(object))
		for key, value := range object {
			if key != "symbol" && key != "tags" {
				fields[key] = value
			}
		}
		tags, _ := stringSliceParam(object, "tags")
		out = append(out, Candidate{Symbol: symbol, Fields: fields, Tags: tags})
	}
	return out, nil
}

func candidatesFromSymbols(symbols []string) []Candidate {
	out := make([]Candidate, 0, len(symbols))
	for _, symbol := range symbols {
		out = append(out, newCandidate(symbol))
	}
	return out
}

func newCandidate(symbol string) Candidate {
	return Candidate{Symbol: strings.TrimSpace(symbol), Fields: map[string]any{}}
}

func cloneCandidate(candidate Candidate) Candidate {
	fields := make(map[string]any, len(candidate.Fields))
	for key, value := range candidate.Fields {
		fields[key] = value
	}
	return Candidate{Symbol: candidate.Symbol, Fields: fields, Tags: append([]string(nil), candidate.Tags...)}
}

func cloneCandidates(candidates []Candidate) []Candidate {
	out := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, cloneCandidate(candidate))
	}
	return out
}

func mergeCandidate(left, right Candidate) Candidate {
	out := cloneCandidate(left)
	if out.Symbol == "" {
		out.Symbol = right.Symbol
	}
	ensureFields(&out)
	for key, value := range right.Fields {
		out.Fields[key] = value
	}
	out.Tags = mergeTags(out.Tags, right.Tags)
	return out
}

func mapCandidates(values map[string]Candidate) []Candidate {
	out := make([]Candidate, 0, len(values))
	for _, candidate := range values {
		out = append(out, candidate)
	}
	sortCandidates(out)
	return out
}

func sortCandidates(candidates []Candidate) {
	slices.SortFunc(candidates, func(a, b Candidate) int {
		if a.Symbol < b.Symbol {
			return -1
		}
		if a.Symbol > b.Symbol {
			return 1
		}
		return compareField(a, b, "trading_date")
	})
}

func sortCandidatesByField(candidates []Candidate, field string, order string) {
	slices.SortFunc(candidates, func(a, b Candidate) int {
		cmp := compareValues(a.Fields[field], b.Fields[field])
		if cmp == 0 {
			cmp = strings.Compare(a.Symbol, b.Symbol)
		}
		if order == "desc" {
			return -cmp
		}
		return cmp
	})
}

func compareField(a, b Candidate, field string) int {
	return compareValues(a.Fields[field], b.Fields[field])
}

func compareValues(left, right any) int {
	leftFloat, leftOK := numeric(left)
	rightFloat, rightOK := numeric(right)
	if leftOK && rightOK {
		switch {
		case leftFloat < rightFloat:
			return -1
		case leftFloat > rightFloat:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(valueString(left), valueString(right))
}

func symbolsFromCandidates(candidates []Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.Symbol)
	}
	return out
}

func includeDecisions(at time.Time, stepID string, candidates []Candidate, reason string) []Decision {
	out := make([]Decision, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, includeDecision(at, stepID, candidate.Symbol, reason))
	}
	return out
}

func includeDecision(at time.Time, stepID, symbol, reason string) Decision {
	return Decision{Time: at, StepID: stepID, Symbol: symbol, Action: "included", Reason: reason}
}

func excludeDecision(at time.Time, stepID, symbol, reason string) Decision {
	return Decision{Time: at, StepID: stepID, Symbol: symbol, Action: "excluded", Reason: reason}
}

func candidatesBySymbol(candidates []Candidate) map[string][]Candidate {
	out := map[string][]Candidate{}
	for _, candidate := range candidates {
		out[candidate.Symbol] = append(out[candidate.Symbol], cloneCandidate(candidate))
	}
	return out
}

func groupCandidates(candidates []Candidate, field string) map[string][]Candidate {
	out := map[string][]Candidate{}
	for _, candidate := range candidates {
		key := valueString(candidate.Fields[field])
		out[key] = append(out[key], cloneCandidate(candidate))
	}
	return out
}

func barsFromCandidates(candidates []Candidate) []Bar {
	out := make([]Bar, 0, len(candidates))
	for _, candidate := range candidates {
		tradingDate, _ := dateField(candidate, "trading_date")
		out = append(out, Bar{
			Time:         tradingDate,
			Symbol:       candidate.Symbol,
			Open:         floatField(candidate, "open"),
			High:         floatField(candidate, "high"),
			Low:          floatField(candidate, "low"),
			Close:        floatField(candidate, "close"),
			Volume:       floatField(candidate, "volume"),
			TradedAmount: floatField(candidate, "traded_amount"),
		})
	}
	return out
}

func ensureFields(candidate *Candidate) {
	if candidate.Fields == nil {
		candidate.Fields = map[string]any{}
	}
}

func mergeTags(left []string, right []string) []string {
	seen := set(left)
	out := append([]string(nil), left...)
	for _, tag := range right {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func set(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func mapParam(params map[string]any, key string) (map[string]any, bool) {
	if params == nil {
		return nil, false
	}
	value, ok := params[key]
	if !ok {
		return nil, false
	}
	object, ok := value.(map[string]any)
	return object, ok
}

func stringParam(params map[string]any, key string) (string, bool) {
	if params == nil {
		return "", false
	}
	value, ok := params[key]
	if !ok {
		return "", false
	}
	return valueString(value), true
}

func stringParamDefault(params map[string]any, key string, fallback string) string {
	value, ok := stringParam(params, key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stringSliceParam(params map[string]any, key string) ([]string, bool) {
	if params == nil {
		return nil, false
	}
	value, ok := params[key]
	if !ok {
		return nil, false
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, valueString(item))
		}
		return out, true
	case string:
		if typed == "" {
			return nil, false
		}
		return []string{typed}, true
	default:
		return nil, false
	}
}

func intParamDefault(params map[string]any, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func floatParamDefault(params map[string]any, key string, fallback float64) float64 {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	parsed, ok := numeric(value)
	if !ok {
		return fallback
	}
	return parsed
}

func boolParamDefault(params map[string]any, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case nil:
		return ""
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toJSON(typed)), `"`, ""))
	}
}

func toJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func normalizeScalar(value any) any {
	if parsed, ok := numeric(value); ok {
		return parsed
	}
	return value
}

func numeric(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		trimmed := strings.TrimSpace(strings.ReplaceAll(typed, ",", ""))
		parsed, err := strconv.ParseFloat(trimmed, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func toFloat(value any) float64 {
	parsed, _ := numeric(value)
	return parsed
}

func floatField(candidate Candidate, field string) float64 {
	value, _ := numeric(candidate.Fields[field])
	return value
}

func dateField(candidate Candidate, field string) (time.Time, bool) {
	text := valueString(candidate.Fields[field])
	if text == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.DateOnly, text)
	return parsed, err == nil
}

func earliestBarDate(bars []Bar, symbol string) (time.Time, bool) {
	var out time.Time
	for _, bar := range bars {
		if bar.Symbol != symbol {
			continue
		}
		if out.IsZero() || bar.Time.Before(out) {
			out = bar.Time
		}
	}
	return out, !out.IsZero()
}

func compareAny(left any, op string, right any) bool {
	cmp := compareValues(left, right)
	switch op {
	case "gt":
		return cmp > 0
	case "gte":
		return cmp >= 0
	case "lt":
		return cmp < 0
	case "lte":
		return cmp <= 0
	case "eq":
		return cmp == 0
	case "between":
		values, ok := right.([]any)
		if !ok || len(values) != 2 {
			return false
		}
		return compareValues(left, values[0]) >= 0 && compareValues(left, values[1]) <= 0
	default:
		return false
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
