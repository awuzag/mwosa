//go:build integration

package integrationtest

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	PostgresUser     = "mwosa"
	PostgresPassword = "mwosa-secret"
	PostgresDatabase = "mwosa"
)

type Postgres struct {
	DSN string
}

func StartPostgres(t *testing.T) Postgres {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithUsername(PostgresUser),
		postgres.WithPassword(PostgresPassword),
		postgres.WithDatabase(PostgresDatabase),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx, testcontainers.StopTimeout(10*time.Second)); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve postgres connection string: %v", err)
	}
	return Postgres{DSN: dsn}
}
