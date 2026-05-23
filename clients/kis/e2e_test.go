//go:build e2e

package kis_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	kis "github.com/ev3rlit/mwosa/clients/kis"
)

const (
	kisE2EEnabledEnv       = "KIS_E2E"
	kisE2EExtendedEnv      = "KIS_E2E_EXTENDED"
	kisE2ERealOnlyEnv      = "KIS_E2E_REAL_ONLY"
	kisAppKeyEnv           = "KIS_APP_KEY"
	kisAppSecretEnv        = "KIS_APP_SECRET"
	kisAccessTokenEnv      = "KIS_ACCESS_TOKEN"
	kisVirtualEnv          = "KIS_VIRTUAL"
	kisSymbolEnv           = "KIS_SYMBOL"
	kisETFSymbolEnv        = "KIS_ETF_SYMBOL"
	kisDailyFromEnv        = "KIS_DAILY_FROM"
	kisDailyToEnv          = "KIS_DAILY_TO"
	kisInputHourEnv        = "KIS_INPUT_HOUR"
	kisE2ETimeoutEnv       = "KIS_E2E_TIMEOUT"
	defaultKISE2ESymbol    = "005930"
	defaultKISE2EETFSymbol = "069500"
	defaultKISE2EDailyFrom = "20250102"
	defaultKISE2EDailyTo   = "20250131"
	defaultKISE2EInputHour = "153000"
	defaultKISE2ETimeout   = 15 * time.Second
)

func TestE2EPrice(t *testing.T) {
	client, ctx, config := newLiveClient(t)

	price, err := client.Quote().Price(ctx, kis.InquirePriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
	})
	if err != nil {
		t.Fatalf("live price symbol=%s: %v", config.Symbol, err)
	}
	if strings.TrimSpace(price.Output.StckPrpr) == "" {
		t.Fatalf("live price symbol=%s returned empty current price: %+v", config.Symbol, price)
	}
	logSymbol := strings.TrimSpace(price.Output.StckShrnISCD)
	if logSymbol == "" {
		logSymbol = config.Symbol
	}
	t.Logf("symbol=%s current=%s volume=%s", logSymbol, price.Output.StckPrpr, price.Output.AcmlVol)
}

func TestE2EDaily(t *testing.T) {
	client, ctx, config := newLiveClient(t)

	bars, err := client.Quote().Daily(ctx, kis.InquireDailyItemChartPriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
		FidInputDate1:      config.DailyFrom,
		FidInputDate2:      config.DailyTo,
		FidPeriodDivCode:   "D",
		FidOrgAdjPrc:       "0",
	})
	if err != nil {
		t.Fatalf("live daily symbol=%s from=%s to=%s: %v", config.Symbol, config.DailyFrom, config.DailyTo, err)
	}
	if len(bars.Output2) == 0 {
		t.Fatalf("live daily symbol=%s from=%s to=%s returned no bars", config.Symbol, config.DailyFrom, config.DailyTo)
	}
	first := bars.Output2[0]
	t.Logf("symbol=%s first_date=%s close=%s volume=%s count=%d", config.Symbol, first.StckBsopDate, first.StckClpr, first.AcmlVol, len(bars.Output2))
}

func TestE2EExtendedDomesticQuotation(t *testing.T) {
	skipUnlessEnabled(t, kisE2EExtendedEnv, "extended live KIS quotation e2e tests")
	client, ctx, config := newLiveClient(t)

	minutes, err := client.Quote().Intraday(ctx, kis.InquireTimeItemChartPriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
		FidInputHour1:      config.InputHour,
		FidPwDataIncuYn:    "Y",
	})
	if err != nil {
		t.Fatalf("live intraday symbol=%s input_hour=%s: %v", config.Symbol, config.InputHour, err)
	}
	if len(minutes.Output2) == 0 {
		t.Fatalf("live intraday symbol=%s input_hour=%s returned no rows", config.Symbol, config.InputHour)
	}

	book, err := client.Quote().Orderbook(ctx, kis.InquireAskingPriceExpCcnRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
	})
	if err != nil {
		t.Fatalf("live orderbook symbol=%s: %v", config.Symbol, err)
	}
	if strings.TrimSpace(book.Output1.Askp1) == "" || strings.TrimSpace(book.Output1.Bidp1) == "" {
		t.Fatalf("live orderbook symbol=%s returned empty levels: %+v", config.Symbol, book)
	}

	trades, err := client.Quote().Trades(ctx, kis.InquireCcnlRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
	})
	if err != nil {
		t.Fatalf("live trades symbol=%s: %v", config.Symbol, err)
	}
	if len(trades.Output) == 0 {
		t.Fatalf("live trades symbol=%s returned no rows", config.Symbol)
	}

	timedTrades, err := client.Quote().TimeTrades(ctx, kis.InquireTimeItemConclusionRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       config.Symbol,
		FidInputHour1:      config.InputHour,
	})
	if err != nil {
		t.Fatalf("live time trades symbol=%s input_hour=%s: %v", config.Symbol, config.InputHour, err)
	}
	if strings.TrimSpace(timedTrades.Output2.StckCntgHour) == "" {
		t.Fatalf("live time trades symbol=%s input_hour=%s returned no rows", config.Symbol, config.InputHour)
	}

	t.Logf("symbol=%s minutes=%d first_ask=%s first_bid=%s trades=%d timed_trade_time=%s", config.Symbol, len(minutes.Output2), book.Output1.Askp1, book.Output1.Bidp1, len(trades.Output), timedTrades.Output2.StckCntgHour)
}

