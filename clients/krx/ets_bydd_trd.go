package krx

import "context"

var emissionTradingSchemeEndpoint = endpoint(GroupCommodity, APIEmissionTradingSchemeByddTrd)

// EmissionTradingSchemeDailyTrade is a provider-native OutBlock_1 row from ets_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type EmissionTradingSchemeDailyTrade struct {
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

type emissionTradingSchemeEnvelope struct {
	OutBlock1 []EmissionTradingSchemeDailyTrade `json:"OutBlock_1"`
}

// EmissionTradingScheme fetches emission trading scheme daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) EmissionTradingScheme(ctx context.Context, baseDate string) ([]EmissionTradingSchemeDailyTrade, error) {
	var envelope emissionTradingSchemeEnvelope
	if err := c.outBlock(ctx, emissionTradingSchemeEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
