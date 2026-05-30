package krx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/awuzag/mwosa/providers/core/instrument"
	dailyservice "github.com/awuzag/mwosa/service/daily"
)

func TestFetchETFDailyBars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/etp/etf_bydd_trd" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("AUTH_KEY"); got != "test-key" {
			t.Fatalf("AUTH_KEY = %q, want test-key", got)
		}
		if got := r.URL.Query().Get("basDd"); got != "20240415" {
			t.Fatalf("basDd = %q, want 20240415", got)
		}
		fmt.Fprint(w, `{
			"OutBlock_1": [
				{
					"BAS_DD": "20240415",
					"ISU_CD": "069500",
					"ISU_NM": "KODEX 200",
					"TDD_CLSPRC": "35120",
					"CMPPREVDD_PRC": "-15",
					"FLUC_RT": "-0.04",
					"NAV": "35155.1",
					"TDD_OPNPRC": "35100",
					"TDD_HGPRC": "35200",
					"TDD_LWPRC": "35000",
					"ACC_TRDVOL": "123456",
					"ACC_TRDVAL": "4321000",
					"MKTCAP": "1000000000",
					"INVSTASST_NETASST_TOTAMT": "2000000000",
					"LIST_SHRS": "100000"
				}
			]
		}`)
	}))
	defer server.Close()

	p, err := New(Config{AuthKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	result, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
		From:         "20240415",
		To:           "20240415",
	})
	if err != nil {
		t.Fatalf("fetch daily bars: %v", err)
	}
	if len(result.Bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(result.Bars))
	}
	bar := result.Bars[0]
	if bar.Provider != provider.ProviderKRX || bar.Group != provider.GroupKRXETPDailyTrade || bar.Operation != provider.OperationETFByddTrd {
		t.Fatalf("unexpected provenance: %+v", bar)
	}
	if bar.TradingDate != "2024-04-15" || bar.Symbol != "069500" || bar.Close != "35120" {
		t.Fatalf("unexpected bar: %+v", bar)
	}
	if bar.Extensions["nav"] != "35155.1" || bar.Extensions["nPptTotAmt"] != "2000000000" {
		t.Fatalf("unexpected extensions: %+v", bar.Extensions)
	}
}

func TestFetchKOSPIIndexBars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/idx/kospi_dd_trd" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("basDd"); got != "20240415" {
			t.Fatalf("basDd = %q, want 20240415", got)
		}
		fmt.Fprint(w, `{
			"OutBlock_1": [
				{
					"BAS_DD": "20240415",
					"IDX_CLSS": "KOSPI",
					"IDX_NM": "KOSPI",
					"CLSPRC_IDX": "2670.43",
					"CMPPREVDD_IDX": "11.39",
					"FLUC_RT": "0.43",
					"OPNPRC_IDX": "2660.00",
					"HGPRC_IDX": "2680.10",
					"LWPRC_IDX": "2655.20",
					"ACC_TRDVOL": "450000000",
					"ACC_TRDVAL": "9000000000000",
					"MKTCAP": "2100000000000000"
				}
			]
		}`)
	}))
	defer server.Close()

	p, err := New(Config{AuthKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	registry := provider.NewRegistry()
	if err := Register(registry, p); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	fetcher, err := indexbar.NewRouter(provider.NewRouter(registry)).RouteIndexBars(context.Background(), indexbar.RouteInput{
		ProviderID: provider.ProviderKRX,
		Market:     provider.MarketKRX,
		IndexCode:  "KOSPI",
	})
	if err != nil {
		t.Fatalf("route index bars: %v", err)
	}
	if _, ok := fetcher.(indexbar.BatchFetcher); !ok {
		t.Fatalf("routed fetcher type = %T, want BatchFetcher", fetcher)
	}
	result, err := fetcher.FetchIndexBars(context.Background(), indexbar.FetchInput{
		Market:    provider.MarketKRX,
		IndexCode: "KOSPI",
		From:      "20240415",
		To:        "20240415",
	})
	if err != nil {
		t.Fatalf("fetch index bars: %v", err)
	}
	if len(result.Bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(result.Bars))
	}
	bar := result.Bars[0]
	if bar.Provider != provider.ProviderKRX || bar.Group != provider.GroupKRXIndexDailyTrade || bar.Operation != provider.OperationKOSPIDDTrd {
		t.Fatalf("unexpected provenance: %+v", bar)
	}
	if bar.IndexCode != "KOSPI" || bar.Name != "KOSPI" || bar.Close != "2670.43" || bar.TradingDate != "2024-04-15" {
		t.Fatalf("unexpected index bar: %+v", bar)
	}
}

