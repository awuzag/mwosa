package aggregate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/awuzag/mwosa/packages/hashutil"
	"github.com/awuzag/mwosa/packages/idgen"
	"github.com/samber/oops"
)

const runPersistenceTimeout = 10 * time.Second

type Service struct {
	repo     Repository
	executor Executor
	now      func() time.Time
}

type Option func(*Service) error

func WithExecutor(executor Executor) Option {
	return func(service *Service) error {
		service.executor = executor
		return nil
	}
}

func NewService(repo Repository, opts ...Option) (Service, error) {
	if repo == nil {
		return Service{}, oops.In("aggregate_service").New("aggregate repository is nil")
	}
	service := Service{repo: repo, now: time.Now}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&service); err != nil {
			return Service{}, err
		}
	}
	return service, nil
}

type UpsertRequest struct {
	Name     string
	Spec     Spec
	YAMLText string
	Note     string
}

type RunRequest struct {
	Name     string
	Alias    string
	Version  string
	SpecHash string
	Params   []string
}

type PlanRequest struct {
	Name     string
	Spec     *Spec
	Version  string
	SpecHash string
	Params   []string
}

type Plan struct {
	Name     string         `json:"name"`
	Params   map[string]any `json:"params"`
	Stages   []StageSpec    `json:"stages"`
	Output   OutputSpec     `json:"output"`
	SpecHash string         `json:"spec_hash,omitempty"`
}

func (s Service) Upsert(ctx context.Context, req UpsertRequest) (Detail, error) {
	errb := oops.In("aggregate_service").With("name", req.Name)
	spec := req.Spec
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = req.Name
	}
	if spec.Name != req.Name {
		return Detail{}, errb.With("spec_name", spec.Name).Errorf("aggregate name mismatch: cli=%s yaml=%s", req.Name, spec.Name)
	}
	if err := ValidateSpec(spec); err != nil {
		return Detail{}, errb.Wrap(err)
	}
	specJSON, specHash, err := canonicalSpecPayload(spec)
	if err != nil {
		return Detail{}, errb.Wrap(err)
	}
	existing, err := s.repo.GetAggregate(ctx, req.Name)
	if err != nil {
		if !strings.Contains(err.Error(), "aggregate not found") {
			return Detail{}, errb.Wrapf(err, "load aggregate before upsert")
		}
		return s.createFromSpec(ctx, spec, req.YAMLText, req.Note, specJSON, specHash)
	}
	if existing.ActiveVersion.SpecHash == specHash {
		return existing, nil
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return Detail{}, errb.Wrapf(err, "generate aggregate version id")
	}
	now := s.now()
	version := Version{
		ID:          versionID,
		AggregateID: existing.Aggregate.ID,
		Version:     existing.ActiveVersion.Version + 1,
		YAMLText:    req.YAMLText,
		SpecJSON:    specJSON,
		SpecHash:    specHash,
		CreatedAt:   now,
		Note:        req.Note,
	}
	return s.repo.AddAggregateVersion(ctx, req.Name, version, now)
}

func (s Service) createFromSpec(ctx context.Context, spec Spec, yamlText string, note string, specJSON json.RawMessage, specHash string) (Detail, error) {
	errb := oops.In("aggregate_service").With("name", spec.Name)
	aggregateID, err := idgen.NewUUIDV7()
	if err != nil {
		return Detail{}, errb.Wrapf(err, "generate aggregate id")
	}
	versionID, err := idgen.NewUUIDV7()
	if err != nil {
		return Detail{}, errb.Wrapf(err, "generate aggregate version id")
	}
	now := s.now()
	aggregate := Aggregate{
		ID:              aggregateID,
		Name:            spec.Name,
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := Version{
		ID:          versionID,
		AggregateID: aggregateID,
		Version:     1,
		YAMLText:    yamlText,
		SpecJSON:    specJSON,
		SpecHash:    specHash,
		CreatedAt:   now,
		Note:        note,
	}
	return s.repo.CreateAggregateWithVersion(ctx, aggregate, version)
}

func (s Service) List(ctx context.Context) ([]Detail, error) {
	return s.repo.ListAggregates(ctx)
}

func (s Service) Inspect(ctx context.Context, name string, ref VersionRef) (Detail, error) {
	if strings.TrimSpace(name) == "" {
		return Detail{}, oops.In("aggregate_service").New("inspect aggregate requires name")
	}
	if ref.Version != "" || ref.SpecHash != "" {
		return s.repo.GetAggregateVersion(ctx, name, ref)
	}
	return s.repo.GetAggregate(ctx, name)
}

func (s Service) Delete(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return oops.In("aggregate_service").New("delete aggregate requires name")
	}
	return s.repo.ArchiveAggregate(ctx, name, s.now())
}

