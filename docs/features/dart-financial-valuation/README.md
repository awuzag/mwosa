# DART Financial Valuation

## 목적

`mwosa` 의 주식 스크리닝에 DART 기반 재무 데이터와 밸류에이션 점수를 붙이는
피처 제안이다.

KRX provider 로 일봉, 거래량, 시가총액, 52주 고가, MDD 같은 가격 기반 지표는
계산할 수 있다. 하지만 최근 강하게 오른 종목이 정말 비싼지, 실적과 자본 대비
정당한 재평가인지 판단하려면 손익계산서, 재무상태표, 현금흐름표 데이터가
필요하다.

이 문서는 `mwosa screen stocks` 나 저장된 `screen strategy` 가 가격 모멘텀과
밸류에이션을 함께 평가할 수 있도록 DART 재무 데이터 수집, 정규화, 점수화
방향을 정리한다.

## 배경

추세/모멘텀 스크리닝은 자연스럽게 이미 많이 오른 종목을 찾는다.

```yaml
trend:
  - close > ma_20w
  - ma_10w > ma_20w
  - ma_20w > ma_20w_4weeks_ago

momentum:
  rank_by: return_13w
  select: top_20_percent
```

이 조건은 강한 후보를 찾는 데 유용하지만, 다음 질문에는 답하지 못한다.

- 이 가격 상승이 이익 성장으로 설명되는가?
- 현재 PBR, PER, ROE 는 같은 업종 안에서 과한가?
- 영업현금흐름이 순이익을 따라오고 있는가?
- 부채와 자본 구조가 상승 국면을 버틸 만큼 안정적인가?

따라서 주식 스크리닝은 가격 모멘텀과 재무 품질을 분리해서 계산한 뒤, 마지막에
가중치를 합산하는 방향이 더 적합하다.

## 데이터 출처

초기 DART 연동은 OpenDART 를 기본 원천으로 본다.

- OpenDART 정기보고서 재무정보: 단일회사 주요계정, 다중회사 주요계정,
  단일회사 전체 재무제표를 제공한다.
- OpenDART 고유번호: DART `corp_code` 와 KRX 종목코드 매핑에 사용한다.
- DART 공시 검색: 최신 사업보고서, 반기보고서, 분기보고서 접수 여부와
  보고서 번호를 확인하는 보조 경로다.

공공데이터포털 `금융위원회_기업 재무정보` 는 보조 provider 로 둔다. 이 API 는
법인등록번호(`crno`) 와 사업연도로 요약재무제표, 재무상태표, 손익계산서를
조회할 수 있지만, 현재 계획상 현금흐름표 원천으로 보지 않는다.

관련 공식 문서:

