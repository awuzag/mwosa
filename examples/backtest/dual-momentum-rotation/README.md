# Dual Momentum Rotation Backtest

## What It Shows

상대 모멘텀과 절대 모멘텀을 함께 쓰는 월간 ETF rotation 예제다. 먼저 60일
수익률이 0보다 큰 ETF 만 남겨 절대 모멘텀 조건을 만들고, 그 안에서 60일
수익률이 높은 2개 ETF 를 고른다. 조건을 만족하는 ETF 가 없으면 universe 가
비어 있으므로 새 포지션을 만들지 않고, 기존 보유 종목은 `position_policy:
liquidate` 로 청산되어 현금 대기 상태가 된다.

현재 엔진은 조건 미충족 시 특정 대체자산으로 자동 전환하는 규칙까지는 직접
표현하지 않는다. 그래서 이 예제는 dual momentum 의 방어 구간을 채권/현금성 ETF
전환이 아니라 현금 대기형으로 둔다.

## Engine Features

- `universe.schedule.frequency: monthly`
- `transform.window_metrics` 로 20일/60일 수익률과 거래대금 평균 계산
- `filter.field` 로 60일 절대 모멘텀 양수 후보만 유지
- `rank.by_field` 로 상대 모멘텀 상위 2개 선별
- `position_policy: liquidate`
- `order_type: rebalance`, `target_weight_changed`

## Files

- `dual-momentum-rotation.yaml`: KRX ETF dual momentum 현금 대기형 예제.
- `summary.csv`: 저장된 run 식별자, 기간, 최종 equity.
- `metrics.csv`: 수익률, MDD, 회전율, 거래 수 같은 성과 지표.
- `trades.csv`: 실제 체결된 매수/매도 거래 내역.
- `events.csv`: order intent, 비용, 미체결 같은 실행 이벤트.

## Commands

```bash
mwosa validate backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml -o json
mwosa run backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml --view summary -o table
mwosa run backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml --view metrics -o csv
mwosa run backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml --view trades -o csv
mwosa run backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml --view events -o table
mwosa run backtest examples/backtest/dual-momentum-rotation/dual-momentum-rotation.yaml --view universe -o json
```
