package app

import (
	"context"

	providerrawstorage "github.com/awuzag/mwosa/storage/providerraw"
)

type ProviderRawRepository interface {
	UpsertSnapshot(context.Context, providerrawstorage.Snapshot) (providerrawstorage.WriteResult, error)
	ListSnapshots(context.Context, providerrawstorage.Query) ([]providerrawstorage.SnapshotRecord, error)
}
