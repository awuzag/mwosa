package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeRawUsesRegistryDefaultsAndFriendlyValueNormalization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "FHKST03010100")
		assert.Equal(t, "J", r.URL.Query().Get("FID_COND_MRKT_DIV_CODE"))
		assert.Equal(t, "005930", r.URL.Query().Get("FID_INPUT_ISCD"))
		assert.Equal(t, "20250101", r.URL.Query().Get("FID_INPUT_DATE_1"))
		assert.Equal(t, "20250131", r.URL.Query().Get("FID_INPUT_DATE_2"))
		assert.Equal(t, "D", r.URL.Query().Get("FID_PERIOD_DIV_CODE"))
		assert.Equal(t, "0", r.URL.Query().Get("FID_ORG_ADJ_PRC"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "정상처리 되었습니다!",
			"output1": {},
			"output2": []
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	response, err := client.InvokeRaw(context.Background(), "inquire-daily-itemchartprice", map[string]string{
		"FID_INPUT_ISCD":      "005930",
		"FID_INPUT_DATE_1":    "20250101",
		"FID_INPUT_DATE_2":    "20250131",
		"FID_PERIOD_DIV_CODE": "daily",
	})

	require.NoError(t, err)
	require.IsType(t, InquireDailyItemChartPriceResponse{}, response)
}

func TestRawRequestTemplateIncludesGeneratedDefaults(t *testing.T) {
	template, err := RawRequestTemplate("inquire-price")

	require.NoError(t, err)
	assert.Equal(t, "J", template["FID_COND_MRKT_DIV_CODE"])
	assert.Equal(t, "", template["FID_INPUT_ISCD"])
}
