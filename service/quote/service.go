package quote

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	quoterole "github.com/awuzag/mwosa/providers/core/quote"
	"github.com/samber/oops"
)

type Router interface {
	RouteQuoteSnapshot(ctx context.Context, input quoterole.RouteInput) (quoterole.Snapshotter, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("quote_service").New("quote service router is nil")
	}
	return Service{router: router}, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
}

func (s Service) Get(ctx context.Context, req Request) (quoterole.SnapshotResult, error) {
	return s.fetchSnapshot(ctx, "get quote", req)
}

func (s Service) Ensure(ctx context.Context, req Request) (quoterole.SnapshotResult, error) {
	return s.fetchSnapshot(ctx, "ensure quote", req)
}

func (s Service) fetchSnapshot(ctx context.Context, operation string, req Request) (quoterole.SnapshotResult, error) {
	errb := oops.In("quote_service").With("operation", operation, "provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", req.Symbol)
	if req.Symbol == "" {
		return quoterole.SnapshotResult{}, errb.New("quote request requires symbol")
	}
	if s.router == nil {
		return quoterole.SnapshotResult{}, errb.New("quote service router is nil")
	}

	snapshotter, err := s.router.RouteQuoteSnapshot(ctx, quoterole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return quoterole.SnapshotResult{}, errb.Wrapf(err, "route quote snapshot")
	}

	result, err := snapshotter.FetchQuoteSnapshot(ctx, quoterole.SnapshotInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
	})
	if err != nil {
		return quoterole.SnapshotResult{}, errb.Wrapf(err, "fetch quote snapshot")
	}
	return result, nil
}
