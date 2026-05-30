package daily

import (
	"fmt"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
)

type NotFoundError struct {
	Symbol       string
	Market       provider.Market
	SecurityType provider.SecurityType
	From         string
	To           string
	Hint         string
}

func (e *NotFoundError) Error() string {
	parts := []string{"daily data not found"}
	if e.Market != "" {
		parts = append(parts, fmt.Sprintf("market=%s", e.Market))
	}
	if e.SecurityType != "" {
		parts = append(parts, fmt.Sprintf("security_type=%s", e.SecurityType))
	}
	if e.Symbol != "" {
		parts = append(parts, fmt.Sprintf("symbol=%s", e.Symbol))
	}
	if e.From != "" || e.To != "" {
		parts = append(parts, fmt.Sprintf("range=%s..%s", e.From, e.To))
	}
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf("hint=%s", e.Hint))
	}
	return strings.Join(parts, " ")
}
