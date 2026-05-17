package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/oops"
)

// ETFComponentStockPrice is one provider-native ETF component price row.
type ETFComponentStockPrice struct {
	Symbol             string            `json:"symbol"`
	Name               string            `json:"name"`
	Current            string            `json:"current_price,omitempty"`
	PreviousChange     string            `json:"previous_change,omitempty"`
	PreviousChangeSign string            `json:"previous_change_sign,omitempty"`
	PreviousChangeRate string            `json:"previous_change_rate,omitempty"`
	Volume             string            `json:"volume,omitempty"`
	Weight             string            `json:"weight,omitempty"`
	ValuationAmount    string            `json:"valuation_amount,omitempty"`
	Quantity           string            `json:"quantity,omitempty"`
	Raw                map[string]string `json:"raw,omitempty"`
}

// ETFComponentStockPriceResult contains component rows for one ETF symbol.
type ETFComponentStockPriceResult struct {
	Symbol     string                   `json:"symbol"`
	Rows       []ETFComponentStockPrice `json:"rows"`
	Output1    map[string]string        `json:"output1,omitempty"`
	RawPayload map[string]any           `json:"raw_payload,omitempty"`
}

// ETFComponentStockPrices fetches KIS ETF component stock price rows for symbol.
//
// The request uses TR ID FHKST121600C0. Symbol should be the KRX short code.
func (c *Client) ETFComponentStockPrices(ctx context.Context, symbol string) (ETFComponentStockPriceResult, error) {
	symbol = strings.TrimSpace(symbol)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationETFComponentStockPrice,
		"endpoint", "/uapi/etfetn/v1/quotations/inquire-component-stock-price",
		"tr_id", trIDETFComponentStockPrice,
		"symbol", symbol,
	)
	if symbol == "" {
		return ETFComponentStockPriceResult{}, errb.New("kis ETF component stock price request: symbol is required")
	}

	var envelope etfComponentStockPriceEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationETFComponentStockPrice, trIDETFComponentStockPrice, errb)
	if err != nil {
		return ETFComponentStockPriceResult{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"fid_cond_mrkt_div_code": "J",
		"fid_input_iscd":         symbol,
		"fid_cond_scr_div_code":  "11216",
	}).
		SetResult(&envelope).
		Get("/uapi/etfetn/v1/quotations/inquire-component-stock-price")
	if err != nil {
		return ETFComponentStockPriceResult{}, errb.Wrapf(err, "request kis ETF component stock prices")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationETFComponentStockPrice, trIDETFComponentStockPrice); err != nil {
		return ETFComponentStockPriceResult{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationETFComponentStockPrice, trIDETFComponentStockPrice); err != nil {
		return ETFComponentStockPriceResult{}, err
	}
	result, err := etfComponentStockPriceResultFromEnvelope(symbol, envelope)
	if err != nil {
		return ETFComponentStockPriceResult{}, errb.Wrapf(err, "decode kis ETF component stock price output")
	}
	return result, nil
}

type etfComponentStockPriceEnvelope struct {
	responseFields
	Output     json.RawMessage `json:"output"`
	Output1    json.RawMessage `json:"output1"`
	Output2    json.RawMessage `json:"output2"`
	RawPayload map[string]any  `json:"-"`
}

func (e *etfComponentStockPriceEnvelope) UnmarshalJSON(data []byte) error {
	type alias etfComponentStockPriceEnvelope
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*e = etfComponentStockPriceEnvelope(decoded)
	e.RawPayload = raw
	return nil
}

func etfComponentStockPriceResultFromEnvelope(symbol string, envelope etfComponentStockPriceEnvelope) (ETFComponentStockPriceResult, error) {
	rows, err := decodeETFComponentRowsFromCandidates(envelope.Output2, envelope.Output1, envelope.Output)
	if err != nil {
		return ETFComponentStockPriceResult{}, err
	}
	output1, err := decodeStringObject(envelope.Output1)
	if err != nil {
		return ETFComponentStockPriceResult{}, err
	}
	return ETFComponentStockPriceResult{
		Symbol:     symbol,
		Rows:       rows,
		Output1:    output1,
		RawPayload: envelope.RawPayload,
	}, nil
}

func decodeETFComponentRows(raw json.RawMessage) ([]ETFComponentStockPrice, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err == nil {
		return etfComponentRowsFromObjects(objects), nil
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	return etfComponentRowsFromObjects([]map[string]any{object}), nil
}

func decodeETFComponentRowsFromCandidates(values ...json.RawMessage) ([]ETFComponentStockPrice, error) {
	for _, value := range values {
		if !rawJSONArray(value) {
			continue
		}
		rows, err := decodeETFComponentRows(value)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return rows, nil
		}
	}
	for _, value := range values {
		if len(value) == 0 || string(value) == "null" {
			continue
		}
		rows, err := decodeETFComponentRows(value)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return rows, nil
		}
	}
	return nil, nil
}

func rawJSONArray(raw json.RawMessage) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return true
		default:
			return false
		}
	}
	return false
}

func etfComponentRowsFromObjects(objects []map[string]any) []ETFComponentStockPrice {
	rows := make([]ETFComponentStockPrice, 0, len(objects))
	for _, object := range objects {
		raw := stringifyObject(object)
		row := ETFComponentStockPrice{
			Symbol:             firstRaw(raw, "stck_shrn_iscd", "mksc_shrn_iscd", "pdno", "isu_cd", "isu_srt_cd"),
			Name:               firstRaw(raw, "hts_kor_isnm", "stck_kor_isnm", "prdt_name", "isu_kor_nm"),
			Current:            firstRaw(raw, "stck_prpr", "prpr"),
			PreviousChange:     firstRaw(raw, "prdy_vrss"),
			PreviousChangeSign: firstRaw(raw, "prdy_vrss_sign"),
			PreviousChangeRate: firstRaw(raw, "prdy_ctrt"),
			Volume:             firstRaw(raw, "acml_vol"),
			Weight:             firstRaw(raw, "etf_cnfg_issu_rlim", "etf_cnfg_issu_rate", "cnfg_issu_wht"),
			ValuationAmount:    firstRaw(raw, "etf_vltn_amt", "etf_cnfg_issu_avls", "cnfg_issu_evlu_amt"),
			Quantity:           firstRaw(raw, "cnfg_issu_qty", "etf_cnfg_issu_qty"),
			Raw:                raw,
		}
		if row.Symbol == "" && row.Name == "" {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func decodeStringObject(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	return stringifyObject(object), nil
}

func stringifyObject(object map[string]any) map[string]string {
	if len(object) == 0 {
		return nil
	}
	values := make(map[string]string, len(object))
	for key, value := range object {
		values[key] = stringifyJSONValue(value)
	}
	return values
}

func stringifyJSONValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func firstRaw(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstRawMessage(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 && string(value) != "null" {
			return value
		}
	}
	return nil
}
