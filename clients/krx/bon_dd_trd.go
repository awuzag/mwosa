package krx

import "context"

var bondIndexEndpoint = endpoint(GroupIndex, APIBondDDTrd)

// BondIndexDailyTrade is a provider-native OutBlock_1 row from bon_dd_trd.
//
// Numeric values are kept as KRX-provided strings.
type BondIndexDailyTrade struct {
	BaseDate                            string `json:"BAS_DD"`
	BondIndexGroupName                  string `json:"BND_IDX_GRP_NM"`
	TotalEarningIndex                   string `json:"TOT_EARNG_IDX"`
	TotalEarningIndexPreviousChange     string `json:"TOT_EARNG_IDX_CMPPREVDD"`
	NetPriceIndex                       string `json:"NETPRC_IDX"`
	NetPriceIndexPreviousChange         string `json:"NETPRC_IDX_CMPPREVDD"`
	ZeroReinvestmentIndex               string `json:"ZERO_REINVST_IDX"`
	ZeroReinvestmentIndexPreviousChange string `json:"ZERO_REINVST_IDX_CMPPREVDD"`
	CallReinvestmentIndex               string `json:"CALL_REINVST_IDX"`
	CallReinvestmentIndexPreviousChange string `json:"CALL_REINVST_IDX_CMPPREVDD"`
	MarketPriceIndex                    string `json:"MKT_PRC_IDX"`
	MarketPriceIndexPreviousChange      string `json:"MKT_PRC_IDX_CMPPREVDD"`
	AverageDuration                     string `json:"AVG_DURATION"`
	AverageConvexityPrice               string `json:"AVG_CONVEXITY_PRC"`
	BondIndexAverageYield               string `json:"BND_IDX_AVG_YD"`
}

type bondIndexEnvelope struct {
	OutBlock1 []BondIndexDailyTrade `json:"OutBlock_1"`
}

// BondIndex fetches bond index daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) BondIndex(ctx context.Context, baseDate string) ([]BondIndexDailyTrade, error) {
	var envelope bondIndexEnvelope
	if err := c.outBlock(ctx, bondIndexEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
