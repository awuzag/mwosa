package krx

import (
	"context"
	"testing"
)

func TestSubscriptionRightBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/sto/sr_bydd_trd",
		"{\"OutBlock_1\":[{\"BAS_DD\":\"BAS_DD-value\",\"MKT_NM\":\"MKT_NM-value\",\"ISU_CD\":\"ISU_CD-value\",\"ISU_NM\":\"ISU_NM-value\",\"TDD_CLSPRC\":\"TDD_CLSPRC-value\",\"CMPPREVDD_PRC\":\"CMPPREVDD_PRC-value\",\"FLUC_RT\":\"FLUC_RT-value\",\"TDD_OPNPRC\":\"TDD_OPNPRC-value\",\"TDD_HGPRC\":\"TDD_HGPRC-value\",\"TDD_LWPRC\":\"TDD_LWPRC-value\",\"ACC_TRDVOL\":\"ACC_TRDVOL-value\",\"ACC_TRDVAL\":\"ACC_TRDVAL-value\",\"MKTCAP\":\"MKTCAP-value\",\"LIST_SHRS\":\"LIST_SHRS-value\",\"ISU_PRC\":\"ISU_PRC-value\",\"DELIST_DD\":\"DELIST_DD-value\",\"TARSTK_ISU_SRT_CD\":\"TARSTK_ISU_SRT_CD-value\",\"TARSTK_ISU_NM\":\"TARSTK_ISU_NM-value\",\"TARSTK_ISU_PRSNT_PRC\":\"TARSTK_ISU_PRSNT_PRC-value\"}]}",
		"BAS_DD",
		func(ctx context.Context, client *Client) (any, error) {
			return client.SubscriptionRight(ctx, "20250131")
		},
	)
}
