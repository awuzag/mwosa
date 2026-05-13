package universe

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	core "github.com/ev3rlit/mwosa/packages/universe"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	strategyservice "github.com/ev3rlit/mwosa/service/strategy"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

const KindScreenRun = "ScreenRun"

type DailyBarRepository interface {
	QueryDailyBars(ctx context.Context, query daily.Query) ([]dailybar.Bar, error)
}

type ScreenRepository interface {
	GetScreenRun(ctx context.Context, ref string) (strategyservice.ScreenRunDetail, error)
}

type ScreenRunner interface {
	Screen(ctx context.Context, req strategyservice.ScreenStrategyRequest) (strategyservice.ScreenRunDetail, error)
}

type Runner struct {
	reader       DailyBarRepository
	screenRepo   ScreenRepository
	screenRunner ScreenRunner
}

type ContextRequest struct {
	YAMLPath     string
	Market       string
	SecurityType string
	From         time.Time
	To           time.Time
	Pipeline     []core.StepSpec
}

type ScreenRunSpec struct {
	Kind          string          `json:"kind" yaml:"kind"`
	SchemaVersion int             `json:"schema_version" yaml:"schema_version"`
	Data          ScreenDataSpec  `json:"data" yaml:"data"`
	Pipeline      []core.StepSpec `json:"pipeline" yaml:"pipeline"`
}

type ScreenDataSpec struct {
	Market       string `json:"market" yaml:"market"`
	SecurityType string `json:"security_type" yaml:"security_type"`
	AsOf         string `json:"as_of,omitempty" yaml:"as_of,omitempty"`
	From         string `json:"from,omitempty" yaml:"from,omitempty"`
	To           string `json:"to,omitempty" yaml:"to,omitempty"`
}

type ScreenPipelineResult struct {
	Kind            string           `json:"kind"`
	SchemaVersion   int              `json:"schema_version"`
	Market          string           `json:"market"`
	SecurityType    string           `json:"security_type"`
	AsOf            string           `json:"as_of,omitempty"`
	SelectedSymbols []string         `json:"selected_symbols"`
	ResultCount     int              `json:"result_count"`
	Candidates      []core.Candidate `json:"candidates,omitempty"`
	Explain         core.Explain     `json:"explain"`
}

func NewRunner(reader DailyBarRepository, screenRepo ScreenRepository, screenRunner ScreenRunner) (Runner, error) {
	if reader == nil {
		return Runner{}, oops.In("universe_service").New("daily bar reader is nil")
	}
	return Runner{reader: reader, screenRepo: screenRepo, screenRunner: screenRunner}, nil
}

func (r Runner) Explain(ctx context.Context, req ContextRequest, plan core.Plan) (core.Explain, error) {
	execCtx, err := r.ExecutionContext(ctx, req)
	if err != nil {
		return core.Explain{}, err
	}
	explain, err := core.BuildSnapshots(ctx, plan, execCtx)
	if err != nil {
		return core.Explain{}, oops.In("universe_service").Wrap(err)
	}
	return explain, nil
}

func (r Runner) ExecutionContext(ctx context.Context, req ContextRequest) (core.ExecutionContext, error) {
	execCtx := core.ExecutionContext{
		From:             req.From,
		To:               req.To,
		Market:           req.Market,
		SecurityType:     req.SecurityType,
		SavedScreens:     make(map[string][]core.Candidate),
		ScreenStrategies: make(map[string][]core.Candidate),
		Files:            make(map[string][]core.Candidate),
		Metadata:         make(map[string]core.Candidate),
		Watchlists:       make(map[string][]core.Candidate),
	}
	if PipelineNeedsDailyBars(req.Pipeline) {
		bars, err := r.loadDailyBars(ctx, req)
		if err != nil {
			return core.ExecutionContext{}, err
		}
		execCtx.DailyBars = bars
	}
	if err := r.loadExternalSources(ctx, req.YAMLPath, req.Pipeline, &execCtx); err != nil {
		return core.ExecutionContext{}, err
	}
	return execCtx, nil
}

