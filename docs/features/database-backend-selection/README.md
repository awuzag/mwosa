# Database Backend Selection

## 목적

`mwosa` 의 canonical storage 를 실행 환경에 맞게 선택할 수 있게 한다.

현재 기본 저장소는 SQLite 파일이다. 이 피처는 기존 SQLite 기본값을 유지하면서,
Docker 로 띄운 PostgreSQL 서버처럼 URL 로 접속하는 database backend 를 선택할 수
있는 경로를 추가한다.

## 문제

SQLite 파일은 설치와 개인 사용이 단순하다. 별도 서버가 필요 없고 CLI 기본값으로
좋다.

하지만 장시간 수집, 여러 프로세스 접근, 서버형 실험, Docker 기반 개발 환경에서는
PostgreSQL 같은 URL 기반 database 가 더 다루기 쉽다. 지금 구조에서는 database 입력이
대체로 파일 경로로 해석되므로, backend 종류와 연결 정보를 분리해서 표현하기 어렵다.

## 기본 방향

- 기본 backend 는 계속 `sqlite` 로 둔다.
- PostgreSQL 은 먼저 Docker 기반 개발/실험 backend 로 추가한다.
- database driver 는 CLI 배포 제약을 줄이기 위해 CGO 없는 pure Go 구현을 우선한다.
- service layer 와 domain layer 는 database backend 를 알지 않는다.
- backend 선택은 config 를 읽고 runtime 을 조립하는 단계에서 끝낸다.
- repository 는 지금처럼 service layer 의 repository interface 를 구현한다.
- PostgreSQL URL 에 포함될 수 있는 비밀번호는 inspect 출력에서 마스킹한다.

## 레이어 경계

```text
+--------------------------------------------------+
| CLI Layer                                        |
|--------------------------------------------------|
| flags, env, config use-database, inspect config  |
| database path/url 입력을 config layer 로 전달한다 |
+-------------------------+------------------------+
                          |
                          v
+--------------------------------------------------+
| App / Runtime Layer                              |
|--------------------------------------------------|
| config 를 읽고 database backend 를 결정한다       |
| storage.Database 와 repository 를 조립한다        |
+-------------------------+------------------------+
                          |
                          v
+--------------------------------------------------+
| Service Layer                                    |
|--------------------------------------------------|
| daily, macro, strategy, backtest service          |
| repository interface 만 의존한다                 |
+-------------------------+------------------------+
                          |
                          v
+--------------------------------------------------+
| Storage Repository Layer                         |
|--------------------------------------------------|
| dailybar, macro, strategy, backtest repository    |
| Bun client 를 사용하되 config/env 를 읽지 않는다  |
+-------------------------+------------------------+
                          |
                          v
+--------------------------------------------------+
| Storage Database Runtime                         |
|--------------------------------------------------|
| storage.Database                                 |
| backend 별 driver, dialect, schema setup 을 직접 분기한다 |
+-------------+----------------------+-------------+
              |                      |
              v                      v
+-------------------------+   +-------------------------+
| SQLite Backend          |   | PostgreSQL Backend       |
|-------------------------|   |-------------------------|
| file path               |   | connection URL           |
| local canonical DB      |   | Docker/server DB         |
| default                 |   | opt-in                   |
+-------------------------+   +-------------------------+
```

## 설정 모델 초안

SQLite 기본 설정:

```json
{
  "app": {
    "database": {
      "backend": "sqlite",
      "path": "~/.local/share/mwosa/mwosa.db"
    }
  }
}
```

PostgreSQL 설정:

```json
{
  "app": {
    "database": {
      "backend": "postgres",
      "url_env": "MWOSA_DATABASE_URL"
    }
  }
}
```

`url` 직접 저장도 가능하게 할 수 있지만, 기본 문서와 예시는 `url_env` 를 우선한다.
connection URL 에 사용자 이름과 비밀번호가 들어갈 수 있기 때문이다.

## 입력 우선순위 초안

SQLite 파일 경로:

1. `--database`
2. `MWOSA_DATABASE`
3. config file 의 `app.database.path`
4. OS 기본 data 경로

PostgreSQL URL:

1. `--database-url`
2. `MWOSA_DATABASE_URL`
3. config file 의 `app.database.url`
4. config file 의 `app.database.url_env` 가 가리키는 환경변수

backend 선택:

1. `--database-backend`
2. `MWOSA_DATABASE_BACKEND`
3. config file 의 `app.database.backend`
4. 기본값 `sqlite`

## CLI 명령 초안

사용자가 직접 쓸 기본 명령은 `config use-database` 로 둔다. Kubernetes 의
`kubectl config use-context` 처럼 "현재 사용할 database backend 를 고른다"는 의미를
앞세운다.

PostgreSQL 로 전환:

```bash
mwosa config use-database postgres --url-env MWOSA_DATABASE_URL
```

SQLite 파일로 전환:

```bash
mwosa config use-database sqlite --path ~/.local/share/mwosa/mwosa.db
```

직접 URL 을 넘기는 명령도 지원할 수 있지만, 출력과 config 저장 시 secret 마스킹을
반드시 적용한다.

```bash
mwosa config use-database postgres \
  --url 'postgres://mwosa:mwosa@localhost:5432/mwosa?sslmode=disable'
```

전환 후 확인은 기존 inspect 흐름을 쓴다.

```bash
mwosa inspect config
```

기존 `config set` 으로도 같은 값을 직접 바꿀 수는 있다. 다만 이는 저수준 escape
hatch 이며 권장 사용법은 아니다. 일반 문서와 예시는 한 줄 전환 명령만 우선한다.

```bash
mwosa config set app.database.backend postgres
mwosa config set app.database.url_env MWOSA_DATABASE_URL
```

## Docker 개발 경험

1차 목표는 아래 흐름이면 충분하다.

```text
docker compose up postgres
export MWOSA_DATABASE_URL='postgres://mwosa:mwosa@localhost:5432/mwosa?sslmode=disable'
mwosa config use-database postgres --url-env MWOSA_DATABASE_URL
mwosa inspect config
mwosa sync ...
```

Docker compose 예시는 별도 구현 단계에서 `docs/development` 또는 `examples` 아래에 둔다.
이 피처 문서는 먼저 config contract 와 layer boundary 를 고정한다.

## Provider token cache

canonical storage backend 와 provider token cache 는 같은 개념이 아니다.

PostgreSQL 을 canonical storage 로 선택하더라도 provider token cache 는 당장 기존 SQLite
sidecar 파일을 유지한다. token cache 까지 PostgreSQL 로 옮기는 것은 별도 보안/운영
결정으로 분리한다.

## PostgreSQL driver 후보

PostgreSQL backend 의 1차 driver 후보는 `github.com/uptrace/bun/driver/pgdriver`
로 둔다. 이유는 현재 storage 가 Bun 기반이고, `pgdriver` 는 Bun 공식 PostgreSQL
경로에서 `pgdialect` 와 함께 안내되는 `database/sql` driver 이기 때문이다.
CGO/libpq 의존은 도입하지 않는다. 단, `pgdriver` 는 과거 취약점 이력이 있으므로
`v1.2.15` 이상만 허용한다.

조사 근거는 [pgdriver research notes](references/pgdriver-research.md) 에 둔다.

부가적인 Bun SQLite/PostgreSQL 호환성 조사 근거는
[Bun SQLite / PostgreSQL compatibility notes](references/bun-sqlite-postgres-compatibility.md)
에 둔다.

현재 코드베이스 기준 호환성 분석은 [current codebase compatibility](current-codebase-compatibility.md)
에 둔다.

SQLite 전용 setup 과 PostgreSQL setup 의 분리 기준은
[backend-specific storage boundaries](backend-specific-storage-boundaries.md)
에 둔다.

## 구현 시 고려 사항

- `storage.Database` 는 backend, path, url 을 포함한 option 을 받도록 확장한다.
- 기존 `storage.NewDatabase(path)` 호출은 SQLite path 호환 생성자로 유지한다.
- PostgreSQL 은 Bun `pgdialect` 와 CGO 없는 pure Go PostgreSQL driver 를 사용한다.
- 후보 driver 는 Bun 과 같은 생태계의 `github.com/uptrace/bun/driver/pgdriver` 를 먼저 검토한다.
- SQLite 전용 SQL 은 backend 별로 분기한다.
- `PRAGMA`, `sqlite_schema`, `json_valid` 기반 보정은 PostgreSQL 에 그대로 적용하지 않는다.
- `inspect config` 는 backend, source, path/url 상태를 보여주되 secret 을 출력하지 않는다.
- `config use-database` 는 backend 별 필수 인자를 검증하고 잘못된 조합을 명시적 error 로 반환한다.
- 기본 `go test ./...` 는 SQLite 기반으로 유지한다.
- PostgreSQL 검증은 `integration` 또는 `e2e` build tag 로 분리한다.

## 제외 범위

- SQLite 데이터를 PostgreSQL 로 자동 migration 하는 기능.
- PostgreSQL 운영 배포 가이드.
- provider token cache backend 전환.
- MySQL backend 지원.
- service layer repository 계약 변경.

## 완료 기준

- 기존 SQLite 파일 기반 실행이 설정 변경 없이 계속 동작한다.
- `mwosa config use-database ...` 한 줄로 backend 를 전환할 수 있다.
- config/env/flag 로도 backend 를 명시할 수 있다.
- PostgreSQL URL 은 Docker PostgreSQL 서버에 접속할 수 있다.
- service layer 코드는 backend 종류를 알지 않는다.
- `inspect config` 에서 URL secret 이 마스킹된다.
- PostgreSQL 테스트는 opt-in 으로 분리된다.
