# Backtest Package Architecture

## 목적

`packages/backtest` 는 `mwosa` 안에서 독립적으로 관리하는 결정론적 시장
시뮬레이션 엔진이다.

이 패키지의 목표는 과거 데이터를 보고 수익률을 계산하는 작은 helper 가 아니다.
시장 데이터를 시간 순서대로 흘리고, 전략 룰셋을 평가하고, 주문과 체결 가정을
적용하고, 포트폴리오 상태를 바꾸고, 모든 변화를 재현 가능한 이벤트로 남기는
엔진이다.

CLI, SQLite, provider, YAML 파일 경로는 이 엔진의 바깥에 있다. `mwosa` 의 다른
레이어는 이 엔진을 가져다 쓰는 쪽이고, `packages/backtest` 는 스스로 실행 규칙,
상태 전이, 결과 모델을 소유한다.

## 핵심 관점

백테스터는 전략 판단기가 아니라 시뮬레이터다.

```text
historical market data
  -> simulation clock
  -> data-oriented strategy rules
  -> order intents
  -> execution model
  -> fills and costs
  -> portfolio state
  -> events, trades, equity curve, statistics
```

게임 엔진의 전투 시스템과 비슷하게 볼 수 있다. 전략, 리스크, 비용, 태그,
상태 변화는 코드에 흩어진 분기문이 아니라 데이터로 정의된 룰셋이 된다.

다만 게임 엔진과 달리 백테스터는 미래를 만들어내지 않는다. 이미 지나간 시간
안에서, 그 시점에 알 수 있었던 정보만 사용해 결정론적으로 다시 실행한다.

## GAS 비유

Unreal Engine GAS 의 데이터 지향 구조는 백테스터 설계에 좋은 참고 모델이다.

| GAS 개념 | 백테스터 개념 |
| --- | --- |
| Attribute | cash, position quantity, average price, equity, drawdown, volatility |
| Gameplay Tag | `market.trend.up`, `position.open`, `risk.blocked`, `signal.entry` |
| Gameplay Effect | fill buy, fill sell, charge commission, update stop, close position |
| Ability | entry rule, exit rule, rebalance rule, risk rule |
| Cost / Cooldown | commission, slippage, max positions, rebuy cooldown |
| Gameplay Cue | signal event, rejected order event, trade event, risk block reason |
| Data Asset | YAML strategy spec, YAML execution spec, execution model, commission model, risk model |

이 비유의 핵심은 "데이터가 행동을 설명한다"는 점이다. `packages/backtest` 는
YAML 자체를 읽지 않아도, YAML 이 compile 된 data asset 을 받아 같은 방식으로
실행할 수 있어야 한다.

## 위치와 구조 결정 순서

현재 확정하는 것은 `packages/backtest` 라는 독립 엔진 경계다. 하위 폴더 구조는
아직 확정하지 않는다. 먼저
`docs/architectures/packages/backtest/trading-engine-layers.md` 의 레이어
결정을 기준으로 책임과 public contract 를 정하고, 그 다음에 package 와 directory
를 나눈다.

첫 레이어 기준은 eolmasa 에서 정리한 trading engine 흐름을 가져온다.

```text
StrategySpec
  -> UniverseSelector
  -> DataFeed
  -> EntryRule / ExitRule
  -> PositionSizer
  -> RiskManager
  -> OrderExecutor
  -> Portfolio
  -> Reporter
```

이 목록은 폴더명이 아니다. 책임을 먼저 나누기 위한 레이어 결정이다.
`packages/backtest` 가 커질 가능성이 높으므로 디렉터리보다 레이어 계약을 먼저
잡는다. `ExecutionModel` 은 백테스트와 페이퍼 트레이딩에서 `OrderExecutor` 가
사용하는 체결 가정으로 둔다.

`packages/backtest` 를 독립 Go module 로 둘지도 아직 결정하지 않는다. CLI 없이
테스트와 benchmark 를 독립적으로 돌릴 필요가 분명해지면 그때 `go.mod` 분리를
결정한다.

## 레이어 결정

레이어별 상세 책임은 `trading-engine-layers.md` 와 각 `layers/*/README.md` 에
둔다. 이 README 는 `packages/backtest` 관점에서 각 레이어가 엔진 안에서 어떤
경계를 이루는지만 요약한다.

