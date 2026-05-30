package spec

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/samber/oops"
)

func HistoricalFinancials(fetch financials.FetchFunc) FinancialsFetchBuilder {
	return FinancialsFetchBuilder{
		builder: Financials().
			Freshness(provider.FreshnessFiling).
			Compatibility(Historical().Notes("financial statements are filing-derived historical data")),
		fetch: fetch,
	}
}

type FinancialsFetchBuilder struct {
	builder FinancialsBuilder
	fetch   financials.FetchFunc
	err     error
}

func (b FinancialsFetchBuilder) Markets(markets ...provider.Market) FinancialsFetchBuilder {
	b.builder = b.builder.Markets(markets...)
	return b
}

func (b FinancialsFetchBuilder) SecurityTypes(securityTypes ...provider.SecurityType) FinancialsFetchBuilder {
	b.builder = b.builder.SecurityTypes(securityTypes...)
	return b
}

func (b FinancialsFetchBuilder) Group(group provider.GroupID) FinancialsFetchBuilder {
	b.builder = b.builder.Group(group)
	return b
}

func (b FinancialsFetchBuilder) Operations(operations ...provider.OperationID) FinancialsFetchBuilder {
	b.builder = b.builder.Operations(operations...)
	return b
}

func (b FinancialsFetchBuilder) RequiresAuth(scope provider.CredentialScope) FinancialsFetchBuilder {
	b.builder = b.builder.RequiresAuth(scope)
	return b
}

func (b FinancialsFetchBuilder) NoAuth() FinancialsFetchBuilder {
	b.builder = b.builder.NoAuth()
	return b
}

func (b FinancialsFetchBuilder) Freshness(freshness provider.Freshness) FinancialsFetchBuilder {
	b.builder = b.builder.Freshness(freshness)
	return b
}

func (b FinancialsFetchBuilder) Compatibility(source CompatibilitySource) FinancialsFetchBuilder {
	b.builder = b.builder.Compatibility(source)
	return b
}

func (b FinancialsFetchBuilder) CompatibilityNotes(notes ...string) FinancialsFetchBuilder {
	b.builder.role = b.builder.role.compatibilityNotes(notes...)
	return b
}

func (b FinancialsFetchBuilder) Priority(priority int) FinancialsFetchBuilder {
	b.builder = b.builder.Priority(priority)
	return b
}

func (b FinancialsFetchBuilder) Limitations(limitations ...string) FinancialsFetchBuilder {
	b.builder = b.builder.Limitations(limitations...)
	return b
}

func (b FinancialsFetchBuilder) Build() (financials.Fetch, error) {
	if b.err != nil {
		return financials.Fetch{}, b.err
	}
	if b.fetch == nil {
		return financials.Fetch{}, oops.In("provider_spec").With("role", provider.RoleFinancials).New("financials provider role requires callable fetch")
	}
	profile, err := b.builder.Build()
	if err != nil {
		return financials.Fetch{}, err
	}
	return financials.NewFetch(profile, b.fetch), nil
}

func (b FinancialsFetchBuilder) MustBuild() financials.Fetch {
	role, err := b.Build()
	if err != nil {
		panic(err)
	}
	return role
}
