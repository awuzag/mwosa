package kis

import (
	"context"
	"testing"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/quote"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistersReadOnlyKISRoles(t *testing.T) {
	p := NewWithClient(&fakeKISClient{}, true)

	registrations := p.RoleRegistrations()

	require.Len(t, registrations, 4)
	roles := map[provider.Role]int{}
	for _, registration := range registrations {
		require.Equal(t, provider.ProviderKIS, p.ProviderIdentity().ID)
		require.True(t, registration.Profile.RequiresAuth)
		require.Equal(t, provider.CredentialScopeKIS, registration.Profile.AuthScope)
		roles[registration.Profile.Role]++
	}
	require.Equal(t, 2, roles[provider.RoleQuote])
	require.Equal(t, 1, roles[provider.RoleDailyBar])
	require.Equal(t, 1, roles[provider.RoleInstrument])
}

func TestFetchStockQuoteUsesPrice(t *testing.T) {
	client := &fakeKISClient{
		price: kisclient.Price{
			Symbol:  "005930",
			Current: "75000",
		},
	}
	p := NewWithClient(client, false)

	result, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tokenCalls)
	require.Equal(t, 1, client.priceCalls)
	require.Equal(t, "005930", result.Symbol)
	require.Equal(t, "75000", result.Price)
}

func TestFetchETFQuoteUsesETFETNPrice(t *testing.T) {
	client := &fakeKISClient{
		etfetnPrice: kisclient.ETFETNPrice{
			Current: "10250",
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})

	require.NoError(t, err)
	require.Equal(t, 0, client.tokenCalls)
	require.Equal(t, 1, client.etfetnCalls)
	require.Equal(t, "069500", result.Symbol)
	require.Equal(t, "10250", result.Price)
}

func TestFetchDailyBarsNormalizesKISBars(t *testing.T) {
	client := &fakeKISClient{
		bars: []kisclient.Bar{
			{
				Date:               "20260508",
				Open:               "70000",
				High:               "76000",
				Low:                "69000",
				Close:              "75000",
				PreviousChange:     "1000",
				PreviousChangeSign: "2",
				Volume:             "12345",
				Amount:             "98765",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		From:         "2026-05-08",
		To:           "2026-05-08",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.dailyCalls)
	require.Len(t, result.Bars, 1)
	bar := result.Bars[0]
	require.Equal(t, provider.ProviderKIS, bar.Provider)
	require.Equal(t, provider.GroupKISDomesticStockQuotation, bar.Group)
	require.Equal(t, provider.OperationKISDaily, bar.Operation)
	require.Equal(t, "2026-05-08", bar.TradingDate)
	require.Equal(t, "005930", bar.Symbol)
	require.Equal(t, "75000", bar.Close)
	require.Equal(t, "1000", bar.Change)
}

func TestSearchStockInstrumentCombinesProductAndStock(t *testing.T) {
	client := &fakeKISClient{
		product: kisclient.Product{
			ProductNo:         "005930",
			Name:              "Samsung Electronics",
			StandardProductNo: "KR7005930003",
			ProductTypeCode:   "300",
			ProductClassCode:  "STK",
		},
		stock: kisclient.Stock{
			ProductNo:           "005930",
			ProductTypeCode:     "300",
			StandardProductNo:   "KR7005930003",
			Name:                "Samsung Electronics",
			MarketIDCode:        "STK",
			SecurityGroupIDCode: "ST",
			ListedShares:        "5969782550",
			IndustryName:        "Semiconductors",
		},
	}
	p := NewWithClient(client, true)

	result, err := p.searchInstruments(context.Background(), instrument.SearchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Query:        "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.productCalls)
	require.Equal(t, 1, client.stockCalls)
	require.Len(t, result.Instruments, 1)
	item := result.Instruments[0]
	require.Equal(t, provider.ProviderKIS, item.Provider)
	require.Equal(t, provider.OperationKISStock, item.Operation)
	require.Equal(t, "005930", item.SecurityCode)
	require.Equal(t, "KR7005930003", item.ISIN)
	require.Equal(t, "Semiconductors", item.Extensions["kis_industry_name"])
}

func TestBuilderUsesEnvCredentialFallbacks(t *testing.T) {
	builder := NewBuilder()
	config := provider.Config{
		"env": map[string]any{
			"KIS_APP_KEY":    "key",
			"KIS_APP_SECRET": "secret",
		},
	}

	decision := builder.Decide(provider.RegisterOptions{}, config)
	require.True(t, decision.Register)

	instance, err := builder.Build(config)
	require.NoError(t, err)
	require.Equal(t, provider.ProviderKIS, instance.ProviderIdentity().ID)
}

type fakeKISClient struct {
	tokenCalls int

	priceCalls   int
	etfetnCalls  int
	dailyCalls   int
	productCalls int
	stockCalls   int

	price       kisclient.Price
	etfetnPrice kisclient.ETFETNPrice
	bars        []kisclient.Bar
	product     kisclient.Product
	stock       kisclient.Stock
}

func (c *fakeKISClient) Token(context.Context) (kisclient.Token, error) {
	c.tokenCalls++
	return kisclient.Token{AccessToken: "token"}, nil
}

func (c *fakeKISClient) Price(context.Context, string) (kisclient.Price, error) {
	c.priceCalls++
	return c.price, nil
}

func (c *fakeKISClient) ETFETNPrice(context.Context, string) (kisclient.ETFETNPrice, error) {
	c.etfetnCalls++
	return c.etfetnPrice, nil
}

func (c *fakeKISClient) Daily(_ context.Context, _ string, options ...kisclient.DailyOption) ([]kisclient.Bar, error) {
	c.dailyCalls++
	return c.bars, nil
}

func (c *fakeKISClient) Product(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Product, error) {
	c.productCalls++
	return c.product, nil
}

func (c *fakeKISClient) Stock(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Stock, error) {
	c.stockCalls++
	return c.stock, nil
}
