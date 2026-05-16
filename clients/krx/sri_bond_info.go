package krx

import "context"

var sriBondInfoEndpoint = endpoint(GroupESG, APISRIBondInfo)

// SRIBondInfo is a provider-native OutBlock_1 row from sri_bond_info.
//
// Numeric values are kept as KRX-provided strings.
type SRIBondInfo struct {
	BaseDate        string `json:"BAS_DD"`
	IssuerName      string `json:"ISUR_NM"`
	IssueCode       string `json:"ISU_CD"`
	SRIBondTypeName string `json:"SRI_BND_TP_NM"`
	IssueName       string `json:"ISU_NM"`
	ListingDate     string `json:"LIST_DD"`
	IssueDate       string `json:"ISU_DD"`
	RedemptionDate  string `json:"REDMPT_DD"`
	IssueRate       string `json:"ISU_RT"`
	IssueAmount     string `json:"ISU_AMT"`
	ListedAmount    string `json:"LIST_AMT"`
	BondTypeName    string `json:"BND_TP_NM"`
}

type sriBondInfoEnvelope struct {
	OutBlock1 []SRIBondInfo `json:"OutBlock_1"`
}

// SRIBondInfo fetches SRI bond info rows for baseDate in KRX YYYYMMDD form.
func (c *Client) SRIBondInfo(ctx context.Context, baseDate string) ([]SRIBondInfo, error) {
	var envelope sriBondInfoEnvelope
	if err := c.outBlock(ctx, sriBondInfoEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
