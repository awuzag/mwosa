# Backtester Architecture Spike

## 목적

`mwosa` 의 다음 단계 백테스터는 QuantConnect 같은 완성형 플랫폼을 목표로
하지 않는다. 첫 목표는 사용자가 YAML 로 전략을 정의하고, 저장된 canonical
시장 데이터에 대해 같은 전략을 여러 종목, 기간, 타임프레임으로 반복 실행할
수 있게 만드는 것이다.

현재 `mwosa` 는 `jq` 기반 스크리닝 전략을 저장하고 실행할 수 있다. 이 흐름은
후보군을 고르는 데 강하지만, 포지션, 주문, 체결 가정, 수수료, 손익곡선,
drawdown 같은 시간 순서 기반 검증은 다루지 않는다. 백테스터는 이 빈 구간을
채우는 별도 실행 흐름이다.

## 현재 출발점

이미 있는 자산:

- `service/strategy`: 저장된 `jq` 스크리닝 전략과 실행 기록
- `storage/dailybar`: SQLite canonical daily bar 조회
- `packages/indicators`: provider, storage, CLI 에서 분리된 지표 계산 후보
- `docs/features/jq-screening-strategies/README.md`: 저장 전략 모델과 실행 기록
- `docs/architectures/layers/README.md`: command, handler, service, domain,
  persistence, presentation 경계

백테스터가 새로 가져와야 할 축:

- 시간 순서대로 bar 를 흘리는 engine loop
- signal 과 order 를 분리하는 rule/strategy 모델
- position, cash, equity, trade ledger 를 관리하는 portfolio simulation
- 수수료, 슬리피지, 주문 체결 시점 같은 실행 가정
- 결과 metric 과 실행 재현 정보
- YAML spec 을 검증 가능한 실행 계획으로 compile 하는 단계
- 여러 기간과 파라미터 조합을 반복 검증하는 `Evaluation` 실행 단위

반복 검증과 walk-forward 흐름은
[`evaluation/README.md`](evaluation/README.md) 에서 별도로 관리한다.

## 후보 라이브러리 구조 분석

