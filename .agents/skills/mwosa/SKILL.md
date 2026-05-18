---
name: mwosa
description: Use when working in the mwosa repo or explaining mwosa CLI workflows, especially provider setup, OpenDART company and financial research, Datago ETF daily collection, canonical SQLite data, jq-based ETF screening, raw Datago JSON experiments, and saved screening strategies.
---

# mwosa

## First checks

- Work from the repository root.
- Before code edits, read `RULE.md` and check `git status --short --branch`.
- Prefer the installed `mwosa` CLI for user-facing commands. Use `go run` only when testing local source changes or running experiment-only tools that are not installed as CLI commands.
- When you need the complete installed command surface, read `references/cli-command-help.md`; it is generated from `mwosa --help` plus subcommand help.
- Do not edit `references/cli-command-help.md` by hand. Regenerate it only when the CLI binary changes.
- Keep stdout machine-readable for `json`, `ndjson`, `csv`, and `jq` pipelines. Put progress, diagnostics, and explanations on stderr or in chat.

## Provider command surface

Use provider-generic commands first, then specialize with `--provider` or a provider id. Known providers include:

| Provider | Primary use |
| --- | --- |
| `datago` | KRX daily market data such as ETF/ETN/ELW bars |
| `krx` | KRX Open API snapshots and instrument master data |
| `kis` | Korea Investment & Securities market-data endpoints |
| `opendart` | Listed companies, filings, financial statements, facts, dividends, and material events |

Common provider commands:

```bash
mwosa list providers -o table
mwosa login provider <provider>
mwosa doctor provider <provider> -o json
mwosa inspect provider <provider> -o table
mwosa list provider-apis <provider> -o table
```

Use provider raw commands only for debugging, API coverage checks, or when the user explicitly asks for source payloads:

```bash
mwosa get provider-raw <provider> <operation> -o json
mwosa get provider-raw-snapshots --provider <provider> -o json
```

Never print API keys or credential values. Prefer `-o json` for automation and `-o table` for human inspection.

## OpenDART stock research workflow

Use OpenDART when the user asks about Korean listed-company fundamentals, filings, financial statements, disclosure facts, dividends, or material events. Prefer this order:

```bash
mwosa login provider opendart
mwosa doctor provider opendart -o json
mwosa sync companies --provider opendart --listed-only -o json
mwosa search companies 005930 --provider opendart -o table
mwosa get company-identifiers 005930 -o table
```

Collect annual financial data before calculating metrics:

```bash
mwosa sync financials statements 005930 \
  --provider opendart \
  --from 2023 \
  --to 2025 \
  --period annual \
  -o json

mwosa sync financials facts 005930 \
  --provider opendart \
  --from 2023 \
  --to 2025 \
  -o json

mwosa calc financials metrics 005930 \
  --window 3y \
  --period annual \
  -o json
```

Use the human-facing stock report after collection:

```bash
mwosa inspect stock 005930 -o table
mwosa inspect stock 005930 --section all -o table
```

`inspect stock` is the preferred human-readable summary. It should present overview, investment metrics, financial summary, trends, statement tables, dividends, risk, and missing values as purpose-specific tables. Missing or unavailable values should appear naturally in those tables, not as a long debug dump.

Use detailed commands when the user asks for raw rows or a specific data family:

```bash
mwosa get financials statements 005930 -o table
mwosa get financials facts 005930 -o json
mwosa get financials dividends 005930 -o table
mwosa get financials metrics 005930 -o table
mwosa get financials valuation 005930 -o table
mwosa get financials health 005930 -o table
mwosa list filings 005930 --provider opendart -o table
mwosa get filing <rcept-no> --provider opendart -o json
mwosa sync events 005930 --provider opendart -o json
mwosa list events 005930 -o table
```

Do not invent valuation or market-cap data when the local database has no compatible price point. In that case, explain that valuation metrics require price or market-cap data from another provider such as KIS or another market-data source.

