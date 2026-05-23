package kis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/composition"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/intradaybar"
	"github.com/ev3rlit/mwosa/providers/core/orderbook"
	"github.com/ev3rlit/mwosa/providers/core/quote"
	tradesrole "github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/ev3rlit/mwosa/providers/spec"
	"github.com/samber/oops"
)

type Config struct {
	AppKey         string
	AppSecret      string
	AccessToken    string
	BaseURL        string
	VirtualBaseURL string
	Virtual        bool
	CustomerType   string
	Account        string
	TokenCache     TokenCache
}

type marketDataClient interface {
	Token(context.Context) (kisclient.Token, error)
	UseToken(kisclient.Token)
	Quote() quoteAPI
	ETFETNPrice(context.Context, string) (kisclient.ETFETNPrice, error)
	ETFComponentStockPrices(context.Context, string) (kisclient.ETFComponentStockPriceResult, error)
	Instrument() instrumentAPI
}

type quoteAPI interface {
	Price(context.Context, kisclient.InquirePriceRequest) (kisclient.InquirePriceResponse, error)
	Daily(context.Context, kisclient.InquireDailyItemChartPriceRequest) (kisclient.InquireDailyItemChartPriceResponse, error)
	Intraday(context.Context, kisclient.InquireTimeItemChartPriceRequest) (kisclient.InquireTimeItemChartPriceResponse, error)
	Orderbook(context.Context, kisclient.InquireAskingPriceExpCcnRequest) (kisclient.InquireAskingPriceExpCcnResponse, error)
	Trades(context.Context, kisclient.InquireCcnlRequest) (kisclient.InquireCcnlResponse, error)
	TimeTrades(context.Context, kisclient.InquireTimeItemConclusionRequest) (kisclient.InquireTimeItemConclusionResponse, error)
}

type instrumentAPI interface {
	Product(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Product, error)
	Stock(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Stock, error)
}

type Provider struct {
	provider.Identity

	client marketDataClient

	tokenMu             sync.Mutex
	accessTokenProvided bool
	tokenIssued         bool
	tokenCache          TokenCache
	tokenCacheKey       TokenCacheKey
	tokenExpiryBuffer   time.Duration
	now                 func() time.Time

	groups []provider.GroupRoleProvider
}

type ProviderOption func(*Provider)

type kisClientAdapter struct {
	client *kisclient.Client
}

func (a kisClientAdapter) Token(ctx context.Context) (kisclient.Token, error) {
	return a.client.Token(ctx)
}

func (a kisClientAdapter) UseToken(token kisclient.Token) {
	a.client.UseToken(token)
}

func (a kisClientAdapter) Quote() quoteAPI {
	return a.client.Quote()
}

func (a kisClientAdapter) ETFETNPrice(ctx context.Context, symbol string) (kisclient.ETFETNPrice, error) {
	return a.client.ETFETNPrice(ctx, symbol)
}

func (a kisClientAdapter) ETFComponentStockPrices(ctx context.Context, symbol string) (kisclient.ETFComponentStockPriceResult, error) {
	return a.client.ETFComponentStockPrices(ctx, symbol)
}

func (a kisClientAdapter) Instrument() instrumentAPI {
	return a.client.Instrument()
}

func New(config Config) (*Provider, error) {
	errb := oops.In("kis_adapter").With("provider", provider.ProviderKIS)
	options := []kisclient.Option{
		kisclient.WithAppKey(config.AppKey),
		kisclient.WithAppSecret(config.AppSecret),
		kisclient.WithCustomerType(config.CustomerType),
	}
	if strings.TrimSpace(config.AccessToken) != "" {
		options = append(options, kisclient.WithAccessToken(config.AccessToken))
	}
	if strings.TrimSpace(config.BaseURL) != "" {
		options = append(options, kisclient.WithBaseURL(config.BaseURL))
	}
	if strings.TrimSpace(config.VirtualBaseURL) != "" {
		options = append(options, kisclient.WithVirtualBaseURL(config.VirtualBaseURL))
	}
	if config.Virtual {
		options = append(options, kisclient.WithVirtual())
	}
	if strings.TrimSpace(config.Account) != "" {
		options = append(options, kisclient.WithAccount(config.Account))
	}

	client, err := kisclient.New(options...)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	return NewWithClient(
		kisClientAdapter{client: client},
		strings.TrimSpace(config.AccessToken) != "",
		WithTokenCache(config.TokenCache, newTokenCacheKey(config.AppKey, config.Virtual)),
	), nil
}

func NewWithClient(client marketDataClient, accessTokenProvided bool, options ...ProviderOption) *Provider {
	p := &Provider{
		Identity: provider.Identity{
			ID:          provider.ProviderKIS,
			DisplayName: "한국투자증권 KIS",
		},
		client:              client,
		accessTokenProvided: accessTokenProvided,
		tokenExpiryBuffer:   defaultTokenExpiryBuffer,
		now:                 time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(p)
		}
	}
	p.groups = []provider.GroupRoleProvider{
		newQuoteGroup(p.fetchQuoteSnapshot, p.fetchDailyBars, p.fetchIntradayBars, p.fetchOrderbookSnapshot, p.listMarketTrades, p.listConstituents),
		newInstrumentGroup(p.searchInstruments),
	}
	return p
}

