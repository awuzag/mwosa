package krx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
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
