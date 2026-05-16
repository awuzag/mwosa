package kis

import (
	"testing"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	"github.com/stretchr/testify/require"
)

func TestTokenCacheKeyHashesAppKeyAndIncludesEnvironment(t *testing.T) {
	key := newTokenCacheKey("plain-app-key", true)

	require.Equal(t, "kis", key.ProviderID)
	require.Equal(t, "kis", key.AuthScope)
	require.Equal(t, "virtual", key.Environment)
	require.NotEqual(t, "plain-app-key", key.AppKeyHash)
	require.Len(t, key.AppKeyHash, 64)
}

func TestCachedTokenFromKISPrefersProviderExpiration(t *testing.T) {
	issuedAt := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	key := newTokenCacheKey("app-key", false)

	token, err := cachedTokenFromKIS(key, kisclient.Token{
		AccessToken: "issued-token",
		TokenType:   "Bearer",
		ExpiresIn:   1,
		ExpiredAt:   "2026-05-11 09:00:00",
	}, issuedAt)

	require.NoError(t, err)
	require.Equal(t, "issued-token", token.AccessToken)
	require.Equal(t, time.Date(2026, 5, 11, 9, 0, 0, 0, time.Local), token.ExpiresAt)
}

func TestCachedTokenUsesExpiresInWhenProviderExpirationMissing(t *testing.T) {
	issuedAt := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	key := newTokenCacheKey("app-key", false)

	token, err := cachedTokenFromKIS(key, kisclient.Token{
		AccessToken: "issued-token",
		ExpiresIn:   3600,
	}, issuedAt)

	require.NoError(t, err)
	require.Equal(t, issuedAt.Add(time.Hour), token.ExpiresAt)
}

func TestCachedTokenValidAtUsesExpiryBuffer(t *testing.T) {
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	token := CachedToken{
		AccessToken: "cached-token",
		ExpiresAt:   now.Add(time.Minute),
	}

	require.False(t, cachedTokenValidAt(token, now, 2*time.Minute))
	require.True(t, cachedTokenValidAt(token, now, 30*time.Second))
}
