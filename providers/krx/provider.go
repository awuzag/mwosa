package krx

import (
	"context"
	"net/http"
	"strings"
	"time"

	krxclient "github.com/ev3rlit/mwosa/clients/krx"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/spec"
	"github.com/samber/oops"
)

type Config struct {
	AuthKey       string
	BaseURL       string
	SampleBaseURL string
	UseSample     bool
	HTTPClient    *http.Client
	EnabledAPIs   map[provider.OperationID]bool
	Now           func() time.Time
}

type client interface {
	KRXIndex(context.Context, string) ([]krxclient.KRXIndexDailyTrade, error)
	KOSPIIndex(context.Context, string) ([]krxclient.KOSPIIndexDailyTrade, error)
	KOSDAQIndex(context.Context, string) ([]krxclient.KOSDAQIndexDailyTrade, error)
	BondIndex(context.Context, string) ([]krxclient.BondIndexDailyTrade, error)
	DerivativesProductIndex(context.Context, string) ([]krxclient.DerivativesProductIndexDailyTrade, error)
	Stock(context.Context, string) ([]krxclient.StockDailyTrade, error)
	KOSDAQStock(context.Context, string) ([]krxclient.KOSDAQStockDailyTrade, error)
	KONEXStock(context.Context, string) ([]krxclient.KONEXStockDailyTrade, error)
	SubscriptionWarrant(context.Context, string) ([]krxclient.SubscriptionWarrantDailyTrade, error)
	SubscriptionRight(context.Context, string) ([]krxclient.SubscriptionRightDailyTrade, error)
	StockIssueBaseInfo(context.Context, string) ([]krxclient.StockIssueBaseInfo, error)
	KOSDAQIssueBaseInfo(context.Context, string) ([]krxclient.KOSDAQIssueBaseInfo, error)
	KONEXIssueBaseInfo(context.Context, string) ([]krxclient.KONEXIssueBaseInfo, error)
	ETF(context.Context, string) ([]krxclient.ETFDailyTrade, error)
	ETN(context.Context, string) ([]krxclient.ETNDailyTrade, error)
	ELW(context.Context, string) ([]krxclient.ELWDailyTrade, error)
	KTSBond(context.Context, string) ([]krxclient.KTSBondDailyTrade, error)
	GeneralBond(context.Context, string) ([]krxclient.GeneralBondDailyTrade, error)
	SmallBond(context.Context, string) ([]krxclient.SmallBondDailyTrade, error)
	Futures(context.Context, string) ([]krxclient.FuturesDailyTrade, error)
	KOSPIStockFutures(context.Context, string) ([]krxclient.KOSPIStockFuturesDailyTrade, error)
	KOSDAQStockFutures(context.Context, string) ([]krxclient.KOSDAQStockFuturesDailyTrade, error)
	Options(context.Context, string) ([]krxclient.OptionsDailyTrade, error)
	KOSPIStockOptions(context.Context, string) ([]krxclient.KOSPIStockOptionsDailyTrade, error)
	KOSDAQStockOptions(context.Context, string) ([]krxclient.KOSDAQStockOptionsDailyTrade, error)
	Oil(context.Context, string) ([]krxclient.OilDailyTrade, error)
	Gold(context.Context, string) ([]krxclient.GoldDailyTrade, error)
	EmissionTradingScheme(context.Context, string) ([]krxclient.EmissionTradingSchemeDailyTrade, error)
	ESGETPInfo(context.Context, string) ([]krxclient.ESGETPInfo, error)
	SRIBondInfo(context.Context, string) ([]krxclient.SRIBondInfo, error)
	ESGIndexInfo(context.Context, string) ([]krxclient.ESGIndexInfo, error)
}

type Provider struct {
	provider.Identity

	client  client
	enabled map[provider.OperationID]bool
	now     func() time.Time
	groups  []provider.GroupRoleProvider
}

func New(config Config) (*Provider, error) {
	errb := oops.In("krx_adapter").With("provider", provider.ProviderKRX)
	c, err := krxclient.New(clientOptions(config)...)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	return NewWithClient(c, config.EnabledAPIs, config.Now), nil
}

