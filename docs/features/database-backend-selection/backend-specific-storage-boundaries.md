# Backend-specific Storage Boundaries

작성일: 2026-06-07

## 목적

SQLite 와 PostgreSQL 을 함께 지원하려면 storage layer 안에서도 공통으로 유지할
부분과 backend 별로 갈라야 할 부분을 먼저 나눠야 한다.

이번 확인의 결론은 단순히 "raw SQL 을 모두 없앤다" 가 아니다. 일반적인
`SELECT`, `JOIN`, `WHERE`, `ORDER BY`, `LIMIT` 형태의 raw SQL 은 PostgreSQL 에서도
대체로 유지 가능성이 있다. 반대로 `PRAGMA`, SQLite catalog, SQLite trigger,
legacy schema repair, file path 기반 connection setup 은 backend 별 책임으로
분리해야 한다.

## 현재 코드 기준 위험도

### 1. 반드시 backend 별로 분리할 영역

`storage.Database` 는 현재 SQLite runtime 을 직접 소유한다.

- `sql.Open("sqlite", path)`
- `bun.NewDB(..., sqlitedialect.New())`
- `modernc.org/sqlite` blank import
- `os.MkdirAll(filepath.Dir(path))`
- read-only SQLite DSN
- SQLite 전용 error message

PostgreSQL URL backend 에서는 위 흐름이 맞지 않는다. 다만 별도 추상 인터페이스를
만들 필요는 없다. `storage.Database` 가 backend 값을 보고 직접 SQLite/PostgreSQL
생성 함수를 호출하는 편이 더 명확하다.

```text
+-------------------------------+
| storage.Database             |
|-------------------------------|
| Client(ctx) / Reader(ctx)    |
| Close()                      |
+---------------+---------------+
                |
                v
+-------------------------------+
| direct backend branch         |
|-------------------------------|
| switch backend                |
| openSQLiteWriter/Reader      |
| openPostgresClient           |
| setupSQLite...               |
| setupPostgres...             |
+---------------+---------------+
        |                       |
        v                       v
+---------------+       +----------------+
| SQLite funcs  |       | PostgreSQL funcs |
+---------------+       +----------------+
```

### 2. PRAGMA 와 connection setup

현재 SQLite writer setup:

```text
PRAGMA busy_timeout = 5000
PRAGMA journal_mode = WAL
PRAGMA foreign_keys = ON
SetMaxOpenConns(1)
```

현재 SQLite reader setup:

```text
file:<path>?mode=ro
PRAGMA busy_timeout = 5000
PRAGMA foreign_keys = ON
SetMaxOpenConns(4)
```

PostgreSQL 에서는 `PRAGMA` 와 file read-only DSN 이 없다. PostgreSQL 쪽은 별도
setup 으로 둔다.

```text
PostgreSQL setup 후보:
  - pgdriver.NewConnector(pgdriver.WithDSN(url))
  - bun.NewDB(sqldb, pgdialect.New())
  - PingContext 로 연결 확인
  - application_name 지정
  - dial/read/write timeout 지정
  - SetMaxOpenConns / SetMaxIdleConns / SetConnMaxLifetime 를 PostgreSQL 기준으로 설정
```

따라서 pool 설정도 backend 별 default 를 가져야 한다. SQLite writer 의
`SetMaxOpenConns(1)` 은 단일 writer 제한 때문에 타당하지만, PostgreSQL 에 그대로
적용하면 동시성 장점을 일부러 꺼버리는 셈이다.

### 3. Schema bootstrap

테이블과 인덱스 생성은 Bun model 과 query builder 를 쓰고 있어서 공통화 가능성이
있다.

```text
NewCreateTable().Model(...).IfNotExists()
NewCreateIndex().Model(...).Index(...).Column(...).IfNotExists()
```

다만 공통으로 둔다고 확정하기 전에 PostgreSQL dialect 로 실제 DDL 을 검증해야
한다. 특히 아래 항목은 PostgreSQL 에서 타입과 default 표현이 달라질 수 있다.

- `autoincrement`
- `default:CURRENT_TIMESTAMP`
- JSON 문자열 default `default:'{}'`
- `bool`
- nullable unique index 의 동작

권장 경계:

```text
setupSchema(ctx):
  createCommonTables(ctx)
  createCommonIndexes(ctx)
  ensureSQLiteCompatibility(ctx) or ensurePostgresCompatibility(ctx)
```

공통 schema 생성 뒤에 작은 backend 별 보정 함수를 직접 호출하는 구조가 가장 작다.

### 4. SQLite trigger / JSON validation

현재 `macro_indicator_provider_doc.document_json` 은 Go model 에서는 `string` 이고,
SQLite 에서는 `TEXT` 로 저장한 뒤 trigger 로 JSON 유효성을 검사한다.

```text
CREATE TRIGGER ...
WHEN json_valid(NEW.document_json) = 0
BEGIN
  SELECT RAISE(ABORT, ...)
END
```

이 방식은 PostgreSQL 에 그대로 갈 수 없다.

PostgreSQL 후보:

- `document_json` 을 `jsonb` 로 만든다.
- repository boundary 에서 `string` 을 계속 받되 DB 저장 타입만 `jsonb` 로 둔다.
- Bun scan/insert 시 `jsonb` 와 Go `string` 의 왕복이 자연스럽지 않으면
  `json.RawMessage` 또는 별도 row type 을 검토한다.
- TEXT 를 유지한다면 PostgreSQL 전용 check/trigger/function 이 필요하지만, v1
  backend 에서는 `jsonb` 가 더 단순하다.

권장 판단:

```text
SQLite:
  TEXT + json_valid trigger 유지

PostgreSQL:
  jsonb column 우선 검토
  필요 시 PostgreSQL 전용 row/model 또는 schema override 사용
```

### 5. Legacy schema repair

현재 column 보정 함수들은 아래 패턴을 쓴다.

```text
ALTER TABLE <table> ADD COLUMN <name> <definition>
if error contains "duplicate column name": ignore
```

이 방식은 PostgreSQL 에 맞지 않는다.

- PostgreSQL 의 duplicate column error 문구는 SQLite 와 다르다.
- PostgreSQL 은 `ADD COLUMN IF NOT EXISTS` 를 지원한다.
- 장기적으로는 schema repair 와 migration runner 의 책임 경계도 정리해야 한다.

권장 경계:

```text
sqliteEnsureColumns(ctx):
  ALTER TABLE ... ADD COLUMN ...
  duplicate column name 문자열은 SQLite 전용 함수 안에서만 처리

postgresEnsureColumns(ctx):
  ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...
  PostgreSQL 타입/default 문법 사용
```

`ensureStrategyVersionColumns`, `ensureBacktest*Columns` 는 backend 별 schema repair
함수로 직접 분리하는 것이 좋다.

### 6. Raw SELECT query

repository 에 있는 raw query 중 상당수는 데이터 조회용이다.

예:

- `dailybar` 의 일봉 조회/coverage query
- `composition` 의 observation/member 조회
- `instrument` 의 extension 조회
- `companyidentity` 의 company/identifier/link 조회
- `strategyfundamentals` 의 분석용 조회

이들은 대체로 표준 SQL 에 가깝다. 바로 분리하기보다 PostgreSQL integration test 로
검증하면서 필요한 것만 고치는 편이 좋다.

주의할 표현:

- `?` placeholder 는 Bun/driver 조합에서 그대로 허용되는지 확인한다.
- `LIKE` / `lower()` 정렬은 PostgreSQL 에서도 동작하지만 collation 차이가 있을 수 있다.
- `COUNT(DISTINCT ...)`, `COALESCE`, `CASE WHEN`, `LIMIT` 은 대체로 호환 가능성이 높다.

권장 판단:

```text
raw SELECT 는 v1 에서 유지한다.
PostgreSQL smoke test 가 깨지는 query 만 query builder 또는 dialect branch 로 바꾼다.
```

### 7. Migration SQL

`storage/migration/dailybar_v2_extension_cleanup.go` 의 SQL 은 단순한
`DELETE` 와 `DROP INDEX IF EXISTS` 이므로 PostgreSQL 에서도 형태상 가능성이 있다.

다만 migration 이 특정 backend 의 과거 schema 를 전제로 한다면 backend support
matrix 를 가져야 한다.

```text
migration definition:
  id
  resource
  from/to
  supported_backends: sqlite, postgres
```

처음부터 모든 과거 SQLite migration 을 PostgreSQL 에도 적용하려고 하지 말고,
PostgreSQL 은 새 schema bootstrap 이후부터 지원하는 migration 만 opt-in 하는 방향이
더 안전하다.

## 분리 우선순위

1. `storage.Database` 에 backend 옵션을 추가하고, 내부에서 직접 `switch` 한다.
2. SQLite PRAGMA/setup/pool/read-only DSN 을 SQLite 전용 함수로 옮긴다.
3. PostgreSQL 전용 함수는 pgdriver/pgdialect/open/ping/pool 만 먼저 가진다.
4. schema bootstrap 은 공통으로 시도하되 backend 별 보정 함수를 직접 호출한다.
5. SQLite trigger 와 legacy column repair 는 backend 별 함수로 분리한다.
6. raw SELECT 는 유지하고 PostgreSQL integration test 결과로 필요한 부분만 수정한다.
7. migration definition 에 backend support 개념을 추가할지 검토한다.

## 설계 문서에 넣을 문장 후보

`mwosa` 의 database backend 분리는 repository 전체를 backend 별로 복제하지 않는다.
우선 `storage.Database` 내부에서 backend 를 직접 분기하고, driver/dialect 연결,
PRAGMA 또는 pool 설정, schema repair, JSON validation trigger 같은 database-specific
setup 만 작은 backend 별 함수로 분리한다. 일반 조회용 raw SQL 은 PostgreSQL 검증에서
깨지는 지점만 좁게 수정한다.

PostgreSQL backend 는 SQLite 의 file/read-only DSN 과 PRAGMA 흐름을 공유하지 않는다.
대신 pgdriver/pgdialect 기반 connection, `PingContext`, PostgreSQL 기준 pool default,
그리고 필요한 schema 보정 함수를 별도로 가진다.
