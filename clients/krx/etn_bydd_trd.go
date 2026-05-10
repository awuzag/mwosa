package krx

import "context"

var etnEndpoint = endpoint(GroupETP, APIETNByddTrd)

// ETNDailyTrade is a provider-native OutBlock_1 row from etn_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type ETNDailyTrade struct {
	BaseDate             string `json:"BAS_DD"`
	IssueCode            string `json:"ISU_CD"`
	IssueName            string `json:"ISU_NM"`
	Close                string `json:"TDD_CLSPRC"`
	PreviousChange       string `json:"CMPPREVDD_PRC"`
	FluctuationRate      string `json:"FLUC_RT"`
	IndicativeValue      string `json:"PER1SECU_INDIC_VAL"`
	Open                 string `json:"TDD_OPNPRC"`
	High                 string `json:"TDD_HGPRC"`
	Low                  string `json:"TDD_LWPRC"`
	Volume               string `json:"ACC_TRDVOL"`
	Amount               string `json:"ACC_TRDVAL"`
	MarketCap            string `json:"MKTCAP"`
	IndicativeValueTotal string `json:"INDIC_VAL_AMT"`
	ListedShares         string `json:"LIST_SHRS"`
	IndexName            string `json:"IDX_IND_NM"`
	IndexClose           string `json:"OBJ_STKPRC_IDX"`
	IndexChange          string `json:"CMPPREVDD_IDX"`
	IndexFluctuationRate string `json:"FLUC_RT_IDX"`
}

type etnEnvelope struct {
	OutBlock1 []ETNDailyTrade `json:"OutBlock_1"`
}

// ETN fetches ETN daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) ETN(ctx context.Context, baseDate string) ([]ETNDailyTrade, error) {
	var envelope etnEnvelope
	if err := c.outBlock(ctx, etnEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