func (r Runner) InspectScreenPipeline(ctx context.Context, path string) (ScreenPipelineResult, error) {
	spec, err := LoadScreenRunFile(ctx, path)
	if err != nil {
		return ScreenPipelineResult{}, err
	}
	plan, req, err := compileScreenRun(spec, path)
	if err != nil {
		return ScreenPipelineResult{}, err
	}
	explain, err := r.Explain(ctx, req, plan)
	if err != nil {
		return ScreenPipelineResult{}, err
	}
	candidates := []core.Candidate(nil)
	if len(explain.Snapshots) > 0 {
		candidates = append([]core.Candidate(nil), explain.Snapshots[len(explain.Snapshots)-1].Candidates...)
	}
	return ScreenPipelineResult{
		Kind:            spec.Kind,
		SchemaVersion:   spec.SchemaVersion,
		Market:          spec.Data.Market,
		SecurityType:    spec.Data.SecurityType,
		AsOf:            spec.Data.AsOf,
		SelectedSymbols: append([]string(nil), explain.SelectedSymbols...),
		ResultCount:     len(explain.SelectedSymbols),
		Candidates:      candidates,
		Explain:         explain,
	}, nil
}

func LoadScreenRunFile(ctx context.Context, path string) (ScreenRunSpec, error) {
	if err := ctx.Err(); err != nil {
		return ScreenRunSpec{}, oops.In("universe_yaml").With("path", path).Wrap(err)
	}
	file, err := os.Open(path)
	if err != nil {
		return ScreenRunSpec{}, oops.In("universe_yaml").With("path", path).Wrapf(err, "read screen pipeline YAML file")
	}
	defer file.Close()
	var spec ScreenRunSpec
	if err := yaml.NewDecoder(file).Decode(&spec); err != nil {
		return ScreenRunSpec{}, oops.In("universe_yaml").With("path", path).Wrapf(err, "decode screen pipeline YAML")
	}
	return spec, nil
}

func compileScreenRun(spec ScreenRunSpec, path string) (core.Plan, ContextRequest, error) {
	errb := oops.In("screen_pipeline").With("path", path)
	if spec.Kind != KindScreenRun {
		return core.Plan{}, ContextRequest{}, errb.With("kind", spec.Kind).New("screen pipeline kind must be ScreenRun")
	}
	if spec.SchemaVersion != 1 {
		return core.Plan{}, ContextRequest{}, errb.With("schema_version", spec.SchemaVersion).New("unsupported screen pipeline schema version")
	}
	if strings.TrimSpace(spec.Data.Market) == "" {
		return core.Plan{}, ContextRequest{}, errb.New("screen pipeline data market is required")
	}
	if strings.TrimSpace(spec.Data.SecurityType) == "" {
		return core.Plan{}, ContextRequest{}, errb.New("screen pipeline data security type is required")
	}
	from, to, asOf, err := screenRunDates(spec.Data)
	if err != nil {
		return core.Plan{}, ContextRequest{}, errb.Wrap(err)
	}
	pipelineSpec := core.PipelineSpec{
		Schedule: core.ScheduleSpec{Frequency: core.ScheduleOnce},
		Pipeline: spec.Pipeline,
	}
	plan, err := core.Compile(pipelineSpec, core.DataWindow{
		Market:       spec.Data.Market,
		SecurityType: spec.Data.SecurityType,
		From:         asOf,
		To:           asOf,
	}, core.DefaultSelectorRegistry())
	if err != nil {
		return core.Plan{}, ContextRequest{}, errb.Wrap(err)
	}
	return plan, ContextRequest{
		YAMLPath:     path,
		Market:       spec.Data.Market,
		SecurityType: spec.Data.SecurityType,
		From:         from,
		To:           to,
		Pipeline:     spec.Pipeline,
	}, nil
}

func screenRunDates(data ScreenDataSpec) (time.Time, time.Time, time.Time, error) {
	asOfText := strings.TrimSpace(data.AsOf)
	if asOfText == "" {
		asOfText = strings.TrimSpace(data.To)
	}
	if asOfText == "" {
		return time.Time{}, time.Time{}, time.Time{}, oops.In("screen_pipeline").New("screen pipeline data as_of is required")
	}
	asOf, err := time.Parse(time.DateOnly, asOfText)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, oops.In("screen_pipeline").With("as_of", asOfText).Wrapf(err, "parse screen pipeline as_of")
	}
	var from time.Time
	if strings.TrimSpace(data.From) != "" {
		parsed, err := time.Parse(time.DateOnly, data.From)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, oops.In("screen_pipeline").With("from", data.From).Wrapf(err, "parse screen pipeline from")
		}
		from = parsed
	}
	to := asOf
	if strings.TrimSpace(data.To) != "" {
		parsed, err := time.Parse(time.DateOnly, data.To)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, oops.In("screen_pipeline").With("to", data.To).Wrapf(err, "parse screen pipeline to")
		}
		to = parsed
	}
	if !from.IsZero() && to.Before(from) {
		return time.Time{}, time.Time{}, time.Time{}, oops.In("screen_pipeline").New("screen pipeline to date must be on or after from date")
	}
	return from, to, asOf, nil
}

