package index

import (
	"context"
	"fmt"
	"sort"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/indexbar"
	"github.com/samber/oops"
)

type ReadRepository interface {
	QueryIndexBars(ctx context.Context, query Query) ([]indexbar.Bar, error)
}

type WriteRepository interface {
	UpsertIndexBars(ctx context.Context, bars []indexbar.Bar) (WriteResult, error)
}

type Query struct {
	Market    provider.Market
	IndexCode string
	From      string
	To        string
}

type WriteResult struct {
	RowsAffected int
	BarsWritten  int
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	IndexCode      string
	From           string
	To             string
	AsOf           string
}

type BarsResult struct {
	Bars []indexbar.Bar
}

type CollectResult struct {
	Market       provider.Market     `json:"market" csv:"market"`
	ProviderID   provider.ProviderID `json:"provider" csv:"provider"`
	Group        provider.GroupID    `json:"provider_group" csv:"group"`
	IndexCode    string              `json:"index_code,omitempty" csv:"index_code"`
	Dates        DateList            `json:"dates" csv:"dates"`
	BarsFetched  int                 `json:"bars_fetched" csv:"fetched"`
	BarsStored   int                 `json:"bars_stored" csv:"stored"`
	RowsAffected int                 `json:"rows_affected" csv:"rows_affected"`

	bars []indexbar.Bar `csv:"-"`
}

type DateList []string

func (d DateList) MarshalCSV() ([]byte, error) {
	return []byte(fmt.Sprint(len(d))), nil
}

type ReadService struct {
	reader ReadRepository
}

type Service struct {
	router indexbar.Router
	reader ReadRepository
	writer WriteRepository
}

func NewReadService(reader ReadRepository) (ReadService, error) {
	if reader == nil {
		return ReadService{}, oops.In("index_service").New("index service read repository is nil")
	}
	return ReadService{reader: reader}, nil
}

func NewService(reader ReadRepository, writer WriteRepository, router indexbar.Router) (Service, error) {
	errb := oops.In("index_service")
	if reader == nil {
		return Service{}, errb.New("index service read repository is nil")
	}
	if writer == nil {
		return Service{}, errb.New("index service write repository is nil")
	}
	if router == nil {
		return Service{}, errb.New("index service router is nil")
	}
	return Service{router: router, reader: reader, writer: writer}, nil
}

func (s ReadService) Get(ctx context.Context, req Request) (BarsResult, error) {
	errb := oops.In("index_service").With("index_code", req.IndexCode, "from", req.From, "to", req.To, "as_of", req.AsOf)
	if s.reader == nil {
		return BarsResult{}, errb.New("index service read repository is nil")
	}
	if req.IndexCode == "" {
		return BarsResult{}, errb.New("get index requires index code")
	}
	dates, err := resolveDateRange(req.From, req.To, req.AsOf)
	if err != nil {
		return BarsResult{}, errb.Wrap(err)
	}
	query := queryFromRequest(req, dates)
	bars, err := s.reader.QueryIndexBars(ctx, query)
	if err != nil {
		return BarsResult{}, errb.With("market", query.Market).Wrapf(err, "get index bars")
	}
	if len(bars) == 0 {
		return BarsResult{}, notFound(req, query)
	}
	return BarsResult{Bars: bars}, nil
}

func (s Service) Get(ctx context.Context, req Request) (BarsResult, error) {
	if req.ProviderID != "" || req.PreferProvider != "" {
		result, err := s.fetchRange(ctx, req)
		if err != nil {
			return BarsResult{}, err
		}
		if len(result.bars) == 0 {
			return BarsResult{}, notFound(req, queryFromRequest(req, nil))
		}
		return BarsResult{Bars: result.bars}, nil
	}
	return ReadService{reader: s.reader}.Get(ctx, req)
}

func (s Service) Sync(ctx context.Context, req Request) (CollectResult, error) {
	date, err := parseDate(req.AsOf, "--as-of")
	if err != nil {
		return CollectResult{}, oops.In("index_service").With("as_of", req.AsOf).Wrap(err)
	}
	result, err := s.fetchDate(ctx, req, date)
	if err != nil {
		return CollectResult{}, err
	}
	return s.store(ctx, result)
}

func (s Service) fetchDate(ctx context.Context, req Request, date time.Time) (CollectResult, error) {
	req.From = apiDate(date)
	req.To = apiDate(date)
	req.AsOf = ""
	return s.fetchRange(ctx, req)
}

