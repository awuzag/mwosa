# Backtester Final Implementation Goal

## 목적

이 문서는 `mwosa` 백테스터의 최종 구현 목표를 정리한다. 현재 구현 범위나
1차 spike 를 줄이는 문서가 아니라, 앞으로 구현을 이어갈 때 흔들리지 않아야 할
아키텍처 계약과 완료 조건을 정의한다.

최종 백테스터는 단일 종목 수익률 계산 helper 가 아니다. `packages/backtest` 안의
결정론적 portfolio-level simulation engine 이며, 같은 입력, 같은 strategy
version, 같은 data fingerprint 라면 같은 result hash 를 재현해야 한다.

`packages/backtest` 는 다음을 몰라야 한다.

- storage 구현과 SQLite schema
- YAML 파일 경로와 파일 입출력
- Cobra command, flag, stdin 처리
- provider API 와 provider-native 응답
- table, JSON, CSV, NDJSON 렌더링

엔진의 public contract 는 Go type 과 interface 다. YAML, 저장소, CLI, renderer 는
service, storage, cli, presentation 계층에서 엔진 계약에 맞게 변환한다.

## 현재 구현 표면

현재 코드에는 다음 축이 이미 있다.

- `packages/backtest`: `StrategySpec`, `BacktestRunSpec`, `EvaluationSpec`,
  streaming `BarFrame`, `Engine.Run`, rule/indicator/metric registry, portfolio,
  `next_open` execution, result hash.
- `packages/universe`: candidate field 와 pipeline 기반 universe core.
- `service/backtest`: YAML multi-document loader, canonical daily bar streaming
  feed adapter, universe resolution, evaluation 병렬 실행, saved strategy/evaluation
  조립.
- `storage/backtest`: saved backtest strategy version, experiment, case, result,
  metric summary, walk-forward step 저장.
- `cli/backtest.go`: verb-first command surface. `validate/run backtest`,
  `validate/run evaluation`, `list/inspect/compare/rank evaluation`,
  `update/list/inspect/delete backtest strategy`.

현재 제한도 명확하다.

- `BacktestRunSpec.data.timeframe` 은 `1d` 만 허용한다.
- execution fill 은 `next_open` 만 허용한다.
- order intent 는 내부 타입이고 order/fill/event ledger 가 최종 view model 로
  분리되어 있지 않다.
- sizing/risk 는 percent-of-equity, max positions, max symbol weight 중심이다.
- indicator/rule registry 는 확장 가능한 형태지만 실제 등록 범위는 제한적이다.
- single run 저장은 evaluation 저장 모델만큼 독립적인 축으로 정리되지 않았다.

## 문서 위치 결정

이 문서는 `docs/architectures/packages/backtest/` 아래에 둔다. 이유는 최종 목표의
중심이 기능 목록이 아니라 engine boundary, type contract, deterministic
simulation loop 이기 때문이다.

`docs/features/backtester/` 는 사용자-facing workflow, CLI 사용 흐름, roadmap 을
설명할 때 사용한다. evaluation 처럼 별도 workflow 가 커지는 문서는 feature 문서로
유지하되, core 계약은 이 architecture 문서를 참조한다.

## 최종 엔진 목표

엔진은 하나의 simulation clock 을 기준으로 시장 데이터, 전략 판단, 주문 의도,
체결, 비용, 포트폴리오 변경, 리포트를 순서대로 처리한다.

최종 loop 의 책임은 다음 순서로 고정한다.

```text
SimulationClock
  -> MarketDataFrame
  -> UniverseSnapshot
  -> IndicatorState
  -> StrategyRuleEvaluation
  -> OrderIntent
  -> PositionSizing
  -> RiskReview
  -> ExecutionModel
  -> Fill
  -> PortfolioMutation
  -> EventLog
  -> ReporterSnapshot
```

레이어별 책임은 다음과 같다.