func PipelineNeedsDailyBars(pipeline []core.StepSpec) bool {
	for _, step := range pipeline {
		switch step.ID {
		case "source.daily_bars", "source.instrument_master", "filter.has_daily_bars", "filter.listing_age":
			return true
		case "combine.union", "combine.intersect", "combine.difference", "combine.concat":
			for _, sub := range SubPipelineSpecs(step) {
				if PipelineNeedsDailyBars(sub) {
					return true
				}
			}
		}
	}
	return false
}

func MaxLookbackDays(pipeline []core.StepSpec) int {
	maxLookback := 0
	for _, step := range pipeline {
		if value := intParam(step.Params, "lookback_days"); value > maxLookback {
			maxLookback = value
		}
		if step.ID == "transform.window_metrics" {
			if metrics, ok := step.Params["metrics"].(map[string]any); ok {
				for _, raw := range metrics {
					if spec, ok := raw.(map[string]any); ok {
						if params, ok := spec["params"].(map[string]any); ok {
							if value := intParam(params, "window"); value > maxLookback {
								maxLookback = value
							}
						}
					}
				}
			}
		}
		for _, sub := range SubPipelineSpecs(step) {
			if value := MaxLookbackDays(sub); value > maxLookback {
				maxLookback = value
			}
		}
	}
	return maxLookback
}

