package datago

import (
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
)

const (
	serviceKeyEnv                    = "MWOSA_DATAGO_SERVICE_KEY"
	serviceKeyFallbackEnv            = "DATAGO_SERVICE_KEY"
	baseURLEnv                       = "MWOSA_DATAGO_BASE_URL"
	etpServiceKeyEnv                 = "MWOSA_DATAGO_ETP_SERVICE_KEY"
	etpServiceKeyFallbackEnv         = "DATAGO_ETP_SERVICE_KEY"
	etpBaseURLEnv                    = "MWOSA_DATAGO_ETP_BASE_URL"
	stockServiceKeyEnv               = "MWOSA_DATAGO_STOCK_PRICE_SERVICE_KEY"
	stockServiceKeyFallbackEnv       = "DATAGO_STOCK_PRICE_SERVICE_KEY"
	stockBaseURLEnv                  = "MWOSA_DATAGO_STOCK_PRICE_BASE_URL"
	corporateFinanceServiceKeyEnv    = "MWOSA_DATAGO_CORPFIN_SERVICE_KEY"
	corporateFinanceBaseURLEnv       = "MWOSA_DATAGO_CORPFIN_BASE_URL"
	krxListedInfoServiceKeyEnv       = "MWOSA_DATAGO_KRX_LISTED_SERVICE_KEY"
	krxListedInfoBaseURLEnv          = "MWOSA_DATAGO_KRX_LISTED_BASE_URL"
	corporateFinanceDependencyConfig = "dependencies.krxListedInfo"
)

type Builder struct{}
type CorporateFinanceBuilder struct{}

var _ provider.ProviderBuilder = Builder{}
var _ provider.ProviderBuilder = CorporateFinanceBuilder{}

func NewBuilder() Builder {
	return Builder{}
}

func NewCorporateFinanceBuilder() CorporateFinanceBuilder {
	return CorporateFinanceBuilder{}
}

func (Builder) ID() provider.ProviderID {
	return provider.ProviderDataGo
}

func (CorporateFinanceBuilder) ID() provider.ProviderID {
	return provider.ProviderDataGoCorporateFinance
}

func (Builder) DefaultConfig() provider.Config {
	return provider.Config{
		"id":       string(provider.ProviderDataGo),
		"enabled":  true,
		"base_url": "",
		"auth": map[string]any{
			"service_key": "",
		},
		"groups": map[string]any{
			string(provider.GroupSecuritiesProductPrice): map[string]any{
				"enabled": true,
				"auth": map[string]any{
					"service_key": "",
				},
				"base_url": "",
			},
			string(provider.GroupStockPrice): map[string]any{
				"enabled": true,
				"auth": map[string]any{
					"service_key": "",
				},
				"base_url": "",
			},
		},
	}
}

func (CorporateFinanceBuilder) DefaultConfig() provider.Config {
	return provider.Config{
		"id":       string(provider.ProviderDataGoCorporateFinance),
		"enabled":  true,
		"base_url": "",
		"auth": map[string]any{
			"service_key": "",
		},
		"groups": map[string]any{
			string(provider.GroupCorporateFinance): map[string]any{
				"enabled": true,
			},
		},
		"dependencies": map[string]any{
			string(provider.GroupKRXListedInfo): map[string]any{
				"base_url": "",
				"auth": map[string]any{
					"service_key": "",
				},
			},
		},
	}
}

func (Builder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{
		ProviderID: provider.ProviderDataGo,
		Fields: []provider.ConfigField{
			{
				Path:        "auth.service_key",
				Flag:        "service-key",
				Required:    false,
				Secret:      true,
				Description: "legacy 공공데이터포털 service key for securitiesProductPrice",
				Env:         []string{serviceKeyEnv, serviceKeyFallbackEnv},
			},
			{
				Path:        "groups.securitiesProductPrice.auth.service_key",
				Flag:        "etp-service-key",
				Required:    false,
				Secret:      true,
				Description: "공공데이터포털 securitiesProductPrice service key",
				Env:         []string{etpServiceKeyEnv, etpServiceKeyFallbackEnv},
			},
			{
				Path:        "groups.stockPrice.auth.service_key",
				Flag:        "stock-price-service-key",
				Required:    false,
				Secret:      true,
				Description: "공공데이터포털 stockPrice service key",
				Env:         []string{stockServiceKeyEnv, stockServiceKeyFallbackEnv},
			},
			{
				Path:        "base_url",
				Flag:        "base-url",
				Description: "legacy override datago securitiesProductPrice API base URL",
				Env:         []string{baseURLEnv},
			},
			{
				Path:        "groups.securitiesProductPrice.base_url",
				Flag:        "etp-base-url",
				Description: "override datago securitiesProductPrice API base URL",
				Env:         []string{etpBaseURLEnv},
			},
			{
				Path:        "groups.stockPrice.base_url",
				Flag:        "stock-price-base-url",
				Description: "override datago stockPrice API base URL",
				Env:         []string{stockBaseURLEnv},
			},
			{
				Path:        "groups.securitiesProductPrice.enabled",
				Description: "enable datago securitiesProductPrice group",
			},
			{
				Path:        "groups.stockPrice.enabled",
				Description: "enable datago stockPrice group",
			},
		},
	}
}

