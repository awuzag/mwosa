# Technology Stack

## 목적

이 문서는 `mwosa` Go CLI 의 기술 스택 결정을 정리한다.

지금은 구현을 시작하기 위한 기본 선택만 적는다. 세부 라이브러리와 운영 도구는 실제 구현 대화를 진행하면서 이 문서에 추가한다.

## 현재 확정

### Language

- `Go`

선정 이유:

- 단일 바이너리 CLI 배포가 쉽다.
- 파일 I/O, HTTP client, context cancellation, 병렬 처리에 적합하다.
- provider adapter 와 service interface 를 명확하게 나누기 좋다.

### CLI framework

- `spf13/cobra`

선정 이유:

- subcommand 기반 CLI 구조에 적합하다.
- help, completion, persistent flag, local flag 를 기본 지원한다.
- `mwosa inspect portfolio`, `mwosa get quote`, `mwosa calc rsi` 같은 다층 command tree 를 표현하기 좋다.

적용 범위:

- `cmd/mwosa`
- `cli`
- `command/*`

### Error handling

- `github.com/samber/oops`

선정 이유:

- operation, domain, key/value context, cause 를 함께 보존하기 좋다.
- provider, storage, service, CLI 경계에서 실패 맥락을 명시적으로 붙이기 좋다.
- fallback 판단, 사용자 메시지, 로그 진단에 필요한 정보를 error chain 에 남기기 좋다.

결정:

- 하위 레이어 error 는 호출 경계에서 `oops.In(...).With(...).Wrap(err)` 형태로 명시적으로 감싼다.
- error code, fallback 판단, 사용자 메시지 분리는 각 아키텍처 결정 문서에서 해당 경계에 맞게 관리한다.
- invalid input, partial data, provider failure 를 성공처럼 숨기지 않는다.

적용 범위:

- `cli`
- `service`
- `providers/*`
- `storage`

### Testing

- `testing`
- `github.com/stretchr/testify`

선정 이유:

- Go 표준 `testing` 은 기본 테스트 실행과 benchmark 에 그대로 사용한다.
- `testify` 는 assertion 과 require 를 명확하게 표현하기 좋다.
- 보조지표 계산처럼 기준값을 비교해야 하는 테스트에서 실패 위치와 기대값 차이를 읽기 좋다.

결정:

- 단위 테스트는 Go 표준 `testing` 을 기반으로 작성한다.
- assertion 은 `testify/assert` 와 `testify/require` 를 사용한다.
- 계산 함수는 함수별 단위 테스트를 필수로 둔다.
- 입력 검증 실패처럼 테스트를 계속 진행할 수 없는 조건은 `require` 로 멈춘다.
- 계산 결과 비교처럼 여러 값을 확인해야 하는 조건은 `assert` 로 표현한다.

### Canonical source of truth

- `Local SQLite database`

결정:

- provider-neutral canonical record 를 정규화된 로컬 SQLite database 에 저장한다.
- ETF 일봉, 종목 정보, 산출 지표, 리더 바스켓 같은 CLI 기능의 기준 데이터는 SQLite 를 정본으로 삼는다.
- 원천 API 응답은 검증과 실험용 산출물로만 다루고, 기본 저장소에는 중복 보관하지 않는다.
- 사람이 읽기 쉬운 export 가 필요하면 별도 CLI command 에서 SQLite 데이터를 변환한다.

### Embedded database

- `SQLite`
- `modernc.org/sqlite`

결정:

- CLI 배포 제약을 줄이기 위해 CGO 없는 pure Go SQLite driver 를 사용한다.
- ETF 일봉과 일일 순위 규모에서는 단일 SQLite 파일로 충분하다는 실험 결과를 기준으로 삼는다.
- 순위 조회는 `(bas_dt, metric DESC)` 계열 인덱스를 적극적으로 사용한다.
- 리더 스냅샷은 꼭 보관해야 하는 의사결정 결과만 저장하고, 일반 순위는 쿼리로 계산한다.

적용 범위:

- `storage`
- `storage/<resource>`
- `testing/experiments/sqlite_capacity_runtime`

### Database access

- `Bun`
- `modernc.org/sqlite`

결정:

- schema source of truth 는 `.sql` 파일이 아니라 Bun model 과 schema 생성 코드로 둔다.
- `storage.Database` 는 lazy하게 `*bun.DB` 를 열고, SQLite 설정과 schema/index 생성을 적용한다.
- resource 별 repository 는 `storage/<resource>` 아래에 둔다.
- service layer 는 Bun DB 나 storage row type 을 직접 알지 않는다.
- persistence layer 는 `ReadRepository` 와 `WriteRepository` interface 를 분리해서 구현한다.
- embedded SQLite 는 네트워크 왕복 비용이 없으므로, 작은 CLI 조회 경로에서는 전통적인 N+1 회피를 우선 설계 목표로 두지 않는다.
- 전체 ETF, 장기간 일봉, 백테스트, 지표 재계산처럼 반복 범위가 큰 작업은 Bun query 를 우선하되, 필요하면 persistence layer 안에서만 명시 SQL helper 를 보조적으로 사용할 수 있다.

### Database migration

- `Bun model-managed schema`

결정:

- schema 변경은 Bun model 과 schema/index 생성 코드 변경으로 관리한다.
- CLI command 가 SQLite 저장소에 실제로 접근할 때 Bun 기반 table/index 생성을 실행한다.
- 현재 단계에서는 별도 `.sql` migration 파일을 source of truth 로 만들지 않는다.
- destructive schema 변경은 자동으로 숨기지 않고 별도 결정과 테스트를 거친다.

### Provider implementation

- `Independent Go modules in a Go workspace`

결정:

- `mwosa` repository root 는 Go workspace 로 관리한다.
- root 의 `go.work` 로 CLI module 과 provider client module 을 함께 개발한다.
- 각 provider client 는 workspace 안의 독립 Go module 로 생성한다.
- provider client module 은 자체 `go.mod` 와 단위 테스트를 가진다.
- CLI module 에는 `providers/core` 와 provider 별 adapter 를 둔다.
- provider 를 등록하기 전에 client module 단위 테스트를 먼저 통과시킨다.

예:

- provider client module: `./clients/marketdata-provider-kis`
- provider client module: `./clients/datago-etp`
- in-repo adapter: `providers/kis`
- in-repo adapter: `providers/datago`

provider 이름은 큰 데이터 소스 단위로 둔다. 공공데이터포털처럼 개별 API 서비스가 따로 승인되는 경우에도 provider id 는 `datago` 로 유지하고, 세부 API 범위는 provider group 으로 표현한다.

### Core packages

- `packages/indicators`
- `packages/backtest`

결정:

- 투자 리서치에 필요한 보조지표 계산은 provider 나 CLI command 내부에 두지 않는다.
- `packages/indicators` 를 독립 계산 패키지 후보로 둔다.
- YAML 기반 백테스트 engine 은 `packages/backtest` 를 독립 core package 후보로 둔다.
- MACD, 일목균형표, 이동평균은 검토 대상의 예시일 뿐 전체 범위를 제한하지 않는다.
- 이 core package 들은 provider, storage, Cobra, presentation 에 의존하지 않는다.
- service layer 는 canonical candle 데이터를 indicators 입력 타입으로 변환한 뒤 계산 패키지를 호출한다.
- 외부 보조지표 라이브러리는 public API 로 노출하지 않고, 필요하면 package 내부 구현 세부사항으로만 사용한다.

### HTTP client

- `Not fixed yet`

결정:

- provider 가 REST API 를 사용할 때의 HTTP client 는 실제 provider 구현 단계에서 정한다.
- 기본 후보는 Go 표준 라이브러리 `net/http` 로 둔다.
- retry/backoff, request builder, tracing 같은 필요가 반복될 때만 wrapper 나 별도 client 를 검토한다.
- provider role interface 와 service layer 에는 특정 HTTP client library type 을 노출하지 않는다.

세부 협업 기준은 `docs/development/README.md` 에 둔다.

### Configuration

- `Go standard library + explicit config package`

결정:

- 초기에는 별도 설정 framework 를 도입하지 않는다.
- 환경변수, 설정 파일, 기본 경로 처리는 `config` 에서 직접 다룬다.
- 설정 파일과 로컬 데이터 파일의 기본 경로 기준은 `docs/architectures/configuration/README.md` 에 둔다.

## 아직 정하지 않음

다음 항목은 구현 과정에서 필요가 분명해질 때 결정한다.

- logging library
- table rendering library
- config file format
- NDJSON 이후 추가 저장 포맷
- provider package repository strategy
- indicator engine implementation library
- backtester engine implementation library

## 관련 문서

- `docs/architectures/layers/README.md`
- `docs/architectures/provider/README.md`
- `docs/architectures/indicator/README.md`
- `docs/architectures/packages/README.md`
- `docs/architectures/configuration/README.md`
- `docs/architectures/packages/indicators/README.md`
- `docs/architectures/packages/backtest/README.md`
- `docs/architectures/completion/README.md`
- `docs/features/backtester/README.md`
- `docs/development/README.md`
- `docs/canonical-schema.md`