| 라이브러리 | 구조에서 배울 점 | `mwosa` 적용 판단 |
| --- | --- | --- |
| [`gobacktest/gobacktest`](https://github.com/gobacktest/gobacktest) | `DataHandler -> StrategyHandler -> PortfolioHandler -> ExecutionHandler -> StatisticHandler` 로 이벤트 경계를 분명히 나눈다. data, signal, order, fill event 를 분리하는 사고방식이 좋다. | 직접 의존보다 아키텍처 참고가 적합하다. `mwosa` 도 첫 설계에서는 event type 을 두되, goroutine 기반 event bus 보다 단순한 순차 loop 로 시작한다. |
| [`sdcoffey/techan`](https://github.com/sdcoffey/techan) | `TimeSeries`, `Indicator`, `Rule`, `RuleStrategy`, `TradingRecord` 가 작고 읽기 쉽다. `And`, `Or`, cross rule, position rule 조합이 YAML rule compile 대상과 잘 맞는다. | MIT 라이선스라 지표/규칙 모델 참고 부담이 작다. 다만 full backtester 가 아니므로 engine, execution, portfolio 는 직접 둔다. |
| [`izzoa/backtesting.go`](https://github.com/izzoa/backtesting.go) | `Strategy.Init()` / `Strategy.Next()` 모델, `Broker`, `BacktestConfig`, `Results`, `stats`, `optimize`, `plot` 패키지를 나누고 OHLCV resample 을 제공한다. | 기능 범위는 매력적이지만 AGPL 이므로 의존성 도입은 보류한다. `Broker`, `Stats`, `Optimization` 경계만 참고한다. |
| [`cinar/indicator`](https://github.com/cinar/indicator) | v2 는 channel 기반 stream 처리, indicators, strategies, backtest 를 함께 제공한다. 여러 indicator 와 compound strategy 구성이 풍부하다. | 지표 참고에는 좋지만 v2 라이선스와 범위가 부담이다. 기존 indicator feature 문서처럼 public API 에 외부 타입을 노출하지 않는 wrapper 방향이 맞다. |
| [`c9s/bbgo`](https://github.com/c9s/bbgo) | KLine stream, exchange abstraction, user data stream, matching engine, order update 를 실거래와 백테스트에서 공유한다. | 크립토 실거래 프레임워크라 너무 무겁고 AGPL 이다. `market data -> matching -> user data -> position update` 순서만 참고한다. |
| [`thrasher-corp/gocryptotrader`](https://github.com/thrasher-corp/gocryptotrader) | event-driven backtester, `.strat` config, config builder, report/statistics 흐름이 있다. 전략 실행 설정을 파일로 관리하는 관점이 YAML spec 과 가깝다. | 크립토 봇 전체 프레임워크라 `mwosa` 의존성으로는 과하다. config-driven run 구조와 warning/disclaimer 수준만 참고한다. |
| [`grinply/kate-backtester`](https://github.com/grinply/kate-backtester) | `PreProcessIndicators`, `OpenNewPosition`, `SetStoploss`, `SetTakeProfit` 로 전략 interface 를 작게 유지한다. OHLCV loop 와 포지션 하나 중심의 단순성이 좋다. | 첫 spike 의 크기를 제한하는 참고 모델로 좋다. 다만 멀티 종목, portfolio, saved run 은 `mwosa` 가 직접 설계해야 한다. |
| [`yiplee/go-pine`](https://github.com/yiplee/go-pine) | PineScript 처럼 `OnNextOHLCV` 에서 indicator 를 계산하고 `Strategy.Entry`, `Strategy.Exit` 을 호출한다. state map 을 넘겨 전략 상태를 유지한다. | TradingView 식 사용감은 YAML DSL 설계에 참고할 만하다. production-ready 전제가 약하므로 의존하지 않는다. |

## 구조 패턴

### 1. 이벤트 파이프라인

`gobacktest`, `bbgo`, `gocryptotrader` 쪽에서 공통으로 보이는 구조다.

```text
market data
  -> strategy signal
  -> portfolio/order decision
  -> execution simulation
  -> fill/trade
  -> portfolio state
  -> statistics/report
```

장점은 역할이 선명하다는 점이다. 단점은 첫 구현부터 event bus, channel,
async 처리까지 가져오면 repo 영향도가 커진다는 점이다.

`mwosa` 첫 spike 에서는 동일한 경계를 type 으로 두되 실행은 순차 loop 로 둔다.

```text
for each timestamp:
  load aligned bars
  evaluate compiled rules
  create intended orders
  simulate fills
  update portfolio
  record snapshot
```

### 2. 규칙 조합 모델

`techan` 의 `RuleStrategy` 가 가장 좋은 참고점이다. 진입과 청산을 별도 rule 로
두고, rule 은 `all`, `any`, `not`, `crosses_above`, `crosses_below`,
`greater_than`, `less_than`, `position_open` 처럼 작게 나눈다.

YAML 은 실행 언어가 아니라 rule tree 를 적는 방식이어야 한다.

```yaml
entry:
  all:
    - indicator: close
      above:
        indicator: sma
        window: 20
    - indicator: rsi
      window: 14
      below: 35
exit:
  any:
    - indicator: close
      below:
        indicator: sma
        window: 20
    - stop_loss_pct: 7
```

이 YAML 은 곧바로 실행되지 않는다. 먼저 schema validation 을 통과하고,
`CompiledStrategy` 로 바뀐 뒤 engine 에 들어간다.

### 3. 명령형 전략 모델

`backtesting.go`, `kate-backtester`, `go-pine` 은 모두 bar 가 들어올 때 전략
코드가 호출되는 구조다. 이 방식은 Go 코드 전략을 열어둘 때 필요하다.

다만 `mwosa` 의 1차 목표는 YAML 이므로, Go interface 는 내부 compile target 으로
둔다.

```go
type Strategy interface {
	OnBar(ctx Context) ([]OrderIntent, error)
}
```

YAML 전략도 compile 후에는 이 interface 를 구현하는 값이 된다. 나중에 고급
사용자가 Go 전략 플러그인을 원할 때 같은 engine 을 재사용할 수 있다.

### 4. 설정 파일 기반 실행

`gocryptotrader` 의 `.strat` config 와 `mwosa` 의 saved `jq` strategy 흐름은 같은
방향을 가리킨다. 전략 본문, 입력 데이터 계약, 실행 파라미터, 결과 기록을 함께
보관해야 재현 가능하다.

다만 한 YAML object 안에 모든 것을 넣지는 않는다. 전략 스펙은 "무엇을 보면
사고팔 것인가"를 맡고, 실행 스펙은 "그 전략을 어떤 데이터와 체결 가정으로
실행할 것인가"를 맡는다. 작성 편의성을 위해 Kubernetes manifest 처럼 `---` 로
나눈 multi-document YAML 은 허용한다.

```yaml
kind: Strategy
schema_version: 1
name: trend-pullback

universe:
  symbols: ["069500", "102110"]

entry:
  all:
    - close_above_sma:
        window: 20
    - rsi_below:
        window: 14
        value: 35

exit:
  any:
    - close_below_sma:
        window: 20

sizing:
  type: percent_of_equity
  value: 10

risk:
  max_symbol_weight_pct: 20
---
kind: BacktestRun
schema_version: 1
name: trend-pullback-2024-2026

strategy:
  name: trend-pullback

data:
  market: krx
  timeframe: 1d
  from: 2024-01-01
  to: 2026-05-01

universe:
  pipeline:
    - id: source.symbols
      params:
        symbols: ["069500", "102110"]
        fields:
          market: krx
          security_type: etf

portfolio:
  initial_cash: 10000000

execution:
  fill: next_open
  commission:
    type: bps
    value: 1.5
```

### 5. 멀티 타임프레임

`backtesting.go` 는 resample, `bbgo` 는 KLine interval 수집, `cinar/indicator`
는 stream 기반 처리를 강조한다. `mwosa` 에서도 “일봉 20이평 위에 있고,
현재 5분봉이 5이평 위” 같은 조건을 언젠가는 다뤄야 한다.

첫 spike 에서 멀티 타임프레임을 완성하지는 않되, YAML shape 는 확장을 막지
않게 둔다.

```yaml
data:
  feeds:
    daily:
      timeframe: 1d
    intraday:
      timeframe: 5m
strategy:
  entry:
    all:
      - feed: daily
        indicator: close
        above:
          indicator: sma
          window: 20
      - feed: intraday
        indicator: close
        above:
          indicator: sma
          window: 5
```

엔진은 feed 별 bar 를 정렬하고, 각 timestamp 에서 사용할 수 있는 마지막 완료
bar 만 context 에 넣어야 한다. 이 규칙이 없으면 future data leakage 가 생긴다.

## `mwosa` 권장 경계

백테스터의 엔진 경계는
`docs/architectures/packages/backtest/README.md` 를 기준으로 삼는다. 이 문서는
후보 라이브러리 분석과 첫 spike 범위를 다루고, 실제 engine package 의 책임,
시간 규칙, 데이터 지향 룰셋, 결과 모델은 backtest package architecture 문서에서
관리한다.

레이어 기준은 eolmasa 에서 가져온
`docs/architectures/packages/backtest/trading-engine-layers.md` 의
`StrategySpec -> UniverseSelector -> DataFeed -> EntryRule / ExitRule ->
PositionSizer -> RiskManager -> OrderExecutor -> Portfolio -> Reporter` 흐름을
따른다.

```text
cli/backtest
  -> app/handler/backtest
  -> service/backtest
  -> storage/dailybar
  -> packages/backtest
  -> packages/indicators
```

| 위치 | 맡는 일 | 맡지 않는 일 |
| --- | --- | --- |
| `cli/backtest` | `mwosa run backtest <yaml>`, `mwosa validate backtest <yaml>`, saved strategy upsert/list/inspect/delete 명령 | YAML 직접 실행, portfolio 계산 |
| `app/handler/backtest` | CLI request 를 service request 로 변환 | engine loop, storage SQL |
| `service/backtest` | YAML 로드, schema validation, 데이터 조회, 실행 기록 조립 | indicator 세부 계산, table 렌더링 |
| `storage/backtest` | saved spec, run, equity curve, trades 저장 | 전략 판단 |
| `packages/backtest` | engine loop, rule compile target, broker simulation, result metric | Cobra, Bun, provider, file path |
| `packages/indicators` | 순수 지표 계산 | portfolio, 주문, 저장소 |

`packages/backtest` 는 core package 로 시작하는 편이 좋다. provider, storage,
presentation 을 모르게 두면 나중에 YAML schema 와 engine 을 별도로 테스트하기
쉽다. service layer 는 daily bar 를 읽어 engine input 으로 변환하고, 실행 결과를
storage/presentation 에 넘긴다.

## 첫 스키마 초안

초기 YAML schema 는 넓게 열어두기보다 최소 실행 가능한 항목만 둔다. 스키마는
두 kind 로 분리한다.

`Strategy` 필수:

- `kind: Strategy`
- `schema_version`
- `name`
- `universe.symbols`
- `entry`
- `exit`
- `sizing`

`Strategy` 선택:

- `description`
- `risk.max_positions`
- `risk.max_symbol_weight_pct`
- `tags`

`BacktestRun` 필수:

- `kind: BacktestRun`
- `schema_version`
- `name`
- `strategy.name`
- `data.market`
- `universe` 후보 field 또는 `filter.security_type`
- `data.timeframe`
- `data.from`
- `data.to`
- `portfolio.initial_cash`
- `execution.fill`

`BacktestRun` 선택:

- `benchmark`
- `execution.commission`
- `execution.slippage`
- `report.metrics`

초기에는 아래를 제외한다.

- live trading
- margin/leverage
- short selling
- option/futures payoff
- partial fill
- order book simulation
- genetic optimization
- Python 호환 layer

## 첫 spike 범위

1차 spike 는 “정확한 플랫폼” 이 아니라 “흐름이 보이는 작은 엔진” 이다.

완료 기준:

- YAML spec 1개를 schema validation 할 수 있다.
- canonical daily bar 로 단일 종목, 단일 timeframe 백테스트를 실행한다. 단일 종목은 멀티 종목 포트폴리오 엔진의 특수 케이스로 다룬다.
- 종목별 rule 평가는 독립적으로 테스트하되, sizing, risk, execution, portfolio 갱신은 하나의 portfolio global zone 에서 처리한다.
- 진입/청산 rule 은 `all`, `any`, `crosses_above`, `above`, `below` 정도만 지원한다.
- 체결 가정은 `next_open` 하나로 시작한다.
- position, cash, equity curve, trade list 를 JSON 으로 반환한다.
- 같은 spec, 같은 data range 는 같은 result hash 를 만든다.
- `go test ./...` 로 engine/rule/schema 단위 테스트를 확인한다.

첫 구현에서 피할 것:

- 외부 백테스팅 프레임워크 의존성 추가
- AGPL 라이브러리 의존성 추가
- 실거래 provider/exchange abstraction
- 비동기 event bus
- 복잡한 최적화와 차트 생성

## 결론

`mwosa-backtester` 는 직접 작은 엔진을 만드는 방향이 맞다. 참고 모델은 다음처럼
나눠서 가져온다.

- engine 경계: `gobacktest`
- rule 조합: `techan`
- broker/statistics 경계: `backtesting.go`
- config-driven 실행: `gocryptotrader`
- 단순 spike 크기: `kate-backtester`
- PineScript 식 사용감: `go-pine`
- stream/matching 순서: `bbgo`

의존성은 당장 추가하지 않는다. 먼저 `packages/backtest` 의 순수 Go engine 과
YAML schema 를 작게 만들고, indicator 계산은 기존 `packages/indicators` 방향과
연결한다.
