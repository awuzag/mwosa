package krx

import (
	"strings"

	krxclient "github.com/ev3rlit/mwosa/clients/krx"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

const (
	authKeyEnv           = "MWOSA_KRX_AUTH_KEY"
	authKeyFallbackEnv   = "KRX_AUTH_KEY"
	baseURLEnv           = "MWOSA_KRX_BASE_URL"
	sampleBaseURLEnv     = "MWOSA_KRX_SAMPLE_BASE_URL"
	useSampleEnv         = "MWOSA_KRX_USE_SAMPLE"
	defaultServiceEnable = true
)

type Builder struct{}

var _ provider.ProviderBuilder = Builder{}

func NewBuilder() Builder {
	return Builder{}
}

func (Builder) ID() provider.ProviderID {
	return provider.ProviderKRX
}

func (Builder) DefaultConfig() provider.Config {
	services := make(map[string]any, len(ServiceCatalog()))
	for _, service := range ServiceCatalog() {
		services[string(service.Operation)] = map[string]any{
			"enabled": defaultServiceEnable,
		}
	}
	return provider.Config{
		"id":              string(provider.ProviderKRX),
		"enabled":         true,
		"base_url":        "",
		"sample_base_url": "",
		"use_sample":      false,
		"auth": map[string]any{
			"auth_key": "",
		},
		"services": services,
	}
}

func (Builder) ConfigSpec() provider.ConfigSpec {
	fields := []provider.ConfigField{
		{
			Path:        "auth.auth_key",
			Flag:        "auth-key",
			Required:    true,
			Secret:      true,
			Description: "KRX OPEN API AUTH_KEY",
			Env:         []string{authKeyEnv, authKeyFallbackEnv},
		},
		{
			Path:        "base_url",
			Flag:        "base-url",
			Description: "override KRX OPEN API production base URL",
			Env:         []string{baseURLEnv},
		},
		{
			Path:        "sample_base_url",
			Flag:        "sample-base-url",
			Description: "override KRX OPEN API sample base URL",
			Env:         []string{sampleBaseURLEnv},
		},
		{
			Path:        "use_sample",
			Flag:        "use-sample",
			Description: "use the KRX OPEN API sample endpoint",
			Env:         []string{useSampleEnv},
		},
	}
	for _, service := range ServiceCatalog() {
		fields = append(fields, provider.ConfigField{
			Path:        "services." + string(service.Operation) + ".enabled",
			Description: "enable KRX OPEN API service " + string(service.Operation),
		})
	}
	return provider.ConfigSpec{
		ProviderID: provider.ProviderKRX,
		Fields:     fields,
	}
}

func (Builder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderKRX) {
		return provider.RegistrationDecision{Register: false, Reason: "krx disabled"}
	}
	requested := requestsProvider(opts, provider.ProviderKRX)
	if forcedOtherProvider(opts, provider.ProviderKRX) && !requested {
		return provider.RegistrationDecision{Register: false, Reason: "another provider is forced"}
	}
	if authKeyFromConfig(config) != "" {
		return provider.RegistrationDecision{Register: true, Reason: "krx config is present"}
	}
	if requested {
		return provider.RegistrationDecision{Register: true, Reason: "krx requested"}
	}
	return provider.RegistrationDecision{Register: false, Reason: "krx config missing"}
}

func (Builder) Build(config provider.Config) (provider.IdentityProvider, error) {
	authKey := authKeyFromConfig(config)
	if authKey == "" {
		return nil, oops.In("provider_registry").
			With("provider", provider.ProviderKRX).
			Errorf("krx provider config requires auth key: configure providers.krx.auth.auth_key or set %s or %s", authKeyEnv, authKeyFallbackEnv)
	}
	return New(Config{
		AuthKey:       authKey,
		BaseURL:       baseURLFromConfig(config),
		SampleBaseURL: sampleBaseURLFromConfig(config),
		UseSample:     useSampleFromConfig(config),
		EnabledAPIs:   enabledAPIsFromConfig(config),
	})
}

func authKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKRX), "auth", "auth_key"}, authKeyEnv, authKeyFallbackEnv)
}

func baseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKRX), "base_url"}, baseURLEnv)
}

func sampleBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKRX), "sample_base_url"}, sampleBaseURLEnv)
}

func useSampleFromConfig(config provider.Config) bool {
	if enabled, ok := config.Bool("providers", string(provider.ProviderKRX), "use_sample"); ok {
		return enabled
	}
	for _, env := range []string{useSampleEnv} {
		value := strings.TrimSpace(config.Env(env))
		if value != "" {
			return strings.EqualFold(value, "1") || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
		}
	}
	return false
}

func enabledAPIsFromConfig(config provider.Config) map[provider.OperationID]bool {
	enabled := make(map[provider.OperationID]bool, len(ServiceCatalog()))
	for _, service := range ServiceCatalog() {
		value, ok := config.Bool("providers", string(provider.ProviderKRX), "services", string(service.Operation), "enabled")
		if !ok {
			enabled[service.Operation] = defaultServiceEnable
			continue
		}
		enabled[service.Operation] = value
	}
	return enabled
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

func clientOptions(config Config) []krxclient.Option {
	options := []krxclient.Option{krxclient.WithAuthKey(config.AuthKey)}
	if strings.TrimSpace(config.BaseURL) != "" {
		options = append(options, krxclient.WithBaseURL(config.BaseURL))
	}
	if strings.TrimSpace(config.SampleBaseURL) != "" {
		options = append(options, krxclient.WithSampleBaseURL(config.SampleBaseURL))
	} else if config.UseSample {
		options = append(options, krxclient.WithSampleBaseURL(krxclient.DefaultSampleBaseURL))
	}
	if config.HTTPClient != nil {
		options = append(options, krxclient.WithHTTPClient(config.HTTPClient))
	}
	return options
}
