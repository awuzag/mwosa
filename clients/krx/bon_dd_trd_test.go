package krx

import (
	"context"
	"testing"
)

func TestBondIndexBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/idx/bon_dd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"BND_IDX_GRP_NM\":\"BND_IDX_GRP_NM-value\",\"TOT_EARNG_IDX\":\"TOT_EARNG_IDX-value\",\"TOT_EARNG_IDX_CMPPREVDD\":\"TOT_EARNG_IDX_CMPPREVDD-value\",\"NETPRC_IDX\":\"NETPRC_IDX-value\",\"NETPRC_IDX_CMPPREVDD\":\"NETPRC_IDX_CMPPREVDD-value\",\"ZERO_REINVST_IDX\":\"ZERO_REINVST_IDX-value\",\"ZERO_REINVST_IDX_CMPPREVDD\":\"ZERO_REINVST_IDX_CMPPREVDD-value\",\"CALL_REINVST_IDX\":\"CALL_REINVST_IDX-value\",\"CALL_REINVST_IDX_CMPPREVDD\":\"CALL_REINVST_IDX_CMPPREVDD-value\",\"MKT_PRC_IDX\":\"MKT_PRC_IDX-value\",\"MKT_PRC_IDX_CMPPREVDD\":\"MKT_PRC_IDX_CMPPREVDD-value\",\"AVG_DURATION\":\"AVG_DURATION-value\",\"AVG_CONVEXITY_PRC\":\"AVG_CONVEXITY_PRC-value\",\"BND_IDX_AVG_YD\":\"BND_IDX_AVG_YD-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.BondIndex(ctx, "20250131") },
	)
}
