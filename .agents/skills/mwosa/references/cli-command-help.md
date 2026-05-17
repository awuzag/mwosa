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
mwosa v0.1.1-0.20260517143101-cb6938400971
schema dev
commit cb6938400971ccbae2007e27334bcc788419d78e
built 2026-05-17T14:31:01Z
go go1.26.3
Investment research CLI for provider-backed market data

Usage:
  mwosa [command]

Available Commands:
  backfill    Collect historical data ranges
  calc        Calculate derived mwosa resources
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


### mwosa calc --help
Calculate derived mwosa resources

Usage:
  mwosa calc [command]

Available Commands:
  financials  Calculate financial derived data

Flags:
  -h, --help   help for calc

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa calc [command] --help" for more information about a command.


### mwosa calc financials --help
Calculate financial derived data

Usage:
  mwosa calc financials [command]

Available Commands:
  metrics     Calculate stored canonical financial metrics
  valuation   Calculate stored canonical valuation snapshots

Flags:
  -h, --help   help for financials

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa calc financials [command] --help" for more information about a command.


### mwosa calc financials metrics --help
Calculate stored canonical financial metrics.

This reads financial_statement_v1 and financial_line_item_v1, then writes
financial_metric_v1. Missing source accounts are stored as uncomputable metrics
with explicit reasons.

Usage:
  mwosa calc financials metrics <company> [flags]

Flags:
  -h, --help            help for metrics
      --period string   financial period: annual, quarter (default "annual")
      --window string   metric window, for example 3y (default "3y")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa calc financials valuation --help
Calculate stored canonical valuation snapshots.

This combines the issuer instrument's daily_bar_v2 price/market cap with stored
financial statement line items, then writes valuation_snapshot_v1.

Usage:
  mwosa calc financials valuation <company> [flags]

Flags:
      --as-of string   valuation date, YYYY-MM-DD or latest (default "latest")
  -h, --help           help for valuation

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
  company-identifiers    List canonical company identifiers
  daily                  Read stored daily bars for a symbol
  filing                 Fetch provider-backed filing document metadata
  financials             Fetch provider-backed financial statements by company name or KRX code
  index                  Fetch or read canonical index bars
  intraday               Fetch provider intraday bars for a symbol
  krx                    Fetch a provider-native KRX OPEN API response
  orderbook              Fetch a provider orderbook snapshot for a symbol
  provider-raw           Read stored provider-native raw payload snapshots
  provider-raw-snapshots Read stored provider-native raw snapshots
  quote                  Fetch a provider quote snapshot for a symbol

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


### mwosa get company-identifiers --help
List canonical company identifiers.

The query is resolved from local company_v1 and company_identifier_v1 rows. OpenDART
corp_code and KRX stock_code are both identifiers, not canonical company keys.

Usage:
  mwosa get company-identifiers <company> [flags]

Flags:
  -h, --help   help for company-identifiers

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


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


### mwosa get filing --help
Fetch provider-backed filing document metadata.

With --provider opendart, this calls document.xml by rcept_no. Binary payload is
omitted by default so table, csv, json, and ndjson output remain safe for normal
stdout pipelines. Use --include-payload with json or ndjson to include the file
body as base64.

Usage:
  mwosa get filing <rcept-no> [flags]

Flags:
  -h, --help              help for filing
      --include-payload   include base64 file payload in json or ndjson output

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials --help
Fetch provider-backed financial statements by company name or KRX code.

With --provider opendart, <company> may be an OpenDART corp_code or a listed-company
stock_code. stock_code is resolved to corp_code before OpenDART financial API calls;
corp_code and stock_code remain separate fields in output extensions.

Usage:
  mwosa get financials <company> [flags]
  mwosa get financials [command]

Available Commands:
  dividends   Read stored canonical dividend facts
  facts       Read stored canonical company financial facts
  health      Read stored financial health metrics and audit facts
  metrics     Read stored canonical financial metrics
  statements  Read stored canonical financial statements
  valuation   Read stored canonical valuation snapshots

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

Use "mwosa get financials [command] --help" for more information about a command.


