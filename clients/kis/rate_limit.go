package kis

import (
	"context"
	"errors"
	"strings"
)

const RateLimitMsgCD = "EGW00201"

// Limiter decides whether a KIS request may be executed immediately.
//
// Client-level limiters must not sleep. Provider adapters that want waiting or
// retry behavior should do that before calling the client.
type Limiter interface {
	Allow(context.Context, RateLimitRequest) error
}

type RateLimitRequest struct {
	Provider  string
	Group     string
	Operation string
	TRID      string
	Endpoint  string
	Limit     string
}

type RateLimitError struct {
	Request RateLimitRequest
	Code    string
	Message string
}

func (e *RateLimitError) Error() string {
	if e == nil {
		return "kis rate limit exceeded"
	}
	parts := []string{"kis rate limit exceeded:"}
	if e.Request.Provider != "" {
		parts = append(parts, "provider="+e.Request.Provider)
	}
	if e.Request.Group != "" {
		parts = append(parts, "group="+e.Request.Group)
	}
	if e.Request.Operation != "" {
		parts = append(parts, "operation="+e.Request.Operation)
	}
	if e.Request.TRID != "" {
		parts = append(parts, "tr_id="+e.Request.TRID)
	}
	if e.Request.Endpoint != "" {
		parts = append(parts, "endpoint="+e.Request.Endpoint)
	}
	if e.Request.Limit != "" {
		parts = append(parts, "limit="+e.Request.Limit)
	}
	if e.Code != "" {
		parts = append(parts, "msg_cd="+e.Code)
	}
	if e.Message != "" {
		parts = append(parts, "msg1="+e.Message)
	}
	return strings.Join(parts, " ")
}

func RateLimitExceeded(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

func newRateLimitRequest(group string, operation string, trID string, endpoint string) RateLimitRequest {
	return RateLimitRequest{
		Provider:  ProviderKIS,
		Group:     strings.TrimSpace(group),
		Operation: strings.TrimSpace(operation),
		TRID:      strings.TrimSpace(trID),
		Endpoint:  strings.TrimSpace(endpoint),
	}
}

func kisRateLimitError(request RateLimitRequest, code string, message string) *RateLimitError {
	if request.Provider == "" {
		request.Provider = ProviderKIS
	}
	return &RateLimitError{
		Request: request,
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	}
}
