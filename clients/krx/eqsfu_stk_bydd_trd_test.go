package krx

import (
	"context"
	"testing"
)

func TestKOSPIStockFuturesBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/drv/eqsfu_stk_bydd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"PROD_NM\":\"PROD_NM-value\",\"MKT_NM\":\"MKT_NM-value\",\"ISU_CD\":\"ISU_CD-value\",\"ISU_NM\":\"ISU_NM-value\",\"TDD_CLSPRC\":\"TDD_CLSPRC-value\",\"CMPPREVDD_PRC\":\"CMPPREVDD_PRC-value\",\"TDD_OPNPRC\":\"TDD_OPNPRC-value\",\"TDD_HGPRC\":\"TDD_HGPRC-value\",\"TDD_LWPRC\":\"TDD_LWPRC-value\",\"SPOT_PRC\":\"SPOT_PRC-value\",\"SETL_PRC\":\"SETL_PRC-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\",\"ACC_OPNINT_QTY\":\"ACC_OPNINT_QTY-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIStockFutures(ctx, "20250131")
		},
	)
}