| 영역 | 최종 책임 |
| --- | --- |
| `SimulationClock` | frame 시간을 단조 증가로 진행하고, calendar/session/warmup 규칙을 적용한다. |
| `EventLog` | signal, order intent, risk decision, fill, rejection, data issue, portfolio mutation 을 재현 가능한 순서로 남긴다. |
| `OrderIntent` | 전략과 리스크 검토 전의 주문 의도를 표현한다. side, order type, quantity/amount target, reason, tags, time-in-force 를 가진다. |
| `ExecutionModel` | order intent 를 fill 또는 non-fill event 로 바꾼다. 가격, 수량, 비용, 유동성 제약, 체결 실패 사유를 계산한다. |
| `Portfolio` | fill/effect 만 적용해 cash, positions, realized/unrealized PnL, exposure, equity 를 갱신한다. 전략이 직접 변경하지 않는다. |
| `Reporter` | 엔진 내부 result 를 사람이 보는 view model 과 기계가 읽는 structured result 로 나눌 수 있게 기록한다. 렌더링은 하지 않는다. |

단일 종목, 멀티 종목, 동적 universe, 리밸런싱은 별도 엔진으로 나누지 않는다.
모두 같은 loop 의 특수 케이스다.

- 단일 종목은 universe size 가 1인 portfolio simulation 이다.
- 멀티 종목은 같은 clock 안에서 종목별 signal 을 만들고 portfolio global zone 에서
  sizing, risk, execution 순서를 결정한다.
- 동적 universe 는 clock 또는 schedule 에 따라 `UniverseSnapshot` 을 갱신한다.
  universe 에서 빠진 보유 포지션은 `position_policy` 로 유지, 청산, 감축 중 하나를
  명시한다.
- 리밸런싱은 entry/exit 의 예외가 아니라 target allocation 을 만드는 strategy
  rule 과 order intent 생성 방식 중 하나다.

완성 조건:

- 같은 canonical input, strategy version, run spec, data fingerprint,
  engine version, registry version 으로 반복 실행하면 `result_hash` 가 같다.
- 현재 core result 는 `runtime.engine_version`,
  `runtime.indicator_registry_version`, `runtime.metric_registry_version` 을
  기록하고, 이 runtime metadata 를 `result_hash` 계산 payload 에 포함한다.
- single-symbol, multi-symbol, dynamic universe, rebalance fixture 가 같은
  `Engine.Run` 경로를 사용한다.
- engine loop 에서 portfolio mutation 은 fill/effect 적용 경로 하나로만 일어난다.
- event log 만 읽어도 주문 의도, 승인/거절, 체결, 포트폴리오 변경 순서를 설명할
  수 있다.
- `packages/backtest` 테스트에서 storage/YAML/Cobra/provider/renderer import 가
  없어야 한다.

## 타임프레임 최종 목표

`BarFrame` 은 `time.Time` 기반 계약을 유지한다. 다만 최종적으로 frame 은
timeframe 을 명시한 market data event 를 담을 수 있어야 한다.

지원 방향:

| timeframe | 의미 |
| --- | --- |
| `1m` | 1분봉. intraday execution 과 짧은 진입 rule 의 기본 단위. |
| `5m` | 5분봉. 장중 추세/진입 판단에 사용. |
| `15m` | 15분봉. intraday noise 를 줄인 판단 단위. |
| `30m` | 30분봉. 세션 내 중기 판단 단위. |
| `1h` | 1시간봉. intraday swing 판단 단위. |
| `1d` | 일봉. 현재 구현된 기본 단위. |
| `1w` | 주봉. 장기 trend/filter, 리밸런싱 판단. |
| `1mo` | 월봉. 장기 regime, allocation 판단. |
| `custom` | calendar, session, bar builder, resample rule 을 명시한 사용자 정의 frame. |

최종 계약:

- `DataSpec` 은 단일 `timeframe` 뿐 아니라 named feed 를 표현할 수 있어야 한다.
- `BarFrame.Time` 은 engine clock 이며, 각 feed 는 그 시점에 사용 가능한 마지막
  완료 bar 만 context 에 제공한다.
- `BarFrame` 또는 그 후속 타입은 symbol, market, optional security type,
  timeframe, OHLCV, traded amount, session, data status 를 표현한다.
- resample 은 원천 데이터보다 더 큰 timeframe 으로만 가능하다. 예를 들어 1분봉에서
  5분봉/1시간봉/1일봉 생성은 가능하지만, 일봉에서 5분봉을 만들지는 않는다.
