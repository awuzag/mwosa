package datago

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	datagocorpfin "github.com/ev3rlit/mwosa/clients/datago-corpfin"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/quote"
)

func TestFetchDailyBarsDecodesSingleObjectItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		assertCommonQuery(t, r, "1", "1000")
		if got := r.URL.Query().Get("resultType"); got != "json" {
			t.Fatalf("resultType = %q, want json", got)
		}
		if got := r.URL.Query().Get("likeSrtnCd"); got != "069500" {
			t.Fatalf("likeSrtnCd = %q, want 069500", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 1,
				"items": {
					"item": {
						"basDt": "20240415",
						"srtnCd": "069500",
						"isinCd": "KR7069500007",
						"itmsNm": "KODEX 200",
						"clpr": 35120,
						"vs": "-15",
						"fltRt": "-0.04",
						"mkp": "35100",
						"hipr": "35200",
						"lopr": "35000",
						"trqu": "123456",
						"trPrc": "4321000",
						"mrktTotAmt": "1000000000",
						"nav": "35155.1"
					}
				}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
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
	if bar.Provider != provider.ProviderDataGo || bar.Group != provider.GroupSecuritiesProductPrice {
		t.Fatalf("unexpected provenance: provider=%s group=%s", bar.Provider, bar.Group)
	}
	if bar.Operation != provider.OperationGetETFPriceInfo {
		t.Fatalf("operation = %s, want %s", bar.Operation, provider.OperationGetETFPriceInfo)
	}
	if bar.TradingDate != "2024-04-15" {
		t.Fatalf("trading date = %q, want 2024-04-15", bar.TradingDate)
	}
	if bar.Close != "35120" {
		t.Fatalf("close = %q, want 35120", bar.Close)
	}
	if bar.Extensions["nav"] != "35155.1" {
		t.Fatalf("nav extension = %q, want 35155.1", bar.Extensions["nav"])
	}
}

func TestFetchDailyBarsUsesNameSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		assertCommonQuery(t, r, "1", "1000")
		if got := r.URL.Query().Get("itmsNm"); got != "KODEX 200" {
			t.Fatalf("itmsNm = %q, want KODEX 200", got)
		}
		if got := r.URL.Query().Get("likeSrtnCd"); got != "" {
			t.Fatalf("likeSrtnCd = %q, want empty", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 1,
				"items": {
					"item": {
						"basDt": "20240415",
						"srtnCd": "069500",
						"itmsNm": "KODEX 200",
						"clpr": "35120"
					}
				}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "KODEX 200",
		From:         "20240415",
		To:           "20240415",
	})
	if err != nil {
		t.Fatalf("fetch daily bars: %v", err)
	}
	if len(result.Bars) != 1 || result.Bars[0].Symbol != "069500" {
		t.Fatalf("bars = %+v, want KODEX 200 match", result.Bars)
	}
}

func TestFetchDailyBarsCollectsAllPagesWhenLimitOmitted(t *testing.T) {
	seenPages := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		pageNo := r.URL.Query().Get("pageNo")
		seenPages = append(seenPages, pageNo)
		assertCommonQuery(t, r, pageNo, "1000")
		if got := r.URL.Query().Get("beginBasDt"); got != "20240415" {
			t.Fatalf("beginBasDt = %q, want 20240415", got)
		}
		if got := r.URL.Query().Get("endBasDt"); got != "20240417" {
			t.Fatalf("endBasDt = %q, want 20240417", got)
		}
		if got := r.URL.Query().Get("basDt"); got != "" {
			t.Fatalf("basDt = %q, want empty", got)
		}
		switch pageNo {
		case "1":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 1001,
					"items": {"item": [
						{"basDt": "20240415", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35120"}
					]}
				}
			}`)
		case "2":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 2,
					"totalCount": 1001,
					"items": {"item": [
						{"basDt": "20240415", "srtnCd": "069501", "itmsNm": "KODEX Next", "clpr": "1000"}
					]}
				}
			}`)
		default:
			t.Fatalf("unexpected pageNo: %s", pageNo)
		}
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		From:         "20240415",
		To:           "20240416",
	})
	if err != nil {
		t.Fatalf("fetch daily bars: %v", err)
	}
	if strings.Join(seenPages, ",") != "1,2" {
		t.Fatalf("seen pages = %v, want [1 2]", seenPages)
	}
	if len(result.Bars) != 2 || result.TotalCount != 1001 {
		t.Fatalf("result = %+v, want two fetched bars with totalCount 1001", result)
	}
}

