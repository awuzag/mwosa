package krx

import (
	"context"
	"testing"
)

func TestKTSBondBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/bon/kts_bydd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"MKT_NM\":\"MKT_NM-value\",\"ISU_CD\":\"ISU_CD-value\",\"ISU_NM\":\"ISU_NM-value\",\"BND_EXP_TP_NM\":\"BND_EXP_TP_NM-value\",\"GOVBND_ISU_TP_NM\":\"GOVBND_ISU_TP_NM-value\",\"CLSPRC\":\"CLSPRC-value\",\"CMPPREVDD_PRC\":\"CMPPREVDD_PRC-value\",\"CLSPRC_YD\":\"CLSPRC_YD-value\",\"OPNPRC\":\"OPNPRC-value\",\"OPNPRC_YD\":\"OPNPRC_YD-value\",\"HGPRC\":\"HGPRC-value\",\"HGPRC_YD\":\"HGPRC_YD-value\",\"LWPRC\":\"LWPRC-value\",\"LWPRC_YD\":\"LWPRC_YD-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.KTSBond(ctx, "20250131") },
	)
}
