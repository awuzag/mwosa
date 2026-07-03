package instrument

import (
	"context"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	instrumentrole "github.com/awuzag/mwosa/providers/core/instrument"
)

func TestAssignAliasesGeneratesStockAndETFAliases(t *testing.T) {
	instruments := []instrumentrole.Instrument{
		stockInstrument("005930", "삼성전자", "Samsung Electronics"),
		stockInstrument("000660", "SK하이닉스", "SK Hynix"),
		stockInstrument("035420", "NAVER", "NAVER"),
		etfInstrument("069500", "KODEX 200"),
		etfInstrument("229200", "KODEX 코스닥150"),
	}

	aliases, err := assignAliases(instruments)
	if err != nil {
		t.Fatalf("assign aliases: %v", err)
	}

	assertAlias(t, aliases, instruments[0], "SAMS")
	assertAlias(t, aliases, instruments[1], "SHKY")
	assertAlias(t, aliases, instruments[2], "NAVE")
	assertAlias(t, aliases, instruments[3], "K200")
	assertAlias(t, aliases, instruments[4], "KQ15")
}

func TestAssignAliasesHandlesShortNamesAndDuplicateCandidates(t *testing.T) {
	instruments := []instrumentrole.Instrument{
		stockInstrument("000001", "LG", "LG"),
		stockInstrument("000002", "LG", "LG"),
		stockInstrument("000003", "A A", "AA"),
	}

	aliases, err := assignAliases(instruments)
	if err != nil {
		t.Fatalf("assign aliases: %v", err)
	}

	seen := make(map[string]bool)
	for _, item := range instruments {
		alias := aliases[aliasInstrumentKey(item)]
		if len(alias) < 3 || len(alias) > 4 {
			t.Fatalf("alias %q length = %d, want 3-4", alias, len(alias))
		}
		if seen[alias] {
			t.Fatalf("duplicate alias %q in %#v", alias, aliases)
		}
		seen[alias] = true
	}
}

func TestAssignAliasesSkipsReservedAlias(t *testing.T) {
	item := stockInstrument("999999", "AAPL", "AAPL")

	aliases, err := assignAliases([]instrumentrole.Instrument{item})
	if err != nil {
		t.Fatalf("assign aliases: %v", err)
	}

	alias := aliases[aliasInstrumentKey(item)]
	if alias == "AAPL" {
		t.Fatalf("alias = %q, want reserved alias skipped", alias)
	}
	if reservedAliases[alias] {
		t.Fatalf("alias = %q, want non-reserved alias", alias)
	}
}

func TestSyncOverwritesExistingAlias(t *testing.T) {
	fetched := stockInstrument("005930", "삼성전자", "Samsung Electronics")
	fetched.Extensions[instrumentAliasExtensionKey] = "OLD"
	existing := stockInstrument("005930", "삼성전자", "Samsung Electronics")
	existing.Extensions[instrumentAliasExtensionKey] = "USER"

	searcher := instrumentrole.NewSearch(instrumentrole.Profile{}, func(_ context.Context, input instrumentrole.SearchInput) (instrumentrole.SearchResult, error) {
		return instrumentrole.SearchResult{
			Instruments: []instrumentrole.Instrument{fetched},
			Provider:    provider.Identity{ID: provider.ProviderKRX},
			Group:       provider.GroupKRXStockInstrument,
			Operations:  []provider.OperationID{provider.OperationStockIssueBaseInfo},
		}, nil
	})
	repository := &fakeInstrumentRepository{
		listResult: instrumentrole.SearchResult{Instruments: []instrumentrole.Instrument{existing}},
	}
	service, err := NewService(&fakeInstrumentRouter{searcher: searcher}, WithRepository(repository))
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Sync(context.Background(), SyncRequest{
		ProviderID:   provider.ProviderKRX,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		AsOf:         "2026-07-03",
	})
	if err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	if got := repository.upserted[0].Extensions[instrumentAliasExtensionKey]; got != "SAMS" {
		t.Fatalf("upserted alias = %q, want SAMS overwrite", got)
	}
}

func TestSyncWritesExistingInstrumentWhenAliasIsRebalanced(t *testing.T) {
	fetched := stockInstrument("000001", "Alpha", "Alpha")
	existing := stockInstrument("000002", "Alpha", "Alpha")
	existing.Extensions[instrumentAliasExtensionKey] = "ALPH"

	instruments, err := instrumentsWithAssignedAliases([]instrumentrole.Instrument{fetched}, []instrumentrole.Instrument{existing})
	if err != nil {
		t.Fatalf("assign aliases: %v", err)
	}
	if len(instruments) != 2 {
		t.Fatalf("assigned instruments = %d, want fetched and changed existing instruments", len(instruments))
	}

	aliases := make(map[string]string)
	for _, item := range instruments {
		aliases[aliasSymbol(item)] = item.Extensions[instrumentAliasExtensionKey]
	}
	if got := aliases["000001"]; got != "ALPH" {
		t.Fatalf("fetched alias = %q, want ALPH", got)
	}
	if got := aliases["000002"]; got == "" || got == "ALPH" {
		t.Fatalf("existing alias = %q, want rebalanced alias", got)
	}
}

func assertAlias(t *testing.T, aliases map[string]string, item instrumentrole.Instrument, want string) {
	t.Helper()
	if got := aliases[aliasInstrumentKey(item)]; got != want {
		t.Fatalf("alias for %s = %q, want %q", aliasSymbol(item), got, want)
	}
}

func stockInstrument(symbol string, name string, englishName string) instrumentrole.Instrument {
	return instrumentrole.Instrument{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXStockInstrument,
		Operation:    provider.OperationStockIssueBaseInfo,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		SecurityCode: symbol,
		ISIN:         "KR7" + symbol + "003",
		Name:         name,
		ExchangeCode: symbol,
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions: map[string]string{
			"issueName":        name,
			"issueEnglishName": englishName,
		},
	}
}

func etfInstrument(symbol string, name string) instrumentrole.Instrument {
	return instrumentrole.Instrument{
		Provider:     provider.ProviderKRX,
		Group:        provider.GroupKRXETPDailyTrade,
		Operation:    provider.OperationETFByddTrd,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		SecurityCode: symbol,
		ISIN:         "KR7" + symbol + "007",
		Name:         name,
		ExchangeCode: symbol,
		CountryCode:  "KR",
		Timezone:     "Asia/Seoul",
		Extensions: map[string]string{
			"issueName": name,
		},
	}
}