func (s Service) fetchRange(ctx context.Context, req Request) (CollectResult, error) {
	market := withDefaultMarket(req.Market)
	errb := oops.In("index_service").With("market", market, "index_code", req.IndexCode, "from", req.From, "to", req.To, "as_of", req.AsOf)
	if s.router == nil {
		return CollectResult{}, errb.New("index service router is nil")
	}
	dates, err := resolveDateRange(req.From, req.To, req.AsOf)
	if err != nil {
		return CollectResult{}, errb.Wrap(err)
	}
	if len(dates) == 0 {
		return CollectResult{}, errb.New("index request requires --as-of or --from/--to")
	}
	fetcher, err := s.router.RouteIndexBars(ctx, indexbar.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         market,
		IndexCode:      req.IndexCode,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID, "prefer_provider", req.PreferProvider).Wrapf(err, "route index bars")
	}
	fromText := apiDate(dates[0])
	toText := apiDate(dates[len(dates)-1])
	if batchFetcher, ok := fetcher.(indexbar.BatchFetcher); ok {
		result, err := batchFetcher.FetchIndexBatch(ctx, indexbar.BatchFetchInput{
			Market:    market,
			IndexCode: req.IndexCode,
			From:      fromText,
			To:        toText,
		})
		if err != nil {
			return CollectResult{}, errb.With("provider", req.ProviderID).Wrapf(err, "fetch index bars batch")
		}
		return collectResultFromBars(req, market, result.Provider.ID, result.Group, result.Bars), nil
	}
	result, err := fetcher.FetchIndexBars(ctx, indexbar.FetchInput{
		Market:    market,
		IndexCode: req.IndexCode,
		From:      fromText,
		To:        toText,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID).Wrapf(err, "fetch index bars")
	}
	return collectResultFromBars(req, market, result.Provider.ID, result.Group, result.Bars), nil
}

func (s Service) store(ctx context.Context, result CollectResult) (CollectResult, error) {
	errb := oops.In("index_service").With("provider", result.ProviderID, "group", result.Group, "bars", len(result.bars))
	if s.writer == nil {
		return CollectResult{}, errb.New("index service write repository is nil")
	}
	writeResult, err := s.writer.UpsertIndexBars(ctx, result.bars)
	if err != nil {
		return CollectResult{}, errb.Wrapf(err, "store index bars")
	}
	result.BarsStored = writeResult.BarsWritten
	result.RowsAffected = writeResult.RowsAffected
	return result, nil
}

func collectResultFromBars(req Request, market provider.Market, providerID provider.ProviderID, group provider.GroupID, bars []indexbar.Bar) CollectResult {
	return CollectResult{
		Market:      market,
		ProviderID:  providerID,
		Group:       group,
		IndexCode:   req.IndexCode,
		Dates:       collectDates(bars),
		BarsFetched: len(bars),
		bars:        bars,
	}
}

func resolveDateRange(from string, to string, asOf string) ([]time.Time, error) {
	errb := oops.In("index_service").With("as_of", asOf, "from", from, "to", to)
	if asOf != "" {
		if from != "" || to != "" {
			return nil, errb.New("--as-of cannot be combined with --from or --to")
		}
		date, err := parseDate(asOf, "--as-of")
		if err != nil {
			return nil, err
		}
		return []time.Time{date}, nil
	}
	fromDate, hasFrom, err := parseOptionalDate(from, "--from")
	if err != nil {
		return nil, err
	}
	toDate, hasTo, err := parseOptionalDate(to, "--to")
	if err != nil {
		return nil, err
	}
	switch {
	case !hasFrom && !hasTo:
		return nil, nil
	case hasFrom && !hasTo:
		toDate = fromDate
	case !hasFrom && hasTo:
		fromDate = toDate
	}
	if fromDate.After(toDate) {
		return nil, errb.New("--from must be on or before --to")
	}
	dates := make([]time.Time, 0)
	for date := fromDate; !date.After(toDate); date = date.AddDate(0, 0, 1) {
		dates = append(dates, date)
	}
	return dates, nil
}

func parseDate(value string, field string) (time.Time, error) {
	errb := oops.In("index_service").With("field", field)
	if value == "" {
		return time.Time{}, errb.Errorf("%s is required", field)
	}
	for _, layout := range []string{"20060102", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errb.With("value", value).Errorf("%s must be YYYYMMDD or YYYY-MM-DD: %q", field, value)
}

func parseOptionalDate(value string, field string) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := parseDate(value, field)
	if err != nil {
		return time.Time{}, false, err
	}
	return parsed, true, nil
}

func apiDate(date time.Time) string {
	return date.Format("20060102")
}

func isoDate(date time.Time) string {
	return date.Format("2006-01-02")
}

func queryFromRequest(req Request, dates []time.Time) Query {
	query := Query{
		Market:    withDefaultMarket(req.Market),
		IndexCode: req.IndexCode,
	}
	if len(dates) > 0 {
		query.From = isoDate(dates[0])
		query.To = isoDate(dates[len(dates)-1])
	}
	return query
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func collectDates(bars []indexbar.Bar) DateList {
	seen := make(map[string]bool)
	dates := make([]string, 0)
	for _, bar := range bars {
		if bar.TradingDate == "" || seen[bar.TradingDate] {
			continue
		}
		seen[bar.TradingDate] = true
		dates = append(dates, bar.TradingDate)
	}
	sort.Strings(dates)
	return DateList(dates)
}

func notFound(req Request, query Query) error {
	hint := fmt.Sprintf("run `mwosa sync index --provider <provider> --as-of <YYYYMMDD>`")
	if query.From != "" || query.To != "" {
		hint = fmt.Sprintf("run `mwosa sync index --provider <provider> --as-of %s`", query.From)
	}
	return oops.In("index_service").With(
		"market", query.Market,
		"index_code", req.IndexCode,
		"from", query.From,
		"to", query.To,
	).New("index data not found " + hint)
}
