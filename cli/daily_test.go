package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/indexbar"
	"github.com/ev3rlit/mwosa/storage"
	indexbarstorage "github.com/ev3rlit/mwosa/storage/indexbar"
	"github.com/spf13/cobra"
)

func TestEnsureDailyFetchesBatchAndGetReadsStoredData(t *testing.T) {
	requests := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("likeSrtnCd"); got != "069500" {
			t.Fatalf("likeSrtnCd = %q, want 069500", got)
		}
		if got := r.URL.Query().Get("basDt"); got != "20240415" {
			t.Fatalf("basDt = %q, want 20240415", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 2,
				"items": {"item": [
					{"basDt": "20240415", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35120", "trqu": "10"},
					{"basDt": "20240415", "srtnCd": "123456", "itmsNm": "OTHER ETF", "clpr": "1000", "trqu": "20"}
				]}
			}
		}`)
	}))
	defer server.Close()

	t.Setenv("MWOSA_KRX_AUTH_KEY", "test-key")
	t.Setenv("MWOSA_KRX_BASE_URL", server.URL)
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	setDataGoEnv(t, server.URL)

	var ensureOut bytes.Buffer
	ensureCmd := NewRootCommand(BuildInfo{})
	ensureCmd.SetOut(&ensureOut)
	ensureCmd.SetErr(&ensureOut)
	if err := executeForTest(t, context.Background(), ensureCmd,
		"--database", databasePath,
		"--output", "json",
		"ensure", "daily", "069500",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("ensure daily: %v\n%s", err, ensureOut.String())
	}
	if len(requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(requests))
	}
	if !strings.Contains(ensureOut.String(), `"symbol": "069500"`) {
		t.Fatalf("ensure output should include requested symbol:\n%s", ensureOut.String())
	}

	var getOut bytes.Buffer
	getCmd := NewRootCommand(BuildInfo{})
	getCmd.SetOut(&getOut)
	getCmd.SetErr(&getOut)
	if err := executeForTest(t, context.Background(), getCmd,
		"--database", databasePath,
		"--output", "json",
		"get", "daily", "069500",
		"--as-of", "2024-04-15",
	); err != nil {
		t.Fatalf("get daily: %v\n%s", err, getOut.String())
	}
	if !strings.Contains(getOut.String(), `"closing_price": "35120"`) {
		t.Fatalf("get output should include stored close:\n%s", getOut.String())
	}
}

func TestGetIndexReadsStoredData(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	database := storage.NewDatabase(databasePath)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	_, writer, err := indexbarstorage.NewRepository(database)
	if err != nil {
		t.Fatalf("new index repository: %v", err)
	}
	if _, err := writer.UpsertIndexBars(context.Background(), []indexbar.Bar{{
		Provider:    provider.ProviderKRX,
		Group:       provider.GroupKRXIndexDailyTrade,
		Operation:   provider.OperationKOSPIDDTrd,
		Market:      provider.MarketKRX,
		IndexCode:   "KOSPI",
		Name:        "KOSPI",
		Family:      "KOSPI",
		TradingDate: "2024-04-15",
		Currency:    "KRW",
		Close:       "2670.43",
	}}); err != nil {
		t.Fatalf("seed index bar: %v", err)
	}

	var getOut bytes.Buffer
	getCmd := NewRootCommand(BuildInfo{})
	getCmd.SetOut(&getOut)
	getCmd.SetErr(&getOut)
	if err := executeForTest(t, context.Background(), getCmd,
		"--database", databasePath,
		"--output", "json",
		"get", "index", "KOSPI",
		"--as-of", "2024-04-15",
	); err != nil {
		t.Fatalf("get index: %v\n%s", err, getOut.String())
	}
	for _, want := range []string{`"index_code": "KOSPI"`, `"close_value": "2670.43"`, `"operation": "kospi_dd_trd"`} {
		if !strings.Contains(getOut.String(), want) {
			t.Fatalf("get output missing %q in:\n%s", want, getOut.String())
		}
	}
}

func TestEnsureDailyCanSearchByInstrumentName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("itmsNm"); got != "KODEX 200" {
			t.Fatalf("itmsNm = %q, want KODEX 200", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 100,
				"pageNo": 1,
				"totalCount": 1,
				"items": {"item": [
					{"basDt": "20240415", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35120", "trqu": "10"}
				]}
			}
		}`)
	}))
	defer server.Close()

	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	setDataGoEnv(t, server.URL)

	var ensureOut bytes.Buffer
	ensureCmd := NewRootCommand(BuildInfo{})
	ensureCmd.SetOut(&ensureOut)
	ensureCmd.SetErr(&ensureOut)
	if err := executeForTest(t, context.Background(), ensureCmd,
		"--database", databasePath,
		"--output", "json",
		"ensure", "daily", "KODEX 200",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("ensure daily: %v\n%s", err, ensureOut.String())
	}
	if !strings.Contains(ensureOut.String(), `"symbol": "069500"`) {
		t.Fatalf("ensure output should include matched symbol:\n%s", ensureOut.String())
	}
}

