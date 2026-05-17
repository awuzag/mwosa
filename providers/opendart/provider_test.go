package opendart

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	opendartsdk "github.com/ev3rlit/opendart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	corpCodeBody    []byte
	listParams      opendartsdk.ListParams
	financialParams opendartsdk.FnlttSinglAcntAllParams
	alotParams      opendartsdk.AlotMatterParams
	tesstkParams    opendartsdk.TesstkAcqsDspsSttusParams
	hyslrParams     opendartsdk.HyslrSttusParams
	hyslrChgParams  opendartsdk.HyslrChgSttusParams
	empParams       opendartsdk.EmpSttusParams
	auditParams     opendartsdk.AccnutAdtorNmNdAdtOpinionParams
	dfOcrParams     opendartsdk.DfOcrParams
	piicParams      opendartsdk.PiicDecsnParams
	fricParams      opendartsdk.FricDecsnParams
	pifricParams    opendartsdk.PifricDecsnParams
	crParams        opendartsdk.CrDecsnParams
	bnkMngtParams   opendartsdk.BnkMngtPcbgParams
	lwstLgParams    opendartsdk.LwstLgParams
	bsnInhParams    opendartsdk.BsnInhDecsnParams
	bsnTrfParams    opendartsdk.BsnTrfDecsnParams
	tgastInhParams  opendartsdk.TgastInhDecsnParams
	tgastTrfParams  opendartsdk.TgastTrfDecsnParams
	cvbdParams      opendartsdk.CvbdIsDecsnParams
	bdwtParams      opendartsdk.BdwtIsDecsnParams
	exbdParams      opendartsdk.ExbdIsDecsnParams
	cmpMgParams     opendartsdk.CmpMgDecsnParams
	cmpDvParams     opendartsdk.CmpDvDecsnParams
	cmpDvmgParams   opendartsdk.CmpDvmgDecsnParams
	stkExtrParams   opendartsdk.StkExtrDecsnParams
	tsstkAqParams   opendartsdk.TsstkAqDecsnParams
	tsstkDpParams   opendartsdk.TsstkDpDecsnParams
}

func TestServiceCatalogExposesCanonicalSupport(t *testing.T) {
	catalog := ServiceCatalog()
	require.Len(t, catalog, 29)

	byOperation := make(map[provider.OperationID]CatalogService, len(catalog))
	for _, service := range catalog {
		byOperation[service.Operation] = service
	}

	require.Equal(t, "company_registry", byOperation[provider.OperationOpenDARTCorpCode].CanonicalSupport)
	require.Equal(t, "financials", byOperation[provider.OperationOpenDARTSinglAcntAll].CanonicalSupport)
	require.Equal(t, "company_facts/audit_opinion", byOperation[provider.OperationOpenDARTAuditOpinion].CanonicalSupport)
	require.Equal(t, "company_events/company_merger", byOperation[provider.OperationOpenDARTCmpMgDecsn].CanonicalSupport)
}

func (f *fakeClient) CorpCode(context.Context) (*opendartsdk.FileResponse, error) {
	return &opendartsdk.FileResponse{ContentType: "application/zip", Body: f.corpCodeBody}, nil
}

func (f *fakeClient) List(_ context.Context, params opendartsdk.ListParams) (*opendartsdk.ListResponse, error) {
	f.listParams = params
	return &opendartsdk.ListResponse{
		Status:     "000",
		Message:    "OK",
		TotalCount: "1",
		List: []opendartsdk.ListItem{
			{
				CorpCode:  params.CorpCode,
				CorpName:  "삼성전자",
				StockCode: "005930",
				ReportNm:  "사업보고서",
				RceptNo:   "20260330000001",
				RceptDt:   "20260330",
			},
		},
	}, nil
}

func (f *fakeClient) FnlttSinglAcntAll(_ context.Context, params opendartsdk.FnlttSinglAcntAllParams) (*opendartsdk.FnlttSinglAcntAllResponse, error) {
	f.financialParams = params
	return &opendartsdk.FnlttSinglAcntAllResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.FnlttSinglAcntAllItem{
			{
				CorpCode:     params.CorpCode,
				BsnsYear:     params.BsnsYear,
				ReprtCode:    params.ReprtCode,
				SjDiv:        "BS",
				SjNm:         "재무상태표",
				AccountId:    "ifrs-full_Assets",
				AccountNm:    "자산총계",
				ThstrmAmount: "1000",
				Currency:     "KRW",
				RceptNo:      "20260330000001",
			},
		},
	}, nil
}

