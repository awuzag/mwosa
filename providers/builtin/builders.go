package builtin

import (
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/datago"
	"github.com/awuzag/mwosa/providers/kis"
	"github.com/awuzag/mwosa/providers/krx"
	"github.com/awuzag/mwosa/providers/opendart"
)

func Builders() []provider.ProviderBuilder {
	return []provider.ProviderBuilder{
		datago.NewBuilder(),
		datago.NewCorporateFinanceBuilder(),
		kis.NewBuilder(),
		krx.NewBuilder(),
		opendart.NewBuilder(),
	}
}
