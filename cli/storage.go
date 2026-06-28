package cli

import (
	"context"
	"strings"
	"time"

	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type storageMongoDBFlags struct {
	URI      string
	Database string
	Timeout  time.Duration
}

type storageInitResult struct {
	Status      string `json:"status"`
	Backend     string `json:"backend"`
	Database    string `json:"database"`
	Collections int    `json:"collections"`
}

type storageDoctorResult struct {
	Backend     string                                     `json:"backend"`
	Database    string                                     `json:"database"`
	Ping        storagemongodb.PingStatus                  `json:"ping"`
	Server      storagemongodb.ServerStatus                `json:"server"`
	Collections map[string]storagemongodb.CollectionStatus `json:"collections"`
}

func registerStorageCommands(roots commandRoots, opts *Options) {
	roots.Init.AddCommand(newInitStorageCommand(opts))
	roots.Doctor.AddCommand(newDoctorStorageCommand(opts))
}

func newInitStorageCommand(_ *Options) *cobra.Command {
	flags := storageMongoDBFlags{}
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Initialize MongoDB storage collections and indexes",
		Args:  cobra.NoArgs,
		RunE: runJSONResult(func(cmd *cobra.Command, _ []string) (any, error) {
			runtime, config, err := newStorageMongoDBRuntime(cmd.Context(), flags)
			if err != nil {
				return nil, err
			}
			if err := runtime.Init(cmd.Context()); err != nil {
				return nil, closeStorageMongoDBRuntime(runtime, oops.In("cli").Wrapf(err, "initialize mongodb storage"))
			}
			result := storageInitResult{
				Status:      "ok",
				Backend:     "mongodb",
				Database:    config.Database,
				Collections: len(storagemongodb.CollectionSpecs()),
			}
			if err := closeStorageMongoDBRuntime(runtime, nil); err != nil {
				return nil, err
			}
			return result, nil
		}),
	}
	bindStorageMongoDBFlags(cmd, &flags)
	return cmd
}

func newDoctorStorageCommand(_ *Options) *cobra.Command {
	flags := storageMongoDBFlags{}
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Diagnose MongoDB storage readiness",
		Args:  cobra.NoArgs,
		RunE: runJSONResult(func(cmd *cobra.Command, _ []string) (any, error) {
			runtime, config, err := newStorageMongoDBRuntime(cmd.Context(), flags)
			if err != nil {
				return nil, err
			}
			status, err := runtime.Check(cmd.Context())
			if err != nil {
				return nil, closeStorageMongoDBRuntime(runtime, oops.In("cli").Wrapf(err, "diagnose mongodb storage"))
			}
			result := storageDoctorResult{
				Backend:     "mongodb",
				Database:    config.Database,
				Ping:        status.Ping,
				Server:      status.Server,
				Collections: status.Collections,
			}
			if err := closeStorageMongoDBRuntime(runtime, nil); err != nil {
				return nil, err
			}
			return result, nil
		}),
	}
	bindStorageMongoDBFlags(cmd, &flags)
	return cmd
}

func closeStorageMongoDBRuntime(runtime *storagemongodb.Runtime, commandErr error) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return oops.Join(commandErr, runtime.Close(shutdownCtx))
}

func bindStorageMongoDBFlags(cmd *cobra.Command, flags *storageMongoDBFlags) {
	cmd.Flags().StringVar(&flags.URI, "mongodb-uri", flags.URI, "MongoDB connection URI")
	cmd.Flags().StringVar(&flags.Database, "mongodb-database", flags.Database, "MongoDB database name")
	cmd.Flags().DurationVar(&flags.Timeout, "mongodb-timeout", flags.Timeout, "MongoDB operation timeout")
}

func newStorageMongoDBRuntime(ctx context.Context, flags storageMongoDBFlags) (*storagemongodb.Runtime, storagemongodb.Config, error) {
	config := storagemongodb.Config{
		URI:      strings.TrimSpace(flags.URI),
		Database: strings.TrimSpace(flags.Database),
		Timeout:  flags.Timeout,
	}
	runtime, err := storagemongodb.NewRuntime(ctx, config)
	if err != nil {
		return nil, storagemongodb.Config{}, oops.In("cli").Wrapf(err, "create mongodb storage runtime")
	}
	applied, err := config.WithDefaults()
	if err != nil {
		_ = runtime.Close(context.Background())
		return nil, storagemongodb.Config{}, oops.In("cli").Wrap(err)
	}
	return runtime, applied, nil
}