func (r Runner) loadDailyBars(ctx context.Context, req ContextRequest) ([]core.Bar, error) {
	from := req.From
	if lookback := MaxLookbackDays(req.Pipeline); lookback > 0 {
		if from.IsZero() {
			from = req.To
		}
		from = from.AddDate(0, 0, -lookback)
	}
	fromText := ""
	if !from.IsZero() {
		fromText = from.Format(time.DateOnly)
	}
	rows, err := r.reader.QueryDailyBars(ctx, daily.Query{
		Market:       provider.Market(req.Market),
		SecurityType: provider.SecurityType(req.SecurityType),
		From:         fromText,
		To:           req.To.Format(time.DateOnly),
	})
	if err != nil {
		return nil, oops.In("universe_service").With("market", req.Market, "security_type", req.SecurityType).Wrapf(err, "query universe daily bars")
	}
	bars := make([]core.Bar, 0, len(rows))
	for _, row := range rows {
		bar, err := canonicalDailyBarToUniverseBar(row)
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	return bars, nil
}

func (r Runner) loadExternalSources(ctx context.Context, yamlPath string, pipeline []core.StepSpec, execCtx *core.ExecutionContext) error {
	for _, step := range pipeline {
		switch step.ID {
		case "source.saved_screen":
			name := stringParam(step.Params, "name", stringParam(step.Params, "ref", ""))
			if name == "" {
				return oops.In("universe_service").New("source.saved_screen requires name")
			}
			if _, ok := execCtx.SavedScreens[name]; !ok {
				if r.screenRepo == nil {
					return oops.In("universe_service").With("name", name).New("screen repository is nil")
				}
				detail, err := r.screenRepo.GetScreenRun(ctx, name)
				if err != nil {
					return oops.In("universe_service").With("name", name).Wrapf(err, "load saved screen")
				}
				execCtx.SavedScreens[name] = screenRunItemsToCandidates(detail.Items)
			}
		case "source.screen_strategy":
			name := stringParam(step.Params, "name", "")
			if name == "" {
				return oops.In("universe_service").New("source.screen_strategy requires name")
			}
			if _, ok := execCtx.ScreenStrategies[name]; !ok {
				if r.screenRunner == nil {
					return oops.In("universe_service").With("name", name).New("screen runner is nil")
				}
				detail, err := r.screenRunner.Screen(ctx, strategyservice.ScreenStrategyRequest{Name: name, Alias: stringParam(step.Params, "alias", "")})
				if err != nil {
					return oops.In("universe_service").With("name", name).Wrapf(err, "run screen strategy")
				}
				execCtx.ScreenStrategies[name] = screenRunItemsToCandidates(detail.Items)
			}
		case "source.file":
			path := stringParam(step.Params, "path", "")
			if path == "" {
				return oops.In("universe_service").New("source.file requires path")
			}
			resolved := resolveFilePath(yamlPath, path)
			rows, err := ReadCandidateFile(resolved)
			if err != nil {
				return err
			}
			execCtx.Files[path] = rows
			execCtx.Files[resolved] = rows
		case "combine.union", "combine.intersect", "combine.difference", "combine.concat":
			for _, sub := range SubPipelineSpecs(step) {
				if err := r.loadExternalSources(ctx, yamlPath, sub, execCtx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func screenRunItemsToCandidates(items []strategyservice.ScreenRunItem) []core.Candidate {
	out := make([]core.Candidate, 0, len(items))
	for _, item := range items {
		fields := map[string]any{"ordinal": item.Ordinal}
		var payload map[string]any
		if err := json.Unmarshal(item.PayloadJSON, &payload); err == nil {
			for key, value := range payload {
				fields[key] = value
			}
		}
		symbol := item.Symbol
		if symbol == "" {
			symbol = stringParam(fields, "symbol", "")
		}
		out = append(out, core.Candidate{Symbol: symbol, Fields: fields})
	}
	return out
}

func ReadCandidateFile(path string) ([]core.Candidate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, oops.In("universe_service").With("path", path).Wrapf(err, "open universe source file")
	}
	defer file.Close()
	if strings.HasSuffix(path, ".csv") {
		return readCSV(file, path)
	}
	if strings.HasSuffix(path, ".ndjson") || strings.HasSuffix(path, ".jsonl") {
		return readNDJSON(file, path)
	}
	return readJSON(file, path)
}

func readCSV(reader io.Reader, path string) ([]core.Candidate, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, oops.In("universe_service").With("path", path).Wrapf(err, "read universe csv")
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	out := make([]core.Candidate, 0, len(records)-1)
	for _, record := range records[1:] {
		fields := make(map[string]any, len(headers))
		for i, header := range headers {
			if i < len(record) {
				fields[header] = record[i]
			}
		}
		out = append(out, core.Candidate{Symbol: stringParam(fields, "symbol", ""), Fields: fields})
	}
	return out, nil
}

func readJSON(reader io.Reader, path string) ([]core.Candidate, error) {
	var rows []map[string]any
	if err := json.NewDecoder(reader).Decode(&rows); err != nil {
		return nil, oops.In("universe_service").With("path", path).Wrapf(err, "read universe json")
	}
	return mapsToCandidates(rows), nil
}

func readNDJSON(reader io.Reader, path string) ([]core.Candidate, error) {
	decoder := json.NewDecoder(reader)
	rows := make([]map[string]any, 0)
	for {
		var row map[string]any
		err := decoder.Decode(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, oops.In("universe_service").With("path", path).Wrapf(err, "read universe ndjson")
		}
		rows = append(rows, row)
	}
	return mapsToCandidates(rows), nil
}

func mapsToCandidates(rows []map[string]any) []core.Candidate {
	out := make([]core.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, core.Candidate{Symbol: stringParam(row, "symbol", ""), Fields: row})
	}
	return out
}

func resolveFilePath(yamlPath string, sourcePath string) string {
	if filepath.IsAbs(sourcePath) {
		return sourcePath
	}
	return filepath.Join(filepath.Dir(yamlPath), sourcePath)
}

func SubPipelineSpecs(step core.StepSpec) [][]core.StepSpec {
	raw, ok := step.Params["pipelines"].([]any)
	if !ok {
		return nil
	}
	out := make([][]core.StepSpec, 0, len(raw))
	for _, item := range raw {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rawSteps, ok := object["pipeline"].([]any)
		if !ok {
			continue
		}
		steps := make([]core.StepSpec, 0, len(rawSteps))
		for _, rawStep := range rawSteps {
			payload, err := json.Marshal(rawStep)
			if err != nil {
				continue
			}
			var step core.StepSpec
			if err := json.Unmarshal(payload, &step); err == nil {
				steps = append(steps, step)
			}
		}
		out = append(out, steps)
	}
	return out
}

func canonicalDailyBarToUniverseBar(row dailybar.Bar) (core.Bar, error) {
	errb := oops.In("universe_dailybar_mapper").With("symbol", row.Symbol, "trading_date", row.TradingDate)
	tradingDate, err := time.Parse(time.DateOnly, row.TradingDate)
	if err != nil {
		return core.Bar{}, errb.Wrapf(err, "parse trading date")
	}
	open, err := optionalFloat(row.Open, "opening_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	high, err := optionalFloat(row.High, "highest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	low, err := optionalFloat(row.Low, "lowest_price")
	if err != nil {
		return core.Bar{}, errb.Wrap(err)
	}
	closePrice, err := optionalFloat(row.Close, "closing_price")
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

func intParam(params map[string]any, key string) int {
	value, ok := params[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func stringParam(params map[string]any, key string, fallback string) string {
	value, ok := params[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func optionalFloat(value string, field string) (float64, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, oops.In("universe_dailybar_mapper").With("field", field, "value", value).Wrapf(err, "parse daily bar numeric field")
	}
	return parsed, nil
}
