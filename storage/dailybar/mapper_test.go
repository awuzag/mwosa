package dailybar

import (
	"testing"
)

func TestParseScaledDecimalAcceptsLeadingDecimalPoint(t *testing.T) {
	tests := []struct {
		name  string
		value string
		scale int
		want  int64
	}{
		{name: "positive", value: ".68", scale: 2, want: 68},
		{name: "positive with explicit sign", value: "+.01", scale: 2, want: 1},
		{name: "negative", value: "-.42", scale: 2, want: -42},
		{name: "pads fractional digit", value: ".1", scale: 2, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScaledDecimal(tt.value, tt.scale)
			if err != nil {
				t.Fatalf("parseScaledDecimal(%q, %d) error = %v", tt.value, tt.scale, err)
			}
			if !got.Valid || got.Int64 != tt.want {
				t.Fatalf("parseScaledDecimal(%q, %d) = %+v, want %d", tt.value, tt.scale, got, tt.want)
			}
		})
	}
}

func TestParseScaledDecimalRejectsBareDecimalPoint(t *testing.T) {
	if _, err := parseScaledDecimal(".", 2); err == nil {
		t.Fatal("parseScaledDecimal bare decimal point error = nil")
	}
}
