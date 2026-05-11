# Execution Spec

실행 스펙은 "같은 전략을 어떤 조건으로 실행할 것인가"를 정의한다.

전략 스펙이 신호와 룰을 설명한다면, 실행 스펙은 데이터 범위, 초기 포트폴리오,
체결 모델, 비용, 리포트 설정을 설명한다. 첫 구현에서는 `kind: BacktestRun` 을
사용한다. 나중에 페이퍼 트레이딩을 추가하면 `PaperRun`, 실거래 후보 검증을
추가하면 `LiveRunPlan` 같은 별도 kind 로 확장할 수 있다.

## 기본 형태

```yaml
kind: BacktestRun
schema_version: 1
name: trend-pullback-2024-2026

strategy:
  name: trend-pullback

data:
  market: krx
  security_type: etf
  timeframe: 1d
  from: 2024-01-01
  to: 2026-05-01

universe:
  symbols: ["069500", "102110"]

portfolio:
  initial_cash: 10000000
  currency: KRW

execution:
  fill: next_open
  commission:
    type: bps
    value: 1.5
  slippage:
    type: bps
    value: 0

report:
  metrics:
    preset: core
    include:
      - benchmark_total_return
      - excess_return
    exclude:
      - trade_count
```

## 맡는 일

`BacktestRun` 이 맡는 일:

- 어떤 strategy 를 실행할지 참조한다.
- market, security type, timeframe, 기간을 정한다.
- 실행할 종목 universe 를 정한다.
- 초기 현금과 기준 통화를 정한다.
- 체결 가격, 수수료, 슬리피지 같은 execution model 을 정한다.
- 리포트 metric 과 결과 기록 방식을 정한다.

`BacktestRun` 이 맡지 않는 일:

- 진입 rule tree
- 청산 rule tree
- strategy-local sizing rule
- strategy-local risk limit
- indicator 계산 로직
- portfolio 상태 직접 변경

## 필수 필드

| 필드 | 의미 |
| --- | --- |
| `kind` | 첫 구현에서는 `BacktestRun` |
| `schema_version` | execution schema version |
| `name` | 실행 기록 식별 이름 |
| `strategy` | 같은 YAML stream 안의 strategy 또는 저장된 strategy 참조 |
| `data.market` | market id |
| `data.security_type` | security type |
| `data.timeframe` | 실행 timeframe |
| `data.from` | 시작일 |
| `data.to` | 종료일 |
| `universe` | 실행할 종목 집합 또는 selector |
| `portfolio.initial_cash` | 초기 현금 |
| `execution.fill` | 체결 가격 가정 |

## Strategy 참조

같은 파일 안에 `kind: Strategy` 문서가 있으면 `strategy.name` 으로 참조한다.

```yaml
strategy:
  name: trend-pullback
```

저장된 strategy 를 실행할 때는 저장소 참조를 따로 열어둘 수 있다.

```yaml
strategy:
  ref: saved:trend-pullback
```

첫 구현에서는 한 파일 안의 `strategy.name` 참조를 우선 지원하고, 저장소 참조는
service layer 에서 확장한다.

## Report metric 선택

`report.metrics` 를 생략하면 기본 core metric 만 출력한다.

```yaml
report:
  metrics:
    preset: core
    include:
      - average_trade_return
    exclude:
      - trade_count
```

metric id 는 schema enum 으로 고정하지 않고 backtest package 의 metric registry
에서 검증한다. 알 수 없는 metric id 는 compile 단계에서 실패한다.

benchmark 관련 metric 은 `benchmark.symbol` 이 있을 때만 사용할 수 있다.

```yaml
benchmark:
  symbol: "069500"
  name: "KODEX 200"

report:
  metrics:
    include:
      - benchmark_total_return
      - excess_return
      - benchmark_max_drawdown
      - monthly_win_rate_vs_benchmark
```

## Compile 결과

```text
BacktestRun YAML
  -> schema validation
  -> typed BacktestRunSpec
  -> data query request + portfolio seed + execution model config
```

`BacktestRunSpec` 은 단독으로 엔진에 들어가지 않는다. `StrategySpec` 과 함께
compile 되어 하나의 `StrategyPlan` 이 된다.
