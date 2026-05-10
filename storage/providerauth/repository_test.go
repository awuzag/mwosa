package providerauth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ev3rlit/mwosa/providers/core/authcache"
	"github.com/stretchr/testify/require"
)

func TestRepositoryStoresAndReadsTokenByScopeKey(t *testing.T) {
	ctx := context.Background()
	repository, closeDatabase := newTestRepository(t)
	defer closeDatabase()

	key := testTokenCacheKey("real", "hash-a")
	token := authcache.Token{
		Key:         key,
		AccessToken: "cached-token",
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		ExpiresAt:   time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
		IssuedAt:    time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
	}

	require.NoError(t, repository.Put(ctx, token))

	got, ok, err := repository.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "cached-token", got.AccessToken)
	require.Equal(t, "Bearer", got.TokenType)
	require.True(t, token.ExpiresAt.Equal(got.ExpiresAt))
}

func TestRepositoryTreatsDifferentEnvironmentOrAppKeyHashAsMiss(t *testing.T) {
	ctx := context.Background()
	repository, closeDatabase := newTestRepository(t)
	defer closeDatabase()

	key := testTokenCacheKey("real", "hash-a")
	require.NoError(t, repository.Put(ctx, authcache.Token{
		Key:         key,
		AccessToken: "cached-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		IssuedAt:    time.Now(),
		UpdatedAt:   time.Now(),
	}))

	for _, missKey := range []authcache.Key{
		testTokenCacheKey("virtual", "hash-a"),
		testTokenCacheKey("real", "hash-b"),
	} {
		_, ok, err := repository.Get(ctx, missKey)
		require.NoError(t, err)
		require.False(t, ok)
	}
}

func TestRepositoryCreatesSeparateProviderAuthSchema(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "provider-token-cache.sqlite")
	database := NewDatabase(dbPath)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	client, err := database.Client(ctx)
	require.NoError(t, err)
	require.FileExists(t, dbPath)

	rows, err := client.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	require.NoError(t, err)
	defer rows.Close()

	tables := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables[name] = true
	}
	require.NoError(t, rows.Err())
	require.True(t, tables["provider_auth_tokens"])
	require.False(t, tables["daily_bar"])
}

func TestRepositoryRejectsEmptyPathLazily(t *testing.T) {
	repository, err := NewRepository(NewDatabase(""))
	require.NoError(t, err)

	_, _, err = repository.Get(context.Background(), testTokenCacheKey("real", "hash-a"))
	require.Error(t, err)
}

func newTestRepository(t *testing.T) (*Repository, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "provider-token-cache.sqlite")
	database := NewDatabase(dbPath)
	repository, err := NewRepository(database)
	require.NoError(t, err)
	return repository, func() {
		require.NoError(t, database.Close())
		if _, err := os.Stat(dbPath); err != nil {
			require.NoError(t, err)
		}
	}
}

func testTokenCacheKey(environment string, appKeyHash string) authcache.Key {
	return authcache.Key{
		ProviderID:  "kis",
		AuthScope:   "kis",
		Environment: environment,
		AppKeyHash:  appKeyHash,
	}
}