func (f *fakeClient) AlotMatter(_ context.Context, params opendartsdk.AlotMatterParams) (*opendartsdk.AlotMatterResponse, error) {
	f.alotParams = params
	return &opendartsdk.AlotMatterResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.AlotMatterItem{
			{
				CorpCode: "00126380",
				CorpName: "삼성전자",
				Se:       "현금배당금총액",
				StockKnd: "보통주",
				Thstrm:   "9,809,437,000,000",
				Frmtrm:   "9,800,000,000,000",
				Lwfr:     "-",
				StlmDt:   "2025-12-31",
				RceptNo:  "20260330000001",
			},
		},
	}, nil
}

func (f *fakeClient) TesstkAcqsDspsSttus(_ context.Context, params opendartsdk.TesstkAcqsDspsSttusParams) (*opendartsdk.TesstkAcqsDspsSttusResponse, error) {
	f.tesstkParams = params
	return &opendartsdk.TesstkAcqsDspsSttusResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.TesstkAcqsDspsSttusItem{
			{
				CorpCode:      "00126380",
				CorpName:      "삼성전자",
				StockKnd:      "보통주",
				AcqsMth1:      "총계",
				AcqsMth2:      "직접취득",
				AcqsMth3:      "소계",
				BsisQy:        "1,000",
				ChangeQyAcqs:  "200",
				ChangeQyDsps:  "50",
				ChangeQyIncnr: "10",
				TrmendQy:      "1,140",
				StlmDt:        "2025-12-31",
				RceptNo:       "20260330000002",
			},
		},
	}, nil
}

func (f *fakeClient) HyslrSttus(_ context.Context, params opendartsdk.HyslrSttusParams) (*opendartsdk.HyslrSttusResponse, error) {
	f.hyslrParams = params
	return &opendartsdk.HyslrSttusResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.HyslrSttusItem{
			{
				CorpCode:                "00126380",
				CorpName:                "삼성전자",
				Nm:                      "테스트 최대주주",
				Relate:                  "본인",
				StockKnd:                "보통주",
				BsisPosesnStockCo:       "100,000",
				BsisPosesnStockQotaRt:   "12.30",
				TrmendPosesnStockCo:     "110,000",
				TrmendPosesnStockQotaRt: "13.20",
				StlmDt:                  "2025-12-31",
				RceptNo:                 "20260330000003",
			},
		},
	}, nil
}

func (f *fakeClient) HyslrChgSttus(_ context.Context, params opendartsdk.HyslrChgSttusParams) (*opendartsdk.HyslrChgSttusResponse, error) {
	f.hyslrChgParams = params
	return &opendartsdk.HyslrChgSttusResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.HyslrChgSttusItem{
			{
				CorpCode:       "00126380",
				CorpName:       "삼성전자",
				MxmmShrholdrNm: "테스트 최대주주",
				ChangeCause:    "장내매수",
				ChangeOn:       "2025.12.15",
				PosesnStockCo:  "111,000",
				QotaRt:         "13.30",
				StlmDt:         "2025-12-31",
				RceptNo:        "20260330000004",
			},
		},
	}, nil
}

func (f *fakeClient) EmpSttus(_ context.Context, params opendartsdk.EmpSttusParams) (*opendartsdk.EmpSttusResponse, error) {
	f.empParams = params
	return &opendartsdk.EmpSttusResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.EmpSttusItem{
			{
				CorpCode:         "00126380",
				CorpName:         "삼성전자",
				FoBbm:            "DX",
				Sexdstn:          "남",
				Sm:               "50,000",
				RgllbrCo:         "49,000",
				CnttkCo:          "1,000",
				FyerSalaryTotamt: "5,000,000,000",
				JanSalaryAm:      "100,000",
				StlmDt:           "2025-12-31",
				RceptNo:          "20260330000005",
			},
		},
	}, nil
}

func (f *fakeClient) AccnutAdtorNmNdAdtOpinion(_ context.Context, params opendartsdk.AccnutAdtorNmNdAdtOpinionParams) (*opendartsdk.AccnutAdtorNmNdAdtOpinionResponse, error) {
	f.auditParams = params
	return &opendartsdk.AccnutAdtorNmNdAdtOpinionResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.AccnutAdtorNmNdAdtOpinionItem{
			{
				CorpCode:      "00126380",
				CorpName:      "삼성전자",
				BsnsYear:      "당기",
				Adtor:         "테스트회계법인",
				AdtOpinion:    "적정",
				EmphsMatter:   "해당사항 없음",
				CoreAdtMatter: "수익 인식",
				StlmDt:        "2025-12-31",
				RceptNo:       "20260330000006",
			},
		},
	}, nil
}

