# Aggregate Architecture

## 목적

이 문서는 `kind: Aggregate` 를 `mwosa` 의 상위 데이터 처리 리소스로 정의한다.

핵심 문장은 다음과 같다.

> Aggregate는 provider/API 호출, provider raw 호출, 로컬 MongoDB 조회, MongoDB aggregation, 출력 포맷, 실행 기록을 하나로 묶는 저장 가능한 데이터 처리 리소스

`Aggregate` 는 사용자가 기존 Python 스크립트로 처리하던 데이터 수집, 합성,
가공, 테이블 출력을 `mwosa` 안에서 저장하고 반복 실행할 수 있게 하는
리소스다. 장기적으로는 기존 `screen strategy`, `jq strategy`, 일부
`backtest` 준비 작업의 상위 기반이 될 수 있지만, 첫 버전에서 이 기능들을
모두 대체한다고 보지 않는다.

관련 경계는 아래 문서를 따른다.

- [Layer Architecture](../layers/README.md)
- [Provider Architecture](../provider/README.md)
- [Research Aggregate / Projection Boundary](../research-aggregate/README.md)
- [Indicator Architecture](../indicator/README.md)
- [Technology Stack](../tech-stack/README.md)

## 문서 위치

문서 위치는 `docs/architectures/aggregate/README.md` 로 둔다.

`docs/architectures/research-aggregate/README.md` 는 provider 응답이나
storage row 를 domain aggregate 로 삼지 않고 read model/projection 경계를
분리하는 문서다. 이번 `Aggregate` 는 DDD aggregate 가 아니라 저장 가능한
실행 리소스이므로 별도 문서가 더 맞다.

## 요구사항 요약

- 기존 Python 스크립트가 맡던 파이프라인 처리와 데이터 가공을 `mwosa` 안에서
  저장하고 언제든 실행할 수 있어야 한다.
- YAML 로 `kind: Aggregate` 를 정의하고, 이름, version, spec hash 를 가진
  저장 리소스로 관리한다.
- `mwosa run aggregate <name>` 과 `--param key=value` 로 저장된 Aggregate 를
  실행하고 기본 param 을 덮어쓴다.
- pipeline stage 는 provider API, provider raw API, 로컬 MongoDB collection,
  mwosa canonical dataset, raw snapshot, 이전 Aggregate 결과, MongoDB aggregation,
  jq 변환을 명시적으로 고른다.
- API 호출과 local 조회 사이에 애매한 fallback 은 두지 않는다.
- provider raw stage 는 live API 호출이고, local raw snapshot 은 `snapshot`
  stage 로 별도 선언한다.
- 각 stage 결과는 다음 stage 가 읽을 수 있는 이름 있는 결과로 materialize 한다.
- MongoDB aggregation 은 원문 pipeline 을 최대한 유지하고, SQLite fallback 은
  제공하지 않는다.
- 실행 기록, 입력 stage, runtime params, spec hash, 결과 row 를 저장하고
  나중에 history/inspect 계열 명령으로 조회한다.
- 결과는 `table`, `json`, `ndjson`, `csv` 로 출력하고, 출력 컬럼과 포맷은
  YAML 에서 정의한다.
- provider 인증, API 실패, 누락 데이터는 조용히 넘기지 않는다.
- rate limit, retry, pagination 은 provider adapter 가 책임진다. Aggregate YAML 은
  provider 별 호출 제한값을 받지 않고, provider 가 드러낸 실패를 실행 기록에 남긴다.
- Aggregate 결과는 다른 Aggregate, screen, backtest, report 의 입력으로
  재사용할 수 있어야 한다.

## 현재 코드 조사 결과

이 문서의 예시는 현재 코드에 있는 이름을 우선 사용한다. 아직 없는 이름은
요구사항이나 확장 후보로만 적는다.

- KIS raw operation `inquire-daily-itemchartprice` 는 실제 구현된 이름이다.
  `docs/providers/kis/README.md` 의 raw fetch 예시와 `providers/kis/raw.go` 가
  같은 operation 을 사용한다.
- KIS API 와 raw 호출은 provider 내부 `withReadRetry` 경로를 탄다.
  `providers/kis/rate_limit.go` 에 token bucket 기반 read limiter 와 retry
  정책이 있고, `providers/kis/provider.go`, `providers/kis/raw.go` 가 이 경로를
  호출한다. Aggregate 가 YAML 에서 rate limit 값을 받는 구조로 만들지 않는다.
