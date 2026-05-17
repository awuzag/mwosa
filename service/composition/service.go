package composition

import (
	"context"
	"fmt"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	compositionrole "github.com/ev3rlit/mwosa/providers/core/composition"
	"github.com/samber/oops"
)

type Router interface {
	RouteComposition(ctx context.Context, input compositionrole.RouteInput) (compositionrole.Lister, error)
}

type Service struct {
	router     Router
	repository Repository
}

type Repository interface {
	UpsertComposition(ctx context.Context, aggregate compositionrole.Composition) (WriteResult, error)
	GetComposition(ctx context.Context, query Query) (compositionrole.Composition, error)
}

type Option func(*Service) error

func WithRepository(repository Repository) Option {
	return func(service *Service) error {
		if repository == nil {
			return oops.In("composition_service").New("composition repository is nil")
		}
		service.repository = repository
		return nil
	}
}

func NewService(router Router, options ...Option) (Service, error) {
	if router == nil {
		return Service{}, oops.In("composition_service").New("composition service router is nil")
	}
	service := Service{router: router}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&service); err != nil {
			return Service{}, err
		}
	}
	return service, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	Limit          int
}

type StoreRequest struct {
	Composition compositionrole.Composition
}

type Query struct {
	ProviderID   provider.ProviderID
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	AsOfDate     string
	ObservedAtMS int64
}

type WriteResult struct {
	RowsAffected       int `json:"rows_affected" csv:"rows_affected"`
	CompositionsStored int `json:"compositions_stored" csv:"compositions_stored"`
	MembersStored      int `json:"members_stored" csv:"members_stored"`
}

type NotFoundError struct {
	Symbol       string
	Market       provider.Market
	SecurityType provider.SecurityType
	AsOfDate     string
}

func (e *NotFoundError) Error() string {
	parts := []string{"composition not found"}
	if e.Market != "" {
		parts = append(parts, fmt.Sprintf("market=%s", e.Market))
	}
	if e.SecurityType != "" {
		parts = append(parts, fmt.Sprintf("security_type=%s", e.SecurityType))
	}
	if e.Symbol != "" {
		parts = append(parts, fmt.Sprintf("symbol=%s", e.Symbol))
	}
	if e.AsOfDate != "" {
		parts = append(parts, fmt.Sprintf("as_of_date=%s", e.AsOfDate))
	}
	return strings.Join(parts, " ")
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

func (s Service) Store(ctx context.Context, req StoreRequest) (WriteResult, error) {
	errb := oops.In("composition_service").With(
		"provider", req.Composition.Source.Provider,
		"group", req.Composition.Source.Group,
		"operation", req.Composition.Source.Operation,
		"market", req.Composition.Subject.Market,
		"security_type", req.Composition.Subject.SecurityType,
		"symbol", req.Composition.Subject.Symbol,
		"as_of_date", req.Composition.AsOfDate,
	)
	if s.repository == nil {
		return WriteResult{}, errb.New("composition repository is nil")
	}
	result, err := s.repository.UpsertComposition(ctx, req.Composition)
	if err != nil {
		return WriteResult{}, errb.Wrapf(err, "store composition")
	}
	return result, nil
}

func (s Service) Get(ctx context.Context, query Query) (compositionrole.Composition, error) {
	errb := oops.In("composition_service").With("provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "symbol", query.Symbol, "as_of_date", query.AsOfDate)
	if strings.TrimSpace(query.Symbol) == "" {
		return compositionrole.Composition{}, errb.New("get composition requires symbol")
	}
	if s.repository == nil {
		return compositionrole.Composition{}, errb.New("composition repository is nil")
	}
	aggregate, err := s.repository.GetComposition(ctx, query)
	if err != nil {
		return compositionrole.Composition{}, errb.Wrapf(err, "get composition")
	}
	return aggregate, nil
}