- [OpenDART 정기보고서 재무정보](https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS003)
- [OpenDART 고유번호](https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001)
- [금융위원회_기업 재무정보](https://www.data.go.kr/data/15043459/openapi.do)

## 범위

초기 범위는 국내 상장 주식의 스크리닝 보강이다.

- KRX symbol 에서 DART `corp_code` 를 해석한다.
- 사업보고서, 반기보고서, 분기보고서 기준 재무제표를 수집한다.
- 연결 재무제표를 우선 사용하고, 연결이 없을 때 별도 재무제표를 사용한다.
- 손익계산서, 재무상태표, 현금흐름표의 핵심 계정을 canonical financial
  metric 으로 정규화한다.
- 시가총액과 재무 데이터를 결합해 PER, PBR, ROE, 부채비율, 현금흐름 품질
  지표를 계산한다.
- screen 결과에 `valuation_score`, `quality_score`, `growth_score` 를
  제공한다.

초기 범위에서 제외한다.

- DART 원문 XBRL 전체 parser 완성
- 주석 데이터와 세그먼트 데이터 정규화
- 컨센서스, 예상 실적, 목표가
- 금융업 전용 회계 모델의 완전한 처리
- 재무 데이터 기반 매수/매도 추천 문구

## Canonical 지표

스크리닝에 바로 쓰는 1차 지표는 적게 시작한다.

| 그룹 | 지표 | 계산 방향 |
| --- | --- | --- |
| 가치 | `per_ttm` | `market_cap / trailing_net_income` |
| 가치 | `pbr` | `market_cap / equity` |
| 품질 | `roe` | `net_income / average_equity` |
| 품질 | `operating_margin` | `operating_income / revenue` |
| 안정성 | `debt_to_equity` | `total_liabilities / equity` |
| 현금흐름 | `operating_cashflow_to_net_income` | `operating_cashflow / net_income` |
| 성장 | `revenue_growth_yoy` | 전년 동기 매출 성장률 |
| 성장 | `operating_income_growth_yoy` | 전년 동기 영업이익 성장률 |

계정명은 provider-native 이름을 그대로 screen surface 로 노출하지 않는다. DART
계정 ID 와 한국어 계정명은 raw provenance 와 extension 으로 남기고, CLI 와
screen strategy 는 canonical metric 이름을 사용한다.

## 점수화 방향

밸류에이션은 단일 pass/fail 보다 가중치 기반 score 로 제공한다.

```yaml
valuation:
  weights:
    roe_quality: 0.30
    earnings_growth: 0.25
    pbr_relative: 0.20
    per_relative: 0.15
    cashflow_quality: 0.10
```

상대 비교는 전체 시장이 아니라 가능한 한 같은 업종 안에서 한다.

- `per_relative`: 업종 PER percentile 이 낮을수록 높은 점수
- `pbr_relative`: 업종 PBR percentile 이 낮을수록 높은 점수
- `roe_quality`: 업종 ROE 중앙값보다 높을수록 높은 점수
- `earnings_growth`: 매출과 영업이익이 함께 개선될수록 높은 점수
- `cashflow_quality`: 영업현금흐름이 순이익을 뒷받침할수록 높은 점수

가격 모멘텀 점수와 밸류에이션 점수는 분리해서 출력한다. 그래야 사용자가
`강하지만 비싼 종목`, `덜 올랐지만 싼 종목`, `강하면서도 재무가 따라오는 종목`
을 구분할 수 있다.

## CLI 초안

재무 데이터 수집은 provider-backed 명령으로 시작한다.

```bash
mwosa sync financials \
  --provider dart \
  --security-type stock \
  --year 2025 \
  --report annual \
  -o json
```

단일 종목 조회는 기존 `get financials` 흐름과 맞춘다.

```bash
mwosa get financials 005930 \
  --provider dart \
  --year 2025 \
  --statement income_statement \
  -o json
```

스크리닝은 가격 지표와 재무 점수를 함께 받는다.

```bash
mwosa screen stocks \
  --universe kospi \
  --strategy strategies/kospi-trend-value.yaml \
  --as-of latest \
  -o table
```

전략 파일 예시는 다음처럼 둔다.

```yaml
kind: ScreenStrategy
schema_version: 1
name: kospi-trend-value

universe:
  market: krx
  security_type: stock
  segment: kospi

trend:
  all:
    - close > ma_20w
    - ma_10w > ma_20w
    - ma_20w > ma_20w_4weeks_ago

momentum:
  rank_by: return_13w
  select: top_20_percent

risk:
  all:
    - weekly_volume_avg >= minimum_liquidity
    - volatility_13w <= max_volatility
    - close >= 0.85 * high_52w

valuation:
  score:
    roe_quality: 0.30
    earnings_growth: 0.25
    pbr_relative: 0.20
    per_relative: 0.15
    cashflow_quality: 0.10
  minimum_score: 60
```

## 저장 모델 방향

가격 데이터와 재무 데이터는 갱신 주기와 provenance 가 다르다. 같은 daily bar
테이블에 섞지 않는다.

- `instrument_v2`: KRX symbol, ISIN, DART `corp_code`, 법인등록번호 같은
  식별자 연결점
- `financial_statement`: provider, statement type, fiscal year, fiscal
  period, report code, consolidated 여부
- `financial_line_item`: 계정 ID, 계정명, 금액, 통화, 단위, raw order
- `financial_metric`: screen 에 쓰는 canonical metric 과 계산 provenance
- `valuation_snapshot`: 특정 as-of 기준의 시가총액과 재무 metric 결합 결과

provider-native 원문은 raw snapshot 으로 보존하되, screen 은 canonical metric 만
읽는다.

## 완료 기준

MVP 완료 기준은 다음과 같다.

- DART `corp_code` 와 KRX symbol 을 연결할 수 있다.
- 단일 종목의 사업보고서 기준 손익계산서, 재무상태표, 현금흐름표를 조회할 수
  있다.
- canonical metric 최소 6개 이상을 계산한다.
- `mwosa get financials <symbol> --provider dart -o json` 이 jq 로 다루기 쉬운
  구조를 출력한다.
- KOSPI 후보군에 대해 `valuation_score` 를 계산해 screen 결과에 합칠 수 있다.
- missing filing, unsupported statement, account mapping 실패는 빈 성공이 아니라
  명시적인 error 또는 uncomputable reason 으로 드러난다.

## 확인할 질문

- 업종 분류 원천은 KRX, DART, KIS 중 무엇을 우선할 것인가?
- 연결/별도 재무제표 우선순위는 모든 업종에 동일하게 적용할 것인가?
- 금융업, 지주사, 리츠는 같은 valuation score 로 평가할 것인가, 별도 모델로
  분리할 것인가?
- TTM 계산은 최근 4개 분기를 합산할 것인가, 최근 사업연도와 최신 분기를
  조합할 것인가?
- 공시 정정이 발생했을 때 기존 valuation snapshot 을 어떻게 무효화할 것인가?
