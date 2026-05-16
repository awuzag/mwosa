package kis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenPostsCredentialsAndStoresAccessToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/tokenP", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("content-type"))
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "issued-token",
			"token_type": "Bearer",
			"expires_in": 86400,
			"access_token_token_expired": "2026-05-11 09:00:00"
		}`))
	})
	mux.HandleFunc("/uapi/domestic-stock/v1/quotations/inquire-price", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer issued-token", r.Header.Get("authorization"))
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "0",
			"msg_cd": "MCA00000",
			"msg1": "ok",
			"output": {"stck_shrn_iscd": "005930", "stck_prpr": "70000"}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL, "")
	token, err := client.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "issued-token", token.AccessToken)

	price, err := client.Price(context.Background(), "005930")
	require.NoError(t, err)
	assert.Equal(t, "70000", price.Current)
}
