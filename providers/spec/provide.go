package spec

import (
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/intradaybar"
	"github.com/ev3rlit/mwosa/providers/core/orderbook"
	"github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/samber/oops"
)

type DailyBarRoleBuilder struct {
	profile    DailyBarBuilder
	fetch      dailybar.FetchFunc
	pageFetch  dailybar.PageFetchFunc
	batchFetch dailybar.BatchFetchFunc
}

func DailyBarFetcher(fetch dailybar.FetchFunc) DailyBarRoleBuilder {
	return DailyBarRoleBuilder{
		profile: DailyBar(),
		fetch:   fetch,
	}
}

func PreviousBusinessDayDailyBar(fetch dailybar.FetchFunc) DailyBarRoleBuilder {
	return DailyBarFetcher(fetch).
		Freshness(provider.FreshnessDaily).
		Compatibility(PreviousBusinessDay())
}

func (b DailyBarRoleBuilder) Markets(markets ...provider.Market) DailyBarRoleBuilder {
	b.profile = b.profile.Markets(markets...)
	return b
}

func (b DailyBarRoleBuilder) SecurityTypes(securityTypes ...provider.SecurityType) DailyBarRoleBuilder {
	b.profile = b.profile.SecurityTypes(securityTypes...)
	return b
}

func (b DailyBarRoleBuilder) Group(group provider.GroupID) DailyBarRoleBuilder {
	b.profile = b.profile.Group(group)
	return b
}

func (b DailyBarRoleBuilder) Operations(operations ...provider.OperationID) DailyBarRoleBuilder {
	b.profile = b.profile.Operations(operations...)
	return b
}

func (b DailyBarRoleBuilder) RequiresAuth(scope provider.CredentialScope) DailyBarRoleBuilder {
	b.profile = b.profile.RequiresAuth(scope)
	return b
}

func (b DailyBarRoleBuilder) NoAuth() DailyBarRoleBuilder {
	b.profile = b.profile.NoAuth()
	return b
}

func (b DailyBarRoleBuilder) Freshness(freshness provider.Freshness) DailyBarRoleBuilder {
	b.profile = b.profile.Freshness(freshness)
	return b
}

func (b DailyBarRoleBuilder) Compatibility(source CompatibilitySource) DailyBarRoleBuilder {
	b.profile = b.profile.Compatibility(source)
	return b
}

func (b DailyBarRoleBuilder) CompatibilityValue(compatibility provider.Compatibility) DailyBarRoleBuilder {
	b.profile = b.profile.CompatibilityValue(compatibility)
	return b
}

func (b DailyBarRoleBuilder) CompatibilityNotes(notes ...string) DailyBarRoleBuilder {
	b.profile.role = b.profile.role.compatibilityNotes(notes...)
	return b
}

func (b DailyBarRoleBuilder) RangeQuery(rangeQuery dailybar.RangeQuerySupport) DailyBarRoleBuilder {
	b.profile = b.profile.RangeQuery(rangeQuery)
	return b
}

func (b DailyBarRoleBuilder) Priority(priority int) DailyBarRoleBuilder {
	b.profile = b.profile.Priority(priority)
	return b
}

func (b DailyBarRoleBuilder) Limitations(limitations ...string) DailyBarRoleBuilder {
	b.profile = b.profile.Limitations(limitations...)
	return b
}

func (b DailyBarRoleBuilder) PageFetch(pageFetch dailybar.PageFetchFunc) DailyBarRoleBuilder {
	b.pageFetch = pageFetch
	return b
}

func (b DailyBarRoleBuilder) BatchFetch(batchFetch dailybar.BatchFetchFunc) DailyBarRoleBuilder {
	b.batchFetch = batchFetch
	return b
}

