package daily

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/samber/oops"
)

const (
	defaultBackfillWorkers = 1
	maxBackfillWorkers     = 16
)

type ReadRepository interface {
	QueryDailyBars(ctx context.Context, query Query) ([]dailybar.Bar, error)
}

type BarStream interface {
	Next(ctx context.Context) (dailybar.Bar, bool, error)
	Close() error
}

type StreamRepository interface {
	StreamDailyBars(ctx context.Context, query Query) (BarStream, error)
}

type WriteRepository interface {
	UpsertDailyBars(ctx context.Context, bars []dailybar.Bar) (WriteResult, error)
}

type Query struct {
	Market       provider.Market
	SecurityType provider.SecurityType
	Symbol       string
	From         string
	To           string
}

type WriteResult struct {
	RowsAffected int
	BarsWritten  int
}

type ReadService struct {
	reader ReadRepository
}

type Service struct {
	router dailybar.Router
	reader ReadRepository
	writer WriteRepository
}

func NewReadService(reader ReadRepository) (ReadService, error) {
	if reader == nil {
		return ReadService{}, oops.In("daily_service").New("daily service read repository is nil")
	}
	return ReadService{reader: reader}, nil
}

func NewService(reader ReadRepository, writer WriteRepository, router dailybar.Router) (Service, error) {
	errb := oops.In("daily_service")
	if reader == nil {
		return Service{}, errb.New("daily service read repository is nil")
	}
	if writer == nil {
		return Service{}, errb.New("daily service write repository is nil")
	}
	if router == nil {
		return Service{}, errb.New("daily service router is nil")
	}
	return Service{
		router: router,
		reader: reader,
		writer: writer,
	}, nil
}

type Request struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Market         provider.Market
	SecurityType   provider.SecurityType
	Symbol         string
	From           string
	To             string
	AsOf           string
	Workers        int
	Progress       io.Writer
}

type BarsResult struct {
	Bars []dailybar.Bar
}

type DateList []string

func (d DateList) MarshalCSV() ([]byte, error) {
	return []byte(strconv.Itoa(len(d))), nil
}

type CollectResult struct {
	Market       provider.Market       `json:"market" csv:"market"`
	SecurityType provider.SecurityType `json:"security_type" csv:"security_type"`
	ProviderID   provider.ProviderID   `json:"provider" csv:"provider"`
	Group        provider.GroupID      `json:"provider_group" csv:"group"`
	Dates        DateList              `json:"dates" csv:"dates"`
	BarsFetched  int                   `json:"bars_fetched" csv:"fetched"`
	BarsStored   int                   `json:"bars_stored" csv:"stored"`
	RowsAffected int                   `json:"rows_affected" csv:"rows_affected"`

	bars []dailybar.Bar `csv:"-"`
}

func (s ReadService) Get(ctx context.Context, req Request) (BarsResult, error) {
	reqErrb := oops.In("daily_service").With("symbol", req.Symbol, "from", req.From, "to", req.To, "as_of", req.AsOf)
	if s.reader == nil {
		return BarsResult{}, reqErrb.New("daily service read repository is nil")
	}
	dates, err := resolveDateRange(req.From, req.To, req.AsOf)
	if err != nil {
		return BarsResult{}, reqErrb.Wrap(err)
	}
	query := queryFromRequest(req, dates)
	queryErrb := oops.In("daily_service").With("market", query.Market, "security_type", query.SecurityType, "symbol", query.Symbol, "from", query.From, "to", query.To)
	bars, err := s.reader.QueryDailyBars(ctx, query)
	if err != nil {
		return BarsResult{}, queryErrb.Wrapf(err, "get daily bars")
	}
	if len(bars) == 0 {
		return BarsResult{}, notFound(req, query)
	}
	return BarsResult{Bars: bars}, nil
}

func (s Service) Get(ctx context.Context, req Request) (BarsResult, error) {
	return ReadService{reader: s.reader}.Get(ctx, req)
}

