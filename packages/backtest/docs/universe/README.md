# Universe Pipeline

## 목적

백테스트의 `universe` 는 "어떤 종목을 볼 것인가"를 정의한다. 직접 종목을
지정하는 `symbols: ["069500"]` 방식은 가능한 표현 중 하나일 뿐이고, 핵심 모델은
여러 selector 함수를 합성해 실행 대상군을 만드는 pipeline 이다.

이 문서는 사용자 관점의 YAML 형태와 구현해야 할 selector 목록을 제약 없이
정리한다. 특정 구현 순서나 최소 스펙을 제안하지 않는다. 어떤 selector 가
추가되더라도 같은 문법과 같은 실행 모델 안에 들어오게 하는 것이 목표다.

## 핵심 관점

`Strategy` 는 "선택된 종목 안에서 언제 사고팔 것인가"를 맡는다.

`BacktestRun` 의 `universe` 는 "이번 실행에서 어떤 종목을 후보로 볼 것인가"를
맡는다.

```text
BacktestRun
  -> data range
  -> universe pipeline
  -> selected symbols by date
  -> StrategyPlan
  -> Engine
```

따라서 universe 조건은 `StrategySpec` 이 아니라 `BacktestRunSpec` 에 둔다. 같은
전략을 고정 종목, ETF 전체, 거래대금 상위, 저장된 스크리닝 결과, 리밸런싱 후보군
등 여러 universe 에 재사용할 수 있어야 한다.

## 기본 YAML 형태

`universe.pipeline` 은 위에서 아래로 실행되는 함수 합성이다. 각 step 은 `id` 로
selector 를 고르고, `params` 로 인자를 넘긴다.

```yaml
universe:
  pipeline:
    - id: source.symbols
      params:
        symbols: ["069500", "360750"]

    - id: filter.has_daily_bars

    - id: rank.by_field
      params:
        field: traded_amount
        order: desc
        limit: 10
```

`universe.symbols` 는 계속 허용한다. 다만 canonical model 에서는 아래 pipeline 의
축약형으로 정규화한다.

```yaml
universe:
  symbols: ["069500"]
```

```yaml
universe:
  pipeline:
    - id: source.symbols
      params:
        symbols: ["069500"]
```

## Selector Step 개념

selector step 은 하나의 함수 호출이다. `id` 는 호출할 selector 를 고르고,
`params` 는 그 selector 에 넘길 인자다. `name` 은 사람이 읽는 설명이나 explain
출력에서 사용할 수 있는 선택 필드다.

```yaml
- id: rank.by_field
  name: liquidity-leaders
  params:
    field: traded_amount
    order: desc
    limit: 20
```

selector id 는 JSON schema enum 처럼 고정하지 않는다. `indicator.id`,
`metric.id` 와 같은 방향으로 registry 에 등록된 selector metadata 로 검증한다.

```text
YAML shape validation
  -> typed BacktestRunSpec
  -> universe selector registry validation
  -> compiled UniversePipeline
  -> service resolves storage/provider-backed inputs
  -> engine receives universe snapshots and execution data
```

## 실행 결과 모델

Universe pipeline 은 단순히 symbol 목록만 반환하지 않는다. 백테스트 결과를
설명할 수 있어야 하므로 선택과 제외의 근거를 함께 남긴다.

| 개념 | 의미 |
| --- | --- |
| selection time | 이 universe 가 결정된 시점 |
| selected symbols | 해당 시점부터 관찰할 종목 목록 |
| included decisions | 어떤 step 이 어떤 값을 근거로 종목을 포함했는지 |
| excluded decisions | 어떤 step 이 어떤 값을 근거로 종목을 제외했는지 |
| step summaries | 각 step 의 입력 개수, 출력 개수, score, reason |

universe 는 실행 전체에 대해 고정될 수도 있고, 날짜별로 바뀔 수도 있다. 모델은
둘 다 같은 개념 안에서 표현할 수 있어야 한다.

```text
static universe:
  one selection for the whole run

scheduled universe:
  daily / weekly / monthly selection snapshots
```

## 레이어 경계

`packages/backtest` 는 selector pipeline 의 순수 contract 와 universe snapshot 을
알 수 있다. 하지만 YAML 파일 경로, Cobra, SQLite, 저장된 screen, provider 호출은
알면 안 된다.

