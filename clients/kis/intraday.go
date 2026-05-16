package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// IntradayBar is a KIS same-day minute bar.
type IntradayBar struct {
	Date    string
	Time    string
	Current string
	Open    string
	High    string
	Low     string
	Volume  string
	Amount  string
	Raw     IntradayBarOutput
}

// IntradaySummary is the provider-native output1 object from Intraday.
type IntradaySummary struct {
	Current            string `json:"stck_prpr"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChangeRate string `json:"prdy_ctrt"`
	PreviousClose      string `json:"stck_prdy_clpr"`
	Volume             string `json:"acml_vol"`
	Amount             string `json:"acml_tr_pbmn"`
	Name               string `json:"hts_kor_isnm"`
}

// IntradayBarOutput is a provider-native output2 row from Intraday.
type IntradayBarOutput struct {
	Date    string `json:"stck_bsop_date"`
	Time    string `json:"stck_cntg_hour"`
	Current string `json:"stck_prpr"`
	Open    string `json:"stck_oprc"`
	High    string `json:"stck_hgpr"`
	Low     string `json:"stck_lwpr"`
	Volume  string `json:"cntg_vol"`
	Amount  string `json:"acml_tr_pbmn"`
}

// IntradayOption configures an Intraday request.
type IntradayOption func(*intradayQuery) error

type intradayQuery struct {
	market          string
	inputHour       string
	includePastData string
	etcCode         string
}

func defaultIntradayQuery() intradayQuery {
	return intradayQuery{
		market:          "J",
		inputHour:       "153000",
		includePastData: "Y",
	}
}

// WithInputHour sets the KIS FID_INPUT_HOUR_1 value in HHMMSS form.
func WithInputHour(inputHour string) IntradayOption {
	return func(q *intradayQuery) error {
		q.inputHour = strings.TrimSpace(inputHour)
		return nil
	}
}

// WithPastData controls FID_PW_DATA_INCU_YN for Intraday.
func WithPastData(include bool) IntradayOption {
	return func(q *intradayQuery) error {
		if include {
			q.includePastData = "Y"
			return nil
		}
		q.includePastData = "N"
		return nil
	}
}

// Intraday fetches same-day minute bars for a domestic stock symbol.
//
// The request uses TR ID FHKST03010200. Symbol should be a KIS short stock code.
func (c *Client) Intraday(ctx context.Context, symbol string, options ...IntradayOption) ([]IntradayBar, error) {
	query := defaultIntradayQuery()
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationIntraday,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice",
		"tr_id", trIDDomesticStockIntraday,
		"symbol", strings.TrimSpace(symbol),
	)
	for _, option := range options {
		if option == nil {
			return nil, errb.New("kis intraday request: option is required")
		}
		if err := option(&query); err != nil {
			return nil, errb.Wrapf(err, "apply kis intraday option")
		}
	}
	if err := query.validate(symbol); err != nil {
		return nil, errb.Wrap(err)
	}

	var envelope intradayEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationIntraday, trIDDomesticStockIntraday, errb)
	if err != nil {
		return nil, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": query.market,
		"FID_INPUT_ISCD":         strings.TrimSpace(symbol),
		"FID_INPUT_HOUR_1":       query.inputHour,
		"FID_PW_DATA_INCU_YN":    query.includePastData,
		"FID_ETC_CLS_CODE":       query.etcCode,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice")
	if err != nil {
		return nil, errb.Wrapf(err, "request kis intraday bars")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationIntraday, trIDDomesticStockIntraday); err != nil {
		return nil, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationIntraday, trIDDomesticStockIntraday); err != nil {
		return nil, err
	}
	return intradayBarsFromOutput(envelope.Output2), nil
}

func (q intradayQuery) validate(symbol string) error {
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationIntraday,
	)
	if strings.TrimSpace(symbol) == "" {
		return errb.New("kis intraday request: symbol is required")
	}
	if q.market == "" {
		return errb.New("kis intraday request: market is required")
	}
	if q.inputHour == "" {
		return errb.New("kis intraday request: input hour is required")
	}
	if q.includePastData != "Y" && q.includePastData != "N" {
		return errb.With("include_past_data", q.includePastData).Errorf("kis intraday request: unsupported past data flag=%s", q.includePastData)
	}
	return nil
}

func intradayBarsFromOutput(outputs []IntradayBarOutput) []IntradayBar {
	bars := make([]IntradayBar, 0, len(outputs))
	for _, output := range outputs {
		bars = append(bars, IntradayBar{
			Date:    output.Date,
			Time:    output.Time,
			Current: output.Current,
			Open:    output.Open,
			High:    output.High,
			Low:     output.Low,
			Volume:  output.Volume,
			Amount:  output.Amount,
			Raw:     output,
		})
	}
	return bars
}

type intradayEnvelope struct {
	responseFields
	Output1 IntradaySummary     `json:"output1"`
	Output2 []IntradayBarOutput `json:"output2"`
}