### mwosa get financials dividends --help
Read stored canonical dividend facts

Usage:
  mwosa get financials dividends <company> [flags]

Flags:
  -h, --help            help for dividends
      --window string   dividend window, for example 3y (default "3y")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials facts --help
Read stored canonical company financial facts

Usage:
  mwosa get financials facts <company> [flags]

Flags:
      --fact-type string   fact type filter, for example dividend
      --from string        fact date lower bound, YYYY-MM-DD
  -h, --help               help for facts
      --limit int          maximum fact rows to return
      --to string          fact date upper bound, YYYY-MM-DD
      --year string        fiscal year, for example 2025

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials health --help
Read stored financial health metrics and audit facts.

This command reads provider-neutral financial_metric_v1 rows plus audit opinion
facts from company_fact_v1. Run calc financials metrics and sync financials facts
first to populate the underlying canonical data.

Usage:
  mwosa get financials health <company> [flags]

Flags:
  -h, --help            help for health
      --period string   financial period: annual, quarter (default "annual")
      --window string   metric window, for example 3y (default "3y")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials metrics --help
Read stored canonical financial metrics

Usage:
  mwosa get financials metrics <company> [flags]

Flags:
  -h, --help            help for metrics
      --period string   financial period: annual, quarter (default "annual")
      --window string   metric window, for example 3y (default "3y")

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get financials statements --help
Read stored canonical financial statements.

This reads financial_statement_v1 and financial_line_item_v1 from local SQLite.
The legacy shortcut get financials <company> still fetches provider-backed data
through the router.

Usage:
  mwosa get financials statements <company> [flags]

Flags:
  -h, --help                   help for statements
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


### mwosa get financials valuation --help
Read stored canonical valuation snapshots

Usage:
  mwosa get financials valuation <company> [flags]

Flags:
      --as-of string   valuation date, YYYY-MM-DD or latest (default "latest")
  -h, --help           help for valuation

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


### mwosa get provider-raw --help
Read stored provider-native raw payload snapshots.

This is a friendlier alias over provider_raw_snapshots for canonicalization
escape hatches. It does not call the provider live; it only reads snapshots that
previous sync commands have already stored locally.

Usage:
  mwosa get provider-raw [provider] [operation] [flags]

Flags:
      --from string        base date lower bound, YYYYMMDD or YYYY-MM-DD
      --group string       provider group filter
  -h, --help               help for provider-raw
      --include-payload    include decoded provider-native payload in JSON/NDJSON output
      --limit int          maximum snapshots to return (default 50)
      --operation string   provider operation/api id filter
      --to string          base date upper bound, YYYYMMDD or YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa get provider-raw-snapshots --help
Read stored provider-native raw snapshots.

This reads provider_raw_snapshots from local SQLite. It is an escape hatch for
provider APIs that are not yet canonicalized, while keeping canonical analysis
tables separate from provider-native payloads.

Usage:
  mwosa get provider-raw-snapshots [flags]

Flags:
      --from string        base date lower bound, YYYYMMDD or YYYY-MM-DD
      --group string       provider group filter
  -h, --help               help for provider-raw-snapshots
      --include-payload    include decoded provider-native payload in JSON/NDJSON output
      --limit int          maximum snapshots to return (default 50)
      --operation string   provider operation/api id filter
      --to string          base date upper bound, YYYYMMDD or YYYY-MM-DD

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
  company           Inspect one canonical company
  config            Inspect resolved config and data paths
  coverage          Inspect local daily bar coverage for a symbol
  evaluation        Inspect a saved backtest evaluation
  instrument        Inspect one provider instrument
  market-regime     Inspect a YAML market regime model
  provider          Inspect provider configuration and readiness
  screen            Inspect a saved screening run
  screen-pipeline   Inspect a YAML screen universe pipeline
  stock             Inspect a stored stock profile with financial analysis sections
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


### mwosa inspect company --help
Inspect one canonical company.

The query is resolved from local company_v1 and company_identifier_v1 rows. Use
sync companies --provider opendart to populate the initial Korean listed-company
identity graph from corpCode.xml.

