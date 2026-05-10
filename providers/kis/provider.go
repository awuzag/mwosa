package kis

import (
	"context"
	"fmt"
	"strings"
	"sync"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/quote"
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
}

type marketDataClient interface {
	Token(context.Context) (kisclient.Token, error)
	Price(context.Context, string) (kisclient.Price, error)
	ETFETNPrice(context.Context, string) (kisclient.ETFETNPrice, error)
	Daily(context.Context, string, ...kisclient.DailyOption) ([]kisclient.Bar, error)
	Product(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Product, error)
	Stock(context.Context, string, ...kisclient.InstrumentOption) (kisclient.Stock, error)
}

type Provider struct {
	provider.Identity

	client marketDataClient

	tokenMu             sync.Mutex
	accessTokenProvided bool
	tokenIssued         bool

	groups []provider.GroupRoleProvider
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
	return NewWithClient(client, strings.TrimSpace(config.AccessToken) != ""), nil
}

func NewWithClient(client marketDataClient, accessTokenProvided bool) *Provider {
	p := &Provider{
		Identity: provider.Identity{
			ID:          provider.ProviderKIS,
			DisplayName: "한국투자증권 KIS",
		},
		client:              client,
		accessTokenProvided: accessTokenProvided,
	}
	p.groups = []provider.GroupRoleProvider{
		newDomesticStockQuotationGroup(p.fetchQuoteSnapshot, p.fetchDailyBars),
		newDomesticStockInstrumentGroup(p.searchInstruments),
	}
	return p
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
	if _, err := p.client.Token(ctx); err != nil {
		return oops.In("kis_adapter").With("provider", provider.ProviderKIS, "operation", provider.OperationKISPrice).Wrapf(err, "issue kis access token")
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
	quote.Snapshotter
	dailybar.Fetcher
}

var _ provider.GroupRoleProvider = domesticStockQuotationGroup{}

func newDomesticStockQuotationGroup(snapshot quote.SnapshotFunc, fetch dailybar.FetchFunc) domesticStockQuotationGroup {
	return domesticStockQuotationGroup{
		Snapshotter: quote.NewSnapshot(
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
		Fetcher: dailybar.NewFetch(
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
	}
}

func (g domesticStockQuotationGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKISDomesticStockQuotation
}

func (g domesticStockQuotationGroup) RoleRegistrations() []provider.RoleRegistration {
	stockQuote := g.Snapshotter.RoleRegistration()
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
		g.Snapshotter.FetchQuoteSnapshot,
	).RoleRegistration()
	return []provider.RoleRegistration{
		stockQuote,
		etpQuote,
		g.Fetcher.RoleRegistration(),
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
