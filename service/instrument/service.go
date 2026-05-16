package instrument

import (
	"context"
	"errors"
	"fmt"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	instrumentrole "github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/samber/oops"
)

type Router interface {
	RouteInstrumentSearch(ctx context.Context, input instrumentrole.RouteInput) (instrumentrole.Searcher, error)
}

type Service struct {
	router     Router
	repository Repository
}

const inspectSearchLimit = 25

type Repository interface {
	UpsertInstruments(ctx context.Context, instruments []instrumentrole.Instrument) (WriteResult, error)
	SearchInstruments(ctx context.Context, query Query) (instrumentrole.SearchResult, error)
	InspectInstrument(ctx context.Context, query Query) (instrumentrole.Instrument, error)
}

type Option func(*Service) error

func WithRepository(repository Repository) Option {
	return func(service *Service) error {
		if repository == nil {
			return oops.In("instrument_service").New("instrument repository is nil")
		}
		service.repository = repository
		return nil
	}
}

func NewService(router Router, options ...Option) (Service, error) {
	if router == nil {
		return Service{}, oops.In("instrument_service").New("instrument service router is nil")
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

type SearchRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Query          string
	Limit          int
}

type InspectRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
}

type SyncRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	AsOf           string
}

type Query struct {
	ProviderID   provider.ProviderID
	Market       provider.Market
	SecurityType provider.SecurityType
	Query        string
	Symbol       string
	Limit        int
}

type WriteResult struct {
	RowsAffected       int `json:"rows_affected" csv:"rows_affected"`
	InstrumentsWritten int `json:"instruments_written" csv:"instruments_written"`
}

type OperationList []provider.OperationID

func (l OperationList) MarshalCSV() ([]byte, error) {
	values := make([]string, 0, len(l))
	for _, value := range l {
		values = append(values, string(value))
	}
	return []byte(strings.Join(values, ",")), nil
}

type SyncResult struct {
	Market             provider.Market       `json:"market" csv:"market"`
	SecurityType       provider.SecurityType `json:"security_type" csv:"security_type"`
	ProviderID         provider.ProviderID   `json:"provider" csv:"provider"`
	Group              provider.GroupID      `json:"provider_group" csv:"group"`
	Operations         OperationList         `json:"operations" csv:"operations"`
	AsOf               string                `json:"as_of" csv:"as_of"`
	InstrumentsFetched int                   `json:"instruments_fetched" csv:"instruments_fetched"`
	InstrumentsStored  int                   `json:"instruments_stored" csv:"instruments_stored"`
	RowsAffected       int                   `json:"rows_affected" csv:"rows_affected"`
}

type InspectResult struct {
	Instrument instrumentrole.Instrument `json:"instrument"`
	Provider   provider.Identity         `json:"provider_identity"`
	Group      provider.GroupID          `json:"provider_group"`
	Operations []provider.OperationID    `json:"operations"`
}

type NotFoundError struct {
	Query        string
	Market       provider.Market
	SecurityType provider.SecurityType
}

func (e *NotFoundError) Error() string {
	parts := []string{"instrument not found"}
	if e.Market != "" {
		parts = append(parts, fmt.Sprintf("market=%s", e.Market))
	}
	if e.SecurityType != "" {
		parts = append(parts, fmt.Sprintf("security_type=%s", e.SecurityType))
	}
	if e.Query != "" {
		parts = append(parts, fmt.Sprintf("query=%s", e.Query))
	}
	return strings.Join(parts, " ")
}

func (s Service) Search(ctx context.Context, req SearchRequest) (instrumentrole.SearchResult, error) {
	errb := oops.In("instrument_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "query", req.Query)
	if req.Query == "" {
		return instrumentrole.SearchResult{}, errb.New("search instruments requires query")
	}
	if s.repository != nil {
		result, err := s.repository.SearchInstruments(ctx, Query{
			ProviderID:   firstProvider(req.ProviderID, req.PreferProvider),
			Market:       req.Market,
			SecurityType: req.SecurityType,
			Query:        req.Query,
			Limit:        req.Limit,
		})
		if err != nil {
			return instrumentrole.SearchResult{}, errb.Wrapf(err, "search local instruments")
		}
		if len(result.Instruments) > 0 {
			return result, nil
		}
	}
	if s.router == nil {
		return instrumentrole.SearchResult{}, errb.New("instrument service router is nil")
	}

	searcher, err := s.router.RouteInstrumentSearch(ctx, instrumentrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Query,
	})
	if err != nil {
		return instrumentrole.SearchResult{}, errb.Wrapf(err, "route instrument search")
	}

	result, err := searcher.SearchInstruments(ctx, instrumentrole.SearchInput{
		Market:       req.Market,
		SecurityType: req.SecurityType,
		Query:        req.Query,
		Limit:        req.Limit,
	})
	if err != nil {
		return instrumentrole.SearchResult{}, errb.With("provider", req.ProviderID).Wrapf(err, "search instruments")
	}
	return result, nil
}

