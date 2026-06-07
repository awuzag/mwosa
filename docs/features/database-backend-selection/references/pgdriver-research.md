# pgdriver Research Notes

조사일: 2026-06-07

## 요약

`github.com/uptrace/bun/driver/pgdriver` 는 `mwosa` 의 PostgreSQL backend 1차
후보로 적합하다.

이유는 현재 `mwosa` storage 가 Bun 기반이고, Bun 공식 문서가 PostgreSQL 연결
경로로 `pgdriver` 와 `pgdialect` 조합을 직접 안내하기 때문이다. `pgdriver` 는
`database/sql` driver 이므로 현재 `storage.Database` 가 쓰는 Bun runtime 조립
방식과도 잘 맞는다.

다만 `pgdriver` 공식 문서에서 "pure Go" 라는 표현을 직접 확인하지는 못했다.
따라서 설계 문서에서는 "CGO/libpq 의존이 없는 Go 구현을 우선한다" 로 표현한다.

## 주요 근거

- Bun PostgreSQL 문서는 `pgdriver.NewConnector(pgdriver.WithDSN(dsn))` 로
  `sql.DB` 를 만들고, `bun.NewDB(sqldb, pgdialect.New())` 로 연결하는 예시를
  제공한다.
  - <https://bun.uptrace.dev/postgres/>
- Bun driver/dialect 문서는 database 연결이 `database/sql` 호환 driver 와 Bun
  dialect 의 2층 구조라고 설명한다. 이는 현재 `storage.Database` 가 SQLite 에서
  `sql.DB + Bun dialect` 를 조립하는 방식과 같은 구조다.
  - <https://bun.uptrace.dev/guide/drivers.html>
- `pgdriver` Go package 문서는 이 패키지를 PostgreSQL 용 `database/sql` driver 로
  설명하고, `sql.Open("pg", dsn)` 또는 `sql.OpenDB(pgdriver.NewConnector(...))`
  사용 예시를 제공한다.
  - <https://pkg.go.dev/github.com/uptrace/bun/driver/pgdriver>
- `pgdriver` README 는 이 driver 가 `go-pg` code 기반의 PostgreSQL
  `database/sql` driver 라고 설명한다.
  - <https://raw.githubusercontent.com/uptrace/bun/master/driver/pgdriver/README.md>

## CGO/libpq 의존성 검토

`pgdriver` 문서에서 "pure Go" 라는 문장은 직접 확인하지 못했다. 대신 아래 근거로
CGO/libpq 계열 driver 는 아니라고 판단한다.

- `pgdriver` module 의 `go.mod` 는 Go module dependency 만 나열하고, `libpq`,
  `cgo`, C binding 계열 의존성이 보이지 않는다.
  - <https://raw.githubusercontent.com/uptrace/bun/master/driver/pgdriver/go.mod>
- `driver.go` 는 `database/sql`, `database/sql/driver`, `net`, `crypto/tls` 등을
  사용해 connector 와 connection 을 구현한다.
  - <https://raw.githubusercontent.com/uptrace/bun/master/driver/pgdriver/driver.go>
- `config.go` 는 DSN parsing, TCP/Unix socket, TLS, timeout, connection params 를
  Go code 로 처리한다.
  - <https://raw.githubusercontent.com/uptrace/bun/master/driver/pgdriver/config.go>

## 기능 메모

`pgdriver` 는 단순 연결 외에 다음 API 를 제공한다.

- `CopyFrom`, `CopyTo`
- `Notify`
- `Listener`
- DSN option: `sslmode`, `dial_timeout`, `read_timeout`, `write_timeout`,
  `timeout`, `application_name`
- option API: `WithDSN`, `WithTLSConfig`, `WithUser`, `WithPassword`,
  `WithDatabase`, `WithConnParams`, `WithTimeout`

이 기능들은 1차 backend selection 범위에는 필수는 아니지만, 나중에 대량 적재나
PostgreSQL 운영성 개선으로 확장할 여지는 있다.

## 보안 이력

`pgdriver` 는 과거 SQL injection 취약점 이력이 있다.

- GitHub Advisory `GHSA-h4h6-vccr-44h2` 는 affected version 을 `< 1.2.15`,
  patched version 을 `1.2.15` 로 표시한다.
  - <https://github.com/advisories/GHSA-h4h6-vccr-44h2>
- Go vulnerability database `GO-2025-3765` 도 같은 취약점을 기록하고,
  `github.com/uptrace/bun/driver/pgdriver` 의 `before v1.2.15` 를 affected 로
  표시한다.
  - <https://pkg.go.dev/vuln/GO-2025-3765>

따라서 `mwosa` 에 도입할 때는 최소 `v1.2.15` 이상, 가능하면 현재 Bun 버전과 맞춘
`v1.2.18` 이상을 사용한다.

## 비교 메모

- `pgx` 는 공식 README 에서 "pure Go driver and toolkit" 이라고 직접 명시한다.
  pure Go 문구 자체가 중요한 결정 기준이라면 가장 선명한 대안이다.
  - <https://github.com/jackc/pgx>
- 다만 `mwosa` 는 이미 Bun repository 를 사용하므로, 1차 후보는 Bun 공식 경로와
  호환성이 높은 `pgdriver` 로 둔다.

## 설계 문서 문장 후보

```text
PostgreSQL backend 의 1차 driver 후보는 github.com/uptrace/bun/driver/pgdriver 로
둔다. 이유는 현재 storage 가 Bun 기반이고, pgdriver 는 Bun 공식 PostgreSQL 경로에서
pgdialect 와 함께 안내되는 database/sql driver 이기 때문이다. CGO/libpq 의존은
도입하지 않는다. 단, pgdriver 는 과거 취약점 이력이 있으므로 v1.2.15 이상만
허용한다.
```
