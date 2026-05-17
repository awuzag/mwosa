package opendart

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	opendartsdk "github.com/ev3rlit/opendart"
	"github.com/samber/oops"
)

type FilingDocumentRequest struct {
	ReceiptNo string
}

type FilingDocument struct {
	Provider      provider.ProviderID  `json:"provider" csv:"provider"`
	Group         provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation     provider.OperationID `json:"operation" csv:"operation"`
	ReceiptNo     string               `json:"rcept_no" csv:"rcept_no"`
	ContentType   string               `json:"content_type" csv:"content_type"`
	SizeBytes     int                  `json:"size_bytes" csv:"size_bytes"`
	SHA256        string               `json:"sha256" csv:"sha256"`
	PayloadBase64 string               `json:"payload_base64,omitempty" csv:"-"`
}

func (p *Provider) FetchFilingDocument(ctx context.Context, req FilingDocumentRequest) (FilingDocument, error) {
	receiptNo := strings.TrimSpace(req.ReceiptNo)
	errb := oops.In("opendart_adapter").
		With("provider", provider.ProviderOpenDART, "operation", provider.OperationOpenDARTDocumentRaw, "rcept_no", receiptNo)
	if receiptNo == "" {
		return FilingDocument{}, errb.New("OpenDART filing document requires rcept_no")
	}
	if err := p.requireClient(); err != nil {
		return FilingDocument{}, errb.Wrap(err)
	}
	response, err := p.client.DocumentRaw(ctx, opendartsdk.DocumentParams{RceptNo: receiptNo})
	if err != nil {
		return FilingDocument{}, errb.Wrapf(err, "fetch OpenDART filing document")
	}
	if response == nil {
		return FilingDocument{}, errb.New("OpenDART filing document response is nil")
	}
	sum := sha256.Sum256(response.Body)
	return FilingDocument{
		Provider:      provider.ProviderOpenDART,
		Group:         provider.GroupOpenDARTDisclosure,
		Operation:     provider.OperationOpenDARTDocumentRaw,
		ReceiptNo:     receiptNo,
		ContentType:   response.ContentType,
		SizeBytes:     len(response.Body),
		SHA256:        hex.EncodeToString(sum[:]),
		PayloadBase64: base64.StdEncoding.EncodeToString(response.Body),
	}, nil
}
