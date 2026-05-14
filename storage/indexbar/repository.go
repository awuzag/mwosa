package indexbar

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coreindexbar "github.com/ev3rlit/mwosa/providers/core/indexbar"
	indexservice "github.com/ev3rlit/mwosa/service/index"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type repository struct {
	database *storage.Database
}

var _ indexservice.ReadRepository = (*repository)(nil)
var _ indexservice.WriteRepository = (*repository)(nil)

func NewRepository(database *storage.Database) (indexservice.ReadRepository, indexservice.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("indexbar_repository").New("index bar repository database is nil")
	}
	repo := &repository{database: database}
	return repo, repo, nil
}

func (r *repository) QueryIndexBars(ctx context.Context, query indexservice.Query) ([]coreindexbar.Bar, error) {
	errb := oops.In("indexbar_repository").With("market", query.Market, "index_code", query.IndexCode, "from", query.From, "to", query.To)
	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	sqlQuery, args, err := indexBarSelectSQL(query, false)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows, err := client.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, errb.Wrapf(err, "query index bars sqlite")
	}
	defer rows.Close()

	records := make([]indexBarRecord, 0)
	indexExtensions := make(map[int64]map[string]string)
	for rows.Next() {
		record, err := scanIndexBarRecord(rows)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		records = append(records, record)
		if _, ok := indexExtensions[record.IndexID]; !ok {
			extensions, err := decodeExtensions(record.IndexExtension)
			if err != nil {
				return nil, errb.With("index_id", record.IndexID).Wrap(err)
			}
			indexExtensions[record.IndexID] = extensions
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errb.Wrapf(err, "iterate index bars sqlite")
	}

	barExtensions, err := queryIndexBarExtensions(ctx, client, query)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	bars := make([]coreindexbar.Bar, 0, len(records))
	for _, record := range records {
		bars = append(bars, recordToCanonical(record, indexExtensions[record.IndexID], barExtensions[indexBarKey(record.IndexID, record.SourceID, record.TradingDate)]))
	}
	return bars, nil
}

