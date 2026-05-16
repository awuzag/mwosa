package krx

import "context"

var subscriptionWarrantEndpoint = endpoint(GroupStock, APISubscriptionWarrantByddTrd)

// SubscriptionWarrantDailyTrade is a provider-native OutBlock_1 row from sw_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type SubscriptionWarrantDailyTrade struct {
	BaseDate                     string `json:"BAS_DD"`
	MarketName                   string `json:"MKT_NM"`
	IssueCode                    string `json:"ISU_CD"`
	IssueName                    string `json:"ISU_NM"`
	Close                        string `json:"TDD_CLSPRC"`
	PreviousChange               string `json:"CMPPREVDD_PRC"`
	FluctuationRate              string `json:"FLUC_RT"`
	Open                         string `json:"TDD_OPNPRC"`
	High                         string `json:"TDD_HGPRC"`
	Low                          string `json:"TDD_LWPRC"`
	Volume                       string `json:"ACC_TRDVOL"`
	Amount                       string `json:"ACC_TRDVAL"`
	MarketCap                    string `json:"MKTCAP"`
	ListedShares                 string `json:"LIST_SHRS"`
	ExercisePrice                string `json:"EXER_PRC"`
	ExerciseStartDate            string `json:"EXST_STRT_DD"`
	ExerciseEndDate              string `json:"EXST_END_DD"`
	TargetStockIssueShortCode    string `json:"TARSTK_ISU_SRT_CD"`
	TargetStockIssueName         string `json:"TARSTK_ISU_NM"`
	TargetStockIssuePresentPrice string `json:"TARSTK_ISU_PRSNT_PRC"`
}

type subscriptionWarrantEnvelope struct {
	OutBlock1 []SubscriptionWarrantDailyTrade `json:"OutBlock_1"`
}

// SubscriptionWarrant fetches subscription warrant daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) SubscriptionWarrant(ctx context.Context, baseDate string) ([]SubscriptionWarrantDailyTrade, error) {
	var envelope subscriptionWarrantEnvelope
	if err := c.outBlock(ctx, subscriptionWarrantEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
