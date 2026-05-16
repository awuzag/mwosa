package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// Trade is a domestic stock execution row.
type Trade struct {
	Time               string
	Current            string
	PreviousChange     string
	PreviousChangeSign string
	PreviousChangeRate string
	Volume             string
	Strength           string
	Raw                TradeOutput
}

// TimedTrade is a time-filtered execution row returned by Client.TimeTrades.
type TimedTrade struct {
	Time               string
	Current            string
	Ask                string
	Bid                string
	Volume             string
	AccumulatedVolume  string
	PreviousChange     string
	PreviousChangeSign string
	PreviousChangeRate string
	Strength           string
	Raw                TimedTradeOutput
}

// TradeOutput is a provider-native output row from Trades.
type TradeOutput struct {
	Time               string `json:"stck_cntg_hour"`
	Current            string `json:"stck_prpr"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	Volume             string `json:"cntg_vol"`
	Strength           string `json:"tday_rltv"`
	PreviousChangeRate string `json:"prdy_ctrt"`
}

// TimeTradesSummary is the provider-native output1 object from TimeTrades.
type TimeTradesSummary struct {
	Current            string `json:"stck_prpr"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChangeRate string `json:"prdy_ctrt"`
	Volume             string `json:"acml_vol"`
	PreviousVolume     string `json:"prdy_vol"`
	MarketName         string `json:"rprs_mrkt_kor_name"`
}

// TimedTradeOutput is a provider-native output2 row from TimeTrades.
type TimedTradeOutput struct {
	Time               string `json:"stck_cntg_hour"`
	Current            string `json:"stck_prpr"`
	Ask                string `json:"askp"`
	Bid                string `json:"bidp"`
	Volume             string `json:"cnqn"`
	AccumulatedVolume  string `json:"acml_vol"`
	PreviousChange     string `json:"prdy_vrss"`
	PreviousChangeSign string `json:"prdy_vrss_sign"`
	PreviousChangeRate string `json:"prdy_ctrt"`
	Strength           string `json:"tday_rltv"`
}

// Trades fetches recent executions for a domestic stock symbol.
//
// The request uses TR ID FHKST01010300.
func (c *Client) Trades(ctx context.Context, symbol string) ([]Trade, error) {
	symbol = strings.TrimSpace(symbol)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationTrades,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-ccnl",
		"tr_id", trIDDomesticStockTrades,
		"symbol", symbol,
	)
	if symbol == "" {
		return nil, errb.New("kis trades request: symbol is required")
	}

	var envelope tradesEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationTrades, trIDDomesticStockTrades, errb)
	if err != nil {
		return nil, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         symbol,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-ccnl")
	if err != nil {
		return nil, errb.Wrapf(err, "request kis trades")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationTrades, trIDDomesticStockTrades); err != nil {
		return nil, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationTrades, trIDDomesticStockTrades); err != nil {
		return nil, err
	}
	return tradesFromOutput(envelope.Output), nil
}

// TimeTrades fetches time-filtered same-day executions for a domestic stock symbol.
//
// The request uses TR ID FHPST01060000. inputHour must use HHMMSS form.
func (c *Client) TimeTrades(ctx context.Context, symbol string, inputHour string) ([]TimedTrade, error) {
	symbol = strings.TrimSpace(symbol)
	inputHour = strings.TrimSpace(inputHour)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationTimeTrades,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion",
		"tr_id", trIDDomesticStockTimeTrades,
		"symbol", symbol,
	)
	if symbol == "" {
		return nil, errb.New("kis time trades request: symbol is required")
	}
	if inputHour == "" {
		return nil, errb.New("kis time trades request: input hour is required")
	}

	var envelope timeTradesEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationTimeTrades, trIDDomesticStockTimeTrades, errb)
	if err != nil {
		return nil, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         symbol,
		"FID_INPUT_HOUR_1":       inputHour,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion")
	if err != nil {
		return nil, errb.Wrapf(err, "request kis time trades")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationTimeTrades, trIDDomesticStockTimeTrades); err != nil {
		return nil, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationTimeTrades, trIDDomesticStockTimeTrades); err != nil {
		return nil, err
	}
	return timedTradesFromOutput(envelope.Output2), nil
}

func tradesFromOutput(outputs []TradeOutput) []Trade {
	trades := make([]Trade, 0, len(outputs))
	for _, output := range outputs {
		trades = append(trades, Trade{
			Time:               output.Time,
			Current:            output.Current,
			PreviousChange:     output.PreviousChange,
			PreviousChangeSign: output.PreviousChangeSign,
			PreviousChangeRate: output.PreviousChangeRate,
			Volume:             output.Volume,
			Strength:           output.Strength,
			Raw:                output,
		})
	}
	return trades
}

func timedTradesFromOutput(outputs []TimedTradeOutput) []TimedTrade {
	trades := make([]TimedTrade, 0, len(outputs))
	for _, output := range outputs {
		trades = append(trades, TimedTrade{
			Time:               output.Time,
			Current:            output.Current,
			Ask:                output.Ask,
			Bid:                output.Bid,
			Volume:             output.Volume,
			AccumulatedVolume:  output.AccumulatedVolume,
			PreviousChange:     output.PreviousChange,
			PreviousChangeSign: output.PreviousChangeSign,
			PreviousChangeRate: output.PreviousChangeRate,
			Strength:           output.Strength,
			Raw:                output,
		})
	}
	return trades
}

type tradesEnvelope struct {
	responseFields
	Output []TradeOutput `json:"output"`
}

type timeTradesEnvelope struct {
	responseFields
	Output1 TimeTradesSummary  `json:"output1"`
	Output2 []TimedTradeOutput `json:"output2"`
}
