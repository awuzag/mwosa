package quote

import (
	"context"
	"strings"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	quoterole "github.com/awuzag/mwosa/providers/core/quote"
)

func TestNewServiceRequiresRouter(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatal("NewService error = nil, want router error")
	}
	if !strings.Contains(err.Error(), "router") {
		t.Fatalf("error = %q, want router context", err.Error())
	}
}

func TestGetRoutesAndFetchesQuoteSnapshot(t *testing.T) {
	var gotSnapshot quoterole.SnapshotInput
	snapshotter := quoterole.NewSnapshot(quoterole.Profile{}, func(_ context.Context, input quoterole.SnapshotInput) (quoterole.SnapshotResult, error) {
		gotSnapshot = input
		return quoterole.SnapshotResult{
			Provider: provider.Identity{ID: provider.ProviderID("fake")},
			Symbol:   input.Symbol,
			Price:    "1000",
		}, nil
	})
	router := &fakeQuoteRouter{snapshotter: snapshotter}
	service, err := NewService(router)
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Get(context.Background(), Request{
		ProviderID:   provider.ProviderID("fake"),
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}

	if router.gotRoute.ProviderID != provider.ProviderID("fake") || router.gotRoute.Symbol != "069500" {
		t.Fatalf("route input = %+v, want provider and symbol", router.gotRoute)
	}
	if gotSnapshot.Symbol != "069500" || gotSnapshot.SecurityType != provider.SecurityTypeETF {
		t.Fatalf("snapshot input = %+v, want symbol and security type", gotSnapshot)
	}
	if result.Price != "1000" {
		t.Fatalf("price = %q, want 1000", result.Price)
	}
}

func TestEnsureUsesQuoteSnapshotPath(t *testing.T) {
	snapshotter := quoterole.NewSnapshot(quoterole.Profile{}, func(_ context.Context, input quoterole.SnapshotInput) (quoterole.SnapshotResult, error) {
		return quoterole.SnapshotResult{
			Symbol: input.Symbol,
			Price:  "1000",
		}, nil
	})
	service, err := NewService(&fakeQuoteRouter{snapshotter: snapshotter})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	result, err := service.Ensure(context.Background(), Request{Symbol: "069500"})
	if err != nil {
		t.Fatalf("Ensure error = %v", err)
	}
	if result.Symbol != "069500" {
		t.Fatalf("symbol = %q, want 069500", result.Symbol)
	}
}

func TestGetRequiresSymbol(t *testing.T) {
	service, err := NewService(&fakeQuoteRouter{})
	if err != nil {
		t.Fatalf("NewService error = %v", err)
	}

	_, err = service.Get(context.Background(), Request{})
	if err == nil {
		t.Fatal("Get error = nil, want symbol error")
	}
	if !strings.Contains(err.Error(), "requires symbol") {
		t.Fatalf("error = %q, want symbol context", err.Error())
	}
}

type fakeQuoteRouter struct {
	snapshotter quoterole.Snapshotter
	gotRoute    quoterole.RouteInput
}

func (r *fakeQuoteRouter) RouteQuoteSnapshot(_ context.Context, input quoterole.RouteInput) (quoterole.Snapshotter, error) {
	r.gotRoute = input
	return r.snapshotter, nil
}
