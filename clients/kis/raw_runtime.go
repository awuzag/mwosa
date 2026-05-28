package kis

import (
	"context"
	"strings"

	"github.com/ev3rlit/mwosa/clients/kis/internal/generated/rawapi"
	"github.com/go-resty/resty/v2"
	"github.com/samber/oops"
)

type rawAPIExecutor struct {
	appKey       string
	appSecret    string
	customerType string
	virtual      bool
	http         *resty.Client
	accessToken  func() string
	rateLimiter  Limiter
}

func (c *Client) rawAPIExecutor() rawAPIExecutor {
	if c == nil {
		return rawAPIExecutor{}
	}
	return rawAPIExecutor{
		appKey:       c.appKey,
		appSecret:    c.appSecret,
		customerType: c.customerType,
		virtual:      c.virtual,
		http:         c.http,
		accessToken:  c.currentAccessToken,
		rateLimiter:  c.rateLimiter,
	}
}

func (r rawAPIExecutor) ExecuteKIS(ctx context.Context, request rawapi.Request, result any) error {
	if r.http == nil || r.accessToken == nil {
		return oops.In("kis_client").New("kis raw API executor: client is required")
	}
	if result == nil {
		return oops.In("kis_client").New("kis raw API executor: result is required")
	}

	trID, err := r.rawTRID(request)
	if err != nil {
		return err
	}
	errb := oops.In("kis_client").With(
		"provider", ProviderKIS,
		"group", request.Group,
		"operation", request.Operation,
		"endpoint", request.Path,
		"tr_id", trID,
	)

	restyRequest, err := r.request(ctx, request.Group, request.Operation, trID, request.Path, errb)
	if err != nil {
		return err
	}
	response, err := restyRequest.
		SetQueryParams(request.Query).
		SetResult(result).
		Execute(strings.ToUpper(request.Method), request.Path)
	if err != nil {
		return errb.Wrapf(err, "request kis raw API")
	}
	if err := checkHTTP(response, errb, request.Group, request.Operation, trID); err != nil {
		return err
	}
	statusProvider, ok := result.(rawapi.StatusProvider)
	if !ok {
		return errb.New("kis raw response does not expose KIS status fields")
	}
	status := statusProvider.KISStatus()
	return checkKIS(responseFields{
		RTCD:  status.RTCD,
		MsgCD: status.MsgCD,
		Msg1:  status.Msg1,
	}, errb, newRateLimitRequest(request.Group, request.Operation, trID, request.Path))
}

func (r rawAPIExecutor) request(ctx context.Context, group string, operation string, trID string, endpoint string, errb oops.OopsErrorBuilder) (*resty.Request, error) {
	token := r.accessToken()
	if token == "" {
		return nil, errb.New("kis request: provider=" + ProviderKIS + " group=" + group + " operation=" + operation + " tr_id=" + trID + " access token is required")
	}
	if r.rateLimiter != nil {
		request := newRateLimitRequest(group, operation, trID, endpoint)
		if err := r.rateLimiter.Allow(ctx, request); err != nil {
			return nil, errb.With(
				"endpoint", request.Endpoint,
				"limit", request.Limit,
			).Wrapf(err, "check kis rate limit")
		}
	}
	return r.http.R().
		SetContext(ctx).
		SetHeader("authorization", bearer(token)).
		SetHeader("appkey", r.appKey).
		SetHeader("appsecret", r.appSecret).
		SetHeader("tr_id", trID).
		SetHeader("custtype", r.customerType), nil
}

func (r rawAPIExecutor) rawTRID(request rawapi.Request) (string, error) {
	if r.virtual {
		if unsupportedVirtualTRID(request.VirtualTRID) {
			return "", oops.In("kis_client").With(
				"provider", ProviderKIS,
				"group", request.Group,
				"operation", request.Operation,
			).New("kis raw request: virtual domain is not supported for this operation")
		}
		return request.VirtualTRID, nil
	}
	return request.RealTRID, nil
}

func unsupportedVirtualTRID(trID string) bool {
	trID = strings.TrimSpace(trID)
	return trID == "" || strings.Contains(trID, "모의투자 미지원")
}