| 레이어 | 질문 | 엔진 안의 책임 |
| --- | --- | --- |
| `StrategySpec` | 어떤 룰셋을 실행할까? | YAML 같은 선언형 전략을 검증 가능한 data asset 으로 표현한다. 실행 기간과 체결 가정은 갖지 않는다. |
| `UniverseSelector` | 무엇을 볼까? | 현재 시점에 관찰할 종목 집합을 만든다. 진입 판단은 하지 않는다. |
| `DataFeed` | 무엇을 알고 있나? | 백테스트에서는 과거 데이터를 시간순으로 재생하고, 현재 clock 에서 알 수 있는 snapshot 만 제공한다. |
| `EntryRule` | 언제 들어갈까? | 신규 진입 의도나 신호를 만든다. 포트폴리오는 직접 바꾸지 않는다. |
| `ExitRule` | 언제 나올까? | 보유 포지션의 축소 또는 청산 의도를 만든다. |
| `PositionSizer` | 얼마나 걸까? | 주문 후보의 수량 또는 금액을 계산한다. |
| `RiskManager` | 이 위험을 받아도 될까? | 주문 후보를 승인, 거절, 축소하고 이유를 남긴다. |
| `OrderExecutor` | 주문을 어떻게 실행할까? | 리스크 검증을 통과한 주문 의도를 체결 결과로 바꾼다. |
| `ExecutionModel` | 어떻게 체결됐다고 볼까? | 백테스트/페이퍼 트레이딩의 가상 체결 가격, 수량, 비용을 결정한다. |
| `Portfolio` | 지금 계좌는 어떤 상태인가? | 체결 결과를 반영해 현금, 포지션, 손익, equity 를 갱신한다. |
| `Reporter` | 결과가 좋았나? | 거래, 계좌, 리스크, 체결 이벤트와 통계를 기록한다. |

## 종목별 병렬 구간과 포트폴리오 전역 구간

멀티 종목 백테스트는 종목별 백테스트를 독립 실행한 뒤 결과를 합치는 방식으로
설계하지 않는다. 현금, 최대 보유 종목 수, 종목별 비중, 동일 시점의 주문 충돌,
리스크 제한은 모두 하나의 `Portfolio` 상태를 기준으로 판단해야 한다.

단일 종목 백테스트는 별도 엔진이 아니라 아래 구조에서 `Universe` 가 종목 하나인
특수 케이스로 본다.

```text
per-symbol parallel zone
  data load
  indicator calculation
  entry/exit rule evaluation

portfolio global zone
  order intent aggregation
  position sizing
  risk review
  execution
  portfolio mutation
  reporting
```

병렬화는 후보 신호를 만드는 데까지만 우선 허용한다. `PositionSizer`,
`RiskManager`, `OrderExecutor`, `Portfolio` 는 전체 계좌 상태를 기준으로 같은
clock 안에서 순서를 보장하며 실행한다.

## 맡는 일

`packages/backtest` 가 맡는 일:

- simulation clock 을 기준으로 market data 를 시간 순서대로 소비한다.
- strategy data asset 을 실행 가능한 rule graph 로 다룬다.
- `UniverseSelector`, `EntryRule`, `ExitRule`, `PositionSizer`, `RiskManager` 를 순서대로 평가한다.
- signal 을 order intent 로 바꾼다.
- `OrderExecutor` 와 `ExecutionModel` 로 fill, commission, slippage 를 계산한다.
- portfolio state 를 변경한다.
- trade ledger, equity curve, event log, statistics 를 만든다.
- future data leakage 를 막기 위한 시간 규칙을 강제한다.
- 같은 입력과 같은 설정에서 같은 결과가 나오도록 결정론을 유지한다.

`packages/backtest` 가 맡지 않는 일:

- provider API 호출
- SQLite 조회와 저장
- YAML 파일 읽기
- Cobra command, flag, stdin 처리
- table, json, ndjson, csv 렌더링
- 사용자의 전략 파일 관리
- 저장된 backtest run 검색
- live trading

## 외부 레이어와의 관계

```text
cli/backtest
  -> app/handler/backtest
  -> service/backtest
  -> storage/dailybar
  -> packages/backtest
  -> packages/indicators
```

`service/backtest` 는 YAML 또는 저장된 spec 을 검증하고, 저장소의 canonical
bar reader 를 `packages/backtest` 의 streaming feed port 로 감싼다. 엔진은
repository, SQLite, YAML 을 직접 알지 않고, compile 된 strategy plan 과
시간순 `BarFrame` stream 만 소비한다.

