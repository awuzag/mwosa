# Universe Pipeline Backtest

## What It Shows

월간 schedule 로 universe 를 다시 고르고, 최근 20일 수익률과 20일 평균 거래대금을
함께 사용해 유동성 있는 ETF 후보를 ranking 하는 예제다. universe 에서 빠진 보유
종목은 `position_policy: liquidate` 로 처리한다.

## Files

- `universe-pipeline.yaml`: universe pipeline, SMA strategy, portfolio/run 설정.

## Commands

```bash
mwosa validate backtest examples/backtest/universe-pipeline/universe-pipeline.yaml -o json
mwosa inspect backtest-universe examples/backtest/universe-pipeline/universe-pipeline.yaml --view summary -o table
mwosa inspect backtest-universe examples/backtest/universe-pipeline/universe-pipeline.yaml --view raw -o json
mwosa run backtest examples/backtest/universe-pipeline/universe-pipeline.yaml --view summary -o table
mwosa run backtest examples/backtest/universe-pipeline/universe-pipeline.yaml --view universe -o json
```

