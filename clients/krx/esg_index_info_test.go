package krx

import (
	"context"
	"testing"
)

func TestESGIndexInfoBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/esg/esg_index_info",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"IDX_NM\":\"IDX_NM-value\",\"CLSPRC_IDX\":\"CLSPRC_IDX-value\",\"PRV_DD_CMPR\":\"PRV_DD_CMPR-value\",\"UPDN_RATE\":\"UPDN_RATE-value\",\"TRD_ISU_CNT\":\"TRD_ISU_CNT-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.ESGIndexInfo(ctx, "20250131") },
	)
}