func WithTokenCache(cache TokenCache, key TokenCacheKey) ProviderOption {
	return func(p *Provider) {
		p.tokenCache = cache
		p.tokenCacheKey = key
	}
}

func WithClock(now func() time.Time) ProviderOption {
	return func(p *Provider) {
		if now != nil {
			p.now = now
		}
	}
}

func Register(registry *provider.Registry, p provider.IdentityProvider) error {
	return registry.RegisterProvider(p)
}

func (p *Provider) RoleRegistrations() []provider.RoleRegistration {
	if p == nil {
		return nil
	}
	registrations := make([]provider.RoleRegistration, 0)
	for _, group := range p.groups {
		registrations = append(registrations, group.RoleRegistrations()...)
	}
	return registrations
}

func (p *Provider) ensureAccessToken(ctx context.Context) error {
	if p.accessTokenProvided {
		return nil
	}
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	if p.tokenIssued {
		return nil
	}
	if p.client == nil {
		return oops.In("kis_adapter").With("provider", provider.ProviderKIS).New("kis provider client is nil")
	}
	if p.tokenCache != nil {
		cached, ok, err := p.tokenCache.Get(ctx, p.tokenCacheKey)
		if err != nil {
			return oops.In("kis_adapter").With("provider", provider.ProviderKIS).Wrapf(err, "read kis access token cache")
		}
		if ok && cachedTokenValidAt(cached, p.now(), p.tokenExpiryBuffer) {
			p.client.UseToken(cachedTokenToKIS(cached))
			p.tokenIssued = true
			return nil
		}
	}
	token, err := p.client.Token(ctx)
	if err != nil {
		return oops.In("kis_adapter").With("provider", provider.ProviderKIS, "operation", provider.OperationKISPrice).Wrapf(err, "issue kis access token")
	}
	if p.tokenCache != nil {
		cached, err := cachedTokenFromKIS(p.tokenCacheKey, token, p.now())
		if err != nil {
			return oops.In("kis_adapter").With("provider", provider.ProviderKIS).Wrapf(err, "prepare kis access token cache")
		}
		if err := p.tokenCache.Put(ctx, cached); err != nil {
			return oops.In("kis_adapter").With("provider", provider.ProviderKIS).Wrapf(err, "write kis access token cache")
		}
	}
	p.tokenIssued = true
	return nil
}

func (p *Provider) fetchQuoteSnapshot(ctx context.Context, input quote.SnapshotInput) (quote.SnapshotResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleQuote, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol)
	if err := validateMarket(input.Market); err != nil {
		return quote.SnapshotResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return quote.SnapshotResult{}, errb.New("kis quote request requires symbol")
	}
	if p.client == nil {
		return quote.SnapshotResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return quote.SnapshotResult{}, errb.Wrap(err)
	}

	switch input.SecurityType {
	case provider.SecurityTypeStock:
		price, err := p.client.Quote().Price(ctx, kisclient.InquirePriceRequest{
			FidCondMrktDivCode: "J",
			FidInputISCD:       symbol,
		})
		if err != nil {
			return quote.SnapshotResult{}, errb.With("operation", provider.OperationKISPrice).Wrapf(err, "fetch kis quote")
		}
		return quote.SnapshotResult{
			Provider: p.Identity,
			Symbol:   firstNonEmpty(price.Output.StckShrnISCD, symbol),
			Price:    price.Output.StckPrpr,
		}, nil
	case provider.SecurityTypeETF, provider.SecurityTypeETN:
		price, err := p.client.ETFETNPrice(ctx, symbol)
		if err != nil {
			return quote.SnapshotResult{}, errb.With("operation", provider.OperationKISETFETNPrice).Wrapf(err, "fetch kis ETF/ETN quote")
		}
		return quote.SnapshotResult{
			Provider: p.Identity,
			Symbol:   symbol,
			Price:    price.Current,
		}, nil
	default:
		return quote.SnapshotResult{}, unsupportedSecurityTypeError(provider.RoleQuote, input.SecurityType)
	}
}

