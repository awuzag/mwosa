package instrument

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	instrumentrole "github.com/awuzag/mwosa/providers/core/instrument"
)

func TestNewServiceRequiresRouter(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatal("NewService error = nil, want router error")
	}
	if !strings.Contains(err.Error(), "router") {
		t.Fatalf("error = %q, want router context", err.Error())
	}
}

func TestSearchRoutesAndCallsInstrumentSearcher(t *testing.T) {
	var gotSearch instrumentrole.SearchInput
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(_ context.Context, input instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		gotSearch = input
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{
				{
					Provider:     provider.ProviderID("fake"),
					Market:       provider.MarketKRX,
					SecurityType: provider.SecurityTypeETF,
					SecurityCode: "069500",
					Name:         "KODEX 200",
				},
			},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
			Group:    provider.GroupID("fakeGroup"),
		}, nil
	})
	router := &fakeInstrumentRouter{searcher: searcher}
	service, err := NewService(router)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Search(context.Background(), SearchRequest{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Query:        "069500",
		Limit:        5,
	})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}

	if router.gotRoute.ProviderID != provider.ProviderID("fake") || router.gotRoute.Symbol != "069500" {
		t.Fatalf("route input = %+v, want provider and symbol", router.gotRoute)
	}
	if gotSearch.Query != "069500" || gotSearch.Limit != 5 {
		t.Fatalf("search input = %+v, want query and limit", gotSearch)
	}
	if len(result.Instruments) != 1 {
		t.Fatalf("instruments len = %d, want 1", len(result.Instruments))
	}
}

func TestSearchUsesLocalRepositoryBeforeProvider(t *testing.T) {
	repository := &fakeInstrumentRepository{
		searchResult: instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{{
				Provider:     provider.ProviderKRX,
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				SecurityCode: "005930",
				Name:         "삼성전자",
			}},
			Provider: provider.Identity{ID: provider.ProviderKRX},
			Group:    provider.GroupKRXStockInstrument,
		},
	}
	service, err := NewService(&fakeInstrumentRouter{}, WithRepository(repository))
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Search(context.Background(), SearchRequest{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Query:        "삼성",
		Limit:        5,
	})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(result.Instruments) != 1 || result.Instruments[0].SecurityCode != "005930" {
		t.Fatalf("result = %+v, want local instrument", result)
	}
	if repository.searchQuery.Query != "삼성" || repository.searchQuery.Limit != 5 {
		t.Fatalf("local query = %+v", repository.searchQuery)
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	service, err := NewService(&fakeInstrumentRouter{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Search(context.Background(), SearchRequest{})
	if err == nil {
		t.Fatal("Search error = nil, want query error")
	}
	if !strings.Contains(err.Error(), "requires query") {
		t.Fatalf("error = %q, want query context", err.Error())
	}
}

func TestInspectReturnsExactMatchedInstrument(t *testing.T) {
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(_ context.Context, input instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		if input.Limit != inspectSearchLimit {
			t.Fatalf("inspect search limit = %d, want %d", input.Limit, inspectSearchLimit)
		}
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{
				{SecurityCode: "069501", Name: "KODEX 200 Similar"},
				{SecurityCode: "069500", Name: "KODEX 200"},
			},
			Provider:   provider.Identity{ID: provider.ProviderID("fake")},
			Group:      provider.GroupID("fakeGroup"),
			Operations: []provider.OperationID{provider.OperationID("fakeOperation")},
		}, nil
	})
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Inspect(context.Background(), InspectRequest{Symbol: "069500"})
	if err != nil {
		t.Fatalf("Inspect error = %v", err)
	}
	if result.Instrument.SecurityCode != "069500" || result.Provider.ID != provider.ProviderID("fake") {
		t.Fatalf("inspect result = %+v", result)
	}
}

func TestInspectMatchesISIN(t *testing.T) {
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(_ context.Context, input instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{
				{SecurityCode: "069500", ISIN: "KR7069500007", Name: "KODEX 200"},
			},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
		}, nil
	})
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Inspect(context.Background(), InspectRequest{Symbol: "kr7069500007"})
	if err != nil {
		t.Fatalf("Inspect error = %v", err)
	}
	if result.Instrument.ISIN != "KR7069500007" {
		t.Fatalf("inspect result = %+v, want ISIN match", result)
	}
}