```text
YAML stream or saved specs
  -> service/backtest schema validation
  -> StrategySpec + BacktestRunSpec
  -> compiled StrategyPlan
  -> service/backtest repository-backed StreamingFeed
  -> packages/backtest.Engine.Run(plan, BarStream)
  -> backtest result
  -> storage/backtest and presentation
```

`packages/backtest` 는 YAML parser 를 직접 소유하지 않는다. YAML 은 `mwosa`
사용자 경험의 한 입력 방식이고, 엔진의 public contract 는 Go type 이다.

YAML 은 Kubernetes manifest 처럼 `---` 로 나뉜 여러 document 를 한 파일에 담을
수 있다. 이때 `kind: Strategy` 는 전략 룰셋만 정의하고, `kind: BacktestRun` 은
데이터 기간, 초기 현금, 체결 모델, 리포트 설정 같은 실행 조건을 정의한다.

## Streaming feed 와 rolling indicator

`packages/backtest` 의 data port 는 repository 가 아니라 lazy 가능한 market
data feed 다. 현재 core 계약은 `StreamingFeed.Open(ctx, DataRequest)` 로 stream
을 열고, `BarStream.Next(ctx)` 가 같은 timestamp 의 종목별 snapshot 인
`BarFrame` 을 하나씩 반환한다. `MemoryFeed` 는 테스트와 fixture 를 위한 같은
streaming 계약의 in-memory 구현이다.

engine loop 는 simulation clock 을 frame 단위로 순차 진행한다. pending order
execution, signal evaluation, sizing, risk review, portfolio mutation, equity
curve 기록은 결정론을 위해 한 clock 안에서 순차 처리한다.

indicator 는 전체 series 를 먼저 계산하지 않는다. stream 에서 현재 frame 이
들어오면 종목별 rolling state 를 갱신하고, rule evaluator 는 현재와 이전 값만
읽는다. warmup 이 끝나기 전 `NaN` 이거나 준비되지 않은 indicator 값은 rule
match 로 이어지지 않는다.

evaluation 병렬화는 engine 내부 portfolio state 를 병렬 mutation 하지 않는다.
여러 case 또는 walk-forward train case 를 bounded worker pool 로 독립 실행하고,
각 worker 는 독립 feed stream, engine instance, rolling indicator state 를 가진다.
ranking, 저장, walk-forward test 실행은 spec 순서를 기준으로 결정론을 유지한다.

## Techan 에서 가져올 관점

`techan` 이 좋은 이유는 full platform 이라서가 아니라, rule model 이 작고
조합하기 쉽기 때문이다.

`packages/backtest` 의 rule 은 아래 성질을 가져야 한다.

- rule 은 현재 simulation context 를 받아 true/false 를 돌려준다.
- rule 은 `All`, `Any`, `Not` 으로 조합된다.
- rule 은 indicator, tag, position state, portfolio attribute 를 읽을 수 있다.
- rule 은 직접 주문을 만들지 않는다.
- entry rule, exit rule, risk rule 은 같은 rule interface 를 공유하되 역할이 다르다.

예시 interface:

```go
type Rule interface {
	Evaluate(Context) (Decision, error)
}

type Decision struct {
	Matched bool
	Tags    []Tag
	Reason  string
}
```

`Decision` 에 `Reason` 과 `Tags` 를 남겨야 backtest 결과에서 "왜 진입했는지",
"왜 차단됐는지"를 설명할 수 있다.

## Gobacktest 에서 가져올 관점

`gobacktest` 의 전체 파이프라인은 첫 엔진 경계에 유용하다.

```text
DataHandler
  -> StrategyHandler
  -> PortfolioHandler
  -> ExecutionHandler
  -> StatisticHandler
```

`mwosa` 는 이 구조를 그대로 복사하지 않는다. 대신 아래처럼 순차 엔진 내부의
책임으로 가져온다.

```text
MarketFeed
  -> StrategyPlan
  -> Portfolio
  -> ExecutionModel
  -> Stats
```

첫 구현은 goroutine, channel, event bus 없이 순차 loop 로 둔다.

```text
for clock.Next():
  universe = universeSelector.Select(clock.Time)
  snapshot = dataFeed.Snapshot(clock.Time, universe)
  context = buildContext(snapshot, portfolio, tags, indicators)
  exits = exitRules.EvaluateBySymbol(context)
  entries = entryRules.EvaluateBySymbol(context)
  intents = aggregateOrderIntents(exits, entries)
  sized = positionSizer.Size(intents, context)
  approved = riskManager.Review(sized, context)
  fills = orderExecutor.Execute(approved, executionModel, snapshot)
  portfolio.Apply(fills)
  recorder.Append(events, portfolio)
```