- 현재 MongoDB collection 으로 확인한 이름은 `instruments`, `daily_bars`,
  `provider_raw_snapshots`, `valuation_snapshots` 등이다.
- 현재 jq/screen 입력 dataset 으로 확인한 이름은 `daily_bar`, `daily_bars`,
  `etf_daily_metrics`, `stock_daily_metrics`, `stock_daily_fundamentals` 다.
- `krx_listed_stocks` 는 현재 코드에 있는 dataset 이름이 아니다. 종목 universe 는
  우선 `instruments` collection 또는 기존 dataset 을 쓰고, 별도 canonical dataset
  이 필요하면 나중에 추가한다.

## 기존 경계와의 관계

`Aggregate` 는 새 provider 구현체나 새 계산 패키지가 아니다. 기존 레이어
경계를 유지하면서 실행 단위를 하나 더 추가한다.

```text
CLI command
  -> cli handler
  -> service/aggregate
  -> pipeline stage executor
  -> provider router, provider raw adapter, MongoDB, jq engine
  -> storage/mongodb workspace and aggregate repository
  -> presentation
```

역할 기준은 다음과 같다.

| 위치 | 책임 |
| --- | --- |
| `cli` | `mwosa run aggregate <name>`, `history`, `inspect` 같은 명령과 flag parsing |
| `service/aggregate` | YAML 검증, param 적용, pipeline stage 실행, materialize, 이력 저장 |
| provider router | canonical provider API stage 실행, 인증, pagination, rate limit, retry |
| provider raw adapter | provider-native raw API stage 실행, raw endpoint 별 인증, rate limit, retry |
| jq engine | stage 입력 row 를 jq query 로 변환하고 실패를 명시적으로 반환 |
| `storage/mongodb` | MongoDB runtime, `aggregate_stage_items` validator/index, TTL 정책 |
| aggregate repository | aggregate/version/run/item 저장과 조회 |
| presentation | `table`, `json`, `ndjson`, `csv` 출력 |

service 는 provider client module 타입을 직접 알지 않는다. provider API stage 는
provider router 의 role interface 를 쓰고, provider raw stage 는 provider raw
adapter 표면을 쓴다. MongoDB client lifecycle 은 repository 가 아니라
`storage/mongodb` runtime 이 관리한다.

Aggregate 는 stage fan-out 크기, timeout, run workspace 같은 실행 범위는
검증할 수 있다. 하지만 provider 별 초당 호출 수, backoff, retry 횟수는
provider adapter 의 정책이다. 호출자는 YAML 에서 API 를 쓸지 local 데이터를
쓸지만 명확히 고른다.

## 리소스 모델

초기 저장 모델은 기존 `strategy/screen_run` 계열을 확장하지 않고 분리한다.
`screen strategy` 는 후보 선별과 jq/yaml pipeline 실행 이력이고,
`Aggregate` 는 MongoDB aggregation 을 중심에 둔 범용 데이터 처리 실행이기
때문이다.

| collection | 의미 |
| --- | --- |
| `aggregates` | aggregate 이름, active version, 생성/수정 시각, archive 상태 |
| `aggregate_versions` | YAML 원문, canonical spec JSON, spec hash, version, note |
| `aggregate_runs` | 실행 시각, params, stage 요약, workspace, spec hash, 상태, 출력 요약 |
| `aggregate_run_items` | 결과 row. 큰 결과를 위해 run 과 분리한다. |

공통 document field 는 MongoDB storage 기준을 따른다. `_id`,
`schema_version`, `revision`, `created_at`, `updated_at` 을 최상위에 두고,
시간은 CLI/JSON 표면에서 ISO8601 UTC 문자열로 직렬화한다.

`aggregate_runs` 는 최소한 아래 값을 저장한다.

- 실행한 aggregate 이름, version, spec hash
- runtime params 와 기본값 적용 결과
- stage 별 type, input ref, provider/group/operation, local collection, dataset,
  snapshot ref, jq hash, MongoDB pipeline hash
- stage row 수, materialized collection 이름, 오류 또는 누락 데이터
- 출력 stage, 출력 포맷
- 결과 row 수, result hash, preview
- 상태: `succeeded`, `failed`