Usage:
  mwosa inspect company <company> [flags]

Flags:
  -h, --help   help for company

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


### mwosa inspect stock --help
Inspect a stored stock profile with financial analysis sections.

This command reads canonical local storage only. Use sync companies, sync
financials statements, calc financials metrics, calc financials valuation, and
sync financials facts or sync events to populate the underlying sections.

Usage:
  mwosa inspect stock <symbol-or-company> [flags]

Flags:
      --as-of string     valuation date, YYYY-MM-DD or latest (default "latest")
  -h, --help             help for stock
      --period string    financial period: annual, quarter (default "annual")
      --section string   comma-separated sections: profile,investment,financials,scores,dividends,events,all; use facts explicitly for large fact rows (default "profile,investment,financials,scores,dividends,events")
      --window string    financial metric/dividend window, for example 3y (default "3y")

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
  backtest      List backtest resources
  constituents  List composition constituents for a symbol
  evaluations   List saved backtest evaluations
  events        List stored canonical company events
  filings       List provider-backed filings
  instruments   Search stored instruments, falling back to provider search
  krx-apis      List KRX OPEN API services known to mwosa
  provider-apis List provider-native APIs known to mwosa diagnostics
  providers     List configured and available providers
  strategies    List saved screening strategies
  trades        List recent market trade prints for a symbol

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


### mwosa list constituents --help
List composition constituents for a symbol

Usage:
  mwosa list constituents <symbol> [flags]

Flags:
  -h, --help        help for constituents
      --limit int   maximum number of constituents to return

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


### mwosa list events --help
List stored canonical company events.

This reads company_event_v1 from local SQLite. Use sync events --provider
opendart to fetch canonicalized OpenDART material events first.

Usage:
  mwosa list events <company> [flags]

Flags:
      --event-type string   event type filter
      --from string         event date lower bound, YYYY-MM-DD
  -h, --help                help for events
      --limit int           maximum event rows to return (default 50)
      --to string           event date upper bound, YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa list filings --help
List provider-backed filings.

With --provider opendart, the positional argument may be an OpenDART corp_code or
a listed-company stock_code. stock_code is resolved to corp_code before querying
OpenDART. Use --corp-code to bypass stock_code resolution.

Usage:
  mwosa list filings [corp-code-or-stock-code] [flags]

Flags:
      --corp-code string    OpenDART corp_code; bypasses stock_code resolution
      --from string         filing start date, YYYYMMDD or YYYY-MM-DD
  -h, --help                help for filings
      --last-report         request only final reports
      --page-count string   OpenDART page size, max 100 (default "10")
      --page-no string      OpenDART page number (default "1")
      --to string           filing end date, YYYYMMDD or YYYY-MM-DD

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


### mwosa list provider-apis --help
List provider-native APIs known to mwosa diagnostics

Usage:
  mwosa list provider-apis <provider> [flags]

Flags:
  -h, --help   help for provider-apis

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
  opendart       Register opendart provider credentials

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


### mwosa login provider opendart --help
Register opendart provider credentials

Usage:
  mwosa login provider opendart [flags]

Flags:
      --api-key string    OpenDART API key; OPENDART_API_KEY is preferred
      --base-url string   override OpenDART base URL
  -h, --help              help for opendart

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
  stock       Run an inline jq screen against stored stock daily records enriched with financial metrics
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


### mwosa screen stock --help
Run an inline jq screen against stored stock daily records enriched with financial metrics

Usage:
  mwosa screen stock [flags]

Aliases:
  stock, stocks

Flags:
  -h, --help             help for stock
      --input string     input dataset name (default "stock_daily_metrics")
      --jq string        inline jq query
      --jq-file string   path to a jq query file

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
  companies   Search a provider-backed company registry
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


### mwosa search companies --help
Search a provider-backed company registry.

With --provider opendart, the query matches local OpenDART corp_code, stock_code,
corp_name, or corp_eng_name. OpenDART corp_code is the disclosure identifier and
stock_code is the listed-company KRX mapping.

Usage:
  mwosa search companies <query> [flags]

