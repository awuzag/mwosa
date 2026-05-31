package ecos

import (
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

const (
	apiKeyEnv         = "MWOSA_ECOS_API_KEY"
	apiKeyFallbackEnv = "ECOS_API_KEY"
	baseURLEnv        = "MWOSA_ECOS_BASE_URL"
)

type Builder struct{}

var _ provider.ProviderBuilder = Builder{}

func NewBuilder() Builder {
	return Builder{}
}

func (Builder) ID() provider.ProviderID {
	return provider.ProviderECOS
}

func (Builder) DefaultConfig() provider.Config {
	return provider.Config{
		"id":       string(provider.ProviderECOS),
		"enabled":  true,
		"base_url": "",
		"auth": map[string]any{
			"api_key": "",
		},
		"groups": map[string]any{
			string(provider.GroupECOSKeyStatistics): map[string]any{
				"enabled": true,
			},
		},
	}
}

func (Builder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{
		ProviderID: provider.ProviderECOS,
		Fields: []provider.ConfigField{
			{
				Path:        "auth.api_key",
				Flag:        "api-key",
				Required:    true,
				Secret:      true,
				Description: "Bank of Korea ECOS API key",
				Env:         []string{apiKeyEnv, apiKeyFallbackEnv},
			},
			{
				Path:        "base_url",
				Flag:        "base-url",
				Description: "override ECOS base URL",
				Env:         []string{baseURLEnv},
			},
		},
	}
}

func (Builder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderECOS) {
		return provider.RegistrationDecision{Register: false, Reason: "ecos disabled"}
	}
	requested := requestsProvider(opts, provider.ProviderECOS)
	if forcedOtherProvider(opts, provider.ProviderECOS) && !requested {
		return provider.RegistrationDecision{Register: false, Reason: "another provider is forced"}
	}
	if apiKeyFromConfig(config) != "" {
		return provider.RegistrationDecision{Register: true, Reason: "ecos config is present"}
	}
	if requested {
		return provider.RegistrationDecision{Register: true, Reason: "ecos requested"}
	}
	return provider.RegistrationDecision{Register: false, Reason: "ecos config missing"}
}

func (Builder) Build(config provider.Config) (provider.IdentityProvider, error) {
	apiKey := apiKeyFromConfig(config)
	if apiKey == "" {
		return nil, oops.In("provider_registry").
			With("provider", provider.ProviderECOS).
			Errorf("ecos provider config requires API key: configure providers.ecos.auth.api_key or set %s or %s", apiKeyEnv, apiKeyFallbackEnv)
	}
	return New(Config{
		APIKey:  apiKey,
		BaseURL: baseURLFromConfig(config),
	})
}

func apiKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderECOS), "auth", "api_key"}, apiKeyEnv, apiKeyFallbackEnv)
}

func baseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderECOS), "base_url"}, baseURLEnv)
}

func stringFromConfigOrEnv(config provider.Config, path []string, envs ...string) string {
	value := strings.TrimSpace(config.String(path...))
	if value != "" {
		return value
	}
	for _, env := range envs {
		value = strings.TrimSpace(config.Env(env))
		if value != "" {
			return value
		}
	}
	return ""
}

func providerEnabledFromConfig(config provider.Config, id provider.ProviderID) bool {
	enabled, ok := config.Bool("providers", string(id), "enabled")
	return !ok || enabled
}

func requestsProvider(opts provider.RegisterOptions, id provider.ProviderID) bool {
	return opts.ProviderID == id || opts.PreferProvider == id
}

func forcedOtherProvider(opts provider.RegisterOptions, id provider.ProviderID) bool {
	return opts.ProviderID != "" && opts.ProviderID != id
}