func (p *Provider) fetchDailyBars(ctx context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleDailyBar, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol, "from", input.From, "to", input.To)
	if err := validateMarket(input.Market); err != nil {
		return dailybar.FetchResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return dailybar.FetchResult{}, errb.New("kis daily request requires symbol")
	}
	if !supportsKISDailySecurityType(input.SecurityType) {
		return dailybar.FetchResult{}, unsupportedSecurityTypeError(provider.RoleDailyBar, input.SecurityType)
	}
	from := normalizeKISDate(input.From)
	to := normalizeKISDate(input.To)
	if from == "" || to == "" {
		return dailybar.FetchResult{}, errb.New("kis daily request requires from and to dates")
	}
	if p.client == nil {
		return dailybar.FetchResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return dailybar.FetchResult{}, errb.Wrap(err)
	}

	bars, err := p.client.Quote().Daily(ctx, kisclient.InquireDailyItemChartPriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       symbol,
		FidInputDate1:      from,
		FidInputDate2:      to,
		FidPeriodDivCode:   "D",
		FidOrgAdjPrc:       "0",
	})
	if err != nil {
		return dailybar.FetchResult{}, errb.With("operation", provider.OperationKISDaily).Wrapf(err, "fetch kis daily bars")
	}
	resultBars := make([]dailybar.Bar, 0, len(bars.Output2))
	for _, bar := range bars.Output2 {
		resultBars = append(resultBars, normalizeDailyBar(bar, symbol, input.SecurityType))
	}
	return dailybar.FetchResult{
		Bars:       resultBars,
		Provider:   p.Identity,
		Group:      provider.GroupKISQuote,
		Operation:  provider.OperationKISDaily,
		TotalCount: len(resultBars),
	}, nil
}

func (p *Provider) fetchIntradayBars(ctx context.Context, input intradaybar.FetchInput) (intradaybar.FetchResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleIntradayBar, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol, "at", input.At, "limit", input.Limit)
	if err := validateMarket(input.Market); err != nil {
		return intradaybar.FetchResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return intradaybar.FetchResult{}, errb.New("kis intraday request requires symbol")
	}
	if !supportsKISMarketDataSecurityType(input.SecurityType) {
		return intradaybar.FetchResult{}, unsupportedSecurityTypeError(provider.RoleIntradayBar, input.SecurityType)
	}
	if p.client == nil {
		return intradaybar.FetchResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return intradaybar.FetchResult{}, errb.Wrap(err)
	}

	inputHour := normalizeKISTime(input.At)
	if inputHour == "" {
		inputHour = "153000"
	}
	bars, err := p.client.Quote().Intraday(ctx, kisclient.InquireTimeItemChartPriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       symbol,
		FidInputHour1:      inputHour,
		FidPwDataIncuYn:    "Y",
	})
	if err != nil {
		return intradaybar.FetchResult{}, errb.With("operation", provider.OperationKISIntraday).Wrapf(err, "fetch kis intraday bars")
	}
	resultBars := make([]intradaybar.Bar, 0, len(bars.Output2))
	for _, bar := range limitSlice(bars.Output2, input.Limit) {
		resultBars = append(resultBars, normalizeIntradayBar(bar, symbol, input.SecurityType))
	}
	return intradaybar.FetchResult{
		Bars:       resultBars,
		Provider:   p.Identity,
		Group:      provider.GroupKISQuote,
		Operation:  provider.OperationKISIntraday,
		TotalCount: len(resultBars),
	}, nil
}

func (p *Provider) fetchOrderbookSnapshot(ctx context.Context, input orderbook.SnapshotInput) (orderbook.SnapshotResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleOrderbook, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol)
	if err := validateMarket(input.Market); err != nil {
		return orderbook.SnapshotResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return orderbook.SnapshotResult{}, errb.New("kis orderbook request requires symbol")
	}
	if !supportsKISMarketDataSecurityType(input.SecurityType) {
		return orderbook.SnapshotResult{}, unsupportedSecurityTypeError(provider.RoleOrderbook, input.SecurityType)
	}
	if p.client == nil {
		return orderbook.SnapshotResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return orderbook.SnapshotResult{}, errb.Wrap(err)
	}

	book, err := p.client.Quote().Orderbook(ctx, kisclient.InquireAskingPriceExpCcnRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       symbol,
	})
	if err != nil {
		return orderbook.SnapshotResult{}, errb.With("operation", provider.OperationKISOrderbook).Wrapf(err, "fetch kis orderbook")
	}
	snapshot := normalizeOrderbook(book, symbol, input.SecurityType)
	return orderbook.SnapshotResult{
		Snapshot:   snapshot,
		Provider:   p.Identity,
		Group:      provider.GroupKISQuote,
		Operation:  provider.OperationKISOrderbook,
		TotalCount: len(snapshot.Levels),
	}, nil
}

