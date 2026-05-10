package krx

import (
	"context"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/samber/oops"
)

// Client calls the KRX Data Marketplace OPEN API.
//
// Client owns KRX-specific endpoint paths, AUTH_KEY header handling,
// provider-native response parsing, and remote error context. It does not
// expose resty or depend on mwosa CLI, provider adapters, storage, or Cobra.
type Client struct {
	authKey string
	http    *resty.Client
}

// New creates a KRX OPEN API client.
func New(options ...Option) (*Client, error) {
	cfg := defaultConfig()
	errb := oops.In("krx_client").With("provider", ProviderKRX)
	for _, option := range options {
		if option == nil {
			return nil, errb.New("krx client config: option is required")
		}
		if err := option(&cfg); err != nil {
			return nil, errb.Wrapf(err, "apply krx client option")
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	restyClient := resty.NewWithClient(cfg.httpClient).
		SetBaseURL(cfg.selectedBaseURL()).
		SetHeader("accept", "application/json")

	return &Client{
		authKey: cfg.authKey,
		http:    restyClient,
	}, nil
}

type apiEndpoint struct {
	group string
	apiID string
	path  string
}

func endpoint(group string, apiID string) apiEndpoint {
	return apiEndpoint{
		group: group,
		apiID: apiID,
		path:  "/" + group + "/" + apiID,
	}
}

func (e apiEndpoint) validate(errb oops.OopsErrorBuilder) error {
	if strings.TrimSpace(e.group) == "" || strings.TrimSpace(e.apiID) == "" || strings.TrimSpace(e.path) == "" {
		return errb.Errorf("krx API is unsupported: provider=%s group=%s api_id=%s", ProviderKRX, e.group, e.apiID)
	}
	return nil
}

func (c *Client) fetch(ctx context.Context, endpoint apiEndpoint, baseDate string, result any) error {
	baseDate = strings.TrimSpace(baseDate)
	errb := oops.In("krx_client").With(
		"provider", ProviderKRX,
		"group", endpoint.group,
		"api_id", endpoint.apiID,
		"endpoint", endpoint.path,
		"base_date", baseDate,
	)
	if err := endpoint.validate(errb); err != nil {
		return err
	}
	if baseDate == "" {
		return errb.New("krx request: base date is required")
	}

	response, err := c.http.R().
		SetContext(ctx).
		SetHeader("AUTH_KEY", c.authKey).
		SetQueryParam("basDd", baseDate).
		SetResult(result).
		Get(endpoint.path)
	if err != nil {
		return errb.Wrapf(err, "request krx API")
	}
	return checkHTTP(response, errb, endpoint)
}

func (c *Client) outBlock(ctx context.Context, endpoint apiEndpoint, baseDate string, result any, outBlockPresent func() bool) error {
	if err := c.fetch(ctx, endpoint, baseDate, result); err != nil {
		return err
	}
	errb := oops.In("krx_client").With(
		"provider", ProviderKRX,
		"group", endpoint.group,
		"api_id", endpoint.apiID,
		"endpoint", endpoint.path,
		"base_date", strings.TrimSpace(baseDate),
	)
	if !outBlockPresent() {
		return errb.New("krx response: OutBlock_1 is required")
	}
	return nil
}

func checkHTTP(response *resty.Response, errb oops.OopsErrorBuilder, endpoint apiEndpoint) error {
	if response == nil {
		return errb.New("krx response is required")
	}
	if response.IsError() {
		return errb.With("status", response.StatusCode()).Errorf(
			"krx HTTP request failed: provider=%s group=%s api_id=%s status=%d body=%s",
			ProviderKRX,
			endpoint.group,
			endpoint.apiID,
			response.StatusCode(),
			string(response.Body()),
		)
	}
	return nil
}
