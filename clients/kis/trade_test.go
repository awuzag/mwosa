package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTradesBuildsKISRequestAndParsesRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-ccnl", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST01010300")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output": [
				{
					"stck_cntg_hour": "155955",
					"stck_prpr": "78900",
					"prdy_vrss": "900",
					"prdy_vrss_sign": "2",
					"cntg_vol": "2",
					"tday_rltv": "114.05",
					"prdy_ctrt": "1.15"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	trades, err := client.Quote().Trades(context.Background(), InquireCcnlRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
	})
	require.NoError(t, err)
	require.Len(t, trades.Output, 1)
	assert.Equal(t, "155955", trades.Output[0].StckCntgHour)
	assert.Equal(t, "78900", trades.Output[0].StckPrpr)
	assert.Equal(t, "2", trades.Output[0].CntgVol)
}

func TestTimeTradesBuildsKISRequestAndParsesRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHPST01060000")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))
		assert.Equal(t, "141200", r.URL.Query().Get("FID_INPUT_HOUR_1"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output1": {"stck_prpr": "104000"},
			"output2": {
				"stck_cntg_hour": "141159",
				"stck_pbpr": "104500",
				"askp": "105000",
				"bidp": "104500",
				"cnqn": "20",
				"acml_vol": "1979727",
				"prdy_ctrt": "-2.34",
				"prdy_vrss": "-2500",
				"prdy_vrss_sign": "5",
				"tday_rltv": "42.43"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	trades, err := client.Quote().TimeTrades(context.Background(), InquireTimeItemConclusionRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
		FidInputHour1:      "141200",
	})
	require.NoError(t, err)
	assert.Equal(t, "141159", trades.Output2.StckCntgHour)
	assert.Equal(t, "104500", trades.Output2.Bidp)
	assert.Equal(t, "20", trades.Output2.Cnqn)
}