| 위치 | 책임 |
| --- | --- |
| `service/backtest` | YAML decode, selector registry 검증, storage/provider-backed selector 실행, selected symbols 로딩 |
| `packages/backtest` | 순수 selector contract, engine 이 사용할 universe snapshot |
| `storage/*` | canonical bars, saved screen result, instrument metadata 조회 |
| `app/handler` / `cli` | 입력 파일과 출력 포맷 처리 |

저장소나 provider 가 필요한 selector 는 service layer 에 둔다. 예를 들어
`source.saved_screen`, `source.daily_bars`, `filter.has_daily_bars` 는 storage 를
읽어야 하므로 `packages/backtest` 안에서 직접 실행하지 않는다.

## 함수 합성 규칙

pipeline 은 아래 규칙을 따른다.

- 각 step 은 입력 universe 를 받아 출력 universe 를 만든다.
- `source.*` step 은 입력 없이 universe 를 시작할 수 있다.
- `filter.*` step 은 입력 universe 에서 일부를 제거한다.
- `rank.*` step 은 score 를 계산하고 순서를 정한다.
- `limit.*` step 은 개수를 줄인다.
- `combine.*` step 은 여러 sub-pipeline 을 합친다.
- 모든 step 은 선택/제외 이유를 남길 수 있어야 한다.
- 같은 입력과 같은 데이터에서는 같은 결과가 나와야 한다.

## 사용자 YAML 예시

### 1. 직접 종목 지정

가장 작은 형태다. 기존 실행과 호환된다.

```yaml
universe:
  symbols: ["069500", "360750"]
```

명시적 pipeline 으로 쓰면 아래와 같다.

```yaml
universe:
  pipeline:
    - id: source.symbols
      params:
        symbols: ["069500", "360750"]
```

### 2. ETF 전체에서 유동성 상위만 선택

```yaml
universe:
  pipeline:
    - id: source.daily_bars
      params:
        market: krx

    - id: filter.security_type
      params:
        value: etf

    - id: transform.latest_per_symbol

    - id: filter.field
      params:
        field: traded_amount
        op: gte
        value: 1000000000

    - id: rank.by_field
      params:
        field: traded_amount
        order: desc
        limit: 20
```

### 3. 저장된 스크리닝 결과를 백테스트 universe 로 사용

```yaml
universe:
  pipeline:
    - id: source.saved_screen
      params:
        name: etf-weekly-leaders
        run: latest

    - id: rank.by_field
      params:
        field: score
        order: desc
        limit: 10
```

### 4. 모멘텀과 유동성 조건을 합성

```yaml
universe:
  pipeline:
    - id: source.daily_bars
      params:
        market: krx
        lookback_days: 120

    - id: filter.security_type
      params:
        value: etf

    - id: transform.window_metrics
      params:
        metrics:
          return_20d:
            id: return
            params:
              window: 20
          traded_amount_avg_20d:
            id: average
            params:
              field: traded_amount
              window: 20

    - id: filter.expr
      params:
        all:
          - gte:
              - field: return_20d
              - value: 0.03
          - gte:
              - field: traded_amount_avg_20d
              - value: 500000000

    - id: rank.by_field
      params:
        field: return_20d
        order: desc
        limit: 5
```

### 5. 여러 후보군을 합치고 중복 제거

```yaml
universe:
  pipeline:
    - id: combine.union
      params:
        pipelines:
          - name: liquidity
            pipeline:
              - id: source.daily_bars
                params:
                  market: krx
              - id: filter.security_type
                params:
                  value: etf
              - id: transform.latest_per_symbol
              - id: rank.by_field
                params:
                  field: traded_amount
                  order: desc
                  limit: 20

          - name: momentum
            pipeline:
              - id: source.saved_screen
                params:
                  name: etf-momentum-leaders
                  run: latest
              - id: limit.count
                params:
                  count: 20

    - id: transform.distinct
      params:
        by: symbol
```

### 6. 교집합으로 더 엄격한 후보군 만들기