func NewWithClient(client client, enabledAPIs map[provider.OperationID]bool, now func() time.Time) *Provider {
	if now == nil {
		now = time.Now
	}
	p := &Provider{
		Identity: provider.Identity{
			ID:          provider.ProviderKRX,
			DisplayName: "KRX OPEN API",
		},
		client:  client,
		enabled: enabledAPIs,
		now:     now,
	}
	if p.enabled == nil {
		p.enabled = make(map[provider.OperationID]bool, len(ServiceCatalog()))
		for _, service := range ServiceCatalog() {
			p.enabled[service.Operation] = true
		}
	}
	p.groups = []provider.GroupRoleProvider{
		newETPDailyTradeGroup(p.fetchDailyBars, p.fetchDailyBatch, p.searchInstruments),
		newStockDailyTradeGroup(p.fetchDailyBars, p.fetchDailyBatch),
		newStockInstrumentGroup(p.searchInstruments),
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

func (p *Provider) FetchDailyBars(ctx context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
	return p.fetchDailyBars(ctx, input)
}

func (p *Provider) FetchDailyBatch(ctx context.Context, input dailybar.BatchFetchInput) (dailybar.BatchFetchResult, error) {
	return p.fetchDailyBatch(ctx, input)
}

func (p *Provider) SearchInstruments(ctx context.Context, input instrument.SearchInput) (instrument.SearchResult, error) {
	return p.searchInstruments(ctx, input)
}

func (p *Provider) enabledAPI(operation provider.OperationID) bool {
	if p == nil {
		return false
	}
	enabled, ok := p.enabled[operation]
	return !ok || enabled
}

func (p *Provider) requireAPI(operation provider.OperationID, role provider.Role, securityType provider.SecurityType, symbol string) error {
	if p.enabledAPI(operation) {
		return nil
	}
	return provider.NewUnsupported(provider.UnsupportedError{
		Capability:   role,
		ProviderID:   provider.ProviderKRX,
		GroupID:      groupForOperation(operation),
		OperationID:  operation,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		Symbol:       symbol,
		Reason:       "KRX OPEN API service is disabled or not approved in provider config",
	})
}

func (p *Provider) fetchDailyBars(ctx context.Context, input dailybar.FetchInput) (dailybar.FetchResult, error) {
	result, err := p.fetchDailyBatch(ctx, dailybar.BatchFetchInput{
		Market:       input.Market,
		SecurityType: input.SecurityType,
		Symbol:       input.Symbol,
		From:         input.From,
		To:           input.To,
	})
	if err != nil {
		return dailybar.FetchResult{}, err
	}
	return dailybar.FetchResult{
		Bars:       result.Bars,
		Provider:   result.Provider,
		Group:      result.Group,
		Operation:  result.Operation,
		TotalCount: result.TotalCount,
	}, nil
}

func (p *Provider) fetchDailyBatch(ctx context.Context, input dailybar.BatchFetchInput) (dailybar.BatchFetchResult, error) {
	market := withDefaultMarket(input.Market)
	errb := oops.In("krx_adapter").With("role", provider.RoleDailyBar, "market", market, "security_type", input.SecurityType, "symbol", input.Symbol, "from", input.From, "to", input.To)
	if err := validateMarket(provider.RoleDailyBar, market, input.SecurityType, input.Symbol); err != nil {
		return dailybar.BatchFetchResult{}, errb.Wrap(err)
	}
	if p.client == nil {
		return dailybar.BatchFetchResult{}, errb.New("krx provider client is nil")
	}
	from, to, err := resolveInputRange(input.From, input.To)
	if err != nil {
		return dailybar.BatchFetchResult{}, errb.Wrap(err)
	}
	records, group, err := p.fetchDailyRecords(ctx, input.SecurityType, input.Symbol, from, to)
	if err != nil {
		return dailybar.BatchFetchResult{}, errb.Wrap(err)
	}
	return dailybar.BatchFetchResult{
		Bars:       records,
		Provider:   p.Identity,
		Group:      group,
		TotalCount: len(records),
	}, nil
}

func (p *Provider) searchInstruments(ctx context.Context, input instrument.SearchInput) (instrument.SearchResult, error) {
	market := withDefaultMarket(input.Market)
	errb := oops.In("krx_adapter").With("role", provider.RoleInstrument, "market", market, "security_type", input.SecurityType, "query", input.Query)
	if err := validateMarket(provider.RoleInstrument, market, input.SecurityType, input.Query); err != nil {
		return instrument.SearchResult{}, errb.Wrap(err)
	}
	if p.client == nil {
		return instrument.SearchResult{}, errb.New("krx provider client is nil")
	}
	baseDate := previousBusinessDay(p.now()).Format("20060102")
	switch input.SecurityType {
	case provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW:
		return p.searchETPInstruments(ctx, input, baseDate)
	case provider.SecurityTypeStock, "":
		return p.searchStockInstruments(ctx, input, baseDate)
	default:
		return instrument.SearchResult{}, unsupportedSecurityTypeError(provider.RoleInstrument, input.SecurityType, input.Query)
	}
}

func (p *Provider) fetchDailyRecords(ctx context.Context, securityType provider.SecurityType, symbol string, from time.Time, to time.Time) ([]dailybar.Bar, provider.GroupID, error) {
	switch securityType {
	case provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW:
		bars, err := p.fetchETPDailyRecords(ctx, securityType, symbol, from, to)
		return bars, provider.GroupKRXETPDailyTrade, err
	case provider.SecurityTypeStock:
		bars, err := p.fetchStockDailyRecords(ctx, symbol, from, to)
		return bars, provider.GroupKRXStockDailyTrade, err
	case "":
		return nil, "", provider.NewUnsupported(provider.UnsupportedError{
			Capability: provider.RoleDailyBar,
			ProviderID: provider.ProviderKRX,
			GroupID:    provider.GroupKRXETPDailyTrade,
			Market:     provider.MarketKRX,
			Symbol:     symbol,
			Reason:     "security_type is required for KRX daily bars",
		})
	default:
		return nil, "", unsupportedSecurityTypeError(provider.RoleDailyBar, securityType, symbol)
	}
}

func (p *Provider) fetchETPDailyRecords(ctx context.Context, securityType provider.SecurityType, symbol string, from time.Time, to time.Time) ([]dailybar.Bar, error) {
	operation, err := etpOperation(securityType)
	if err != nil {
		return nil, err
	}
	if err := p.requireAPI(operation, provider.RoleDailyBar, securityType, symbol); err != nil {
		return nil, err
	}
	bars := make([]dailybar.Bar, 0)
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		baseDate := date.Format("20060102")
		switch securityType {
		case provider.SecurityTypeETF:
			rows, err := p.client.ETF(ctx, baseDate)
			if err != nil {
				return nil, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ETF daily trade")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, symbol) {
					bars = append(bars, normalizeETF(row))
				}
			}
		case provider.SecurityTypeETN:
			rows, err := p.client.ETN(ctx, baseDate)
			if err != nil {
				return nil, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ETN daily trade")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, symbol) {
					bars = append(bars, normalizeETN(row))
				}
			}
		case provider.SecurityTypeELW:
			rows, err := p.client.ELW(ctx, baseDate)
			if err != nil {
				return nil, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ELW daily trade")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, symbol) {
					bars = append(bars, normalizeELW(row))
				}
			}
		}
	}
	return bars, nil
}

