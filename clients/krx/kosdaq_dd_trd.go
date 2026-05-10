package krx

import "context"

var kosdaqIndexEndpoint = endpoint(GroupIndex, APIKOSDAQDDTrd)

// KOSDAQIndexDailyTrade is a provider-native OutBlock_1 row from kosdaq_dd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSDAQIndexDailyTrade struct {
	BaseDate            string `json:"BAS_DD"`
	IndexClass          string `json:"IDX_CLSS"`
	IndexName           string `json:"IDX_NM"`
	IndexClose          string `json:"CLSPRC_IDX"`
	IndexPreviousChange string `json:"CMPPREVDD_IDX"`
	FluctuationRate     string `json:"FLUC_RT"`
	IndexOpen           string `json:"OPNPRC_IDX"`
	IndexHigh           string `json:"HGPRC_IDX"`
	IndexLow            string `json:"LWPRC_IDX"`
	Volume              string `json:"ACC_TRDVOL"`
	Amount              string `json:"ACC_TRDVAL"`
	MarketCap           string `json:"MKTCAP"`
}

type kosdaqIndexEnvelope struct {
	OutBlock1 []KOSDAQIndexDailyTrade `json:"OutBlock_1"`
}

// KOSDAQIndex fetches KOSDAQ index daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSDAQIndex(ctx context.Context, baseDate string) ([]KOSDAQIndexDailyTrade, error) {
	var envelope kosdaqIndexEnvelope
	if err := c.outBlock(ctx, kosdaqIndexEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
