package datago

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/providers/spec"
)

type corporateFinanceGroup struct {
	financials.Fetch
}

var _ provider.GroupRoleProvider = corporateFinanceGroup{}

func newCorporateFinanceGroup(fetch financials.FetchFunc) corporateFinanceGroup {
	return corporateFinanceGroup{
		Fetch: spec.HistoricalFinancials(fetch).
			Markets(provider.MarketKRX).
			SecurityTypes(provider.SecurityTypeStock).
			Group(provider.GroupCorporateFinance).
			Operations(
				provider.OperationGetSummFinaStatV2,
				provider.OperationGetBalanceSheetV2,
				provider.OperationGetIncomeStatementV2,
			).
			RequiresAuth(provider.CredentialScopeDataGo).
			CompatibilityNotes(
				"company financial statements are looked up by corporation registration number (crno)",
				"KRX short codes and ISINs are resolved through krxListedInfo/getItemInfo before financial statement lookup",
				"financial commission APIs are not realtime; data is refreshed after provider-side collection",
			).
			Priority(40).
			Limitations(
				"foreign companies may not provide a domestic corporation registration number",
				"requires both Data.go.kr corporateFinance and krxListedInfo API approvals for name/code resolution",
				"cash flow statements are not provided by this Datago API",
			).
			MustBuild(),
	}
}

func (g corporateFinanceGroup) ProviderGroup() provider.GroupID {
	return provider.GroupCorporateFinance
}

func (g corporateFinanceGroup) RoleRegistrations() []provider.RoleRegistration {
	return []provider.RoleRegistration{g.Fetch.RoleRegistration()}
}
