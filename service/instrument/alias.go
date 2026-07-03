package instrument

import (
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"unicode"

	provider "github.com/awuzag/mwosa/providers/core"
	instrumentrole "github.com/awuzag/mwosa/providers/core/instrument"
	"github.com/samber/oops"
	"golang.org/x/text/unicode/norm"
)

const (
	instrumentAliasExtensionKey = "alias"
	aliasMinLength              = 3
	aliasMaxLength              = 4
)

var aliasStopWords = map[string]bool{
	"A": true, "AN": true, "AND": true, "CO": true, "COMPANY": true,
	"CORP": true, "CORPORATION": true, "HOLDING": true, "HOLDINGS": true,
	"INC": true, "INCORPORATED": true, "INDUSTRIAL": true, "INDUSTRIES": true,
	"LIMITED": true, "LTD": true, "PLC": true, "THE": true,
}

var reservedAliases = map[string]bool{
	"AAPL": true, "MSFT": true, "NVDA": true, "TSLA": true, "META": true,
	"AMZN": true, "GOOG": true, "GOOGL": true, "NFLX": true, "SPY": true,
	"QQQ": true, "VOO": true, "VTI": true, "DIA": true, "IWM": true,
	"GLD": true, "SLV": true, "TLT": true, "ARKK": true, "USD": true,
	"KRW": true, "BTC": true, "ETH": true, "DART": true, "KRX": true,
	"ETF": true, "ETN": true, "ELW": true,
}

var etfBrands = []struct {
	needle string
	code   string
}{
	{"KODEX", "K"}, {"TIGER", "T"}, {"RISE", "R"}, {"SOL", "S"},
	{"ACE", "A"}, {"PLUS", "P"}, {"HANARO", "H"}, {"KBSTAR", "B"},
	{"KOSEF", "O"}, {"ARIRANG", "G"}, {"TIMEFOLIO", "F"}, {"KIWOOM", "W"},
	{"KOACT", "C"}, {"HK", "H"}, {"IBK", "I"}, {"BNK", "N"},
	{"1Q", "Q"}, {"WOORI", "W"}, {"마이티", "M"}, {"에셋플러스", "E"},
	{"파워", "P"}, {"히어로즈", "Z"},
}

var etfKeywords = []struct {
	needle string
	code   string
}{
	{"S&P500", "SP5"}, {"SP500", "SP5"}, {"나스닥100", "NQ1"},
	{"NASDAQ100", "NQ1"}, {"NASDAQ", "NQ"}, {"코스닥150", "Q15"},
	{"KOSDAQ150", "Q15"}, {"코스피200", "K20"}, {"KOSPI200", "K20"},
	{"200", "200"}, {"150", "150"}, {"30년", "30Y"}, {"10년", "10Y"},
	{"미국채", "UST"}, {"국채", "KTB"}, {"회사채", "CRD"}, {"배당", "DIV"},
	{"커버드콜", "COV"}, {"반도체", "SEM"}, {"2차전지", "BAT"},
	{"배터리", "BAT"}, {"바이오", "BIO"}, {"은행", "BNK"}, {"금융", "FIN"},
	{"자동차", "CAR"}, {"로봇", "BOT"}, {"원자력", "NUK"}, {"AI", "AI"},
	{"인도", "IND"}, {"중국", "CHN"}, {"일본", "JPN"}, {"미국", "US"},
	{"글로벌", "GLB"}, {"금", "GLD"}, {"달러", "USD"}, {"레버리지", "LEV"},
	{"인버스", "INV"}, {"액티브", "ACT"}, {"TOP10", "T10"},
	{"TOP30", "T30"}, {"TOP3", "TP3"},
}

type aliasCandidate struct {
	value  string
	source string
}

type aliasRecord struct {
	key        string
	instrument instrumentrole.Instrument
	candidates []aliasCandidate
}

func instrumentsWithAssignedAliases(fetched []instrumentrole.Instrument, existing []instrumentrole.Instrument) ([]instrumentrole.Instrument, error) {
	if len(fetched) == 0 {
		return nil, nil
	}
	universe := mergeAliasUniverse(existing, fetched)
	aliases, err := assignAliases(universe)
	if err != nil {
		return nil, err
	}
	out := make([]instrumentrole.Instrument, 0, len(fetched))
	fetchedKeys := make(map[string]bool, len(fetched))
	for _, item := range fetched {
		key := aliasInstrumentKey(item)
		fetchedKeys[key] = true
		alias := aliases[key]
		if alias == "" {
			return nil, oops.In("instrument_service").With("symbol", aliasSymbol(item), "security_type", item.SecurityType).New("instrument alias was not assigned")
		}
		out = append(out, withAliasExtension(item, alias))
	}
	for _, item := range existing {
		key := aliasInstrumentKey(item)
		if key == "" || fetchedKeys[key] {
			continue
		}
		alias := aliases[key]
		if alias == "" {
			return nil, oops.In("instrument_service").With("symbol", aliasSymbol(item), "security_type", item.SecurityType).New("instrument alias was not assigned")
		}
		if item.Extensions[instrumentAliasExtensionKey] == alias {
			continue
		}
		out = append(out, withAliasExtension(item, alias))
	}
	return out, nil
}

