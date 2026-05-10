package krx

import (
	"context"
	"testing"
)

func TestOilBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/gen/oil_bydd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"OIL_NM\":\"OIL_NM-value\",\"WT_AVG_PRC\":\"WT_AVG_PRC-value\",\"WT_DIS_AVG_PRC\":\"WT_DIS_AVG_PRC-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.Oil(ctx, "20250131") },
	)
}