func (f *fakeClient) DfOcr(_ context.Context, params opendartsdk.DfOcrParams) (*opendartsdk.DfOcrResponse, error) {
	f.dfOcrParams = params
	return &opendartsdk.DfOcrResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.DfOcrItem{
			{
				CorpCode: "00126380",
				CorpName: "삼성전자",
				RceptNo:  "20260102000001",
				Dfd:      "20260102",
				DfAmt:    "700,000,000",
				DfBnk:    "테스트은행",
				DfCn:     "당좌거래정지",
				DfRs:     "자금 사정 악화",
			},
		},
	}, nil
}

func (f *fakeClient) PiicDecsn(_ context.Context, params opendartsdk.PiicDecsnParams) (*opendartsdk.PiicDecsnResponse, error) {
	f.piicParams = params
	return &opendartsdk.PiicDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.PiicDecsnItem{
			{
				CorpCode:    "00126380",
				CorpName:    "삼성전자",
				RceptNo:     "20260105000001",
				IcMthn:      "주주배정후 실권주 일반공모",
				NstkOstkCnt: "100,000",
				FvPs:        "5000",
				FdppOp:      "10,000,000,000",
			},
		},
	}, nil
}

func (f *fakeClient) FricDecsn(_ context.Context, params opendartsdk.FricDecsnParams) (*opendartsdk.FricDecsnResponse, error) {
	f.fricParams = params
	return &opendartsdk.FricDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.FricDecsnItem{
			{
				CorpCode:        "00126380",
				CorpName:        "삼성전자",
				RceptNo:         "20260106000001",
				Bddd:            "20260106",
				NstkOstkCnt:     "50,000",
				NstkAscntPsOstk: "0.1",
				NstkAsstd:       "20260120",
				NstkLstprd:      "20260220",
			},
		},
	}, nil
}

func (f *fakeClient) PifricDecsn(_ context.Context, params opendartsdk.PifricDecsnParams) (*opendartsdk.PifricDecsnResponse, error) {
	f.pifricParams = params
	return &opendartsdk.PifricDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.PifricDecsnItem{
			{
				CorpCode:            "00126380",
				CorpName:            "삼성전자",
				RceptNo:             "20260107000001",
				PiicIcMthn:          "제3자배정",
				PiicNstkOstkCnt:     "20,000",
				PiicFdppOp:          "4,000,000,000",
				FricBddd:            "20260107",
				FricNstkOstkCnt:     "20,000",
				FricNstkAscntPsOstk: "0.05",
				FricNstkAsstd:       "20260121",
			},
		},
	}, nil
}

func (f *fakeClient) CrDecsn(_ context.Context, params opendartsdk.CrDecsnParams) (*opendartsdk.CrDecsnResponse, error) {
	f.crParams = params
	return &opendartsdk.CrDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.CrDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260108000001",
				Bddd:         "20260108",
				CrMth:        "무상감자",
				CrRs:         "결손금 보전",
				CrstkOstkCnt: "30,000",
				CrRtOstk:     "10",
				CrStd:        "20260201",
				BfcrCpt:      "100,000,000,000",
				AtcrCpt:      "90,000,000,000",
			},
		},
	}, nil
}

func (f *fakeClient) BnkMngtPcbg(_ context.Context, params opendartsdk.BnkMngtPcbgParams) (*opendartsdk.BnkMngtPcbgResponse, error) {
	f.bnkMngtParams = params
	return &opendartsdk.BnkMngtPcbgResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.BnkMngtPcbgItem{
			{
				CorpCode:   "00126380",
				CorpName:   "삼성전자",
				RceptNo:    "20260109000001",
				MngtPcbgDd: "20260109",
				Cfd:        "20260110",
				MngtInt:    "주채권은행",
				MngtRs:     "채권단 공동관리",
				MngtPd:     "2026-01-09~2026-12-31",
			},
		},
	}, nil
}

func (f *fakeClient) LwstLg(_ context.Context, params opendartsdk.LwstLgParams) (*opendartsdk.LwstLgResponse, error) {
	f.lwstLgParams = params
	return &opendartsdk.LwstLgResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.LwstLgItem{
			{
				CorpCode: "00126380",
				CorpName: "삼성전자",
				RceptNo:  "20260110000001",
				Lgd:      "20260110",
				Cfd:      "20260111",
				Icnm:     "손해배상 청구",
				AcAp:     "테스트 원고",
				Cpct:     "서울중앙지방법원",
				RqCn:     "손해배상 청구",
				FtCtp:    "법적 대응",
			},
		},
	}, nil
}

func (f *fakeClient) BsnInhDecsn(_ context.Context, params opendartsdk.BsnInhDecsnParams) (*opendartsdk.BsnInhDecsnResponse, error) {
	f.bsnInhParams = params
	return &opendartsdk.BsnInhDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.BsnInhDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260111000001",
				Bddd:         "20260111",
				InhBsn:       "반도체 테스트 사업",
				InhBsnMc:     "테스트 사업부 양수",
				DlptnCmpnm:   "테스트양도인",
				InhPp:        "사업 확장",
				InhPrc:       "5,000,000,000",
				InhPrdInhStd: "20260211",
			},
		},
	}, nil
}

