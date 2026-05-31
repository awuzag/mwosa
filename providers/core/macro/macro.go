package macro

import (
	"context"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

type Preset string
type Frequency string

const (
	PresetKeyStatistics Preset = "key-statistics"

	FrequencyDaily     Frequency = "daily"
	FrequencyMonthly   Frequency = "monthly"
	FrequencyQuarterly Frequency = "quarterly"
	FrequencyAnnual    Frequency = "annual"
)

type Profile struct {
	Group         provider.GroupID
	Operations    []provider.OperationID
	AuthScope     provider.CredentialScope
	Presets       []Preset
	Freshness     provider.Freshness
	Compatibility provider.Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
	FetchMode     string
}

func (p Profile) RoleProfile() provider.RoleProfile {
	return provider.RoleProfile{
		Role:          provider.RoleMacro,
		Group:         p.Group,
		Operations:    p.Operations,
		AuthScope:     p.AuthScope,
		Freshness:     p.Freshness,
		Compatibility: p.Compatibility,
		RequiresAuth:  p.RequiresAuth,
		Priority:      p.Priority,
		Limitations:   p.Limitations,
		FetchMode:     p.FetchMode,
	}
}

type IndicatorInput struct {
	Preset Preset
}

type ObservationInput struct {
	IndicatorID string
	SourceCode  string
	From        string
	To          string
}

type Indicator struct {
	ID           string               `json:"indicator_id" csv:"indicator_id"`
	Preset       Preset               `json:"preset,omitempty" csv:"preset"`
	Provider     provider.ProviderID  `json:"provider" csv:"provider"`
	Group        provider.GroupID     `json:"provider_group,omitempty" csv:"provider_group"`
	Operation    provider.OperationID `json:"operation,omitempty" csv:"operation"`
	SourceCode   string               `json:"source_code" csv:"source_code"`
	SourceName   string               `json:"source_name,omitempty" csv:"source_name"`
	SourceURL    string               `json:"source_url,omitempty" csv:"source_url"`
	Name         string               `json:"name" csv:"name"`
	FriendlyName string               `json:"friendly_name,omitempty" csv:"friendly_name"`
	Category     string               `json:"category,omitempty" csv:"category"`
	Frequency    Frequency            `json:"frequency,omitempty" csv:"frequency"`
	Unit         string               `json:"unit,omitempty" csv:"unit"`
	Scale        string               `json:"scale,omitempty" csv:"scale"`
	Active       bool                 `json:"active" csv:"active"`

	ProviderDoc *ProviderDocument `json:"provider_document,omitempty" csv:"-"`
}

type Observation struct {
	IndicatorID string              `json:"indicator_id" csv:"indicator_id"`
	Provider    provider.ProviderID `json:"provider,omitempty" csv:"provider"`
	SourceCode  string              `json:"source_code,omitempty" csv:"source_code"`
	Period      string              `json:"period" csv:"period"`
	Value       string              `json:"value" csv:"value"`
	PublishedAt string              `json:"published_at,omitempty" csv:"published_at"`
	CollectedAt string              `json:"collected_at" csv:"collected_at"`
	Revision    int                 `json:"revision" csv:"revision"`
}

type ProviderDocument struct {
	IndicatorID   string              `json:"indicator_id,omitempty"`
	Provider      provider.ProviderID `json:"provider"`
	SchemaVersion string              `json:"schema_version"`
	Document      map[string]any      `json:"document"`
	UpdatedAt     string              `json:"updated_at,omitempty"`
}

type IndicatorResult struct {
	Indicators []Indicator
	Provider   provider.Identity
	Group      provider.GroupID
	Operation  provider.OperationID
	TotalCount int
}

type ObservationResult struct {
	Observations []Observation
	Provider     provider.Identity
	Group        provider.GroupID
	Operation    provider.OperationID
	TotalCount   int
}

type Fetcher interface {
	provider.RoleProvider
	FetchMacroIndicators(ctx context.Context, input IndicatorInput) (IndicatorResult, error)
	FetchMacroObservations(ctx context.Context, input ObservationInput) (ObservationResult, error)
	MacroProfile() Profile
}

type IndicatorFetchFunc func(context.Context, IndicatorInput) (IndicatorResult, error)
type ObservationFetchFunc func(context.Context, ObservationInput) (ObservationResult, error)

type Fetch struct {
	profile           Profile
	fetchIndicators   IndicatorFetchFunc
	fetchObservations ObservationFetchFunc
}

func NewFetch(profile Profile, fetchIndicators IndicatorFetchFunc, fetchObservations ObservationFetchFunc) Fetch {
	return Fetch{profile: profile, fetchIndicators: fetchIndicators, fetchObservations: fetchObservations}
}

func (f Fetch) FetchMacroIndicators(ctx context.Context, input IndicatorInput) (IndicatorResult, error) {
	if f.fetchIndicators == nil {
		return IndicatorResult{}, oops.In("provider_role").With("role", provider.RoleMacro).New("macro indicator fetch role is not configured")
	}
	return f.fetchIndicators(ctx, input)
}

func (f Fetch) FetchMacroObservations(ctx context.Context, input ObservationInput) (ObservationResult, error) {
	if f.fetchObservations == nil {
		return ObservationResult{}, oops.In("provider_role").With("role", provider.RoleMacro).New("macro observation fetch role is not configured")
	}
	return f.fetchObservations(ctx, input)
}

func (f Fetch) MacroProfile() Profile {
	return f.profile
}

func (f Fetch) RoleRegistration() provider.RoleRegistration {
	return provider.RoleRegistration{
		Profile: f.profile.RoleProfile(),
		Impl:    f,
	}
}
