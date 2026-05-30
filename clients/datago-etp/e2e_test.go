package etp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	etp "github.com/awuzag/mwosa/clients/datago-etp"
)

const defaultLiveConfigPath = "config.local.json"
const liveTestEnabledEnv = "DATAGO_ETP_E2E"
const liveFixtureBasDt = "20260424"

type liveConfig struct {
	ServiceKey string `json:"service_key"`
	NumOfRows  int    `json:"num_of_rows"`
	Timeout    string `json:"timeout"`
}

func TestLiveETFPriceInfo(t *testing.T) {
	t.Parallel()
	skipUnlessLiveE2EEnabled(t)
	client, ctx, config := newLiveClient(t)

	// KODEX 200 is a long-lived ETF, so this fixture keeps the live request narrow
	// without depending on a moving "latest date" query.
	result, err := client.GetETFPriceInfo(ctx, etp.ETFPriceInfoQuery{
		SecuritiesProductPriceQuery: etp.SecuritiesProductPriceQuery{
			BasDt:      liveFixtureBasDt,
			LikeSrtnCd: "069500",
			NumOfRows:  config.NumOfRows,
		},
	})
	if err != nil {
		t.Fatalf("get live ETF price info: %v", err)
	}
	assertLiveItems(t, etp.OperationGetETFPriceInfo, result.TotalCount, len(result.Items))
	item := result.Items[0]
	assertExpectedItem(t, etp.OperationGetETFPriceInfo, item.CommonPriceInfo, "069500", "")
	t.Logf("operation=%s basDt=%s srtnCd=%s itmsNm=%s", etp.OperationGetETFPriceInfo, item.BasDt, item.SrtnCd, item.ItmsNm)
}

func TestLiveETNPriceInfo(t *testing.T) {
	t.Parallel()
	skipUnlessLiveE2EEnabled(t)
	client, ctx, config := newLiveClient(t)

	// This checks the ETN endpoint through the public ETN query type. If the
	// fixture is delisted in the future, replace only this concrete query.
	result, err := client.GetETNPriceInfo(ctx, etp.ETNPriceInfoQuery{
		SecuritiesProductPriceQuery: etp.SecuritiesProductPriceQuery{
			BasDt:      liveFixtureBasDt,
			LikeItmsNm: "ETN",
			NumOfRows:  config.NumOfRows,
		},
	})
	if err != nil {
		t.Fatalf("get live ETN price info: %v", err)
	}
	assertLiveItems(t, etp.OperationGetETNPriceInfo, result.TotalCount, len(result.Items))
	item := result.Items[0]
	assertExpectedItem(t, etp.OperationGetETNPriceInfo, item.CommonPriceInfo, "", "ETN")
	t.Logf("operation=%s basDt=%s srtnCd=%s itmsNm=%s", etp.OperationGetETNPriceInfo, item.BasDt, item.SrtnCd, item.ItmsNm)
}

func TestLiveELWPriceInfo(t *testing.T) {
	t.Parallel()
	skipUnlessLiveE2EEnabled(t)
	client, ctx, config := newLiveClient(t)

	// ELW short codes rotate often, so the fixture searches by a common underlying
	// asset instead of pinning one warrant code.
	result, err := client.GetELWPriceInfo(ctx, etp.ELWPriceInfoQuery{
		SecuritiesProductPriceQuery: etp.SecuritiesProductPriceQuery{
			BasDt:     liveFixtureBasDt,
			NumOfRows: config.NumOfRows,
		},
		LikeUdasAstNm: "삼성전자",
	})
	if err != nil {
		t.Fatalf("get live ELW price info: %v", err)
	}
	assertLiveItems(t, etp.OperationGetELWPriceInfo, result.TotalCount, len(result.Items))
	item := result.Items[0]
	assertExpectedAsset(t, etp.OperationGetELWPriceInfo, item.UdasAstNm, "삼성전자")
	t.Logf("operation=%s basDt=%s srtnCd=%s itmsNm=%s", etp.OperationGetELWPriceInfo, item.BasDt, item.SrtnCd, item.ItmsNm)
}

func skipUnlessLiveE2EEnabled(t *testing.T) {
	t.Helper()

	if os.Getenv(liveTestEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run live Datago ETP e2e tests", liveTestEnabledEnv)
	}
}

func newLiveClient(t *testing.T) (*etp.Client, context.Context, liveConfig) {
	t.Helper()

	config := loadLiveConfig(t)
	timeout := parseLiveTimeout(t, config.Timeout)
	client, err := etp.New(etp.Config{
		ServiceKey: config.ServiceKey,
	})
	if err != nil {
		t.Fatalf("new live client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return client, ctx, config
}

func loadLiveConfig(t *testing.T) liveConfig {
	t.Helper()

	path := strings.TrimSpace(os.Getenv("DATAGO_ETP_CONFIG"))
	if path == "" {
		path = defaultLiveConfigPath
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live config %q: %v", path, err)
	}

	var config liveConfig
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		t.Fatalf("decode live config %q: %v", path, err)
	}
	if strings.TrimSpace(config.ServiceKey) == "" {
		t.Fatalf("live config %q: service_key is required", path)
	}
	if config.NumOfRows <= 0 {
		config.NumOfRows = etp.DefaultNumOfRows
	}
	if strings.TrimSpace(config.Timeout) == "" {
		config.Timeout = etp.DefaultHTTPClientTimeout.String()
	}
	return config
}

func parseLiveTimeout(t *testing.T, value string) time.Duration {
	t.Helper()

	timeout, err := time.ParseDuration(value)
	if err != nil {
		t.Fatalf("parse live timeout %q: %v", value, err)
	}
	if timeout <= 0 {
		t.Fatalf("live timeout must be positive: %s", value)
	}
	return timeout
}

func assertLiveItems(t *testing.T, operation string, totalCount int, itemCount int) {
	t.Helper()

	if totalCount <= 0 {
		t.Fatalf("%s totalCount = %d, want > 0", operation, totalCount)
	}
	if itemCount == 0 {
		t.Fatalf("%s returned no items", operation)
	}
}

func assertExpectedItem(t *testing.T, operation string, item etp.CommonPriceInfo, expectedCode string, expectedNamePart string) {
	t.Helper()

	if expectedCode != "" && item.SrtnCd != expectedCode {
		t.Fatalf("%s srtnCd = %q, want %q", operation, item.SrtnCd, expectedCode)
	}
	if expectedNamePart != "" && !strings.Contains(item.ItmsNm, expectedNamePart) {
		t.Fatalf("%s itmsNm = %q, want to contain %q", operation, item.ItmsNm, expectedNamePart)
	}
}

func assertExpectedAsset(t *testing.T, operation string, assetName string, expectedAssetPart string) {
	t.Helper()

	if expectedAssetPart != "" && !strings.Contains(assetName, expectedAssetPart) {
		t.Fatalf("%s underlying asset = %q, want to contain %q", operation, assetName, expectedAssetPart)
	}
}
