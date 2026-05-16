package krx

import (
	"testing"

	krxclient "github.com/ev3rlit/mwosa/clients/krx"
	provider "github.com/ev3rlit/mwosa/providers/core"
)

func TestNormalizeStockIssueBaseInfoToCanonicalInstrument(t *testing.T) {
	got := normalizeStockIssue(krxclient.StockIssueBaseInfo{
		IssueCode:                "KR7005930003",
		IssueShortCode:           "005930",
		IssueName:                "삼성전자",
		IssueAbbreviation:        "삼성전자",
		IssueEnglishName:         "Samsung Electronics",
		ListingDate:              "19750611",
		MarketTypeName:           "KOSPI",
		SecurityGroupName:        "주권",
		SectionTypeName:          "일반",
		StockCertificateTypeName: "보통주",
		ParValue:                 "100",
		ListedShares:             "5969782550",
	}, provider.OperationStockIssueBaseInfo)

	if got.Provider != provider.ProviderKRX || got.Group != provider.GroupKRXStockInstrument || got.Operation != provider.OperationStockIssueBaseInfo {
		t.Fatalf("unexpected provenance: %+v", got)
	}
	if got.SecurityCode != "005930" || got.ISIN != "KR7005930003" || got.Name != "삼성전자" {
		t.Fatalf("unexpected identity: %+v", got)
	}
	for key, want := range map[string]string{
		"issueName":                "삼성전자",
		"issueEnglishName":         "Samsung Electronics",
		"listingDate":              "19750611",
		"marketTypeName":           "KOSPI",
		"securityGroupName":        "주권",
		"sectionTypeName":          "일반",
		"stockCertificateTypeName": "보통주",
		"parValue":                 "100",
		"listedShares":             "5969782550",
	} {
		if got.Extensions[key] != want {
			t.Fatalf("extension %s = %q, want %q", key, got.Extensions[key], want)
		}
	}
}

func TestMatchesStockIssueMatchesEnglishName(t *testing.T) {
	if !matchesStockIssue("KR7005930003", "005930", "삼성전자", "삼성전자", "Samsung Electronics", "samsung") {
		t.Fatal("matchesStockIssue should match English name")
	}
}
