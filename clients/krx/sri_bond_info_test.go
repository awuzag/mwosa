package krx

import (
	"context"
	"testing"
)

func TestSRIBondInfoBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/esg/sri_bond_info",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"ISUR_NM\":\"ISUR_NM-value\",\"ISU_CD\":\"ISU_CD-value\",\"SRI_BND_TP_NM\":\"SRI_BND_TP_NM-value\",\"ISU_NM\":\"ISU_NM-value\",\"LIST_DD\":\"LIST_DD-value\",\"ISU_DD\":\"ISU_DD-value\",\"REDMPT_DD\":\"REDMPT_DD-value\",\"ISU_RT\":\"ISU_RT-value\",\"ISU_AMT\":\"ISU_AMT-value\",\"LIST_AMT\":\"LIST_AMT-value\",\"BND_TP_NM\":\"BND_TP_NM-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) { return client.SRIBondInfo(ctx, "20250131") },
	)
}
