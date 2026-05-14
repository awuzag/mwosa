package strategy

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ev3rlit/mwosa/packages/hashutil"
	"github.com/ev3rlit/mwosa/packages/idgen"
	universecore "github.com/ev3rlit/mwosa/packages/universe"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	"github.com/itchyny/gojq"
	"github.com/samber/oops"
)

type Engine string

const (
	EngineJQ           Engine = "jq"
	EngineYAMLPipeline Engine = "yaml_pipeline"
)

type ScreenRunStatus string

const (
	ScreenRunSucceeded ScreenRunStatus = "succeeded"
	ScreenRunFailed    ScreenRunStatus = "failed"
)

const defaultInputSchemaVersion = 1

type Strategy struct {
	ID              string     `json:"id" csv:"id"`
	Name            string     `json:"name" csv:"name"`
	Engine          Engine     `json:"engine" csv:"engine"`
	ActiveVersionID string     `json:"active_version_id" csv:"active_version_id"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" csv:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty" csv:"archived_at"`
}

type StrategyVersion struct {
	ID                 string          `json:"id" csv:"id"`
	StrategyID         string          `json:"strategy_id" csv:"strategy_id"`
	Version            int             `json:"version" csv:"version"`
	QueryText          string          `json:"query_text" csv:"query_text"`
	QueryHash          string          `json:"query_hash" csv:"query_hash"`
	InputDataset       string          `json:"input_dataset" csv:"input_dataset"`
	InputSchemaVersion int             `json:"input_schema_version" csv:"input_schema_version"`
	ParamsJSON         json.RawMessage `json:"params,omitempty" csv:"-"`
	SpecJSON           json.RawMessage `json:"spec_json,omitempty" csv:"-"`
	SpecHash           string          `json:"spec_hash" csv:"spec_hash"`
	CreatedAt          time.Time       `json:"created_at" csv:"created_at"`
	Note               string          `json:"note,omitempty" csv:"note"`
}

type ScreenRun struct {
	ID                 string          `json:"id" csv:"id"`
	Alias              string          `json:"alias,omitempty" csv:"alias"`
	StrategyID         string          `json:"strategy_id" csv:"strategy_id"`
	StrategyVersionID  string          `json:"strategy_version_id" csv:"strategy_version_id"`
	QueryHash          string          `json:"query_hash" csv:"query_hash"`
	InputDataset       string          `json:"input_dataset" csv:"input_dataset"`
	InputSchemaVersion int             `json:"input_schema_version" csv:"input_schema_version"`
	ParamsJSON         json.RawMessage `json:"params,omitempty" csv:"-"`
	DataFrom           string          `json:"data_from,omitempty" csv:"data_from"`
	DataTo             string          `json:"data_to,omitempty" csv:"data_to"`
	DataAsOf           string          `json:"data_as_of,omitempty" csv:"data_as_of"`
	StartedAt          time.Time       `json:"started_at" csv:"started_at"`
	FinishedAt         *time.Time      `json:"finished_at,omitempty" csv:"finished_at"`
	Status             ScreenRunStatus `json:"status" csv:"status"`
	ResultCount        int             `json:"result_count" csv:"result_count"`
	ResultHash         string          `json:"result_hash" csv:"result_hash"`
	ResultSizeBytes    int64           `json:"result_size_bytes" csv:"result_size_bytes"`
	SummaryJSON        json.RawMessage `json:"summary,omitempty" csv:"-"`
	ErrorMessage       string          `json:"error_message,omitempty" csv:"error_message"`
}

type ScreenRunItem struct {
	ID          string          `json:"id" csv:"id"`
	ScreenRunID string          `json:"screen_run_id" csv:"screen_run_id"`
	Ordinal     int             `json:"ordinal" csv:"ordinal"`
	Symbol      string          `json:"symbol,omitempty" csv:"symbol"`
	PayloadJSON json.RawMessage `json:"payload" csv:"-"`
}

type ScreenResultItem struct {
	Ordinal     int             `json:"ordinal" csv:"ordinal"`
	Symbol      string          `json:"symbol,omitempty" csv:"symbol"`
	PayloadJSON json.RawMessage `json:"payload" csv:"-"`
}

type ScreenResult struct {
	QueryHash          string             `json:"query_hash" csv:"query_hash"`
	InputDataset       string             `json:"input_dataset" csv:"input_dataset"`
	InputSchemaVersion int                `json:"input_schema_version" csv:"input_schema_version"`
	ResultCount        int                `json:"result_count" csv:"result_count"`
	Items              []ScreenResultItem `json:"items" csv:"-"`
}

type CompareScreenStrategiesRequest struct {
	Names []string
	AsOf  string
	TopN  int
}

type ScreenStrategyComparison struct {
	AsOf       string                         `json:"as_of,omitempty"`
	TopN       int                            `json:"top_n"`
	Strategies []ScreenStrategyComparisonItem `json:"strategies"`
	Overlaps   []ScreenStrategyOverlap        `json:"overlaps"`
}

type ScreenStrategyComparisonItem struct {
	StrategyName string                       `json:"strategy_name" csv:"strategy_name"`
	Engine       Engine                       `json:"engine" csv:"engine"`
	Version      int                          `json:"version" csv:"version"`
	SpecHash     string                       `json:"spec_hash" csv:"spec_hash"`
	DataAsOf     string                       `json:"data_as_of,omitempty" csv:"data_as_of"`
	ResultCount  int                          `json:"result_count" csv:"result_count"`
	TopSymbols   []string                     `json:"top_symbols" csv:"-"`
	Metrics      ScreenStrategyCompareMetrics `json:"metrics" csv:"-"`
}

