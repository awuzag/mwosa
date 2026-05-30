package intraday

import (
	"context"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/intradaybar"
	"github.com/samber/oops"
)

type Router interface {
	RouteIntradayBars(ctx context.Context, input intradaybar.RouteInput) (intradaybar.Fetcher, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("intraday_service").New("intraday service router is nil")
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

func (s Service) Get(ctx context.Context, req Request) (intradaybar.FetchResult, error) {
	errb := oops.In("intraday_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", req.Symbol, "at", req.At, "limit", req.Limit)
	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		return intradaybar.FetchResult{}, errb.New("get intraday requires symbol")
	}
	if s.router == nil {
		return intradaybar.FetchResult{}, errb.New("intraday service router is nil")
	}

	fetcher, err := s.router.RouteIntradayBars(ctx, intradaybar.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         symbol,
	})
	if err != nil {
		return intradaybar.FetchResult{}, errb.Wrapf(err, "route intraday bars")
	}

	result, err := fetcher.FetchIntradayBars(ctx, intradaybar.FetchInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       symbol,
		At:           req.At,
		Limit:        req.Limit,
	})
	if err != nil {
		return intradaybar.FetchResult{}, errb.Wrapf(err, "fetch intraday bars")
	}
	return result, nil
}
