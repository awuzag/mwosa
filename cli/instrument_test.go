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
)

func TestSyncSearchAndInspectKRXInstrumentMaster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if got := r.URL.Query().Get("basDd"); got != "20240415" {
			t.Fatalf("basDd = %q, want 20240415", got)
		}
		switch r.URL.Path {
		case "/sto/stk_isu_base_info":
			fmt.Fprint(w, `{"OutBlock_1":[{"ISU_CD":"KR7005930003","ISU_SRT_CD":"005930","ISU_NM":"삼성전자","ISU_ABBRV":"삼성전자","ISU_ENG_NM":"Samsung Electronics","LIST_DD":"19750611","MKT_TP_NM":"KOSPI","SECUGRP_NM":"주권","SECT_TP_NM":"일반","KIND_STKCERT_TP_NM":"보통주","PARVAL":"100","LIST_SHRS":"5969782550"}]}`)
		case "/sto/ksq_isu_base_info", "/sto/knx_isu_base_info":
			fmt.Fprint(w, `{"OutBlock_1":[]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("MWOSA_KRX_AUTH_KEY", "test-key")
	t.Setenv("MWOSA_KRX_BASE_URL", server.URL)
	databasePath := filepath.Join(t.TempDir(), "mwosa.db")

	var syncOut bytes.Buffer
	syncCmd := NewRootCommand(BuildInfo{})
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	if err := executeForTest(t, context.Background(), syncCmd,
		"--database", databasePath,
		"--provider", "krx",
		"--output", "json",
		"sync", "instruments",
		"--as-of", "20240415",
	); err != nil {
		t.Fatalf("sync instruments: %v\n%s", err, syncOut.String())
	}
	if !strings.Contains(syncOut.String(), `"instruments_fetched": 1`) {
		t.Fatalf("sync output should include fetched count:\n%s", syncOut.String())
	}

	var searchOut bytes.Buffer
	searchCmd := NewRootCommand(BuildInfo{})
	searchCmd.SetOut(&searchOut)
	searchCmd.SetErr(&searchOut)
	if err := executeForTest(t, context.Background(), searchCmd,
		"--database", databasePath,
		"--output", "json",
		"search", "instruments", "Samsung",
	); err != nil {
		t.Fatalf("search instruments: %v\n%s", err, searchOut.String())
	}
	if !strings.Contains(searchOut.String(), `"security_code": "005930"`) {
		t.Fatalf("search output should include local instrument:\n%s", searchOut.String())
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, context.Background(), inspectCmd,
		"--database", databasePath,
		"--output", "json",
		"inspect", "instrument", "005930",
	); err != nil {
		t.Fatalf("inspect instrument: %v\n%s", err, inspectOut.String())
	}
	for _, want := range []string{`"security_code": "005930"`, `"issueEnglishName": "Samsung Electronics"`, `"listingDate": "19750611"`} {
		if !strings.Contains(inspectOut.String(), want) {
			t.Fatalf("inspect output missing %q in:\n%s", want, inspectOut.String())
		}
	}
}
