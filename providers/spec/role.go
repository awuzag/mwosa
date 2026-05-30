package spec

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/awuzag/mwosa/providers/core/instrument"
	"github.com/awuzag/mwosa/providers/core/intradaybar"
	"github.com/awuzag/mwosa/providers/core/orderbook"
	"github.com/awuzag/mwosa/providers/core/quote"
	"github.com/awuzag/mwosa/providers/core/trades"
	"github.com/samber/oops"
)

type roleBuilder struct {
	profile provider.RoleProfile
	err     error
}

func newRoleBuilder(role provider.Role) roleBuilder {
	return roleBuilder{
		profile: provider.RoleProfile{
			Role: role,
		},
	}
}

func (b roleBuilder) markets(markets ...provider.Market) roleBuilder {
	b.profile.Markets = append([]provider.Market(nil), markets...)
	return b
}

func (b roleBuilder) securityTypes(securityTypes ...provider.SecurityType) roleBuilder {
	b.profile.SecurityTypes = append([]provider.SecurityType(nil), securityTypes...)
	return b
}

func (b roleBuilder) group(group provider.GroupID) roleBuilder {
	b.profile.Group = group
	return b
}

func (b roleBuilder) operations(operations ...provider.OperationID) roleBuilder {
	b.profile.Operations = append([]provider.OperationID(nil), operations...)
	return b
}

func (b roleBuilder) requiresAuth(scope provider.CredentialScope) roleBuilder {
	b.profile.RequiresAuth = true
	b.profile.AuthScope = scope
	return b
}

func (b roleBuilder) noAuth() roleBuilder {
	b.profile.RequiresAuth = false
	b.profile.AuthScope = ""
	return b
}

func (b roleBuilder) freshness(freshness provider.Freshness) roleBuilder {
	b.profile.Freshness = freshness
	return b
}

func (b roleBuilder) compatibility(source CompatibilitySource) roleBuilder {
	if source == nil {
		return b.withError(oops.In("provider_spec").With("role", b.profile.Role).New("provider role compatibility source is nil"))
	}
	compatibility, err := source.BuildCompatibility()
	if err != nil {
		return b.withError(err)
	}
	b.profile.Compatibility = compatibility
	return b
}

func (b roleBuilder) compatibilityValue(compatibility provider.Compatibility) roleBuilder {
	if err := ValidateCompatibility(compatibility); err != nil {
		return b.withError(err)
	}
	b.profile.Compatibility = compatibility
	return b
}

func (b roleBuilder) compatibilityNotes(notes ...string) roleBuilder {
	if b.profile.Compatibility.DataLatency == "" {
		return b.withError(oops.In("provider_spec").With("role", b.profile.Role).New("provider role compatibility must be selected before adding notes"))
	}
	b.profile.Compatibility.Notes = append(b.profile.Compatibility.Notes, notes...)
	return b
}

func (b roleBuilder) priority(priority int) roleBuilder {
	b.profile.Priority = priority
	return b
}

func (b roleBuilder) limitations(limitations ...string) roleBuilder {
	b.profile.Limitations = append([]string(nil), limitations...)
	return b
}

func (b roleBuilder) build() (provider.RoleProfile, error) {
	if b.err != nil {
		return provider.RoleProfile{}, b.err
	}

	profile := b.profile
	errb := oops.In("provider_spec").With("role", profile.Role)
	if profile.Role == "" {
		return provider.RoleProfile{}, errb.New("provider role spec requires role")
	}
	if len(profile.Markets) == 0 {
		return provider.RoleProfile{}, errb.New("provider role spec requires at least one market")
	}
	for _, market := range profile.Markets {
		if market == "" {
			return provider.RoleProfile{}, errb.New("provider role spec contains empty market")
		}
	}
	if len(profile.SecurityTypes) == 0 && profile.Role != provider.RoleIndexBar {
		return provider.RoleProfile{}, errb.New("provider role spec requires at least one security type")
	}
	for _, securityType := range profile.SecurityTypes {
		if securityType == "" {
			return provider.RoleProfile{}, errb.New("provider role spec contains empty security type")
		}
	}
	if profile.Group == "" {
		return provider.RoleProfile{}, errb.New("provider role spec requires provider group")
	}
	if len(profile.Operations) == 0 {
		return provider.RoleProfile{}, errb.New("provider role spec requires at least one operation")
	}
	for _, operation := range profile.Operations {
		if operation == "" {
			return provider.RoleProfile{}, errb.New("provider role spec contains empty operation")
		}
	}
	if profile.Freshness == "" {
		return provider.RoleProfile{}, errb.New("provider role spec requires freshness")
	}
	if err := ValidateCompatibility(profile.Compatibility); err != nil {
		return provider.RoleProfile{}, errb.Wrap(err)
	}
	if profile.RequiresAuth && profile.AuthScope == "" {
		return provider.RoleProfile{}, errb.New("provider role spec requires auth scope when auth is required")
	}

	return profile, nil
}

