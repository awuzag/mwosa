package handler

import (
	"context"
	"strconv"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/intradaybar"
	"github.com/awuzag/mwosa/providers/core/orderbook"
	tradesrole "github.com/awuzag/mwosa/providers/core/trades"
	"github.com/awuzag/mwosa/service/intraday"
	orderbookservice "github.com/awuzag/mwosa/service/orderbook"
	tradesservice "github.com/awuzag/mwosa/service/trades"
)

type Intraday struct {
	service intraday.Service
}

func NewIntraday(service intraday.Service) Intraday {
	return Intraday{service: service}
}

type GetIntradayRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	At             string
	Limit          int
}

func (h Intraday) Get(ctx context.Context, req GetIntradayRequest) (IntradayBarsOutput, error) {
	result, err := h.service.Get(ctx, intraday.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
		At:             req.At,
		Limit:          req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return IntradayBarsOutput(result.Bars), nil
}

type IntradayBarsOutput []intradaybar.Bar

func (o IntradayBarsOutput) JSONValue() any {
	return []intradaybar.Bar(o)
}

func (o IntradayBarsOutput) NDJSONRows() any {
	return []intradaybar.Bar(o)
}

func (o IntradayBarsOutput) CSVRows() any {
	return []intradaybar.Bar(o)
}

func (o IntradayBarsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, bar := range o {
		rows = append(rows, []string{bar.TradingDate, bar.Time, bar.Symbol, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume})
	}
	return []string{"date", "time", "symbol", "open", "high", "low", "close", "volume"}, rows
}

type Orderbook struct {
	service orderbookservice.Service
}

func NewOrderbook(service orderbookservice.Service) Orderbook {
	return Orderbook{service: service}
}

type GetOrderbookRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
}

func (h Orderbook) Get(ctx context.Context, req GetOrderbookRequest) (OrderbookOutput, error) {
	result, err := h.service.Get(ctx, orderbookservice.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return OrderbookOutput{}, err
	}
	return OrderbookOutput{Result: result}, nil
}

type OrderbookOutput struct {
	Result orderbook.SnapshotResult
}

func (o OrderbookOutput) JSONValue() any {
	return o.Result.Snapshot
}

func (o OrderbookOutput) NDJSONRows() any {
	return []orderbook.Snapshot{o.Result.Snapshot}
}

func (o OrderbookOutput) CSVRows() any {
	return o.Result.Snapshot.Levels
}

func (o OrderbookOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Snapshot.Levels))
	for _, level := range o.Result.Snapshot.Levels {
		rows = append(rows, []string{o.Result.Snapshot.Symbol, string(level.Side), strconv.Itoa(level.Level), level.Price, level.Quantity, level.QuantityDelta})
	}
	return []string{"symbol", "side", "level", "price", "quantity", "quantity_delta"}, rows
}

type Trades struct {
	service tradesservice.Service
}

func NewTrades(service tradesservice.Service) Trades {
	return Trades{service: service}
}

type ListTradesRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	At             string
	Limit          int
}

func (h Trades) List(ctx context.Context, req ListTradesRequest) (TradesOutput, error) {
	result, err := h.service.List(ctx, tradesservice.Request{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         req.Market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
		At:             req.At,
		Limit:          req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return TradesOutput(result.Trades), nil
}

type TradesOutput []tradesrole.Trade

func (o TradesOutput) JSONValue() any {
	return []tradesrole.Trade(o)
}

func (o TradesOutput) NDJSONRows() any {
	return []tradesrole.Trade(o)
}

func (o TradesOutput) CSVRows() any {
	return []tradesrole.Trade(o)
}

func (o TradesOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, trade := range o {
		rows = append(rows, []string{trade.Symbol, trade.Time, trade.Price, trade.Volume, trade.AccumulatedVolume, trade.Ask, trade.Bid, trade.Strength})
	}
	return []string{"symbol", "time", "price", "volume", "accumulated_volume", "ask", "bid", "strength"}, rows
}