type ScreenStrategyCompareMetrics struct {
	AverageReturn20D    *float64 `json:"average_return_20d,omitempty"`
	MedianReturn20D     *float64 `json:"median_return_20d,omitempty"`
	AverageMaxDD20D     *float64 `json:"average_max_dd_20d,omitempty"`
	MedianMaxDD20D      *float64 `json:"median_max_dd_20d,omitempty"`
	AverageTradedAmount *float64 `json:"average_traded_amount,omitempty"`
}

type ScreenStrategyOverlap struct {
	LeftStrategy  string   `json:"left_strategy"`
	RightStrategy string   `json:"right_strategy"`
	TopN          int      `json:"top_n"`
	Count         int      `json:"count"`
	Symbols       []string `json:"symbols"`
}

type StrategyDetail struct {
	Strategy      Strategy        `json:"strategy"`
	ActiveVersion StrategyVersion `json:"active_version"`
}

type StrategyVersionRef struct {
	Version  string
	SpecHash string
}

type ScreenRunDetail struct {
	Run             ScreenRun       `json:"run"`
	Strategy        Strategy        `json:"strategy"`
	StrategyVersion StrategyVersion `json:"strategy_version"`
	Items           []ScreenRunItem `json:"items"`
}

type ScreenStrategySpec struct {
	Kind          string                      `json:"kind" yaml:"kind"`
	SchemaVersion int                         `json:"schema_version" yaml:"schema_version"`
	Name          string                      `json:"name" yaml:"name"`
	Engine        Engine                      `json:"engine" yaml:"engine"`
	JQ            *JQStrategySpec             `json:"jq,omitempty" yaml:"jq,omitempty"`
	Pipeline      *ScreenPipelineStrategySpec `json:"pipeline_strategy,omitempty" yaml:"pipeline_strategy,omitempty"`
}

type JQStrategySpec struct {
	InputDataset string          `json:"input_dataset" yaml:"input_dataset"`
	QueryText    string          `json:"query_text" yaml:"query_text"`
	Params       json.RawMessage `json:"params,omitempty" yaml:"params,omitempty"`
}

type ScreenPipelineStrategySpec struct {
	Data     ScreenPipelineDataSpec  `json:"data" yaml:"data"`
	Pipeline []universecore.StepSpec `json:"pipeline" yaml:"pipeline"`
}

type ScreenPipelineDataSpec struct {
	Market       string `json:"market" yaml:"market"`
	SecurityType string `json:"security_type" yaml:"security_type"`
	AsOf         string `json:"as_of,omitempty" yaml:"as_of,omitempty"`
	From         string `json:"from,omitempty" yaml:"from,omitempty"`
	To           string `json:"to,omitempty" yaml:"to,omitempty"`
}

type PipelineExecutionResult struct {
	InputDataset       string
	InputSchemaVersion int
	DataFrom           string
	DataTo             string
	DataAsOf           string
	Rows               []json.RawMessage
}

type Repository interface {
	CreateStrategyWithVersion(ctx context.Context, strategy Strategy, version StrategyVersion) (StrategyDetail, error)
	ListStrategies(ctx context.Context) ([]StrategyDetail, error)
	GetStrategy(ctx context.Context, name string) (StrategyDetail, error)
	GetStrategyVersion(ctx context.Context, name string, ref StrategyVersionRef) (StrategyDetail, error)
	AddStrategyVersion(ctx context.Context, name string, engine Engine, version StrategyVersion, now time.Time) (StrategyDetail, error)
	ArchiveStrategy(ctx context.Context, name string, archivedAt time.Time) error
	CreateScreenRun(ctx context.Context, run ScreenRun, items []ScreenRunItem) (ScreenRunDetail, error)
	ListScreenRuns(ctx context.Context, limit int) ([]ScreenRun, error)
	GetScreenRun(ctx context.Context, ref string) (ScreenRunDetail, error)
}

type Dataset struct {
	Name          string
	SchemaVersion int
	Records       []json.RawMessage
}

type DatasetReader interface {
	ReadDataset(ctx context.Context, name string) (Dataset, error)
}

type PipelineExecutor interface {
	ExecuteScreenStrategyPipeline(ctx context.Context, spec ScreenStrategySpec) (PipelineExecutionResult, error)
}

type pipelineExecutorSlot struct {
	executor PipelineExecutor
}

type Service struct {
	repo             Repository
	dataset          DatasetReader
	pipelineExecutor *pipelineExecutorSlot
	now              func() time.Time
}

func NewService(repo Repository, dataset DatasetReader) (Service, error) {
	errb := oops.In("strategy_service")
	if repo == nil {
		return Service{}, errb.New("strategy repository is nil")
	}
	if dataset == nil {
		return Service{}, errb.New("strategy dataset reader is nil")
	}
	return Service{
		repo:             repo,
		dataset:          dataset,
		pipelineExecutor: &pipelineExecutorSlot{},
		now:              time.Now,
	}, nil
}

func (s Service) SetPipelineExecutor(executor PipelineExecutor) {
	if s.pipelineExecutor == nil {
		return
	}
	s.pipelineExecutor.executor = executor
}

