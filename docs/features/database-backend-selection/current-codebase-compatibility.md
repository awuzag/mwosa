# Current Codebase Compatibility

조사일: 2026-06-07

## 요약

현재 코드베이스는 repository/service 경계가 비교적 잘 나뉘어 있어서 PostgreSQL
backend 를 추가할 구조적 여지는 있다. 특히 storage repository 들은 대부분 Bun query
builder 와 service layer repository interface 를 사용하므로, service/domain layer 를
크게 흔들 필요는 없어 보인다.

반면 실제 database runtime, config, CLI 입력면은 SQLite file path 에 강하게 묶여
있다. PostgreSQL URL backend 를 붙이려면 먼저 `storage.Database` 안에서 backend 를
직접 분기하고, config/runtime 이 path 와 URL 을 분리해서 전달하도록 해야 한다.

## 호환 가능성이 높은 부분

### Service layer 계약

`app.NewRuntimeWithProviderBuilders` 는 storage repository 를 만든 뒤 service 에
repository interface 로 주입한다. service layer 는 `storage.Database` 의 내부 driver 나
dialect 를 직접 알지 않는다.

관련 파일:

- `app/runtime.go`
- `service/*`
- `storage/*/repository.go`

판단:

- PostgreSQL backend 선택은 app/runtime 과 `storage.Database` 내부 분기에서 끝낼 수 있다.
- service layer 에 backend switch 나 config/env 접근을 넣을 필요는 없다.

### Bun query builder 중심 repository

대부분 repository 는 `NewSelect`, `NewInsert`, `NewUpdate`, `RunInTx`,
`On("CONFLICT ... DO UPDATE")` 같은 Bun API 를 사용한다.

관련 예:

- `storage/dailybar/write_repository.go`
- `storage/indexbar/repository.go`
- `storage/macro/repository.go`
- `storage/companyidentity/repository.go`
- `storage/providerraw/repository.go`
- `storage/financialstatement/repository.go`

판단:

- PostgreSQL 은 `ON CONFLICT DO UPDATE` 를 지원하므로, 이 방향 자체는 PostgreSQL 과
  잘 맞는다.
- Bun `pgdialect` 로 생성 SQL 을 검증해야 하지만, repository 전체를 SQL 문자열로
  다시 쓰는 작업은 필요 없어 보인다.

### Schema source of truth

현재 schema 와 index 는 `storage/database.go` 의 Bun model + schema setup 코드에서
관리한다.

관련 파일:

- `storage/database.go`
- `storage/*_row.go`

판단:

- PostgreSQL 에서도 Bun model 을 기준으로 table/index 를 만들 수 있는 구조다.
- 단, SQLite 전용 schema 보정 함수는 backend 별로 분리해야 한다.

## 조건부 호환 영역

### Bun model tag 와 column type

row model 은 `bun:"...,pk"`, `autoincrement`, `notnull`, `default:''`,
`default:'{}'`, `default:CURRENT_TIMESTAMP` 같은 tag 를 많이 쓴다.

관련 파일:

- `storage/dailybar_row.go`
- `storage/macro_row.go`
- `storage/strategy_row.go`
- `storage/backtest_*_row.go`
- `storage/company_*_row.go`

판단:

- 단순 text/int/bool/time column 은 PostgreSQL 에서도 대체로 매핑 가능성이 높다.
- `autoincrement`, timestamp default, JSON 문자열 default 는 실제 `pgdialect` DDL 로
  확인해야 한다.
- `document_json` 은 현재 TEXT + SQLite trigger 검증 방식이다. PostgreSQL 에서는
  `jsonb` column 으로 바꾸거나 PostgreSQL 전용 CHECK/trigger 가 필요하다.

### Index 생성

`storage/database.go` 는 `db.NewCreateIndex().IfNotExists()` 로 index 를 만든다.

판단:

- PostgreSQL 도 `CREATE INDEX IF NOT EXISTS` 를 지원하므로 기본 index 생성 방식은
  호환 가능성이 높다.
