package dailybar

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coredailybar "github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
	"golang.org/x/text/currency"
)

const defaultPriceScale = 4

const (
	dailyBarExtensionNAV          = "nav"
	dailyBarExtensionListedShares = "stLstgCnt"
	dailyBarExtensionNetAsset     = "nPptTotAmt"
)

type dailyBarV2Record struct {
	InstrumentID int64
	SourceID     int64
	TradingDate  int

	OpenPrice    sql.NullInt64
	HighPrice    sql.NullInt64
	LowPrice     sql.NullInt64
	ClosePrice   sql.NullInt64
	ChangePrice  sql.NullInt64
	ChangeRateBP sql.NullInt64

	Volume            sql.NullInt64
	TradedAmountMinor sql.NullInt64
	MarketCapMinor    sql.NullInt64
	NAVPrice          sql.NullInt64
	ListedShares      sql.NullInt64
	NetAssetMinor     sql.NullInt64

	Market        string
	SecurityType  string
	Symbol        string
	ISIN          string
	Name          string
	CurrencyCode  string
	PriceScale    int
	Provider      string
	ProviderGroup string
	Operation     string
}

func validateBarKey(bar coredailybar.Bar) error {
	if bar.Market == "" || bar.SecurityType == "" || bar.TradingDate == "" || bar.Symbol == "" || bar.Provider == "" || bar.Group == "" {
		return oops.In("dailybar_repository").With(
			"provider", bar.Provider,
			"group", bar.Group,
			"market", bar.Market,
			"security_type", bar.SecurityType,
			"date", bar.TradingDate,
			"symbol", bar.Symbol,
		).New("daily bar missing sqlite key")
	}
	return nil
}

func encodeExtensions(extensions map[string]string) (string, error) {
	if len(extensions) == 0 {
		return "{}", nil
	}
	bytes, err := json.Marshal(extensions)
	if err != nil {
		return "", oops.In("dailybar_repository").Wrapf(err, "encode daily bar extensions")
	}
	return string(bytes), nil
}

func decodeExtensions(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" || raw == "{}" {
		return nil, nil
	}
	var extensions map[string]string
	if err := json.Unmarshal([]byte(raw), &extensions); err != nil {
		return nil, oops.In("dailybar_repository").With("raw", raw).Wrapf(err, "decode daily bar extensions")
	}
	return extensions, nil
}

func dailyBarV2ToCanonical(record dailyBarV2Record, extensions map[string]string) coredailybar.Bar {
	mergedExtensions := dailyBarV2ExtensionsToCanonical(record, extensions)
	return coredailybar.Bar{
		Provider:     provider.ProviderID(record.Provider),
		Group:        provider.GroupID(record.ProviderGroup),
		Operation:    provider.OperationID(record.Operation),
		Market:       provider.Market(record.Market),
		SecurityType: provider.SecurityType(record.SecurityType),
		Symbol:       record.Symbol,
		ISIN:         record.ISIN,
		Name:         record.Name,
		TradingDate:  formatTradingDate(record.TradingDate),
		Currency:     record.CurrencyCode,
		Open:         formatScaledDecimal(record.OpenPrice, record.PriceScale),
		High:         formatScaledDecimal(record.HighPrice, record.PriceScale),
		Low:          formatScaledDecimal(record.LowPrice, record.PriceScale),
		Close:        formatScaledDecimal(record.ClosePrice, record.PriceScale),
		Change:       formatScaledDecimal(record.ChangePrice, record.PriceScale),
		ChangeRate:   formatScaledDecimal(record.ChangeRateBP, 2),
		Volume:       formatScaledDecimal(record.Volume, 0),
		TradedValue:  formatScaledDecimal(record.TradedAmountMinor, currencyMinorScale(record.CurrencyCode)),
		MarketCap:    formatScaledDecimal(record.MarketCapMinor, currencyMinorScale(record.CurrencyCode)),
		Extensions:   mergedExtensions,
	}
}

