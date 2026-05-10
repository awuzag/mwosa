package krx

import "context"

var etfEndpoint = endpoint(GroupETP, APIETFByddTrd)

// ETFDailyTrade is a provider-native OutBlock_1 row from etf_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type ETFDailyTrade struct {
	BaseDate             string `json:"BAS_DD"`
	IssueCode            string `json:"ISU_CD"`
	IssueName            string `json:"ISU_NM"`
	Close                string `json:"TDD_CLSPRC"`
	PreviousChange       string `json:"CMPPREVDD_PRC"`
	FluctuationRate      string `json:"FLUC_RT"`
	NAV                  string `json:"NAV"`
	Open                 string `json:"TDD_OPNPRC"`
	High                 string `json:"TDD_HGPRC"`
	Low                  string `json:"TDD_LWPRC"`
	Volume               string `json:"ACC_TRDVOL"`
	Amount               string `json:"ACC_TRDVAL"`
	MarketCap            string `json:"MKTCAP"`
	InvestmentAssetValue string `json:"INVSTASST_NETASST_TOTAMT"`
	ListedShares         string `json:"LIST_SHRS"`
	IndexName            string `json:"IDX_IND_NM"`
	IndexClose           string `json:"OBJ_STKPRC_IDX"`
	IndexChange          string `json:"CMPPREVDD_IDX"`
	IndexFluctuationRate string `json:"FLUC_RT_IDX"`
}

type etfEnvelope struct {
	OutBlock1 []ETFDailyTrade `json:"OutBlock_1"`
}

// ETF fetches ETF daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) ETF(ctx context.Context, baseDate string) ([]ETFDailyTrade, error) {
	var envelope etfEnvelope
	if err := c.outBlock(ctx, etfEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