func (f *fakeClient) BsnTrfDecsn(_ context.Context, params opendartsdk.BsnTrfDecsnParams) (*opendartsdk.BsnTrfDecsnResponse, error) {
	f.bsnTrfParams = params
	return &opendartsdk.BsnTrfDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.BsnTrfDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260112000001",
				Bddd:         "20260112",
				TrfBsn:       "비핵심 사업",
				TrfBsnMc:     "테스트 사업부 양도",
				DlptnCmpnm:   "테스트양수인",
				TrfPp:        "사업 재편",
				TrfPrc:       "6,000,000,000",
				TrfPrdTrfStd: "20260212",
			},
		},
	}, nil
}

func (f *fakeClient) TgastInhDecsn(_ context.Context, params opendartsdk.TgastInhDecsnParams) (*opendartsdk.TgastInhDecsnResponse, error) {
	f.tgastInhParams = params
	return &opendartsdk.TgastInhDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.TgastInhDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260113000001",
				Bddd:         "20260113",
				AstNm:        "테스트 공장",
				AstSen:       "토지 및 건물",
				DlptnCmpnm:   "테스트매도인",
				InhPp:        "생산능력 확대",
				InhdtlInhprc: "7,000,000,000",
				InhPrdInhStd: "20260213",
			},
		},
	}, nil
}

func (f *fakeClient) TgastTrfDecsn(_ context.Context, params opendartsdk.TgastTrfDecsnParams) (*opendartsdk.TgastTrfDecsnResponse, error) {
	f.tgastTrfParams = params
	return &opendartsdk.TgastTrfDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.TgastTrfDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260114000001",
				Bddd:         "20260114",
				AstNm:        "테스트 부동산",
				AstSen:       "건물",
				DlptnCmpnm:   "테스트매수인",
				TrfPp:        "자산 효율화",
				TrfdtlTrfprc: "8,000,000,000",
				TrfPrdTrfStd: "20260214",
			},
		},
	}, nil
}

func (f *fakeClient) CvbdIsDecsn(_ context.Context, params opendartsdk.CvbdIsDecsnParams) (*opendartsdk.CvbdIsDecsnResponse, error) {
	f.cvbdParams = params
	return &opendartsdk.CvbdIsDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.CvbdIsDecsnItem{
			{
				CorpCode:   "00126380",
				CorpName:   "삼성전자",
				RceptNo:    "20260115000001",
				Bddd:       "20260115",
				Pymd:       "20260201",
				BdFta:      "1,000,000,000",
				BdIntrEx:   "1.0",
				BdIntrSf:   "3.0",
				CvPrc:      "70000",
				CvRt:       "100",
				CvisstkKnd: "보통주",
			},
		},
	}, nil
}

func (f *fakeClient) BdwtIsDecsn(_ context.Context, params opendartsdk.BdwtIsDecsnParams) (*opendartsdk.BdwtIsDecsnResponse, error) {
	f.bdwtParams = params
	return &opendartsdk.BdwtIsDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.BdwtIsDecsnItem{
			{
				CorpCode:   "00126380",
				CorpName:   "삼성전자",
				RceptNo:    "20260120000001",
				Bddd:       "20260120",
				Pymd:       "20260205",
				BdFta:      "1,500,000,000",
				BdIntrEx:   "1.5",
				BdIntrSf:   "3.5",
				ExPrc:      "72000",
				ExRt:       "100",
				BdwtDivAtn: "분리",
			},
		},
	}, nil
}

func (f *fakeClient) ExbdIsDecsn(_ context.Context, params opendartsdk.ExbdIsDecsnParams) (*opendartsdk.ExbdIsDecsnResponse, error) {
	f.exbdParams = params
	return &opendartsdk.ExbdIsDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.ExbdIsDecsnItem{
			{
				CorpCode:   "00126380",
				CorpName:   "삼성전자",
				RceptNo:    "20260125000001",
				Bddd:       "20260125",
				Pymd:       "20260210",
				BdFta:      "900,000,000",
				BdIntrEx:   "0.5",
				BdIntrSf:   "2.5",
				ExPrc:      "75000",
				ExRt:       "100",
				Extg:       "보통주",
				ExtgStkcnt: "12,000",
			},
		},
	}, nil
}