```yaml
universe:
  pipeline:
    - id: combine.intersect
      params:
        pipelines:
          - name: liquid
            pipeline:
              - id: source.saved_screen
                params:
                  name: latest-liquidity-leaders
                  run: latest

          - name: low_vol_uptrend
            pipeline:
              - id: source.saved_screen
                params:
                  name: low-vol-uptrend
                  run: latest
```

### 7. 월별 리밸런싱 universe

```yaml
universe:
  schedule:
    frequency: monthly
    anchor: first_trading_day
  pipeline:
    - id: source.daily_bars
      params:
        market: krx
        lookback_days: 252

    - id: filter.security_type
      params:
        value: etf

    - id: transform.window_metrics
      params:
        metrics:
          return_3m:
            id: return
            params:
              window: 63
          max_drawdown_6m:
            id: max_drawdown
            params:
              window: 126

    - id: filter.expr
      params:
        all:
          - gte:
              - field: return_3m
              - value: 0
          - gte:
              - field: max_drawdown_6m
              - value: -0.15

    - id: rank.weighted
      params:
        fields:
          return_3m: 0.7
          max_drawdown_6m: 0.3
        order: desc
        limit: 10
```

### 8. 명시적 제외 목록

```yaml
universe:
  pipeline:
    - id: source.saved_screen
      params:
        name: etf-weekly-leaders
        run: latest

    - id: filter.exclude_symbols
      params:
        symbols: ["252670", "251340"]
        reason: leveraged_or_inverse
```

## Selector 카탈로그

### Source selector

| id | 의미 |
| --- | --- |
| `source.symbols` | 사용자가 직접 지정한 종목 목록으로 시작한다. |
| `source.daily_bars` | canonical daily bar 저장소에서 후보 rows 를 읽는다. |
| `source.instrument_master` | 종목 master 또는 instrument metadata 에서 후보를 읽는다. |
| `source.saved_screen` | 저장된 screening run 결과를 후보로 가져온다. |
| `source.screen_strategy` | 저장된 screen strategy 를 실행해 후보를 만든다. |
| `source.watchlist` | 사용자가 저장한 관심 종목 목록을 가져온다. |
| `source.file` | CSV, JSON, NDJSON 같은 외부 파일에서 후보를 읽는다. |
| `source.inline` | YAML 안에 rows 를 직접 적는다. 테스트와 작은 실험용이다. |

### Transform selector

| id | 의미 |
| --- | --- |
| `transform.latest_per_symbol` | 종목별 최신 row 만 남긴다. |
| `transform.window_metrics` | lookback window 기반 보조 필드를 계산한다. |
| `transform.indicator` | indicator registry 를 사용해 보조지표를 계산한다. |
| `transform.join_metadata` | instrument, ETF metadata, extension field 를 붙인다. |
| `transform.normalize_fields` | numeric string, currency, percent 값을 비교 가능한 값으로 정규화한다. |
| `transform.distinct` | 중복 symbol 을 제거한다. |
| `transform.tag` | 후보군에 source tag 나 설명 tag 를 붙인다. |

### Filter selector

| id | 의미 |
| --- | --- |
| `filter.field` | 단일 field 비교 조건으로 거른다. |
| `filter.expr` | `all`, `any`, `not`, 비교식을 조합해 거른다. |
| `filter.has_daily_bars` | 실행 기간에 필요한 일봉이 있는 종목만 남긴다. |
| `filter.listing_age` | 상장 이후 최소 기간을 만족하는 종목만 남긴다. |
| `filter.liquidity` | 거래대금, 거래량, 스프레드 조건을 만족하는 종목만 남긴다. |
| `filter.exclude_symbols` | 명시적 제외 목록을 적용한다. |
| `filter.include_symbols` | 현재 후보군 안에서 명시 목록만 남긴다. |
| `filter.security_type` | candidate field 의 security type 을 제한한다. |
| `filter.market` | KRX, NASDAQ 같은 market 을 제한한다. |
| `filter.tags` | metadata tag 를 기준으로 포함 또는 제외한다. |

### Rank selector

