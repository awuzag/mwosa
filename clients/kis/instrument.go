package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// DefaultDomesticStockProductType is the KIS product type for domestic stocks,
// ETF, ETN, and ELW.
const DefaultDomesticStockProductType = "300"

// InstrumentOption configures Instrument service metadata requests.
type InstrumentOption func(*instrumentQuery) error

type instrumentQuery struct {
	productTypeCode string
}

func defaultInstrumentQuery() instrumentQuery {
	return instrumentQuery{productTypeCode: DefaultDomesticStockProductType}
}

// WithProductType sets the KIS PRDT_TYPE_CD query value.
func WithProductType(productTypeCode string) InstrumentOption {
	return func(q *instrumentQuery) error {
		q.productTypeCode = strings.TrimSpace(productTypeCode)
		return nil
	}
}

// Product is product metadata returned by InstrumentService.Product.
type Product struct {
	ProductNo              string
	ProductTypeCode        string
	Name                   string
	Name120                string
	AbbreviatedName        string
	EnglishName            string
	StandardProductNo      string
	ShortProductNo         string
	ProductClassCode       string
	ProductClassName       string
	InvestmentTypeCode     string
	InvestmentTypeCodeName string
	Raw                    ProductOutput
}

// ProductOutput is the provider-native output object from the KIS product endpoint.
type ProductOutput struct {
	ProductNo              string `json:"pdno"`
	ProductTypeCode        string `json:"prdt_type_cd"`
	Name                   string `json:"prdt_name"`
	Name120                string `json:"prdt_name120"`
	AbbreviatedName        string `json:"prdt_abrv_name"`
	EnglishName            string `json:"prdt_eng_name"`
	EnglishName120         string `json:"prdt_eng_name120"`
	EnglishAbbreviatedName string `json:"prdt_eng_abrv_name"`
	StandardProductNo      string `json:"std_pdno"`
	ShortProductNo         string `json:"shtn_pdno"`
	SaleStatusCode         string `json:"prdt_sale_stat_cd"`
	RiskGradeCode          string `json:"prdt_risk_grad_cd"`
	ProductClassCode       string `json:"prdt_clsf_cd"`
	ProductClassName       string `json:"prdt_clsf_name"`
	SaleStartDate          string `json:"sale_strt_dt"`
	SaleEndDate            string `json:"sale_end_dt"`
	WrapAssetTypeCode      string `json:"wrap_asst_type_cd"`
	InvestmentTypeCode     string `json:"ivst_prdt_type_cd"`
	InvestmentTypeCodeName string `json:"ivst_prdt_type_cd_name"`
	FirstRegisteredDate    string `json:"frst_erlm_dt"`
}

// Stock is domestic stock metadata returned by InstrumentService.Stock.
type Stock struct {
	ProductNo            string
	ProductTypeCode      string
	MarketIDCode         string
	SecurityGroupIDCode  string
	ExchangeDivisionCode string
	SettlementMonthDay   string
	ListedShares         string
	Capital              string
	ParValue             string
	Name                 string
	AbbreviatedName      string
	EnglishName          string
	StandardProductNo    string
	TradingHalt          string
	AdministrativeIssue  string
	IndustryCode         string
	IndustryName         string
	Raw                  StockOutput
}

