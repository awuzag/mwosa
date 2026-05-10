package krx

import "context"

var oilEndpoint = endpoint(GroupCommodity, APIOilByddTrd)

// OilDailyTrade is a provider-native OutBlock_1 row from oil_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type OilDailyTrade struct {
	BaseDate                     string `json:"BAS_DD"`
	OilName                      string `json:"OIL_NM"`
	WeightedAveragePrice         string `json:"WT_AVG_PRC"`
	WeightedDiscountAveragePrice string `json:"WT_DIS_AVG_PRC"`
	Volume                       string `json:"ACC_TRDVOL"`
	Amount                       string `json:"ACC_TRDVAL"`
}

type oilEnvelope struct {
	OutBlock1 []OilDailyTrade `json:"OutBlock_1"`
}

// Oil fetches oil daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) Oil(ctx context.Context, baseDate string) ([]OilDailyTrade, error) {
	var envelope oilEnvelope
	if err := c.outBlock(ctx, oilEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
