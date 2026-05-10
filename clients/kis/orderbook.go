package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// Orderbook is the current 10-level ask/bid book plus expected conclusion data.
type Orderbook struct {
	AcceptanceTime   string
	Asks             []OrderbookLevel
	Bids             []OrderbookLevel
	TotalAskQuantity string
	TotalBidQuantity string
	Expected         ExpectedConclusion
	Raw              OrderbookOutput
}

// OrderbookLevel is one ask or bid level.
type OrderbookLevel struct {
	Price    string
	Quantity string
	Delta    string
}

// ExpectedConclusion is expected execution data returned with the orderbook.
type ExpectedConclusion struct {
	Symbol             string
	Current            string
	Open               string
	High               string
	Low                string
	PreviousClose      string
	ExpectedPrice      string
	ExpectedVolume     string
	PreviousChange     string
	PreviousChangeSign string
	PreviousChangeRate string
	Raw                ExpectedConclusionOutput
}

// Orderbook fetches the current domestic stock orderbook for symbol.
//
// The request uses TR ID FHKST01010200 and returns up to 10 ask and bid levels.
func (c *Client) Orderbook(ctx context.Context, symbol string) (Orderbook, error) {
	symbol = strings.TrimSpace(symbol)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupDomesticStockQuotation,
		"operation", OperationOrderbook,
		"endpoint", "/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn",
		"tr_id", trIDDomesticStockOrderbook,
		"symbol", symbol,
	)
	if symbol == "" {
		return Orderbook{}, errb.New("kis orderbook request: symbol is required")
	}

	var envelope orderbookEnvelope
	request, err := c.request(ctx, GroupDomesticStockQuotation, OperationOrderbook, trIDDomesticStockOrderbook, errb)
	if err != nil {
		return Orderbook{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         symbol,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn")
	if err != nil {
		return Orderbook{}, errb.Wrapf(err, "request kis orderbook")
	}
	if err := checkHTTP(response, errb, GroupDomesticStockQuotation, OperationOrderbook, trIDDomesticStockOrderbook); err != nil {
		return Orderbook{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupDomesticStockQuotation, OperationOrderbook, trIDDomesticStockOrderbook); err != nil {
		return Orderbook{}, err
	}
	return orderbookFromOutput(envelope.Output1, envelope.Output2), nil
}

func orderbookFromOutput(output OrderbookOutput, expected ExpectedConclusionOutput) Orderbook {
	return Orderbook{
		AcceptanceTime:   output.AcceptanceTime,
		Asks:             askLevels(output),
		Bids:             bidLevels(output),
		TotalAskQuantity: output.TotalAskQuantity,
		TotalBidQuantity: output.TotalBidQuantity,
		Expected: ExpectedConclusion{
			Symbol:             expected.Symbol,
			Current:            expected.Current,
			Open:               expected.Open,
			High:               expected.High,
			Low:                expected.Low,
			PreviousClose:      expected.PreviousClose,
			ExpectedPrice:      expected.ExpectedPrice,
			ExpectedVolume:     expected.ExpectedVolume,
			PreviousChange:     expected.PreviousChange,
			PreviousChangeSign: expected.PreviousChangeSign,
			PreviousChangeRate: expected.PreviousChangeRate,
			Raw:                expected,
		},
		Raw: output,
	}
}

func askLevels(output OrderbookOutput) []OrderbookLevel {
	return []OrderbookLevel{
		{Price: output.AskPrice1, Quantity: output.AskQuantity1, Delta: output.AskQuantityDelta1},
		{Price: output.AskPrice2, Quantity: output.AskQuantity2, Delta: output.AskQuantityDelta2},
		{Price: output.AskPrice3, Quantity: output.AskQuantity3, Delta: output.AskQuantityDelta3},
		{Price: output.AskPrice4, Quantity: output.AskQuantity4, Delta: output.AskQuantityDelta4},
		{Price: output.AskPrice5, Quantity: output.AskQuantity5, Delta: output.AskQuantityDelta5},
		{Price: output.AskPrice6, Quantity: output.AskQuantity6, Delta: output.AskQuantityDelta6},
		{Price: output.AskPrice7, Quantity: output.AskQuantity7, Delta: output.AskQuantityDelta7},
		{Price: output.AskPrice8, Quantity: output.AskQuantity8, Delta: output.AskQuantityDelta8},
		{Price: output.AskPrice9, Quantity: output.AskQuantity9, Delta: output.AskQuantityDelta9},
		{Price: output.AskPrice10, Quantity: output.AskQuantity10, Delta: output.AskQuantityDelta10},
	}
}

func bidLevels(output OrderbookOutput) []OrderbookLevel {
	return []OrderbookLevel{
		{Price: output.BidPrice1, Quantity: output.BidQuantity1, Delta: output.BidQuantityDelta1},
		{Price: output.BidPrice2, Quantity: output.BidQuantity2, Delta: output.BidQuantityDelta2},
		{Price: output.BidPrice3, Quantity: output.BidQuantity3, Delta: output.BidQuantityDelta3},
		{Price: output.BidPrice4, Quantity: output.BidQuantity4, Delta: output.BidQuantityDelta4},
		{Price: output.BidPrice5, Quantity: output.BidQuantity5, Delta: output.BidQuantityDelta5},
		{Price: output.BidPrice6, Quantity: output.BidQuantity6, Delta: output.BidQuantityDelta6},
		{Price: output.BidPrice7, Quantity: output.BidQuantity7, Delta: output.BidQuantityDelta7},
		{Price: output.BidPrice8, Quantity: output.BidQuantity8, Delta: output.BidQuantityDelta8},
		{Price: output.BidPrice9, Quantity: output.BidQuantity9, Delta: output.BidQuantityDelta9},
		{Price: output.BidPrice10, Quantity: output.BidQuantity10, Delta: output.BidQuantityDelta10},
	}
}

type orderbookEnvelope struct {
	responseFields
	Output1 OrderbookOutput          `json:"output1"`
	Output2 ExpectedConclusionOutput `json:"output2"`
}

// OrderbookOutput is the provider-native output1 object from Orderbook.
type OrderbookOutput struct {
	AcceptanceTime string `json:"aspr_acpt_hour"`

	AskPrice1  string `json:"askp1"`
	AskPrice2  string `json:"askp2"`
	AskPrice3  string `json:"askp3"`
	AskPrice4  string `json:"askp4"`
	AskPrice5  string `json:"askp5"`
	AskPrice6  string `json:"askp6"`
	AskPrice7  string `json:"askp7"`
	AskPrice8  string `json:"askp8"`
	AskPrice9  string `json:"askp9"`
	AskPrice10 string `json:"askp10"`

	BidPrice1  string `json:"bidp1"`
	BidPrice2  string `json:"bidp2"`
	BidPrice3  string `json:"bidp3"`
	BidPrice4  string `json:"bidp4"`
	BidPrice5  string `json:"bidp5"`
	BidPrice6  string `json:"bidp6"`
	BidPrice7  string `json:"bidp7"`
	BidPrice8  string `json:"bidp8"`
	BidPrice9  string `json:"bidp9"`
	BidPrice10 string `json:"bidp10"`

	AskQuantity1  string `json:"askp_rsqn1"`
	AskQuantity2  string `json:"askp_rsqn2"`
	AskQuantity3  string `json:"askp_rsqn3"`
	AskQuantity4  string `json:"askp_rsqn4"`
	AskQuantity5  string `json:"askp_rsqn5"`
	AskQuantity6  string `json:"askp_rsqn6"`
	AskQuantity7  string `json:"askp_rsqn7"`
	AskQuantity8  string `json:"askp_rsqn8"`
	AskQuantity9  string `json:"askp_rsqn9"`
	AskQuantity10 string `json:"askp_rsqn10"`

	BidQuantity1  string `json:"bidp_rsqn1"`
	BidQuantity2  string `json:"bidp_rsqn2"`
	BidQuantity3  string `json:"bidp_rsqn3"`
	BidQuantity4  string `json:"bidp_rsqn4"`
	BidQuantity5  string `json:"bidp_rsqn5"`
	BidQuantity6  string `json:"bidp_rsqn6"`
	BidQuantity7  string `json:"bidp_rsqn7"`
	BidQuantity8  string `json:"bidp_rsqn8"`
	BidQuantity9  string `json:"bidp_rsqn9"`
	BidQuantity10 string `json:"bidp_rsqn10"`

	AskQuantityDelta1  string `json:"askp_rsqn_icdc1"`
	AskQuantityDelta2  string `json:"askp_rsqn_icdc2"`
	AskQuantityDelta3  string `json:"askp_rsqn_icdc3"`
	AskQuantityDelta4  string `json:"askp_rsqn_icdc4"`
	AskQuantityDelta5  string `json:"askp_rsqn_icdc5"`
	AskQuantityDelta6  string `json:"askp_rsqn_icdc6"`
	AskQuantityDelta7  string `json:"askp_rsqn_icdc7"`
	AskQuantityDelta8  string `json:"askp_rsqn_icdc8"`
	AskQuantityDelta9  string `json:"askp_rsqn_icdc9"`
	AskQuantityDelta10 string `json:"askp_rsqn_icdc10"`

	BidQuantityDelta1  string `json:"bidp_rsqn_icdc1"`
	BidQuantityDelta2  string `json:"bidp_rsqn_icdc2"`
	BidQuantityDelta3  string `json:"bidp_rsqn_icdc3"`
	BidQuantityDelta4  string `json:"bidp_rsqn_icdc4"`
	BidQuantityDelta5  string `json:"bidp_rsqn_icdc5"`
	BidQuantityDelta6  string `json:"bidp_rsqn_icdc6"`
	BidQuantityDelta7  string `json:"bidp_rsqn_icdc7"`
	BidQuantityDelta8  string `json:"bidp_rsqn_icdc8"`
	BidQuantityDelta9  string `json:"bidp_rsqn_icdc9"`
	BidQuantityDelta10 string `json:"bidp_rsqn_icdc10"`

	TotalAskQuantity string `json:"total_askp_rsqn"`
	TotalBidQuantity string `json:"total_bidp_rsqn"`
}

// ExpectedConclusionOutput is the provider-native output2 object from Orderbook.
type ExpectedConclusionOutput struct {
	Symbol             string `json:"stck_shrn_iscd"`
	Current            string `json:"stck_prpr"`
	Open               string `json:"stck_oprc"`
	High               string `json:"stck_hgpr"`
	Low                string `json:"stck_lwpr"`
	PreviousClose      string `json:"stck_sdpr"`
	ExpectedPrice      string `json:"antc_cnpr"`
	ExpectedVolume     string `json:"antc_vol"`
	PreviousChange     string `json:"antc_cntg_vrss"`
	PreviousChangeSign string `json:"antc_cntg_vrss_sign"`
	PreviousChangeRate string `json:"antc_cntg_prdy_ctrt"`
}