func (p *Provider) listMarketTrades(ctx context.Context, input tradesrole.ListInput) (tradesrole.ListResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleTrades, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol, "at", input.At, "limit", input.Limit)
	if err := validateMarket(input.Market); err != nil {
		return tradesrole.ListResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return tradesrole.ListResult{}, errb.New("kis trades request requires symbol")
	}
	if !supportsKISMarketDataSecurityType(input.SecurityType) {
		return tradesrole.ListResult{}, unsupportedSecurityTypeError(provider.RoleTrades, input.SecurityType)
	}
	if p.client == nil {
		return tradesrole.ListResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return tradesrole.ListResult{}, errb.Wrap(err)
	}

	at := normalizeKISTime(input.At)
	if at != "" {
		trades, err := p.client.Quote().TimeTrades(ctx, kisclient.InquireTimeItemConclusionRequest{
			FidCondMrktDivCode: "J",
			FidInputISCD:       symbol,
			FidInputHour1:      at,
		})
		if err != nil {
			return tradesrole.ListResult{}, errb.With("operation", provider.OperationKISTimeTrades).Wrapf(err, "fetch kis time trades")
		}
		resultTrades := make([]tradesrole.Trade, 0, 1)
		for _, trade := range limitSlice([]kisclient.InquireTimeItemConclusionOutput2{trades.Output2}, input.Limit) {
			resultTrades = append(resultTrades, normalizeTimedTrade(trade, symbol, input.SecurityType))
		}
		return tradesrole.ListResult{
			Trades:     resultTrades,
			Provider:   p.Identity,
			Group:      provider.GroupKISQuote,
			Operation:  provider.OperationKISTimeTrades,
			TotalCount: len(resultTrades),
		}, nil
	}

	trades, err := p.client.Quote().Trades(ctx, kisclient.InquireCcnlRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       symbol,
	})
	if err != nil {
		return tradesrole.ListResult{}, errb.With("operation", provider.OperationKISTrades).Wrapf(err, "fetch kis trades")
	}
	resultTrades := make([]tradesrole.Trade, 0, len(trades.Output))
	for _, trade := range limitSlice(trades.Output, input.Limit) {
		resultTrades = append(resultTrades, normalizeTrade(trade, symbol, input.SecurityType))
	}
	return tradesrole.ListResult{
		Trades:     resultTrades,
		Provider:   p.Identity,
		Group:      provider.GroupKISQuote,
		Operation:  provider.OperationKISTrades,
		TotalCount: len(resultTrades),
	}, nil
}

func (p *Provider) listConstituents(ctx context.Context, input composition.ListInput) (composition.ListResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleComposition, "market", input.Market, "security_type", input.SecurityType, "symbol", input.Symbol, "limit", input.Limit)
	if err := validateMarket(input.Market); err != nil {
		return composition.ListResult{}, errb.Wrap(err)
	}
	symbol := strings.TrimSpace(input.Symbol)
	if symbol == "" {
		return composition.ListResult{}, errb.New("kis composition request requires symbol")
	}
	if input.SecurityType != provider.SecurityTypeETF {
		return composition.ListResult{}, unsupportedSecurityTypeError(provider.RoleComposition, input.SecurityType)
	}
	if p.client == nil {
		return composition.ListResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return composition.ListResult{}, errb.Wrap(err)
	}

	result, err := p.client.ETFComponentStockPrices(ctx, symbol)
	if err != nil {
		return composition.ListResult{}, errb.With("operation", provider.OperationKISETFComponentStockPrice).Wrapf(err, "fetch kis ETF component stock prices")
	}
	now := p.now().UTC()
	observedAtMS := now.UnixMilli()
	source := kisComponentSource()
	subject := composition.InstrumentRef{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       symbol,
	}
	members := make([]composition.CompositionMember, 0, len(result.Rows))
	for _, row := range limitSlice(result.Rows, input.Limit) {
		member := compositionMemberFromKISComponentRow(row)
		members = append(members, member)
	}
	return composition.ListResult{
		Composition: composition.Composition{
			Source:       source,
			Subject:      subject,
			AsOfDate:     now.In(koreaLocation()).Format("2006-01-02"),
			ObservedAtMS: observedAtMS,
			Members:      members,
		},
		Provider:   p.Identity,
		Group:      provider.GroupKISQuote,
		Operation:  provider.OperationKISETFComponentStockPrice,
		TotalCount: len(members),
	}, nil
}

func (p *Provider) searchInstruments(ctx context.Context, input instrument.SearchInput) (instrument.SearchResult, error) {
	errb := oops.In("kis_adapter").With("role", provider.RoleInstrument, "market", input.Market, "security_type", input.SecurityType, "query", input.Query)
	if err := validateMarket(input.Market); err != nil {
		return instrument.SearchResult{}, errb.Wrap(err)
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return instrument.SearchResult{}, errb.New("kis instrument search requires query")
	}
	if !supportsKISInstrumentSecurityType(input.SecurityType) {
		return instrument.SearchResult{}, unsupportedSecurityTypeError(provider.RoleInstrument, input.SecurityType)
	}
	if p.client == nil {
		return instrument.SearchResult{}, errb.New("kis provider client is nil")
	}
	if err := p.ensureAccessToken(ctx); err != nil {
		return instrument.SearchResult{}, errb.Wrap(err)
	}

	instrumentService := p.client.Instrument()
	product, err := instrumentService.Product(ctx, query)
	if err != nil {
		return instrument.SearchResult{}, errb.With("operation", provider.OperationKISProduct).Wrapf(err, "fetch kis product")
	}
	item := instrumentFromProduct(product, input.SecurityType)
	operations := []provider.OperationID{provider.OperationKISProduct}
	if input.SecurityType == provider.SecurityTypeStock {
		stock, err := instrumentService.Stock(ctx, query)
		if err != nil {
			return instrument.SearchResult{}, errb.With("operation", provider.OperationKISStock).Wrapf(err, "fetch kis stock")
		}
		item = instrumentFromProductAndStock(product, stock, input.SecurityType)
		operations = append(operations, provider.OperationKISStock)
	}

	return instrument.SearchResult{
		Instruments: []instrument.Instrument{item},
		Provider:    p.Identity,
		Group:       provider.GroupKISInstrument,
		Operations:  operations,
		TotalCount:  1,
	}, nil
}