func TestInspectUsesLocalRepositoryBeforeProvider(t *testing.T) {
	repository := &fakeInstrumentRepository{
		inspectResult: instrumentrole.Instrument{
			Provider:     provider.ProviderKRX,
			Group:        provider.GroupKRXStockInstrument,
			Operation:    provider.OperationStockIssueBaseInfo,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			SecurityCode: "005930",
			Name:         "삼성전자",
		},
	}
	service, err := NewService(&fakeInstrumentRouter{}, WithRepository(repository))
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Inspect(context.Background(), InspectRequest{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})
	if err != nil {
		t.Fatalf("Inspect error = %v", err)
	}
	if result.Instrument.SecurityCode != "005930" || result.Group != provider.GroupKRXStockInstrument {
		t.Fatalf("result = %+v, want local inspect result", result)
	}
}

func TestSyncFetchesAndStoresInstrumentMaster(t *testing.T) {
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(_ context.Context, input instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		if input.Query != "" || input.AsOf != "2026-05-13" {
			t.Fatalf("search input = %+v, want full master sync with as-of", input)
		}
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{{
				Provider:     provider.ProviderKRX,
				Group:        provider.GroupKRXStockInstrument,
				Operation:    provider.OperationStockIssueBaseInfo,
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				SecurityCode: "005930",
				Name:         "삼성전자",
				Extensions:   map[string]string{"issueEnglishName": "Samsung Electronics"},
			}},
			Provider:   provider.Identity{ID: provider.ProviderKRX},
			Group:      provider.GroupKRXStockInstrument,
			Operations: []provider.OperationID{provider.OperationStockIssueBaseInfo},
		}, nil
	})
	repository := &fakeInstrumentRepository{}
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher}, WithRepository(repository))
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Sync(context.Background(), SyncRequest{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		AsOf:         "2026-05-13",
	})
	if err != nil {
		t.Fatalf("Sync error = %v", err)
	}
	if result.InstrumentsFetched != 1 || result.InstrumentsStored != 1 || len(repository.upserted) != 1 {
		t.Fatalf("result = %+v upserted=%d, want one stored", result, len(repository.upserted))
	}
	if repository.listQuery.Market != provider.MarketKRX {
		t.Fatalf("list query = %+v, want KRX market", repository.listQuery)
	}
	if got := repository.upserted[0].Extensions[instrumentAliasExtensionKey]; got != "SAMS" {
		t.Fatalf("stored alias = %q, want SAMS", got)
	}
}

func TestInspectReportsNotFoundForOnlyFuzzyResults(t *testing.T) {
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(context.Context, instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{
				{SecurityCode: "069501", Name: "KODEX 200 Similar"},
			},
		}, nil
	})
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Inspect(context.Background(), InspectRequest{Symbol: "069500"})
	if err == nil {
		t.Fatal("Inspect error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "instrument not found") {
		t.Fatalf("error = %q, want not found context", err.Error())
	}
}

func TestInspectReportsNotFound(t *testing.T) {
	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(context.Context, instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		return instrumentrole.SearchResult{}, nil
	})
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Inspect(context.Background(), InspectRequest{Symbol: "missing"})
	if err == nil {
		t.Fatal("Inspect error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "instrument not found") {
		t.Fatalf("error = %q, want not found context", err.Error())
	}
}

type fakeInstrumentRouter struct {
	searcher instrumentrole.Searcher
	gotRoute instrumentrole.RouteInput
}

func (r *fakeInstrumentRouter) RouteInstrumentSearch(_ context.Context, input instrumentrole.RouteInput) (instrumentrole.Searcher, error) {
	r.gotRoute = input
	return r.searcher, nil
}

type fakeInstrumentRepository struct {
	searchResult  instrumentrole.SearchResult
	searchErr     error
	searchQuery   Query
	listResult    instrumentrole.SearchResult
	listErr       error
	listQuery     Query
	inspectResult instrumentrole.Instrument
	inspectErr    error
	inspectQuery  Query
	upserted      []instrumentrole.Instrument
}

func (r *fakeInstrumentRepository) UpsertInstruments(_ context.Context, instruments []instrumentrole.Instrument) (WriteResult, error) {
	r.upserted = instruments
	return WriteResult{InstrumentsWritten: len(instruments), RowsAffected: len(instruments)}, nil
}

func (r *fakeInstrumentRepository) SearchInstruments(_ context.Context, query Query) (instrumentrole.SearchResult, error) {
	r.searchQuery = query
	return r.searchResult, r.searchErr
}

func (r *fakeInstrumentRepository) ListInstruments(_ context.Context, query Query) (instrumentrole.SearchResult, error) {
	r.listQuery = query
	return r.listResult, r.listErr
}

func (r *fakeInstrumentRepository) InspectInstrument(_ context.Context, query Query) (instrumentrole.Instrument, error) {
	r.inspectQuery = query
	if r.inspectErr != nil {
		return instrumentrole.Instrument{}, r.inspectErr
	}
	return r.inspectResult, nil
}
