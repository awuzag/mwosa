package handler

import (
	"context"
	"io"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/service/daily"
)

type Daily struct {
	reader    daily.ReadService
	collector daily.Service
}

func NewDaily(reader daily.ReadService, collector daily.Service) Daily {
	return Daily{
		reader:    reader,
		collector: collector,
	}
}

type GetDailyRequest struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	From         string
	To           string
	AsOf         string
}

type EnsureDailyRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	From           string
	To             string
	AsOf           string
}

type SyncDailyRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	AsOf           string
}

type BackfillDailyRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	From           string
	To             string
	Workers        int
	Progress       io.Writer
}

type DailyStorageSummaryRequest struct {
	Market       provider.Market
	SecurityType provider.SecurityType
}

type DailyCoverageRequest struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
}

func (h Daily) Get(ctx context.Context, req GetDailyRequest) (DailyBarsOutput, error) {
	result, err := h.reader.Get(ctx, daily.Request{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
		From:         req.From,
		To:           req.To,
		AsOf:         req.AsOf,
	})
	if err != nil {
		return nil, err
	}
	return DailyBarsOutput(result.Bars), nil
}

func (h Daily) StorageSummary(ctx context.Context, req DailyStorageSummaryRequest) (DailyStorageSummaryOutput, error) {
	result, err := h.reader.StorageSummary(ctx, daily.StorageSummaryRequest{
		Market:       req.Market,
		SecurityType: req.SecurityType,
	})
	if err != nil {
		return DailyStorageSummaryOutput{}, err
	}
	return DailyStorageSummaryOutput{Result: result}, nil
}

func (h Daily) Coverage(ctx context.Context, req DailyCoverageRequest) (DailyCoverageOutput, error) {
	result, err := h.reader.Coverage(ctx, daily.CoverageRequest{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
	})
	if err != nil {
		return DailyCoverageOutput{}, err
	}
	return DailyCoverageOutput{Result: result}, nil
}

func (h Daily) Ensure(ctx context.Context, req EnsureDailyRequest) (DailyBarsOutput, error) {
	result, err := h.collector.Ensure(ctx, daily.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
		From:           req.From,
		To:             req.To,
		AsOf:           req.AsOf,
	})
	if err != nil {
		return nil, err
	}
	return DailyBarsOutput(result.Bars), nil
}

func (h Daily) Sync(ctx context.Context, req SyncDailyRequest) (CollectResultOutput, error) {
	result, err := h.collector.Sync(ctx, daily.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		AsOf:           req.AsOf,
	})
	if err != nil {
		return CollectResultOutput{}, err
	}
	return CollectResultOutput{Result: result}, nil
}

func (h Daily) Backfill(ctx context.Context, req BackfillDailyRequest) (CollectResultOutput, error) {
	result, err := h.collector.Backfill(ctx, daily.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		From:           req.From,
		To:             req.To,
		Workers:        req.Workers,
		Progress:       req.Progress,
	})
	if err != nil {
		return CollectResultOutput{}, err
	}
	return CollectResultOutput{Result: result}, nil
}