func (f *fakeClient) CmpMgDecsn(_ context.Context, params opendartsdk.CmpMgDecsnParams) (*opendartsdk.CmpMgDecsnResponse, error) {
	f.cmpMgParams = params
	return &opendartsdk.CmpMgDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.CmpMgDecsnItem{
			{
				CorpCode:      "00126380",
				CorpName:      "삼성전자",
				RceptNo:       "20260126000001",
				Bddd:          "20260126",
				MgptncmpCmpnm: "테스트합병대상",
				MgMth:         "흡수합병",
				MgStn:         "소규모합병",
				MgPp:          "경영효율화",
				MgRt:          "1:0.5",
				MgscMgdt:      "20260301",
				MgscMgrgsprd:  "20260305",
				NmgcmpCmpnm:   "테스트신설회사",
			},
		},
	}, nil
}

func (f *fakeClient) CmpDvDecsn(_ context.Context, params opendartsdk.CmpDvDecsnParams) (*opendartsdk.CmpDvDecsnResponse, error) {
	f.cmpDvParams = params
	return &opendartsdk.CmpDvDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.CmpDvDecsnItem{
			{
				CorpCode:       "00126380",
				CorpName:       "삼성전자",
				RceptNo:        "20260127000001",
				Bddd:           "20260127",
				DvMth:          "인적분할",
				DvRt:           "0.8:0.2",
				DvfcmpCmpnm:    "테스트분할신설",
				AtdvExcmpCmpnm: "테스트존속회사",
				DvTrfbsnprtCn:  "테스트 사업부",
				DvImpef:        "사업 전문성 강화",
				Dvdt:           "20260310",
				Dvrgsprd:       "20260312",
			},
		},
	}, nil
}

func (f *fakeClient) CmpDvmgDecsn(_ context.Context, params opendartsdk.CmpDvmgDecsnParams) (*opendartsdk.CmpDvmgDecsnResponse, error) {
	f.cmpDvmgParams = params
	return &opendartsdk.CmpDvmgDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.CmpDvmgDecsnItem{
			{
				CorpCode:       "00126380",
				CorpName:       "삼성전자",
				RceptNo:        "20260128000001",
				Bddd:           "20260128",
				DvmgMth:        "분할합병",
				DvmgRt:         "1:0.3",
				MgptncmpCmpnm:  "테스트분할합병대상",
				DvfcmpCmpnm:    "테스트분할회사",
				AtdvExcmpCmpnm: "테스트존속회사",
				DvmgImpef:      "조직 재편",
				DvmgscDvmgdt:   "20260320",
				DvmgscDvmgctrd: "20260201",
			},
		},
	}, nil
}

func (f *fakeClient) StkExtrDecsn(_ context.Context, params opendartsdk.StkExtrDecsnParams) (*opendartsdk.StkExtrDecsnResponse, error) {
	f.stkExtrParams = params
	return &opendartsdk.StkExtrDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.StkExtrDecsnItem{
			{
				CorpCode:       "00126380",
				CorpName:       "삼성전자",
				RceptNo:        "20260129000001",
				Bddd:           "20260129",
				ExtrSen:        "주식교환",
				ExtrTgcmpCmpnm: "테스트대상법인",
				AtextrCpcmpnm:  "테스트완전모회사",
				ExtrStn:        "완전자회사화",
				ExtrPp:         "지배구조 개편",
				ExtrRt:         "1:0.2",
				ExtrscExtrdt:   "20260330",
				ExtrscExtrctrd: "20260210",
			},
		},
	}, nil
}

func (f *fakeClient) TsstkAqDecsn(_ context.Context, params opendartsdk.TsstkAqDecsnParams) (*opendartsdk.TsstkAqDecsnResponse, error) {
	f.tsstkAqParams = params
	return &opendartsdk.TsstkAqDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.TsstkAqDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260201000001",
				AqDd:         "20260201",
				AqPp:         "주주가치 제고",
				AqMth:        "장내매수",
				AqplnPrcOstk: "2,000,000,000",
				AqplnStkOstk: "10,000",
				AqexpdBgd:    "20260202",
				AqexpdEdd:    "20260501",
			},
		},
	}, nil
}

func (f *fakeClient) TsstkDpDecsn(_ context.Context, params opendartsdk.TsstkDpDecsnParams) (*opendartsdk.TsstkDpDecsnResponse, error) {
	f.tsstkDpParams = params
	return &opendartsdk.TsstkDpDecsnResponse{
		Status:  "000",
		Message: "OK",
		List: []opendartsdk.TsstkDpDecsnItem{
			{
				CorpCode:     "00126380",
				CorpName:     "삼성전자",
				RceptNo:      "20260301000001",
				DpDd:         "20260301",
				DpPp:         "임직원 보상",
				DpplnPrcOstk: "300,000,000",
				DpplnStkOstk: "1,000",
				DpprpdBgd:    "20260302",
				DpprpdEdd:    "20260401",
			},
		},
	}, nil
}

