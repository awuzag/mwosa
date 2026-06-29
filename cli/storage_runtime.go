package cli

import (
	"context"

	"github.com/awuzag/mwosa/app"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
)

func newSQLAppRuntime(ctx context.Context, opts *Options, feature string) (*app.Runtime, *storage.SQLDatabase, error) {
	runtime, err := newAppRuntime(ctx, opts, false)
	if err != nil {
		return nil, nil, err
	}
	database, err := runtime.Storage.RequireSQLDatabase(feature)
	if err != nil {
		return nil, nil, oops.Join(err, runtime.Close(ctx))
	}
	return runtime, database, nil
}
