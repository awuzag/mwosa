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