실패는 빈 성공으로 저장하지 않는다. provider 인증 실패, provider 가 반환한
rate limit 초과, API 실패, 필수 stage 입력 누락, MongoDB aggregation 실패,
jq 실패는 `failed` 실행으로 기록하고 error message 와 stage context 를 남긴다.

## YAML 구조

저장 파일은 `kind: Aggregate` 로 시작한다.

```yaml
kind: Aggregate
schema_version: 1
name: impressive-krx-candidates
description: KOSPI/KOSDAQ 후보 테이블

params:
  as_of:
    type: date
    default: "2026-07-01"
  market:
    type: string
    default: krx
  limit:
    type: int
    default: 50

pipeline: []

output:
  from: stage_name
  default_format: table
  columns: []
```

`mwosa update aggregate <name> --file <yaml>` 은 YAML 을 canonical JSON 으로
정규화하고 `spec_hash` 를 계산해 새 version 으로 저장한다.

`mwosa run aggregate <name>` 은 active version 을 실행한다.

```bash
mwosa run aggregate impressive-krx-candidates \
  --param as_of=2026-07-01 \
  --param limit=30 \
  -o table
```

`--param key=value` 는 YAML 의 `params.<key>.default` 를 실행 시점에 덮어쓴다.
정의되지 않은 param, 타입 변환 실패, 사용되지 않은 필수 param 은 error 로
처리한다.

param 치환 문법은 `${params.key}` 로 둔다. 치환은 provider request, local query,
MongoDB aggregation literal, jq query, output 설정에서만 허용한다. `${params.key}` 가
남아 있으면 실행하지 않는다.

fan-out 처럼 앞 stage 의 row 를 기준으로 반복 호출할 때는
`${each.key}` 를 사용한다. 반복 대상은 `foreach.stage` 로 지정하고, `each` 값은
해당 stage 의 `foreach.as` 이름에서 온다.
`params` 와 `each` 에 없는 값이 남아 있으면 실행 전에 error 로 처리한다.

## 명령어 트리

Aggregate 명령은 기존 CLI 의 최상위 동사 구조를 따른다. 새 root command 를
만들지 않고 `run`, `update`, `list`, `inspect`, `history`, `validate`,
`delete` 아래에 붙인다.

MVP 명령어 트리는 다음과 같다.

```text
mwosa update aggregate <name> --file <yaml>
mwosa validate aggregate <yaml> [--view summary|raw]
mwosa list aggregates
mwosa inspect aggregate <name> [--version <n>|--spec-hash <hash>] [--view summary|versions|raw]
mwosa inspect aggregate-plan <name|yaml> [--param key=value] [--view summary|stages|pipeline|raw]
mwosa run aggregate <name> [--version <n>|--spec-hash <hash>] [--alias <alias>] [--param key=value] [-o table|json|ndjson|csv]
mwosa history aggregate [--name <name>] [--status succeeded|failed] [--limit <n>]
mwosa inspect aggregate-run <id|alias> [--view summary|stages|params|pipeline|items|raw] [--limit <n>]
mwosa delete aggregate <name>
```

각 명령의 책임은 아래처럼 둔다.

| 명령 | 책임 |
| --- | --- |
| `update aggregate` | `kind: Aggregate` YAML 을 저장하거나 새 version 으로 갱신한다. |
| `validate aggregate` | YAML 구조, param 치환, pipeline stage type, MongoDB stage 허용 여부를 검증한다. stage 는 실행하지 않는다. |
| `list aggregates` | 저장된 Aggregate 목록과 active version, spec hash, 최근 실행 상태를 보여준다. |
| `inspect aggregate` | 저장된 Aggregate 정의, version, spec hash, stage 요약을 조회한다. |
| `inspect aggregate-plan` | runtime param 적용 뒤 stage 계획, 중간 결과 alias, pipeline 해석 결과를 보여준다. stage 는 실행하지 않는다. |
| `run aggregate` | 저장된 Aggregate 를 실행하고 run, stage provenance, result row 를 저장한 뒤 결과를 출력한다. |
| `history aggregate` | 저장된 Aggregate 실행 이력을 조회한다. |
| `inspect aggregate-run` | 특정 실행의 params, stage, 오류, pipeline, result row 를 다시 조회한다. |
| `delete aggregate` | 저장된 Aggregate 를 soft delete/archive 한다. 과거 run 은 보존한다. |

