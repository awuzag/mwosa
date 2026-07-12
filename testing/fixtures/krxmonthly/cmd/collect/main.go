package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/awuzag/mwosa/testing/fixtures/krxmonthly"
	"github.com/samber/oops"
)

const dateLayout = "2006-01-02"

type options struct {
	Binary       string
	Config       string
	From         string
	To           string
	Output       string
	FixtureID    string
	Overwrite    bool
	RequestDelay time.Duration
}

type output struct {
	Archive       string   `json:"archive"`
	FixtureID     string   `json:"fixture_id"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	TradingDates  []string `json:"trading_dates"`
	SnapshotCount int      `json:"snapshot_count"`
	TotalRows     int      `json:"total_rows"`
	DatasetSHA256 string   `json:"dataset_sha256"`
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("krx-monthly-fixture-collector", flag.ContinueOnError)
	opts := options{}
	flags.StringVar(&opts.Binary, "mwosa-bin", ".mwosa/bin/mwosa", "mwosa binary built from the current checkout")
	flags.StringVar(&opts.Config, "config", defaultConfigPath(), "mwosa config containing KRX credentials")
	flags.StringVar(&opts.From, "from", "2026-06-01", "first calendar date, YYYY-MM-DD")
	flags.StringVar(&opts.To, "to", "2026-06-30", "last calendar date, YYYY-MM-DD")
	flags.StringVar(&opts.Output, "output", "testdata/aggregate/krx/krx-stock-daily-2026-06.zip", "output ZIP archive")
	flags.StringVar(&opts.FixtureID, "fixture-id", "krx-stock-daily-2026-06", "stable fixture identity")
	flags.BoolVar(&opts.Overwrite, "overwrite", false, "replace an existing archive")
	flags.DurationVar(&opts.RequestDelay, "request-delay", 100*time.Millisecond, "delay between provider calls")
	if err := flags.Parse(args); err != nil {
		return oops.In("krx_monthly_fixture_collector").Wrap(err)
	}

	from, err := time.Parse(dateLayout, opts.From)
	if err != nil {
		return oops.In("krx_monthly_fixture_collector").With("from", opts.From).Wrapf(err, "parse from date")
	}
	to, err := time.Parse(dateLayout, opts.To)
	if err != nil {
		return oops.In("krx_monthly_fixture_collector").With("to", opts.To).Wrapf(err, "parse to date")
	}
	if from.After(to) {
		return oops.In("krx_monthly_fixture_collector").New("from date must not be after to date")
	}
	if _, err := os.Stat(opts.Binary); err != nil {
		return oops.In("krx_monthly_fixture_collector").With("binary", opts.Binary).Wrapf(err, "inspect mwosa binary")
	}

	temporary, err := os.MkdirTemp("", "mwosa-krx-fixture-collector-*")
	if err != nil {
		return oops.In("krx_monthly_fixture_collector").Wrapf(err, "create temporary collector directory")
	}
	defer os.RemoveAll(temporary)
	config := strings.TrimSpace(opts.Config)
	if config == "" {
		config = filepath.Join(temporary, "config.json")
	}
	database := filepath.Join(temporary, "mwosa.db")

	operations := []string{"stk_bydd_trd", "ksq_bydd_trd"}
	snapshots := make([]krxmonthly.RawSnapshot, 0, 46)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		for _, operation := range operations {
			snapshot, err := fetchSnapshot(ctx, opts.Binary, config, database, operation, day)
			if err != nil {
				return err
			}
			if snapshot.RowCount == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "skip empty KRX snapshot: %s %s\n", operation, day.Format(dateLayout))
				continue
			}
			snapshots = append(snapshots, snapshot)
			_, _ = fmt.Fprintf(os.Stderr, "collected KRX snapshot: %s %s rows=%d\n", operation, day.Format(dateLayout), snapshot.RowCount)
			if opts.RequestDelay > 0 {
				time.Sleep(opts.RequestDelay)
			}
		}
	}

	manifest, err := krxmonthly.WriteArchive(opts.Output, krxmonthly.BuildOptions{
		FixtureID:   opts.FixtureID,
		From:        from.Format(dateLayout),
		To:          to.Format(dateLayout),
		CollectedAt: time.Now().UTC(),
		Overwrite:   opts.Overwrite,
	}, snapshots)
	if err != nil {
		return err
	}
	result := output{
		Archive:       opts.Output,
		FixtureID:     manifest.FixtureID,
		From:          manifest.From,
		To:            manifest.To,
		TradingDates:  manifest.TradingDates,
		SnapshotCount: manifest.SnapshotCount,
		TotalRows:     manifest.TotalRows,
		DatasetSHA256: manifest.DatasetSHA256,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return oops.In("krx_monthly_fixture_collector").Wrapf(err, "encode collector result")
	}
	_, _ = os.Stdout.Write(append(encoded, '\n'))
	return nil
}

func fetchSnapshot(ctx context.Context, binary string, config string, database string, operation string, day time.Time) (krxmonthly.RawSnapshot, error) {
	errB := oops.In("krx_monthly_fixture_collector").With("operation", operation, "base_date", day.Format(dateLayout))
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	args := []string{
		"--config", config,
		"--database-backend", "sqlite",
		"--database", database,
		"get", "krx", operation,
		"--as-of", day.Format(dateLayout),
		"-o", "json",
	}
	command := exec.CommandContext(callCtx, binary, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(redact(stderr.String(), os.Getenv("MWOSA_KRX_AUTH_KEY")))
		if message == "" {
			return krxmonthly.RawSnapshot{}, errB.Wrapf(err, "execute mwosa KRX request")
		}
		return krxmonthly.RawSnapshot{}, errB.With("stderr", message).Wrapf(err, "execute mwosa KRX request")
	}
	var snapshot krxmonthly.RawSnapshot
	if err := json.Unmarshal(stdout, &snapshot); err != nil {
		return krxmonthly.RawSnapshot{}, errB.Wrapf(err, "decode mwosa KRX response")
	}
	if snapshot.APIID != operation || snapshot.BaseDate != day.Format(dateLayout) {
		return krxmonthly.RawSnapshot{}, errB.New("mwosa KRX response identity mismatch")
	}
	return snapshot, nil
}

func defaultConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("MWOSA_CONFIG")); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "mwosa", "config.json")
}

func redact(value string, secret string) string {
	if secret == "" {
		return value
	}
	return strings.ReplaceAll(value, secret, "<redacted>")
}