func (s Service) Ensure(ctx context.Context, req Request) (BarsResult, error) {
	reqErrb := oops.In("daily_service").With("symbol", req.Symbol, "from", req.From, "to", req.To, "as_of", req.AsOf)
	if req.Symbol == "" {
		return BarsResult{}, reqErrb.New("ensure daily requires symbol")
	}
	dates, err := resolveDateRange(req.From, req.To, req.AsOf)
	if err != nil {
		return BarsResult{}, reqErrb.Wrap(err)
	}
	if len(dates) == 0 {
		return BarsResult{}, reqErrb.New("ensure daily requires --as-of or --from/--to")
	}

	query := queryFromRequest(req, dates)
	queryErrb := oops.In("daily_service").With("market", query.Market, "security_type", query.SecurityType, "symbol", query.Symbol, "from", query.From, "to", query.To)
	if s.reader == nil {
		return BarsResult{}, queryErrb.New("daily service read repository is nil")
	}
	existing, err := s.reader.QueryDailyBars(ctx, query)
	if err != nil {
		return BarsResult{}, queryErrb.Wrapf(err, "query existing daily bars")
	}
	missingDates := datesWithoutSymbol(existing, dates, req.Symbol)
	collectedBars := make([]dailybar.Bar, 0)
	for _, date := range missingDates {
		result, err := s.collectDate(ctx, req, date)
		if err != nil {
			return BarsResult{}, reqErrb.With("market", withDefaultMarket(req.Market), "security_type", req.SecurityType, "date", apiDate(date)).Wrapf(err, "collect missing daily bars")
		}
		collectedBars = append(collectedBars, result.bars...)
	}

	bars, err := s.reader.QueryDailyBars(ctx, query)
	if err != nil {
		return BarsResult{}, queryErrb.Wrapf(err, "query stored daily bars")
	}
	if len(bars) == 0 {
		if resolvedSymbol := singleCollectedSymbol(collectedBars); resolvedSymbol != "" && resolvedSymbol != req.Symbol {
			resolvedQuery := query
			resolvedQuery.Symbol = resolvedSymbol
			bars, err = s.reader.QueryDailyBars(ctx, resolvedQuery)
			if err != nil {
				return BarsResult{}, queryErrb.With("resolved_symbol", resolvedSymbol).Wrapf(err, "query stored daily bars by resolved symbol")
			}
		}
	}
	if len(bars) == 0 && len(collectedBars) > 0 {
		return BarsResult{Bars: collectedBars}, nil
	}
	if len(bars) == 0 {
		return BarsResult{}, notFound(req, query)
	}
	return BarsResult{Bars: bars}, nil
}

func (s Service) Sync(ctx context.Context, req Request) (CollectResult, error) {
	date, err := parseDate(req.AsOf, "--as-of")
	if err != nil {
		return CollectResult{}, oops.In("daily_service").With("as_of", req.AsOf).Wrap(err)
	}
	return s.collectDate(ctx, req, date)
}

func (s Service) Backfill(ctx context.Context, req Request) (CollectResult, error) {
	errb := oops.In("daily_service").With("from", req.From, "to", req.To, "as_of", req.AsOf, "workers", req.Workers)
	dates, err := resolveDateRange(req.From, req.To, req.AsOf)
	if err != nil {
		return CollectResult{}, errb.Wrap(err)
	}
	if len(dates) == 0 {
		return CollectResult{}, errb.New("backfill daily requires --from/--to")
	}
	workers, err := normalizeBackfillWorkers(req.Workers)
	if err != nil {
		return CollectResult{}, errb.Wrap(err)
	}
	result, err := s.collectRange(ctx, req, dates[0], dates[len(dates)-1], workers)
	if err != nil {
		return CollectResult{}, errb.With("market", withDefaultMarket(req.Market), "security_type", req.SecurityType, "from", apiDate(dates[0]), "to", apiDate(dates[len(dates)-1])).Wrapf(err, "backfill daily range")
	}
	return result, nil
}

func (s Service) collectDate(ctx context.Context, req Request, date time.Time) (CollectResult, error) {
	result, err := s.fetchDate(ctx, req, date)
	if err != nil {
		return CollectResult{}, err
	}
	return s.storeCollection(ctx, result)
}

func (s Service) fetchDate(ctx context.Context, req Request, date time.Time) (CollectResult, error) {
	return s.fetchRange(ctx, req, date, date, 1)
}

