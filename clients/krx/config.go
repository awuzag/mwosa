package krx

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	// DefaultBaseURL is the KRX OPEN API production endpoint root.
	DefaultBaseURL = "https://data-dbg.krx.co.kr/svc/apis"

	// DefaultSampleBaseURL is the KRX OPEN API sample endpoint root.
	DefaultSampleBaseURL = "https://data-dbg.krx.co.kr/svc/sample/apis"

	// ProviderKRX is the provider identifier used in SDK error context.
	ProviderKRX = "krx"

	// GroupETP is the KRX securities product API path group.
	GroupETP = "etp"

	// GroupIndex is the KRX index API path group.
	GroupIndex = "idx"

	// GroupStock is the KRX stock API path group.
	GroupStock = "sto"

	// GroupBond is the KRX bond API path group.
	GroupBond = "bon"

	// GroupDerivatives is the KRX derivatives API path group.
	GroupDerivatives = "drv"

	// GroupCommodity is the KRX commodity API path group.
	GroupCommodity = "gen"

	// GroupESG is the KRX ESG API path group.
	GroupESG = "esg"

	// APIKRXDDTrd is the KRX index daily trade API ID.
	APIKRXDDTrd = "krx_dd_trd"

	// APIKOSPIDDTrd is the KOSPI index daily trade API ID.
	APIKOSPIDDTrd = "kospi_dd_trd"

	// APIKOSDAQDDTrd is the KOSDAQ index daily trade API ID.
	APIKOSDAQDDTrd = "kosdaq_dd_trd"

	// APIBondDDTrd is the bond index daily trade API ID.
	APIBondDDTrd = "bon_dd_trd"

	// APIDerivativesProductDDTrd is the derivatives product index daily trade API ID.
	APIDerivativesProductDDTrd = "drvprod_dd_trd"

	// APIStockByddTrd is the stock daily trade API ID.
	APIStockByddTrd = "stk_bydd_trd"

	// APIKOSDAQByddTrd is the KOSDAQ stock daily trade API ID.
	APIKOSDAQByddTrd = "ksq_bydd_trd"

	// APIKONEXByddTrd is the KONEX stock daily trade API ID.
	APIKONEXByddTrd = "knx_bydd_trd"

	// APISubscriptionWarrantByddTrd is the subscription warrant daily trade API ID.
	APISubscriptionWarrantByddTrd = "sw_bydd_trd"

	// APISubscriptionRightByddTrd is the subscription right daily trade API ID.
	APISubscriptionRightByddTrd = "sr_bydd_trd"

	// APIStockIssueBaseInfo is the stock issue base info API ID.
	APIStockIssueBaseInfo = "stk_isu_base_info"

	// APIKOSDAQIssueBaseInfo is the KOSDAQ issue base info API ID.
	APIKOSDAQIssueBaseInfo = "ksq_isu_base_info"

	// APIKONEXIssueBaseInfo is the KONEX issue base info API ID.
	APIKONEXIssueBaseInfo = "knx_isu_base_info"

	// APIETFByddTrd is the ETF daily trade API ID.
	APIETFByddTrd = "etf_bydd_trd"

	// APIETNByddTrd is the ETN daily trade API ID.
	APIETNByddTrd = "etn_bydd_trd"

	// APIELWByddTrd is the ELW daily trade API ID.
	APIELWByddTrd = "elw_bydd_trd"

	// APIKTSByddTrd is the KTS bond daily trade API ID.
	APIKTSByddTrd = "kts_bydd_trd"

	// APIGeneralBondByddTrd is the general bond daily trade API ID.
	APIGeneralBondByddTrd = "bnd_bydd_trd"

	// APISmallBondByddTrd is the small bond daily trade API ID.
	APISmallBondByddTrd = "smb_bydd_trd"

	// APIFuturesByddTrd is the futures daily trade API ID.
	APIFuturesByddTrd = "fut_bydd_trd"

	// APIKOSPIStockFuturesByddTrd is the KOSPI stock futures daily trade API ID.
	APIKOSPIStockFuturesByddTrd = "eqsfu_stk_bydd_trd"

	// APIKOSDAQStockFuturesByddTrd is the KOSDAQ stock futures daily trade API ID.
	APIKOSDAQStockFuturesByddTrd = "eqkfu_ksq_bydd_trd"

	// APIOptionsByddTrd is the options daily trade API ID.
	APIOptionsByddTrd = "opt_bydd_trd"

	// APIKOSPIStockOptionsByddTrd is the KOSPI stock options daily trade API ID.
	APIKOSPIStockOptionsByddTrd = "eqsop_bydd_trd"

	// APIKOSDAQStockOptionsByddTrd is the KOSDAQ stock options daily trade API ID.
	APIKOSDAQStockOptionsByddTrd = "eqkop_bydd_trd"

	// APIOilByddTrd is the oil daily trade API ID.
	APIOilByddTrd = "oil_bydd_trd"

	// APIGoldByddTrd is the gold daily trade API ID.
	APIGoldByddTrd = "gold_bydd_trd"

	// APIEmissionTradingSchemeByddTrd is the ETS daily trade API ID.
	APIEmissionTradingSchemeByddTrd = "ets_bydd_trd"

	// APIESGETPInfo is the ESG ETP info API ID.
	APIESGETPInfo = "esg_etp_info"

	// APISRIBondInfo is the SRI bond info API ID.
	APISRIBondInfo = "sri_bond_info"

	// APIESGIndexInfo is the ESG index info API ID.
	APIESGIndexInfo = "esg_index_info"

	// DefaultHTTPClientTimeout is used when no custom HTTP client is provided.
	DefaultHTTPClientTimeout = 15 * time.Second
)