func (b roleBuilder) withError(err error) roleBuilder {
	if b.err != nil {
		return b
	}
	b.err = err
	return b
}

type DailyBarBuilder struct {
	role       roleBuilder
	rangeQuery dailybar.RangeQuerySupport
}

func DailyBar() DailyBarBuilder {
	return DailyBarBuilder{role: newRoleBuilder(provider.RoleDailyBar)}
}

func (b DailyBarBuilder) Markets(markets ...provider.Market) DailyBarBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b DailyBarBuilder) SecurityTypes(securityTypes ...provider.SecurityType) DailyBarBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b DailyBarBuilder) Group(group provider.GroupID) DailyBarBuilder {
	b.role = b.role.group(group)
	return b
}

func (b DailyBarBuilder) Operations(operations ...provider.OperationID) DailyBarBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b DailyBarBuilder) RequiresAuth(scope provider.CredentialScope) DailyBarBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b DailyBarBuilder) NoAuth() DailyBarBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b DailyBarBuilder) Freshness(freshness provider.Freshness) DailyBarBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b DailyBarBuilder) Compatibility(source CompatibilitySource) DailyBarBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b DailyBarBuilder) CompatibilityValue(compatibility provider.Compatibility) DailyBarBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b DailyBarBuilder) RangeQuery(rangeQuery dailybar.RangeQuerySupport) DailyBarBuilder {
	b.rangeQuery = rangeQuery
	return b
}

func (b DailyBarBuilder) Priority(priority int) DailyBarBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b DailyBarBuilder) Limitations(limitations ...string) DailyBarBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b DailyBarBuilder) Build() (dailybar.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return dailybar.Profile{}, err
	}
	if b.rangeQuery == "" {
		return dailybar.Profile{}, oops.In("provider_spec").With("role", profile.Role).New("daily-bar provider spec requires range query support")
	}
	if b.rangeQuery != dailybar.RangeQuerySupported && b.rangeQuery != dailybar.RangeQueryUnsupported {
		return dailybar.Profile{}, oops.In("provider_spec").With("role", profile.Role, "range_query", b.rangeQuery).New("daily-bar provider spec has unknown range query support")
	}
	return dailybar.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		RangeQuery:    b.rangeQuery,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b DailyBarBuilder) MustBuild() dailybar.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type IndexBarBuilder struct {
	role       roleBuilder
	rangeQuery indexbar.RangeQuerySupport
}

func IndexBar() IndexBarBuilder {
	return IndexBarBuilder{role: newRoleBuilder(provider.RoleIndexBar)}
}

func (b IndexBarBuilder) Markets(markets ...provider.Market) IndexBarBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b IndexBarBuilder) Group(group provider.GroupID) IndexBarBuilder {
	b.role = b.role.group(group)
	return b
}

func (b IndexBarBuilder) Operations(operations ...provider.OperationID) IndexBarBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b IndexBarBuilder) RequiresAuth(scope provider.CredentialScope) IndexBarBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b IndexBarBuilder) NoAuth() IndexBarBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b IndexBarBuilder) Freshness(freshness provider.Freshness) IndexBarBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b IndexBarBuilder) Compatibility(source CompatibilitySource) IndexBarBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b IndexBarBuilder) CompatibilityValue(compatibility provider.Compatibility) IndexBarBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b IndexBarBuilder) RangeQuery(rangeQuery indexbar.RangeQuerySupport) IndexBarBuilder {
	b.rangeQuery = rangeQuery
	return b
}

func (b IndexBarBuilder) Priority(priority int) IndexBarBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b IndexBarBuilder) Limitations(limitations ...string) IndexBarBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b IndexBarBuilder) Build() (indexbar.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return indexbar.Profile{}, err
	}
	if b.rangeQuery == "" {
		return indexbar.Profile{}, oops.In("provider_spec").With("role", profile.Role).New("index-bar provider spec requires range query support")
	}
	if b.rangeQuery != indexbar.RangeQuerySupported && b.rangeQuery != indexbar.RangeQueryUnsupported {
		return indexbar.Profile{}, oops.In("provider_spec").With("role", profile.Role, "range_query", b.rangeQuery).New("index-bar provider spec has unknown range query support")
	}
	return indexbar.Profile{
		Markets:       profile.Markets,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		RangeQuery:    b.rangeQuery,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b IndexBarBuilder) MustBuild() indexbar.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type InstrumentBuilder struct {
	role roleBuilder
}

