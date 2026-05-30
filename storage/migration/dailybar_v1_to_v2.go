package migration

import (
	"context"
	"encoding/json"
	"strings"

	migrationcore "github.com/awuzag/mwosa/migration"
	provider "github.com/awuzag/mwosa/providers/core"
	coredailybar "github.com/awuzag/mwosa/providers/core/dailybar"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

const DailyBarV1ToV2ID = "daily_bar_v1_to_v2"

type clientProvider interface {
	Client(ctx context.Context) (*bun.DB, error)
}

type DailyBarV1ToV2Executor struct {
	database clientProvider
	writer   daily.WriteRepository
}

func NewDailyBarV1ToV2Executor(database clientProvider, writer daily.WriteRepository) (DailyBarV1ToV2Executor, error) {
	if database == nil {
		return DailyBarV1ToV2Executor{}, oops.In("daily_bar_v1_to_v2_migration").New("database is nil")
	}
	if writer == nil {
		return DailyBarV1ToV2Executor{}, oops.In("daily_bar_v1_to_v2_migration").New("daily bar writer is nil")
	}
	return DailyBarV1ToV2Executor{
		database: database,
		writer:   writer,
	}, nil
}

func NewDailyBarV1ToV2Definition(executor migrationcore.Executor) migrationcore.Definition {
	return migrationcore.Definition{
		ID:          DailyBarV1ToV2ID,
		Name:        "Daily bar v1 to v2",
		Resource:    "daily_bar",
		FromVersion: "1",
		ToVersion:   storage.DailyBarV2SchemaVersion,
		Description: "Copy wide daily_bar rows into normalized daily_bar_v2 tables",
		Executor:    executor,
	}
}

func (e DailyBarV1ToV2Executor) Apply(ctx context.Context) (int64, error) {
	errb := oops.In("daily_bar_v1_to_v2_migration")
	client, err := e.database.Client(ctx)
	if err != nil {
		return 0, errb.Wrap(err)
	}
	var rows []storage.DailyBarV1Row
	if err := client.NewSelect().
		Model(&rows).
		Order("trading_date ASC", "symbol ASC", "provider ASC", "provider_group ASC").
		Scan(ctx); err != nil {
		return 0, errb.Wrapf(err, "select daily bar v1 rows")
	}
	if len(rows) == 0 {
		return 0, nil
	}

	bars := make([]coredailybar.Bar, 0, len(rows))
	for i := range rows {
		bar, err := dailyBarV1ToCanonical(rows[i])
		if err != nil {
			return 0, errb.With("row_id", rows[i].ID).Wrap(err)
		}
		bars = append(bars, bar)
	}
	if _, err := e.writer.UpsertDailyBars(ctx, bars); err != nil {
		return 0, errb.Wrapf(err, "write daily bar v2 rows")
	}
	return int64(len(bars)), nil
}

func dailyBarV1ToCanonical(row storage.DailyBarV1Row) (coredailybar.Bar, error) {
	extensions, err := decodeExtensions(row.ExtensionsJSON)
	if err != nil {
		return coredailybar.Bar{}, err
	}
	return coredailybar.Bar{
		Provider:     provider.ProviderID(row.Provider),
		Group:        provider.GroupID(row.ProviderGroup),
		Operation:    provider.OperationID(row.Operation),
		Market:       provider.Market(row.Market),
		SecurityType: provider.SecurityType(row.SecurityType),
		Symbol:       row.Symbol,
		ISIN:         row.ISIN,
		Name:         row.Name,
		TradingDate:  row.TradingDate,
		Currency:     row.Currency,
		Open:         row.OpeningPrice,
		High:         row.HighestPrice,
		Low:          row.LowestPrice,
		Close:        row.ClosingPrice,
		Change:       row.PriceChangeFromPreviousClose,
		ChangeRate:   row.PriceChangeRateFromPreviousClose,
		Volume:       row.TradedVolume,
		TradedValue:  row.TradedAmount,
		MarketCap:    row.MarketCapitalization,
		Extensions:   extensions,
	}, nil
}

func decodeExtensions(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" || raw == "{}" {
		return nil, nil
	}
	var extensions map[string]string
	if err := json.Unmarshal([]byte(raw), &extensions); err != nil {
		return nil, oops.In("daily_bar_v1_to_v2_migration").With("raw", raw).Wrapf(err, "decode daily bar v1 extensions")
	}
	return extensions, nil
}
