package providerauth

import (
	"context"
	stdsql "database/sql"
	"errors"
	"strings"
	"time"

	"github.com/awuzag/mwosa/providers/core/authcache"
	"github.com/samber/oops"
)

type Repository struct {
	database *Database
}

var _ authcache.Store = (*Repository)(nil)

func NewRepository(database *Database) (*Repository, error) {
	if database == nil {
		return nil, oops.In("provider_auth_repository").New("provider auth repository database is nil")
	}
	return &Repository{database: database}, nil
}

func (r *Repository) Get(ctx context.Context, key authcache.Key) (authcache.Token, bool, error) {
	errb := tokenErrBuilder(key)
	if err := validateKey(key); err != nil {
		return authcache.Token{}, false, errb.Wrap(err)
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return authcache.Token{}, false, errb.Wrap(err)
	}

	var row TokenRow
	err = client.NewSelect().
		Model(&row).
		Where("provider_id = ?", key.ProviderID).
		Where("auth_scope = ?", key.AuthScope).
		Where("environment = ?", key.Environment).
		Where("app_key_hash = ?", key.AppKeyHash).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, stdsql.ErrNoRows) {
			return authcache.Token{}, false, nil
		}
		return authcache.Token{}, false, errb.Wrapf(err, "read provider auth token cache")
	}
	return tokenFromRow(row), true, nil
}

func (r *Repository) Put(ctx context.Context, token authcache.Token) error {
	errb := tokenErrBuilder(token.Key)
	if err := validateToken(token); err != nil {
		return errb.Wrap(err)
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return errb.Wrap(err)
	}

	row := tokenToRow(token)
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now()
	}
	_, err = client.NewInsert().
		Model(&row).
		On("CONFLICT (provider_id, auth_scope, environment, app_key_hash) DO UPDATE").
		Set("access_token = EXCLUDED.access_token").
		Set("token_type = EXCLUDED.token_type").
		Set("expires_in = EXCLUDED.expires_in").
		Set("expires_at = EXCLUDED.expires_at").
		Set("issued_at = EXCLUDED.issued_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return errb.Wrapf(err, "upsert provider auth token cache")
	}
	return nil
}

func validateKey(key authcache.Key) error {
	errb := oops.In("provider_auth_repository")
	if strings.TrimSpace(key.ProviderID) == "" {
		return errb.New("provider auth token cache key requires provider id")
	}
	if strings.TrimSpace(key.AuthScope) == "" {
		return errb.New("provider auth token cache key requires auth scope")
	}
	if strings.TrimSpace(key.Environment) == "" {
		return errb.New("provider auth token cache key requires environment")
	}
	if strings.TrimSpace(key.AppKeyHash) == "" {
		return errb.New("provider auth token cache key requires app key hash")
	}
	return nil
}

func validateToken(token authcache.Token) error {
	if err := validateKey(token.Key); err != nil {
		return err
	}
	errb := oops.In("provider_auth_repository")
	if strings.TrimSpace(token.AccessToken) == "" {
		return errb.New("provider auth token cache requires access token")
	}
	if token.ExpiresAt.IsZero() {
		return errb.New("provider auth token cache requires expires at")
	}
	if token.IssuedAt.IsZero() {
		return errb.New("provider auth token cache requires issued at")
	}
	return nil
}

func tokenErrBuilder(key authcache.Key) oops.OopsErrorBuilder {
	return oops.In("provider_auth_repository").With(
		"provider", key.ProviderID,
		"auth_scope", key.AuthScope,
		"environment", key.Environment,
		"app_key_hash", key.AppKeyHash,
	)
}

func tokenToRow(token authcache.Token) TokenRow {
	return TokenRow{
		ProviderID:  strings.TrimSpace(token.ProviderID),
		AuthScope:   strings.TrimSpace(token.AuthScope),
		Environment: strings.TrimSpace(token.Environment),
		AppKeyHash:  strings.TrimSpace(token.AppKeyHash),
		AccessToken: strings.TrimSpace(token.AccessToken),
		TokenType:   strings.TrimSpace(token.TokenType),
		ExpiresIn:   token.ExpiresIn,
		ExpiresAt:   token.ExpiresAt,
		IssuedAt:    token.IssuedAt,
		UpdatedAt:   token.UpdatedAt,
	}
}

func tokenFromRow(row TokenRow) authcache.Token {
	return authcache.Token{
		Key: authcache.Key{
			ProviderID:  row.ProviderID,
			AuthScope:   row.AuthScope,
			Environment: row.Environment,
			AppKeyHash:  row.AppKeyHash,
		},
		AccessToken: strings.TrimSpace(row.AccessToken),
		TokenType:   strings.TrimSpace(row.TokenType),
		ExpiresIn:   row.ExpiresIn,
		ExpiresAt:   row.ExpiresAt,
		IssuedAt:    row.IssuedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
