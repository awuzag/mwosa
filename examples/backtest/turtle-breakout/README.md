# Turtle Breakout Backtest

## What It Shows

터틀 스타일 추세 추종을 `mwosa` 백테스터 문법으로 표현한 예제다. 종가가 100일
SMA 위에 있고, 60일 ROC 가 양수이며, 3종 universe 안에서 60일 ROC 상위 2위
안에 드는 종목만 최근 20일 Donchian high 돌파 시 진입한다. 최근 10일
Donchian low 를 이탈하면 청산하고, 추세가 급하게 꺾일 때를 대비해 14일 ATR 의
2배를 기준으로 하는 `volatility_stop` 도 함께 둔다.

이 예제는 원전 터틀 규칙 전체를 그대로 재현하기보다, 현재 엔진에서 지원하는
Donchian channel, SMA trend filter, ROC relative strength, portfolio state rule,
ATR, stop rule 을 조합해 돌파형 추세 추종 전략의 기본 형태를 보여준다.

## Files

- `turtle-breakout.yaml`: Donchian breakout strategy 와 KRX ETF 3종 실행 조건.
- `buy-and-hold-2024-02-02-to-2024-10-21.yaml`: 같은 ETF 3종을
  2024-02-02 종가에 동일 비중으로 매수하고 2024-10-21 종가에 매도하는 비교용
  baseline.

## Commands

```bash
mwosa validate backtest examples/backtest/turtle-breakout/turtle-breakout.yaml -o json
mwosa run backtest examples/backtest/turtle-breakout/turtle-breakout.yaml --view summary -o table
mwosa run backtest examples/backtest/turtle-breakout/turtle-breakout.yaml --view trades -o csv
mwosa run backtest examples/backtest/turtle-breakout/turtle-breakout.yaml --view events -o table
mwosa run backtest examples/backtest/turtle-breakout/turtle-breakout.yaml --view metrics -o json
```

단순 보유 baseline 과 비교할 때는 두 summary 를 나란히 실행한다.

```bash
mwosa validate backtest examples/backtest/turtle-breakout/buy-and-hold-2024-02-02-to-2024-10-21.yaml -o json
mwosa run backtest examples/backtest/turtle-breakout/buy-and-hold-2024-02-02-to-2024-10-21.yaml --view summary -o table
mwosa run backtest examples/backtest/turtle-breakout/turtle-breakout.yaml --view summary -o table
```

저장된 run 을 다시 조회할 때는 실행 결과의 `id`, `run_name`, 또는 `result_hash` 를
사용한다.

```bash
mwosa inspect backtest-run turtle-breakout-krx-etf --view events -o table
```
