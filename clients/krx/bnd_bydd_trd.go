package krx

import "context"

var generalBondEndpoint = endpoint(GroupBond, APIGeneralBondByddTrd)

// GeneralBondDailyTrade is a provider-native OutBlock_1 row from bnd_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type GeneralBondDailyTrade struct {
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

type generalBondEnvelope struct {
	OutBlock1 []GeneralBondDailyTrade `json:"OutBlock_1"`
}

// GeneralBond fetches general bond daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) GeneralBond(ctx context.Context, baseDate string) ([]GeneralBondDailyTrade, error) {
	var envelope generalBondEnvelope
	if err := c.outBlock(ctx, generalBondEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
