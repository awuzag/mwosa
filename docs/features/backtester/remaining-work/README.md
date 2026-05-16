# Backtester Remaining Work

## Purpose

이 문서는 백테스터 최종 목표 중 아직 구현이 덜 된 항목을 추후 작업자가 바로
이어받을 수 있게 정리한다. 안정적인 아키텍처 계약은
`docs/architectures/packages/backtest/final-implementation-goal.md` 를 기준으로
보고, 이 문서는 실행 가능한 backlog 로 유지한다.

완료된 항목은 오래 쌓아두지 않는다. 관련 PR, 커밋, 검증 명령을 남긴 뒤 표에서
상태를 `done` 으로 바꾸거나 별도 완료 기록으로 옮긴다.

## Status Legend

| Status | Meaning |
| --- | --- |
| `todo` | 아직 구현을 시작하지 않았다. |
| `doing` | 현재 작업 중이다. |
| `blocked` | 외부 결정, 데이터, 선행 작업이 필요하다. |
| `done` | 구현과 검증이 끝났다. |

## Priority Index

| ID | Priority | Area | Summary | Status |
| --- | --- | --- | --- | --- |
| `BT-001` | `P0` | Timeframe | intraday feed 와 named multi-feed 구현 | `todo` |
| `BT-002` | `P0` | Registry | rule/expression registry 분리 | `todo` |
| `BT-003` | `P1` | Strategy | indicator/expression 표현력 확장 | `todo` |
| `BT-004` | `P1` | Risk | sizing/risk/account model 확장 | `todo` |
| `BT-005` | `P1` | Execution | 고급 execution/account 정책 보강 | `todo` |
| `BT-006` | `P2` | Research | Bayesian optimizer 실제 구현 | `todo` |
| `BT-007` | `P2` | Reporting | core Reporter 계약 정리 | `todo` |
| `BT-008` | `P2` | Docs | 최종 목표 문서와 현재 구현 상태 정합성 갱신 | `todo` |

## BT-001 Intraday Feed And Named Multi-Feed

- 상태: `todo`
- 목표: canonical daily 기반 `1d/1w/1mo` 실행을 넘어 `1m`, `5m`, `15m`,
  `30m`, `1h`, `custom` 을 실제 feed 계약으로 연결한다.
- 현재 상태: `packages/backtest` 의 timeframe parser 는 intraday 값을 받지만,
  `service/backtest` 의 canonical daily feed 는 daily-compatible timeframe 만
  허용한다.
- 주요 파일:
  - `packages/backtest/feed.go`
  - `packages/backtest/timeframe.go`
  - `packages/backtest/resample.go`
  - `service/backtest/feed.go`
  - `docs/architectures/packages/backtest/timeframes/README.md`
- 완료 조건:
  - `daily trend + 5m entry` fixture 가 같은 `Engine.Run` loop 에서 동작한다.
  - named feed 또는 그에 준하는 data contract 로 rule 이 timeframe source 를
    명확히 선택한다.
  - 닫히지 않은 상위 timeframe bar 를 읽는 future leakage 가 테스트에서 실패한다.
  - intraday fixture 와 daily fixture 가 같은 `BarFrame`/clock 추상화를 사용한다.
- 비범위:
  - live trading 또는 실시간 주문 전송.
  - provider API 직접 호출을 `packages/backtest` 로 끌어오는 변경.

## BT-002 Rule And Expression Registry

- 상태: `todo`
- 목표: indicator/metric registry 처럼 rule 과 value expression 도 registry 기반
  확장 지점으로 분리한다.
- 현재 상태: rule 과 expression 검증/평가는 `switch` 중심으로 구현되어 있다.
- 주요 파일:
  - `packages/backtest/compiler.go`
  - `packages/backtest/rule.go`
  - `packages/backtest/registry.go`
  - `docs/architectures/packages/backtest/strategy-spec/README.md`