func (CorporateFinanceBuilder) ConfigSpec() provider.ConfigSpec {
	return provider.ConfigSpec{
		ProviderID: provider.ProviderDataGoCorporateFinance,
		Fields: []provider.ConfigField{
			{
				Path:        "auth.service_key",
				Flag:        "service-key",
				Required:    true,
				Secret:      true,
				Description: "공공데이터포털 corporateFinance service key",
				Env:         []string{corporateFinanceServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv},
			},
			{
				Path:        "base_url",
				Flag:        "corpfin-base-url",
				Description: "override datago corporateFinance API base URL",
				Env:         []string{corporateFinanceBaseURLEnv},
			},
			{
				Path:        corporateFinanceDependencyConfig + ".auth.service_key",
				Flag:        "krx-listed-service-key",
				Required:    true,
				Secret:      true,
				Description: "공공데이터포털 krxListedInfo service key used for company-name and KRX-code resolution",
				Env:         []string{krxListedInfoServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv},
			},
			{
				Path:        corporateFinanceDependencyConfig + ".base_url",
				Flag:        "krx-listed-base-url",
				Description: "override datago krxListedInfo API base URL",
				Env:         []string{krxListedInfoBaseURLEnv},
			},
			{
				Path:        "groups.corporateFinance.enabled",
				Description: "enable datago-corpfin corporateFinance group",
			},
		},
	}
}

func (Builder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderDataGo) {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "datago disabled",
		}
	}
	requested := requestsProvider(opts, provider.ProviderDataGo)
	if forcedOtherProvider(opts, provider.ProviderDataGo) && !requested {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "another provider is forced",
		}
	}
	if !anyDataGoGroupEnabledFromConfig(config) {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "datago groups disabled",
		}
	}
	if hasAnyDataGoGroupServiceKey(config) {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "datago config is present",
		}
	}
	if requested {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "datago requested",
		}
	}
	return provider.RegistrationDecision{
		Register: false,
		Reason:   "datago config missing",
	}
}

func (CorporateFinanceBuilder) Decide(opts provider.RegisterOptions, config provider.Config) provider.RegistrationDecision {
	if !providerEnabledFromConfig(config, provider.ProviderDataGoCorporateFinance) {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "datago-corpfin disabled",
		}
	}
	requested := requestsProvider(opts, provider.ProviderDataGoCorporateFinance)
	if forcedOtherProvider(opts, provider.ProviderDataGoCorporateFinance) && !requested {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "another provider is forced",
		}
	}
	if !corporateFinanceEnabledFromConfig(config) {
		return provider.RegistrationDecision{
			Register: false,
			Reason:   "datago-corpfin corporateFinance group disabled",
		}
	}
	if corporateFinanceServiceKeyFromConfig(config) != "" {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "datago-corpfin config is present",
		}
	}
	if requested {
		return provider.RegistrationDecision{
			Register: true,
			Reason:   "datago-corpfin requested",
		}
	}
	return provider.RegistrationDecision{
		Register: false,
		Reason:   "datago-corpfin config missing",
	}
}

func (Builder) Build(config provider.Config) (provider.IdentityProvider, error) {
	etpServiceKey := etpServiceKeyFromConfig(config)
	stockServiceKey := stockServiceKeyFromConfig(config)
	if etpServiceKey == "" && stockServiceKey == "" {
		return nil, oops.In("provider_registry").
			With("provider", provider.ProviderDataGo).
			New("datago provider config requires at least one group service key: configure providers.datago.groups.securitiesProductPrice.auth.service_key or providers.datago.groups.stockPrice.auth.service_key")
	}
	return New(Config{
		ServiceKey: legacyServiceKeyFromConfig(config),
		BaseURL:    legacyBaseURLFromConfig(config),
		SecuritiesProductPrice: GroupConfig{
			ServiceKey: etpServiceKey,
			BaseURL:    etpBaseURLFromConfig(config),
		},
		StockPrice: GroupConfig{
			ServiceKey: stockServiceKey,
			BaseURL:    stockBaseURLFromConfig(config),
		},
	})
}

