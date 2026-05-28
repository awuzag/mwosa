package kis

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRateLimiterFailsWithoutHTTPRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	limiter := fakeLimiter{
		err: &RateLimitError{
			Request: RateLimitRequest{
				Provider:  ProviderKIS,
				Group:     GroupQuote,
				Operation: "inquire-price",
				TRID:      trIDDomesticStockPrice,
				Endpoint:  "/uapi/domestic-stock/v1/quotations/inquire-price",
				Limit:     "1 rps",
			},
		},
	}
	client := newTestClient(t, server.URL, "token", WithRateLimiter(limiter))

	_, err := client.Quote().Price(context.Background(), InquirePriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
	})

	require.Error(t, err)
	require.True(t, RateLimitExceeded(err))
	assert.Equal(t, 0, requests)
	assert.Contains(t, err.Error(), "operation=inquire-price")
	assert.Contains(t, err.Error(), "tr_id=FHKST01010100")
	assert.Contains(t, err.Error(), "endpoint=/uapi/domestic-stock/v1/quotations/inquire-price")
	assert.Contains(t, err.Error(), "limit=1 rps")
}

func TestKISRateLimitBusinessErrorIsTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"rt_cd": "1",
			"msg_cd": "EGW00201",
			"msg1": "초당 거래건수를 초과하였습니다.",
			"output": {}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, "token")
	_, err := client.Quote().Price(context.Background(), InquirePriceRequest{
		FidCondMrktDivCode: "J",
		FidInputISCD:       "005930",
	})

	require.Error(t, err)
	require.True(t, RateLimitExceeded(err))
	var rateLimitErr *RateLimitError
	require.True(t, errors.As(err, &rateLimitErr))
	assert.Equal(t, RateLimitMsgCD, rateLimitErr.Code)
	assert.Equal(t, "inquire-price", rateLimitErr.Request.Operation)
	assert.Equal(t, "FHKST01010100", rateLimitErr.Request.TRID)
	assert.Contains(t, rateLimitErr.Message, "초당 거래건수")
}

type fakeLimiter struct {
	err error
}

func (l fakeLimiter) Allow(context.Context, RateLimitRequest) error {
	return l.err
}
