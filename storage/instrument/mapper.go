package instrument

import (
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coreinstrument "github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/samber/oops"
)

const defaultPriceScale = 4

type record struct {
	InstrumentID int64
	SourceID     int64

	Provider      string
	ProviderGroup string
	Operation     string
	Market        string
	SecurityType  string
	Symbol        string
	ISIN          string
	Name          string
	CurrencyCode  string
	Timezone      string
}

func validateInstrumentKey(item coreinstrument.Instrument) error {
	symbol := canonicalSymbol(item)
	if item.Provider == "" || item.Group == "" || item.Operation == "" || item.SecurityType == "" || symbol == "" {
		return oops.In("instrument_repository").With(
			"provider", item.Provider,
			"group", item.Group,
			"operation", item.Operation,
			"market", item.Market,
			"security_type", item.SecurityType,
			"symbol", symbol,
		).New("instrument missing sqlite key")
	}
	return nil
}

func canonicalSymbol(item coreinstrument.Instrument) string {
	return firstNonEmpty(item.SecurityCode, item.ExchangeCode)
}

func providerSymbol(item coreinstrument.Instrument) string {
	return canonicalSymbol(item)
}

func sourceNaturalKey(item coreinstrument.Instrument) string {
	return strings.Join([]string{
		string(item.Provider),
		string(item.Group),
		string(item.Operation),
		providerSymbol(item),
	}, "\x00")
}

func recordToCanonical(row record, extensions map[string]string) coreinstrument.Instrument {
	return coreinstrument.Instrument{
		Provider:     provider.ProviderID(row.Provider),
		Group:        provider.GroupID(row.ProviderGroup),
		Operation:    provider.OperationID(row.Operation),
		Market:       provider.Market(row.Market),
		SecurityType: provider.SecurityType(row.SecurityType),
		SecurityCode: row.Symbol,
		ISIN:         row.ISIN,
		Name:         row.Name,
		ExchangeCode: row.Symbol,
		CountryCode:  countryCode(provider.Market(row.Market)),
		Timezone:     row.Timezone,
		Extensions:   extensions,
	}
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func countryCode(market provider.Market) string {
	if market == provider.MarketKRX {
		return "KR"
	}
	return ""
}

func marketTimezone(market provider.Market) string {
	if market == provider.MarketKRX {
		return "Asia/Seoul"
	}
	return "UTC"
}

func marketRegularOpenMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 9 * 60
	}
	return 0
}

func marketRegularCloseMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 15*60 + 30
	}
	return 0
}

func normalizedCurrencyCode(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "KRW"
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
