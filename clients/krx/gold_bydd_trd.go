package krx

import "context"

var goldEndpoint = endpoint(GroupCommodity, APIGoldByddTrd)

// GoldDailyTrade is a provider-native OutBlock_1 row from gold_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type GoldDailyTrade struct {
	BaseDate        string `json:"BAS_DD"`
	IssueCode       string `json:"ISU_CD"`
	IssueName       string `json:"ISU_NM"`
	Close           string `json:"TDD_CLSPRC"`
	PreviousChange  string `json:"CMPPREVDD_PRC"`
	FluctuationRate string `json:"FLUC_RT"`
	Open            string `json:"TDD_OPNPRC"`
	High            string `json:"TDD_HGPRC"`
	Low             string `json:"TDD_LWPRC"`
	Volume          string `json:"ACC_TRDVOL"`
	Amount          string `json:"ACC_TRDVAL"`
}

type goldEnvelope struct {
	OutBlock1 []GoldDailyTrade `json:"OutBlock_1"`
}

// Gold fetches gold daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) Gold(ctx context.Context, baseDate string) ([]GoldDailyTrade, error) {
	var envelope goldEnvelope
	if err := c.outBlock(ctx, goldEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
