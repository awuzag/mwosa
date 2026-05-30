package quote

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type Profile struct {
	Markets       []provider.Market
	SecurityTypes []provider.SecurityType
	Group         provider.GroupID
	Operations    []provider.OperationID
	AuthScope     provider.CredentialScope
	Freshness     provider.Freshness
	Compatibility provider.Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
}

func (p Profile) RoleProfile() provider.RoleProfile {
	return provider.RoleProfile{
		Role:          provider.RoleQuote,
		Markets:       p.Markets,
		SecurityTypes: p.SecurityTypes,
		Group:         p.Group,
		Operations:    p.Operations,
		AuthScope:     p.AuthScope,
		Freshness:     p.Freshness,
		Compatibility: p.Compatibility,
		RequiresAuth:  p.RequiresAuth,
		Priority:      p.Priority,
		Limitations:   p.Limitations,
	}
}

type SnapshotInput struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
}

type SnapshotResult struct {
	Provider provider.Identity
	Symbol   string
	Price    string
}

type Snapshotter interface {
	provider.RoleProvider
	FetchQuoteSnapshot(ctx context.Context, input SnapshotInput) (SnapshotResult, error)
	QuoteProfile() Profile
}

type SnapshotFunc func(context.Context, SnapshotInput) (SnapshotResult, error)

type Snapshot struct {
	profile  Profile
	snapshot SnapshotFunc
}

func NewSnapshot(profile Profile, snapshot SnapshotFunc) Snapshot {
	return Snapshot{profile: profile, snapshot: snapshot}
}

func (s Snapshot) FetchQuoteSnapshot(ctx context.Context, input SnapshotInput) (SnapshotResult, error) {
	if s.snapshot == nil {
		return SnapshotResult{}, oops.In("provider_role").With("role", provider.RoleQuote).New("quote snapshot role is not configured")
	}
	return s.snapshot(ctx, input)
}

func (s Snapshot) QuoteProfile() Profile {
	return s.profile
}

func (s Snapshot) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: s.profile.RoleProfile(),
		Impl:    s,
	}
}