func TestFetchDailyBarsDecodesOnlyRequestedPage(t *testing.T) {
	seenPages := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETNPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		pageNo := r.URL.Query().Get("pageNo")
		seenPages = append(seenPages, pageNo)
		assertCommonQuery(t, r, pageNo, "1")
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 1,
				"pageNo": 1,
				"totalCount": 2,
				"items": {"item": [
					{"basDt": "20240415", "srtnCd": "580001", "itmsNm": "ETN A", "clpr": "1000"}
				]}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETN,
		Symbol:       "580001",
		Limit:        1,
	})
	if err != nil {
		t.Fatalf("fetch daily bars: %v", err)
	}

	if strings.Join(seenPages, ",") != "1" {
		t.Fatalf("seen pages = %v, want [1]", seenPages)
	}
	if len(result.Bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(result.Bars))
	}
	if result.Bars[0].Close != "1000" {
		t.Fatalf("close = %q, want 1000", result.Bars[0].Close)
	}
}

func TestFetchDailyBarsPageUsesRequestedPage(t *testing.T) {
	seenPages := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		pageNo := r.URL.Query().Get("pageNo")
		seenPages = append(seenPages, pageNo)
		assertCommonQuery(t, r, "2", "1")
		if got := r.URL.Query().Get("beginBasDt"); got != "20240415" {
			t.Fatalf("beginBasDt = %q, want 20240415", got)
		}
		if got := r.URL.Query().Get("endBasDt"); got != "20240417" {
			t.Fatalf("endBasDt = %q, want 20240417", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 1,
				"pageNo": 2,
				"totalCount": 2,
				"items": {"item": [
					{"basDt": "20240416", "srtnCd": "069501", "itmsNm": "KODEX Next", "clpr": "1000"}
				]}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.FetchDailyBarsPage(context.Background(), dailybar.PageFetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		From:         "20240415",
		To:           "20240416",
		PageNo:       2,
		PageSize:     1,
	})
	if err != nil {
		t.Fatalf("fetch daily bars page: %v", err)
	}
	if strings.Join(seenPages, ",") != "2" {
		t.Fatalf("seen pages = %v, want [2]", seenPages)
	}
	if result.PageNo != 2 || result.PageSize != 1 || result.TotalCount != 2 {
		t.Fatalf("page metadata = %+v, want page 2 size 1 total 2", result)
	}
	if len(result.Bars) != 1 || result.Bars[0].Symbol != "069501" {
		t.Fatalf("bars = %+v, want requested page bar", result.Bars)
	}
}

func TestSearchInstrumentsReturnsEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 0,
				"items": {}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.SearchInstruments(context.Background(), instrumentInput("missing"))
	if err != nil {
		t.Fatalf("search instruments: %v", err)
	}
	if len(result.Instruments) != 0 {
		t.Fatalf("instruments len = %d, want 0", len(result.Instruments))
	}
	if len(result.Operations) != 3 {
		t.Fatalf("operations len = %d, want ETF/ETN/stock defaults", len(result.Operations))
	}
}

func TestFetchStockDailyBarsUsesStockPriceGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getStockPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		assertCommonQuery(t, r, "1", "1000")
		if got := r.URL.Query().Get("resultType"); got != "json" {
			t.Fatalf("resultType = %q, want json", got)
		}
		if got := r.URL.Query().Get("likeSrtnCd"); got != "005930" {
			t.Fatalf("likeSrtnCd = %q, want 005930", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 1,
				"items": {
					"item": {
						"basDt": "20240415",
						"srtnCd": "005930",
						"isinCd": "KR7005930003",
						"itmsNm": "Samsung Electronics",
						"mrktCtg": "KOSPI",
						"clpr": "82200",
						"vs": "-100",
						"fltRt": "-0.12",
						"mkp": "82400",
						"hipr": "82500",
						"lopr": "81500",
						"trqu": "12345678",
						"trPrc": "1000000000000",
						"lstgStCnt": "5969782550",
						"mrktTotAmt": "490000000000000"
					}
				}
			}
		}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	result, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		From:         "20240415",
		To:           "20240415",
	})
	if err != nil {
		t.Fatalf("fetch stock daily bars: %v", err)
	}
	if len(result.Bars) != 1 {
		t.Fatalf("bars len = %d, want 1", len(result.Bars))
	}
	bar := result.Bars[0]
	if bar.Group != provider.GroupStockPrice {
		t.Fatalf("group = %s, want %s", bar.Group, provider.GroupStockPrice)
	}
	if bar.Operation != provider.OperationGetStockPriceInfo {
		t.Fatalf("operation = %s, want %s", bar.Operation, provider.OperationGetStockPriceInfo)
	}
	if bar.SecurityType != provider.SecurityTypeStock || bar.Symbol != "005930" || bar.Close != "82200" {
		t.Fatalf("unexpected stock bar: %+v", bar)
	}
	if bar.Extensions["lstgStCnt"] != "5969782550" || bar.Extensions["mrktCtg"] != "KOSPI" {
		t.Fatalf("stock extensions = %+v, want listing count and market category", bar.Extensions)
	}
}