func normalizeDailyBar(bar kisclient.InquireDailyItemChartPriceOutput2Item, symbol string, securityType provider.SecurityType) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKIS,
		Group:        provider.GroupKISQuote,
		Operation:    provider.OperationKISDaily,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		TradingDate:  normalizeISODate(bar.StckBsopDate),
		Symbol:       symbol,
		Currency:     "KRW",
		Open:         bar.StckOprc,
		High:         bar.StckHgpr,
		Low:          bar.StckLwpr,
		Close:        bar.StckClpr,
		Change:       bar.PrdyVrss,
		Volume:       bar.AcmlVol,
		TradedValue:  bar.AcmlTRPbmn,
		Extensions: map[string]string{
			"previous_change_sign": bar.PrdyVrssSign,
			"lock_code":            bar.FlngClsCode,
			"split_rate":           bar.PrttRate,
			"modified":             bar.ModYn,
			"revaluation_reason":   bar.RevlIssuReas,
		},
	}
}

func normalizeIntradayBar(bar kisclient.InquireTimeItemChartPriceOutput2Item, symbol string, securityType provider.SecurityType) intradaybar.Bar {
	return intradaybar.Bar{
		Provider:     provider.ProviderKIS,
		Group:        provider.GroupKISQuote,
		Operation:    provider.OperationKISIntraday,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		TradingDate:  normalizeISODate(bar.StckBsopDate),
		Time:         normalizeISOTime(bar.StckCntgHour),
		Symbol:       symbol,
		Currency:     "KRW",
		Open:         bar.StckOprc,
		High:         bar.StckHgpr,
		Low:          bar.StckLwpr,
		Close:        bar.StckPrpr,
		Volume:       bar.CntgVol,
		TradedValue:  bar.AcmlTRPbmn,
	}
}

func normalizeOrderbook(book kisclient.InquireAskingPriceExpCcnResponse, symbol string, securityType provider.SecurityType) orderbook.Snapshot {
	asks := []struct {
		price    string
		quantity string
		delta    string
	}{
		{book.Output1.Askp1, book.Output1.AskpRsqn1, book.Output1.AskpRsqnIcdc1},
		{book.Output1.Askp2, book.Output1.AskpRsqn2, book.Output1.AskpRsqnIcdc2},
		{book.Output1.Askp3, book.Output1.AskpRsqn3, book.Output1.AskpRsqnIcdc3},
		{book.Output1.Askp4, book.Output1.AskpRsqn4, book.Output1.AskpRsqnIcdc4},
		{book.Output1.Askp5, book.Output1.AskpRsqn5, book.Output1.AskpRsqnIcdc5},
		{book.Output1.Askp6, book.Output1.AskpRsqn6, book.Output1.AskpRsqnIcdc6},
		{book.Output1.Askp7, book.Output1.AskpRsqn7, book.Output1.AskpRsqnIcdc7},
		{book.Output1.Askp8, book.Output1.AskpRsqn8, book.Output1.AskpRsqnIcdc8},
		{book.Output1.Askp9, book.Output1.AskpRsqn9, book.Output1.AskpRsqnIcdc9},
		{book.Output1.Askp10, book.Output1.AskpRsqn10, book.Output1.AskpRsqnIcdc10},
	}
	bids := []struct {
		price    string
		quantity string
		delta    string
	}{
		{book.Output1.Bidp1, book.Output1.BidpRsqn1, book.Output1.BidpRsqnIcdc1},
		{book.Output1.Bidp2, book.Output1.BidpRsqn2, book.Output1.BidpRsqnIcdc2},
		{book.Output1.Bidp3, book.Output1.BidpRsqn3, book.Output1.BidpRsqnIcdc3},
		{book.Output1.Bidp4, book.Output1.BidpRsqn4, book.Output1.BidpRsqnIcdc4},
		{book.Output1.Bidp5, book.Output1.BidpRsqn5, book.Output1.BidpRsqnIcdc5},
		{book.Output1.Bidp6, book.Output1.BidpRsqn6, book.Output1.BidpRsqnIcdc6},
		{book.Output1.Bidp7, book.Output1.BidpRsqn7, book.Output1.BidpRsqnIcdc7},
		{book.Output1.Bidp8, book.Output1.BidpRsqn8, book.Output1.BidpRsqnIcdc8},
		{book.Output1.Bidp9, book.Output1.BidpRsqn9, book.Output1.BidpRsqnIcdc9},
		{book.Output1.Bidp10, book.Output1.BidpRsqn10, book.Output1.BidpRsqnIcdc10},
	}
	levels := make([]orderbook.Level, 0, len(asks)+len(bids))
	for i, level := range asks {
		if strings.TrimSpace(level.price) == "" && strings.TrimSpace(level.quantity) == "" {
			continue
		}
		levels = append(levels, orderbook.Level{
			Side:          orderbook.SideAsk,
			Level:         i + 1,
			Price:         level.price,
			Quantity:      level.quantity,
			QuantityDelta: level.delta,
		})
	}
	for i, level := range bids {
		if strings.TrimSpace(level.price) == "" && strings.TrimSpace(level.quantity) == "" {
			continue
		}
		levels = append(levels, orderbook.Level{
			Side:          orderbook.SideBid,
			Level:         i + 1,
			Price:         level.price,
			Quantity:      level.quantity,
			QuantityDelta: level.delta,
		})
	}
	return orderbook.Snapshot{
		Provider:         provider.ProviderKIS,
		Group:            provider.GroupKISQuote,
		Operation:        provider.OperationKISOrderbook,
		Market:           provider.MarketKRX,
		SecurityType:     securityType,
		Symbol:           firstNonEmpty(book.Output2.StckShrnISCD, symbol),
		AcceptanceTime:   normalizeISOTime(book.Output1.AsprAcptHour),
		Currency:         "KRW",
		Levels:           levels,
		TotalAskQuantity: book.Output1.TotalAskpRsqn,
		TotalBidQuantity: book.Output1.TotalBidpRsqn,
		Expected: orderbook.ExpectedConclusion{
			Price:              book.Output2.AntcCnpr,
			Volume:             book.Output2.AntcVol,
			Current:            book.Output2.StckPrpr,
			Open:               book.Output2.StckOprc,
			High:               book.Output2.StckHgpr,
			Low:                book.Output2.StckLwpr,
			PreviousClose:      book.Output2.StckSdpr,
			PreviousChange:     book.Output2.AntcCntgVrss,
			PreviousChangeSign: book.Output2.AntcCntgVrssSign,
			PreviousChangeRate: book.Output2.AntcCntgPrdyCtrt,
		},
	}
}

