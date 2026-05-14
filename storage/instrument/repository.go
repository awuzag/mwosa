package instrument

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coreinstrument "github.com/ev3rlit/mwosa/providers/core/instrument"
	instrumentservice "github.com/ev3rlit/mwosa/service/instrument"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type repository struct {
	database *storage.Database
}

var _ instrumentservice.Repository = (*repository)(nil)

func NewRepository(database *storage.Database) (instrumentservice.Repository, error) {
	if database == nil {
		return nil, oops.In("instrument_repository").New("instrument repository database is nil")
	}
	return &repository{database: database}, nil
}

func (r *repository) UpsertInstruments(ctx context.Context, instruments []coreinstrument.Instrument) (instrumentservice.WriteResult, error) {
	errb := oops.In("instrument_repository").With("instruments", len(instruments))
	client, err := r.database.Client(ctx)
	if err != nil {
		return instrumentservice.WriteResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return instrumentservice.WriteResult{}, errb.Wrapf(err, "begin instrument sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cache := upsertCache{
		markets:     make(map[string]storage.MarketV2Row),
		instruments: make(map[string]storage.InstrumentV2Row),
		sources:     make(map[string]int64),
	}
	for _, item := range instruments {
		itemErrb := errb.With("provider", item.Provider, "group", item.Group, "operation", item.Operation, "market", item.Market, "security_type", item.SecurityType, "symbol", canonicalSymbol(item))
		if err := validateInstrumentKey(item); err != nil {
			return instrumentservice.WriteResult{}, itemErrb.Wrap(err)
		}
		nowMS := time.Now().UTC().UnixMilli()
		market, err := ensureMarket(ctx, tx, cache, withDefaultMarket(item.Market), nowMS)
		if err != nil {
			return instrumentservice.WriteResult{}, itemErrb.Wrap(err)
		}
		instrumentRow, err := ensureInstrument(ctx, tx, cache, market.ID, item, nowMS)
		if err != nil {
			return instrumentservice.WriteResult{}, itemErrb.Wrap(err)
		}
		sourceID, err := ensureSource(ctx, tx, cache, instrumentRow.ID, item, nowMS)
		if err != nil {
			return instrumentservice.WriteResult{}, itemErrb.Wrap(err)
		}
		if err := replaceExtensions(ctx, tx, instrumentRow.ID, sourceID, item.Extensions); err != nil {
			return instrumentservice.WriteResult{}, itemErrb.Wrap(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return instrumentservice.WriteResult{}, errb.Wrapf(err, "commit instrument sqlite transaction")
	}
	committed = true
	return instrumentservice.WriteResult{
		InstrumentsWritten: len(instruments),
		RowsAffected:       len(instruments),
	}, nil
}

func (r *repository) SearchInstruments(ctx context.Context, query instrumentservice.Query) (coreinstrument.SearchResult, error) {
	errb := oops.In("instrument_repository").With("provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "query", query.Query, "limit", query.Limit)
	if strings.TrimSpace(query.Query) == "" {
		return coreinstrument.SearchResult{}, errb.New("search instruments requires query")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return coreinstrument.SearchResult{}, errb.Wrap(err)
	}
	records, err := queryRecords(ctx, client, query, false)
	if err != nil {
		return coreinstrument.SearchResult{}, errb.Wrap(err)
	}
	return recordsToSearchResult(ctx, client, records)
}

func (r *repository) InspectInstrument(ctx context.Context, query instrumentservice.Query) (coreinstrument.Instrument, error) {
	symbol := strings.TrimSpace(query.Symbol)
	errb := oops.In("instrument_repository").With("provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "symbol", symbol)
	if symbol == "" {
		return coreinstrument.Instrument{}, errb.New("inspect instrument requires symbol")
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return coreinstrument.Instrument{}, errb.Wrap(err)
	}
	records, err := queryRecords(ctx, client, query, true)
	if err != nil {
		return coreinstrument.Instrument{}, errb.Wrap(err)
	}
	if len(records) == 0 {
		return coreinstrument.Instrument{}, &instrumentservice.NotFoundError{
			Query:        symbol,
			Market:       query.Market,
			SecurityType: query.SecurityType,
		}
	}
	extensions, err := queryExtensions(ctx, client, records[0].InstrumentID, records[0].SourceID)
	if err != nil {
		return coreinstrument.Instrument{}, errb.Wrap(err)
	}
	return recordToCanonical(records[0], extensions), nil
}

type upsertCache struct {
	markets     map[string]storage.MarketV2Row
	instruments map[string]storage.InstrumentV2Row
	sources     map[string]int64
}

func ensureMarket(ctx context.Context, tx bun.Tx, cache upsertCache, market provider.Market, nowMS int64) (storage.MarketV2Row, error) {
	code := string(withDefaultMarket(market))
	if row, ok := cache.markets[code]; ok {
		return row, nil
	}
	row := storage.MarketV2Row{
		Code:               code,
		Timezone:           marketTimezone(provider.Market(code)),
		RegularOpenMinute:  marketRegularOpenMinute(provider.Market(code)),
		RegularCloseMinute: marketRegularCloseMinute(provider.Market(code)),
		CreatedAtMS:        nowMS,
		UpdatedAtMS:        nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (code) DO UPDATE").
		Set("timezone = EXCLUDED.timezone").
		Set("regular_open_minute = EXCLUDED.regular_open_minute").
		Set("regular_close_minute = EXCLUDED.regular_close_minute").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("instrument_repository").With("market", code).Wrapf(err, "upsert market")
	}
	var stored storage.MarketV2Row
	if err := tx.NewSelect().Model(&stored).Where("code = ?", code).Limit(1).Scan(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("instrument_repository").With("market", code).Wrapf(err, "select market")
	}
	cache.markets[code] = stored
	return stored, nil
}

func ensureInstrument(ctx context.Context, tx bun.Tx, cache upsertCache, marketID int64, item coreinstrument.Instrument, nowMS int64) (storage.InstrumentV2Row, error) {
	symbol := canonicalSymbol(item)
	key := strconv.FormatInt(marketID, 10) + "\x00" + string(item.SecurityType) + "\x00" + symbol
	if row, ok := cache.instruments[key]; ok {
		return row, nil
	}
	row := storage.InstrumentV2Row{
		MarketID:     marketID,
		SecurityType: string(item.SecurityType),
		Symbol:       symbol,
		ISIN:         strings.TrimSpace(item.ISIN),
		Name:         strings.TrimSpace(item.Name),
		CurrencyCode: normalizedCurrencyCode("KRW"),
		PriceScale:   defaultPriceScale,
		CreatedAtMS:  nowMS,
		UpdatedAtMS:  nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (market_id, security_type, symbol) DO UPDATE").
		Set("isin = EXCLUDED.isin").
		Set("name = EXCLUDED.name").
		Set("currency_code = EXCLUDED.currency_code").
		Set("price_scale = EXCLUDED.price_scale").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("instrument_repository").With("market_id", marketID, "security_type", item.SecurityType, "symbol", symbol).Wrapf(err, "upsert instrument")
	}
	var stored storage.InstrumentV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("market_id = ?", marketID).
		Where("security_type = ?", string(item.SecurityType)).
		Where("symbol = ?", symbol).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("instrument_repository").With("market_id", marketID, "security_type", item.SecurityType, "symbol", symbol).Wrapf(err, "select instrument")
	}
	cache.instruments[key] = stored
	return stored, nil
}

func ensureSource(ctx context.Context, tx bun.Tx, cache upsertCache, instrumentID int64, item coreinstrument.Instrument, nowMS int64) (int64, error) {
	providerSymbol := providerSymbol(item)
	key := sourceNaturalKey(item)
	if id, ok := cache.sources[key]; ok {
		return id, nil
	}
	row := storage.InstrumentSourceV1Row{
		Provider:       string(item.Provider),
		ProviderGroup:  string(item.Group),
		Operation:      string(item.Operation),
		ProviderSymbol: providerSymbol,
		InstrumentID:   instrumentID,
		CreatedAtMS:    nowMS,
		UpdatedAtMS:    nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation, provider_symbol) DO UPDATE").
		Set("instrument_id = EXCLUDED.instrument_id").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return 0, oops.In("instrument_repository").With("provider", item.Provider, "group", item.Group, "operation", item.Operation, "provider_symbol", providerSymbol).Wrapf(err, "upsert instrument source")
	}
	var stored storage.InstrumentSourceV1Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("provider = ?", string(item.Provider)).
		Where("provider_group = ?", string(item.Group)).
		Where("operation = ?", string(item.Operation)).
		Where("provider_symbol = ?", providerSymbol).
		Limit(1).
		Scan(ctx); err != nil {
		return 0, oops.In("instrument_repository").With("provider", item.Provider, "group", item.Group, "operation", item.Operation, "provider_symbol", providerSymbol).Wrapf(err, "select instrument source")
	}
	cache.sources[key] = stored.ID
	return stored.ID, nil
}

func replaceExtensions(ctx context.Context, tx bun.Tx, instrumentID int64, sourceID int64, extensions map[string]string) error {
	errb := oops.In("instrument_repository").With("instrument_id", instrumentID, "source_id", sourceID)
	if _, err := tx.NewDelete().
		Model((*storage.InstrumentExtensionV1Row)(nil)).
		Where("instrument_id = ?", instrumentID).
		Where("source_id = ?", sourceID).
		Exec(ctx); err != nil {
		return errb.Wrapf(err, "delete instrument extensions")
	}
	for key, value := range extensions {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		row := storage.InstrumentExtensionV1Row{
			InstrumentID: instrumentID,
			SourceID:     sourceID,
			Key:          key,
			Value:        value,
		}
		if _, err := tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return errb.With("key", key).Wrapf(err, "insert instrument extension")
		}
	}
	return nil
}

type queryDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func queryRecords(ctx context.Context, db queryDB, query instrumentservice.Query, exact bool) ([]record, error) {
	sqlQuery, args := selectSQL(query, exact)
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, oops.In("instrument_repository").Wrapf(err, "query instruments")
	}
	defer rows.Close()
	records := make([]record, 0)
	for rows.Next() {
		var row record
		if err := rows.Scan(
			&row.InstrumentID,
			&row.SourceID,
			&row.Provider,
			&row.ProviderGroup,
			&row.Operation,
			&row.Market,
			&row.SecurityType,
			&row.Symbol,
			&row.ISIN,
			&row.Name,
			&row.CurrencyCode,
			&row.Timezone,
		); err != nil {
			return nil, oops.In("instrument_repository").Wrapf(err, "scan instrument")
		}
		records = append(records, row)
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("instrument_repository").Wrapf(err, "iterate instruments")
	}
	return records, nil
}

func selectSQL(query instrumentservice.Query, exact bool) (string, []any) {
	market := string(withDefaultMarket(query.Market))
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	var builder strings.Builder
	builder.WriteString(`SELECT
	i.id,
	s.id,
	s.provider,
	s.provider_group,
	s.operation,
	m.code,
	i.security_type,
	i.symbol,
	i.isin,
	i.name,
	i.currency_code,
	m.timezone
FROM instrument_v2 AS i
JOIN market_v2 AS m ON m.id = i.market_id
JOIN instrument_source_v1 AS s ON s.instrument_id = i.id
WHERE m.code = ?`)
	args := []any{market}
	if query.SecurityType != "" {
		builder.WriteString(" AND i.security_type = ?")
		args = append(args, string(query.SecurityType))
	}
	if query.ProviderID != "" {
		builder.WriteString(" AND s.provider = ?")
		args = append(args, string(query.ProviderID))
	}
	needle := strings.TrimSpace(query.Query)
	if exact {
		needle = strings.TrimSpace(query.Symbol)
		builder.WriteString(" AND (lower(i.symbol) = lower(?) OR lower(i.isin) = lower(?) OR lower(s.provider_symbol) = lower(?))")
		args = append(args, needle, needle, needle)
	} else {
		like := "%" + needle + "%"
		builder.WriteString(` AND (
	lower(i.symbol) = lower(?)
	OR lower(i.isin) = lower(?)
	OR lower(s.provider_symbol) = lower(?)
	OR i.name LIKE ?
	OR EXISTS (
		SELECT 1
		FROM instrument_extension_v1 AS e
		WHERE e.instrument_id = i.id
		  AND e.source_id = s.id
		  AND e.key IN ('issueName', 'issueEnglishName', 'listingDate')
		  AND e.value LIKE ?
	)
)`)
		args = append(args, needle, needle, needle, like, like)
	}
	builder.WriteString(" ORDER BY CASE WHEN lower(i.symbol) = lower(?) THEN 0 WHEN lower(i.isin) = lower(?) THEN 1 ELSE 2 END, i.symbol ASC, s.provider ASC, s.operation ASC LIMIT ?")
	args = append(args, needle, needle, limit)
	return builder.String(), args
}

func recordsToSearchResult(ctx context.Context, db queryDB, records []record) (coreinstrument.SearchResult, error) {
	instruments := make([]coreinstrument.Instrument, 0, len(records))
	operations := make([]provider.OperationID, 0)
	seenOperations := make(map[provider.OperationID]bool)
	var identity provider.Identity
	var group provider.GroupID
	for _, row := range records {
		extensions, err := queryExtensions(ctx, db, row.InstrumentID, row.SourceID)
		if err != nil {
			return coreinstrument.SearchResult{}, err
		}
		instruments = append(instruments, recordToCanonical(row, extensions))
		if identity.ID == "" {
			identity = provider.Identity{ID: provider.ProviderID(row.Provider)}
		}
		if group == "" {
			group = provider.GroupID(row.ProviderGroup)
		}
		operation := provider.OperationID(row.Operation)
		if !seenOperations[operation] {
			seenOperations[operation] = true
			operations = append(operations, operation)
		}
	}
	return coreinstrument.SearchResult{
		Instruments: instruments,
		Provider:    identity,
		Group:       group,
		Operations:  operations,
		TotalCount:  len(instruments),
	}, nil
}

func queryExtensions(ctx context.Context, db queryDB, instrumentID int64, sourceID int64) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM instrument_extension_v1 WHERE instrument_id = ? AND source_id = ? ORDER BY key ASC`, instrumentID, sourceID)
	if err != nil {
		return nil, oops.In("instrument_repository").With("instrument_id", instrumentID, "source_id", sourceID).Wrapf(err, "query instrument extensions")
	}
	defer rows.Close()
	extensions := make(map[string]string)
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, oops.In("instrument_repository").Wrapf(err, "scan instrument extension")
		}
		extensions[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("instrument_repository").Wrapf(err, "iterate instrument extensions")
	}
	if len(extensions) == 0 {
		return nil, nil
	}
	return extensions, nil
}
