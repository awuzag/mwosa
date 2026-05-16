package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestETFETNPriceBuildsKISRequestAndParsesTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/etfetn/v1/quotations/inquire-price", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHPST02400000")
		assert.Equal(t, "J", r.URL.Query().Get("fid_cond_mrkt_div_code"))
		assert.Equal(t, "069500", r.URL.Query().Get("fid_input_iscd"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output": {
				"stck_prpr": "36090",
				"prdy_vrss_sign": "2",
				"prdy_vrss": "110",
				"prdy_ctrt": "0.31",
				"acml_vol": "3719307",
				"stck_prdy_clpr": "35980",
				"stck_oprc": "36300",
				"stck_hgpr": "36510",
				"stck_lwpr": "36040",
				"nav": "36127.30",
				"nav_prdy_vrss": "91.08",
				"nav_prdy_vrss_sign": "2",
				"nav_prdy_ctrt": "0.25",
				"crcd": "KRW",
				"etf_ntas_ttam": "69027",
				"etf_rprs_bstp_kor_isnm": "KOSPI200"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	price, err := client.ETFETNPrice(context.Background(), "069500")
	require.NoError(t, err)
	assert.Equal(t, "36090", price.Current)
	assert.Equal(t, "36127.30", price.NAV)
	assert.Equal(t, "KRW", price.Currency)
	assert.Equal(t, "KOSPI200", price.UnderlyingName)
}
