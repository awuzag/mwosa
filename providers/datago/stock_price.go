package datago

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/instrument"
	"github.com/awuzag/mwosa/providers/spec"
)

type stockPriceGroup struct {
	dailybar.Fetcher
	instrument.Searcher
}

var _ provider.GroupRoleProvider = stockPriceGroup{}

func newStockPriceGroup(fetch dailybar.FetchFunc, pageFetch dailybar.PageFetchFunc, search instrument.SearchFunc) stockPriceGroup {
	return stockPriceGroup{
		Fetcher: spec.PreviousBusinessDayDailyBar(fetch).
			PageFetch(pageFetch).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock).
			Group(provider.GroupStockPrice).
			Operations(provider.OperationGetStockPriceInfo).
			RequiresAuth(provider.CredentialScopeDataGo).
			RangeQuery(dailybar.RangeQuerySupported).
			CompatibilityNotes(
				"latest available basDt is typically the previous business day",
				"current trading-day data is not supported",
			).
			Priority(50).
			Limitations(
				"daily basDt data only; not a realtime or current trading-day provider",
				"latest available data is typically D-1 business day EOD",
			).
			MustBuild(),
		Searcher: spec.PreviousBusinessDayInstrumentSearch(search).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock).
			Group(provider.GroupStockPrice).
			Operations(provider.OperationGetStockPriceInfo).
			RequiresAuth(provider.CredentialScopeDataGo).
			CompatibilityNotes(
				"instrument snapshots are derived from D-1 business day EOD stock price rows",
				"current trading-day data is not supported",
			).
			Priority(50).
			Limitations(
				"searches public D-1 business day EOD stock price rows and derives instrument snapshots",
				"not suitable for realtime or current trading-day instrument state",
			).
			MustBuild(),
	}
}

func (g stockPriceGroup) ProviderGroup() provider.GroupID {
	return provider.GroupStockPrice
}

func (g stockPriceGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{
		g.Fetcher.RoleRegistration(),
		g.Searcher.RoleRegistration(),
	}
}