이 구조는 단순하지만 event 경계가 보인다. 나중에 필요하면 내부 recorder 나
market feed 를 더 정교하게 바꿀 수 있다.

## 데이터 모델

초기 public type 은 provider-neutral 해야 한다.

```go
type Bar struct {
	Symbol string
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

type MarketSnapshot struct {
	Time time.Time
	Bars map[Symbol]Bar
}
```

portfolio state 는 engine 내부에서 변경되며, rule 은 읽기 전용 view 를 받는다.

```go
type PortfolioView interface {
	Cash() float64
	Equity() float64
	Position(Symbol) Position
	OpenPositions() []Position
}
```

상태 변경은 직접 field 를 바꾸는 방식이 아니라 event/effect 적용으로 처리한다.

```go
type Event struct {
	Time   time.Time
	Type   EventType
	Symbol Symbol
	Tags   []Tag
	Data   map[string]any
}
```

## 태그와 속성

태그는 문자열을 그대로 흩뿌리지 않고 type 으로 관리한다.

```go
type Tag string

const (
	TagMarketTrendUp Tag = "market.trend.up"
	TagPositionOpen  Tag = "position.open"
	TagRiskBlocked   Tag = "risk.blocked"
	TagSignalEntry   Tag = "signal.entry"
)
```

태그는 아래 용도로 쓴다.

- rule 평가 결과 설명
- risk rule 차단 사유
- event log 필터링
- YAML rule 간 조건 연결
- report 에서 진입/청산 사유 요약

attribute 는 portfolio, market, symbol context 의 숫자 상태를 뜻한다. 예:

- `portfolio.cash`
- `portfolio.equity`
- `symbol.position_quantity`
- `symbol.unrealized_pnl`
- `symbol.drawdown_pct`
- `market.volatility`

첫 구현에서 attribute registry 를 크게 만들 필요는 없다. 다만 rule 이 숫자
상태를 읽는 경로를 tag 와 분리해둔다.

## StrategyPlan

YAML 은 engine 밖에서 `StrategyPlan` 으로 compile 된다. `StrategySpec` 과
`BacktestRunSpec` 은 분리된 입력이지만, engine 이 실행하는 단위는 둘이 결합된
하나의 plan 이다.

```text
StrategySpec
  + BacktestRunSpec
  + market data request result
  -> StrategyPlan
```

```go
type StrategyPlan struct {
	Name             string
	RunName          string
	UniverseSelector UniverseSelector
	DataFeed         DataFeed
	EntryRules       []Rule
	ExitRules        []Rule
	PositionSizer    PositionSizer
	RiskManager      RiskManager
	OrderExecutor    OrderExecutor
	ExecutionModel   ExecutionModel
}
```

초기에는 하나의 `StrategySpec` 과 하나의 `BacktestRunSpec` 이 하나의
`StrategyPlan` 으로 compile 된다. 나중에 여러 strategy 를 같은 portfolio 에
올리는 multi-strategy simulation 을 지원할 수 있지만, 첫 spike 에서는 제외한다.

## ExecutionModel

execution 은 신호와 포트폴리오 상태를 실제 fill 로 바꾸는 경계다.

```go
type ExecutionModel interface {
	Fill(ctx Context, orders []OrderIntent) ([]Fill, error)
}
```

초기 execution model:

- `next_open`
- `market`, `limit`, `stop`, `stop_limit`, `trailing_stop`, `rebalance`
  order type
- fixed bps commission
- no slippage 또는 fixed bps slippage
- volume/traded amount 기반 partial fill
- short selling 없음

`next_open` 은 close 기준 신호와 다음 거래일 open 체결을 분리하기 위한 기본
가정이다. 같은 bar 의 close 로 신호를 만들고 같은 close 로 체결하면 결과가
과하게 좋아질 수 있다.

## 결과 모델

엔진 결과는 presentation-ready text 가 아니라 구조화된 데이터다.

```go
type Result struct {
	SpecHash      string
	DataHash      string
	ResultHash    string
	StartedAt     time.Time
	FinishedAt    time.Time
	Trades        []Trade
	EquityCurve   []EquityPoint
	Events        []Event
	Stats         Stats
}
```

`SpecHash`, `DataHash`, `ResultHash` 는 재현성을 위해 필요하다. 같은 spec,
같은 market data, 같은 engine version, 같은 registry version 이면 같은 결과가
나와야 한다. 현재 `Result.runtime` 은 `engine_version`,
`indicator_registry_version`, `metric_registry_version` 을 남기며, 이 값들은
`result_hash` 계산 payload 에 포함된다. 저장소는 saved run, evaluation case,
walk-forward step, raw result row 에도 같은 runtime metadata 를 독립 컬럼으로
남긴다.