func TestSearchStockInstruments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/sto/stk_isu_base_info":
			fmt.Fprint(w, `{"OutBlock_1":[{"ISU_CD":"KR7005930003","ISU_SRT_CD":"005930","ISU_NM":"삼성전자","ISU_ABBRV":"삼성전자","LIST_DD":"19750611","MKT_TP_NM":"KOSPI","SECUGRP_NM":"주권"}]}`)
		case "/sto/ksq_isu_base_info", "/sto/knx_isu_base_info":
			fmt.Fprint(w, `{"OutBlock_1":[]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p, err := New(Config{
		AuthKey: "test-key",
		BaseURL: server.URL,
		Now:     func() time.Time { return time.Date(2024, 4, 16, 9, 0, 0, 0, time.Local) },
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	result, err := p.SearchInstruments(context.Background(), instrument.SearchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Query:        "005930",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("search instruments: %v", err)
	}
	if len(result.Instruments) != 1 {
		t.Fatalf("instruments len = %d, want 1", len(result.Instruments))
	}
	got := result.Instruments[0]
	if got.SecurityCode != "005930" || got.ISIN != "KR7005930003" || got.Extensions["listingDate"] != "19750611" {
		t.Fatalf("unexpected instrument: %+v", got)
	}
}

func TestSearchETPInstrumentsReportsExecutedOperations(t *testing.T) {
	var pathsMu sync.Mutex
	paths := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathsMu.Lock()
		paths[r.URL.Path]++
		pathsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/etp/etf_bydd_trd":
			fmt.Fprint(w, `{"OutBlock_1":[{"BAS_DD":"20240415","ISU_CD":"069500","ISU_NM":"KODEX 200"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p, err := New(Config{AuthKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	result, err := p.SearchInstruments(context.Background(), instrument.SearchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Query:        "069500",
		Limit:        10,
		AsOf:         "20240415",
	})
	if err != nil {
		t.Fatalf("search ETF instruments: %v", err)
	}
	if len(result.Instruments) != 1 {
		t.Fatalf("instruments len = %d, want 1", len(result.Instruments))
	}
	if len(result.Operations) != 1 || result.Operations[0] != provider.OperationETFByddTrd {
		t.Fatalf("operations = %v, want [%s]", result.Operations, provider.OperationETFByddTrd)
	}

	pathsMu.Lock()
	defer pathsMu.Unlock()
	if paths["/etp/etf_bydd_trd"] != 1 || paths["/etp/etn_bydd_trd"] != 0 || paths["/etp/elw_bydd_trd"] != 0 {
		t.Fatalf("paths = %+v, want only ETF endpoint called once", paths)
	}
}

func TestStockBackfillUsesOneKRXBatchPerEndpoint(t *testing.T) {
	var countsMu sync.Mutex
	counts := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countsMu.Lock()
		counts[r.URL.Path]++
		countsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if got := r.URL.Query().Get("basDd"); got != "20240415" {
			t.Fatalf("basDd = %q, want 20240415", got)
		}
		switch r.URL.Path {
		case "/sto/stk_bydd_trd":
			writeStockDailyRows(w, "KOSPI", 1001)
		case "/sto/ksq_bydd_trd":
			writeStockDailyRows(w, "KOSDAQ", 1)
		case "/sto/knx_bydd_trd":
			writeStockDailyRows(w, "KONEX", 1)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p, err := New(Config{AuthKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	registry := provider.NewRegistry()
	if err := Register(registry, p); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	fetcher, err := dailybar.NewRouter(provider.NewRouter(registry)).RouteDailyBars(context.Background(), dailybar.RouteInput{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
	})
	if err != nil {
		t.Fatalf("route daily bars: %v", err)
	}
	if _, ok := fetcher.(dailybar.BatchFetcher); !ok {
		t.Fatalf("routed fetcher type = %T, want BatchFetcher", fetcher)
	}
	if _, ok := fetcher.(dailybar.PageFetcher); ok {
		t.Fatalf("routed fetcher type = %T, want no PageFetcher compatibility shim", fetcher)
	}

	writer := &fakeDailyWriter{}
	service, err := dailyservice.NewService(fakeDailyReader{}, writer, dailybar.NewRouter(provider.NewRouter(registry)))
	if err != nil {
		t.Fatalf("new daily service: %v", err)
	}
	result, err := service.Backfill(context.Background(), dailyservice.Request{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		From:         "20240415",
		To:           "20240415",
		Workers:      4,
	})
	if err != nil {
		t.Fatalf("backfill stock daily: %v", err)
	}
	if result.BarsFetched != 1003 || result.BarsStored != 1003 || writer.bars != 1003 {
		t.Fatalf("result = %+v writer=%d, want 1003 fetched/stored", result, writer.bars)
	}

	countsMu.Lock()
	gotCounts := map[string]int{
		"/sto/stk_bydd_trd": counts["/sto/stk_bydd_trd"],
		"/sto/ksq_bydd_trd": counts["/sto/ksq_bydd_trd"],
		"/sto/knx_bydd_trd": counts["/sto/knx_bydd_trd"],
	}
	countsMu.Unlock()
	for path, got := range gotCounts {
		if got != 1 {
			t.Fatalf("%s calls = %d, want 1", path, got)
		}
	}
}

func TestDisabledAPIIsExplicitUnsupported(t *testing.T) {
	p := NewWithClient(nil, map[provider.OperationID]bool{
		provider.OperationETFByddTrd: false,
	}, nil)
	_, err := p.FetchRaw(context.Background(), RawRequest{
		APIID:    provider.OperationETFByddTrd,
		BaseDate: "20240415",
	})
	if err == nil {
		t.Fatal("FetchRaw error = nil, want unsupported")
	}
	if !strings.Contains(err.Error(), "service is disabled") {
		t.Fatalf("error = %q, want disabled service context", err.Error())
	}
}

func writeStockDailyRows(w http.ResponseWriter, market string, count int) {
	fmt.Fprint(w, `{"OutBlock_1":[`)
	for i := 0; i < count; i++ {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		fmt.Fprintf(w, `{
			"BAS_DD":"20240415",
			"ISU_CD":"%06d",
			"ISU_NM":"%s %06d",
			"MKT_NM":"%s",
			"SECT_TP_NM":"stock",
			"TDD_CLSPRC":"70000",
			"CMPPREVDD_PRC":"100",
			"FLUC_RT":"0.14",
			"TDD_OPNPRC":"69900",
			"TDD_HGPRC":"70100",
			"TDD_LWPRC":"69800",
			"ACC_TRDVOL":"1000",
			"ACC_TRDVAL":"70000000",
			"MKTCAP":"1000000000",
			"LIST_SHRS":"100000"
		}`, i+1, market, i+1, market)
	}
	fmt.Fprint(w, `]}`)
}

type fakeDailyReader struct{}

func (fakeDailyReader) QueryDailyBars(context.Context, dailyservice.Query) ([]dailybar.Bar, error) {
	return nil, nil
}

func (fakeDailyReader) SummarizeDailyBarStorage(context.Context, dailyservice.Query) (dailyservice.StorageSummaryResult, error) {
	return dailyservice.StorageSummaryResult{}, nil
}

func (fakeDailyReader) QueryDailyBarCoverage(context.Context, dailyservice.Query) (dailyservice.CoverageResult, error) {
	return dailyservice.CoverageResult{}, nil
}

type fakeDailyWriter struct {
	bars int
}

func (w *fakeDailyWriter) UpsertDailyBars(_ context.Context, bars []dailybar.Bar) (dailyservice.WriteResult, error) {
	w.bars += len(bars)
	return dailyservice.WriteResult{BarsWritten: len(bars), RowsAffected: len(bars)}, nil
}