- 다만 index 이름 충돌, unique index 와 nullable column 동작은 PostgreSQL 에서
  별도 smoke test 가 필요하다.

### Reader/Writer 분리

현재 `Database.Reader` 는 SQLite read-only DSN 을 따로 열고, writer 는 단일 connection
으로 둔다.

관련 파일:

- `storage/database.go`

판단:

- repository interface 관점에서는 reader/writer 분리가 유지 가능하다.
- PostgreSQL 에서는 read-only file DSN 이 없으므로 같은 pool 을 반환하거나,
  필요하면 transaction/session read-only 정책을 별도로 설계해야 한다.
- PostgreSQL 에서는 `SetMaxOpenConns(1)` 같은 SQLite writer 제한을 그대로 가져가면
  장점이 사라질 수 있다.

## 현재 그대로는 호환되지 않는 부분

### storage.Database 가 SQLite 전용이다

`storage.Database` 는 현재 `path string` 만 들고 있고, `modernc.org/sqlite` driver 와
`sqlitedialect` 를 직접 import 한다. `Client`, `Reader`, `DB`, `Close` 의 error 메시지도
SQLite path 기준이다.

관련 파일:

- `storage/database.go`

문제:

- PostgreSQL URL 을 받을 필드가 없다.
- `sql.Open("sqlite", db.path)` 와 `bun.NewDB(..., sqlitedialect.New())` 가 고정되어
  있다.
- `os.MkdirAll(filepath.Dir(db.path))` 는 URL backend 에 적용할 수 없다.

필요 작업:

- `DatabaseConfig` 또는 `DatabaseOptions` 에 `Backend`, `Path`, `URL` 을 둔다.
- SQLite 생성자와 PostgreSQL 생성자를 내부에서 분리한다.
- 기존 `NewDatabase(path)` 는 SQLite 호환 생성자로 유지한다.

### SQLite PRAGMA 설정

`setupDatabase` 와 `setupReadDatabase` 는 SQLite PRAGMA 를 실행한다.

관련 파일:

- `storage/database.go`

문제:

- `PRAGMA busy_timeout`, `journal_mode = WAL`, `foreign_keys = ON` 은 PostgreSQL 에서
  동작하지 않는다.

필요 작업:

- SQLite 전용 setup 과 PostgreSQL setup 을 분리한다.
- PostgreSQL 쪽은 `PingContext`, pool 설정, `application_name`, timeout 같은
  connection-level 검증으로 바꾼다.

### SQLite JSON validation trigger

`ensureMacroProviderDocJSONValidation` 은 SQLite 의 `json_valid` 와
`RAISE(ABORT, ...)` 를 사용한다.

관련 파일:

- `storage/database.go`
- `storage/macro_row.go`

문제:

- PostgreSQL 에는 `json_valid` 함수와 SQLite trigger 문법이 없다.

필요 작업:

- PostgreSQL 에서는 `document_json` 을 `jsonb` 로 바꾸는 방안을 우선 검토한다.
- TEXT 로 유지한다면 PostgreSQL 전용 validation function/check/trigger 가 필요하다.

### Legacy column 보정 함수

`ensureStrategyVersionColumns`, `ensureBacktest*Columns` 는
`ALTER TABLE ... ADD COLUMN ...` 을 실행하고, error 문자열에
`duplicate column name` 이 있으면 무시한다.

관련 파일:

- `storage/database.go`

문제:

- PostgreSQL duplicate column error 문구와 타입은 SQLite 와 다르다.
- PostgreSQL 은 `ADD COLUMN IF NOT EXISTS` 를 쓸 수 있으므로 다른 SQL 이 더 맞다.

필요 작업:

- backend 별 column ensure helper 를 둔다.
- PostgreSQL 에서는 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...` 를 쓴다.

### Config 는 path 만 알고 있다

`app/config.DatabaseConfig` 는 `Path` 하나만 가진다. `LoadOrCreate` 는
`DatabasePath`, `DataDirectory`, `ProviderAuthDatabasePath` 를 모두 file path 기준으로
계산한다.

관련 파일:

- `app/config/config.go`

문제:

- `backend`, `url`, `url_env` 를 표현할 수 없다.
- `absolutePath` 와 `filepath.Dir` 흐름은 PostgreSQL URL 에 맞지 않는다.
- `MWOSA_DATABASE` 는 현재 file path 의미다.

필요 작업:

- `DatabaseConfig` 에 `Backend`, `Path`, `URL`, `URLEnv` 를 추가한다.
- resolved config 에 path source 와 url source 를 분리한다.
- provider token cache path 는 canonical database URL 에서 파생하지 않는다.

### CLI 옵션은 SQLite path 기준이다

`cli.Options.Database` 는 "로컬 SQLite database 경로" 로 주석 처리되어 있고,
`--database` flag help 도 `local SQLite database path` 이다. validation 도
`Database` 를 required 로 본다.

관련 파일:

- `cli/root.go`
- `cli/config.go`

문제:

- PostgreSQL backend 에서 `--database` path required 는 맞지 않는다.
- `--database-url`, `--database-backend`, `config use-database` 입력면이 없다.
- `inspect config` 출력도 `database_file` 중심이다.

필요 작업:

- 기존 `--database` 는 SQLite path 호환 flag 로 유지한다.
- `config use-database` 를 권장 명령으로 추가한다.
- `inspect config` 는 backend, path/url source, masked URL 을 출력한다.

### Runtime 의 provider token cache path 파생

`providerAuthDatabasePath` 는 `filepath.Dir(opts.Database)` 에서
`provider-token-cache.sqlite` 를 만든다.

관련 파일:

- `app/runtime.go`

문제:

- `opts.Database` 가 PostgreSQL URL 이 되면 `filepath.Dir` 결과가 의미 없어질 수 있다.

필요 작업:

- provider auth database path 는 config resolved 의 data directory 또는 별도 설정값에서
  받아야 한다.
- canonical storage backend 와 provider token cache backend 는 분리한다.

### Provider auth database 는 별도 SQLite 구현이다

`storage/providerauth.Database` 는 별도 SQLite database 로 고정되어 있다.

관련 파일:

- `storage/providerauth/database.go`

판단:

- feature 설계상 provider token cache 는 제외 범위라서 당장 바꿀 필요는 없다.
- 다만 canonical database URL 과 token cache file path 를 혼동하지 않게 runtime 입력을
  분리해야 한다.

### 테스트는 SQLite metadata 에 의존한다

여러 테스트 helper 가 `PRAGMA`, `sqlite_schema`, `sqlite_master` 를 직접 조회한다.

관련 파일:

- `storage/database_test.go`
- `storage/migration/repository_test.go`
- `storage/composition/repository_test.go`
- `storage/providerauth/repository_test.go`

문제:

- PostgreSQL backend 테스트로 재사용할 수 없다.

필요 작업:

- 기본 `go test ./...` 는 SQLite 테스트로 유지한다.
- PostgreSQL 검증은 Docker/PostgreSQL opt-in integration test 로 분리한다.
- schema/index 검증 helper 는 backend 별로 나눈다.

## 우선 구현 순서 제안

1. `app/config` 에 database backend 설정 모델을 추가한다.
2. `storage.Database` 에 SQLite/PostgreSQL 직접 분기를 만든다.
3. `storage.Database` 의 PostgreSQL path 에 `pgdriver + pgdialect` 를 붙인다.
4. SQLite 전용 setup/schema 보정 함수를 backend 별로 분리한다.
5. `mwosa config use-database ...` 와 `inspect config` 출력 모델을 추가한다.
6. Docker PostgreSQL opt-in smoke test 를 추가한다.

## 판단

현재 구조에서 PostgreSQL backend 는 "service/domain 을 건드리지 않고 storage runtime 과
config/CLI 입력면을 확장하는 작업" 으로 접근할 수 있다.

가장 큰 리스크는 repository logic 이 아니라 SQLite 전용 database setup/schema 보정과
path-only config 모델이다. 따라서 구현 전에는 `storage.Database` 에 작은 backend 별
생성/setup 함수를 직접 두는 것이 먼저다.