func (p *Provider) fetchStockDailyRecords(ctx context.Context, symbol string, from time.Time, to time.Time) ([]dailybar.Bar, error) {
	operations := []provider.OperationID{provider.OperationStockByddTrd, provider.OperationKOSDAQByddTrd, provider.OperationKONEXByddTrd}
	for _, operation := range operations {
		if err := p.requireAPI(operation, provider.RoleDailyBar, provider.SecurityTypeStock, symbol); err != nil {
			return nil, err
		}
	}
	bars := make([]dailybar.Bar, 0)
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		baseDate := date.Format("20060102")
		rows, err := p.client.Stock(ctx, baseDate)
		if err != nil {
			return nil, oops.In("krx_adapter").With("operation", provider.OperationStockByddTrd, "base_date", baseDate).Wrapf(err, "fetch KRX stock daily trade")
		}
		for _, row := range rows {
			if matchesQuery(row.IssueCode, row.IssueName, symbol) {
				bars = append(bars, normalizeStock(row, provider.OperationStockByddTrd))
			}
		}
		kosdaqRows, err := p.client.KOSDAQStock(ctx, baseDate)
		if err != nil {
			return nil, oops.In("krx_adapter").With("operation", provider.OperationKOSDAQByddTrd, "base_date", baseDate).Wrapf(err, "fetch KRX KOSDAQ daily trade")
		}
		for _, row := range kosdaqRows {
			if matchesQuery(row.IssueCode, row.IssueName, symbol) {
				bars = append(bars, normalizeKOSDAQStock(row))
			}
		}
		konexRows, err := p.client.KONEXStock(ctx, baseDate)
		if err != nil {
			return nil, oops.In("krx_adapter").With("operation", provider.OperationKONEXByddTrd, "base_date", baseDate).Wrapf(err, "fetch KRX KONEX daily trade")
		}
		for _, row := range konexRows {
			if matchesQuery(row.IssueCode, row.IssueName, symbol) {
				bars = append(bars, normalizeKONEXStock(row))
			}
		}
	}
	return bars, nil
}