`create aggregate` 는 MVP 에서 별도 명령으로 두지 않는다. YAML 기반 저장
리소스는 `update aggregate` 가 create/update 를 모두 처리하는 방향이 기존
`update screen strategy`, `update backtest strategy` 와 잘 맞다.

## 실행 단계

`mwosa run aggregate <name>` 의 실행 단계는 아래 순서로 둔다.

1. aggregate 이름, version, spec hash 조건으로 저장된 spec 을 읽는다.
2. YAML 기본 param 과 `--param key=value` 를 합치고 타입을 검증한다.
3. pipeline stage 정의를 검증한다. API stage 와 local stage 는 여기서 이미 결정된다.
4. stage 를 순서대로 실행하고 각 결과를 `aggregate_stage_items`에 `run_id`와 stage alias로 materialize 한다.
5. `aggregate` stage 는 stage alias 를 같은 collection의 `run_id`·stage 조건으로 해석해 MongoDB aggregation 을 실행한다.
6. `jq` stage 는 앞 stage 의 row 를 jq 입력으로 받아 새 row 결과를 만든다.
7. `output.from` stage 의 row 를 `aggregate_run_items` 에 저장하고, run 요약과 provenance 를 `aggregate_runs` 에 저장한다.
8. 요청한 출력 포맷에 맞춰 presentation layer 로 결과를 넘긴다.
9. `aggregate_stage_items`의 임시 document는 TTL 정책에 따라 정리한다.

어느 단계든 provider 인증 실패, provider 가 반환한 rate limit 초과, API 실패,
local collection 누락, aggregation 실패, jq 실패가 발생하면 실행을 실패로
기록하고 error 를 반환한다. 조용한 fallback 은 없다.

## Pipeline stage 종류

pipeline 의 각 stage 는 반드시 `type` 으로 실행 경로를 고른다. API 와 local 저장
데이터 사이에 애매한 fallback 은 없다. `aggregate` 와 `jq` 는 pipeline 안에서
같은 지위의 stage 다.

| type | 의미 | 기본 실행 경로 |
| --- | --- | --- |
| `provider` | provider role interface 를 통한 canonical API 호출 | API 호출 |
| `provider_raw` | provider-native raw API 호출 | API 호출 |
| `local_collection` | 사용자가 지정한 MongoDB collection 조회 | local MongoDB |
| `local_dataset` | mwosa canonical dataset 조회 | local MongoDB |
| `snapshot` | 저장된 provider raw snapshot 조회 | local MongoDB |
| `aggregate_run` | 이전 Aggregate 실행 결과 조회 | local MongoDB |
| `aggregate` | 앞 stage 결과와 MongoDB aggregation pipeline 을 결합 | MongoDB |
| `jq` | 앞 stage 결과를 jq query 로 변환 | jq engine |

`provider_raw` 는 이름 그대로 live API 호출이다. 저장된 raw payload 를 쓰려면
`type: snapshot` 을 명시한다. 예를 들어 `get provider-raw` 계열은 local snapshot
조회이고, `fetch/sync provider-raw` 계열은 live 호출 또는 호출 후 저장에 가깝다.
Aggregate YAML 에서는 이 차이를 `type` 으로 드러낸다.

provider stage 에 `rate_limit` 이나 `retry` 필드를 두지 않는다. KIS 처럼 현재
구현된 provider 는 API 와 raw 호출 모두 provider 내부 limiter/retry 를 사용한다.
Aggregate 는 provider 가 반환한 rate limit 초과나 retry exhausted 오류를 stage
context 와 함께 기록한다.

`jq` stage 는 어느 위치에나 둘 수 있다. raw provider 응답을 MongoDB aggregation
전에 평평한 row 로 바꿀 수도 있고, MongoDB aggregation 으로 만든 결과를 다시
정리할 수도 있다. 단, jq stage 도 입력 stage 와 출력 stage 를 가진다. 숨은 전역
상태나 shell pipeline 으로 동작하지 않는다.

예:

```yaml
pipeline:
  - name: quote
    type: provider
    provider: kis
    role: quote
    request:
      market: krx
      security_type: stock
      symbol: "005930"

  - name: raw_daily
    type: provider_raw
    provider: kis
    operation: inquire-daily-itemchartprice
    params:
      FID_INPUT_ISCD: "005930"
      FID_INPUT_DATE_1: "${params.from}"
      FID_INPUT_DATE_2: "${params.as_of}"

  - name: normalized_daily
    type: jq
    from: raw_daily
    query: |
      map({
        symbol: .context.symbol,
        rows: (.payload.output2 // .payload.output // [])
      })

  - name: stored_raw_daily
    type: snapshot
    provider: kis
    operation: inquire-daily-itemchartprice
    base_date: "${params.as_of}"

  - name: market_cap
    type: local_collection
    collection: valuation_snapshots
    filter:
      as_of_date: "${params.as_of}"

  - name: candidates
    type: aggregate
    from: quote
    pipeline:
      - $lookup:
          from: normalized_daily
          localField: symbol
          foreignField: symbol
          as: daily
```

시가총액 stage 도 반드시 사용자가 고른다. KIS API 를 쓰면 `provider` 또는
`provider_raw`, 로컬 정본을 쓰면 `local_collection` 또는 `local_dataset`,
다른 Aggregate 결과를 쓰면 `aggregate_run` 으로 선언한다.

## Stage materialize

각 stage 결과는 다음 stage가 읽을 수 있는 이름 있는 결과로 materialize 한다.
MongoDB 실행 작업 공간은 고정된 `aggregate_stage_items` collection 하나를 쓴다.

기본 규칙:

- stage 이름은 pipeline 안에서 결과 alias 로 쓴다.
- runtime 은 stage alias 를 `run_id`와 `_aggregate_stage` 조건으로 바꾼다.
- 각 stage document 는 provenance 와 원본 row 또는 변환 row 를 함께 가진다.
- aggregation 과 jq 에서는 payload field 를 최대한 펼친 형태로 읽을 수 있게 한다.
- 임시 document 는 `expires_at` 필드를 가지며 TTL index 로 정리한다.
- 실행마다 새로운 physical collection을 만들지 않는다.

예:

```text
aggregate_stage_items
  {_aggregate_run_id: <run_id>, _aggregate_stage: movers, ...}
  {_aggregate_run_id: <run_id>, _aggregate_stage: technical_windows, ...}
  {_aggregate_run_id: <run_id>, _aggregate_stage: market_cap, ...}
```

YAML 안에서는 실제 collection 이름을 쓰지 않고 stage alias 를 쓴다.

```yaml
pipeline:
  - name: enriched_movers
    type: aggregate
    from: movers
    pipeline:
      - $lookup:
          from: technical_windows
          localField: symbol
          foreignField: symbol
          as: technical
```

runtime은 `movers`, `technical_windows`를 같은 `aggregate_stage_items` collection의
stage 조건으로 해석하고 현재 `run_id` 조건을 자동으로 추가한다. 사용자는 MongoDB
pipeline을 거의 원문 그대로 쓰면서도 다른 실행과 stage의 document가 섞이지 않는다.

임시 document TTL 기본값은 24시간으로 둔다. YAML 에서 필요하면 아래처럼
조정할 수 있지만, 영구 재사용은 stage item이 아니라 `aggregate_run_items`
또는 명시적인 export 대상으로 한다.

```yaml
workspace:
  ttl: 24h
  timeout: 5m
  max_rows: 100000
  max_fanout: 5000
```

기본 실행 상한은 전체 실행 5분, stage당 100,000행, fan-out 5,000건이다.
`workspace` 값은 Aggregate 실행 범위만 제한하며 provider별 호출 속도와 retry
정책은 계속 provider adapter가 관리한다.

기존 버전이 만든 `aggregate_tmp_*` collection은 `mwosa init storage`에서
정리한다. 새 runtime은 이 이름의 collection을 만들지 않는다.

## MongoDB aggregation 정책

`Aggregate` 는 MongoDB 전용 기능이다. SQLite 호환성이나 SQLite fallback 은
요구사항이 아니다.

aggregation pipeline 은 MongoDB pipeline 문법을 가능한 한 그대로 사용한다.
`mwosa` 자체 DSL 을 크게 만드는 것이 목표가 아니다.

기본 정책:

- `$lookup`, `$group`, `$project`, `$addFields`, `$set`, `$unwind`, `$sort`,
  `$limit`, `$setWindowFields` 를 지원 대상으로 둔다.
- side effect 를 만드는 `$out`, `$merge` 는 Aggregate runtime 이 맡는 저장 모델과
  충돌하므로 기본 차단한다.
- 서버 코드 실행이나 운영 명령에 가까운 stage 는 별도 검토 전에는 차단한다.
- aggregation 실패는 input stage, stage index, MongoDB error 를 포함해 실행 실패로
  저장한다.