func (CorporateFinanceBuilder) Build(config provider.Config) (provider.IdentityProvider, error) {
	serviceKey := corporateFinanceServiceKeyFromConfig(config)
	if serviceKey == "" {
		return nil, missingServiceKeyError(provider.ProviderDataGoCorporateFinance, "providers.datago-corpfin.auth.service_key", corporateFinanceServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv)
	}
	listedInfoKey := krxListedInfoServiceKeyFromConfig(config)
	if listedInfoKey == "" {
		return nil, missingServiceKeyError(provider.ProviderDataGoCorporateFinance, "providers.datago-corpfin.dependencies.krxListedInfo.auth.service_key", krxListedInfoServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv)
	}
	return NewCorporateFinance(CorporateFinanceConfig{
		ServiceKey:              serviceKey,
		CorporateFinanceBaseURL: corporateFinanceBaseURLFromConfig(config),
		KRXListedInfoServiceKey: listedInfoKey,
		KRXListedInfoBaseURL:    krxListedInfoBaseURLFromConfig(config),
		RetryMaxAttempts:        0,
	})
}

func hasAnyDataGoGroupServiceKey(config provider.Config) bool {
	return etpServiceKeyFromConfig(config) != "" || stockServiceKeyFromConfig(config) != ""
}

func etpServiceKeyFromConfig(config provider.Config) string {
	if !dataGoGroupEnabledFromConfig(config, provider.GroupSecuritiesProductPrice) {
		return ""
	}
	serviceKey := stringFromConfigOrEnv(
		config,
		[]string{"providers", string(provider.ProviderDataGo), "groups", string(provider.GroupSecuritiesProductPrice), "auth", "service_key"},
		etpServiceKeyEnv,
		etpServiceKeyFallbackEnv,
	)
	if serviceKey != "" {
		return serviceKey
	}
	return legacyServiceKeyFromConfig(config)
}

func stockServiceKeyFromConfig(config provider.Config) string {
	if !dataGoGroupEnabledFromConfig(config, provider.GroupStockPrice) {
		return ""
	}
	return stringFromConfigOrEnv(
		config,
		[]string{"providers", string(provider.ProviderDataGo), "groups", string(provider.GroupStockPrice), "auth", "service_key"},
		stockServiceKeyEnv,
		stockServiceKeyFallbackEnv,
	)
}

func legacyServiceKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGo), "auth", "service_key"}, serviceKeyEnv, serviceKeyFallbackEnv)
}

func etpBaseURLFromConfig(config provider.Config) string {
	baseURL := stringFromConfigOrEnv(
		config,
		[]string{"providers", string(provider.ProviderDataGo), "groups", string(provider.GroupSecuritiesProductPrice), "base_url"},
		etpBaseURLEnv,
	)
	if baseURL != "" {
		return baseURL
	}
	return legacyBaseURLFromConfig(config)
}

func stockBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(
		config,
		[]string{"providers", string(provider.ProviderDataGo), "groups", string(provider.GroupStockPrice), "base_url"},
		stockBaseURLEnv,
	)
}

func legacyBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGo), "base_url"}, baseURLEnv)
}

func corporateFinanceServiceKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGoCorporateFinance), "auth", "service_key"}, corporateFinanceServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv)
}

func krxListedInfoServiceKeyFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGoCorporateFinance), "dependencies", string(provider.GroupKRXListedInfo), "auth", "service_key"}, krxListedInfoServiceKeyEnv, serviceKeyEnv, serviceKeyFallbackEnv)
}

func corporateFinanceBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGoCorporateFinance), "base_url"}, corporateFinanceBaseURLEnv)
}

func krxListedInfoBaseURLFromConfig(config provider.Config) string {
	return stringFromConfigOrEnv(config, []string{"providers", string(provider.ProviderDataGoCorporateFinance), "dependencies", string(provider.GroupKRXListedInfo), "base_url"}, krxListedInfoBaseURLEnv)
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

func dataGoGroupEnabledFromConfig(config provider.Config, group provider.GroupID) bool {
	enabled, ok := config.Bool("providers", string(provider.ProviderDataGo), "groups", string(group), "enabled")
	return !ok || enabled
}

func corporateFinanceEnabledFromConfig(config provider.Config) bool {
	enabled, ok := config.Bool("providers", string(provider.ProviderDataGoCorporateFinance), "groups", string(provider.GroupCorporateFinance), "enabled")
	return !ok || enabled
}

func anyDataGoGroupEnabledFromConfig(config provider.Config) bool {
	return dataGoGroupEnabledFromConfig(config, provider.GroupSecuritiesProductPrice) ||
		dataGoGroupEnabledFromConfig(config, provider.GroupStockPrice)
}

func requestsProvider(opts provider.RegisterOptions, id provider.ProviderID) bool {
	return opts.ProviderID == id || opts.PreferProvider == id
}

func forcedOtherProvider(opts provider.RegisterOptions, id provider.ProviderID) bool {
	return opts.ProviderID != "" && opts.ProviderID != id
}

func missingServiceKeyError(id provider.ProviderID, path string, envs ...string) error {
	return oops.In("provider_registry").
		With("provider", id).
		Errorf("%s provider config requires service key: configure %s or set %s", id, path, strings.Join(envs, " or "))
}
