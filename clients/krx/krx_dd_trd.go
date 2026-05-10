package krx

import "context"

var krxIndexEndpoint = endpoint(GroupIndex, APIKRXDDTrd)

// KRXIndexDailyTrade is a provider-native OutBlock_1 row from krx_dd_trd.
//
// Numeric values are kept as KRX-provided strings.
type KRXIndexDailyTrade struct {
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

type krxIndexEnvelope struct {
	OutBlock1 []KRXIndexDailyTrade `json:"OutBlock_1"`
}

// KRXIndex fetches KRX index daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) KRXIndex(ctx context.Context, baseDate string) ([]KRXIndexDailyTrade, error) {
	var envelope krxIndexEnvelope
	if err := c.outBlock(ctx, krxIndexEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
