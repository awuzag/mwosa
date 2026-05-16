package krx

import (
	"strings"
	"time"

	krxclient "github.com/ev3rlit/mwosa/clients/krx"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/indexbar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/samber/oops"
)

const currencyKRW = "KRW"

func resolveInputRange(fromText string, toText string) (time.Time, time.Time, error) {
	fromText = strings.TrimSpace(fromText)
	toText = strings.TrimSpace(toText)
	if fromText == "" && toText == "" {
		return time.Time{}, time.Time{}, oops.In("krx_adapter").New("KRX daily request requires --as-of or --from/--to")
	}
	if fromText == "" {
		fromText = toText
	}
	if toText == "" {
		toText = fromText
	}
	from, err := parseDate(fromText)
	if err != nil {
		return time.Time{}, time.Time{}, oops.In("krx_adapter").With("from", fromText).Wrapf(err, "parse KRX from date")
	}
	to, err := parseDate(toText)
	if err != nil {
		return time.Time{}, time.Time{}, oops.In("krx_adapter").With("to", toText).Wrapf(err, "parse KRX to date")
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, oops.In("krx_adapter").With("from", fromText, "to", toText).New("KRX date range requires from <= to")
	}
	return from, to, nil
}

func parseDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	for _, layout := range []string{"20060102", "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, oops.In("krx_adapter").With("date", value).New("date must be YYYYMMDD or YYYY-MM-DD")
}

func formatDate(value string) string {
	parsed, err := parseDate(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return parsed.Format("2006-01-02")
}

type indexDailyTradeRow struct {
	BaseDate            string
	IndexClass          string
	IndexName           string
	IndexClose          string
	IndexPreviousChange string
	FluctuationRate     string
	IndexOpen           string
	IndexHigh           string
	IndexLow            string
	Volume              string
	Amount              string
	MarketCap           string
}

func normalizeKRXIndex(row krxclient.KRXIndexDailyTrade) indexbar.Bar {
	return normalizeIndexDailyTrade(indexDailyTradeRow(row), provider.OperationKRXDDTrd, "KRX")
}

func normalizeKOSPIIndex(row krxclient.KOSPIIndexDailyTrade) indexbar.Bar {
	return normalizeIndexDailyTrade(indexDailyTradeRow(row), provider.OperationKOSPIDDTrd, "KOSPI")
}

func normalizeKOSDAQIndex(row krxclient.KOSDAQIndexDailyTrade) indexbar.Bar {
	return normalizeIndexDailyTrade(indexDailyTradeRow(row), provider.OperationKOSDAQDDTrd, "KOSDAQ")
}

func normalizeDerivativesProductIndex(row krxclient.DerivativesProductIndexDailyTrade) indexbar.Bar {
	return normalizeIndexDailyTrade(indexDailyTradeRow{
		BaseDate:            row.BaseDate,
		IndexClass:          row.IndexClass,
		IndexName:           row.IndexName,
		IndexClose:          row.IndexClose,
		IndexPreviousChange: row.IndexPreviousChange,
		FluctuationRate:     row.FluctuationRate,
		IndexOpen:           row.IndexOpen,
		IndexHigh:           row.IndexHigh,
		IndexLow:            row.IndexLow,
	}, provider.OperationDerivativesDDTrd, "DERIVATIVES")
}

func normalizeIndexDailyTrade(row indexDailyTradeRow, operation provider.OperationID, family string) indexbar.Bar {
	return indexbar.Bar{
		Provider:    provider.ProviderKRX,
		Group:       provider.GroupKRXIndexDailyTrade,
		Operation:   operation,
		Market:      provider.MarketKRX,
		IndexCode:   canonicalIndexCode(firstNonEmpty(row.IndexName, row.IndexClass)),
		Name:        strings.TrimSpace(row.IndexName),
		Family:      firstNonEmpty(strings.TrimSpace(row.IndexClass), family),
		TradingDate: formatDate(row.BaseDate),
		Currency:    currencyKRW,
		Open:        row.IndexOpen,
		High:        row.IndexHigh,
		Low:         row.IndexLow,
		Close:       row.IndexClose,
		Change:      row.IndexPreviousChange,
		ChangeRate:  row.FluctuationRate,
		Volume:      row.Volume,
		TradedValue: row.Amount,
		MarketCap:   row.MarketCap,
	}
}

func canonicalIndexCode(value string) string {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "코스피":
		return "KOSPI"
	case trimmed == "코스닥":
		return "KOSDAQ"
	}
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", "/", "")
	code := strings.ToUpper(replacer.Replace(trimmed))
	if code == "" {
		return ""
	}
	return code
}

func normalizeETF(row krxclient.ETFDailyTrade) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXETPDailyTrade,
		Operation:    provider.OperationETFByddTrd,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		TradingDate:  formatDate(row.BaseDate),
		Symbol:       row.IssueCode,
		Name:         row.IssueName,
		Currency:     currencyKRW,
		Open:         row.Open,
		High:         row.High,
		Low:          row.Low,
		Close:        row.Close,
		Change:       row.PreviousChange,
		ChangeRate:   row.FluctuationRate,
		Volume:       row.Volume,
		TradedValue:  row.Amount,
		MarketCap:    row.MarketCap,
		Extensions: compactExtensions(map[string]string{
			"nav":          row.NAV,
			"nPptTotAmt":   row.InvestmentAssetValue,
			"stLstgCnt":    row.ListedShares,
			"idxIndNm":     row.IndexName,
			"objStkprcIdx": row.IndexClose,
			"cmpPrevDdIdx": row.IndexChange,
			"flucRtIdx":    row.IndexFluctuationRate,
		}),
	}
}