func (r *repository) UpsertIndexBars(ctx context.Context, bars []coreindexbar.Bar) (indexservice.WriteResult, error) {
	errb := oops.In("indexbar_repository").With("bars", len(bars))
	client, err := r.database.Client(ctx)
	if err != nil {
		return indexservice.WriteResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return indexservice.WriteResult{}, errb.Wrapf(err, "begin index bar sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cache := upsertCache{
		indexes: make(map[string]storage.IndexV1Row),
		sources: make(map[string]int64),
	}
	for _, bar := range bars {
		barErrb := errb.With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation, "market", bar.Market, "index_code", bar.IndexCode, "date", bar.TradingDate)
		if err := validateBarKey(bar); err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		nowMS := time.Now().UTC().UnixMilli()
		indexRow, err := ensureIndex(ctx, tx, cache, bar, nowMS)
		if err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		sourceID, err := ensureSource(ctx, tx, cache, indexRow.ID, bar, nowMS)
		if err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		row, err := indexBarToRow(bar, indexRow.ID, sourceID, nowMS)
		if err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
		if _, err := tx.NewInsert().
			Model(&row).
			On("CONFLICT (index_id, source_id, trading_date) DO UPDATE").
			Set("schema_version = EXCLUDED.schema_version").
			Set("open_value = EXCLUDED.open_value").
			Set("high_value = EXCLUDED.high_value").
			Set("low_value = EXCLUDED.low_value").
			Set("close_value = EXCLUDED.close_value").
			Set("change_value = EXCLUDED.change_value").
			Set("change_rate_bp = EXCLUDED.change_rate_bp").
			Set("volume = EXCLUDED.volume").
			Set("traded_amount_minor = EXCLUDED.traded_amount_minor").
			Set("market_cap_minor = EXCLUDED.market_cap_minor").
			Set("updated_at_ms = EXCLUDED.updated_at_ms").
			Exec(ctx); err != nil {
			return indexservice.WriteResult{}, barErrb.Wrapf(err, "upsert index bar sqlite row")
		}
		if err := replaceExtensions(ctx, tx, row, bar.Extensions); err != nil {
			return indexservice.WriteResult{}, barErrb.Wrap(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return indexservice.WriteResult{}, errb.Wrapf(err, "commit index bar sqlite transaction")
	}
	committed = true

	return indexservice.WriteResult{BarsWritten: len(bars), RowsAffected: len(bars)}, nil
}

type upsertCache struct {
	indexes map[string]storage.IndexV1Row
	sources map[string]int64
}

func ensureIndex(ctx context.Context, tx bun.Tx, cache upsertCache, bar coreindexbar.Bar, nowMS int64) (storage.IndexV1Row, error) {
	market := withDefaultMarket(bar.Market)
	indexCode := strings.TrimSpace(bar.IndexCode)
	key := string(market) + "\x00" + indexCode
	if row, ok := cache.indexes[key]; ok {
		return row, nil
	}
	extensions, err := encodeExtensions(nil)
	if err != nil {
		return storage.IndexV1Row{}, err
	}
	row := storage.IndexV1Row{
		Market:         string(market),
		IndexCode:      indexCode,
		Name:           strings.TrimSpace(bar.Name),
		Family:         strings.TrimSpace(bar.Family),
		CountryCode:    countryCode(market),
		CurrencyCode:   normalizedCurrencyCode(bar.Currency),
		Timezone:       marketTimezone(market),
		IndexType:      "price",
		ExtensionsJSON: extensions,
		CreatedAtMS:    nowMS,
		UpdatedAtMS:    nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (market, index_code) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("family = EXCLUDED.family").
		Set("country_code = EXCLUDED.country_code").
		Set("currency_code = EXCLUDED.currency_code").
		Set("timezone = EXCLUDED.timezone").
		Set("index_type = EXCLUDED.index_type").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.IndexV1Row{}, oops.In("indexbar_repository").With("market", market, "index_code", indexCode).Wrapf(err, "upsert index")
	}
	var stored storage.IndexV1Row
	if err := tx.NewSelect().Model(&stored).Where("market = ?", string(market)).Where("index_code = ?", indexCode).Limit(1).Scan(ctx); err != nil {
		return storage.IndexV1Row{}, oops.In("indexbar_repository").With("market", market, "index_code", indexCode).Wrapf(err, "select index")
	}
	cache.indexes[key] = stored
	return stored, nil
}

func ensureSource(ctx context.Context, tx bun.Tx, cache upsertCache, indexID int64, bar coreindexbar.Bar, nowMS int64) (int64, error) {
	providerSymbol := strings.TrimSpace(bar.IndexCode)
	key := string(bar.Provider) + "\x00" + string(bar.Group) + "\x00" + string(bar.Operation) + "\x00" + providerSymbol
	if id, ok := cache.sources[key]; ok {
		return id, nil
	}
	row := storage.IndexSourceV1Row{
		Provider:       string(bar.Provider),
		ProviderGroup:  string(bar.Group),
		Operation:      string(bar.Operation),
		ProviderSymbol: providerSymbol,
		IndexID:        indexID,
		CreatedAtMS:    nowMS,
		UpdatedAtMS:    nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation, provider_symbol) DO UPDATE").
		Set("index_id = EXCLUDED.index_id").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return 0, oops.In("indexbar_repository").With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation, "provider_symbol", providerSymbol).Wrapf(err, "upsert index source")
	}
	var stored storage.IndexSourceV1Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("provider = ?", string(bar.Provider)).
		Where("provider_group = ?", string(bar.Group)).
		Where("operation = ?", string(bar.Operation)).
		Where("provider_symbol = ?", providerSymbol).
		Limit(1).
		Scan(ctx); err != nil {
		return 0, oops.In("indexbar_repository").With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation, "provider_symbol", providerSymbol).Wrapf(err, "select index source")
	}
	cache.sources[key] = stored.ID
	return stored.ID, nil
}

func indexBarSelectSQL(query indexservice.Query, extension bool) (string, []any, error) {
	market := withDefaultMarket(query.Market)
	var builder strings.Builder
	if extension {
		builder.WriteString(`SELECT e.index_id, e.source_id, e.trading_date, e.key, e.value
FROM index_bar_extension_v1 AS e
JOIN index_bar_v1 AS b ON b.index_id = e.index_id AND b.source_id = e.source_id AND b.trading_date = e.trading_date
JOIN index_v1 AS i ON i.id = b.index_id
JOIN index_source_v1 AS s ON s.id = b.source_id`)
	} else {
		builder.WriteString(`SELECT
	b.index_id,
	b.source_id,
	b.trading_date,
	b.open_value,
	b.high_value,
	b.low_value,
	b.close_value,
	b.change_value,
	b.change_rate_bp,
	b.volume,
	b.traded_amount_minor,
	b.market_cap_minor,
	i.market,
	i.index_code,
	i.name,
	i.family,
	i.currency_code,
	i.timezone,
	i.index_type,
	i.extensions_json,
	s.provider,
	s.provider_group,
	s.operation
FROM index_bar_v1 AS b
JOIN index_v1 AS i ON i.id = b.index_id
JOIN index_source_v1 AS s ON s.id = b.source_id`)
	}
	args := []any{string(market)}
	builder.WriteString(" WHERE i.market = ?")
	if query.IndexCode != "" {
		builder.WriteString(" AND i.index_code = ?")
		args = append(args, strings.TrimSpace(query.IndexCode))
	}
	if query.From != "" {
		from, err := parseTradingDate(query.From)
		if err != nil {
			return "", nil, err
		}
		builder.WriteString(" AND b.trading_date >= ?")
		args = append(args, from)
	}
	if query.To != "" {
		to, err := parseTradingDate(query.To)
		if err != nil {
			return "", nil, err
		}
		builder.WriteString(" AND b.trading_date <= ?")
		args = append(args, to)
	}
	if extension {
		builder.WriteString(" ORDER BY e.trading_date ASC, i.index_code ASC, s.provider ASC, s.provider_group ASC, e.key ASC")
	} else {
		builder.WriteString(" ORDER BY b.trading_date ASC, i.index_code ASC, s.provider ASC, s.provider_group ASC")
	}
	return builder.String(), args, nil
}

