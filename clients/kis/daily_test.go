package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyBuildsKISRequestAndParsesBars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST03010100")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))
		assert.Equal(t, "20250101", r.URL.Query().Get("FID_INPUT_DATE_1"))
		assert.Equal(t, "20250131", r.URL.Query().Get("FID_INPUT_DATE_2"))
		assert.Equal(t, "D", r.URL.Query().Get("FID_PERIOD_DIV_CODE"))
		assert.Equal(t, "1", r.URL.Query().Get("FID_ORG_ADJ_PRC"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output1": {
				"stck_shrn_iscd": "005930",
				"hts_kor_isnm": "삼성전자"
			},
			"output2": [
				{
					"stck_bsop_date": "20250131",
					"stck_clpr": "70000",
					"stck_oprc": "69000",
					"stck_hgpr": "71000",
					"stck_lwpr": "68500",
					"acml_vol": "1000000",
					"acml_tr_pbmn": "70000000000",
					"prdy_vrss_sign": "2",
					"prdy_vrss": "1000"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	bars, err := client.Daily(context.Background(), "005930",
		WithPeriod("D"),
		WithDateRange("20250101", "20250131"),
		WithOriginalPrice(),
	)
	require.NoError(t, err)
	require.Len(t, bars, 1)
	assert.Equal(t, "20250131", bars[0].Date)
	assert.Equal(t, "70000", bars[0].Close)
	assert.Equal(t, "1000000", bars[0].Volume)
	assert.Equal(t, "70000", bars[0].Raw.Close)
}

func TestDailyRequiresDateRange(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1", "token")
	_, err := client.Daily(context.Background(), "005930")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start date is required")
}