type CreateStrategyRequest struct {
	Name         string
	Engine       Engine
	InputDataset string
	QueryText    string
}

type UpsertStrategyRequest struct {
	Name string
	Spec ScreenStrategySpec
}

func (s Service) Create(ctx context.Context, req CreateStrategyRequest) (StrategyDetail, error) {
	errb := oops.In("strategy_service").With("name", req.Name, "engine", req.Engine, "input_dataset", req.InputDataset)
	if s.repo == nil {
		return StrategyDetail{}, errb.New("strategy repository is nil")
	}
	if req.Engine == "" {
		req.Engine = EngineJQ
	}
	if req.Engine != EngineJQ {
		return StrategyDetail{}, errb.Errorf("unsupported strategy engine: %s", req.Engine)
	}
	spec := JQScreenStrategySpec(req.Name, req.InputDataset, req.QueryText, nil)
	if err := validateScreenStrategySpec(spec); err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}
	specJSON, specHash, err := canonicalStrategySpecPayload(spec)
	if err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}

	strategyID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy id")
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy version id")
	}
	now := s.now()
	queryHash := hashutil.SHA256([]byte(req.QueryText))
	strategy := Strategy{
		ID:              strategyID,
		Name:            req.Name,
		Engine:          req.Engine,
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := StrategyVersion{
		ID:                 versionID,
		StrategyID:         strategyID,
		Version:            1,
		QueryText:          req.QueryText,
		QueryHash:          queryHash,
		InputDataset:       req.InputDataset,
		InputSchemaVersion: defaultInputSchemaVersion,
		ParamsJSON:         json.RawMessage(`{}`),
		SpecJSON:           specJSON,
		SpecHash:           specHash,
		CreatedAt:          now,
	}
	return s.repo.CreateStrategyWithVersion(ctx, strategy, version)
}

func (s Service) Upsert(ctx context.Context, req UpsertStrategyRequest) (StrategyDetail, error) {
	errb := oops.In("strategy_service").With("name", req.Name)
	if s.repo == nil {
		return StrategyDetail{}, errb.New("strategy repository is nil")
	}
	spec := req.Spec
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = req.Name
	}
	if spec.Name != req.Name {
		return StrategyDetail{}, errb.With("spec_name", spec.Name).Errorf("screen strategy name mismatch: cli=%s yaml=%s", req.Name, spec.Name)
	}
	if err := validateScreenStrategySpec(spec); err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}
	specJSON, specHash, err := canonicalStrategySpecPayload(spec)
	if err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}
	existing, err := s.repo.GetStrategy(ctx, req.Name)
	if err != nil {
		if !strings.Contains(err.Error(), "strategy not found") {
			return StrategyDetail{}, errb.Wrapf(err, "load strategy before upsert")
		}
		return s.createFromCanonicalSpec(ctx, spec, specJSON, specHash)
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy version id")
	}
	now := s.now()
	version := strategyVersionFromSpec(versionID, existing.Strategy.ID, existing.ActiveVersion.Version+1, spec, specJSON, specHash, now)
	return s.repo.AddStrategyVersion(ctx, req.Name, spec.Engine, version, now)
}

func (s Service) createFromCanonicalSpec(ctx context.Context, spec ScreenStrategySpec, specJSON json.RawMessage, specHash string) (StrategyDetail, error) {
	errb := oops.In("strategy_service").With("name", spec.Name, "engine", spec.Engine)
	strategyID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy id")
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy version id")
	}
	now := s.now()
	strategy := Strategy{
		ID:              strategyID,
		Name:            spec.Name,
		Engine:          spec.Engine,
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := strategyVersionFromSpec(versionID, strategyID, 1, spec, specJSON, specHash, now)
	return s.repo.CreateStrategyWithVersion(ctx, strategy, version)
}

func (s Service) List(ctx context.Context) ([]StrategyDetail, error) {
	if s.repo == nil {
		return nil, oops.In("strategy_service").New("strategy repository is nil")
	}
	return s.repo.ListStrategies(ctx)
}

func (s Service) Inspect(ctx context.Context, name string) (StrategyDetail, error) {
	if strings.TrimSpace(name) == "" {
		return StrategyDetail{}, oops.In("strategy_service").New("inspect strategy requires name")
	}
	return s.repo.GetStrategy(ctx, name)
}

type UpdateStrategyRequest struct {
	Name      string
	QueryText string
}

func (s Service) Update(ctx context.Context, req UpdateStrategyRequest) (StrategyDetail, error) {
	errb := oops.In("strategy_service").With("name", req.Name)
	if strings.TrimSpace(req.Name) == "" {
		return StrategyDetail{}, errb.New("update strategy requires name")
	}
	detail, err := s.repo.GetStrategy(ctx, req.Name)
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "load strategy before update")
	}
	if err := validateStrategySource(detail.Strategy.Name, detail.Strategy.Engine, detail.ActiveVersion.InputDataset, req.QueryText); err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}
	spec := JQScreenStrategySpec(detail.Strategy.Name, detail.ActiveVersion.InputDataset, req.QueryText, normalizeJSON(detail.ActiveVersion.ParamsJSON))
	specJSON, specHash, err := canonicalStrategySpecPayload(spec)
	if err != nil {
		return StrategyDetail{}, errb.Wrap(err)
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return StrategyDetail{}, errb.Wrapf(err, "generate strategy version id")
	}
	now := s.now()
	version := StrategyVersion{
		ID:                 versionID,
		StrategyID:         detail.Strategy.ID,
		Version:            detail.ActiveVersion.Version + 1,
		QueryText:          req.QueryText,
		QueryHash:          hashutil.SHA256([]byte(req.QueryText)),
		InputDataset:       detail.ActiveVersion.InputDataset,
		InputSchemaVersion: detail.ActiveVersion.InputSchemaVersion,
		ParamsJSON:         normalizeJSON(detail.ActiveVersion.ParamsJSON),
		SpecJSON:           specJSON,
		SpecHash:           specHash,
		CreatedAt:          now,
	}
	return s.repo.AddStrategyVersion(ctx, req.Name, detail.Strategy.Engine, version, now)
}