// StockOutput is the provider-native output object from the KIS stock endpoint.
type StockOutput struct {
	ProductNo                   string `json:"pdno"`
	ProductTypeCode             string `json:"prdt_type_cd"`
	MarketIDCode                string `json:"mket_id_cd"`
	SecurityGroupIDCode         string `json:"scty_grp_id_cd"`
	ExchangeDivisionCode        string `json:"excg_dvsn_cd"`
	SettlementMonthDay          string `json:"setl_mmdd"`
	ListedShares                string `json:"lstg_stqt"`
	ListedCapitalAmount         string `json:"lstg_cptl_amt"`
	Capital                     string `json:"cpta"`
	ParValue                    string `json:"papr"`
	IssuePrice                  string `json:"issu_pric"`
	KOSPI200                    string `json:"kospi200_item_yn"`
	KOSPIListedDate             string `json:"scts_mket_lstg_dt"`
	KOSDAQListedDate            string `json:"kosdaq_mket_lstg_dt"`
	ETFDivisionCode             string `json:"etf_dvsn_cd"`
	StockKindCode               string `json:"stck_kind_cd"`
	ProductName                 string `json:"prdt_name"`
	ProductName120              string `json:"prdt_name120"`
	AbbreviatedName             string `json:"prdt_abrv_name"`
	StandardProductNo           string `json:"std_pdno"`
	EnglishName                 string `json:"prdt_eng_name"`
	EnglishName120              string `json:"prdt_eng_name120"`
	EnglishAbbreviatedName      string `json:"prdt_eng_abrv_name"`
	TradingHalt                 string `json:"tr_stop_yn"`
	AdministrativeIssue         string `json:"admn_item_yn"`
	TodayClose                  string `json:"thdt_clpr"`
	PreviousClose               string `json:"bfdy_clpr"`
	CloseChangeDate             string `json:"clpr_chng_dt"`
	StandardIndustryCode        string `json:"std_idst_clsf_cd"`
	StandardIndustryName        string `json:"std_idst_clsf_cd_name"`
	IndexIndustryLargeName      string `json:"idx_bztp_lcls_cd_name"`
	IndexIndustryMiddleName     string `json:"idx_bztp_mcls_cd_name"`
	IndexIndustrySmallName      string `json:"idx_bztp_scls_cd_name"`
	NXTTradable                 string `json:"cptt_trad_tr_psbl_yn"`
	NXTTradingHalt              string `json:"nxt_tr_stop_yn"`
	ElectronicSecurities        string `json:"elec_scty_yn"`
	ETFETNInvestmentWarningItem string `json:"etf_etn_ivst_heed_item_yn"`
}

// InstrumentService calls handwritten KIS instrument metadata APIs.
type InstrumentService struct {
	client *Client
}

// Instrument returns the KIS instrument metadata service.
func (c *Client) Instrument() InstrumentService {
	return InstrumentService{client: c}
}

