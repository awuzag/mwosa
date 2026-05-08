package spec

import (
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/samber/oops"
)

type DailyBarRoleBuilder struct {
	profile   DailyBarBuilder
	fetch     dailybar.FetchFunc
	pageFetch dailybar.PageFetchFunc
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

func (b DailyBarRoleBuilder) Build() (dailybar.Fetch, error) {
	if b.fetch == nil {
		return dailybar.Fetch{}, oops.In("provider_spec").With("role", provider.RoleDailyBar).New("daily-bar provider spec requires fetch callable")
	}
	profile, err := b.profile.Build()
	if err != nil {
		return dailybar.Fetch{}, err
	}
	if b.pageFetch != nil {
		return dailybar.NewPagedFetch(profile, b.fetch, b.pageFetch), nil
	}
	return dailybar.NewFetch(profile, b.fetch), nil
}

func (b DailyBarRoleBuilder) MustBuild() dailybar.Fetch {
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
