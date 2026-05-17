package builtin

import (
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/datago"
	"github.com/ev3rlit/mwosa/providers/kis"
	"github.com/ev3rlit/mwosa/providers/krx"
	"github.com/ev3rlit/mwosa/providers/opendart"
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
