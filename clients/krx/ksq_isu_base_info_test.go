package krx

import (
	"context"
	"testing"
)

func TestKOSDAQIssueBaseInfoBuildsKRXRequestAndParsesOutBlock(t *testing.T) {
	assertKRXAPICall(
		t,
		"/sto/ksq_isu_base_info",
		"{\"OutBlock_1\":[{\"ISU_CD\":\"ISU_CD-value\",\"ISU_SRT_CD\":\"ISU_SRT_CD-value\",\"ISU_NM\":\"ISU_NM-value\",\"ISU_ABBRV\":\"ISU_ABBRV-value\",\"ISU_ENG_NM\":\"ISU_ENG_NM-value\",\"LIST_DD\":\"LIST_DD-value\",\"MKT_TP_NM\":\"MKT_TP_NM-value\",\"SECUGRP_NM\":\"SECUGRP_NM-value\",\"SECT_TP_NM\":\"SECT_TP_NM-value\",\"KIND_STKCERT_TP_NM\":\"KIND_STKCERT_TP_NM-value\",\"PARVAL\":\"PARVAL-value\",\"LIST_SHRS\":\"LIST_SHRS-value\"}]}",
		"ISU_CD",
		func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQIssueBaseInfo(ctx, "20250131")
		},
	)
}
