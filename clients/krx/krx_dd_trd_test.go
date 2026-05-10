package krx

import (
	"context"
	"testing"
)

func TestKRXIndexBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/idx/krx_dd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"IDX_CLSS\":\"IDX_CLSS-value\",\"IDX_NM\":\"IDX_NM-value\",\"CLSPRC_IDX\":\"CLSPRC_IDX-value\",\"CMPPREVDD_IDX\":\"CMPPREVDD_IDX-value\",\"FLUC_RT\":\"FLUC_RT-value\",\"OPNPRC_IDX\":\"OPNPRC_IDX-value\",\"HGPRC_IDX\":\"HGPRC_IDX-value\",\"LWPRC_IDX\":\"LWPRC_IDX-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\",\"MKTCAP\":\"MKTCAP-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.KRXIndex(ctx, "20250131") },
	)
}