func (p *Provider) searchETPInstruments(ctx context.Context, input instrument.SearchInput, baseDate string) (instrument.SearchResult, error) {
	operations := []provider.OperationID{provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd}
	instruments := make([]instrument.Instrument, 0)
	for _, operation := range operationsForETPSearch(input.SecurityType) {
		if err := p.requireAPI(operation, provider.RoleInstrument, input.SecurityType, input.Query); err != nil {
			return instrument.SearchResult{}, err
		}
		switch operation {
		case provider.OperationETFByddTrd:
			rows, err := p.client.ETF(ctx, baseDate)
			if err != nil {
				return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ETF instruments")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, input.Query) {
					instruments = append(instruments, instrumentFromDailyBar(normalizeETF(row)))
				}
			}
		case provider.OperationETNByddTrd:
			rows, err := p.client.ETN(ctx, baseDate)
			if err != nil {
				return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ETN instruments")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, input.Query) {
					instruments = append(instruments, instrumentFromDailyBar(normalizeETN(row)))
				}
			}
		case provider.OperationELWByddTrd:
			rows, err := p.client.ELW(ctx, baseDate)
			if err != nil {
				return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", operation, "base_date", baseDate).Wrapf(err, "fetch KRX ELW instruments")
			}
			for _, row := range rows {
				if matchesQuery(row.IssueCode, row.IssueName, input.Query) {
					instruments = append(instruments, instrumentFromDailyBar(normalizeELW(row)))
				}
			}
		}
		instruments = limitInstruments(instruments, input.Limit)
		if input.Limit > 0 && len(instruments) >= input.Limit {
			break
		}
	}
	return instrument.SearchResult{
		Instruments: instruments,
		Provider:    p.Identity,
		Group:       provider.GroupKRXETPDailyTrade,
		Operations:  operations,
		TotalCount:  len(instruments),
	}, nil
}

