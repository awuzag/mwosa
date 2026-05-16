package krx

import "context"

var kospiStockOptionsEndpoint = endpoint(GroupDerivatives, APIKOSPIStockOptionsByddTrd)

// KOSPIStockOptionsDailyTrade is a provider-native OutBlock_1 row from eqsop_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSPIStockOptionsDailyTrade struct {
	BaseDate             string `json:"BAS_DD"`
	ProductName          string `json:"PROD_NM"`
	RightTypeName        string `json:"RGHT_TP_NM"`
	IssueCode            string `json:"ISU_CD"`
	IssueName            string `json:"ISU_NM"`
	Close                string `json:"TDD_CLSPRC"`
	PreviousChange       string `json:"CMPPREVDD_PRC"`
	Open                 string `json:"TDD_OPNPRC"`
	High                 string `json:"TDD_HGPRC"`
	Low                  string `json:"TDD_LWPRC"`
	ImpliedVolatility    string `json:"IMP_VOLT"`
	NextDayBasePrice     string `json:"NXTDD_BAS_PRC"`
	Volume               string `json:"ACC_TRDVOL"`
	Amount               string `json:"ACC_TRDVAL"`
	OpenInterestQuantity string `json:"ACC_OPNINT_QTY"`
}

type kospiStockOptionsEnvelope struct {
	OutBlock1 []KOSPIStockOptionsDailyTrade `json:"OutBlock_1"`
}

// KOSPIStockOptions fetches KOSPI stock options daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSPIStockOptions(ctx context.Context, baseDate string) ([]KOSPIStockOptionsDailyTrade, error) {
	var envelope kospiStockOptionsEnvelope
	if err := c.outBlock(ctx, kospiStockOptionsEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
