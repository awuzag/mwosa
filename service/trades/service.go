package trades

import (
	"context"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	tradesrole "github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/samber/oops"
)

type Router interface {
	RouteMarketTrades(ctx context.Context, input tradesrole.RouteInput) (tradesrole.Lister, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("trades_service").New("trades service router is nil")
	}
	return Service{router: router}, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	At             string
	Limit          int
}

func (s Service) List(ctx context.Context, req Request) (tradesrole.ListResult, error) {
	errb := oops.In("trades_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", req.Symbol, "at", req.At, "limit", req.Limit)
	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		return tradesrole.ListResult{}, errb.New("list trades requires symbol")
	}
	if s.router == nil {
		return tradesrole.ListResult{}, errb.New("trades service router is nil")
	}

	lister, err := s.router.RouteMarketTrades(ctx, tradesrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         symbol,
	})
	if err != nil {
		return tradesrole.ListResult{}, errb.Wrapf(err, "route market trades")
	}

	result, err := lister.ListMarketTrades(ctx, tradesrole.ListInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       symbol,
		At:           req.At,
		Limit:        req.Limit,
	})
	if err != nil {
		return tradesrole.ListResult{}, errb.Wrapf(err, "list market trades")
	}
	return result, nil
}
