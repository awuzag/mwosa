package krx

import "context"

var elwEndpoint = endpoint(GroupETP, APIELWByddTrd)

// ELWDailyTrade is a provider-native OutBlock_1 row from elw_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type ELWDailyTrade struct {
	BaseDate                 string `json:"BAS_DD"`
	IssueCode                string `json:"ISU_CD"`
	IssueName                string `json:"ISU_NM"`
	Close                    string `json:"TDD_CLSPRC"`
	PreviousChange           string `json:"CMPPREVDD_PRC"`
	Open                     string `json:"TDD_OPNPRC"`
	High                     string `json:"TDD_HGPRC"`
	Low                      string `json:"TDD_LWPRC"`
	Volume                   string `json:"ACC_TRDVOL"`
	Amount                   string `json:"ACC_TRDVAL"`
	MarketCap                string `json:"MKTCAP"`
	ListedShares             string `json:"LIST_SHRS"`
	UnderlyingName           string `json:"ULY_NM"`
	UnderlyingPrice          string `json:"ULY_PRC"`
	UnderlyingPreviousChange string `json:"CMPPREVDD_PRC_ULY"`
	UnderlyingFluctuation    string `json:"FLUC_RT_ULY"`
}

type elwEnvelope struct {
	OutBlock1 []ELWDailyTrade `json:"OutBlock_1"`
}

// ELW fetches ELW daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) ELW(ctx context.Context, baseDate string) ([]ELWDailyTrade, error) {
	var envelope elwEnvelope
	if err := c.outBlock(ctx, elwEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