- calendar 는 market/session/holiday/half-day 를 가진다. `custom` 은 calendar 와
  bar boundary 를 명시해야 한다.
- warmup 은 indicator lookback, timeframe, resample dependency 를 모두 반영한다.
- multi-timeframe rule 은 future leakage 를 막기 위해 하위 timeframe clock 에서
  아직 닫히지 않은 상위 timeframe bar 를 읽지 않는다.
- benchmark 와 universe filter 도 같은 시간 접근 규칙을 따른다.

future leakage 방지 규칙:

- clock 은 단조 증가한다.
- rule 은 현재 clock 에서 닫힌 bar 와 이전 state 만 읽는다.
- `same_close` 처럼 같은 bar 가격을 쓰는 체결 모델은 spec 에 명시되어야 하고,
  그 위험이 result metadata 에 남아야 한다.
- resampled bar 는 source bar 의 마지막 시간이 닫힌 뒤에만 노출된다.
- warmup 이전 값은 trade signal 로 이어지지 않는다.
- walk-forward 는 train 구간의 선택 결과만 다음 test 구간에 전달하고, test 결과를
  다시 train selection 에 쓰지 않는다.

완성 조건:

- `1d` fixture 와 intraday fixture 가 같은 feed/clock 추상화를 사용한다.
- `daily trend + 5m entry` 같은 multi-timeframe 전략이 닫힌 bar 규칙을 어기면
  테스트에서 실패한다.
- resample 결과는 calendar boundary, missing bar, no-trade bar 를 설명 가능한
  event 또는 data issue 로 남긴다.
- timeframe 을 바꿨을 때 result metadata 에 source timeframe, execution
  timeframe, resample policy, warmup policy 가 남는다.

## 체결, 슬리피지, 거래비용 최종 목표

최종 execution layer 는 order type, fill timing, liquidity cap, partial fill,
no-trade bar policy, slippage model, cost model 을 분리한다.

지원할 order type:

- `market`
- `limit`
- `stop`
- `stop_limit`
- `trailing_stop`
- `rebalance_target`
- `close_position`

지원할 fill timing 또는 price policy:

- `next_open`: 현재 bar 에서 만든 의도를 다음 사용 가능한 bar open 에 체결한다.
- `same_close`: 현재 bar close 체결을 허용하되 look-ahead 위험을 metadata 에 남긴다.
- `next_close`: 다음 bar close 기준 체결.
- `intrabar_ohlc`: 한 bar 안에서 high/low/open/close 경로 가정을 명시한다.
- `limit`: limit price 조건이 충족될 때 체결한다.
- `stop`: stop trigger 뒤 market 또는 지정 execution policy 로 체결한다.
- `stop_limit`: trigger 와 limit 조건을 모두 만족해야 한다.
- `trailing_stop`: position state 에 따라 stop price 를 갱신한다.

체결 정책:

- no-trade bar 는 성공 체결로 만들지 않는다. `next_open` pending order 는 보류하거나
  spec 에 정한 만료 정책으로 취소한다.
- invalid OHLCV 는 hard failure 또는 data issue policy 에 따라 명시적으로 처리한다.
- liquidity cap 은 bar volume/traded amount 의 일정 비율, fixed notional cap,
  symbol-specific cap 을 지원한다.
- partial fill 은 남은 수량을 pending 으로 유지하거나 취소하는 policy 를 가진다.
- order priority 는 clock 안에서 deterministic sort key 를 가진다.
- short, leverage, margin 은 portfolio account model 이 열릴 때 별도 account/risk
  계약으로 추가한다.

슬리피지 모델:

- `none`
- `bps`
- `fixed_amount`
- `spread_bps`
- `volume_share`
- `volatility_adjusted`
- `custom`

거래비용 모델:

- commission: bps, fixed per order, fixed per share/unit, tiered.
- tax: market/security type/account 에 따른 bps 또는 rule table.
- fees: exchange/regulatory/settlement fee.
- currency/fx: multi-currency portfolio 를 열 때 환율 source 와 conversion timing 을
  명시한다.

완성 조건:

- 같은 order intent 가 어떤 모델에서 fill, partial fill, reject, defer 되는지
  event 로 설명된다.
- no-trade bar, invalid bar, insufficient cash, liquidity cap, limit miss,
  stop trigger, trailing stop update 가 각각 테스트 fixture 를 가진다.
- cost/slippage 가 trade, cash, equity, metrics 에 일관되게 반영된다.
- execution model 변경만으로 같은 strategy/run/data 를 비교할 수 있고, result
  metadata 에 체결 가정 차이가 남는다.

## 전략 표현력 최종 목표

전략은 Go 코드 분기문이 아니라 data-oriented spec 으로 표현한다. `StrategySpec`,
`BacktestRunSpec`, `EvaluationSpec` 분리는 유지한다.

registry 는 세 축으로 나눈다.

| registry | 책임 |
| --- | --- |
| indicator registry | 시계열 또는 횡단면 값을 계산한다. `id`, input, params, output, warmup, timeframe dependency 를 검증한다. |
| expression registry | price, indicator, portfolio state, position state, universe candidate field, benchmark, literal, arithmetic, lag 같은 값을 읽고 조합한다. |
| rule registry | expression 결과를 비교하거나 event/order intent 로 변환한다. logical, comparison, cross, entry, exit, sizing, risk, rebalance, stop rule 을 검증한다. |

표현식 목록:

| 영역 | 예시 |
| --- | --- |
| trend | `sma`, `ema`, `wma`, `kama`, `hma`, `adx`, `aroon`, `psar`, moving-average slope |
| momentum | `rsi`, `macd`, `stochastic`, `roc`, `momentum`, `relative_strength`, breakout |
| volatility | `atr`, `natr`, `bollinger`, `keltner`, `donchian`, `rolling_std`, `variance`, realized volatility |
| volume | `volume`, `amount`, `vwap`, `obv`, `mfi`, volume spike, liquidity percentile |
| statistical | `return`, `log_return`, `zscore`, `correlation`, `beta`, percentile, drawdown |
| cross-sectional | rank, percentile rank, top-N, spread, ratio, sector/asset-class relative score |
| portfolio state | cash, equity, exposure, drawdown, turnover, current weight, available buying power |
| position state | quantity, avg price, unrealized PnL, holding period, highest since entry, stop price |
| universe candidate | candidate fields, tags, screen score, source metadata, regime tags |

전략 spec 이 표현해야 할 rule 영역:

- entry: 신규 진입 또는 추가 진입 signal.
- exit: 보유 포지션 청산/감축 signal.
- sizing: percent of equity, fixed notional, fixed quantity, volatility target,
  risk-per-trade, target weight.
- risk: max positions, max symbol/sector/asset exposure, max turnover, drawdown stop,
  daily loss limit, cooldown, duplicate entry block.
- rebalance: target allocation, drift band, schedule, cash buffer, min trade size.
- stop: stop loss, take profit, trailing stop, time stop, volatility stop.

DSL 방향:

- 기존 `StrategySpec`, `BacktestRunSpec`, `EvaluationSpec` 분리는 유지한다.
- Go struct 가 canonical model 이며 JSON/YAML round-trip 가능해야 한다.
- v1 과 호환 가능한 확장은 점진 적용한다.
- multi-timeframe feed, order type, state expression, rebalance/stop DSL 처럼 shape 가
  크게 바뀌는 부분은 `schema_version: 2` 로 새 형태를 정리한다.
- 저장소에는 YAML text 가 아니라 canonical spec JSON, spec hash, schema version,
  registry version 을 저장한다.

완성 조건:

- registry metadata 만으로 unknown indicator/expression/rule 과 잘못된 params 를
  compile 단계에서 실패시킨다.
- entry/exit/sizing/risk/rebalance/stop 이 모두 데이터 spec 으로 표현된다.
- strategy spec 은 실행 기간, provider, SQLite, CLI output format 을 갖지 않는다.
- saved strategy 는 canonical structured state 로 저장되고, YAML/JSON 은 입출력
  형식일 뿐이다.
- 복잡한 전략도 event log 에 signal reason, rule id, matched values 를 남긴다.

## 실험과 평가 최종 목표

