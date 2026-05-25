package kis

import (
	"context"
	"testing"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/composition"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/providers/core/instrument"
	"github.com/ev3rlit/mwosa/providers/core/intradaybar"
	"github.com/ev3rlit/mwosa/providers/core/orderbook"
	"github.com/ev3rlit/mwosa/providers/core/quote"
	tradesrole "github.com/ev3rlit/mwosa/providers/core/trades"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistersReadOnlyKISRoles(t *testing.T) {
	p := NewWithClient(&fakeKISClient{}, true)

	registrations := p.RoleRegistrations()

	require.Len(t, registrations, 8)
	roles := map[provider.Role]int{}
	for _, registration := range registrations {
		require.Equal(t, provider.ProviderKIS, p.ProviderIdentity().ID)
		require.True(t, registration.Profile.RequiresAuth)
		require.Equal(t, provider.CredentialScopeKIS, registration.Profile.AuthScope)
		roles[registration.Profile.Role]++
	}
	require.Equal(t, 2, roles[provider.RoleQuote])
	require.Equal(t, 1, roles[provider.RoleDailyBar])
	require.Equal(t, 1, roles[provider.RoleInstrument])
	require.Equal(t, 1, roles[provider.RoleIntradayBar])
	require.Equal(t, 1, roles[provider.RoleOrderbook])
	require.Equal(t, 1, roles[provider.RoleComposition])
	require.Equal(t, 1, roles[provider.RoleTrades])
}

func TestFetchStockQuoteUsesPrice(t *testing.T) {
	client := &fakeKISClient{
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, false)

	result, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tokenCalls)
	require.Equal(t, 1, client.priceCalls)
	require.Equal(t, "005930", result.Symbol)
	require.Equal(t, "75000", result.Price)
}

func TestFetchETFQuoteUsesETFETNPrice(t *testing.T) {
	client := &fakeKISClient{
		etfetnPrice: kisclient.ETFETNPrice{
			Current: "10250",
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})

	require.NoError(t, err)
	require.Equal(t, 0, client.tokenCalls)
	require.Equal(t, 1, client.etfetnCalls)
	require.Equal(t, "069500", result.Symbol)
	require.Equal(t, "10250", result.Price)
}

