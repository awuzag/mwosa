package opendart

import (
	"net/http"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	opendartsdk "github.com/ev3rlit/opendart"
	"github.com/samber/oops"
)

const (
	apiKeyEnv         = "OPENDART_API_KEY"
	apiKeyFallbackEnv = "MWOSA_OPENDART_API_KEY"
	baseURLEnv        = "MWOSA_OPENDART_BASE_URL"
)

type Builder struct{}

var _ provider.ProviderBuilder = Builder{}

func NewBuilder() Builder {
	return Builder{}
}

func (Builder) ID() provider.ProviderID {
	return provider.ProviderOpenDART
}

func (Builder) DefaultConfig() provider.Config {
	return provider.Config{
		"id":       string(provider.ProviderOpenDART),
		"enabled":  true,
		"base_url": "",
		"auth": map[string]any{
			"api_key": "",
		},
	}
}

func (Builder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{
		ProviderID: provider.ProviderOpenDART,
		Fields: []provider.ConfigField{
			{
				Path:        "auth.api_key",
				Flag:        "api-key",
				Required:    true,
				Secret:      true,
				Description: "OpenDART API key; OPENDART_API_KEY is preferred",
				Env:         []string{apiKeyEnv, apiKeyFallbackEnv},
			},
			{
				Path:        "base_url",
				Flag:        "base-url",
				Description: "override OpenDART base URL",
				Env:         []string{baseURLEnv},
			},
		},
	}
}

func (Builder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderOpenDART) {
		return provider.RegistrationDecision{Register: false, Reason: "opendart disabled"}
	}
	requested := requestsProvider(opts, provider.ProviderOpenDART)
	if forcedOtherProvider(opts, provider.ProviderOpenDART) && !requested {
		return provider.RegistrationDecision{Register: false, Reason: "another provider is forced"}
	}
	if apiKeyFromConfig(config) != "" {
		return provider.RegistrationDecision{Register: true, Reason: "opendart config is present"}
	}
	if requested {
		return provider.RegistrationDecision{Register: true, Reason: "opendart requested"}
	}
	return provider.RegistrationDecision{Register: false, Reason: "opendart config missing"}
}

func (Builder) Build(config provider.Config) (provider.IdentityProvider, error) {
	apiKey := apiKeyFromConfig(config)
	if apiKey == "" {
		return nil, oops.In("provider_registry").
			With("provider", provider.ProviderOpenDART).
			Errorf("opendart provider config requires API key: configure providers.opendart.auth.api_key or set %s", apiKeyEnv)
	}
	return New(Config{
		APIKey:     apiKey,
		BaseURL:    baseURLFromConfig(config),
		HTTPClient: nil,
	})
}

func apiKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderOpenDART), "auth", "api_key"}, apiKeyEnv, apiKeyFallbackEnv)
}

func baseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderOpenDART), "base_url"}, baseURLEnv)
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

func clientOptions(config Config) []opendartsdk.Option {
	options := []opendartsdk.Option{}
	if strings.TrimSpace(config.BaseURL) != "" {
		options = append(options, opendartsdk.WithBaseURL(config.BaseURL))
	}
	if config.HTTPClient != nil {
		options = append(options, opendartsdk.WithHTTPClient(config.HTTPClient))
	}
	return options
}

type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}
