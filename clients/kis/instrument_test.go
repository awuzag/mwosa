package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductBuildsKISRequestAndParsesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/search-info", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "CTPF1604R")
		assert.Equal(t, "005930", r.URL.Query().Get("PDNO"))
		assert.Equal(t, "300", r.URL.Query().Get("PRDT_TYPE_CD"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "KIOK0530",
			"msg1": "조회되었습니다",
			"output": {
				"pdno": "005930",
				"prdt_type_cd": "300",
				"prdt_name": "삼성전자",
				"prdt_abrv_name": "삼성전자",
				"prdt_eng_name": "Samsung Electronics",
				"std_pdno": "KR7005930003",
				"shtn_pdno": "005930",
				"prdt_clsf_cd": "101210",
				"prdt_clsf_name": "국내주식",
				"ivst_prdt_type_cd": "1012",
				"ivst_prdt_type_cd_name": "국내주식"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	product, err := client.Product(context.Background(), "005930")
	require.NoError(t, err)
	assert.Equal(t, "005930", product.ProductNo)
	assert.Equal(t, "삼성전자", product.Name)
	assert.Equal(t, "KR7005930003", product.StandardProductNo)
}

func TestStockBuildsKISRequestAndParsesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/uapi/domestic-stock/v1/quotations/search-stock-info", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		assertKISHeaders(t, r, "CTPF1002R")
		assert.Equal(t, "005930", r.URL.Query().Get("PDNO"))
		assert.Equal(t, "300", r.URL.Query().Get("PRDT_TYPE_CD"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "KIOK0530",
			"msg1": "조회되었습니다",
			"output": {
				"pdno": "00000A005930",
				"prdt_type_cd": "300",
				"mket_id_cd": "STK",
				"scty_grp_id_cd": "ST",
				"excg_dvsn_cd": "02",
				"setl_mmdd": "12",
				"lstg_stqt": "5969782550",
				"cpta": "897514000000",
				"papr": "100",
				"prdt_name": "삼성전자보통주",
				"prdt_abrv_name": "삼성전자",
				"prdt_eng_name": "Samsung Electronics",
				"std_pdno": "KR7005930003",
				"tr_stop_yn": "N",
				"admn_item_yn": "N",
				"std_idst_clsf_cd": "032601",
				"std_idst_clsf_cd_name": "통신 및 방송 장비 제조업"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	stock, err := client.Stock(context.Background(), "005930")
	require.NoError(t, err)
	assert.Equal(t, "00000A005930", stock.ProductNo)
	assert.Equal(t, "삼성전자", stock.AbbreviatedName)
	assert.Equal(t, "5969782550", stock.ListedShares)
	assert.Equal(t, "032601", stock.IndustryCode)
}
