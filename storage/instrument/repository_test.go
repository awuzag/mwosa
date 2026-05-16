package instrument

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coreinstrument "github.com/ev3rlit/mwosa/providers/core/instrument"
	instrumentservice "github.com/ev3rlit/mwosa/service/instrument"
	"github.com/ev3rlit/mwosa/storage"
)

func TestRepositoryUpsertSearchAndInspectInstrumentMaster(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	item := samsungInstrument("Samsung Electronics")
	if _, err := repository.UpsertInstruments(context.Background(), []coreinstrument.Instrument{item}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	item.Extensions["issueEnglishName"] = "Samsung Electronics Co Ltd"
	delete(item.Extensions, "parValue")
	if _, err := repository.UpsertInstruments(context.Background(), []coreinstrument.Instrument{item}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	assertInstrumentRowCounts(t, database, 1, 1, 8)

	result, err := repository.SearchInstruments(context.Background(), instrumentservice.Query{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Query:        "Samsung",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Instruments) != 1 {
		t.Fatalf("search len = %d, want 1", len(result.Instruments))
	}
	if got := result.Instruments[0]; got.SecurityCode != "005930" || got.Extensions["issueEnglishName"] != "Samsung Electronics Co Ltd" {
		t.Fatalf("unexpected search result: %+v", got)
	}

	inspected, err := repository.InspectInstrument(context.Background(), instrumentservice.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "KR7005930003",
	})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspected.SecurityCode != "005930" || inspected.ISIN != "KR7005930003" {
		t.Fatalf("unexpected inspect result: %+v", inspected)
	}
}

func TestRepositorySearchMatchesKoreanNameShortCodeAndISIN(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	if _, err := repository.UpsertInstruments(context.Background(), []coreinstrument.Instrument{samsungInstrument("Samsung")}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	for _, query := range []string{"삼성", "005930", "KR7005930003"} {
		result, err := repository.SearchInstruments(context.Background(), instrumentservice.Query{
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			Query:        query,
		})
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if len(result.Instruments) != 1 {
			t.Fatalf("search %q len = %d, want 1", query, len(result.Instruments))
		}
	}
}

func TestRepositoryValidatesRequiredInstrumentKey(t *testing.T) {
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	_, err = repository.UpsertInstruments(context.Background(), []coreinstrument.Instrument{{Provider: provider.ProviderKRX}})
	if err == nil {
		t.Fatal("upsert error = nil, want key validation error")
	}
	if !strings.Contains(err.Error(), "missing sqlite key") {
		t.Fatalf("error = %q, want key validation", err.Error())
	}
}

func TestSourceNaturalKeyIncludesProviderSymbol(t *testing.T) {
	item := samsungInstrument("Samsung")
	got := sourceNaturalKey(item)
	for _, want := range []string{"krx", "stockInstrument", "stk_isu_base_info", "005930"} {
		if !strings.Contains(got, want) {
			t.Fatalf("source key %q missing %q", got, want)
		}
	}
}

func samsungInstrument(englishName string) coreinstrument.Instrument {
	return coreinstrument.Instrument{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXStockInstrument,
		Operation:    provider.OperationStockIssueBaseInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		SecurityCode: "005930",
		ISIN:         "KR7005930003",
		Name:         "삼성전자",
		ExchangeCode: "005930",
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions: map[string]string{
			"issueName":                "삼성전자",
			"issueEnglishName":         englishName,
			"listingDate":              "19750611",
			"marketTypeName":           "KOSPI",
			"securityGroupName":        "주권",
			"sectionTypeName":          "일반",
			"stockCertificateTypeName": "보통주",
			"parValue":                 "100",
			"listedShares":             "5969782550",
		},
	}
}

func assertInstrumentRowCounts(t *testing.T, database *storage.Database, wantInstruments, wantSources, wantExtensions int) {
	t.Helper()
	client, err := database.Client(context.Background())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	counts := map[string]int{
		"instrument_v2":           0,
		"instrument_source_v1":    0,
		"instrument_extension_v1": 0,
	}
	for table := range counts {
		var got int
		if err := client.QueryRowContext(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = got
	}
	if counts["instrument_v2"] != wantInstruments || counts["instrument_source_v1"] != wantSources || counts["instrument_extension_v1"] != wantExtensions {
		t.Fatalf("counts = %+v, want instrument/source/extension = %d/%d/%d", counts, wantInstruments, wantSources, wantExtensions)
	}
}
