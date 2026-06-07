# Bun SQLite / PostgreSQL Compatibility Notes

조사일: 2026-06-07

## 요약

Bun 공식 문서 기준으로 SQLite 와 PostgreSQL 은 같은 Bun API 위에서 함께 다룰 수
있다. 다만 호환성의 의미는 "동일 SQL 이 어디서나 그대로 동작한다" 가 아니라,
`database/sql` driver 와 Bun dialect 를 조합하고, feature detection 또는 dialect check 로
DB 별 차이를 관리한다는 뜻에 가깝다.

`mwosa` 관점에서는 현재 Bun model 과 repository query builder 를 유지할 수 있는 근거가
있다. 반면 SQLite 전용 raw SQL, PRAGMA, trigger, connection pool 정책은 PostgreSQL 에
그대로 가져갈 수 없다.

## 공식 문서 근거

### Driver + dialect 구조

Bun driver/dialect 문서는 database 연결에 두 요소가 필요하다고 설명한다.

- `database/sql` 호환 driver
- Bun dialect

이 구조는 `mwosa` 의 현재 SQLite runtime 과 PostgreSQL 후보 runtime 을
`storage.Database` 안에서 같은 생성 흐름으로 직접 조립하기 좋은 근거다.

출처:

- <https://bun.uptrace.dev/guide/drivers.html>

관련 스크랩:

```text
Bun connects through a database/sql-compatible driver plus a Bun dialect.
The dialect translates Bun query builder syntax into each database's SQL dialect.
```

### Multi-database support

Bun guide 는 Bun 이 PostgreSQL, MySQL, SQLite, SQL Server 를 지원한다고 설명한다.
또한 Bun 이 `database/sql` 위에서 동작하므로 기존 `sql.DB` 와 함께 사용할 수 있다고
설명한다.

출처:

- <https://bun.uptrace.dev/guide/>

관련 스크랩:

```text
Bun supports multiple databases including PostgreSQL and SQLite.
Bun works on top of database/sql and keeps access to the underlying sql.DB.
```

### PostgreSQL 지원

Bun driver/dialect 문서는 PostgreSQL 을 Bun 의 primary supported database 라고 설명하고,
JSON/JSONB, advanced index, window function, CTE, `ON CONFLICT DO UPDATE` 지원을
강조한다.

PostgreSQL 전용 문서는 `pgdriver + pgdialect` 조합 예시를 제공한다.

출처:

- <https://bun.uptrace.dev/guide/drivers.html>
- <https://bun.uptrace.dev/postgres/>

관련 스크랩:

```text
PostgreSQL is Bun's primary supported database with full feature compatibility.
The PostgreSQL setup path uses pgdriver.NewConnector(...WithDSN(...)) and pgdialect.New().
```

### SQLite 지원

Bun driver/dialect 문서는 SQLite 를 development, testing, lightweight application 에
적합하다고 설명한다. SQLite 연결 예시는 `sqliteshim + sqlitedialect` 조합을 사용한다.

문서상 `sqliteshim` 은 `modernc.org/sqlite` 와 `mattn/go-sqlite3` 중 플랫폼에 맞는
구현을 고른다. `modernc.org/sqlite` 는 pure Go 구현이며 대부분 플랫폼의 기본값으로
설명된다.

출처:

- <https://bun.uptrace.dev/guide/drivers.html>

관련 스크랩:

```text
SQLite uses sqlitedialect with a SQLite driver.
sqliteshim chooses modernc.org/sqlite as a pure Go implementation by default on most platforms,
or mattn/go-sqlite3 as the CGO-based implementation.
```

### 공통 feature 와 차이 관리

Bun 문서는 database-specific code 를 다루는 방법으로 feature detection 과 direct
dialect checking 을 제안한다.

공식 feature 표 기준:

- `InsertOnConflict`: PostgreSQL, SQLite
- `InsertReturning`: PostgreSQL, SQLite
- `CTE`: PostgreSQL, SQLite, SQL Server
- `UpdateReturning`: PostgreSQL
- `DeleteReturning`: PostgreSQL
- `Window`: PostgreSQL, MySQL 8+, SQL Server