func (s Service) Delete(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return oops.In("strategy_service").New("delete strategy requires name")
	}
	return s.repo.ArchiveStrategy(ctx, name, s.now())
}

type ScreenStrategyRequest struct {
	Name     string
	Alias    string
	Version  string
	SpecHash string
}

type screenExecutionResult struct {
	Dataset    Dataset
	Pipeline   PipelineExecutionResult
	Rows       []json.RawMessage
	IsPipeline bool
}

type ScreenJQRequest struct {
	InputDataset string
	QueryText    string
}

func (s Service) ScreenJQ(ctx context.Context, req ScreenJQRequest) (ScreenResult, error) {
	errb := oops.In("strategy_service").With("input_dataset", req.InputDataset)
	if s.dataset == nil {
		return ScreenResult{}, errb.New("strategy dataset reader is nil")
	}
	if strings.TrimSpace(req.InputDataset) == "" {
		return ScreenResult{}, errb.New("screen jq input dataset is required")
	}
	if strings.TrimSpace(req.QueryText) == "" {
		return ScreenResult{}, errb.New("screen jq query is required")
	}
	dataset, rows, err := s.executeJQAgainstDataset(ctx, req.InputDataset, req.QueryText)
	if err != nil {
		return ScreenResult{}, errb.Wrap(err)
	}
	return screenResultFromRows(req.QueryText, dataset, rows), nil
}

func (s Service) Screen(ctx context.Context, req ScreenStrategyRequest) (ScreenRunDetail, error) {
	errb := oops.In("strategy_service").With("name", req.Name, "alias", req.Alias)
	if strings.TrimSpace(req.Name) == "" {
		return ScreenRunDetail{}, errb.New("screen strategy requires name")
	}
	if strings.TrimSpace(req.Version) != "" && strings.TrimSpace(req.SpecHash) != "" {
		return ScreenRunDetail{}, errb.New("screen strategy requires either version or spec_hash, not both")
	}
	detail, err := s.repo.GetStrategyVersion(ctx, req.Name, StrategyVersionRef{
		Version:  req.Version,
		SpecHash: req.SpecHash,
	})
	if err != nil {
		return ScreenRunDetail{}, errb.Wrapf(err, "load strategy")
	}
	started := s.now()
	result, err := s.executeStrategyVersion(ctx, detail, "")
	if err != nil {
		return s.recordFailedRun(ctx, detail, req.Alias, started, err)
	}
	if result.IsPipeline {
		return s.recordSucceededPipelineRun(ctx, detail, req.Alias, started, result.Pipeline)
	}
	return s.recordSucceededRun(ctx, detail, req.Alias, started, result.Dataset, result.Rows)
}

func (s Service) CompareScreenStrategies(ctx context.Context, req CompareScreenStrategiesRequest) (ScreenStrategyComparison, error) {
	errb := oops.In("strategy_compare_service").With("as_of", req.AsOf)
	if len(req.Names) < 2 {
		return ScreenStrategyComparison{}, errb.New("compare screen strategies requires at least two strategies")
	}
	if strings.TrimSpace(req.AsOf) != "" {
		if _, err := time.Parse(time.DateOnly, req.AsOf); err != nil {
			return ScreenStrategyComparison{}, errb.Wrapf(err, "parse compare screen strategies as_of")
		}
	}
	topN := req.TopN
	if topN <= 0 {
		topN = 10
	}
	items := make([]ScreenStrategyComparisonItem, 0, len(req.Names))
	for _, name := range req.Names {
		name = strings.TrimSpace(name)
		if name == "" {
			return ScreenStrategyComparison{}, errb.New("compare screen strategies contains empty strategy name")
		}
		detail, err := s.repo.GetStrategyVersion(ctx, name, StrategyVersionRef{Version: "latest"})
		if err != nil {
			return ScreenStrategyComparison{}, errb.With("name", name).Wrapf(err, "load strategy")
		}
		result, err := s.executeStrategyVersion(ctx, detail, req.AsOf)
		if err != nil {
			return ScreenStrategyComparison{}, errb.With("name", name).Wrap(err)
		}
		dataAsOf := req.AsOf
		if result.IsPipeline && result.Pipeline.DataAsOf != "" {
			dataAsOf = result.Pipeline.DataAsOf
		}
		items = append(items, ScreenStrategyComparisonItem{
			StrategyName: detail.Strategy.Name,
			Engine:       detail.Strategy.Engine,
			Version:      detail.ActiveVersion.Version,
			SpecHash:     detail.ActiveVersion.SpecHash,
			DataAsOf:     dataAsOf,
			ResultCount:  len(result.Rows),
			TopSymbols:   topSymbols(result.Rows, topN),
			Metrics:      compareMetrics(result.Rows),
		})
	}
	return ScreenStrategyComparison{
		AsOf:       strings.TrimSpace(req.AsOf),
		TopN:       topN,
		Strategies: items,
		Overlaps:   compareOverlaps(items, topN),
	}, nil
}