func normalizeETN(row krxclient.ETNDailyTrade) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXETPDailyTrade,
		Operation:    provider.OperationETNByddTrd,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETN,
		TradingDate:  formatDate(row.BaseDate),
		Symbol:       row.IssueCode,
		Name:         row.IssueName,
		Currency:     currencyKRW,
		Open:         row.Open,
		High:         row.High,
		Low:          row.Low,
		Close:        row.Close,
		Change:       row.PreviousChange,
		ChangeRate:   row.FluctuationRate,
		Volume:       row.Volume,
		TradedValue:  row.Amount,
		MarketCap:    row.MarketCap,
		Extensions: compactExtensions(map[string]string{
			"indicativeValue": row.IndicativeValue,
			"indicValAmt":     row.IndicativeValueTotal,
			"stLstgCnt":       row.ListedShares,
			"idxIndNm":        row.IndexName,
			"objStkprcIdx":    row.IndexClose,
			"cmpPrevDdIdx":    row.IndexChange,
			"flucRtIdx":       row.IndexFluctuationRate,
		}),
	}
}

func normalizeELW(row krxclient.ELWDailyTrade) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXETPDailyTrade,
		Operation:    provider.OperationELWByddTrd,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeELW,
		TradingDate:  formatDate(row.BaseDate),
		Symbol:       row.IssueCode,
		Name:         row.IssueName,
		Currency:     currencyKRW,
		Open:         row.Open,
		High:         row.High,
		Low:          row.Low,
		Close:        row.Close,
		Change:       row.PreviousChange,
		Volume:       row.Volume,
		TradedValue:  row.Amount,
		MarketCap:    row.MarketCap,
		Extensions: compactExtensions(map[string]string{
			"stLstgCnt":        row.ListedShares,
			"underlyingName":   row.UnderlyingName,
			"underlyingPrice":  row.UnderlyingPrice,
			"underlyingChange": row.UnderlyingPreviousChange,
			"underlyingFlucRt": row.UnderlyingFluctuation,
		}),
	}
}

func normalizeStock(row krxclient.StockDailyTrade, operation provider.OperationID) dailybar.Bar {
	return dailybar.Bar{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXStockDailyTrade,
		Operation:    operation,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		TradingDate:  formatDate(row.BaseDate),
		Symbol:       row.IssueCode,
		Name:         row.IssueName,
		Currency:     currencyKRW,
		Open:         row.Open,
		High:         row.High,
		Low:          row.Low,
		Close:        row.Close,
		Change:       row.PreviousChange,
		ChangeRate:   row.FluctuationRate,
		Volume:       row.Volume,
		TradedValue:  row.Amount,
		MarketCap:    row.MarketCap,
		Extensions: compactExtensions(map[string]string{
			"marketName":      row.MarketName,
			"sectionTypeName": row.SectionTypeName,
			"stLstgCnt":       row.ListedShares,
		}),
	}
}

func normalizeKOSDAQStock(row krxclient.KOSDAQStockDailyTrade) dailybar.Bar {
	return normalizeStock(krxclient.StockDailyTrade(row), provider.OperationKOSDAQByddTrd)
}

func normalizeKONEXStock(row krxclient.KONEXStockDailyTrade) dailybar.Bar {
	return normalizeStock(krxclient.StockDailyTrade(row), provider.OperationKONEXByddTrd)
}

func normalizeStockIssue(row krxclient.StockIssueBaseInfo, operation provider.OperationID) instrument.Instrument {
	return instrument.Instrument{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXStockInstrument,
		Operation:    operation,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		SecurityCode: row.IssueShortCode,
		ISIN:         row.IssueCode,
		Name:         firstNonEmpty(row.IssueAbbreviation, row.IssueName),
		ExchangeCode: row.IssueShortCode,
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions: compactExtensions(map[string]string{
			"issueName":                row.IssueName,
			"issueEnglishName":         row.IssueEnglishName,
			"listingDate":              row.ListingDate,
			"marketTypeName":           row.MarketTypeName,
			"securityGroupName":        row.SecurityGroupName,
			"sectionTypeName":          row.SectionTypeName,
			"stockCertificateTypeName": row.StockCertificateTypeName,
			"parValue":                 row.ParValue,
			"listedShares":             row.ListedShares,
		}),
	}
}

func normalizeKOSDAQIssue(row krxclient.KOSDAQIssueBaseInfo) instrument.Instrument {
	return normalizeStockIssue(krxclient.StockIssueBaseInfo(row), provider.OperationKOSDAQIssueBaseInfo)
}

func normalizeKONEXIssue(row krxclient.KONEXIssueBaseInfo) instrument.Instrument {
	return normalizeStockIssue(krxclient.StockIssueBaseInfo(row), provider.OperationKONEXIssueBaseInfo)
}

func instrumentFromDailyBar(bar dailybar.Bar) instrument.Instrument {
	return instrument.Instrument{
		Provider:     bar.Provider,
		Group:        bar.Group,
		Operation:    bar.Operation,
		Market:       bar.Market,
		SecurityType: bar.SecurityType,
		SecurityCode: bar.Symbol,
		ISIN:         bar.ISIN,
		Name:         bar.Name,
		ExchangeCode: bar.Symbol,
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions:   bar.Extensions,
	}
}

func compactExtensions(values map[string]string) map[string]string {
	compacted := make(map[string]string, len(values))
	for key, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			compacted[key] = value
		}
	}
	if len(compacted) == 0 {
		return nil
	}
	return compacted
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
