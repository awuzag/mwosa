package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceBuildsKISRequestAndParsesTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-price", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST01010100")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "정상처리 되었습니다!",
			"output": {
				"stck_shrn_iscd": "005930",
				"stck_prpr": "70100",
				"stck_oprc": "70000",
				"stck_hgpr": "71000",
				"stck_lwpr": "69500",
				"stck_sdpr": "69900",
				"prdy_vrss": "200",
				"prdy_vrss_sign": "2",
				"prdy_ctrt": "0.29",
				"acml_vol": "1234567",
				"acml_tr_pbmn": "86000000000",
				"hts_avls": "418000000",
				"per": "12.34",
				"pbr": "1.20"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	price, err := client.Price(context.Background(), "005930")
	require.NoError(t, err)
	assert.Equal(t, "005930", price.Symbol)
	assert.Equal(t, "70100", price.Current)
	assert.Equal(t, "70000", price.Open)
	assert.Equal(t, "71000", price.High)
	assert.Equal(t, "69500", price.Low)
	assert.Equal(t, "418000000", price.MarketCap)
	assert.Equal(t, "12.34", price.PER)
	assert.Equal(t, "70100", price.Raw.Current)
}

func TestBusinessErrorIncludesKISContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "1",
			"msg_cd": "EGW00123",
			"msg1": "invalid request",
			"output": {}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	_, err := client.Price(context.Background(), "005930")
	require.Error(t, err)
	for _, want := range []string{
		"provider=kis",
		"group=domesticStockQuotation",
		"operation=price",
		"tr_id=FHKST01010100",
		"rt_cd=1",
		"msg_cd=EGW00123",
	} {
		assert.Contains(t, err.Error(), want)
	}
}