Flags:
  -h, --help          help for companies
      --limit int     maximum company rows to return (default 20)
      --listed-only   return only rows with OpenDART stock_code

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


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
  companies   Sync provider-backed company registry
  daily       Collect one provider daily batch for a date
  events      Fetch provider-backed material events and store canonical rows
  financials  Sync provider-backed financial resources
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


### mwosa sync companies --help
Sync a provider-backed company registry.

With --provider opendart, this fetches OpenDART corpCode.xml and stores corp_code,
corp_name, corp_eng_name, stock_code, and modify_date. OpenDART corp_code is not a
KRX stock_code; stock_code is stored only as a listed-company mapping.

Usage:
  mwosa sync companies [flags]

Flags:
  -h, --help          help for companies
      --listed-only   store only companies with OpenDART stock_code

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


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


### mwosa sync events --help
Fetch provider-backed material events and store canonical rows.

With --provider opendart, this currently canonicalizes default, bank-management,
lawsuit, capital increase/reduction, business/asset transfer, CB/BW/EB,
merger/division, stock exchange/transfer, and treasury-stock decision APIs into
company_event_v1. Additional material event APIs remain separate until each
operation has an explicit mapping.

Usage:
  mwosa sync events <company> [flags]

Flags:
      --from string   event filing start date, YYYYMMDD or YYYY-MM-DD
  -h, --help          help for events
      --to string     event filing end date, YYYYMMDD or YYYY-MM-DD

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync financials --help
Sync provider-backed financial resources

Usage:
  mwosa sync financials [command]

Available Commands:
  dividends   Fetch OpenDART dividend facts and store canonical company facts
  facts       Fetch OpenDART periodic report facts and store canonical company facts
  statements  Fetch provider-backed financial statements and store canonical rows

Flags:
  -h, --help   help for financials

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id

Use "mwosa sync financials [command] --help" for more information about a command.


### mwosa sync financials dividends --help
Fetch OpenDART dividend matters and store canonical company facts.

This writes company_fact_v1 rows with fact_type=dividend. The company must already
exist in the canonical company identity graph so the OpenDART corp_code can be
read as an identifier rather than treated as the canonical key.

Usage:
  mwosa sync financials dividends <company> [flags]

Flags:
      --from string   first fiscal year to sync, for example 2023
  -h, --help          help for dividends
      --to string     last fiscal year to sync, for example 2025
      --year string   fiscal year, for example 2025

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync financials facts --help
Fetch OpenDART periodic report facts and store canonical company facts.

With --provider opendart, this currently canonicalizes dividends, treasury stock,
major shareholders, major shareholder changes, employee status, and audit opinion
rows into company_fact_v1. The company must already exist in the canonical company
identity graph so OpenDART corp_code is used as a provider identifier.

Usage:
  mwosa sync financials facts <company> [flags]

Flags:
      --from string   first fiscal year to sync, for example 2023
  -h, --help          help for facts
      --to string     last fiscal year to sync, for example 2025
      --year string   fiscal year, for example 2025

Global Flags:
      --config string            config file path
      --database string          local SQLite database path
      --market string            market id (default "krx")
  -o, --output output            output format: table, json, ndjson, csv (default table)
      --prefer-provider string   prefer a provider when multiple candidates match
      --provider string          force a provider by id


### mwosa sync financials statements --help
Fetch provider-backed financial statements and store canonical rows.

The company must already exist in the canonical company identity graph. For
OpenDART, run sync companies --provider opendart first so corp_code and stock_code
are available as identifiers.

Usage:
  mwosa sync financials statements <company> [flags]

Flags:
      --from string            first fiscal year to sync, for example 2023
  -h, --help                   help for statements
      --limit int              maximum number of statement rows to fetch
      --period string          financial period: annual, quarter (default "annual")
      --security-type string   security type: stock, etf, etn, elw (default "stock")
      --statement string       statement type: summary, income_statement, balance_sheet, cash_flow; empty fetches all
      --to string              last fiscal year to sync, for example 2025
      --year string            fiscal year, for example 2025

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
