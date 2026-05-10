package krx

import "context"

var stockIssueBaseInfoEndpoint = endpoint(GroupStock, APIStockIssueBaseInfo)

// StockIssueBaseInfo is a provider-native OutBlock_1 row from stk_isu_base_info.
//
// Numeric values are kept as KRX-provided strings.
type StockIssueBaseInfo struct {
	IssueCode                string `json:"ISU_CD"`
	IssueShortCode           string `json:"ISU_SRT_CD"`
	IssueName                string `json:"ISU_NM"`
	IssueAbbreviation        string `json:"ISU_ABBRV"`
	IssueEnglishName         string `json:"ISU_ENG_NM"`
	ListingDate              string `json:"LIST_DD"`
	MarketTypeName           string `json:"MKT_TP_NM"`
	SecurityGroupName        string `json:"SECUGRP_NM"`
	SectionTypeName          string `json:"SECT_TP_NM"`
	StockCertificateTypeName string `json:"KIND_STKCERT_TP_NM"`
	ParValue                 string `json:"PARVAL"`
	ListedShares             string `json:"LIST_SHRS"`
}

type stockIssueBaseInfoEnvelope struct {
	OutBlock1 []StockIssueBaseInfo `json:"OutBlock_1"`
}

// StockIssueBaseInfo fetches stock issue base info rows for baseDate in KRX YYYYMMDD form.
func (c *Client) StockIssueBaseInfo(ctx context.Context, baseDate string) ([]StockIssueBaseInfo, error) {
	var envelope stockIssueBaseInfoEnvelope
	if err := c.outBlock(ctx, stockIssueBaseInfoEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
