package authcache

import (
	"context"
	"time"
)

type Key struct {
	ProviderID  string
	AuthScope   string
	Environment string
	AppKeyHash  string
}

type Token struct {
	Key
	AccessToken string
	TokenType   string
	ExpiresIn   int
	ExpiresAt   time.Time
	IssuedAt    time.Time
	UpdatedAt   time.Time
}

type Store interface {
	Get(ctx context.Context, key Key) (Token, bool, error)
	Put(ctx context.Context, token Token) error
}
