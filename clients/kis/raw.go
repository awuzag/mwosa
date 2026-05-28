package kis

import (
	"context"

	"github.com/ev3rlit/mwosa/clients/kis/internal/generated/rawapi"
)

type RawOperationMetadata = rawapi.OperationMetadata
type RawParameterMetadata = rawapi.ParameterMetadata

func RawOperations() []RawOperationMetadata {
	return rawapi.Operations()
}

func LookupRawOperation(operationID string) (RawOperationMetadata, bool) {
	return rawapi.LookupOperation(operationID)
}

func RawRequestTemplate(operationID string) (map[string]string, error) {
	return rawapi.RequestTemplate(operationID)
}

func (c *Client) RawOperations() []RawOperationMetadata {
	return RawOperations()
}

func (c *Client) LookupRawOperation(operationID string) (RawOperationMetadata, bool) {
	return LookupRawOperation(operationID)
}

func (c *Client) RawRequestTemplate(operationID string) (map[string]string, error) {
	return RawRequestTemplate(operationID)
}

func (c *Client) InvokeRaw(ctx context.Context, operationID string, input map[string]string) (any, error) {
	return rawapi.Invoke(ctx, c.rawAPIExecutor(), operationID, input)
}
