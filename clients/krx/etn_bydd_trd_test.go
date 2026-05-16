package krx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestETNBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertKRXRequest(t, r, "/etp/etn_bydd_trd")

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"OutBlock_1": [
				{
					"BAS_DD": "20250131",
					"ISU_CD": "580027",
					"ISU_NM": "QV 레버리지 WTI원유 선물 ETN",
					"TDD_CLSPRC": "12200",
					"CMPPREVDD_PRC": "-80",
					"FLUC_RT": "-0.65",
					"PER1SECU_INDIC_VAL": "12198.44",
					"TDD_OPNPRC": "12280",
					"TDD_HGPRC": "12320",
					"TDD_LWPRC": "12170",
					"ACC_TRDVOL": "10000",
					"ACC_TRDVAL": "122000000",
					"MKTCAP": "61000000000",
					"INDIC_VAL_AMT": "60992200000",
					"LIST_SHRS": "5000000",
					"IDX_IND_NM": "WTI Futures",
					"OBJ_STKPRC_IDX": "72.10",
					"CMPPREVDD_IDX": "-0.20",
					"FLUC_RT_IDX": "-0.28"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := client.ETN(context.Background(), "20250131")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "580027", rows[0].IssueCode)
	assert.Equal(t, "12198.44", rows[0].IndicativeValue)
	assert.Equal(t, "60992200000", rows[0].IndicativeValueTotal)
	assert.Equal(t, "WTI Futures", rows[0].IndexName)
}
