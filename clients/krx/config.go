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

	// APIETFByddTrd is the ETF daily trade API ID.
	APIETFByddTrd = "etf_bydd_trd"

	// APIETNByddTrd is the ETN daily trade API ID.
	APIETNByddTrd = "etn_bydd_trd"

	// APIELWByddTrd is the ELW daily trade API ID.
	APIELWByddTrd = "elw_bydd_trd"

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