func (s Service) collectRange(ctx context.Context, req Request, from time.Time, to time.Time, workers int) (CollectResult, error) {
	market := withDefaultMarket(req.Market)
	fromText := apiDate(from)
	toText := apiDate(to)
	errb := oops.In("daily_service").With("market", market, "security_type", req.SecurityType, "from", fromText, "to", toText, "workers", workers)

	if s.router == nil {
		return CollectResult{}, errb.New("daily service router is nil")
	}
	if s.writer == nil {
		return CollectResult{}, errb.New("daily service write repository is nil")
	}
	if req.SecurityType == "" {
		return CollectResult{}, errb.New("daily collection requires --security-type")
	}

	startedAt := time.Now()
	fetcher, err := s.router.RouteDailyBars(ctx, dailybar.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "symbol", req.Symbol).Wrapf(err, "route daily bars")
	}
	pageFetcher, ok := fetcher.(dailybar.PageFetcher)
	if !ok {
		return CollectResult{}, errb.With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "symbol", req.Symbol).New("routed dailybar fetcher does not support page fetch")
	}

	first, err := pageFetcher.FetchDailyBarsPage(ctx, dailybar.PageFetchInput{
		Market:       market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
		From:         fromText,
		To:           toText,
		PageNo:       1,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID, "page", 1).Wrapf(err, "fetch daily bars page")
	}

	totalPages := backfillPageCount(first.TotalCount, first.PageSize, len(first.Bars))
	result := CollectResult{
		Market:       market,
		SecurityType: req.SecurityType,
		ProviderID:   first.Provider.ID,
		Group:        first.Group,
	}
	if totalPages == 0 {
		progressBackfill(req.Progress, 0, 0, 0, 0, startedAt)
		return result, nil
	}

	pipelineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	chunks := make(chan dailybar.PageFetchResult)
	writerDone := make(chan backfillWriteSummary, 1)
	go s.writeBackfillChunks(ctx, chunks, cancel, totalPages, req.Progress, startedAt, writerDone)

	fetchErrs := make(chan error, 1)
	recordFetchErr := func(err error) {
		select {
		case fetchErrs <- err:
		default:
		}
	}

	if !sendBackfillChunk(pipelineCtx, chunks, first) {
		close(chunks)
		summary := <-writerDone
		if summary.err != nil {
			return CollectResult{}, summary.err
		}
		return CollectResult{}, errb.Wrap(pipelineCtx.Err())
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	workerCount := workers
	if remainingPages := totalPages - 1; remainingPages < workerCount {
		workerCount = remainingPages
	}
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pageNo := range jobs {
				page, err := pageFetcher.FetchDailyBarsPage(pipelineCtx, dailybar.PageFetchInput{
					Market:       market,
					SecurityType: req.SecurityType,
					Symbol:       req.Symbol,
					From:         fromText,
					To:           toText,
					PageNo:       pageNo,
					PageSize:     first.PageSize,
				})
				if err != nil {
					recordFetchErr(errb.With("provider", req.ProviderID, "page", pageNo).Wrapf(err, "fetch daily bars page"))
					cancel()
					return
				}
				if !sendBackfillChunk(pipelineCtx, chunks, page) {
					return
				}
			}
		}()
	}

	if workerCount > 0 {
		go func() {
			defer close(jobs)
			for pageNo := 2; pageNo <= totalPages; pageNo++ {
				select {
				case jobs <- pageNo:
				case <-pipelineCtx.Done():
					return
				}
			}
		}()
	} else {
		close(jobs)
	}

	wg.Wait()
	close(chunks)
	summary := <-writerDone
	if summary.err != nil {
		return CollectResult{}, summary.err
	}
	select {
	case err := <-fetchErrs:
		return CollectResult{}, err
	default:
	}

	result.Dates = summary.dates
	result.BarsFetched = summary.barsFetched
	result.BarsStored = summary.barsStored
	result.RowsAffected = summary.rowsAffected
	return result, nil
}

func (s Service) fetchRange(ctx context.Context, req Request, from time.Time, to time.Time, workers int) (CollectResult, error) {
	market := withDefaultMarket(req.Market)
	fromText := apiDate(from)
	toText := apiDate(to)
	errb := oops.In("daily_service").With("market", market, "security_type", req.SecurityType, "from", fromText, "to", toText, "workers", workers)

	if s.router == nil {
		return CollectResult{}, errb.New("daily service router is nil")
	}
	if req.SecurityType == "" {
		return CollectResult{}, errb.New("daily collection requires --security-type")
	}

	fetcher, err := s.router.RouteDailyBars(ctx, dailybar.RouteInput{
		ProviderID:     req.ProviderID,
		PreferProvider: req.PreferProvider,
		Market:         market,
		SecurityType:   req.SecurityType,
		Symbol:         req.Symbol,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID, "prefer_provider", req.PreferProvider, "symbol", req.Symbol).Wrapf(err, "route daily bars")
	}

	result, err := fetcher.FetchDailyBars(ctx, dailybar.FetchInput{
		Market:       market,
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
		From:         fromText,
		To:           toText,
		Workers:      workers,
	})
	if err != nil {
		return CollectResult{}, errb.With("provider", req.ProviderID).Wrapf(err, "fetch daily bars")
	}

	return CollectResult{
		Market:       market,
		SecurityType: req.SecurityType,
		ProviderID:   result.Provider.ID,
		Group:        result.Group,
		Dates:        collectDatesFromBars(result.Bars),
		BarsFetched:  len(result.Bars),
		bars:         result.Bars,
	}, nil
}

func (s Service) storeCollection(ctx context.Context, result CollectResult) (CollectResult, error) {
	errb := oops.In("daily_service").With("provider", result.ProviderID, "group", result.Group, "bars", len(result.bars))
	if s.writer == nil {
		return CollectResult{}, errb.New("daily service write repository is nil")
	}
	writeResult, err := s.writer.UpsertDailyBars(ctx, result.bars)
	if err != nil {
		return CollectResult{}, errb.Wrapf(err, "store daily bars")
	}
	result.BarsStored = writeResult.BarsWritten
	result.RowsAffected = writeResult.RowsAffected
	return result, nil
}

