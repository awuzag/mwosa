# Strategy Optimization

## 목적

`mwosa optimize strategy` 는 YAML 기반 전략 스펙과 백테스터가 준비된 뒤,
전략의 세부 파라미터를 반복 검증해 실전 운용에 더 견딜 수 있는 값을 찾는
후속 피처다.

이 기능의 목표는 백테스트 수익률이 가장 높은 숫자 하나를 고르는 것이 아니다.
거래비용과 슬리피지를 반영한 뒤에도 최대 낙폭을 통제하고, 여러 기간과 주변
파라미터에서 성과가 유지되는 조합을 찾는 것이다.

## 배경

시스템 트레이딩 전략은 같은 규칙을 쓰더라도 세부 수치에 따라 완전히 다른
결과가 나온다.

예를 들어 20일 이동평균선 돌파 전략은 다음 질문을 가진다.

```text
종가가 20일 이동평균보다 몇 % 이상 높을 때 진입해야 할까?
보유 기간은 5일, 10일, 20일 중 무엇이 나을까?
최대 낙폭을 낮추려면 어떤 손절이나 리스크 제한이 필요할까?
```

단순히 전체 기간에서 최고 수익률을 낸 조합을 고르면 과최적화 위험이 크다.
따라서 `mwosa` 는 최고점 하나보다 다음 성질을 우선한다.

- 여러 walk-forward 구간에서 반복적으로 통과한다.
- 주변 파라미터도 비슷한 성과를 낸다.
- 거래비용과 슬리피지를 높여도 결과가 급격히 무너지지 않는다.
- MDD, 최장 손실 기간, 최악의 월 손실이 사용자의 제약 안에 들어온다.
- 결과를 JSON/CSV 로 재현 가능하게 남긴다.

## 전제 조건

이 피처는 백테스터와 전략 스펙 스키마가 먼저 안정된 뒤 진행한다.

필요한 선행 기능:

- YAML 기반 `StrategySpec` schema validation
- `BacktestRun` 실행 스펙
- canonical daily bar 기반 backtest 실행
- commission, tax, slippage 같은 `ExecutionModel`
- equity curve, trade ledger, drawdown, statistics report
- strategy/run 결과를 저장하거나 재현할 수 있는 실행 기록

`optimize strategy` 는 새 백테스트 엔진을 만들지 않는다. 이미 준비된
`packages/backtest` 엔진과 전략 스펙을 여러 파라미터 조합으로 반복 실행하는
연구 자동화 레이어다.

## 사용자 흐름

사용자는 먼저 파라미터 placeholder 를 가진 전략 파일을 작성한다.

```yaml
kind: Strategy
schema_version: 1
name: ma20-breakout

universe:
  symbols: ["069500", "102110"]

entry:
  all:
    - close_above_sma:
        window: ${ma_period}
        buffer_pct: ${entry_buffer_pct}

exit:
  any:
    - holding_days:
        value: ${hold_days}
    - close_below_sma:
        window: ${ma_period}

sizing:
  type: percent_of_equity
  value: 20
```

그 다음 여러 후보 값을 지정해 최적화를 실행한다.

```bash
mwosa optimize strategy strategies/ma20-breakout.yaml \
  --param ma_period=10,20,40,60 \
  --param entry_buffer_pct=0,0.5,1,1.5,2,3 \
  --param hold_days=5,10,20 \
  --cost-bps 15 \
  --slippage-bps 10 \
  --walk-forward train=720d,test=120d \
  --objective calmar \
  --constraint max_mdd_pct=10 \
  --constraint min_trades=30 \
  --select robust \
  -o json
```

## CLI 초안

```bash
mwosa optimize strategy <strategy-file> [flags]
```

핵심 flag:

| Flag | 의미 |
| --- | --- |
| `--param name=v1,v2,...` | 탐색할 파라미터 후보 값 |
| `--objective cagr|calmar|sharpe|sortino|min_mdd` | 최적화 목표 |
| `--constraint key=value` | 후보가 통과해야 하는 제약 조건 |
| `--walk-forward train=720d,test=120d` | rolling train/test 검증 |
| `--cost-bps` | 거래비용 bps |
| `--slippage-bps` | 슬리피지 bps |
| `--select best|robust` | 최고값 또는 안정 구간 선택 |
| `--top` | 상위 후보 출력 개수 |
| `-o json|csv|table` | 출력 형식 |

초기 MVP 는 grid search 만 지원한다. Bayesian optimization, genetic
algorithm, parallel distributed run 은 제외한다.

## 평가 기준

수익률만으로 후보를 고르지 않는다. 기본 리포트는 최소한 다음 지표를 포함한다.

| 지표 | 이유 |
| --- | --- |
| CAGR | 장기 복리 수익률 |
| total return | 전체 기간 수익 |
| max drawdown | 최악의 누적 손실 |
| Calmar ratio | MDD 대비 수익률 |
| Sharpe ratio | 변동성 대비 수익률 |
| Sortino ratio | 하방 변동성 대비 수익률 |
| trades | 표본 수 부족 방지 |
| win rate | 거래 승률 |
| profit factor | 손익 비율 |
| longest drawdown days | 손실 구간 지속 기간 |
| worst month return | 월간 최악 손실 |
| cash exposure | 현금 보유 비중 |

