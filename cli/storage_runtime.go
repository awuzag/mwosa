package cli

import (
	"github.com/awuzag/mwosa/app"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
)

func newSQLAppRuntime(opts *Options, feature string) (*app.Runtime, *storage.SQLDatabase, error) {
	runtime, err := newAppRuntime(opts, false)
	if err != nil {
		return nil, nil, err
	}
	database, err := runtime.Storage.RequireSQLDatabase(feature)
	if err != nil {
		return nil, nil, oops.Join(err, runtime.Close())
	}
	return runtime, database, nil
}
