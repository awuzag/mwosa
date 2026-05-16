package kis

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	// DefaultRealBaseURL is the KIS production OpenAPI domain.
	DefaultRealBaseURL = "https://openapi.koreainvestment.com:9443"

	// DefaultVirtualBaseURL is the KIS virtual investment OpenAPI domain.
	DefaultVirtualBaseURL = "https://openapivts.koreainvestment.com:29443"

	// ProviderKIS is the provider identifier used in SDK error context.
	ProviderKIS = "kis"

	// GroupDomesticStockQuotation is the KIS domestic stock quotation API group.
	GroupDomesticStockQuotation = "domesticStockQuotation"

	// GroupDomesticStockInstrument is the KIS domestic stock instrument API group.
	GroupDomesticStockInstrument = "domesticStockInstrument"

	// OperationToken is the SDK operation name for OAuth token issuance.
	OperationToken = "token"

	// OperationPrice is the SDK operation name for domestic stock price quotes.
	OperationPrice = "price"

	// OperationDaily is the SDK operation name for domestic stock daily bars.
	OperationDaily = "daily"

	// OperationIntraday is the SDK operation name for domestic stock minute bars.
	OperationIntraday = "intraday"

	// OperationOrderbook is the SDK operation name for domestic stock orderbooks.
	OperationOrderbook = "orderbook"

	// OperationTrades is the SDK operation name for recent domestic stock trades.
	OperationTrades = "trades"

	// OperationTimeTrades is the SDK operation name for time-filtered domestic stock trades.
	OperationTimeTrades = "timeTrades"

	// OperationProduct is the SDK operation name for product metadata.
	OperationProduct = "product"

	// OperationStock is the SDK operation name for stock metadata.
	OperationStock = "stock"

	// OperationETFETNPrice is the SDK operation name for ETF/ETN price quotes.
	OperationETFETNPrice = "etfetnPrice"

	// DefaultCustomerType is the KIS customer type for individual customers.
	DefaultCustomerType = "P"

	// DefaultHTTPClientTimeout is used when no custom HTTP client is provided.
	DefaultHTTPClientTimeout = 15 * time.Second

	trIDDomesticStockPrice          = "FHKST01010100"
	trIDDomesticStockDailyItemChart = "FHKST03010100"
	trIDDomesticStockIntraday       = "FHKST03010200"
	trIDDomesticStockOrderbook      = "FHKST01010200"
	trIDDomesticStockTrades         = "FHKST01010300"
	trIDDomesticStockTimeTrades     = "FHPST01060000"
	trIDDomesticStockProduct        = "CTPF1604R"
	trIDDomesticStockInfo           = "CTPF1002R"
	trIDETFETNPrice                 = "FHPST02400000"
)

type config struct {
	appKey         string
	appSecret      string
	accessToken    string
	virtual        bool
	realBaseURL    string
	virtualBaseURL string
	customerType   string
	httpClient     *http.Client
	account        string
}

func defaultConfig() config {
	return config{
		realBaseURL:    DefaultRealBaseURL,
		virtualBaseURL: DefaultVirtualBaseURL,
		customerType:   DefaultCustomerType,
		httpClient:     &http.Client{Timeout: DefaultHTTPClientTimeout},
	}
}

func (c config) selectedBaseURL() string {
	if c.virtual {
		return c.virtualBaseURL
	}
	return c.realBaseURL
}

func (c config) validate() error {
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
	)
	if strings.TrimSpace(c.appKey) == "" {
		return errb.New("kis client config: app key is required")
	}
	if strings.TrimSpace(c.appSecret) == "" {
		return errb.New("kis client config: app secret is required")
	}
	if _, err := url.ParseRequestURI(c.realBaseURL); err != nil {
		return errb.With("base_url", c.realBaseURL).Wrapf(err, "kis client config: invalid real base URL")
	}
	if _, err := url.ParseRequestURI(c.virtualBaseURL); err != nil {
		return errb.With("base_url", c.virtualBaseURL).Wrapf(err, "kis client config: invalid virtual base URL")
	}
	if strings.TrimSpace(c.customerType) == "" {
		return errb.New("kis client config: customer type is required")
	}
	if c.httpClient == nil {
		return errb.New("kis client config: HTTP client is required")
	}
	return nil
}

// Option configures a Client during New.
type Option func(*config) error

// WithAppKey sets the KIS app key issued for the selected real or virtual API.
func WithAppKey(appKey string) Option {
	return func(c *config) error {
		c.appKey = strings.TrimSpace(appKey)
		return nil
	}
}

// WithAppSecret sets the KIS app secret paired with the app key.
func WithAppSecret(appSecret string) Option {
	return func(c *config) error {
		c.appSecret = strings.TrimSpace(appSecret)
		return nil
	}
}

// WithAccessToken sets an already-issued OAuth access token for API calls.
func WithAccessToken(accessToken string) Option {
	return func(c *config) error {
		c.accessToken = strings.TrimSpace(accessToken)
		return nil
	}
}

// WithVirtual selects the KIS virtual investment domain.
func WithVirtual() Option {
	return func(c *config) error {
		c.virtual = true
		return nil
	}
}

// WithBaseURL overrides the production base URL, mainly for tests.
func WithBaseURL(baseURL string) Option {
	return func(c *config) error {
		c.realBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		return nil
	}
}

// WithVirtualBaseURL overrides the virtual investment base URL, mainly for tests.
func WithVirtualBaseURL(baseURL string) Option {
	return func(c *config) error {
		c.virtualBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		return nil
	}
}

// WithHTTPClient sets the underlying HTTP client used by resty.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *config) error {
		if httpClient == nil {
			return oops.In("kis_client").New("kis client config: HTTP client is required")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithCustomerType sets the KIS custtype header value.
func WithCustomerType(customerType string) Option {
	return func(c *config) error {
		c.customerType = strings.TrimSpace(customerType)
		return nil
	}
}

// WithAccount stores an account identifier for future account or trading APIs.
func WithAccount(account string) Option {
	return func(c *config) error {
		c.account = strings.TrimSpace(account)
		return nil
	}
}