func TestFetchFinancialStatementsFetchesSummaryBalanceSheetAndIncomeStatement(t *testing.T) {
	seenPaths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		assertCommonQuery(t, r, "1", "1000")
		if got := r.URL.Query().Get("crno"); got != "1746110000741" {
			t.Fatalf("crno = %q, want 1746110000741", got)
		}
		if got := r.URL.Query().Get("bizYear"); got != "2019" {
			t.Fatalf("bizYear = %q, want 2019", got)
		}
		switch r.URL.Path {
		case "/getSummFinaStat_V2":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 1,
					"items": {"item": {
						"basDt": "20200101",
						"bizYear": "2019",
						"crno": "1746110000741",
						"curCd": "KRW",
						"enpSaleAmt": "1000",
						"enpBzopPft": "200"
					}}
				}
			}`)
		case "/getBs_V2":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 1,
					"items": {"item": [
						{"basDt": "20200101", "bizYear": "2019", "crno": "1746110000741", "curCd": "KRW", "acitId": "ifrs_Assets", "acitNm": "Assets", "crtmAcitAmt": "5000"}
					]}
				}
			}`)
		case "/getIncoStat_V2":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 1,
					"items": {"item": [
						{"basDt": "20200101", "bizYear": "2019", "crno": "1746110000741", "curCd": "KRW", "acitId": "ifrs_Revenue", "acitNm": "Revenue", "crtmAcitAmt": "1000", "thqrAcitAmt": "300"}
					]}
				}
			}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := newTestCorporateFinanceProvider(t, server.URL)
	result, err := p.FetchFinancialStatements(context.Background(), financials.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "1746110000741",
		FiscalYear:   "2019",
	})
	if err != nil {
		t.Fatalf("fetch financial statements: %v", err)
	}
	if strings.Join(seenPaths, ",") != "/getSummFinaStat_V2,/getBs_V2,/getIncoStat_V2" {
		t.Fatalf("seen paths = %v, want all financial statement operations", seenPaths)
	}
	if len(result.Statements) != 3 || result.TotalCount != 3 {
		t.Fatalf("result = %+v, want three statements", result)
	}
	if result.Statements[0].Statement != financials.StatementTypeSummary || result.Statements[0].Lines[0].Value != "1000" {
		t.Fatalf("summary statement = %+v", result.Statements[0])
	}
	if result.Statements[1].Statement != financials.StatementTypeBalanceSheet || result.Statements[1].Lines[0].AccountName != "Assets" {
		t.Fatalf("balance statement = %+v", result.Statements[1])
	}
	if result.Statements[2].Statement != financials.StatementTypeIncomeStatement || result.Statements[2].Lines[0].Value != "1000" {
		t.Fatalf("income statement = %+v", result.Statements[2])
	}
}

