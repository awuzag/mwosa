package orderbook

import (
	"context"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	orderbookrole "github.com/ev3rlit/mwosa/providers/core/orderbook"
	"github.com/samber/oops"
)

type Router interface {
	RouteOrderbookSnapshot(ctx context.Context, input orderbookrole.RouteInput) (orderbookrole.Snapshotter, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("orderbook_service").New("orderbook service router is nil")
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

func (s Service) Get(ctx context.Context, req Request) (orderbookrole.SnapshotResult, error) {
	errb := oops.In("orderbook_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", req.Symbol)
	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		return orderbookrole.SnapshotResult{}, errb.New("get orderbook requires symbol")
	}
	if s.router == nil {
		return orderbookrole.SnapshotResult{}, errb.New("orderbook service router is nil")
	}

	snapshotter, err := s.router.RouteOrderbookSnapshot(ctx, orderbookrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         symbol,
	})
	if err != nil {
		return orderbookrole.SnapshotResult{}, errb.Wrapf(err, "route orderbook snapshot")
	}

	result, err := snapshotter.FetchOrderbookSnapshot(ctx, orderbookrole.SnapshotInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       symbol,
	})
	if err != nil {
		return orderbookrole.SnapshotResult{}, errb.Wrapf(err, "fetch orderbook snapshot")
	}
	return result, nil
}