MDD 중심 사용자는 다음처럼 조건을 걸 수 있어야 한다.

```bash
mwosa optimize strategy strategies/ma20-breakout.yaml \
  --objective calmar \
  --constraint max_mdd_pct=10 \
  --constraint worst_month_pct=5 \
  --constraint min_trades=30
```

## Robust 선택

`--select robust` 는 백테스트 점수가 가장 높은 단일 조합을 바로 고르지 않는다.
대신 주변 파라미터가 함께 살아 있는지를 본다.

예를 들어 아래 결과는 건강한 후보로 본다.

```text
entry_buffer_pct=1.0  CAGR 18%, MDD -8%
entry_buffer_pct=1.5  CAGR 17%, MDD -9%
entry_buffer_pct=2.0  CAGR 16%, MDD -8.5%
```

반대로 아래 결과는 과최적화 의심 후보로 본다.

```text
entry_buffer_pct=1.37  CAGR 40%, MDD -7%
entry_buffer_pct=1.0   CAGR -3%
entry_buffer_pct=1.5   CAGR 2%
```

초기 구현의 robust score 는 단순하게 시작한다.

- 주변 후보 중 제약을 통과한 비율
- walk-forward test window 통과 비율
- 비용과 슬리피지 민감도 통과 여부
- 상위 후보 간 metric 분산

정교한 통계 검정은 MVP 범위에서 제외하되, 결과에는 사용한 후보 수와 선택
기준을 반드시 남긴다.

## 출력 초안

JSON 출력은 자동화와 후속 분석을 우선한다.

```json
{
  "strategy": "ma20-breakout",
  "objective": "calmar",
  "selection": "robust",
  "constraints": {
    "max_mdd_pct": 10,
    "min_trades": 30
  },
  "recommended": {
    "params": {
      "ma_period": 20,
      "entry_buffer_pct": 1.0,
      "hold_days": 10
    },
    "reason": "MDD constraint passed, neighboring parameters stayed stable, and Calmar ranked highest among robust candidates"
  },
  "metrics": {
    "cagr_pct": 17.8,
    "total_return_pct": 42.1,
    "max_drawdown_pct": -8.6,
    "calmar": 2.07,
    "sharpe": 1.21,
    "sortino": 1.58,
    "trades": 84,
    "longest_drawdown_days": 63
  },
  "robustness": {
    "candidate_count": 72,
    "walk_forward_windows_passed": 7,
    "walk_forward_windows_total": 9,
    "neighbor_pass_rate": 0.78,
    "cost_sensitivity_passed": true
  }
}
```

CSV 출력은 후보 조합을 사람이 비교하기 쉽게 flat row 로 둔다.

```text
ma_period,entry_buffer_pct,hold_days,cagr_pct,max_drawdown_pct,calmar,sharpe,trades,passed
20,1.0,10,17.8,-8.6,2.07,1.21,84,true
20,1.5,10,17.1,-9.0,1.90,1.15,79,true
```

## 저장과 재현

최적화 실행은 많은 후보를 시험하므로 실행 재현 정보가 중요하다.

저장해야 할 정보:

- strategy spec hash
- parameter grid
- objective
- constraints
- walk-forward 설정
- 비용과 슬리피지 설정
- 입력 데이터 범위와 snapshot 기준
- candidate count
- 추천 후보와 상위 후보
- 실패한 후보의 실패 사유 요약

초기에는 전체 candidate 결과를 JSON/CSV 파일로 내보내는 것부터 시작하고,
저장소 모델은 backtest run 저장 기능이 안정된 뒤 붙인다.

## 제외 범위

- 실거래 자동 주문
- 수익 보장 또는 종목 매수 추천
- 백테스터 엔진 재구현
- 모든 최적화 알고리즘 지원
- 머신러닝 기반 파라미터 탐색
- 분산 실행과 장기 job scheduler
- 외부 차트 UI

## 완료 기준

- 파라미터 grid 를 생성하고 모든 후보를 deterministic backtest 로 실행한다.
- 비용과 슬리피지를 모든 후보 평가에 동일하게 반영한다.
- `--objective` 와 `--constraint` 로 후보 선택 기준을 분리한다.
- walk-forward train/test window 결과를 출력한다.
- `--select robust` 가 단일 최고값보다 안정 구간을 우선할 수 있다.
- JSON 출력은 후속 자동화가 읽기 쉬운 구조를 가진다.
- CSV 출력은 상위 후보 비교에 충분한 flat schema 를 가진다.
- 실패한 후보는 성공처럼 숨기지 않고 invalid parameter, insufficient data,
  no trades 같은 사유를 남긴다.

## 열어둘 질문

- 파라미터 placeholder 문법은 `${name}` 으로 충분한가?
- `StrategySpec` 안에 optimization search space 를 둘 것인가, CLI flag 로만
  받을 것인가?
- walk-forward train/test window 는 calendar day 와 trading day 중 무엇을
  기준으로 표현할 것인가?
- robust neighbor 는 numeric parameter 에만 적용할 것인가?
- optimization run 을 backtest run 과 같은 저장소에 둘 것인가, 별도 run type 으로
  둘 것인가?
