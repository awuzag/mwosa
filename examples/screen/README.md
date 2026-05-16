# Screening Examples

`mwosa` screening spec 은 두 실행 방식을 함께 둔다.

- `jq`: 저장된 daily-bar JSON row 를 자유롭게 그룹핑, 집계, 계산하는 탐색형 spec.
- `yaml_pipeline`: selector, transform, filter, rank 를 검증 가능한 단계로 반복 실행하는 구조형 spec.

둘은 대체 관계가 아니다. `jq` 는 symbol 별 첫 row/최신 row 추출처럼 shape 를 크게
바꾸는 리서치에 강하고, YAML pipeline 은 이미 정의된 selector 를 조합해 후보군을
재현하고 설명하는 데 강하다.

## Examples

- `new-etf-listing-performance`: 2025년 이후 저장된 첫 ETF 일봉을 신규상장 proxy 로
  보고, 첫 일봉 대비 최신 종가 수익률을 계산하는 jq 기반 예제.
- `recent-etf-liquidity-rank`: 최근 일봉과 20일 평균 거래대금을 이용해 유동성
  있는 ETF 후보를 고르는 YAML pipeline 예제.

