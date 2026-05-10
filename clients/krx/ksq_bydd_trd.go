package krx

import "context"

var kosdaqStockEndpoint = endpoint(GroupStock, APIKOSDAQByddTrd)

// KOSDAQStockDailyTrade is a provider-native OutBlock_1 row from ksq_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSDAQStockDailyTrade struct {
	BaseDate        string `json:"BAS_DD"`
	IssueCode       string `json:"ISU_CD"`
	IssueName       string `json:"ISU_NM"`
	MarketName      string `json:"MKT_NM"`
	SectionTypeName string `json:"SECT_TP_NM"`
	Close           string `json:"TDD_CLSPRC"`
	PreviousChange  string `json:"CMPPREVDD_PRC"`
	FluctuationRate string `json:"FLUC_RT"`
	Open            string `json:"TDD_OPNPRC"`
	High            string `json:"TDD_HGPRC"`
	Low             string `json:"TDD_LWPRC"`
	Volume          string `json:"ACC_TRDVOL"`
	Amount          string `json:"ACC_TRDVAL"`
	MarketCap       string `json:"MKTCAP"`
	ListedShares    string `json:"LIST_SHRS"`
}

type kosdaqStockEnvelope struct {
	OutBlock1 []KOSDAQStockDailyTrade `json:"OutBlock_1"`
}

// KOSDAQStock fetches KOSDAQ stock daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSDAQStock(ctx context.Context, baseDate string) ([]KOSDAQStockDailyTrade, error) {
	var envelope kosdaqStockEnvelope
	if err := c.outBlock(ctx, kosdaqStockEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
