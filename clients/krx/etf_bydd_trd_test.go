package krx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestETFBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertKRXRequest(t, r, "/etp/etf_bydd_trd")

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"OutBlock_1": [
				{
					"BAS_DD": "20250131",
					"ISU_CD": "069500",
					"ISU_NM": "KODEX 200",
					"TDD_CLSPRC": "36090",
					"CMPPREVDD_PRC": "110",
					"FLUC_RT": "0.31",
					"NAV": "36127.30",
					"TDD_OPNPRC": "36300",
					"TDD_HGPRC": "36510",
					"TDD_LWPRC": "36040",
					"ACC_TRDVOL": "3719307",
					"ACC_TRDVAL": "134200000000",
					"MKTCAP": "6902700000000",
					"INVSTASST_NETASST_TOTAMT": "6903000000000",
					"LIST_SHRS": "191200000",
					"IDX_IND_NM": "KOSPI 200",
					"OBJ_STKPRC_IDX": "481.20",
					"CMPPREVDD_IDX": "1.12",
					"FLUC_RT_IDX": "0.23"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := client.ETF(context.Background(), "20250131")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "20250131", rows[0].BaseDate)
	assert.Equal(t, "069500", rows[0].IssueCode)
	assert.Equal(t, "36090", rows[0].Close)
	assert.Equal(t, "36127.30", rows[0].NAV)
	assert.Equal(t, "KOSPI 200", rows[0].IndexName)
}

func TestETFRequiresBaseDate(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1")
	_, err := client.ETF(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base date is required")
}
