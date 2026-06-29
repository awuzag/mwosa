package dailybar

import (
	"context"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	coredailybar "github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

type writeRepository struct {
	database *storage.SQLDatabase
}

var _ daily.WriteRepository = (*writeRepository)(nil)

func NewWriteRepository(database *storage.SQLDatabase) (daily.WriteRepository, error) {
	if database == nil {
		return nil, oops.In("dailybar_repository").New("daily bar repository database is nil")
	}
	return &writeRepository{database: database}, nil
}

func (r *writeRepository) UpsertDailyBars(ctx context.Context, bars []coredailybar.Bar) (daily.WriteResult, error) {
	errb := oops.In("dailybar_repository").With("bars", len(bars))

	client, err := r.database.Client(ctx)
	if err != nil {
		return daily.WriteResult{}, errb.Wrap(err)
	}

	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return daily.WriteResult{}, errb.Wrapf(err, "begin daily bar sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cache := dailyBarV2UpsertCache{
		markets:     make(map[string]storage.MarketV2Row),
		sources:     make(map[string]int64),
		instruments: make(map[string]storage.InstrumentV2Row),
	}
	for _, bar := range bars {
		barErrb := errb.With(
			"provider", bar.Provider,
			"group", bar.Group,
			"market", bar.Market,
			"security_type", bar.SecurityType,
			"date", bar.TradingDate,
			"symbol", bar.Symbol,
		)

		if err := validateBarKey(bar); err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
		nowMS := time.Now().UTC().UnixMilli()
		market, err := ensureMarketV2(ctx, tx, cache, string(bar.Market), nowMS)
		if err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
		sourceID, err := ensureProviderSourceV2(ctx, tx, cache, bar, nowMS)
		if err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
		instrument, err := ensureInstrumentV2(ctx, tx, cache, market.ID, bar, nowMS)
		if err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
		amountScale := currencyMinorScale(instrument.CurrencyCode)
		row, err := dailyBarToV2Row(bar, instrument.ID, sourceID, instrument.PriceScale, amountScale, nowMS)
		if err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
		if _, err := tx.NewInsert().
			Model(&row).
			On("CONFLICT (instrument_id, source_id, trading_date) DO UPDATE").
			Set("schema_version = EXCLUDED.schema_version").
			Set("open_price = EXCLUDED.open_price").
			Set("high_price = EXCLUDED.high_price").
			Set("low_price = EXCLUDED.low_price").
			Set("close_price = EXCLUDED.close_price").
			Set("change_price = EXCLUDED.change_price").
			Set("change_rate_bp = EXCLUDED.change_rate_bp").
			Set("volume = EXCLUDED.volume").
			Set("traded_amount_minor = EXCLUDED.traded_amount_minor").
			Set("market_cap_minor = EXCLUDED.market_cap_minor").
			Set("nav_price = EXCLUDED.nav_price").
			Set("listed_shares = EXCLUDED.listed_shares").
			Set("net_asset_minor = EXCLUDED.net_asset_minor").
			Set("updated_at_ms = EXCLUDED.updated_at_ms").
			Exec(ctx); err != nil {
			return daily.WriteResult{}, barErrb.Wrapf(err, "upsert daily bar v2 sqlite row")
		}
		if err := replaceDailyBarExtensionsV2(ctx, tx, row, bar.Extensions); err != nil {
			return daily.WriteResult{}, barErrb.Wrap(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return daily.WriteResult{}, errb.Wrapf(err, "commit daily bar sqlite transaction")
	}
	committed = true

	return daily.WriteResult{
		BarsWritten:  len(bars),
		RowsAffected: len(bars),
	}, nil
}

type dailyBarV2UpsertCache struct {
	markets     map[string]storage.MarketV2Row
	sources     map[string]int64
	instruments map[string]storage.InstrumentV2Row
}

func ensureMarketV2(ctx context.Context, tx bun.Tx, cache dailyBarV2UpsertCache, marketCode string, nowMS int64) (storage.MarketV2Row, error) {
	code := strings.TrimSpace(marketCode)
	if code == "" {
		code = string(provider.MarketKRX)
	}
	if row, ok := cache.markets[code]; ok {
		return row, nil
	}
	row := storage.MarketV2Row{
		Code:               code,
		Timezone:           marketTimezone(code),
		RegularOpenMinute:  marketRegularOpenMinute(code),
		RegularCloseMinute: marketRegularCloseMinute(code),
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
		return storage.MarketV2Row{}, oops.In("dailybar_repository").With("market", code).Wrapf(err, "upsert market v2")
	}
	var stored storage.MarketV2Row
	if err := tx.NewSelect().Model(&stored).Where("code = ?", code).Limit(1).Scan(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("dailybar_repository").With("market", code).Wrapf(err, "select market v2")
	}
	cache.markets[code] = stored
	return stored, nil
}

func ensureProviderSourceV2(ctx context.Context, tx bun.Tx, cache dailyBarV2UpsertCache, bar coredailybar.Bar, nowMS int64) (int64, error) {
	key := string(bar.Provider) + "\x00" + string(bar.Group) + "\x00" + string(bar.Operation)
	if id, ok := cache.sources[key]; ok {
		return id, nil
	}
	row := storage.ProviderSourceV2Row{
		Provider:      string(bar.Provider),
		ProviderGroup: string(bar.Group),
		Operation:     string(bar.Operation),
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation) DO UPDATE").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return 0, oops.In("dailybar_repository").With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation).Wrapf(err, "upsert provider source v2")
	}
	var stored storage.ProviderSourceV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("provider = ?", string(bar.Provider)).
		Where("provider_group = ?", string(bar.Group)).
		Where("operation = ?", string(bar.Operation)).
		Limit(1).
		Scan(ctx); err != nil {
		return 0, oops.In("dailybar_repository").With("provider", bar.Provider, "group", bar.Group, "operation", bar.Operation).Wrapf(err, "select provider source v2")
	}
	cache.sources[key] = stored.ID
	return stored.ID, nil
}

func ensureInstrumentV2(ctx context.Context, tx bun.Tx, cache dailyBarV2UpsertCache, marketID int64, bar coredailybar.Bar, nowMS int64) (storage.InstrumentV2Row, error) {
	currencyCode := normalizedCurrencyCode(bar.Currency)
	key := strconv.FormatInt(marketID, 10) + "\x00" + string(bar.SecurityType) + "\x00" + bar.Symbol
	if row, ok := cache.instruments[key]; ok {
		return row, nil
	}
	row := storage.InstrumentV2Row{
		MarketID:     marketID,
		SecurityType: string(bar.SecurityType),
		Symbol:       bar.Symbol,
		ISIN:         bar.ISIN,
		Name:         bar.Name,
		CurrencyCode: currencyCode,
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
		return storage.InstrumentV2Row{}, oops.In("dailybar_repository").With("market_id", marketID, "security_type", bar.SecurityType, "symbol", bar.Symbol).Wrapf(err, "upsert instrument v2")
	}
	var stored storage.InstrumentV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("market_id = ?", marketID).
		Where("security_type = ?", string(bar.SecurityType)).
		Where("symbol = ?", bar.Symbol).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("dailybar_repository").With("market_id", marketID, "security_type", bar.SecurityType, "symbol", bar.Symbol).Wrapf(err, "select instrument v2")
	}
	cache.instruments[key] = stored
	return stored, nil
}

func replaceDailyBarExtensionsV2(ctx context.Context, tx bun.Tx, row storage.DailyBarV2Row, extensions map[string]string) error {
	errb := oops.In("dailybar_repository").With("instrument_id", row.InstrumentID, "source_id", row.SourceID, "date", row.TradingDate)
	if _, err := tx.NewDelete().
		Model((*storage.DailyBarExtensionV2Row)(nil)).
		Where("instrument_id = ?", row.InstrumentID).
		Where("source_id = ?", row.SourceID).
		Where("trading_date = ?", row.TradingDate).
		Exec(ctx); err != nil {
		return errb.Wrapf(err, "delete daily bar v2 extensions")
	}
	for key, value := range extensions {
		if isPromotedDailyBarExtensionKey(key) {
			continue
		}
		extensionRow := storage.DailyBarExtensionV2Row{
			InstrumentID: row.InstrumentID,
			SourceID:     row.SourceID,
			TradingDate:  row.TradingDate,
			Key:          key,
			Value:        value,
		}
		if _, err := tx.NewInsert().Model(&extensionRow).Exec(ctx); err != nil {
			return errb.With("key", key).Wrapf(err, "insert daily bar v2 extension")
		}
	}
	return nil
}

func marketTimezone(code string) string {
	if code == string(provider.MarketKRX) {
		return "Asia/Seoul"
	}
	return "UTC"
}

func marketRegularOpenMinute(code string) int {
	if code == string(provider.MarketKRX) {
		return 9 * 60
	}
	return 0
}

func marketRegularCloseMinute(code string) int {
	if code == string(provider.MarketKRX) {
		return 15*60 + 30
	}
	return 0
}
