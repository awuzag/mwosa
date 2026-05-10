package krx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorIncludesKRXContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid auth", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.ETF(context.Background(), "20250131")
	require.Error(t, err)
	for _, want := range []string{
		"provider=krx",
		"group=etp",
		"api_id=etf_bydd_trd",
		"status=401",
		"invalid auth",
	} {
		assert.Contains(t, err.Error(), want)
	}
}

func TestDecodeErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"OutBlock_1": [`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.ETF(context.Background(), "20250131")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request krx API")
}

func TestMissingOutBlockIsReturnedAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.ETF(context.Background(), "20250131")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OutBlock_1 is required")
}