## ETF daily collection

Use canonical SQLite collection when the user asks to collect ETF data for analysis:

```bash
mwosa backfill daily \
  --provider datago \
  --security-type etf \
  --from YYYY-MM-DD \
  --to YYYY-MM-DD \
  --workers 4 \
  -o json
```

For one trading date, set `--from` and `--to` to the same date. If `datago` is already the configured/default provider, `--provider datago` can be omitted, but keep it in handoff commands when clarity matters.

Verify a known ETF after collection:

```bash
mwosa get daily 069500 \
  --security-type etf \
  --from YYYY-MM-DD \
  --to YYYY-MM-DD \
  -o json
```

If provider auth is suspect, inspect or validate the provider without printing secrets:

```bash
mwosa doctor provider datago -o json
```

## Current jq screening surface

Check `references/cli-command-help.md` before recommending jq commands. The installed `mwosa v0.1.0` exposes saved strategy commands, while newer source checkouts may also expose one-off `mwosa screen etfs --jq/--jq-file`.

Installed `v0.1.0` strategy flow:

```bash
mwosa create strategy etf-weekly-leaders \
  --engine jq \
  --input etf_daily_metrics \
  --jq-file strategies/etf-weekly-leaders.jq \
  -o json

mwosa screen strategy etf-weekly-leaders \
  --alias YYYY-MM-DD-weekly-leaders \
  -o json

mwosa history screen -o table
mwosa inspect screen YYYY-MM-DD-weekly-leaders -o json
```

Do not promise runtime `--argjson` support for saved strategy execution unless the codebase has been checked again; the current CLI flags only expose `--alias` on `screen strategy`.

## Dataset schema and keys

`etf_daily_metrics` currently reads canonical ETF daily bars from SQLite. Despite the name, treat it as daily-bar records unless the codebase has added derived metrics.

Canonical strategy input rows use these JSON keys:

| Key | Meaning | Type |
| --- | --- | --- |
| `provider` | provider id, usually `datago` | string |
| `provider_group` | provider group, usually `securitiesProductPrice` | string |
| `operation` | source operation, e.g. `getETFPriceInfo` | string |
| `market` | market id, usually `krx` | string |
| `security_type` | `etf`, `etn`, or `elw` | string |
| `trading_date` | trading date, `YYYY-MM-DD` | string |
| `symbol` | KRX short code such as `069500` | string |
| `isin` | ISIN code | string |
| `name` | item name | string |
| `currency` | currency, usually `KRW` | string |
| `opening_price` | open price | numeric string |
| `highest_price` | high price | numeric string |
| `lowest_price` | low price | numeric string |
| `closing_price` | close price | numeric string |
| `price_change_from_previous_close` | absolute change from previous close | numeric string |
| `price_change_rate_from_previous_close` | percent change from previous close | numeric string |
| `traded_volume` | traded volume | numeric string |
| `traded_amount` | traded value/amount | numeric string |
| `market_capitalization` | market capitalization | numeric string |
| `extensions` | provider-specific extra fields | object of strings |

Convert numeric strings inside jq with `tonumber? // 0`.

Common Datago ETF extras are stored under `extensions` after canonical storage:

| Extension key | Meaning |
| --- | --- |
| `nPptTotAmt` | ETF net asset total amount |
| `stLstgCnt` | ETF listed share count |
| `nav` | ETF NAV |
| `bssIdxIdxNm` | underlying index name |
| `bssIdxClpr` | underlying index close |

For raw Datago JSON files, use provider-native keys instead of canonical keys:

| Raw key | Canonical key |
| --- | --- |
| `basDt` | `trading_date` |
| `srtnCd` | `symbol` |
| `isinCd` | `isin` |
| `itmsNm` | `name` |
| `mkp` | `opening_price` |
| `hipr` | `highest_price` |
| `lopr` | `lowest_price` |
| `clpr` | `closing_price` |
| `vs` | `price_change_from_previous_close` |
| `fltRt` | `price_change_rate_from_previous_close` |
| `trqu` | `traded_volume` |
| `trPrc` | `traded_amount` |
| `mrktTotAmt` | `market_capitalization` |

