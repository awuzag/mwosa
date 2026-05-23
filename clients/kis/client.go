package kis

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/oops"
)

// Client calls the KIS Developers OpenAPI.
//
// Client owns KIS-specific domains, headers, TR IDs, OAuth token state, response
// envelopes, and business error mapping. It does not expose resty or depend on
// mwosa CLI, provider adapters, storage, or Cobra.
type Client struct {
	appKey       string
	appSecret    string
	customerType string
	account      string
	virtual      bool
	http         *resty.Client

	tokenMu sync.RWMutex
	token   tokenState
}

type tokenState struct {
	accessToken string
	tokenType   string
	expiresIn   int
	expiredAt   string
	issuedAt    time.Time
}

// New creates a KIS OpenAPI client.
//
// App key and app secret are required. An access token may be supplied with
// WithAccessToken, or issued later with Client.Token.
func New(options ...Option) (*Client, error) {
	cfg := defaultConfig()
	errb := oops.In("kis_client").With("provider", ProviderKIS)
	for _, option := range options {
		if option == nil {
			return nil, errb.New("kis client config: option is required")
		}
		if err := option(&cfg); err != nil {
			return nil, errb.Wrapf(err, "apply kis client option")
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	restyClient := resty.NewWithClient(cfg.httpClient).
		SetBaseURL(cfg.selectedBaseURL()).
		SetHeader("accept", "application/json").
		SetHeader("content-type", "application/json; charset=utf-8")

	return &Client{
		appKey:       cfg.appKey,
		appSecret:    cfg.appSecret,
		customerType: cfg.customerType,
		account:      cfg.account,
		virtual:      cfg.virtual,
		http:         restyClient,
		token: tokenState{
			accessToken: cfg.accessToken,
		},
	}, nil
}

func (c *Client) request(ctx context.Context, group string, operation string, trID string, errb oops.OopsErrorBuilder) (*resty.Request, error) {
	token := c.currentAccessToken()
	if token == "" {
		return nil, errb.New("kis request: provider=" + ProviderKIS + " group=" + group + " operation=" + operation + " tr_id=" + trID + " access token is required")
	}
	return c.http.R().
		SetContext(ctx).
		SetHeader("authorization", bearer(token)).
		SetHeader("appkey", c.appKey).
		SetHeader("appsecret", c.appSecret).
		SetHeader("tr_id", trID).
		SetHeader("custtype", c.customerType), nil
}

func (c *Client) currentAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.token.accessToken
}

func (c *Client) setAccessToken(accessToken string) {
	c.setToken(Token{AccessToken: accessToken}, time.Time{})
}

func (c *Client) setToken(token Token, issuedAt time.Time) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.token = tokenState{
		accessToken: strings.TrimSpace(token.AccessToken),
		tokenType:   strings.TrimSpace(token.TokenType),
		expiresIn:   token.ExpiresIn,
		expiredAt:   strings.TrimSpace(token.ExpiredAt),
		issuedAt:    issuedAt,
	}
}

// UseToken installs an already-issued OAuth token on this client instance.
func (c *Client) UseToken(token Token) {
	c.setToken(token, time.Time{})
}

func bearer(accessToken string) string {
	if strings.HasPrefix(strings.ToLower(accessToken), "bearer ") {
		return accessToken
	}
	return "Bearer " + accessToken
}

func checkHTTP(response *resty.Response, errb oops.OopsErrorBuilder, group string, operation string, trID string) error {
	if response == nil {
		return errb.New("kis response is required")
	}
	if response.IsError() {
		return errb.With("status", response.StatusCode()).Errorf(
			"kis HTTP request failed: provider=%s group=%s operation=%s tr_id=%s status=%d body=%s",
			ProviderKIS,
			group,
			operation,
			trID,
			response.StatusCode(),
			string(response.Body()),
		)
	}
	return nil
}

func checkKIS(response responseFields, errb oops.OopsErrorBuilder, group string, operation string, trID string) error {
	if response.RTCD == "" && response.MsgCD == "" && response.Msg1 == "" {
		return errb.New("kis response envelope missing status fields")
	}
	if response.RTCD != "0" {
		return errb.With(
			"rt_cd", response.RTCD,
			"msg_cd", response.MsgCD,
			"msg1", response.Msg1,
		).Errorf(
			"kis business error: provider=%s group=%s operation=%s tr_id=%s rt_cd=%s msg_cd=%s msg1=%s",
			ProviderKIS,
			group,
			operation,
			trID,
			response.RTCD,
			response.MsgCD,
			response.Msg1,
		)
	}
	return nil
}

type responseFields struct {
	RTCD  string `json:"rt_cd"`
	MsgCD string `json:"msg_cd"`
	Msg1  string `json:"msg1"`
}