`$lookup.from`은 stage alias 또는 허용된 local collection 이름이어야 한다.
stage alias는 `aggregate_stage_items`와 현재 `run_id`·stage 조건으로 해석한다.
허용되지 않은 collection 참조는 실행 전에 error로 처리한다.

## 출력 모델

출력은 `output.from` 으로 지정한 stage 의 row 를 저장한 뒤 presentation layer 에
넘긴다. `output` 은 화면과 파일 포맷을 정하는 영역이지, pipeline stage 를 대신
실행하는 영역이 아니다.

지원 포맷:

- `table`
- `json`
- `ndjson`
- `csv`

YAML 은 출력 컬럼, 컬럼명, 정렬, 숫자 포맷을 정의할 수 있다.

```yaml
output:
  from: ranked_candidates
  default_format: table
  sort:
    - field: change_pct
      order: desc
  columns:
    - key: ordinal
      title: "#"
      format: integer
      align: right
    - key: symbol
      title: 코드
    - key: name
      title: 종목
    - key: change_pct
      title: 등락%
      format: percent
      precision: 2
    - key: market_cap_trillion
      title: 시총(조)
      format: number
      precision: 2
```

정렬은 presentation 단계 정렬이 필요한 경우에만 쓴다. 계산 의미가 있는 정렬은
`aggregate` stage 의 `$sort` 또는 `jq` stage 안에 둔다.

## 실행 기록 조회와 재사용

저장된 실행 결과는 나중에 `history/inspect` 계열 명령으로 조회한다.