func dailyBarV2ExtensionsToCanonical(record dailyBarV2Record, extensions map[string]string) map[string]string {
	merged := make(map[string]string, len(extensions)+3)
	for key, value := range extensions {
		merged[key] = value
	}
	if value := formatScaledDecimal(record.NAVPrice, record.PriceScale); value != "" {
		merged[dailyBarExtensionNAV] = value
	}
	if value := formatScaledDecimal(record.ListedShares, 0); value != "" {
		merged[dailyBarExtensionListedShares] = value
	}
	if value := formatScaledDecimal(record.NetAssetMinor, currencyMinorScale(record.CurrencyCode)); value != "" {
		merged[dailyBarExtensionNetAsset] = value
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func isPromotedDailyBarExtensionKey(key string) bool {
	switch key {
	case dailyBarExtensionNAV, dailyBarExtensionListedShares, dailyBarExtensionNetAsset:
		return true
	default:
		return false
	}
}

func dailyBarToV2Row(bar coredailybar.Bar, instrumentID, sourceID int64, priceScale, amountScale int, nowMS int64) (storage.DailyBarV2Row, error) {
	errb := oops.In("dailybar_repository").With(
		"provider", bar.Provider,
		"group", bar.Group,
		"market", bar.Market,
		"security_type", bar.SecurityType,
		"date", bar.TradingDate,
		"symbol", bar.Symbol,
	)
	tradingDate, err := parseTradingDate(bar.TradingDate)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.Wrap(err)
	}
	openPrice, err := parseScaledDecimal(bar.Open, priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "opening_price").Wrap(err)
	}
	highPrice, err := parseScaledDecimal(bar.High, priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "highest_price").Wrap(err)
	}
	lowPrice, err := parseScaledDecimal(bar.Low, priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "lowest_price").Wrap(err)
	}
	closePrice, err := parseScaledDecimal(bar.Close, priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "closing_price").Wrap(err)
	}
	changePrice, err := parseScaledDecimal(bar.Change, priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "price_change_from_previous_close").Wrap(err)
	}
	changeRateBP, err := parseScaledDecimal(bar.ChangeRate, 2)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "price_change_rate_from_previous_close").Wrap(err)
	}
	volume, err := parseScaledDecimal(bar.Volume, 0)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "traded_volume").Wrap(err)
	}
	tradedAmount, err := parseScaledDecimal(bar.TradedValue, amountScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "traded_amount").Wrap(err)
	}
	marketCap, err := parseScaledDecimal(bar.MarketCap, amountScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "market_capitalization").Wrap(err)
	}
	nav, err := parseScaledDecimal(bar.Extensions[dailyBarExtensionNAV], priceScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "extensions.nav").Wrap(err)
	}
	listedShares, err := parseScaledDecimal(bar.Extensions[dailyBarExtensionListedShares], 0)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "extensions.stLstgCnt").Wrap(err)
	}
	netAsset, err := parseScaledDecimal(bar.Extensions[dailyBarExtensionNetAsset], amountScale)
	if err != nil {
		return storage.DailyBarV2Row{}, errb.With("field", "extensions.nPptTotAmt").Wrap(err)
	}

	return storage.DailyBarV2Row{
		SchemaVersion:     storage.DailyBarV2SchemaVersion,
		InstrumentID:      instrumentID,
		SourceID:          sourceID,
		TradingDate:       tradingDate,
		OpenPrice:         openPrice,
		HighPrice:         highPrice,
		LowPrice:          lowPrice,
		ClosePrice:        closePrice,
		ChangePrice:       changePrice,
		ChangeRateBP:      changeRateBP,
		Volume:            volume,
		TradedAmountMinor: tradedAmount,
		MarketCapMinor:    marketCap,
		NAVPrice:          nav,
		ListedShares:      listedShares,
		NetAssetMinor:     netAsset,
		CreatedAtMS:       nowMS,
		UpdatedAtMS:       nowMS,
	}, nil
}