func TestDecodeCorpCodeZIPPreservesStockCodeMapping(t *testing.T) {
	companies, err := DecodeCorpCodeZIP(testCorpCodeZIP(t))
	require.NoError(t, err)
	require.Len(t, companies, 1)
	assert.Equal(t, "00126380", companies[0].CorpCode)
	assert.Equal(t, "005930", companies[0].StockCode)
	assert.Equal(t, "SAMSUNG ELECTRONICS CO,.LTD", companies[0].CorpEngName)
	assert.Equal(t, "20240101", companies[0].ModifyDate)
}

func TestOpenDARTProviderResolvesStockCodeForFilingsAndFinancials(t *testing.T) {
	fake := &fakeClient{corpCodeBody: testCorpCodeZIP(t)}
	p := NewWithClient(fake)

	filings, err := p.FetchFilings(context.Background(), FilingRequest{
		Identifier: "005930",
		From:       "2026-01-01",
		To:         "2026-03-31",
		PageCount:  "10",
	})
	require.NoError(t, err)
	assert.Equal(t, "00126380", fake.listParams.CorpCode)
	assert.Equal(t, "20260101", fake.listParams.BgnDe)
	assert.Equal(t, "20260331", fake.listParams.EndDe)
	assert.Equal(t, "005930", filings.StockCode)
	assert.Len(t, filings.Items, 1)

	result, err := p.FetchFinancialStatements(context.Background(), financials.FetchInput{
		Symbol:     "005930",
		FiscalYear: "2025",
		Period:     financials.PeriodTypeAnnual,
	})
	require.NoError(t, err)
	assert.Equal(t, "00126380", fake.financialParams.CorpCode)
	assert.Equal(t, "2025", fake.financialParams.BsnsYear)
	assert.Equal(t, reportCodeAnnual, fake.financialParams.ReprtCode)
	assert.Equal(t, fsDivConsolidated, fake.financialParams.FsDiv)
	require.Len(t, result.Statements, 1)
	assert.Equal(t, provider.ProviderOpenDART, result.Statements[0].Provider)
	assert.Equal(t, "00126380", result.Statements[0].Extensions["corp_code"])
	assert.Equal(t, "005930", result.Statements[0].Extensions["stock_code"])
}

func TestOpenDARTProviderFetchesDividendFacts(t *testing.T) {
	fake := &fakeClient{corpCodeBody: testCorpCodeZIP(t)}
	p := NewWithClient(fake)

	result, err := p.FetchDividendFacts(context.Background(), DividendFactRequest{
		CorpCode:   "00126380",
		FiscalYear: "2025",
		ReportCode: reportCodeAnnual,
	})
	require.NoError(t, err)
	assert.Equal(t, "00126380", fake.alotParams.CorpCode)
	assert.Equal(t, "2025", fake.alotParams.BsnsYear)
	assert.Equal(t, reportCodeAnnual, fake.alotParams.ReprtCode)
	require.Len(t, result.Facts, 2)
	assert.Equal(t, provider.GroupOpenDARTPeriodicReport, result.Group)
	assert.Equal(t, provider.OperationOpenDARTAlotMatter, result.Operation)
	assert.Equal(t, "dividend", result.Facts[0].FactType)
	assert.Equal(t, "thstrm:현금배당금총액:보통주", result.Facts[0].Key)
	assert.Equal(t, "9809437000000", result.Facts[0].ValueNumber)
	assert.Equal(t, "00126380", result.Facts[0].ProviderCompanyIdentifierValue)
}