조회 명령은 [명령어 트리](#명령어-트리) 의 `history aggregate` 와
`inspect aggregate-run` 을 기준으로 한다.

Aggregate 결과는 다른 Aggregate, screen, backtest, report 의 입력으로 재사용할
수 있어야 한다.

재사용 경로:

- `type: aggregate_run` stage 로 특정 run 의 결과를 읽는다.
- `local_dataset` stage 로 stable dataset 승격 결과를 읽는다.
- `screen` 과 `backtest` 는 필요하면 aggregate run 결과를 universe/source 로
  참조한다.
- report 는 `aggregate_runs` 요약과 `aggregate_run_items` row 를 읽어 표와
  설명을 만든다.

## 1순위 적용 사례

가장 먼저 만들 사용 사례는 다음이다.

> 오늘 인상적인 KOSPI/KOSDAQ 주식 후보를 KIS API 호출 결과와 추가 provider API 호출 결과를 합성해 한눈에 보는 테이블

이 기능은 투자 추천이 아니라 사용자가 정의한 데이터 처리와 관찰용 테이블 출력을
자동화하는 기능이다. 매수, 매도, 보유 같은 판단은 내리지 않는다.

목표 출력 컬럼:

| 컬럼 | 의미 |
| --- | --- |
| `#` | 출력 순번 |
| `코드` | 종목 코드 |
| `종목` | 종목명 |
| `등락%` | 사용자가 고른 stage 기준 등락률 |
| `시총(조)` | 사용자가 명시한 시가총액 stage |
| `거래대금(억)` | 거래대금 |
| `거래량x20` | 20일 평균 대비 거래량 |
| `52주고점%` | 52주 고점 대비 위치 |
| `종가위치%` | 당일 또는 기간 내 종가 위치 |
| `RSI` | YAML stage/API 호출 결과와 pipeline 계산 결과 |
| `ADX` | YAML stage/API 호출 결과와 pipeline 계산 결과 |
| `ATR%` | YAML stage/API 호출 결과와 pipeline 계산 결과 |
| `추세` | 사용자가 pipeline 에서 만든 custom field |
| `메모/라벨` | 사용자가 pipeline 에서 만든 custom field |

RSI, ADX, ATR 은 사전 수집된 고정 정보가 아니다. 사용자가 YAML 에 선언한
stage/API 호출 결과를 materialize 하고, pipeline 에서 합성한 결과로 본다.
`추세` 와 `메모/라벨` 도 `mwosa` 코드에 박힌 판단 로직이 아니라 사용자가
pipeline 안에서 만든 필드다.

아래 YAML 에서 `instruments`, `valuation_snapshots`,
`inquire-daily-itemchartprice` 는 현재 코드에 있는 이름이다. RSI/ADX/ATR 계산
stage 는 길어지므로 필드가 만들어지는 위치만 축약해서 보여준다. 실제 구현
문서나 fixture 에서는 `normalized_daily` 결과를 `$unwind`, `$setWindowFields`,
`$group`, `$addFields` 로 풀어 `rsi_14`, `adx_14`, `atr_pct_14` 를 만든다.

예시 YAML:

```yaml
kind: Aggregate
schema_version: 1
name: impressive-krx-candidates
description: KIS 호출과 로컬 데이터를 합성한 KRX 후보 관찰 테이블

params:
  as_of:
    type: date
    default: "2026-07-01"
  from:
    type: date
    default: "2025-07-01"
  limit:
    type: int
    default: 30

workspace:
  ttl: 24h
  timeout: 5m
  max_rows: 100000
  max_fanout: 5000

pipeline:
  - name: universe
    type: local_collection
    collection: instruments
    filter:
      market_key: krx
      security_type: stock
      extensions.marketTypeName:
        $in: [KOSPI, KOSDAQ]

  - name: quotes
    type: provider
    provider: kis
    role: quote
    foreach:
      stage: universe
      field: symbol
      as: symbol
    request:
      market: krx
      security_type: stock
      symbol: "${each.symbol}"

  - name: daily_windows
    type: provider_raw
    provider: kis
    operation: inquire-daily-itemchartprice
    foreach:
      stage: universe
      field: symbol
      as: symbol
    params:
      FID_INPUT_ISCD: "${each.symbol}"
      FID_INPUT_DATE_1: "${params.from}"
      FID_INPUT_DATE_2: "${params.as_of}"
      FID_PERIOD_DIV_CODE: D

  - name: normalized_daily
    type: jq
    from: daily_windows
    query: |
      map({
        symbol: .context.symbol,
        rows: (.payload.output2 // .payload.output // [])
      })

  - name: market_cap
    type: local_collection
    collection: valuation_snapshots
    filter:
      as_of_date: "${params.as_of}"

  - name: scored_candidates
    type: aggregate
    from: quotes
    pipeline:
      - $lookup:
          from: universe
          localField: symbol
          foreignField: symbol
          as: instrument
      - $unwind: "$instrument"
      - $lookup:
          from: normalized_daily
          localField: symbol
          foreignField: symbol
          as: daily
      - $lookup:
          from: market_cap
          localField: symbol
          foreignField: symbol
          as: valuation
      # 실제 stage 는 normalized_daily 의 OHLCV 배열을 풀어
      # rsi_14, adx_14, atr_pct_14 를 만든다.
      - $addFields:
          name: "$instrument.name"
          market_cap_minor:
            $ifNull: [{ $first: "$valuation.market_cap_minor" }, 0]
          market_cap_trillion:
            $divide:
              - { $ifNull: [{ $first: "$valuation.market_cap_minor" }, 0] }
              - 10000000000000000
          turnover_100m:
            $divide: ["$traded_amount", 100000000]
          relative_volume_20d:
            $divide: ["$volume", "$volume_avg_20d"]
          high_52w_pct:
            $multiply:
              - { $subtract: [{ $divide: ["$close", "$high_52w"] }, 1] }
              - 100
          close_position_pct:
            $multiply:
              - { $divide: [{ $subtract: ["$close", "$low"] }, { $subtract: ["$high", "$low"] }] }
              - 100
      - $addFields:
          trend:
            $switch:
              branches:
                - case: { $and: [{ $gte: ["$rsi_14", 55] }, { $gte: ["$adx_14", 20] }] }
                  then: momentum
                - case: { $lte: ["$rsi_14", 35] }
                  then: oversold_watch
              default: neutral
          label:
            $concat:
              - "vol="
              - { $toString: { $round: ["$relative_volume_20d", 1] } }
              - ", cap="
              - { $toString: { $round: ["$market_cap_trillion", 1] } }
      - $sort:
          relative_volume_20d: -1
          turnover_100m: -1
      - $limit: "${params.limit}"

  - name: ranked_candidates
    type: jq
    from: scored_candidates
    query: |
      map({
        symbol,
        name,
        change_pct,
        market_cap_trillion,
        turnover_100m,
        relative_volume_20d,
        high_52w_pct,
        close_position_pct,
        rsi_14,
        adx_14,
        atr_pct_14,
        trend,
        label
      })

output:
  from: ranked_candidates
  default_format: table
  columns:
    - key: ordinal
      title: "#"
      format: integer
    - key: symbol
      title: 코드
    - key: name
      title: 종목
    - key: change_pct
      title: 등락%
      format: percent
      precision: 2
    - key: market_cap_trillion
      title: 시총(조)
      format: number
      precision: 2
    - key: turnover_100m
      title: 거래대금(억)
      format: number
      precision: 0
    - key: relative_volume_20d
      title: 거래량x20
      format: number
      precision: 1
    - key: high_52w_pct
      title: 52주고점%
      format: percent
      precision: 1
    - key: close_position_pct
      title: 종가위치%
      format: percent
      precision: 1
    - key: rsi_14
      title: RSI
      format: number
      precision: 1
    - key: adx_14
      title: ADX
      format: number
      precision: 1
    - key: atr_pct_14
      title: ATR%
      format: percent
      precision: 1
    - key: trend
      title: 추세
    - key: label
      title: 메모/라벨
```

예시 출력:

| # | 코드 | 종목 | 등락% | 시총(조) | 거래대금(억) | 거래량x20 | 52주고점% | 종가위치% | RSI | ADX | ATR% | 추세 | 메모/라벨 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 005930 | 삼성전자 | 3.42 | 415.20 | 9800 | 2.4 | -4.8 | 86.2 | 61.3 | 24.1 | 2.2 | momentum | vol=2.4, cap=415.2 |
| 2 | 035720 | 카카오 | 2.10 | 21.30 | 1420 | 1.9 | -18.5 | 74.0 | 57.8 | 21.0 | 3.1 | momentum | vol=1.9, cap=21.3 |

## MVP 기준

- `kind: Aggregate` 의 리소스 모델, YAML 구조, 실행 단계, stage 종류,
  MongoDB aggregation 사용 방식, 저장/출력 모델을 설명한다.
- KIS provider raw/API stage, 로컬 MongoDB stage, jq stage, 단일 stage item collection,
  aggregation pipeline, output columns 의 관계를 예시 YAML 로 보여준다.
- 저장, 검증, 계획 확인, 실행, 이력 조회, run inspect 를 위한 명령어 트리를 둔다.
- API stage 와 local stage 의 명시적 구분을 둔다.
- MongoDB 전용 기능이며 SQLite fallback 이 없다는 정책을 명시한다.
- RSI, ADX, ATR, 추세, 라벨은 고정 내장 판단이 아니라 사용자 pipeline 결과라고
  설명한다.
- `aggregate_stage_items` document TTL 정책을 둔다.
- 1순위 후보 테이블 사용 사례를 요구사항과 예시 출력 수준으로 포함한다.

## 비요구사항

- Python 스크립트를 `mwosa` 저장소 안에 개인 스크립트로 보관하지 않는다.
- MongoDB aggregation 보다 큰 자체 DSL 을 새로 만드는 것이 목표가 아니다.
- 첫 문서에서 전체 구현을 완료하지 않는다.
- 첫 버전에서 모든 provider API 를 지원한다고 약속하지 않는다.
- 첫 버전에서 backtest 전체를 대체한다고 쓰지 않는다.
- `mwosa` 가 매수/매도 판단이나 투자 추천을 자동으로 내리는 기능으로 설명하지
  않는다.

## 열린 질문

- 저장 모델은 이 문서에서 분리 collection 을 기본안으로 잡았다. 다만
  `aggregate_versions` 를 항상 분리할지, 작은 version 목록은 `aggregates.versions[]`
  에 embed 할지 구현 전 한 번 더 결정한다.
- pipeline stage 문법은 `type` 기반 구분을 기본안으로 잡았다. provider role
  stage 의 세부 role 이름과 request shape 는 현재 provider router 계약에 맞춰
  좁게 시작한다.
- MongoDB aggregation stage 는 read-only pass-through 와 deny-list 조합을 기본안으로
  잡았다. 운영상 차단할 stage 목록은 storage/runtime 구현 때 테스트로 고정한다.
- runtime params 치환은 `${params.key}` 를 기본안으로 잡았다. 날짜, 숫자, boolean 의
  타입 보존 규칙은 YAML loader 와 canonical spec 변환에서 확정한다.
- output table 은 기존 presentation contract 에 맞춘다. 다만 column format 이름과
  기존 CLI renderer 의 정확한 매핑은 구현 단계에서 맞춘다.
