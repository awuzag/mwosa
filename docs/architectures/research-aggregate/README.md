# Research Aggregate / Projection Boundary

## 목적

`mwosa` 의 리서치 모델은 provider API 응답 모양이나 SQLite table row 모양을
그대로 domain aggregate 로 삼지 않는다. OpenDART, KRX, Datago, KIS 같은 원천은
서로 다른 식별자와 갱신 주기를 가지므로, application layer 는
`company_id` 와 `instrument_id` 를 기준으로 provider-neutral read model 을 조립한다.

현재 구현은 DDD aggregate 를 강하게 도입하기보다 read model/projection 경계를 먼저
분리한다. 이 경계의 첫 코드 표면은 `service/research` 의
`StockResearchProfile` 과 `ScreenCandidate` 다.

## 현재 persistence 와 projection

현재 Bun model 기반 storage 는 canonical persistence 이지만, 여전히 table row
중심이다. row 는 재계산과 provenance 를 위해 provider/source 힌트를 보존한다.

| persistence row | projection 역할 |
| --- | --- |
| `company_v1`, `company_identifier_v1`, `instrument_company_link_v1` | `CompanyIdentity` |
| `financial_statement_v1`, `financial_line_item_v1` | `FinancialStatementSet` 후보 |
| `financial_metric_v1` | `FinancialProfile.metrics` |
| `valuation_snapshot_v1` | `ValuationProfile.snapshots` |
| `company_fact_v1` | `CapitalPolicyProfile`, `GovernanceProfile` |
| `company_event_v1` | `DisclosureEventTimeline` |
| stored daily bars + fundamentals | `ScreenCandidate` |

`rcept_no`, `report_code`, `fs_div`, `account_id`, `ord`, `raw_json`,
`extensions_json` 같은 값은 canonical key 가 아니다. 필요한 경우 provenance 또는
source reference 로 보존하지만, CLI 와 strategy 는 가능한 한 `financial_metrics`,
`valuation`, `company_facts`, `company_events` 같은 mwosa domain language 를 우선
사용한다.

## Read model 목표

`StockResearchProfile` 은 종목 리서치 화면을 위한 application read model 이다.
대략 아래 모델들을 한 번에 읽을 수 있게 묶는다.

| 모델 | 의미 |
| --- | --- |
| `CompanyIdentity` | 회사, 식별자, 상장 instrument 연결 |
| `FinancialProfile` | canonical metric, 성장/품질/안정성 계산값 |
| `ValuationProfile` | 가격/시가총액과 재무 metric 결합 결과 |
| `CapitalPolicyProfile` | 배당, 자기주식, 증자/감자, 희석 후보 |
| `GovernanceProfile` | 감사의견, 최대주주, 임원/주요주주 후보 |
| `DisclosureEventTimeline` | 공시 기반 이벤트와 risk flag 후보 |
| `ScreenCandidate` | 가격 모멘텀과 재무/가치/이벤트 입력을 결합한 전략 평가 단위 |

현재 `inspect stock` 은 기존 summary JSON 을 유지하면서 `research_profile` 을 함께
제공한다. 이는 기존 CLI 소비자를 깨지 않으면서 provider-neutral projection 으로
옮겨가는 중간 단계다.

## 경계 규칙

- `corp_code`, `stock_code`, `crno`, `isin` 은 canonical key 가 아니라 identifier 다.
- OpenDART API 응답 구조를 그대로 domain aggregate 로 삼지 않는다.
- raw/file 응답은 `provider_raw_snapshots` 또는 file escape hatch 로 둔다.
- storage row type 은 persistence detail 이며, service/CLI 표면에서는 projection
  DTO 를 우선한다.
- missing, uncomputable, unsupported 는 빈 성공처럼 숨기지 않고 `missing`,
  `uncomputable_reason`, error 로 드러낸다.
- 대규모 schema migration 은 read model 경계가 안정된 뒤에 별도 결정으로 다룬다.