func (p *Provider) searchStockInstruments(ctx context.Context, input instrument.SearchInput, baseDate string) (instrument.SearchResult, error) {
	operations := []provider.OperationID{provider.OperationStockIssueBaseInfo, provider.OperationKOSDAQIssueBaseInfo, provider.OperationKONEXIssueBaseInfo}
	for _, operation := range operations {
		if err := p.requireAPI(operation, provider.RoleInstrument, provider.SecurityTypeStock, input.Query); err != nil {
			return instrument.SearchResult{}, err
		}
	}
	instruments := make([]instrument.Instrument, 0)
	rows, err := p.client.StockIssueBaseInfo(ctx, baseDate)
	if err != nil {
		return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", provider.OperationStockIssueBaseInfo, "base_date", baseDate).Wrapf(err, "fetch KRX stock issue base info")
	}
	for _, row := range rows {
		if matchesStockIssue(row.IssueCode, row.IssueShortCode, row.IssueName, row.IssueAbbreviation, input.Query) {
			instruments = append(instruments, normalizeStockIssue(row, provider.OperationStockIssueBaseInfo))
		}
	}
	kosdaqRows, err := p.client.KOSDAQIssueBaseInfo(ctx, baseDate)
	if err != nil {
		return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", provider.OperationKOSDAQIssueBaseInfo, "base_date", baseDate).Wrapf(err, "fetch KRX KOSDAQ issue base info")
	}
	for _, row := range kosdaqRows {
		if matchesStockIssue(row.IssueCode, row.IssueShortCode, row.IssueName, row.IssueAbbreviation, input.Query) {
			instruments = append(instruments, normalizeKOSDAQIssue(row))
		}
	}
	konexRows, err := p.client.KONEXIssueBaseInfo(ctx, baseDate)
	if err != nil {
		return instrument.SearchResult{}, oops.In("krx_adapter").With("operation", provider.OperationKONEXIssueBaseInfo, "base_date", baseDate).Wrapf(err, "fetch KRX KONEX issue base info")
	}
	for _, row := range konexRows {
		if matchesStockIssue(row.IssueCode, row.IssueShortCode, row.IssueName, row.IssueAbbreviation, input.Query) {
			instruments = append(instruments, normalizeKONEXIssue(row))
		}
	}
	instruments = limitInstruments(instruments, input.Limit)
	return instrument.SearchResult{
		Instruments: instruments,
		Provider:    p.Identity,
		Group:       provider.GroupKRXStockInstrument,
		Operations:  operations,
		TotalCount:  len(instruments),
	}, nil
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func validateMarket(role provider.Role, market provider.Market, securityType provider.SecurityType, symbol string) error {
	if market == provider.MarketKRX {
		return nil
	}
	return provider.NewUnsupported(provider.UnsupportedError{
		Capability:   role,
		ProviderID:   provider.ProviderKRX,
		Market:       market,
		SecurityType: securityType,
		Symbol:       symbol,
		Reason:       "market is not supported by KRX",
	})
}

func unsupportedSecurityTypeError(role provider.Role, securityType provider.SecurityType, symbol string) error {
	return provider.NewUnsupported(provider.UnsupportedError{
		Capability:   role,
		ProviderID:   provider.ProviderKRX,
		GroupID:      provider.GroupKRXETPDailyTrade,
		Market:       provider.MarketKRX,
		SecurityType: securityType,
		Symbol:       symbol,
		Reason:       "security_type is not supported by KRX canonical adapter",
	})
}

func etpOperation(securityType provider.SecurityType) (provider.OperationID, error) {
	switch securityType {
	case provider.SecurityTypeETF:
		return provider.OperationETFByddTrd, nil
	case provider.SecurityTypeETN:
		return provider.OperationETNByddTrd, nil
	case provider.SecurityTypeELW:
		return provider.OperationELWByddTrd, nil
	default:
		return "", unsupportedSecurityTypeError(provider.RoleDailyBar, securityType, "")
	}
}

func operationsForETPSearch(securityType provider.SecurityType) []provider.OperationID {
	switch securityType {
	case provider.SecurityTypeETF:
		return []provider.OperationID{provider.OperationETFByddTrd}
	case provider.SecurityTypeETN:
		return []provider.OperationID{provider.OperationETNByddTrd}
	case provider.SecurityTypeELW:
		return []provider.OperationID{provider.OperationELWByddTrd}
	default:
		return []provider.OperationID{provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd}
	}
}

func groupForOperation(operation provider.OperationID) provider.GroupID {
	for _, service := range ServiceCatalog() {
		if service.Operation == operation {
			return service.Group
		}
	}
	return ""
}

func matchesQuery(symbol string, name string, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(symbol), query) ||
		strings.EqualFold(strings.TrimSpace(name), query) ||
		strings.Contains(strings.ToLower(name), strings.ToLower(query))
}

func matchesStockIssue(isin string, shortCode string, name string, abbreviation string, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	lowerQuery := strings.ToLower(query)
	return strings.EqualFold(strings.TrimSpace(isin), query) ||
		strings.EqualFold(strings.TrimSpace(shortCode), query) ||
		strings.EqualFold(strings.TrimSpace(name), query) ||
		strings.EqualFold(strings.TrimSpace(abbreviation), query) ||
		strings.Contains(strings.ToLower(name), lowerQuery) ||
		strings.Contains(strings.ToLower(abbreviation), lowerQuery)
}

