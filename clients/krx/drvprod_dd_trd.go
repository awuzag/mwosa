package krx

import "context"

var derivativesProductIndexEndpoint = endpoint(GroupIndex, APIDerivativesProductDDTrd)

// DerivativesProductIndexDailyTrade is a provider-native OutBlock_1 row from drvprod_dd_trd.
//
// Numeric values are kept as KRX-provided strings.
type DerivativesProductIndexDailyTrade struct {
	BaseDate            string `json:"BAS_DD"`
	IndexClass          string `json:"IDX_CLSS"`
	IndexName           string `json:"IDX_NM"`
	IndexClose          string `json:"CLSPRC_IDX"`
	IndexPreviousChange string `json:"CMPPREVDD_IDX"`
	FluctuationRate     string `json:"FLUC_RT"`
	IndexOpen           string `json:"OPNPRC_IDX"`
	IndexHigh           string `json:"HGPRC_IDX"`
	IndexLow            string `json:"LWPRC_IDX"`
}

type derivativesProductIndexEnvelope struct {
	OutBlock1 []DerivativesProductIndexDailyTrade `json:"OutBlock_1"`
}

// DerivativesProductIndex fetches derivatives product index daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) DerivativesProductIndex(ctx context.Context, baseDate string) ([]DerivativesProductIndexDailyTrade, error) {
	var envelope derivativesProductIndexEnvelope
	if err := c.outBlock(ctx, derivativesProductIndexEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
