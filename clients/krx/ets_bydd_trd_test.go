package krx

import (
	"context"
	"testing"
)

func TestEmissionTradingSchemeBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/gen/ets_bydd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"ISU_CD\":\"ISU_CD-value\",\"ISU_NM\":\"ISU_NM-value\",\"TDD_CLSPRC\":\"TDD_CLSPRC-value\",\"CMPPREVDD_PRC\":\"CMPPREVDD_PRC-value\",\"FLUC_RT\":\"FLUC_RT-value\",\"TDD_OPNPRC\":\"TDD_OPNPRC-value\",\"TDD_HGPRC\":\"TDD_HGPRC-value\",\"TDD_LWPRC\":\"TDD_LWPRC-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) {
			return client.EmissionTradingScheme(ctx, "20250131")
		},
	)
}