// Product fetches product metadata for a KIS product number.
//
// The request uses TR ID CTPF1604R. The default product type is "300" for
// domestic stocks, ETF, ETN, and ELW.
func (s InstrumentService) Product(ctx context.Context, productNo string, options ...InstrumentOption) (Product, error) {
	query := defaultInstrumentQuery()
	errb := instrumentErrBuilder(OperationProduct, "/uapi/domestic-stock/v1/quotations/search-info", trIDDomesticStockProduct, productNo)
	if s.client == nil {
		return Product{}, errb.New("kis product request: client is required")
	}
	for _, option := range options {
		if option == nil {
			return Product{}, errb.New("kis product request: option is required")
		}
		if err := option(&query); err != nil {
			return Product{}, errb.Wrapf(err, "apply kis product option")
		}
	}
	if err := query.validate(productNo, OperationProduct); err != nil {
		return Product{}, errb.Wrap(err)
	}

	var envelope productEnvelope
	request, err := s.client.request(ctx, GroupInstrument, OperationProduct, trIDDomesticStockProduct, errb)
	if err != nil {
		return Product{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"PDNO":         strings.TrimSpace(productNo),
		"PRDT_TYPE_CD": query.productTypeCode,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/search-info")
	if err != nil {
		return Product{}, errb.Wrapf(err, "request kis product")
	}
	if err := checkHTTP(response, errb, GroupInstrument, OperationProduct, trIDDomesticStockProduct); err != nil {
		return Product{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupInstrument, OperationProduct, trIDDomesticStockProduct); err != nil {
		return Product{}, err
	}
	return productFromOutput(envelope.Output), nil
}

// Stock fetches domestic stock metadata for a KIS short stock code.
//
// The request uses TR ID CTPF1002R. The default product type is "300".
func (s InstrumentService) Stock(ctx context.Context, symbol string, options ...InstrumentOption) (Stock, error) {
	query := defaultInstrumentQuery()
	errb := instrumentErrBuilder(OperationStock, "/uapi/domestic-stock/v1/quotations/search-stock-info", trIDDomesticStockInfo, symbol)
	if s.client == nil {
		return Stock{}, errb.New("kis stock request: client is required")
	}
	for _, option := range options {
		if option == nil {
			return Stock{}, errb.New("kis stock request: option is required")
		}
		if err := option(&query); err != nil {
			return Stock{}, errb.Wrapf(err, "apply kis stock option")
		}
	}
	if err := query.validate(symbol, OperationStock); err != nil {
		return Stock{}, errb.Wrap(err)
	}

	var envelope stockEnvelope
	request, err := s.client.request(ctx, GroupInstrument, OperationStock, trIDDomesticStockInfo, errb)
	if err != nil {
		return Stock{}, err
	}
	response, err := request.SetQueryParams(map[string]string{
		"PDNO":         strings.TrimSpace(symbol),
		"PRDT_TYPE_CD": query.productTypeCode,
	}).
		SetResult(&envelope).
		Get("/uapi/domestic-stock/v1/quotations/search-stock-info")
	if err != nil {
		return Stock{}, errb.Wrapf(err, "request kis stock")
	}
	if err := checkHTTP(response, errb, GroupInstrument, OperationStock, trIDDomesticStockInfo); err != nil {
		return Stock{}, err
	}
	if err := checkKIS(envelope.responseFields, errb, GroupInstrument, OperationStock, trIDDomesticStockInfo); err != nil {
		return Stock{}, err
	}
	return stockFromOutput(envelope.Output), nil
}

func (q instrumentQuery) validate(productNo string, operation string) error {
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupInstrument,
		"operation", operation,
	)
	if strings.TrimSpace(productNo) == "" {
		return errb.New("kis instrument request: product number is required")
	}
	if q.productTypeCode == "" {
		return errb.New("kis instrument request: product type code is required")
	}
	return nil
}

func instrumentErrBuilder(operation string, endpoint string, trID string, productNo string) oops.OopsErrorBuilder {
	return oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", GroupInstrument,
		"operation", operation,
		"endpoint", endpoint,
		"tr_id", trID,
		"product_no", strings.TrimSpace(productNo),
	)
}

func productFromOutput(output ProductOutput) Product {
	return Product{
		ProductNo:              output.ProductNo,
		ProductTypeCode:        output.ProductTypeCode,
		Name:                   output.Name,
		Name120:                output.Name120,
		AbbreviatedName:        output.AbbreviatedName,
		EnglishName:            output.EnglishName,
		StandardProductNo:      output.StandardProductNo,
		ShortProductNo:         output.ShortProductNo,
		ProductClassCode:       output.ProductClassCode,
		ProductClassName:       output.ProductClassName,
		InvestmentTypeCode:     output.InvestmentTypeCode,
		InvestmentTypeCodeName: output.InvestmentTypeCodeName,
		Raw:                    output,
	}
}

func stockFromOutput(output StockOutput) Stock {
	return Stock{
		ProductNo:            output.ProductNo,
		ProductTypeCode:      output.ProductTypeCode,
		MarketIDCode:         output.MarketIDCode,
		SecurityGroupIDCode:  output.SecurityGroupIDCode,
		ExchangeDivisionCode: output.ExchangeDivisionCode,
		SettlementMonthDay:   output.SettlementMonthDay,
		ListedShares:         output.ListedShares,
		Capital:              output.Capital,
		ParValue:             output.ParValue,
		Name:                 output.ProductName,
		AbbreviatedName:      output.AbbreviatedName,
		EnglishName:          output.EnglishName,
		StandardProductNo:    output.StandardProductNo,
		TradingHalt:          output.TradingHalt,
		AdministrativeIssue:  output.AdministrativeIssue,
		IndustryCode:         output.StandardIndustryCode,
		IndustryName:         output.StandardIndustryName,
		Raw:                  output,
	}
}

type productEnvelope struct {
	responseFields
	Output ProductOutput `json:"output"`
}

type stockEnvelope struct {
	responseFields
	Output StockOutput `json:"output"`
}