func normalizeTrade(trade kisclient.InquireCcnlOutputItem, symbol string, securityType provider.SecurityType) tradesrole.Trade {
	return tradesrole.Trade{
		Provider:           provider.ProviderKIS,
		Group:              provider.GroupKISQuote,
		Operation:          provider.OperationKISTrades,
		Market:             provider.MarketKRX,
		SecurityType:       securityType,
		Symbol:             symbol,
		Time:               normalizeISOTime(trade.StckCntgHour),
		Price:              trade.StckPrpr,
		Volume:             trade.CntgVol,
		PreviousChange:     trade.PrdyVrss,
		PreviousChangeSign: trade.PrdyVrssSign,
		PreviousChangeRate: trade.PrdyCtrt,
		Strength:           trade.TdayRltv,
	}
}

func normalizeTimedTrade(trade kisclient.InquireTimeItemConclusionOutput2, symbol string, securityType provider.SecurityType) tradesrole.Trade {
	return tradesrole.Trade{
		Provider:           provider.ProviderKIS,
		Group:              provider.GroupKISQuote,
		Operation:          provider.OperationKISTimeTrades,
		Market:             provider.MarketKRX,
		SecurityType:       securityType,
		Symbol:             symbol,
		Time:               normalizeISOTime(trade.StckCntgHour),
		Price:              trade.StckPbpr,
		Volume:             trade.Cnqn,
		AccumulatedVolume:  trade.AcmlVol,
		Ask:                trade.Askp,
		Bid:                trade.Bidp,
		PreviousChange:     trade.PrdyVrss,
		PreviousChangeSign: trade.PrdyVrssSign,
		PreviousChangeRate: trade.PrdyCtrt,
		Strength:           trade.TdayRltv,
	}
}

func compositionMemberFromKISComponentRow(row kisclient.ETFComponentStockPrice) composition.CompositionMember {
	instrument := composition.InstrumentRef{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       row.Symbol,
		Name:         row.Name,
	}
	return composition.CompositionMember{
		Instrument: instrument,
		Weight:     decimalValue(row.Weight),
		Quantity:   decimalValue(row.Quantity),
		Valuation:  moneyValue(row.ValuationAmount),
	}
}

func kisComponentSource() composition.SourceRef {
	return composition.SourceRef{
		Provider:  provider.ProviderKIS,
		Group:     provider.GroupKISQuote,
		Operation: provider.OperationKISETFComponentStockPrice,
	}
}

func moneyValue(value string) composition.MoneyValue {
	value = strings.TrimSpace(value)
	if value == "" {
		return composition.MoneyValue{}
	}
	return composition.MoneyValue{Currency: "KRW", Value: value}
}

func decimalValue(value string) composition.DecimalValue {
	return composition.DecimalValue{Value: strings.TrimSpace(value)}
}

func koreaLocation() *time.Location {
	return time.FixedZone("Asia/Seoul", 9*60*60)
}

