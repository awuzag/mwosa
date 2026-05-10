# kis

`kis` is a standalone provider client module for the Korea Investment &
Securities KIS Developers OpenAPI.

The SDK uses `github.com/go-resty/resty/v2` as its HTTP client stack. This
module owns KIS endpoint paths, domain selection, OAuth token flows, common
headers such as `tr_id`, typed request/response models, and provider-native
error context. Adapter-level canonical mapping lives in `providers/kis`.

Planning and crawled source documentation live under `docs/`.

## Constructor

Required values and optional behavior are configured with functional options so
the import surface stays small.

```go
client, err := kis.New(
	kis.WithAppKey(appKey),
	kis.WithAppSecret(appSecret),
	kis.WithAccessToken(accessToken),
	kis.WithVirtual(),
)
```

Initial options:

- `WithAppKey(appKey)` sets the KIS app key issued for the selected real or
  virtual service.
- `WithAppSecret(appSecret)` sets the app secret issued with the app key.
- `WithAccessToken(token)` sets an already-issued OAuth access token.
- `WithVirtual()` selects the virtual investment domain and virtual TR IDs.
- `WithBaseURL(url)` overrides the real domain for tests or custom routing.
- `WithVirtualBaseURL(url)` overrides the virtual domain for tests or custom routing.
- `WithAccount(account)` sets account information for account or trading APIs later.

Token issuance is explicit:

```go
token, err := client.Token(ctx)
```

Method arguments should stay minimal. When an endpoint needs extra request
fields, prefer endpoint-specific options instead of widening the primary
signature.

```go
price, err := client.Price(ctx, "005930")

bars, err := client.Daily(ctx, "005930",
	kis.WithPeriod("D"),
	kis.WithDateRange("20250101", "20250131"),
)

minutes, err := client.Intraday(ctx, "005930",
	kis.WithInputHour("100000"),
	kis.WithPastData(true),
)

book, err := client.Orderbook(ctx, "005930")

trades, err := client.Trades(ctx, "005930")

timedTrades, err := client.TimeTrades(ctx, "005930", "141200")

product, err := client.Product(ctx, "005930")

stock, err := client.Stock(ctx, "005930")

etf, err := client.ETFETNPrice(ctx, "069500")
```

Implemented REST market-data APIs:

| Method | KIS API | TR ID |
| --- | --- | --- |
| `Token` | `/oauth2/tokenP` | |
| `Price` | `/uapi/domestic-stock/v1/quotations/inquire-price` | `FHKST01010100` |
| `Daily` | `/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice` | `FHKST03010100` |
| `Intraday` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice` | `FHKST03010200` |
| `Orderbook` | `/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn` | `FHKST01010200` |
| `Trades` | `/uapi/domestic-stock/v1/quotations/inquire-ccnl` | `FHKST01010300` |
| `TimeTrades` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion` | `FHPST01060000` |
| `Product` | `/uapi/domestic-stock/v1/quotations/search-info` | `CTPF1604R` |
| `Stock` | `/uapi/domestic-stock/v1/quotations/search-stock-info` | `CTPF1002R` |
| `ETFETNPrice` | `/uapi/etfetn/v1/quotations/inquire-price` | `FHPST02400000` |

## E2E tests

Live KIS API tests are behind the `e2e` build tag and the `KIS_E2E=1`
environment gate. They are excluded from the default `go test ./...` run.

```bash
cd clients/kis

KIS_E2E=1 \
KIS_APP_KEY="..." \
KIS_APP_SECRET="..." \
KIS_VIRTUAL=1 \
go test -tags=e2e ./...
```

When `KIS_ACCESS_TOKEN` is omitted, the test suite calls `Token` first and uses
the issued token for the remaining requests. Useful optional variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `KIS_ACCESS_TOKEN` | | Reuses an already-issued OAuth token. |
| `KIS_SYMBOL` | `005930` | Domestic stock symbol used by stock quotation tests. |
| `KIS_ETF_SYMBOL` | `069500` | ETF symbol used by ETF/ETN tests. |
| `KIS_DAILY_FROM` | `20250102` | Start date for daily chart tests. |
| `KIS_DAILY_TO` | `20250131` | End date for daily chart tests. |
| `KIS_INPUT_HOUR` | `153000` | Input time for intraday and time-trade tests. |
| `KIS_E2E_TIMEOUT` | `15s` | Per-test context timeout. |
| `KIS_E2E_EXTENDED` | | Set to `1` to run intraday, orderbook, and trade-detail tests. |
| `KIS_E2E_REAL_ONLY` | | Set to `1` to run APIs that are available only on the real domain. |

Real-domain-only tests must be run without the virtual domain:

```bash
cd clients/kis

KIS_E2E=1 \
KIS_E2E_REAL_ONLY=1 \
KIS_VIRTUAL=0 \
KIS_APP_KEY="..." \
KIS_APP_SECRET="..." \
go test -tags=e2e ./...
```
