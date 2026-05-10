module github.com/ev3rlit/mwosa

go 1.25.6

require (
	github.com/ev3rlit/mwosa/clients/datago-corpfin v0.0.0
	github.com/ev3rlit/mwosa/clients/datago-etp v0.0.0-20260503103611-57138e7267ca
	github.com/ev3rlit/mwosa/clients/datago-krxlisted v0.0.0
	github.com/ev3rlit/mwosa/clients/datago-stock-price v0.0.0
	github.com/ev3rlit/mwosa/clients/kis v0.0.0
	github.com/ev3rlit/mwosa/clients/krx v0.0.0
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/google/uuid v1.6.0
	github.com/itchyny/gojq v0.12.19
	github.com/jszwec/csvutil v1.10.0
	github.com/olekukonko/tablewriter v1.1.4
	github.com/samber/oops v1.21.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/uptrace/bun v1.2.18
	github.com/uptrace/bun/dialect/sqlitedialect v1.2.18
	golang.org/x/text v0.34.0
	modernc.org/sqlite v1.50.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-resty/resty/v2 v2.17.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/timefmt-go v0.1.8 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/samber/lo v1.52.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.72.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/ev3rlit/mwosa/clients/datago-corpfin => ./clients/datago-corpfin

replace github.com/ev3rlit/mwosa/clients/datago-krxlisted => ./clients/datago-krxlisted

replace github.com/ev3rlit/mwosa/clients/datago-stock-price => ./clients/datago-stock-price

replace github.com/ev3rlit/mwosa/clients/kis => ./clients/kis

replace github.com/ev3rlit/mwosa/clients/krx => ./clients/krx
