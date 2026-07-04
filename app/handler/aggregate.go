package handler

import (
	"context"
	"os"

	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
)

type Aggregate struct {
	service aggregateservice.Service
}

func NewAggregate(service aggregateservice.Service) Aggregate {
	return Aggregate{service: service}
}

type UpsertAggregateRequest struct {
	Name string
	Path string
}

type ValidateAggregateRequest struct {
	Path string
	View string
}

type ListAggregatesRequest struct{}

type InspectAggregateRequest struct {
	Name     string
	Version  string
	SpecHash string
	View     string
}

type PlanAggregateRequest struct {
	Ref      string
	Path     string
	Version  string
	SpecHash string
	Params   []string
	View     string
}

type RunAggregateRequest struct {
	Name     string
	Alias    string
	Version  string
	SpecHash string
	Params   []string
}

type AggregateHistoryRequest struct {
	Name   string
	Status aggregateservice.RunStatus
	Limit  int
}

type InspectAggregateRunRequest struct {
	Ref   string
	Limit int
	View  string
}

type DeleteAggregateRequest struct {
	Name string
}

func (h Aggregate) Upsert(ctx context.Context, req UpsertAggregateRequest) (AggregateDetailOutput, error) {
	data, err := os.ReadFile(req.Path)
	if err != nil {
		return AggregateDetailOutput{}, err
	}
	spec, err := aggregateservice.LoadSpecBytes(ctx, data)
	if err != nil {
		return AggregateDetailOutput{}, err
	}
	detail, err := h.service.Upsert(ctx, aggregateservice.UpsertRequest{
		Name:     req.Name,
		Spec:     spec,
		YAMLText: string(data),
	})
	if err != nil {
		return AggregateDetailOutput{}, err
	}
	return AggregateDetailOutput{Detail: detail}, nil
}

func (h Aggregate) Validate(ctx context.Context, req ValidateAggregateRequest) (AggregateValidationOutput, error) {
	spec, err := aggregateservice.LoadSpecFile(ctx, req.Path)
	if err != nil {
		return AggregateValidationOutput{}, err
	}
	if err := aggregateservice.ValidateSpec(spec); err != nil {
		return AggregateValidationOutput{}, err
	}
	return AggregateValidationOutput{Spec: spec, Valid: true, View: req.View}, nil
}

func (h Aggregate) List(ctx context.Context, _ ListAggregatesRequest) (AggregateListOutput, error) {
	details, err := h.service.List(ctx)
	if err != nil {
		return nil, err
	}
	return AggregateListOutput(details), nil
}

func (h Aggregate) Inspect(ctx context.Context, req InspectAggregateRequest) (AggregateDetailOutput, error) {
	detail, err := h.service.Inspect(ctx, req.Name, aggregateservice.VersionRef{Version: req.Version, SpecHash: req.SpecHash})
	if err != nil {
		return AggregateDetailOutput{}, err
	}
	return AggregateDetailOutput{Detail: detail, View: req.View}, nil
}

func (h Aggregate) Plan(ctx context.Context, req PlanAggregateRequest) (AggregatePlanOutput, error) {
	planReq := aggregateservice.PlanRequest{
		Name:     req.Ref,
		Version:  req.Version,
		SpecHash: req.SpecHash,
		Params:   req.Params,
	}
	if req.Path != "" {
		spec, err := aggregateservice.LoadSpecFile(ctx, req.Path)
		if err != nil {
			return AggregatePlanOutput{}, err
		}
		planReq.Spec = &spec
	}
	plan, err := h.service.Plan(ctx, planReq)
	if err != nil {
		return AggregatePlanOutput{}, err
	}
	return AggregatePlanOutput{Plan: plan, View: req.View}, nil
}

func (h Aggregate) Run(ctx context.Context, req RunAggregateRequest) (AggregateRunOutput, error) {
	detail, rows, err := h.service.Run(ctx, aggregateservice.RunRequest{
		Name:     req.Name,
		Alias:    req.Alias,
		Version:  req.Version,
		SpecHash: req.SpecHash,
		Params:   req.Params,
	})
	if err != nil {
		return AggregateRunOutput{Detail: detail, Rows: rows}, err
	}
	return AggregateRunOutput{Detail: detail, Rows: rows}, nil
}

func (h Aggregate) History(ctx context.Context, req AggregateHistoryRequest) (AggregateRunHistoryOutput, error) {
	runs, err := h.service.History(ctx, aggregateservice.RunHistoryFilter{Name: req.Name, Status: req.Status, Limit: req.Limit})
	if err != nil {
		return nil, err
	}
	return AggregateRunHistoryOutput(runs), nil
}

func (h Aggregate) InspectRun(ctx context.Context, req InspectAggregateRunRequest) (AggregateRunDetailOutput, error) {
	detail, err := h.service.InspectRun(ctx, req.Ref, req.Limit)
	if err != nil {
		return AggregateRunDetailOutput{}, err
	}
	return AggregateRunDetailOutput{Detail: detail, View: req.View}, nil
}

func (h Aggregate) Delete(ctx context.Context, req DeleteAggregateRequest) (DeleteAggregateResult, error) {
	if err := h.service.Delete(ctx, req.Name); err != nil {
		return DeleteAggregateResult{}, err
	}
	return DeleteAggregateResult{Name: req.Name, Deleted: true}, nil
}
