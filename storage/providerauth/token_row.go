package providerauth

import (
	"time"

	"github.com/uptrace/bun"
)

type TokenRow struct {
	bun.BaseModel `bun:"table:provider_auth_tokens,alias:provider_auth_tokens"`

	ID          int64     `bun:"id,pk,autoincrement"`
	ProviderID  string    `bun:"provider_id,notnull"`
	AuthScope   string    `bun:"auth_scope,notnull"`
	Environment string    `bun:"environment,notnull"`
	AppKeyHash  string    `bun:"app_key_hash,notnull"`
	AccessToken string    `bun:"access_token,notnull"`
	TokenType   string    `bun:"token_type,notnull,default:''"`
	ExpiresIn   int       `bun:"expires_in,notnull,default:0"`
	ExpiresAt   time.Time `bun:"expires_at,notnull"`
	IssuedAt    time.Time `bun:"issued_at,notnull"`
	UpdatedAt   time.Time `bun:"updated_at,notnull"`
}
