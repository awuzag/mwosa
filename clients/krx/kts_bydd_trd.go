package krx

import "context"

var ktsBondEndpoint = endpoint(GroupBond, APIKTSByddTrd)

// KTSBondDailyTrade is a provider-native OutBlock_1 row from kts_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KTSBondDailyTrade struct {
	BaseDate                    string `json:"BAS_DD"`
	MarketName                  string `json:"MKT_NM"`
	IssueCode                   string `json:"ISU_CD"`
	IssueName                   string `json:"ISU_NM"`
	BondExpirationTypeName      string `json:"BND_EXP_TP_NM"`
	GovernmentBondIssueTypeName string `json:"GOVBND_ISU_TP_NM"`
	Close                       string `json:"CLSPRC"`
	PreviousChange              string `json:"CMPPREVDD_PRC"`
	CloseYield                  string `json:"CLSPRC_YD"`
	Open                        string `json:"OPNPRC"`
	OpenYield                   string `json:"OPNPRC_YD"`
	High                        string `json:"HGPRC"`
	HighYield                   string `json:"HGPRC_YD"`
	Low                         string `json:"LWPRC"`
	LowYield                    string `json:"LWPRC_YD"`
	Volume                      string `json:"ACC_TRDVOL"`
	Amount                      string `json:"ACC_TRDVAL"`
}

type ktsBondEnvelope struct {
	OutBlock1 []KTSBondDailyTrade `json:"OutBlock_1"`
}

// KTSBond fetches KTS bond daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KTSBond(ctx context.Context, baseDate string) ([]KTSBondDailyTrade, error) {
	var envelope ktsBondEnvelope
	if err := c.outBlock(ctx, ktsBondEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
