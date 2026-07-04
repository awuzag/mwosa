package core

import "context"

type RawFetchInput struct {
	OperationID OperationID
	Input       map[string]string
	Context     map[string]any
}

type RawFetchResult struct {
	Provider  ProviderID
	Group     GroupID
	Operation OperationID
	Endpoint  string
	Response  any
	RowCount  int
	BaseDate  string
}

type RawFetcher interface {
	FetchProviderRaw(context.Context, RawFetchInput) (RawFetchResult, error)
}
