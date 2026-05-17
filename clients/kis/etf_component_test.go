package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestETFComponentStockPricesBuildsKISRequestAndParsesRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/etfetn/v1/quotations/inquire-component-stock-price", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST121600C0")
		assert.Equal(t, "J", r.URL.Query().Get("fid_cond_mrkt_div_code"))
		assert.Equal(t, "069500", r.URL.Query().Get("fid_input_iscd"))
		assert.Equal(t, "11216", r.URL.Query().Get("fid_cond_scr_div_code"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output1": {
				"stck_prpr": "36090"
			},
			"output2": [
				{
					"mksc_shrn_iscd": "005930",
					"hts_kor_isnm": "삼성전자",
					"stck_prpr": "75000",
					"prdy_vrss": "100",
					"prdy_vrss_sign": "2",
					"prdy_ctrt": "0.13",
					"acml_vol": "123456",
					"etf_cnfg_issu_rlim": "28.15",
					"etf_cnfg_issu_avls": "1942000000",
					"cnfg_issu_qty": "25893"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	result, err := client.ETFComponentStockPrices(context.Background(), "069500")
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, "069500", result.Symbol)
	assert.Equal(t, "005930", result.Rows[0].Symbol)
	assert.Equal(t, "삼성전자", result.Rows[0].Name)
	assert.Equal(t, "75000", result.Rows[0].Current)
	assert.Equal(t, "28.15", result.Rows[0].Weight)
	assert.Equal(t, "36090", result.Output1["stck_prpr"])
}