func scanIndexBarRecord(rows *sql.Rows) (indexBarRecord, error) {
	var record indexBarRecord
	if err := rows.Scan(
		&record.IndexID,
		&record.SourceID,
		&record.TradingDate,
		&record.OpenValue,
		&record.HighValue,
		&record.LowValue,
		&record.CloseValue,
		&record.ChangeValue,
		&record.ChangeRateBP,
		&record.Volume,
		&record.TradedAmountMinor,
		&record.MarketCapMinor,
		&record.Market,
		&record.IndexCode,
		&record.Name,
		&record.Family,
		&record.CurrencyCode,
		&record.Timezone,
		&record.IndexType,
		&record.IndexExtension,
		&record.Provider,
		&record.ProviderGroup,
		&record.Operation,
	); err != nil {
		return indexBarRecord{}, oops.In("indexbar_repository").Wrapf(err, "scan index bar row")
	}
	return record, nil
}

type bunDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func queryIndexBarExtensions(ctx context.Context, db bunDB, query indexservice.Query) (map[string]map[string]string, error) {
	sqlQuery, args, err := indexBarSelectSQL(query, true)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, oops.In("indexbar_repository").Wrapf(err, "query index bar extensions")
	}
	defer rows.Close()

	extensions := make(map[string]map[string]string)
	for rows.Next() {
		var indexID int64
		var sourceID int64
		var tradingDate int
		var key string
		var value string
		if err := rows.Scan(&indexID, &sourceID, &tradingDate, &key, &value); err != nil {
			return nil, oops.In("indexbar_repository").Wrapf(err, "scan index bar extension")
		}
		barKey := indexBarKey(indexID, sourceID, tradingDate)
		if extensions[barKey] == nil {
			extensions[barKey] = make(map[string]string)
		}
		extensions[barKey][key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("indexbar_repository").Wrapf(err, "iterate index bar extensions")
	}
	return extensions, nil
}

func replaceExtensions(ctx context.Context, tx bun.Tx, row storage.IndexBarV1Row, extensions map[string]string) error {
	errb := oops.In("indexbar_repository").With("index_id", row.IndexID, "source_id", row.SourceID, "date", row.TradingDate)
	if _, err := tx.NewDelete().
		Model((*storage.IndexBarExtensionV1Row)(nil)).
		Where("index_id = ?", row.IndexID).
		Where("source_id = ?", row.SourceID).
		Where("trading_date = ?", row.TradingDate).
		Exec(ctx); err != nil {
		return errb.Wrapf(err, "delete index bar extensions")
	}
	for key, value := range extensions {
		extensionRow := storage.IndexBarExtensionV1Row{
			IndexID:     row.IndexID,
			SourceID:    row.SourceID,
			TradingDate: row.TradingDate,
			Key:         key,
			Value:       value,
		}
		if _, err := tx.NewInsert().Model(&extensionRow).Exec(ctx); err != nil {
			return errb.With("key", key).Wrapf(err, "insert index bar extension")
		}
	}
	return nil
}

func withDefaultMarket(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func countryCode(market provider.Market) string {
	if market == provider.MarketKRX {
		return "KR"
	}
	return ""
}

func marketTimezone(market provider.Market) string {
	if market == provider.MarketKRX {
		return "Asia/Seoul"
	}
	return "UTC"
}

func indexBarKey(indexID, sourceID int64, tradingDate int) string {
	return strconv.FormatInt(indexID, 10) + "\x00" + strconv.FormatInt(sourceID, 10) + "\x00" + strconv.FormatInt(int64(tradingDate), 10)
}