| id | 의미 |
| --- | --- |
| `rank.by_field` | 단일 field 기준으로 정렬하고 상위 N개를 남긴다. |
| `rank.weighted` | 여러 field 를 가중합 score 로 정렬한다. |
| `rank.percentile` | percentile 기준으로 상위 또는 하위 구간을 남긴다. |
| `rank.group_top_n` | category, sector, asset_class 별 상위 N개를 남긴다. |
| `rank.round_robin` | 여러 그룹에서 순환 방식으로 후보를 뽑는다. |

### Limit selector

| id | 의미 |
| --- | --- |
| `limit.count` | 앞에서부터 N개만 남긴다. |
| `limit.per_group` | 그룹별 최대 개수를 제한한다. |
| `limit.min_count` | 후보가 너무 적으면 실패하거나 경고한다. |
| `limit.max_count` | 후보가 너무 많으면 실패하거나 잘라낸다. |

### Combine selector

| id | 의미 |
| --- | --- |
| `combine.union` | 여러 sub-pipeline 결과를 합친다. |
| `combine.intersect` | 여러 sub-pipeline 에 공통으로 등장한 종목만 남긴다. |
| `combine.difference` | 첫 pipeline 결과에서 다른 pipeline 결과를 뺀다. |
| `combine.concat` | 여러 결과를 단순 연결한다. 중복 제거는 별도 step 으로 둔다. |

### Debug selector

| id | 의미 |
| --- | --- |
| `debug.snapshot` | 현재 후보군과 field 값을 결과 explain 에 남긴다. |
| `debug.assert_count` | 후보 수가 예상 범위인지 검증한다. |
| `debug.sample` | 개발 중 일부 후보만 샘플링한다. production run 에서는 비활성화할 수 있다. |

## Expression 모델

`filter.expr` 는 strategy rule 과 비슷한 함수형 표현식을 쓴다. 다만 이 표현식은
매매 신호가 아니라 universe row 를 평가한다.

```yaml
id: filter.expr
params:
  all:
    - gte:
        - field: traded_amount_avg_20d
        - value: 500000000
    - any:
        - gte:
            - field: return_20d
            - value: 0.03
        - gte:
            - field: relative_strength_60d
            - value: 80
```

표현식 함수도 registry 기반으로 검증한다.

| 분류 | 함수 |
| --- | --- |
| 논리 | `all`, `any`, `not` |
| 비교 | `gt`, `gte`, `lt`, `lte`, `eq`, `between`, `in` |
| 값 | `field`, `value`, `param`, `ref` |
| 문자열 | `contains`, `prefix`, `suffix`, `matches` |
| 결측 처리 | `exists`, `missing`, `coalesce` |

## Schedule 모델

schedule 은 universe pipeline 을 언제 다시 평가할지 정한다. static universe,
리밸런싱, walk-forward 실행은 모두 같은 pipeline 에 schedule 만 다르게 붙인
형태로 본다.

```yaml
universe:
  schedule:
    frequency: weekly
    anchor: monday
    lookback_policy: closed_bars_only
  pipeline:
    - id: source.daily_bars
      params:
        lookback_days: 120
```

후보 schedule:

| 값 | 의미 |
| --- | --- |
| `once` | 실행 시작 시 한 번만 선택한다. 기본값이다. |
| `daily` | 매 거래일 universe 를 다시 만든다. |
| `weekly` | 주 1회 universe 를 다시 만든다. |
| `monthly` | 월 1회 universe 를 다시 만든다. |
| `custom_calendar` | 지정한 날짜 목록에서 다시 만든다. |

백테스트에서는 schedule 의 각 평가 시점에서 이미 닫힌 데이터만 사용해야 한다.
예를 들어 `next_open` 체결을 쓴다면, 당일 장 시작 전에 알 수 없는 당일 종가를
universe selection 에 사용하면 안 된다.

## Validation 방향

- validation 은 고정된 step 순서가 아니라 selector registry 의 입출력 계약을 기준으로 한다.
- `universe.symbols` 는 `source.symbols` pipeline 으로 정규화한다.
- pipeline 이 비어 있을 때 identity 로 볼지 실패로 볼지는 실행 모드가 결정한다.
- 등록되지 않은 selector id 는 compile 단계에서 실패한다.
- selector params 는 selector registry 가 검증한다.
- storage/provider-backed selector 는 service layer 에서 capability 를 확인한다.
- 실행 기간의 데이터를 요구하는 selector 는 `BacktestRun.data` 와 모순되면 실패한다.
- benchmark symbol 은 universe 후보에 자동 포함하지 않는다. benchmark 는 비교 기준이고 거래 대상이 아니다.

