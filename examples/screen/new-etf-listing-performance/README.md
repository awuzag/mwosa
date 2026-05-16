# New ETF Listing Performance Screen

2025년 이후 신규상장 ETF처럼 보이는 종목을 찾고, 저장된 첫 일봉 대비 최신 종가
성과를 계산하는 screening spec 예제다.

## 기준

- 입력 dataset: `etf_daily_metrics`
- 대상: `security_type == "etf"`
- 신규상장 proxy: 로컬 DB에 저장된 해당 ETF의 첫 `trading_date` 가 `2025-01-01`
  이후인 경우
- 기준가:
  - `first_open`: 첫 저장 일봉의 시가
  - `first_close`: 첫 저장 일봉의 종가
  - `latest_close`: 최신 저장 일봉의 종가
- 수익률:
  - `return_from_first_open_pct`
  - `return_from_first_close_pct`
- 유동성 참고값:
  - `latest_traded_amount`
  - `avg_traded_amount_5d`
  - `avg_traded_amount_20d`

로컬 DB에 공식 상장일이나 상장가가 없을 수 있으므로, 이 예제는 공식 상장일
분석이 아니라 "저장된 첫 ETF 일봉" 기준의 관찰용 screen 이다.

## Files

- `new-etf-listing-performance.jq`: jq 실행 본체.
- `new-etf-listing-performance.screen-strategy.yaml`: 같은 jq를 저장형
  `ScreenStrategy` 로 등록하기 위한 spec.
- `yaml-pipeline-follow-up.md`: 현재 YAML pipeline 으로 같은 계산을 표현할 때
  부족한 selector 와 후속 작업.

## Commands

빠른 일회성 실행:

```bash
go run ./cmd/mwosa screen etf \
  --jq-file examples/screen/new-etf-listing-performance/new-etf-listing-performance.jq \
  -o csv
```

전략으로 저장한 뒤 반복 실행:

```bash
go run ./cmd/mwosa create strategy new-etf-listing-performance \
  --engine jq \
  --input etf_daily_metrics \
  --jq-file examples/screen/new-etf-listing-performance/new-etf-listing-performance.jq \
  -o json

go run ./cmd/mwosa screen strategy new-etf-listing-performance \
  --alias new-etf-listing-performance-2026-05-14 \
  -o csv
```

`ScreenStrategy` YAML 파일로 저장할 수도 있다.

```bash
go run ./cmd/mwosa update screen strategy new-etf-listing-performance \
  --file examples/screen/new-etf-listing-performance/new-etf-listing-performance.screen-strategy.yaml \
  -o json
```

## 해석 주의

신규 ETF는 상장 초반 가격 조정, 거래대금 변화, 괴리율 변화가 크게 나타날 수 있다.
이 screen 은 첫날 매수 전략이 아니라, 상장 이후 관찰 후보를 빠르게 정리하는
리서치 도구로 본다.

CSV 출력은 flat row 를 전제로 한다. jq 결과가 중첩 object 를 만들면 CSV 컬럼이
불편해지므로, 이 예제는 스프레드시트와 `jq` 후처리에 바로 넣을 수 있는 scalar
필드만 출력한다.