func TestFetchFinancialStatementsCanSelectQuarterIncomeStatement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getIncoStat_V2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 1000,
				"pageNo": 1,
				"totalCount": 1,
				"items": {"item": [
					{"basDt": "20200101", "bizYear": "2019", "crno": "1746110000741", "curCd": "KRW", "acitId": "ifrs_Revenue", "acitNm": "Revenue", "crtmAcitAmt": "1000", "thqrAcitAmt": "300"}
				]}
			}
		}`)
	}))
	defer server.Close()

	p := newTestCorporateFinanceProvider(t, server.URL)
	result, err := p.FetchFinancialStatements(context.Background(), financials.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "1746110000741",
		FiscalYear:   "2019",
		Period:       financials.PeriodTypeQuarter,
		Statement:    financials.StatementTypeIncomeStatement,
	})
	if err != nil {
		t.Fatalf("fetch financial statements: %v", err)
	}
	if len(result.Statements) != 1 || result.Statements[0].Lines[0].Value != "300" {
		t.Fatalf("result = %+v, want quarter amount", result)
	}
}

func TestFetchFinancialStatementsResolvesShortCodeWithKRXListedInfo(t *testing.T) {
	seenPaths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		switch r.URL.Path {
		case "/getItemInfo":
			switch len(seenPaths) {
			case 1:
				assertCommonQuery(t, r, "1", "10")
				if got := r.URL.Query().Get("itmsNm"); got != "005930" {
					t.Fatalf("itmsNm = %q, want 005930", got)
				}
				fmt.Fprint(w, `{
					"header": {"resultCode": "00", "resultMsg": "OK"},
					"body": {
						"numOfRows": 10,
						"pageNo": 1,
						"totalCount": 0,
						"items": {}
					}
				}`)
				return
			case 2:
				assertCommonQuery(t, r, "1", "10")
				if got := r.URL.Query().Get("likeSrtnCd"); got != "005930" {
					t.Fatalf("likeSrtnCd = %q, want 005930", got)
				}
			default:
				t.Fatalf("unexpected getItemInfo call count: %d", len(seenPaths))
			}
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 10,
					"pageNo": 1,
					"totalCount": 1,
					"items": {"item": {
						"basDt": "20260430",
						"srtnCd": "005930",
						"isinCd": "KR7005930003",
						"mrktCtg": "KOSPI",
						"itmsNm": "삼성전자",
						"crno": "1301110006246",
						"corpNm": "삼성전자주식회사"
					}}
				}
			}`)
		case "/getSummFinaStat_V2":
			assertCommonQuery(t, r, "1", "1000")
			if got := r.URL.Query().Get("crno"); got != "1301110006246" {
				t.Fatalf("crno = %q, want 1301110006246", got)
			}
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 1,
					"items": {"item": {
						"basDt": "20200101",
						"bizYear": "2019",
						"crno": "1301110006246",
						"curCd": "KRW",
						"enpSaleAmt": "1000"
					}}
				}
			}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := newTestCorporateFinanceProvider(t, server.URL)
	result, err := p.FetchFinancialStatements(context.Background(), financials.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		FiscalYear:   "2019",
		Statement:    financials.StatementTypeSummary,
	})
	if err != nil {
		t.Fatalf("fetch financial statements: %v", err)
	}
	if strings.Join(seenPaths, ",") != "/getItemInfo,/getItemInfo,/getSummFinaStat_V2" {
		t.Fatalf("seen paths = %v, want symbol resolution then summary", seenPaths)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("statements len = %d, want 1", len(result.Statements))
	}
	statement := result.Statements[0]
	if statement.Symbol != "1301110006246" || statement.Name != "삼성전자주식회사" {
		t.Fatalf("statement identity = %+v, want resolved crno and name", statement)
	}
	if statement.Extensions["request_symbol"] != "005930" || statement.Extensions["srtnCd"] != "005930" {
		t.Fatalf("statement extensions = %+v, want symbol resolution metadata", statement.Extensions)
	}
}

