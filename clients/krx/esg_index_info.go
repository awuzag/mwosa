package krx

import "context"

var esgIndexInfoEndpoint = endpoint(GroupESG, APIESGIndexInfo)

// ESGIndexInfo is a provider-native OutBlock_1 row from esg_index_info.
//
// Numeric values are kept as KRX-provided strings.
type ESGIndexInfo struct {
	BaseDate              string `json:"BAS_DD"`
	IndexName             string `json:"IDX_NM"`
	IndexClose            string `json:"CLSPRC_IDX"`
	PreviousDayComparison string `json:"PRV_DD_CMPR"`
	UpDownRate            string `json:"UPDN_RATE"`
	TradedIssueCount      string `json:"TRD_ISU_CNT"`
	Volume                string `json:"ACC_TRDVOL"`
	Amount                string `json:"ACC_TRDVAL"`
}

type esgIndexInfoEnvelope struct {
	OutBlock1 []ESGIndexInfo `json:"OutBlock_1"`
}

// ESGIndexInfo fetches ESG index info rows for baseDate in KRX YYYYMMDD form.
func (c *Client) ESGIndexInfo(ctx context.Context, baseDate string) ([]ESGIndexInfo, error) {
	var envelope esgIndexInfoEnvelope
	if err := c.outBlock(ctx, esgIndexInfoEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
