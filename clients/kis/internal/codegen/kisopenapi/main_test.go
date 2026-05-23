package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOperationConfigsDiscoversCatalogAndMergesOverrides(t *testing.T) {
	source := testCatalog(
		testCollection("[국내주식] 기본시세",
			testAPI("주식현재가 시세", "/uapi/domestic-stock/v1/quotations/inquire-price"),
			testAPI("주식현재가 일자별", "/uapi/domestic-stock/v1/quotations/inquire-daily-price"),
		),
	)
	overridesDir := writeTestOverrides(t, `
version: 1
operations:
  inquire-price:
    enabled: true
    access_url: /uapi/domestic-stock/v1/quotations/inquire-price
    go_name: StablePrice
    raw_function: StablePrice
    group_method: StablePrice
    role_hint: quote_snapshot
`, `
version: 1
groups:
  quote:
    service_name: Quote
    source_category: "[국내주식] 기본시세"
    operations:
      - inquire-daily-price
`, `
version: 1
role_hints:
  inquire-daily-price:
    candidate: daily_bar
    adapter_boundary: generated_assisted
`)

	configs, err := loadOperationConfigs(source, overridesDir)
	require.NoError(t, err)
	require.Len(t, configs, 2)
	assert.Equal(t, "inquire-price", configs[0].OperationID)
	assert.Equal(t, "StablePrice", configs[0].GoName)
	assert.Equal(t, "quote_snapshot", configs[0].RoleCandidate)
	assert.Equal(t, "inquire-daily-price", configs[1].OperationID)
	assert.Equal(t, "InquireDailyPrice", configs[1].GoName)
	assert.Equal(t, "Quote", configs[1].ServiceGroup)
}

func TestLoadOperationConfigsDisabledOverrideWinsOverGroup(t *testing.T) {
	source := testCatalog(
		testCollection("[국내주식] 종목정보",
			testAPI("상품기본조회", "/uapi/domestic-stock/v1/quotations/search-info"),
		),
	)
	overridesDir := writeTestOverrides(t, `
version: 1
operations:
  search-info:
    enabled: false
    go_name: SearchInfo
`, `
version: 1
groups:
  instrument:
    service_name: Instrument
    source_category: "[국내주식] 종목정보"
    operations:
      - search-info
`, `
version: 1
role_hints: {}
`)

	configs, err := loadOperationConfigs(source, overridesDir)
	require.NoError(t, err)
	assert.Empty(t, configs)
}

func TestLoadOperationConfigsRejectsExcludedEnabledAPI(t *testing.T) {
	source := testCatalog(
		testCollection("[국내주식] 주문/계좌",
			testAPI("주식주문(현금)", "/uapi/domestic-stock/v1/trading/order-cash"),
		),
	)
	overridesDir := writeTestOverrides(t, `
version: 1
operations:
  order-cash:
    enabled: true
`, `
version: 1
groups: {}
`, `
version: 1
role_hints: {}
`)

	_, err := loadOperationConfigs(source, overridesDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order-cash")
	assert.Contains(t, err.Error(), "excluded")
}

func TestValidateGeneratedSurfaceReturnsActionableCollision(t *testing.T) {
	file := splitFile{
		Config: operationConfig{
			OperationID:   "collision",
			GoName:        "Collision",
			RawFunction:   "Collision",
			Group:         "quote",
			ServiceGroup:  "Quote",
			GroupMethod:   "Collision",
			AccessURL:     "/uapi/domestic-stock/v1/quotations/collision",
			SplitPath:     "apis/domestic-stock/v1/quotations/collision.json",
			RoleCandidate: "read_only",
			SecurityType:  "oauth2-access-token",
		},
		API: catalogAPI{
			AccessURL:   "/uapi/domestic-stock/v1/quotations/collision",
			HTTPMethod:  "GET",
			Name:        "충돌 테스트",
			RealTRID:    "TR",
			VirtualTRID: "TR",
			Properties: []catalogProperty{
				{BodyType: "req_b", PropertyCD: "AB_CD", PropertyOrder: "1"},
				{BodyType: "req_b", PropertyCD: "ab-cd", PropertyOrder: "2"},
			},
		},
	}

	err := validateGeneratedSurface([]splitFile{file})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field name collision")
	assert.Contains(t, err.Error(), "operation=collision")
	assert.Contains(t, err.Error(), "struct=CollisionRequest")
}

func TestGeneratedCodeKeepsExecutorBoundary(t *testing.T) {
	file := splitFile{
		Config: operationConfig{
			OperationID:   "inquire-price",
			GoName:        "InquirePrice",
			RawFunction:   "InquirePrice",
			Group:         "quote",
			ServiceGroup:  "Quote",
			GroupMethod:   "Price",
			AccessURL:     "/uapi/domestic-stock/v1/quotations/inquire-price",
			SplitPath:     "apis/domestic-stock/quotation/inquire-price.json",
			RoleCandidate: "quote_snapshot",
			SecurityType:  "oauth2-access-token",
		},
		API: testAPI("주식현재가 시세", "/uapi/domestic-stock/v1/quotations/inquire-price"),
	}

	runtime := generatedRuntime()
	service := generatedService(fileGroup{Group: file.Config.Group, ServiceGroup: file.Config.ServiceGroup, Files: []splitFile{file}})
	api := generatedGroupAPI([]splitFile{file})
	assert.Contains(t, runtime, "type Executor interface")
	assert.NotContains(t, runtime, "type Runtime interface")
	assert.Contains(t, service, "executor rawapi.Executor")
	assert.Contains(t, service, "func (c *Client) Quote() QuoteService")
	assert.Contains(t, service, "func (s QuoteService) Price(")
	assert.NotContains(t, service, "client *Client")
	assert.Contains(t, api, "executor Executor")
	assert.False(t, strings.Contains(runtime+service+api, "rawRuntime"))
}

func writeTestOverrides(t *testing.T, sdkNames string, groups string, roles string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sdk-names.yaml"), []byte(strings.TrimSpace(sdkNames)+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "groups.yaml"), []byte(strings.TrimSpace(groups)+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "role-mapping.yaml"), []byte(strings.TrimSpace(roles)+"\n"), 0o644))
	return dir
}

func testCatalog(collections ...catalogCollection) catalog {
	return catalog{SchemaVersion: 1, Source: "test", FetchedAt: "2026-05-23T00:00:00Z", Collections: collections}
}

func testCollection(name string, apis ...catalogAPI) catalogCollection {
	return catalogCollection{ID: goName(name), Name: name, APIs: apis}
}

func testAPI(name string, accessURL string) catalogAPI {
	return catalogAPI{
		ID:          operationIDFromAccessURL(accessURL),
		Name:        name,
		AccessURL:   accessURL,
		HTTPMethod:  "GET",
		RealTRID:    "TR",
		VirtualTRID: "TR",
		OAuth:       true,
		Properties: []catalogProperty{
			{BodyType: "req_b", PropertyCD: "FID_COND_MRKT_DIV_CODE", PropertyName: "시장 분류 코드", PropertyOrder: "1", RequireYN: "Y"},
			{BodyType: "res_b", PropertyCD: "rt_cd", PropertyName: "성공 실패 여부", PropertyOrder: "1", RequireYN: "Y"},
			{BodyType: "res_b", PropertyCD: "msg_cd", PropertyName: "응답코드", PropertyOrder: "2", RequireYN: "Y"},
			{BodyType: "res_b", PropertyCD: "msg1", PropertyName: "응답메세지", PropertyOrder: "3", RequireYN: "Y"},
		},
	}
}
