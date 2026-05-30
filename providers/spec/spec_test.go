package spec

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/awuzag/mwosa/providers/core/instrument"
)

func TestDailyBarBuilderBuildsProfile(t *testing.T) {
	profile, err := DailyBar().
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF, provider.SecurityTypeETN).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo, provider.OperationGetETNPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		RangeQuery(dailybar.RangeQuerySupported).
		Freshness(provider.FreshnessDaily).
		Compatibility(PreviousBusinessDay().
			LagBusinessDays(1).
			NoCurrentDay().
			Notes("D-1 business day EOD")).
		Priority(50).
		Limitations("not realtime").
		Build()
	if err != nil {
		t.Fatalf("build profile: %v", err)
	}

	if profile.RoleProfile().Role != provider.RoleDailyBar {
		t.Fatalf("role = %s, want %s", profile.RoleProfile().Role, provider.RoleDailyBar)
	}
	if profile.RangeQuery != dailybar.RangeQuerySupported {
		t.Fatalf("range query = %s, want %s", profile.RangeQuery, dailybar.RangeQuerySupported)
	}
	if profile.Compatibility.DataLatency != provider.DataLatencyPreviousBusinessDay {
		t.Fatalf("data latency = %s, want %s", profile.Compatibility.DataLatency, provider.DataLatencyPreviousBusinessDay)
	}
	if profile.Compatibility.CurrentDaySupported {
		t.Fatal("current day supported = true, want false")
	}
}

