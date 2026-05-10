package krx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestELWBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertKRXRequest(t, r, "/etp/elw_bydd_trd")

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"OutBlock_1": [
				{
					"BAS_DD": "20250131",
					"ISU_CD": "57JB70",
					"ISU_NM": "삼성전자콜",
					"TDD_CLSPRC": "120",
					"CMPPREVDD_PRC": "5",
					"TDD_OPNPRC": "115",
					"TDD_HGPRC": "125",
					"TDD_LWPRC": "110",
					"ACC_TRDVOL": "500000",
					"ACC_TRDVAL": "60000000",
					"MKTCAP": "1200000000",
					"LIST_SHRS": "10000000",
					"ULY_NM": "삼성전자",
					"ULY_PRC": "70100",
					"CMPPREVDD_PRC_ULY": "200",
					"FLUC_RT_ULY": "0.29"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := client.ELW(context.Background(), "20250131")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "57JB70", rows[0].IssueCode)
	assert.Equal(t, "120", rows[0].Close)
	assert.Equal(t, "삼성전자", rows[0].UnderlyingName)
	assert.Equal(t, "0.29", rows[0].UnderlyingFluctuation)
}
