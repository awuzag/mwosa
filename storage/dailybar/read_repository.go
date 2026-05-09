package dailybar

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	coredailybar "github.com/ev3rlit/mwosa/providers/core/dailybar"
	"github.com/ev3rlit/mwosa/service/daily"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
)

type readRepository struct {
	database *storage.Database
}

var _ daily.ReadRepository = (*readRepository)(nil)

func NewReadRepository(database *storage.Database) (daily.ReadRepository, error) {
	if database == nil {
		return nil, oops.In("dailybar_repository").New("daily bar repository database is nil")
	}
	return &readRepository{database: database}, nil
}

func NewRepositories(database *storage.Database) (daily.ReadRepository, daily.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("dailybar_repository").New("daily bar repository database is nil")
	}
	return &readRepository{database: database}, &writeRepository{database: database}, nil
}

func (r *readRepository) QueryDailyBars(ctx context.Context, query daily.Query) ([]coredailybar.Bar, error) {
	errb := oops.In("dailybar_repository").With(
		"market", query.Market,
		"security_type", query.SecurityType,
		"symbol", query.Symbol,
		"from", query.From,
		"to", query.To,
	)

	client, err := r.database.Client(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	sqlQuery, args, err := dailyBarV2SelectSQL(query, false)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	rows, err := client.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, errb.Wrapf(err, "query daily bars sqlite")
	}
	defer rows.Close()

	records := make([]dailyBarV2Record, 0)
	for rows.Next() {
		record, err := scanDailyBarV2Record(rows)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errb.Wrapf(err, "iterate daily bars sqlite")
	}

	extensions, err := queryDailyBarV2Extensions(ctx, client, query)
	if err != nil {
		return nil, errb.Wrap(err)
	}

	bars := make([]coredailybar.Bar, 0, len(records))
	for _, record := range records {
		bars = append(bars, dailyBarV2ToCanonical(record, extensions[dailyBarV2Key(record.InstrumentID, record.SourceID, record.TradingDate)]))
	}
	return bars, nil
}

func dailyBarV2SelectSQL(query daily.Query, extension bool) (string, []any, error) {
	market := query.Market
	if market == "" {
		market = provider.MarketKRX
	}

	var builder strings.Builder
	if extension {
		builder.WriteString(`SELECT e.instrument_id, e.source_id, e.trading_date, e.key, e.value
FROM daily_bar_extension_v2 AS e
JOIN daily_bar_v2 AS b ON b.instrument_id = e.instrument_id AND b.source_id = e.source_id AND b.trading_date = e.trading_date
JOIN instrument_v2 AS i ON i.id = b.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
JOIN provider_source_v2 AS s ON s.id = b.source_id`)
	} else {
		builder.WriteString(`SELECT
	b.instrument_id,
	b.source_id,
	b.trading_date,
	b.open_price,
	b.high_price,
	b.low_price,
	b.close_price,
	b.change_price,
	b.change_rate_bp,
	b.volume,
	b.traded_amount_minor,
	b.market_cap_minor,
	b.nav_price,
	b.listed_shares,
	b.net_asset_minor,
	m.code AS market,
	i.security_type,
	i.symbol,
	i.isin,
	i.name,
	i.currency_code,
	i.price_scale,
	s.provider,
	s.provider_group,
	s.operation
FROM daily_bar_v2 AS b
JOIN instrument_v2 AS i ON i.id = b.instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
JOIN provider_source_v2 AS s ON s.id = b.source_id`)
	}

	args := []any{string(market)}
	builder.WriteString(" WHERE m.code = ?")
	if query.SecurityType != "" {
		builder.WriteString(" AND i.security_type = ?")
		args = append(args, string(query.SecurityType))
	}
	if query.Symbol != "" {
		builder.WriteString(" AND i.symbol = ?")
		args = append(args, query.Symbol)
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
		builder.WriteString(" ORDER BY e.trading_date ASC, i.symbol ASC, s.provider ASC, s.provider_group ASC, e.key ASC")
	} else {
		builder.WriteString(" ORDER BY b.trading_date ASC, i.symbol ASC, s.provider ASC, s.provider_group ASC")
	}
	return builder.String(), args, nil
}

func scanDailyBarV2Record(rows *sql.Rows) (dailyBarV2Record, error) {
	var record dailyBarV2Record
	if err := rows.Scan(
		&record.InstrumentID,
		&record.SourceID,
		&record.TradingDate,
		&record.OpenPrice,
		&record.HighPrice,
		&record.LowPrice,
		&record.ClosePrice,
		&record.ChangePrice,
		&record.ChangeRateBP,
		&record.Volume,
		&record.TradedAmountMinor,
		&record.MarketCapMinor,
		&record.NAVPrice,
		&record.ListedShares,
		&record.NetAssetMinor,
		&record.Market,
		&record.SecurityType,
		&record.Symbol,
		&record.ISIN,
		&record.Name,
		&record.CurrencyCode,
		&record.PriceScale,
		&record.Provider,
		&record.ProviderGroup,
		&record.Operation,
	); err != nil {
		return dailyBarV2Record{}, oops.In("dailybar_repository").Wrapf(err, "scan daily bar v2 row")
	}
	return record, nil
}

func queryDailyBarV2Extensions(ctx context.Context, db bunDB, query daily.Query) (map[string]map[string]string, error) {
	sqlQuery, args, err := dailyBarV2SelectSQL(query, true)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, oops.In("dailybar_repository").Wrapf(err, "query daily bar v2 extensions")
	}
	defer rows.Close()

	extensions := make(map[string]map[string]string)
	for rows.Next() {
		var instrumentID int64
		var sourceID int64
		var tradingDate int
		var key string
		var value string
		if err := rows.Scan(&instrumentID, &sourceID, &tradingDate, &key, &value); err != nil {
			return nil, oops.In("dailybar_repository").Wrapf(err, "scan daily bar v2 extension")
		}
		barKey := dailyBarV2Key(instrumentID, sourceID, tradingDate)
		if extensions[barKey] == nil {
			extensions[barKey] = make(map[string]string)
		}
		extensions[barKey][key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("dailybar_repository").Wrapf(err, "iterate daily bar v2 extensions")
	}
	return extensions, nil
}

type bunDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func dailyBarV2Key(instrumentID, sourceID int64, tradingDate int) string {
	return strconvFormatInt(instrumentID) + "\x00" + strconvFormatInt(sourceID) + "\x00" + strconvFormatInt(int64(tradingDate))
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