func (s Service) executeStrategyVersion(ctx context.Context, detail StrategyDetail, asOfOverride string) (screenExecutionResult, error) {
	errb := oops.In("strategy_service").With("name", detail.Strategy.Name, "engine", detail.Strategy.Engine)
	switch detail.Strategy.Engine {
	case EngineJQ:
		dataset, rows, err := s.executeJQAgainstDataset(ctx, detail.ActiveVersion.InputDataset, detail.ActiveVersion.QueryText)
		if err != nil {
			return screenExecutionResult{}, errb.Wrapf(err, "execute jq strategy")
		}
		return screenExecutionResult{Dataset: dataset, Rows: rows}, nil
	case EngineYAMLPipeline:
		if s.pipelineExecutor == nil || s.pipelineExecutor.executor == nil {
			return screenExecutionResult{}, errb.New("screen pipeline executor is nil")
		}
		spec, err := screenStrategySpecFromVersion(detail)
		if err != nil {
			return screenExecutionResult{}, errb.Wrap(err)
		}
		if strings.TrimSpace(asOfOverride) != "" {
			spec = withScreenStrategyAsOf(spec, asOfOverride)
		}
		result, err := s.pipelineExecutor.executor.ExecuteScreenStrategyPipeline(ctx, spec)
		if err != nil {
			return screenExecutionResult{}, errb.Wrapf(err, "execute yaml pipeline strategy")
		}
		return screenExecutionResult{Pipeline: result, Rows: result.Rows, IsPipeline: true}, nil
	default:
		return screenExecutionResult{}, errb.Errorf("unsupported strategy engine: %s", detail.Strategy.Engine)
	}
}

func (s Service) History(ctx context.Context, limit int) ([]ScreenRun, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListScreenRuns(ctx, limit)
}

func (s Service) InspectScreen(ctx context.Context, ref string) (ScreenRunDetail, error) {
	if strings.TrimSpace(ref) == "" {
		return ScreenRunDetail{}, oops.In("strategy_service").New("inspect screen requires id or alias")
	}
	return s.repo.GetScreenRun(ctx, ref)
}

func (s Service) executeJQAgainstDataset(ctx context.Context, inputDataset string, queryText string) (Dataset, []json.RawMessage, error) {
	errb := oops.In("strategy_service").With("input_dataset", inputDataset)
	dataset, err := s.dataset.ReadDataset(ctx, inputDataset)
	if err != nil {
		return Dataset{}, nil, errb.Wrapf(err, "read input dataset")
	}
	input, err := datasetInputValue(dataset.Records)
	if err != nil {
		return Dataset{}, nil, errb.Wrapf(err, "decode input dataset")
	}
	rows, err := executeJQ(ctx, queryText, input)
	if err != nil {
		return Dataset{}, nil, errb.Wrapf(err, "execute jq")
	}
	return dataset, rows, nil
}

func (s Service) recordSucceededRun(ctx context.Context, detail StrategyDetail, alias string, started time.Time, dataset Dataset, rows []json.RawMessage) (ScreenRunDetail, error) {
	errb := oops.In("strategy_service").With("strategy_id", detail.Strategy.ID, "alias", alias)
	resultJSON, err := json.Marshal(rows)
	if err != nil {
		return ScreenRunDetail{}, errb.Wrapf(err, "encode jq result rows")
	}
	summaryJSON, err := summarizeRows(rows)
	if err != nil {
		return ScreenRunDetail{}, err
	}
	runID, err := idgen.NewUUIDV7()
	if err != nil {
		return ScreenRunDetail{}, errb.Wrapf(err, "generate screen run id")
	}
	finished := s.now()
	run := screenRunFromStrategy(runID, alias, detail, dataset.SchemaVersion, started, &finished)
	run.Status = ScreenRunSucceeded
	run.ResultCount = len(rows)
	run.ResultHash = hashutil.SHA256(resultJSON)
	run.ResultSizeBytes = int64(len(resultJSON))
	run.SummaryJSON = summaryJSON

	items := make([]ScreenRunItem, 0, len(rows))
	for i, row := range rows {
		itemID, err := idgen.NewUUIDV7()
		if err != nil {
			return ScreenRunDetail{}, errb.With("ordinal", i).Wrapf(err, "generate screen run item id")
		}
		items = append(items, ScreenRunItem{
			ID:          itemID,
			ScreenRunID: run.ID,
			Ordinal:     i,
			Symbol:      extractSymbol(row),
			PayloadJSON: row,
		})
	}
	return s.repo.CreateScreenRun(ctx, run, items)
}

