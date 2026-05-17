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
	for _, want := range []string{`"provider": "opendart"`, `"operation": "corpCode"`, `"total_count": 1`, `"listed_count": 1`, `"companies_written": 1`, `"identifiers_written": 2`, `"instruments_linked": 1`} {
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

	var identifiersOut bytes.Buffer
	identifiersCmd := NewRootCommand(BuildInfo{})
	identifiersCmd.SetOut(&identifiersOut)
	identifiersCmd.SetErr(&identifiersOut)
	if err := executeForTest(t, context.Background(), identifiersCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"get", "company-identifiers", "005930",
	); err != nil {
		t.Fatalf("get company-identifiers: %v\n%s", err, identifiersOut.String())
	}
	for _, want := range []string{`"identifier_type": "dart_corp_code"`, `"identifier_value": "00126380"`, `"identifier_type": "krx_stock_code"`, `"identifier_value": "005930"`} {
		if !strings.Contains(identifiersOut.String(), want) {
			t.Fatalf("company identifiers output missing %q in:\n%s", want, identifiersOut.String())
		}
	}

	var inspectOut bytes.Buffer
	inspectCmd := NewRootCommand(BuildInfo{})
	inspectCmd.SetOut(&inspectOut)
	inspectCmd.SetErr(&inspectOut)
	if err := executeForTest(t, context.Background(), inspectCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"inspect", "company", "005930",
	); err != nil {
		t.Fatalf("inspect company: %v\n%s", err, inspectOut.String())
	}
	for _, want := range []string{`"company"`, `"name": "삼성전자"`, `"instruments"`, `"relation_type": "issuer"`} {
		if !strings.Contains(inspectOut.String(), want) {
			t.Fatalf("inspect company output missing %q in:\n%s", want, inspectOut.String())
		}
	}
}

func TestSyncOpenDARTFinancialFacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/corpCode.xml":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testOpenDARTCorpCodeZIP(t))
		case "/api/alotMatter.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			if got := r.URL.Query().Get("bsns_year"); got != "2025" {
				t.Fatalf("bsns_year = %q, want 2025", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","se":"현금배당금총액","stock_knd":"보통주","thstrm":"9,809,437,000,000","frmtrm":"9,800,000,000,000","lwfr":"-","stlm_dt":"2025-12-31","rcept_no":"20260330000001"}]}`)
		case "/api/tesstkAcqsDspsSttus.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","stock_knd":"보통주","acqs_mth1":"총계","acqs_mth2":"직접취득","acqs_mth3":"소계","bsis_qy":"1,000","change_qy_acqs":"200","change_qy_dsps":"50","change_qy_incnr":"10","trmend_qy":"1,140","stlm_dt":"2025-12-31","rcept_no":"20260330000002"}]}`)
		case "/api/hyslrSttus.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","nm":"테스트 최대주주","relate":"본인","stock_knd":"보통주","bsis_posesn_stock_co":"100,000","bsis_posesn_stock_qota_rt":"12.30","trmend_posesn_stock_co":"110,000","trmend_posesn_stock_qota_rt":"13.20","stlm_dt":"2025-12-31","rcept_no":"20260330000003"}]}`)
		case "/api/hyslrChgSttus.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","mxmm_shrholdr_nm":"테스트 최대주주","change_cause":"장내매수","change_on":"2025.12.15","posesn_stock_co":"111,000","qota_rt":"13.30","stlm_dt":"2025-12-31","rcept_no":"20260330000004"}]}`)
		case "/api/empSttus.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","fo_bbm":"DX","sexdstn":"남","sm":"50,000","rgllbr_co":"49,000","cnttk_co":"1,000","fyer_salary_totamt":"5,000,000,000","jan_salary_am":"100,000","stlm_dt":"2025-12-31","rcept_no":"20260330000005"}]}`)
		case "/api/accnutAdtorNmNdAdtOpinion.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","bsns_year":"당기","adtor":"테스트회계법인","adt_opinion":"적정","emphs_matter":"해당사항 없음","core_adt_matter":"수익 인식","stlm_dt":"2025-12-31","rcept_no":"20260330000006"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OPENDART_API_KEY", "test-key")
	t.Setenv("MWOSA_OPENDART_BASE_URL", server.URL)

	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	configPath := filepath.Join(dir, "config.json")
	var companyOut bytes.Buffer
	companyCmd := NewRootCommand(BuildInfo{})
	companyCmd.SetOut(&companyOut)
	companyCmd.SetErr(&companyOut)
	if err := executeForTest(t, context.Background(), companyCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"sync", "companies",
	); err != nil {
		t.Fatalf("sync companies --provider opendart: %v\n%s", err, companyOut.String())
	}

	var syncOut bytes.Buffer
	syncCmd := NewRootCommand(BuildInfo{})
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	if err := executeForTest(t, context.Background(), syncCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"sync", "financials", "facts", "005930",
		"--year", "2025",
	); err != nil {
		t.Fatalf("sync financials facts --provider opendart: %v\n%s", err, syncOut.String())
	}
	for _, want := range []string{`"facts_written": 22`, `"provider_group": "periodicReport"`, `"operation": "alotMatter"`, `"operation": "tesstkAcqsDspsSttus"`, `"operation": "hyslrSttus"`, `"operation": "hyslrChgSttus"`, `"operation": "empSttus"`, `"operation": "accnutAdtorNmNdAdtOpinion"`, `"total_count": 22`} {
		if !strings.Contains(syncOut.String(), want) {
			t.Fatalf("sync financials facts output missing %q in:\n%s", want, syncOut.String())
		}
	}
	if strings.Contains(syncOut.String(), "test-key") {
		t.Fatalf("sync financials facts output exposed secret:\n%s", syncOut.String())
	}

	var listOut bytes.Buffer
	listCmd := NewRootCommand(BuildInfo{})
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	if err := executeForTest(t, context.Background(), listCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"get", "financials", "facts", "005930",
		"--year", "2025",
	); err != nil {
		t.Fatalf("get financials facts: %v\n%s", err, listOut.String())
	}
	for _, want := range []string{`"fact_type": "treasury_stock"`, `"fact_type": "major_shareholder"`, `"fact_type": "major_shareholder_change"`, `"fact_type": "employee"`, `"fact_type": "audit_opinion"`, `"key": "당기:opinion"`, `"value_text": "적정"`, `"value_number": "50000"`} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("get financials facts output missing %q in:\n%s", want, listOut.String())
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

func TestSyncAndListOpenDARTEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/corpCode.xml":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testOpenDARTCorpCodeZIP(t))
		case "/api/cvbdIsDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			if got := r.URL.Query().Get("bgn_de"); got != "20260101" {
				t.Fatalf("bgn_de = %q, want 20260101", got)
			}
			if got := r.URL.Query().Get("end_de"); got != "20261231" {
				t.Fatalf("end_de = %q, want 20261231", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260115000001","bddd":"20260115","pymd":"20260201","bd_fta":"1,000,000,000","bd_intr_ex":"1.0","bd_intr_sf":"3.0","cv_prc":"70000","cv_rt":"100"}]}`)
		case "/api/dfOcr.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260102000001","dfd":"20260102","df_amt":"700,000,000","df_bnk":"테스트은행","df_cn":"당좌거래정지","df_rs":"자금 사정 악화"}]}`)
		case "/api/piicDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260105000001","ic_mthn":"주주배정후 실권주 일반공모","nstk_ostk_cnt":"100,000","fv_ps":"5000","fdpp_op":"10,000,000,000"}]}`)
		case "/api/fricDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260106000001","bddd":"20260106","nstk_ostk_cnt":"50,000","nstk_ascnt_ps_ostk":"0.1","nstk_asstd":"20260120","nstk_lstprd":"20260220"}]}`)
		case "/api/pifricDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260107000001","piic_ic_mthn":"제3자배정","piic_nstk_ostk_cnt":"20,000","piic_fdpp_op":"4,000,000,000","fric_bddd":"20260107","fric_nstk_ostk_cnt":"20,000","fric_nstk_ascnt_ps_ostk":"0.05","fric_nstk_asstd":"20260121"}]}`)
		case "/api/crDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260108000001","bddd":"20260108","cr_mth":"무상감자","cr_rs":"결손금 보전","crstk_ostk_cnt":"30,000","cr_rt_ostk":"10","cr_std":"20260201","bfcr_cpt":"100,000,000,000","atcr_cpt":"90,000,000,000"}]}`)
		case "/api/bnkMngtPcbg.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260109000001","mngt_pcbg_dd":"20260109","cfd":"20260110","mngt_int":"주채권은행","mngt_rs":"채권단 공동관리","mngt_pd":"2026-01-09~2026-12-31"}]}`)
		case "/api/lwstLg.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260110000001","lgd":"20260110","cfd":"20260111","icnm":"손해배상 청구","ac_ap":"테스트 원고","cpct":"서울중앙지방법원","rq_cn":"손해배상 청구","ft_ctp":"법적 대응"}]}`)
		case "/api/bsnInhDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260111000001","bddd":"20260111","inh_bsn":"반도체 테스트 사업","inh_bsn_mc":"테스트 사업부 양수","dlptn_cmpnm":"테스트양도인","inh_pp":"사업 확장","inh_prc":"5,000,000,000","inh_prd_inh_std":"20260211"}]}`)
		case "/api/bsnTrfDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260112000001","bddd":"20260112","trf_bsn":"비핵심 사업","trf_bsn_mc":"테스트 사업부 양도","dlptn_cmpnm":"테스트양수인","trf_pp":"사업 재편","trf_prc":"6,000,000,000","trf_prd_trf_std":"20260212"}]}`)
		case "/api/tgastInhDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260113000001","bddd":"20260113","ast_nm":"테스트 공장","ast_sen":"토지 및 건물","dlptn_cmpnm":"테스트매도인","inh_pp":"생산능력 확대","inhdtl_inhprc":"7,000,000,000","inh_prd_inh_std":"20260213"}]}`)
		case "/api/tgastTrfDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260114000001","bddd":"20260114","ast_nm":"테스트 부동산","ast_sen":"건물","dlptn_cmpnm":"테스트매수인","trf_pp":"자산 효율화","trfdtl_trfprc":"8,000,000,000","trf_prd_trf_std":"20260214"}]}`)
		case "/api/bdwtIsDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260120000001","bddd":"20260120","pymd":"20260205","bd_fta":"1,500,000,000","bd_intr_ex":"1.5","bd_intr_sf":"3.5","ex_prc":"72000","ex_rt":"100","bdwt_div_atn":"분리"}]}`)
		case "/api/exbdIsDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260125000001","bddd":"20260125","pymd":"20260210","bd_fta":"900,000,000","bd_intr_ex":"0.5","bd_intr_sf":"2.5","ex_prc":"75000","ex_rt":"100","extg":"보통주","extg_stkcnt":"12,000"}]}`)
		case "/api/cmpMgDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260126000001","bddd":"20260126","mgptncmp_cmpnm":"테스트합병대상","mg_mth":"흡수합병","mg_stn":"소규모합병","mg_pp":"경영효율화","mg_rt":"1:0.5","mgsc_mgdt":"20260301","mgsc_mgrgsprd":"20260305","nmgcmp_cmpnm":"테스트신설회사"}]}`)
		case "/api/cmpDvDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260127000001","bddd":"20260127","dv_mth":"인적분할","dv_rt":"0.8:0.2","dvfcmp_cmpnm":"테스트분할신설","atdv_excmp_cmpnm":"테스트존속회사","dv_trfbsnprt_cn":"테스트 사업부","dv_impef":"사업 전문성 강화","dvdt":"20260310","dvrgsprd":"20260312"}]}`)
		case "/api/cmpDvmgDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260128000001","bddd":"20260128","dvmg_mth":"분할합병","dvmg_rt":"1:0.3","mgptncmp_cmpnm":"테스트분할합병대상","dvfcmp_cmpnm":"테스트분할회사","atdv_excmp_cmpnm":"테스트존속회사","dvmg_impef":"조직 재편","dvmgsc_dvmgdt":"20260320","dvmgsc_dvmgctrd":"20260201"}]}`)
		case "/api/stkExtrDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260129000001","bddd":"20260129","extr_sen":"주식교환","extr_tgcmp_cmpnm":"테스트대상법인","atextr_cpcmpnm":"테스트완전모회사","extr_stn":"완전자회사화","extr_pp":"지배구조 개편","extr_rt":"1:0.2","extrsc_extrdt":"20260330","extrsc_extrctrd":"20260210"}]}`)
		case "/api/tsstkAqDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			if got := r.URL.Query().Get("bgn_de"); got != "20260101" {
				t.Fatalf("bgn_de = %q, want 20260101", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260201000001","aq_dd":"20260201","aq_pp":"주주가치 제고","aq_mth":"장내매수","aqpln_prc_ostk":"2,000,000,000","aqpln_stk_ostk":"10,000","aqexpd_bgd":"20260202","aqexpd_edd":"20260501"}]}`)
		case "/api/tsstkDpDecsn.json":
			if got := r.URL.Query().Get("corp_code"); got != "00126380" {
				t.Fatalf("corp_code = %q, want 00126380", got)
			}
			if got := r.URL.Query().Get("end_de"); got != "20261231" {
				t.Fatalf("end_de = %q, want 20261231", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"000","message":"OK","list":[{"corp_code":"00126380","corp_name":"삼성전자","rcept_no":"20260301000001","dp_dd":"20260301","dp_pp":"임직원 보상","dppln_prc_ostk":"300,000,000","dppln_stk_ostk":"1,000","dpprpd_bgd":"20260302","dpprpd_edd":"20260401"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OPENDART_API_KEY", "test-key")
	t.Setenv("MWOSA_OPENDART_BASE_URL", server.URL)

	dir := t.TempDir()
	databasePath := filepath.Join(dir, "mwosa.db")
	configPath := filepath.Join(dir, "config.json")
	var companyOut bytes.Buffer
	companyCmd := NewRootCommand(BuildInfo{})
	companyCmd.SetOut(&companyOut)
	companyCmd.SetErr(&companyOut)
	if err := executeForTest(t, context.Background(), companyCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"sync", "companies",
	); err != nil {
		t.Fatalf("sync companies --provider opendart: %v\n%s", err, companyOut.String())
	}

	var syncOut bytes.Buffer
	syncCmd := NewRootCommand(BuildInfo{})
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	if err := executeForTest(t, context.Background(), syncCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"--provider", "opendart",
		"sync", "events", "005930",
		"--from", "2026-01-01",
		"--to", "2026-12-31",
	); err != nil {
		t.Fatalf("sync events --provider opendart: %v\n%s", err, syncOut.String())
	}
	for _, want := range []string{`"events_written": 20`, `"provider_group": "materialEvents"`, `"operation": "dfOcr"`, `"operation": "piicDecsn"`, `"operation": "fricDecsn"`, `"operation": "pifricDecsn"`, `"operation": "crDecsn"`, `"operation": "bnkMngtPcbg"`, `"operation": "lwstLg"`, `"operation": "bsnInhDecsn"`, `"operation": "bsnTrfDecsn"`, `"operation": "tgastInhDecsn"`, `"operation": "tgastTrfDecsn"`, `"operation": "cvbdIsDecsn"`, `"operation": "bdwtIsDecsn"`, `"operation": "exbdIsDecsn"`, `"operation": "cmpMgDecsn"`, `"operation": "cmpDvDecsn"`, `"operation": "cmpDvmgDecsn"`, `"operation": "stkExtrDecsn"`, `"operation": "tsstkAqDecsn"`, `"operation": "tsstkDpDecsn"`, `"total_count": 20`} {
		if !strings.Contains(syncOut.String(), want) {
			t.Fatalf("sync events output missing %q in:\n%s", want, syncOut.String())
		}
	}
	if strings.Contains(syncOut.String(), "test-key") {
		t.Fatalf("sync events output exposed secret:\n%s", syncOut.String())
	}

	var listOut bytes.Buffer
	listCmd := NewRootCommand(BuildInfo{})
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	if err := executeForTest(t, context.Background(), listCmd,
		"--config", configPath,
		"--database", databasePath,
		"--output", "json",
		"list", "events", "005930",
	); err != nil {
		t.Fatalf("list events: %v\n%s", err, listOut.String())
	}
	for _, want := range []string{`"event_type": "default_occurrence"`, `"event_type": "paid_in_capital_increase"`, `"event_type": "free_capital_increase"`, `"event_type": "paid_in_free_capital_increase"`, `"event_type": "capital_reduction"`, `"event_type": "bank_management_procedure_start"`, `"event_type": "lawsuit_filing"`, `"event_type": "business_transfer_in"`, `"event_type": "business_transfer_out"`, `"event_type": "tangible_asset_transfer_in"`, `"event_type": "tangible_asset_transfer_out"`, `"event_type": "convertible_bond_issuance"`, `"event_type": "bond_with_warrant_issuance"`, `"event_type": "exchangeable_bond_issuance"`, `"event_type": "company_merger"`, `"event_type": "company_division"`, `"event_type": "company_division_merger"`, `"event_type": "stock_exchange_transfer"`, `"event_type": "treasury_stock_acquisition"`, `"event_type": "treasury_stock_disposal"`, `"event_date": "2026-01-15"`, `"event_date": "2026-01-29"`, `"rcept_no": "20260115000001"`, `"amount_minor": 700000000`, `"amount_minor": 10000000000`, `"amount_minor": 4000000000`, `"amount_minor": 5000000000`, `"amount_minor": 6000000000`, `"amount_minor": 7000000000`, `"amount_minor": 8000000000`, `"amount_minor": 1000000000`, `"amount_minor": 1500000000`, `"amount_minor": 900000000`, `"amount_minor": 2000000000`, `"amount_minor": 300000000`} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("list events output missing %q in:\n%s", want, listOut.String())
		}
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
	for _, want := range []string{`"provider_group": "disclosure"`, `"api_id": "corpCode"`, `"canonical_support": "financials"`, `"api_id": "alotMatter"`, `"canonical_support": "company_facts/dividends"`, `"api_id": "tesstkAcqsDspsSttus"`, `"canonical_support": "company_facts/treasury_stock"`, `"api_id": "hyslrSttus"`, `"canonical_support": "company_facts/major_shareholder"`, `"api_id": "hyslrChgSttus"`, `"canonical_support": "company_facts/major_shareholder_change"`, `"api_id": "empSttus"`, `"canonical_support": "company_facts/employee"`, `"api_id": "accnutAdtorNmNdAdtOpinion"`, `"canonical_support": "company_facts/audit_opinion"`, `"api_id": "dfOcr"`, `"canonical_support": "company_events/default_occurrence"`, `"api_id": "piicDecsn"`, `"canonical_support": "company_events/paid_in_capital_increase"`, `"api_id": "fricDecsn"`, `"canonical_support": "company_events/free_capital_increase"`, `"api_id": "pifricDecsn"`, `"canonical_support": "company_events/paid_in_free_capital_increase"`, `"api_id": "crDecsn"`, `"canonical_support": "company_events/capital_reduction"`, `"api_id": "bnkMngtPcbg"`, `"canonical_support": "company_events/bank_management_procedure_start"`, `"api_id": "lwstLg"`, `"canonical_support": "company_events/lawsuit_filing"`, `"api_id": "bsnInhDecsn"`, `"canonical_support": "company_events/business_transfer_in"`, `"api_id": "bsnTrfDecsn"`, `"canonical_support": "company_events/business_transfer_out"`, `"api_id": "tgastInhDecsn"`, `"canonical_support": "company_events/tangible_asset_transfer_in"`, `"api_id": "tgastTrfDecsn"`, `"canonical_support": "company_events/tangible_asset_transfer_out"`, `"api_id": "cvbdIsDecsn"`, `"canonical_support": "company_events/convertible_bond_issuance"`, `"api_id": "bdwtIsDecsn"`, `"canonical_support": "company_events/bond_with_warrant_issuance"`, `"api_id": "exbdIsDecsn"`, `"canonical_support": "company_events/exchangeable_bond_issuance"`, `"api_id": "cmpMgDecsn"`, `"canonical_support": "company_events/company_merger"`, `"api_id": "cmpDvDecsn"`, `"canonical_support": "company_events/company_division"`, `"api_id": "cmpDvmgDecsn"`, `"canonical_support": "company_events/company_division_merger"`, `"api_id": "stkExtrDecsn"`, `"canonical_support": "company_events/stock_exchange_transfer"`, `"api_id": "tsstkAqDecsn"`, `"canonical_support": "company_events/treasury_stock_acquisition"`, `"api_id": "tsstkDpDecsn"`, `"canonical_support": "company_events/treasury_stock_disposal"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("provider API output missing %q in:\n%s", want, out.String())
		}
	}
}

func TestListEventsCommandSurface(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	listEvents, _, err := cmd.Find([]string{"list", "events"})
	if err != nil {
		t.Fatalf("find list events: %v", err)
	}
	if listEvents == nil || listEvents.Use != "events <company>" {
		t.Fatalf("list events command = %#v", listEvents)
	}
	syncEvents, _, err := cmd.Find([]string{"sync", "events"})
	if err != nil {
		t.Fatalf("find sync events: %v", err)
	}
	if syncEvents == nil || syncEvents.Use != "events <company>" {
		t.Fatalf("sync events command = %#v", syncEvents)
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