type backfillWriteSummary struct {
	dates        DateList
	barsFetched  int
	barsStored   int
	rowsAffected int
	err          error
}

func (s Service) writeBackfillChunks(ctx context.Context, chunks <-chan dailybar.PageFetchResult, cancel context.CancelFunc, totalPages int, progress io.Writer, startedAt time.Time, done chan<- backfillWriteSummary) {
	errb := oops.In("daily_service")
	summary := backfillWriteSummary{}
	seenDates := make(map[string]bool)
	pagesDone := 0

	for chunk := range chunks {
		writeResult, err := s.writer.UpsertDailyBars(ctx, chunk.Bars)
		if err != nil {
			summary.err = errb.With("page", chunk.PageNo, "bars", len(chunk.Bars)).Wrapf(err, "store daily bars chunk")
			cancel()
			done <- summary
			return
		}
		pagesDone++
		summary.barsFetched += len(chunk.Bars)
		summary.barsStored += writeResult.BarsWritten
		summary.rowsAffected += writeResult.RowsAffected
		for _, bar := range chunk.Bars {
			if bar.TradingDate == "" || seenDates[bar.TradingDate] {
				continue
			}
			seenDates[bar.TradingDate] = true
			summary.dates = append(summary.dates, bar.TradingDate)
		}
		progressBackfill(progress, pagesDone, totalPages, summary.barsFetched, summary.barsStored, startedAt)
	}
	sort.Strings(summary.dates)
	done <- summary
}

func sendBackfillChunk(ctx context.Context, chunks chan<- dailybar.PageFetchResult, chunk dailybar.PageFetchResult) bool {
	select {
	case chunks <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

func backfillPageCount(totalCount int, pageSize int, firstPageBars int) int {
	if totalCount <= 0 {
		if firstPageBars > 0 {
			return 1
		}
		return 0
	}
	if pageSize <= 0 {
		return 1
	}
	return (totalCount + pageSize - 1) / pageSize
}

func progressBackfill(w io.Writer, pagesDone int, totalPages int, barsFetched int, barsStored int, startedAt time.Time) {
	if w == nil {
		return
	}
	elapsed := time.Since(startedAt).Truncate(time.Second)
	_, _ = fmt.Fprintf(w, "backfill daily: fetched pages %d/%d, fetched rows %d, stored rows %d, elapsed %s\n", pagesDone, totalPages, barsFetched, barsStored, elapsed)
}

func normalizeBackfillWorkers(workers int) (int, error) {
	if workers == 0 {
		return defaultBackfillWorkers, nil
	}
	if workers < 0 {
		return 0, oops.In("daily_service").With("workers", workers).Errorf("workers must be positive: %d", workers)
	}
	if workers > maxBackfillWorkers {
		return 0, oops.In("daily_service").With("workers", workers, "max_workers", maxBackfillWorkers).Errorf("workers must be <= %d: %d", maxBackfillWorkers, workers)
	}
	return workers, nil
}

func collectDatesFromBars(bars []dailybar.Bar) DateList {
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

func queryFromRequest(req Request, dates []time.Time) Query {
	query := Query{
		Market:       withDefaultMarket(req.Market),
		SecurityType: req.SecurityType,
		Symbol:       req.Symbol,
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

func datesWithoutSymbol(existing []dailybar.Bar, dates []time.Time, symbol string) []time.Time {
	found := make(map[string]bool, len(existing))
	for _, bar := range existing {
		if bar.Symbol == symbol {
			found[bar.TradingDate] = true
		}
	}
	missing := make([]time.Time, 0)
	for _, date := range dates {
		if !found[isoDate(date)] {
			missing = append(missing, date)
		}
	}
	return missing
}

func singleCollectedSymbol(bars []dailybar.Bar) string {
	symbol := ""
	for _, bar := range bars {
		if bar.Symbol == "" {
			continue
		}
		if symbol == "" {
			symbol = bar.Symbol
			continue
		}
		if bar.Symbol != symbol {
			return ""
		}
	}
	return symbol
}

func notFound(req Request, query Query) error {
	hint := fmt.Sprintf("run `mwosa ensure daily %s --from %s --to %s`", req.Symbol, query.From, query.To)
	if query.From == "" && query.To == "" {
		hint = fmt.Sprintf("run `mwosa ensure daily %s --as-of <YYYYMMDD>`", req.Symbol)
	}
	return oops.In("daily_service").With(
		"market", query.Market,
		"security_type", query.SecurityType,
		"symbol", req.Symbol,
		"from", query.From,
		"to", query.To,
	).Wrap(&NotFoundError{
		Symbol:       req.Symbol,
		Market:       query.Market,
		SecurityType: query.SecurityType,
		From:         query.From,
		To:           query.To,
		Hint:         hint,
	})
}