When writing jq for canonical SQLite screens, use canonical snake_case keys. Use raw camel-case Datago keys only inside `testing/experiments/datago_daily_json_collector` scripts or raw snapshot files.

## Practical jq examples

Latest-date liquidity leaders query:

```jq
def n: tonumber? // 0;
group_by(.symbol)
| map(max_by(.trading_date))
| sort_by((.traded_amount | n))
| reverse
| .[:50]
```

Save and run a query file on installed `v0.1.0`:

```bash
mwosa create strategy latest-liquidity-leaders \
  --engine jq \
  --input etf_daily_metrics \
  --jq-file strategies/latest-liquidity-leaders.jq \
  -o json

mwosa screen strategy latest-liquidity-leaders \
  --alias YYYY-MM-DD-liquidity-leaders \
  -o json
```

Period return leaders query from stored daily bars:

```jq
def n: tonumber? // 0;
map(select(.trading_date >= "YYYY-MM-DD" and .trading_date <= "YYYY-MM-DD"))
| group_by(.symbol)
| map(
    sort_by(.trading_date) as $rows
    | ($rows[0].closing_price | n) as $start
    | ($rows[-1].closing_price | n) as $end
    | select(length >= 2 and $start > 0 and $end > 0)
    | {
        symbol: $rows[-1].symbol,
        name: $rows[-1].name,
        from: $rows[0].trading_date,
        to: $rows[-1].trading_date,
        return_pct: (($end / $start - 1) * 100),
        traded_amount: ($rows[-1].traded_amount | n)
      }
  )
| sort_by(.return_pct)
| reverse
| .[:50]
```

For CSV output from one-off screens, prefer `-o csv` only when rows are flat. If payloads are nested, use `-o json | jq -r ...` to choose exact columns.

## Raw Datago JSON experiment scripts

Use the raw collector only when the user specifically wants date-partitioned source JSON or the existing experiment screeners. It writes one JSON file per date under `tmp/testing/datago-daily-json-collector/raw`.

```bash
DATAGO_SERVICE_KEY="$DATAGO_SERVICE_KEY" \
go run ./testing/experiments/datago_daily_json_collector \
  --products etf \
  --start-date YYYY-MM-DD \
  --end-date YYYY-MM-DD \
  --workers 2 \
  --overwrite=true
```

Run the momentum screener:

```bash
testing/experiments/datago_daily_json_collector/scripts/next_day_momentum/screen_next_day_momentum.sh \
  --raw-dir tmp/testing/datago-daily-json-collector/raw \
  --preset balanced \
  --format csv
```

Run the low-volatility uptrend screener:

```bash
testing/experiments/datago_daily_json_collector/scripts/low_vol_uptrend/screen_low_vol_uptrend.sh \
  --raw-dir tmp/testing/datago-daily-json-collector/raw \
  --format csv
```

Always pass `--raw-dir tmp/testing/datago-daily-json-collector/raw` explicitly for these scripts. When results look stale or empty, inspect the target date file and rerun with overwrite; old empty date files have caused misleading screens before.

## How to answer users

- If the user asks for a command, give the installed `mwosa` command first.
- If the user asks about providers, start with `list providers`, `doctor provider`, and `inspect provider`; use `list provider-apis` for provider-native coverage.
- If the user asks about OpenDART, lead with the company and financial workflow, then mention raw/detail commands only if needed.
- If the user asks how jq screening works, explain the two layers: canonical SQLite strategy screens through installed `create/screen strategy`, raw JSON experiments through `testing/experiments/.../scripts`.
- Keep Korean explanations concise and command-first.
- When producing candidate files for review, create both machine-friendly JSON and small human-openable CSV when the output could be large.