func Instrument() InstrumentBuilder {
	return InstrumentBuilder{role: newRoleBuilder(provider.RoleInstrument)}
}

func (b InstrumentBuilder) Markets(markets ...provider.Market) InstrumentBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b InstrumentBuilder) SecurityTypes(securityTypes ...provider.SecurityType) InstrumentBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b InstrumentBuilder) Group(group provider.GroupID) InstrumentBuilder {
	b.role = b.role.group(group)
	return b
}

func (b InstrumentBuilder) Operations(operations ...provider.OperationID) InstrumentBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b InstrumentBuilder) RequiresAuth(scope provider.CredentialScope) InstrumentBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b InstrumentBuilder) NoAuth() InstrumentBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b InstrumentBuilder) Freshness(freshness provider.Freshness) InstrumentBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b InstrumentBuilder) Compatibility(source CompatibilitySource) InstrumentBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b InstrumentBuilder) CompatibilityValue(compatibility provider.Compatibility) InstrumentBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b InstrumentBuilder) Priority(priority int) InstrumentBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b InstrumentBuilder) Limitations(limitations ...string) InstrumentBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b InstrumentBuilder) Build() (instrument.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return instrument.Profile{}, err
	}
	return instrument.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b InstrumentBuilder) MustBuild() instrument.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type QuoteBuilder struct {
	role roleBuilder
}

func Quote() QuoteBuilder {
	return QuoteBuilder{role: newRoleBuilder(provider.RoleQuote)}
}

func (b QuoteBuilder) Markets(markets ...provider.Market) QuoteBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b QuoteBuilder) SecurityTypes(securityTypes ...provider.SecurityType) QuoteBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b QuoteBuilder) Group(group provider.GroupID) QuoteBuilder {
	b.role = b.role.group(group)
	return b
}

func (b QuoteBuilder) Operations(operations ...provider.OperationID) QuoteBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b QuoteBuilder) RequiresAuth(scope provider.CredentialScope) QuoteBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b QuoteBuilder) NoAuth() QuoteBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b QuoteBuilder) Freshness(freshness provider.Freshness) QuoteBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b QuoteBuilder) Compatibility(source CompatibilitySource) QuoteBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b QuoteBuilder) CompatibilityValue(compatibility provider.Compatibility) QuoteBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b QuoteBuilder) Priority(priority int) QuoteBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b QuoteBuilder) Limitations(limitations ...string) QuoteBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b QuoteBuilder) Build() (quote.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return quote.Profile{}, err
	}
	return quote.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b QuoteBuilder) MustBuild() quote.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type IntradayBarBuilder struct {
	role roleBuilder
}

func IntradayBar() IntradayBarBuilder {
	return IntradayBarBuilder{role: newRoleBuilder(provider.RoleIntradayBar)}
}

func (b IntradayBarBuilder) Markets(markets ...provider.Market) IntradayBarBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b IntradayBarBuilder) SecurityTypes(securityTypes ...provider.SecurityType) IntradayBarBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b IntradayBarBuilder) Group(group provider.GroupID) IntradayBarBuilder {
	b.role = b.role.group(group)
	return b
}

func (b IntradayBarBuilder) Operations(operations ...provider.OperationID) IntradayBarBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b IntradayBarBuilder) RequiresAuth(scope provider.CredentialScope) IntradayBarBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b IntradayBarBuilder) NoAuth() IntradayBarBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b IntradayBarBuilder) Freshness(freshness provider.Freshness) IntradayBarBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b IntradayBarBuilder) Compatibility(source CompatibilitySource) IntradayBarBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b IntradayBarBuilder) CompatibilityValue(compatibility provider.Compatibility) IntradayBarBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b IntradayBarBuilder) Priority(priority int) IntradayBarBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b IntradayBarBuilder) Limitations(limitations ...string) IntradayBarBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b IntradayBarBuilder) Build() (intradaybar.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return intradaybar.Profile{}, err
	}
	return intradaybar.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b IntradayBarBuilder) MustBuild() intradaybar.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type OrderbookBuilder struct {
	role roleBuilder
}

func Orderbook() OrderbookBuilder {
	return OrderbookBuilder{role: newRoleBuilder(provider.RoleOrderbook)}
}

func (b OrderbookBuilder) Markets(markets ...provider.Market) OrderbookBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b OrderbookBuilder) SecurityTypes(securityTypes ...provider.SecurityType) OrderbookBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b OrderbookBuilder) Group(group provider.GroupID) OrderbookBuilder {
	b.role = b.role.group(group)
	return b
}

