# Evaluation Walk-Forward Backtest

## What It Shows

3년 train 구간에서 가장 좋은 파라미터를 고르고 다음 1년 test 구간에 적용하는
walk-forward evaluation 예제다. test 결과를 다시 train selection 에 쓰지 않도록
train/test 구간을 분리해 out-of-sample degradation 을 확인한다.

## Files

- `evaluation-walk-forward.yaml`: walk-forward period, parameter grid, selection
  objective 를 포함한 evaluation 예제.

## Commands

```bash
mwosa validate evaluation examples/backtest/evaluation-walk-forward/evaluation-walk-forward.yaml -o json
mwosa run evaluation examples/backtest/evaluation-walk-forward/evaluation-walk-forward.yaml --parallelism 4 -o table
mwosa inspect evaluation sma-cross-walk-forward --view summary -o table
mwosa inspect evaluation sma-cross-walk-forward --view walk_forward -o table
mwosa inspect evaluation sma-cross-walk-forward --view robustness -o json
mwosa rank evaluation sma-cross-walk-forward --objective calmar -o table
```