`EvaluationSpec` 은 `StrategySpec + BacktestRunSpec` 위의 반복 검증 계층이다.
단일 실행과 evaluation 은 같은 엔진 결과를 공유하되 저장과 조회 모델은 독립적인
축으로 확장한다.

지원할 실행 단위:

| 단위 | 의미 |
| --- | --- |
| single run | 하나의 strategy/run/data fingerprint 로 한 번 실행한다. |
| saved run | single run result, spec hash, data fingerprint, engine/registry version, result hash 를 저장한다. |
| evaluation | 여러 period/parameter/regime case 를 펼쳐 반복 실행한다. |
| walk-forward | train 구간에서 parameter 를 고르고 다음 test 구간에 적용한다. |
| regime split | bull/bear/sideways/high_vol/low_vol 또는 사용자 regime 별 case 를 만든다. |
| parameter search | grid, random, bounded search 를 지원하되 deterministic seed 를 metadata 에 남긴다. Bayesian search 는 optimizer registry 로 확장할 수 있고, 기본 registry 에 없으면 명시적으로 실패한다. |
| robustness check | 기간 이동, 비용/슬리피지 변화, liquidity cap 변화, universe 변화에 대한 민감도를 본다. |

metric/constraint/objective 계약:

- metric 은 registry 로 관리한다.
- core metric 과 research metric preset 을 구분한다.
- benchmark metric 은 benchmark data fingerprint 가 있을 때만 허용한다.
- constraint 는 pass/fail 과 actual/limit/message 를 저장한다.
- objective 는 ranking 가능한 metric 또는 composite score 를 참조한다.
- composite score 는 normalization 방식, weight, missing metric policy 를 명시한다.

저장 계약:

- single run 은 evaluation case 의 부속물이 아니라 독립 `RunRecord` 로 저장한다.
- evaluation case 는 여러 `RunRecord` 를 참조할 수 있다.
- `backtest_results` 는 raw result JSON 만이 아니라 result hash, engine version,
  strategy spec hash, run spec hash, data fingerprint 를 함께 가진다.
- metric summary, order/fill/trade/equity/event view 는 필요한 경우 별도 materialized
  table 로 분리할 수 있다.
- 같은 run 이 이미 존재하면 hash 기준으로 재사용하거나 duplicate run 으로 명시한다.

완성 조건:

- single run 저장/조회와 evaluation 저장/조회가 같은 result identity 를 공유한다.
- walk-forward 는 out-of-sample test 결과와 train selection 결과를 분리해 저장한다.
- parameter search 는 같은 seed/spec/data 에서 같은 case order 와 ranking 을 만든다.
- benchmark comparison 은 benchmark 누락, 기간 불일치, 데이터 fingerprint 차이를
  명시적으로 드러낸다.
- evaluation table 출력은 사람용 summary 이고, JSON/NDJSON 은 case/result/metric
  구조를 안정적으로 제공한다.

## 결과 출력과 view model 최종 목표

엔진의 raw result 하나를 그대로 CLI 에 내보내지 않는다. service 또는 presentation
계층은 목적별 view model 을 만든다. renderer 는 view model 을 `json`, `csv`,
`table`, 필요한 경우 `ndjson` 으로 출력한다.

최종 view model:

| view | 내용 |
| --- | --- |
| `summary` | run name, strategy, period, universe size, final equity, total return, MDD, result hash, warnings. |
| `metrics` | metric id/value/unit/category, benchmark 여부, selected preset. |
| `orders` | order intent, target amount/quantity, order type, status, reason, tags. |
| `fills` | fill/non-fill event, price, quantity, cost, slippage, liquidity cap, execution policy. |
| `trades` | entry/exit 또는 round-trip trade, realized PnL, return, holding period, reason. |
| `positions` | clock별 또는 final position, quantity, avg price, market value, weight, PnL. |
| `equity` | equity curve, cash, positions value, drawdown, exposure. |
| `universe` | selected symbols/candidates, schedule, source, step summaries, explain snapshots. |
| `events` | data issue, signal, risk, execution, portfolio mutation, warning/debug events. |
| `evaluation` | cases, ranking, constraints, walk-forward steps, objective, regime tags. |

