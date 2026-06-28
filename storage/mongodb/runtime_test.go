package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRequiresURIAndDatabase(t *testing.T) {
	require.Error(t, Config{}.Validate())
	require.Error(t, Config{URI: "mongodb://localhost:27017"}.Validate())
	require.Error(t, Config{Database: "mwosa"}.Validate())
	require.NoError(t, Config{URI: "mongodb://localhost:27017", Database: "mwosa"}.Validate())
}

func TestConfigWithDefaultsSetsTimeoutAndAppName(t *testing.T) {
	cfg, err := Config{URI: "mongodb://localhost:27017", Database: "mwosa"}.WithDefaults()

	require.NoError(t, err)
	require.Equal(t, DefaultAppName, cfg.AppName)
	require.Equal(t, DefaultTimeout, cfg.Timeout)
}

func TestNewRuntimeRejectsInvalidConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runtime, err := NewRuntime(ctx, Config{})

	require.Nil(t, runtime)
	require.Error(t, err)
}
