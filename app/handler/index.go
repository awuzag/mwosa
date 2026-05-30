package handler

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	indexservice "github.com/awuzag/mwosa/service/index"
)

type Index struct {
	reader    indexservice.ReadService
	collector indexservice.Service
}

func NewIndex(reader indexservice.ReadService, collector indexservice.Service) Index {
	return Index{reader: reader, collector: collector}
}

type GetIndexRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	IndexCode      string
	From           string
	To             string
	AsOf           string
}

type SyncIndexRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	IndexCode      string
	AsOf           string
}

func (h Index) Get(ctx context.Context, req GetIndexRequest) (IndexBarsOutput, error) {
	result, err := h.collector.Get(ctx, indexservice.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		IndexCode:      req.IndexCode,
		From:           req.From,
		To:             req.To,
		AsOf:           req.AsOf,
	})
	if err != nil {
		return nil, err
	}
	return IndexBarsOutput(result.Bars), nil
}

func (h Index) GetStored(ctx context.Context, req GetIndexRequest) (IndexBarsOutput, error) {
	result, err := h.reader.Get(ctx, indexservice.Request{
		Market:    req.Market,
		IndexCode: req.IndexCode,
		From:      req.From,
		To:        req.To,
		AsOf:      req.AsOf,
	})
	if err != nil {
		return nil, err
	}
	return IndexBarsOutput(result.Bars), nil
}

func (h Index) Sync(ctx context.Context, req SyncIndexRequest) (IndexCollectResultOutput, error) {
	result, err := h.collector.Sync(ctx, indexservice.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		IndexCode:      req.IndexCode,
		AsOf:           req.AsOf,
	})
	if err != nil {
		return IndexCollectResultOutput{}, err
	}
	return IndexCollectResultOutput{Result: result}, nil
}
