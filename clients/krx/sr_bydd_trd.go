package krx

import "context"

var subscriptionRightEndpoint = endpoint(GroupStock, APISubscriptionRightByddTrd)

// SubscriptionRightDailyTrade is a provider-native OutBlock_1 row from sr_bydd_trd.
//
// Numeric values are kept as KRX-provided strings.
type SubscriptionRightDailyTrade struct {
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
	IssuePrice                   string `json:"ISU_PRC"`
	DelistingDate                string `json:"DELIST_DD"`
	TargetStockIssueShortCode    string `json:"TARSTK_ISU_SRT_CD"`
	TargetStockIssueName         string `json:"TARSTK_ISU_NM"`
	TargetStockIssuePresentPrice string `json:"TARSTK_ISU_PRSNT_PRC"`
}

type subscriptionRightEnvelope struct {
	OutBlock1 []SubscriptionRightDailyTrade `json:"OutBlock_1"`
}

// SubscriptionRight fetches subscription right daily trade rows for baseDate in KRX YYYYMMDD form.
func (c *Client) SubscriptionRight(ctx context.Context, baseDate string) ([]SubscriptionRightDailyTrade, error) {
	var envelope subscriptionRightEnvelope
	if err := c.outBlock(ctx, subscriptionRightEndpoint, baseDate, &envelope, func() bool {
		return envelope.OutBlock1 != nil
	}); err != nil {
		return nil, err
	}
	return envelope.OutBlock1, nil
}
