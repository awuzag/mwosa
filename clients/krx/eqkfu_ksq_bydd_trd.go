package krx

import "context"

var kosdaqStockFuturesEndpoint = endpoint(GroupDerivatives, APIKOSDAQStockFuturesByddTrd)

// KOSDAQStockFuturesDailyTrade is a provider-native OutBlock_1 row from eqkfu_ksq_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSDAQStockFuturesDailyTrade struct {
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

type kosdaqStockFuturesEnvelope struct {
	OutBlock1 []KOSDAQStockFuturesDailyTrade `json:"OutBlock_1"`
}

// KOSDAQStockFutures fetches KOSDAQ stock futures daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSDAQStockFutures(ctx context.Context, baseDate string) ([]KOSDAQStockFuturesDailyTrade, error) {
	var envelope kosdaqStockFuturesEnvelope
	if err := c.outBlock(ctx, kosdaqStockFuturesEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