- 완료 조건:
  - `RuleRegistry`, `ExpressionRegistry` 또는 그에 준하는 public contract 가 있다.
  - unknown rule/expression 은 registry 에서 명시적으로 실패한다.
  - 기존 YAML 예제와 테스트는 동작을 유지한다.
  - registry version 이 result metadata/hash 정책과 충돌하지 않는다.
- 비범위:
  - 외부 라이브러리 타입을 public strategy API 에 노출하는 변경.

## BT-003 Strategy Expressiveness

- 상태: `todo`
- 목표: 최종 목표 문서의 indicator/expression roadmap 중 아직 빠진 항목을
  단계적으로 채운다.
- 현재 상태: `sma`, `ema`, `wma`, `kama`, `hma`, `rsi`, `macd`,
  `stochastic`, `roc`, `adx`, `atr`, `natr`, `bollinger`, `keltner`,
  `donchian`, `zscore`, `correlation`, `beta` 와 주요 횡단면 표현식은 있다.
- 남은 후보:
  - 이동평균: `dema`, `tema`, `vwma`
  - 모멘텀/오실레이터: `cci`, `momentum`, `williams_r`
  - 추세/강도: `aroon`, `trix`, `psar`
  - 거래량 기반: `obv`, `mfi`, `vwap`, `accumulation_distribution`, `chaikin`
  - 수익률/통계: `return`, `log_return`, `rolling_mean`, `rolling_std`,
    `standard_deviation`, `variance`
  - 변환 표현식: `lag`, `change`, `pct_change`, `rolling`
- 주요 파일:
  - `packages/backtest/registry.go`
  - `packages/backtest/rolling.go`
  - `packages/backtest/rule.go`
  - `packages/backtest/engine_test.go`
- 완료 조건:
  - 각 indicator 는 warmup, source, params, output 검증을 가진다.
  - future leakage 없이 streaming runtime 에서 계산된다.
  - YAML round-trip 및 core engine fixture 테스트가 있다.

## BT-004 Sizing, Risk, And Account Model

- 상태: `todo`
- 목표: 단순 percent-of-equity 와 기본 한도를 넘어 portfolio-level risk/account
  모델을 확장한다.
- 현재 상태: sizing 은 `percent_of_equity`, risk 는 `max_positions`,
  `max_symbol_weight_pct` 중심이다.
- 남은 후보:
  - sizing: equal weight, fixed amount, volatility target, inverse volatility,
    risk parity style allocation
  - risk: sector/product exposure, max turnover, max drawdown guard, per-symbol
    cooldown, policy chain
  - account: short, leverage, margin, multi-currency/fx
- 주요 파일:
  - `packages/backtest/spec.go`
  - `packages/backtest/portfolio.go`
  - `packages/backtest/engine.go`
  - `docs/architectures/packages/backtest/layers/position-sizer/README.md`
  - `docs/architectures/packages/backtest/layers/risk-manager/README.md`
- 완료 조건:
  - sizing 과 risk decision 이 event 로 설명된다.
  - risk 축소, 거절, 강제 청산 같은 결과가 deterministic 하다.
  - account model 변경이 기존 long-only ETF fixture 를 깨지 않는다.

## BT-005 Execution And Account Policies

- 상태: `todo`
- 목표: 현재 구현된 order/fill/cost/liquidity 정책을 product/account 수준까지
  확장한다.
- 현재 상태: `market`, `limit`, `stop`, `stop_limit`, `trailing_stop`,
  `rebalance`, TIF, partial fill, slippage, cost, liquidity cap 은 들어가 있다.
- 남은 후보:
  - order type: `rebalance_target`, `close_position`
  - fee: tiered commission, product-specific fee/tax table
  - liquidity: symbol-specific cap, no-trade policy variants
  - account: short/leverage/margin constraint 와 forced rejection event
  - metadata: `same_close` look-ahead risk marker 강화
