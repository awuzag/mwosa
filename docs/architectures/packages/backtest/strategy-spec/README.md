# Strategy Spec

`Strategy` spec 은 "무엇을 보면 사고팔 것인가"를 정의한다.

전략은 실행 환경을 직접 소유하지 않는다. 데이터 기간, 초기 현금, 수수료,
슬리피지, 결과 저장 방식은 실행 스펙의 책임이다. 이렇게 나누면 같은 전략을
다른 기간, 다른 체결 가정, 다른 포트폴리오 크기로 반복 검증할 수 있다.

## 목표

- 전략 신호와 실행 조건을 명확히 분리한다.
- 비교 연산자와 값 표현식을 분리해 rule 종류가 폭발하지 않게 한다.
- YAML 은 실행 언어가 아니라 검증 가능한 선언형 rule tree 로 둔다.
- 하나의 `Strategy` 와 하나의 실행 스펙을 compile 해 `StrategyPlan` 을 만든다.
- 작성 편의성을 위해 Kubernetes manifest 처럼 multi-document YAML 을 허용한다.

## 기본 형태

```yaml
kind: Strategy
schema_version: 1
name: trend-pullback

indicators:
  trend:
    id: sma
    source:
      price: close
    params:
      window: 20
  rsi14:
    id: rsi
    source:
      price: close
    params:
      window: 14

entry:
  all:
    - gt:
        - price: close
        - ref: trend
    - lt:
        - ref: rsi14
        - value: 35

exit:
  any:
    - crosses_below:
        - price: close
        - ref: trend

sizing:
  type: percent_of_equity
  value: 10

risk:
  max_symbol_weight_pct: 20
```

## 맡는 일

`Strategy` 가 맡는 일:

- 재사용할 지표와 값 표현식 alias 를 정의한다.
- 진입 rule tree 를 정의한다.
- 청산 rule tree 를 정의한다.
- 진입 또는 추가 진입 시 포지션 크기 계산 방식을 정의한다.
- 전략 자체의 위험 한도를 정의한다.

`Strategy` 가 맡지 않는 일:

- 과거 데이터 조회 기간
- market, security type, timeframe 같은 실행 데이터 소스
- 초기 현금과 benchmark
- 수수료, 세금, 슬리피지, 체결 가격 가정
- 리포트 포맷과 저장 위치
- CLI flag, 파일 경로, 저장소 조회

## 필수 필드

| 필드 | 의미 |
| --- | --- |
| `kind` | 항상 `Strategy` |
| `schema_version` | strategy schema version |
| `name` | strategy 식별 이름 |
| `entry` | 신규 진입 rule tree |
| `exit` | 청산 rule tree |
| `sizing` | 주문 후보 크기 계산 방식 |

## 선택 필드

| 필드 | 의미 |
| --- | --- |
| `description` | 사람을 위한 설명 |
| `indicators` | rule 에서 재사용할 지표 alias |
| `risk` | strategy-local risk limit |
| `tags` | 검색, 리포트, 실행 기록용 tag |

## Rule tree

rule 은 작은 조건을 `all`, `any`, `not` 으로 조합한다. 비교 조건은
`close_above_sma` 같은 접두사형 이름으로 만들지 않는다. `above`, `below`,
`equal` 은 별도 지표가 아니라 비교 연산자이므로, engine 은 `operator`,
`args` 로 정규화할 수 있어야 한다.

```yaml
entry:
  all:
    - gt:
        - price: close
        - ref: trend
    - any:
        - lt:
            - ref: rsi14
            - value: 35
        - crosses_above:
            - price: close
            - indicator:
                id: sma
                source:
                  price: close
                params:
                  window: 5
```

## 표현식 모델

전략 YAML 은 함수형 조합식에 가깝게 본다.

```text
RuleExpr =
  all([]RuleExpr)
  any([]RuleExpr)
  not(RuleExpr)
  call(operator, []Expr)

Expr =
  price(field)
  value(number)
  ref(alias)
  indicator(id, source Expr, params)
```

지원 함수는 닫힌 목록으로 고정하지 않는다. Go 구현에서는 registry 에 등록된
function metadata 를 기준으로 YAML 을 검증하고 compile 한다.

| 분류 | 함수 |
| --- | --- |
| 논리 | `all`, `any`, `not` |
| 비교 | `gt`, `gte`, `lt`, `lte`, `eq`, `between` |
| 교차 | `crosses_above`, `crosses_below` |
| 값 | `price`, `value`, `ref`, `indicator` |
| 산술 | `add`, `sub`, `mul`, `div`, `abs`, `min`, `max` |
| 변환 | `lag`, `change`, `pct_change`, `rolling` |

이 구조를 쓰면 `close > sma(20)`, `rsi(14) < 35`,
`sma(5) > sma(20)`, `volume > sma(volume, 20)` 같은 조건을 같은 비교
함수로 표현할 수 있다. Go 쪽 compile 결과도 `CloseAboveSMA` 같은 개별 타입보다
`CallExpr{Operator, Args}` 또는 이를 정규화한 `CompareRule{Operator, Args}` 에
가깝게 잡는 편이 확장하기 쉽다.

## Indicator roadmap

indicator 는 스펙 레벨에서 넓게 열어둔다. 특정 라이브러리에 YAML 형태를
종속시키지 않고, engine 내부의 indicator registry 가 `id`, `source`, `params`,
`output` 을 검증한다. 외부 Go indicator 라이브러리를 가져오더라도 전략 에셋은
같은 구조를 유지해야 한다.

이 로드맵은 구현 순서가 아니라 strategy spec 이 수용해야 할 표현 범위다.

| 영역 | 제공 방향 |
| --- | --- |
| 가격/거래량 source | `open`, `high`, `low`, `close`, `adjusted_close`, `volume`, `amount` |
| 이동평균/평활화 | `sma`, `ema`, `wma`, `dema`, `tema`, `kama`, `hma`, `vwma` |
| 모멘텀/오실레이터 | `rsi`, `macd`, `stochastic`, `cci`, `roc`, `momentum`, `williams_r` |
| 추세/강도 | `adx`, `di_plus`, `di_minus`, `aroon`, `trix`, `psar` |
| 변동성/밴드 | `atr`, `bollinger`, `keltner`, `donchian`, `standard_deviation`, `variance` |
| 거래량 기반 | `obv`, `mfi`, `vwap`, `accumulation_distribution`, `chaikin` |
| 수익률/통계 | `return`, `log_return`, `rolling_mean`, `rolling_std`, `zscore`, `correlation`, `beta` |
| 상대/횡단면 | `rank`, `percentile`, `relative_strength`, `spread`, `ratio` |
| 이벤트/패턴 | `new_high`, `new_low`, `breakout`, `drawdown`, candle pattern 계열 |
| 사용자 확장 | custom indicator registry, Go plugin, 저장된 expression alias |

따라서 `indicator.id` 는 strategy schema 의 하드코딩된 enum 이 아니다. schema 는
표현식의 형태를 검증하고, 실제 `id` 와 `params` 의 유효성은 registry 가 판단한다.

## Compile 결과

`Strategy` YAML 은 engine 에 직접 들어가지 않는다.

```text
Strategy YAML
  -> schema validation
  -> typed StrategySpec
  -> compiled EntryRule / ExitRule / PositionSizer / RiskManager
```

최종 실행 단위인 `StrategyPlan` 은 실행 스펙과 결합된 뒤 만들어진다.
