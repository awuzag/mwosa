package handler

import (
	"context"

	strategyservice "github.com/awuzag/mwosa/service/strategy"
	universeservice "github.com/awuzag/mwosa/service/universe"
)

type Strategy struct {
	service  strategyservice.Service
	universe universeservice.Runner
}

func NewStrategy(service strategyservice.Service, universe ...universeservice.Runner) Strategy {
	handler := Strategy{service: service}
	if len(universe) > 0 {
		handler.universe = universe[0]
	}
	return handler
}

type CreateStrategyRequest struct {
	Name         string
	Engine       strategyservice.Engine
	InputDataset string
	QueryText    string
}

type ListStrategiesRequest struct{}

type UpdateStrategyRequest struct {
	Name      string
	QueryText string
}

type UpsertScreenStrategyRequest struct {
	Name string
	Path string
}

type DeleteStrategyRequest struct {
	Name string
}

type ScreenJQRequest struct {
	InputDataset string
	QueryText    string
}

type ScreenStrategyRequest struct {
	Name     string
	Alias    string
	Version  string
	SpecHash string
}

type CompareScreenStrategiesRequest struct {
	Names []string
	AsOf  string
	TopN  int
}

type ScreenPipelineRequest struct {
	Path string
}

type ScreenHistoryRequest struct {
	Limit int
}

type InspectStrategyRequest struct {
	Name string
}

type InspectScreenRequest struct {
	Ref string
}

type InspectScreenPipelineRequest struct {
	Path string
}

type InspectMarketRegimeRequest struct {
	Path string
	AsOf string
}

type InspectStrategySetRequest struct {
	Path string
	AsOf string
}

func (h Strategy) Create(ctx context.Context, req CreateStrategyRequest) (StrategyDetailOutput, error) {
	detail, err := h.service.Create(ctx, strategyservice.CreateStrategyRequest{
		Name:         req.Name,
		Engine:       req.Engine,
		InputDataset: req.InputDataset,
		QueryText:    req.QueryText,
	})
	if err != nil {
		return StrategyDetailOutput{}, err
	}
	return StrategyDetailOutput{Detail: detail}, nil
}

func (h Strategy) List(ctx context.Context, _ ListStrategiesRequest) (StrategyListOutput, error) {
	details, err := h.service.List(ctx)
	if err != nil {
		return nil, err
	}
	return StrategyListOutput(details), nil
}

func (h Strategy) Update(ctx context.Context, req UpdateStrategyRequest) (StrategyDetailOutput, error) {
	detail, err := h.service.Update(ctx, strategyservice.UpdateStrategyRequest{
		Name:      req.Name,
		QueryText: req.QueryText,
	})
	if err != nil {
		return StrategyDetailOutput{}, err
	}
	return StrategyDetailOutput{Detail: detail}, nil
}

func (h Strategy) UpsertScreenStrategy(ctx context.Context, req UpsertScreenStrategyRequest) (StrategyDetailOutput, error) {
	spec, err := strategyservice.LoadScreenStrategyFile(ctx, req.Path)
	if err != nil {
		return StrategyDetailOutput{}, err
	}
	detail, err := h.service.Upsert(ctx, strategyservice.UpsertStrategyRequest{
		Name: req.Name,
		Spec: spec,
	})
	if err != nil {
		return StrategyDetailOutput{}, err
	}
	return StrategyDetailOutput{Detail: detail}, nil
}

func (h Strategy) Delete(ctx context.Context, req DeleteStrategyRequest) (DeleteStrategyResult, error) {
	if err := h.service.Delete(ctx, req.Name); err != nil {
		return DeleteStrategyResult{}, err
	}
	return DeleteStrategyResult{Name: req.Name, Deleted: true}, nil
}

func (h Strategy) ScreenJQ(ctx context.Context, req ScreenJQRequest) (ScreenResultOutput, error) {
	result, err := h.service.ScreenJQ(ctx, strategyservice.ScreenJQRequest{
		InputDataset: req.InputDataset,
		QueryText:    req.QueryText,
	})
	if err != nil {
		return ScreenResultOutput{}, err
	}
	return ScreenResultOutput{Result: result}, nil
}

func (h Strategy) Screen(ctx context.Context, req ScreenStrategyRequest) (ScreenRunDetailOutput, error) {
	detail, err := h.service.Screen(ctx, strategyservice.ScreenStrategyRequest{
		Name:     req.Name,
		Alias:    req.Alias,
		Version:  req.Version,
		SpecHash: req.SpecHash,
	})
	if err != nil {
		return ScreenRunDetailOutput{}, err
	}
	return ScreenRunDetailOutput{Detail: detail}, nil
}

func (h Strategy) CompareScreenStrategies(ctx context.Context, req CompareScreenStrategiesRequest) (ScreenStrategyComparisonOutput, error) {
	result, err := h.service.CompareScreenStrategies(ctx, strategyservice.CompareScreenStrategiesRequest{
		Names: req.Names,
		AsOf:  req.AsOf,
		TopN:  req.TopN,
	})
	if err != nil {
		return ScreenStrategyComparisonOutput{}, err
	}
	return ScreenStrategyComparisonOutput{Result: result}, nil
}

func (h Strategy) ScreenPipeline(ctx context.Context, req ScreenPipelineRequest) (ScreenPipelineOutput, error) {
	result, err := h.universe.InspectScreenPipeline(ctx, req.Path)
	if err != nil {
		return ScreenPipelineOutput{}, err
	}
	return ScreenPipelineOutput{Result: result}, nil
}

func (h Strategy) History(ctx context.Context, req ScreenHistoryRequest) (ScreenRunHistoryOutput, error) {
	runs, err := h.service.History(ctx, req.Limit)
	if err != nil {
		return nil, err
	}
	return ScreenRunHistoryOutput(runs), nil
}

func (h Strategy) Inspect(ctx context.Context, req InspectStrategyRequest) (StrategyDetailOutput, error) {
	detail, err := h.service.Inspect(ctx, req.Name)
	if err != nil {
		return StrategyDetailOutput{}, err
	}
	return StrategyDetailOutput{Detail: detail}, nil
}

func (h Strategy) InspectScreen(ctx context.Context, req InspectScreenRequest) (ScreenRunDetailOutput, error) {
	detail, err := h.service.InspectScreen(ctx, req.Ref)
	if err != nil {
		return ScreenRunDetailOutput{}, err
	}
	return ScreenRunDetailOutput{Detail: detail}, nil
}

func (h Strategy) InspectScreenPipeline(ctx context.Context, req InspectScreenPipelineRequest) (ScreenPipelineOutput, error) {
	result, err := h.universe.InspectScreenPipeline(ctx, req.Path)
	if err != nil {
		return ScreenPipelineOutput{}, err
	}
	return ScreenPipelineOutput{Result: result}, nil
}

func (h Strategy) InspectMarketRegime(ctx context.Context, req InspectMarketRegimeRequest) (MarketRegimeOutput, error) {
	result, err := h.universe.InspectMarketRegime(ctx, req.Path, req.AsOf)
	if err != nil {
		return MarketRegimeOutput{}, err
	}
	return MarketRegimeOutput{Result: result}, nil
}

func (h Strategy) InspectStrategySet(ctx context.Context, req InspectStrategySetRequest) (StrategySetOutput, error) {
	result, err := h.universe.InspectStrategySet(ctx, req.Path, req.AsOf)
	if err != nil {
		return StrategySetOutput{}, err
	}
	return StrategySetOutput{Result: result}, nil
}
