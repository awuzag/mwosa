# Evaluation Grid Backtest

## What It Shows

하나의 SMA cross 전략을 연도별 기간과 파라미터 조합으로 반복 검증하는
evaluation 예제다. SMA window 와 `risk.max_positions` 조합을 grid 로 펼친 뒤
research metric preset 과 Calmar objective 로 ranking 한다.

## Files

- `evaluation-grid.yaml`: `Strategy`, `BacktestRun`, `Evaluation` 세 document 를
  포함한 grid evaluation 예제.

## Commands

```bash
mwosa validate evaluation examples/backtest/evaluation-grid/evaluation-grid.yaml -o json
mwosa run evaluation examples/backtest/evaluation-grid/evaluation-grid.yaml --parallelism 4 -o table
mwosa list evaluations -o table
mwosa inspect evaluation sma-cross-robustness --view summary -o table
mwosa inspect evaluation sma-cross-robustness --view cases -o csv
mwosa inspect evaluation sma-cross-robustness --view regime -o table
mwosa inspect evaluation sma-cross-robustness --view robustness -o json
mwosa compare evaluation sma-cross-robustness -o table
mwosa rank evaluation sma-cross-robustness --objective calmar -o table
```

