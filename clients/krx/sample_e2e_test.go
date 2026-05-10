//go:build e2e

package krx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSampleEndpointAllAPIs(t *testing.T) {
	authKey := os.Getenv("KRX_SAMPLE_AUTH_KEY")
	if authKey == "" {
		t.Skip("KRX_SAMPLE_AUTH_KEY is required for KRX sample endpoint e2e tests")
	}

	client, err := New(
		WithAuthKey(authKey),
		WithSampleBaseURL(DefaultSampleBaseURL),
	)
	require.NoError(t, err)

	tests := []struct {
		name string
		call func(context.Context, *Client) (any, error)
	}{
		{name: "krx_dd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KRXIndex(ctx, "20200414") }},
		{name: "kospi_dd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KOSPIIndex(ctx, "20200414") }},
		{name: "kosdaq_dd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KOSDAQIndex(ctx, "20200414") }},
		{name: "bon_dd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.BondIndex(ctx, "20200414") }},
		{name: "drvprod_dd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.DerivativesProductIndex(ctx, "20200414")
		}},
		{name: "stk_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Stock(ctx, "20200414") }},
		{name: "ksq_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KOSDAQStock(ctx, "20200414") }},
		{name: "knx_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KONEXStock(ctx, "20200414") }},
		{name: "sw_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SubscriptionWarrant(ctx, "20200414")
		}},
		{name: "sr_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SubscriptionRight(ctx, "20200414")
		}},
		{name: "stk_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.StockIssueBaseInfo(ctx, "20200414")
		}},
		{name: "ksq_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQIssueBaseInfo(ctx, "20200414")
		}},
		{name: "knx_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KONEXIssueBaseInfo(ctx, "20200414")
		}},
		{name: "etf_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ETF(ctx, "20200414") }},
		{name: "etn_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ETN(ctx, "20200414") }},
		{name: "elw_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ELW(ctx, "20200414") }},
		{name: "kts_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KTSBond(ctx, "20200414") }},
		{name: "bnd_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.GeneralBond(ctx, "20200414") }},
		{name: "smb_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.SmallBond(ctx, "20200414") }},
		{name: "fut_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Futures(ctx, "20200414") }},
		{name: "eqsfu_stk_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIStockFutures(ctx, "20200414")
		}},
		{name: "eqkfu_ksq_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQStockFutures(ctx, "20200414")
		}},
		{name: "opt_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Options(ctx, "20200414") }},
		{name: "eqsop_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIStockOptions(ctx, "20200414")
		}},
		{name: "eqkop_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQStockOptions(ctx, "20200414")
		}},
		{name: "oil_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Oil(ctx, "20200414") }},
		{name: "gold_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Gold(ctx, "20200414") }},
		{name: "ets_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.EmissionTradingScheme(ctx, "20200414")
		}},
		{name: "esg_etp_info", call: func(ctx context.Context, client *Client) (any, error) { return client.ESGETPInfo(ctx, "20200414") }},
		{name: "sri_bond_info", call: func(ctx context.Context, client *Client) (any, error) { return client.SRIBondInfo(ctx, "20200414") }},
		{name: "esg_index_info", call: func(ctx context.Context, client *Client) (any, error) { return client.ESGIndexInfo(ctx, "20200414") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			rows, err := tt.call(ctx, client)
			require.NoError(t, err)
			require.NotNil(t, rows)
		})
	}
}
