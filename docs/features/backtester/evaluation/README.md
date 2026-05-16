# Backtester Evaluation

## 목적

`Evaluation`은 하나의 전략을 한 번 실행하는 기능이 아니라, 같은 전략 가설을
여러 조건에서 반복 검증하는 리서치 단위다. `Strategy`는 매매 규칙을, `BacktestRun`
은 단일 실행 조건을, `Evaluation`은 여러 실행 case를 만드는 검증 계획을 맡는다.

백테스터는 특정 상품군 전용 도구가 아니다. `stock`, `etf`, `etn`, `elw` 같은
값은 universe candidate field 또는 `filter.security_type` 조건으로 다룬다. 실행
전체를 하나의 `security_type`으로 잠그지 않는다.

## YAML 구조

하나의 파일에는 보통 세 document를 둔다.

```yaml
kind: Strategy
schema_version: 1
name: sma-cross
---
kind: BacktestRun
schema_version: 1
name: sma-cross-krx
strategy:
  name: sma-cross
---
kind: Evaluation
schema_version: 1
name: sma-cross-robustness
strategy:
  name: sma-cross
base_run:
  ref: sma-cross-krx
periods:
  mode: yearly
  from: 2020-01-01
  to: 2025-12-31
parameters:
  indicators.trend.params.window: [20, 60, 120]
metrics:
  preset: research
ranking:
  objective: calmar
  order: desc
execution:
  parallelism: 4
```

`Evaluation`은 `Strategy` 문법을 뒤집지 않는다. 기간과 파라미터 조합을 펼쳐서
여러 `BacktestRun` case를 만들고, 각 case를 같은 엔진으로 실행한다.
`parameters`와 `search`는 동시에 쓰지 않는다. 하나의 evaluation은 명시 grid 또는
search plan 중 하나만 선택한다.

## 지원 범위

- `periods.mode: explicit`: 명시한 기간 목록을 실행한다.
- `periods.mode: yearly`: `from`부터 `to`까지 연도별 case를 만든다.
- `periods.mode: rolling`: `window`와 `step`으로 rolling 기간을 만든다.
- `periods.mode: expanding`: 고정 시작일부터 `window`로 첫 기간을 만들고 `step`마다 종료일을 확장한다.
- `periods.mode: walk_forward`: 일반 case grid를 만들지 않고 `walk_forward` train/test step만 만든다.
- `parameters`: dot path 기반 grid search를 만든다.
- `search.mode: bounded`: `min/max/step` 또는 `values`를 dot path별 후보군으로 펼친다.
- `search.mode: random`: deterministic `seed`와 `samples`로 후보군에서 재현 가능한 random search를 만든다.
- `search.mode: bayesian`: Bayesian optimizer 를 위한 확장 mode 다. core 는
  `EvaluationSearchOptimizerRegistry` 로 optimizer 를 주입받을 수 있고, 기본
  registry 에 optimizer 가 없으면 명시적으로 실패한다.
- `metrics.preset: research`: CAGR, MDD, volatility, Sharpe, Calmar, turnover, trade count, win rate, profit factor, exposure, unfilled count, data issue count를 포함한다. benchmark가 설정된 run에서는 benchmark return, excess return, relative drawdown, monthly win rate, alpha, beta도 함께 포함한다.
- `constraints`: `max_drawdown_lte`, `min_cagr_gte`, `max_turnover_lte`, `min_trade_count_gte`, `max_exposure_lte`, `max_unfilled_count_lte`, `max_data_issue_count_lte`를 평가한다.
- `ranking`: 통과한 case만 objective 기준으로 정렬한다.
- `ranking.objective: weighted_score`: `ranking.weights`에 지정한 metric 가중합으로 composite objective를 만든다.
- `regime split`: 각 case의 bull/bear/sideways, high_vol/low_vol tag를 evaluation-level split으로 집계한다. `regime.benchmark`를 지정하면 base run에 benchmark가 없어도 각 case run에 benchmark를 주입해 benchmark-driven tag와 benchmark metrics를 함께 계산한다. `regime.return_threshold`와 `regime.volatility_threshold`로 bull/bear/sideways 및 high/low volatility 기준을 조정할 수 있다.
- `robustness`: parameter sensitivity, 기간별 top-N parameter stability, walk-forward out-of-sample degradation을 집계한다.
- `execution.parallelism`: case 실행 worker 수를 지정한다. 없으면 1이다.
- `walk_forward`: train 구간에서 best parameter를 고르고 다음 test 구간에 적용한다.

## CLI

```bash
mwosa validate evaluation examples/backtest/evaluation-grid/evaluation-grid.yaml -o json
mwosa run evaluation examples/backtest/evaluation-grid/evaluation-grid.yaml --parallelism 4 -o table
mwosa list evaluations -o table
mwosa inspect evaluation sma-cross-robustness -o json
mwosa inspect evaluation sma-cross-robustness --view regime -o table
mwosa inspect evaluation sma-cross-robustness --view robustness -o json
mwosa compare evaluation sma-cross-robustness -o table
mwosa rank evaluation sma-cross-robustness --objective calmar -o json
```

JSON 출력은 case, metrics, parameters, constraints, regime tags, result hash를
구조화해서 제공한다. table 출력은 사람이 빠르게 볼 수 있도록 rank, case, 기간,
통과 여부, objective, hash 중심으로 줄인다.
`--view regime`은 tag별 case 수, 통과 case 수, best case, 평균 objective와 평균
metric map을 분리해 제공한다.
`--view robustness`는 parameter sensitivity와 top-N stability를 보여주며,
walk-forward가 저장된 evaluation에서는 train objective 대비 test objective
degradation도 함께 제공한다.

병렬도 우선순위는 CLI `--parallelism` > YAML `execution.parallelism` > 기본값 1이다.
case 결과는 병렬 실행 후 spec 순서로 수집하고, ranking 은 objective 와 case id
tie-breaker 로 결정론을 유지한다. walk-forward 는 step 순서를 깨지 않으며, 각
step 안에서 train case만 병렬 실행한 뒤 선택된 parameter 로 test case를 실행한다.

## 저장 모델

실행된 evaluation은 SQLite에 저장된다.

- `backtest_experiments`: evaluation spec, spec hash, strategy hash, data range
- `backtest_experiment_cases`: case 기간, parameter JSON, regime tags, rank, result hash
- `backtest_results`: case별 전체 backtest result JSON
- `backtest_metric_summaries`: case별 metric key/value
- `backtest_walk_forward_steps`: train/test 기간, 선택된 parameter, test metric summary, out-of-sample result hash

같은 spec, 같은 데이터, 같은 전략 버전으로 실행하면 engine result hash가 재현
가능해야 한다.