출처:

- <https://bun.uptrace.dev/guide/drivers.html>

관련 스크랩:

```text
Bun recommends feature detection or direct dialect checks for database-specific behavior.
PostgreSQL and SQLite both support InsertOnConflict and InsertReturning in Bun's feature table.
Some features, such as UpdateReturning and DeleteReturning, are PostgreSQL-only.
```

### Connection pool 권장 차이

Bun 문서는 production 에서 `sql.DB` pool 설정을 중요하게 다룬다. driver/dialect 문서의
database-specific recommendation 에서는 PostgreSQL 과 SQLite pool 특성이 다르게 제시된다.

출처:

- <https://bun.uptrace.dev/guide/running-bun-in-production.html>
- <https://bun.uptrace.dev/guide/drivers.html>

관련 스크랩:

```text
Bun uses sql.DB for database communication and recommends configuring connection pools.
PostgreSQL can use a larger connection pool, while SQLite is documented with single-writer
limitations and small pool settings.
```

## mwosa 에 적용되는 호환 판단

### 호환 가능성이 높은 부분

- 현재 repository 의 Bun query builder 사용은 PostgreSQL 로 확장하기 좋은 방향이다.
- `ON CONFLICT DO UPDATE` 는 Bun feature 표에서 PostgreSQL 과 SQLite 모두 지원된다.
- `INSERT ... RETURNING` 도 Bun feature 표에서 PostgreSQL 과 SQLite 모두 지원된다.
- Bun model tag 의 기본적인 table/column/primary key/default/notnull 표현은 dialect 를
  통해 변환될 가능성이 있다.

### 주의가 필요한 부분

- Bun 이 여러 DB 를 지원하더라도 raw SQL 은 자동으로 이식되지 않는다.
- SQLite PRAGMA, `sqlite_schema`, `sqlite_master`, `json_valid`, SQLite trigger 문법은
  PostgreSQL 에 맞지 않는다.
- PostgreSQL 은 JSON/JSONB 지원이 강하므로, 현재 TEXT + SQLite JSON validation trigger 는
  PostgreSQL 에서 `jsonb` 또는 별도 PostgreSQL validation 으로 다시 판단해야 한다.
- SQLite 와 PostgreSQL 의 connection pool 권장값은 다르다. 현재 SQLite writer 를
  `SetMaxOpenConns(1)` 로 둔 정책을 PostgreSQL 에 그대로 적용하면 안 된다.
- `sqliteshim` 이 있지만, 현재 `mwosa` 는 `modernc.org/sqlite` 를 직접 import 하고
  `sql.Open("sqlite", path)` 를 쓴다. `sqliteshim` 도입은 별도 선택지이며 필수는 아니다.

## 구현 메모

PostgreSQL backend 를 붙일 때 Bun 호환성 관점의 최소 구조는 아래와 같다.

```go
sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
db := bun.NewDB(sqldb, pgdialect.New())
```

SQLite 기존 구조는 아래와 같다.

```go
sqldb, err := sql.Open("sqlite", path)
db := bun.NewDB(sqldb, sqlitedialect.New())
```

따라서 `storage.Database` 는 driver/dialect 조립을 backend 별 함수로 직접 나누면 된다.

```text
sqlite:
  driver  = modernc.org/sqlite
  dialect = sqlitedialect.New()
  setup   = PRAGMA + SQLite schema repair

postgres:
  driver  = github.com/uptrace/bun/driver/pgdriver
  dialect = pgdialect.New()
  setup   = ping/pool/application_name + PostgreSQL schema repair
```

## 결론

Bun 문서만 놓고 보면 PostgreSQL 과 SQLite 를 하나의 Bun 기반 storage layer 안에서 함께
지원하는 방향은 타당하다. 단, 이 호환성은 Bun query builder 와 dialect 수준의 호환성이다.
`mwosa` 의 SQLite 전용 setup, schema 보정, 테스트 helper, file path config 는 별도
분리가 필요하다.