func (s Service) Plan(ctx context.Context, req PlanRequest) (Plan, error) {
	spec, specHash, err := s.planSpec(ctx, req)
	if err != nil {
		return Plan{}, err
	}
	params, err := ApplyParams(spec, req.Params)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		Name:     spec.Name,
		Params:   params,
		Stages:   spec.Pipeline,
		Output:   spec.Output,
		SpecHash: specHash,
	}, nil
}

func (s Service) History(ctx context.Context, filter RunHistoryFilter) ([]Run, error) {
	switch filter.Status {
	case "", RunSucceeded, RunFailed, RunCancelled:
	default:
		return nil, oops.In("aggregate_service").With("status", filter.Status).Errorf("unsupported aggregate run status: %s", filter.Status)
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return s.repo.ListRuns(ctx, filter)
}

func (s Service) InspectRun(ctx context.Context, ref string, limit int) (RunDetail, error) {
	if strings.TrimSpace(ref) == "" {
		return RunDetail{}, oops.In("aggregate_service").New("inspect aggregate run requires id or alias")
	}
	return s.repo.GetRun(ctx, ref, limit)
}

func (s Service) Run(ctx context.Context, req RunRequest) (RunDetail, OutputRows, error) {
	errb := oops.In("aggregate_service").With("name", req.Name, "alias", req.Alias)
	if s.executor == nil {
		return RunDetail{}, OutputRows{}, errb.New("aggregate executor is not configured")
	}
	detail, err := s.repo.GetAggregateVersion(ctx, req.Name, VersionRef{Version: req.Version, SpecHash: req.SpecHash})
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "load aggregate")
	}
	spec, err := specFromVersion(detail.ActiveVersion)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrap(err)
	}
	params, err := ApplyParams(spec, req.Params)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrap(err)
	}
	alias := strings.TrimSpace(req.Alias)
	if alias != "" {
		exists, err := s.repo.HasRunAlias(ctx, alias)
		if err != nil {
			return RunDetail{}, OutputRows{}, errb.Wrapf(err, "check aggregate run alias")
		}
		if exists {
			return RunDetail{}, OutputRows{}, errb.Errorf("aggregate run alias already exists: %s", alias)
		}
	}
	started := s.now()
	runID, err := idgen.NewUUIDV7()
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "generate aggregate run id")
	}
	result, err := s.executor.Execute(ctx, spec, params, runID)
	if err != nil {
		failed, saveErr := s.recordFailedRun(ctx, detail, alias, started, runID, params, spec, result.Stages, err)
		if saveErr != nil {
			return failed, OutputRows{}, saveErr
		}
		return failed, OutputRows{}, err
	}
	return s.recordSucceededRun(ctx, detail, alias, started, runID, params, spec, result)
}

func (s Service) recordSucceededRun(ctx context.Context, detail Detail, alias string, started time.Time, runID string, params map[string]any, spec Spec, result ExecutionResult) (RunDetail, OutputRows, error) {
	errb := oops.In("aggregate_service").With("aggregate_id", detail.Aggregate.ID, "alias", alias)
	output, err := FormatOutputRows(spec.Output, result.Rows)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrap(err)
	}
	resultJSON, err := json.Marshal(result.Rows)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "encode aggregate result rows")
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "encode aggregate params")
	}
	stagesJSON, err := json.Marshal(result.Stages)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "encode aggregate stages")
	}
	pipelineJSON, err := json.Marshal(spec.Pipeline)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "encode aggregate pipeline")
	}
	summaryJSON, err := summarizeResultRows(result.Rows)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrap(err)
	}
	finished := s.now()
	run := Run{
		ID:                 runID,
		Alias:              strings.TrimSpace(alias),
		AggregateID:        detail.Aggregate.ID,
		AggregateVersionID: detail.ActiveVersion.ID,
		AggregateName:      detail.Aggregate.Name,
		Version:            detail.ActiveVersion.Version,
		SpecHash:           detail.ActiveVersion.SpecHash,
		ParamsJSON:         paramsJSON,
		StagesJSON:         stagesJSON,
		PipelineJSON:       pipelineJSON,
		StartedAt:          started,
		FinishedAt:         &finished,
		Status:             RunSucceeded,
		ResultCount:        len(result.Rows),
		ResultHash:         hashutil.SHA256(resultJSON),
		ResultSizeBytes:    int64(len(resultJSON)),
		SummaryJSON:        summaryJSON,
	}
	items, err := runItemsFromRows(run.ID, result.Rows)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrap(err)
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), runPersistenceTimeout)
	defer cancel()
	saved, err := s.repo.CreateRun(persistCtx, run, items)
	if err != nil {
		return RunDetail{}, OutputRows{}, errb.Wrapf(err, "save aggregate run")
	}
	return saved, output, nil
}

