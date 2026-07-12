package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRequiresURI(t *testing.T) {
	require.Error(t, Config{}.Validate())
	require.Error(t, Config{Database: "mwosa"}.Validate())
	require.NoError(t, Config{URI: "mongodb://localhost:27017", Database: "mwosa"}.Validate())
}

func TestConfigWithDefaultsSetsDatabaseTimeoutAndAppName(t *testing.T) {
	cfg, err := Config{URI: "mongodb://localhost:27017"}.WithDefaults()

	require.NoError(t, err)
	require.Equal(t, DefaultDatabaseName, cfg.Database)
	require.Equal(t, DefaultAppName, cfg.AppName)
	require.Equal(t, DefaultTimeout, cfg.Timeout)
}

func TestConfigWithDefaultsUsesDatabaseFromURI(t *testing.T) {
	cfg, err := Config{URI: "mongodb://localhost:27017/from_uri?authSource=admin"}.WithDefaults()

	require.NoError(t, err)
	require.Equal(t, "from_uri", cfg.Database)
}

func TestConfigWithDefaultsPrefersExplicitDatabaseOverURI(t *testing.T) {
	cfg, err := Config{
		URI:      "mongodb://localhost:27017/from_uri",
		Database: "explicit_db",
	}.WithDefaults()

	require.NoError(t, err)
	require.Equal(t, "explicit_db", cfg.Database)
}

func TestConfigWithDefaultsAppliesDevelopmentHostnameScope(t *testing.T) {
	cfg, err := Config{
		URI:         "mongodb://localhost:27017/mwosa",
		Development: true,
		Hostname:    "Dev.Host.local",
	}.WithDefaults()

	require.NoError(t, err)
	require.Equal(t, "dev-host-local-mwosa", cfg.Database)
}

func TestDevelopmentDatabaseNameKeepsExistingScope(t *testing.T) {
	database, err := DevelopmentDatabaseName("dev-host-mwosa", "dev.host")

	require.NoError(t, err)
	require.Equal(t, "dev-host-mwosa", database)
}

func TestDevelopmentURIRewritesDatabasePath(t *testing.T) {
	uri, err := DevelopmentURI("mongodb://user:pass@localhost:27017/mwosa?authSource=admin", "dev.host")

	require.NoError(t, err)
	require.Equal(t, "mongodb://user:pass@localhost:27017/dev-host-mwosa?authSource=admin", uri)
}

func TestNewRuntimeRejectsInvalidConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runtime, err := NewRuntime(ctx, Config{})

	require.Nil(t, runtime)
	require.Error(t, err)
}