func mergeAliasUniverse(existing []instrumentrole.Instrument, fetched []instrumentrole.Instrument) []instrumentrole.Instrument {
	byKey := make(map[string]instrumentrole.Instrument, len(existing)+len(fetched))
	for _, item := range existing {
		key := aliasInstrumentKey(item)
		if key != "" {
			byKey[key] = item
		}
	}
	for _, item := range fetched {
		key := aliasInstrumentKey(item)
		if key != "" {
			byKey[key] = item
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]instrumentrole.Instrument, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
}

func assignAliases(instruments []instrumentrole.Instrument) (map[string]string, error) {
	records := make([]aliasRecord, 0, len(instruments))
	for _, item := range instruments {
		key := aliasInstrumentKey(item)
		if key == "" {
			continue
		}
		records = append(records, aliasRecord{
			key:        key,
			instrument: item,
			candidates: generateAliasCandidates(item),
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		if len(records[i].candidates) != len(records[j].candidates) {
			return len(records[i].candidates) < len(records[j].candidates)
		}
		return records[i].key < records[j].key
	})

	used := make(map[string]bool, len(reservedAliases)+len(records))
	for alias := range reservedAliases {
		used[alias] = true
	}
	assigned := make(map[string]string, len(records))
	for _, record := range records {
		picked := ""
		for _, candidate := range record.candidates {
			if !used[candidate.value] {
				picked = candidate.value
				break
			}
		}
		if picked == "" {
			var err error
			picked, err = fallbackAlias(record.instrument, used)
			if err != nil {
				return nil, err
			}
		}
		used[picked] = true
		assigned[record.key] = picked
	}
	return assigned, nil
}

func generateAliasCandidates(item instrumentrole.Instrument) []aliasCandidate {
	options := aliasOptions{allowDigits: aliasAllowsDigits(item.SecurityType)}
	candidates := make([]aliasCandidate, 0)
	seen := make(map[string]bool)
	if item.SecurityType == provider.SecurityTypeETF {
		pushETFAliasCandidates(&candidates, seen, item, options)
	}

	sourceName := firstNonEmptyString(item.Extensions["issueEnglishName"], item.Name, aliasSymbol(item), item.ISIN)
	tokens := aliasTokens(sourceName)
	joined := strings.Join(tokens, "")
	acronym := aliasAcronym(tokens)
	consonants := aliasConsonants(joined)
	words := aliasWords(tokens)

	pushAliasCandidate(&candidates, seen, acronym, "acronym", options)
	if len(words) > 0 && len(words[0]) >= aliasMaxLength {
		pushAliasCandidate(&candidates, seen, leadingAlias(words[0], aliasMaxLength), "primary_word_prefix", options)
	}
	pushAliasCandidate(&candidates, seen, padAliasFromWords(acronym, words), "acronym_words", options)
	pushAliasCandidate(&candidates, seen, leadingAlias(joined, aliasMaxLength), "prefix", options)
	pushAliasCandidate(&candidates, seen, leadingAlias(consonants, aliasMaxLength), "consonants", options)
	pushAliasCandidate(&candidates, seen, edgeAlias(joined), "edges", options)
	for _, word := range words {
		pushAliasCandidate(&candidates, seen, leadingAlias(word, aliasMaxLength), "word_prefix", options)
		pushAliasCandidate(&candidates, seen, leadingAlias(aliasConsonants(word), aliasMaxLength), "word_consonants", options)
	}
	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words); j++ {
			pushAliasCandidate(&candidates, seen, leadingAlias(words[i], 2)+leadingAlias(words[j], 2), "word_pair", options)
			pushAliasCandidate(&candidates, seen, leadingAlias(words[i], 1)+leadingAlias(words[j], 3), "word_initial_pair", options)
		}
	}
	for salt := 0; salt < 32; salt++ {
		pushAliasCandidate(&candidates, seen, aliasFallbackCode(item, salt), "hash_seed", options)
	}
	return candidates
}