func TestOpenDARTProviderFetchesPeriodicFacts(t *testing.T) {
	fake := &fakeClient{corpCodeBody: testCorpCodeZIP(t)}
	p := NewWithClient(fake)

	result, err := p.FetchPeriodicFacts(context.Background(), PeriodicFactRequest{
		CorpCode:   "00126380",
		FiscalYear: "2025",
		ReportCode: reportCodeAnnual,
	})
	require.NoError(t, err)
	assert.Equal(t, "00126380", fake.alotParams.CorpCode)
	assert.Equal(t, "2025", fake.tesstkParams.BsnsYear)
	assert.Equal(t, reportCodeAnnual, fake.hyslrParams.ReprtCode)
	assert.Equal(t, "2025", fake.hyslrChgParams.BsnsYear)
	assert.Equal(t, reportCodeAnnual, fake.empParams.ReprtCode)
	assert.Equal(t, "00126380", fake.auditParams.CorpCode)
	require.Len(t, result.Sources, 6)
	require.Len(t, result.Facts, 22)
	assert.Equal(t, provider.GroupOpenDARTPeriodicReport, result.Group)
	assert.Equal(t, provider.OperationOpenDARTAlotMatter, result.Sources[0].Operation)
	assert.Equal(t, provider.OperationOpenDARTTesstkAcqs, result.Sources[1].Operation)
	assert.Equal(t, provider.OperationOpenDARTHyslrSttus, result.Sources[2].Operation)
	assert.Equal(t, provider.OperationOpenDARTHyslrChg, result.Sources[3].Operation)
	assert.Equal(t, provider.OperationOpenDARTEmpSttus, result.Sources[4].Operation)
	assert.Equal(t, provider.OperationOpenDARTAuditOpinion, result.Sources[5].Operation)

	byKey := map[string]CompanyFact{}
	for _, fact := range result.Facts {
		byKey[fact.FactType+":"+fact.Key] = fact
	}
	assert.Equal(t, "1140", byKey["treasury_stock:보통주:총계:직접취득:소계:ending_quantity"].ValueNumber)
	assert.Equal(t, "13.20", byKey["major_shareholder:테스트 최대주주:본인:보통주:ending_ownership_ratio"].ValueNumber)
	assert.Equal(t, "13.30", byKey["major_shareholder_change:테스트 최대주주:장내매수:ownership_ratio"].ValueNumber)
	assert.Equal(t, "50000", byKey["employee:DX:남:total_count"].ValueNumber)
	assert.Equal(t, "적정", byKey["audit_opinion:당기:opinion"].ValueText)
	assert.Equal(t, "00126380", byKey["audit_opinion:당기:opinion"].ProviderCompanyIdentifierValue)
}

func TestOpenDARTProviderFetchesConvertibleBondEvents(t *testing.T) {
	fake := &fakeClient{corpCodeBody: testCorpCodeZIP(t)}
	p := NewWithClient(fake)

	result, err := p.FetchConvertibleBondEvents(context.Background(), EventRequest{
		CorpCode: "00126380",
		From:     "2026-01-01",
		To:       "2026-12-31",
	})
	require.NoError(t, err)
	assert.Equal(t, "00126380", fake.cvbdParams.CorpCode)
	assert.Equal(t, "20260101", fake.cvbdParams.BgnDe)
	assert.Equal(t, "20261231", fake.cvbdParams.EndDe)
	require.Len(t, result.Events, 1)
	event := result.Events[0]
	assert.Equal(t, provider.GroupOpenDARTMaterialEvents, result.Group)
	assert.Equal(t, provider.OperationOpenDARTCvbdIsDecsn, result.Operation)
	assert.Equal(t, "convertible_bond_issuance", event.EventType)
	assert.Equal(t, "2026-01-15", event.EventDate)
	assert.Equal(t, "20260115000001", event.RceptNo)
	require.NotNil(t, event.AmountMinor)
	assert.EqualValues(t, 1000000000, *event.AmountMinor)
	assert.Contains(t, event.ValueText, "권면총액=1,000,000,000")
	assert.Equal(t, "00126380", event.Raw["corp_code"])
}