func TestFetchFinancialStatementsRequiresResolutionClientForShortCode(t *testing.T) {
	p := NewCorporateFinanceWithClients(stubFinancialClient{}, nil)
	_, err := p.FetchFinancialStatements(context.Background(), financials.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		FiscalYear:   "2019",
	})
	if err == nil {
		t.Fatal("fetch financial statements error = nil, want unsupported crno error")
	}
	var unsupported *provider.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error type = %T, want UnsupportedError: %v", err, err)
	}
	for _, want := range []string{"provider=datago-corpfin", "group=krxListedInfo", "symbol=005930", "symbol resolution"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

func TestRemoteErrorIncludesProviderContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	_, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err == nil {
		t.Fatal("fetch daily bars error = nil, want remote error")
	}
	for _, want := range []string{"provider=datago", "group=securitiesProductPrice", "operation=getETFPriceInfo", "status=502"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

func TestUnsupportedSecurityTypeIsNotHiddenAsEmptySuccess(t *testing.T) {
	p := NewWithClient(nil)
	_, err := p.FetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityType("bond"),
		Symbol:       "KRW-BOND",
	})
	if err == nil {
		t.Fatal("fetch unsupported security type error = nil, want unsupported error")
	}
	var unsupported *provider.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error type = %T, want UnsupportedError: %v", err, err)
	}
	for _, want := range []string{"provider=datago", "group=securitiesProductPrice", "security_type=bond", "symbol=KRW-BOND"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

func TestRouterReportsUnsupportedQuotePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL)
	registry := provider.NewRegistry()
	if err := Register(registry, p); err != nil {
		t.Fatalf("register datago provider: %v", err)
	}

	router := quote.NewRouter(provider.NewRouter(registry))
	_, err := router.RouteQuoteSnapshot(context.Background(), quote.RouteInput{
		ProviderID:   provider.ProviderDataGo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err == nil {
		t.Fatal("route quote error = nil, want no provider")
	}
	if !provider.IsNoProvider(err) {
		t.Fatalf("route quote error = %v, want ErrNoProvider", err)
	}
	if !strings.Contains(err.Error(), "role=quote_snapshot") {
		t.Fatalf("error should include role context: %v", err)
	}
}

func TestProviderRegistersGroupRoleEntries(t *testing.T) {
	p := newTestProvider(t, "http://example.invalid")
	registrations := p.RoleRegistrations()
	if len(registrations) != 4 {
		t.Fatalf("registrations len = %d, want 4", len(registrations))
	}

	roles := make(map[provider.GroupID]map[provider.Role]bool)
	for _, registration := range registrations {
		group := registration.Profile.Group
		if roles[group] == nil {
			roles[group] = map[provider.Role]bool{}
		}
		roles[group][registration.Profile.Role] = true
	}
	for _, group := range []provider.GroupID{provider.GroupSecuritiesProductPrice, provider.GroupStockPrice} {
		for _, role := range []provider.Role{provider.RoleDailyBar, provider.RoleInstrument} {
			if !roles[group][role] {
				t.Fatalf("group %s missing role %s in registrations: %+v", group, role, registrations)
			}
		}
	}

	registry := provider.NewRegistry()
	if err := Register(registry, p); err != nil {
		t.Fatalf("register datago provider: %v", err)
	}
	if len(registry.Entries(provider.RoleDailyBar)) != 2 {
		t.Fatalf("dailybar entries len = %d, want 2", len(registry.Entries(provider.RoleDailyBar)))
	}
	if len(registry.Entries(provider.RoleInstrument)) != 2 {
		t.Fatalf("instrument entries len = %d, want 2", len(registry.Entries(provider.RoleInstrument)))
	}
}

func newTestProvider(t *testing.T, baseURL string) *Provider {
	t.Helper()
	p, err := New(Config{
		ServiceKey: "test-key",
		BaseURL:    baseURL,
		StockPrice: GroupConfig{
			ServiceKey: "test-key",
			BaseURL:    baseURL,
		},
		RetryMaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("new datago provider: %v", err)
	}
	return p
}

func newTestCorporateFinanceProvider(t *testing.T, baseURL string) *CorporateFinanceProvider {
	t.Helper()
	p, err := NewCorporateFinance(CorporateFinanceConfig{
		ServiceKey:              "test-key",
		CorporateFinanceBaseURL: baseURL,
		KRXListedInfoServiceKey: "test-key",
		KRXListedInfoBaseURL:    baseURL,
		RetryMaxAttempts:        1,
	})
	if err != nil {
		t.Fatalf("new datago corporate finance provider: %v", err)
	}
	return p
}

func assertCommonQuery(t *testing.T, r *http.Request, pageNo string, numOfRows string) {
	t.Helper()
	if got := r.URL.Query().Get("serviceKey"); got != "test-key" {
		t.Fatalf("serviceKey = %q, want test-key", got)
	}
	if got := r.URL.Query().Get("pageNo"); got != pageNo {
		t.Fatalf("pageNo = %q, want %s", got, pageNo)
	}
	if got := r.URL.Query().Get("numOfRows"); got != numOfRows {
		t.Fatalf("numOfRows = %q, want %s", got, numOfRows)
	}
}

func instrumentInput(query string) instrument.SearchInput {
	return instrument.SearchInput{
		Market: provider.MarketKRX,
		Query:  query,
	}
}

type stubFinancialClient struct{}

func (stubFinancialClient) GetAllSummaryFinancialStatements(context.Context, datagocorpfin.Query) (datagocorpfin.SummaryFinancialStatementResult, error) {
	return datagocorpfin.SummaryFinancialStatementResult{}, nil
}

func (stubFinancialClient) GetAllBalanceSheets(context.Context, datagocorpfin.Query) (datagocorpfin.BalanceSheetResult, error) {
	return datagocorpfin.BalanceSheetResult{}, nil
}

func (stubFinancialClient) GetAllIncomeStatements(context.Context, datagocorpfin.Query) (datagocorpfin.IncomeStatementResult, error) {
	return datagocorpfin.IncomeStatementResult{}, nil
}
