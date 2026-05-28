package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntradayBuildsKISRequestAndParsesBars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST03010200")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))
		assert.Equal(t, "100000", r.URL.Query().Get("FID_INPUT_HOUR_1"))
		assert.Equal(t, "N", r.URL.Query().Get("FID_PW_DATA_INCU_YN"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output1": {"stck_prpr": "70100"},
			"output2": [
				{
					"stck_bsop_date": "20250131",
					"stck_cntg_hour": "100000",
					"stck_prpr": "70100",
					"stck_oprc": "70000",
					"stck_hgpr": "70200",
					"stck_lwpr": "69900",
					"cntg_vol": "1234",
					"acml_tr_pbmn": "86000000"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	response, err := client.Quote().Intraday(context.Background(), InquireTimeItemChartPriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
		FidInputHour1:      "100000",
		FidPwDataIncuYn:    "N",
	})
	require.NoError(t, err)
	require.Len(t, response.Output2, 1)
	assert.Equal(t, "20250131", response.Output2[0].StckBsopDate)
	assert.Equal(t, "100000", response.Output2[0].StckCntgHour)
	assert.Equal(t, "70100", response.Output2[0].StckPrpr)
	assert.Equal(t, "1234", response.Output2[0].CntgVol)
}