func instrumentFromProduct(product kisclient.Product, securityType provider.SecurityType) instrument.Instrument {
	securityCode := firstNonEmpty(product.ShortProductNo, product.ProductNo)
	return instrument.Instrument{
		Provider:     provider.ProviderKIS,
		Group:        provider.GroupKISInstrument,
		Operation:    provider.OperationKISProduct,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		SecurityCode: securityCode,
		ISIN:         product.StandardProductNo,
		Name:         firstNonEmpty(product.Name, product.AbbreviatedName),
		ExchangeCode: "KRX",
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions: map[string]string{
			"security_key":             fmt.Sprintf("krx:%s", securityCode),
			"canonical_record_key":     fmt.Sprintf("instrument:krx:%s:current", securityCode),
			"kis_product_no":           product.ProductNo,
			"kis_product_type_code":    product.ProductTypeCode,
			"kis_product_class_code":   product.ProductClassCode,
			"kis_product_class_name":   product.ProductClassName,
			"kis_investment_type_code": product.InvestmentTypeCode,
			"kis_investment_type_name": product.InvestmentTypeCodeName,
			"english_name":             product.EnglishName,
		},
	}
}

func instrumentFromProductAndStock(product kisclient.Product, stock kisclient.Stock, securityType provider.SecurityType) instrument.Instrument {
	item := instrumentFromProduct(product, securityType)
	item.Operation = provider.OperationKISStock
	item.SecurityCode = firstNonEmpty(stock.ProductNo, item.SecurityCode)
	item.ISIN = firstNonEmpty(stock.StandardProductNo, item.ISIN)
	item.Name = firstNonEmpty(stock.Name, item.Name)
	item.Extensions["kis_market_id_code"] = stock.MarketIDCode
	item.Extensions["kis_security_group_id_code"] = stock.SecurityGroupIDCode
	item.Extensions["kis_exchange_division_code"] = stock.ExchangeDivisionCode
	item.Extensions["kis_settlement_month_day"] = stock.SettlementMonthDay
	item.Extensions["kis_listed_shares"] = stock.ListedShares
	item.Extensions["kis_capital"] = stock.Capital
	item.Extensions["kis_par_value"] = stock.ParValue
	item.Extensions["kis_trading_halt"] = stock.TradingHalt
	item.Extensions["kis_administrative_issue"] = stock.AdministrativeIssue
	item.Extensions["kis_industry_code"] = stock.IndustryCode
	item.Extensions["kis_industry_name"] = stock.IndustryName
	return item
}

func validateMarket(market provider.Market) error {
	if market == "" || market == provider.MarketKRX {
		return nil
	}
	return oops.In("kis_adapter").With("market", market).Errorf("kis provider supports only market=%s: %s", provider.MarketKRX, market)
}

func supportsKISDailySecurityType(securityType provider.SecurityType) bool {
	switch securityType {
	case provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN:
		return true
	default:
		return false
	}
}

func supportsKISMarketDataSecurityType(securityType provider.SecurityType) bool {
	switch securityType {
	case provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN:
		return true
	default:
		return false
	}
}

func supportsKISInstrumentSecurityType(securityType provider.SecurityType) bool {
	switch securityType {
	case provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW:
		return true
	default:
		return false
	}
}

func unsupportedSecurityTypeError(role provider.Role, securityType provider.SecurityType) error {
	return oops.In("kis_adapter").
		With("role", role, "security_type", securityType).
		Errorf("kis provider does not support role=%s security_type=%s", role, securityType)
}

func normalizeKISDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 10 && value[4] == '-' && value[7] == '-' {
		return value[:4] + value[5:7] + value[8:10]
	}
	return value
}

func normalizeISODate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 {
		return value[:4] + "-" + value[4:6] + "-" + value[6:8]
	}
	return value
}

func normalizeKISTime(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 && value[2] == ':' && value[5] == ':' {
		return value[:2] + value[3:5] + value[6:8]
	}
	return value
}

func normalizeISOTime(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 6 {
		return value[:2] + ":" + value[2:4] + ":" + value[4:6]
	}
	return value
}

