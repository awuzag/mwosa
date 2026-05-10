package krx

import "context"

var kospiIndexEndpoint = endpoint(GroupIndex, APIKOSPIDDTrd)

// KOSPIIndexDailyTrade is a provider-native OutBlock_1 row from kospi_dd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KOSPIIndexDailyTrade struct {
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

type kospiIndexEnvelope struct {
	OutBlock1 []KOSPIIndexDailyTrade `json:"OutBlock_1"`
}

// KOSPIIndex fetches KOSPI index daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KOSPIIndex(ctx context.Context, baseDate string) ([]KOSPIIndexDailyTrade, error) {
	var envelope kospiIndexEnvelope
	if err := c.outBlock(ctx, kospiIndexEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
