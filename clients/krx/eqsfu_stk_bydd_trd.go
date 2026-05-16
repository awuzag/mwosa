package krx

import "context"

var kospiStockFuturesEndpoint = endpoint(GroupDerivatives, APIKOSPIStockFuturesByddTrd)

// KOSPIStockFuturesDailyTrade is a provider-native OutBlock_1 row from eqsfu_stk_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSPIStockFuturesDailyTrade struct {
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

type kospiStockFuturesEnvelope struct {
	OutBlock1 []KOSPIStockFuturesDailyTrade `json:"OutBlock_1"`
}

// KOSPIStockFutures fetches KOSPI stock futures daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSPIStockFutures(ctx context.Context, baseDate string) ([]KOSPIStockFuturesDailyTrade, error) {
	var envelope kospiStockFuturesEnvelope
	if err := c.outBlock(ctx, kospiStockFuturesEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
