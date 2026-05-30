package indexbar

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	coreindexbar "github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"golang.org/x/text/currency"
)

const defaultValueScale = 4

type indexBarRecord struct {
	IndexID     int64
	SourceID    int64
	TradingDate int

	OpenValue         sql.NullInt64
	HighValue         sql.NullInt64
	LowValue          sql.NullInt64
	CloseValue        sql.NullInt64
	ChangeValue       sql.NullInt64
	ChangeRateBP      sql.NullInt64
	Volume            sql.NullInt64
	TradedAmountMinor sql.NullInt64
	MarketCapMinor    sql.NullInt64

	Market         string
	IndexCode      string
	Name           string
	Family         string
	CurrencyCode   string
	Timezone       string
	IndexType      string
	IndexExtension string
	Provider       string
	ProviderGroup  string
	Operation      string
}

func validateBarKey(bar coreindexbar.Bar) error {
	if bar.Market == "" || bar.IndexCode == "" || bar.TradingDate == "" || bar.Provider == "" || bar.Group == "" || bar.Operation == "" {
		return oops.In("indexbar_repository").With(
			"provider", bar.Provider,
			"group", bar.Group,
			"operation", bar.Operation,
			"market", bar.Market,
			"index_code", bar.IndexCode,
			"date", bar.TradingDate,
		).New("index bar missing sqlite key")
	}
	return nil
}

func indexBarToRow(bar coreindexbar.Bar, indexID, sourceID int64, nowMS int64) (storage.IndexBarV1Row, error) {
	errb := oops.In("indexbar_repository").With(
		"provider", bar.Provider,
		"group", bar.Group,
		"operation", bar.Operation,
		"market", bar.Market,
		"index_code", bar.IndexCode,
		"date", bar.TradingDate,
	)
	tradingDate, err := parseTradingDate(bar.TradingDate)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.Wrap(err)
	}
	openValue, err := parseScaledDecimal(bar.Open, defaultValueScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "open_value").Wrap(err)
	}
	highValue, err := parseScaledDecimal(bar.High, defaultValueScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "high_value").Wrap(err)
	}
	lowValue, err := parseScaledDecimal(bar.Low, defaultValueScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "low_value").Wrap(err)
	}
	closeValue, err := parseScaledDecimal(bar.Close, defaultValueScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "close_value").Wrap(err)
	}
	changeValue, err := parseScaledDecimal(bar.Change, defaultValueScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "change_value").Wrap(err)
	}
	changeRateBP, err := parseScaledDecimal(bar.ChangeRate, 2)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "change_rate").Wrap(err)
	}
	volume, err := parseScaledDecimal(bar.Volume, 0)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "volume").Wrap(err)
	}
	amountScale := currencyMinorScale(normalizedCurrencyCode(bar.Currency))
	tradedAmount, err := parseScaledDecimal(bar.TradedValue, amountScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "traded_amount").Wrap(err)
	}
	marketCap, err := parseScaledDecimal(bar.MarketCap, amountScale)
	if err != nil {
		return storage.IndexBarV1Row{}, errb.With("field", "market_cap").Wrap(err)
	}
	return storage.IndexBarV1Row{
		SchemaVersion:     storage.IndexBarV1SchemaVersion,
		IndexID:           indexID,
		SourceID:          sourceID,
		TradingDate:       tradingDate,
		OpenValue:         openValue,
		HighValue:         highValue,
		LowValue:          lowValue,
		CloseValue:        closeValue,
		ChangeValue:       changeValue,
		ChangeRateBP:      changeRateBP,
		Volume:            volume,
		TradedAmountMinor: tradedAmount,
		MarketCapMinor:    marketCap,
		CreatedAtMS:       nowMS,
		UpdatedAtMS:       nowMS,
	}, nil
}

func recordToCanonical(record indexBarRecord, indexExtensions map[string]string, barExtensions map[string]string) coreindexbar.Bar {
	extensions := make(map[string]string, len(indexExtensions)+len(barExtensions)+2)
	for key, value := range indexExtensions {
		extensions["index."+key] = value
	}
	if record.Timezone != "" {
		extensions["index.timezone"] = record.Timezone
	}
	if record.IndexType != "" {
		extensions["index.type"] = record.IndexType
	}
	for key, value := range barExtensions {
		extensions[key] = value
	}
	if len(extensions) == 0 {
		extensions = nil
	}
	return coreindexbar.Bar{
		Provider:    provider.ProviderID(record.Provider),
		Group:       provider.GroupID(record.ProviderGroup),
		Operation:   provider.OperationID(record.Operation),
		Market:      provider.Market(record.Market),
		IndexCode:   record.IndexCode,
		Name:        record.Name,
		Family:      record.Family,
		TradingDate: formatTradingDate(record.TradingDate),
		Currency:    record.CurrencyCode,
		Open:        formatScaledDecimal(record.OpenValue, defaultValueScale),
		High:        formatScaledDecimal(record.HighValue, defaultValueScale),
		Low:         formatScaledDecimal(record.LowValue, defaultValueScale),
		Close:       formatScaledDecimal(record.CloseValue, defaultValueScale),
		Change:      formatScaledDecimal(record.ChangeValue, defaultValueScale),
		ChangeRate:  formatScaledDecimal(record.ChangeRateBP, 2),
		Volume:      formatScaledDecimal(record.Volume, 0),
		TradedValue: formatScaledDecimal(record.TradedAmountMinor, currencyMinorScale(record.CurrencyCode)),
		MarketCap:   formatScaledDecimal(record.MarketCapMinor, currencyMinorScale(record.CurrencyCode)),
		Extensions:  extensions,
	}
}

func encodeExtensions(extensions map[string]string) (string, error) {
	if len(extensions) == 0 {
		return "{}", nil
	}
	bytes, err := json.Marshal(extensions)
	if err != nil {
		return "", oops.In("indexbar_repository").Wrapf(err, "encode index extensions")
	}
	return string(bytes), nil
}

func decodeExtensions(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" || raw == "{}" {
		return nil, nil
	}
	var extensions map[string]string
	if err := json.Unmarshal([]byte(raw), &extensions); err != nil {
		return nil, oops.In("indexbar_repository").With("raw", raw).Wrapf(err, "decode index extensions")
	}
	return extensions, nil
}

func parseTradingDate(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 8 {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, oops.In("indexbar_repository").With("date", value).Wrapf(err, "parse trading date")
		}
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return 0, oops.In("indexbar_repository").With("date", value).Wrapf(err, "parse trading date")
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
		return sql.NullInt64{}, oops.In("indexbar_repository").With("value", value, "scale", scale).Errorf("invalid scaled decimal: value=%q scale=%d", value, scale)
	}
	if strings.HasPrefix(trimmed, ".") {
		trimmed = "0" + trimmed
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 || parts[0] == "" {
		return sql.NullInt64{}, oops.In("indexbar_repository").With("value", value, "scale", scale).Errorf("invalid scaled decimal: value=%q scale=%d", value, scale)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return sql.NullInt64{}, oops.In("indexbar_repository").With("value", value, "scale", scale).Wrapf(err, "parse scaled decimal")
	}
	fractional := ""
	if len(parts) == 2 {
		fractional = parts[1]
	}
	if len(fractional) > scale {
		extra := strings.Trim(fractional[scale:], "0")
		if extra != "" {
			return sql.NullInt64{}, oops.In("indexbar_repository").With("value", value, "scale", scale).New("scaled decimal has too many fractional digits")
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
			return sql.NullInt64{}, oops.In("indexbar_repository").With("value", value, "scale", scale).Wrapf(err, "parse scaled decimal fraction")
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