func (s Service) recordFailedRun(ctx context.Context, detail Detail, alias string, started time.Time, runID string, params map[string]any, spec Spec, stages []StageSummary, runErr error) (RunDetail, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return RunDetail{}, oops.Join(runErr, oops.In("aggregate_service").Wrapf(err, "encode failed aggregate params"))
	}
	stagesJSON, err := json.Marshal(stages)
	if err != nil {
		return RunDetail{}, oops.Join(runErr, oops.In("aggregate_service").Wrapf(err, "encode failed aggregate stages"))
	}
	pipelineJSON, err := json.Marshal(spec.Pipeline)
	if err != nil {
		return RunDetail{}, oops.Join(runErr, oops.In("aggregate_service").Wrapf(err, "encode failed aggregate pipeline"))
	}
	finished := s.now()
	status := RunFailed
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		status = RunCancelled
	}
	run := Run{
		ID:                 runID,
		Alias:              strings.TrimSpace(alias),
		AggregateID:        detail.Aggregate.ID,
		AggregateVersionID: detail.ActiveVersion.ID,
		AggregateName:      detail.Aggregate.Name,
		Version:            detail.ActiveVersion.Version,
		SpecHash:           detail.ActiveVersion.SpecHash,
		ParamsJSON:         paramsJSON,
		StagesJSON:         stagesJSON,
		PipelineJSON:       pipelineJSON,
		StartedAt:          started,
		FinishedAt:         &finished,
		Status:             status,
		ErrorMessage:       runErr.Error(),
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), runPersistenceTimeout)
	defer cancel()
	saved, saveErr := s.repo.CreateRun(persistCtx, run, nil)
	if saveErr != nil {
		return saved, oops.Join(runErr, oops.In("aggregate_service").Wrapf(saveErr, "save failed aggregate run"))
	}
	return saved, nil
}

func (s Service) planSpec(ctx context.Context, req PlanRequest) (Spec, string, error) {
	if req.Spec != nil {
		if err := ValidateSpec(*req.Spec); err != nil {
			return Spec{}, "", err
		}
		_, specHash, err := canonicalSpecPayload(*req.Spec)
		return *req.Spec, specHash, err
	}
	detail, err := s.repo.GetAggregateVersion(ctx, req.Name, VersionRef{Version: req.Version, SpecHash: req.SpecHash})
	if err != nil {
		return Spec{}, "", err
	}
	spec, err := specFromVersion(detail.ActiveVersion)
	if err != nil {
		return Spec{}, "", err
	}
	return spec, detail.ActiveVersion.SpecHash, nil
}

func specFromVersion(version Version) (Spec, error) {
	var spec Spec
	if err := json.Unmarshal(version.SpecJSON, &spec); err != nil {
		return Spec{}, oops.In("aggregate_service").With("version_id", version.ID).Wrapf(err, "decode aggregate spec")
	}
	return spec, ValidateSpec(spec)
}

func canonicalSpecPayload(spec Spec) (json.RawMessage, string, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, "", oops.In("aggregate_service").With("name", spec.Name).Wrapf(err, "encode aggregate spec")
	}
	return data, hashutil.SHA256(data), nil
}

func runItemsFromRows(runID string, rows []json.RawMessage) ([]RunItem, error) {
	items := make([]RunItem, 0, len(rows))
	for i, row := range rows {
		itemID, err := idgen.NewUUIDV7()
		if err != nil {
			return nil, oops.In("aggregate_service").With("ordinal", i).Wrapf(err, "generate aggregate run item id")
		}
		items = append(items, RunItem{
			ID:          itemID,
			RunID:       runID,
			Ordinal:     i,
			PayloadJSON: row,
		})
	}
	return items, nil
}

func summarizeResultRows(rows []json.RawMessage) (json.RawMessage, error) {
	summary := map[string]any{"count": len(rows)}
	data, err := json.Marshal(summary)
	if err != nil {
		return nil, oops.In("aggregate_service").Wrapf(err, "encode aggregate result summary")
	}
	return data, nil
}