func TestPreviousBusinessDayDailyBarBuildsExecutableRole(t *testing.T) {
	role, err := PreviousBusinessDayDailyBar(func(context.Context, dailybar.FetchInput) (dailybar.FetchResult, error) {
		return dailybar.FetchResult{TotalCount: 1}, nil
	}).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		RangeQuery(dailybar.RangeQuerySupported).
		CompatibilityNotes("D-1 business day EOD").
		Build()
	if err != nil {
		t.Fatalf("build role: %v", err)
	}

	profile := role.DailyBarProfile()
	if profile.Compatibility.DataLatency != provider.DataLatencyPreviousBusinessDay {
		t.Fatalf("data latency = %s, want %s", profile.Compatibility.DataLatency, provider.DataLatencyPreviousBusinessDay)
	}
	if profile.Freshness != provider.FreshnessDaily {
		t.Fatalf("freshness = %s, want %s", profile.Freshness, provider.FreshnessDaily)
	}
	result, err := role.FetchDailyBars(context.Background(), dailybar.FetchInput{})
	if err != nil {
		t.Fatalf("fetch daily bars: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("total count = %d, want 1", result.TotalCount)
	}
}

func TestPreviousBusinessDayIndexBarBuildsExecutableRoleWithoutSecurityType(t *testing.T) {
	role, err := PreviousBusinessDayIndexBar(func(context.Context, indexbar.FetchInput) (indexbar.FetchResult, error) {
		return indexbar.FetchResult{TotalCount: 1}, nil
	}).
		Markets(provider.MarketKRX).
		Group(provider.GroupKRXIndexDailyTrade).
		Operations(provider.OperationKOSPIDDTrd).
		RequiresAuth(provider.CredentialScopeKRX).
		RangeQuery(indexbar.RangeQuerySupported).
		Build()
	if err != nil {
		t.Fatalf("build role: %v", err)
	}

	profile := role.IndexBarProfile()
	if profile.RoleProfile().Role != provider.RoleIndexBar {
		t.Fatalf("role = %s, want %s", profile.RoleProfile().Role, provider.RoleIndexBar)
	}
	if len(profile.RoleProfile().SecurityTypes) != 0 {
		t.Fatalf("security types = %v, want empty", profile.RoleProfile().SecurityTypes)
	}
	result, err := role.FetchIndexBars(context.Background(), indexbar.FetchInput{})
	if err != nil {
		t.Fatalf("fetch index bars: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("total count = %d, want 1", result.TotalCount)
	}
}

func TestPreviousBusinessDayInstrumentSearchBuildsExecutableRole(t *testing.T) {
	role, err := PreviousBusinessDayInstrumentSearch(func(context.Context, instrument.SearchInput) (instrument.SearchResult, error) {
		return instrument.SearchResult{TotalCount: 1}, nil
	}).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		Build()
	if err != nil {
		t.Fatalf("build role: %v", err)
	}

	profile := role.InstrumentSearchProfile()
	if profile.Compatibility.DataLatency != provider.DataLatencyPreviousBusinessDay {
		t.Fatalf("data latency = %s, want %s", profile.Compatibility.DataLatency, provider.DataLatencyPreviousBusinessDay)
	}
	result, err := role.SearchInstruments(context.Background(), instrument.SearchInput{})
	if err != nil {
		t.Fatalf("search instruments: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("total count = %d, want 1", result.TotalCount)
	}
}

func TestHistoricalFinancialsBuildsExecutableRole(t *testing.T) {
	role, err := HistoricalFinancials(func(context.Context, financials.FetchInput) (financials.FetchResult, error) {
		return financials.FetchResult{TotalCount: 1}, nil
	}).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeStock).
		Group(provider.GroupID("dartFinancialStatements")).
		Operations(provider.OperationID("getFinancialStatements")).
		RequiresAuth(provider.CredentialScope("dart")).
		Build()
	if err != nil {
		t.Fatalf("build role: %v", err)
	}

	profile := role.FinancialsProfile()
	if profile.RoleProfile().Role != provider.RoleFinancials {
		t.Fatalf("role = %s, want %s", profile.RoleProfile().Role, provider.RoleFinancials)
	}
	if profile.Freshness != provider.FreshnessFiling {
		t.Fatalf("freshness = %s, want %s", profile.Freshness, provider.FreshnessFiling)
	}
	result, err := role.FetchFinancialStatements(context.Background(), financials.FetchInput{})
	if err != nil {
		t.Fatalf("fetch financials: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("total count = %d, want 1", result.TotalCount)
	}
}

func TestDailyBarRoleBuilderRequiresCallable(t *testing.T) {
	_, err := PreviousBusinessDayDailyBar(nil).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		RangeQuery(dailybar.RangeQuerySupported).
		Build()
	if err == nil {
		t.Fatal("build error = nil, want callable error")
	}
	if !strings.Contains(err.Error(), "callable") {
		t.Fatalf("error = %q, want callable context", err.Error())
	}
}

func TestRoleBuilderRequiresCompatibility(t *testing.T) {
	_, err := DailyBar().
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		Freshness(provider.FreshnessDaily).
		Build()
	if err == nil {
		t.Fatal("build error = nil, want compatibility error")
	}
	if !strings.Contains(err.Error(), "data latency") {
		t.Fatalf("error = %q, want data latency context", err.Error())
	}
}

func TestDailyBarBuilderRequiresRangeQuery(t *testing.T) {
	_, err := DailyBar().
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeETF).
		Group(provider.GroupSecuritiesProductPrice).
		Operations(provider.OperationGetETFPriceInfo).
		RequiresAuth(provider.CredentialScopeDataGo).
		Freshness(provider.FreshnessDaily).
		Compatibility(PreviousBusinessDay()).
		Build()
	if err == nil {
		t.Fatal("build error = nil, want range query error")
	}
	if !strings.Contains(err.Error(), "range query") {
		t.Fatalf("error = %q, want range query context", err.Error())
	}
}

func TestPreviousBusinessDayRequiresPositiveLag(t *testing.T) {
	_, err := PreviousBusinessDay().LagBusinessDays(0).BuildCompatibility()
	if err == nil {
		t.Fatal("build compatibility error = nil, want lag error")
	}
	if !strings.Contains(err.Error(), "positive business-day lag") {
		t.Fatalf("error = %q, want lag context", err.Error())
	}
}

func TestRealtimeRequiresCurrentDay(t *testing.T) {
	_, err := Realtime().NoCurrentDay().BuildCompatibility()
	if err == nil {
		t.Fatal("build compatibility error = nil, want current-day error")
	}
	if !strings.Contains(err.Error(), "current trading-day") {
		t.Fatalf("error = %q, want current-day context", err.Error())
	}
}