func (s Service) recordSucceededPipelineRun(ctx context.Context, detail StrategyDetail, alias string, started time.Time, result PipelineExecutionResult) (ScreenRunDetail, error) {
	errb := oops.In("strategy_service").With("strategy_id", detail.Strategy.ID, "alias", alias)
	resultJSON, err := json.Marshal(result.Rows)
	if err != nil {
		return ScreenRunDetail{}, errb.Wrapf(err, "encode pipeline result rows")
	}
	summaryJSON, err := summarizeRows(result.Rows)
	if err != nil {
		return ScreenRunDetail{}, err
	}
	runID, err := idgen.NewUUIDV7()
	if err != nil {
		return ScreenRunDetail{}, errb.Wrapf(err, "generate screen run id")
	}
	finished := s.now()
	schemaVersion := result.InputSchemaVersion
	if schemaVersion == 0 {
		schemaVersion = detail.ActiveVersion.InputSchemaVersion
	}
	run := screenRunFromStrategy(runID, alias, detail, schemaVersion, started, &finished)
	run.InputDataset = withDefault(result.InputDataset, detail.ActiveVersion.InputDataset)
	run.DataFrom = result.DataFrom
	run.DataTo = result.DataTo
	run.DataAsOf = result.DataAsOf
	run.Status = ScreenRunSucceeded
	run.ResultCount = len(result.Rows)
	run.ResultHash = hashutil.SHA256(resultJSON)
	run.ResultSizeBytes = int64(len(resultJSON))
	run.SummaryJSON = summaryJSON

	items := make([]ScreenRunItem, 0, len(result.Rows))
	for i, row := range result.Rows {
		itemID, err := idgen.NewUUIDV7()
		if err != nil {
			return ScreenRunDetail{}, errb.With("ordinal", i).Wrapf(err, "generate screen run item id")
		}
		items = append(items, ScreenRunItem{
			ID:          itemID,
			ScreenRunID: run.ID,
			Ordinal:     i,
			Symbol:      extractSymbol(row),
			PayloadJSON: row,
		})
	}
	return s.repo.CreateScreenRun(ctx, run, items)
}

func screenResultFromRows(queryText string, dataset Dataset, rows []json.RawMessage) ScreenResult {
	items := make([]ScreenResultItem, 0, len(rows))
	for i, row := range rows {
		items = append(items, ScreenResultItem{
			Ordinal:     i,
			Symbol:      extractSymbol(row),
			PayloadJSON: row,
		})
	}
	return ScreenResult{
		QueryHash:          hashutil.SHA256([]byte(queryText)),
		InputDataset:       dataset.Name,
		InputSchemaVersion: dataset.SchemaVersion,
		ResultCount:        len(rows),
		Items:              items,
	}
}

func withScreenStrategyAsOf(spec ScreenStrategySpec, asOf string) ScreenStrategySpec {
	if spec.Pipeline == nil {
		return spec
	}
	next := spec
	pipeline := *spec.Pipeline
	pipeline.Data.AsOf = asOf
	pipeline.Data.To = asOf
	next.Pipeline = &pipeline
	return next
}

func topSymbols(rows []json.RawMessage, topN int) []string {
	if topN > len(rows) {
		topN = len(rows)
	}
	out := make([]string, 0, topN)
	for _, row := range rows[:topN] {
		symbol := extractSymbol(row)
		if symbol == "" {
			continue
		}
		out = append(out, symbol)
	}
	return out
}

func compareMetrics(rows []json.RawMessage) ScreenStrategyCompareMetrics {
	return ScreenStrategyCompareMetrics{
		AverageReturn20D:    averageMetric(rows, "return_20d"),
		MedianReturn20D:     medianMetric(rows, "return_20d"),
		AverageMaxDD20D:     averageMetric(rows, "max_dd_20d"),
		MedianMaxDD20D:      medianMetric(rows, "max_dd_20d"),
		AverageTradedAmount: averageMetric(rows, "traded_amount"),
	}
}

func averageMetric(rows []json.RawMessage, field string) *float64 {
	values := metricValues(rows, field)
	if len(values) == 0 {
		return nil
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	avg := sum / float64(len(values))
	return &avg
}

func medianMetric(rows []json.RawMessage, field string) *float64 {
	values := metricValues(rows, field)
	if len(values) == 0 {
		return nil
	}
	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		value := values[mid]
		return &value
	}
	value := (values[mid-1] + values[mid]) / 2
	return &value
}

func metricValues(rows []json.RawMessage, field string) []float64 {
	out := make([]float64, 0, len(rows))
	for _, row := range rows {
		var object map[string]any
		if err := json.Unmarshal(row, &object); err != nil {
			continue
		}
		if value, ok := numericAny(object[field]); ok {
			out = append(out, value)
		}
	}
	return out
}

func numericAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(typed), ",", ""), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func compareOverlaps(items []ScreenStrategyComparisonItem, topN int) []ScreenStrategyOverlap {
	out := make([]ScreenStrategyOverlap, 0)
	for i := 0; i < len(items); i++ {
		left := setStrings(items[i].TopSymbols)
		for j := i + 1; j < len(items); j++ {
			symbols := make([]string, 0)
			for _, symbol := range items[j].TopSymbols {
				if _, ok := left[symbol]; ok {
					symbols = append(symbols, symbol)
				}
			}
			sort.Strings(symbols)
			out = append(out, ScreenStrategyOverlap{
				LeftStrategy:  items[i].StrategyName,
				RightStrategy: items[j].StrategyName,
				TopN:          topN,
				Count:         len(symbols),
				Symbols:       symbols,
			})
		}
	}
	return out
}

