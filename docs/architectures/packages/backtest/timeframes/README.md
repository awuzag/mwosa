# Backtest Timeframes

## 목적

이 문서는 `packages/backtest` 의 timeframe 계약을 정리한다. 상세한 최종 구현
범위는 [`final-implementation-goal.md`](../final-implementation-goal.md) 를 따른다.

현재 구현은 `1d` daily bar 중심이지만, 엔진 계약은 `1m`, `5m`, `15m`, `30m`,
`1h`, `1d`, `1w`, `1mo`, `custom` 을 수용할 수 있는 방향으로 확장한다.

## 핵심 계약

- `BarFrame` 은 `time.Time` 기반 simulation clock 을 유지한다.
- 각 market data event 는 symbol, market, optional security type, timeframe,
  OHLCV, 거래대금, session, data status 를 표현할 수 있어야 한다.
- 현재 `Bar` 계약은 `timeframe`, `session`, `status` 를 가진다. canonical
  daily source 는 `1d/regular/ok` 또는 no-trade bar 의 경우
  `1d/regular/no_trade` 로 정규화된다. daily source 에서 `1w`, `1mo` 로
  resample 된 bar 는 target timeframe 을 자신의 `timeframe` 으로 가진다.
- multi-timeframe 전략은 named feed 를 사용한다. 예를 들어 `daily` feed 는 큰
  추세를, `intraday` feed 는 진입과 체결을 맡을 수 있다.
- rule evaluator 는 현재 clock 에서 닫힌 bar 와 이전 state 만 읽는다.
- value expression 은 `timeframe` wrapper 로 target timeframe 을 지정할 수
  있다. evaluator 는 현재 clock 이하에서 닫힌 같은 symbol 의 target timeframe
  bar 중 가장 최근 값만 읽는다.
- 상위 timeframe bar 는 아직 닫히지 않았으면 하위 timeframe rule context 에
  노출하지 않는다.
- resample 은 더 작은 source timeframe 에서 더 큰 target timeframe 으로만 수행한다.
- warmup 은 indicator lookback, resample dependency, calendar/session boundary 를
  반영한다.

## 지원 방향

| timeframe | 사용 방향 |
| --- | --- |
| `1m` | intraday 체결, 짧은 진입 rule, 세밀한 stop 판단 |
| `5m` | 장중 추세와 진입 판단 |
| `15m` | intraday noise 를 줄인 판단 |
| `30m` | 세션 내 중기 판단 |
| `1h` | intraday swing 판단 |
| `1d` | 일봉 전략, 현재 구현의 기본 단위 |
| `1w` | 장기 trend/filter, 주간 리밸런싱 |
| `1mo` | 장기 regime, 월간 allocation |
| `custom` | 사용자 정의 calendar, session, bar builder, resample rule |

## Future Leakage 방지

- clock 은 단조 증가한다.
- engine 은 `BarStream` 이 같은 timestamp 를 다시 내보내거나 과거 timestamp 로
  되돌아가면 실행을 중단한다. feed adapter 는 같은 timestamp 의 종목 snapshot 을
  하나의 `BarFrame` 으로 묶어야 한다.
- 현재 bar close 를 보고 같은 close 로 체결하는 정책은 `same_close` 로 명시한다.
- `next_open` 은 다음 사용 가능한 bar open 을 사용한다.
- resampled bar 는 source bar 집합이 모두 닫힌 뒤에만 노출한다.
- timeframe-qualified value 는 current frame 이후 timestamp 를 가진 target
  timeframe bar 를 건너뛴다. 즉 같은 series 에 미래 주봉/월봉이 이미 적재되어
  있어도 현재 clock 에서는 보이지 않는다.
- pending order 가 현재 frame 에서 bar 를 찾지 못하거나 no-trade bar 를 만나면
  engine result 의 `data_events` 에 `data_issue` 로 남기고, execution event 는
  기존 defer/unfilled 흐름을 유지한다.
- warmup 이전 값은 trade signal 로 이어지지 않는다.
- walk-forward 는 train selection 과 out-of-sample test 를 분리한다.

## 완료 조건

- `1d` fixture 와 intraday fixture 가 같은 feed/clock 추상화를 사용한다.
- `daily trend + 5m entry` 전략이 닫힌 bar 규칙을 지킨다.
- future leakage 를 만드는 feed 접근은 테스트에서 실패한다.
- resample, missing bar, no-trade bar, custom calendar 처리 결과가 event 또는
  data issue 로 설명된다.
- result metadata 에 source timeframe, execution timeframe, resample policy,
  warmup policy 가 남는다.
