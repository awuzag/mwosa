package krx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func assertKRXAPICall(
	t *testing.T,
	path string,
	body string,
	expectedField string,
	call func(context.Context, *Client) (any, error),
) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertKRXRequest(t, r, path)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := call(context.Background(), client)
	require.NoError(t, err)
	encoded, err := json.Marshal(rows)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), expectedField+"-value")
}