type aliasOptions struct {
	allowDigits bool
}

func pushETFAliasCandidates(candidates *[]aliasCandidate, seen map[string]bool, item instrumentrole.Instrument, options aliasOptions) {
	source := strings.TrimSpace(item.Name + " " + firstNonEmptyString(item.Extensions["idxIndNm"], item.Extensions["bssIdxIdxNm"]))
	brand := detectETFBrand(item.Name)
	codes := extractETFCodes(source)
	numbers := extractAliasNumbers(source)
	ascii := aliasClean(source, true)
	prefersNamedIndex := strings.Contains(strings.ToUpper(strings.ReplaceAll(source, " ", "")), "S&P500") ||
		strings.Contains(strings.ToUpper(strings.ReplaceAll(source, " ", "")), "SP500") ||
		strings.Contains(strings.ToUpper(source), "NASDAQ") ||
		strings.Contains(strings.ToUpper(source), "KOSDAQ") ||
		strings.Contains(strings.ToUpper(source), "KOSPI") ||
		strings.Contains(source, "나스닥") ||
		strings.Contains(source, "코스닥") ||
		strings.Contains(source, "코스피")

	if prefersNamedIndex {
		for _, code := range codes {
			pushAliasCandidate(candidates, seen, brand+code, "etf_brand_keyword", options)
		}
		for _, number := range numbers {
			pushAliasCandidate(candidates, seen, brand+number, "etf_brand_number", options)
		}
	} else {
		for _, number := range numbers {
			pushAliasCandidate(candidates, seen, brand+number, "etf_brand_number", options)
		}
		for _, code := range codes {
			pushAliasCandidate(candidates, seen, brand+code, "etf_brand_keyword", options)
		}
	}
	symbol := aliasSymbol(item)
	pushAliasCandidate(candidates, seen, brand+lastN(aliasClean(symbol, true), 3), "etf_brand_symbol", options)
	for _, code := range codes {
		pushAliasCandidate(candidates, seen, code, "etf_keyword", options)
	}
	pushAliasCandidate(candidates, seen, leadingAlias(ascii, aliasMaxLength), "etf_ascii_prefix", options)
}

func pushAliasCandidate(candidates *[]aliasCandidate, seen map[string]bool, raw string, source string, options aliasOptions) {
	value := aliasClean(raw, options.allowDigits)
	if len(value) < aliasMinLength {
		return
	}
	value = leadingAlias(value, aliasMaxLength)
	if len(value) < aliasMinLength || seen[value] || reservedAliases[value] {
		return
	}
	seen[value] = true
	*candidates = append(*candidates, aliasCandidate{value: value, source: source})
}

func fallbackAlias(item instrumentrole.Instrument, used map[string]bool) (string, error) {
	for salt := 32; salt < 100000; salt++ {
		value := aliasFallbackCode(item, salt)
		if !used[value] && !reservedAliases[value] {
			return value, nil
		}
	}
	return "", oops.In("instrument_service").With("symbol", aliasSymbol(item), "security_type", item.SecurityType).New("could not assign unique instrument alias")
}

func aliasFallbackCode(item instrumentrole.Instrument, salt int) string {
	allowDigits := aliasAllowsDigits(item.SecurityType)
	source := firstNonEmptyString(item.Extensions["issueEnglishName"], item.Name, aliasSymbol(item), item.ISIN)
	prefixSource := aliasCleanLetters(source)
	prefixLength := 1
	if len(prefixSource) >= 2 {
		prefixLength = 2
	}
	if len(prefixSource) == 0 {
		prefixSource = "K"
	}
	prefix := leadingAlias(prefixSource, prefixLength)
	seed := strings.Join([]string{aliasSymbol(item), item.ISIN, source, string(item.SecurityType), strconv.Itoa(salt)}, ":")
	suffixLength := aliasMaxLength - len(prefix)
	if suffixLength < 1 {
		suffixLength = 1
	}
	if allowDigits {
		return leadingAlias(prefix+baseN(hashString(seed), "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", suffixLength), aliasMaxLength)
	}
	return leadingAlias(prefix+baseN(hashString(seed), "ABCDEFGHIJKLMNOPQRSTUVWXYZ", suffixLength), aliasMaxLength)
}

func aliasTokens(value string) []string {
	spaced := splitAliasCamel(value)
	raw := strings.FieldsFunc(strings.ToUpper(norm.NFKD.String(spaced)), func(r rune) bool {
		return !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9')
	})
	important := make([]string, 0, len(raw))
	for _, token := range raw {
		if token != "" && !aliasStopWords[token] {
			important = append(important, token)
		}
	}
	if len(important) > 0 {
		return important
	}
	return raw
}