## 시간 규칙

백테스터에서 가장 중요한 규칙은 미래 데이터 누수를 막는 것이다.

기본 원칙:

- simulation clock 은 단조 증가한다.
- rule 은 현재 clock 에서 사용 가능한 market snapshot 만 볼 수 있다.
- indicator 는 현재 clock 까지 닫힌 bar 만 사용한다.
- 장마감 기준 신호는 같은 장마감 가격으로 체결하지 않는다.
- `next_open` 은 다음 사용 가능한 bar 의 open 을 사용한다.
- multi-timeframe feed 는 각 feed 의 마지막 완료 bar 만 context 에 넣는다.
- warm-up 구간은 trade 하지 않거나 명시적인 `warmup` 상태로 남긴다.

이 규칙은 convenience 보다 우선한다. 사용자가 더 낙관적인 체결 모델을 원하더라도
그 가정은 YAML/spec 에 명시되어야 한다.

## 테스트 기준

`packages/backtest` 는 `mwosa` 안에서 가장 큰 core package 가 될 가능성이 높다.
따라서 단위 테스트와 fixture 를 처음부터 패키지의 일부로 본다.

필수 테스트:

- 같은 입력과 같은 spec 은 같은 result hash 를 만든다.
- 정렬되지 않은 bar 입력은 실패한다.
- future data leakage 를 만드는 rule 또는 feed 접근은 실패한다.
- warm-up 이전에는 거래가 발생하지 않는다.
- `next_open` 체결이 다음 bar open 을 사용한다.
- commission 과 slippage 가 cash/equity/trade 에 반영된다.
- insufficient cash 는 성공처럼 숨기지 않고 reject event 를 남긴다.
- entry/exit rule 의 `All`, `Any`, `Not` 조합이 기대대로 동작한다.

선택 테스트:

- 단일 종목 fixture
- 다중 종목 fixture
- multi-timeframe fixture
- benchmark: bars x symbols 규모별 실행 시간

## 첫 구현 범위

1차 구현은 아래만 포함한다.

- daily bar 기반 단일 timeframe simulation
- long-only
- full cash account
- one strategy per run
- `All`, `Any`, `Not`, `Above`, `Below`, `CrossesAbove`, `CrossesBelow`
- fixed percent-of-equity sizing
- `next_open` execution
- fixed bps commission
- trade ledger, equity curve, event log, basic stats

1차 구현에서 제외한다.

- live trading
- margin, leverage, short selling
- partial fill
- order book simulation
- optimization
- chart rendering
- async event bus
- external backtesting framework dependency
- AGPL library dependency

## 남은 결정

아직 정해야 할 항목:

- `packages/backtest` 를 독립 Go module 로 바로 만들지 여부
- `packages/backtest` 내부 하위 package 와 directory 를 어떻게 나눌지 여부
- Strategy/BacktestRun YAML schema 를 service layer 에 둘지 별도 `service/backtest/spec` 로 둘지 여부
- decimal type 을 도입할지, 첫 구현은 `float64` 로 제한할지 여부
- `packages/backtest` 가 `packages/indicators` 를 직접 import 할지, indicator 값도 compile 단계에서 주입할지 여부
- tag catalog 를 코드 상수로만 둘지, YAML schema 와 함께 문서화할지 여부
- result hash 에 포함되는 runtime metadata 를 schema/storage view 에서 어디까지
  독립 컬럼으로 노출할지 여부
- backtest run 저장소를 첫 spike 에 포함할지 여부
- eolmasa 문서의 `PaperExecutor`, `LiveExecutor` 개념을 `mwosa` 범위에서 언제까지 보류할지 여부

## 관련 문서

- `docs/architectures/packages/backtest/final-implementation-goal.md`
- `docs/architectures/packages/backtest/trading-engine-layers.md`
- `docs/architectures/packages/backtest/strategy-spec/README.md`
- `docs/architectures/packages/backtest/execution-spec/README.md`
- `docs/architectures/packages/backtest/references/strategy-asset-schema-references.md`
- `docs/architectures/packages/README.md`
- `docs/architectures/packages/indicators/README.md`
- `docs/architectures/layers/README.md`
- `docs/architectures/tech-stack/README.md`
- `docs/features/backtester/README.md`