- 주요 파일:
  - `packages/backtest/compiler.go`
  - `packages/backtest/portfolio.go`
  - `packages/backtest/engine.go`
  - `docs/architectures/packages/backtest/execution-spec/README.md`
- 완료 조건:
  - fill, partial fill, reject, defer, cancel 사유가 view 별 출력에 남는다.
  - 비용/슬리피지가 trade, cash, equity, metrics 에 일관되게 반영된다.
  - execution model 만 바꿔 같은 strategy/run/data 결과를 비교할 수 있다.

## BT-006 Bayesian Optimizer

- 상태: `todo`
- 목표: `search.mode: bayesian` 을 실제 optimizer 로 연결한다.
- 현재 상태: `bayesian` 은 확장 mode 로 문서화되어 있고, 기본 optimizer registry 에는
  `bounded`, `random` 만 들어 있다.
- 주요 파일:
  - `packages/backtest/evaluation.go`
  - `packages/backtest/evaluation_test.go`
  - `docs/features/backtester/evaluation/README.md`
- 완료 조건:
  - optimizer interface 를 통해 Bayesian 구현을 주입할 수 있다.
  - seed 와 tie-breaker 로 재현 가능한 parameter proposal 을 만든다.
  - optimizer 미등록 시 현재처럼 명시적으로 실패한다.
- 비범위:
  - heavy external dependency 를 public API 에 노출하는 변경.

## BT-007 Core Reporter Contract

- 상태: `todo`
- 목표: CLI handler 의 view model 과 별개로 `packages/backtest` 내부의 reporter
  기록 계약을 명확히 한다.
- 현재 상태: handler 쪽에는 `summary`, `metrics`, `orders`, `fills`, `trades`,
  `positions`, `equity`, `universe`, `events`, `evaluation` view 가 있다. core
  package 에는 별도 `Reporter` public layer 가 아직 선명하지 않다.
- 주요 파일:
  - `packages/backtest/engine.go`
  - `app/handler/backtest.go`
  - `cli/backtest.go`
  - `docs/architectures/packages/backtest/layers/reporter/README.md`
- 완료 조건:
  - core result 와 human/machine view model 의 경계가 문서와 코드에서 일치한다.
  - 큰 universe explain/debug payload 가 기본 human output 을 망치지 않는다.
  - renderer 세부사항은 `packages/backtest` 로 들어오지 않는다.

## BT-008 Documentation Sync

- 상태: `todo`
- 목표: 최종 목표 문서의 "현재 제한" 문단을 현재 HEAD 구현 상태와 맞춘다.
- 현재 상태: `final-implementation-goal.md` 에는 `1d only`, `next_open only`,
  single run 저장 제한 등 현재 코드와 일부 어긋나는 표현이 남아 있다.
- 주요 파일:
  - `docs/architectures/packages/backtest/final-implementation-goal.md`
  - `docs/architectures/packages/backtest/timeframes/README.md`
  - `docs/architectures/packages/backtest/execution-spec/README.md`
  - `docs/features/backtester/evaluation/README.md`
- 완료 조건:
  - 문서가 현재 구현, 남은 작업, 최종 목표를 구분한다.
  - 현재 가능한 CLI 예시와 아직 목표인 예시를 섞지 않는다.
  - 구현 변경 없이 문서만 고치는 경우에도 `git diff --check` 를 통과한다.

## Update Rules

- 새 작업은 `BT-###` 형식으로 추가한다.
- 구현을 시작하면 `Priority Index` 와 작업 카드의 상태를 함께 바꾼다.
- 하나의 작업 카드에는 목표, 현재 상태, 주요 파일, 완료 조건, 비범위를 둔다.
- 완료 후에는 검증 명령과 결과를 작업 카드에 짧게 남긴다.
- 아키텍처 계약이 바뀌면 이 문서가 아니라 `docs/architectures/packages/backtest`
  문서를 먼저 갱신하고, 이 문서는 추적용 요약만 남긴다.