func TestE2ERealOnlyInstrumentAndETFETN(t *testing.T) {
	skipUnlessEnabled(t, kisE2ERealOnlyEnv, "real-domain-only live KIS e2e tests")
	client, ctx, config := newLiveClient(t)
	if config.Virtual {
		t.Skipf("set %s=0 because Product, Stock, and ETFETNPrice are not virtual-investment APIs", kisVirtualEnv)
	}

	product, err := client.Instrument().Product(ctx, config.Symbol)
	if err != nil {
		t.Fatalf("live product symbol=%s: %v", config.Symbol, err)
	}
	if strings.TrimSpace(product.Name) == "" {
		t.Fatalf("live product symbol=%s returned empty name: %+v", config.Symbol, product)
	}

	stock, err := client.Instrument().Stock(ctx, config.Symbol)
	if err != nil {
		t.Fatalf("live stock symbol=%s: %v", config.Symbol, err)
	}
	if strings.TrimSpace(stock.Name) == "" {
		t.Fatalf("live stock symbol=%s returned empty name: %+v", config.Symbol, stock)
	}

	etf, err := client.ETFETNPrice(ctx, config.ETFSymbol)
	if err != nil {
		t.Fatalf("live ETF/ETN price symbol=%s: %v", config.ETFSymbol, err)
	}
	if strings.TrimSpace(etf.Current) == "" {
		t.Fatalf("live ETF/ETN price symbol=%s returned empty current price: %+v", config.ETFSymbol, etf)
	}

	components, err := client.ETFComponentStockPrices(ctx, config.ETFSymbol)
	if err != nil {
		t.Fatalf("live ETF component stock prices symbol=%s: %v", config.ETFSymbol, err)
	}
	if len(components.Rows) == 0 {
		t.Fatalf("live ETF component stock prices symbol=%s returned no rows: %+v", config.ETFSymbol, components)
	}

	t.Logf("symbol=%s product=%s stock=%s etf_symbol=%s etf_current=%s etf_components=%d", config.Symbol, product.Name, stock.Name, config.ETFSymbol, etf.Current, len(components.Rows))
}

type liveConfig struct {
	AppKey    string
	AppSecret string
	Token     string
	Virtual   bool
	Symbol    string
	ETFSymbol string
	DailyFrom string
	DailyTo   string
	InputHour string
	Timeout   time.Duration
}

func newLiveClient(t *testing.T) (*kis.Client, context.Context, liveConfig) {
	t.Helper()
	skipUnlessEnabled(t, kisE2EEnabledEnv, "live KIS e2e tests")

	config := loadLiveConfig(t)
	options := []kis.Option{
		kis.WithAppKey(config.AppKey),
		kis.WithAppSecret(config.AppSecret),
	}
	if config.Virtual {
		options = append(options, kis.WithVirtual())
	}
	if config.Token != "" {
		options = append(options, kis.WithAccessToken(config.Token))
	}

	client, err := kis.New(options...)
	if err != nil {
		t.Fatalf("new live KIS client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	t.Cleanup(cancel)

	if config.Token == "" {
		token, err := client.Token(ctx)
		if err != nil {
			t.Fatalf("issue live KIS OAuth token: %v", err)
		}
		if strings.TrimSpace(token.AccessToken) == "" {
			t.Fatal("live KIS OAuth token response had empty access token")
		}
	}
	return client, ctx, config
}

func loadLiveConfig(t *testing.T) liveConfig {
	t.Helper()

	config := liveConfig{
		AppKey:    strings.TrimSpace(os.Getenv(kisAppKeyEnv)),
		AppSecret: strings.TrimSpace(os.Getenv(kisAppSecretEnv)),
		Token:     strings.TrimSpace(os.Getenv(kisAccessTokenEnv)),
		Virtual:   envBoolDefault(kisVirtualEnv, true),
		Symbol:    envDefault(kisSymbolEnv, defaultKISE2ESymbol),
		ETFSymbol: envDefault(kisETFSymbolEnv, defaultKISE2EETFSymbol),
		DailyFrom: envDefault(kisDailyFromEnv, defaultKISE2EDailyFrom),
		DailyTo:   envDefault(kisDailyToEnv, defaultKISE2EDailyTo),
		InputHour: envDefault(kisInputHourEnv, defaultKISE2EInputHour),
		Timeout:   envDurationDefault(t, kisE2ETimeoutEnv, defaultKISE2ETimeout),
	}
	if config.AppKey == "" {
		t.Fatalf("%s is required for live KIS e2e tests", kisAppKeyEnv)
	}
	if config.AppSecret == "" {
		t.Fatalf("%s is required for live KIS e2e tests", kisAppSecretEnv)
	}
	return config
}

func skipUnlessEnabled(t *testing.T, envName string, label string) {
	t.Helper()
	if os.Getenv(envName) != "1" {
		t.Skipf("set %s=1 to run %s", envName, label)
	}
}

func envDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBoolDefault(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return fallback
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
