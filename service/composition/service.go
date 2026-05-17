package composition

import (
	"context"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	compositionrole "github.com/ev3rlit/mwosa/providers/core/composition"
	"github.com/samber/oops"
)

type Router interface {
	RouteComposition(ctx context.Context, input compositionrole.RouteInput) (compositionrole.Lister, error)
}

type Service struct {
	router Router
}

func NewService(router Router) (Service, error) {
	if router == nil {
		return Service{}, oops.In("composition_service").New("composition service router is nil")
	}
	return Service{router: router}, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	Limit          int
}

func (s Service) List(ctx context.Context, req Request) (compositionrole.ListResult, error) {
	errb := oops.In("composition_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", req.Symbol, "limit", req.Limit)
	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		return compositionrole.ListResult{}, errb.New("list constituents requires symbol")
	}
	if s.router == nil {
		return compositionrole.ListResult{}, errb.New("composition service router is nil")
	}

	lister, err := s.router.RouteComposition(ctx, compositionrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         symbol,
	})
	if err != nil {
		return compositionrole.ListResult{}, errb.Wrapf(err, "route composition")
	}

	result, err := lister.ListConstituents(ctx, compositionrole.ListInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Symbol:       symbol,
		Limit:        req.Limit,
	})
	if err != nil {
		return compositionrole.ListResult{}, errb.Wrapf(err, "list constituents")
	}
	return result, nil
}