func splitAliasCamel(value string) string {
	var out strings.Builder
	var prev rune
	for index, r := range value {
		if index > 0 && unicode.IsLower(prev) && unicode.IsUpper(r) {
			out.WriteByte(' ')
		}
		if r == '&' {
			out.WriteString(" AND ")
		} else {
			out.WriteRune(r)
		}
		prev = r
	}
	return out.String()
}

func aliasAcronym(tokens []string) string {
	var out strings.Builder
	for _, token := range tokens {
		if token != "" {
			out.WriteByte(token[0])
		}
	}
	return out.String()
}

func aliasWords(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		for _, r := range token {
			if r >= 'A' && r <= 'Z' {
				out = append(out, token)
				break
			}
		}
	}
	return out
}

func padAliasFromWords(acronym string, words []string) string {
	out := acronym
	for _, word := range words {
		if len(word) > 1 {
			out += word[1:]
		}
		if len(out) >= aliasMaxLength {
			break
		}
	}
	return out
}

func aliasConsonants(value string) string {
	normalized := aliasClean(value, true)
	if len(normalized) <= 1 {
		return normalized
	}
	return normalized[:1] + strings.Map(func(r rune) rune {
		switch r {
		case 'A', 'E', 'I', 'O', 'U':
			return -1
		default:
			return r
		}
	}, normalized[1:])
}

func edgeAlias(value string) string {
	normalized := aliasClean(value, true)
	if len(normalized) <= aliasMaxLength {
		return normalized
	}
	return normalized[:2] + normalized[len(normalized)-2:]
}

func detectETFBrand(name string) string {
	upper := strings.ToUpper(name)
	for _, brand := range etfBrands {
		if strings.Contains(upper, strings.ToUpper(brand.needle)) {
			return brand.code
		}
	}
	ascii := aliasClean(upper, true)
	if ascii != "" {
		return ascii[:1]
	}
	return "E"
}

func extractETFCodes(value string) []string {
	upper := strings.ToUpper(strings.ReplaceAll(value, " ", ""))
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, keyword := range etfKeywords {
		needle := strings.ToUpper(strings.ReplaceAll(keyword.needle, " ", ""))
		if strings.Contains(upper, needle) && !seen[keyword.code] {
			seen[keyword.code] = true
			out = append(out, keyword.code)
		}
	}
	return out
}

func extractAliasNumbers(value string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		number := current.String()
		current.Reset()
		if len(number) > 3 {
			return
		}
		for len(number) < 2 {
			number = "0" + number
		}
		if !seen[number] {
			seen[number] = true
			out = append(out, number)
		}
	}
	for _, r := range value {
		if r >= '0' && r <= '9' {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func aliasInstrumentKey(item instrumentrole.Instrument) string {
	symbol := aliasSymbol(item)
	if symbol == "" || item.SecurityType == "" {
		return ""
	}
	return strings.Join([]string{string(withDefaultMarket(item.Market)), string(item.SecurityType), symbol}, "\x00")
}

func aliasSymbol(item instrumentrole.Instrument) string {
	return firstNonEmptyString(item.SecurityCode, item.ExchangeCode)
}

func withAliasExtension(item instrumentrole.Instrument, alias string) instrumentrole.Instrument {
	extensions := make(map[string]string, len(item.Extensions)+1)
	for key, value := range item.Extensions {
		extensions[key] = value
	}
	extensions[instrumentAliasExtensionKey] = alias
	item.Extensions = extensions
	return item
}

func aliasAllowsDigits(securityType provider.SecurityType) bool {
	return securityType != provider.SecurityTypeStock
}

func aliasClean(value string, allowDigits bool) string {
	var out strings.Builder
	for _, r := range strings.ToUpper(value) {
		if r >= 'A' && r <= 'Z' {
			out.WriteRune(r)
			continue
		}
		if allowDigits && r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func aliasCleanLetters(value string) string {
	return aliasClean(value, false)
}

func leadingAlias(value string, size int) string {
	if len(value) <= size {
		return value
	}
	return value[:size]
}

func lastN(value string, size int) string {
	if len(value) <= size {
		return value
	}
	return value[len(value)-size:]
}

func hashString(value string) uint32 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(value))
	return hash.Sum32()
}

func baseN(value uint32, alphabet string, length int) string {
	if length <= 0 {
		return ""
	}
	base := uint32(len(alphabet))
	current := value
	var out strings.Builder
	for i := 0; i < length; i++ {
		out.WriteByte(alphabet[current%base])
		current = current / base
	}
	return out.String()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