func limitInstruments(instruments []instrument.Instrument, limit int) []instrument.Instrument {
	if limit <= 0 || len(instruments) <= limit {
		return instruments
	}
	return instruments[:limit]
}

func previousBusinessDay(now time.Time) time.Time {
	date := now.In(time.Local)
	for {
		date = date.AddDate(0, 0, -1)
		switch date.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		default:
			return date
		}
	}
}

type etpDailyTradeGroup struct {
	dailybar.Fetcher
	instrument.Searcher
}

var _ provider.GroupRoleProvider = etpDailyTradeGroup{}

func newETPDailyTradeGroup(fetch dailybar.FetchFunc, batchFetch dailybar.BatchFetchFunc, search instrument.SearchFunc) etpDailyTradeGroup {
	return etpDailyTradeGroup{
		Fetcher: spec.PreviousBusinessDayDailyBar(fetch).
			BatchFetch(batchFetch).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW).
			Group(provider.GroupKRXETPDailyTrade).
			Operations(provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd).
			RequiresAuth(provider.CredentialScopeKRX).
			RangeQuery(dailybar.RangeQuerySupported).
			Priority(45).
			Limitations("KRX endpoint is base-date only; ranges are collected one trading date at a time").
			MustBuild(),
		Searcher: spec.PreviousBusinessDayInstrumentSearch(search).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeETF, provider.SecurityTypeETN, provider.SecurityTypeELW).
			Group(provider.GroupKRXETPDailyTrade).
			Operations(provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd).
			RequiresAuth(provider.CredentialScopeKRX).
			Priority(45).
			Limitations("instrument snapshots are derived from recent ETP daily trade rows").
			MustBuild(),
	}
}

func (g etpDailyTradeGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKRXETPDailyTrade
}

func (g etpDailyTradeGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Fetcher.RoleRegistration(), g.Searcher.RoleRegistration()}
}

type stockDailyTradeGroup struct {
	dailybar.Fetcher
}

var _ provider.GroupRoleProvider = stockDailyTradeGroup{}

func newStockDailyTradeGroup(fetch dailybar.FetchFunc, batchFetch dailybar.BatchFetchFunc) stockDailyTradeGroup {
	return stockDailyTradeGroup{
		Fetcher: spec.PreviousBusinessDayDailyBar(fetch).
			BatchFetch(batchFetch).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock).
			Group(provider.GroupKRXStockDailyTrade).
			Operations(provider.OperationStockByddTrd, provider.OperationKOSDAQByddTrd, provider.OperationKONEXByddTrd).
			RequiresAuth(provider.CredentialScopeKRX).
			RangeQuery(dailybar.RangeQuerySupported).
			Priority(45).
			Limitations("KRX stock daily trade combines KOSPI, KOSDAQ, and KONEX daily services").
			MustBuild(),
	}
}

func (g stockDailyTradeGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKRXStockDailyTrade
}

func (g stockDailyTradeGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Fetcher.RoleRegistration()}
}

type stockInstrumentGroup struct {
	instrument.Searcher
}

var _ provider.GroupRoleProvider = stockInstrumentGroup{}

func newStockInstrumentGroup(search instrument.SearchFunc) stockInstrumentGroup {
	return stockInstrumentGroup{
		Searcher: spec.PreviousBusinessDayInstrumentSearch(search).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock).
			Group(provider.GroupKRXStockInstrument).
			Operations(provider.OperationStockIssueBaseInfo, provider.OperationKOSDAQIssueBaseInfo, provider.OperationKONEXIssueBaseInfo).
			RequiresAuth(provider.CredentialScopeKRX).
			Priority(60).
			Limitations("base-date is resolved to the previous business day for generic instrument search").
			MustBuild(),
	}
}

func (g stockInstrumentGroup) ProviderGroup() provider.GroupID {
	return provider.GroupKRXStockInstrument
}

func (g stockInstrumentGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Searcher.RoleRegistration()}
}
