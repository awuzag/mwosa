package kis

import (
	"context"
	"reflect"
	"strings"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type RawOperation struct {
	OperationID     provider.OperationID             `json:"operation_id"`
	Endpoint        string                           `json:"endpoint"`
	Method          string                           `json:"method"`
	Group           provider.GroupID                 `json:"provider_group"`
	ServiceGroup    string                           `json:"service_group"`
	RoleHint        string                           `json:"role_hint"`
	Summary         string                           `json:"summary"`
	Description     string                           `json:"description"`
	RealTRID        string                           `json:"real_tr_id"`
	VirtualTRID     string                           `json:"virtual_tr_id"`
	SupportsVirtual bool                             `json:"supports_virtual"`
	Parameters      []kisclient.RawParameterMetadata `json:"parameters"`
}

type RawRequest struct {
	OperationID provider.OperationID
	Input       map[string]string
}

type RawResult struct {
	Provider  provider.ProviderID  `json:"provider"`
	Group     provider.GroupID     `json:"provider_group"`
	Operation provider.OperationID `json:"operation"`
	Endpoint  string               `json:"endpoint"`
	Response  any                  `json:"response"`
	RowCount  int                  `json:"row_count"`
	Canonical string               `json:"canonical_support"`
	BaseDate  string               `json:"base_date"`
}

func (p *Provider) RawOperations() []RawOperation {
	if p == nil || p.client == nil {
		return nil
	}
	operations := p.client.RawOperations()
	out := make([]RawOperation, 0, len(operations))
	for _, operation := range operations {
		out = append(out, rawOperationFromMetadata(operation))
	}
	return out
}

func (p *Provider) LookupRawOperation(operationID provider.OperationID) (RawOperation, bool) {
	if p == nil || p.client == nil {
		return RawOperation{}, false
	}
	metadata, ok := p.client.LookupRawOperation(string(operationID))
	if !ok {
		return RawOperation{}, false
	}
	return rawOperationFromMetadata(metadata), true
}

func (p *Provider) RawRequestTemplate(operationID provider.OperationID) (map[string]string, error) {
	if p == nil || p.client == nil {
		return nil, oops.In("kis_adapter").With("provider", provider.ProviderKIS, "operation", operationID).New("kis provider client is nil")
	}
	return p.client.RawRequestTemplate(string(operationID))
}

func (p *Provider) FetchRaw(ctx context.Context, req RawRequest) (RawResult, error) {
	errb := oops.In("kis_adapter").With("provider", provider.ProviderKIS, "operation", req.OperationID)
	if p == nil {
		return RawResult{}, errb.New("kis provider is nil")
	}
	if p.client == nil {
		return RawResult{}, errb.New("kis provider client is nil")
	}
	metadata, ok := p.client.LookupRawOperation(string(req.OperationID))
	if !ok {
		return RawResult{}, errb.New("kis raw operation is not registered")
	}
	errb = errb.With("group", metadata.Group, "endpoint", metadata.Endpoint)
	if err := p.ensureAccessToken(ctx); err != nil {
		return RawResult{}, errb.Wrap(err)
	}
	response, err := withReadRetry(ctx, p, kisclient.RateLimitRequest{
		Provider:  kisclient.ProviderKIS,
		Group:     metadata.Group,
		Operation: string(req.OperationID),
		TRID:      firstNonEmpty(metadata.RealTRID, metadata.VirtualTRID),
		Endpoint:  metadata.Endpoint,
	}, func(ctx context.Context) (any, error) {
		return p.client.InvokeRaw(ctx, string(req.OperationID), req.Input)
	})
	if err != nil {
		return RawResult{}, errb.Wrapf(err, "fetch kis raw API")
	}
	return RawResult{
		Provider:  provider.ProviderKIS,
		Group:     provider.GroupID(metadata.Group),
		Operation: req.OperationID,
		Endpoint:  metadata.Endpoint,
		Response:  response,
		RowCount:  rawRowCount(response),
		Canonical: rawCanonicalSupport(metadata.RoleHint),
		BaseDate:  rawBaseDate(req.Input),
	}, nil
}

func rawOperationFromMetadata(metadata kisclient.RawOperationMetadata) RawOperation {
	return RawOperation{
		OperationID:     provider.OperationID(metadata.OperationID),
		Endpoint:        metadata.Endpoint,
		Method:          metadata.Method,
		Group:           provider.GroupID(metadata.Group),
		ServiceGroup:    metadata.ServiceGroup,
		RoleHint:        metadata.RoleHint,
		Summary:         metadata.Summary,
		Description:     metadata.Description,
		RealTRID:        metadata.RealTRID,
		VirtualTRID:     metadata.VirtualTRID,
		SupportsVirtual: metadata.SupportsVirtual,
		Parameters:      metadata.Parameters,
	}
}

func rawCanonicalSupport(roleHint string) string {
	switch strings.TrimSpace(roleHint) {
	case "", "read_only", "market_scan":
		return "raw_only"
	default:
		return roleHint
	}
}

func rawBaseDate(input map[string]string) string {
	for _, key := range []string{"FID_INPUT_DATE_2", "FID_INPUT_DATE", "FID_INPUT_DATE_1"} {
		if value := strings.TrimSpace(input[key]); value != "" {
			return formatRawDate(value)
		}
	}
	return time.Now().Format("2006-01-02")
}

func formatRawDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 {
		return value[:4] + "-" + value[4:6] + "-" + value[6:8]
	}
	return value
}

func rawRowCount(response any) int {
	if response == nil {
		return 0
	}
	value := reflect.ValueOf(response)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 1
	}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			return field.Len()
		}
	}
	return 1
}
