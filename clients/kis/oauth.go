package kis

import (
	"context"
	"strings"

	"github.com/samber/oops"
)

// Token is the OAuth token response returned by Client.Token.
type Token struct {
	// AccessToken is the bearer token used in authenticated KIS API requests.
	AccessToken string `json:"access_token"`

	// TokenType is usually "Bearer".
	TokenType string `json:"token_type"`

	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`

	// ExpiredAt is KIS's display-form expiration timestamp.
	ExpiredAt string `json:"access_token_token_expired"`
}

// Token issues an OAuth access token through /oauth2/tokenP.
//
// The method uses the configured app key and app secret with the
// client_credentials grant. On success, the returned access token is also stored
// on the client so later authenticated calls can use it immediately.
func (c *Client) Token(ctx context.Context) (Token, error) {
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"operation", OperationToken,
		"endpoint", "/oauth2/tokenP",
	)

	var token Token
	response, err := c.http.R().
		SetContext(ctx).
		SetBody(tokenRequest{
			GrantType: "client_credentials",
			AppKey:    c.appKey,
			AppSecret: c.appSecret,
		}).
		SetResult(&token).
		Post("/oauth2/tokenP")
	if err != nil {
		return Token{}, errb.Wrapf(err, "request kis OAuth token")
	}
	if response.IsError() {
		return Token{}, errb.With("status", response.StatusCode()).Errorf("kis OAuth token failed: status=%d body=%s", response.StatusCode(), string(response.Body()))
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return Token{}, errb.New("kis OAuth token response missing access token")
	}
	c.setAccessToken(token.AccessToken)
	return token, nil
}

type tokenRequest struct {
	GrantType string `json:"grant_type"`
	AppKey    string `json:"appkey"`
	AppSecret string `json:"appsecret"`
}