type config struct {
	authKey       string
	baseURL       string
	sampleBaseURL string
	useSample     bool
	httpClient    *http.Client
}

func defaultConfig() config {
	return config{
		baseURL:       DefaultBaseURL,
		sampleBaseURL: DefaultSampleBaseURL,
		httpClient:    &http.Client{Timeout: DefaultHTTPClientTimeout},
	}
}

func (c config) selectedBaseURL() string {
	if c.useSample {
		return c.sampleBaseURL
	}
	return c.baseURL
}

func (c config) validate() error {
	errb := oops.In("krx_client").With("provider", ProviderKRX)
	if strings.TrimSpace(c.authKey) == "" {
		return errb.New("krx client config: auth key is required")
	}
	if _, err := url.ParseRequestURI(c.baseURL); err != nil {
		return errb.With("base_url", c.baseURL).Wrapf(err, "krx client config: invalid base URL")
	}
	if _, err := url.ParseRequestURI(c.sampleBaseURL); err != nil {
		return errb.With("base_url", c.sampleBaseURL).Wrapf(err, "krx client config: invalid sample base URL")
	}
	if c.httpClient == nil {
		return errb.New("krx client config: HTTP client is required")
	}
	return nil
}

// Option configures a Client during New.
type Option func(*config) error

// WithAuthKey sets the KRX OPEN API auth key sent in the AUTH_KEY header.
func WithAuthKey(authKey string) Option {
	return func(c *config) error {
		c.authKey = strings.TrimSpace(authKey)
		return nil
	}
}

// WithBaseURL overrides the production base URL, mainly for tests.
func WithBaseURL(baseURL string) Option {
	return func(c *config) error {
		c.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		return nil
	}
}

// WithSampleBaseURL selects and overrides the sample base URL.
func WithSampleBaseURL(baseURL string) Option {
	return func(c *config) error {
		c.sampleBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		c.useSample = true
		return nil
	}
}

// WithHTTPClient sets the underlying HTTP client used by resty.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *config) error {
		if httpClient == nil {
			return oops.In("krx_client").New("krx client config: HTTP client is required")
		}
		c.httpClient = httpClient
		return nil
	}
}