func setStrings(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func (s Service) recordFailedRun(ctx context.Context, detail StrategyDetail, alias string, started time.Time, runErr error) (ScreenRunDetail, error) {
	runID, err := idgen.NewUUIDV7()
	if err != nil {
		return ScreenRunDetail{}, oops.Join(runErr, oops.In("strategy_service").With("strategy_id", detail.Strategy.ID, "alias", alias).Wrapf(err, "generate failed screen run id"))
	}
	finished := s.now()
	run := screenRunFromStrategy(runID, alias, detail, detail.ActiveVersion.InputSchemaVersion, started, &finished)
	run.Status = ScreenRunFailed
	run.ErrorMessage = runErr.Error()
	saved, saveErr := s.repo.CreateScreenRun(ctx, run, nil)
	if saveErr != nil {
		return saved, oops.Join(runErr, oops.In("strategy_service").Wrapf(saveErr, "save failed screen run"))
	}
	return saved, runErr
}

func screenRunFromStrategy(id string, alias string, detail StrategyDetail, schemaVersion int, started time.Time, finished *time.Time) ScreenRun {
	return ScreenRun{
		ID:                 id,
		Alias:              strings.TrimSpace(alias),
		StrategyID:         detail.Strategy.ID,
		StrategyVersionID:  detail.ActiveVersion.ID,
		QueryHash:          detail.ActiveVersion.QueryHash,
		InputDataset:       detail.ActiveVersion.InputDataset,
		InputSchemaVersion: schemaVersion,
		ParamsJSON:         normalizeJSON(detail.ActiveVersion.ParamsJSON),
		StartedAt:          started,
		FinishedAt:         finished,
	}
}

func validateStrategySource(name string, engine Engine, inputDataset string, queryText string) error {
	errb := oops.In("strategy_service").With("name", name, "engine", engine, "input_dataset", inputDataset)
	if strings.TrimSpace(name) == "" {
		return errb.New("strategy name is required")
	}
	if engine != EngineJQ {
		return errb.Errorf("unsupported strategy engine: %s", engine)
	}
	if strings.TrimSpace(inputDataset) == "" {
		return errb.New("strategy input dataset is required")
	}
	if strings.TrimSpace(queryText) == "" {
		return errb.New("strategy jq query is required")
	}
	return nil
}

func validateScreenStrategySpec(spec ScreenStrategySpec) error {
	errb := oops.In("strategy_service").With("name", spec.Name, "engine", spec.Engine)
	if strings.TrimSpace(spec.Name) == "" {
		return errb.New("strategy name is required")
	}
	if spec.SchemaVersion != defaultInputSchemaVersion {
		return errb.With("schema_version", spec.SchemaVersion).New("unsupported screen strategy schema version")
	}
	switch spec.Engine {
	case EngineJQ:
		if spec.JQ == nil {
			return errb.New("jq screen strategy requires jq spec")
		}
		return validateStrategySource(spec.Name, spec.Engine, spec.JQ.InputDataset, spec.JQ.QueryText)
	case EngineYAMLPipeline:
		if spec.Pipeline == nil {
			return errb.New("yaml pipeline screen strategy requires pipeline spec")
		}
		if strings.TrimSpace(spec.Pipeline.Data.Market) == "" {
			return errb.New("screen strategy data market is required")
		}
		if strings.TrimSpace(spec.Pipeline.Data.SecurityType) == "" {
			return errb.New("screen strategy data security type is required")
		}
		if strings.TrimSpace(spec.Pipeline.Data.AsOf) == "" && strings.TrimSpace(spec.Pipeline.Data.To) == "" {
			return errb.New("screen strategy data as_of is required")
		}
		if len(spec.Pipeline.Pipeline) == 0 {
			return errb.New("screen strategy pipeline requires at least one selector")
		}
		return nil
	default:
		return errb.Errorf("unsupported strategy engine: %s", spec.Engine)
	}
}

func JQScreenStrategySpec(name string, inputDataset string, queryText string, params json.RawMessage) ScreenStrategySpec {
	return ScreenStrategySpec{
		Kind:          "ScreenStrategy",
		SchemaVersion: defaultInputSchemaVersion,
		Name:          name,
		Engine:        EngineJQ,
		JQ: &JQStrategySpec{
			InputDataset: inputDataset,
			QueryText:    queryText,
			Params:       normalizeJSON(params),
		},
	}
}

func strategyVersionFromSpec(id string, strategyID string, versionNumber int, spec ScreenStrategySpec, specJSON json.RawMessage, specHash string, createdAt time.Time) StrategyVersion {
	version := StrategyVersion{
		ID:                 id,
		StrategyID:         strategyID,
		Version:            versionNumber,
		QueryHash:          specHash,
		InputDataset:       "screen_pipeline",
		InputSchemaVersion: spec.SchemaVersion,
		ParamsJSON:         json.RawMessage(`{}`),
		SpecJSON:           specJSON,
		SpecHash:           specHash,
		CreatedAt:          createdAt,
	}
	if spec.Engine == EngineJQ && spec.JQ != nil {
		version.QueryText = spec.JQ.QueryText
		version.QueryHash = hashutil.SHA256([]byte(spec.JQ.QueryText))
		version.InputDataset = spec.JQ.InputDataset
		version.ParamsJSON = normalizeJSON(spec.JQ.Params)
	}
	return version
}

func canonicalStrategySpecPayload(spec ScreenStrategySpec) (json.RawMessage, string, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, "", oops.In("strategy_service").With("name", spec.Name).Wrapf(err, "encode canonical screen strategy spec")
	}
	return data, hashutil.SHA256(data), nil
}