func TestFetchRawUsesGeneratedRegistryAndKeepsProviderNativeResponse(t *testing.T) {
	client := &fakeKISClient{
		rawResponse: kisclient.InquirePriceResponse{
			RtCd:  "0",
			MsgCd: "ok",
			Msg1:  "ok",
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.FetchRaw(context.Background(), RawRequest{
		OperationID: provider.OperationID("inquire-price"),
		Input: map[string]string{
			"FID_INPUT_ISCD": "005930",
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.rawCalls)
	require.Equal(t, provider.ProviderKIS, result.Provider)
	require.Equal(t, provider.GroupID("quote"), result.Group)
	require.Equal(t, provider.OperationID("inquire-price"), result.Operation)
	require.Equal(t, "quote_snapshot", result.Canonical)
	require.Equal(t, "005930", client.rawInput["FID_INPUT_ISCD"])
	require.IsType(t, kisclient.InquirePriceResponse{}, result.Response)
}

func TestFetchRawWaitsBeforeProviderNativeCall(t *testing.T) {
	client := &fakeKISClient{
		rawResponse: kisclient.InquirePriceResponse{RtCd: "0", MsgCd: "ok", Msg1: "ok"},
	}
	limiter := &fakeReadLimiter{}
	p := NewWithClient(client, true, WithReadRateLimiter(limiter))

	_, err := p.FetchRaw(context.Background(), RawRequest{
		OperationID: provider.OperationID("inquire-price"),
		Input:       map[string]string{"FID_INPUT_ISCD": "005930"},
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.rawCalls)
	require.Len(t, limiter.requests, 1)
	require.Equal(t, "inquire-price", limiter.requests[0].Operation)
	require.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-price", limiter.requests[0].Endpoint)
}

func TestFetchRawRetriesKISRateLimitError(t *testing.T) {
	client := &fakeKISClient{
		rawErrors: []error{
			&kisclient.RateLimitError{
				Request: kisclient.RateLimitRequest{
					Provider:  kisclient.ProviderKIS,
					Group:     "quote",
					Operation: "inquire-price",
					TRID:      "FHKST01010100",
					Endpoint:  "/uapi/domestic-stock/v1/quotations/inquire-price",
				},
				Code:    kisclient.RateLimitMsgCD,
				Message: "초당 거래건수를 초과하였습니다.",
			},
		},
		rawResponse: kisclient.InquirePriceResponse{RtCd: "0", MsgCd: "ok", Msg1: "ok"},
	}
	limiter := &fakeReadLimiter{}
	var sleeps []time.Duration
	p := NewWithClient(client, true,
		WithReadRateLimiter(limiter),
		WithRateLimitPolicy(RateLimitPolicy{ReadRPS: 15, Burst: 1, MaxAttempts: 2, BaseDelay: 10 * time.Millisecond, MaxDelay: 10 * time.Millisecond}),
		WithRateLimitSleeper(func(_ context.Context, d time.Duration) error {
			sleeps = append(sleeps, d)
			return nil
		}),
	)

	result, err := p.FetchRaw(context.Background(), RawRequest{
		OperationID: provider.OperationID("inquire-price"),
		Input:       map[string]string{"FID_INPUT_ISCD": "005930"},
	})

	require.NoError(t, err)
	require.Equal(t, 2, client.rawCalls)
	require.Len(t, limiter.requests, 2)
	require.Len(t, sleeps, 1)
	require.Equal(t, 10*time.Millisecond, sleeps[0])
	require.IsType(t, kisclient.InquirePriceResponse{}, result.Response)
}

func TestFetchRawStopsRetryWhenContextIsCancelled(t *testing.T) {
	client := &fakeKISClient{
		rawErrors: []error{
			&kisclient.RateLimitError{
				Request: kisclient.RateLimitRequest{Provider: kisclient.ProviderKIS, Operation: "inquire-price"},
				Code:    kisclient.RateLimitMsgCD,
			},
		},
	}
	cancelled := context.Canceled
	p := NewWithClient(client, true,
		WithReadRateLimiter(&fakeReadLimiter{}),
		WithRateLimitPolicy(RateLimitPolicy{ReadRPS: 15, Burst: 1, MaxAttempts: 2, BaseDelay: 10 * time.Millisecond, MaxDelay: 10 * time.Millisecond}),
		WithRateLimitSleeper(func(context.Context, time.Duration) error {
			return cancelled
		}),
	)

	_, err := p.FetchRaw(context.Background(), RawRequest{
		OperationID: provider.OperationID("inquire-price"),
		Input:       map[string]string{"FID_INPUT_ISCD": "005930"},
	})

	require.ErrorIs(t, err, cancelled)
	require.Equal(t, 1, client.rawCalls)
}

func TestListConstituentsReturnsOnlyCompositionMembers(t *testing.T) {
	client := &fakeKISClient{
		etfComponents: kisclient.ETFComponentStockPriceResult{
			Rows: []kisclient.ETFComponentStockPrice{
				{
					Symbol:             "005930",
					Name:               "삼성전자",
					Current:            "75000",
					PreviousChange:     "100",
					PreviousChangeRate: "0.13",
					Volume:             "123456",
					Weight:             "28.15",
					ValuationAmount:    "1942000000",
					Quantity:           "25893",
				},
			},
			Output1: map[string]string{"stck_prpr": "36090"},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.listConstituents(context.Background(), composition.ListInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeETF,
		Symbol:       "069500",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.etfComponentCalls)
	require.Equal(t, provider.OperationKISETFComponentStockPrice, result.Operation)
	require.Equal(t, "069500", result.Composition.Subject.Symbol)
	require.Len(t, result.Composition.Members, 1)
	require.Equal(t, "005930", result.Composition.Members[0].Instrument.Symbol)
	require.Equal(t, "삼성전자", result.Composition.Members[0].Instrument.Name)
	require.Equal(t, "28.15", result.Composition.Members[0].Weight.Value)
	require.Equal(t, "1942000000", result.Composition.Members[0].Valuation.Value)
}

func TestFetchQuoteUsesValidCachedTokenWithoutIssuingToken(t *testing.T) {
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	key := newTokenCacheKey("app-key", false)
	cache := newFakeTokenCache()
	cache.tokens[key] = CachedToken{
		Key:         key,
		AccessToken: "cached-token",
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		ExpiresAt:   now.Add(time.Hour),
		IssuedAt:    now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}
	client := &fakeKISClient{
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, false, WithTokenCache(cache, key), WithClock(func() time.Time { return now }))

	result, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 0, client.tokenCalls)
	require.Equal(t, 1, client.useTokenCalls)
	require.Equal(t, "cached-token", client.usedToken.AccessToken)
	require.Equal(t, 1, client.priceCalls)
	require.Equal(t, "75000", result.Price)
	require.Len(t, cache.puts, 0)
}

func TestFetchQuoteStoresTokenOnCacheMiss(t *testing.T) {
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	key := newTokenCacheKey("app-key", false)
	cache := newFakeTokenCache()
	client := &fakeKISClient{
		token: kisclient.Token{
			AccessToken: "issued-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
			ExpiredAt:   "2026-05-11 09:00:00",
		},
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, false, WithTokenCache(cache, key), WithClock(func() time.Time { return now }))

	_, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tokenCalls)
	require.Len(t, cache.puts, 1)
	require.Equal(t, "issued-token", cache.puts[0].AccessToken)
	require.Equal(t, key, cache.puts[0].Key)
}

func TestFetchQuoteRefreshesExpiredCachedToken(t *testing.T) {
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	key := newTokenCacheKey("app-key", false)
	cache := newFakeTokenCache()
	cache.tokens[key] = CachedToken{
		Key:         key,
		AccessToken: "expired-token",
		ExpiresAt:   now.Add(-time.Minute),
		IssuedAt:    now.Add(-25 * time.Hour),
		UpdatedAt:   now.Add(-25 * time.Hour),
	}
	client := &fakeKISClient{
		token: kisclient.Token{
			AccessToken: "issued-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
			ExpiredAt:   "2026-05-11 09:00:00",
		},
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, false, WithTokenCache(cache, key), WithClock(func() time.Time { return now }))

	_, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tokenCalls)
	require.Equal(t, 0, client.useTokenCalls)
	require.Len(t, cache.puts, 1)
	require.Equal(t, "issued-token", cache.puts[0].AccessToken)
}

func TestFetchQuoteTreatsDifferentEnvironmentAsCacheMiss(t *testing.T) {
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	realKey := newTokenCacheKey("app-key", false)
	virtualKey := newTokenCacheKey("app-key", true)
	cache := newFakeTokenCache()
	cache.tokens[realKey] = CachedToken{
		Key:         realKey,
		AccessToken: "real-token",
		ExpiresAt:   now.Add(time.Hour),
		IssuedAt:    now,
		UpdatedAt:   now,
	}
	client := &fakeKISClient{
		token: kisclient.Token{
			AccessToken: "virtual-token",
			ExpiresIn:   86400,
			ExpiredAt:   "2026-05-11 09:00:00",
		},
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, false, WithTokenCache(cache, virtualKey), WithClock(func() time.Time { return now }))

	_, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tokenCalls)
	require.Equal(t, 0, client.useTokenCalls)
	require.Equal(t, virtualKey, cache.gets[0])
	require.Equal(t, virtualKey, cache.puts[0].Key)
}

func TestExplicitAccessTokenSkipsTokenCache(t *testing.T) {
	key := newTokenCacheKey("app-key", false)
	cache := newFakeTokenCache()
	client := &fakeKISClient{
		price: kisclient.InquirePriceResponse{
			Output: kisclient.InquirePriceOutput{
				StckShrnISCD: "005930",
				StckPrpr:     "75000",
			},
		},
	}
	p := NewWithClient(client, true, WithTokenCache(cache, key))

	_, err := p.fetchQuoteSnapshot(context.Background(), quote.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 0, client.tokenCalls)
	require.Empty(t, cache.gets)
	require.Empty(t, cache.puts)
}

func TestFetchDailyBarsNormalizesKISBars(t *testing.T) {
	client := &fakeKISClient{
		bars: []kisclient.InquireDailyItemChartPriceOutput2Item{
			{
				StckBsopDate: "20260508",
				StckOprc:     "70000",
				StckHgpr:     "76000",
				StckLwpr:     "69000",
				StckClpr:     "75000",
				PrdyVrss:     "1000",
				PrdyVrssSign: "2",
				AcmlVol:      "12345",
				AcmlTRPbmn:   "98765",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchDailyBars(context.Background(), dailybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		From:         "2026-05-08",
		To:           "2026-05-08",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.dailyCalls)
	require.Len(t, result.Bars, 1)
	bar := result.Bars[0]
	require.Equal(t, provider.ProviderKIS, bar.Provider)
	require.Equal(t, provider.GroupKISQuote, bar.Group)
	require.Equal(t, provider.OperationKISDaily, bar.Operation)
	require.Equal(t, "2026-05-08", bar.TradingDate)
	require.Equal(t, "005930", bar.Symbol)
	require.Equal(t, "75000", bar.Close)
	require.Equal(t, "1000", bar.Change)
}

func TestFetchIntradayBarsNormalizesKISBars(t *testing.T) {
	client := &fakeKISClient{
		intradayBars: []kisclient.InquireTimeItemChartPriceOutput2Item{
			{
				StckBsopDate: "20260508",
				StckCntgHour: "141200",
				StckOprc:     "70000",
				StckHgpr:     "76000",
				StckLwpr:     "69000",
				StckPrpr:     "75000",
				CntgVol:      "123",
				AcmlTRPbmn:   "9225000",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchIntradayBars(context.Background(), intradaybar.FetchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		At:           "14:12:00",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.intradayCalls)
	require.Len(t, result.Bars, 1)
	bar := result.Bars[0]
	require.Equal(t, provider.OperationKISIntraday, bar.Operation)
	require.Equal(t, "2026-05-08", bar.TradingDate)
	require.Equal(t, "14:12:00", bar.Time)
	require.Equal(t, "75000", bar.Close)
	require.Equal(t, "123", bar.Volume)
}

func TestFetchOrderbookSnapshotNormalizesLevels(t *testing.T) {
	client := &fakeKISClient{
		orderbook: kisclient.InquireAskingPriceExpCcnResponse{
			Output1: kisclient.InquireAskingPriceExpCcnOutput1{
				AsprAcptHour:  "141200",
				Askp1:         "75100",
				AskpRsqn1:     "10",
				AskpRsqnIcdc1: "1",
				Bidp1:         "75000",
				BidpRsqn1:     "20",
				BidpRsqnIcdc1: "-2",
				TotalAskpRsqn: "100",
				TotalBidpRsqn: "200",
			},
			Output2: kisclient.InquireAskingPriceExpCcnOutput2{
				StckShrnISCD: "005930",
				AntcCnpr:     "75050",
				AntcVol:      "30",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.fetchOrderbookSnapshot(context.Background(), orderbook.SnapshotInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.orderbookCalls)
	require.Equal(t, provider.OperationKISOrderbook, result.Operation)
	require.Equal(t, "14:12:00", result.Snapshot.AcceptanceTime)
	require.Equal(t, "75050", result.Snapshot.Expected.Price)
	require.Len(t, result.Snapshot.Levels, 2)
	require.Equal(t, orderbook.SideAsk, result.Snapshot.Levels[0].Side)
	require.Equal(t, "75100", result.Snapshot.Levels[0].Price)
	require.Equal(t, orderbook.SideBid, result.Snapshot.Levels[1].Side)
}

func TestListMarketTradesUsesRecentTrades(t *testing.T) {
	client := &fakeKISClient{
		trades: []kisclient.InquireCcnlOutputItem{
			{
				StckCntgHour: "141200",
				StckPrpr:     "75000",
				CntgVol:      "12",
				TdayRltv:     "120.5",
				PrdyVrss:     "1000",
			},
		},
	}
	p := NewWithClient(client, true)

	result, err := p.listMarketTrades(context.Background(), tradesrole.ListInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.tradesCalls)
	require.Equal(t, 0, client.timeTradesCalls)
	require.Len(t, result.Trades, 1)
	trade := result.Trades[0]
	require.Equal(t, provider.OperationKISTrades, trade.Operation)
	require.Equal(t, "14:12:00", trade.Time)
	require.Equal(t, "75000", trade.Price)
	require.Equal(t, "12", trade.Volume)
}

func TestListMarketTradesWithAtUsesTimedTrades(t *testing.T) {
	client := &fakeKISClient{
		timedTrade: kisclient.InquireTimeItemConclusionOutput2{
			StckCntgHour: "141200",
			StckPbpr:     "75000",
			Askp:         "75100",
			Bidp:         "75000",
			Cnqn:         "12",
			AcmlVol:      "1234",
		},
	}
	p := NewWithClient(client, true)

	result, err := p.listMarketTrades(context.Background(), tradesrole.ListInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
		At:           "14:12:00",
	})

	require.NoError(t, err)
	require.Equal(t, 0, client.tradesCalls)
	require.Equal(t, 1, client.timeTradesCalls)
	require.Equal(t, "141200", client.lastTimeTradesInputHour)
	require.Len(t, result.Trades, 1)
	trade := result.Trades[0]
	require.Equal(t, provider.OperationKISTimeTrades, trade.Operation)
	require.Equal(t, "75100", trade.Ask)
	require.Equal(t, "1234", trade.AccumulatedVolume)
}

func TestRegistryRoutesNewKISMarketDataRoles(t *testing.T) {
	p := NewWithClient(&fakeKISClient{}, true)
	registry := provider.NewRegistry()
	require.NoError(t, Register(registry, p))
	router := provider.NewRouter(registry)

	intradayFetcher, err := intradaybar.NewRouter(router).RouteIntradayBars(context.Background(), intradaybar.RouteInput{
		ProviderID:   provider.ProviderKIS,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})
	require.NoError(t, err)
	require.Equal(t, provider.GroupKISQuote, intradayFetcher.IntradayBarProfile().Group)

	orderbookSnapshotter, err := orderbook.NewRouter(router).RouteOrderbookSnapshot(context.Background(), orderbook.RouteInput{
		ProviderID:   provider.ProviderKIS,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})
	require.NoError(t, err)
	require.Equal(t, provider.GroupKISQuote, orderbookSnapshotter.OrderbookProfile().Group)

	tradesLister, err := tradesrole.NewRouter(router).RouteMarketTrades(context.Background(), tradesrole.RouteInput{
		ProviderID:   provider.ProviderKIS,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "005930",
	})
	require.NoError(t, err)
	require.Equal(t, provider.GroupKISQuote, tradesLister.TradesProfile().Group)
}

func TestSearchStockInstrumentCombinesProductAndStock(t *testing.T) {
	client := &fakeKISClient{
		product: kisclient.SearchInfoResponse{Output: kisclient.SearchInfoOutput{
			Pdno:         "005930",
			PrdtName:     "Samsung Electronics",
			StdPdno:      "KR7005930003",
			PrdtTypeCd:   "300",
			PrdtClsfCd:   "STK",
			PrdtAbrvName: "Samsung Electronics",
			ShtnPdno:     "005930",
		}},
		stock: kisclient.SearchStockInfoResponse{Output: kisclient.SearchStockInfoOutput{
			Pdno:              "005930",
			PrdtTypeCd:        "300",
			StdPdno:           "KR7005930003",
			PrdtName:          "Samsung Electronics",
			MketIDCd:          "STK",
			SctyGrpIDCd:       "ST",
			LstgStqt:          "5969782550",
			StdIdstClsfCdName: "Semiconductors",
		}},
	}
	p := NewWithClient(client, true)

	result, err := p.searchInstruments(context.Background(), instrument.SearchInput{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Query:        "005930",
	})

	require.NoError(t, err)
	require.Equal(t, 1, client.productCalls)
	require.Equal(t, 1, client.stockCalls)
	require.Len(t, result.Instruments, 1)
	item := result.Instruments[0]
	require.Equal(t, provider.ProviderKIS, item.Provider)
	require.Equal(t, provider.OperationKISStock, item.Operation)
	require.Equal(t, "005930", item.SecurityCode)
	require.Equal(t, "KR7005930003", item.ISIN)
	require.Equal(t, "Semiconductors", item.Extensions["kis_industry_name"])
}

func TestBuilderUsesEnvCredentialFallbacks(t *testing.T) {
	builder := NewBuilder()
	config := provider.Config{
		"env": map[string]any{
			"KIS_APP_KEY":    "key",
			"KIS_APP_SECRET": "secret",
		},
	}

	decision := builder.Decide(provider.RegisterOptions{}, config)
	require.True(t, decision.Register)

	instance, err := builder.Build(config)
	require.NoError(t, err)
	require.Equal(t, provider.ProviderKIS, instance.ProviderIdentity().ID)
}

type fakeKISClient struct {
	tokenCalls    int
	useTokenCalls int

	priceCalls        int
	etfetnCalls       int
	etfComponentCalls int
	dailyCalls        int
	intradayCalls     int
	orderbookCalls    int
	tradesCalls       int
	timeTradesCalls   int
	productCalls      int
	stockCalls        int
	rawCalls          int

	price                   kisclient.InquirePriceResponse
	token                   kisclient.Token
	usedToken               kisclient.Token
	etfetnPrice             kisclient.ETFETNPrice
	etfComponents           kisclient.ETFComponentStockPriceResult
	bars                    []kisclient.InquireDailyItemChartPriceOutput2Item
	intradayBars            []kisclient.InquireTimeItemChartPriceOutput2Item
	orderbook               kisclient.InquireAskingPriceExpCcnResponse
	trades                  []kisclient.InquireCcnlOutputItem
	timedTrade              kisclient.InquireTimeItemConclusionOutput2
	product                 kisclient.SearchInfoResponse
	stock                   kisclient.SearchStockInfoResponse
	rawResponse             any
	rawErrors               []error
	rawInput                map[string]string
	lastTimeTradesInputHour string
}

func (c *fakeKISClient) Token(context.Context) (kisclient.Token, error) {
	c.tokenCalls++
	if c.token.AccessToken != "" {
		return c.token, nil
	}
	return kisclient.Token{AccessToken: "token", TokenType: "Bearer", ExpiresIn: 86400}, nil
}

func (c *fakeKISClient) UseToken(token kisclient.Token) {
	c.useTokenCalls++
	c.usedToken = token
}

func (c *fakeKISClient) RawOperations() []kisclient.RawOperationMetadata {
	return kisclient.RawOperations()
}

func (c *fakeKISClient) LookupRawOperation(operationID string) (kisclient.RawOperationMetadata, bool) {
	return kisclient.LookupRawOperation(operationID)
}

func (c *fakeKISClient) RawRequestTemplate(operationID string) (map[string]string, error) {
	return kisclient.RawRequestTemplate(operationID)
}

func (c *fakeKISClient) InvokeRaw(_ context.Context, _ string, input map[string]string) (any, error) {
	c.rawCalls++
	c.rawInput = input
	if len(c.rawErrors) > 0 {
		err := c.rawErrors[0]
		c.rawErrors = c.rawErrors[1:]
		return nil, err
	}
	if c.rawResponse != nil {
		return c.rawResponse, nil
	}
	return kisclient.InquirePriceResponse{RtCd: "0", MsgCd: "ok", Msg1: "ok"}, nil
}

func (c *fakeKISClient) Quote() quoteAPI {
	return fakeKISQuoteService{client: c}
}

func (c *fakeKISClient) Instrument() instrumentAPI {
	return fakeKISInstrumentService{client: c}
}

type fakeKISQuoteService struct {
	client *fakeKISClient
}

func (s fakeKISQuoteService) Price(context.Context, kisclient.InquirePriceRequest) (kisclient.InquirePriceResponse, error) {
	c := s.client
	c.priceCalls++
	return c.price, nil
}

func (c *fakeKISClient) ETFETNPrice(context.Context, string) (kisclient.ETFETNPrice, error) {
	c.etfetnCalls++
	return c.etfetnPrice, nil
}

func (c *fakeKISClient) ETFComponentStockPrices(context.Context, string) (kisclient.ETFComponentStockPriceResult, error) {
	c.etfComponentCalls++
	return c.etfComponents, nil
}

func (s fakeKISQuoteService) Daily(context.Context, kisclient.InquireDailyItemChartPriceRequest) (kisclient.InquireDailyItemChartPriceResponse, error) {
	c := s.client
	c.dailyCalls++
	return kisclient.InquireDailyItemChartPriceResponse{Output2: c.bars}, nil
}

func (s fakeKISQuoteService) Intraday(context.Context, kisclient.InquireTimeItemChartPriceRequest) (kisclient.InquireTimeItemChartPriceResponse, error) {
	c := s.client
	c.intradayCalls++
	return kisclient.InquireTimeItemChartPriceResponse{Output2: c.intradayBars}, nil
}

func (s fakeKISQuoteService) Orderbook(context.Context, kisclient.InquireAskingPriceExpCcnRequest) (kisclient.InquireAskingPriceExpCcnResponse, error) {
	c := s.client
	c.orderbookCalls++
	return c.orderbook, nil
}

func (s fakeKISQuoteService) Trades(context.Context, kisclient.InquireCcnlRequest) (kisclient.InquireCcnlResponse, error) {
	c := s.client
	c.tradesCalls++
	return kisclient.InquireCcnlResponse{Output: c.trades}, nil
}

func (s fakeKISQuoteService) TimeTrades(_ context.Context, input kisclient.InquireTimeItemConclusionRequest) (kisclient.InquireTimeItemConclusionResponse, error) {
	c := s.client
	c.timeTradesCalls++
	c.lastTimeTradesInputHour = input.FidInputHour1
	return kisclient.InquireTimeItemConclusionResponse{Output2: c.timedTrade}, nil
}

type fakeKISInstrumentService struct {
	client *fakeKISClient
}

func (s fakeKISInstrumentService) Product(context.Context, kisclient.SearchInfoRequest) (kisclient.SearchInfoResponse, error) {
	c := s.client
	c.productCalls++
	return c.product, nil
}

func (s fakeKISInstrumentService) Stock(context.Context, kisclient.SearchStockInfoRequest) (kisclient.SearchStockInfoResponse, error) {
	c := s.client
	c.stockCalls++
	return c.stock, nil
}

type fakeTokenCache struct {
	tokens map[TokenCacheKey]CachedToken
	gets   []TokenCacheKey
	puts   []CachedToken
}

func newFakeTokenCache() *fakeTokenCache {
	return &fakeTokenCache{
		tokens: map[TokenCacheKey]CachedToken{},
	}
}

func (c *fakeTokenCache) Get(_ context.Context, key TokenCacheKey) (CachedToken, bool, error) {
	c.gets = append(c.gets, key)
	token, ok := c.tokens[key]
	return token, ok, nil
}

func (c *fakeTokenCache) Put(_ context.Context, token CachedToken) error {
	c.puts = append(c.puts, token)
	c.tokens[token.Key] = token
	return nil
}

type fakeReadLimiter struct {
	requests []kisclient.RateLimitRequest
	err      error
}

func (l *fakeReadLimiter) Wait(_ context.Context, request kisclient.RateLimitRequest) error {
	l.requests = append(l.requests, request)
	return l.err
}
