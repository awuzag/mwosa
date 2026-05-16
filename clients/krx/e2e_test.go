//go:build e2e

package krx

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	krxE2EEnabledEnv     = "KRX_E2E"
	krxE2EAllEnv         = "KRX_E2E_ALL"
	krxE2EAuthKeyEnv     = "MWOSA_KRX_AUTH_KEY"
	krxE2ETimeoutEnv     = "KRX_E2E_TIMEOUT"
	defaultKRXE2EAsOf    = "20240415"
	defaultKRXE2ETimeout = 20 * time.Second
)

func TestE2ELiveDailyTradeSmoke(t *testing.T) {
	client, config := newLiveKRXClient(t)

	t.Run("etf_bydd_trd", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()

		rows, err := client.ETF(ctx, defaultKRXE2EAsOf)
		require.NoError(t, err)
		require.NotEmpty(t, rows)
		require.NotEmpty(t, strings.TrimSpace(rows[0].BaseDate))
		require.NotEmpty(t, strings.TrimSpace(rows[0].IssueCode))
		require.NotEmpty(t, strings.TrimSpace(rows[0].IssueName))
		t.Logf("api_id=etf_bydd_trd row_count=%d first_symbol=%s", len(rows), rows[0].IssueCode)
	})

	t.Run("stk_bydd_trd", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()

		rows, err := client.Stock(ctx, defaultKRXE2EAsOf)
		require.NoError(t, err)
		require.NotEmpty(t, rows)
		require.NotEmpty(t, strings.TrimSpace(rows[0].BaseDate))
		require.NotEmpty(t, strings.TrimSpace(rows[0].IssueCode))
		require.NotEmpty(t, strings.TrimSpace(rows[0].IssueName))
		t.Logf("api_id=stk_bydd_trd row_count=%d first_symbol=%s", len(rows), rows[0].IssueCode)
	})
}

func TestE2ELiveAllAPIsApproved(t *testing.T) {
	skipUnlessKRXE2EEnabled(t, krxE2EAllEnv, "full live KRX API approval e2e tests")
	client, config := newLiveKRXClient(t)

	for _, tt := range liveKRXAPICases() {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
			defer cancel()

			rows, err := tt.call(ctx, client)
			require.NoError(t, err)
			rowCount := requireNonEmptyLiveRows(t, tt.name, rows)
			requireFirstRowHasStringField(t, tt.name, rows)
			t.Logf("api_id=%s row_count=%d", tt.name, rowCount)
		})
	}
}

type liveKRXConfig struct {
	AuthKey string
	Timeout time.Duration
}

type liveKRXAPICase struct {
	name string
	call func(context.Context, *Client) (any, error)
}

func newLiveKRXClient(t *testing.T) (*Client, liveKRXConfig) {
	t.Helper()
	skipUnlessKRXE2EEnabled(t, krxE2EEnabledEnv, "live KRX e2e tests")

	config := liveKRXConfig{
		AuthKey: strings.TrimSpace(os.Getenv(krxE2EAuthKeyEnv)),
		Timeout: envDurationDefault(t, krxE2ETimeoutEnv, defaultKRXE2ETimeout),
	}
	if config.AuthKey == "" {
		t.Skipf("set %s to run live KRX e2e tests", krxE2EAuthKeyEnv)
	}

	client, err := New(WithAuthKey(config.AuthKey))
	require.NoError(t, err)
	return client, config
}

