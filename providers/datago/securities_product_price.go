package datago

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/instrument"
	"github.com/awuzag/mwosa/providers/spec"
)

type securitiesProductPriceGroup struct {
	dailybar.Fetcher
	instrument.Searcher
}

var _ provider.GroupRoleProvider = securitiesProductPriceGroup{}

func newSecuritiesProductPriceGroup(fetch dailybar.FetchFunc, pageFetch dailybar.PageFetchFunc, search instrument.SearchFunc) securitiesProductPriceGroup {
	return securitiesProductPriceGroup{
		Fetcher: spec.PreviousBusinessDayDailyBar(fetch).
			PageFetch(pageFetch).
			Markets(provider.MarketKRX).
			SecurityTypes(
				provider.SecurityTypeETF,
				provider.SecurityTypeETN,
				provider.SecurityTypeELW,
			).
			Group(provider.GroupSecuritiesProductPrice).
			Operations(
				provider.OperationGetETFPriceInfo,
				provider.OperationGetETNPriceInfo,
				provider.OperationGetELWPriceInfo,
			).
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
				"ELW uses explicit security_type=elw because canonical schema policy is separate from ETF/ETN",
			).
			MustBuild(),
		Searcher: spec.PreviousBusinessDayInstrumentSearch(search).
			Markets(provider.MarketKRX).
			SecurityTypes(
				provider.SecurityTypeETF,
				provider.SecurityTypeETN,
				provider.SecurityTypeELW,
			).
			Group(provider.GroupSecuritiesProductPrice).
			Operations(
				provider.OperationGetETFPriceInfo,
				provider.OperationGetETNPriceInfo,
				provider.OperationGetELWPriceInfo,
			).
			RequiresAuth(provider.CredentialScopeDataGo).
			CompatibilityNotes(
				"instrument snapshots are derived from D-1 business day EOD price rows",
				"current trading-day data is not supported",
			).
			Priority(50).
			Limitations(
				"searches public D-1 business day EOD price rows and derives instrument snapshots",
				"not suitable for realtime or current trading-day instrument state",
				"ELW search requires explicit security_type=elw",
			).
			MustBuild(),
	}
}

func (g securitiesProductPriceGroup) ProviderGroup() provider.GroupID {
	return provider.GroupSecuritiesProductPrice
}

func (g securitiesProductPriceGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{
		g.Fetcher.RoleRegistration(),
		g.Searcher.RoleRegistration(),
	}
}