func TestOpenDARTProviderFetchesMaterialEvents(t *testing.T) {
	fake := &fakeClient{corpCodeBody: testCorpCodeZIP(t)}
	p := NewWithClient(fake)

	result, err := p.FetchMaterialEvents(context.Background(), EventRequest{
		CorpCode: "00126380",
		From:     "2026-01-01",
		To:       "2026-12-31",
	})
	require.NoError(t, err)
	assert.Equal(t, "20260101", fake.dfOcrParams.BgnDe)
	assert.Equal(t, "20260101", fake.piicParams.BgnDe)
	assert.Equal(t, "20261231", fake.crParams.EndDe)
	assert.Equal(t, "20260101", fake.bnkMngtParams.BgnDe)
	assert.Equal(t, "20261231", fake.lwstLgParams.EndDe)
	assert.Equal(t, "20260101", fake.bsnInhParams.BgnDe)
	assert.Equal(t, "20261231", fake.tgastTrfParams.EndDe)
	assert.Equal(t, "20260101", fake.tsstkAqParams.BgnDe)
	assert.Equal(t, "20260101", fake.bdwtParams.BgnDe)
	assert.Equal(t, "20261231", fake.exbdParams.EndDe)
	assert.Equal(t, "20260101", fake.cmpMgParams.BgnDe)
	assert.Equal(t, "20261231", fake.stkExtrParams.EndDe)
	assert.Equal(t, "20261231", fake.tsstkDpParams.EndDe)
	require.Len(t, result.Sources, 20)
	require.Len(t, result.Events, 20)
	assert.Equal(t, provider.GroupOpenDARTMaterialEvents, result.Group)
	assert.Equal(t, "default_occurrence", result.Events[0].EventType)
	assert.Equal(t, provider.OperationOpenDARTDfOcr, result.Events[0].Operation)
	assert.Equal(t, "2026-01-02", result.Events[0].EventDate)
	assert.Equal(t, "paid_in_capital_increase", result.Events[1].EventType)
	assert.Equal(t, provider.OperationOpenDARTPiicDecsn, result.Events[1].Operation)
	assert.Empty(t, result.Events[1].EventDate)
	assert.Equal(t, "free_capital_increase", result.Events[2].EventType)
	assert.Equal(t, "2026-01-06", result.Events[2].EventDate)
	assert.Equal(t, "paid_in_free_capital_increase", result.Events[3].EventType)
	assert.Equal(t, provider.OperationOpenDARTPifricDecsn, result.Events[3].Operation)
	assert.Equal(t, "capital_reduction", result.Events[4].EventType)
	assert.Equal(t, provider.OperationOpenDARTCrDecsn, result.Events[4].Operation)
	assert.Equal(t, "bank_management_procedure_start", result.Events[5].EventType)
	assert.Equal(t, provider.OperationOpenDARTBnkMngtPcbg, result.Events[5].Operation)
	assert.Equal(t, "lawsuit_filing", result.Events[6].EventType)
	assert.Equal(t, provider.OperationOpenDARTLwstLg, result.Events[6].Operation)
	assert.Equal(t, "손해배상 청구", result.Events[6].Title)
	assert.Equal(t, "business_transfer_in", result.Events[7].EventType)
	assert.Equal(t, provider.OperationOpenDARTBsnInhDecsn, result.Events[7].Operation)
	assert.Equal(t, "2026-01-11", result.Events[7].EventDate)
	require.NotNil(t, result.Events[7].AmountMinor)
	assert.EqualValues(t, 5000000000, *result.Events[7].AmountMinor)
	assert.Equal(t, "business_transfer_out", result.Events[8].EventType)
	assert.Equal(t, provider.OperationOpenDARTBsnTrfDecsn, result.Events[8].Operation)
	assert.Equal(t, "tangible_asset_transfer_in", result.Events[9].EventType)
	assert.Equal(t, provider.OperationOpenDARTTgastInhDecsn, result.Events[9].Operation)
	assert.Equal(t, "tangible_asset_transfer_out", result.Events[10].EventType)
	assert.Equal(t, provider.OperationOpenDARTTgastTrfDecsn, result.Events[10].Operation)
	assert.Equal(t, "convertible_bond_issuance", result.Events[11].EventType)
	assert.Equal(t, "bond_with_warrant_issuance", result.Events[12].EventType)
	assert.Equal(t, provider.OperationOpenDARTBdwtIsDecsn, result.Events[12].Operation)
	assert.Equal(t, "exchangeable_bond_issuance", result.Events[13].EventType)
	assert.Equal(t, provider.OperationOpenDARTExbdIsDecsn, result.Events[13].Operation)
	assert.Equal(t, "company_merger", result.Events[14].EventType)
	assert.Equal(t, provider.OperationOpenDARTCmpMgDecsn, result.Events[14].Operation)
	assert.Equal(t, "회사합병 결정", result.Events[14].Title)
	assert.Equal(t, "company_division", result.Events[15].EventType)
	assert.Equal(t, provider.OperationOpenDARTCmpDvDecsn, result.Events[15].Operation)
	assert.Equal(t, "company_division_merger", result.Events[16].EventType)
	assert.Equal(t, provider.OperationOpenDARTCmpDvmgDecsn, result.Events[16].Operation)
	assert.Equal(t, "stock_exchange_transfer", result.Events[17].EventType)
	assert.Equal(t, provider.OperationOpenDARTStkExtrDecsn, result.Events[17].Operation)
	assert.Equal(t, "treasury_stock_acquisition", result.Events[18].EventType)
	assert.Equal(t, "2026-02-01", result.Events[18].EventDate)
	assert.Equal(t, provider.OperationOpenDARTTsstkAqDecsn, result.Events[18].Operation)
	require.NotNil(t, result.Events[18].AmountMinor)
	assert.EqualValues(t, 2000000000, *result.Events[18].AmountMinor)
	assert.Equal(t, "treasury_stock_disposal", result.Events[19].EventType)
	assert.Equal(t, "2026-03-01", result.Events[19].EventDate)
	assert.Equal(t, provider.OperationOpenDARTTsstkDpDecsn, result.Events[19].Operation)
}

func TestOpenDARTBuilderMissingAPIKeyDoesNotExposeSecret(t *testing.T) {
	t.Setenv("OPENDART_API_KEY", "")
	t.Setenv("MWOSA_OPENDART_API_KEY", "")

	_, err := NewBuilder().Build(provider.ConfigFromEnv())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPENDART_API_KEY")
	assert.NotContains(t, err.Error(), "test-key")
}

func testCorpCodeZIP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("CORPCODE.xml")
	require.NoError(t, err)
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
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes()
}