func TestSyncDailyCollectsAllPages(t *testing.T) {
	seenPages := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		pageNo := r.URL.Query().Get("pageNo")
		seenPages = append(seenPages, pageNo)
		if got := r.URL.Query().Get("numOfRows"); got != "1000" {
			t.Fatalf("numOfRows = %q, want 1000", got)
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
						{"basDt": "20240415", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35120", "trqu": "10"}
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
						{"basDt": "20240415", "srtnCd": "069501", "itmsNm": "KODEX Next", "clpr": "1000", "trqu": "20"}
					]}
				}
			}`)
		default:
			t.Fatalf("unexpected pageNo: %s", pageNo)
		}
	}))
	defer server.Close()

	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	setDataGoEnv(t, server.URL)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--database", databasePath,
		"--output", "json",
		"sync", "daily",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("sync daily: %v\n%s", err, out.String())
	}
	if strings.Join(seenPages, ",") != "1,2" {
		t.Fatalf("seen pages = %v, want [1 2]", seenPages)
	}
	if !strings.Contains(out.String(), `"bars_fetched": 2`) {
		t.Fatalf("sync output should summarize fetched bars:\n%s", out.String())
	}
}

func TestGetDailyMissingDataReturnsEnsureHint(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := executeForTest(t, context.Background(), cmd,
		"--database", filepath.Join(t.TempDir(), "mwosa.db"),
		"get", "daily", "069500",
		"--from", "20240101",
		"--to", "20240102",
	)
	if err == nil {
		t.Fatal("get daily error = nil, want missing data error")
	}
	for _, want := range []string{"daily data not found", "symbol=069500", "mwosa ensure daily 069500"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q in %q", want, err.Error())
		}
	}
}

func TestBackfillDailyCollectsRange(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("beginBasDt"); got != "20240415" {
			t.Fatalf("beginBasDt = %q, want 20240415", got)
		}
		if got := r.URL.Query().Get("endBasDt"); got != "20240417" {
			t.Fatalf("endBasDt = %q, want 20240417", got)
		}
		if got := r.URL.Query().Get("basDt"); got != "" {
			t.Fatalf("basDt = %q, want empty", got)
		}
		fmt.Fprint(w, `{
			"header": {"resultCode": "00", "resultMsg": "OK"},
			"body": {
				"numOfRows": 1000,
				"pageNo": 1,
				"totalCount": 2,
				"items": {"item": [
					{"basDt": "20240415", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35120"},
					{"basDt": "20240416", "srtnCd": "069500", "itmsNm": "KODEX 200", "clpr": "35200"}
				]}
			}
		}`)
	}))
	defer server.Close()

	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	setDataGoEnv(t, server.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := executeForTest(t, context.Background(), cmd,
		"--database", databasePath,
		"--output", "json",
		"backfill", "daily",
		"--from", "20240415",
		"--to", "20240416",
	); err != nil {
		t.Fatalf("backfill daily: %v\n%s", err, out.String())
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want one range request", requests)
	}
	if !strings.Contains(out.String(), `"bars_fetched": 2`) {
		t.Fatalf("backfill output should summarize fetched bars:\n%s", out.String())
	}
	if strings.Contains(out.String(), "backfill daily: fetched pages") {
		t.Fatalf("progress should not be written to stdout:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "backfill daily: fetched pages 1/1") {
		t.Fatalf("progress should be written to stderr:\n%s", errOut.String())
	}
}

func TestBackfillDailyUsesWorkersForPages(t *testing.T) {
	seenPages := make(map[string]bool)
	var seenMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getETFPriceInfo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("beginBasDt"); got != "20240415" {
			t.Fatalf("beginBasDt = %q, want 20240415", got)
		}
		if got := r.URL.Query().Get("endBasDt"); got != "20240417" {
			t.Fatalf("endBasDt = %q, want 20240417", got)
		}
		pageNo := r.URL.Query().Get("pageNo")
		seenMu.Lock()
		seenPages[pageNo] = true
		seenMu.Unlock()
		switch pageNo {
		case "1":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 1,
					"totalCount": 2001,
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
					"totalCount": 2001,
					"items": {"item": [
						{"basDt": "20240416", "srtnCd": "069501", "itmsNm": "KODEX Next", "clpr": "1000"}
					]}
				}
			}`)
		case "3":
			fmt.Fprint(w, `{
				"header": {"resultCode": "00", "resultMsg": "OK"},
				"body": {
					"numOfRows": 1000,
					"pageNo": 3,
					"totalCount": 2001,
					"items": {"item": [
						{"basDt": "20240416", "srtnCd": "069502", "itmsNm": "KODEX Last", "clpr": "1001"}
					]}
				}
			}`)
		default:
			t.Fatalf("unexpected pageNo: %s", pageNo)
		}
	}))
	defer server.Close()

	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	setDataGoEnv(t, server.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := executeForTest(t, context.Background(), cmd,
		"--database", databasePath,
		"--output", "json",
		"backfill", "daily",
		"--from", "20240415",
		"--to", "20240416",
		"--workers", "2",
	); err != nil {
		t.Fatalf("backfill daily: %v\n%s", err, out.String())
	}
	seenMu.Lock()
	defer seenMu.Unlock()
	if !seenPages["1"] || !seenPages["2"] || !seenPages["3"] {
		t.Fatalf("seen pages = %v, want pages 1, 2, and 3", seenPages)
	}
	if !strings.Contains(out.String(), `"bars_fetched": 3`) {
		t.Fatalf("backfill output should summarize fetched bars:\n%s", out.String())
	}
	if strings.Contains(out.String(), "backfill daily: fetched pages") {
		t.Fatalf("progress should not be written to stdout:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "backfill daily: fetched pages") {
		t.Fatalf("progress should be written to stderr:\n%s", errOut.String())
	}
}

func setDataGoEnv(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("MWOSA_DATAGO_SERVICE_KEY", "test-key")
	t.Setenv("MWOSA_DATAGO_BASE_URL", baseURL)
}

func executeForTest(t *testing.T, ctx context.Context, cmd *cobra.Command, args ...string) error {
	t.Helper()
	if !hasConfigFlag(args) {
		args = append([]string{"--config", filepath.Join(t.TempDir(), "config.json")}, args...)
	}
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

func hasConfigFlag(args []string) bool {
	for index, arg := range args {
		if arg == "--config" || strings.HasPrefix(arg, "--config=") {
			return true
		}
		if index > 0 && args[index-1] == "--config" {
			return true
		}
	}
	return false
}
