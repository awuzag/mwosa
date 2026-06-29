//go:build integration

package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
)

func TestSyncSearchAndInspectKRXInstrumentMasterWithMongoDB(t *testing.T) {
	krxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if got := r.URL.Query().Get("basDd"); got != "20260626" {
			t.Fatalf("basDd = %q, want 20260626", got)
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
	defer krxServer.Close()

	mongoServer := integrationtest.StartMongoDB(t)
	t.Setenv("MWOSA_KRX_AUTH_KEY", "test-key")
	t.Setenv("MWOSA_KRX_BASE_URL", krxServer.URL)

	initCmd := NewRootCommand(BuildInfo{})
	var initOut bytes.Buffer
	initCmd.SetOut(&initOut)
	initCmd.SetErr(&initOut)
	initCmd.SetArgs([]string{
		"--database-url", mongoServer.URI,
		"init", "storage",
	})
	requireExecute(t, initCmd, &initOut)

	var syncOut bytes.Buffer
	syncCmd := NewRootCommand(BuildInfo{})
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	if err := executeForTest(t, context.Background(), syncCmd,
		"--database-backend", "mongodb",
		"--database-url", mongoServer.URI,
		"--provider", "krx",
		"--output", "json",
		"sync", "instruments",
		"--as-of", "20260626",
	); err != nil {
		t.Fatalf("sync instruments with mongodb: %v\n%s", err, syncOut.String())
	}
	if !strings.Contains(syncOut.String(), `"instruments_fetched": 1`) {
		t.Fatalf("sync output should include fetched count:\n%s", syncOut.String())
	}

	var searchOut bytes.Buffer
	searchCmd := NewRootCommand(BuildInfo{})
	searchCmd.SetOut(&searchOut)
	searchCmd.SetErr(&searchOut)
	if err := executeForTest(t, context.Background(), searchCmd,
		"--database-backend", "mongodb",
		"--database-url", mongoServer.URI,
		"--output", "json",
		"search", "instruments", "Samsung",
	); err != nil {
		t.Fatalf("search mongodb instruments: %v\n%s", err, searchOut.String())
	}
	if !strings.Contains(searchOut.String(), `"security_code": "005930"`) {
		t.Fatalf("search output should include mongodb instrument:\n%s", searchOut.String())
	}
}