func screenStrategySpecFromVersion(detail StrategyDetail) (ScreenStrategySpec, error) {
	if len(bytes.TrimSpace(detail.ActiveVersion.SpecJSON)) > 0 && string(bytes.TrimSpace(detail.ActiveVersion.SpecJSON)) != "{}" {
		var spec ScreenStrategySpec
		if err := json.Unmarshal(detail.ActiveVersion.SpecJSON, &spec); err != nil {
			return ScreenStrategySpec{}, oops.In("strategy_service").With("name", detail.Strategy.Name).Wrapf(err, "decode canonical screen strategy spec")
		}
		return spec, nil
	}
	return JQScreenStrategySpec(detail.Strategy.Name, detail.ActiveVersion.InputDataset, detail.ActiveVersion.QueryText, detail.ActiveVersion.ParamsJSON), nil
}

func withDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeJSON(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

type screenSummary struct {
	ResultCount int               `json:"result_count"`
	Preview     []json.RawMessage `json:"preview,omitempty"`
}

func summarizeRows(rows []json.RawMessage) (json.RawMessage, error) {
	limit := len(rows)
	if limit > 5 {
		limit = 5
	}
	summary := screenSummary{
		ResultCount: len(rows),
		Preview:     rows[:limit],
	}
	data, err := json.Marshal(summary)
	if err != nil {
		return nil, oops.In("strategy_service").Wrapf(err, "encode screen run summary")
	}
	return data, nil
}

func extractSymbol(raw json.RawMessage) string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	for _, key := range []string{"symbol", "security_code", "srtn_cd", "srtnCd"} {
		value, ok := object[key]
		if !ok {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			return text
		}
	}
	return ""
}

type DailyBarDatasetReader struct {
	reader daily.ReadRepository
	market provider.Market
}

func NewDailyBarDatasetReader(reader daily.ReadRepository, market provider.Market) (DailyBarDatasetReader, error) {
	if reader == nil {
		return DailyBarDatasetReader{}, oops.In("strategy_service").New("daily bar dataset reader repository is nil")
	}
	if market == "" {
		market = provider.MarketKRX
	}
	return DailyBarDatasetReader{reader: reader, market: market}, nil
}

func (r DailyBarDatasetReader) ReadDataset(ctx context.Context, name string) (Dataset, error) {
	errb := oops.In("strategy_service").With("input_dataset", name)
	records, err := r.readDailyBars(ctx, name)
	if err != nil {
		return Dataset{}, errb.Wrap(err)
	}
	return Dataset{
		Name:          name,
		SchemaVersion: defaultInputSchemaVersion,
		Records:       records,
	}, nil
}

func (r DailyBarDatasetReader) readDailyBars(ctx context.Context, name string) ([]json.RawMessage, error) {
	query := daily.Query{Market: r.market}
	switch name {
	case "daily_bar", "daily_bars":
	case "etf_daily_metrics":
		query.SecurityType = provider.SecurityTypeETF
	default:
		return nil, oops.In("strategy_service").With("input_dataset", name).Errorf("unsupported input dataset: %s", name)
	}
	bars, err := r.reader.QueryDailyBars(ctx, query)
	if err != nil {
		return nil, oops.In("strategy_service").With("input_dataset", name).Wrapf(err, "query daily bar dataset")
	}
	return dailyBarsToRawMessages(bars)
}

func dailyBarsToRawMessages(bars []dailybar.Bar) ([]json.RawMessage, error) {
	records := make([]json.RawMessage, 0, len(bars))
	for _, bar := range bars {
		data, err := json.Marshal(bar)
		if err != nil {
			return nil, oops.In("strategy_service").With("symbol", bar.Symbol).Wrapf(err, "encode daily bar record")
		}
		records = append(records, data)
	}
	return records, nil
}

func executeJQ(ctx context.Context, queryText string, input any) ([]json.RawMessage, error) {
	errb := oops.In("strategy_service").With("query_hash", hashutil.SHA256([]byte(queryText)))
	query, err := gojq.Parse(queryText)
	if err != nil {
		return nil, errb.Wrapf(err, "parse jq query")
	}
	iter := query.RunWithContext(ctx, input)
	rows := make([]json.RawMessage, 0)
	for {
		value, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := value.(error); ok {
			if halt, ok := err.(*gojq.HaltError); ok && halt.Value() == nil {
				break
			}
			return nil, errb.Wrapf(err, "run jq query")
		}
		flattened, err := flattenJQValue(value)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		rows = append(rows, flattened...)
	}
	return rows, nil
}

func datasetInputValue(records []json.RawMessage) ([]any, error) {
	values := make([]any, 0, len(records))
	for _, record := range records {
		var value any
		if err := json.Unmarshal(record, &value); err != nil {
			return nil, oops.In("strategy_service").Wrapf(err, "decode input record")
		}
		values = append(values, value)
	}
	return values, nil
}

func flattenJQValue(value any) ([]json.RawMessage, error) {
	if array, ok := value.([]any); ok {
		rows := make([]json.RawMessage, 0, len(array))
		for _, item := range array {
			row, err := marshalJQValue(item)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
		return rows, nil
	}
	row, err := marshalJQValue(value)
	if err != nil {
		return nil, err
	}
	return []json.RawMessage{row}, nil
}

func marshalJQValue(value any) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, oops.In("strategy_service").Wrapf(err, "encode jq result")
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, data); err != nil {
		return nil, oops.In("strategy_service").Wrapf(err, "compact json payload")
	}
	return buffer.Bytes(), nil
}
