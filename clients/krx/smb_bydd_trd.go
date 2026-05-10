package krx

import "context"

var smallBondEndpoint = endpoint(GroupBond, APISmallBondByddTrd)

// SmallBondDailyTrade is a provider-native OutBlock_1 row from smb_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type SmallBondDailyTrade struct {
	BaseDate       string `json:"BAS_DD"`
	MarketName     string `json:"MKT_NM"`
	IssueCode      string `json:"ISU_CD"`
	IssueName      string `json:"ISU_NM"`
	Close          string `json:"CLSPRC"`
	PreviousChange string `json:"CMPPREVDD_PRC"`
	CloseYield     string `json:"CLSPRC_YD"`
	Open           string `json:"OPNPRC"`
	OpenYield      string `json:"OPNPRC_YD"`
	High           string `json:"HGPRC"`
	HighYield      string `json:"HGPRC_YD"`
	Low            string `json:"LWPRC"`
	LowYield       string `json:"LWPRC_YD"`
	Volume         string `json:"ACC_TRDVOL"`
	Amount         string `json:"ACC_TRDVAL"`
}

type smallBondEnvelope struct {
	OutBlock1 []SmallBondDailyTrade `json:"OutBlock_1"`
}

// SmallBond fetches small bond daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) SmallBond(ctx context.Context, baseDate string) ([]SmallBondDailyTrade, error) {
	var envelope smallBondEnvelope
	if err := c.outBlock(ctx, smallBondEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
