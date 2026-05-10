package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// Price is a simplified current quote returned by Client.Price.
//
// Numeric values are kept as strings because KIS returns decimal-like market
// fields as strings and callers may choose their own numeric representation.
type Price struct {
	// Symbol is the KIS short stock code.
	Symbol             string
	Current            string
	Open               string
	High               string
	Low                string
	PreviousClose      string
	PreviousChange     string
	PreviousChangeSign string
	PreviousChangeRate string
	Volume             string
	Amount             string
	MarketCap          string
	PER                string
	PBR                string
	Raw                PriceOutput
}

// PriceOutput is the provider-native output object from the KIS price endpoint.
type PriceOutput struct {
	StockStatusCode    string `json:"iscd_stat_cls_code"`
	MarketName         string `json:"rprs_mrkt_kor_name"`
	SectorName         string `json:"bstp_kor_isnm"`
	Current            string `json:"stck_prpr"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChangeRate string `json:"prdy_ctrt"`
	Amount             string `json:"acml_tr_pbmn"`
	Volume             string `json:"acml_vol"`
	Open               string `json:"stck_oprc"`
	High               string `json:"stck_hgpr"`
	Low                string `json:"stck_lwpr"`
	PreviousClose      string `json:"stck_sdpr"`
	Symbol             string `json:"stck_shrn_iscd"`
	MarketCap          string `json:"hts_avls"`
	PER                string `json:"per"`
	PBR                string `json:"pbr"`
}

// Price fetches the current domestic stock quote for symbol.
//
// The request uses TR ID FHKST01010100 and the KRX market division code J.
// Symbol should be the KIS short stock code, such as "005930".
func (c *Client) Price(ctx context.Context, symbol string) (Price, error) {
	symbol = strings.TrimSpace(symbol)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationPrice,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-price",
		"tr_id", trIDDomesticStockPrice,
		"symbol", symbol,
	)
	if symbol == "" {
		return Price{}, errb.New("kis price request: symbol is required")
	}

	var envelope priceEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationPrice, trIDDomesticStockPrice, errb)
	if err != nil {
		return Price{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         symbol,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-price")
	if err != nil {
		return Price{}, errb.Wrapf(err, "request kis price")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationPrice, trIDDomesticStockPrice); err != nil {
		return Price{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationPrice, trIDDomesticStockPrice); err != nil {
		return Price{}, err
	}
	return priceFromOutput(envelope.Output), nil
}

func priceFromOutput(output PriceOutput) Price {
	return Price{
		Symbol:             output.Symbol,
		Current:            output.Current,
		Open:               output.Open,
		High:               output.High,
		Low:                output.Low,
		PreviousClose:      output.PreviousClose,
		PreviousChange:     output.PreviousChange,
		PreviousChangeSign: output.PreviousChangeSign,
		PreviousChangeRate: output.PreviousChangeRate,
		Volume:             output.Volume,
		Amount:             output.Amount,
		MarketCap:          output.MarketCap,
		PER:                output.PER,
		PBR:                output.PBR,
		Raw:                output,
	}
}

type priceEnvelope struct {
	responseFields
	Output PriceOutput `json:"output"`
}
