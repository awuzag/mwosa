package krx

import "context"

var futuresEndpoint = endpoint(GroupDerivatives, APIFuturesByddTrd)

// FuturesDailyTrade is a provider-native OutBlock_1 row from fut_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type FuturesDailyTrade struct {
	BaseDate             string `json:"BAS_DD"`
	ProductName          string `json:"PROD_NM"`
	MarketName           string `json:"MKT_NM"`
	IssueCode            string `json:"ISU_CD"`
	IssueName            string `json:"ISU_NM"`
	Close                string `json:"TDD_CLSPRC"`
	PreviousChange       string `json:"CMPPREVDD_PRC"`
	Open                 string `json:"TDD_OPNPRC"`
	High                 string `json:"TDD_HGPRC"`
	Low                  string `json:"TDD_LWPRC"`
	SpotPrice            string `json:"SPOT_PRC"`
	SettlementPrice      string `json:"SETL_PRC"`
	Volume               string `json:"ACC_TRDVOL"`
	Amount               string `json:"ACC_TRDVAL"`
	OpenInterestQuantity string `json:"ACC_OPNINT_QTY"`
}

type futuresEnvelope struct {
	OutBlock1 []FuturesDailyTrade `json:"OutBlock_1"`
}

// Futures fetches futures daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) Futures(ctx context.Context, baseDate string) ([]FuturesDailyTrade, error) {
	var envelope futuresEnvelope
	if err := c.outBlock(ctx, futuresEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
