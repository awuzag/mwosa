package repositoryregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryReturnsRegisteredCreator(t *testing.T) {
	registry := New()
	creator := func(BuildContext) (Result, error) {
		return One("repository"), nil
	}
	require.NoError(t, registry.Register(NameDailyBar, BackendSQLite, creator))

	found, err := registry.Creator(NameDailyBar, BackendSQLite)
	require.NoError(t, err)
	result, err := found(BuildContext{Name: NameDailyBar, Backend: BackendSQLite})
	require.NoError(t, err)

	assert.Equal(t, []any{"repository"}, result.Values)
}

func TestRegistryRejectsDuplicateRegistration(t *testing.T) {
	registry := New()
	creator := func(BuildContext) (Result, error) {
		return One("repository"), nil
	}

	require.NoError(t, registry.Register(NameDailyBar, BackendSQLite, creator))
	err := registry.Register(NameDailyBar, BackendSQLite, creator)

	require.Error(t, err)
	assert.ErrorContains(t, err, "already registered")
}

func TestRegistryReportsMissingBackend(t *testing.T) {
	registry := New()
	creator := func(BuildContext) (Result, error) {
		return One("repository"), nil
	}
	require.NoError(t, registry.Register(NameDailyBar, BackendSQLite, creator))

	_, err := registry.Creator(NameDailyBar, BackendMongoDB)

	require.Error(t, err)
	assert.ErrorContains(t, err, "backend is not registered")
}
