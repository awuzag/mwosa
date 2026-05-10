package kis

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, baseURL string, accessToken string) *Client {
	t.Helper()
	options := []Option{
		WithAppKey("test-app-key"),
		WithAppSecret("test-app-secret"),
		WithBaseURL(baseURL),
	}
	if accessToken != "" {
		options = append(options, WithAccessToken(accessToken))
	}
	client, err := New(options...)
	require.NoError(t, err)
	return client
}

func assertKISHeaders(t *testing.T, r *http.Request, trID string) {
	t.Helper()
	assert.Equal(t, "Bearer token", r.Header.Get("authorization"))
	assert.Equal(t, "test-app-key", r.Header.Get("appkey"))
	assert.Equal(t, "test-app-secret", r.Header.Get("appsecret"))
	assert.Equal(t, trID, r.Header.Get("tr_id"))
	assert.Equal(t, "P", r.Header.Get("custtype"))
}