func (s Service) Inspect(ctx context.Context, req InspectRequest) (InspectResult, error) {
	symbol := strings.TrimSpace(req.Symbol)
	errb := oops.In("instrument_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", req.Market, "security_type", req.SecurityType, "symbol", symbol)
	if symbol == "" {
		return InspectResult{}, errb.New("inspect instrument requires symbol")
	}
	if s.repository != nil {
		item, err := s.repository.InspectInstrument(ctx, Query{
			ProviderID:   firstProvider(req.ProviderID, req.PreferProvider),
			Market:       req.Market,
			SecurityType: req.SecurityType,
			Symbol:       symbol,
			Limit:        1,
		})
		if err == nil {
			return InspectResult{
				Instrument: item,
				Provider:   provider.Identity{ID: item.Provider},
				Group:      item.Group,
				Operations: []provider.OperationID{item.Operation},
			}, nil
		}
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			return InspectResult{}, errb.Wrapf(err, "inspect local instrument")
		}
	}

	result, err := s.Search(ctx, SearchRequest{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Query:          symbol,
		Limit:          inspectSearchLimit,
	})
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	for _, candidate := range result.Instruments {
		if !matchesInstrumentIdentity(candidate, symbol) {
			continue
		}
		return InspectResult{
			Instrument: candidate,
			Provider:   result.Provider,
			Group:      result.Group,
			Operations: result.Operations,
		}, nil
	}

	return InspectResult{}, errb.Wrap(&NotFoundError{
		Query:        symbol,
		Market:       req.Market,
		SecurityType: req.SecurityType,
	})
}

func (s Service) Sync(ctx context.Context, req SyncRequest) (SyncResult, error) {
	market := withDefaultMarket(req.Market)
	securityType := req.SecurityType
	if securityType == "" {
		securityType = provider.SecurityTypeStock
	}
	errb := oops.In("instrument_service").With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "market", market, "security_type", securityType, "as_of", req.AsOf)
	if strings.TrimSpace(req.AsOf) == "" {
		return SyncResult{}, errb.New("sync instruments requires --as-of")
	}
	if s.repository == nil {
		return SyncResult{}, errb.New("instrument service repository is nil")
	}
	if s.router == nil {
		return SyncResult{}, errb.New("instrument service router is nil")
	}

	searcher, err := s.router.RouteInstrumentSearch(ctx, instrumentrole.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         market,
		SecurityType:   securityType,
	})
	if err != nil {
		return SyncResult{}, errb.Wrapf(err, "route instrument sync")
	}
	fetched, err := searcher.SearchInstruments(ctx, instrumentrole.SearchInput{
		Market:       market,
		SecurityType: securityType,
		AsOf:         req.AsOf,
	})
	if err != nil {
		return SyncResult{}, errb.Wrapf(err, "fetch instrument master")
	}
	writeResult, err := s.repository.UpsertInstruments(ctx, fetched.Instruments)
	if err != nil {
		return SyncResult{}, errb.With("instruments", len(fetched.Instruments)).Wrapf(err, "store instrument master")
	}
	return SyncResult{
		Market:             market,
		SecurityType:       securityType,
		ProviderID:         fetched.Provider.ID,
		Group:              fetched.Group,
		Operations:         OperationList(fetched.Operations),
		AsOf:               req.AsOf,
		InstrumentsFetched: len(fetched.Instruments),
		InstrumentsStored:  writeResult.InstrumentsWritten,
		RowsAffected:       writeResult.RowsAffected,
	}, nil
}

func matchesInstrumentIdentity(candidate instrumentrole.Instrument, symbol string) bool {
	return matchesInstrumentIdentifier(candidate.SecurityCode, symbol) ||
		matchesInstrumentIdentifier(candidate.ISIN, symbol) ||
		matchesInstrumentIdentifier(candidate.ExchangeCode, symbol)
}

func matchesInstrumentIdentifier(value string, symbol string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return strings.EqualFold(value, symbol)
}

func firstProvider(values ...provider.ProviderID) provider.ProviderID {
	for _, value := range values {
		if strings.TrimSpace(string(value)) != "" {
			return value
		}
	}
	return ""
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}
