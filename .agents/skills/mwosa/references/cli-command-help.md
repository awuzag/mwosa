# mwosa CLI Command Help

Generated from `./bin/mwosa`. Use this when you need the complete installed or built CLI command surface instead of relying on source-code assumptions.

## Refresh Command

Run this from the repository root when the CLI changes:

```bash
skills/mwosa/references/generate-cli-command-help.sh

# Or use a freshly built binary:
MWOSA_HELP_COMMAND=./bin/mwosa skills/mwosa/references/generate-cli-command-help.sh

# If running the global skill copy outside the repo:
MWOSA_HELP_REPO_ROOT=/path/to/mwosa skills/mwosa/references/generate-cli-command-help.sh
```

## Captured Help

```text
mwosa v0.1.1-0.20260516051841-07b2a8392bda
schema dev
commit 07b2a8392bda21f2ffa286c345fdf3045523f12b
built 2026-05-16T05:18:41Z
go go1.25.6
Investment research CLI for provider-backed market data

Usage:
  mwosa [command]

Available Commands:
  backfill    Collect historical data ranges
  compare     Compare mwosa resources
  completion  Generate shell completion script
  config      Manage mwosa config file
  create      Create mwosa resources
  delete      Delete mwosa resources
  disable     Disable a resource
  doctor      Diagnose local configuration and resources
  enable      Enable a resource
  ensure      Fetch missing data and store it locally
  get         Read source-like data from local storage
  help        Help about any command
  history     List mwosa execution history
  inspect     Inspect mwosa resources and local state
  list        List mwosa resources
  login       Register credentials for a resource
  logout      Remove credentials for a resource
  migrate     Manage local data migrations
  prefer      Set resource preference
  rank        Rank mwosa resources
  run         Run mwosa workflows
  screen      Run screening workflows
  search      Search local and provider-backed resources
  sync        Refresh provider-backed data batches
  update      Update mwosa resources
  validate    Validate local configuration and resources
  version     Print mwosa build information

Flags:
      --config string            config file path
      --database string          local SQLite database path
  -h, --help                     help for mwosa
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa [command] --help" for more information about a command.


### mwosa backfill --help
Collect historical data ranges

Usage:
  mwosa backfill [command]

Available Commands:
  daily       Collect provider daily batches for a date range

Flags:
  -h, --help   help for backfill

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa backfill [command] --help" for more information about a command.


### mwosa backfill daily --help
Collect provider daily batches for a date range

Usage:
  mwosa backfill daily [flags]

Flags:
      --from string            start trading date, YYYYMMDD or YYYY-MM-DD
  -h, --help                   help for daily
      --security-type string   security type: stock, etf, etn, elw (default "etf")
      --to string              end trading date, YYYYMMDD or YYYY-MM-DD
      --workers int            number of workers for page-based daily providers (default 1)

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa compare --help
Compare mwosa resources

Usage:
  mwosa compare [command]

Available Commands:
  backtest-runs Compare two saved backtest runs
  evaluation    Compare saved backtest evaluation cases
  screen        Compare saved screen resources

Flags:
  -h, --help   help for compare

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa compare [command] --help" for more information about a command.


### mwosa compare backtest-runs --help
Compare two saved backtest runs

Usage:
  mwosa compare backtest-runs <left-id|name|result_hash> <right-id|name|result_hash> [flags]

Flags:
  -h, --help   help for backtest-runs

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa compare evaluation --help
Compare saved backtest evaluation cases

Usage:
  mwosa compare evaluation <name|id> [flags]

Flags:
  -h, --help          help for evaluation
      --view string   evaluation view: raw, summary, cases, regime, robustness, walk_forward (default "raw")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa compare screen --help
Compare saved screen resources

Usage:
  mwosa compare screen [command]

Available Commands:
  strategies  Compare saved screen strategies without recording screen history

Flags:
  -h, --help   help for screen

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa compare screen [command] --help" for more information about a command.


### mwosa compare screen strategies --help
Compare saved screen strategies without recording screen history

Usage:
  mwosa compare screen strategies <name> <name> [name...] [flags]

Flags:
      --as-of string   override YAML pipeline strategy as_of date in YYYY-MM-DD
  -h, --help           help for strategies
      --top int        top symbol count used for overlap (default 10)

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa completion --help
Generate shell completion script

Usage:
  mwosa completion <shell> [flags]

Flags:
  -h, --help   help for completion

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa config --help
Manage mwosa config file

Usage:
  mwosa config [command]

Available Commands:
  set         Set a config value

Flags:
  -h, --help   help for config

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa config [command] --help" for more information about a command.


### mwosa config set --help
Set a config value

Usage:
  mwosa config set <path> <value> [flags]

Flags:
  -h, --help   help for set

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa create --help
Create mwosa resources

Usage:
  mwosa create [command]

Available Commands:
  strategy    Create a saved screening strategy

Flags:
  -h, --help   help for create

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa create [command] --help" for more information about a command.


### mwosa create strategy --help
Create a saved screening strategy

Usage:
  mwosa create strategy <name> [flags]

Flags:
      --engine string    strategy engine: jq (default "jq")
  -h, --help             help for strategy
      --input string     input dataset name
      --jq string        inline jq query
      --jq-file string   path to a jq query file

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa delete --help
Delete mwosa resources

Usage:
  mwosa delete [command]

Available Commands:
  backtest    Delete backtest resources
  strategy    Soft delete a saved screening strategy

Flags:
  -h, --help   help for delete

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa delete [command] --help" for more information about a command.


### mwosa delete backtest --help
Delete backtest resources

Usage:
  mwosa delete backtest [command]

Available Commands:
  strategy    Soft delete a saved backtest strategy

Flags:
  -h, --help   help for backtest

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa delete backtest [command] --help" for more information about a command.


### mwosa delete backtest strategy --help
Soft delete a saved backtest strategy

Usage:
  mwosa delete backtest strategy <name> [flags]

Flags:
  -h, --help   help for strategy

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa delete strategy --help
Soft delete a saved screening strategy

Usage:
  mwosa delete strategy <name> [flags]

Flags:
  -h, --help   help for strategy

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa disable --help
Disable a resource

Usage:
  mwosa disable [command]

Available Commands:
  provider    Disable a provider

Flags:
  -h, --help   help for disable

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa disable [command] --help" for more information about a command.


### mwosa disable provider --help
Disable a provider

Usage:
  mwosa disable provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa doctor --help
Diagnose local configuration and resources

Usage:
  mwosa doctor [command]

Available Commands:
  provider    Diagnose provider configuration and client construction

Flags:
  -h, --help   help for doctor

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa doctor [command] --help" for more information about a command.


### mwosa doctor provider --help
Diagnose provider configuration and client construction

Usage:
  mwosa doctor provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa enable --help
Enable a resource

Usage:
  mwosa enable [command]

Available Commands:
  provider    Enable a provider

Flags:
  -h, --help   help for enable

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa enable [command] --help" for more information about a command.


### mwosa enable provider --help
Enable a provider

Usage:
  mwosa enable provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa ensure --help
Fetch missing data and store it locally

Usage:
  mwosa ensure [command]

Available Commands:
  daily       Fetch missing daily bars for a symbol and store them locally

Flags:
  -h, --help   help for ensure

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa ensure [command] --help" for more information about a command.


### mwosa ensure daily --help
Fetch missing daily bars for a symbol and store them locally

Usage:
  mwosa ensure daily <symbol> [flags]

Flags:
      --as-of string           single trading date, YYYYMMDD or YYYY-MM-DD
      --from string            start trading date, YYYYMMDD or YYYY-MM-DD
  -h, --help                   help for daily
      --security-type string   security type: stock, etf, etn, elw (default "etf")
      --to string              end trading date, YYYYMMDD or YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get --help
Read source-like data from local storage

Usage:
  mwosa get [command]

Available Commands:
  daily       Read stored daily bars for a symbol
  financials  Fetch provider-backed financial statements by company name or KRX code
  index       Fetch or read canonical index bars
  intraday    Fetch provider intraday bars for a symbol
  krx         Fetch a provider-native KRX OPEN API response
  orderbook   Fetch a provider orderbook snapshot for a symbol
  quote       Fetch a provider quote snapshot for a symbol

Flags:
  -h, --help   help for get

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa get [command] --help" for more information about a command.


### mwosa get daily --help
Read stored daily bars for a symbol

Usage:
  mwosa get daily <symbol> [flags]

Flags:
      --as-of string           single trading date, YYYYMMDD or YYYY-MM-DD
      --from string            start trading date, YYYYMMDD or YYYY-MM-DD
  -h, --help                   help for daily
      --security-type string   security type: stock, etf, etn, elw (default "etf")
      --to string              end trading date, YYYYMMDD or YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials --help
Fetch provider-backed financial statements by company name or KRX code

Usage:
  mwosa get financials <company> [flags]

Flags:
  -h, --help                   help for financials
      --limit int              maximum number of statement rows to fetch
      --period string          financial period: annual, quarter (default "annual")
      --security-type string   security type: stock, etf, etn, elw (default "stock")
      --statement string       statement type: summary, income_statement, balance_sheet, cash_flow; empty fetches all
      --year string            fiscal year, for example 2025

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get index --help
Fetch or read canonical index bars

Usage:
  mwosa get index <index-code> [flags]

Flags:
      --as-of string   single trading date, YYYYMMDD or YYYY-MM-DD
      --from string    start trading date, YYYYMMDD or YYYY-MM-DD
  -h, --help           help for index
      --to string      end trading date, YYYYMMDD or YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get intraday --help
Fetch provider intraday bars for a symbol

Usage:
  mwosa get intraday <symbol> [flags]

Flags:
      --at string              provider-neutral time anchor in HHMMSS or HH:MM:SS form
  -h, --help                   help for intraday
      --limit int              maximum number of intraday bars to return
      --security-type string   security type: stock, etf, etn (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get krx --help
Fetch a provider-native KRX OPEN API response

Usage:
  mwosa get krx <api-id> [flags]

Flags:
      --as-of string   base date to query, YYYYMMDD or YYYY-MM-DD
  -h, --help           help for krx

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get orderbook --help
Fetch a provider orderbook snapshot for a symbol

Usage:
  mwosa get orderbook <symbol> [flags]

Flags:
  -h, --help                   help for orderbook
      --security-type string   security type: stock, etf, etn (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get quote --help
Fetch a provider quote snapshot for a symbol

Usage:
  mwosa get quote <symbol> [flags]

Flags:
  -h, --help                   help for quote
      --security-type string   security type: stock, etf, etn (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa help --help
Help provides help for any command in the application.
Simply type mwosa help [path to command] for full details.

Usage:
  mwosa help [command] [flags]

Flags:
  -h, --help   help for help

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa history --help
List mwosa execution history

Usage:
  mwosa history [command]

Available Commands:
  screen      List saved screening runs

Flags:
  -h, --help   help for history

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa history [command] --help" for more information about a command.


### mwosa history screen --help
List saved screening runs

Usage:
  mwosa history screen [flags]

Flags:
  -h, --help        help for screen
      --limit int   maximum number of screen runs to list (default 50)

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect --help
Inspect mwosa resources and local state

Usage:
  mwosa inspect [command]

Available Commands:
  backtest          Inspect backtest resources
  backtest-run      Inspect a saved backtest run
  backtest-universe Inspect a YAML backtest universe pipeline
  config            Inspect resolved config and data paths
  coverage          Inspect local daily bar coverage for a symbol
  evaluation        Inspect a saved backtest evaluation
  instrument        Inspect one provider instrument
  market-regime     Inspect a YAML market regime model
  provider          Inspect provider configuration and readiness
  screen            Inspect a saved screening run
  screen-pipeline   Inspect a YAML screen universe pipeline
  storage           Summarize local daily bar storage coverage
  strategy          Inspect a saved screening strategy
  strategy-set      Inspect a YAML strategy set route by market regime

Flags:
  -h, --help   help for inspect

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa inspect [command] --help" for more information about a command.


### mwosa inspect backtest --help
Inspect backtest resources

Usage:
  mwosa inspect backtest [command]

Available Commands:
  run         Inspect a saved backtest run
  strategy    Inspect a saved backtest strategy
  universe    Inspect a YAML backtest universe pipeline

Flags:
  -h, --help   help for backtest

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa inspect backtest [command] --help" for more information about a command.


### mwosa inspect backtest run --help
Inspect a saved backtest run

Usage:
  mwosa inspect backtest run <id|name|result_hash> [flags]

Flags:
  -h, --help          help for run
      --view string   backtest result view: raw, summary, metrics, orders, fills, trades, positions, equity, universe, events (default "summary")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect backtest strategy --help
Inspect a saved backtest strategy

Usage:
  mwosa inspect backtest strategy <name> [flags]

Flags:
  -h, --help   help for strategy

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect backtest universe --help
Inspect a YAML backtest universe pipeline

Usage:
  mwosa inspect backtest universe <yaml> [flags]

Flags:
  -h, --help          help for universe
      --view string   universe explain view: summary, raw (default "summary")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect backtest-run --help
Inspect a saved backtest run

Usage:
  mwosa inspect backtest-run <id|name|result_hash> [flags]

Flags:
  -h, --help          help for backtest-run
      --view string   backtest result view: raw, summary, metrics, orders, fills, trades, positions, equity, universe, events (default "summary")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect backtest-universe --help
Inspect a YAML backtest universe pipeline

Usage:
  mwosa inspect backtest-universe <yaml> [flags]

Flags:
  -h, --help          help for backtest-universe
      --view string   universe explain view: summary, raw (default "summary")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect config --help
Inspect resolved config and data paths

Usage:
  mwosa inspect config [flags]

Flags:
  -h, --help   help for config

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect coverage --help
Inspect local daily bar coverage for a symbol

Usage:
  mwosa inspect coverage <symbol> [flags]

Flags:
  -h, --help                   help for coverage
      --security-type string   security type: stock, etf, etn, elw (default "etf")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect evaluation --help
Inspect a saved backtest evaluation

Usage:
  mwosa inspect evaluation <name|id> [flags]

Flags:
  -h, --help          help for evaluation
      --view string   evaluation view: raw, summary, cases, regime, robustness, walk_forward (default "raw")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect instrument --help
Inspect one provider instrument

Usage:
  mwosa inspect instrument <symbol> [flags]

Flags:
  -h, --help                   help for instrument
      --security-type string   security type: stock, etf, etn, elw (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect market-regime --help
Inspect a YAML market regime model

Usage:
  mwosa inspect market-regime <yaml> [flags]

Flags:
      --as-of string   regime calculation date in YYYY-MM-DD
  -h, --help           help for market-regime

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect provider --help
Inspect provider configuration and readiness

Usage:
  mwosa inspect provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect screen --help
Inspect a saved screening run

Usage:
  mwosa inspect screen <screen-id-or-alias> [flags]

Flags:
  -h, --help   help for screen

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect screen-pipeline --help
Inspect a YAML screen universe pipeline

Usage:
  mwosa inspect screen-pipeline <yaml> [flags]

Flags:
  -h, --help   help for screen-pipeline

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect storage --help
Summarize local daily bar storage coverage

Usage:
  mwosa inspect storage [flags]

Flags:
  -h, --help                   help for storage
      --security-type string   security type: stock, etf, etn, elw (default "etf")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect strategy --help
Inspect a saved screening strategy

Usage:
  mwosa inspect strategy <name> [flags]

Flags:
  -h, --help   help for strategy

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa inspect strategy-set --help
Inspect a YAML strategy set route by market regime

Usage:
  mwosa inspect strategy-set <yaml> [flags]

Flags:
      --as-of string   strategy set routing date in YYYY-MM-DD
  -h, --help           help for strategy-set

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list --help
List mwosa resources

Usage:
  mwosa list [command]

Available Commands:
  backtest    List backtest resources
  evaluations List saved backtest evaluations
  instruments Search stored instruments, falling back to provider search
  krx-apis    List KRX OPEN API services known to mwosa
  providers   List configured and available providers
  strategies  List saved screening strategies
  trades      List recent market trade prints for a symbol

Flags:
  -h, --help   help for list

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa list [command] --help" for more information about a command.


### mwosa list backtest --help
List backtest resources

Usage:
  mwosa list backtest [command]

Available Commands:
  runs        List saved backtest runs
  strategies  List saved backtest strategies

Flags:
  -h, --help   help for backtest

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa list backtest [command] --help" for more information about a command.


### mwosa list backtest runs --help
List saved backtest runs

Usage:
  mwosa list backtest runs [flags]

Flags:
  -h, --help   help for runs

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list backtest strategies --help
List saved backtest strategies

Usage:
  mwosa list backtest strategies [flags]

Flags:
  -h, --help   help for strategies

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list evaluations --help
List saved backtest evaluations

Usage:
  mwosa list evaluations [flags]

Flags:
  -h, --help   help for evaluations

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list instruments --help
Search stored instruments, falling back to provider search

Usage:
  mwosa list instruments <query> [flags]

Flags:
  -h, --help                   help for instruments
      --limit int              maximum number of instruments to return (default 25)
      --security-type string   security type: stock, etf, etn, elw (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list krx-apis --help
List KRX OPEN API services known to mwosa

Usage:
  mwosa list krx-apis [flags]

Flags:
  -h, --help   help for krx-apis

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list providers --help
List configured and available providers

Usage:
  mwosa list providers [flags]

Flags:
  -h, --help   help for providers

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list strategies --help
List saved screening strategies

Usage:
  mwosa list strategies [flags]

Flags:
  -h, --help   help for strategies

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list trades --help
List recent market trade prints for a symbol

Usage:
  mwosa list trades <symbol> [flags]

Flags:
      --at string              provider-neutral time anchor in HHMMSS or HH:MM:SS form
  -h, --help                   help for trades
      --limit int              maximum number of market trades to return
      --security-type string   security type: stock, etf, etn (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa login --help
Register credentials for a resource

Usage:
  mwosa login [command]

Available Commands:
  provider    Register provider credentials

Flags:
  -h, --help   help for login

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa login [command] --help" for more information about a command.


### mwosa login provider --help
Register provider credentials

Usage:
  mwosa login provider [command]

Available Commands:
  datago         Register datago provider credentials
  datago-corpfin Register datago-corpfin provider credentials
  kis            Register kis provider credentials
  krx            Register krx provider credentials

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa login provider [command] --help" for more information about a command.


### mwosa login provider datago --help
Register datago provider credentials

Usage:
  mwosa login provider datago [flags]

Flags:
      --base-url string                  legacy override datago securitiesProductPrice API base URL
      --etp-base-url string              override datago securitiesProductPrice API base URL
      --etp-service-key string           공공데이터포털 securitiesProductPrice service key
  -h, --help                             help for datago
      --service-key string               legacy 공공데이터포털 service key for securitiesProductPrice
      --stock-price-base-url string      override datago stockPrice API base URL
      --stock-price-service-key string   공공데이터포털 stockPrice service key

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa login provider datago-corpfin --help
Register datago-corpfin provider credentials

Usage:
  mwosa login provider datago-corpfin [flags]

Flags:
      --corpfin-base-url string         override datago corporateFinance API base URL
  -h, --help                            help for datago-corpfin
      --krx-listed-base-url string      override datago krxListedInfo API base URL
      --krx-listed-service-key string   공공데이터포털 krxListedInfo service key used for company-name and KRX-code resolution
      --service-key string              공공데이터포털 corporateFinance service key

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa login provider kis --help
Register kis provider credentials

Usage:
  mwosa login provider kis [flags]

Flags:
      --access-token string       optional pre-issued KIS OAuth access token
      --account string            optional KIS account identifier for future read-only account capabilities
      --app-key string            KIS Developers app key
      --app-secret string         KIS Developers app secret
      --base-url string           override KIS real API base URL
      --customer-type string      KIS custtype header value
  -h, --help                      help for kis
      --virtual string            use KIS virtual investment domain
      --virtual-base-url string   override KIS virtual API base URL

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa login provider krx --help
Register krx provider credentials

Usage:
  mwosa login provider krx [flags]

Flags:
      --auth-key string          KRX OPEN API AUTH_KEY
      --base-url string          override KRX OPEN API production base URL
  -h, --help                     help for krx
      --sample-base-url string   override KRX OPEN API sample base URL
      --use-sample string        use the KRX OPEN API sample endpoint

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa logout --help
Remove credentials for a resource

Usage:
  mwosa logout [command]

Available Commands:
  provider    Remove provider credentials

Flags:
  -h, --help   help for logout

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa logout [command] --help" for more information about a command.


### mwosa logout provider --help
Remove provider credentials

Usage:
  mwosa logout provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa migrate --help
Manage local data migrations

Usage:
  mwosa migrate [command]

Available Commands:
  apply       Apply pending local data migrations
  status      List local data migration status

Flags:
  -h, --help   help for migrate

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa migrate [command] --help" for more information about a command.


### mwosa migrate apply --help
Apply pending local data migrations

Usage:
  mwosa migrate apply [flags]

Flags:
  -h, --help   help for apply

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa migrate status --help
List local data migration status

Usage:
  mwosa migrate status [flags]

Flags:
  -h, --help   help for status

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa prefer --help
Set resource preference

Usage:
  mwosa prefer [command]

Available Commands:
  provider    Prefer a provider when multiple providers match

Flags:
  -h, --help   help for prefer

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa prefer [command] --help" for more information about a command.


### mwosa prefer provider --help
Prefer a provider when multiple providers match

Usage:
  mwosa prefer provider <name> [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa rank --help
Rank mwosa resources

Usage:
  mwosa rank [command]

Available Commands:
  evaluation  Rank saved backtest evaluation cases

Flags:
  -h, --help   help for rank

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa rank [command] --help" for more information about a command.


### mwosa rank evaluation --help
Rank saved backtest evaluation cases

Usage:
  mwosa rank evaluation <name|id> [flags]

Flags:
  -h, --help               help for evaluation
      --objective string   metric objective for ranking (default "calmar")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa run --help
Run mwosa workflows

Usage:
  mwosa run [command]

Available Commands:
  backtest    Run a YAML backtest against stored canonical daily bars
  evaluation  Run a YAML backtest evaluation against stored canonical daily bars

Flags:
  -h, --help   help for run

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa run [command] --help" for more information about a command.


### mwosa run backtest --help
Run a YAML backtest against stored canonical daily bars

Usage:
  mwosa run backtest <yaml> [flags]

Flags:
  -h, --help          help for backtest
      --view string   backtest result view: raw, summary, metrics, orders, fills, trades, positions, equity, universe, events (default "raw")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa run evaluation --help
Run a YAML backtest evaluation against stored canonical daily bars

Usage:
  mwosa run evaluation <yaml> [flags]

Flags:
  -h, --help              help for evaluation
      --parallelism int   bounded worker count for evaluation cases; overrides YAML execution.parallelism when positive

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa screen --help
Run screening workflows

Usage:
  mwosa screen [command]

Available Commands:
  etf         Run an inline jq screen against stored ETF daily records
  pipeline    Run a YAML screen universe pipeline
  strategy    Run a saved screening strategy

Flags:
  -h, --help   help for screen

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa screen [command] --help" for more information about a command.


### mwosa screen etf --help
Run an inline jq screen against stored ETF daily records

Usage:
  mwosa screen etf [flags]

Aliases:
  etf, etfs

Flags:
  -h, --help             help for etf
      --input string     input dataset name (default "etf_daily_metrics")
      --jq string        inline jq query
      --jq-file string   path to a jq query file

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa screen pipeline --help
Run a YAML screen universe pipeline

Usage:
  mwosa screen pipeline <yaml> [flags]

Flags:
  -h, --help   help for pipeline

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa screen strategy --help
Run a saved screening strategy

Usage:
  mwosa screen strategy <name> [flags]

Flags:
      --alias string       optional screen run alias
  -h, --help               help for strategy
      --spec-hash string   strategy spec hash to run
      --version string     strategy version number or latest

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa search --help
Search local and provider-backed resources

Usage:
  mwosa search [command]

Available Commands:
  instruments Search stored instruments

Flags:
  -h, --help   help for search

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa search [command] --help" for more information about a command.


### mwosa search instruments --help
Search stored instruments

Usage:
  mwosa search instruments <query> [flags]

Flags:
  -h, --help                   help for instruments
      --limit int              maximum number of instruments to return (default 25)
      --security-type string   security type: stock, etf, etn, elw (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync --help
Refresh provider-backed data batches

Usage:
  mwosa sync [command]

Available Commands:
  daily       Collect one provider daily batch for a date
  index       Fetch and store canonical index bars for a date
  instruments Fetch provider instrument master and store it locally
  krx         Fetch and store a provider-native KRX OPEN API snapshot

Flags:
  -h, --help   help for sync

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa sync [command] --help" for more information about a command.


### mwosa sync daily --help
Collect one provider daily batch for a date

Usage:
  mwosa sync daily [flags]

Flags:
      --as-of string           trading date to collect, YYYYMMDD or YYYY-MM-DD
  -h, --help                   help for daily
      --security-type string   security type: stock, etf, etn, elw (default "etf")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync index --help
Fetch and store canonical index bars for a date

Usage:
  mwosa sync index [index-code] [flags]

Flags:
      --as-of string   trading date to collect, YYYYMMDD or YYYY-MM-DD
  -h, --help           help for index

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync instruments --help
Fetch provider instrument master and store it locally

Usage:
  mwosa sync instruments [flags]

Flags:
      --as-of string           base date to collect, YYYYMMDD or YYYY-MM-DD
  -h, --help                   help for instruments
      --security-type string   security type: stock (default "stock")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync krx --help
Fetch and store a provider-native KRX OPEN API snapshot

Usage:
  mwosa sync krx <api-id> [flags]

Flags:
      --as-of string   base date to fetch and store, YYYYMMDD or YYYY-MM-DD
  -h, --help           help for krx

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa update --help
Update mwosa resources

Usage:
  mwosa update [command]

Available Commands:
  backtest    Update backtest resources
  screen      Update saved screen resources
  strategy    Create a new version of a saved screening strategy

Flags:
  -h, --help   help for update

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa update [command] --help" for more information about a command.


### mwosa update backtest --help
Update backtest resources

Usage:
  mwosa update backtest [command]

Available Commands:
  strategy    Create or update a saved backtest strategy from YAML

Flags:
  -h, --help   help for backtest

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa update backtest [command] --help" for more information about a command.


### mwosa update backtest strategy --help
Create or update a saved backtest strategy from YAML

Usage:
  mwosa update backtest strategy <name> [flags]

Flags:
  -h, --help               help for strategy
      --yaml-file string   YAML file containing a Strategy document

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa update screen --help
Update saved screen resources

Usage:
  mwosa update screen [command]

Available Commands:
  strategy    Create or update a saved screen strategy from YAML

Flags:
  -h, --help   help for screen

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa update screen [command] --help" for more information about a command.


### mwosa update screen strategy --help
Create or update a saved screen strategy from YAML

Usage:
  mwosa update screen strategy <name> [flags]

Flags:
      --file string   path to a ScreenStrategy or ScreenRun YAML file
  -h, --help          help for strategy

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa update strategy --help
Create a new version of a saved screening strategy

Usage:
  mwosa update strategy <name> [flags]

Flags:
  -h, --help             help for strategy
      --jq string        inline jq query
      --jq-file string   path to a jq query file

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa validate --help
Validate local configuration and resources

Usage:
  mwosa validate [command]

Available Commands:
  backtest    Validate a YAML backtest strategy and run spec
  evaluation  Validate a YAML backtest evaluation spec
  provider    Validate provider configuration

Flags:
  -h, --help   help for validate

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa validate [command] --help" for more information about a command.


### mwosa validate backtest --help
Validate a YAML backtest strategy and run spec

Usage:
  mwosa validate backtest <yaml> [flags]

Flags:
  -h, --help          help for backtest
      --view string   validation view: summary, raw (default "summary")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa validate evaluation --help
Validate a YAML backtest evaluation spec

Usage:
  mwosa validate evaluation <yaml> [flags]

Flags:
  -h, --help   help for evaluation

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa validate provider --help
Validate provider configuration

Usage:
  mwosa validate provider [name] [flags]

Flags:
  -h, --help   help for provider

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa version --help
Print mwosa build information

Usage:
  mwosa version [flags]

Flags:
  -h, --help   help for version

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id
```
