# Relative Strength Rotation Backtest

## What It Shows

상승장에서 최근 수익률이 강한 ETF 로 자본을 주간 회전시키는 예제다. universe 는
최근 20일 수익률이 양수이고 20일 평균 거래대금이 충분한 ETF 만 남긴 뒤, 유동성
상위 30개 안에서 20일/60일 수익률 가중 점수가 높은 3개를 고른다.

전략 자체는 `order_type: rebalance` 와 `target_weight_changed` 를 사용해 선택된
ETF 를 약 1/3 비중으로 맞춘다. 다음 주 universe 에서 빠진 종목은
`position_policy: liquidate` 로 청산한다. 그래서 상승장이 이어질 때는 강한
종목으로 자본이 계속 이동하고, 조건을 만족하는 후보가 줄어들면 현금 비중이
자연스럽게 늘어난다.

## Engine Features

- `universe.schedule.frequency: weekly`
- `source.daily_bars`, `transform.window_metrics`, `filter.field`
- `rank.by_field` 로 유동성 상위 후보 선별
- `rank.weighted` 로 최근 상대강도 상위 후보 선별
- `position_policy: liquidate`
- `order_type: rebalance`, `time_in_force: cancel_on_rebalance`

## Files

- `relative-strength-rotation.yaml`: KRX ETF 상대강도 rotation 전략과 실행 조건.
- `summary.csv`: 저장된 run 식별자, 기간, 최종 equity.
- `metrics.csv`: 수익률, MDD, 회전율, 거래 수 같은 성과 지표.
- `trades.csv`: 실제 체결된 매수/매도 거래 내역.
- `events.csv`: order intent, 비용, 미체결 같은 실행 이벤트.

## Commands

```bash
mwosa validate backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml -o json
mwosa run backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml --view summary -o table
mwosa run backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml --view metrics -o csv
mwosa run backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml --view trades -o csv
mwosa run backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml --view events -o table
mwosa run backtest examples/backtest/relative-strength-rotation/relative-strength-rotation.yaml --view universe -o json
```
