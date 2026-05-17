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
	Price(context.Context, string) (kisclient.Price, error)
	ETFETNPrice(context.Context, string) (kisclient.ETFETNPrice, error)
	ETFComponentStockPrices(context.Context, string) (kisclient.ETFComponentStockPriceResult, error)
	Daily(context.Context, string, ...kisclient.DailyOption) ([]kisclient.Bar, error)
	Intraday(context.Context, string, ...kisclient.IntradayOption) ([]kisclient.IntradayBar, error)
	Orderbook(context.Context, string) (kisclient.Orderbook, error)
	Trades(context.Context, string) ([]kisclient.Trade, error)
	TimeTrades(context.Context, string, string) ([]kisclient.TimedTrade, error)
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
		client,
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
		newDomesticStockQuotationGroup(p.fetchQuoteSnapshot, p.fetchDailyBars, p.fetchIntradayBars, p.fetchOrderbookSnapshot, p.listMarketTrades, p.listConstituents),
		newDomesticStockInstrumentGroup(p.searchInstruments),
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
		price, err := p.client.Price(ctx, symbol)
		if err != nil {
			return quote.SnapshotResult{}, errb.With("operation", provider.OperationKISPrice).Wrapf(err, "fetch kis quote")
		}
		return quote.SnapshotResult{
			Provider: p.Identity,
			Symbol:   firstNonEmpty(price.Symbol, symbol),
			Price:    price.Current,
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

	bars, err := p.client.Daily(ctx, symbol,
		kisclient.WithPeriod("D"),
		kisclient.WithDateRange(from, to),
	)
	if err != nil {
		return dailybar.FetchResult{}, errb.With("operation", provider.OperationKISDaily).Wrapf(err, "fetch kis daily bars")
	}
	resultBars := make([]dailybar.Bar, 0, len(bars))
	for _, bar := range bars {
		resultBars = append(resultBars, normalizeDailyBar(bar, symbol, input.SecurityType))
	}
	return dailybar.FetchResult{
		Bars:       resultBars,
		Provider:   p.Identity,
		Group:      provider.GroupKISDomesticStockQuotation,
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

	options := make([]kisclient.IntradayOption, 0, 1)
	if at := normalizeKISTime(input.At); at != "" {
		options = append(options, kisclient.WithInputHour(at))
	}
	bars, err := p.client.Intraday(ctx, symbol, options...)
	if err != nil {
		return intradaybar.FetchResult{}, errb.With("operation", provider.OperationKISIntraday).Wrapf(err, "fetch kis intraday bars")
	}
	resultBars := make([]intradaybar.Bar, 0, len(bars))
	for _, bar := range limitSlice(bars, input.Limit) {
		resultBars = append(resultBars, normalizeIntradayBar(bar, symbol, input.SecurityType))
	}
	return intradaybar.FetchResult{
		Bars:       resultBars,
		Provider:   p.Identity,
		Group:      provider.GroupKISDomesticStockQuotation,
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

	book, err := p.client.Orderbook(ctx, symbol)
	if err != nil {
		return orderbook.SnapshotResult{}, errb.With("operation", provider.OperationKISOrderbook).Wrapf(err, "fetch kis orderbook")
	}
	snapshot := normalizeOrderbook(book, symbol, input.SecurityType)
	return orderbook.SnapshotResult{
		Snapshot:   snapshot,
		Provider:   p.Identity,
		Group:      provider.GroupKISDomesticStockQuotation,
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
		trades, err := p.client.TimeTrades(ctx, symbol, at)
		if err != nil {
			return tradesrole.ListResult{}, errb.With("operation", provider.OperationKISTimeTrades).Wrapf(err, "fetch kis time trades")
		}
		resultTrades := make([]tradesrole.Trade, 0, len(trades))
		for _, trade := range limitSlice(trades, input.Limit) {
			resultTrades = append(resultTrades, normalizeTimedTrade(trade, symbol, input.SecurityType))
		}
		return tradesrole.ListResult{
			Trades:     resultTrades,
			Provider:   p.Identity,
			Group:      provider.GroupKISDomesticStockQuotation,
			Operation:  provider.OperationKISTimeTrades,
			TotalCount: len(resultTrades),
		}, nil
	}

	trades, err := p.client.Trades(ctx, symbol)
	if err != nil {
		return tradesrole.ListResult{}, errb.With("operation", provider.OperationKISTrades).Wrapf(err, "fetch kis trades")
	}
	resultTrades := make([]tradesrole.Trade, 0, len(trades))
	for _, trade := range limitSlice(trades, input.Limit) {
		resultTrades = append(resultTrades, normalizeTrade(trade, symbol, input.SecurityType))
	}
	return tradesrole.ListResult{
		Trades:     resultTrades,
		Provider:   p.Identity,
		Group:      provider.GroupKISDomesticStockQuotation,
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
	quotes := make([]composition.QuoteObservation, 0, len(result.Rows)+1)
	if subjectQuote := quoteObservationFromHeader(result.Output1, subject, observedAtMS); subjectQuote.Price.Value != "" {
		quotes = append(quotes, subjectQuote)
	}
	for _, row := range limitSlice(result.Rows, input.Limit) {
		member, quote := compositionMemberFromKISComponentRow(row, observedAtMS)
		members = append(members, member)
		if quote.Price.Value != "" || quote.Volume.Value != "" {
			quotes = append(quotes, quote)
		}
	}
	return composition.ListResult{
		Composition: composition.Composition{
			Source:       source,
			Subject:      subject,
			AsOfDate:     now.In(koreaLocation()).Format("2006-01-02"),
			ObservedAtMS: observedAtMS,
			Members:      members,
		},
		QuoteObservations: quotes,
		Provider:          p.Identity,
		Group:             provider.GroupKISDomesticStockQuotation,
		Operation:         provider.OperationKISETFComponentStockPrice,
		TotalCount:        len(members),
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

	product, err := p.client.Product(ctx, query)
	if err != nil {
		return instrument.SearchResult{}, errb.With("operation", provider.OperationKISProduct).Wrapf(err, "fetch kis product")
	}
	item := instrumentFromProduct(product, input.SecurityType)
	operations := []provider.OperationID{provider.OperationKISProduct}
	if input.SecurityType == provider.SecurityTypeStock {
		stock, err := p.client.Stock(ctx, query)
		if err != nil {
			return instrument.SearchResult{}, errb.With("operation", provider.OperationKISStock).Wrapf(err, "fetch kis stock")
		}
		item = instrumentFromProductAndStock(product, stock, input.SecurityType)
		operations = append(operations, provider.OperationKISStock)
	}

	return instrument.SearchResult{
		Instruments: []instrument.Instrument{item},
		Provider:    p.Identity,
		Group:       provider.GroupKISDomesticStockInstrument,
		Operations:  operations,
		TotalCount:  1,
	}, nil
}

func normalizeDailyBar(bar kisclient.Bar, symbol string, securityType provider.SecurityType) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKIS,
		Group:        provider.GroupKISDomesticStockQuotation,
		Operation:    provider.OperationKISDaily,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		TradingDate:  normalizeISODate(bar.Date),
		Symbol:       symbol,
		Currency:     "KRW",
		Open:         bar.Open,
		High:         bar.High,
		Low:          bar.Low,
		Close:        bar.Close,
		Change:       bar.PreviousChange,
		Volume:       bar.Volume,
		TradedValue:  bar.Amount,
		Extensions: map[string]string{
			"previous_change_sign": bar.PreviousChangeSign,
			"lock_code":            bar.Raw.LockCode,
			"split_rate":           bar.Raw.SplitRate,
			"modified":             bar.Raw.Modified,
			"revaluation_reason":   bar.Raw.RevaluationReason,
		},
	}
}

func normalizeIntradayBar(bar kisclient.IntradayBar, symbol string, securityType provider.SecurityType) intradaybar.Bar {
	return intradaybar.Bar{
		Provider:     provider.ProviderKIS,
		Group:        provider.GroupKISDomesticStockQuotation,
		Operation:    provider.OperationKISIntraday,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		TradingDate:  normalizeISODate(bar.Date),
		Time:         normalizeISOTime(bar.Time),
		Symbol:       symbol,
		Currency:     "KRW",
		Open:         bar.Open,
		High:         bar.High,
		Low:          bar.Low,
		Close:        bar.Current,
		Volume:       bar.Volume,
		TradedValue:  bar.Amount,
	}
}

func normalizeOrderbook(book kisclient.Orderbook, symbol string, securityType provider.SecurityType) orderbook.Snapshot {
	levels := make([]orderbook.Level, 0, len(book.Asks)+len(book.Bids))
	for i, level := range book.Asks {
		levels = append(levels, orderbook.Level{
			Side:          orderbook.SideAsk,
			Level:         i + 1,
			Price:         level.Price,
			Quantity:      level.Quantity,
			QuantityDelta: level.Delta,
		})
	}
	for i, level := range book.Bids {
		levels = append(levels, orderbook.Level{
			Side:          orderbook.SideBid,
			Level:         i + 1,
			Price:         level.Price,
			Quantity:      level.Quantity,
			QuantityDelta: level.Delta,
		})
	}
	return orderbook.Snapshot{
		Provider:         provider.ProviderKIS,
		Group:            provider.GroupKISDomesticStockQuotation,
		Operation:        provider.OperationKISOrderbook,
		Market:           provider.MarketKRX,
		SecurityType:     securityType,
		Symbol:           firstNonEmpty(book.Expected.Symbol, symbol),
		AcceptanceTime:   normalizeISOTime(book.AcceptanceTime),
		Currency:         "KRW",
		Levels:           levels,
		TotalAskQuantity: book.TotalAskQuantity,
		TotalBidQuantity: book.TotalBidQuantity,
		Expected: orderbook.ExpectedConclusion{
			Price:              book.Expected.ExpectedPrice,
			Volume:             book.Expected.ExpectedVolume,
			Current:            book.Expected.Current,
			Open:               book.Expected.Open,
			High:               book.Expected.High,
			Low:                book.Expected.Low,
			PreviousClose:      book.Expected.PreviousClose,
			PreviousChange:     book.Expected.PreviousChange,
			PreviousChangeSign: book.Expected.PreviousChangeSign,
			PreviousChangeRate: book.Expected.PreviousChangeRate,
		},
	}
}

func normalizeTrade(trade kisclient.Trade, symbol string, securityType provider.SecurityType) tradesrole.Trade {
	return tradesrole.Trade{
		Provider:           provider.ProviderKIS,
		Group:              provider.GroupKISDomesticStockQuotation,
		Operation:          provider.OperationKISTrades,
		Market:             provider.MarketKRX,
		SecurityType:       securityType,
		Symbol:             symbol,
		Time:               normalizeISOTime(trade.Time),
		Price:              trade.Current,
		Volume:             trade.Volume,
		PreviousChange:     trade.PreviousChange,
		PreviousChangeSign: trade.PreviousChangeSign,
		PreviousChangeRate: trade.PreviousChangeRate,
		Strength:           trade.Strength,
	}
}

func normalizeTimedTrade(trade kisclient.TimedTrade, symbol string, securityType provider.SecurityType) tradesrole.Trade {
	return tradesrole.Trade{
		Provider:           provider.ProviderKIS,
		Group:              provider.GroupKISDomesticStockQuotation,
		Operation:          provider.OperationKISTimeTrades,
		Market:             provider.MarketKRX,
		SecurityType:       securityType,
		Symbol:             symbol,
		Time:               normalizeISOTime(trade.Time),
		Price:              trade.Current,
		Volume:             trade.Volume,
		AccumulatedVolume:  trade.AccumulatedVolume,
		Ask:                trade.Ask,
		Bid:                trade.Bid,
		PreviousChange:     trade.PreviousChange,
		PreviousChangeSign: trade.PreviousChangeSign,
		PreviousChangeRate: trade.PreviousChangeRate,
		Strength:           trade.Strength,
	}
}

func compositionMemberFromKISComponentRow(row kisclient.ETFComponentStockPrice, observedAtMS int64) (composition.CompositionMember, composition.QuoteObservation) {
	instrument := composition.InstrumentRef{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       row.Symbol,
		Name:         row.Name,
	}
	member := composition.CompositionMember{
		Instrument: instrument,
		Weight:     decimalValue(row.Weight),
		Quantity:   decimalValue(row.Quantity),
		Valuation:  moneyValue(row.ValuationAmount),
	}
	quote := composition.QuoteObservation{
		Instrument:   instrument,
		ObservedAtMS: observedAtMS,
		Price:        moneyValue(row.Current),
		Change:       moneyValue(row.PreviousChange),
		ChangeRate:   decimalValue(row.PreviousChangeRate),
		Volume:       decimalValue(row.Volume),
	}
	return member, quote
}

func quoteObservationFromHeader(header map[string]string, subject composition.InstrumentRef, observedAtMS int64) composition.QuoteObservation {
	return composition.QuoteObservation{
		Instrument:   subject,
		ObservedAtMS: observedAtMS,
		Price:        moneyValue(firstNonEmpty(header["stck_prpr"], header["prpr"])),
		Change:       moneyValue(header["prdy_vrss"]),
		ChangeRate:   decimalValue(header["prdy_ctrt"]),
		Volume:       decimalValue(header["acml_vol"]),
	}
}

func kisComponentSource() composition.SourceRef {
	return composition.SourceRef{
		Provider:  provider.ProviderKIS,
		Group:     provider.GroupKISDomesticStockQuotation,
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
		Group:        provider.GroupKISDomesticStockInstrument,
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

type domesticStockQuotationGroup struct {
	quoteSnapshotter     quote.Snapshotter
	dailyFetcher         dailybar.Fetcher
	intradayFetcher      intradaybar.Fetcher
	orderbookSnapshotter orderbook.Snapshotter
	tradesLister         tradesrole.Lister
	compositionLister    composition.Lister
}

var _ provider.GroupRoleProvider = domesticStockQuotationGroup{}

func newDomesticStockQuotationGroup(snapshot quote.SnapshotFunc, fetch dailybar.FetchFunc, fetchIntraday intradaybar.FetchFunc, snapshotOrderbook orderbook.SnapshotFunc, listTrades tradesrole.ListFunc, listConstituents composition.ListFunc) domesticStockQuotationGroup {
	return domesticStockQuotationGroup{
		quoteSnapshotter: quote.NewSnapshot(
			spec.Quote().
				Markets(provider.MarketKRX).
				SecurityTypes(provider.SecurityTypeStock).
				Group(provider.GroupKISDomesticStockQuotation).
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
				Group(provider.GroupKISDomesticStockQuotation).
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
			Group(provider.GroupKISDomesticStockQuotation).
			Operations(provider.OperationKISIntraday).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS intraday bars are same-day symbol-scoped minute bars").
			Priority(90).
			Limitations("symbol-scoped same-day minute bars only; no historical multi-day intraday backfill").
			MustBuild(),
		orderbookSnapshotter: spec.RealtimeOrderbook(snapshotOrderbook).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISDomesticStockQuotation).
			Operations(provider.OperationKISOrderbook).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS orderbook snapshot returns up to 10 ask and 10 bid levels with expected conclusion data").
			Priority(90).
			Limitations("single symbol snapshot only; no streaming orderbook updates").
			MustBuild(),
		tradesLister: spec.RealtimeTrades(listTrades).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISDomesticStockQuotation).
			Operations(provider.OperationKISTrades, provider.OperationKISTimeTrades).
			RequiresAuth(provider.CredentialScopeKIS).
			CompatibilityNotes("KIS trade rows are market trade prints, not account order executions").
			Priority(90).
			Limitations("recent same-day market trades only; account executions are intentionally out of scope").
			MustBuild(),
		compositionLister: composition.NewList(composition.Profile{
			Markets:       []provider.Market{provider.MarketKRX},
			SecurityTypes: []provider.SecurityType{provider.SecurityTypeETF},
			Group:         provider.GroupKISDomesticStockQuotation,
			Operations:    []provider.OperationID{provider.OperationKISETFComponentStockPrice},
			AuthScope:     provider.CredentialScopeKIS,
			Freshness:     provider.FreshnessDaily,
			Compatibility: provider.Compatibility{
				DataLatency:         provider.DataLatencyRealtime,
				CurrentDaySupported: true,
				Notes:               []string{"KIS ETF component stock price API is adapted into canonical composition members plus quote observations"},
			},
			RequiresAuth: true,
			Priority:     90,
			Limitations: []string{
				"live read-through composition only; canonical composition storage is not implemented in this path",
				"ETF only; ETN and ELW component rows are not registered",
			},
		}, listConstituents),
	}
}

func (g domesticStockQuotationGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKISDomesticStockQuotation
}

func (g domesticStockQuotationGroup) RoleRegistrations() []provider.RoleRegistration {
	stockQuote := g.quoteSnapshotter.RoleRegistration()
	etpQuote := quote.NewSnapshot(
		spec.Quote().
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeETF, provider.SecurityTypeETN).
			Group(provider.GroupKISDomesticStockQuotation).
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

type domesticStockInstrumentGroup struct {
	instrument.Searcher
}

var _ provider.GroupRoleProvider = domesticStockInstrumentGroup{}

func newDomesticStockInstrumentGroup(search instrument.SearchFunc) domesticStockInstrumentGroup {
	return domesticStockInstrumentGroup{
		Searcher: spec.InstrumentSearcher(search).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock, provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW).
			Group(provider.GroupKISDomesticStockInstrument).
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

func (g domesticStockInstrumentGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKISDomesticStockInstrument
}

func (g domesticStockInstrumentGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Searcher.RoleRegistration()}
}
