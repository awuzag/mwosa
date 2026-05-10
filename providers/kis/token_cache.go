package kis

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/authcache"
	"github.com/samber/oops"
)

const defaultTokenExpiryBuffer = 2 * time.Minute

type TokenCache = authcache.Store
type TokenCacheKey = authcache.Key
type CachedToken = authcache.Token

func newTokenCacheKey(appKey string, virtual bool) TokenCacheKey {
	return TokenCacheKey{
		ProviderID:  string(provider.ProviderKIS),
		AuthScope:   string(provider.CredentialScopeKIS),
		Environment: kisEnvironment(virtual),
		AppKeyHash:  appKeyHash(appKey),
	}
}

func appKeyHash(appKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(appKey)))
	return hex.EncodeToString(sum[:])
}

func kisEnvironment(virtual bool) string {
	if virtual {
		return "virtual"
	}
	return "real"
}

func cachedTokenFromKIS(key TokenCacheKey, token kisclient.Token, issuedAt time.Time) (CachedToken, error) {
	errb := oops.In("kis_token_cache").With(
		"provider", key.ProviderID,
		"auth_scope", key.AuthScope,
		"environment", key.Environment,
		"app_key_hash", key.AppKeyHash,
	)
	if strings.TrimSpace(token.AccessToken) == "" {
		return CachedToken{}, errb.New("kis token cache: access token is empty")
	}
	expiresAt, err := tokenExpiresAt(token, issuedAt)
	if err != nil {
		return CachedToken{}, errb.Wrap(err)
	}
	now := issuedAt
	return CachedToken{
		Key:         key,
		AccessToken: strings.TrimSpace(token.AccessToken),
		TokenType:   strings.TrimSpace(token.TokenType),
		ExpiresIn:   token.ExpiresIn,
		ExpiresAt:   expiresAt,
		IssuedAt:    issuedAt,
		UpdatedAt:   now,
	}, nil
}

func tokenExpiresAt(token kisclient.Token, issuedAt time.Time) (time.Time, error) {
	text := strings.TrimSpace(token.ExpiredAt)
	if text != "" {
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			time.RFC3339Nano,
		} {
			if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
				return parsed, nil
			}
		}
	}
	if token.ExpiresIn > 0 {
		return issuedAt.Add(time.Duration(token.ExpiresIn) * time.Second), nil
	}
	return time.Time{}, oops.In("kis_token_cache").New("kis token cache: token expiration is missing")
}

func cachedTokenValidAt(t CachedToken, now time.Time, buffer time.Duration) bool {
	if strings.TrimSpace(t.AccessToken) == "" {
		return false
	}
	if t.ExpiresAt.IsZero() {
		return false
	}
	return now.Add(buffer).Before(t.ExpiresAt)
}

func cachedTokenToKIS(t CachedToken) kisclient.Token {
	return kisclient.Token{
		AccessToken: strings.TrimSpace(t.AccessToken),
		TokenType:   strings.TrimSpace(t.TokenType),
		ExpiresIn:   t.ExpiresIn,
		ExpiredAt:   t.ExpiresAt.Format("2006-01-02 15:04:05"),
	}
}
