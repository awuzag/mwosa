package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// Bar is a simplified period bar returned by Client.Daily.
//
// Numeric values are kept as KIS-provided strings.
type Bar struct {
	Date               string
	Close              string
	Open               string
	High               string
	Low                string
	Volume             string
	Amount             string
	PreviousChange     string
	PreviousChangeSign string
	Raw                BarOutput
}

// DailySummary is the provider-native output1 object from the KIS daily endpoint.
type DailySummary struct {
	Symbol             string `json:"stck_shrn_iscd"`
	Name               string `json:"hts_kor_isnm"`
	Current            string `json:"stck_prpr"`
	PreviousClose      string `json:"stck_prdy_clpr"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChangeRate string `json:"prdy_ctrt"`
	Volume             string `json:"acml_vol"`
	Amount             string `json:"acml_tr_pbmn"`
	Open               string `json:"stck_oprc"`
	High               string `json:"stck_hgpr"`
	Low                string `json:"stck_lwpr"`
}

// BarOutput is a provider-native output2 row from the KIS daily endpoint.
type BarOutput struct {
	Date               string `json:"stck_bsop_date"`
	Close              string `json:"stck_clpr"`
	Open               string `json:"stck_oprc"`
	High               string `json:"stck_hgpr"`
	Low                string `json:"stck_lwpr"`
	Volume             string `json:"acml_vol"`
	Amount             string `json:"acml_tr_pbmn"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChange     string `json:"prdy_vrss"`
	LockCode           string `json:"flng_cls_code"`
	SplitRate          string `json:"prtt_rate"`
	Modified           string `json:"mod_yn"`
	RevaluationReason  string `json:"revl_issu_reas"`
}

// DailyOption configures a Daily request.
type DailyOption func(*dailyQuery) error

type dailyQuery struct {
	market            string
	period            string
	startDate         string
	endDate           string
	adjustedPriceCode string
}

func defaultDailyQuery() dailyQuery {
	return dailyQuery{
		market:            "J",
		period:            "D",
		adjustedPriceCode: "0",
	}
}

// WithPeriod sets the period division code for Daily.
//
// Supported values are "D" for day, "W" for week, "M" for month, and "Y" for
// year.
func WithPeriod(period string) DailyOption {
	return func(q *dailyQuery) error {
		q.period = strings.TrimSpace(period)
		return nil
	}
}

// WithDateRange sets the inclusive date range for Daily.
//
// Dates must use the KIS YYYYMMDD format, such as "20250101".
func WithDateRange(startDate string, endDate string) DailyOption {
	return func(q *dailyQuery) error {
		q.startDate = strings.TrimSpace(startDate)
		q.endDate = strings.TrimSpace(endDate)
		return nil
	}
}

// WithOriginalPrice requests original prices instead of adjusted prices.
//
// By default Daily sends FID_ORG_ADJ_PRC=0, which asks KIS for adjusted prices.
// This option sends FID_ORG_ADJ_PRC=1.
func WithOriginalPrice() DailyOption {
	return func(q *dailyQuery) error {
		q.adjustedPriceCode = "1"
		return nil
	}
}

// Daily fetches domestic stock period bars for symbol.
//
// The request uses TR ID FHKST03010100. Use WithDateRange to set the required
// start and end dates in YYYYMMDD form, and WithPeriod to choose D, W, M, or Y.
func (c *Client) Daily(ctx context.Context, symbol string, options ...DailyOption) ([]Bar, error) {
	query := defaultDailyQuery()
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationDaily,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
		"tr_id", trIDDomesticStockDailyItemChart,
		"symbol", strings.TrimSpace(symbol),
	)
	for _, option := range options {
		if option == nil {
			return nil, errb.New("kis daily request: option is required")
		}
		if err := option(&query); err != nil {
			return nil, errb.Wrapf(err, "apply kis daily option")
		}
	}
	if err := query.validate(symbol); err != nil {
		return nil, errb.Wrap(err)
	}

	var envelope dailyEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationDaily, trIDDomesticStockDailyItemChart, errb)
	if err != nil {
		return nil, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": query.market,
		"FID_INPUT_ISCD":         strings.TrimSpace(symbol),
		"FID_INPUT_DATE_1":       query.startDate,
		"FID_INPUT_DATE_2":       query.endDate,
		"FID_PERIOD_DIV_CODE":    query.period,
		"FID_ORG_ADJ_PRC":        query.adjustedPriceCode,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice")
	if err != nil {
		return nil, errb.Wrapf(err, "request kis daily bars")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationDaily, trIDDomesticStockDailyItemChart); err != nil {
		return nil, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationDaily, trIDDomesticStockDailyItemChart); err != nil {
		return nil, err
	}
	return barsFromOutput(envelope.Output2), nil
}

func (q dailyQuery) validate(symbol string) error {
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationDaily,
	)
	if strings.TrimSpace(symbol) == "" {
		return errb.New("kis daily request: symbol is required")
	}
	if q.market == "" {
		return errb.New("kis daily request: market is required")
	}
	switch q.period {
	case "D", "W", "M", "Y":
	default:
		return errb.With("period", q.period).Errorf("kis daily request: unsupported period=%s", q.period)
	}
	if q.startDate == "" {
		return errb.New("kis daily request: start date is required")
	}
	if q.endDate == "" {
		return errb.New("kis daily request: end date is required")
	}
	switch q.adjustedPriceCode {
	case "0", "1":
	default:
		return errb.With("adjusted_price_code", q.adjustedPriceCode).Errorf("kis daily request: unsupported adjusted price code=%s", q.adjustedPriceCode)
	}
	return nil
}

func barsFromOutput(outputs []BarOutput) []Bar {
	bars := make([]Bar, 0, len(outputs))
	for _, output := range outputs {
		bars = append(bars, Bar{
			Date:               output.Date,
			Close:              output.Close,
			Open:               output.Open,
			High:               output.High,
			Low:                output.Low,
			Volume:             output.Volume,
			Amount:             output.Amount,
			PreviousChange:     output.PreviousChange,
			PreviousChangeSign: output.PreviousChangeSign,
			Raw:                output,
		})
	}
	return bars
}

type dailyEnvelope struct {
	responseFields
	Output1 DailySummary `json:"output1"`
	Output2 []BarOutput  `json:"output2"`
}