func (b DailyBarRoleBuilder) Build() (dailybar.Fetcher, error) {
	if b.fetch == nil {
		return nil, oops.In("provider_spec").With("role", provider.RoleDailyBar).New("daily-bar provider spec requires fetch callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return nil, err
	}
	if b.batchFetch != nil {
		return dailybar.NewBatchFetch(profile, b.fetch, b.batchFetch), nil
	}
	if b.pageFetch != nil {
		return dailybar.NewPagedFetch(profile, b.fetch, b.pageFetch), nil
	}
	return dailybar.NewFetch(profile, b.fetch), nil
}

func (b DailyBarRoleBuilder) MustBuild() dailybar.Fetcher {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}

type InstrumentRoleBuilder struct {
	profile InstrumentBuilder
	search  instrument.SearchFunc
}

func InstrumentSearcher(search instrument.SearchFunc) InstrumentRoleBuilder {
	return InstrumentRoleBuilder{
		profile: Instrument(),
		search:  search,
	}
}

func PreviousBusinessDayInstrumentSearch(search instrument.SearchFunc) InstrumentRoleBuilder {
	return InstrumentSearcher(search).
		Freshness(provider.FreshnessDaily).
		Compatibility(PreviousBusinessDay())
}

func (b InstrumentRoleBuilder) Markets(markets ...provider.Market) InstrumentRoleBuilder {
	b.profile = b.profile.Markets(markets...)
	return b
}

func (b InstrumentRoleBuilder) SecurityTypes(securityTypes ...provider.SecurityType) InstrumentRoleBuilder {
	b.profile = b.profile.SecurityTypes(securityTypes...)
	return b
}

func (b InstrumentRoleBuilder) Group(group provider.GroupID) InstrumentRoleBuilder {
	b.profile = b.profile.Group(group)
	return b
}

func (b InstrumentRoleBuilder) Operations(operations ...provider.OperationID) InstrumentRoleBuilder {
	b.profile = b.profile.Operations(operations...)
	return b
}

func (b InstrumentRoleBuilder) RequiresAuth(scope provider.CredentialScope) InstrumentRoleBuilder {
	b.profile = b.profile.RequiresAuth(scope)
	return b
}

func (b InstrumentRoleBuilder) NoAuth() InstrumentRoleBuilder {
	b.profile = b.profile.NoAuth()
	return b
}

func (b InstrumentRoleBuilder) Freshness(freshness provider.Freshness) InstrumentRoleBuilder {
	b.profile = b.profile.Freshness(freshness)
	return b
}

func (b InstrumentRoleBuilder) Compatibility(source CompatibilitySource) InstrumentRoleBuilder {
	b.profile = b.profile.Compatibility(source)
	return b
}

func (b InstrumentRoleBuilder) CompatibilityValue(compatibility provider.Compatibility) InstrumentRoleBuilder {
	b.profile = b.profile.CompatibilityValue(compatibility)
	return b
}

func (b InstrumentRoleBuilder) CompatibilityNotes(notes ...string) InstrumentRoleBuilder {
	b.profile.role = b.profile.role.compatibilityNotes(notes...)
	return b
}

func (b InstrumentRoleBuilder) Priority(priority int) InstrumentRoleBuilder {
	b.profile = b.profile.Priority(priority)
	return b
}

func (b InstrumentRoleBuilder) Limitations(limitations ...string) InstrumentRoleBuilder {
	b.profile = b.profile.Limitations(limitations...)
	return b
}

func (b InstrumentRoleBuilder) Build() (instrument.Search, error) {
	if b.search == nil {
		return instrument.Search{}, oops.In("provider_spec").With("role", provider.RoleInstrument).New("instrument provider spec requires search callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return instrument.Search{}, err
	}
	return instrument.NewSearch(profile, b.search), nil
}

func (b InstrumentRoleBuilder) MustBuild() instrument.Search {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}

type IntradayBarRoleBuilder struct {
	profile IntradayBarBuilder
	fetch   intradaybar.FetchFunc
}

func IntradayBarFetcher(fetch intradaybar.FetchFunc) IntradayBarRoleBuilder {
	return IntradayBarRoleBuilder{
		profile: IntradayBar(),
		fetch:   fetch,
	}
}

func RealtimeIntradayBar(fetch intradaybar.FetchFunc) IntradayBarRoleBuilder {
	return IntradayBarFetcher(fetch).
		Freshness(provider.FreshnessIntraday).
		Compatibility(Realtime())
}

func (b IntradayBarRoleBuilder) Markets(markets ...provider.Market) IntradayBarRoleBuilder {
	b.profile = b.profile.Markets(markets...)
	return b
}

func (b IntradayBarRoleBuilder) SecurityTypes(securityTypes ...provider.SecurityType) IntradayBarRoleBuilder {
	b.profile = b.profile.SecurityTypes(securityTypes...)
	return b
}

func (b IntradayBarRoleBuilder) Group(group provider.GroupID) IntradayBarRoleBuilder {
	b.profile = b.profile.Group(group)
	return b
}

func (b IntradayBarRoleBuilder) Operations(operations ...provider.OperationID) IntradayBarRoleBuilder {
	b.profile = b.profile.Operations(operations...)
	return b
}

func (b IntradayBarRoleBuilder) RequiresAuth(scope provider.CredentialScope) IntradayBarRoleBuilder {
	b.profile = b.profile.RequiresAuth(scope)
	return b
}

func (b IntradayBarRoleBuilder) Freshness(freshness provider.Freshness) IntradayBarRoleBuilder {
	b.profile = b.profile.Freshness(freshness)
	return b
}

func (b IntradayBarRoleBuilder) Compatibility(source CompatibilitySource) IntradayBarRoleBuilder {
	b.profile = b.profile.Compatibility(source)
	return b
}

func (b IntradayBarRoleBuilder) CompatibilityNotes(notes ...string) IntradayBarRoleBuilder {
	b.profile.role = b.profile.role.compatibilityNotes(notes...)
	return b
}

func (b IntradayBarRoleBuilder) Priority(priority int) IntradayBarRoleBuilder {
	b.profile = b.profile.Priority(priority)
	return b
}

func (b IntradayBarRoleBuilder) Limitations(limitations ...string) IntradayBarRoleBuilder {
	b.profile = b.profile.Limitations(limitations...)
	return b
}

func (b IntradayBarRoleBuilder) Build() (intradaybar.Fetch, error) {
	if b.fetch == nil {
		return intradaybar.Fetch{}, oops.In("provider_spec").With("role", provider.RoleIntradayBar).New("intraday-bar provider spec requires fetch callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return intradaybar.Fetch{}, err
	}
	return intradaybar.NewFetch(profile, b.fetch), nil
}

func (b IntradayBarRoleBuilder) MustBuild() intradaybar.Fetch {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}

type OrderbookRoleBuilder struct {
	profile  OrderbookBuilder
	snapshot orderbook.SnapshotFunc
}

func OrderbookSnapshotter(snapshot orderbook.SnapshotFunc) OrderbookRoleBuilder {
	return OrderbookRoleBuilder{
		profile:  Orderbook(),
		snapshot: snapshot,
	}
}

func RealtimeOrderbook(snapshot orderbook.SnapshotFunc) OrderbookRoleBuilder {
	return OrderbookSnapshotter(snapshot).
		Freshness(provider.FreshnessIntraday).
		Compatibility(Realtime())
}

func (b OrderbookRoleBuilder) Markets(markets ...provider.Market) OrderbookRoleBuilder {
	b.profile = b.profile.Markets(markets...)
	return b
}

func (b OrderbookRoleBuilder) SecurityTypes(securityTypes ...provider.SecurityType) OrderbookRoleBuilder {
	b.profile = b.profile.SecurityTypes(securityTypes...)
	return b
}

func (b OrderbookRoleBuilder) Group(group provider.GroupID) OrderbookRoleBuilder {
	b.profile = b.profile.Group(group)
	return b
}

func (b OrderbookRoleBuilder) Operations(operations ...provider.OperationID) OrderbookRoleBuilder {
	b.profile = b.profile.Operations(operations...)
	return b
}

func (b OrderbookRoleBuilder) RequiresAuth(scope provider.CredentialScope) OrderbookRoleBuilder {
	b.profile = b.profile.RequiresAuth(scope)
	return b
}

func (b OrderbookRoleBuilder) Freshness(freshness provider.Freshness) OrderbookRoleBuilder {
	b.profile = b.profile.Freshness(freshness)
	return b
}

func (b OrderbookRoleBuilder) Compatibility(source CompatibilitySource) OrderbookRoleBuilder {
	b.profile = b.profile.Compatibility(source)
	return b
}

func (b OrderbookRoleBuilder) CompatibilityNotes(notes ...string) OrderbookRoleBuilder {
	b.profile.role = b.profile.role.compatibilityNotes(notes...)
	return b
}

func (b OrderbookRoleBuilder) Priority(priority int) OrderbookRoleBuilder {
	b.profile = b.profile.Priority(priority)
	return b
}

func (b OrderbookRoleBuilder) Limitations(limitations ...string) OrderbookRoleBuilder {
	b.profile = b.profile.Limitations(limitations...)
	return b
}

func (b OrderbookRoleBuilder) Build() (orderbook.SnapshotRole, error) {
	if b.snapshot == nil {
		return orderbook.SnapshotRole{}, oops.In("provider_spec").With("role", provider.RoleOrderbook).New("orderbook provider spec requires snapshot callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return orderbook.SnapshotRole{}, err
	}
	return orderbook.NewSnapshot(profile, b.snapshot), nil
}

func (b OrderbookRoleBuilder) MustBuild() orderbook.SnapshotRole {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}

type TradesRoleBuilder struct {
	profile TradesBuilder
	list    trades.ListFunc
}

func TradesLister(list trades.ListFunc) TradesRoleBuilder {
	return TradesRoleBuilder{
		profile: Trades(),
		list:    list,
	}
}

func RealtimeTrades(list trades.ListFunc) TradesRoleBuilder {
	return TradesLister(list).
		Freshness(provider.FreshnessIntraday).
		Compatibility(Realtime())
}

func (b TradesRoleBuilder) Markets(markets ...provider.Market) TradesRoleBuilder {
	b.profile = b.profile.Markets(markets...)
	return b
}

func (b TradesRoleBuilder) SecurityTypes(securityTypes ...provider.SecurityType) TradesRoleBuilder {
	b.profile = b.profile.SecurityTypes(securityTypes...)
	return b
}

func (b TradesRoleBuilder) Group(group provider.GroupID) TradesRoleBuilder {
	b.profile = b.profile.Group(group)
	return b
}

func (b TradesRoleBuilder) Operations(operations ...provider.OperationID) TradesRoleBuilder {
	b.profile = b.profile.Operations(operations...)
	return b
}

func (b TradesRoleBuilder) RequiresAuth(scope provider.CredentialScope) TradesRoleBuilder {
	b.profile = b.profile.RequiresAuth(scope)
	return b
}

func (b TradesRoleBuilder) Freshness(freshness provider.Freshness) TradesRoleBuilder {
	b.profile = b.profile.Freshness(freshness)
	return b
}

func (b TradesRoleBuilder) Compatibility(source CompatibilitySource) TradesRoleBuilder {
	b.profile = b.profile.Compatibility(source)
	return b
}

func (b TradesRoleBuilder) CompatibilityNotes(notes ...string) TradesRoleBuilder {
	b.profile.role = b.profile.role.compatibilityNotes(notes...)
	return b
}

func (b TradesRoleBuilder) Priority(priority int) TradesRoleBuilder {
	b.profile = b.profile.Priority(priority)
	return b
}

func (b TradesRoleBuilder) Limitations(limitations ...string) TradesRoleBuilder {
	b.profile = b.profile.Limitations(limitations...)
	return b
}

func (b TradesRoleBuilder) Build() (trades.List, error) {
	if b.list == nil {
		return trades.List{}, oops.In("provider_spec").With("role", provider.RoleTrades).New("trades provider spec requires list callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return trades.List{}, err
	}
	return trades.NewList(profile, b.list), nil
}

func (b TradesRoleBuilder) MustBuild() trades.List {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}
