package ecos

import (
	"context"
	"net/http"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/macro"
	"github.com/samber/oops"
)

type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Now        func() time.Time
	Client     Client
}

// Client is the connection point where a future github.com/awuzag/ecos client is wired.
// It returns adapter-selected canonical records, not raw ECOS response bodies.
type Client interface {
	FetchKeyStatistics(ctx context.Context) ([]macro.Indicator, error)
	FetchObservations(ctx context.Context, input ObservationRequest) ([]macro.Observation, error)
}

type ObservationRequest struct {
	IndicatorID string
	SourceCode  string
	From        string
	To          string
}

type Provider struct {
	provider.Identity

	macro.Fetcher

	client Client
	now    func() time.Time
}

func New(config Config) (*Provider, error) {
	// The external ECOS client module is intentionally not wired yet. Keep the
	// adapter boundary explicit so the future client only has to satisfy Client.
	return NewWithClient(config.Client, config.Now), nil
}

func NewWithClient(client Client, now func() time.Time) *Provider {
	if now == nil {
		now = time.Now
	}
	p := &Provider{
		Identity: provider.Identity{
			ID:          provider.ProviderECOS,
			DisplayName: "Bank of Korea ECOS",
		},
		client: client,
		now:    now,
	}
	p.Fetcher = macro.NewFetch(macro.Profile{
		Group:      provider.GroupECOSKeyStatistics,
		Operations: []provider.OperationID{provider.OperationECOSKeyStatistics},
		AuthScope:  provider.CredentialScopeECOS,
		Presets:    []macro.Preset{macro.PresetKeyStatistics},
		Freshness:  provider.FreshnessDaily,
		Compatibility: provider.Compatibility{
			DataLatency: provider.DataLatencyHistorical,
			Notes: []string{
				"ECOS 100 key statistics metadata and observations are modeled as macro indicators.",
				"Provider documents store adapter-selected metadata only, never full raw ECOS responses.",
			},
		},
		RequiresAuth: true,
		Priority:     10,
		Limitations: []string{
			"live ECOS client wiring is pending",
		},
	}, p.fetchIndicators, p.fetchObservations)
	return p
}

func Register(registry *provider.Registry, p provider.IdentityProvider) error {
	return registry.RegisterProvider(p)
}

func (p *Provider) fetchIndicators(ctx context.Context, input macro.IndicatorInput) (macro.IndicatorResult, error) {
	errb := oops.In("ecos_adapter").With("provider", provider.ProviderECOS, "preset", input.Preset)
	if input.Preset != "" && input.Preset != macro.PresetKeyStatistics {
		return macro.IndicatorResult{}, provider.NewUnsupported(provider.UnsupportedError{
			Capability:  provider.RoleMacro,
			ProviderID:  provider.ProviderECOS,
			GroupID:     provider.GroupECOSKeyStatistics,
			OperationID: provider.OperationECOSKeyStatistics,
			Symbol:      string(input.Preset),
			Reason:      "ecos macro adapter currently supports key-statistics preset only",
		})
	}
	if p.client == nil {
		return macro.IndicatorResult{}, errb.New("ecos macro client is not configured; wire github.com/awuzag/ecos through providers/ecos.Client")
	}
	indicators, err := p.client.FetchKeyStatistics(ctx)
	if err != nil {
		return macro.IndicatorResult{}, errb.Wrapf(err, "fetch ECOS key statistics indicators")
	}
	indicators = p.normalizeIndicators(indicators)
	return macro.IndicatorResult{
		Indicators: indicators,
		Provider:   p.Identity,
		Group:      provider.GroupECOSKeyStatistics,
		Operation:  provider.OperationECOSKeyStatistics,
		TotalCount: len(indicators),
	}, nil
}

func (p *Provider) fetchObservations(ctx context.Context, input macro.ObservationInput) (macro.ObservationResult, error) {
	errb := oops.In("ecos_adapter").With("provider", provider.ProviderECOS, "indicator_id", input.IndicatorID, "source_code", input.SourceCode, "from", input.From, "to", input.To)
	if strings.TrimSpace(input.IndicatorID) == "" {
		return macro.ObservationResult{}, errb.New("ecos observation request requires indicator id")
	}
	if p.client == nil {
		return macro.ObservationResult{}, errb.New("ecos macro client is not configured; wire github.com/awuzag/ecos through providers/ecos.Client")
	}
	observations, err := p.client.FetchObservations(ctx, ObservationRequest{
		IndicatorID: strings.TrimSpace(input.IndicatorID),
		SourceCode:  strings.TrimSpace(input.SourceCode),
		From:        strings.TrimSpace(input.From),
		To:          strings.TrimSpace(input.To),
	})
	if err != nil {
		return macro.ObservationResult{}, errb.Wrapf(err, "fetch ECOS macro observations")
	}
	observations = p.normalizeObservations(observations, input)
	return macro.ObservationResult{
		Observations: observations,
		Provider:     p.Identity,
		Group:        provider.GroupECOSKeyStatistics,
		Operation:    provider.OperationECOSKeyStatistics,
		TotalCount:   len(observations),
	}, nil
}

func (p *Provider) normalizeIndicators(indicators []macro.Indicator) []macro.Indicator {
	out := make([]macro.Indicator, 0, len(indicators))
	for _, indicator := range indicators {
		indicator.ID = strings.TrimSpace(indicator.ID)
		indicator.Preset = macro.PresetKeyStatistics
		indicator.Provider = provider.ProviderECOS
		indicator.Group = provider.GroupECOSKeyStatistics
		indicator.Operation = provider.OperationECOSKeyStatistics
		indicator.SourceCode = strings.TrimSpace(indicator.SourceCode)
		indicator.SourceName = strings.TrimSpace(indicator.SourceName)
		indicator.SourceURL = strings.TrimSpace(indicator.SourceURL)
		if indicator.ProviderDoc != nil {
			indicator.ProviderDoc.IndicatorID = indicator.ID
			indicator.ProviderDoc.Provider = provider.ProviderECOS
			if indicator.ProviderDoc.SchemaVersion == "" {
				indicator.ProviderDoc.SchemaVersion = "1.0.0"
			}
			if indicator.ProviderDoc.UpdatedAt == "" {
				indicator.ProviderDoc.UpdatedAt = p.now().UTC().Format(time.RFC3339)
			}
		}
		out = append(out, indicator)
	}
	return out
}

func (p *Provider) normalizeObservations(observations []macro.Observation, input macro.ObservationInput) []macro.Observation {
	out := make([]macro.Observation, 0, len(observations))
	for _, observation := range observations {
		if observation.IndicatorID == "" {
			observation.IndicatorID = strings.TrimSpace(input.IndicatorID)
		}
		if observation.Provider == "" {
			observation.Provider = provider.ProviderECOS
		}
		if observation.SourceCode == "" {
			observation.SourceCode = strings.TrimSpace(input.SourceCode)
		}
		if observation.CollectedAt == "" {
			observation.CollectedAt = p.now().UTC().Format(time.RFC3339)
		}
		out = append(out, observation)
	}
	return out
}
