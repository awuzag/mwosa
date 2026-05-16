package daily

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/samber/oops"
)

func TestNewReadServiceRequiresReader(t *testing.T) {
	_, err := NewReadService(nil)
	if err == nil {
		t.Fatal("NewReadService error = nil, want reader error")
	}
	if !strings.Contains(err.Error(), "read repository") {
		t.Fatalf("error = %q, want read repository context", err.Error())
	}
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	tests := []struct {
		name   string
		reader ReadRepository
		writer WriteRepository
		router dailybar.Router
		want   string
	}{
		{
			name:   "reader",
			writer: fakeWriteRepository{},
			router: fakeDailyBarRouter{},
			want:   "read repository",
		},
		{
			name:   "writer",
			reader: fakeReadRepository{},
			router: fakeDailyBarRouter{},
			want:   "write repository",
		},
		{
			name:   "router",
			reader: fakeReadRepository{},
			writer: fakeWriteRepository{},
			want:   "router",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.reader, tt.writer, tt.router)
			if err == nil {
				t.Fatal("NewService error = nil, want dependency error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestNewServiceAcceptsInjectedRouter(t *testing.T) {
	_, err := NewService(fakeReadRepository{}, fakeWriteRepository{}, fakeDailyBarRouter{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}
}

func TestReadServiceStorageSummaryUsesRepositoryResult(t *testing.T) {
	reader := &recordingCoverageRepository{
		storageSummary: StorageSummaryResult{
			RecordType:   "daily_bar",
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbols:      2,
			Bars:         3,
			From:         "2024-04-15",
			To:           "2024-04-16",
			Dates:        2,
		},
	}
	service, err := NewReadService(reader)
	if err != nil {
		t.Fatalf("NewReadService error = %v", err)
	}

	result, err := service.StorageSummary(context.Background(), StorageSummaryRequest{
		SecurityType: provider.SecurityTypeETF,
	})
	if err != nil {
		t.Fatalf("StorageSummary error = %v", err)
	}
	if reader.storageQuery.Market != provider.MarketKRX || reader.storageQuery.SecurityType != provider.SecurityTypeETF {
		t.Fatalf("storage query = %+v, want default krx etf", reader.storageQuery)
	}
	if result.Symbols != 2 || result.Bars != 3 || result.From != "2024-04-15" || result.To != "2024-04-16" {
		t.Fatalf("result = %+v, want repository summary shape", result)
	}
}

func TestReadServiceCoverageUsesRepositoryResult(t *testing.T) {
	reader := &recordingCoverageRepository{
		coverage: CoverageResult{
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			Symbol:       "069500",
			Name:         "KODEX 200",
			From:         "2024-04-15",
			To:           "2024-04-16",
			Bars:         2,
			Dates:        2,
		},
	}
	service, err := NewReadService(reader)
	if err != nil {
		t.Fatalf("NewReadService error = %v", err)
	}

	result, err := service.Coverage(context.Background(), CoverageRequest{
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err != nil {
		t.Fatalf("Coverage error = %v", err)
	}
	if reader.coverageQuery.Market != provider.MarketKRX || reader.coverageQuery.SecurityType != provider.SecurityTypeETF || reader.coverageQuery.Symbol != "069500" {
		t.Fatalf("coverage query = %+v, want default krx etf 069500", reader.coverageQuery)
	}
	if result.Name != "KODEX 200" || result.Bars != 2 || result.Dates != 2 {
		t.Fatalf("result = %+v, want repository coverage shape", result)
	}
}

func TestEnsurePassesSymbolToProviderFetch(t *testing.T) {
	reader := &sequenceReadRepository{
		results: [][]dailybar.Bar{
			nil,
			{{Symbol: "069500", TradingDate: "2024-04-15"}},
		},
	}
	var gotFetch dailybar.FetchInput
	fetcher := dailybar.NewFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(_ context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
		gotFetch = input
		return dailybar.FetchResult{
			Bars: []dailybar.Bar{
				{Symbol: "069500", TradingDate: "2024-04-15"},
			},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
		}, nil
	})

	service, err := NewService(reader, fakeWriteRepository{}, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Ensure(context.Background(), Request{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		AsOf:         "20240415",
	})
	if err != nil {
		t.Fatalf("Ensure error = %v", err)
	}
	if gotFetch.Symbol != "069500" {
		t.Fatalf("fetch symbol = %q, want 069500", gotFetch.Symbol)
	}
}

func TestBackfillFetchesProviderPagesWithWorkers(t *testing.T) {
	var gotPagesMu sync.Mutex
	gotPages := make([]int, 0)
	legacyFetchCalled := false
	fetcher := dailybar.NewPagedFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(_ context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
		legacyFetchCalled = true
		return dailybar.FetchResult{}, nil
	}, func(_ context.Context, input dailybar.PageFetchInput) (dailybar.PageFetchResult, error) {
		gotPagesMu.Lock()
		gotPages = append(gotPages, input.PageNo)
		gotPagesMu.Unlock()
		return dailybar.PageFetchResult{
			Bars: []dailybar.Bar{
				{Symbol: "069500", TradingDate: "2024-04-15"},
			},
			Provider:   provider.Identity{ID: provider.ProviderID("fake")},
			Group:      provider.GroupID("fakeGroup"),
			PageNo:     input.PageNo,
			PageSize:   1,
			TotalCount: 2,
		}, nil
	})
	writer := &recordingWriteRepository{}
	service, err := NewService(fakeReadRepository{}, writer, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Backfill(context.Background(), Request{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		From:         "20240415",
		To:           "20240416",
		Workers:      2,
	})
	if err != nil {
		t.Fatalf("Backfill error = %v", err)
	}
	if result.BarsFetched != 2 || result.BarsStored != 2 {
		t.Fatalf("result = %+v, want fetched/stored 2", result)
	}
	if legacyFetchCalled {
		t.Fatal("backfill used legacy FetchDailyBars; want page fetch pipeline")
	}
	gotPagesMu.Lock()
	sortInts(gotPages)
	gotPagesText := intsText(gotPages)
	gotPagesMu.Unlock()
	if gotPagesText != "1,2" {
		t.Fatalf("page fetches = %s, want 1,2", gotPagesText)
	}
	if strings.Join(result.Dates, ",") != "2024-04-15" {
		t.Fatalf("dates = %v, want sorted backfill dates", result.Dates)
	}
	if writer.barsWritten != 2 {
		t.Fatalf("writer bars = %d, want 2", writer.barsWritten)
	}
}

func TestBackfillUsesBatchFetcherWhenAvailable(t *testing.T) {
	batchCalls := 0
	fetchCalls := 0
	fetcher := dailybar.NewBatchFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeStock},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(_ context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
		fetchCalls++
		return dailybar.FetchResult{}, nil
	}, func(_ context.Context, input dailybar.BatchFetchInput) (dailybar.BatchFetchResult, error) {
		batchCalls++
		if input.From != "20240415" || input.To != "20240415" {
			t.Fatalf("batch range = %s..%s, want 20240415..20240415", input.From, input.To)
		}
		return dailybar.BatchFetchResult{
			Bars: []dailybar.Bar{
				{Symbol: "005930", TradingDate: "2024-04-15"},
				{Symbol: "000660", TradingDate: "2024-04-15"},
				{Symbol: "950210", TradingDate: "2024-04-15"},
			},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
			Group:    provider.GroupID("fakeGroup"),
		}, nil
	})
	writer := &recordingWriteRepository{}
	service, err := NewService(fakeReadRepository{}, writer, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Backfill(context.Background(), Request{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		From:         "20240415",
		To:           "20240415",
		Workers:      4,
	})
	if err != nil {
		t.Fatalf("Backfill error = %v", err)
	}
	if batchCalls != 1 {
		t.Fatalf("batch calls = %d, want 1", batchCalls)
	}
	if fetchCalls != 0 {
		t.Fatalf("legacy fetch calls = %d, want 0", fetchCalls)
	}
	if result.BarsFetched != 3 || result.BarsStored != 3 || writer.barsWritten != 3 {
		t.Fatalf("result = %+v writer=%d, want fetched/stored 3", result, writer.barsWritten)
	}
}

func TestSyncUsesBatchFetcherWhenAvailable(t *testing.T) {
	batchCalls := 0
	fetcher := dailybar.NewBatchFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{}, nil
	}, func(_ context.Context, input dailybar.BatchFetchInput) (dailybar.BatchFetchResult, error) {
		batchCalls++
		return dailybar.BatchFetchResult{
			Bars:     []dailybar.Bar{{Symbol: "069500", TradingDate: "2024-04-15"}},
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
			Group:    provider.GroupID("fakeGroup"),
		}, nil
	})
	writer := &recordingWriteRepository{}
	service, err := NewService(fakeReadRepository{}, writer, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Sync(context.Background(), Request{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		AsOf:         "20240415",
	})
	if err != nil {
		t.Fatalf("Sync error = %v", err)
	}
	if batchCalls != 1 {
		t.Fatalf("batch calls = %d, want 1", batchCalls)
	}
	if result.BarsFetched != 1 || result.BarsStored != 1 || writer.barsWritten != 1 {
		t.Fatalf("result = %+v writer=%d, want fetched/stored 1", result, writer.barsWritten)
	}
}

func TestBackfillStoresFirstPageBeforeRemainingFetchCompletes(t *testing.T) {
	allowSecondPage := make(chan struct{})
	firstStored := make(chan struct{})
	done := make(chan error, 1)
	fetcher := dailybar.NewPagedFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{}, nil
	}, func(_ context.Context, input dailybar.PageFetchInput) (dailybar.PageFetchResult, error) {
		if input.PageNo == 2 {
			<-allowSecondPage
		}
		return dailybar.PageFetchResult{
			Bars:       []dailybar.Bar{{Symbol: "069500", TradingDate: "2024-04-15"}},
			Provider:   provider.Identity{ID: provider.ProviderID("fake")},
			Group:      provider.GroupID("fakeGroup"),
			PageNo:     input.PageNo,
			PageSize:   1,
			TotalCount: 2,
		}, nil
	})
	writer := &recordingWriteRepository{afterWrite: func(call int) {
		if call == 1 {
			close(firstStored)
		}
	}}
	service, err := NewService(fakeReadRepository{}, writer, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	go func() {
		_, err := service.Backfill(context.Background(), Request{
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeETF,
			From:         "20240415",
			To:           "20240416",
			Workers:      2,
		})
		done <- err
	}()

	select {
	case <-firstStored:
	case <-time.After(time.Second):
		t.Fatal("first page was not stored before remaining fetch completed")
	}
	close(allowSecondPage)
	if err := <-done; err != nil {
		t.Fatalf("Backfill error = %v", err)
	}
}

func TestBackfillFetchErrorPreservesStoredChunks(t *testing.T) {
	fetcher := dailybar.NewPagedFetch(dailybar.Profile{
		Markets:       []provider.Market{provider.MarketKRX},
		SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
		Compatibility: provider.Compatibility{DataLatency: provider.DataLatencyPreviousBusinessDay},
	}, func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{}, nil
	}, func(_ context.Context, input dailybar.PageFetchInput) (dailybar.PageFetchResult, error) {
		if input.PageNo == 2 {
			return dailybar.PageFetchResult{}, oops.In("test").New("page 2 failed")
		}
		return dailybar.PageFetchResult{
			Bars:       []dailybar.Bar{{Symbol: "069500", TradingDate: "2024-04-15"}},
			Provider:   provider.Identity{ID: provider.ProviderID("fake")},
			Group:      provider.GroupID("fakeGroup"),
			PageNo:     input.PageNo,
			PageSize:   1,
			TotalCount: 2,
		}, nil
	})
	writer := &recordingWriteRepository{}
	service, err := NewService(fakeReadRepository{}, writer, fakeDailyBarRouter{fetcher: fetcher})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Backfill(context.Background(), Request{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		From:         "20240415",
		To:           "20240416",
		Workers:      2,
	})
	if err == nil {
		t.Fatal("Backfill error = nil, want page fetch error")
	}
	if writer.barsWritten != 1 {
		t.Fatalf("writer bars = %d, want first chunk preserved", writer.barsWritten)
	}
}

type fakeReadRepository struct{}

func (fakeReadRepository) QueryDailyBars(context.Context, Query) ([]dailybar.Bar, error) {
	return nil, nil
}

func (fakeReadRepository) SummarizeDailyBarStorage(context.Context, Query) (StorageSummaryResult, error) {
	return StorageSummaryResult{}, nil
}

func (fakeReadRepository) QueryDailyBarCoverage(context.Context, Query) (CoverageResult, error) {
	return CoverageResult{}, nil
}

type fakeWriteRepository struct{}

func (fakeWriteRepository) UpsertDailyBars(context.Context, []dailybar.Bar) (WriteResult, error) {
	return WriteResult{}, nil
}

type recordingWriteRepository struct {
	barsWritten int
	calls       int
	afterWrite  func(call int)
}

func (r *recordingWriteRepository) UpsertDailyBars(_ context.Context, bars []dailybar.Bar) (WriteResult, error) {
	r.calls++
	r.barsWritten += len(bars)
	if r.afterWrite != nil {
		r.afterWrite(r.calls)
	}
	return WriteResult{BarsWritten: len(bars), RowsAffected: len(bars)}, nil
}

type fakeDailyBarRouter struct {
	fetcher dailybar.Fetcher
}

func sortInts(values []int) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func intsText(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func (r fakeDailyBarRouter) RouteDailyBars(context.Context, dailybar.RouteInput) (dailybar.Fetcher, error) {
	return r.fetcher, nil
}

func (fakeDailyBarRouter) PlanDailyBars(context.Context, dailybar.RouteInput) (dailybar.RoutePlan, error) {
	return dailybar.RoutePlan{}, nil
}

type sequenceReadRepository struct {
	results [][]dailybar.Bar
	calls   int
}

func (r *sequenceReadRepository) QueryDailyBars(context.Context, Query) ([]dailybar.Bar, error) {
	if r.calls >= len(r.results) {
		return nil, nil
	}
	result := r.results[r.calls]
	r.calls++
	return result, nil
}

func (r *sequenceReadRepository) SummarizeDailyBarStorage(context.Context, Query) (StorageSummaryResult, error) {
	return StorageSummaryResult{}, nil
}

func (r *sequenceReadRepository) QueryDailyBarCoverage(context.Context, Query) (CoverageResult, error) {
	return CoverageResult{}, nil
}

type recordingCoverageRepository struct {
	storageSummary StorageSummaryResult
	coverage       CoverageResult
	storageQuery   Query
	coverageQuery  Query
}

func (r *recordingCoverageRepository) QueryDailyBars(context.Context, Query) ([]dailybar.Bar, error) {
	return nil, nil
}

func (r *recordingCoverageRepository) SummarizeDailyBarStorage(_ context.Context, query Query) (StorageSummaryResult, error) {
	r.storageQuery = query
	return r.storageSummary, nil
}

func (r *recordingCoverageRepository) QueryDailyBarCoverage(_ context.Context, query Query) (CoverageResult, error) {
	r.coverageQuery = query
	return r.coverage, nil
}