func (b OrderbookBuilder) Operations(operations ...provider.OperationID) OrderbookBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b OrderbookBuilder) RequiresAuth(scope provider.CredentialScope) OrderbookBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b OrderbookBuilder) NoAuth() OrderbookBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b OrderbookBuilder) Freshness(freshness provider.Freshness) OrderbookBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b OrderbookBuilder) Compatibility(source CompatibilitySource) OrderbookBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b OrderbookBuilder) CompatibilityValue(compatibility provider.Compatibility) OrderbookBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b OrderbookBuilder) Priority(priority int) OrderbookBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b OrderbookBuilder) Limitations(limitations ...string) OrderbookBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b OrderbookBuilder) Build() (orderbook.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return orderbook.Profile{}, err
	}
	return orderbook.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b OrderbookBuilder) MustBuild() orderbook.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type TradesBuilder struct {
	role roleBuilder
}

func Trades() TradesBuilder {
	return TradesBuilder{role: newRoleBuilder(provider.RoleTrades)}
}

func (b TradesBuilder) Markets(markets ...provider.Market) TradesBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b TradesBuilder) SecurityTypes(securityTypes ...provider.SecurityType) TradesBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b TradesBuilder) Group(group provider.GroupID) TradesBuilder {
	b.role = b.role.group(group)
	return b
}

func (b TradesBuilder) Operations(operations ...provider.OperationID) TradesBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b TradesBuilder) RequiresAuth(scope provider.CredentialScope) TradesBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b TradesBuilder) NoAuth() TradesBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b TradesBuilder) Freshness(freshness provider.Freshness) TradesBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b TradesBuilder) Compatibility(source CompatibilitySource) TradesBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b TradesBuilder) CompatibilityValue(compatibility provider.Compatibility) TradesBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b TradesBuilder) Priority(priority int) TradesBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b TradesBuilder) Limitations(limitations ...string) TradesBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b TradesBuilder) Build() (trades.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return trades.Profile{}, err
	}
	return trades.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b TradesBuilder) MustBuild() trades.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}

type FinancialsBuilder struct {
	role roleBuilder
}

func Financials() FinancialsBuilder {
	return FinancialsBuilder{role: newRoleBuilder(provider.RoleFinancials)}
}

func (b FinancialsBuilder) Markets(markets ...provider.Market) FinancialsBuilder {
	b.role = b.role.markets(markets...)
	return b
}

func (b FinancialsBuilder) SecurityTypes(securityTypes ...provider.SecurityType) FinancialsBuilder {
	b.role = b.role.securityTypes(securityTypes...)
	return b
}

func (b FinancialsBuilder) Group(group provider.GroupID) FinancialsBuilder {
	b.role = b.role.group(group)
	return b
}

func (b FinancialsBuilder) Operations(operations ...provider.OperationID) FinancialsBuilder {
	b.role = b.role.operations(operations...)
	return b
}

func (b FinancialsBuilder) RequiresAuth(scope provider.CredentialScope) FinancialsBuilder {
	b.role = b.role.requiresAuth(scope)
	return b
}

func (b FinancialsBuilder) NoAuth() FinancialsBuilder {
	b.role = b.role.noAuth()
	return b
}

func (b FinancialsBuilder) Freshness(freshness provider.Freshness) FinancialsBuilder {
	b.role = b.role.freshness(freshness)
	return b
}

func (b FinancialsBuilder) Compatibility(source CompatibilitySource) FinancialsBuilder {
	b.role = b.role.compatibility(source)
	return b
}

func (b FinancialsBuilder) CompatibilityValue(compatibility provider.Compatibility) FinancialsBuilder {
	b.role = b.role.compatibilityValue(compatibility)
	return b
}

func (b FinancialsBuilder) Priority(priority int) FinancialsBuilder {
	b.role = b.role.priority(priority)
	return b
}

func (b FinancialsBuilder) Limitations(limitations ...string) FinancialsBuilder {
	b.role = b.role.limitations(limitations...)
	return b
}

func (b FinancialsBuilder) Build() (financials.Profile, error) {
	profile, err := b.role.build()
	if err != nil {
		return financials.Profile{}, err
	}
	return financials.Profile{
		Markets:       profile.Markets,
		SecurityTypes: profile.SecurityTypes,
		Group:         profile.Group,
		Operations:    profile.Operations,
		AuthScope:     profile.AuthScope,
		Freshness:     profile.Freshness,
		Compatibility: profile.Compatibility,
		RequiresAuth:  profile.RequiresAuth,
		Priority:      profile.Priority,
		Limitations:   profile.Limitations,
	}, nil
}

func (b FinancialsBuilder) MustBuild() financials.Profile {
	profile, err := b.Build()
	if err != nil {
		panic(err)
	}
	return profile
}
