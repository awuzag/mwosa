package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorProviderOpenDARTReportsMissingAuthWithoutSecret(t *testing.T) {
	t.Setenv("OPENDART_API_KEY", "")
	t.Setenv("MWOSA_OPENDART_API_KEY", "")

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"--output", "json",
		"doctor", "provider", "opendart",
	); err != nil {
		t.Fatalf("doctor provider opendart: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"id": "opendart"`, `"status": "error"`, `"secret": true`, "providers.opendart.auth.api_key"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q in:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "test-key") {
		t.Fatalf("doctor output exposed secret:\n%s", out.String())
	}
}

func TestSyncAndSearchOpenDARTCompanies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/corpCode.xml" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("crtfc_key"); got != "test-key" {
			t.Fatalf("crtfc_key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(testOpenDARTCorpCodeZIP(t))
	}))
	defer server.Close()

	t.Setenv("OPENDART_API_KEY", "test-key")
	t.Setenv("MWOSA_OPENDART_BASE_URL", server.URL)

	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	configPath := filepath.Join(dir, "config.json")
	var syncOut bytes.Buffer
	syncCmd := NewRootCommand(BuildInfo{})
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	if err := executeForTest(t, context.Background(), syncCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"sync", "companies",
	); err != nil {
		t.Fatalf("sync companies --provider opendart: %v\n%s", err, syncOut.String())
	}
	for _, want := range []string{`"provider": "opendart"`, `"operation": "corpCode"`, `"total_count": 1`, `"listed_count": 1`} {
		if !strings.Contains(syncOut.String(), want) {
			t.Fatalf("sync output missing %q in:\n%s", want, syncOut.String())
		}
	}
	if strings.Contains(syncOut.String(), "test-key") {
		t.Fatalf("sync output exposed secret:\n%s", syncOut.String())
	}

	var searchOut bytes.Buffer
	searchCmd := NewRootCommand(BuildInfo{})
	searchCmd.SetOut(&searchOut)
	searchCmd.SetErr(&searchOut)
	if err := executeForTest(t, context.Background(), searchCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"search", "companies", "005930",
	); err != nil {
		t.Fatalf("search companies --provider opendart: %v\n%s", err, searchOut.String())
	}
	for _, want := range []string{`"corp_code": "00126380"`, `"stock_code": "005930"`, `"corp_name": "삼성전자"`} {
		if !strings.Contains(searchOut.String(), want) {
			t.Fatalf("search output missing %q in:\n%s", want, searchOut.String())
		}
	}
}

func TestListOpenDARTFilingsResolvesStockCodeToCorpCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/corpCode.xml":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testOpenDARTCorpCodeZIP(t))
		case "/api/list.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			if got := r.URL.Query().Get("bgn_de"); got != "20260101" {
				t.Fatalf("bgn_de = %q, want 20260101", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","page_no":"1","page_count":"10","total_count":"1","total_page":"1","list":[{"corp_code":"00126380","corp_name":"삼성전자","stock_code":"005930","report_nm":"사업보고서","rcept_no":"20260330000001","rcept_dt":"20260330"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OPENDART_API_KEY", "test-key")
	t.Setenv("MWOSA_OPENDART_BASE_URL", server.URL)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"--output", "json",
		"--provider", "opendart",
		"list", "filings", "005930",
		"--from", "2026-01-01",
	); err != nil {
		t.Fatalf("list filings --provider opendart: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider": "opendart"`, `"corp_code": "00126380"`, `"stock_code": "005930"`, `"rcept_no": "20260330000001"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("filing output missing %q in:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "test-key") {
		t.Fatalf("filing output exposed secret:\n%s", out.String())
	}
}

func TestListProviderAPIsOpenDART(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"--output", "json",
		"list", "provider-apis", "opendart",
	); err != nil {
		t.Fatalf("list provider-apis opendart: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider_group": "disclosure"`, `"api_id": "corpCode"`, `"canonical_support": "financials"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("provider API output missing %q in:\n%s", want, out.String())
		}
	}
}

func testOpenDARTCorpCodeZIP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("CORPCODE.xml")
	if err != nil {
		t.Fatalf("create corp code XML: %v", err)
	}
	_, err = file.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<result>
  <list>
    <corp_code>00126380</corp_code>
    <corp_name>삼성전자</corp_name>
    <corp_eng_name>SAMSUNG ELECTRONICS CO,.LTD</corp_eng_name>
    <stock_code>005930</stock_code>
    <modify_date>20240101</modify_date>
  </list>
</result>`))
	if err != nil {
		t.Fatalf("write corp code XML: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
