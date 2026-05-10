package krx

import "context"

var esgETPInfoEndpoint = endpoint(GroupESG, APIESGETPInfo)

// ESGETPInfo is a provider-native OutBlock_1 row from esg_etp_info.
//
// Numeric values are kept as KRX-provided strings.
type ESGETPInfo struct {
	BaseDate          string `json:"BAS_DD"`
	IssueAbbreviation string `json:"ISU_ABBRV"`
	Close             string `json:"TDD_CLSPRC"`
	PreviousChange    string `json:"CMPPREVDD_PRC"`
	FluctuationRate   string `json:"FLUC_RT"`
	ListedShares      string `json:"LIST_SHRS"`
	Volume            string `json:"ACC_TRDVOL"`
	Amount            string `json:"ACC_TRDVAL"`
}

type esgETPInfoEnvelope struct {
	OutBlock1 []ESGETPInfo `json:"OutBlock_1"`
}

// ESGETPInfo fetches ESG ETP info rows for baseDate in KRX YYYYMMDD form.
func (c *Client) ESGETPInfo(ctx context.Context, baseDate string) ([]ESGETPInfo, error) {
	var envelope esgETPInfoEnvelope
	if err := c.outBlock(ctx, esgETPInfoEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
