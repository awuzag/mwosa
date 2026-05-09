package sqlite_capacity_runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/samber/oops"
	_ "modernc.org/sqlite"
)

const minuteBarStorageSizeProbeConfigJSON = `
{
  "enabled": true,
  "output_dir": "tmp/testing/sqlite-capacity-runtime/minute-bars",
  "clean_output_dir": true,
  "symbols": 1,
  "trading_days": [1, 5, 20],
  "minutes_per_day": 390,
  "timeout_seconds": 60
}
`

type minuteBarStorageSizeProbeConfig struct {
	Enabled        bool   `json:"enabled"`
	OutputDir      string `json:"output_dir"`
	CleanOutputDir bool   `json:"clean_output_dir"`
	Symbols        int    `json:"symbols"`
	TradingDays    []int  `json:"trading_days"`
	MinutesPerDay  int    `json:"minutes_per_day"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type barStorageSizeResult struct {
	Kind               string
	TradingDays        int
	Rows               int
	EmptyBytes         int64
	FileBytes          int64
	PayloadBytes       int64
	PayloadBytesPerDay float64
	PayloadBytesPerRow float64
	BuildElapsed       time.Duration
}

func TestMinuteBarStorageSizeProbe(t *testing.T) {
	config := loadMinuteBarStorageSizeProbeConfig(t)
	if !config.Enabled {
		t.Skip("set enabled=true in minuteBarStorageSizeProbeConfigJSON to measure daily/minute SQLite storage size")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	outputDir := resolveProbeOutputDir(t, config.OutputDir, "minute-bars")
	if config.CleanOutputDir {
		if err := os.RemoveAll(outputDir); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dailyEmptyBytes := buildEmptyBarStorageDB(ctx, t, filepath.Join(outputDir, "daily-empty.db"), createDailyLikeSchema)
	minuteEmptyBytes := buildEmptyBarStorageDB(ctx, t, filepath.Join(outputDir, "minute-empty.db"), createMinuteLikeSchema)

	results := make([]barStorageSizeResult, 0, len(config.TradingDays)*2)
	for _, tradingDays := range config.TradingDays {
		if tradingDays <= 0 {
			continue
		}
		dailyPath := filepath.Join(outputDir, fmt.Sprintf("daily-like-%dd.db", tradingDays))
		dailyRows, dailyElapsed := buildSyntheticDailyLikeDB(ctx, t, dailyPath, config.Symbols, tradingDays)
		dailyBytes, err := fileSize(dailyPath)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, newBarStorageSizeResult("daily_like", tradingDays, config.Symbols, dailyRows, dailyEmptyBytes, dailyBytes, dailyElapsed))

		minutePath := filepath.Join(outputDir, fmt.Sprintf("minute-like-%dd.db", tradingDays))
		minuteRows, minuteElapsed := buildSyntheticMinuteLikeDB(ctx, t, minutePath, config.Symbols, tradingDays, config.MinutesPerDay)
		minuteBytes, err := fileSize(minutePath)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, newBarStorageSizeResult("minute_like", tradingDays, config.Symbols, minuteRows, minuteEmptyBytes, minuteBytes, minuteElapsed))
	}

	resultsPath := filepath.Join(outputDir, "results.md")
	if err := writeMinuteBarStorageSizeResults(resultsPath, config, results); err != nil {
		t.Fatal(err)
	}

	for _, result := range results {
		t.Logf(
			"kind=%s days=%d rows=%d file_bytes=%d payload_bytes=%d payload_bytes_per_symbol_day=%.1f payload_bytes_per_row=%.1f build=%s",
			result.Kind,
			result.TradingDays,
			result.Rows,
			result.FileBytes,
			result.PayloadBytes,
			result.PayloadBytesPerDay,
			result.PayloadBytesPerRow,
			formatDuration(result.BuildElapsed),
		)
	}
	t.Logf("minute bar storage size results written to %s", resultsPath)
}

func loadMinuteBarStorageSizeProbeConfig(t *testing.T) minuteBarStorageSizeProbeConfig {
	t.Helper()

	var config minuteBarStorageSizeProbeConfig
	if err := json.Unmarshal([]byte(minuteBarStorageSizeProbeConfigJSON), &config); err != nil {
		t.Fatalf("decode minuteBarStorageSizeProbeConfigJSON: %v", err)
	}
	if config.Symbols <= 0 {
		config.Symbols = 1
	}
	if len(config.TradingDays) == 0 {
		config.TradingDays = []int{1, 5, 20}
	}
	if config.MinutesPerDay <= 0 {
		config.MinutesPerDay = 390
	}
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 60
	}
	return config
}

func newBarStorageSizeResult(kind string, tradingDays, symbols, rows int, emptyBytes, fileBytes int64, buildElapsed time.Duration) barStorageSizeResult {
	payloadBytes := fileBytes - emptyBytes
	if payloadBytes < 0 {
		payloadBytes = 0
	}
	result := barStorageSizeResult{
		Kind:         kind,
		TradingDays:  tradingDays,
		Rows:         rows,
		EmptyBytes:   emptyBytes,
		FileBytes:    fileBytes,
		PayloadBytes: payloadBytes,
		BuildElapsed: buildElapsed,
	}
	symbolDays := tradingDays * symbols
	if symbolDays > 0 {
		result.PayloadBytesPerDay = float64(payloadBytes) / float64(symbolDays)
	}
	if rows > 0 {
		result.PayloadBytesPerRow = float64(payloadBytes) / float64(rows)
	}
	return result
}

func buildEmptyBarStorageDB(ctx context.Context, t *testing.T, dbPath string, createSchema func(context.Context, *sql.DB) error) int64 {
	t.Helper()

	db := openProbeDB(ctx, t, dbPath)
	if err := createSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	closeCompactedProbeDB(ctx, t, db)

	bytes, err := fileSize(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func buildSyntheticDailyLikeDB(ctx context.Context, t *testing.T, dbPath string, symbols, tradingDays int) (int, time.Duration) {
	t.Helper()
	startedAt := time.Now()

	db := openProbeDB(ctx, t, dbPath)
	if err := createDailyLikeSchema(ctx, db); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO daily_bar (
		provider, provider_group, operation, market, security_type, symbol, isin, name, trading_date,
		currency, opening_price, highest_price, lowest_price, closing_price,
		price_change_from_previous_close, price_change_rate_from_previous_close,
		traded_volume, traded_amount, market_capitalization, extensions_json, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for dayIndex := range tradingDays {
		tradingDate := syntheticTradingDate(dayIndex)
		for symbolIndex := range symbols {
			symbol := syntheticSymbol(symbolIndex)
			closePrice := 35000 + symbolIndex%250*20 + dayIndex%120
			volume := int64(100_000 + (symbolIndex*1009+dayIndex*307)%8_000_000)
			tradedAmount := volume * int64(closePrice)
			if _, err := statement.ExecContext(ctx,
				"datago",
				"securitiesProductPrice",
				"getETFPriceInfo",
				"krx",
				"etf",
				symbol,
				"KR7"+symbol+"0000",
				"KODEX 200",
				tradingDate,
				"KRW",
				fmt.Sprint(closePrice-120),
				fmt.Sprint(closePrice+160),
				fmt.Sprint(closePrice-200),
				fmt.Sprint(closePrice),
				fmt.Sprint((dayIndex%21)-10),
				fmt.Sprintf("%.2f", float64((dayIndex%21)-10)/100),
				fmt.Sprint(volume),
				fmt.Sprint(tradedAmount),
				fmt.Sprint(tradedAmount*100),
				`{"nav":"35155.1","bssIdxIdxNm":"KOSPI 200","bssIdxClpr":"351.2"}`,
				"2026-05-08T00:00:00Z",
				"2026-05-08T00:00:00Z",
			); err != nil {
				t.Fatal(err)
			}
			rows++
		}
	}
	if err := statement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	closeCompactedProbeDB(ctx, t, db)
	return rows, time.Since(startedAt)
}

func buildSyntheticMinuteLikeDB(ctx context.Context, t *testing.T, dbPath string, symbols, tradingDays, minutesPerDay int) (int, time.Duration) {
	t.Helper()
	startedAt := time.Now()

	db := openProbeDB(ctx, t, dbPath)
	if err := createMinuteLikeSchema(ctx, db); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO minute_bar (
		provider, provider_group, operation, market, security_type, symbol, isin, name, trading_date, trading_minute,
		currency, opening_price, highest_price, lowest_price, closing_price,
		price_change_from_previous_close, price_change_rate_from_previous_close,
		traded_volume, traded_amount, market_capitalization, extensions_json, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for dayIndex := range tradingDays {
		tradingDate := syntheticTradingDate(dayIndex)
		for symbolIndex := range symbols {
			symbol := syntheticSymbol(symbolIndex)
			basePrice := 35000 + symbolIndex%250*20 + dayIndex%120
			for minuteIndex := range minutesPerDay {
				closePrice := basePrice + (minuteIndex%19 - 9)
				volume := int64(100 + (symbolIndex*1009+dayIndex*307+minuteIndex*17)%80_000)
				tradedAmount := volume * int64(closePrice)
				if _, err := statement.ExecContext(ctx,
					"datago",
					"securitiesProductPrice",
					"getETFMinutePriceInfo",
					"krx",
					"etf",
					symbol,
					"KR7"+symbol+"0000",
					"KODEX 200",
					tradingDate,
					syntheticTradingMinute(minuteIndex),
					"KRW",
					fmt.Sprint(closePrice-3),
					fmt.Sprint(closePrice+4),
					fmt.Sprint(closePrice-5),
					fmt.Sprint(closePrice),
					fmt.Sprint(closePrice-basePrice),
					fmt.Sprintf("%.4f", float64(closePrice-basePrice)/float64(basePrice)*100),
					fmt.Sprint(volume),
					fmt.Sprint(tradedAmount),
					"",
					"{}",
					"2026-05-08T00:00:00Z",
					"2026-05-08T00:00:00Z",
				); err != nil {
					t.Fatal(err)
				}
				rows++
			}
		}
	}
	if err := statement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	closeCompactedProbeDB(ctx, t, db)
	return rows, time.Since(startedAt)
}

func openProbeDB(ctx context.Context, t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"PRAGMA journal_mode=DELETE",
		"PRAGMA synchronous=OFF",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			_ = db.Close()
			t.Fatal(err)
		}
	}
	return db
}

func closeCompactedProbeDB(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func createDailyLikeSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE daily_bar (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		provider_group TEXT NOT NULL,
		operation TEXT NOT NULL,
		market TEXT NOT NULL,
		security_type TEXT NOT NULL,
		symbol TEXT NOT NULL,
		isin TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL DEFAULT '',
		trading_date TEXT NOT NULL,
		currency TEXT NOT NULL DEFAULT '',
		opening_price TEXT NOT NULL DEFAULT '',
		highest_price TEXT NOT NULL DEFAULT '',
		lowest_price TEXT NOT NULL DEFAULT '',
		closing_price TEXT NOT NULL DEFAULT '',
		price_change_from_previous_close TEXT NOT NULL DEFAULT '',
		price_change_rate_from_previous_close TEXT NOT NULL DEFAULT '',
		traded_volume TEXT NOT NULL DEFAULT '',
		traded_amount TEXT NOT NULL DEFAULT '',
		market_capitalization TEXT NOT NULL DEFAULT '',
		extensions_json TEXT NOT NULL DEFAULT '{}',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	return createBarIndexes(ctx, db, "daily_bar", []string{
		"CREATE UNIQUE INDEX daily_bar_natural_key ON daily_bar (market, security_type, trading_date, symbol, provider, provider_group)",
		"CREATE INDEX idx_daily_bar_date ON daily_bar (market, security_type, trading_date)",
		"CREATE INDEX idx_daily_bar_symbol_date ON daily_bar (market, security_type, symbol, trading_date)",
	})
}

func createMinuteLikeSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE minute_bar (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		provider_group TEXT NOT NULL,
		operation TEXT NOT NULL,
		market TEXT NOT NULL,
		security_type TEXT NOT NULL,
		symbol TEXT NOT NULL,
		isin TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL DEFAULT '',
		trading_date TEXT NOT NULL,
		trading_minute TEXT NOT NULL,
		currency TEXT NOT NULL DEFAULT '',
		opening_price TEXT NOT NULL DEFAULT '',
		highest_price TEXT NOT NULL DEFAULT '',
		lowest_price TEXT NOT NULL DEFAULT '',
		closing_price TEXT NOT NULL DEFAULT '',
		price_change_from_previous_close TEXT NOT NULL DEFAULT '',
		price_change_rate_from_previous_close TEXT NOT NULL DEFAULT '',
		traded_volume TEXT NOT NULL DEFAULT '',
		traded_amount TEXT NOT NULL DEFAULT '',
		market_capitalization TEXT NOT NULL DEFAULT '',
		extensions_json TEXT NOT NULL DEFAULT '{}',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	return createBarIndexes(ctx, db, "minute_bar", []string{
		"CREATE UNIQUE INDEX minute_bar_natural_key ON minute_bar (market, security_type, trading_date, trading_minute, symbol, provider, provider_group)",
		"CREATE INDEX idx_minute_bar_date ON minute_bar (market, security_type, trading_date, trading_minute)",
		"CREATE INDEX idx_minute_bar_symbol_date ON minute_bar (market, security_type, symbol, trading_date, trading_minute)",
	})
}

func createBarIndexes(ctx context.Context, db *sql.DB, table string, statements []string) error {
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return oops.In("sqlite_capacity_runtime").With("table", table).Wrapf(err, "create bar storage index")
		}
	}
	return nil
}

func writeMinuteBarStorageSizeResults(path string, config minuteBarStorageSizeProbeConfig, results []barStorageSizeResult) error {
	var builder strings.Builder
	builder.WriteString("# Minute Bar Storage Size Probe\n\n")
	builder.WriteString(fmt.Sprintf("- symbols: %d\n", config.Symbols))
	builder.WriteString(fmt.Sprintf("- minutes_per_day: %d\n", config.MinutesPerDay))
	builder.WriteString("- storage shape: current daily_bar-like wide text columns plus 3 SQLite indexes\n\n")
	builder.WriteString("| kind | trading_days | rows | file_bytes | payload_bytes | payload_kib_per_symbol_day | payload_bytes_per_row |\n")
	builder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, result := range results {
		builder.WriteString(fmt.Sprintf(
			"| %s | %d | %d | %d | %d | %.2f | %.1f |\n",
			result.Kind,
			result.TradingDays,
			result.Rows,
			result.FileBytes,
			result.PayloadBytes,
			result.PayloadBytesPerDay/1024,
			result.PayloadBytesPerRow,
		))
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func syntheticTradingDate(dayIndex int) string {
	return fmt.Sprintf("2026-05-%02d", dayIndex+1)
}

func syntheticTradingMinute(minuteIndex int) string {
	minutesFromOpen := 9*60 + minuteIndex
	return fmt.Sprintf("%02d:%02d", minutesFromOpen/60, minutesFromOpen%60)
}