## Explain 출력

사용자는 왜 어떤 종목이 선택됐는지 확인할 수 있어야 한다. 따라서 backtest 결과는
선택된 symbol 뿐 아니라 universe 설명을 별도 필드로 가질 수 있다.

```json
{
  "universe": {
    "mode": "pipeline",
    "schedule": "once",
    "selected_symbols": ["069500", "360750"],
    "steps": [
      {
        "id": "source.daily_bars",
        "input_count": 0,
        "output_count": 840
      },
      {
        "id": "filter.field",
        "input_count": 840,
        "output_count": 42,
        "reason": "traded_amount >= 1000000000"
      },
      {
        "id": "rank.by_field",
        "input_count": 42,
        "output_count": 10,
        "field": "traded_amount"
      }
    ]
  }
}
```

machine-readable 출력에서는 이 구조를 그대로 JSON 으로 제공하고, table 출력에서는
최종 symbol 수와 주요 step 만 간단히 보여준다.

## 구현 범위

### Spec / compile

- YAML 의 `universe` 표현을 canonical universe pipeline 으로 정규화한다.
- schedule, pipeline, selector step, sub-pipeline 표현을 canonical model 에 담는다.
- `UniverseSelectorRegistry` 를 만든다.
- selector id, params, 입출력 요구사항을 registry metadata 로 검증한다.
- unknown selector id 는 명확한 error 로 처리한다.
- `BacktestRunSpec` compile 결과에 universe plan 과 explain metadata 를 포함한다.

### Service

- `service/backtest` 에서 storage-backed selector 실행기를 조립한다.
- canonical daily bar 기반 `source.daily_bars` 를 구현한다.
- saved screen 기반 `source.saved_screen` 을 구현한다.
- selector 실행 중 필요한 데이터 범위와 실제 backtest data range 를 분리한다.
- universe explain 결과를 service result 에 포함한다.

### Engine

- engine 은 static universe 와 scheduled universe 를 모두 표현할 수 있는
  `UniverseSnapshot` 을 받는다.
- scheduled universe 에서 포지션 보유 종목이 universe 에서 빠졌을 때의 정책을
  정한다. 예: 보유 유지, 청산 신호 생성, 신규 진입만 금지.

### CLI / output

- `mwosa validate backtest` 가 universe pipeline compile 결과를 보여준다.
- JSON 결과에 universe explain 을 포함한다.
- table 결과는 selected symbol count, schedule, step count 를 간단히 보여준다.
- 디버그용으로 universe pipeline 만 실행하는 command 를 검토한다.

```bash
mwosa inspect backtest-universe <yaml> -o json
```

### Tests

- `universe.symbols` 기존 YAML 이 계속 동작한다.
- `source.symbols` pipeline 이 `universe.symbols` 와 같은 결과를 낸다.
- unknown selector id 는 validate 단계에서 실패한다.
- selector 입출력 계약이 맞지 않으면 validate 단계에서 실패한다.
- `filter.field` 로 후보가 줄어든다.
- `rank.by_field` 와 `limit.count` 가 deterministic 하게 동작한다.
- saved screen selector 는 storage boundary 를 통해 실행된다.
- scheduled universe 는 각 selection date 에 closed data 만 사용한다.

## 열린 질문

- universe selector 표현식을 strategy rule 표현식과 같은 AST 로 둘지, 별도 AST 로
  둘지 정해야 한다.
- `source.daily_bars` 는 row 단위 후보를 반환할지, symbol 단위 candidate 를 반환할지
  정해야 한다.
- pipeline 중간 결과를 저장할지, 실행마다 재계산할지 정해야 한다.
- scheduled universe 에서 탈락한 보유 종목을 자동 청산할지, 전략 exit 에만 맡길지
  정해야 한다.
- saved screen 결과의 `score`, `rank`, custom field 를 canonical candidate field 로
  어떻게 정규화할지 정해야 한다.
- ETF metadata, instrument metadata, daily bar extension field 를 하나의 field
  namespace 로 묶을지 정해야 한다.
