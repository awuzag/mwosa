# Recent ETF Liquidity Rank Screen

YAML pipeline 기반 screening spec 예제다. jq처럼 자유로운 그룹핑을 하기보다,
이미 정의된 selector 를 순서대로 조합해서 검증 가능하고 설명 가능한 후보군을
만든다.

## Files

- `recent-etf-liquidity-rank.screen-run.yaml`: `screen pipeline` 과
  `inspect screen-pipeline` 으로 바로 실행할 수 있는 `ScreenRun` spec.

## Commands

```bash
go run ./cmd/mwosa screen pipeline \
  examples/screen/recent-etf-liquidity-rank/recent-etf-liquidity-rank.screen-run.yaml \
  -o table

go run ./cmd/mwosa inspect screen-pipeline \
  examples/screen/recent-etf-liquidity-rank/recent-etf-liquidity-rank.screen-run.yaml \
  -o json

go run ./cmd/mwosa screen pipeline \
  examples/screen/recent-etf-liquidity-rank/recent-etf-liquidity-rank.screen-run.yaml \
  -o csv
```

## 역할

이 예제는 `source.daily_bars`, `transform.window_metrics`,
`transform.latest_per_symbol`, `filter.field`, `rank.by_field`, `limit.count` 로
구성된다. 같은 조건을 반복 실행하고 explain 을 확인하는 데 적합하다.

첫 일봉과 최신 일봉을 동시에 비교하는 신규상장 ETF 성과 계산은 현재 selector 만으로
표현하기 어렵기 때문에 `new-etf-listing-performance` 예제에서는 jq를 사용한다.