func parseTradingDate(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 8 {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, oops.In("dailybar_repository").With("date", value).Wrapf(err, "parse trading date")
		}
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return 0, oops.In("dailybar_repository").With("date", value).Wrapf(err, "parse trading date")
	}
	return parsed.Year()*10000 + int(parsed.Month())*100 + parsed.Day(), nil
}

func formatTradingDate(value int) string {
	year := value / 10000
	month := value / 100 % 100
	day := value % 100
	return strconv.Itoa(year) + "-" + twoDigits(month) + "-" + twoDigits(day)
}

func parseScaledDecimal(value string, scale int) (sql.NullInt64, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if trimmed == "" {
		return sql.NullInt64{}, nil
	}
	sign := int64(1)
	if strings.HasPrefix(trimmed, "-") {
		sign = -1
		trimmed = strings.TrimPrefix(trimmed, "-")
	} else {
		trimmed = strings.TrimPrefix(trimmed, "+")
	}
	if trimmed == "." {
		return sql.NullInt64{}, oops.In("dailybar_repository").With("value", value, "scale", scale).Errorf("invalid scaled decimal: value=%q scale=%d", value, scale)
	}
	if strings.HasPrefix(trimmed, ".") {
		trimmed = "0" + trimmed
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 || parts[0] == "" {
		return sql.NullInt64{}, oops.In("dailybar_repository").With("value", value, "scale", scale).Errorf("invalid scaled decimal: value=%q scale=%d", value, scale)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return sql.NullInt64{}, oops.In("dailybar_repository").With("value", value, "scale", scale).Wrapf(err, "parse scaled decimal")
	}
	fractional := ""
	if len(parts) == 2 {
		fractional = parts[1]
	}
	if len(fractional) > scale {
		extra := strings.Trim(fractional[scale:], "0")
		if extra != "" {
			return sql.NullInt64{}, oops.In("dailybar_repository").With("value", value, "scale", scale).New("scaled decimal has too many fractional digits")
		}
		fractional = fractional[:scale]
	}
	for len(fractional) < scale {
		fractional += "0"
	}
	frac := int64(0)
	if fractional != "" {
		parsed, err := strconv.ParseInt(fractional, 10, 64)
		if err != nil {
			return sql.NullInt64{}, oops.In("dailybar_repository").With("value", value, "scale", scale).Wrapf(err, "parse scaled decimal fraction")
		}
		frac = parsed
	}
	scaled := whole*pow10(scale) + frac
	return sql.NullInt64{Int64: scaled * sign, Valid: true}, nil
}

func formatScaledDecimal(value sql.NullInt64, scale int) string {
	if !value.Valid {
		return ""
	}
	if scale == 0 {
		return strconv.FormatInt(value.Int64, 10)
	}
	sign := ""
	raw := value.Int64
	if raw < 0 {
		sign = "-"
		raw = -raw
	}
	divisor := pow10(scale)
	whole := raw / divisor
	fractional := strconv.FormatInt(raw%divisor, 10)
	for len(fractional) < scale {
		fractional = "0" + fractional
	}
	fractional = strings.TrimRight(fractional, "0")
	if fractional == "" {
		return sign + strconv.FormatInt(whole, 10)
	}
	return sign + strconv.FormatInt(whole, 10) + "." + fractional
}

func currencyMinorScale(code string) int {
	unit, err := currency.ParseISO(strings.TrimSpace(code))
	if err != nil {
		return 0
	}
	scale, _ := currency.Standard.Rounding(unit)
	return scale
}

func normalizedCurrencyCode(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return "KRW"
	}
	return strings.ToUpper(trimmed)
}

func pow10(scale int) int64 {
	value := int64(1)
	for range scale {
		value *= 10
	}
	return value
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
