# SMA Cross Backtest

## What It Shows

단일 KRX ETF(`069500`)를 대상으로 20일 SMA 를 기준으로 진입/청산하는 가장 작은
백테스트 smoke 예제다. `Strategy` 와 `BacktestRun` 이 하나의 YAML stream 에 함께
들어 있으며, canonical daily bar 를 `next_open` 체결 가정으로 실행한다.

## Files

- `sma-cross.yaml`: SMA cross strategy 와 단일 실행 조건.

## Commands

```bash
mwosa validate backtest examples/backtest/sma-cross/sma-cross.yaml -o json
mwosa run backtest examples/backtest/sma-cross/sma-cross.yaml --view summary -o table
mwosa run backtest examples/backtest/sma-cross/sma-cross.yaml --view trades -o csv
mwosa run backtest examples/backtest/sma-cross/sma-cross.yaml --view equity -o json
```

저장된 단일 run 을 다시 조회할 때는 실행 결과의 `id`, `run_name`, 또는
`result_hash` 를 사용한다.

```bash
mwosa inspect backtest-run sma-cross-krx-etf --view events -o table
```