func liveKRXAPICases() []liveKRXAPICase {
	return []liveKRXAPICase{
		{name: "krx_dd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KRXIndex(ctx, defaultKRXE2EAsOf) }},
		{name: "kospi_dd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIIndex(ctx, defaultKRXE2EAsOf)
		}},
		{name: "kosdaq_dd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQIndex(ctx, defaultKRXE2EAsOf)
		}},
		{name: "bon_dd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.BondIndex(ctx, defaultKRXE2EAsOf)
		}},
		{name: "drvprod_dd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.DerivativesProductIndex(ctx, defaultKRXE2EAsOf)
		}},
		{name: "stk_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Stock(ctx, defaultKRXE2EAsOf) }},
		{name: "ksq_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQStock(ctx, defaultKRXE2EAsOf)
		}},
		{name: "knx_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KONEXStock(ctx, defaultKRXE2EAsOf)
		}},
		{name: "sw_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SubscriptionWarrant(ctx, defaultKRXE2EAsOf)
		}},
		{name: "sr_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SubscriptionRight(ctx, defaultKRXE2EAsOf)
		}},
		{name: "stk_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.StockIssueBaseInfo(ctx, defaultKRXE2EAsOf)
		}},
		{name: "ksq_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQIssueBaseInfo(ctx, defaultKRXE2EAsOf)
		}},
		{name: "knx_isu_base_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KONEXIssueBaseInfo(ctx, defaultKRXE2EAsOf)
		}},
		{name: "etf_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ETF(ctx, defaultKRXE2EAsOf) }},
		{name: "etn_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ETN(ctx, defaultKRXE2EAsOf) }},
		{name: "elw_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.ELW(ctx, defaultKRXE2EAsOf) }},
		{name: "kts_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.KTSBond(ctx, defaultKRXE2EAsOf) }},
		{name: "bnd_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.GeneralBond(ctx, defaultKRXE2EAsOf)
		}},
		{name: "smb_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SmallBond(ctx, defaultKRXE2EAsOf)
		}},
		{name: "fut_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Futures(ctx, defaultKRXE2EAsOf) }},
		{name: "eqsfu_stk_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIStockFutures(ctx, defaultKRXE2EAsOf)
		}},
		{name: "eqkfu_ksq_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQStockFutures(ctx, defaultKRXE2EAsOf)
		}},
		{name: "opt_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Options(ctx, defaultKRXE2EAsOf) }},
		{name: "eqsop_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSPIStockOptions(ctx, defaultKRXE2EAsOf)
		}},
		{name: "eqkop_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.KOSDAQStockOptions(ctx, defaultKRXE2EAsOf)
		}},
		{name: "oil_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Oil(ctx, defaultKRXE2EAsOf) }},
		{name: "gold_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) { return client.Gold(ctx, defaultKRXE2EAsOf) }},
		{name: "ets_bydd_trd", call: func(ctx context.Context, client *Client) (any, error) {
			return client.EmissionTradingScheme(ctx, defaultKRXE2EAsOf)
		}},
		{name: "esg_etp_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.ESGETPInfo(ctx, defaultKRXE2EAsOf)
		}},
		{name: "sri_bond_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.SRIBondInfo(ctx, defaultKRXE2EAsOf)
		}},
		{name: "esg_index_info", call: func(ctx context.Context, client *Client) (any, error) {
			return client.ESGIndexInfo(ctx, defaultKRXE2EAsOf)
		}},
	}
}

func requireNonEmptyLiveRows(t *testing.T, apiID string, rows any) int {
	t.Helper()
	value := reflect.ValueOf(rows)
	require.Equalf(t, reflect.Slice, value.Kind(), "api_id=%s returned non-slice rows: %T", apiID, rows)
	require.NotZerof(t, value.Len(), "api_id=%s returned no rows", apiID)
	return value.Len()
}

func requireFirstRowHasStringField(t *testing.T, apiID string, rows any) {
	t.Helper()
	value := reflect.ValueOf(rows)
	first := value.Index(0)
	if first.Kind() == reflect.Pointer {
		first = first.Elem()
	}
	require.Equalf(t, reflect.Struct, first.Kind(), "api_id=%s first row is not a struct: %T", apiID, rows)
	for index := 0; index < first.NumField(); index++ {
		field := first.Field(index)
		if field.Kind() == reflect.String && strings.TrimSpace(field.String()) != "" {
			return
		}
	}
	t.Fatalf("api_id=%s first row has no non-empty string field: %+v", apiID, first.Interface())
}

func skipUnlessKRXE2EEnabled(t *testing.T, envName string, label string) {
	t.Helper()
	if os.Getenv(envName) != "1" {
		t.Skipf("set %s=1 to run %s", envName, label)
	}
}

func envDurationDefault(t *testing.T, name string, fallback time.Duration) time.Duration {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	timeout, err := time.ParseDuration(value)
	if err != nil {
		t.Fatalf("parse %s=%q: %v", name, value, err)
	}
	if timeout <= 0 {
		t.Fatalf("%s must be positive: %s", name, value)
	}
	return timeout
}
