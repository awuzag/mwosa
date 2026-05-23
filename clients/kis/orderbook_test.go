package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderbookBuildsKISRequestAndParsesLevels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST01010200")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output1": {
				"aspr_acpt_hour": "160000",
				"askp1": "70100",
				"askp2": "70200",
				"bidp1": "70000",
				"bidp2": "69900",
				"askp_rsqn1": "100",
				"askp_rsqn2": "200",
				"bidp_rsqn1": "300",
				"bidp_rsqn2": "400",
				"askp_rsqn_icdc1": "1",
				"bidp_rsqn_icdc1": "2",
				"total_askp_rsqn": "300",
				"total_bidp_rsqn": "700"
			},
			"output2": {
				"stck_shrn_iscd": "005930",
				"stck_prpr": "70000",
				"antc_cnpr": "70100",
				"antc_vol": "1000",
				"antc_cntg_vrss": "100",
				"antc_cntg_vrss_sign": "2",
				"antc_cntg_prdy_ctrt": "0.14"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	book, err := client.Quote().Orderbook(context.Background(), InquireAskingPriceExpCcnRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
	})
	require.NoError(t, err)
	assert.Equal(t, "160000", book.Output1.AsprAcptHour)
	assert.Equal(t, "70100", book.Output1.Askp1)
	assert.Equal(t, "100", book.Output1.AskpRsqn1)
	assert.Equal(t, "70000", book.Output1.Bidp1)
	assert.Equal(t, "300", book.Output1.BidpRsqn1)
	assert.Equal(t, "70100", book.Output2.AntcCnpr)
}