func limitSlice[T any](values []T, limit int) []T {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type quoteGroup struct {
	quoteSnapshotter     quote.Snapshotter
	dailyFetcher         dailybar.Fetcher
	intradayFetcher      intradaybar.Fetcher
	orderbookSnapshotter orderbook.Snapshotter
	tradesLister         tradesrole.Lister
	compositionLister    composition.Lister
}

var _ provider.GroupRoleProvider = quoteGroup{}

func newQuoteGroup(snapshot quote.SnapshotFunc, fetch dailybar.FetchFunc, fetchIntraday intradaybar.FetchFunc, snapshotOrderbook orderbook.SnapshotFunc, listTrades tradesrole.ListFunc, listConstituents composition.ListFunc) quoteGroup {
	return quoteGroup{
		quoteSnapshotter: quote.NewSnapshot(
			spec.Quote().
				Markets(provider.MarketKRX).
				SecurityTypes(provider.SecurityTypeStock).
				Group(provider.GroupKISQuote).
				Operations(provider.OperationKISPrice).
				RequiresAuth(provider.CredentialScopeKIS).
				Freshness(provider.FreshnessDaily).
				Compatibility(spec.Realtime()).
				Priority(100).
				Limitations("stock quote snapshots only expose the canonical price field for now").
				MustBuild(),
			snapshot,
		),
		dailyFetcher: dailybar.NewFetch(
			spec.DailyBar().
				Markets(provider.MarketKRX).
				SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
				Group(provider.GroupKISQuote).
				Operations(provider.OperationKISDaily).
				RequiresAuth(provider.CredentialScopeKIS).
				Freshness(provider.FreshnessDaily).
				CompatibilityValue(provider.Compatibility{
					DataLatency:         provider.DataLatencyEndOfDay,
					CurrentDaySupported: true,
					Notes: []string{
						"KIS daily bars are symbol-scoped and support D/W/M/Y provider periods; mwosa maps only D into daily_bar",
					},
				}).
				RangeQuery(dailybar.RangeQuerySupported).
				Priority(80).
				Limitations(
					"symbol-scoped daily bars only; provider-wide sync/backfill batches are not supported",
					"weekly, monthly, and yearly KIS periods are not exposed through canonical daily_bar",
				).
				MustBuild(),
			fetch,
		),
		intradayFetcher: spec.RealtimeIntradayBar(fetchIntraday).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISQuote).
			Operations(provider.OperationKISIntraday).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS intraday bars are same-day symbol-scoped minute bars").
			Priority(90).
			Limitations("symbol-scoped same-day minute bars only; no historical multi-day intraday backfill").
			MustBuild(),
		orderbookSnapshotter: spec.RealtimeOrderbook(snapshotOrderbook).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISQuote).
			Operations(provider.OperationKISOrderbook).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS orderbook snapshot returns up to 10 ask and 10 bid levels with expected conclusion data").
			Priority(90).
			Limitations("single symbol snapshot only; no streaming orderbook updates").
			MustBuild(),
		tradesLister: spec.RealtimeTrades(listTrades).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISQuote).
			Operations(provider.OperationKISTrades, provider.OperationKISTimeTrades).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS trade rows are market trade prints, not account order executions").
			Priority(90).
			Limitations("recent same-day market trades only; account executions are intentionally out of scope").
			MustBuild(),
		compositionLister: composition.NewList(composition.Profile{
			Markets:       []provider.Market{provider.MarketKRX},
			SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
			Group:         provider.GroupKISQuote,
			Operations:    []provider.OperationID{provider.OperationKISETFComponentStockPrice},
			AuthScope:     provider.CredentialScopeKIS,
			Freshness:     provider.FreshnessDaily,
			Compatibility: provider.Compatibility{
				DataLatency:         provider.DataLatencyRealtime,
				CurrentDaySupported: true,
				Notes:               []string{"KIS ETF component stock price API is adapted into canonical composition members"},
			},
			RequiresAuth: true,
			Priority:     90,
			Limitations: []string{
				"live read-through composition command; canonical composition storage is handled by the storage repository path",
				"ETF only; ETN and ELW component rows are not registered",
			},
		}, listConstituents),
	}
}

func (g quoteGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKISQuote
}

func (g quoteGroup) RoleRegistrations() []provider.RoleRegistration {
	stockQuote := g.quoteSnapshotter.RoleRegistration()
	etpQuote := quote.NewSnapshot(
		spec.Quote().
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISQuote).
			Operations(provider.OperationKISETFETNPrice).
			RequiresAuth(provider.CredentialScopeKIS).
			Freshness(provider.FreshnessDaily).
			Compatibility(spec.Realtime()).
			Priority(100).
			Limitations("ETF/ETN quote snapshots only expose the canonical price field for now").
			MustBuild(),
		g.quoteSnapshotter.FetchQuoteSnapshot,
	).RoleRegistration()
	return []provider.RoleRegistration{
		stockQuote,
		etpQuote,
		g.dailyFetcher.RoleRegistration(),
		g.intradayFetcher.RoleRegistration(),
		g.orderbookSnapshotter.RoleRegistration(),
		g.tradesLister.RoleRegistration(),
		g.compositionLister.RoleRegistration(),
	}
}

type instrumentGroup struct {
	instrument.Searcher
}

var _ provider.GroupRoleProvider = instrumentGroup{}

func newInstrumentGroup(search instrument.SearchFunc) instrumentGroup {
	return instrumentGroup{
		Searcher: spec.InstrumentSearcher(search).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW).
			Group(provider.GroupKISInstrument).
			Operations(provider.OperationKISProduct, provider.OperationKISStock).
			RequiresAuth(provider.CredentialScopeKIS).
			Freshness(provider.FreshnessDaily).
			Compatibility(spec.EndOfDay().Notes("KIS product and stock metadata are point-in-time lookup APIs")).
			Priority(90).
			Limitations(
				"current adapter performs exact product-number lookup, not whole-universe listing",
				"stock detail enrichment is used only for security_type=stock",
			).
			MustBuild(),
	}
}

func (g instrumentGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKISInstrument
}

func (g instrumentGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Searcher.RoleRegistration()}
}