출력 원칙:

- `json` 은 stable field name 과 nested structure 를 유지한다.
- `csv` 는 flat row view 에만 허용한다. nested payload 는 명시적으로 view 를 고른다.
- `table` 은 사람이 빠르게 판단할 핵심 column 만 보여준다.
- `ndjson` 은 큰 events, orders, fills, equity, universe explain 같은 streaming-friendly
  출력에 사용한다.
- 큰 universe explain 은 기본 summary 에서 숨기고, raw/debug view 에서만 전체
  snapshots/decisions 를 보여준다.
- stdout 은 결과, stderr 는 diagnostics/progress/log 를 유지한다.

완성 조건:

- `mwosa run backtest ... --view summary -o table` 과
  `mwosa run backtest ... --view fills -o ndjson` 같은 목적별 출력이 가능하다.
- table 과 JSON 은 같은 raw result 에서 파생되지만 서로 다른 view model 을 사용한다.
- CSV 출력은 orders/fills/trades/positions/equity/metrics 처럼 flat schema 가 있는
  view 에서만 성공한다.
- universe explain 과 event debug 는 기본 출력의 가독성을 해치지 않는다.
- 기계가 처리하는 output 은 jq/CSV/NDJSON pipeline 에서 필드가 예측 가능하다.

## 데이터 저장 결정

intraday 데이터 저장 구조는 이번 최종 목표 문서에서 schema 까지 확정하지 않는다.
대신 engine contract 는 intraday 와 custom timeframe 을 받을 수 있게 열어둔다.

이유:

- `packages/backtest` 는 storage schema 를 몰라야 한다.
- intraday schema 는 provider, calendar, session, 압축/파티셔닝, 조회 성능과 함께
  결정해야 한다.
- 지금 필요한 결정은 `BarFrame`/market data event/timeframe/calendar/warmup 계약이지
  SQLite table column 이 아니다.

따라서 intraday 저장은 별도 architecture 문서 또는 ADR 에서 다룬다. 그 문서는
`service/backtest` feed adapter 와 `storage/*` schema 를 대상으로 하고, 이 문서의
engine contract 를 깨지 않아야 한다.

## 남은 결정 포인트

- `schema_version: 2` 를 언제 도입할지: multi-timeframe/order DSL 을 넣는 시점에
  올리는 방향이 유력하다.
- `RunRecord` 저장 모델을 기존 `backtest_results` 중심으로 확장할지, single run
  전용 table 을 추가할지 정해야 한다.
- runtime metadata 는 single run, evaluation case, walk-forward step,
  backtest result row 에 독립 컬럼으로 materialize 한다. core result JSON 에도
  engine/indicator registry/metric registry version 과 data fingerprint 가
  포함된다.
- money type 을 계속 `float64` 로 둘지, execution/portfolio 금액부터 decimal 로
  바꿀지 정해야 한다.
- intraday storage 는 SQLite v2 확장, 별도 table, DuckDB 보조 저장소 중 어느
  방향으로 갈지 별도 문서에서 결정한다.
- multi-strategy single portfolio simulation 을 언제 열지 정해야 한다.
- live/paper execution 과 shared rule/risk package 를 `mwosa` 안에 둘지, `eolmasa`
  또는 별도 공용 패키지와 나눌지 다시 확인해야 한다.

## 구현 전환 조건

이 문서를 구현 목표로 전환할 때는 영역별로 작은 slice 를 고른다. 다만 최종 목표를
MVP 로 축소하지 않는다. 각 slice 는 최종 계약을 향해 가는 관찰 가능한 변경이어야
한다.

구현 세션의 기본 완료 조건:

- 실패 테스트를 먼저 추가한다.
- Go 코드를 수정했다면 `gofmt` 를 실행한다.
- 저장소 루트에서 `go test ./...` 를 실행한다.
- provider client 모듈을 수정했다면 해당 모듈 안에서 `go test ./...` 와
  `go mod verify` 를 실행한다.
- CLI surface 를 바꿨다면 실제 CLI smoke 를 실행하고 stdout/stderr 경계를 확인한다.
- 문서와 코드의 책임 경계가 어긋나면 문서를 같이 갱신한다.
