package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// ETFETNPrice is a simplified ETF/ETN current quote returned by Client.ETFETNPrice.
type ETFETNPrice struct {
	Current            string
	Open               string
	High               string
	Low                string
	PreviousClose      string
	PreviousChange     string
	PreviousChangeSign string
	PreviousChangeRate string
	Volume             string
	NAV                string
	NAVChange          string
	NAVChangeSign      string
	NAVChangeRate      string
	Currency           string
	NetAssets          string
	UnderlyingName     string
	Raw                ETFETNPriceOutput
}

// ETFETNPriceOutput is the provider-native output object from the ETF/ETN price endpoint.
type ETFETNPriceOutput struct {
	Current              string `json:"stck_prpr"`
	PreviousChangeSign   string `json:"prdy_vrss_sign"`
	PreviousChange       string `json:"prdy_vrss"`
	PreviousChangeRate   string `json:"prdy_ctrt"`
	Volume               string `json:"acml_vol"`
	PreviousVolume       string `json:"prdy_vol"`
	UpperLimit           string `json:"stck_mxpr"`
	LowerLimit           string `json:"stck_llam"`
	PreviousClose        string `json:"stck_prdy_clpr"`
	Open                 string `json:"stck_oprc"`
	High                 string `json:"stck_hgpr"`
	Low                  string `json:"stck_lwpr"`
	PreviousNAV          string `json:"prdy_last_nav"`
	NAV                  string `json:"nav"`
	NAVChange            string `json:"nav_prdy_vrss"`
	NAVChangeSign        string `json:"nav_prdy_vrss_sign"`
	NAVChangeRate        string `json:"nav_prdy_ctrt"`
	TrackingErrorRate    string `json:"trc_errt"`
	BasePrice            string `json:"stck_sdpr"`
	Currency             string `json:"crcd"`
	CirculatingShares    string `json:"etf_crcl_stcn"`
	NetAssets            string `json:"etf_ntas_ttam"`
	SectorName           string `json:"bstp_kor_isnm"`
	ListedShares         string `json:"lstn_stcn"`
	ForeignHolding       string `json:"frgn_hldn_qty"`
	ForeignHoldingRate   string `json:"frgn_hldn_qty_rate"`
	TrackingReturnFactor string `json:"etf_trc_ert_mltp"`
	DisparityRate        string `json:"dprt"`
	ManagerName          string `json:"mbcr_name"`
	ListedDate           string `json:"stck_lstn_date"`
	ETFDivisionName      string `json:"etf_div_name"`
	UnderlyingName       string `json:"etf_rprs_bstp_kor_isnm"`
}

// ETFETNPrice fetches the current ETF or ETN quote for symbol.
//
// The request uses TR ID FHPST02400000. Symbol should be the KRX short code.
func (c *Client) ETFETNPrice(ctx context.Context, symbol string) (ETFETNPrice, error) {
	symbol = strings.TrimSpace(symbol)
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupQuote,
		"operation", OperationETFETNPrice,
		"endpoint", "/uapi/etfetn/v1/quotations/inquire-price",
		"tr_id", trIDETFETNPrice,
		"symbol", symbol,
	)
	if symbol == "" {
		return ETFETNPrice{}, errb.New("kis ETF/ETN price request: symbol is required")
	}

	var envelope etfetnPriceEnvelope
	request, err := c.request(ctx, GroupQuote, OperationETFETNPrice, trIDETFETNPrice, errb)
	if err != nil {
		return ETFETNPrice{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"fid_cond_mrkt_div_code": "J",
		"fid_input_iscd":         symbol,
	}).
		SetResult(&envelope).
		Get("/uapi/etfetn/v1/quotations/inquire-price")
	if err != nil {
		return ETFETNPrice{}, errb.Wrapf(err, "request kis ETF/ETN price")
	}
	if err := checkHTTP(response, errb, GroupQuote, OperationETFETNPrice, trIDETFETNPrice); err != nil {
		return ETFETNPrice{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupQuote, OperationETFETNPrice, trIDETFETNPrice); err != nil {
		return ETFETNPrice{}, err
	}
	return etfetnPriceFromOutput(envelope.Output), nil
}

func etfetnPriceFromOutput(output ETFETNPriceOutput) ETFETNPrice {
	return ETFETNPrice{
		Current:            output.Current,
		Open:               output.Open,
		High:               output.High,
		Low:                output.Low,
		PreviousClose:      output.PreviousClose,
		PreviousChange:     output.PreviousChange,
		PreviousChangeSign: output.PreviousChangeSign,
		PreviousChangeRate: output.PreviousChangeRate,
		Volume:             output.Volume,
		NAV:                output.NAV,
		NAVChange:          output.NAVChange,
		NAVChangeSign:      output.NAVChangeSign,
		NAVChangeRate:      output.NAVChangeRate,
		Currency:           output.Currency,
		NetAssets:          output.NetAssets,
		UnderlyingName:     output.UnderlyingName,
		Raw:                output,
	}
}

type etfetnPriceEnvelope struct {
	responseFields
	Output ETFETNPriceOutput `json:"output"`
}
