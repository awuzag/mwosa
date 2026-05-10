package krx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := New(
		WithAuthKey("test-auth-key"),
		WithBaseURL(baseURL),
	)
	require.NoError(t, err)
	return client
}

func assertKRXRequest(t *testing.T, r *http.Request, path string) {
	t.Helper()
	require.Equal(t, path, r.URL.Path)
	require.Equal(t, http.MethodGet, r.Method)
	assert.Equal(t, "test-auth-key", r.Header.Get("AUTH_KEY"))
	assert.Equal(t, "application/json", r.Header.Get("accept"))
	assert.Equal(t, "20250131", r.URL.Query().Get("basDd"))
}
