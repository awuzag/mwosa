package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ev3rlit/mwosa/storage"
)

func TestGetKRXRawAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/etp/etf_bydd_trd" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("AUTH_KEY"); got != "test-key" {
			t.Fatalf("AUTH_KEY = %q, want test-key", got)
		}
		fmt.Fprint(w, `{"OutBlock_1":[{"BAS_DD":"20240415","ISU_CD":"069500","ISU_NM":"KODEX 200","TDD_CLSPRC":"35120"}]}`)
	}))
	defer server.Close()

	t.Setenv("MWOSA_KRX_AUTH_KEY", "test-key")
	t.Setenv("MWOSA_KRX_BASE_URL", server.URL)

	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"--output", "json",
		"get", "krx", "etf_bydd_trd",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("get krx: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider": "krx"`, `"api_id": "etf_bydd_trd"`, `"row_count": 1`, `"ISU_CD": "069500"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q in:\n%s", want, out.String())
		}
	}
}

func TestListKRXAPIsIncludesAllServices(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--output", "json",
		"list", "krx-apis",
	); err != nil {
		t.Fatalf("list krx-apis: %v\n%s", err, out.String())
	}
	if got := strings.Count(out.String(), `"api_id"`); got != 31 {
		t.Fatalf("api_id count = %d, want 31\n%s", got, out.String())
	}
}

func TestSyncKRXStoresRawSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/idx/krx_dd_trd" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"OutBlock_1":[{"BAS_DD":"20240415","IDX_NM":"KRX 300","CLSPRC_IDX":"1700.11"}]}`)
	}))
	defer server.Close()

	t.Setenv("MWOSA_KRX_AUTH_KEY", "test-key")
	t.Setenv("MWOSA_KRX_BASE_URL", server.URL)

	databasePath := filepath.Join(t.TempDir(), "mwosa.db")
	var out bytes.Buffer
	cmd := NewRootCommand(BuildInfo{})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := executeForTest(t, context.Background(), cmd,
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"--database", databasePath,
		"--output", "json",
		"sync", "krx", "krx_dd_trd",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("sync krx: %v\n%s", err, out.String())
	}
	for _, want := range []string{`"provider": "krx"`, `"api_id": "krx_dd_trd"`, `"row_count": 1`, `"rows_affected": 1`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q in:\n%s", want, out.String())
		}
	}

	database := storage.NewDatabase(databasePath)
	defer func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	}()
	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var row storage.ProviderRawSnapshotRow
	if err := client.NewSelect().Model(&row).Where("operation = ?", "krx_dd_trd").Scan(context.Background()); err != nil {
		t.Fatalf("select raw snapshot: %v", err)
	}
	if row.RowCount != 1 || !strings.Contains(row.PayloadJSON, "KRX 300") {
		t.Fatalf("unexpected raw snapshot row: %+v", row)
	}
}
