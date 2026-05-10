package kis

import (
	"strings"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

const (
	appKeyEnv                  = "MWOSA_KIS_APP_KEY"
	appKeyFallbackEnv          = "KIS_APP_KEY"
	appKeyDotenvFallbackEnv    = "APP_KEY"
	appSecretEnv               = "MWOSA_KIS_APP_SECRET"
	appSecretFallbackEnv       = "KIS_APP_SECRET"
	appSecretDotenvFallbackEnv = "APP_SECRET"
	accessTokenEnv             = "MWOSA_KIS_ACCESS_TOKEN"
	accessTokenFallbackEnv     = "KIS_ACCESS_TOKEN"
	baseURLEnv                 = "MWOSA_KIS_BASE_URL"
	baseURLFallbackEnv         = "KIS_BASE_URL"
	virtualBaseURLEnv          = "MWOSA_KIS_VIRTUAL_BASE_URL"
	virtualBaseURLFallbackEnv  = "KIS_VIRTUAL_BASE_URL"
	virtualEnv                 = "MWOSA_KIS_VIRTUAL"
	virtualFallbackEnv         = "KIS_VIRTUAL"
	customerTypeEnv            = "MWOSA_KIS_CUSTOMER_TYPE"
	accountEnv                 = "MWOSA_KIS_ACCOUNT"
)

type Builder struct {
	tokenCache TokenCache
}

var _ provider.ProviderBuilder = Builder{}

func NewBuilder() Builder {
	return Builder{}
}

func (b Builder) WithTokenCache(cache TokenCache) provider.ProviderBuilder {
	b.tokenCache = cache
	return b
}

func (Builder) ID() provider.ProviderID {
	return provider.ProviderKIS
}

func (Builder) DefaultConfig() provider.Config {
	return provider.Config{
		"id":               string(provider.ProviderKIS),
		"enabled":          true,
		"virtual":          false,
		"base_url":         "",
		"virtual_base_url": "",
		"auth": map[string]any{
			"app_key":      "",
			"app_secret":   "",
			"access_token": "",
			"account":      "",
		},
	}
}

func (Builder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{
		ProviderID: provider.ProviderKIS,
		Fields: []provider.ConfigField{
			{
				Path:        "auth.app_key",
				Flag:        "app-key",
				Required:    true,
				Secret:      true,
				Description: "KIS Developers app key",
				Env:         []string{appKeyEnv, appKeyFallbackEnv, appKeyDotenvFallbackEnv},
			},
			{
				Path:        "auth.app_secret",
				Flag:        "app-secret",
				Required:    true,
				Secret:      true,
				Description: "KIS Developers app secret",
				Env:         []string{appSecretEnv, appSecretFallbackEnv, appSecretDotenvFallbackEnv},
			},
			{
				Path:        "auth.access_token",
				Flag:        "access-token",
				Required:    false,
				Secret:      true,
				Description: "optional pre-issued KIS OAuth access token",
				Env:         []string{accessTokenEnv, accessTokenFallbackEnv},
			},
			{
				Path:        "auth.account",
				Flag:        "account",
				Required:    false,
				Secret:      true,
				Description: "optional KIS account identifier for future read-only account capabilities",
				Env:         []string{accountEnv},
			},
			{
				Path:        "virtual",
				Flag:        "virtual",
				Required:    false,
				Description: "use KIS virtual investment domain",
				Env:         []string{virtualEnv, virtualFallbackEnv},
			},
			{
				Path:        "base_url",
				Flag:        "base-url",
				Description: "override KIS real API base URL",
				Env:         []string{baseURLEnv, baseURLFallbackEnv},
			},
			{
				Path:        "virtual_base_url",
				Flag:        "virtual-base-url",
				Description: "override KIS virtual API base URL",
				Env:         []string{virtualBaseURLEnv, virtualBaseURLFallbackEnv},
			},
			{
				Path:        "customer_type",
				Flag:        "customer-type",
				Description: "KIS custtype header value",
				Env:         []string{customerTypeEnv},
			},
		},
	}
}

func (Builder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderKIS) {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "kis disabled",
		}
	}
	requested := requestsProvider(opts, provider.ProviderKIS)
	if forcedOtherProvider(opts, provider.ProviderKIS) && !requested {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "another provider is forced",
		}
	}
	if kisAppKeyFromConfig(config) != "" && kisAppSecretFromConfig(config) != "" {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "kis config is present",
		}
	}
	if requested {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "kis requested",
		}
	}
	return provider.RegistrationDecision{
		Register: false,
		Reason:   "kis config missing",
	}
}

func (b Builder) Build(config provider.Config) (provider.IdentityProvider, error) {
	appKey := kisAppKeyFromConfig(config)
	appSecret := kisAppSecretFromConfig(config)
	if appKey == "" {
		return nil, missingKISConfigError("providers.kis.auth.app_key", appKeyEnv, appKeyFallbackEnv, appKeyDotenvFallbackEnv)
	}
	if appSecret == "" {
		return nil, missingKISConfigError("providers.kis.auth.app_secret", appSecretEnv, appSecretFallbackEnv, appSecretDotenvFallbackEnv)
	}
	return New(Config{
		AppKey:         appKey,
		AppSecret:      appSecret,
		AccessToken:    kisAccessTokenFromConfig(config),
		BaseURL:        kisBaseURLFromConfig(config),
		VirtualBaseURL: kisVirtualBaseURLFromConfig(config),
		Virtual:        kisVirtualFromConfig(config),
		CustomerType:   kisCustomerTypeFromConfig(config),
		Account:        kisAccountFromConfig(config),
		TokenCache:     b.tokenCache,
	})
}

func kisAppKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "auth", "app_key"}, appKeyEnv, appKeyFallbackEnv, appKeyDotenvFallbackEnv)
}

func kisAppSecretFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "auth", "app_secret"}, appSecretEnv, appSecretFallbackEnv, appSecretDotenvFallbackEnv)
}

func kisAccessTokenFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "auth", "access_token"}, accessTokenEnv, accessTokenFallbackEnv)
}

func kisAccountFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "auth", "account"}, accountEnv)
}

func kisBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "base_url"}, baseURLEnv, baseURLFallbackEnv)
}

func kisVirtualBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "virtual_base_url"}, virtualBaseURLEnv, virtualBaseURLFallbackEnv)
}

func kisCustomerTypeFromConfig(config provider.Config) string {
	value := stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderKIS), "customer_type"}, customerTypeEnv)
	if value == "" {
		return kisclient.DefaultCustomerType
	}
	return value
}

func kisVirtualFromConfig(config provider.Config) bool {
	if enabled, ok := config.Bool("providers", string(provider.ProviderKIS), "virtual"); ok {
		return enabled
	}
	for _, env := range []string{virtualEnv, virtualFallbackEnv} {
		if value := strings.TrimSpace(config.Env(env)); value != "" {
			return strings.EqualFold(value, "1") || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
		}
	}
	return false
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

func missingKISConfigError(path string, envs ...string) error {
	return oops.In("provider_registry").
		With("provider", provider.ProviderKIS).
		Errorf("kis provider config requires credential: configure %s or set %s", path, strings.Join(envs, " or "))
}
