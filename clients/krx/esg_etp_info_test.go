package krx

import (
	"context"
	"testing"
)

func TestESGETPInfoBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/esg/esg_etp_info",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"ISU_ABBRV\":\"ISU_ABBRV-value\",\"TDD_CLSPRC\":\"TDD_CLSPRC-value\",\"CMPPREVDD_PRC\":\"CMPPREVDD_PRC-value\",\"FLUC_RT\":\"FLUC_RT-value\",\"LIST_SHRS\":\"LIST_SHRS-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.ESGETPInfo(ctx, "20250131") },
	)
}
