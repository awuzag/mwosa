package builtin

import (
	backteststorage "github.com/awuzag/mwosa/storage/backtest"
	compositionstorage "github.com/awuzag/mwosa/storage/composition"
	dailybarstorage "github.com/awuzag/mwosa/storage/dailybar"
	indexbarstorage "github.com/awuzag/mwosa/storage/indexbar"
	instrumentstorage "github.com/awuzag/mwosa/storage/instrument"
	macrostorage "github.com/awuzag/mwosa/storage/macro"
	providerrawstorage "github.com/awuzag/mwosa/storage/providerraw"
	"github.com/awuzag/mwosa/storage/repositoryregistry"
	strategystorage "github.com/awuzag/mwosa/storage/strategy"
	strategyfundamentalsstorage "github.com/awuzag/mwosa/storage/strategyfundamentals"
)

func Register(registry *repositoryregistry.Registry) error {
	for _, register := range []func(*repositoryregistry.Registry) error{
		dailybarstorage.Register,
		indexbarstorage.Register,
		macrostorage.Register,
		instrumentstorage.Register,
		compositionstorage.Register,
		strategystorage.Register,
		strategyfundamentalsstorage.Register,
		backteststorage.Register,
		providerrawstorage.Register,
	} {
		if err := register(registry); err != nil {
			return err
		}
	}
	return nil
}
